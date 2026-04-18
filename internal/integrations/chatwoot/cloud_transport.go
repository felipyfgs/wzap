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

type mediaUploader func(ctx context.Context, key string, reader io.Reader, size int64, mimeType string, userMeta map[string]string) error

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

func (s *Service) uploadRawMedia(ctx context.Context, _ *Config, data []byte, sessionID, msgID, filename, mimeType string) (string, error) {
	if s.mediaPresigner == nil {
		return "", fmt.Errorf("MinIO not configured, cannot upload media for cloud mode")
	}

	key := fmt.Sprintf("chatwoot/%s/%s", sessionID, msgID)

	upload := s.getMediaUploader()
	if upload == nil {
		return "", fmt.Errorf("media storage not available")
	}

	// Persiste o filename original como user metadata no MinIO. O
	// CloudAPIHandler.DownloadCloudMedia lê esse metadata e emite o header
	// `Content-Disposition: inline; filename="..."` para que o Chatwoot
	// preserve o nome real do arquivo (ex.: "report.pdf") em vez de usar
	// o mediaID como nome do anexo.
	var userMeta map[string]string
	if filename != "" {
		userMeta = map[string]string{"filename": filename}
	}

	if err := upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType, userMeta); err != nil {
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
	return func(ctx context.Context, key string, reader io.Reader, size int64, mimeType string, userMeta map[string]string) error {
		return s.mediaStorage.UploadWithMeta(ctx, key, reader, size, mimeType, userMeta)
	}
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
