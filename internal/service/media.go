package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"

	"wzap/internal/async"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/storage"
	"wzap/internal/wa"
	"wzap/internal/wautil"
)

type urlPersister interface {
	UpdateMediaURL(ctx context.Context, sessionID, msgID, mediaURL string) error
}

type MediaService struct {
	minio           *storage.Minio
	pool            *async.Pool
	runtimeResolver *RuntimeResolver
	msgRepo         urlPersister
	statusRepo      urlPersister
}

func (s *MediaService) SetMessageRepo(r urlPersister) { s.msgRepo = r }
func (s *MediaService) SetStatusRepo(r urlPersister)  { s.statusRepo = r }

func NewMediaService(engine *wa.Manager, minio *storage.Minio, sessRepo *repo.SessionRepository, pool *async.Pool, runtimeResolver *RuntimeResolver) *MediaService {
	if runtimeResolver == nil {
		runtimeResolver = NewRuntimeResolver(sessRepo, engine)
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

func (s *MediaService) autoUploadGeneric(ctx context.Context, sessionID, messageID, chatJID, senderJID, mimeType string, fromMe bool, timestamp time.Time, downloadable whatsmeow.DownloadableMessage, persister urlPersister, logPrefix string) {
	if s.minio == nil {
		return
	}

	runtime, err := s.runtimeResolver.ResolveMedia(ctx, sessionID, model.CapabilityMediaDownload)
	if err != nil {
		logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg(logPrefix + ": media download not supported for engine")
		return
	}

	_, err = runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
		data, err := client.Download(ctx, downloadable)
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg(logPrefix + ": failed to download media")
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
			logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg(logPrefix + ": failed to upload to S3")
			return struct{}{}, nil
		}

		if persister != nil {
			if err := persister.UpdateMediaURL(ctx, session.ID, messageID, key); err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", messageID).Msg(logPrefix + ": failed to persist media key")
			}
		}

		logger.Debug().Str("component", "service").Str("session", session.ID).Str("mid", messageID).Msg(logPrefix + ": media stored in S3")
		return struct{}{}, nil
	})
	if err != nil {
		logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg(logPrefix + ": failed to get client")
	}
}

func (s *MediaService) OnMediaReceived(input wa.MediaUploadInput) {
	_ = s.pool.Submit(func(ctx context.Context) {
		s.autoUploadGeneric(ctx, input.SessionID, input.MessageID, input.ChatJID, input.SenderJID, input.MimeType, input.FromMe, input.Timestamp, input.Downloadable, s.msgRepo, "Auto-upload")
	})
}

func (s *MediaService) OnStatusMediaReceived(input wa.MediaUploadInput) {
	_ = s.pool.Submit(func(ctx context.Context) {
		s.autoUploadGeneric(ctx, input.SessionID, input.MessageID, input.ChatJID, input.SenderJID, input.MimeType, input.FromMe, input.Timestamp, input.Downloadable, s.statusRepo, "Status auto-upload")
	})
}

func (s *MediaService) RetryMediaUpload(input wa.MediaRetryInput) {
	if s.minio == nil {
		return
	}

	_ = s.pool.Submit(func(ctx context.Context) {
		runtime, err := s.runtimeResolver.ResolveMedia(ctx, input.SessionID, model.CapabilityMediaDownload)
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", input.SessionID).Str("mid", input.MessageID).Msg("Media retry upload: engine not available")
			return
		}

		_, err = runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
			data, err := client.DownloadMediaWithPath(ctx, input.DirectPath, input.EncFileHash, input.FileHash, input.MediaKey, input.FileLength, wautil.MimeTypeToMediaType(input.MimeType), "")
			if err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", input.MessageID).Msg("Media retry upload: failed to download media")
				return struct{}{}, nil
			}

			key := storage.MediaObjectKey(storage.MediaKeyParams{
				SessionID: session.ID,
				ChatJID:   input.ChatJID,
				SenderJID: input.SenderJID,
				FromMe:    input.FromMe,
				MessageID: input.MessageID,
				MimeType:  input.MimeType,
				Timestamp: input.Timestamp,
			})
			if err := s.minio.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), input.MimeType); err != nil {
				logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", input.MessageID).Msg("Media retry upload: failed to upload to S3")
				return struct{}{}, nil
			}

			if s.msgRepo != nil {
				if err := s.msgRepo.UpdateMediaURL(ctx, session.ID, input.MessageID, key); err != nil {
					logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Str("mid", input.MessageID).Msg("Media retry upload: failed to persist media key")
				}
			}

			logger.Debug().Str("component", "service").Str("session", session.ID).Str("mid", input.MessageID).Msg("Media retry upload: media stored in S3")
			return struct{}{}, nil
		})
		if err != nil {
			logger.Warn().Str("component", "service").Err(err).Str("session", input.SessionID).Str("mid", input.MessageID).Msg("Media retry upload: runtime error")
		}
	})
}

func convertToOGG(input []byte) ([]byte, error) {
	// Parâmetros canônicos do WhatsApp PTT:
	// - codec libopus @ 48kHz mono, bitrate ~32k (o que o app oficial grava)
	// - application=voip melhora compressão de voz
	// Sem esses parâmetros o áudio aparece como arquivo em alguns clientes.
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-vn",
		"-c:a", "libopus",
		"-b:a", "32k",
		"-ar", "48000",
		"-ac", "1",
		"-application", "voip",
		"-f", "ogg", "pipe:1",
	)
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

// probeOggDurationSeconds returns the duration in seconds of an OGG/Opus
// stream using ffprobe. Returns 0 on any error — duration is best-effort
// metadata for WhatsApp clients to render the waveform length.
func probeOggDurationSeconds(input []byte) uint32 {
	cmd := exec.Command("ffprobe",
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
	)
	cmd.Stdin = bytes.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0
	}
	s := strings.TrimSpace(out.String())
	if s == "" {
		return 0
	}
	// Ex.: "4.547000"
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return uint32(f + 0.5)
}

func isOGGOpus(mimeType string) bool {
	return strings.Contains(strings.ToLower(mimeType), "ogg") ||
		strings.Contains(strings.ToLower(mimeType), "opus")
}

func checkFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}
