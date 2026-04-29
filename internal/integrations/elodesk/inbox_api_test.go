package elodesk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestIsStaleConvError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("oops"), false},
		{"500", &APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}, false},
		{"403", &APIError{StatusCode: http.StatusForbidden, Message: "denied"}, false},
		{"404", &APIError{StatusCode: http.StatusNotFound, Message: "Not Found"}, true},
		{"wrapped 404", fmt.Errorf("ctx: %w", &APIError{StatusCode: http.StatusNotFound}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStaleConvError(tt.err); got != tt.want {
				t.Fatalf("isStaleConvError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// staleConvClient simula uma conversa apagada no elodesk: a primeira
// chamada a CreateMessage com o convID "velho" devolve 404; assim que o
// caller cria uma conv nova (via GetOrCreateConversation), as chamadas
// subsequentes funcionam.
type staleConvClient struct {
	staleConvID    int64
	convCounter    int32 // gera IDs novos sequencialmente após o "velho"
	createCalls    int32
	upsertCalls    int32
	getOrCreateOps int32
}

func (s *staleConvClient) UpsertContact(_ context.Context, _ string, req UpsertContactReq) (*Contact, error) {
	atomic.AddInt32(&s.upsertCalls, 1)
	return &Contact{ID: 1, Identifier: req.Identifier, Name: req.Name, SourceID: req.Identifier}, nil
}

func (s *staleConvClient) GetOrCreateConversation(_ context.Context, _, _ string, _ GetOrCreateConvReq) (*Conversation, error) {
	atomic.AddInt32(&s.getOrCreateOps, 1)
	// IDs novos começam em staleConvID+1, simulando o auto-increment do
	// elodesk após o DELETE da conv anterior.
	newID := int(s.staleConvID) + int(atomic.AddInt32(&s.convCounter, 1))
	return &Conversation{ID: newID, ContactID: 1, InboxID: 1, Status: ConversationStatusOpen}, nil
}

func (s *staleConvClient) CreateMessage(_ context.Context, _, _ string, convID int64, req MessageReq) (*Message, error) {
	atomic.AddInt32(&s.createCalls, 1)
	if convID == s.staleConvID {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: `{"error":"Not Found"}`}
	}
	return &Message{ID: 99, ConversationID: convID, SourceID: req.SourceID}, nil
}

func (s *staleConvClient) CreateAttachment(_ context.Context, _, _ string, convID int64, _, _ string, _ []byte, _, _, sourceID string, _ map[string]any) (*Message, error) {
	if convID == s.staleConvID {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: `{"error":"Not Found"}`}
	}
	return &Message{ID: 99, ConversationID: convID, SourceID: sourceID}, nil
}

func (s *staleConvClient) UpdateConversationStatus(_ context.Context, _, _ string, _ int64, _ string) error {
	return nil
}

func (s *staleConvClient) CreateInbox(_ context.Context, _ int, _, _, _ string) (*CreateInboxResp, error) {
	return &CreateInboxResp{Identifier: "stale", ApiToken: "t", ChannelID: 1}, nil
}

func (s *staleConvClient) UpdateInboxWebhook(_ context.Context, _, _ int, _, _ string) error {
	return nil
}

// TestDispatchWithStaleRetry_RecreatesConversation simula o cenário
// principal do bug: agente apagou a conversa pela UI do elodesk; o cache
// do wzap ainda aponta pro convID antigo; uma nova mensagem chega.
// Esperado: a primeira tentativa toma 404, o cache é invalidado, e o
// retry cria uma conversa NOVA (id diferente) e posta a mensagem nela.
func TestDispatchWithStaleRetry_RecreatesConversation(t *testing.T) {
	svc := NewService(context.Background(), newInMemRepo(), newMockMsgRepo(), nil)
	const staleID int64 = 7
	stub := &staleConvClient{staleConvID: staleID}
	svc.clientFn = func(_ *Config) Client { return stub }

	cfg := &Config{SessionID: "sess", InboxIdentifier: "inb"}
	chatJID := "5511999999999@s.whatsapp.net"

	// Cache "envenenado": aponta pro convID que foi apagado no elodesk.
	svc.cache.SetConv(context.Background(), cfg.SessionID, chatJID, staleID, 1)

	h := newAPIInboxHandler(svc)
	d := messageDispatch{
		client:       stub,
		cfg:          cfg,
		chatJID:      chatJID,
		contactPushN: "Tester",
		sourceID:     "WAID:abc",
		messageType:  "incoming",
		text:         "olá",
	}

	out, convID, err := h.dispatchWithStaleRetry(context.Background(), d)
	if err != nil {
		t.Fatalf("dispatchWithStaleRetry: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil Message after stale retry")
	}
	if convID == staleID {
		t.Fatalf("expected a new conv id, got the stale one (%d)", staleID)
	}
	if out.ConversationID != convID {
		t.Fatalf("returned message references convID=%d but caller got %d", out.ConversationID, convID)
	}

	// Duas chamadas a CreateMessage: a primeira que tomou 404, a segunda
	// que funcionou.
	if got := atomic.LoadInt32(&stub.createCalls); got != 2 {
		t.Errorf("expected 2 CreateMessage calls (404 + retry), got %d", got)
	}
	// Cache deve ter sido populado com o convID novo após a recriação.
	cachedID, _, ok := svc.cache.GetConv(context.Background(), cfg.SessionID, chatJID)
	if !ok {
		t.Fatal("cache should have the new conv id after retry")
	}
	if cachedID == staleID {
		t.Fatalf("cache still holds the stale conv id (%d)", staleID)
	}
}

// TestDispatchWithStaleRetry_PassesThroughOtherErrors garante que apenas
// 404 dispara o retry: erros de servidor (5xx), erros de rede ou outros
// 4xx propagam direto pro caller para o dispatcher externo decidir.
func TestDispatchWithStaleRetry_PassesThroughOtherErrors(t *testing.T) {
	svc := NewService(context.Background(), newInMemRepo(), newMockMsgRepo(), nil)
	stub := &alwaysFailClient{err: &APIError{StatusCode: http.StatusInternalServerError, Message: "boom"}}
	svc.clientFn = func(_ *Config) Client { return stub }

	cfg := &Config{SessionID: "sess", InboxIdentifier: "inb"}
	chatJID := "5511999999999@s.whatsapp.net"
	svc.cache.SetConv(context.Background(), cfg.SessionID, chatJID, 1, 1)

	h := newAPIInboxHandler(svc)
	_, _, err := h.dispatchWithStaleRetry(context.Background(), messageDispatch{
		client:      stub,
		cfg:         cfg,
		chatJID:     chatJID,
		sourceID:    "WAID:x",
		messageType: "incoming",
		text:        "y",
	})
	if err == nil {
		t.Fatal("expected error from 5xx, got nil")
	}
	if stub.createCalls != 1 {
		t.Errorf("expected single CreateMessage call (no retry on 5xx), got %d", stub.createCalls)
	}
}

type alwaysFailClient struct {
	err         error
	createCalls int
}

func (c *alwaysFailClient) UpsertContact(_ context.Context, _ string, req UpsertContactReq) (*Contact, error) {
	return &Contact{ID: 1, Identifier: req.Identifier, SourceID: req.Identifier}, nil
}

func (c *alwaysFailClient) GetOrCreateConversation(_ context.Context, _, _ string, _ GetOrCreateConvReq) (*Conversation, error) {
	return &Conversation{ID: 1, ContactID: 1, InboxID: 1}, nil
}

func (c *alwaysFailClient) CreateMessage(_ context.Context, _, _ string, _ int64, _ MessageReq) (*Message, error) {
	c.createCalls++
	return nil, c.err
}

func (c *alwaysFailClient) CreateAttachment(_ context.Context, _, _ string, _ int64, _, _ string, _ []byte, _, _, _ string, _ map[string]any) (*Message, error) {
	return nil, c.err
}

func (c *alwaysFailClient) UpdateConversationStatus(_ context.Context, _, _ string, _ int64, _ string) error {
	return nil
}

func (c *alwaysFailClient) CreateInbox(_ context.Context, _ int, _, _, _ string) (*CreateInboxResp, error) {
	return &CreateInboxResp{}, nil
}

func (c *alwaysFailClient) UpdateInboxWebhook(_ context.Context, _, _ int, _, _ string) error {
	return nil
}
