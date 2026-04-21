package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/google/uuid"
)

var sessionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type SessionService struct {
	repo            *repo.SessionRepository
	webhookRepo     *repo.WebhookRepository
	engine          *wa.Manager
	runtimeResolver *RuntimeResolver
}

func NewSessionService(r *repo.SessionRepository, webhookRepo *repo.WebhookRepository, engine *wa.Manager, runtimeResolver *RuntimeResolver) *SessionService {
	if runtimeResolver == nil {
		runtimeResolver = NewRuntimeResolver(r, engine)
	}
	return &SessionService{
		repo:            r,
		webhookRepo:     webhookRepo,
		engine:          engine,
		runtimeResolver: runtimeResolver,
	}
}

func (s *SessionService) Create(ctx context.Context, req dto.SessionCreateReq) (*dto.SessionWithTokenResp, error) {
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
	engine := req.Engine
	if engine == "" {
		engine = "whatsmeow"
	}
	session := &model.Session{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Token:     token,
		Status:    "disconnected",
		Engine:    engine,
		Proxy:     model.SessionProxy(req.Proxy),
		Settings:  model.SessionSettings(req.Settings),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	metrics.SessionsTotal.Inc()

	proxy := req.Proxy
	proxy.Password = ""
	resp := &dto.SessionWithTokenResp{
		ID:        session.ID,
		Name:      session.Name,
		Token:     session.Token,
		Status:    session.Status,
		Connected: session.Connected,
		Engine:    session.Engine,
		Proxy:     proxy,
		Settings:  req.Settings,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}

	if req.Webhook != nil && req.Webhook.URL != "" {
		events := make([]string, 0, len(req.Webhook.Events))
		for _, e := range req.Webhook.Events {
			if !model.ValidEventTypes[model.EventType(e)] {
				logger.Warn().Str("component", "service").Str("event", e).Str("session", session.ID).Msg("Skipping invalid event type in inline webhook")
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
			logger.Warn().Str("component", "service").Err(err).Str("session", session.ID).Msg("Failed to create inline webhook")
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

func (s *SessionService) enrichWithProfile(session model.Session) dto.SessionResp {
	var pushName, businessName, platform string
	if info := s.engine.GetClientInfo(session.ID); info != nil {
		pushName = info.PushName
		businessName = info.BusinessName
		platform = info.Platform
	}
	return dto.SessionToResp(session, pushName, businessName, platform)
}

func (s *SessionService) List(ctx context.Context) ([]dto.SessionResp, error) {
	sessions, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.SessionResp, len(sessions))
	for i, sess := range sessions {
		resp[i] = s.enrichWithProfile(sess)
	}
	return resp, nil
}

func (s *SessionService) Get(ctx context.Context, id string) (*dto.SessionResp, error) {
	session, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := s.enrichWithProfile(*session)
	return &resp, nil
}

func (s *SessionService) Delete(ctx context.Context, id string) error {
	if err := s.engine.Logout(ctx, id); err != nil {
		logger.Warn().Str("component", "service").Err(err).Str("session", id).Msg("Failed to logout session during delete")
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
	if req.Token != nil {
		token := *req.Token
		if token == "" {
			token = "sk_" + uuid.NewString()
		}
		session.Token = token
	}
	if req.Engine != nil {
		session.Engine = *req.Engine
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

	resp := s.enrichWithProfile(*session)
	return &resp, nil
}

func (s *SessionService) Status(ctx context.Context, id string) (*dto.SessionStatusResp, error) {
	runtime, err := s.runtimeResolver.ResolveStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	var loggedIn, connected bool
	session := runtime.Session()

	loggedIn = session.JID != ""
	connected = session.Connected == 1
	if client, cErr := runtime.Client(); cErr == nil {
		connected = client.IsConnected()
		loggedIn = client.Store.ID != nil
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

func (s *SessionService) Profile(ctx context.Context, id string) (*dto.SessionProfileResp, error) {
	runtime, err := s.runtimeResolver.ResolveProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	session := runtime.Session()

	client, err := runtime.Client()
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
	runtimeCtx := runtime.WithContext(ctx)
	if pic, picErr := client.GetProfilePictureInfo(runtimeCtx, selfJID, &whatsmeow.GetProfilePictureParams{}); picErr != nil {
		logger.Warn().Str("component", "service").Err(picErr).Str("session", session.ID).Msg("failed to get profile picture")
	} else if pic != nil {
		resp.PictureURL = pic.URL
	}

	if info, infoErr := client.GetUserInfo(runtimeCtx, []types.JID{*client.Store.ID}); infoErr == nil {
		for _, v := range info {
			resp.Status = v.Status
			break
		}
	}

	return resp, nil
}
