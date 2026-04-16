package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"wzap/internal/dto"
	"wzap/internal/integrations/chatwoot"
)

type mockCloudWARepo struct {
	cfg *chatwoot.Config
}

func (m *mockCloudWARepo) FindBySessionID(_ context.Context, _ string) (*chatwoot.Config, error) {
	return m.cfg, nil
}

func (m *mockCloudWARepo) FindByPhoneAndInboxType(_ context.Context, phone, inboxType string) (*chatwoot.Config, error) {
	if m.cfg != nil && m.cfg.InboxType == inboxType {
		return m.cfg, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCloudWARepo) Upsert(_ context.Context, _ *chatwoot.Config) error { return nil }
func (m *mockCloudWARepo) Delete(_ context.Context, _ string) error           { return nil }

type mockCloudWAPresigner struct {
	url string
	err error
}

func (m *mockCloudWAPresigner) GetPresignedURL(_ context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.url != "" {
		return m.url, nil
	}
	return "https://minio.example.com/bucket/" + key, nil
}

func newCloudWAAPIApp(repo chatwoot.Repo, presigner CloudPresigner) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	h := NewCloudAPIHandler(repo, nil, presigner, nil)
	app.Get("/:version/:phone/messages", h.VerifyWebhook)
	app.Post("/:version/:phone/messages", h.SendMessage)
	app.Get("/:version/:phone/:media_id", h.GetMedia)
	return app
}

func TestVerifyWebhook_ValidToken(t *testing.T) {
	repo := &mockCloudWARepo{
		cfg: &chatwoot.Config{
			SessionID:    "sess1",
			InboxType:    "cloud",
			WebhookToken: "my-verify-token",
			Enabled:      true,
		},
	}
	app := newCloudWAAPIApp(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/v17.0/5511999999999/messages?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestVerifyWebhook_InvalidToken(t *testing.T) {
	repo := &mockCloudWARepo{
		cfg: &chatwoot.Config{
			SessionID:    "sess1",
			InboxType:    "cloud",
			WebhookToken: "correct-token",
			Enabled:      true,
		},
	}
	app := newCloudWAAPIApp(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/v17.0/5511999999999/messages?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestGetMedia_Found(t *testing.T) {
	repo := &mockCloudWARepo{
		cfg: &chatwoot.Config{
			SessionID:    "sess1",
			InboxType:    "cloud",
			WebhookToken: "token",
			Enabled:      true,
		},
	}
	presigner := &mockCloudWAPresigner{url: "https://minio.example.com/bucket/chatwoot/sess1/msg123"}
	app := newCloudWAAPIApp(repo, presigner)

	req := httptest.NewRequest(http.MethodGet, "/v17.0/5511999999999/msg123", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body dto.CloudAPIMediaResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body.URL == "" {
		t.Fatal("expected non-empty URL")
	}
	if body.ID != "msg123" {
		t.Fatalf("expected id msg123, got %s", body.ID)
	}
	if body.MessagingProduct != "whatsapp" {
		t.Fatalf("expected messaging_product whatsapp, got %s", body.MessagingProduct)
	}
}

func TestGetMedia_NotFound(t *testing.T) {
	repo := &mockCloudWARepo{
		cfg: &chatwoot.Config{
			SessionID:    "sess1",
			InboxType:    "cloud",
			WebhookToken: "token",
			Enabled:      true,
		},
	}
	presigner := &mockCloudWAPresigner{err: fmt.Errorf("not found")}
	app := newCloudWAAPIApp(repo, presigner)

	req := httptest.NewRequest(http.MethodGet, "/v17.0/5511999999999/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSendMessage_InvalidAuth(t *testing.T) {
	repo := &mockCloudWARepo{
		cfg: &chatwoot.Config{
			SessionID:    "sess1",
			InboxType:    "cloud",
			WebhookToken: "correct-token",
			Enabled:      true,
		},
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := NewCloudAPIHandler(repo, nil, nil, nil)
	app.Post("/:version/:phone/messages", h.SendMessage)

	body, _ := json.Marshal(dto.CloudAPIMessageReq{
		MessagingProduct: "whatsapp",
		To:               "5511988887777",
		Type:             "text",
		Text:             &dto.CloudAPIText{Body: "hello"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v17.0/5511999999999/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var errResp dto.CloudAPIErrorResp
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if errResp.Error.Code != 190 {
		t.Fatalf("expected error code 190, got %d", errResp.Error.Code)
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"+5511999999999", "5511999999999"},
		{"5511999999999", "5511999999999"},
		{"55-11-999999999", "5511999999999"},
		{"+1 (650) 555-1234", "16505551234"},
	}
	for _, tt := range tests {
		got := normalizePhone(tt.input)
		if got != tt.expected {
			t.Errorf("normalizePhone(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
