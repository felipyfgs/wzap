package elodesk

import (
	"context"
	"strings"

	"wzap/internal/logger"
)

type InboxHandler interface {
	HandleMessage(ctx context.Context, cfg *Config, payload []byte) error
}

func (s *Service) processMessage(ctx context.Context, cfg *Config, payload []byte) error {
	return newAPIInboxHandler(s).HandleMessage(ctx, cfg, payload)
}

type inboxPrologueResult struct {
	data     *waMessagePayload
	chatJID  string
	sourceID string
}

// inboxPrologue roda parse → resolve LID → filtro → checagem idempotência.
// skip=true ⇒ caller encerra silenciosamente.
func (s *Service) inboxPrologue(ctx context.Context, cfg *Config, payload []byte) (*inboxPrologueResult, bool) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Msg("Failed to parse message payload")
		return nil, true
	}

	chatJID := data.Info.Chat
	if chatJID == "" {
		return nil, true
	}

	if data.Info.IsFromMe {
		chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.RecipientAlt)
	} else {
		chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.SenderAlt, data.Info.RecipientAlt)
	}
	chatJID = stripDeviceFromJID(chatJID)
	if strings.HasSuffix(chatJID, "@lid") {
		logger.Warn().Str("component", "elodesk").Str("lid", chatJID).Str("session", cfg.SessionID).Msg("unresolvable LID chat, skipping")
		return nil, true
	}

	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		return nil, true
	}

	msgID := data.Info.ID
	sourceID := "WAID:" + msgID

	if msgID != "" {
		isDup := s.cache.GetIdempotent(ctx, cfg.SessionID, sourceID)
		if !isDup {
			if exists, dbErr := s.msgRepo.ExistsByElodeskSrcID(ctx, cfg.SessionID, sourceID); dbErr == nil && exists {
				s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
				isDup = true
			}
		}
		if isDup {
			logger.Debug().Str("component", "elodesk").Str("sourceID", sourceID).Msg("inbound duplicate, skipping")
			return nil, true
		}
	}

	return &inboxPrologueResult{
		data:     data,
		chatJID:  chatJID,
		sourceID: sourceID,
	}, false
}
