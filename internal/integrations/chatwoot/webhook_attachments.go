package chatwoot

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
)

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

func (s *Service) sendAttachment(ctx context.Context, cfg *Config, chatJID, attachmentURL, caption, fileType string, replyTo *dto.ReplyContext) (string, error) {
	var timeout time.Duration
	if fileType == "video" {
		timeout = time.Duration(cfg.LargeTimeout) * time.Second
		if timeout == 0 {
			timeout = 300 * time.Second
		}
	} else {
		timeout = time.Duration(cfg.MediaTimeout) * time.Second
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
	mimeType, _ := DetectMIME(filename, data)

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
	if idx := strings.Index(mimeType, "/"); idx >= 0 {
		msgType = mimeType[:idx]
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

func (s *Service) sendVCardToWhatsApp(ctx context.Context, cfg *Config, chatJID, content string, _ *dto.ReplyContext) error {
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
			logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to send vCard to WhatsApp")
		}
	}
	return nil
}

func (s *Service) resolveOutboundReply(ctx context.Context, sessionID string, attrs map[string]any) *dto.ReplyContext {
	if attrs == nil {
		return nil
	}
	if extID, ok := attrs["reply_source_id"].(string); ok && strings.HasPrefix(extID, "WAID:") {
		waMsgID := strings.TrimPrefix(extID, "WAID:")
		if origMsg, err := s.msgRepo.FindByID(ctx, sessionID, waMsgID); err == nil {
			logger.Debug().Str("component", "chatwoot").Str("replyToMsgID", waMsgID).Str("participant", origMsg.SenderJID).Msg("found original message for reply via reply_source_id")
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

func signContent(content, senderName, delimiter string) string {
	if senderName == "" {
		return content
	}
	prefix := "*" + senderName + ":*"
	if strings.HasPrefix(content, prefix) {
		return content
	}
	if delimiter == "" {
		delimiter = "\n"
	}
	delimiter = strings.ReplaceAll(delimiter, `\n`, "\n")
	return prefix + delimiter + content
}

func (s *Service) markReadIfEnabled(ctx context.Context, cfg *Config, chatJID string) {
	if !cfg.MessageRead {
		return
	}
	lastMsg, err := s.msgRepo.FindLastReceived(ctx, cfg.SessionID, chatJID)
	if err != nil {
		return
	}
	_ = s.messageSvc.MarkRead(ctx, cfg.SessionID, dto.MarkReadReq{
		Phone:     lastMsg.ChatJID,
		MessageID: lastMsg.ID,
	})
}

func (s *Service) sendErrorToAgent(ctx context.Context, cfg *Config, convID int, sendErr error) {
	client := s.clientFn(cfg)
	errMsg := sendErr.Error()
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "EOF") {
		errMsg = "falha de conexão com o servidor WhatsApp"
	}
	content := fmt.Sprintf("_Mensagem não enviada: %s_", errMsg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     content,
		MessageType: "outgoing",
		Private:     true,
	})
}
