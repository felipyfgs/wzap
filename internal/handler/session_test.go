package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"wzap/internal/dto"
	"wzap/internal/handler"
	"wzap/internal/service"
)

func newSessionApp(sessionSvc *service.SessionService) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	h := handler.NewSessionHandler(sessionSvc, nil, nil, nil)

	// Admin-scoped routes
	grp := app.Group("/sessions")
	grp.Post("/", func(c *fiber.Ctx) error {
		c.Locals("authRole", "admin")
		return c.Next()
	}, h.Create)

	// Session-scoped routes
	sess := app.Group("/sessions/:sessionId")
	sess.Use(func(c *fiber.Ctx) error {
		c.Locals("sessionID", c.Params("sessionId"))
		return c.Next()
	})
	sess.Get("/", h.Get)
	sess.Put("/", h.Update)
	sess.Delete("/", h.Delete)
	sess.Get("/status", h.Status)
	sess.Post("/pair", h.Pair)

	return app
}

func TestSessionCreate_BadJSON(t *testing.T) {
	app := newSessionApp(service.NewSessionService(nil, nil, nil, nil))
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSessionCreate_MissingName(t *testing.T) {
	app := newSessionApp(service.NewSessionService(nil, nil, nil, nil))
	body, _ := json.Marshal(dto.SessionCreateReq{})
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", resp.StatusCode)
	}
}

func TestSessionPair_MissingPhone(t *testing.T) {
	app := newSessionApp(service.NewSessionService(nil, nil, nil, nil))
	body, _ := json.Marshal(dto.PairPhoneReq{})
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/pair", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing phone, got %d", resp.StatusCode)
	}
}

func TestSessionUpdate_ValidEmptyBody(t *testing.T) {
	app := newSessionApp(service.NewSessionService(nil, nil, nil, nil))
	body, _ := json.Marshal(dto.SessionUpdateReq{})
	req := httptest.NewRequest(http.MethodPut, "/sessions/sess1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	// Empty update with nil repo should 500 (no db), not 400
	if resp.StatusCode == http.StatusBadRequest {
		t.Error("empty update should not be a 400")
	}
}
