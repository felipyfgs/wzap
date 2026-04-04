package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	net_http "net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
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

func (s *MessageService) SendSticker(ctx context.Context, sessionID string, req dto.SendStickerReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		data, err := base64.StdEncoding.DecodeString(req.Base64)
		if err != nil {
			return "", fmt.Errorf("invalid base64: %w", err)
		}
		uploadResp, err := s.provider.UploadMedia(ctx, sessionID, "sticker.webp", req.MimeType, data)
		if err != nil {
			return "", fmt.Errorf("failed to upload sticker to cloud api: %w", err)
		}
		media := &cloudWA.MediaIDOrURL{ID: uploadResp.ID}
		resp, err := s.provider.SendSticker(ctx, sessionID, req.Phone, media)
		if err != nil {
			return "", fmt.Errorf("failed to send sticker via cloud api: %w", err)
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

	data, err := base64.StdEncoding.DecodeString(req.Base64)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return "", fmt.Errorf("failed to upload sticker: %w", err)
	}

	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			Mimetype:      proto.String(req.MimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send sticker message: %w", err)
	}

	s.persistSent(sessionID, resp.ID, jid.String(), "sticker", "", req.MimeType, client)

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

func (s *MessageService) EditMessage(ctx context.Context, sessionID string, req dto.EditMessageReq) (string, error) {
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

	newMsg := &waE2E.Message{
		Conversation: proto.String(req.Body),
	}

	msg := client.BuildEdit(jid, req.MessageID, newMsg)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to edit message: %w", err)
	}

	s.persistSent(sessionID, resp.ID, jid.String(), "text", req.Body, "", client)

	return resp.ID, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, sessionID string, req dto.DeleteMessageReq) (string, error) {
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

	msg := client.BuildRevoke(jid, *client.Store.ID, req.MessageID)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to delete message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) ReactMessage(ctx context.Context, sessionID string, req dto.ReactMessageReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		resp, err := s.provider.SendReaction(ctx, sessionID, req.Phone, req.MessageID, req.Reaction)
		if err != nil {
			return "", fmt.Errorf("failed to react via cloud api: %w", err)
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

	msg := client.BuildReaction(jid, *client.Store.ID, req.MessageID, req.Reaction)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to react message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) MarkRead(ctx context.Context, sessionID string, req dto.MarkReadReq) error {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.Engine == "cloud_api" {
		return s.provider.MarkRead(ctx, sessionID, req.MessageID)
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.Phone)
	if err != nil {
		return err
	}

	return client.MarkRead(ctx, []types.MessageID{req.MessageID}, time.Now(), jid, *client.Store.ID)
}

func (s *MessageService) SetPresence(ctx context.Context, sessionID string, req dto.SetPresenceReq) error {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.Engine == "cloud_api" {
		return errCloudAPINotSupported
	}

	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.Phone)
	if err != nil {
		return err
	}

	var presence types.ChatPresence
	var media types.ChatPresenceMedia
	switch req.State {
	case "typing":
		presence = types.ChatPresenceComposing
		media = types.ChatPresenceMediaText
	case "recording":
		presence = types.ChatPresenceComposing
		media = types.ChatPresenceMediaAudio
	case "paused":
		presence = types.ChatPresencePaused
		media = types.ChatPresenceMediaText
	default:
		return fmt.Errorf("invalid presence type: %s", req.State)
	}

	return client.SendChatPresence(ctx, jid, presence, media)
}

func buildContextInfo(reply *dto.ReplyContext, mentionedJIDs []string) *waE2E.ContextInfo {
	ci := &waE2E.ContextInfo{}

	if reply != nil && reply.MessageID != "" {
		ci.StanzaID = proto.String(reply.MessageID)
		if reply.Participant != "" {
			ci.Participant = proto.String(reply.Participant)
		}
		if len(reply.MentionedJID) > 0 {
			ci.MentionedJID = reply.MentionedJID
		}
	}

	if len(mentionedJIDs) > 0 {
		ci.MentionedJID = mentionedJIDs
	}

	if ci.StanzaID == nil && ci.Participant == nil && len(ci.MentionedJID) == 0 {
		return nil
	}

	return ci
}

func buildSendOpts(customID string) []whatsmeow.SendRequestExtra {
	if customID == "" {
		return nil
	}
	return []whatsmeow.SendRequestExtra{{ID: customID}}
}

func (s *MessageService) SendButton(ctx context.Context, sessionID string, req dto.SendButtonReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		buttons := make([]cloudWA.InteractiveButton, len(req.Buttons))
		for i, b := range req.Buttons {
			buttons[i] = cloudWA.InteractiveButton{
				Type:  "reply",
				Title: b.Text,
				ID:    b.ID,
			}
		}
		interactive := &cloudWA.Interactive{
			Type: "button",
			Action: &cloudWA.InteractiveAction{
				Buttons: buttons,
			},
			Body: &cloudWA.InteractiveBody{Text: req.Body},
		}
		if req.Footer != "" {
			interactive.Footer = &cloudWA.InteractiveFooter{Text: req.Footer}
		}
		resp, err := s.provider.SendInteractive(ctx, sessionID, req.Phone, interactive, buildSendOptsCloud(req.CustomID, req.ReplyTo)...)
		if err != nil {
			return "", fmt.Errorf("failed to send button via cloud api: %w", err)
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

	buttons := make([]*waE2E.ButtonsMessage_Button, len(req.Buttons))
	for i, b := range req.Buttons {
		buttons[i] = &waE2E.ButtonsMessage_Button{
			ButtonID: proto.String(b.ID),
			ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{
				DisplayText: proto.String(b.Text),
			},
			Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum(),
		}
	}

	msg := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ButtonsMessage: &waE2E.ButtonsMessage{
					ContentText: proto.String(req.Body),
					FooterText:  proto.String(req.Footer),
					Buttons:     buttons,
					HeaderType:  waE2E.ButtonsMessage_EMPTY.Enum(),
					ContextInfo: buildContextInfo(req.ReplyTo, req.MentionedJIDs),
				},
			},
		},
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, jid, msg, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to send button message: %w", err)
	}

	s.persistSent(sessionID, resp.ID, jid.String(), "buttons", req.Body, "", client)

	return resp.ID, nil
}

func (s *MessageService) SendList(ctx context.Context, sessionID string, req dto.SendListReq) (string, error) {
	session, err := s.sessRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.Engine == "cloud_api" {
		sections := make([]cloudWA.InteractiveSection, len(req.Sections))
		for i, sec := range req.Sections {
			rows := make([]cloudWA.InteractiveSectionRow, len(sec.Rows))
			for j, r := range sec.Rows {
				rows[j] = cloudWA.InteractiveSectionRow{
					ID:          r.ID,
					Title:       r.Title,
					Description: r.Description,
				}
			}
			sections[i] = cloudWA.InteractiveSection{
				Title: sec.Title,
				Rows:  rows,
			}
		}
		interactive := &cloudWA.Interactive{
			Type: "list",
			Action: &cloudWA.InteractiveAction{
				Button:   req.ButtonText,
				Sections: sections,
			},
			Body:   &cloudWA.InteractiveBody{Text: req.Body},
			Header: &cloudWA.InteractiveHeader{Type: "text", Text: req.Title},
		}
		if req.Footer != "" {
			interactive.Footer = &cloudWA.InteractiveFooter{Text: req.Footer}
		}
		resp, err := s.provider.SendInteractive(ctx, sessionID, req.Phone, interactive, buildSendOptsCloud(req.CustomID, req.ReplyTo)...)
		if err != nil {
			return "", fmt.Errorf("failed to send list via cloud api: %w", err)
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

	sections := make([]*waE2E.ListMessage_Section, len(req.Sections))
	for i, sec := range req.Sections {
		rows := make([]*waE2E.ListMessage_Row, len(sec.Rows))
		for j, r := range sec.Rows {
			rows[j] = &waE2E.ListMessage_Row{
				RowID:       proto.String(r.ID),
				Title:       proto.String(r.Title),
				Description: proto.String(r.Description),
			}
		}
		sections[i] = &waE2E.ListMessage_Section{
			Title: proto.String(sec.Title),
			Rows:  rows,
		}
	}

	msg := &waE2E.Message{
		ListMessage: &waE2E.ListMessage{
			Title:       proto.String(req.Title),
			Description: proto.String(req.Body),
			FooterText:  proto.String(req.Footer),
			ButtonText:  proto.String(req.ButtonText),
			ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    sections,
			ContextInfo: buildContextInfo(req.ReplyTo, req.MentionedJIDs),
		},
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, jid, msg, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to send list message: %w", err)
	}

	s.persistSent(sessionID, resp.ID, jid.String(), "list", req.Title, "", client)

	return resp.ID, nil
}

func (s *MessageService) SendPoll(ctx context.Context, sessionID string, req dto.SendPollReq) (string, error) {
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

	msg := client.BuildPollCreation(req.Name, req.Options, req.SelectableCount)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send poll message: %w", err)
	}

	s.persistSent(sessionID, resp.ID, jid.String(), "poll", req.Name, "", client)

	return resp.ID, nil
}

func (s *MessageService) SendStatusText(ctx context.Context, sessionID string, req dto.SendStatusTextReq) (string, error) {
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
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(req.Text),
		},
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, types.StatusBroadcastJID, msg, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to send status text: %w", err)
	}

	s.persistSent(sessionID, resp.ID, types.StatusBroadcastJID.String(), "status_text", req.Text, "", client)

	return resp.ID, nil
}

func (s *MessageService) SendStatusMedia(ctx context.Context, sessionID string, req dto.SendStatusMediaReq, mediaType whatsmeow.MediaType) (string, error) {
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
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	var data []byte
	if req.Base64 != "" {
		data, err = base64.StdEncoding.DecodeString(req.Base64)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64: %w", err)
		}
	} else if req.URL != "" {
		data, err = downloadURL(req.URL)
		if err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("either base64 or url is required")
	}

	uploaded, err := client.Upload(ctx, data, mediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	msg := &waE2E.Message{}

	switch mediaType {
	case whatsmeow.MediaImage:
		msg.ImageMessage = &waE2E.ImageMessage{
			URL:           proto.String(uploaded.URL),
			Mimetype:      proto.String(req.MimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			Caption:       proto.String(req.Caption),
		}
	case whatsmeow.MediaVideo:
		msg.VideoMessage = &waE2E.VideoMessage{
			URL:           proto.String(uploaded.URL),
			Mimetype:      proto.String(req.MimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			Caption:       proto.String(req.Caption),
		}
	default:
		return "", fmt.Errorf("unsupported media type for status")
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, types.StatusBroadcastJID, msg, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to send status media: %w", err)
	}

	msgType := "status_image"
	if mediaType == whatsmeow.MediaVideo {
		msgType = "status_video"
	}

	s.persistSent(sessionID, resp.ID, types.StatusBroadcastJID.String(), msgType, req.Caption, string(mediaType), client)

	return resp.ID, nil
}

func (s *MessageService) ForwardMessage(ctx context.Context, sessionID string, req dto.ForwardMessageReq) (string, error) {
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
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	destJID, err := parseJID(req.Phone)
	if err != nil {
		return "", err
	}

	msgID := client.GenerateMessageID()
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			ContextInfo: &waE2E.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(1),
				StanzaID:        proto.String(req.MessageID),
				RemoteJID:       proto.String(req.FromJID),
			},
		},
	}

	resp, err := client.SendMessage(ctx, destJID, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		return "", fmt.Errorf("failed to forward message: %w", err)
	}

	s.persistSent(sessionID, resp.ID, destJID.String(), "forward", "", "", client)

	return resp.ID, nil
}

func downloadURL(url string) ([]byte, error) {
	httpClient := &net_http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download from url: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download body: %w", err)
	}
	return data, nil
}

func parseJID(target string) (types.JID, error) {
	if !strings.Contains(target, "@") {
		return types.NewJID(target, types.DefaultUserServer), nil
	}
	jid, err := types.ParseJID(target)
	if err != nil {
		return types.JID{}, fmt.Errorf("invalid JID: %w", err)
	}
	return jid, nil
}

func buildSendOptsCloud(customID string, reply *dto.ReplyContext) []cloudWA.SendOption {
	var opts []cloudWA.SendOption
	if customID != "" {
		opts = append(opts, cloudWA.WithCustomID(customID))
	}
	if reply != nil && reply.MessageID != "" {
		opts = append(opts, cloudWA.WithReplyTo(reply.MessageID))
	}
	return opts
}

var errCloudAPINotSupported = fmt.Errorf("operation not supported for cloud_api engine")
