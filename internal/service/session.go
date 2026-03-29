package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

	token := req.Token
	if token == "" {
		token = "sk_" + uuid.NewString()
	}

	now := time.Now()
	session := &model.Session{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Token:     token,
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
				log.Warn().Str("event", string(e)).Str("session", session.ID).Msg("Skipping invalid event type in inline webhook")
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
			log.Warn().Err(err).Str("session", session.ID).Msg("Failed to create inline webhook")
		} else {
			resp.Webhook = wh
		}
	}

	return resp, nil
}

func (s *SessionService) List(ctx context.Context) ([]model.Session, error) {
	return s.repo.FindAll(ctx)
}

func (s *SessionService) Get(ctx context.Context, id string) (*model.Session, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *SessionService) Delete(ctx context.Context, id string) error {
	if err := s.engine.Disconnect(id); err != nil {
		log.Warn().Err(err).Str("session", id).Msg("Failed to disconnect session during delete")
	}
	return s.repo.Delete(ctx, id)
}

func (s *SessionService) SetStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}
