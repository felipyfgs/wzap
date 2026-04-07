package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/model"
)

func (s *MessageService) SendStatusText(ctx context.Context, sessionID string, req dto.SendStatusTextReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageStatusText)
	if err != nil {
		return "", err
	}

	return runConnectedSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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

		s.persistSent(session.ID, resp.ID, types.StatusBroadcastJID.String(), "status_text", req.Text, "", client)

		return resp.ID, nil
	})
}

func (s *MessageService) SendStatusMedia(ctx context.Context, sessionID string, req dto.SendStatusMediaReq, mediaType whatsmeow.MediaType) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageStatusMedia)
	if err != nil {
		return "", err
	}

	return runConnectedSessionRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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

		s.persistSent(session.ID, resp.ID, types.StatusBroadcastJID.String(), msgType, req.Caption, string(mediaType), client)

		return resp.ID, nil
	})
}
