package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
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
		Proxy:     model.SessionProxy(req.Proxy),
		Settings:  model.SessionSettings(req.Settings),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	resp := &dto.SessionCreatedResp{
		ID:        session.ID,
		Name:      session.Name,
		Token:     session.Token,
		Status:    session.Status,
		Connected: session.Connected,
		Proxy:     req.Proxy,
		Settings:  req.Settings,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}

	if req.Webhook != nil && req.Webhook.URL != "" {
		events := make([]string, 0, len(req.Webhook.Events))
		for _, e := range req.Webhook.Events {
			if !model.ValidEventTypes[model.EventType(e)] {
				logger.Warn().Str("event", e).Str("session", session.ID).Msg("Skipping invalid event type in inline webhook")
				continue
			}
			events = append(events, e)
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
			resp.Webhook = &dto.WebhookResp{
				ID:          wh.ID,
				SessionID:   wh.SessionID,
				URL:         wh.URL,
				Events:      wh.Events,
				Enabled:     wh.Enabled,
				NATSEnabled: wh.NATSEnabled,
				CreatedAt:   wh.CreatedAt,
				UpdatedAt:   wh.UpdatedAt,
			}
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

func (s *SessionService) Update(ctx context.Context, id string, req dto.SessionUpdateReq) (*dto.SessionResp, error) {
	session, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		if !sessionNameRegex.MatchString(*req.Name) {
			return nil, fmt.Errorf("name must contain only letters, numbers, hyphens and underscores")
		}
		session.Name = *req.Name
	}
	if req.Proxy != nil {
		session.Proxy = model.SessionProxy(*req.Proxy)
	}
	if req.Settings != nil {
		session.Settings = model.SessionSettings(*req.Settings)
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	if s.engine != nil {
		s.engine.UpdateSessionName(id, session.Name)
	}

	resp := dto.SessionToResp(*session)
	return &resp, nil
}

func (s *SessionService) Status(ctx context.Context, id string) (*dto.SessionStatusResp, error) {
	session, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	loggedIn := session.JID != ""
	connected := session.Connected == 1

	if s.engine != nil {
		if client, cErr := s.engine.GetClient(id); cErr == nil {
			connected = client.IsConnected()
			loggedIn = client.Store.ID != nil
		}
	}

	return &dto.SessionStatusResp{
		ID:        session.ID,
		Name:      session.Name,
		JID:       session.JID,
		Connected: connected,
		LoggedIn:  loggedIn,
		Status:    session.Status,
	}, nil
}

func (s *SessionService) SetStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *SessionService) Profile(ctx context.Context, id string) (*dto.SessionProfileResp, error) {
	client, err := s.engine.GetClient(id)
	if err != nil {
		return nil, fmt.Errorf("session not connected: %w", err)
	}

	resp := &dto.SessionProfileResp{
		PushName:     client.Store.PushName,
		BusinessName: client.Store.BusinessName,
		Platform:     client.Store.Platform,
	}

	if !client.IsConnected() || client.Store.ID == nil {
		return resp, nil
	}

	selfJID := client.Store.ID.ToNonAD()
	if pic, picErr := client.GetProfilePictureInfo(ctx, selfJID, &whatsmeow.GetProfilePictureParams{}); picErr != nil {
		logger.Warn().Err(picErr).Str("session", id).Msg("failed to get profile picture")
	} else if pic != nil {
		resp.PictureURL = pic.URL
	}

	if info, infoErr := client.GetUserInfo(ctx, []types.JID{*client.Store.ID}); infoErr == nil {
		for _, v := range info {
			resp.Status = v.Status
			break
		}
	}

	return resp, nil
}
