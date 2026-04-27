package elodesk

import (
	"context"
	"fmt"
	"strings"

	"wzap/internal/logger"
)

// convResult retorna do singleflight: convID + contactSrcID (identifier do
// contato no elodesk, necessário para todos os POSTs subsequentes).
type convResult struct {
	convID       int64
	contactSrcID string
}

// findOrCreateConversation retorna conv_id + contact source_id, cache-first.
// Dedup por sessionID+chatJID via singleflight para evitar N upserts em
// rajadas de eventos do mesmo chat.
func (s *Service) findOrCreateConversation(ctx context.Context, cfg *Config, chatJID, pushName string) (int64, string, error) {
	if convID, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID); ok {
		contactSrcID := contactSourceIDFromJID(chatJID)
		return convID, contactSrcID, nil
	}

	key := cfg.SessionID + "+" + chatJID
	result, err, _ := s.convFlight.Do(key, func() (any, error) {
		if convID, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID); ok {
			return convResult{convID: convID, contactSrcID: contactSourceIDFromJID(chatJID)}, nil
		}
		convID, contactSrcID, err := s.upsertConversation(ctx, cfg, chatJID, pushName)
		if err != nil {
			return nil, err
		}
		return convResult{convID: convID, contactSrcID: contactSrcID}, nil
	})
	if err != nil {
		return 0, "", err
	}

	r, ok := result.(convResult)
	if !ok {
		return 0, "", fmt.Errorf("invalid conversation result type %T", result)
	}
	return r.convID, r.contactSrcID, nil
}

func (s *Service) upsertConversation(ctx context.Context, cfg *Config, chatJID, pushName string) (int64, string, error) {
	client := s.clientFn(cfg)
	isGroup := strings.HasSuffix(chatJID, "@g.us")

	contactName := pushName
	if s.contactNameGetter != nil {
		if name := s.contactNameGetter.GetContactName(ctx, cfg.SessionID, chatJID); name != "" {
			contactName = name
		}
	}
	if contactName == "" {
		if isGroup {
			contactName = chatJID
		} else {
			contactName = extractPhone(chatJID)
		}
	}

	contactSrcID := contactSourceIDFromJID(chatJID)
	req := UpsertContactReq{
		SourceID:   contactSrcID,
		Identifier: chatJID,
		Name:       contactName,
	}
	if isGroup {
		req.AdditionalAttributes = map[string]any{"is_group": true}
		// Follow-up: when Elodesk exposes the participant sync endpoint on the
		// public API (currently only on the JWT-protected
		// POST /api/v1/accounts/:aid/conversations/:id/participants/sync), call
		// it here with the group's whatsmeow member roster so the UI can render
		// the participants list. Tracked as openspec task 11.2.
	} else {
		req.PhoneNumber = "+" + extractPhone(chatJID)
	}

	if s.picGetter != nil {
		if picURL, err := s.picGetter.GetProfilePicture(ctx, cfg.SessionID, chatJID); err == nil && picURL != "" {
			req.AvatarURL = picURL
		}
	}

	if _, err := client.UpsertContact(ctx, cfg.InboxIdentifier, req); err != nil {
		return 0, "", fmt.Errorf("upsert contact: %w", err)
	}

	convReq := GetOrCreateConvReq{ContactIdentifier: chatJID}
	if cfg.PendingConv {
		convReq.Status = "pending"
	}

	conv, err := client.GetOrCreateConversation(ctx, cfg.InboxIdentifier, contactSrcID, convReq)
	if err != nil {
		return 0, "", fmt.Errorf("get or create conversation: %w", err)
	}

	if conv.Status == ConversationStatusResolved && cfg.ReopenConv {
		reopenStatus := "open"
		if cfg.PendingConv {
			reopenStatus = "pending"
		}
		if err := client.UpdateConversationStatus(ctx, cfg.InboxIdentifier, contactSrcID, int64(conv.ID), reopenStatus); err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Int("convID", conv.ID).Msg("failed to reopen conversation")
		}
	}

	s.cache.SetConv(ctx, cfg.SessionID, chatJID, int64(conv.ID), int64(conv.ContactID))
	return int64(conv.ID), contactSrcID, nil
}

// contactSourceIDFromJID deriva o source_id que o elodesk expõe no path
// /public/api/v1/inboxes/{identifier}/contacts/{source_id}/.../
// Convenção: usar o próprio JID — é único por contato/WA.
func contactSourceIDFromJID(jid string) string {
	return jid
}
