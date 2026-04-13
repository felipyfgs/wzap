package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"wzap/internal/model"
)

type stubSessionLookup struct {
	calls   int
	session *model.Session
	err     error
}

func (s *stubSessionLookup) FindByID(ctx context.Context, id string) (*model.Session, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.session == nil {
		return nil, errors.New("session not found")
	}
	return s.session, nil
}

func TestRuntimeResolverResolveReusesContext(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{
			ID:                 "sess-cloud",
			Engine:             "cloud_api",
			PhoneNumberID:      "123456",
			AccessToken:        "token",
			BusinessAccountID:  "waba",
			AppSecret:          "secret",
			WebhookVerifyToken: "verify",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}
	resolver := newRuntimeResolver(repo, nil, nil)

	runtime, err := resolver.Resolve(context.Background(), "sess-cloud")
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("expected 1 repository call, got %d", repo.calls)
	}
	if !runtime.IsCloudAPI() {
		t.Fatal("expected cloud_api runtime")
	}

	cfg, err := runtime.CloudConfig()
	if err != nil {
		t.Fatalf("unexpected cloud config error: %v", err)
	}
	if cfg.PhoneNumberID != "123456" {
		t.Fatalf("expected phone number id 123456, got %s", cfg.PhoneNumberID)
	}

	reused, err := resolver.Resolve(runtime.WithContext(context.Background()), "sess-cloud")
	if err != nil {
		t.Fatalf("unexpected resolve error using runtime context: %v", err)
	}
	if reused != runtime {
		t.Fatal("expected resolver to reuse runtime from context")
	}
	if repo.calls != 1 {
		t.Fatalf("expected repository calls to remain 1, got %d", repo.calls)
	}
}

func TestSessionConfigReaderReadConfigReusesRuntimeContext(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{
			ID:                 "sess-cloud",
			Engine:             "cloud_api",
			PhoneNumberID:      "phone-id",
			AccessToken:        "token",
			BusinessAccountID:  "waba",
			AppSecret:          "secret",
			WebhookVerifyToken: "verify",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}
	resolver := newRuntimeResolver(repo, nil, nil)
	reader := &SessionConfigReader{resolver: resolver}

	runtime, err := resolver.Resolve(context.Background(), "sess-cloud")
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("expected 1 repository call after resolve, got %d", repo.calls)
	}

	cfg, err := reader.ReadConfig(runtime.WithContext(context.Background()), "sess-cloud")
	if err != nil {
		t.Fatalf("unexpected read config error: %v", err)
	}
	if cfg.PhoneNumberID != "phone-id" {
		t.Fatalf("expected phone-id, got %s", cfg.PhoneNumberID)
	}
	if repo.calls != 1 {
		t.Fatalf("expected read config to reuse runtime context, got %d repository calls", repo.calls)
	}
}

func TestRuntimeResolverResolveMessageCapability(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{
			ID:        "sess-cloud",
			Engine:    "cloud_api",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	resolver := newRuntimeResolver(repo, nil, nil)

	runtime, err := resolver.ResolveMessage(context.Background(), "sess-cloud", model.CapabilityMessageLink)
	if err != nil {
		t.Fatalf("unexpected resolve message error: %v", err)
	}
	if runtime.Support() != model.SupportPartial {
		t.Fatalf("expected partial support, got %s", runtime.Support())
	}

	_, err = resolver.ResolveMessage(context.Background(), "sess-cloud", model.CapabilityMessagePoll)
	if err == nil {
		t.Fatal("expected capability error for unsupported cloud_api poll")
	}
	var capabilityErr *CapabilityError
	if !errors.As(err, &capabilityErr) {
		t.Fatalf("expected CapabilityError, got %T", err)
	}
	if capabilityErr.Support != model.SupportUnavailable {
		t.Fatalf("expected unavailable support, got %s", capabilityErr.Support)
	}
}
