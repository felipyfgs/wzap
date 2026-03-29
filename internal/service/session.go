package service

import (
	"context"

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

func (s *SessionService) List(ctx context.Context) ([]model.Session, error) {
	return s.repo.FindAll(ctx)
}

func (s *SessionService) Get(ctx context.Context, id string) (*model.Session, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *SessionService) GetByUserID(ctx context.Context, userID string) (*model.Session, error) {
	return s.repo.FindByUserID(ctx, userID)
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
