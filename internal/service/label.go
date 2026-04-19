package service

import (
	"context"

	"wzap/internal/dto"
	"wzap/internal/wa"

	"go.mau.fi/whatsmeow/appstate"
)

type LabelService struct {
	engine *wa.Manager
}

func NewLabelService(engine *wa.Manager) *LabelService {
	return &LabelService{engine: engine}
}

func (s *LabelService) AddToChat(ctx context.Context, sessionID string, req dto.LabelChatReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	// The LabelID needs to be parsed to string usually, here keeping as is for app state
	patch := appstate.BuildLabelChat(jid, req.LabelID, true)
	return client.SendAppState(ctx, patch)
}

func (s *LabelService) RemoveFromChat(ctx context.Context, sessionID string, req dto.LabelChatReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelChat(jid, req.LabelID, false)
	return client.SendAppState(ctx, patch)
}

func (s *LabelService) AddToMessage(ctx context.Context, sessionID string, req dto.LabelMessageReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelMessage(jid, req.LabelID, req.MessageID, true)
	return client.SendAppState(ctx, patch)
}

func (s *LabelService) RemoveFromMessage(ctx context.Context, sessionID string, req dto.LabelMessageReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelMessage(jid, req.LabelID, req.MessageID, false)
	return client.SendAppState(ctx, patch)
}

func (s *LabelService) EditLabel(ctx context.Context, sessionID string, req dto.EditLabelReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelEdit(req.LabelID, req.Name, req.Color, req.Deleted)
	return client.SendAppState(ctx, patch)
}
