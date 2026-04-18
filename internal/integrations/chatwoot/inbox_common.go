package chatwoot

import (
	"context"
	"strings"

	"wzap/internal/logger"
	"wzap/internal/metrics"
)

// inboxPrologueOpts configura passos opcionais do prólogo compartilhado.
type inboxPrologueOpts struct {
	// checkDBIdempotency: quando true, além do cache, consulta msgRepo.ExistsBySourceID
	// para detectar duplicatas que já foram persistidas (apenas modo API faz isso).
	checkDBIdempotency bool
}

// inboxPrologueResult carrega o payload desempacotado e metadados derivados
// quando o prólogo decide prosseguir.
type inboxPrologueResult struct {
	data     *waMessagePayload
	chatJID  string
	sourceID string
}

// inboxPrologue executa a sequência inicial compartilhada dos dois handlers de
// inbox (API e Cloud): parse → resolve @lid → filtro de JID ignorado → checagem
// idempotente (cache e, opcionalmente, banco). Retorna (result, skip, err):
//   - skip=true indica que o caller DEVE encerrar silenciosamente com o err
//     retornado (nil em skips benignos como duplicata/LID irresolvível).
//   - skip=false + result preenchido indica que o caller pode prosseguir com
//     o processamento específico do modo.
func (s *Service) inboxPrologue(ctx context.Context, cfg *Config, payload []byte, opts inboxPrologueOpts) (*inboxPrologueResult, bool, error) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to parse message payload")
		return nil, true, nil
	}

	chatJID := data.Info.Chat
	if chatJID == "" {
		logger.Warn().Str("component", "chatwoot").Msg("chatJID empty, skipping")
		return nil, true, nil
	}

	if strings.HasSuffix(chatJID, "@lid") {
		logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).
			Bool("fromMe", data.Info.IsFromMe).
			Str("chatLID", chatJID).
			Str("senderAlt", data.Info.SenderAlt).
			Str("recipientAlt", data.Info.RecipientAlt).
			Msg("resolving LID chat")
	}
	if data.Info.IsFromMe {
		chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.RecipientAlt)
	} else {
		chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.SenderAlt, data.Info.RecipientAlt)
	}
	if strings.HasSuffix(chatJID, "@lid") {
		logger.Warn().Str("component", "chatwoot").Str("lid", chatJID).Str("session", cfg.SessionID).Msg("unresolvable LID chat, skipping")
		return nil, true, nil
	}

	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		logger.Debug().Str("component", "chatwoot").Str("chat", chatJID).Msg("JID ignored, skipping")
		return nil, true, nil
	}

	msgID := data.Info.ID
	sourceID := "WAID:" + msgID

	if msgID != "" {
		_, idemSpan := startSpan(ctx, "chatwoot.check_idempotency",
			spanAttrs(cfg.SessionID, "message", "inbound")...)
		isDup := s.cache.GetIdempotent(ctx, cfg.SessionID, sourceID)
		if !isDup && opts.checkDBIdempotency {
			if exists, dbErr := s.msgRepo.ExistsBySourceID(ctx, cfg.SessionID, sourceID); dbErr == nil && exists {
				s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
				isDup = true
			}
		}
		idemSpan.End()
		if isDup {
			logger.Debug().Str("component", "chatwoot").Str("sourceID", sourceID).Msg("inbound duplicate, skipping")
			metrics.CWIdempotentDrops.WithLabelValues(cfg.SessionID).Inc()
			return nil, true, nil
		}
	}

	return &inboxPrologueResult{
		data:     data,
		chatJID:  chatJID,
		sourceID: sourceID,
	}, false, nil
}
