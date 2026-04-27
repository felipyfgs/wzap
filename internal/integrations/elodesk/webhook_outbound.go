package elodesk

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
)

// elodesk Attachment.fileType — mantém em sync com backend/internal/model.AttachmentFileType.
const (
	elodeskFileTypeImage    = 0
	elodeskFileTypeAudio    = 1
	elodeskFileTypeVideo    = 2
	elodeskFileTypeFile     = 3
	elodeskFileTypeLocation = 4
	elodeskFileTypeFallback = 5
)

// HandleIncomingWebhook processa o payload vindo do elodesk (webhook outbound
// Chatwoot-compat): roteia por event_type e ignora private notes e ecos.
func (s *Service) HandleIncomingWebhook(ctx context.Context, sessionID string, body dto.ElodeskWebhookPayload) error {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load elodesk config: %w", err)
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

	if eventType == "conversation_status_changed" && body.Conversation != nil {
		logger.Debug().Str("component", "elodesk").Str("session", sessionID).Int64("convID", body.Conversation.ID).Int("status", body.Conversation.Status).Msg("conversation_status_changed (no-op in MVP)")
		return nil
	}

	if eventType == "message_updated" && msg != nil {
		// Edit / delete no elodesk. No MVP, no-op — atualização bilateral
		// fica como follow-up.
		return nil
	}

	// MessageType: 0=Incoming, 1=Outgoing, 2=Activity, 3=Template
	if msg != nil && msg.MessageType == 1 {
		if s.isOutboundDuplicate(ctx, sessionID, msg) {
			return nil
		}
		return s.processOutgoingMessage(ctx, cfg, body)
	}

	return nil
}

// isOutboundDuplicate bloqueia o loop de eco: se o source_id já foi gravado
// como outbound, não re-envia ao WA.
func (s *Service) isOutboundDuplicate(ctx context.Context, sessionID string, msg *dto.ElodeskWebhookMessage) bool {
	if sourceID := msg.SourceID; sourceID != "" {
		if s.cache.GetIdempotent(ctx, sessionID, sourceID) {
			return true
		}
		if exists, err := s.msgRepo.ExistsByElodeskSrcID(ctx, sessionID, sourceID); err == nil && exists {
			return true
		}
	}
	if msg.ID > 0 {
		elIdemKey := fmt.Sprintf("el-out:%d", msg.ID)
		if s.cache.GetIdempotent(ctx, sessionID, elIdemKey) {
			return true
		}
		s.cache.SetIdempotent(ctx, sessionID, elIdemKey)
	}
	return false
}

func (s *Service) processOutgoingMessage(ctx context.Context, cfg *Config, body dto.ElodeskWebhookPayload) error {
	msg := body.GetMessage()
	if msg == nil || body.Conversation == nil {
		return nil
	}

	// Eco: o próprio wzap gravou source_id com prefixo "WAID:" ao postar uma
	// mensagem WA no elodesk; se o webhook nos trouxer isso de volta, é eco.
	if strings.HasPrefix(msg.SourceID, "WAID:") {
		return nil
	}

	conv := body.Conversation
	// O payload outbound do elodesk só traz ids — busca o chatJID que o wzap
	// associou a essa conv quando criou a conversation.
	chatJID, err := s.msgRepo.FindChatJIDByElodeskConvID(ctx, cfg.SessionID, conv.ID)
	if err != nil || chatJID == "" || !isValidWhatsAppJID(chatJID) {
		logger.Warn().Str("component", "elodesk").Int64("convID", conv.ID).Err(err).Msg("no valid chat JID found for outgoing message, skipping")
		return nil
	}

	if s.messageSvc == nil {
		return fmt.Errorf("messageSvc not wired")
	}

	if len(msg.Attachments) > 0 {
		return s.sendOutgoingMedia(ctx, cfg, chatJID, conv, msg)
	}

	content := msg.Content
	if content == "" {
		return nil
	}

	waMsgID, err := s.messageSvc.SendText(ctx, cfg.SessionID, dto.SendTextReq{
		Phone: chatJID,
		Body:  content,
	})
	if err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Msg("failed to send outgoing text to WA")
		return err
	}

	if waMsgID == "" {
		return nil
	}

	s.persistOutboundMirror(ctx, cfg.SessionID, waMsgID, chatJID, "text", content, msg.ID, conv.ID)
	return nil
}

// sendOutgoingMedia traduz o primeiro anexo do payload elodesk para a chamada
// Send* equivalente. O elodesk deve ter pré-populado `dataUrl` (presigned
// MinIO) — sem isso o wzap não tem como baixar a mídia. Caption usa
// msg.Content quando presente.
func (s *Service) sendOutgoingMedia(ctx context.Context, cfg *Config, chatJID string, conv *dto.ElodeskWebhookConversation, msg *dto.ElodeskWebhookMessage) error {
	att := msg.Attachments[0]

	if att.DataURL == "" {
		logger.Warn().Str("component", "elodesk").Str("session", cfg.SessionID).Int64("convID", conv.ID).Int64("attachmentId", att.ID).
			Msg("attachment without dataUrl — elodesk presigning missing, skipping")
		return nil
	}

	mimeType, fileName := resolveAttachmentMeta(att)

	req := dto.SendMediaReq{
		Phone:    chatJID,
		MimeType: mimeType,
		Caption:  msg.Content,
		FileName: fileName,
		URL:      att.DataURL,
	}

	var (
		waMsgID string
		err     error
		msgKind string
	)
	switch att.FileType {
	case elodeskFileTypeImage:
		msgKind = "image"
		waMsgID, err = s.messageSvc.SendImage(ctx, cfg.SessionID, req)
	case elodeskFileTypeAudio:
		msgKind = "audio"
		waMsgID, err = s.messageSvc.SendAudio(ctx, cfg.SessionID, req)
	case elodeskFileTypeVideo:
		msgKind = "video"
		waMsgID, err = s.messageSvc.SendVideo(ctx, cfg.SessionID, req)
	case elodeskFileTypeFile, elodeskFileTypeFallback:
		msgKind = "document"
		waMsgID, err = s.messageSvc.SendDocument(ctx, cfg.SessionID, req)
	case elodeskFileTypeLocation:
		logger.Warn().Str("component", "elodesk").Str("session", cfg.SessionID).Int64("convID", conv.ID).Msg("location attachment not supported by outbound webhook bridge")
		return nil
	default:
		logger.Warn().Str("component", "elodesk").Str("session", cfg.SessionID).Int64("convID", conv.ID).Int("fileType", att.FileType).Msg("unknown attachment fileType")
		return nil
	}

	if err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Str("kind", msgKind).Msg("failed to send outgoing media to WA")
		return err
	}
	if waMsgID == "" {
		return nil
	}

	s.persistOutboundMirror(ctx, cfg.SessionID, waMsgID, chatJID, msgKind, msg.Content, msg.ID, conv.ID)
	return nil
}

// resolveAttachmentMeta deriva mimeType e fileName a partir do que o elodesk
// envia. Extension é a fonte primária; sem ela, tenta extrair do FileKey.
// Para áudio/imagem/vídeo, força o prefixo do mime no fileType esperado —
// `mime.TypeByExtension(".webm")` retorna "video/webm", o que confunde o
// whatsmeow quando o anexo era um voice note.
func resolveAttachmentMeta(att dto.ElodeskWebhookAttachment) (mimeType, fileName string) {
	ext := ""
	if att.Extension != nil {
		ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(*att.Extension), "."))
	}
	if ext == "" && att.FileKey != nil {
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(*att.FileKey), "."))
	}

	if ext != "" {
		mimeType = mime.TypeByExtension("." + ext)
	}

	expectedPrefix := mimePrefixForFileType(att.FileType)
	if expectedPrefix != "" && !strings.HasPrefix(mimeType, expectedPrefix) {
		if ext != "" {
			mimeType = expectedPrefix + ext
		} else {
			mimeType = ""
		}
	}
	if mimeType == "" {
		mimeType = defaultMimeForFileType(att.FileType)
	}

	if att.FileKey != nil && *att.FileKey != "" {
		fileName = filepath.Base(*att.FileKey)
	} else if ext != "" {
		fileName = "attachment." + ext
	} else {
		fileName = "attachment"
	}
	return mimeType, fileName
}

// mimePrefixForFileType retorna o prefixo de mime esperado para o fileType.
// Vazio significa "qualquer mime serve" (ex.: file/fallback).
func mimePrefixForFileType(fileType int) string {
	switch fileType {
	case elodeskFileTypeImage:
		return "image/"
	case elodeskFileTypeAudio:
		return "audio/"
	case elodeskFileTypeVideo:
		return "video/"
	default:
		return ""
	}
}

func defaultMimeForFileType(fileType int) string {
	switch fileType {
	case elodeskFileTypeImage:
		return "image/jpeg"
	case elodeskFileTypeAudio:
		return "audio/ogg; codecs=opus"
	case elodeskFileTypeVideo:
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

// persistOutboundMirror grava o espelho do outbound em wz_messages + as refs
// elodesk pra bloquear o próximo eco.
func (s *Service) persistOutboundMirror(ctx context.Context, sessionID, waMsgID, chatJID, msgType, body string, elMsgID, elConvID int64) {
	_ = s.msgRepo.Save(ctx, &model.Message{
		ID:        waMsgID,
		SessionID: sessionID,
		ChatJID:   chatJID,
		FromMe:    true,
		MsgType:   msgType,
		Body:      body,
		Timestamp: time.Now(),
	})
	_ = s.msgRepo.UpdateElodeskRef(ctx, sessionID, waMsgID, elMsgID, elConvID, "WAID:"+waMsgID)
	s.cache.SetIdempotent(ctx, sessionID, "WAID:"+waMsgID)
}

func isValidWhatsAppJID(jid string) bool {
	return strings.HasSuffix(jid, "@s.whatsapp.net") ||
		strings.HasSuffix(jid, "@g.us") ||
		strings.HasSuffix(jid, "@lid") ||
		strings.HasSuffix(jid, "@broadcast")
}
