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
	MessageEchoes    []map[string]any      `json:"message_echoes,omitempty"`
	Contacts         []map[string]any      `json:"contacts,omitempty"`
	Statuses         []map[string]any      `json:"statuses,omitempty"`
	Errors           []map[string]any      `json:"errors,omitempty"`
}

type cloudWebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

func buildCloudWebhookEnvelope(sessionPhone string, outgoing bool, msg map[string]any, contact map[string]any) *cloudWebhookEnvelope {
	if msg == nil {
		return nil
	}

	field := "messages"
	value := cloudWebhookValue{
		MessagingProduct: "whatsapp",
		Metadata: &cloudWebhookMetadata{
			DisplayPhoneNumber: sessionPhone,
			PhoneNumberID:      sessionPhone,
		},
		Statuses: []map[string]any{},
		Errors:   []map[string]any{},
	}
	if outgoing {
		field = "smb_message_echoes"
		value.MessageEchoes = []map[string]any{msg}
	} else {
		value.Messages = []map[string]any{msg}
		value.Contacts = []map[string]any{}
	}

	envelope := &cloudWebhookEnvelope{
		Object: "whatsapp_business_account",
		Entry: []cloudWebhookEntry{
			{
				ID: sessionPhone,
				Changes: []cloudWebhookChange{
					{
						Field: field,
						Value: value,
					},
				},
			},
		},
	}

	if !outgoing && contact != nil {
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

var knownCloudMediaTypes = map[string]struct{}{
	"image":    {},
	"video":    {},
	"audio":    {},
	"document": {},
	"sticker":  {},
}

func cloudMediaType(waType string) string {
	if _, ok := knownCloudMediaTypes[waType]; ok {
		return waType
	}
	return "document"
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

	return s.uploadRawMedia(ctx, mediaData, cfg.SessionID, msgID, filename, mimeType)
}

func (s *Service) uploadRawMedia(ctx context.Context, data []byte, sessionID, msgID, filename, mimeType string) (string, error) {
	if s.mediaPresigner == nil {
		return "", fmt.Errorf("MinIO not configured, cannot upload media for cloud mode")
	}
	if s.mediaStorage == nil {
		return "", fmt.Errorf("media storage not available")
	}

	key := fmt.Sprintf("chatwoot/%s/%s", sessionID, msgID)

	// Persiste o filename original como user metadata no MinIO. O
	// CloudAPIHandler.DownloadCloudMedia lê esse metadata e emite o header
	// `Content-Disposition: inline; filename="..."` para que o Chatwoot
	// preserve o nome real do arquivo (ex.: "report.pdf") em vez de usar
	// o mediaID como nome do anexo.
	var userMeta map[string]string
	if filename != "" {
		userMeta = map[string]string{"filename": filename}
	}

	if err := s.mediaStorage.UploadWithMeta(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType, userMeta); err != nil {
		return "", fmt.Errorf("failed to upload media to storage: %w", err)
	}

	presignedURL, err := s.mediaPresigner.GetPresignedURL(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, nil
}

func (s *Service) UnlockCloudWindow(ctx context.Context, cfg *Config, chatJID string) {
	if cfg == nil || cfg.InboxType != "cloud" || chatJID == "" {
		return
	}
	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		return
	}
	if s.cache != nil {
		if _, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID); ok {
			return
		}
	}

	sessionPhone := ""
	if s.sessionPhoneGet != nil {
		sessionPhone = s.sessionPhoneGet.GetSessionPhone(ctx, cfg.SessionID)
	}
	if sessionPhone == "" {
		logger.Debug().Str("component", "chatwoot").Str("chatJID", chatJID).Msg("UnlockCloudWindow: no session phone, skipping")
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
	unlockNotice := "✓ Conversa iniciada."
	cloudMsg := buildCloudTextMessage(unlockNotice, msgID, from, ts)
	envelope := buildCloudWebhookEnvelope(sessionPhone, false, cloudMsg, buildCloudContact(from, contactName))

	if err := s.postToChatwootCloud(ctx, cfg, sessionPhone, envelope); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("chatJID", chatJID).Msg("UnlockCloudWindow: failed to post webhook")
	} else {
		logger.Debug().Str("component", "chatwoot").Str("chatJID", chatJID).Str("from", from).Msg("UnlockCloudWindow: sent synthetic incoming to unlock 24h window")
	}
}
