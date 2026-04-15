package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/repo"
)

type StatusService struct {
	runtimeResolver *RuntimeResolver
	statusRepo      repo.StatusRepo
}

func NewStatusService(runtimeResolver *RuntimeResolver, statusRepo repo.StatusRepo) *StatusService {
	return &StatusService{
		runtimeResolver: runtimeResolver,
		statusRepo:      statusRepo,
	}
}

func (s *StatusService) persistStatus(sessionID, messageID, senderJID, statusType, body, mediaType string, ts time.Time) {
	status := &model.Status{
		ID:        messageID,
		SessionID: sessionID,
		SenderJID: senderJID,
		FromMe:    true,
		StatusType: statusType,
		Body:      body,
		MediaType: mediaType,
		Timestamp: ts,
		ExpiresAt: ts.Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.statusRepo.Save(context.Background(), status); err != nil {
		_ = err
	}
}

func (s *StatusService) PersistStatusReceived(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw any) {
	if s.statusRepo == nil {
		return
	}
	_ = s.statusRepo.Save(context.Background(), &model.Status{
		ID:         messageID,
		SessionID:  sessionID,
		SenderJID:  senderJID,
		FromMe:     fromMe,
		StatusType: msgType,
		Body:       body,
		MediaType:  mediaType,
		Timestamp:  time.Unix(timestamp, 0),
		ExpiresAt:  time.Unix(timestamp, 0).Add(24 * time.Hour),
		Raw:        raw,
		CreatedAt:  time.Now(),
	})
}

func (s *StatusService) SendStatusText(ctx context.Context, sessionID string, req dto.StatusTextReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageStatusText)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String(req.Text),
			},
		}

		opts := buildSendOpts("")
		resp, err := client.SendMessage(ctx, types.StatusBroadcastJID, msg, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send status text: %w", err)
		}

		senderJID := ""
		if client.Store.ID != nil {
			senderJID = client.Store.ID.String()
		}
		s.persistStatus(session.ID, resp.ID, senderJID, "text", req.Text, "", time.Now())

		return resp.ID, nil
	})
}

func (s *StatusService) SendStatusMedia(ctx context.Context, sessionID string, req dto.StatusMediaReq, mediaType whatsmeow.MediaType) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageStatusMedia)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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

		opts := buildSendOpts("")
		resp, err := client.SendMessage(ctx, types.StatusBroadcastJID, msg, opts...)
		if err != nil {
			return "", fmt.Errorf("failed to send status media: %w", err)
		}

		msgType := "image"
		if mediaType == whatsmeow.MediaVideo {
			msgType = "video"
		}

		senderJID := ""
		if client.Store.ID != nil {
			senderJID = client.Store.ID.String()
		}
		s.persistStatus(session.ID, resp.ID, senderJID, msgType, req.Caption, req.MimeType, time.Now())

		return resp.ID, nil
	})
}

func (s *StatusService) ListStatus(ctx context.Context, sessionID string, limit, offset int) ([]model.Status, error) {
	return s.statusRepo.FindBySession(ctx, sessionID, limit, offset)
}

func (s *StatusService) ListContactStatus(ctx context.Context, sessionID, senderJID string) ([]model.Status, error) {
	return s.statusRepo.FindBySender(ctx, sessionID, senderJID)
}
