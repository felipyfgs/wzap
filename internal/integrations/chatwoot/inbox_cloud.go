package chatwoot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"wzap/internal/logger"
)

type cloudInboxHandler struct {
	svc *Service
}

func newCloudInboxHandler(svc *Service) *cloudInboxHandler {
	return &cloudInboxHandler{svc: svc}
}

func (h *cloudInboxHandler) HandleMessage(ctx context.Context, cfg *Config, payload []byte) error {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to parse message payload for cloud mode")
		return nil
	}

	if data.Info.IsFromMe {
		return nil
	}

	chatJID := data.Info.Chat
	if chatJID == "" {
		return nil
	}

	chatJID = h.svc.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.SenderAlt, data.Info.RecipientAlt)
	if strings.HasSuffix(chatJID, "@lid") {
		return nil
	}

	// Chatwoot Cloud inbox only supports E164 phone numbers as source_id.
	// Group JIDs (e.g. "120363...@g.us") are not valid and would cause the
	// channel to be marked as inactive on Chatwoot side.
	if strings.HasSuffix(chatJID, "@g.us") || strings.HasSuffix(chatJID, "@newsletter") {
		return nil
	}

	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		return nil
	}

	from := extractPhone(chatJID)
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

	msg := data.Message
	if msg == nil {
		return nil
	}

	stanzaID := extractStanzaID(msg)

	var cloudMsg map[string]any
	contactEntry := buildCloudContact(from, pushName)

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
		cloudMsg = buildCloudReactionMessage(targetID, emoji, msgID, from, timestamp)
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

	if stanzaID != "" {
		cloudMsg["context"] = map[string]any{
			"message_id": stanzaID,
		}
	}

	envelope := buildCloudWebhookEnvelope(sessionPhone, from, msgID, timestamp, pushName, cloudMsg, contactEntry)
	if envelope == nil {
		return nil
	}

	if err := h.svc.postToChatwootCloud(ctx, cfg, sessionPhone, envelope); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("mid", msgID).Msg("cloud inbound: failed to post to chatwoot")
		return err
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("mid", msgID).Msg("cloud inbound: message sent to chatwoot")
	return nil
}

func (h *cloudInboxHandler) UnlockWindow(ctx context.Context, cfg *Config, chatJID string) {
	h.svc.unlockCloudWindow(ctx, cfg, chatJID)
}

type cloudWebhookEnvelope struct {
	Object string              `json:"object"`
	Entry  []cloudWebhookEntry `json:"entry"`
}

type cloudWebhookEntry struct {
	ID      string               `json:"id"`
	Changes []cloudWebhookChange `json:"changes"`
}

type cloudWebhookChange struct {
	Value cloudWebhookValue `json:"value"`
	Field string            `json:"field"`
}

type cloudWebhookValue struct {
	MessagingProduct string                `json:"messaging_product"`
	Metadata         *cloudWebhookMetadata `json:"metadata,omitempty"`
	Messages         []map[string]any      `json:"messages,omitempty"`
	Contacts         []map[string]any      `json:"contacts,omitempty"`
	Statuses         []map[string]any      `json:"statuses,omitempty"`
	Errors           []map[string]any      `json:"errors,omitempty"`
}

type cloudWebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

func buildCloudWebhookEnvelope(sessionPhone string, _, _, _, _ string, msg map[string]any, contact map[string]any) *cloudWebhookEnvelope {
	if msg == nil {
		return nil
	}

	envelope := &cloudWebhookEnvelope{
		Object: "whatsapp_business_account",
		Entry: []cloudWebhookEntry{
			{
				ID: sessionPhone,
				Changes: []cloudWebhookChange{
					{
						Field: "messages",
						Value: cloudWebhookValue{
							MessagingProduct: "whatsapp",
							Metadata: &cloudWebhookMetadata{
								DisplayPhoneNumber: sessionPhone,
								PhoneNumberID:      sessionPhone,
							},
							Messages: []map[string]any{msg},
							Contacts: []map[string]any{},
							Statuses: []map[string]any{},
							Errors:   []map[string]any{},
						},
					},
				},
			},
		},
	}

	if contact != nil {
		envelope.Entry[0].Changes[0].Value.Contacts = []map[string]any{contact}
	}

	return envelope
}

func buildCloudTextMessage(body, msgID, from, timestamp string) map[string]any {
	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "text",
		"text": map[string]any{
			"body": body,
		},
	}
}

func buildCloudMediaMessage(mediaType, link, mimeType, caption, filename, msgID, from, timestamp string) map[string]any {
	// Chatwoot Cloud inbox resolves media via `id` → GET /v{version}/{phone}/{id}
	// (handled by CloudAPIHandler.GetMedia). `link` is kept as a fallback for
	// clients that read it directly.
	mediaObj := map[string]any{
		"id":        msgID,
		"link":      link,
		"mime_type": mimeType,
	}
	if caption != "" {
		mediaObj["caption"] = caption
	}
	if filename != "" {
		mediaObj["filename"] = filename
	}

	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      mediaType,
		mediaType:   mediaObj,
	}
}

func buildCloudLocationMessage(lat, lng float64, name, address, msgID, from, timestamp string) map[string]any {
	loc := map[string]any{
		"latitude":  lat,
		"longitude": lng,
	}
	if name != "" {
		loc["name"] = name
	}
	if address != "" {
		loc["address"] = address
	}

	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "location",
		"location":  loc,
	}
}

func buildCloudContactMessage(contacts []map[string]any, msgID, from, timestamp string) map[string]any {
	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "contacts",
		"contacts":  contacts,
	}
}

func buildCloudReactionMessage(targetMsgID, emoji, msgID, from, timestamp string) map[string]any {
	return map[string]any{
		"from":      from,
		"id":        msgID,
		"timestamp": timestamp,
		"type":      "reaction",
		"reaction": map[string]any{
			"message_id": targetMsgID,
			"emoji":      emoji,
		},
	}
}

func buildCloudContact(phone, name string) map[string]any {
	contact := map[string]any{}
	if name != "" {
		contact["profile"] = map[string]any{
			"name": name,
		}
	}
	if phone != "" {
		contact["wa_id"] = strings.TrimPrefix(phone, "+")
	}
	return contact
}

func cloudMediaType(waType string) string {
	switch waType {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "document":
		return "document"
	case "sticker":
		return "sticker"
	default:
		return "document"
	}
}

func parseVCardToCloudContacts(vcard, displayName string) []map[string]any {
	if vcard == "" {
		return []map[string]any{
			{"name": map[string]any{"formatted_name": displayName}},
		}
	}

	lines := strings.Split(vcard, "\n")
	var phones []map[string]any
	fn := displayName

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FN:") {
			fn = strings.TrimPrefix(line, "FN:")
		}
		if strings.HasPrefix(line, "TEL") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				phone := parts[1]
				phones = append(phones, map[string]any{
					"phone": phone,
					"type":  "CELL",
					"wa_id": strings.ReplaceAll(phone, "+", ""),
				})
			}
		}
	}

	contact := map[string]any{
		"name": map[string]any{
			"formatted_name": fn,
		},
	}
	if len(phones) > 0 {
		contact["phones"] = phones
	}

	return []map[string]any{contact}
}

type mediaUploader func(ctx context.Context, key string, reader io.Reader, size int64, mimeType string) error

func (s *Service) postToChatwootCloud(ctx context.Context, cfg *Config, sessionPhone string, payload any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud webhook payload: %w", err)
	}

	url := strings.TrimRight(cfg.URL, "/") + "/webhooks/whatsapp/+" + sessionPhone

	timeout := time.Duration(cfg.TextTimeout) * time.Second
	if cfg.TextTimeout == 0 {
		timeout = 10 * time.Second
	}
	postCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(postCtx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to chatwoot cloud webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chatwoot cloud webhook returned %d: %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logger.Warn().Str("component", "chatwoot").Int("status", resp.StatusCode).Str("body", string(body)).Msg("chatwoot cloud webhook client error")
	}

	return nil
}

func (s *Service) uploadCloudMedia(ctx context.Context, cfg *Config, info *mediaInfo, msgID string) (string, error) {
	if s.mediaDownloader == nil {
		return "", fmt.Errorf("media downloader not configured")
	}
	if s.mediaPresigner == nil {
		return "", fmt.Errorf("MinIO not configured, cannot upload media for cloud mode")
	}

	timeout := time.Duration(cfg.MediaTimeout) * time.Second
	if cfg.MediaTimeout == 0 {
		timeout = 60 * time.Second
	}
	mediaCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	mediaData, err := s.mediaDownloader.DownloadMediaByPath(mediaCtx, cfg.SessionID, info.DirectPath, info.FileEncSHA256, info.FileSHA256, info.MediaKey, info.FileLength, info.MediaType)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	mimeType := info.MimeType
	if mimeType == "" {
		mimeType, _ = DetectMIME(info.FileName, mediaData)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	filename := info.FileName
	if filename == "" {
		ext := mimeTypeToExt(mimeType)
		filename = info.MediaType + ext
	}

	url, err := s.uploadRawMedia(ctx, cfg, mediaData, cfg.SessionID, msgID, filename, mimeType)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (s *Service) uploadRawMedia(ctx context.Context, _ *Config, data []byte, sessionID, msgID, _, mimeType string) (string, error) {
	if s.mediaPresigner == nil {
		return "", fmt.Errorf("MinIO not configured, cannot upload media for cloud mode")
	}

	key := fmt.Sprintf("chatwoot/%s/%s", sessionID, msgID)

	upload := s.getMediaUploader()
	if upload == nil {
		return "", fmt.Errorf("media storage not available")
	}

	if err := upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		return "", fmt.Errorf("failed to upload media to storage: %w", err)
	}

	presignedURL, err := s.mediaPresigner.GetPresignedURL(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, nil
}

func (s *Service) getMediaUploader() mediaUploader {
	if s.mediaStorage == nil {
		return nil
	}
	return func(ctx context.Context, key string, reader io.Reader, size int64, mimeType string) error {
		return s.mediaStorage.Upload(ctx, key, reader, size, mimeType)
	}
}

func (s *Service) unlockCloudWindow(ctx context.Context, cfg *Config, chatJID string) {
	sessionPhone := ""
	if s.sessionPhoneGet != nil {
		sessionPhone = s.sessionPhoneGet.GetSessionPhone(ctx, cfg.SessionID)
	}
	if sessionPhone == "" {
		logger.Debug().Str("component", "chatwoot").Str("chatJID", chatJID).Msg("unlockCloudWindow: no session phone, skipping")
		return
	}

	from := extractPhone(chatJID)
	if from == "" {
		return
	}

	contactName := from
	if s.contactNameGetter != nil {
		if name := s.contactNameGetter.GetContactName(ctx, cfg.SessionID, chatJID); name != "" {
			contactName = name
		}
	}
	if contactName == from {
		client := s.clientFn(cfg)
		if contacts, err := client.FilterContacts(ctx, from); err == nil && len(contacts) > 0 {
			if contacts[0].Name != "" && contacts[0].Name != from {
				contactName = contacts[0].Name
			}
		}
	}

	ts := fmt.Sprintf("%d", time.Now().Unix())
	msgID := fmt.Sprintf("wzap-unlock-%d", time.Now().UnixNano())
	cloudMsg := buildCloudTextMessage("\u200e", msgID, from, ts)
	envelope := buildCloudWebhookEnvelope(sessionPhone, from, msgID, ts, "", cloudMsg, buildCloudContact(from, contactName))

	if err := s.postToChatwootCloud(ctx, cfg, sessionPhone, envelope); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("chatJID", chatJID).Msg("unlockCloudWindow: failed to post webhook")
	} else {
		logger.Debug().Str("component", "chatwoot").Str("chatJID", chatJID).Str("from", from).Msg("unlockCloudWindow: sent synthetic incoming to unlock 24h window")
	}
}
