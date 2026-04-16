package chatwoot

import (
	"context"
	"strings"

	"wzap/internal/logger"
	"wzap/internal/model"
)

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
		if msg, err := s.msgRepo.FindByTimestamp(ctx, sessionID, chatJID, timestamp, 60); err == nil && hasCWMessageID(msg) {
			logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Int("cwMsgID", *msg.CWMessageID).Msg("found CW message ID via FindByTimestamp")
			return *msg.CWMessageID
		}
	}

	logger.Debug().Str("component", "chatwoot").Str("session", sessionID).Str("stanzaID", stanzaID).Msg("could not resolve CW message ID for reply")
	return 0
}

type cwMsgParams struct {
	MessageType  string
	SourceID     string
	ContentAttrs map[string]any
}

func newCWMsgParams(fromMe bool, msgID, stanzaID string, cwReplyID int) cwMsgParams {
	p := cwMsgParams{MessageType: "incoming"}
	if fromMe {
		p.MessageType = "outgoing"
	}
	if msgID != "" {
		p.SourceID = "WAID:" + msgID
	}
	if stanzaID != "" || cwReplyID > 0 {
		p.ContentAttrs = make(map[string]any)
		if stanzaID != "" {
			p.ContentAttrs["reply_source_id"] = "WAID:" + stanzaID
		}
		if cwReplyID > 0 {
			p.ContentAttrs["in_reply_to"] = cwReplyID
		}
	}
	return p
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
