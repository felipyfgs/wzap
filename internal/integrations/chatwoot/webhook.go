package chatwoot

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"wzap/internal/dto"
	"wzap/internal/logger"
)

func (s *Service) HandleIncomingWebhook(ctx context.Context, sessionID string, body dto.ChatwootWebhookPayload) error {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load chatwoot config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}

	if body.Message != nil && body.Message.MessageType == 1 {
		return s.handleOutgoingMessage(ctx, cfg, body)
	}

	if body.EventType == "message_updated" && body.Message != nil {
		if deleted, _ := body.Message.ContentAttributes["deleted"].(bool); deleted {
			return s.handleMessageUpdated(ctx, cfg, body)
		}
	}

	if body.EventType == "conversation_status_changed" && body.Conversation != nil {
		return s.handleConversationStatusChanged(ctx, cfg, body)
	}

	return nil
}

func (s *Service) handleOutgoingMessage(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Message == nil || body.Conversation == nil {
		return nil
	}

	sourceID := body.Message.SourceID
	if strings.HasPrefix(sourceID, "WAID:") {
		return nil
	}

	chatJID := body.Conversation.ContactInbox.SourceID
	if chatJID == "" {
		return nil
	}

	var replyTo *dto.ReplyContext
	if body.Message.ContentAttributes != nil {
		if inReplyTo, ok := body.Message.ContentAttributes["in_reply_to"].(float64); ok && inReplyTo > 0 {
			origMsg, err := s.msgRepo.FindByCWMessageID(ctx, cfg.SessionID, int(inReplyTo))
			if err == nil {
				replyTo = &dto.ReplyContext{MessageID: origMsg.ID}
			}
		}
	}

	if len(body.Message.Attachments) > 0 {
		for _, att := range body.Message.Attachments {
			if err := s.sendAttachmentToWhatsApp(ctx, cfg, chatJID, att.URL, body.Message.Content, att.FileType, replyTo); err != nil {
				logger.Warn().Err(err).Msg("Failed to send attachment from Chatwoot to WhatsApp")
			}
		}
		return nil
	}

	content := body.Message.Content
	content = convertCWToWAMarkdown(content)

	_, err := s.messageSvc.SendText(ctx, cfg.SessionID, dto.SendTextReq{
		Phone:   chatJID,
		Body:    content,
		ReplyTo: replyTo,
	})
	return err
}

func (s *Service) handleMessageUpdated(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Message == nil {
		return nil
	}

	cwMsgID := body.Message.ID
	msg, err := s.msgRepo.FindByCWMessageID(ctx, cfg.SessionID, cwMsgID)
	if err != nil {
		return nil
	}

	_, _ = s.messageSvc.DeleteMessage(ctx, cfg.SessionID, dto.DeleteMessageReq{
		Phone:     msg.ChatJID,
		MessageID: msg.ID,
	})

	if body.Conversation != nil && body.Conversation.Status == "resolved" && !cfg.ReopenConversation {
		cacheKey := cfg.SessionID + "+" + body.Conversation.ContactInbox.SourceID
		s.convCache.Delete(cacheKey)
	}

	return nil
}

func (s *Service) handleConversationStatusChanged(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Conversation == nil {
		return nil
	}

	if body.Conversation.Status == "resolved" && !cfg.ReopenConversation {
		sourceID := body.Conversation.ContactInbox.SourceID
		if sourceID != "" {
			cacheKey := cfg.SessionID + "+" + sourceID
			s.convCache.Delete(cacheKey)
		}
	}

	return nil
}

func (s *Service) sendAttachmentToWhatsApp(ctx context.Context, cfg *ChatwootConfig, chatJID, attachmentURL, caption, mimeType string, replyTo *dto.ReplyContext) error {
	resp, err := s.httpClient.Get(attachmentURL)
	if err != nil {
		return fmt.Errorf("download attachment: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read attachment: %w", err)
	}

	msgType := strings.Split(mimeType, "/")[0]
	mediaReq := dto.SendMediaReq{
		Phone:    chatJID,
		Caption:  caption,
		MimeType: mimeType,
		Base64:   base64.StdEncoding.EncodeToString(data),
		ReplyTo:  replyTo,
	}

	switch msgType {
	case "image":
		_, err = s.messageSvc.SendImage(ctx, cfg.SessionID, mediaReq)
	case "video":
		_, err = s.messageSvc.SendVideo(ctx, cfg.SessionID, mediaReq)
	case "audio":
		_, err = s.messageSvc.SendAudio(ctx, cfg.SessionID, mediaReq)
	default:
		_, err = s.messageSvc.SendDocument(ctx, cfg.SessionID, mediaReq)
	}

	return err
}
