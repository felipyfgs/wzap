package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"

	"wzap/internal/logger"
	"wzap/internal/storage"
	"wzap/internal/wa"
)

type MediaService struct {
	engine *wa.Manager
	minio  *storage.Minio
}

func NewMediaService(engine *wa.Manager, minio *storage.Minio) *MediaService {
	return &MediaService{engine: engine, minio: minio}
}

func (s *MediaService) DownloadAndStore(ctx context.Context, sessionID string, msg whatsmeow.DownloadableMessage, mimeType, messageID string) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	data, err := client.Download(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	key := fmt.Sprintf("%s/%s", sessionID, messageID)
	if err := s.minio.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		return "", fmt.Errorf("failed to upload media to S3: %w", err)
	}

	url, err := s.minio.PresignedURL(ctx, key, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

func (s *MediaService) DownloadMediaMessage(ctx context.Context, sessionID string, msgProto *waE2E.Message) ([]byte, string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, "", err
	}

	var downloadable whatsmeow.DownloadableMessage
	var mimeType string

	switch {
	case msgProto.GetImageMessage() != nil:
		downloadable = msgProto.GetImageMessage()
		mimeType = msgProto.GetImageMessage().GetMimetype()
	case msgProto.GetVideoMessage() != nil:
		downloadable = msgProto.GetVideoMessage()
		mimeType = msgProto.GetVideoMessage().GetMimetype()
	case msgProto.GetAudioMessage() != nil:
		downloadable = msgProto.GetAudioMessage()
		mimeType = msgProto.GetAudioMessage().GetMimetype()
	case msgProto.GetDocumentMessage() != nil:
		downloadable = msgProto.GetDocumentMessage()
		mimeType = msgProto.GetDocumentMessage().GetMimetype()
	case msgProto.GetStickerMessage() != nil:
		downloadable = msgProto.GetStickerMessage()
		mimeType = msgProto.GetStickerMessage().GetMimetype()
	default:
		return nil, "", fmt.Errorf("message does not contain downloadable media")
	}

	data, err := client.Download(ctx, downloadable)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}

	return data, mimeType, nil
}

func (s *MediaService) GetPresignedURL(ctx context.Context, sessionID, messageID string) (string, error) {
	key := fmt.Sprintf("%s/%s", sessionID, messageID)
	url, err := s.minio.PresignedURL(ctx, key, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url, nil
}

func (s *MediaService) AutoUploadMedia(sessionID, messageID, mimeType string, downloadable whatsmeow.DownloadableMessage) {
	if s.minio == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		client, err := s.engine.GetClient(sessionID)
		if err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Msg("Auto-upload: failed to get client")
			return
		}

		data, err := client.Download(ctx, downloadable)
		if err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Auto-upload: failed to download media")
			return
		}

		key := fmt.Sprintf("%s/%s", sessionID, messageID)
		if err := s.minio.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Auto-upload: failed to upload to S3")
			return
		}

		logger.Debug().Str("session", sessionID).Str("mid", messageID).Msg("Auto-upload: media stored in S3")
	}()
}
