package elodesk

import (
	"context"
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

	convID, contactSrcID, err := h.svc.findOrCreateConversation(ctx, cfg, chatJID, contactPushName)
	if err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create elodesk conversation")
		return err
	}

	text := extractText(data.Message)
	if text == "" {
		// Media / poll / reaction / etc. — MVP ignora silenciosamente.
		return nil
	}

	messageType := "outgoing"
	if !fromMe {
		messageType = "incoming"
	}

	msgReq := MessageReq{
		Content:     text,
		MessageType: messageType,
		SourceID:    sourceID,
	}

	client := h.svc.clientFn(cfg)

	// In group chats, the chat-level contact is the group itself; the actual
	// message author is the member identified by data.Info.Sender. Upsert that
	// member as a separate contact and pass its id as sender_contact_id so the
	// Elodesk UI can render the per-message sender avatar/name. Best-effort:
	// failures fall back to the chat-level contact (no sender attribution).
	isGroup := strings.HasSuffix(chatJID, "@g.us")
	memberJID := data.Info.Sender
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
			msgReq.SenderContactID = &id
		} else if err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("memberJID", memberJID).Msg("failed to upsert group member contact")
		}
	}

	out, err := client.CreateMessage(ctx, cfg.InboxIdentifier, contactSrcID, convID, msgReq)
	if err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", cfg.SessionID).Msg("Failed to create elodesk message")
		return err
	}

	if msgID != "" {
		_ = h.svc.msgRepo.UpdateElodeskRef(ctx, cfg.SessionID, msgID, out.ID, convID, out.SourceID)
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
	}
	return nil
}
