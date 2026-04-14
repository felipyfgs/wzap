package chatwoot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"wzap/internal/model"
)

type mockContactNameGetter struct {
	names map[string]string
}

func (m *mockContactNameGetter) GetContactName(_ context.Context, _, jid string) string {
	return m.names[jid]
}

func TestContactNameHierarchy_AgendaNameOverPushName(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "5511999900001", PhoneNumber: "5511999900001"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	nameGetter := &mockContactNameGetter{
		names: map[string]string{
			"5511999900001@s.whatsapp.net": "João Silva",
		},
	}

	svc := &Service{
		repo:              &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:           &mockMsgRepo{},
		clientFn:          func(_ *Config) Client { return client },
		cache:             newMemoryCache(context.Background()),
		contactNameGetter: nameGetter,
		cb:                newCircuitBreakerManager(),
	}

	envelope := model.EventEnvelope{
		Event:     "Message",
		Session:   model.SessionInfo{ID: "sess"},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	p := waMessagePayload{
		Info:    waMessageInfo{Chat: "5511999900001@s.whatsapp.net", ID: "msg-1", PushName: "Beto"},
		Message: map[string]any{"conversation": "hello"},
	}
	envelope.Data, _ = json.Marshal(p)
	payload, _ := json.Marshal(envelope)

	svc.OnEvent(context.Background(), "sess", model.EventMessage, payload)

	if len(client.contacts) == 0 {
		t.Fatal("expected contact to be found")
	}
	updatedName := client.contacts[0].Name
	if updatedName != "João Silva" {
		t.Errorf("expected contact name to be updated to 'João Silva' (from agenda), got '%s'", updatedName)
	}
}

func TestContactNameHierarchy_PushNameFallback(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "5511999900002", PhoneNumber: "5511999900002"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	nameGetter := &mockContactNameGetter{
		names: map[string]string{},
	}

	svc := &Service{
		repo:              &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:           &mockMsgRepo{},
		clientFn:          func(_ *Config) Client { return client },
		cache:             newMemoryCache(context.Background()),
		contactNameGetter: nameGetter,
		cb:                newCircuitBreakerManager(),
	}

	envelope := model.EventEnvelope{
		Event:     "Message",
		Session:   model.SessionInfo{ID: "sess"},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	p := waMessagePayload{
		Info:    waMessageInfo{Chat: "5511999900002@s.whatsapp.net", ID: "msg-2", PushName: "Maria"},
		Message: map[string]any{"conversation": "hi"},
	}
	envelope.Data, _ = json.Marshal(p)
	payload, _ := json.Marshal(envelope)

	svc.OnEvent(context.Background(), "sess", model.EventMessage, payload)

	updatedName := client.contacts[0].Name
	if updatedName != "Maria" {
		t.Errorf("expected contact name to be updated to 'Maria' (pushName fallback), got '%s'", updatedName)
	}
}

func TestHandlePushName_DoesNotOverwriteAgendaName(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "Carlos Agenda", PhoneNumber: "5511999900003"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	nameGetter := &mockContactNameGetter{
		names: map[string]string{
			"5511999900003@s.whatsapp.net": "Carlos Agenda",
		},
	}

	svc := &Service{
		repo:              &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:           &mockMsgRepo{},
		clientFn:          func(_ *Config) Client { return client },
		cache:             newMemoryCache(context.Background()),
		contactNameGetter: nameGetter,
		cb:                newCircuitBreakerManager(),
	}

	envelope := model.EventEnvelope{
		Event:     "PushName",
		Session:   model.SessionInfo{ID: "sess"},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	data := struct {
		JID         string `json:"JID"`
		NewPushName string `json:"NewPushName"`
	}{
		JID:         "5511999900003@s.whatsapp.net",
		NewPushName: "Carlao",
	}
	envelope.Data, _ = json.Marshal(data)
	payload, _ := json.Marshal(envelope)

	_ = svc.processInboundEvent(context.Background(), "sess", model.EventPushName, payload)

	if client.contacts[0].Name != "Carlos Agenda" {
		t.Errorf("expected contact name to remain 'Carlos Agenda', got '%s'", client.contacts[0].Name)
	}
}

func TestHandlePushName_UpdatesWhenNameIsPhone(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "5511999900004", PhoneNumber: "5511999900004"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	nameGetter := &mockContactNameGetter{
		names: map[string]string{
			"5511999900004@s.whatsapp.net": "5511999900004",
		},
	}

	svc := &Service{
		repo:              &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:           &mockMsgRepo{},
		clientFn:          func(_ *Config) Client { return client },
		cache:             newMemoryCache(context.Background()),
		contactNameGetter: nameGetter,
		cb:                newCircuitBreakerManager(),
	}

	envelope := model.EventEnvelope{
		Event:     "PushName",
		Session:   model.SessionInfo{ID: "sess"},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	data := struct {
		JID         string `json:"JID"`
		NewPushName string `json:"NewPushName"`
	}{
		JID:         "5511999900004@s.whatsapp.net",
		NewPushName: "Ana",
	}
	envelope.Data, _ = json.Marshal(data)
	payload, _ := json.Marshal(envelope)

	_ = svc.processInboundEvent(context.Background(), "sess", model.EventPushName, payload)

	if client.contacts[0].Name != "Ana" {
		t.Errorf("expected contact name to be updated to 'Ana' (was phone number), got '%s'", client.contacts[0].Name)
	}
}

func TestImportHistory_UsesContactName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake-image"))
	}))
	defer srv.Close()

	client := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "5511999900005", PhoneNumber: "5511999900005"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	nameGetter := &mockContactNameGetter{
		names: map[string]string{
			"5511999900005@s.whatsapp.net": "Pedro Agenda",
		},
	}

	mr := &importTestMsgRepo{
		historyMsgs: []model.Message{
			{ID: "hist-contact-1", SessionID: "sess", ChatJID: "5511999900005@s.whatsapp.net", FromMe: false, MsgType: "text", Body: "test", Source: "history_sync", Timestamp: time.Now().Add(-1 * time.Hour)},
		},
	}

	svc := &Service{
		repo:              &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:           mr,
		clientFn:          func(_ *Config) Client { return client },
		cache:             newMemoryCache(context.Background()),
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		contactNameGetter: nameGetter,
		cb:                newCircuitBreakerManager(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc.importHistory(ctx, "sess", "7d", 0)

	if len(client.messages) != 1 {
		t.Fatalf("expected 1 message created, got %d", len(client.messages))
	}
	if client.contacts[0].Name != "Pedro Agenda" {
		t.Errorf("expected contact name to be 'Pedro Agenda', got '%s'", client.contacts[0].Name)
	}
}
