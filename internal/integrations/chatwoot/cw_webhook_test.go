package chatwoot

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/repo"
)

func TestSendAttachmentToWhatsApp_ContentLengthTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.FormatInt(maxMediaBytes+1, 10))
		_, _ = w.Write([]byte("x"))
	}))
	defer server.Close()

	svc := &Service{
		httpClient: server.Client(),
	}
	cfg := &Config{
		SessionID:    "sess",
		MediaTimeout: 2,
		LargeTimeout: 2,
	}

	_, err := svc.sendAttachment(context.Background(), cfg, "5511999999999@s.whatsapp.net", server.URL+"/file.bin", "", "file", nil)
	if err == nil {
		t.Fatal("expected attachment too large error")
	}
	if !strings.Contains(err.Error(), "attachment too large") {
		t.Fatalf("expected attachment too large error, got: %v", err)
	}
}

func TestSendErrorToAgent(t *testing.T) {
	client := &mockClient{
		contacts:      []Contact{{ID: 1}},
		conversations: []Conversation{{ID: 1, InboxID: 1, Status: "open"}},
	}
	svc := newTestService(client)
	cfg := &Config{SessionID: "sess", Enabled: true, InboxID: 1}

	sendErr := fmt.Errorf("connection refused")
	svc.sendErrorToAgent(context.Background(), cfg, 1, sendErr)

	if len(client.messages) == 0 {
		t.Fatal("expected error message to be created")
	}
	msg := client.messages[0]
	if msg.MessageType != "outgoing" {
		t.Errorf("expected message_type=outgoing, got %s", msg.MessageType)
	}
	if !msg.Private {
		t.Error("expected message to be private")
	}
	if !containsStr(msg.Content, "falha de conexão") {
		t.Errorf("expected sanitized error text in content, got: %s", msg.Content)
	}
	if !containsStr(msg.Content, "Mensagem não enviada") {
		t.Errorf("expected prefix in content, got: %s", msg.Content)
	}
}

func TestSignContent_WithSenderName(t *testing.T) {
	result := signContent("Hello World", "João", "")
	expected := "*João:*\nHello World"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSignContent_EmptySenderName(t *testing.T) {
	result := signContent("Hello World", "", "")
	if result != "Hello World" {
		t.Errorf("expected unchanged content, got %q", result)
	}
}

func TestSignContent_CustomDelimiter(t *testing.T) {
	result := signContent("Hello", "Agent", " - ")
	expected := "*Agent:* - Hello"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSignContent_DelimiterWithLiteralNewline(t *testing.T) {
	result := signContent("Hello", "Agent", `\n\n`)
	expected := "*Agent:*\n\nHello"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

type cloudSyncMsgRepo struct {
	msg        *model.Message
	updatedMsg struct {
		msgID     string
		cwMsgID   int
		cwConvID  int
		sourceID  string
		wasCalled bool
	}
}

func (m *cloudSyncMsgRepo) Save(_ context.Context, _ *model.Message) error { return nil }
func (m *cloudSyncMsgRepo) FindByChat(_ context.Context, _, _ string, _, _ int) ([]model.Message, error) {
	return nil, nil
}
func (m *cloudSyncMsgRepo) FindByID(_ context.Context, _, msgID string) (*model.Message, error) {
	if m.msg == nil || m.msg.ID != msgID {
		return nil, errors.New("not found")
	}
	return m.msg, nil
}
func (m *cloudSyncMsgRepo) FindSessionByMessageID(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindByCWMessageID(_ context.Context, _ string, _ int) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindAllByCWMessageID(_ context.Context, _ string, _ int) ([]model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) UpdateChatwootRef(_ context.Context, _, msgID string, cwMsgID, cwConvID int, sourceID string) error {
	m.updatedMsg.msgID = msgID
	m.updatedMsg.cwMsgID = cwMsgID
	m.updatedMsg.cwConvID = cwConvID
	m.updatedMsg.sourceID = sourceID
	m.updatedMsg.wasCalled = true
	return nil
}
func (m *cloudSyncMsgRepo) ListMissingChatwootRefs(_ context.Context, _ string, _ int, _ string) ([]string, error) {
	return nil, nil
}
func (m *cloudSyncMsgRepo) ExistsBySourceID(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *cloudSyncMsgRepo) FindBySourceID(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindBySourceIDPrefix(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindByBody(_ context.Context, _, _ string, _ bool) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindByBodyAndChat(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindByBodyAndChatAny(_ context.Context, _, _, _ string, _ bool) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindByTimestamp(_ context.Context, _ string, _ string, _ int64, _ int64) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindLastReceived(_ context.Context, _, _ string) (*model.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *cloudSyncMsgRepo) FindUnimportedHistory(_ context.Context, _ string, _ time.Time, _, _ int) ([]model.Message, error) {
	return nil, nil
}
func (m *cloudSyncMsgRepo) MarkImported(_ context.Context, _, _ string) error      { return nil }
func (m *cloudSyncMsgRepo) UpdateMediaURL(_ context.Context, _, _, _ string) error { return nil }
func (m *cloudSyncMsgRepo) FindBySession(_ context.Context, _ string, _, _ int) ([]model.Message, error) {
	return nil, nil
}
func (m *cloudSyncMsgRepo) FindMedia(_ context.Context, _ string, _ repo.MediaFilter) ([]model.Message, int, error) {
	return nil, 0, nil
}

func TestHandleIncomingWebhook_SyncCloudMessageRef(t *testing.T) {
	msgRepo := &cloudSyncMsgRepo{msg: &model.Message{ID: "3EB02DD5F2915A8FD95E3D", SessionID: "sess"}}
	svc := &Service{
		repo:       &mockRepo{cfg: &Config{SessionID: "sess", Enabled: true, InboxID: 1, InboxType: "cloud"}},
		msgRepo:    msgRepo,
		clientFn:   func(_ *Config) Client { return &mockClient{} },
		cache:      newMemoryCache(context.Background()),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
	}

	body := dto.CWWebhookPayload{
		EventType: "message_created",
		Message: &dto.CWWebhookMsg{
			ID:          1175,
			SourceID:    "3EB02DD5F2915A8FD95E3D",
			MessageType: float64(0),
		},
		Conversation: &struct {
			ID           int `json:"id"`
			ContactInbox struct {
				SourceID string `json:"source_id"`
			} `json:"contact_inbox"`
			ContactID int    `json:"contact_id"`
			InboxID   int    `json:"inbox_id"`
			Status    string `json:"status"`
			Messages  []struct {
				ID      int    `json:"id"`
				Content string `json:"content,omitempty"`
				Sender  *struct {
					AvailableName string `json:"available_name,omitempty"`
				} `json:"sender,omitempty"`
			} `json:"messages,omitempty"`
			Meta struct {
				Sender struct {
					ID                   int            `json:"id"`
					Name                 string         `json:"name"`
					Identifier           string         `json:"identifier,omitempty"`
					PhoneNumber          string         `json:"phone_number,omitempty"`
					AdditionalAttributes map[string]any `json:"additional_attributes,omitempty"`
				} `json:"sender"`
			} `json:"meta"`
		}{ID: 57},
	}

	if err := svc.HandleIncomingWebhook(context.Background(), "sess", body); err != nil {
		t.Fatalf("HandleIncomingWebhook returned error: %v", err)
	}

	if !msgRepo.updatedMsg.wasCalled {
		t.Fatal("expected UpdateChatwootRef to be called")
	}
	if msgRepo.updatedMsg.msgID != "3EB02DD5F2915A8FD95E3D" {
		t.Fatalf("expected msgID to be synced, got %q", msgRepo.updatedMsg.msgID)
	}
	if msgRepo.updatedMsg.cwMsgID != 1175 {
		t.Fatalf("expected cwMsgID=1175, got %d", msgRepo.updatedMsg.cwMsgID)
	}
	if msgRepo.updatedMsg.cwConvID != 57 {
		t.Fatalf("expected cwConvID=57, got %d", msgRepo.updatedMsg.cwConvID)
	}
	if msgRepo.updatedMsg.sourceID != "3EB02DD5F2915A8FD95E3D" {
		t.Fatalf("expected sourceID to be preserved, got %q", msgRepo.updatedMsg.sourceID)
	}
}
