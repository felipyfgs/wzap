package service

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"

	"wzap/internal/dto"
)

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
