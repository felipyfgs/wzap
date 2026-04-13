package service

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	cloudWA "wzap/internal/provider/whatsapp"

	"wzap/internal/dto"
	"wzap/internal/model"
)

func (s *MessageService) MarkRead(ctx context.Context, sessionID string, req dto.MarkReadReq) error {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessageMarkRead)
	if err != nil {
		return err
	}

	return runRuntimeErr(ctx, runtime.SessionRuntime, func(ctx context.Context, session *model.Session, provider *cloudWA.Client) error {
		return provider.MarkRead(ctx, session.ID, req.MessageID)
	}, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) error {
		jid, err := parseJID(req.Phone)
		if err != nil {
			return err
		}

		return client.MarkRead(ctx, []types.MessageID{req.MessageID}, time.Now(), jid, *client.Store.ID)
	})
}

func (s *MessageService) SetPresence(ctx context.Context, sessionID string, req dto.SetPresenceReq) error {
	runtime, err := s.runtimeResolver.ResolveMessage(ctx, sessionID, model.CapabilityMessagePresence)
	if err != nil {
		return err
	}

	return runRuntimeErr(ctx, runtime.SessionRuntime, nil, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) error {
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
	})
}
