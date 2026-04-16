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
			ID:        "sess-wa",
			Engine:    "whatsmeow",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	resolver := &RuntimeResolver{repo: repo}

	runtime, err := resolver.Resolve(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("expected 1 repository call, got %d", repo.calls)
	}

	reused, err := resolver.Resolve(runtime.WithContext(context.Background()), "sess-wa")
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

func TestRuntimeResolverResolveMessageCapability(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{
			ID:        "sess-wa",
			Engine:    "whatsmeow",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	resolver := &RuntimeResolver{repo: repo}

	runtime, err := resolver.ResolveMessage(context.Background(), "sess-wa", model.CapabilityMessageText)
	if err != nil {
		t.Fatalf("unexpected resolve message error: %v", err)
	}
	if runtime.Support() != model.SupportComplete {
		t.Fatalf("expected complete support, got %s", runtime.Support())
	}

	unknownRepo := &stubSessionLookup{err: errors.New("session not found")}
	unknownResolver := &RuntimeResolver{repo: unknownRepo}
	_, err = unknownResolver.ResolveMessage(context.Background(), "unknown", model.CapabilityMessageText)
	if err == nil {
		t.Fatal("expected error for unknown session")
	}
}
