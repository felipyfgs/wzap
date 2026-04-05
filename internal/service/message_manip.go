package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	cloudWA "wzap/internal/provider/whatsapp"
)

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
