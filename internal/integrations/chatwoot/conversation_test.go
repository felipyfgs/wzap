package chatwoot

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestFindOrCreateConversation_ConcurrentCalls(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "Contato"}},
		conversations: []Conversation{},
		filterDelay:   100 * time.Millisecond,
	}

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    &mockMsgRepo{},
		clientFn:   func(_ *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	cfg := &Config{
		SessionID: "sess",
		Enabled:   true,
		InboxID:   1,
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan int, 2)
	errs := make(chan error, 2)

	for range 2 {
		wg.Go(func() {
			<-start
			convID, err := svc.findOrCreateConversation(context.Background(), cfg, "5511999999999@s.whatsapp.net", "User")
			if err != nil {
				errs <- err
				return
			}
			results <- convID
		})
	}

	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("findOrCreateConversation returned error: %v", err)
		}
	}

	for convID := range results {
		if convID != 1 {
			t.Fatalf("expected conversation id 1, got %d", convID)
		}
	}

	if calls := client.FilterContactsCallCount(); calls != 2 {
		t.Fatalf("expected 2 FilterContacts calls (phone + BR variant), got %d", calls)
	}
	if calls := client.CreateConversationCallCount(); calls != 1 {
		t.Fatalf("expected 1 CreateConversation call, got %d", calls)
	}
}
