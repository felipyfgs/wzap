package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"

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
	statusRepo      mediaKeyPersister
}

func (s *MediaService) SetMessageRepo(r mediaKeyPersister) { s.msgRepo = r }
func (s *MediaService) SetStatusRepo(r mediaKeyPersister)  { s.statusRepo = r }

func NewMediaService(engine *wa.Manager, minio *storage.Minio, provider *cloudWA.Client, sessRepo *repo.SessionRepository, pool *async.Pool, runtimeResolver *RuntimeResolver) *MediaService {
	if runtimeResolver == nil {
		runtimeResolver = NewRuntimeResolver(sessRepo, engine, provider)
	}
	return &MediaService{minio: minio, pool: pool, runtimeResolver: runtimeResolver}
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

func (s *MediaService) AutoUploadStatusMedia(sessionID, messageID, chatJID, senderJID, mimeType string, fromMe bool, timestamp time.Time, downloadable whatsmeow.DownloadableMessage) {
	if s.minio == nil {
		return
	}

	_ = s.pool.Submit(func(ctx context.Context) {
		runtime, err := s.runtimeResolver.ResolveMedia(ctx, sessionID, model.CapabilityMediaDownload)
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg("Status auto-upload: media download not supported")
			return
		}

		_, err = runSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
			data, err := client.Download(ctx, downloadable)
			if err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Status auto-upload: failed to download media")
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
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Status auto-upload: failed to upload to S3")
				return struct{}{}, nil
			}

			if s.statusRepo != nil {
				if err := s.statusRepo.UpdateMediaURL(ctx, session.ID, messageID, key); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Status auto-upload: failed to persist media key")
				}
			}

			logger.Debug().Str("component", "service").Str("session", session.ID).Str("mid", messageID).Msg("Status auto-upload: media stored in S3")
			return struct{}{}, nil
		})
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg("Status auto-upload: failed to get client")
		}
	})
}

func (s *MediaService) RetryMediaUpload(sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int) {
	if s.minio == nil {
		return
	}

	_ = s.pool.Submit(func(ctx context.Context) {
		runtime, err := s.runtimeResolver.ResolveMedia(ctx, sessionID, model.CapabilityMediaDownload)
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Media retry upload: engine não disponível")
			return
		}

		_, err = runSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
			data, err := client.DownloadMediaWithPath(ctx, directPath, encFileHash, fileHash, mediaKey, fileLength, mimeTypeToMediaType(mimeType), "")
			if err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Media retry upload: falha ao baixar mídia")
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
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Media retry upload: falha ao enviar para S3")
				return struct{}{}, nil
			}

			if s.msgRepo != nil {
				if err := s.msgRepo.UpdateMediaURL(ctx, session.ID, messageID, key); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg("Media retry upload: falha ao persistir chave S3")
				}
			}

			logger.Debug().Str("component", "service").Str("session", session.ID).Str("mid", messageID).Msg("Media retry upload: mídia armazenada no S3")
			return struct{}{}, nil
		})
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Str("mid", messageID).Msg("Media retry upload: erro de runtime")
		}
	})
}

func mimeTypeToMediaType(mimeType string) whatsmeow.MediaType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return whatsmeow.MediaImage
	case strings.HasPrefix(mimeType, "audio/"):
		return whatsmeow.MediaAudio
	case strings.HasPrefix(mimeType, "video/"):
		return whatsmeow.MediaVideo
	default:
		return whatsmeow.MediaDocument
	}
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
