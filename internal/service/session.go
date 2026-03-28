package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"wzap/internal/model"
	"wzap/internal/repository"
)

type SessionService struct {
	repo   *repository.SessionRepository
	engine *Engine
}

func NewSessionService(repo *repository.SessionRepository, engine *Engine) *SessionService {
	return &SessionService{
		repo:   repo,
		engine: engine,
	}
}

func (s *SessionService) Create(ctx context.Context, req model.SessionCreateReq) (*model.Session, error) {
	session := &model.Session{
		ID:        req.ID,
		APIKey:    "sk_" + uuid.NewString(),
		Status:    model.StatusInit,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

func (s *SessionService) SetStatus(ctx context.Context, id string, status model.SessionStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}
