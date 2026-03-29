package service

import (
	"context"
	"fmt"
	"strconv"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"wzap/internal/model"
)

type LabelService struct {
	engine *Engine
}

func NewLabelService(engine *Engine) *LabelService {
	return &LabelService{engine: engine}
}

func (s *LabelService) AddToChat(ctx context.Context, sessionID string, req model.LabelChatReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	// The LabelID needs to be parsed to string usually, here keeping as is for app state
	patch := appstate.BuildLabelAssociationAction(req.LabelID, jid.String(), "", true)
	return client.SendAppState(patch)
}

func (s *LabelService) RemoveFromChat(ctx context.Context, sessionID string, req model.LabelChatReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelAssociationAction(req.LabelID, jid.String(), "", false)
	return client.SendAppState(patch)
}

func (s *LabelService) AddToMessage(ctx context.Context, sessionID string, req model.LabelMessageReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelAssociationAction(req.LabelID, jid.String(), req.MessageID, true)
	return client.SendAppState(patch)
}

func (s *LabelService) RemoveFromMessage(ctx context.Context, sessionID string, req model.LabelMessageReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := parseJID(req.JID)
	if err != nil {
		return err
	}

	patch := appstate.BuildLabelAssociationAction(req.LabelID, jid.String(), req.MessageID, false)
	return client.SendAppState(patch)
}

func (s *LabelService) EditLabel(ctx context.Context, sessionID string, req model.EditLabelReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	labelIDInt, err := strconv.Atoi(req.LabelID)
	if err != nil {
		return fmt.Errorf("invalid label ID, must be int: %w", err)
	}

	patch := appstate.BuildLabelEditAction(req.LabelID, req.Name, int32(req.Color), int32(labelIDInt), req.Deleted)
	return client.SendAppState(patch)
}
