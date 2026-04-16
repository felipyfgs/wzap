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
	"wzap/internal/middleware"
	"wzap/internal/service"
)

var _ = middleware.Validate

func newMessageApp() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	h := handler.NewMessageHandler(service.NewMessageService(nil, nil, nil))
	sess := app.Group("/sessions/:sessionId/messages")
	sess.Use(func(c *fiber.Ctx) error {
		c.Locals("sessionID", c.Params("sessionId"))
		return c.Next()
	})
	sess.Post("/text", h.SendText)
	sess.Post("/button", h.SendButton)
	sess.Post("/list", h.SendList)
	return app
}

func TestSendText_MissingPhone(t *testing.T) {
	app := newMessageApp()
	body, _ := json.Marshal(dto.SendTextReq{Body: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/messages/text", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing phone, got %d", resp.StatusCode)
	}
}

func TestSendText_MissingBody(t *testing.T) {
	app := newMessageApp()
	body, _ := json.Marshal(dto.SendTextReq{Phone: "5511999999999"})
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/messages/text", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing body, got %d", resp.StatusCode)
	}
}

func TestSendText_BadJSON(t *testing.T) {
	app := newMessageApp()
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/messages/text", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", resp.StatusCode)
	}
}

func TestSendButton_EmptyButtons(t *testing.T) {
	app := newMessageApp()
	body, _ := json.Marshal(dto.SendButtonReq{
		Phone:   "5511999999999",
		Body:    "Choose",
		Buttons: []dto.ButtonItem{},
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/messages/button", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty buttons, got %d", resp.StatusCode)
	}
}

func TestSendList_MissingButtonText(t *testing.T) {
	app := newMessageApp()
	body, _ := json.Marshal(dto.SendListReq{
		Phone:    "5511999999999",
		Title:    "My List",
		Body:     "Pick one",
		Sections: []dto.ListSection{{Rows: []dto.ListRow{{ID: "1", Title: "Row"}}}},
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/messages/list", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing buttonText, got %d", resp.StatusCode)
	}
}
