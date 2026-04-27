package elodesk

import (
	"context"

	"wzap/internal/logger"
	"wzap/internal/model"
)

// eventDispatcher roteia EventType → processXxx. Espelha chatwoot/events.go.
// No MVP a maioria dos eventos cai em debug-log + no-op; a cobertura só
// precisa ser "não-panic + retorna sem erro" para fechar a spec
// (specs/elodesk-integration/spec.md, item "47 EventTypes").
type eventDispatcher struct {
	svc *Service
}

func (d *eventDispatcher) Handle(ctx context.Context, cfg *Config, event model.EventType, payload []byte) error {
	s := d.svc
	switch event {
	case model.EventMessage:
		if err := s.processMessage(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Msg("permanent error in processMessage, dropping")
		}
	case model.EventReceipt:
		s.processReceipt(ctx, cfg, payload)
	case model.EventDeleteForMe:
		s.processDelete(ctx, cfg, payload)
	case model.EventMessageRevoke:
		s.processRevoke(ctx, cfg, payload)
	case model.EventMessageEdit:
		s.processEdit(ctx, cfg, payload)
	case model.EventConnected:
		s.processConnected(ctx, cfg, payload)
	case model.EventDisconnected:
		s.processDisconnected(ctx, cfg, payload)
	case model.EventHistorySync:
		logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Msg("HistorySync received (no-op until import triggered)")
	default:
		// Todos os demais EventTypes válidos caem aqui: QR, Contact, PushName,
		// Picture, GroupInfo, Presence, Chat State, Labels, Calls, Newsletter,
		// Sync, Privacy, FBMessage, etc. Registramos em debug para cobrir o
		// requisito de "handler não panicar" sem poluir logs.
		logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Str("event", string(event)).Msg("event not handled by elodesk MVP, skipping")
	}
	return nil
}

// processReceipt atualiza "last seen" das mensagens que o destinatário WA
// leu, sinalizando isso no elodesk. No MVP o elodesk Channel::Wzap não expõe
// endpoint de update_last_seen dedicado — ficamos em no-op explícito.
func (s *Service) processReceipt(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseReceiptPayload(payload)
	if err != nil {
		return
	}
	if len(data.MessageIDs) == 0 {
		return
	}
	for _, msgID := range data.MessageIDs {
		if msgID == "" {
			continue
		}
		msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
		if err != nil {
			continue
		}
		if msg.ElodeskConvID == nil || *msg.ElodeskConvID == 0 {
			continue
		}
		// placeholder para futura chamada a UpdateLastSeen no elodesk
		logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Str("msgID", msgID).Msg("receipt received (update_last_seen is no-op in MVP)")
	}
}

func (s *Service) processDelete(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseDeletePayload(payload)
	if err != nil {
		return
	}
	if data.MessageID == "" {
		return
	}
	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, data.MessageID)
	if err != nil {
		return
	}
	if msg.ElodeskMessageID == nil || msg.ElodeskConvID == nil {
		return
	}
	// O elodesk Channel::Wzap não expõe DELETE /messages público no MVP.
	logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Str("msgID", data.MessageID).Int64("elMsgID", *msg.ElodeskMessageID).Msg("delete-for-me received (no-op in MVP)")
}

func (s *Service) processRevoke(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		return
	}
	protocolMsg := getMapField(data.Message, "protocolMessage")
	if protocolMsg == nil {
		return
	}
	key := getMapField(protocolMsg, "key")
	if key == nil {
		return
	}
	revokedMsgID := getStringField(key, "ID")
	if revokedMsgID == "" {
		return
	}
	logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Str("revokedMsgID", revokedMsgID).Msg("revoke received (no-op in MVP)")
}

func (s *Service) processEdit(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		return
	}
	protocolMsg := getMapField(data.Message, "protocolMessage")
	if protocolMsg == nil {
		return
	}
	key := getMapField(protocolMsg, "key")
	if key == nil {
		return
	}
	editedMsgID := getStringField(key, "ID")
	if editedMsgID == "" {
		return
	}
	logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Msg("edit received (no-op in MVP)")
}

func (s *Service) processConnected(ctx context.Context, cfg *Config, _ []byte) {
	if cfg.ImportOnConnect {
		period := cfg.ImportPeriod
		if period == "" {
			period = "7d"
		}
		go func() {
			bg := context.Background()
			s.ImportHistoryAsync(bg, cfg.SessionID, period, 0)
		}()
	}
	_ = ctx
}

func (s *Service) processDisconnected(_ context.Context, cfg *Config, _ []byte) {
	logger.Debug().Str("component", "elodesk").Str("session", cfg.SessionID).Msg("session disconnected")
}
