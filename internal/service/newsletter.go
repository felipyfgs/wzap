package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"wzap/internal/model"
)

type NewsletterService struct {
	engine *Engine
}

func NewNewsletterService(engine *Engine) *NewsletterService {
	return &NewsletterService{engine: engine}
}

func (s *NewsletterService) Create(ctx context.Context, sessionID string, req model.CreateNewsletterReq) (*types.NewsletterMetadata, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	var pictureBytes []byte
	if req.Picture != "" {
		pictureBytes, err = base64.StdEncoding.DecodeString(req.Picture)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 picture: %w", err)
		}
	}

	params := whatsmeow.CreateNewsletterParams{
		Name:        req.Name,
		Description: req.Description,
		Picture:     pictureBytes,
	}

	return client.CreateNewsletter(params)
}

func (s *NewsletterService) GetInfo(ctx context.Context, sessionID string, jidStr string) (*types.NewsletterMetadata, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, err
	}

	return client.GetNewsletterInfo(jid)
}

func (s *NewsletterService) GetInvite(ctx context.Context, sessionID string, code string) (*types.NewsletterMetadata, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	return client.GetNewsletterInfoWithInvite(code)
}

func (s *NewsletterService) List(ctx context.Context, sessionID string) ([]*types.NewsletterMetadata, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	return client.GetSubscribedNewsletters()
}

func (s *NewsletterService) Messages(ctx context.Context, sessionID string, req model.NewsletterMessageReq) ([]*types.NewsletterMessage, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return nil, err
	}

	params := &whatsmeow.GetNewsletterMessagesParams{
		Count:  req.Count,
		Before: types.MessageServerID(req.BeforeID),
	}

	return client.GetNewsletterMessages(jid, params)
}

func (s *NewsletterService) Subscribe(ctx context.Context, sessionID string, jidStr string) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}

	return client.FollowNewsletter(jid)
}
