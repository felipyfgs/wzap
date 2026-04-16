package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"wzap/internal/dto"
	"wzap/internal/handler"
	"wzap/internal/model"
	"wzap/internal/service"
)

type stubLifecycleService struct {
	connectResult    *service.ConnectResult
	connectErr       error
	disconnectResult *service.DisconnectResult
	disconnectErr    error
	qrResult         *service.QRResult
	qrErr            error
	pairResult       *service.PairResult
	pairErr          error
	logoutResult     *service.LogoutResult
	logoutErr        error
	reconnectResult  *service.ReconnectResult
	reconnectErr     error
	restartResult    *service.RestartResult
	restartErr       error
}

func (s *stubLifecycleService) Connect(ctx context.Context, sessionID string) (*service.ConnectResult, error) {
	return s.connectResult, s.connectErr
}

func (s *stubLifecycleService) Disconnect(ctx context.Context, sessionID string) (*service.DisconnectResult, error) {
	return s.disconnectResult, s.disconnectErr
}

func (s *stubLifecycleService) QR(ctx context.Context, sessionID string) (*service.QRResult, error) {
	return s.qrResult, s.qrErr
}

func (s *stubLifecycleService) Pair(ctx context.Context, sessionID, phone string) (*service.PairResult, error) {
	return s.pairResult, s.pairErr
}

func (s *stubLifecycleService) Logout(ctx context.Context, sessionID string) (*service.LogoutResult, error) {
	return s.logoutResult, s.logoutErr
}

func (s *stubLifecycleService) Reconnect(ctx context.Context, sessionID string) (*service.ReconnectResult, error) {
	return s.reconnectResult, s.reconnectErr
}

func (s *stubLifecycleService) Restart(ctx context.Context, sessionID string) (*service.RestartResult, error) {
	return s.restartResult, s.restartErr
}

func newSessionLifecycleApp(lifecycle *stubLifecycleService) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	h := handler.NewSessionHandler(nil, lifecycle, nil)
	sess := app.Group("/sessions/:sessionId")
	sess.Use(func(c *fiber.Ctx) error {
		c.Locals("sessionID", c.Params("sessionId"))
		return c.Next()
	})
	sess.Post("/connect", h.Connect)
	sess.Get("/qr", h.QR)
	sess.Post("/pair", h.Pair)
	sess.Post("/restart", h.Restart)
	return app
}

func TestSessionConnect_Conflict(t *testing.T) {
	app := newSessionLifecycleApp(&stubLifecycleService{
		connectErr: &service.ConflictError{Message: "A QR code connection is already pending for this session"},
	})

	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/connect", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestSessionQR_NotFound(t *testing.T) {
	app := newSessionLifecycleApp(&stubLifecycleService{
		qrErr: &service.NotFoundError{Message: "No QR code available. Call connect first, then poll this endpoint."},
	})

	req := httptest.NewRequest(http.MethodGet, "/sessions/sess1/qr", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSessionPair_NotSupported(t *testing.T) {
	app := newSessionLifecycleApp(&stubLifecycleService{
		pairErr: &service.CapabilityError{Engine: "cloud_api", Capability: model.CapabilitySessionPair, Support: model.SupportUnavailable},
	})
	body, _ := json.Marshal(dto.PairPhoneReq{Phone: "5511999999999"})

	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/pair", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSessionRestart_Success(t *testing.T) {
	app := newSessionLifecycleApp(&stubLifecycleService{
		restartResult: &service.RestartResult{Session: &dto.SessionResp{ID: "sess1", Name: "sess1", Engine: "whatsmeow"}},
	})

	req := httptest.NewRequest(http.MethodPost, "/sessions/sess1/restart", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
