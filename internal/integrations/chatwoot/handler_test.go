package chatwoot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"wzap/internal/dto"
)

func newChatwootApp() (*fiber.App, *mockRepo, *mockClient) {
	mockClient := &mockClient{
		contacts:      []Contact{},
		conversations: []Conversation{},
	}
	repository := &mockRepo{}

	svc := &Service{
		repo:     repository,
		msgRepo:  &mockMsgRepo{},
		clientFn: func(cfg *Config) Client { return mockClient },
		cache:    newMemoryCache(context.Background()),
		cb:       newCircuitBreakerManager(),
	}

	h := NewHandler(svc, repository)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())

	sessionMW := func(c *fiber.Ctx) error {
		c.Locals("authRole", "admin")
		c.Locals("sessionID", "test-session")
		return c.Next()
	}

	app.Put("/sessions/:sessionId/integrations/chatwoot", sessionMW, h.Configure)
	app.Get("/sessions/:sessionId/integrations/chatwoot", sessionMW, h.GetConfig)
	app.Delete("/sessions/:sessionId/integrations/chatwoot", sessionMW, h.DeleteConfig)
	app.Post("/sessions/:sessionId/integrations/chatwoot/import", sessionMW, h.ImportHistory)
	app.Post("/chatwoot/webhook/:sessionId", h.IncomingWebhook)

	return app, repository, mockClient
}

func TestConfigure_ValidationMissingURL(t *testing.T) {
	app, _, _ := newChatwootApp()

	body, _ := json.Marshal(dto.ChatwootConfigReq{
		AccountID: 1,
		Token:     "test-token",
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestConfigure_ValidationMissingToken(t *testing.T) {
	app, _, _ := newChatwootApp()

	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:       "https://app.chatwoot.com",
		AccountID: 1,
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestConfigure_ValidationMissingAccountID(t *testing.T) {
	app, _, _ := newChatwootApp()

	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:   "https://app.chatwoot.com",
		Token: "test-token",
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestConfigure_InvalidURL(t *testing.T) {
	app, _, _ := newChatwootApp()

	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:       "not-a-url",
		AccountID: 1,
		Token:     "test-token",
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestConfigure_SuccessEnvelope(t *testing.T) {
	app, _, _ := newChatwootApp()

	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:       "https://app.chatwoot.com",
		AccountID: 1,
		Token:     "test-token",
		InboxID:   1,
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("expected success=true, got %v", result["success"])
	}
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}
	if data["sessionId"] != "test-session" {
		t.Errorf("expected sessionId=test-session, got %v", data["sessionId"])
	}
	if data["url"] != "https://app.chatwoot.com" {
		t.Errorf("expected url, got %v", data["url"])
	}
}

func TestGetConfig_ResponseShapeParity(t *testing.T) {
	app, repo, _ := newChatwootApp()

	repo.cfg = &Config{
		SessionID:  "test-session",
		URL:        "https://app.chatwoot.com",
		AccountID:  1,
		InboxID:    5,
		IgnoreJIDs: []string{"@g.us", "5511@s.whatsapp.net"},
		Enabled:    true,
	}

	req := httptest.NewRequest("GET", "/sessions/test-session/integrations/chatwoot", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if success, ok := result["success"].(bool); !ok || !success {
		t.Errorf("expected success=true, got %v", result["success"])
	}

	data, _ := result["data"].(map[string]interface{})

	if ignoreGroups, ok := data["ignoreGroups"].(bool); !ok || !ignoreGroups {
		t.Error("expected ignoreGroups=true because @g.us is in ignoreJids")
	}

	jids, _ := data["ignoreJids"].([]interface{})
	if len(jids) != 2 {
		t.Errorf("expected 2 ignoreJids, got %d", len(jids))
	}

	if data["inboxId"].(float64) != 5 {
		t.Errorf("expected inboxId=5, got %v", data["inboxId"])
	}
}

func TestGetConfig_MasksRedisURL(t *testing.T) {
	app, repo, _ := newChatwootApp()
	repo.cfg = &Config{
		SessionID: "test-session",
		URL:       "https://app.chatwoot.com",
		AccountID: 1,
		InboxID:   5,
		RedisURL:  "redis://:secret@redis.host:6379/0",
		Enabled:   true,
	}

	req := httptest.NewRequest("GET", "/sessions/test-session/integrations/chatwoot", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data["redisUrl"] != "redis://***@redis.host:6379/0" {
		t.Errorf("expected masked redisUrl, got %v", data["redisUrl"])
	}
}

func TestConfigure_IgnoreGroupsCompatibilityWrite(t *testing.T) {
	app, repo, _ := newChatwootApp()

	ignoreGroups := true
	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:          "https://app.chatwoot.com",
		AccountID:    1,
		Token:        "test-token",
		InboxID:      1,
		IgnoreGroups: &ignoreGroups,
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if repo.cfg == nil {
		t.Fatal("config not saved")
	}
	found := false
	for _, jid := range repo.cfg.IgnoreJIDs {
		if jid == "@g.us" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ignoreGroups=true should add @g.us to ignoreJids")
	}
}

func TestConfigure_IgnoreGroupsAlreadyInJIDs(t *testing.T) {
	app, repo, _ := newChatwootApp()

	ignoreGroups := true
	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:          "https://app.chatwoot.com",
		AccountID:    1,
		Token:        "test-token",
		InboxID:      1,
		IgnoreGroups: &ignoreGroups,
		IgnoreJIDs:   []string{"@g.us"},
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	count := 0
	for _, jid := range repo.cfg.IgnoreJIDs {
		if jid == "@g.us" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("@g.us should appear exactly once in ignoreJids, found %d", count)
	}
}

func TestGetConfig_NotFound(t *testing.T) {
	app, repo, _ := newChatwootApp()
	repo.notFound = true

	req := httptest.NewRequest("GET", "/sessions/test-session/integrations/chatwoot", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteConfig_Success(t *testing.T) {
	app, _, _ := newChatwootApp()

	req := httptest.NewRequest("DELETE", "/sessions/test-session/integrations/chatwoot", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 204 {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestConfigure_ResponseUsesSuccessEnvelope(t *testing.T) {
	app, _, _ := newChatwootApp()

	body, _ := json.Marshal(dto.ChatwootConfigReq{
		URL:       "https://app.chatwoot.com",
		AccountID: 1,
		Token:     "test-token",
		InboxID:   1,
	})
	req := httptest.NewRequest("PUT", "/sessions/test-session/integrations/chatwoot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if _, ok := result["success"]; !ok {
		t.Error("response should contain 'success' field (standard envelope)")
	}
	if _, ok := result["data"]; !ok {
		t.Error("response should contain 'data' field (standard envelope)")
	}
	if _, ok := result["message"]; !ok {
		t.Error("response should contain 'message' field (standard envelope)")
	}
}

func TestJIDsContainGroup(t *testing.T) {
	if jidsContainGroup([]string{"5511@s.whatsapp.net"}) {
		t.Error("should not contain group marker")
	}
	if !jidsContainGroup([]string{"@g.us"}) {
		t.Error("should contain group marker")
	}
	if !jidsContainGroup([]string{"5511@s.whatsapp.net", "@g.us"}) {
		t.Error("should contain group marker")
	}
	if jidsContainGroup(nil) {
		t.Error("nil should not contain group marker")
	}
}

func TestImportHistory_Returns501(t *testing.T) {
	app, _, _ := newChatwootApp()

	req := httptest.NewRequest("POST", "/sessions/test-session/integrations/chatwoot/import", bytes.NewReader([]byte(`{"period":"7d"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 501 {
		t.Fatalf("expected 501, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	success, _ := result["success"].(bool)
	if success {
		t.Fatal("expected success=false")
	}

	if result["error"] != "Not Implemented" {
		t.Errorf("expected error Not Implemented, got %v", result["error"])
	}
	if result["message"] != "History import is not yet implemented" {
		t.Errorf("unexpected message: %v", result["message"])
	}
}

func TestIncomingWebhook_MissingHMAC_WithToken(t *testing.T) {
	app, repo, _ := newChatwootApp()
	repo.cfg = &Config{
		SessionID:    "test-session",
		Enabled:      true,
		Token:        "test-token",
		WebhookToken: "webhook-secret",
	}

	req := httptest.NewRequest("POST", "/chatwoot/webhook/test-session", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestIncomingWebhook_ValidHMAC(t *testing.T) {
	app, repo, _ := newChatwootApp()
	repo.cfg = &Config{
		SessionID:    "test-session",
		Enabled:      true,
		Token:        "test-token",
		WebhookToken: "webhook-secret",
	}

	body := []byte(`{}`)
	mac := hmac.New(sha256.New, []byte(repo.cfg.WebhookToken))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/chatwoot/webhook/test-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Chatwoot-Hmac-Sha256", signature)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestIncomingWebhook_EmptyToken_NoHMAC(t *testing.T) {
	app, repo, _ := newChatwootApp()
	repo.cfg = &Config{
		SessionID: "test-session",
		Enabled:   true,
		Token:     "",
	}

	req := httptest.NewRequest("POST", "/chatwoot/webhook/test-session", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
