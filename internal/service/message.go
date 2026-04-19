package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"
)

type MessageService struct {
	runtimeResolver *RuntimeResolver
	persistFn       wa.PersistFunc
}

func NewMessageService(engine *wa.Manager, sessRepo *repo.SessionRepository, runtimeResolver *RuntimeResolver) *MessageService {
	if runtimeResolver == nil {
		runtimeResolver = NewRuntimeResolver(sessRepo, engine)
	}
	return &MessageService{runtimeResolver: runtimeResolver}
}

func (s *MessageService) SetOnMessagePersist(fn wa.PersistFunc) {
	s.persistFn = fn
}

func (s *MessageService) persistSent(sessionID, messageID, chatJID, msgType, body, mediaType string, client *whatsmeow.Client) {
	if s.persistFn == nil {
		return
	}
	senderJID := ""
	if client.Store.ID != nil {
		senderJID = client.Store.ID.String()
	}
	s.persistFn(wa.PersistInput{
		SessionID: sessionID, MessageID: messageID, ChatJID: chatJID,
		SenderJID: senderJID, FromMe: true, MsgType: msgType,
		Body: body, MediaType: mediaType, Timestamp: time.Now().Unix(),
	})
	metrics.MessagesSent.Inc()
}

func (s *MessageService) SendText(ctx context.Context, sessionID string, req dto.SendTextReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageText)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(req.Body),
				ContextInfo: buildContextInfo(req.ReplyTo, req.MentionedJIDs),
			},
		}

		opts := buildSendOpts(req.CustomID)
		resp, err := client.SendMessage(ctx, jid, msg, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send text message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "text", req.Body, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendImage(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error) {
	return s.sendMedia(ctx, sessionID, req, whatsmeow.MediaImage)
}

func (s *MessageService) SendVideo(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error) {
	return s.sendMedia(ctx, sessionID, req, whatsmeow.MediaVideo)
}

func (s *MessageService) SendDocument(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error) {
	return s.sendMedia(ctx, sessionID, req, whatsmeow.MediaDocument)
}

func (s *MessageService) SendAudio(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error) {
	return s.sendMedia(ctx, sessionID, req, whatsmeow.MediaAudio)
}

func (s *MessageService) sendMedia(ctx context.Context, sessionID string, req dto.SendMediaReq, mediaType whatsmeow.MediaType) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageMedia)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		var data []byte
		if req.URL != "" {
			data, err = downloadURL(ctx, req.URL)
			if err != nil {
				return "", err
			}
		} else {
			data, err = base64.StdEncoding.DecodeString(req.Base64)
			if err != nil {
				return "", fmt.Errorf("invalid base64: %w", err)
			}
		}

		if mediaType == whatsmeow.MediaAudio && !isOGGOpus(req.MimeType) {
			if checkFFmpegAvailable() {
				convertedData, convErr := convertToOGG(data)
				if convErr != nil {
					logger.Warn().Str("component", "service").Err(convErr).Str("session", session.ID).Msg("Failed to convert audio to OGG, sending original")
				} else {
					data = convertedData
					// IMPORTANTE: WhatsApp mobile só renderiza como voice note
					// (PTT) quando o mimetype declara explicitamente o codec.
					// Sem `codecs=opus` o áudio aparece como anexo em iOS/Android.
					req.MimeType = "audio/ogg; codecs=opus"
					logger.Debug().Str("component", "service").Str("session", session.ID).Msg("Audio converted to OGG Opus format")
				}
			} else {
				logger.Warn().Str("component", "service").Str("session", session.ID).Msg("ffmpeg not available, sending audio without conversion")
			}
		}
		// Normaliza mimetype OGG/Opus já recebido sem codec declarado.
		if mediaType == whatsmeow.MediaAudio && strings.EqualFold(strings.TrimSpace(req.MimeType), "audio/ogg") {
			req.MimeType = "audio/ogg; codecs=opus"
		}

		uploaded, err := client.Upload(ctx, data, mediaType)
		if err != nil {
			return "", fmt.Errorf("failed to upload media: %w", err)
		}

		var msg waE2E.Message

		ci := buildContextInfo(req.ReplyTo, req.MentionedJIDs)

		switch mediaType { //nolint:exhaustive // sendMedia only accepts Image/Video/Audio/Document
		case whatsmeow.MediaImage:
			msg.ImageMessage = &waE2E.ImageMessage{
				Caption:       proto.String(req.Caption),
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				ContextInfo:   ci,
			}
		case whatsmeow.MediaVideo:
			msg.VideoMessage = &waE2E.VideoMessage{
				Caption:       proto.String(req.Caption),
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				ContextInfo:   ci,
			}
		case whatsmeow.MediaDocument:
			if req.FileName == "" {
				req.FileName = "document-" + uuid.NewString()
				ext, _ := mime.ExtensionsByType(req.MimeType)
				if len(ext) > 0 {
					req.FileName += ext[0]
				}
			}
			msg.DocumentMessage = &waE2E.DocumentMessage{
				Title:         proto.String(req.FileName),
				FileName:      proto.String(filepath.Base(req.FileName)),
				Mimetype:      proto.String(req.MimeType),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				ContextInfo:   ci,
			}
		case whatsmeow.MediaAudio:
			audioMsg := &waE2E.AudioMessage{
				Mimetype:      proto.String(req.MimeType),
				PTT:           proto.Bool(true),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				ContextInfo:   ci,
			}
			// Seconds é usado pelo WhatsApp para renderizar a duração e o
			// tamanho da waveform do voice note. Sem ele a barra fica "0:00".
			if secs := probeOggDurationSeconds(data); secs > 0 {
				audioMsg.Seconds = proto.Uint32(secs)
			}
			msg.AudioMessage = audioMsg
		}

		var msgType string
		switch mediaType { //nolint:exhaustive // sendMedia only accepts Image/Video/Audio/Document
		case whatsmeow.MediaImage:
			msgType = "image"
		case whatsmeow.MediaVideo:
			msgType = "video"
		case whatsmeow.MediaDocument:
			msgType = "document"
		case whatsmeow.MediaAudio:
			msgType = "audio"
		}

		opts := buildSendOpts(req.CustomID)
		resp, err := client.SendMessage(ctx, jid, &msg, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send media message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), msgType, req.Caption, req.MimeType, client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendContact(ctx context.Context, sessionID string, req dto.SendContactReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageContact)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		msg := &waE2E.Message{
			ContactMessage: &waE2E.ContactMessage{
				DisplayName: proto.String(req.Name),
				Vcard:       proto.String(req.Vcard),
			},
		}

		resp, err := client.SendMessage(ctx, jid, msg)
		if err != nil {
			return "", fmt.Errorf("failed to send contact message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "contact", req.Name, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendLocation(ctx context.Context, sessionID string, req dto.SendLocationReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageLocation)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		msg := &waE2E.Message{
			LocationMessage: &waE2E.LocationMessage{
				DegreesLatitude:  proto.Float64(req.Latitude),
				DegreesLongitude: proto.Float64(req.Longitude),
				Name:             proto.String(req.Name),
				Address:          proto.String(req.Address),
			},
		}

		resp, err := client.SendMessage(ctx, jid, msg)
		if err != nil {
			return "", fmt.Errorf("failed to send location message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "location", req.Name, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendLink(ctx context.Context, sessionID string, req dto.SendLinkReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageLink)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return "", err
		}

		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(req.URL),
				Title:       proto.String(req.Title),
				Description: proto.String(req.Description),
			},
		}

		resp, err := client.SendMessage(ctx, jid, msg)
		if err != nil {
			return "", fmt.Errorf("failed to send link message: %w", err)
		}

		s.persistSent(session.ID, resp.ID, jid.String(), "text", req.URL, "", client)

		return resp.ID, nil
	})
}
