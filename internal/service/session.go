package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"

	"github.com/google/uuid"
)

var sessionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type SessionService struct {
	repo        *repo.SessionRepository
	webhookRepo *repo.WebhookRepository
	engine      *wa.Manager
}

func NewSessionService(r *repo.SessionRepository, webhookRepo *repo.WebhookRepository, engine *wa.Manager) *SessionService {
	return &SessionService{
		repo:        r,
		webhookRepo: webhookRepo,
		engine:      engine,
	}
}

func (s *SessionService) Create(ctx context.Context, req dto.SessionCreateReq) (*dto.SessionCreatedResp, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if !sessionNameRegex.MatchString(req.Name) {
		return nil, fmt.Errorf("name must contain only letters, numbers, hyphens and underscores")
	}

	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = "sk_" + uuid.NewString()
	}

	now := time.Now()
	session := &model.Session{
		ID:        uuid.NewString(),
		Name:      req.Name,
		APIKey:    apiKey,
		Status:    "disconnected",
		Proxy:     req.Proxy,
		Settings:  req.Settings,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	resp := &dto.SessionCreatedResp{Session: *session}

	if req.Webhook != nil && req.Webhook.URL != "" {
		events := make([]string, 0, len(req.Webhook.Events))
		for _, e := range req.Webhook.Events {
			if !model.ValidEventTypes[e] {
				logger.Warn().Str("event", string(e)).Str("session", session.ID).Msg("Skipping invalid event type in inline webhook")
				continue
			}
			events = append(events, string(e))
		}
		wh := &model.Webhook{
			ID:        uuid.NewString(),
			SessionID: session.ID,
			URL:       req.Webhook.URL,
			Events:    events,
			Enabled:   true,
			CreatedAt: now,
		}
		if err := s.webhookRepo.Create(ctx, wh); err != nil {
			logger.Warn().Err(err).Str("session", session.ID).Msg("Failed to create inline webhook")
		} else {
			resp.Webhook = wh
		}
	}

	return resp, nil
}

func (s *SessionService) List(ctx context.Context) ([]dto.SessionResp, error) {
	sessions, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.SessionResp, len(sessions))
	for i, sess := range sessions {
		resp[i] = dto.SessionToResp(sess)
	}
	return resp, nil
}

func (s *SessionService) Get(ctx context.Context, id string) (*dto.SessionResp, error) {
	session, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := dto.SessionToResp(*session)
	return &resp, nil
}

func (s *SessionService) Delete(ctx context.Context, id string) error {
	if err := s.engine.Logout(ctx, id); err != nil {
		logger.Warn().Err(err).Str("session", id).Msg("Failed to logout session during delete")
	}
	return s.repo.Delete(ctx, id)
}

func (s *SessionService) SetStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}
