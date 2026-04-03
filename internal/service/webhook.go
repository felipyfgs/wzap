package service

import (
	"context"
	"fmt"
	"time"

	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/repo"

	"github.com/google/uuid"
)

type WebhookService struct {
	repo *repo.WebhookRepository
}

func NewWebhookService(repo *repo.WebhookRepository) *WebhookService {
	return &WebhookService{repo: repo}
}

func (s *WebhookService) Create(ctx context.Context, sessionID string, req dto.CreateWebhookReq) (*model.Webhook, error) {
	events := make([]string, 0, len(req.Events))
	for _, e := range req.Events {
		if !model.IsValidEventType(model.EventType(e)) {
			return nil, fmt.Errorf("invalid event type: %s", e)
		}
		events = append(events, e)
	}
	webhook := &model.Webhook{
		ID:          uuid.NewString(),
		SessionID:   sessionID,
		URL:         req.URL,
		Secret:      req.Secret,
		Events:      events,
		EventURLs:   req.EventURLs,
		Enabled:     true,
		NATSEnabled: req.NATSEnabled,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, nil
}

func (s *WebhookService) List(ctx context.Context, sessionID string) ([]model.Webhook, error) {
	return s.repo.FindBySessionID(ctx, sessionID)
}

func (s *WebhookService) Update(ctx context.Context, sessionID, webhookID string, req dto.UpdateWebhookReq) (*model.Webhook, error) {
	wh, err := s.repo.FindByID(ctx, sessionID, webhookID)
	if err != nil {
		return nil, err
	}

	if req.URL != nil {
		wh.URL = *req.URL
	}
	if req.Secret != nil {
		wh.Secret = *req.Secret
	}
	if req.Events != nil {
		wh.Events = req.Events
	}
	if req.EventURLs != nil {
		wh.EventURLs = req.EventURLs
	}
	if req.Enabled != nil {
		wh.Enabled = *req.Enabled
	}
	if req.NATSEnabled != nil {
		wh.NATSEnabled = *req.NATSEnabled
	}

	if err := s.repo.Update(ctx, wh); err != nil {
		return nil, err
	}
	return wh, nil
}

func (s *WebhookService) Delete(ctx context.Context, sessionID, webhookID string) error {
	return s.repo.Delete(ctx, sessionID, webhookID)
}
