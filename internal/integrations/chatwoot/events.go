package chatwoot

import (
	"context"

	"wzap/internal/logger"
	"wzap/internal/model"
)

// eventDispatcher encapsula o switch de eventos inbound, delegando cada
// caso ao método process* correspondente no Service. Mantém ponteiro de
// volta para o Service porque os handlers dependem das deps injetadas
// (cache, clientFn, repos, etc.).
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
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("permanent error in processMessage, dropping")
		}
	case model.EventGroupInfo:
		if err := s.processGroupInfo(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("permanent error in processGroupInfo, dropping")
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
	case model.EventQR:
		s.processQR(ctx, cfg, payload)
	case model.EventContact:
		s.processContact(ctx, cfg, payload)
	case model.EventPushName:
		s.processPushName(ctx, cfg, payload)
	case model.EventPicture:
		s.processPicture(ctx, cfg, payload)
	case model.EventHistorySync:
		s.processHistorySync(ctx, cfg, payload)
	}
	return nil
}
