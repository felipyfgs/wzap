package chatwoot

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
)

var (
	googleMapsRegex       = regexp.MustCompile(`[?&]q=(-?\d+\.\d+),(-?\d+\.\d+)`)
	coordRegex            = regexp.MustCompile(`(-?\d+\.\d+),\s*(-?\d+\.\d+)`)
	maxMediaBytes   int64 = 256 * 1024 * 1024
)

func (s *Service) HandleIncomingWebhook(ctx context.Context, sessionID string, body dto.ChatwootWebhookPayload) error {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load chatwoot config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}

	if body.Private {
		return nil
	}

	msg := body.GetMessage()

	if msg != nil && msg.IsOutgoing() {
		sourceID := msg.SourceID
		if sourceID != "" {
			if exists, err := s.msgRepo.ExistsBySourceID(ctx, sessionID, sourceID); err == nil && exists {
				logger.Debug().Str("sourceID", sourceID).Msg("[CW] outbound already processed, skipping (idempotency)")
				metrics.CWIdempotentDrops.WithLabelValues(sessionID).Inc()
				return nil
			}
		}
		if msg.ID > 0 {
			cwIdemKey := fmt.Sprintf("cw-out:%d", msg.ID)
			if s.cache.GetIdempotent(ctx, sessionID, cwIdemKey) {
				logger.Debug().Int("cwMsgID", msg.ID).Msg("[CW] outbound already processed, skipping (CW msg ID idempotency)")
				metrics.CWIdempotentDrops.WithLabelValues(sessionID).Inc()
				return nil
			}
			s.cache.SetIdempotent(ctx, sessionID, cwIdemKey)
		}
		return s.handleOutgoingMessage(ctx, cfg, body)
	}

	eventType := body.EventType
	if eventType == "" {
		eventType = body.Event
	}

	if eventType == "message_updated" && msg != nil {
		if deleted, _ := msg.ContentAttributes["deleted"].(bool); deleted {
			return s.handleMessageUpdated(ctx, cfg, body)
		}
		return s.handleMessageEdited(ctx, cfg, body)
	}

	if eventType == "conversation_status_changed" && body.Conversation != nil {
		return s.handleConversationStatusChanged(ctx, cfg, body)
	}

	return nil
}

func (s *Service) handleOutgoingMessage(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	msg := body.GetMessage()
	if msg == nil || body.Conversation == nil {
		return nil
	}

	if strings.HasPrefix(msg.SourceID, "WAID:") {
		return nil
	}

	conv := body.Conversation
	chatJID := conv.ContactInbox.SourceID
	if chatJID == "" && conv.Meta.Sender.Identifier != "" {
		chatJID = conv.Meta.Sender.Identifier
	}
	if chatJID == "" && conv.Meta.Sender.PhoneNumber != "" {
		phone := strings.TrimPrefix(conv.Meta.Sender.PhoneNumber, "+")
		chatJID = phone + "@s.whatsapp.net"
	}
	if chatJID == "" {
		logger.Warn().Int("convID", conv.ID).Msg("[CW] no chat JID found for outgoing message")
		return nil
	}

	if strings.HasPrefix(chatJID, "bot@") {
		return s.handleBotCommand(ctx, cfg, msg.Content)
	}

	if !isValidWhatsAppJID(chatJID) {
		logger.Debug().Str("chatJID", chatJID).Msg("[CW] skipping outgoing message: invalid WhatsApp JID (bot conversation)")
		return nil
	}

	logger.Debug().Str("chatJID", chatJID).Str("content", msg.Content).Msg("[CW] sending outgoing message to WhatsApp")

	replyTo := s.resolveOutboundReply(ctx, cfg.SessionID, msg.ContentAttributes)

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
		for _, att := range msg.Attachments {
			attURL := att.DataURL
			if attURL == "" {
				attURL = att.URL
			}
			attURL = rewriteAttachmentURL(attURL, cfg.URL)
			waMsgID, err := s.sendAttachmentToWhatsApp(ctx, cfg, chatJID, attURL, msg.Content, att.FileType, replyTo)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to send attachment from Chatwoot to WhatsApp")
				continue
			}
			saveCWRef(waMsgID, "media", msg.Content)
		}
		return nil
	}

	content := msg.Content
	content = convertCWToWAMarkdown(content)

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
	if err == nil {
		saveCWRef(waMsgID, "text", content)
	}
	return err
}

func (s *Service) handleMessageEdited(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	msg := body.GetMessage()
	if msg == nil || msg.Content == "" {
		return nil
	}

	storedMsg, err := s.msgRepo.FindByCWMessageID(ctx, cfg.SessionID, msg.ID)
	if err != nil {
		return nil
	}

	_, err = s.messageSvc.EditMessage(ctx, cfg.SessionID, dto.EditMessageReq{
		Phone:     storedMsg.ChatJID,
		MessageID: storedMsg.ID,
		Body:      msg.Content,
	})
	return err
}

func extractLocationFromText(text string) (lat, lng float64, ok bool) {
	if m := googleMapsRegex.FindStringSubmatch(text); m != nil {
		la, err1 := strconv.ParseFloat(m[1], 64)
		ln, err2 := strconv.ParseFloat(m[2], 64)
		if err1 == nil && err2 == nil {
			return la, ln, true
		}
	}
	if m := coordRegex.FindStringSubmatch(text); m != nil {
		la, err1 := strconv.ParseFloat(m[1], 64)
		ln, err2 := strconv.ParseFloat(m[2], 64)
		if err1 == nil && err2 == nil {
			return la, ln, true
		}
	}
	return 0, 0, false
}

func isVCardContent(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "BEGIN:VCARD")
}

func (s *Service) sendVCardToWhatsApp(ctx context.Context, cfg *ChatwootConfig, chatJID, content string, replyTo *dto.ReplyContext) error {
	vcards := splitVCards(content)
	for _, vcard := range vcards {
		name := extractVCardName(vcard)
		if name == "" {
			name = "Contato"
		}
		if _, err := s.messageSvc.SendContact(ctx, cfg.SessionID, dto.SendContactReq{
			Phone: chatJID,
			Name:  name,
			Vcard: vcard,
		}); err != nil {
			logger.Warn().Err(err).Msg("[CW] failed to send vCard to WhatsApp")
		}
	}
	return nil
}

func splitVCards(content string) []string {
	var vcards []string
	lines := strings.Split(content, "\n")
	var current strings.Builder
	for _, line := range lines {
		current.WriteString(line)
		current.WriteString("\n")
		if strings.TrimSpace(line) == "END:VCARD" {
			vcards = append(vcards, current.String())
			current.Reset()
		}
	}
	return vcards
}

func extractVCardName(vcard string) string {
	for _, line := range strings.Split(vcard, "\n") {
		if strings.HasPrefix(line, "FN:") {
			return strings.TrimPrefix(strings.TrimSpace(line), "FN:")
		}
	}
	return ""
}

func (s *Service) handleMessageUpdated(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	webhookMsg := body.GetMessage()
	if webhookMsg == nil {
		return nil
	}

	cwMsgID := webhookMsg.ID
	storedMsg, err := s.msgRepo.FindByCWMessageID(ctx, cfg.SessionID, cwMsgID)
	if err != nil {
		return nil
	}

	_, _ = s.messageSvc.DeleteMessage(ctx, cfg.SessionID, dto.DeleteMessageReq{
		Phone:     storedMsg.ChatJID,
		MessageID: storedMsg.ID,
	})

	if body.Conversation != nil && body.Conversation.Status == "resolved" && !cfg.ReopenConversation {
		sourceID := body.Conversation.ContactInbox.SourceID
		if sourceID != "" {
			s.cache.DeleteConv(ctx, cfg.SessionID, sourceID)
		}
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
			s.cache.DeleteConv(ctx, cfg.SessionID, sourceID)
		}
	}

	return nil
}

func rewriteAttachmentURL(attURL, chatwootBaseURL string) string {
	if attURL == "" || chatwootBaseURL == "" {
		return attURL
	}

	parsed, err := url.Parse(attURL)
	if err != nil {
		return attURL
	}

	base, err := url.Parse(strings.TrimRight(chatwootBaseURL, "/"))
	if err != nil {
		return attURL
	}

	parsed.Scheme = base.Scheme
	parsed.Host = base.Host
	return parsed.String()
}

func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	segments := strings.Split(parsed.Path, "/")
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if seg != "" && strings.Contains(seg, ".") {
			decoded, err := url.PathUnescape(seg)
			if err != nil {
				return seg
			}
			return decoded
		}
	}
	return ""
}

func (s *Service) sendAttachmentToWhatsApp(ctx context.Context, cfg *ChatwootConfig, chatJID, attachmentURL, caption, fileType string, replyTo *dto.ReplyContext) (string, error) {
	var timeout time.Duration
	if fileType == "video" {
		timeout = time.Duration(cfg.TimeoutLargeSeconds) * time.Second
		if timeout == 0 {
			timeout = 300 * time.Second
		}
	} else {
		timeout = time.Duration(cfg.TimeoutMediaSeconds) * time.Second
		if timeout == 0 {
			timeout = 60 * time.Second
		}
	}
	dlCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := httpGetWithContext(dlCtx, s.httpClient, attachmentURL)
	if err != nil {
		return "", fmt.Errorf("download attachment: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}()

	if req.ContentLength > maxMediaBytes {
		return "", fmt.Errorf("attachment too large: %d bytes (max 256MB)", req.ContentLength)
	}

	data, err := io.ReadAll(io.LimitReader(req.Body, maxMediaBytes+1))
	if err != nil {
		return "", fmt.Errorf("read attachment: %w", err)
	}
	if int64(len(data)) > maxMediaBytes {
		return "", fmt.Errorf("attachment too large")
	}

	metrics.CWMediaDownloadBytes.WithLabelValues(cfg.SessionID, fileType).Add(float64(len(data)))

	filename := filenameFromURL(attachmentURL)
	mimeType, _ := GetMIMETypeAndExt(filename, data)

	if strings.HasSuffix(strings.ToLower(filename), ".webp") || mimeType == "image/webp" {
		waMsgID, err := s.messageSvc.SendDocument(ctx, cfg.SessionID, dto.SendMediaReq{
			Phone:    chatJID,
			Caption:  caption,
			MimeType: mimeType,
			FileName: filename,
			Base64:   base64.StdEncoding.EncodeToString(data),
			ReplyTo:  replyTo,
		})
		return waMsgID, err
	}

	msgType := fileType
	if strings.Contains(mimeType, "/") {
		msgType = strings.Split(mimeType, "/")[0]
	}

	mediaReq := dto.SendMediaReq{
		Phone:    chatJID,
		Caption:  caption,
		MimeType: mimeType,
		FileName: filename,
		Base64:   base64.StdEncoding.EncodeToString(data),
		ReplyTo:  replyTo,
	}

	var waMsgID string
	switch msgType {
	case "image":
		waMsgID, err = s.messageSvc.SendImage(ctx, cfg.SessionID, mediaReq)
	case "video":
		waMsgID, err = s.messageSvc.SendVideo(ctx, cfg.SessionID, mediaReq)
	case "audio":
		waMsgID, err = s.messageSvc.SendAudio(ctx, cfg.SessionID, mediaReq)
	default:
		waMsgID, err = s.messageSvc.SendDocument(ctx, cfg.SessionID, mediaReq)
	}

	return waMsgID, err
}

func (s *Service) resolveOutboundReply(ctx context.Context, sessionID string, attrs map[string]any) *dto.ReplyContext {
	if attrs == nil {
		return nil
	}
	if extID, ok := attrs["in_reply_to_external_id"].(string); ok && strings.HasPrefix(extID, "WAID:") {
		waMsgID := strings.TrimPrefix(extID, "WAID:")
		if origMsg, err := s.msgRepo.FindByID(ctx, sessionID, waMsgID); err == nil {
			logger.Debug().Str("replyToMsgID", waMsgID).Str("participant", origMsg.SenderJID).Msg("[CW] found original message for reply via external_id")
			return &dto.ReplyContext{MessageID: origMsg.ID, Participant: origMsg.SenderJID}
		}
	}
	if inReplyTo, ok := attrs["in_reply_to"].(float64); ok && inReplyTo > 0 {
		if origMsg, err := s.msgRepo.FindByCWMessageID(ctx, sessionID, int(inReplyTo)); err == nil {
			return &dto.ReplyContext{MessageID: origMsg.ID, Participant: origMsg.SenderJID}
		}
	}
	return nil
}
