package chatwoot

import (
	"context"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/model"
)

type InboxHandler interface {
	HandleMessage(ctx context.Context, cfg *Config, payload []byte) error
	UnlockWindow(ctx context.Context, cfg *Config, chatJID string)
}

func (s *Service) getInboxHandler(cfg *Config) InboxHandler {
	if cfg.InboxType == "cloud" {
		return newCloudInboxHandler(s)
	}
	return newAPIInboxHandler(s)
}

func (s *Service) processMessage(ctx context.Context, cfg *Config, payload []byte) error {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to parse message payload")
		return nil
	}

	if cfg.InboxType == "cloud" && !data.Info.IsFromMe {
		return s.getInboxHandler(cfg).HandleMessage(ctx, cfg, payload)
	}

	if cfg.InboxType == "cloud" && data.Info.IsFromMe {
		chatJID := data.Info.Chat
		if chatJID == "" {
			return nil
		}

		if data.Info.RecipientAlt != "" {
			chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.RecipientAlt)
		}
		if strings.HasSuffix(chatJID, "@lid") {
			return nil
		}

		if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
			return nil
		}

		if _, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID); !ok {
			handler := s.getInboxHandler(cfg)
			handler.UnlockWindow(ctx, cfg, chatJID)
			time.Sleep(3 * time.Second)
		}

		return s.getInboxHandler(cfg).HandleMessage(ctx, cfg, payload)
	}

	return s.getInboxHandler(cfg).HandleMessage(ctx, cfg, payload)
}

var _ model.EventType = ""
