package chatwoot

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"wzap/internal/model"
)

func TestOnEvent_IncomingMessage(t *testing.T) {
	mockClient := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "Test Contact"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	envelope := model.EventEnvelope{
		Event:     "Message",
		EventID:   "test-event-id",
		Timestamp: "2024-01-01T00:00:00Z",
		Session:   model.SessionInfo{ID: "test-session", Name: "Test Session"},
	}

	info := waMessageInfo{
		Chat:     "5511999999999@s.whatsapp.net",
		Sender:   "5511999999999@s.whatsapp.net",
		IsFromMe: false,
		IsGroup:  false,
		ID:       "msg-id-123",
		PushName: "Test User",
	}

	msgPayload := waMessagePayload{
		Info:    info,
		Message: map[string]any{"conversation": "Hello World"},
	}

	envelope.Data, _ = json.Marshal(msgPayload)
	payload, _ := json.Marshal(envelope)

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "test-session", Enabled: true, InboxID: 1}},
		msgRepo:    &mockMsgRepo{},
		clientFn:   func(cfg *Config) Client { return mockClient },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	svc.OnEvent(context.Background(), "test-session", "Message", payload)

	if len(mockClient.messages) == 0 {
		t.Fatal("expected message to be created")
	}
	if mockClient.lastMessageType != "incoming" {
		t.Errorf("expected MessageType = incoming, got %s", mockClient.lastMessageType)
	}
}

func TestOnEvent_OutgoingMessage(t *testing.T) {
	mockClient := &mockClient{
		contacts:      []Contact{{ID: 1, Name: "Test Contact"}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}

	envelope := model.EventEnvelope{
		Event:     "Message",
		EventID:   "test-event-id",
		Timestamp: "2024-01-01T00:00:00Z",
		Session:   model.SessionInfo{ID: "test-session", Name: "Test Session"},
	}

	info := waMessageInfo{
		Chat:     "5511999999999@s.whatsapp.net",
		Sender:   "5511999999999@s.whatsapp.net",
		IsFromMe: true,
		IsGroup:  false,
		ID:       "msg-id-456",
		PushName: "Test User",
	}

	msgPayload := waMessagePayload{
		Info:    info,
		Message: map[string]any{"conversation": "Outgoing message"},
	}

	envelope.Data, _ = json.Marshal(msgPayload)
	payload, _ := json.Marshal(envelope)

	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "test-session", Enabled: true, InboxID: 1}},
		msgRepo:    &mockMsgRepo{},
		clientFn:   func(cfg *Config) Client { return mockClient },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	svc.OnEvent(context.Background(), "test-session", "Message", payload)

	if len(mockClient.messages) == 0 {
		t.Fatal("expected message to be created")
	}
	if mockClient.lastMessageType != "outgoing" {
		t.Errorf("expected MessageType = outgoing, got %s", mockClient.lastMessageType)
	}
}

type mockClientWithErr struct {
	mockClient
	createMsgErr error
}

func (m *mockClientWithErr) CreateMessage(_ context.Context, _ int, req MessageReq) (*Message, error) {
	if m.createMsgErr != nil {
		return nil, m.createMsgErr
	}
	return m.mockClient.CreateMessage(context.Background(), 0, req)
}

func newTestServiceWithErr(createMsgErr error) *Service {
	client := &mockClientWithErr{
		mockClient: mockClient{
			contacts:      []Contact{{ID: 1}},
			conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
		},
		createMsgErr: createMsgErr,
	}
	return &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    &mockDupMsgRepo{existingSourceIDs: map[string]bool{}},
		clientFn:   func(_ *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}
}

func TestProcessInboundEvent_RetryableError_ReturnsError(t *testing.T) {
	retryableErr := &APIError{StatusCode: 500, Message: "internal server error"}
	svc := newTestServiceWithErr(retryableErr)

	payload := buildMsgPayload(t, "msg-retryable")
	err := svc.processInboundEvent(context.Background(), "sess", model.EventMessage, payload)
	if err == nil {
		t.Error("expected non-nil error for retryable CreateMessage failure")
	}
}

func TestProcessInboundEvent_PermanentError_ReturnsNil(t *testing.T) {
	permanentErr := &APIError{StatusCode: 422, Message: "unprocessable entity"}
	svc := newTestServiceWithErr(permanentErr)

	payload := buildMsgPayload(t, "msg-permanent")
	err := svc.processInboundEvent(context.Background(), "sess", model.EventMessage, payload)
	if err != nil {
		t.Errorf("expected nil error for permanent CreateMessage failure, got: %v", err)
	}
}
