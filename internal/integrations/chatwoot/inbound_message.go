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

func (s *Service) handleMessage(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] Failed to parse message payload")
		return
	}

	logger.Debug().Str("chat", data.Info.Chat).Str("id", data.Info.ID).Bool("fromMe", data.Info.IsFromMe).Msg("[CW] handleMessage")

	chatJID := data.Info.Chat
	if chatJID == "" {
		logger.Warn().Msg("[CW] chatJID empty, skipping")
		return
	}

	if strings.HasSuffix(chatJID, "@lid") && s.jidResolver != nil {
		if pn := s.jidResolver.GetPNForLID(ctx, cfg.SessionID, chatJID); pn != "" {
			logger.Debug().Str("lid", chatJID).Str("pn", pn).Msg("[CW] resolved LID to PN")
			chatJID = pn + "@s.whatsapp.net"
		}
	}

	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		logger.Debug().Str("chat", chatJID).Msg("[CW] JID ignored, skipping")
		return
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
			logger.Debug().Str("sourceID", sourceID).Msg("[CW] inbound duplicate, skipping")
			metrics.CWIdempotentDrops.WithLabelValues(cfg.SessionID).Inc()
			return
		}
		s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)

		msgBody := extractTextFromMessage(data.Message)
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
			logger.Warn().Err(err).Str("msgID", msgID).Msg("[CW] failed to save message to DB")
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
		logger.Warn().Err(err).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create Chatwoot conversation")
		return
	}

	msg := data.Message

	if pollMsg := getMapField(msg, "pollCreationMessage"); pollMsg != nil {
		s.handlePollCreation(ctx, cfg, convID, msgID, fromMe, pollMsg)
		return
	}
	if pollMsg := getMapField(msg, "pollCreationMessageV3"); pollMsg != nil {
		s.handlePollCreation(ctx, cfg, convID, msgID, fromMe, pollMsg)
		return
	}
	if pollUpdate := getMapField(msg, "pollUpdateMessage"); pollUpdate != nil {
		s.handlePollUpdate(ctx, cfg, pollUpdate)
		return
	}
	if reactMsg := getMapField(msg, "reactionMessage"); reactMsg != nil {
		s.handleReaction(ctx, cfg, convID, msgID, fromMe, reactMsg)
		return
	}
	if btnResp := getMapField(msg, "buttonsResponseMessage"); btnResp != nil {
		s.handleButtonResponse(ctx, cfg, convID, msgID, fromMe, msg, btnResp)
		return
	}
	if listResp := getMapField(msg, "listResponseMessage"); listResp != nil {
		s.handleListResponse(ctx, cfg, convID, msgID, fromMe, msg, listResp)
		return
	}
	if tmplReply := getMapField(msg, "templateButtonReplyMessage"); tmplReply != nil {
		s.handleTemplateButtonReply(ctx, cfg, convID, msgID, fromMe, msg, tmplReply)
		return
	}
	if vonce := getMapField(msg, "viewOnceMessage"); vonce != nil {
		s.handleViewOnce(ctx, cfg, convID, msgID, fromMe, vonce)
		return
	}
	if vonce := getMapField(msg, "viewOnceMessageV2"); vonce != nil {
		s.handleViewOnce(ctx, cfg, convID, msgID, fromMe, vonce)
		return
	}
	if editMsg := getMapField(msg, "editedMessage"); editMsg != nil {
		s.handleEditedMessage(ctx, cfg, editMsg)
		return
	}
	if liveMsg := getMapField(msg, "liveLocationMessage"); liveMsg != nil {
		s.handleLiveLocation(ctx, cfg, convID, msgID, fromMe, liveMsg)
		return
	}

	mediaInfo := extractMediaInfo(msg)
	if mediaInfo != nil {
		if mediaInfo.MediaType == "sticker" {
			s.handleStickerMessage(ctx, cfg, convID, msgID, fromMe, mediaInfo)
		} else {
			s.handleMediaMessage(ctx, cfg, convID, msgID, fromMe, msg)
		}
		return
	}

	text := extractTextFromMessage(msg)
	logger.Debug().Str("text", text).Interface("msg", msg).Msg("[CW] extracted text")
	text = applyMessagePrefixes(msg, convertWAToCWMarkdown(text))

	if !fromMe && data.Info.IsGroup && cfg.SignMsg && pushName != "" {
		phone := extractPhone(chatJID)
		text = formatGroupContent(phone, pushName, text, fromMe)
	}

	if text == "" {
		return
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

	stanzaID := extractStanzaID(msg)
	if stanzaID != "" {
		quotedText := extractQuotedMessageText(msg)
		cwMsgID := s.resolveInboundReply(ctx, cfg.SessionID, chatJID, stanzaID, quotedText, int64(data.Info.Timestamp))
		if cwMsgID > 0 {
			msgReq.SourceReplyID = cwMsgID
		}
		ca := map[string]any{"in_reply_to_external_id": "WAID:" + stanzaID}
		if cwMsgID > 0 {
			ca["in_reply_to"] = cwMsgID
		}
		msgReq.ContentAttributes = ca
		logger.Debug().Str("session", cfg.SessionID).Int("sourceReplyID", cwMsgID).Interface("contentAttributes", ca).Msg("[CW] set ContentAttributes for reply")
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Msg("Failed to create Chatwoot message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
		s.cache.SetIdempotent(ctx, cfg.SessionID, sourceID)
	}
}

func hasCWMessageID(msg *model.Message) bool {
	return msg != nil && msg.CWMessageID != nil && *msg.CWMessageID != 0
}

func (s *Service) resolveInboundReply(ctx context.Context, sessionID, chatJID, stanzaID, quotedText string, timestamp int64) int {
	logger.Debug().Str("session", sessionID).Str("stanzaID", stanzaID).Str("quotedText", quotedText).Msg("[CW] stanzaID found")

	if msg, err := s.msgRepo.FindByID(ctx, sessionID, stanzaID); err == nil && hasCWMessageID(msg) {
		logger.Debug().Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("[CW] found CW message ID via FindByID")
		return *msg.CWMessageID
	}

	if msg, err := s.msgRepo.FindBySourceID(ctx, sessionID, "WAID:"+stanzaID); err == nil && hasCWMessageID(msg) {
		logger.Debug().Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("[CW] found CW message ID via FindBySourceID")
		return *msg.CWMessageID
	}

	if quotedText != "" {
		if msg, err := s.msgRepo.FindByBodyAndChat(ctx, sessionID, chatJID, quotedText, true); err == nil && hasCWMessageID(msg) {
			logger.Debug().Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("[CW] found CW message ID via FindByBodyAndChat")
			return *msg.CWMessageID
		}
	}

	if timestamp > 0 {
		if msg, err := s.msgRepo.FindByTimestampWindow(ctx, sessionID, chatJID, timestamp, 60); err == nil && hasCWMessageID(msg) {
			logger.Debug().Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("[CW] found CW message ID via FindByTimestampWindow")
			return *msg.CWMessageID
		}
	}

	logger.Debug().Str("session", sessionID).Str("stanzaID", stanzaID).Msg("[CW] could not resolve CW message ID for reply")
	return 0
}

func (s *Service) handleMediaMessage(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, msg map[string]interface{}) {
	info := extractMediaInfo(msg)
	if info == nil {
		logger.Warn().Msg("[CW] no media info found in message")
		return
	}

	if int64(info.FileLength) > maxMediaBytes {
		client := s.clientFn(cfg)
		messageType := "incoming"
		if fromMe {
			messageType = "outgoing"
		}
		warnMsg := fmt.Sprintf("⚠️ Arquivo muito grande (%d MB) para download (limite: 256 MB)", info.FileLength/(1024*1024))
		_, _ = client.CreateMessage(ctx, convID, MessageReq{Content: warnMsg, MessageType: messageType})
		return
	}

	if s.mediaDownloader == nil {
		logger.Warn().Msg("[CW] media downloader not configured, cannot download WhatsApp media")
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
		logger.Warn().Err(err).Str("mediaType", info.MediaType).Msg("[CW] failed to download WhatsApp media")
		return
	}

	metrics.CWMediaDownloadBytes.WithLabelValues(cfg.SessionID, info.MediaType).Add(float64(len(data)))

	mimeType := info.MimeType
	if mimeType == "" {
		mimeType, _ = GetMIMETypeAndExt("", data)
	}

	filename := info.FileName
	if filename == "" {
		ext := mimeTypeToExt(mimeType)
		filename = info.MediaType + ext
	}

	caption := extractTextFromMessage(msg)
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

	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, caption, filename, data, mimeType, messageType, sourceID)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to upload media to Chatwoot")
		return
	}

	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleStickerMessage(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, info *mediaInfo) {
	if s.mediaDownloader == nil {
		return
	}

	data, err := s.mediaDownloader.DownloadMediaByPath(ctx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256, info.MediaKey, info.FileLength, "sticker")
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to download sticker")
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

	if len(data) > 0 && len(data) <= 1024*1024 {
		pngData, err := convertWebPToPNG(data)
		if err == nil {
			cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.png", pngData, "image/png", messageType, sourceID)
			if err == nil && msgID != "" {
				_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
				return
			}
		}
		gifData, err := convertWebPToGIF(data)
		if err == nil {
			cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.gif", gifData, "image/gif", messageType, sourceID)
			if err == nil && msgID != "" {
				_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
				return
			}
		}
	}

	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.webp", data, "image/webp", messageType, sourceID)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to upload sticker fallback")
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
		Image: []*image.Paletted{imageToParletted(img)},
		Delay: []int{10},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageToParletted(img image.Image) *image.Paletted {
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
	seen := make(map[rgbaKey]struct{})
	var palette gocolor.Palette
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

func (s *Service) handlePollCreation(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, poll map[string]interface{}) {
	name := getStringField(poll, "name")
	optionsRaw, _ := poll["options"].([]interface{})
	var sb strings.Builder
	sb.WriteString("📊 *Enquete:* ")
	sb.WriteString(name)
	sb.WriteString("\n")
	for i, opt := range optionsRaw {
		if optMap, ok := opt.(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, getStringField(optMap, "optionName")))
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
		logger.Warn().Err(err).Msg("[CW] failed to create poll message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handlePollUpdate(ctx context.Context, cfg *ChatwootConfig, pollUpdate map[string]interface{}) {
	pollCreation := getMapField(pollUpdate, "pollCreationMessageKey")
	if pollCreation == nil {
		return
	}
	pollMsgID := getStringField(pollCreation, "id")
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
	selectedOpts, _ := votes["selectedOptions"].([]interface{})
	var sb strings.Builder
	sb.WriteString("📊 *Voto registrado:*\n")
	for _, opt := range selectedOpts {
		if optMap, ok := opt.(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("✅ %s\n", getStringField(optMap, "optionName")))
		}
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, *origMsg.CWConversationID, MessageReq{
		Content:           sb.String(),
		MessageType:       "incoming",
		SourceReplyID:     *origMsg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *origMsg.CWMessageID, "in_reply_to_external_id": "WAID:" + pollMsgID},
	})
}

func (s *Service) handleReaction(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, reactMsg map[string]interface{}) {
	key := getMapField(reactMsg, "key")
	if key == nil {
		return
	}
	targetMsgID := getStringField(key, "id")
	emoji := getStringField(reactMsg, "text")

	origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, targetMsgID)
	if err != nil || origMsg.CWMessageID == nil {
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
		SourceReplyID:     *origMsg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *origMsg.CWMessageID, "in_reply_to_external_id": "WAID:" + targetMsgID},
	})
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to create reaction message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleButtonResponse(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, msg map[string]interface{}, btnResp map[string]interface{}) {
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
				"in_reply_to":             *origMsg.CWMessageID,
				"in_reply_to_external_id": "WAID:" + stanzaID,
			}
		}
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to create button response message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleListResponse(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, msg map[string]interface{}, listResp map[string]interface{}) {
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
				"in_reply_to":             *origMsg.CWMessageID,
				"in_reply_to_external_id": "WAID:" + stanzaID,
			}
		}
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to create list response message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleTemplateButtonReply(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, msg map[string]interface{}, tmpl map[string]interface{}) {
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
				"in_reply_to":             *origMsg.CWMessageID,
				"in_reply_to_external_id": "WAID:" + stanzaID,
			}
		}
	}

	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessage(ctx, convID, msgReq)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] failed to create template reply message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) handleViewOnce(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, vonce map[string]interface{}) {
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

func (s *Service) handleEditedMessage(ctx context.Context, cfg *ChatwootConfig, editMsg map[string]interface{}) {
	key := getMapField(editMsg, "key")
	if key == nil {
		return
	}
	targetMsgID := getStringField(key, "id")
	if targetMsgID == "" {
		return
	}

	origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, targetMsgID)
	if err != nil || origMsg.CWMessageID == nil || origMsg.CWConversationID == nil {
		return
	}

	newText := ""
	if inner := getMapField(editMsg, "message"); inner != nil {
		newText = extractTextFromMessage(inner)
	}
	if newText == "" {
		return
	}

	client := s.clientFn(cfg)
	_ = client.UpdateMessage(ctx, *origMsg.CWConversationID, *origMsg.CWMessageID, newText)
}

func (s *Service) handleLiveLocation(ctx context.Context, cfg *ChatwootConfig, convID int, msgID string, fromMe bool, liveMsg map[string]interface{}) {
	lat := getFloatField(liveMsg, "degreesLatitude")
	lng := getFloatField(liveMsg, "degreesLongitude")
	text := fmt.Sprintf("📍 [Localização ao vivo]\nhttps://www.google.com/maps?q=%f,%f\n\n_Atualizações subsequentes serão ignoradas._", lat, lng)

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

func applyMessagePrefixes(msg map[string]interface{}, text string) string {
	if msg == nil {
		return text
	}

	var prefixes []string

	if ci := extractContextInfoFromMsg(msg); ci != nil {
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

func extractContextInfoFromMsg(msg map[string]interface{}) map[string]interface{} {
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

func isEphemeral(msg map[string]interface{}) bool {
	if ci := extractContextInfoFromMsg(msg); ci != nil {
		if ts := getFloatField(ci, "ephemeralSettingTimestamp"); ts > 0 {
			return true
		}
	}
	return false
}

func isGIF(msg map[string]interface{}) bool {
	if vidMsg := getMapField(msg, "videoMessage"); vidMsg != nil {
		if gif, _ := vidMsg["gifPlayback"].(bool); gif {
			return true
		}
	}
	return false
}
