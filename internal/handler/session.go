package handler

import (
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"wzap/internal/model"
	"wzap/internal/service"
)

type SessionHandler struct {
	sessionSvc *service.SessionService
	engine     *service.Engine
}

func NewSessionHandler(sessionSvc *service.SessionService, engine *service.Engine) *SessionHandler {
	return &SessionHandler{
		sessionSvc: sessionSvc,
		engine:     engine,
	}
}

func (h *SessionHandler) getSessionID(c *fiber.Ctx) (string, error) {
	if id := c.Params("id"); id != "" {
		return id, nil
	}
	if val := c.Locals("session_id"); val != nil {
		return val.(string), nil
	}
	if id := c.Get("X-Session-ID"); id != "" {
		return id, nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required (path :id, auth token, or header X-Session-ID)")
}

// Create godoc
// @Summary     Create a new session (Admin Only)
// @Description Creates a new WhatsApp session entry in the database. Returns the generated api_key.
// @Tags        Sessions
// @Accept      json
// @Produce     json
// @Param       body body     model.SessionCreateReq true "Session data"
// @Success     200  {object} model.APIResponse
// @Failure     400  {object} model.APIError
// @Security    BearerAuth
// @Router      /sessions [post]
func (h *SessionHandler) Create(c *fiber.Ctx) error {
	if c.Locals("auth_role") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Admin access required"))
	}

	var req model.SessionCreateReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if req.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "id is required"))
	}

	session, err := h.sessionSvc.Create(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(session, "Session created successfully"))
}

// List godoc
// @Summary     List sessions
// @Description Returns all sessions (Admin) or just the authenticated session (User)
// @Tags        Sessions
// @Produce     json
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /sessions [get]
func (h *SessionHandler) List(c *fiber.Ctx) error {
	if c.Locals("auth_role") == "session" {
		id := c.Locals("session_id").(string)
		session, err := h.sessionSvc.Get(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
		}
		return c.JSON(model.SuccessResp([]*model.Session{session}, "Session retrieved"))
	}

	sessions, err := h.sessionSvc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(sessions, "Sessions retrieved"))
}

// Get godoc
// @Summary     Get current session
// @Description Returns the session identified by the Bearer token (or query/header fallback)
// @Tags        Sessions
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /session [get]
func (h *SessionHandler) Get(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	session, err := h.sessionSvc.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(model.SuccessResp(session, "Session retrieved"))
}

// Delete godoc
// @Summary     Delete current session
// @Description Disconnects and deletes the current session
// @Tags        Sessions
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /session [delete]
func (h *SessionHandler) Delete(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	err = h.sessionSvc.Delete(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(nil, "Session deleted"))
}

// Connect godoc
// @Summary     Connect current session
// @Description Connects a WhatsApp session (starts pairing if new)
// @Tags        Sessions
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /session/connect [post]
func (h *SessionHandler) Connect(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	_, err = h.sessionSvc.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", "Session not found"))
	}

	client, _, err := h.engine.Connect(c.Context(), id)
	if err != nil {
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			return c.Status(fiber.StatusConflict).JSON(model.ErrorResp("Conflict", "A QR code connection is already pending for this session"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Connection Error", err.Error()))
	}

	status := "CONNECTING"
	if client != nil && client.IsConnected() {
		status = "CONNECTED"
	}

	return c.JSON(model.SuccessResp(map[string]string{"status": status}, "Connection initiated"))
}

// Disconnect godoc
// @Summary     Disconnect current session
// @Description Disconnects the active WhatsApp session
// @Tags        Sessions
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /session/disconnect [post]
func (h *SessionHandler) Disconnect(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	err = h.engine.Disconnect(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Disconnect Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(nil, "Disconnected successfully"))
}

// QR godoc
// @Summary     Get QR code for pair
// @Description Returns a QR code for pairing a new WhatsApp device for current session
// @Tags        Sessions
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /session/qr [get]
func (h *SessionHandler) QR(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}

	_, qrChan, err := h.engine.Connect(c.Context(), id)
	if err != nil {
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			return c.Status(fiber.StatusConflict).JSON(model.ErrorResp("Conflict", "QR Code is already actively requested or pending on this session channel"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("QR Error", err.Error()))
	}

	if qrChan == nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "Client already connected or already paired"))
	}

	select {
	case evt, ok := <-qrChan:
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("QR Error", "QR channel closed unexpectedly"))
		}
		if evt.Event != "code" {
			return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("QR Error", "Failed to retrieve QR code"))
		}

		qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

		imageBytes, imgErr := qrcode.Encode(evt.Code, qrcode.Medium, 256)
		qrBase64 := ""
		if imgErr == nil {
			qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
		}

		return c.JSON(model.SuccessResp(map[string]interface{}{
			"qr":      evt.Code,
			"image":   qrBase64,
			"timeout": evt.Timeout,
		}, "QR Code retrieved"))
	case <-time.After(45 * time.Second):
		return c.Status(fiber.StatusRequestTimeout).JSON(model.ErrorResp("QR Timeout", "QR code request timed out after 45 seconds"))
	}
}
