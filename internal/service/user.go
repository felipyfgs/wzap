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

type UserService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
	engine      *Engine
}

func NewUserService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository, engine *Engine) *UserService {
	return &UserService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		engine:      engine,
	}
}

func (s *UserService) Create(ctx context.Context, req model.UserCreateReq) (*model.User, error) {
	token := req.Token
	if token == "" {
		token = "sk_" + uuid.NewString()
	}

	now := time.Now()
	user := &model.User{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Token:     token,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	session := &model.Session{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Status:    "disconnected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session for user: %w", err)
	}

	return user, nil
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.userRepo.FindAll(ctx)
}

func (s *UserService) Get(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	session, err := s.sessionRepo.FindByUserID(ctx, id)
	if err == nil {
		if disconnectErr := s.engine.Disconnect(session.ID); disconnectErr != nil {
			log.Warn().Err(disconnectErr).Str("session", session.ID).Msg("Failed to disconnect session during user delete")
		}
	}
	return s.userRepo.Delete(ctx, id)
}
