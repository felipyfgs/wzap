package chatwoot

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
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
