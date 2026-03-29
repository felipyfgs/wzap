package service

import (
	"context"

	"go.mau.fi/whatsmeow/appstate"
	"wzap/internal/model"
)

type ChatService struct {
	engine *Engine
}

func NewChatService(engine *Engine) *ChatService {
	return &ChatService{engine: engine}
}

func (s *ChatService) Archive(ctx context.Context, sessionID string, req model.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildArchive(jid, true, client.Store.ID)
	return client.SendAppState(patch)
}

func (s *ChatService) Mute(ctx context.Context, sessionID string, req model.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildMute(jid, true, 8*60*60) // 8 hours mute as default
	return client.SendAppState(patch)
}

func (s *ChatService) Pin(ctx context.Context, sessionID string, req model.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildPin(jid, true)
	return client.SendAppState(patch)
}

func (s *ChatService) Unpin(ctx context.Context, sessionID string, req model.ChatActionReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildPin(jid, false)
	return client.SendAppState(patch)
}
