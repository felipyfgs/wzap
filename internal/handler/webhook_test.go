package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wzap/internal/dto"
	"wzap/internal/handler"
	"wzap/internal/middleware"
	"wzap/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func newWebhookApp(webhookSvc *service.WebhookService) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	h := handler.NewWebhookHandler(webhookSvc)
	sess := app.Group("/sessions/:sessionId")
	sess.Use(func(c *fiber.Ctx) error {
		c.Locals("sessionID", c.Params("sessionId"))
		return c.Next()
	})
	sess.Post("/webhooks", h.Create)
	sess.Get("/webhooks", h.List)
	sess.Put("/webhooks/:wid", h.Update)
	sess.Delete("/webhooks/:wid", h.Delete)
	_ = middleware.Validate
	return app
}

func TestWebhookCreate_BadJSON(t *testing.T) {
	app := newWebhookApp(service.NewWebhookService(nil))
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/webhooks", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestWebhookCreate_MissingRequiredFields(t *testing.T) {
	app := newWebhookApp(service.NewWebhookService(nil))
	body, _ := json.Marshal(dto.CreateWebhookReq{Events: []string{"Message"}})
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing URL, got %d", resp.StatusCode)
	}
}

func TestWebhookUpdate_BadJSON(t *testing.T) {
	app := newWebhookApp(service.NewWebhookService(nil))
	req := httptest.NewRequest(http.MethodPut, "/sessions/sess1/webhooks/wid1", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}
