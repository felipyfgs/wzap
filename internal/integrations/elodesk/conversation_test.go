package elodesk

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
)

// stubConvClient conta quantas vezes UpsertContact + GetOrCreateConversation
// são invocados — para validar dedup via singleflight.
type stubConvClient struct {
	upserts        int32
	getOrCreates   int32
	createMessages int32
}

func (s *stubConvClient) UpsertContact(_ context.Context, _ string, req UpsertContactReq) (*Contact, error) {
	atomic.AddInt32(&s.upserts, 1)
	return &Contact{ID: 1, Identifier: req.Identifier, Name: req.Name, SourceID: req.Identifier}, nil
}

func (s *stubConvClient) GetOrCreateConversation(_ context.Context, _, _ string, _ GetOrCreateConvReq) (*Conversation, error) {
	atomic.AddInt32(&s.getOrCreates, 1)
	return &Conversation{ID: 42, ContactID: 1, InboxID: 1, Status: ConversationStatusOpen}, nil
}

func (s *stubConvClient) CreateMessage(_ context.Context, _, _ string, convID int64, req MessageReq) (*Message, error) {
	atomic.AddInt32(&s.createMessages, 1)
	return &Message{ID: 99, ConversationID: convID, SourceID: req.SourceID}, nil
}

func (s *stubConvClient) CreateAttachment(_ context.Context, _, _ string, convID int64, _, _ string, _ []byte, _, _, sourceID string, _ map[string]any) (*Message, error) {
	return &Message{ID: 99, ConversationID: convID, SourceID: sourceID}, nil
}

func (s *stubConvClient) UpdateConversationStatus(_ context.Context, _, _ string, _ int64, _ string) error {
	return nil
}

func (s *stubConvClient) CreateInbox(_ context.Context, _ int, _, _, _ string) (*CreateInboxResp, error) {
	return &CreateInboxResp{Identifier: "stub", ApiToken: "stub-token", ChannelID: 1}, nil
}

func (s *stubConvClient) UpdateInboxWebhook(_ context.Context, _, _ int, _, _ string) error {
	return nil
}

// verifica http.Client no lugar do default
var _ = http.Client{}

func TestFindOrCreateConversation_CacheHit(t *testing.T) {
	svc := NewService(context.Background(), newInMemRepo(), newMockMsgRepo(), nil)
	stub := &stubConvClient{}
	svc.clientFn = func(_ *Config) Client { return stub }

	cfg := &Config{SessionID: "sess", InboxIdentifier: "id"}
	svc.cache.SetConv(context.Background(), "sess", "11988887777@s.whatsapp.net", 42, 1)

	convID, _, err := svc.findOrCreateConversation(context.Background(), cfg, "11988887777@s.whatsapp.net", "Jane")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if convID != 42 {
		t.Errorf("convID: got %d, want 42", convID)
	}
	if got := atomic.LoadInt32(&stub.upserts); got != 0 {
		t.Errorf("expected 0 upserts on cache hit, got %d", got)
	}
}

func TestFindOrCreateConversation_CacheMiss(t *testing.T) {
	svc := NewService(context.Background(), newInMemRepo(), newMockMsgRepo(), nil)
	stub := &stubConvClient{}
	svc.clientFn = func(_ *Config) Client { return stub }

	cfg := &Config{SessionID: "sess", InboxIdentifier: "id"}
	convID, _, err := svc.findOrCreateConversation(context.Background(), cfg, "11988887777@s.whatsapp.net", "Jane")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if convID != 42 {
		t.Errorf("convID: got %d, want 42", convID)
	}
	if got := atomic.LoadInt32(&stub.upserts); got != 1 {
		t.Errorf("expected 1 upsert, got %d", got)
	}
	if got := atomic.LoadInt32(&stub.getOrCreates); got != 1 {
		t.Errorf("expected 1 getOrCreate, got %d", got)
	}
}
