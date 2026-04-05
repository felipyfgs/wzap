package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/repo"
	"wzap/internal/wa"
)

type MessageService struct {
	engine    *wa.Manager
	provider  *cloudWA.Client
	sessRepo  *repo.SessionRepository
	persistFn wa.MessagePersistFunc
}

func NewMessageService(engine *wa.Manager, provider *cloudWA.Client, sessRepo *repo.SessionRepository) *MessageService {
	return &MessageService{engine: engine, provider: provider, sessRepo: sessRepo}
}

func (s *MessageService) SetMessagePersist(fn wa.MessagePersistFunc) {
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
	s.persistFn(sessionID, messageID, chatJID, senderJID, true, msgType, body, mediaType, time.Now().Unix(), nil)
	metrics.MessagesSent.Inc()
}

func (s *MessageService) persistSentCloud(sessionID, messageID, phone, msgType, body, mediaType string) {
	if s.persistFn == nil {
		return
	}
	s.persistFn(sessionID, messageID, phone, "", true, msgType, body, mediaType, time.Now().Unix(), nil)
	metrics.MessagesSent.Inc()
}

func (s *MessageService) SendText(ctx context.Context, sessionID string, req dto.SendTextReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		opts := buildSendOptsCloud(req.CustomID, req.ReplyTo)
		resp, err := s.provider.SendText(ctx, sessionID, req.Phone, req.Body, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send text message via cloud api: %w", err)
		}
		return resp.MessageID, nil
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

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

	s.persistSent(sessionID, resp.ID, jid.String(), "text", req.Body, "", client)

	return resp.ID, nil
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
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		return s.sendMediaCloud(ctx, sessionID, req, mediaType)
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	jid, err := parseJID(req.Phone)
	if err != nil {
		return "", err
	}

	var data []byte
	if req.URL != "" {
		data, err = downloadURL(req.URL)
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
				logger.Warn().Err(convErr).Str("session", sessionID).Msg("Failed to convert audio to OGG, sending original")
			} else {
				data = convertedData
				req.MimeType = "audio/ogg"
				logger.Debug().Str("session", sessionID).Msg("Audio converted to OGG Opus format")
			}
		} else {
			logger.Warn().Str("session", sessionID).Msg("ffmpeg not available, sending audio without conversion")
		}
	}

	uploaded, err := client.Upload(ctx, data, mediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	var msg waE2E.Message

	ci := buildContextInfo(req.ReplyTo, req.MentionedJIDs)

	switch mediaType {
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
		msg.AudioMessage = &waE2E.AudioMessage{
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
	}

	var msgType string
	switch mediaType {
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

	s.persistSent(sessionID, resp.ID, jid.String(), msgType, req.Caption, req.MimeType, client)

	return resp.ID, nil
}

func (s *MessageService) sendMediaCloud(ctx context.Context, sessionID string, req dto.SendMediaReq, mediaType whatsmeow.MediaType) (string, error) {
	var data []byte
	var err error
	if req.URL != "" {
		data, err = downloadURL(req.URL)
		if err != nil {
			return "", err
		}
	} else {
		data, err = base64.StdEncoding.DecodeString(req.Base64)
		if err != nil {
			return "", fmt.Errorf("invalid base64: %w", err)
		}
	}

	if req.FileName == "" {
		req.FileName = "file"
		ext, _ := mime.ExtensionsByType(req.MimeType)
		if len(ext) > 0 {
			req.FileName += ext[0]
		}
	}

	uploadResp, err := s.provider.UploadMedia(ctx, sessionID, req.FileName, req.MimeType, data)
	if err != nil {
		return "", fmt.Errorf("failed to upload media to cloud api: %w", err)
	}

	media := &cloudWA.MediaIDOrURL{
		ID:       uploadResp.ID,
		Caption:  req.Caption,
		Filename: req.FileName,
	}

	opts := buildSendOptsCloud(req.CustomID, req.ReplyTo)

	var resp *cloudWA.MessageResponse
	switch mediaType {
	case whatsmeow.MediaImage:
		resp, err = s.provider.SendImage(ctx, sessionID, req.Phone, media, opts...)
	case whatsmeow.MediaVideo:
		resp, err = s.provider.SendVideo(ctx, sessionID, req.Phone, media, opts...)
	case whatsmeow.MediaDocument:
		resp, err = s.provider.SendDocument(ctx, sessionID, req.Phone, media, opts...)
	case whatsmeow.MediaAudio:
		resp, err = s.provider.SendAudio(ctx, sessionID, req.Phone, media, opts...)
	default:
		return "", fmt.Errorf("unsupported media type for cloud api: %s", mediaType)
	}
	if err != nil {
		return "", fmt.Errorf("failed to send media via cloud api: %w", err)
	}

	var msgType string
	switch mediaType {
	case whatsmeow.MediaImage:
		msgType = "image"
	case whatsmeow.MediaVideo:
		msgType = "video"
	case whatsmeow.MediaDocument:
		msgType = "document"
	case whatsmeow.MediaAudio:
		msgType = "audio"
	}

	s.persistSentCloud(sessionID, resp.MessageID, req.Phone, msgType, req.Caption, req.MimeType)

	return resp.MessageID, nil
}

func (s *MessageService) SendContact(ctx context.Context, sessionID string, req dto.SendContactReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		return "", errCloudAPINotSupported
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

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

	s.persistSent(sessionID, resp.ID, jid.String(), "contact", req.Name, "", client)

	return resp.ID, nil
}

func (s *MessageService) SendLocation(ctx context.Context, sessionID string, req dto.SendLocationReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		resp, err := s.provider.SendLocation(ctx, sessionID, req.Phone, req.Latitude, req.Longitude, req.Name, req.Address)
		if err != nil {
			return "", fmt.Errorf("failed to send location via cloud api: %w", err)
		}
		return resp.MessageID, nil
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

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

	s.persistSent(sessionID, resp.ID, jid.String(), "location", req.Name, "", client)

	return resp.ID, nil
}

func (s *MessageService) SendLink(ctx context.Context, sessionID string, req dto.SendLinkReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		opts := []cloudWA.SendOption{cloudWA.WithPreviewURL(true)}
		resp, err := s.provider.SendText(ctx, sessionID, req.Phone, req.URL, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send link via cloud api: %w", err)
		}
		return resp.MessageID, nil
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

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

	s.persistSent(sessionID, resp.ID, jid.String(), "text", req.URL, "", client)

	return resp.ID, nil
}
