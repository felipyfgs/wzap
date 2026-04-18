package chatwoot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
)

const maxMediaBytes int64 = 256 * 1024 * 1024

func (s *Service) HandleIncomingWebhook(ctx context.Context, sessionID string, body dto.CWWebhookPayload) error {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load chatwoot config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}

	msg := body.GetMessage()

	if body.Private || (msg != nil && msg.Private) {
		return nil
	}

	eventType := body.EventType
	if eventType == "" {
		eventType = body.Event
	}

	if eventType == "message_created" && msg != nil && s.syncCloudMessageRef(ctx, cfg, body) {
		return nil
	}

	if eventType == "message_updated" && msg != nil {
		if deleted, _ := msg.ContentAttributes["deleted"].(bool); deleted {
			return s.processMessageUpdated(ctx, cfg, body)
		}
		return s.processMessageEdited(ctx, cfg, body)
	}

	if eventType == "conversation_status_changed" && body.Conversation != nil {
		return s.processStatusChanged(ctx, cfg, body)
	}

	if msg != nil && msg.IsOutgoing() {
		if s.isOutboundDuplicate(ctx, sessionID, msg) {
			return nil
		}
		return s.processOutgoingMessage(ctx, cfg, body)
	}

	return nil
}

func (s *Service) syncCloudMessageRef(ctx context.Context, cfg *Config, body dto.CWWebhookPayload) bool {
	if cfg == nil || cfg.InboxType != "cloud" || body.Conversation == nil {
		return false
	}

	msg := body.GetMessage()
	if msg == nil || msg.ID == 0 || msg.SourceID == "" {
		return false
	}

	waMsgID := strings.TrimPrefix(msg.SourceID, "WAID:")
	storedMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, waMsgID)
	if err != nil || storedMsg == nil {
		return false
	}

	cwConvID := body.Conversation.ID
	if storedMsg.CWMessageID != nil && storedMsg.CWConvID != nil && storedMsg.CWSrcID != nil && *storedMsg.CWMessageID == msg.ID && *storedMsg.CWConvID == cwConvID && *storedMsg.CWSrcID == msg.SourceID {
		return true
	}

	if err := s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, waMsgID, msg.ID, cwConvID, msg.SourceID); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Int("cwMsgID", msg.ID).Int("cwConvID", cwConvID).Msg("failed to sync cloud chatwoot refs")
		return true
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("waMsgID", waMsgID).Int("cwMsgID", msg.ID).Int("cwConvID", cwConvID).Str("sourceID", msg.SourceID).Msg("synced cloud chatwoot refs from webhook")
	return true
}

func (s *Service) isOutboundDuplicate(ctx context.Context, sessionID string, msg *dto.CWWebhookMsg) bool {
	if sourceID := msg.SourceID; sourceID != "" {
		if exists, err := s.msgRepo.ExistsBySourceID(ctx, sessionID, sourceID); err == nil && exists {
			logger.Debug().Str("component", "chatwoot").Str("sourceID", sourceID).Msg("outbound already processed, skipping (idempotency)")
			metrics.CWIdempotentDrops.WithLabelValues(sessionID).Inc()
			return true
		}
	}
	if msg.ID > 0 {
		cwIdemKey := fmt.Sprintf("cw-out:%d", msg.ID)
		if s.cache.GetIdempotent(ctx, sessionID, cwIdemKey) {
			logger.Debug().Str("component", "chatwoot").Int("cwMsgID", msg.ID).Msg("outbound already processed, skipping (CW msg ID idempotency)")
			metrics.CWIdempotentDrops.WithLabelValues(sessionID).Inc()
			return true
		}
		s.cache.SetIdempotent(ctx, sessionID, cwIdemKey)
	}
	return false
}

func (s *Service) processOutgoingMessage(ctx context.Context, cfg *Config, body dto.CWWebhookPayload) error {
	msg := body.GetMessage()
	if msg == nil || body.Conversation == nil {
		return nil
	}

	if strings.HasPrefix(msg.SourceID, "WAID:") {
		return nil
	}

	conv := body.Conversation
	chatJID := conv.ContactInbox.SourceID
	if !isValidWhatsAppJID(chatJID) && conv.Meta.Sender.Identifier != "" {
		chatJID = conv.Meta.Sender.Identifier
	}
	if !isValidWhatsAppJID(chatJID) && conv.Meta.Sender.PhoneNumber != "" {
		phone := strings.TrimPrefix(conv.Meta.Sender.PhoneNumber, "+")
		chatJID = s.resolvePhoneToJID(ctx, cfg.SessionID, phone)
	}
	if chatJID == "" {
		logger.Warn().Str("component", "chatwoot").Int("convID", conv.ID).Msg("no chat JID found for outgoing message")
		return nil
	}

	if strings.HasPrefix(chatJID, "bot@") {
		return s.processBotCommand(ctx, cfg, msg.Content)
	}

	if !isValidWhatsAppJID(chatJID) {
		logger.Debug().Str("component", "chatwoot").Str("chatJID", chatJID).Msg("skipping outgoing message: invalid WhatsApp JID (bot conversation)")
		return nil
	}

	logger.Debug().Str("component", "chatwoot").Str("chatJID", chatJID).Str("content", msg.Content).Msg("sending outgoing message to WhatsApp")

	replyTo := s.resolveOutboundReply(ctx, cfg.SessionID, msg.ContentAttributes)

	var senderName string
	if cfg.SignMsg {
		if len(conv.Messages) > 0 && conv.Messages[0].Sender != nil {
			senderName = conv.Messages[0].Sender.AvailableName
		}
		if senderName == "" && msg.Sender != nil {
			senderName = msg.Sender.Name
		}
	}

	cwMsgID := msg.ID
	cwConvID := conv.ID
	saveCWRef := func(waMsgID, msgType, body string) {
		if waMsgID == "" {
			return
		}
		_ = s.msgRepo.Save(ctx, &model.Message{
			ID:        waMsgID,
			SessionID: cfg.SessionID,
			ChatJID:   chatJID,
			FromMe:    true,
			MsgType:   msgType,
			Body:      body,
			Timestamp: time.Now(),
		})
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, waMsgID, cwMsgID, cwConvID, "WAID:"+waMsgID)
	}

	if len(msg.Attachments) > 0 {
		firstCaption := signContent(msg.Content, senderName, cfg.SignDelimiter)
		for i, att := range msg.Attachments {
			attURL := att.DataURL
			if attURL == "" {
				attURL = att.URL
			}
			attURL = rewriteAttachmentURL(attURL, cfg.URL)
			caption := ""
			if i == 0 {
				caption = firstCaption
			}
			waMsgID, err := s.sendAttachment(ctx, cfg, chatJID, attURL, caption, att.FileType, replyTo)
			if err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to send attachment from Chatwoot to WhatsApp")
				s.sendErrorToAgent(ctx, cfg, cwConvID, err)
				continue
			}
			saveCWRef(waMsgID, "media", msg.Content)
		}
		s.markReadIfEnabled(ctx, cfg, chatJID)
		return nil
	}

	content := msg.Content
	content = convertCWToWAMarkdown(content)
	content = signContent(content, senderName, cfg.SignDelimiter)

	if lat, lng, ok := extractLocationFromText(content); ok {
		waMsgID, err := s.messageSvc.SendLocation(ctx, cfg.SessionID, dto.SendLocationReq{
			Phone:     chatJID,
			Latitude:  lat,
			Longitude: lng,
		})
		if err == nil {
			saveCWRef(waMsgID, "location", content)
		}
		return err
	}

	if isVCardContent(content) {
		return s.sendVCardToWhatsApp(ctx, cfg, chatJID, content, replyTo)
	}

	waMsgID, err := s.messageSvc.SendText(ctx, cfg.SessionID, dto.SendTextReq{
		Phone:   chatJID,
		Body:    content,
		ReplyTo: replyTo,
	})
	if err != nil {
		s.sendErrorToAgent(ctx, cfg, cwConvID, err)
		return err
	}
	saveCWRef(waMsgID, "text", content)
	s.markReadIfEnabled(ctx, cfg, chatJID)
	return nil
}

func (s *Service) processMessageEdited(ctx context.Context, cfg *Config, body dto.CWWebhookPayload) error {
	msg := body.GetMessage()
	if msg == nil || msg.Content == "" {
		return nil
	}

	storedMsgs, err := s.msgRepo.FindAllByCWMessageID(ctx, cfg.SessionID, msg.ID)
	if err != nil || len(storedMsgs) == 0 {
		logger.Warn().Str("component", "chatwoot").Err(err).Int("cwMsgID", msg.ID).Msg("processMessageEdited: message not found in store")
		return nil
	}

	for _, storedMsg := range storedMsgs {
		if _, err := s.messageSvc.EditMessage(ctx, cfg.SessionID, dto.EditMessageReq{
			Phone:     storedMsg.ChatJID,
			MessageID: storedMsg.ID,
			Body:      msg.Content,
		}); err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("msgID", storedMsg.ID).Msg("processMessageEdited: failed to edit WA message")
		}
	}
	return nil
}

func (s *Service) processMessageUpdated(ctx context.Context, cfg *Config, body dto.CWWebhookPayload) error {
	webhookMsg := body.GetMessage()
	if webhookMsg == nil {
		return nil
	}

	cwMsgID := webhookMsg.ID
	storedMsgs, err := s.msgRepo.FindAllByCWMessageID(ctx, cfg.SessionID, cwMsgID)
	if err != nil || len(storedMsgs) == 0 {
		logger.Warn().Str("component", "chatwoot").Err(err).Int("cwMsgID", cwMsgID).Msg("processMessageUpdated: message not found in store")
		return nil
	}

	for _, storedMsg := range storedMsgs {
		_, _ = s.messageSvc.DeleteMessage(ctx, cfg.SessionID, dto.DeleteMessageReq{
			Phone:     storedMsg.ChatJID,
			MessageID: storedMsg.ID,
		})
	}

	if body.Conversation != nil && body.Conversation.Status == "resolved" {
		sourceID := body.Conversation.ContactInbox.SourceID
		if sourceID != "" {
			s.cache.DeleteConv(ctx, cfg.SessionID, sourceID)
		}
	}

	return nil
}

func (s *Service) processStatusChanged(ctx context.Context, cfg *Config, body dto.CWWebhookPayload) error {
	if body.Conversation == nil {
		return nil
	}

	if body.Conversation.Status == "resolved" {
		sourceID := body.Conversation.ContactInbox.SourceID
		if sourceID != "" {
			s.cache.DeleteConv(ctx, cfg.SessionID, sourceID)
		}
	}

	return nil
}
