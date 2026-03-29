package service

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"wzap/internal/dto"
	"wzap/internal/whatsapp"
)

type ChatService struct {
	engine *whatsapp.Engine
}

func NewChatService(engine *whatsapp.Engine) *ChatService {
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

	patch := appstate.BuildMute(jid, true, 8*60*60) // 8 hours mute as default
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
