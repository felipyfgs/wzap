package chatwoot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wzap/internal/imgutil"
	"wzap/internal/logger"
	"wzap/internal/metrics"
)

func (s *Service) processMediaMessage(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, stanzaID string, cwReplyID int) {
	info := extractMediaInfo(msg)
	if info == nil {
		logger.Warn().Str("component", "chatwoot").Msg("no media info found in message")
		return
	}

	if int64(info.FileLength) > maxMediaBytes {
		client := s.clientFn(cfg)
		p := newCWMsgParams(fromMe, "", "", 0)
		warnMsg := fmt.Sprintf("⚠️ Arquivo muito grande (%d MB) para download (limite: 256 MB)", info.FileLength/(1024*1024))
		_, _ = client.CreateMessage(ctx, convID, MessageReq{Content: warnMsg, MessageType: p.MessageType, Private: true})
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
	p := newCWMsgParams(fromMe, msgID, stanzaID, cwReplyID)

	if isGIF(msg) && caption == "" {
		caption = "[GIF]"
	}

	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, caption, filename, data, mimeType, p.MessageType, p.SourceID, cwReplyID, p.ContentAttrs)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to upload media to Chatwoot")
		return
	}

	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) processStickerMessage(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, info *mediaInfo, stanzaID string, cwReplyID int) {
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
	p := newCWMsgParams(fromMe, msgID, stanzaID, cwReplyID)

	if len(data) > 0 && len(data) <= 1024*1024 {
		pngData, err := imgutil.ConvertWebPToPNG(data)
		if err == nil {
			cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.png", pngData, "image/png", p.MessageType, p.SourceID, cwReplyID, p.ContentAttrs)
			if err == nil && msgID != "" {
				_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
				return
			}
		}
		gifData, err := imgutil.ConvertWebPToGIF(data)
		if err == nil {
			cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.gif", gifData, "image/gif", p.MessageType, p.SourceID, cwReplyID, p.ContentAttrs)
			if err == nil && msgID != "" {
				_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
				return
			}
		}
	}

	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", "sticker.webp", data, "image/webp", p.MessageType, p.SourceID, cwReplyID, p.ContentAttrs)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to upload sticker fallback")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) processPollCreation(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, poll map[string]any) {
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
	p := newCWMsgParams(fromMe, msgID, "", 0)
	cwMsg, err := client.CreateMessage(ctx, convID, MessageReq{
		Content:     sb.String(),
		MessageType: p.MessageType,
		SourceID:    p.SourceID,
	})
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("failed to create poll message")
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) processPollUpdate(ctx context.Context, cfg *Config, pollUpdate map[string]any) {
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

func (s *Service) processReaction(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, reactMsg map[string]any) {
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
	p := newCWMsgParams(fromMe, "", "", 0)

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
		MessageType:       p.MessageType,
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

func (s *Service) processButtonResponse(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, btnResp map[string]any) {
	text := getStringField(btnResp, "selectedDisplayText")
	if text == "" {
		text = getStringField(btnResp, "selectedButtonId")
	}
	content := fmt.Sprintf("[Botão] %s", text)

	p := newCWMsgParams(fromMe, msgID, "", 0)
	msgReq := MessageReq{Content: content, MessageType: p.MessageType, SourceID: p.SourceID}
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

func (s *Service) processListResponse(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, listResp map[string]any) {
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

	p := newCWMsgParams(fromMe, msgID, "", 0)
	msgReq := MessageReq{Content: content, MessageType: p.MessageType, SourceID: p.SourceID}
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

func (s *Service) processTemplateReply(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, msg map[string]any, tmpl map[string]any) {
	text := getStringField(tmpl, "selectedDisplayText")
	content := fmt.Sprintf("[Template] %s", text)

	p := newCWMsgParams(fromMe, msgID, "", 0)
	msgReq := MessageReq{Content: content, MessageType: p.MessageType, SourceID: p.SourceID}
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

func (s *Service) processViewOnce(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, vonce map[string]any, tryDownload bool, stanzaID string, cwReplyID int) {
	client := s.clientFn(cfg)
	p := newCWMsgParams(fromMe, msgID, stanzaID, cwReplyID)

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

					cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, "", filename, data, mimeType, p.MessageType, p.SourceID, cwReplyID, p.ContentAttrs)
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
		Content:           "[mensagem vista uma vez]",
		MessageType:       p.MessageType,
		SourceID:          p.SourceID,
		ContentAttributes: p.ContentAttrs,
	})
	if err != nil {
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) processEditedMessage(ctx context.Context, cfg *Config, editMsg map[string]any) {
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
	p := newCWMsgParams(origMsg.FromMe, "", "", 0)

	_, _ = client.CreateMessage(ctx, *origMsg.CWConversationID, MessageReq{
		Content:           "✏️ *Mensagem editada:*\n" + newText,
		MessageType:       p.MessageType,
		Private:           true,
		SourceReplyID:     *origMsg.CWMessageID,
		ContentAttributes: p.ContentAttrs,
	})
}

func (s *Service) processLiveLocation(ctx context.Context, cfg *Config, convID int, msgID string, fromMe bool, liveMsg map[string]any) {
	text := formatLocation(liveMsg)

	client := s.clientFn(cfg)
	p := newCWMsgParams(fromMe, msgID, "", 0)
	cwMsg, err := client.CreateMessage(ctx, convID, MessageReq{
		Content:     text,
		MessageType: p.MessageType,
		SourceID:    p.SourceID,
	})
	if err != nil {
		return
	}
	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}
