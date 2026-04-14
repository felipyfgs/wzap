package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"

	"wzap/internal/async"
	"wzap/internal/logger"
	"wzap/internal/model"
	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/repo"
	"wzap/internal/storage"
	"wzap/internal/wa"
)

type mediaKeyPersister interface {
	UpdateMediaURL(ctx context.Context, sessionID, msgID, mediaURL string) error
}

type MediaService struct {
	minio           *storage.Minio
	pool            *async.Pool
	runtimeResolver *RuntimeResolver
	msgRepo         mediaKeyPersister
}

func (s *MediaService) SetMessageRepo(r mediaKeyPersister) { s.msgRepo = r }

func NewMediaService(engine *wa.Manager, minio *storage.Minio, provider *cloudWA.Client, sessRepo *repo.SessionRepository, pool *async.Pool, runtimeResolver *RuntimeResolver) *MediaService {
	if runtimeResolver == nil {
		runtimeResolver = NewRuntimeResolver(sessRepo, engine, provider)
	}
	return &MediaService{minio: minio, pool: pool, runtimeResolver: runtimeResolver}
}

func (s *MediaService) DownloadAndStore(ctx context.Context, sessionID string, msg whatsmeow.DownloadableMessage, mimeType, messageID, chatJID, senderJID string, fromMe bool) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMedia(ctx, sessionID, model.CapabilityMediaDownload)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		data, err := client.Download(ctx, msg)
		if err != nil {
			return "", fmt.Errorf("failed to download media: %w", err)
		}

		key := storage.MediaObjectKey(storage.MediaKeyParams{
			SessionID: session.ID,
			ChatJID:   chatJID,
			SenderJID: senderJID,
			FromMe:    fromMe,
			MessageID: messageID,
			MimeType:  mimeType,
			Timestamp: time.Now(),
		})
		if err := s.minio.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
			return "", fmt.Errorf("failed to upload media to S3: %w", err)
		}

		url, err := s.minio.PresignedURL(ctx, key, 24*time.Hour)
		if err != nil {
			return "", fmt.Errorf("failed to generate presigned URL: %w", err)
		}

		return url, nil
	})
}

func (s *MediaService) DownloadMediaMessage(ctx context.Context, sessionID string, msgProto *waE2E.Message) ([]byte, string, error) {
	runtime, err := s.runtimeResolver.ResolveMedia(ctx, sessionID, model.CapabilityMediaDownload)
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

	type mediaDownloadResult struct {
		data     []byte
		mimeType string
	}

	result, err := runSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (mediaDownloadResult, error) {
		data, err := client.Download(ctx, downloadable)
		if err != nil {
			return mediaDownloadResult{}, fmt.Errorf("failed to download media: %w", err)
		}

		return mediaDownloadResult{data: data, mimeType: mimeType}, nil
	})
	if err != nil {
		return nil, "", err
	}

	return result.data, result.mimeType, nil
}

func (s *MediaService) GetPresignedURL(ctx context.Context, key string) (string, error) {
	url, err := s.minio.PresignedURL(ctx, key, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url, nil
}

func (s *MediaService) AutoUploadMedia(sessionID, messageID, chatJID, senderJID, mimeType string, fromMe bool, timestamp time.Time, downloadable whatsmeow.DownloadableMessage) {
	if s.minio == nil {
		return
	}

	_ = s.pool.Submit(func(ctx context.Context) {
		runtime, err := s.runtimeResolver.ResolveMedia(ctx, sessionID, model.CapabilityMediaDownload)
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg("Auto-upload: media download not supported for engine")
			return
		}

		_, err = runSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
			data, err := client.Download(ctx, downloadable)
			if err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Auto-upload: failed to download media")
				return struct{}{}, nil
			}

			key := storage.MediaObjectKey(storage.MediaKeyParams{
				SessionID: session.ID,
				ChatJID:   chatJID,
				SenderJID: senderJID,
				FromMe:    fromMe,
				MessageID: messageID,
				MimeType:  mimeType,
				Timestamp: timestamp,
			})
			if err := s.minio.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Auto-upload: failed to upload to S3")
				return struct{}{}, nil
			}

			if s.msgRepo != nil {
				if err := s.msgRepo.UpdateMediaURL(ctx, session.ID, messageID, key); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Auto-upload: failed to persist media key")
				}
			}

			logger.Debug().Str("component", "service").Str("session", session.ID).Str("mid", messageID).Msg("Auto-upload: media stored in S3")
			return struct{}{}, nil
		})
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg("Auto-upload: failed to get client")
		}
	})
}

func convertToOGG(input []byte) ([]byte, error) {
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-c:a", "libopus", "-f", "ogg", "pipe:1")
	cmd.Stdin = bytes.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w, stderr: %s", err, stderr.String())
	}

	return out.Bytes(), nil
}

func isOGGOpus(mimeType string) bool {
	return strings.Contains(strings.ToLower(mimeType), "ogg") ||
		strings.Contains(strings.ToLower(mimeType), "opus")
}

func checkFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}
