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

	jid, err := parseJID(req.Phone)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(req.Body),
			ContextInfo: buildContextInfo(req.ReplyTo),
		},
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, jid, msg, opts...)
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

	uploaded, err := client.Upload(ctx, data, mediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	var msg waE2E.Message

	ci := buildContextInfo(req.ReplyTo)

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

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, jid, &msg, opts...)
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

	return resp.ID, nil
}

func (s *MessageService) SendLocation(ctx context.Context, sessionID string, req dto.SendLocationReq) (string, error) {
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

	return resp.ID, nil
}

func (s *MessageService) SendPoll(ctx context.Context, sessionID string, req dto.SendPollReq) (string, error) {
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

	return resp.ID, nil
}

func (s *MessageService) SendSticker(ctx context.Context, sessionID string, req dto.SendStickerReq) (string, error) {
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

	return resp.ID, nil
}

func (s *MessageService) SendLink(ctx context.Context, sessionID string, req dto.SendLinkReq) (string, error) {
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

	return resp.ID, nil
}

func (s *MessageService) EditMessage(ctx context.Context, sessionID string, req dto.EditMessageReq) (string, error) {
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

	return resp.ID, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, sessionID string, req dto.DeleteMessageReq) (string, error) {
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

func buildContextInfo(reply *dto.ReplyContext) *waE2E.ContextInfo {
	if reply == nil || reply.MessageID == "" {
		return nil
	}
	ci := &waE2E.ContextInfo{
		StanzaID: proto.String(reply.MessageID),
	}
	if reply.Participant != "" {
		ci.Participant = proto.String(reply.Participant)
	}
	if len(reply.MentionedJID) > 0 {
		ci.MentionedJID = reply.MentionedJID
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
					ContextInfo: buildContextInfo(req.ReplyTo),
				},
			},
		},
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, jid, msg, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to send button message: %w", err)
	}

	return resp.ID, nil
}

func (s *MessageService) SendList(ctx context.Context, sessionID string, req dto.SendListReq) (string, error) {
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
			ContextInfo: buildContextInfo(req.ReplyTo),
		},
	}

	opts := buildSendOpts(req.CustomID)
	resp, err := client.SendMessage(ctx, jid, msg, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to send list message: %w", err)
	}

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
