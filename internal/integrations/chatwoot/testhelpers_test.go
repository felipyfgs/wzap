package chatwoot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"wzap/internal/model"
	"wzap/internal/repo"
)

type mockClient struct {
	mu                      sync.Mutex
	messages                []MessageReq
	attachments             []string
	contacts                []Contact
	conversations           []Conversation
	lastMessageType         string
	filterDelay             time.Duration
	filterContactsCalls     int
	createConversationCalls int
}

func (m *mockClient) SearchContacts(_ context.Context, _ string) ([]Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.contacts, nil
}

func (m *mockClient) FilterContacts(_ context.Context, _ string) ([]Contact, error) {
	m.mu.Lock()
	m.filterContactsCalls++
	delay := m.filterDelay
	contacts := m.contacts
	m.mu.Unlock()
	if delay > 0 {
		time.Sleep(delay)
	}
	return contacts, nil
}

func (m *mockClient) CreateContact(_ context.Context, _ CreateContactReq) (*Contact, error) {
	return &Contact{ID: 1}, nil
}

func (m *mockClient) UpdateContact(_ context.Context, _ int, req UpdateContactReq) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.Name != "" && len(m.contacts) > 0 {
		m.contacts[0].Name = req.Name
	}
	return nil
}

func (m *mockClient) ListContactConversations(_ context.Context, _ int) ([]Conversation, error) {
	return m.conversations, nil
}

func (m *mockClient) CreateConversation(_ context.Context, _ CreateConversationReq) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createConversationCalls++
	return &Conversation{ID: 1}, nil
}

func (m *mockClient) UpdateConversationStatus(_ context.Context, _ int, _ string) error {
	return nil
}

func (m *mockClient) CreateMessage(_ context.Context, _ int, req MessageReq) (*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, req)
	m.lastMessageType = req.MessageType
	return &Message{ID: 1, SourceID: "src-1"}, nil
}

func (m *mockClient) CreateMessageWithAttachment(_ context.Context, _ int, _ string, filename string, _ []byte, _ string, _ string, _ string, _ int, _ map[string]any) (*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachments = append(m.attachments, filename)
	return &Message{ID: 1}, nil
}

func (m *mockClient) DeleteMessage(_ context.Context, _, _ int) error {
	return nil
}

func (m *mockClient) UpdateMessage(_ context.Context, _, _ int, _ string) error {
	return nil
}

func (m *mockClient) UpdateLastSeen(_ context.Context, _, _ string, _ int) error {
	return nil
}

func (m *mockClient) ListInboxes(_ context.Context) ([]Inbox, error) {
	return []Inbox{{ID: 1, Name: "test-inbox"}}, nil
}

func (m *mockClient) CreateInbox(_ context.Context, _ string, _ string) (*Inbox, error) {
	return &Inbox{ID: 1}, nil
}

func (m *mockClient) UpdateInboxWebhook(_ context.Context, _ int, _ string) error {
	return nil
}

func (m *mockClient) GetConversation(_ context.Context, convID int) (*Conversation, error) {
	return &Conversation{ID: convID}, nil
}

func (m *mockClient) MergeContacts(_ context.Context, _, _ int) error {
	return nil
}

func (m *mockClient) FilterContactsCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.filterContactsCalls
}

func (m *mockClient) CreateConversationCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createConversationCalls
}

type mockRepo struct {
	cfg      *Config
	notFound bool
}

func (m *mockRepo) Upsert(_ context.Context, cfg *Config) error {
	m.cfg = cfg
	return nil
}

func (m *mockRepo) FindBySessionID(_ context.Context, sessionID string) (*Config, error) {
	if m.notFound {
		return nil, fmt.Errorf("not found")
	}
	if m.cfg == nil {
		return &Config{SessionID: sessionID, Enabled: true, InboxID: 1}, nil
	}
	return m.cfg, nil
}

func (m *mockRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type mockMsgRepo struct{}

func (m *mockMsgRepo) Save(_ context.Context, _ *model.Message) error {
	return nil
}

func (m *mockMsgRepo) FindByChat(_ context.Context, _, _ string, _, _ int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (m *mockMsgRepo) FindByID(_ context.Context, sessionID, msgID string) (*model.Message, error) {
	return &model.Message{ID: msgID, SessionID: sessionID}, nil
}

func (m *mockMsgRepo) FindByCWMessageID(_ context.Context, sessionID string, _ int) (*model.Message, error) {
	return &model.Message{ID: "test-msg", SessionID: sessionID}, nil
}

func (m *mockMsgRepo) FindAllByCWMessageID(_ context.Context, sessionID string, _ int) ([]model.Message, error) {
	return []model.Message{{ID: "test-msg", SessionID: sessionID}}, nil
}

func (m *mockMsgRepo) UpdateChatwootRef(_ context.Context, _, _ string, _, _ int, _ string) error {
	return nil
}

func (m *mockMsgRepo) ExistsBySourceID(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockMsgRepo) FindBySourceID(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepo) FindBySourceIDPrefix(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepo) FindByBody(_ context.Context, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepo) FindByBodyAndChat(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepo) FindByBodyAndChatAny(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepo) FindByTimestampWindow(_ context.Context, _, _ string, _ int64, _ int64) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepo) FindLastReceivedByChat(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockMsgRepo) FindUnimportedHistory(_ context.Context, _ string, _ time.Time, _, _ int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (m *mockMsgRepo) MarkImportedToChatwoot(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockMsgRepo) UpdateMediaURL(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockMsgRepo) FindBySession(_ context.Context, _ string, _, _ int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (m *mockMsgRepo) FindMedia(_ context.Context, _ string, _ repo.MediaFilter) ([]model.Message, int, error) {
	return []model.Message{}, 0, nil
}

type mockMsgRepoWithDuplicates struct {
	existingSourceIDs map[string]bool
}

func (m *mockMsgRepoWithDuplicates) Save(_ context.Context, _ *model.Message) error {
	return nil
}

func (m *mockMsgRepoWithDuplicates) FindByChat(_ context.Context, _, _ string, _, _ int) ([]model.Message, error) {
	return nil, nil
}

func (m *mockMsgRepoWithDuplicates) FindByID(_ context.Context, sessionID, msgID string) (*model.Message, error) {
	return &model.Message{ID: msgID, SessionID: sessionID}, nil
}

func (m *mockMsgRepoWithDuplicates) FindByCWMessageID(_ context.Context, _ string, _ int) (*model.Message, error) {
	return nil, nil
}

func (m *mockMsgRepoWithDuplicates) FindAllByCWMessageID(_ context.Context, _ string, _ int) ([]model.Message, error) {
	return nil, nil
}

func (m *mockMsgRepoWithDuplicates) UpdateChatwootRef(_ context.Context, _, _ string, _, _ int, _ string) error {
	return nil
}

func (m *mockMsgRepoWithDuplicates) ExistsBySourceID(_ context.Context, sessionID, sourceID string) (bool, error) {
	return m.existingSourceIDs[sessionID+":"+sourceID], nil
}

func (m *mockMsgRepoWithDuplicates) FindBySourceID(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepoWithDuplicates) FindBySourceIDPrefix(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepoWithDuplicates) FindByBody(_ context.Context, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepoWithDuplicates) FindByBodyAndChat(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepoWithDuplicates) FindByBodyAndChatAny(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepoWithDuplicates) FindByTimestampWindow(_ context.Context, _, _ string, _ int64, _ int64) (*model.Message, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockMsgRepoWithDuplicates) FindLastReceivedByChat(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockMsgRepoWithDuplicates) FindUnimportedHistory(_ context.Context, _ string, _ time.Time, _, _ int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (m *mockMsgRepoWithDuplicates) MarkImportedToChatwoot(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockMsgRepoWithDuplicates) UpdateMediaURL(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockMsgRepoWithDuplicates) FindBySession(_ context.Context, _ string, _, _ int) ([]model.Message, error) {
	return []model.Message{}, nil
}

func (m *mockMsgRepoWithDuplicates) FindMedia(_ context.Context, _ string, _ repo.MediaFilter) ([]model.Message, int, error) {
	return []model.Message{}, 0, nil
}

type mockMsgRepoFixed struct {
	mockMsgRepoWithDuplicates
	cwMsgID  *int
	cwConvID *int
}

func (m *mockMsgRepoFixed) FindByID(_ context.Context, sessionID, msgID string) (*model.Message, error) {
	msg := &model.Message{ID: msgID, SessionID: sessionID}
	if m.cwMsgID != nil {
		msg.CWMessageID = m.cwMsgID
	}
	if m.cwConvID != nil {
		msg.CWConversationID = m.cwConvID
	}
	return msg, nil
}

func (m *mockMsgRepoFixed) FindByCWMessageID(ctx context.Context, sessionID string, _ int) (*model.Message, error) {
	return m.FindByID(ctx, sessionID, "fixed-msg")
}

func newTestService(client *mockClient) *Service {
	return &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1}},
		msgRepo:    &mockMsgRepoWithDuplicates{existingSourceIDs: map[string]bool{}},
		clientFn:   func(_ *Config) Client { return client },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}
}

func buildPayload(t *testing.T, sessionID string, event model.EventType, data any) []byte {
	t.Helper()
	env := model.EventEnvelope{
		Event:     string(event),
		Session:   model.SessionInfo{ID: sessionID},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	env.Data, _ = json.Marshal(data)
	b, _ := json.Marshal(env)
	return b
}

func buildMsgPayload(t *testing.T, msgID string) []byte {
	t.Helper()
	envelope := model.EventEnvelope{
		Event:     "Message",
		Session:   model.SessionInfo{ID: "sess", Name: "Sess"},
		Timestamp: "2024-01-01T00:00:00Z",
	}
	p := waMessagePayload{
		Info:    waMessageInfo{Chat: "5511@s.whatsapp.net", ID: msgID},
		Message: map[string]any{"conversation": "hello"},
	}
	envelope.Data, _ = json.Marshal(p)
	b, _ := json.Marshal(envelope)
	return b
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && findSubstring(s, sub)
}

func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
