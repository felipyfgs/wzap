package chatwoot

import (
	"context"
	"fmt"
	"strings"

	"wzap/internal/logger"
)

type cloudInboxHandler struct {
	svc *Service
}

func newCloudInboxHandler(svc *Service) *cloudInboxHandler {
	return &cloudInboxHandler{svc: svc}
}

func (h *cloudInboxHandler) HandleMessage(ctx context.Context, cfg *Config, payload []byte) error {
	res, skip, err := h.svc.inboxPrologue(ctx, cfg, payload, inboxPrologueOpts{})
	if err != nil || skip {
		return err
	}
	data := res.data
	chatJID := res.chatJID

	// Chatwoot Cloud inbox only supports E164 phone numbers as source_id.
	// Group JIDs (e.g. "120363...@g.us") are not valid and would cause the
	// channel to be marked as inactive on Chatwoot side.
	if strings.HasSuffix(chatJID, "@g.us") || strings.HasSuffix(chatJID, "@newsletter") {
		return nil
	}

	chatPhone := extractPhone(chatJID)
	from := chatPhone
	if from == "" {
		from = extractPhone(data.Info.Sender)
	}
	msgID := data.Info.ID
	timestamp := fmt.Sprintf("%d", data.Info.Timestamp)
	pushName := data.Info.PushName

	sessionPhone := ""
	if h.svc.sessionPhoneGet != nil {
		sessionPhone = h.svc.sessionPhoneGet.GetSessionPhone(ctx, cfg.SessionID)
	}
	if sessionPhone == "" {
		logger.Warn().Str("component", "chatwoot").Str("session", cfg.SessionID).Msg("Cloud: could not resolve session phone, skipping")
		return nil
	}
	if data.Info.IsFromMe {
		from = sessionPhone
	}

	msg := data.Message
	if msg == nil {
		return nil
	}

	stanzaID := extractStanzaID(msg)
	replyTargetID := stanzaID

	var cloudMsg map[string]any
	contactPhone := chatPhone
	if contactPhone == "" {
		contactPhone = extractPhone(data.Info.Sender)
	}
	contactEntry := buildCloudContact(contactPhone, pushName)

	mediaInfo := extractMediaInfo(msg)
	if mediaInfo != nil {
		mediaType := cloudMediaType(mediaInfo.MediaType)
		link, err := h.svc.uploadCloudMedia(ctx, cfg, mediaInfo, msgID)
		if err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("mid", msgID).Msg("cloud inbound: failed to upload media, sending caption only")
			caption := extractText(msg)
			if caption == "" {
				caption = mediaInfo.FileName
			}
			if caption != "" {
				cloudMsg = buildCloudTextMessage(caption, msgID, from, timestamp)
			}
		} else {
			caption := extractText(msg)
			if mediaType == "" {
				mediaType = "document"
			}
			cloudMsg = buildCloudMediaMessage(mediaType, link, mediaInfo.MimeType, caption, mediaInfo.FileName, msgID, from, timestamp)
		}
	} else if locMsg := getMapField(msg, "locationMessage"); locMsg != nil {
		lat := getFloatField(locMsg, "degreesLatitude")
		lng := getFloatField(locMsg, "degreesLongitude")
		name := getStringField(locMsg, "name")
		address := getStringField(locMsg, "address")
		cloudMsg = buildCloudLocationMessage(lat, lng, name, address, msgID, from, timestamp)
	} else if contactMsg := getMapField(msg, "contactMessage"); contactMsg != nil {
		displayName := getStringField(contactMsg, "displayName")
		vcard := getStringField(contactMsg, "vcard")
		contacts := parseVCardToCloudContacts(vcard, displayName)
		cloudMsg = buildCloudContactMessage(contacts, msgID, from, timestamp)
	} else if reactMsg := getMapField(msg, "reactionMessage"); reactMsg != nil {
		key := getMapField(reactMsg, "key")
		targetID := ""
		if key != nil {
			targetID = getStringField(key, "ID")
		}
		emoji := getStringField(reactMsg, "text")
		if emoji == "" {
			return nil
		}
		replyTargetID = targetID
		cloudMsg = buildCloudTextMessage(emoji, msgID, from, timestamp)
	} else {
		text := extractText(msg)
		if text == "" {
			return nil
		}
		cloudMsg = buildCloudTextMessage(text, msgID, from, timestamp)
	}

	if cloudMsg == nil {
		return nil
	}
	if data.Info.IsFromMe && contactPhone != "" {
		cloudMsg["to"] = contactPhone
	}

	if replyTargetID != "" {
		cloudMsg["context"] = map[string]any{
			"id":         replyTargetID,
			"message_id": replyTargetID,
		}
	}

	envelope := buildCloudWebhookEnvelope(sessionPhone, data.Info.IsFromMe, cloudMsg, contactEntry)
	if envelope == nil {
		return nil
	}

	if err := h.svc.postToChatwootCloud(ctx, cfg, sessionPhone, envelope); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("mid", msgID).Msg("cloud inbound: failed to post to chatwoot")
		return err
	}
	if msgID != "" {
		h.svc.cache.SetIdempotent(ctx, cfg.SessionID, "WAID:"+msgID)
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("mid", msgID).Msg("cloud inbound: message sent to chatwoot")

	// Proactively resolve CW refs so edit/revoke can find them without
	// depending on the message_created return webhook from Chatwoot.
	if msgID != "" && chatJID != "" {
		h.svc.resolveCloudRefAsync(cfg, msgID, chatJID)
	}

	return nil
}
