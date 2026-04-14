package chatwoot

import (
	"bytes"
	"context"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	gocolor "image/color"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
)

func (s *Service) handleMessage(ctx context.Context, cfg *Config, payload []byte) error {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to parse message payload")
		return nil
	}

	logger.Debug().Str("component", "chatwoot").Str("chat", data.Info.Chat).Str("id", data.Info.ID).Bool("fromMe", data.Info.IsFromMe).Msg("handleMessage")

	chatJID := data.Info.Chat
	if chatJID == "" {
		logger.Warn().Str("component", "chatwoot").Msg("chatJID empty, skipping")
		return nil
	}

	if strings.HasSuffix(chatJID, "@lid") {
		resolved := false
		if s.jidResolver != nil {
			if pn := s.jidResolver.GetPNForLID(ctx, cfg.SessionID, chatJID); pn != "" {
				logger.Debug().Str("component", "chatwoot").Str("lid", chatJID).Str("pn", pn).Msg("resolved LID to PN via store")
				chatJID = pn + "@s.whatsapp.net"
				resolved = true
			}
		}
		if !resolved {
			var altJID string
			if !data.Info.IsFromMe {
				altJID = data.Info.SenderAlt
			} else {
				altJID = data.Info.RecipientAlt
			}
			if altJID != "" && !strings.HasSuffix(altJID, "@lid") {
				logger.Debug().Str("component", "chatwoot").Str("lid", chatJID).Str("alt", altJID).Msg("resolved LID to PN via message alt JID")
				chatJID = altJID
				if !strings.Contains(chatJID, "@") {
					chatJID += "@s.whatsapp.net"
				}
				resolved = true
			}
		}
		if !resolved {
			logger.Warn().Str("component", "chatwoot").Str("lid", chatJID).Str("session", cfg.SessionID).Msg("unresolvable LID chat, skipping")
			return nil
		}
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
		isDup := s.cache.GetIdempotent(ctx, cfg.SessionID, sourceID)
		if !isDup {
			if exists, err := s.msgRepo.ExistsBySourceID(ctx, cfg.SessionID, sourceID); err == nil && exists {
				s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
				isDup = true
			}
		}
		idemSpan.End()
		if isDup {
			logger.Debug().Str("component", "chatwoot").Str("sourceID", sourceID).Msg("inbound duplicate, skipping")
			metrics.CWIdempotentDrops.WithLabelValues(cfg.SessionID).Inc()
			return nil
		}
		s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)

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
		if err := s.msgRepo.Save(ctx, msgToSave); err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("msgID", msgID).Msg("failed to save message to DB")
		}
	}

	contactPushName := pushName
	if fromMe {
		contactPushName = ""
	}

	_, convSpan := startSpan(ctx, "chatwoot.find_or_create_conversation",
		spanAttrs(cfg.SessionID, "message", "inbound")...)
	convID, err := s.findOrCreateConversation(ctx, cfg, chatJID, contactPushName)
	convSpan.End()
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create Chatwoot conversation")
		return err
	}

	msg := data.Message

	// Extract reply context once — reused by all message type handlers
	stanzaID := extractStanzaID(msg)
	quotedText := extractQuoteText(msg)
	var cwReplyID int
	if stanzaID != "" {
		cwReplyID = s.resolveInboundReply(ctx, cfg.SessionID, chatJID, stanzaID, quotedText, int64(data.Info.Timestamp))
	}

	if pollMsg := getMapField(msg, "pollCreationMessage"); pollMsg != nil {
		s.handlePollCreation(ctx, cfg, convID, msgID, fromMe, pollMsg)
		return nil
	}
	if pollMsg := getMapField(msg, "pollCreationMessageV3"); pollMsg != nil {
		s.handlePollCreation(ctx, cfg, convID, msgID, fromMe, pollMsg)
		return nil
	}
	if pollUpdate := getMapField(msg, "pollUpdateMessage"); pollUpdate != nil {
		s.handlePollUpdate(ctx, cfg, pollUpdate)
		return nil
	}
	if reactMsg := getMapField(msg, "reactionMessage"); reactMsg != nil {
		s.handleReaction(ctx, cfg, convID, msgID, fromMe, reactMsg)
		return nil
	}
	if btnResp := getMapField(msg, "buttonsResponseMessage"); btnResp != nil {
		s.handleButtonResponse(ctx, cfg, convID, msgID, fromMe, msg, btnResp)
		return nil
	}
	if listResp := getMapField(msg, "listResponseMessage"); listResp != nil {
		s.handleListResponse(ctx, cfg, convID, msgID, fromMe, msg, listResp)
		return nil
	}
	if tmplReply := getMapField(msg, "templateButtonReplyMessage"); tmplReply != nil {
		s.handleTemplateReply(ctx, cfg, convID, msgID, fromMe, msg, tmplReply)
		return nil
	}
	if vonce := getMapField(msg, "viewOnceMessage"); vonce != nil {
		s.handleViewOnce(ctx, cfg, convID, msgID, fromMe, vonce, false, stanzaID, cwReplyID)
		return nil
	}
	if vonce := getMapField(msg, "viewOnceMessageV2"); vonce != nil {
		s.handleViewOnce(ctx, cfg, convID, msgID, fromMe, vonce, true, stanzaID, cwReplyID)
		return nil
	}
	if editMsg := getMapField(msg, "editedMessage"); editMsg != nil {
		s.handleEditedMessage(ctx, cfg, editMsg)
		return nil
	}
	if liveMsg := getMapField(msg, "liveLocationMessage"); liveMsg != nil {
		s.handleLiveLocation(ctx, cfg, convID, msgID, fromMe, liveMsg)
		return nil
	}

	mediaInfo := extractMediaInfo(msg)
	if mediaInfo != nil {
		if mediaInfo.MediaType == "sticker" {
			s.handleStickerMessage(ctx, cfg, convID, msgID, fromMe, mediaInfo, stanzaID, cwReplyID)
		} else {
			s.handleMediaMessage(ctx, cfg, convID, msgID, fromMe, msg, stanzaID, cwReplyID)
		}
		return nil
	}

	text := extractText(msg)
	logger.Debug().Str("component", "chatwoot").Str("text", text).Interface("msg", msg).Msg("extracted text")
	text = applyMessagePrefixes(msg, convertWAToCWMarkdown(text))

	if !fromMe && data.Info.IsGroup && cfg.SignMsg && pushName != "" {
		senderJID := data.Info.Sender
		if strings.HasSuffix(senderJID, "@lid") {
			if data.Info.SenderAlt != "" {
				senderJID = data.Info.SenderAlt
			} else {
				senderJID = s.resolveJID(ctx, cfg.SessionID, senderJID)
			}
		}
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

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			s.cache.DeleteConv(ctx, cfg.SessionID, chatJID)
			convID, err = s.upsertConversation(ctx, cfg, chatJID, contactPushName)
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
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
		s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
	}
	return nil
}

func hasCWMessageID(msg *model.Message) bool {
	return msg != nil && msg.CWMessageID != nil && *msg.CWMessageID != 0
}

func (s *Service) resolveInboundReply(ctx context.Context, sessionID, chatJID, stanzaID, quotedText string, timestamp int64) int {
	logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Str("quotedText", quotedText).Msg("stanzaID found")

	if msg, err := s.msgRepo.FindByID(ctx, sessionID, stanzaID); err == nil && hasCWMessageID(msg) {
		logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("found CW message ID via FindByID")
		return *msg.CWMessageID
	}

	if msg, err := s.msgRepo.FindBySourceID(ctx, sessionID, "WAID:"+stanzaID); err == nil && hasCWMessageID(msg) {
		logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("found CW message ID via FindBySourceID")
		return *msg.CWMessageID
	}

	if quotedText != "" {
		if msg, err := s.msgRepo.FindByBodyAndChat(ctx, sessionID, chatJID, quotedText, true); err == nil && hasCWMessageID(msg) {
			logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("found CW message ID via FindByBodyAndChat")
			return *msg.CWMessageID
		}
	}

	if timestamp > 0 {
		if msg, err := s.msgRepo.FindByTimestampWindow(ctx, sessionID, chatJID, timestamp, 60); err == nil && hasCWMessageID(msg) {
			logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("found CW message ID via FindByTimestampWindow")
			return *msg.CWMessageID
		}
	}

	logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Msg("could not resolve CW message ID for reply")
	return 0
}

func (s *Service) handleMediaMessage(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, stanzaID string, cwReplyID int) {
	info := extractMediaInfo(msg)
	if info == nil {
		logger.Warn().Str("component", "chatwoot").Msg("no media info found in message")
		return
	}

	if int64(info.FileLength) > maxMediaBytes {
		client := s.clientFn(cfg)
		messageType := "incoming"
		if fromMe {
			messageType = "outgoing"
		}
		warnMsg := fmt.Sprintf("⚠️ Arquivo muito grande (%d MB) para download (limite: 256 MB)", info.FileLength/(1024*1024))
		_, _ = client.CreateMessage(ctx, convID, MessageReq{Content: warnMsg, MessageType: messageType, Private: true})
		return
	}

	if s.mediaDownloader == nil {
		logger.Warn().Str("component", "chatwoot").Msg("media downloader not configured, cannot download WhatsApp media")
		return
	}

	timeout := time.Duration(cfg.TimeoutMediaSeconds) * time.Second
	if cfg.TimeoutMediaSeconds == 0 {
		timeout = 60 * time.Second
	}
	if info.FileLength > 10*1024*1024 {
		timeout = time.Duration(cfg.TimeoutLargeSeconds) * time.Second
		if cfg.TimeoutLargeSeconds == 0 {
			timeout = 300 * time.Second
		}
	}
	mediaCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := s.mediaDownloader.DownloadMediaByPath(mediaCtx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256, info.MediaKey, info.FileLength, info.MediaType)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("mediaType", info.MediaType).Msg("failed to download WhatsApp media")
		return
	}

	metrics.CWMediaDownloadBytes.WithLabelValues(cfg.SessionID, info.MediaType).Add(float64(len(data)))

	mimeType := info.MimeType
	if mimeType == "" {
		mimeType, _ = DetectMIME("", data)
	}

	filename := info.FileName
	if filename == "" {
		ext := mimeTypeToExt(mimeType)
		filename = info.MediaType + ext
	}

	caption := extractText(msg)
	client := s.clientFn(cfg)
	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}
	sourceID := ""
	if msgID != "" {
		sourceID = "WAID:" + msgID
	}

	if isGIF(msg) && caption == "" {
		caption = "[GIF]"
	}

	var contentAttrs map[string]any
	if stanzaID != "" {
		contentAttrs = map[string]any{"reply_source_id": "WAID:" + stanzaID}
		if cwReplyID > 0 {
			contentAttrs["in_reply_to"] = cwReplyID
		}
	}

	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, caption, filename, data, mimeType, messageType, sourceID, cwReplyID, contentAttrs)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to upload media to Chatwoot")
		return
	}

	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleStickerMessage(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, info *mediaInfo, stanzaID string, cwReplyID int) {
	if s.mediaDownloader == nil {
		return
	}

	data, err := s.mediaDownloader.DownloadMediaByPath(ctx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256, info.MediaKey, info.FileLength, "sticker")
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to download sticker")
		return
	}

	metrics.CWMediaDownloadBytes.WithLabelValues(cfg.SessionID, "sticker").Add(float64(len(data)))

	client := s.clientFn(cfg)
	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}
	sourceID := ""
	if msgID != "" {
		sourceID = "WAID:" + msgID
	}

	var contentAttrs map[string]any
	if stanzaID != "" {
		contentAttrs = map[string]any{"reply_source_id": "WAID:" + stanzaID}
		if cwReplyID > 0 {
			contentAttrs["in_reply_to"] = cwReplyID
		}
	}

	if len(data) > 0 && len(data) <= 1024*1024 {
		pngData, err := convertWebPToPNG(data)
		if err == nil {
			cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.png", pngData, "image/png", messageType, sourceID, cwReplyID, contentAttrs)
			if err == nil && msgID != "" {
				_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
				return
			}
		}
		gifData, err := convertWebPToGIF(data)
		if err == nil {
			cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.gif", gifData, "image/gif", messageType, sourceID, cwReplyID, contentAttrs)
			if err == nil && msgID != "" {
				_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
				return
			}
		}
	}

	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.webp", data, "image/webp", messageType, sourceID, cwReplyID, contentAttrs)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to upload sticker fallback")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func convertWebPToPNG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func convertWebPToGIF(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	g := &gif.GIF{
		Image: []*image.Paletted{imageToPaletted(img)},
		Delay: []int{10},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	pm := image.NewPaletted(bounds, buildPalette(img, bounds))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pm.Set(x, y, img.At(x, y))
		}
	}
	return pm
}

type rgbaKey struct{ R, G, B, A uint8 }

func buildPalette(img image.Image, bounds image.Rectangle) gocolor.Palette {
	seen := make(map[rgbaKey]struct{}, 256)
	palette := make(gocolor.Palette, 0, 256)
	for y := bounds.Min.Y; y < bounds.Max.Y && len(palette) < 256; y++ {
		for x := bounds.Min.X; x < bounds.Max.X && len(palette) < 256; x++ {
			r32, g32, b32, a32 := img.At(x, y).RGBA()
			k := rgbaKey{uint8(r32 >> 8), uint8(g32 >> 8), uint8(b32 >> 8), uint8(a32 >> 8)}
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				palette = append(palette, gocolor.RGBA{R: k.R, G: k.G, B: k.B, A: k.A})
			}
		}
	}
	for len(palette) < 256 {
		palette = append(palette, gocolor.RGBA{})
	}
	return palette
}

func (s *Service) handlePollCreation(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, poll map[string]any) {
	name := getStringField(poll, "name")
	optionsRaw, _ := poll["options"].([]any)
	var sb strings.Builder
	sb.WriteString("📊 *Enquete:* ")
	sb.WriteString(name)
	sb.WriteString("\n")
	for i, opt := range optionsRaw {
		if optMap, ok := opt.(map[string]any); ok {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, getStringField(optMap, "optionName"))
		}
	}

	client := s.clientFn(cfg)
	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}
	sourceID := ""
	if msgID != "" {
		sourceID = "WAID:" + msgID
	}
	cwMsg, err := client.CreateMessage(ctx, convID, MessageReq{
		Content:     sb.String(),
		MessageType: messageType,
		SourceID:    sourceID,
	})
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to create poll message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handlePollUpdate(ctx context.Context, cfg *Config, pollUpdate map[string]any) {
	pollCreation := getMapField(pollUpdate, "pollCreationMessageKey")
	if pollCreation == nil {
		return
	}
	pollMsgID := getStringField(pollCreation, "ID")
	if pollMsgID == "" {
		return
	}

	origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, pollMsgID)
	if err != nil || origMsg.CWMessageID == nil || origMsg.CWConversationID == nil {
		return
	}

	votes := getMapField(pollUpdate, "vote")
	if votes == nil {
		return
	}
	selectedOpts, _ := votes["selectedOptions"].([]any)
	var sb strings.Builder
	sb.WriteString("📊 *Voto registrado:*\n")
	for _, opt := range selectedOpts {
		if optMap, ok := opt.(map[string]any); ok {
			fmt.Fprintf(&sb, "✅ %s\n", getStringField(optMap, "optionName"))
		}
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, *origMsg.CWConversationID, MessageReq{
		Content:           sb.String(),
		MessageType:       "incoming",
		SourceReplyID:     *origMsg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *origMsg.CWMessageID, "reply_source_id": "WAID:" + pollMsgID},
	})
}

func (s *Service) handleReaction(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, reactMsg map[string]any) {
	key := getMapField(reactMsg, "key")
	if key == nil {
		return
	}
	targetMsgID := getStringField(key, "ID")
	emoji := getStringField(reactMsg, "text")

	origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, targetMsgID)
	if err != nil || origMsg.CWMessageID == nil {
		logger.Warn().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("targetMsgID", targetMsgID).Str("emoji", emoji).Msg("reaction target message not found in DB, skipping")
		return
	}

	client := s.clientFn(cfg)
	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}

	if emoji == "" {
		if origMsg.CWConversationID != nil {
			cwReact, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
			if err == nil && cwReact.CWMessageID != nil && cwReact.CWConversationID != nil {
				_ = client.DeleteMessage(ctx, *cwReact.CWConversationID, *cwReact.CWMessageID)
			}
		}
		return
	}

	cwMsg, err := client.CreateMessage(ctx, convID, MessageReq{
		Content:           emoji,
		MessageType:       messageType,
		Private:           true,
		SourceReplyID:     *origMsg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *origMsg.CWMessageID, "reply_source_id": "WAID:" + targetMsgID},
	})
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to create reaction message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleButtonResponse(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, btnResp map[string]any) {
	text := getStringField(btnResp, "selectedDisplayText")
	if text == "" {
		text = getStringField(btnResp, "selectedButtonId")
	}
	content := fmt.Sprintf("[Botão] %s", text)

	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}

	msgReq := MessageReq{Content: content, MessageType: messageType, SourceID: "WAID:" + msgID}
	if stanzaID := extractStanzaID(msg); stanzaID != "" {
		if origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, stanzaID); err == nil && origMsg.CWMessageID != nil {
			msgReq.SourceReplyID = *origMsg.CWMessageID
			msgReq.ContentAttributes = map[string]any{
				"in_reply_to":     *origMsg.CWMessageID,
				"reply_source_id": "WAID:" + stanzaID,
			}
		}
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to create button response message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleListResponse(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, listResp map[string]any) {
	selection := getMapField(listResp, "singleSelectReply")
	title := ""
	description := ""
	if selection != nil {
		title = getStringField(selection, "title")
		description = getStringField(selection, "description")
	}
	if title == "" {
		title = getStringField(listResp, "title")
	}

	content := fmt.Sprintf("[Lista] %s", title)
	if description != "" {
		content += ": " + description
	}

	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}

	msgReq := MessageReq{Content: content, MessageType: messageType, SourceID: "WAID:" + msgID}
	if stanzaID := extractStanzaID(msg); stanzaID != "" {
		if origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, stanzaID); err == nil && origMsg.CWMessageID != nil {
			msgReq.SourceReplyID = *origMsg.CWMessageID
			msgReq.ContentAttributes = map[string]any{
				"in_reply_to":     *origMsg.CWMessageID,
				"reply_source_id": "WAID:" + stanzaID,
			}
		}
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to create list response message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleTemplateReply(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, tmpl map[string]any) {
	text := getStringField(tmpl, "selectedDisplayText")
	content := fmt.Sprintf("[Template] %s", text)

	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}

	msgReq := MessageReq{Content: content, MessageType: messageType, SourceID: "WAID:" + msgID}
	if stanzaID := extractStanzaID(msg); stanzaID != "" {
		if origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, stanzaID); err == nil && origMsg.CWMessageID != nil {
			msgReq.SourceReplyID = *origMsg.CWMessageID
			msgReq.ContentAttributes = map[string]any{
				"in_reply_to":     *origMsg.CWMessageID,
				"reply_source_id": "WAID:" + stanzaID,
			}
		}
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to create template reply message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleViewOnce(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, vonce map[string]any, tryDownload bool, stanzaID string, cwReplyID int) {
	client := s.clientFn(cfg)
	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}
	sourceID := ""
	if msgID != "" {
		sourceID = "WAID:" + msgID
	}

	if tryDownload {
		if innerMsg := getMapField(vonce, "message"); innerMsg != nil && s.mediaDownloader != nil {
			info := extractMediaInfo(innerMsg)
			if info != nil {
				timeout := time.Duration(cfg.TimeoutMediaSeconds) * time.Second
				if cfg.TimeoutMediaSeconds == 0 {
					timeout = 60 * time.Second
				}
				mediaCtx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				data, err := s.mediaDownloader.DownloadMediaByPath(mediaCtx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256, info.MediaKey, info.FileLength, info.MediaType)
				if err == nil && len(data) > 0 {
					mimeType := info.MimeType
					if mimeType == "" {
						mimeType, _ = DetectMIME("", data)
					}
					filename := info.FileName
					if filename == "" {
						ext := mimeTypeToExt(mimeType)
						filename = info.MediaType + ext
					}

					var contentAttrs map[string]any
					if stanzaID != "" {
						contentAttrs = map[string]any{"reply_source_id": "WAID:" + stanzaID}
						if cwReplyID > 0 {
							contentAttrs["in_reply_to"] = cwReplyID
						}
					}
					cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", filename, data, mimeType, messageType, sourceID, cwReplyID, contentAttrs)
					if err == nil {
						if msgID != "" {
							_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
						}
						return
					}
					logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to upload viewOnce media, falling back to text")
				} else if err != nil {
					logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to download viewOnce media, falling back to text")
				}
			}
		}
	}

	cwMsg, err := client.CreateMessage(ctx, convID, MessageReq{
		Content:     "[mensagem vista uma vez]",
		MessageType: messageType,
		SourceID:    sourceID,
	})
	if err != nil {
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleEditedMessage(ctx context.Context, cfg *Config, editMsg map[string]any) {
	key := getMapField(editMsg, "key")
	if key == nil {
		return
	}
	targetMsgID := getStringField(key, "ID")
	if targetMsgID == "" {
		return
	}

	origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, targetMsgID)
	if err != nil || origMsg.CWMessageID == nil || origMsg.CWConversationID == nil {
		return
	}

	newText := ""
	if inner := getMapField(editMsg, "message"); inner != nil {
		newText = extractText(inner)
	}
	if newText == "" {
		return
	}

	client := s.clientFn(cfg)

	messageType := "incoming"
	if origMsg.FromMe {
		messageType = "outgoing"
	}

	_, _ = client.CreateMessage(ctx, *origMsg.CWConversationID, MessageReq{
		Content:           "✏️ *Mensagem editada:*\n" + newText,
		MessageType:       messageType,
		Private:           true,
		SourceReplyID:     *origMsg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *origMsg.CWMessageID},
	})
}

func (s *Service) handleLiveLocation(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, liveMsg map[string]any) {
	text := formatLocation(liveMsg)

	client := s.clientFn(cfg)
	messageType := "incoming"
	if fromMe {
		messageType = "outgoing"
	}
	sourceID := ""
	if msgID != "" {
		sourceID = "WAID:" + msgID
	}
	cwMsg, err := client.CreateMessage(ctx, convID, MessageReq{
		Content:     text,
		MessageType: messageType,
		SourceID:    sourceID,
	})
	if err != nil {
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func applyMessagePrefixes(msg map[string]any, text string) string {
	if msg == nil {
		return text
	}

	var prefixes []string

	if ci := extractContextInfo(msg); ci != nil {
		if isForwarded, _ := ci["isForwarded"].(bool); isForwarded {
			score := getFloatField(ci, "forwardingScore")
			if score >= 5 {
				prefixes = append(prefixes, "[Encaminhada várias vezes]")
			} else {
				prefixes = append(prefixes, "[Encaminhada]")
			}
		}
	}

	if isEphemeral(msg) {
		prefixes = append(prefixes, "[mensagem temporária]")
	}

	if len(prefixes) == 0 {
		return text
	}

	return strings.Join(prefixes, " ") + " " + text
}

func extractContextInfo(msg map[string]any) map[string]any {
	if ci := getMapField(msg, "contextInfo"); ci != nil {
		return ci
	}
	for _, key := range []string{"extendedTextMessage", "imageMessage", "videoMessage", "audioMessage", "documentMessage", "stickerMessage"} {
		if sub := getMapField(msg, key); sub != nil {
			if ci := getMapField(sub, "contextInfo"); ci != nil {
				return ci
			}
		}
	}
	return nil
}

func isEphemeral(msg map[string]any) bool {
	if ci := extractContextInfo(msg); ci != nil {
		if ts := getFloatField(ci, "ephemeralSettingTimestamp"); ts > 0 {
			return true
		}
	}
	return false
}

func isGIF(msg map[string]any) bool {
	if vidMsg := getMapField(msg, "videoMessage"); vidMsg != nil {
		if gif, _ := vidMsg["gifPlayback"].(bool); gif {
			return true
		}
	}
	return false
}
