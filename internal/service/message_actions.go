package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/model"
)

func (s *MessageService) EditMessage(ctx context.Context, sessionID string, req dto.EditMessageReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageEdit)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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

		s.persistSent(session.ID, resp.ID, jid.String(), "text", req.Body, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) DeleteMessage(ctx context.Context, sessionID string, req dto.DeleteMessageReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageDelete)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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
	})
}

func (s *MessageService) ReactMessage(ctx context.Context, sessionID string, req dto.ReactMessageReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageReaction)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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
	})
}

func (s *MessageService) SendSticker(ctx context.Context, sessionID string, req dto.SendStickerReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageSticker)
	if err != nil {
		return "", err
	}

	return runSessionRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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

		s.persistSent(session.ID, resp.ID, jid.String(), "sticker", "", req.MimeType, client)

		return resp.ID, nil
	})
}
