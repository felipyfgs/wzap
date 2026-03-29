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
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/wa"
)

type MessageService struct {
	engine *wa.Manager
}

func NewMessageService(engine *wa.Manager) *MessageService {
	return &MessageService{engine: engine}
}

func (s *MessageService) SendText(ctx context.Context, sessionID string, req dto.SendTextReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		Conversation: proto.String(req.Text),
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send text message: %w", err)
	}

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
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}
	if !client.IsConnected() {
		return "", fmt.Errorf("client not connected")
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(req.Base64)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	uploaded, err := client.Upload(ctx, data, mediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	var msg waProto.Message

	switch mediaType {
	case whatsmeow.MediaImage:
		msg.ImageMessage = &waProto.ImageMessage{
			Caption:       proto.String(req.Caption),
			Mimetype:      proto.String(req.MimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	case whatsmeow.MediaVideo:
		msg.VideoMessage = &waProto.VideoMessage{
			Caption:       proto.String(req.Caption),
			Mimetype:      proto.String(req.MimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	case whatsmeow.MediaDocument:
		if req.Filename == "" {
			req.Filename = "document-" + uuid.NewString()
			ext, _ := mime.ExtensionsByType(req.MimeType)
			if len(ext) > 0 {
				req.Filename += ext[0]
			}
		}
		msg.DocumentMessage = &waProto.DocumentMessage{
			Title:         proto.String(req.Filename),
			FileName:      proto.String(filepath.Base(req.Filename)),
			Mimetype:      proto.String(req.MimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	case whatsmeow.MediaAudio:
		msg.AudioMessage = &waProto.AudioMessage{
			Mimetype:      proto.String(req.MimeType),
			PTT:           proto.Bool(true),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	}

	resp, err := client.SendMessage(ctx, jid, &msg)
	if err != nil {
		return "", fmt.Errorf("failed to send media message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) SendContact(ctx context.Context, sessionID string, req dto.SendContactReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		ContactMessage: &waProto.ContactMessage{
			DisplayName: proto.String(req.Name),
			Vcard:       proto.String(req.Vcard),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send contact message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) SendLocation(ctx context.Context, sessionID string, req dto.SendLocationReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		LocationMessage: &waProto.LocationMessage{
			DegreesLatitude:  proto.Float64(req.Lat),
			DegreesLongitude: proto.Float64(req.Lng),
			Name:             proto.String(req.Name),
			Address:          proto.String(req.Address),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send location message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) SendPoll(ctx context.Context, sessionID string, req dto.SendPollReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	msg := client.BuildPollCreation(req.Name, req.Options, req.SelectableCount)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send poll message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) SendSticker(ctx context.Context, sessionID string, req dto.SendStickerReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
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

	msg := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
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

	return resp.ID, nil
}

func (s *MessageService) SendLink(ctx context.Context, sessionID string, req dto.SendLinkReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        proto.String(req.URL),
			Title:       proto.String(req.Title),
			Description: proto.String(req.Description),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send link message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) EditMessage(ctx context.Context, sessionID string, req dto.EditMessageReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return "", err
	}

	newMsg := &waProto.Message{
		Conversation: proto.String(req.Text),
	}

	msg := client.BuildEdit(jid, req.MessageID, newMsg)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to edit message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, sessionID string, req dto.DeleteMessageReq) (string, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
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
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	jid, err := parseJID(req.JID)
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
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	return client.MarkRead(ctx, []types.MessageID{req.MessageID}, time.Now(), jid, *client.Store.ID)
}

func (s *MessageService) SetPresence(ctx context.Context, sessionID string, req dto.SetPresenceReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	var presence types.ChatPresence
	var media types.ChatPresenceMedia
	switch req.Presence {
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
		return fmt.Errorf("invalid presence type: %s", req.Presence)
	}

	return client.SendChatPresence(ctx, jid, presence, media)
}

func parseJID(target string) (types.JID, error) {
	jid, err := types.ParseJID(target)
	if err != nil {
		// If not a full JID, treat as phone number
		if !strings.Contains(target, "@") {
			return types.NewJID(target, types.DefaultUserServer), nil
		}
		return types.JID{}, fmt.Errorf("invalid JID: %w", err)
	}
	return jid, nil
}
