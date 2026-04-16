package chatwoot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"wzap/internal/model"
)

func TestImportHistory_InvalidPeriod(t *testing.T) {
	svc := &Service{
		repo: &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
	}
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

	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1, ImportOnConnect: true, ImportPeriod: "7d"}
	svc.processConnected(context.Background(), cfg, nil)

	time.Sleep(50 * time.Millisecond)
}

type importTestMsgRepo struct {
	mockMsgRepo
	mu          sync.Mutex
	findCalls   int
	markedMsgs  []string
	historyMsgs []model.Message
}

func (m *importTestMsgRepo) FindUnimportedHistory(_ context.Context, _ string, _ time.Time, _, _ int) ([]model.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.findCalls++
	if m.findCalls == 1 && len(m.historyMsgs) > 0 {
		return m.historyMsgs, nil
	}
	return []model.Message{}, nil
}

func (m *importTestMsgRepo) MarkImportedToChatwoot(_ context.Context, _, msgID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markedMsgs = append(m.markedMsgs, msgID)
	return nil
}

func TestImportHistory_ProcessesTextMessages(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	mr := &importTestMsgRepo{
		historyMsgs: []model.Message{
			{ID: "hist-1", SessionID: "sess", ChatJID: "5511999900001@s.whatsapp.net", FromMe: false, MsgType: "text", Body: "hello", Source: "history_sync", Timestamp: time.Now().Add(-1 * time.Hour)},
			{ID: "hist-2", SessionID: "sess", ChatJID: "5511999900001@s.whatsapp.net", FromMe: true, MsgType: "text", Body: "world", Source: "history_sync", Timestamp: time.Now().Add(-30 * time.Minute)},
		},
	}

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    mr,
		clientFn:   func(_ *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc.importHistory(ctx, "sess", "7d", 0)

	if len(client.messages) != 2 {
		t.Fatalf("expected 2 messages created in Chatwoot, got %d", len(client.messages))
	}
	if len(mr.markedMsgs) != 2 {
		t.Fatalf("expected 2 messages marked as imported, got %d", len(mr.markedMsgs))
	}
}

func TestImportHistory_MediaMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake-image-data"))
	}))
	defer srv.Close()

	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	mr := &importTestMsgRepo{
		historyMsgs: []model.Message{
			{ID: "media-1", SessionID: "sess", ChatJID: "5511999900001@s.whatsapp.net", FromMe: false, MsgType: "image", MediaType: "image/jpeg", MediaURL: srv.URL, Body: "photo caption", Source: "history_sync", Timestamp: time.Now().Add(-1 * time.Hour)},
		},
	}

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    mr,
		clientFn:   func(_ *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc.importHistory(ctx, "sess", "7d", 0)

	if len(client.attachments) != 1 {
		t.Fatalf("expected 1 attachment uploaded, got %d", len(client.attachments))
	}
	if len(mr.markedMsgs) != 1 {
		t.Fatalf("expected 1 message marked as imported, got %d", len(mr.markedMsgs))
	}
}

type singleflightTestMsgRepo struct {
	mockMsgRepo
	mu        sync.Mutex
	findCalls int
}

func (m *singleflightTestMsgRepo) FindUnimportedHistory(_ context.Context, _ string, _ time.Time, _, _ int) ([]model.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.findCalls++
	time.Sleep(100 * time.Millisecond)
	return []model.Message{}, nil
}

func TestImportHistory_SingleflightPreventsConcurrent(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	mr := &singleflightTestMsgRepo{}

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    mr,
		clientFn:   func(_ *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	done := make(chan struct{})
	for i := 0; i < 3; i++ {
		go func() {
			svc.ImportHistoryAsync(context.Background(), "sess", "7d", 0)
			done <- struct{}{}
		}()
	}

	for i := 0; i < 3; i++ {
		<-done
	}

	if mr.findCalls > 1 {
		t.Fatalf("expected singleflight to prevent concurrent imports, but FindUnimportedHistory was called %d times", mr.findCalls)
	}
}
