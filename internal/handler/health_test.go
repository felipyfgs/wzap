package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wzap/internal/dto"
	"wzap/internal/handler"

	"github.com/gofiber/fiber/v2"
)

func TestHealthHandler_Check_NilDependencies(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h := handler.NewHealthHandler(nil, nil, nil)
	app.Get("/health", h.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var body dto.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}
