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
	repo   *repo.SessionRepository
	engine *wa.Manager
}

func NewSessionService(r *repo.SessionRepository, engine *wa.Manager) *SessionService {
	return &SessionService{
		repo:   r,
		engine: engine,
	}
}

func (s *SessionService) Create(ctx context.Context, req dto.SessionCreateReq) (*model.Session, error) {
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
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
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
