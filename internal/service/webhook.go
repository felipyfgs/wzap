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
	webhook := &model.Webhook{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		URL:       req.URL,
		Secret:    req.Secret,
		Events:    req.Events,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, nil
}

func (s *WebhookService) List(ctx context.Context, sessionID string) ([]model.Webhook, error) {
	return s.repo.FindBySessionID(ctx, sessionID)
}

func (s *WebhookService) Delete(ctx context.Context, sessionID, webhookID string) error {
	return s.repo.Delete(ctx, sessionID, webhookID)
}
