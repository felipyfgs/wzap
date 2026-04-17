package chatwoot

import (
	"context"

	"wzap/internal/logger"
	"wzap/internal/model"
)

type InboxHandler interface {
	HandleMessage(ctx context.Context, cfg *Config, payload []byte) error
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

	// Cloud mode só processa inbound (fromMe=false). Outbound (mensagens
	// enviadas pelo agente via Chatwoot) não precisa ser re-ecoado porque o
	// próprio Chatwoot já persistiu essa mensagem localmente.
	if cfg.InboxType == "cloud" && data.Info.IsFromMe {
		return nil
	}

	return s.getInboxHandler(cfg).HandleMessage(ctx, cfg, payload)
}

var _ model.EventType = ""
