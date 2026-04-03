package service

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"

	"wzap/internal/dto"
	"wzap/internal/wa"
)

type ChatService struct {
	engine *wa.Manager
}

func NewChatService(engine *wa.Manager) *ChatService {
	return &ChatService{engine: engine}
}

func (s *ChatService) Archive(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildArchive(jid, true, time.Now(), nil)
	return client.SendAppState(ctx, patch)
}

func (s *ChatService) Mute(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildMute(jid, true, 8*time.Hour)
	return client.SendAppState(ctx, patch)
}

func (s *ChatService) Pin(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildPin(jid, true)
	return client.SendAppState(ctx, patch)
}

func (s *ChatService) Unpin(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildPin(jid, false)
	return client.SendAppState(ctx, patch)
}

func (s *ChatService) Unarchive(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildArchive(jid, false, time.Now(), nil)
	return client.SendAppState(ctx, patch)
}

func (s *ChatService) Unmute(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildMute(jid, false, 0)
	return client.SendAppState(ctx, patch)
}

func (s *ChatService) DeleteChat(ctx context.Context, sessionID string, req dto.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	return client.SendAppState(ctx, appstate.BuildDeleteChat(jid, time.Now(), nil, true))
}

func (s *ChatService) MarkRead(ctx context.Context, sessionID string, req dto.ChatMarkReadReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	ids := make([]types.MessageID, len(req.MessageIDs))
	for i, id := range req.MessageIDs {
		ids[i] = types.MessageID(id)
	}

	return client.MarkRead(ctx, ids, time.Now(), jid, jid)
}

func (s *ChatService) MarkUnread(ctx context.Context, sessionID string, req dto.ChatMarkUnreadReq) error {
	return fmt.Errorf("marking a chat as unread is not supported by the WhatsApp protocol")
}
