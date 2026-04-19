package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"wzap/internal/dto"
	"wzap/internal/wa"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type NewsletterService struct {
	engine *wa.Manager
}

func NewNewsletterService(engine *wa.Manager) *NewsletterService {
	return &NewsletterService{engine: engine}
}

func (s *NewsletterService) Create(ctx context.Context, sessionID string, req dto.CreateNewsletterReq) (*types.NewsletterMetadata, error) {
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

	return client.CreateNewsletter(ctx, params)
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

	return client.GetNewsletterInfo(ctx, jid)
}

func (s *NewsletterService) GetInvite(ctx context.Context, sessionID string, code string) (*types.NewsletterMetadata, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	return client.GetNewsletterInfoWithInvite(ctx, code)
}

func (s *NewsletterService) List(ctx context.Context, sessionID string) ([]*types.NewsletterMetadata, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	return client.GetSubscribedNewsletters(ctx)
}

func (s *NewsletterService) Messages(ctx context.Context, sessionID string, req dto.NewsletterMessageReq) ([]*types.NewsletterMessage, error) {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.NewsletterJID)
	if err != nil {
		return nil, err
	}

	params := &whatsmeow.GetNewsletterMessagesParams{
		Count:  req.Count,
		Before: req.BeforeID,
	}

	return client.GetNewsletterMessages(ctx, jid, params)
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

	return client.FollowNewsletter(ctx, jid)
}

func (s *NewsletterService) Unsubscribe(ctx context.Context, sessionID string, jidStr string) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}

	return client.UnfollowNewsletter(ctx, jid)
}

func (s *NewsletterService) Mute(ctx context.Context, sessionID, jidStr string, mute bool) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}

	return client.NewsletterToggleMute(ctx, jid, mute)
}

func (s *NewsletterService) React(ctx context.Context, sessionID string, req dto.NewsletterReactReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return err
	}

	return client.NewsletterSendReaction(ctx, jid, req.ServerID, req.Reaction, req.MessageID)
}

func (s *NewsletterService) MarkViewed(ctx context.Context, sessionID string, req dto.NewsletterViewReq) error {
	client, err := s.engine.GetClient(sessionID)
	if err != nil {
		return err
	}

	jid, err := types.ParseJID(req.JID)
	if err != nil {
		return err
	}

	serverIDs := make([]types.MessageServerID, len(req.ServerIDs))
	copy(serverIDs, req.ServerIDs)

	return client.NewsletterMarkViewed(ctx, jid, serverIDs)
}
