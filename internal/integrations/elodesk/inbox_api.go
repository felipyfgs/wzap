package elodesk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/model"
)

type apiInboxHandler struct {
	svc *Service
}

func newAPIInboxHandler(svc *Service) *apiInboxHandler {
	return &apiInboxHandler{svc: svc}
}

// HandleMessage faz a ponte WA → Elodesk:
//  1. prologue (parse + idempotência)
//  2. persiste mensagem no DB (se ainda não estiver)
//  3. upsert contact + conversation no elodesk
//  4. POST mensagem no elodesk via client.CreateMessage
//  5. atualiza refs elodesk_* no wz_messages
//
// Media, polls, reactions, buttons, list, etc. ficam como follow-up —
// caem em no-op silencioso (body vazio após extractText).
func (h *apiInboxHandler) HandleMessage(ctx context.Context, cfg *Config, payload []byte) error {
	res, skip := h.svc.inboxPrologue(ctx, cfg, payload)
	if skip {
		return nil
	}
	data := res.data
	chatJID := res.chatJID
	sourceID := res.sourceID

	pushName := data.Info.PushName
	fromMe := data.Info.IsFromMe
	msgID := data.Info.ID

	if msgID != "" {
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)

		msgBody := extractText(data.Message)
		msgType := detectMessageType(data.Message)
		msgToSave := &model.Message{
			ID:        msgID,
			SessionID: cfg.SessionID,
			ChatJID:   chatJID,
			SenderJID: data.Info.Sender,
			FromMe:    fromMe,
			MsgType:   msgType,
			Body:      msgBody,
			Timestamp: time.Now(),
			CreatedAt: time.Now(),
		}
		if err := h.svc.msgRepo.Save(ctx, msgToSave); err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("msgID", msgID).Msg("failed to save message to DB")
		}
	}

	contactPushName := pushName
	if fromMe {
		contactPushName = ""
	}

	convID, contactSrcID, convErr := h.svc.findOrCreateConversation(ctx, cfg, chatJID, contactPushName)
	if convErr != nil {
		logger.Warn().Str("component", "elodesk").Err(convErr).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create elodesk conversation")
		return convErr
	}

	text := extractText(data.Message)
	mediaMeta := extractMediaInfo(data.Message)

	// Sem texto e sem mídia suportada → polls/reactions/buttons, etc. Ignora
	// (silent no-op por ora).
	if text == "" && mediaMeta == nil {
		return nil
	}

	messageType := "outgoing"
	if !fromMe {
		messageType = "incoming"
	}

	client := h.svc.clientFn(cfg)

	// In group chats, the chat-level contact is the group itself; the actual
	// message author is the member identified by data.Info.Sender. Upsert that
	// member as a separate contact and pass its id as sender_contact_id so the
	// Elodesk UI can render the per-message sender avatar/name. Best-effort:
	// failures fall back to the chat-level contact (no sender attribution).
	isGroup := strings.HasSuffix(chatJID, "@g.us")
	memberJID := data.Info.Sender
	var senderContactID *int64
	if isGroup && !fromMe && memberJID != "" && memberJID != chatJID {
		memberName := pushName
		if memberName == "" {
			memberName = extractPhone(memberJID)
		}
		memberReq := UpsertContactReq{
			SourceID:    contactSourceIDFromJID(memberJID),
			Identifier:  memberJID,
			Name:        memberName,
			PhoneNumber: "+" + extractPhone(memberJID),
		}
		if c, err := client.UpsertContact(ctx, cfg.InboxIdentifier, memberReq); err == nil && c != nil && c.ID > 0 {
			id := int64(c.ID)
			senderContactID = &id
		} else if err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("memberJID", memberJID).Msg("failed to upsert group member contact")
		}
	}

	var out *Message
	var err error

	if mediaMeta != nil {
		out, err = h.uploadMedia(ctx, cfg, client, contactSrcID, convID, sourceID, messageType, text, mediaMeta)
		if err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Str("mediaType", mediaMeta.MediaType).Msg("Failed to upload media to elodesk")
			return err
		}
	} else {
		msgReq := MessageReq{
			Content:         text,
			MessageType:     messageType,
			SourceID:        sourceID,
			SenderContactID: senderContactID,
		}
		out, err = client.CreateMessage(ctx, cfg.InboxIdentifier, contactSrcID, convID, msgReq)
		if err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Msg("Failed to create elodesk message")
			return err
		}
	}

	if msgID != "" && out != nil {
		_ = h.svc.msgRepo.UpdateElodeskRef(ctx, cfg.SessionID, msgID, out.ID, convID, out.SourceID)
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
	}
	return nil
}

// uploadMedia baixa o blob do WhatsApp via mediaDownloader e encaminha pra
// elodesk via CreateAttachment (multipart). Para arquivos acima do limite
// (256 MB) cai em uma mensagem privada de aviso ao agente — espelha o que a
// integração com chatwoot faz.
func (h *apiInboxHandler) uploadMedia(
	ctx context.Context,
	cfg *Config,
	client Client,
	contactSrcID string,
	convID int64,
	sourceID, messageType, caption string,
	info *mediaInfo,
) (*Message, error) {
	if int64(info.FileLength) > maxMediaBytes {
		warn := fmt.Sprintf("⚠️ Arquivo muito grande (%d MB) para download (limite: 256 MB)", info.FileLength/(1024*1024))
		// Mensagem privada — não reflete pro contato, só sinaliza ao agente.
		return client.CreateMessage(ctx, cfg.InboxIdentifier, contactSrcID, convID, MessageReq{
			Content:     warn,
			MessageType: messageType,
			SourceID:    sourceID,
			Private:     true,
		})
	}
	if h.svc.mediaDownloader == nil {
		return nil, fmt.Errorf("media downloader não configurado")
	}

	timeout := time.Duration(cfg.MediaTimeout) * time.Second
	if cfg.MediaTimeout == 0 {
		timeout = 60 * time.Second
	}
	if info.FileLength > 10*1024*1024 {
		timeout = time.Duration(cfg.LargeTimeout) * time.Second
		if cfg.LargeTimeout == 0 {
			timeout = 5 * time.Minute
		}
	}
	mediaCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := h.svc.mediaDownloader.DownloadMediaByPath(
		mediaCtx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256,
		info.MediaKey, info.FileLength, info.MediaType,
	)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}

	mime := info.MimeType
	if mime == "" {
		mime = detectMIME(data)
	}

	filename := resolveFilename(info, mime)

	return client.CreateAttachment(ctx, cfg.InboxIdentifier, contactSrcID, convID,
		caption, filename, data, mime, messageType, sourceID, nil)
}
