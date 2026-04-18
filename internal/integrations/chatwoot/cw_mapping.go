package chatwoot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"wzap/internal/logger"
)

// ChatwootMessageRef is the resolved pair of IDs used by the mapping between a
// WhatsApp message (identified by its WAID source_id on Chatwoot side) and the
// Chatwoot message/conversation IDs. It is used by the cloud inbox flow to
// backfill wz_messages.cw_* columns without relying on the optional webhook.
type ChatwootMessageRef struct {
	MessageID      int
	ConversationID int
	SourceID       string
}

// ResolveMessageBySourceID looks up a message on the Chatwoot database using
// the database_uri configured for the session. It returns (ref, true, nil)
// when found. When database_uri is not configured, or the message cannot be
// located, it returns (nil, false, nil) so callers can gracefully fall back to
// other resolution strategies (retry loop + message_created webhook).
func ResolveMessageBySourceID(ctx context.Context, cfg *Config, sourceID string) (*ChatwootMessageRef, bool, error) {
	if cfg == nil || cfg.DatabaseURI == "" || sourceID == "" {
		return nil, false, nil
	}

	pool, err := getPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, false, fmt.Errorf("chatwoot mapping pool: %w", err)
	}

	ref := &ChatwootMessageRef{}
	err = pool.QueryRow(ctx,
		`SELECT id, conversation_id, source_id
		   FROM messages
		  WHERE source_id = $1 AND account_id = $2
		  ORDER BY id DESC
		  LIMIT 1`,
		sourceID, cfg.AccountID,
	).Scan(&ref.MessageID, &ref.ConversationID, &ref.SourceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("resolve chatwoot message by source id: %w", err)
	}

	return ref, true, nil
}

// ResolveConversationForContactPhone locates the active Chatwoot conversation
// for a given phone number in the inbox configured for this session. It is
// used as a fallback when the WAID lookup fails (e.g. message not yet
// persisted on Chatwoot side) but we still need the conversation ID to issue
// follow-up requests.
func ResolveConversationForContactPhone(ctx context.Context, cfg *Config, phone string) (int, bool, error) {
	if cfg == nil || cfg.DatabaseURI == "" || phone == "" || cfg.InboxID == 0 {
		return 0, false, nil
	}

	pool, err := getPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return 0, false, fmt.Errorf("chatwoot mapping pool: %w", err)
	}

	var convID int
	err = pool.QueryRow(ctx,
		`SELECT c.id
		   FROM conversations c
		   JOIN contact_inboxes ci ON ci.id = c.contact_inbox_id
		  WHERE ci.inbox_id = $1
		    AND ci.source_id = $2
		  ORDER BY c.last_activity_at DESC NULLS LAST, c.id DESC
		  LIMIT 1`,
		cfg.InboxID, phone,
	).Scan(&convID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("resolve chatwoot conversation by phone: %w", err)
	}

	return convID, true, nil
}

// ResolveAndPersistMessageRef is a thin helper used by the cloud flow: it
// resolves the Chatwoot message mapping for a WhatsApp message and persists it
// on wz_messages so subsequent operations (edit/revoke/reply) can reuse the
// cached ref without another DB round-trip. It returns the resolved ref when
// successful so callers can use it immediately.
func (s *Service) resolveAndPersistMessageRef(ctx context.Context, cfg *Config, waMsgID string) (*ChatwootMessageRef, bool) {
	if s == nil || cfg == nil || waMsgID == "" {
		return nil, false
	}

	// Cloud inbox stores source_id as raw ID; API inbox uses "WAID:" prefix.
	// Try raw first (cloud), then with prefix (API).
	ref, ok, err := ResolveMessageBySourceID(ctx, cfg, waMsgID)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Msg("failed to resolve chatwoot message by source id")
		return nil, false
	}
	if !ok || ref == nil {
		ref, ok, err = ResolveMessageBySourceID(ctx, cfg, "WAID:"+waMsgID)
		if err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Msg("failed to resolve chatwoot message by source id (WAID)")
			return nil, false
		}
		if !ok || ref == nil {
			return nil, false
		}
	}

	if err := s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, waMsgID, ref.MessageID, ref.ConversationID, ref.SourceID); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Msg("failed to persist chatwoot ref after resolve")
		return ref, true
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Int("cwMsgID", ref.MessageID).Int("cwConvID", ref.ConversationID).Msg("resolved chatwoot ref via database lookup")
	return ref, true
}

// resolveCloudRefAsync launches a background goroutine that attempts to resolve
// and persist the Chatwoot message ref for a WhatsApp message after it has been
// forwarded to Chatwoot via the cloud webhook. It tries database_uri first, then
// falls back to the Chatwoot REST API. This ensures edit/revoke events can find
// the CW refs without depending on the message_created return webhook.
func (s *Service) resolveCloudRefAsync(cfg *Config, waMsgID, chatJID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		delays := []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}

		for i, delay := range delays {
			time.Sleep(delay)

			// Strategy 1: database_uri (fast, direct read)
			if ref, ok := s.resolveAndPersistMessageRef(ctx, cfg, waMsgID); ok && ref != nil {
				return
			}

			// Strategy 2: Chatwoot REST API (works without database_uri)
			if ref, ok := s.resolveCloudRefViaAPI(ctx, cfg, waMsgID, chatJID); ok && ref != nil {
				return
			}

			logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Int("attempt", i+1).Msg("cloud ref not yet available, will retry")
		}

		logger.Warn().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Msg("cloud ref: could not resolve CW refs after async retries")
	}()
}

// resolveCloudRefViaAPI uses the Chatwoot REST API to find a message by its
// source_id (WAID:xxx) within the conversation for the given chatJID. This
// works without database_uri and is used as a fallback when the direct DB
// lookup is unavailable. In cloud mode, conversations are created by Chatwoot
// (not wzap), so the local cache may be empty — this function resolves the
// conversation via FilterContacts + ListConversations when needed.
func (s *Service) resolveCloudRefViaAPI(ctx context.Context, cfg *Config, waMsgID, chatJID string) (*ChatwootMessageRef, bool) {
	if s == nil || cfg == nil || waMsgID == "" {
		return nil, false
	}

	convID, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID)
	if !ok {
		// Cache miss — in cloud mode, conversations are created by Chatwoot,
		// not by wzap. Resolve via API: FilterContacts → ListConversations.
		convID = s.findConvIDViaAPI(ctx, cfg, chatJID)
		if convID == 0 {
			return nil, false
		}
	}

	// Cloud inbox stores source_id as the raw WA message ID (no "WAID:" prefix).
	client := s.clientFn(cfg)

	msg, err := client.FindMessageBySourceID(ctx, convID, waMsgID)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Msg("cloud ref via API: FindMessageBySourceID failed")
		return nil, false
	}
	if msg == nil {
		return nil, false
	}

	ref := &ChatwootMessageRef{
		MessageID:      msg.ID,
		ConversationID: convID,
		SourceID:       waMsgID,
	}

	if err := s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, waMsgID, ref.MessageID, ref.ConversationID, ref.SourceID); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Msg("cloud ref via API: failed to persist ref")
		return ref, true
	}

	// Cache the resolved conversation for future lookups
	if s.cache != nil {
		s.cache.SetConv(ctx, cfg.SessionID, chatJID, convID, 0)
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Int("cwMsgID", ref.MessageID).Int("cwConvID", ref.ConversationID).Msg("resolved chatwoot ref via API lookup")
	return ref, true
}

// findConvIDViaAPI resolves the Chatwoot conversation ID for a chatJID by
// querying the Chatwoot REST API. Used when the local conversation cache is
// empty (typical in cloud inbox mode where Chatwoot creates conversations).
func (s *Service) findConvIDViaAPI(ctx context.Context, cfg *Config, chatJID string) int {
	// Resolve LID → phone number. In cloud mode, wz_messages may store the
	// @lid JID, but Chatwoot contacts are indexed by phone number.
	resolvedJID := chatJID
	if strings.HasSuffix(chatJID, "@lid") && s.jidResolver != nil {
		if pn := s.jidResolver.GetPNForLID(ctx, cfg.SessionID, chatJID); pn != "" {
			resolvedJID = pn + "@s.whatsapp.net"
		}
	}

	phone := extractPhone(resolvedJID)
	if phone == "" {
		return 0
	}

	client := s.clientFn(cfg)

	contacts, err := client.FilterContacts(ctx, phone)
	if err != nil || len(contacts) == 0 {
		logger.Debug().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("phone", phone).Msg("cloud ref via API: FilterContacts found no contact")
		return 0
	}

	convs, err := client.ListConversations(ctx, contacts[0].ID)
	if err != nil || len(convs) == 0 {
		logger.Debug().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Int("contactID", contacts[0].ID).Msg("cloud ref via API: ListConversations found no conversation")
		return 0
	}

	// Prefer the most recent conversation in the configured inbox
	for _, conv := range convs {
		if conv.InboxID == cfg.InboxID {
			return conv.ID
		}
	}

	// Fallback: return the first conversation
	return convs[0].ID
}
