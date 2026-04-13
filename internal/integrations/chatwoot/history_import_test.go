package chatwoot

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestImportHistory_InvalidPeriod(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
	}
	// Should return without panic for invalid period
	svc.importHistory(context.Background(), "sess", "invalid", 0)
}

func TestImportHistory_ValidPeriod24h(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc.importHistory(ctx, "sess", "24h", 0)
}

func TestImportHistory_ValidPeriod7d(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
	}
	svc.importHistory(context.Background(), "sess", "7d", 0)
}

func TestImportHistory_ValidPeriod30d(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
	}
	svc.importHistory(context.Background(), "sess", "30d", 0)
}

func TestImportHistory_CustomPeriod(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
	}
	svc.importHistory(context.Background(), "sess", "custom", 15)
}

func TestImportHistory_DisabledSession(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: false, InboxID: 1}},
	}
	svc.importHistory(context.Background(), "sess", "7d", 0)
}

func TestImportHistory_TriggerOnConnect(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := &Service{
		repo: &mockRepo{cfg: &Config{
			SessionID:       "sess",
			Enabled:         true,
			InboxID:         1,
			ImportOnConnect: true,
			ImportPeriod:    "7d",
		}},
		msgRepo:    &mockMsgRepoWithDuplicates{existingSourceIDs: map[string]bool{}},
		clientFn:   func(cfg *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	// handleConnected should trigger importHistory via goroutine
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1, ImportOnConnect: true, ImportPeriod: "7d"}
	svc.handleConnected(context.Background(), cfg, nil)

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)
}
