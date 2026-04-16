package chatwoot

import (
	"context"
	"net/http"
	"testing"
	"time"

	"wzap/internal/model"
)

func TestIdempotency_DuplicateSkipped(t *testing.T) {
	mockClient := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo: &mockDupMsgRepo{
			existingSourceIDs: map[string]bool{"sess:WAID:dup-msg": true},
		},
		clientFn:   func(cfg *Config) Client { return mockClient },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	svc.OnEvent(context.Background(), "sess", model.EventMessage, buildMsgPayload(t, "dup-msg"))

	if len(mockClient.messages) != 0 {
		t.Errorf("expected no messages created for duplicate, got %d", len(mockClient.messages))
	}
}

func TestIdempotency_NewMessageProcessed(t *testing.T) {
	mockClient := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo: &mockDupMsgRepo{
			existingSourceIDs: map[string]bool{},
		},
		clientFn:   func(cfg *Config) Client { return mockClient },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	svc.OnEvent(context.Background(), "sess", model.EventMessage, buildMsgPayload(t, "new-msg"))

	if len(mockClient.messages) == 0 {
		t.Error("expected message to be created for new message")
	}
}

func TestIdempotency_CacheHitSkipDB(t *testing.T) {
	mockClient := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	cache := newMemoryCache(context.Background())
	cache.SetIdempotent(context.Background(), "sess", "WAID:cached-msg")

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    &mockDupMsgRepo{existingSourceIDs: map[string]bool{}},
		clientFn:   func(cfg *Config) Client { return mockClient },
		cache:      cache,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	svc.OnEvent(context.Background(), "sess", model.EventMessage, buildMsgPayload(t, "cached-msg"))

	if len(mockClient.messages) != 0 {
		t.Errorf("expected no messages (cache hit), got %d", len(mockClient.messages))
	}
}
