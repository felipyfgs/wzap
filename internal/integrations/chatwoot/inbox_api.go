package chatwoot

import (
	"context"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
)

type apiInboxHandler struct {
	svc *Service
}

func newAPIInboxHandler(svc *Service) *apiInboxHandler {
	return &apiInboxHandler{svc: svc}
}

func (h *apiInboxHandler) HandleMessage(ctx context.Context, cfg *Config, payload []byte) error {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to parse message payload")
		return nil
	}

	logger.Debug().Str("component", "chatwoot").Str("chat", data.Info.Chat).Str("id", data.Info.ID).Bool("fromMe", data.Info.IsFromMe).Msg("processMessage (api)")

	chatJID := data.Info.Chat
	if chatJID == "" {
		logger.Warn().Str("component", "chatwoot").Msg("chatJID empty, skipping")
		return nil
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
		chatJID = h.svc.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.RecipientAlt)
	} else {
		chatJID = h.svc.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.SenderAlt, data.Info.RecipientAlt)
	}
	if strings.HasSuffix(chatJID, "@lid") {
		logger.Warn().Str("component", "chatwoot").Str("lid", chatJID).Str("session", cfg.SessionID).Msg("unresolvable LID chat, skipping")
		return nil
	}

	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		logger.Debug().Str("component", "chatwoot").Str("chat", chatJID).Msg("JID ignored, skipping")
		return nil
	}

	pushName := data.Info.PushName
	fromMe := data.Info.IsFromMe
	msgID := data.Info.ID
	sourceID := "WAID:" + msgID

	if msgID != "" {
		_, idemSpan := startSpan(ctx, "chatwoot.check_idempotency",
			spanAttrs(cfg.SessionID, "message", "inbound")...)
		isDup := h.svc.cache.GetIdempotent(ctx, cfg.SessionID, sourceID)
		if !isDup {
			if exists, err := h.svc.msgRepo.ExistsBySourceID(ctx, cfg.SessionID, sourceID); err == nil && exists {
				h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
				isDup = true
			}
		}
		idemSpan.End()
		if isDup {
			logger.Debug().Str("component", "chatwoot").Str("sourceID", sourceID).Msg("inbound duplicate, skipping")
			metrics.CWIdempotentDrops.WithLabelValues(cfg.SessionID).Inc()
			return nil
		}
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
			logger.Warn().Str("component", "chatwoot").Err(err).Str("msgID", msgID).Msg("failed to save message to DB")
		}
	}

	contactPushName := pushName
	if fromMe {
		contactPushName = ""
	}

	_, convSpan := startSpan(ctx, "chatwoot.find_or_create_conversation",
		spanAttrs(cfg.SessionID, "message", "inbound")...)
	convID, err := h.svc.findOrCreateConversation(ctx, cfg, chatJID, contactPushName)
	convSpan.End()
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create Chatwoot conversation")
		return err
	}

	msg := data.Message

	stanzaID := extractStanzaID(msg)
	quotedText := extractQuoteText(msg)
	var cwReplyID int
	if stanzaID != "" {
		cwReplyID = h.svc.resolveInboundReply(ctx, cfg.SessionID, chatJID, stanzaID, quotedText, int64(data.Info.Timestamp))
	}

	if pollMsg := getMapField(msg, "pollCreationMessage"); pollMsg != nil {
		h.svc.processPollCreation(ctx, cfg, convID, msgID, fromMe, pollMsg)
		return nil
	}
	if pollMsg := getMapField(msg, "pollCreationMessageV3"); pollMsg != nil {
		h.svc.processPollCreation(ctx, cfg, convID, msgID, fromMe, pollMsg)
		return nil
	}
	if pollUpdate := getMapField(msg, "pollUpdateMessage"); pollUpdate != nil {
		h.svc.processPollUpdate(ctx, cfg, pollUpdate)
		return nil
	}
	if reactMsg := getMapField(msg, "reactionMessage"); reactMsg != nil {
		h.svc.processReaction(ctx, cfg, convID, msgID, fromMe, reactMsg)
		return nil
	}
	if btnResp := getMapField(msg, "buttonsResponseMessage"); btnResp != nil {
		h.svc.processButtonResponse(ctx, cfg, convID, msgID, fromMe, msg, btnResp)
		return nil
	}
	if listResp := getMapField(msg, "listResponseMessage"); listResp != nil {
		h.svc.processListResponse(ctx, cfg, convID, msgID, fromMe, msg, listResp)
		return nil
	}
	if tmplReply := getMapField(msg, "templateButtonReplyMessage"); tmplReply != nil {
		h.svc.processTemplateReply(ctx, cfg, convID, msgID, fromMe, msg, tmplReply)
		return nil
	}
	if vonce := getMapField(msg, "viewOnceMessage"); vonce != nil {
		h.svc.processViewOnce(ctx, cfg, convID, msgID, fromMe, vonce, false, stanzaID, cwReplyID)
		return nil
	}
	if vonce := getMapField(msg, "viewOnceMessageV2"); vonce != nil {
		h.svc.processViewOnce(ctx, cfg, convID, msgID, fromMe, vonce, true, stanzaID, cwReplyID)
		return nil
	}
	if editMsg := getMapField(msg, "editedMessage"); editMsg != nil {
		h.svc.processEditedMessage(ctx, cfg, editMsg)
		return nil
	}
	if liveMsg := getMapField(msg, "liveLocationMessage"); liveMsg != nil {
		h.svc.processLiveLocation(ctx, cfg, convID, msgID, fromMe, liveMsg)
		return nil
	}

	mediaInfo := extractMediaInfo(msg)
	if mediaInfo != nil {
		if mediaInfo.MediaType == "sticker" {
			h.svc.processStickerMessage(ctx, cfg, convID, msgID, fromMe, mediaInfo, stanzaID, cwReplyID)
		} else {
			h.svc.processMediaMessage(ctx, cfg, convID, msgID, fromMe, msg, stanzaID, cwReplyID)
		}
		return nil
	}

	text := extractText(msg)
	logger.Debug().Str("component", "chatwoot").Str("text", text).Interface("msg", msg).Msg("extracted text")
	text = applyMessagePrefixes(msg, convertWAToCWMarkdown(text))

	if !fromMe && data.Info.IsGroup && cfg.SignMsg && pushName != "" {
		senderJID := h.svc.resolveLID(ctx, cfg.SessionID, data.Info.Sender, data.Info.SenderAlt)
		phone := extractPhone(senderJID)
		text = formatGroupContent(phone, pushName, text, fromMe)
	}

	if text == "" {
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

	if stanzaID != "" {
		if cwReplyID > 0 {
			msgReq.SourceReplyID = cwReplyID
		}
		ca := map[string]any{"reply_source_id": "WAID:" + stanzaID}
		if cwReplyID > 0 {
			ca["in_reply_to"] = cwReplyID
		}
		msgReq.ContentAttributes = ca
		logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Int("sourceReplyID", cwReplyID).Interface("contentAttributes", ca).Msg("set ContentAttributes for reply")
	}

	client := h.svc.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			h.svc.cache.DeleteConv(ctx, cfg.SessionID, chatJID)
			convID, err = h.svc.upsertConversation(ctx, cfg, chatJID, contactPushName)
			if err == nil {
				cwMsg, err = client.CreateMessage(ctx, convID, msgReq)
			}
		}
		if err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("Failed to create Chatwoot message")
			return err
		}
	}
	if msgID != "" {
		_ = h.svc.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
	}
	return nil
}
