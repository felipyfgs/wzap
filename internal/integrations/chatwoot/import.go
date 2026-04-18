package chatwoot

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
)

// ImportHistoryAsync inicia um import de histórico em background protegido
// por singleflight: múltiplas chamadas para a mesma sessão coalescem em
// uma única execução.
func (s *Service) ImportHistoryAsync(ctx context.Context, sessionID, period string, customDays int) {
	key := "import:" + sessionID
	_, _, _ = s.importFlight.Do(key, func() (any, error) {
		importCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.importHistory(importCtx, sessionID, period, customDays)
		return nil, nil
	})
}

func (s *Service) importHistory(ctx context.Context, sessionID, period string, customDays int) {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil || !cfg.Enabled {
		return
	}

	if s.msgRepo == nil {
		logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Msg("msgRepo is nil, cannot import history")
		return
	}

	days := importPeriodToDays(period, customDays)
	if days <= 0 {
		logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Str("period", period).Msg("invalid import period")
		return
	}

	logger.Info().Str("component", "chatwoot").Str("session", sessionID).Str("period", period).Int("days", days).Msg("Starting history import")
	metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(0)

	rateTicker := time.NewTicker(100 * time.Millisecond)
	defer rateTicker.Stop()

	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	var totalProcessed int
	var failedCount int
	const chunkSize = 100

	for {
		select {
		case <-ctx.Done():
			logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Err(ctx.Err()).Msg("history import context cancelled")
			return
		default:
		}

		msgs, err := s.msgRepo.FindUnimportedHistory(ctx, sessionID, since, chunkSize, failedCount)
		if err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("failed to fetch unimported history")
			return
		}
		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
				logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Err(ctx.Err()).Msg("history import context cancelled during processing")
				return
			case <-rateTicker.C:
			}

			if err := s.importSingleMessage(ctx, cfg, &msg); err != nil {
				failedCount++
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("mid", msg.ID).Msg("failed to import history message")
			} else {
				if err := s.msgRepo.MarkImported(ctx, sessionID, msg.ID); err != nil {
					failedCount++
					logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("mid", msg.ID).Msg("failed to mark message as imported")
				}
			}

			totalProcessed++
			metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(float64(totalProcessed))
		}

		if len(msgs) < chunkSize {
			break
		}
	}

	metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(100)
	logger.Info().Str("component", "chatwoot").Str("session", sessionID).Int("processed", totalProcessed).Msg("history import complete")
}

func (s *Service) importSingleMessage(ctx context.Context, cfg *Config, msg *model.Message) error {
	if msg == nil || msg.ID == "" {
		return nil
	}

	chatJID := msg.ChatJID
	if strings.HasSuffix(chatJID, "@lid") {
		if s.jidResolver != nil {
			if pn := s.jidResolver.GetPNForLID(ctx, cfg.SessionID, chatJID); pn != "" {
				chatJID = pn + "@s.whatsapp.net"
			}
		}
	}

	contactName := ""
	if s.contactNameGetter != nil {
		contactName = s.contactNameGetter.GetContactName(ctx, cfg.SessionID, chatJID)
	}

	convID, err := s.findOrCreateConversation(ctx, cfg, chatJID, contactName)
	if err != nil {
		return fmt.Errorf("findOrCreateConversation: %w", err)
	}

	client := s.clientFn(cfg)
	sourceID := "WAID:" + msg.ID
	messageType := "outgoing"
	if !msg.FromMe {
		messageType = "incoming"
	}

	if msg.MediaURL != "" {
		if !strings.HasPrefix(msg.MediaURL, "http") && s.mediaPresigner != nil {
			url, err := s.mediaPresigner.GetPresignedURL(ctx, msg.MediaURL)
			if err != nil {
				return fmt.Errorf("resolve media URL from key: %w", err)
			}
			msg.MediaURL = url
		}
		return s.importMediaMessage(ctx, cfg, client, convID, msg, messageType, sourceID)
	}

	if msg.Body == "" {
		return nil
	}

	content := msg.Body
	if msg.MsgType == "text" {
		content = convertWAToCWMarkdown(content)
	}

	_, err = client.CreateMessage(ctx, convID, MessageReq{
		Content:     content,
		MessageType: messageType,
		SourceID:    sourceID,
	})
	if err != nil {
		return fmt.Errorf("CreateMessage: %w", err)
	}

	return nil
}

func (s *Service) importMediaMessage(ctx context.Context, _ *Config, client Client, convID int, msg *model.Message, messageType, sourceID string) error {
	mediaCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := httpGetWithContext(mediaCtx, s.httpClient, msg.MediaURL)
	if err != nil {
		return fmt.Errorf("download media from minio: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}()

	data, err := io.ReadAll(io.LimitReader(req.Body, maxMediaBytes+1))
	if err != nil {
		return fmt.Errorf("read media data: %w", err)
	}
	if int64(len(data)) > maxMediaBytes {
		return fmt.Errorf("media too large: %d bytes", len(data))
	}

	mimeType := msg.MediaType
	if mimeType == "" {
		mimeType, _ = DetectMIME("", data)
	}

	filename := msg.MsgType
	ext := mimeTypeToExt(mimeType)
	if ext != "" {
		filename += ext
	}

	caption := msg.Body

	_, err = client.CreateAttachment(ctx, convID, caption, filename, data, mimeType, messageType, sourceID, 0, nil)
	if err != nil {
		return fmt.Errorf("CreateAttachment: %w", err)
	}

	return nil
}

func importPeriodToDays(period string, customDays int) int {
	switch period {
	case "24h":
		return 1
	case "7d":
		return 7
	case "30d":
		return 30
	case "custom":
		return customDays
	}
	return 0
}
