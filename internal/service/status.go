package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"
)

type MediaPresigner interface {
	GetPresignedURL(ctx context.Context, key string) (string, error)
}

type ContactNameGetter interface {
	GetContactName(ctx context.Context, sessionID, jid string) string
}

type StatusService struct {
	runtimeResolver *RuntimeResolver
	statusRepo      repo.StatusRepo
	mediaDownloader wa.MediaUploadFunc
	presigner       MediaPresigner
	nameGetter      ContactNameGetter
}

func (s *StatusService) SetMediaPresigner(p MediaPresigner)       { s.presigner = p }
func (s *StatusService) SetContactNameGetter(g ContactNameGetter) { s.nameGetter = g }

func (s *StatusService) enrichStatuses(ctx context.Context, sessionID string, statuses []model.Status) []model.Status {
	for i, st := range statuses {
		if s.presigner != nil && st.MediaURL != "" && len(st.MediaURL) <= 512 {
			if url, err := s.presigner.GetPresignedURL(ctx, st.MediaURL); err == nil {
				statuses[i].MediaURL = url
			}
		}
		if s.nameGetter != nil && st.SenderName == "" {
			statuses[i].SenderName = s.nameGetter.GetContactName(ctx, sessionID, st.SenderJID)
		}
	}
	return statuses
}

func (s *StatusService) SetMediaDownloader(fn wa.MediaUploadFunc) {
	s.mediaDownloader = fn
}

func (s *StatusService) RetryMissingMedia(ctx context.Context, sessionID string) {
	if s.mediaDownloader == nil || s.statusRepo == nil {
		return
	}

	statuses, err := s.statusRepo.FindWithMissingMedia(ctx, sessionID, 200)
	if err != nil {
		logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg("RetryMissingMedia: query failed")
		return
	}

	for _, st := range statuses {
		rawBytes, ok := st.Raw.([]byte)
		if !ok || len(rawBytes) == 0 {
			continue
		}

		msg := &waE2E.Message{}
		if err := protojson.Unmarshal(rawBytes, msg); err != nil {
			continue
		}

		const statusChat = "status@broadcast"
		switch {
		case msg.GetImageMessage() != nil:
			s.mediaDownloader(wa.MediaUploadInput{SessionID: sessionID, MessageID: st.ID, ChatJID: statusChat, SenderJID: st.SenderJID, MimeType: msg.GetImageMessage().GetMimetype(), FromMe: st.FromMe, Timestamp: st.Timestamp, Downloadable: msg.GetImageMessage()})
		case msg.GetVideoMessage() != nil:
			s.mediaDownloader(wa.MediaUploadInput{SessionID: sessionID, MessageID: st.ID, ChatJID: statusChat, SenderJID: st.SenderJID, MimeType: msg.GetVideoMessage().GetMimetype(), FromMe: st.FromMe, Timestamp: st.Timestamp, Downloadable: msg.GetVideoMessage()})
		}
	}

	if len(statuses) > 0 {
		logger.Info().Str("component", "service").Str("session", sessionID).Int("count", len(statuses)).Msg("RetryMissingMedia: triggered download for statuses")
	}
}

func NewStatusService(runtimeResolver *RuntimeResolver, statusRepo repo.StatusRepo) *StatusService {
	return &StatusService{
		runtimeResolver: runtimeResolver,
		statusRepo:      statusRepo,
	}
}

func (s *StatusService) persistStatus(sessionID, messageID, senderJID, statusType, body, mediaType string, ts time.Time) {
	status := &model.Status{
		ID:         messageID,
		SessionID:  sessionID,
		SenderJID:  senderJID,
		FromMe:     true,
		StatusType: statusType,
		Body:       body,
		MediaType:  mediaType,
		Timestamp:  ts,
		ExpiresAt:  ts.Add(24 * time.Hour),
		CreatedAt:  time.Now(),
	}
	if err := s.statusRepo.Save(context.Background(), status); err != nil {
		logger.Warn().Str("component", "service").Err(err).Str("session", sessionID).Msg("failed to persist status")
	}
}

func (s *StatusService) PersistStatus(input wa.PersistInput) {
	if s.statusRepo == nil {
		return
	}
	_ = s.statusRepo.Save(context.Background(), &model.Status{
		ID:         input.MessageID,
		SessionID:  input.SessionID,
		SenderJID:  input.SenderJID,
		FromMe:     input.FromMe,
		StatusType: input.MsgType,
		Body:       input.Body,
		MediaType:  input.MediaType,
		Timestamp:  time.Unix(input.Timestamp, 0),
		ExpiresAt:  time.Unix(input.Timestamp, 0).Add(24 * time.Hour),
		Raw:        input.Raw,
		CreatedAt:  time.Now(),
	})
}

func (s *StatusService) SendStatusText(ctx context.Context, sessionID string, req dto.StatusTextReq) (string, error) {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageStatusText)
	if err != nil {
		return "", err
	}

	return runConnectedRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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

	return runConnectedRuntime(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (string, error) {
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
	statuses, err := s.statusRepo.FindBySession(ctx, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.enrichStatuses(ctx, sessionID, statuses), nil
}

func (s *StatusService) ListContactStatus(ctx context.Context, sessionID, senderJID string) ([]model.Status, error) {
	statuses, err := s.statusRepo.FindBySender(ctx, sessionID, senderJID)
	if err != nil {
		return nil, err
	}
	return s.enrichStatuses(ctx, sessionID, statuses), nil
}
