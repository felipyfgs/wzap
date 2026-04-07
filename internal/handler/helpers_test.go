package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"wzap/internal/dto"
	"wzap/internal/model"
	"wzap/internal/service"
)

func newHelpersApp(err error) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	app.Get("/", func(c *fiber.Ctx) error {
		if handleCapabilityError(c, err) {
			return nil
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	return app
}

func TestHandleCapabilityErrorUnavailable(t *testing.T) {
	app := newHelpersApp(&service.CapabilityError{
		Engine:     "cloud_api",
		Capability: model.CapabilityMessagePoll,
		Support:    model.CapabilitySupportUnavailable,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var body dto.APIError
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Error != "Not Supported" {
		t.Fatalf("expected Not Supported, got %q", body.Error)
	}
}

func TestHandleCapabilityErrorPartial(t *testing.T) {
	app := newHelpersApp(&service.CapabilityError{
		Engine:     "cloud_api",
		Capability: model.CapabilityMessageLink,
		Support:    model.CapabilitySupportPartial,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var body dto.APIError
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Error != "Partial Support" {
		t.Fatalf("expected Partial Support, got %q", body.Error)
	}
}

func TestHandleCapabilityErrorIgnoresNonCapabilityErrors(t *testing.T) {
	app := newHelpersApp(errors.New("boom"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}
