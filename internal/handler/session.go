package handler

import (
	"encoding/base64"
	"errors"

	"github.com/gofiber/fiber/v2"
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
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
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
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Admin access required"))
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

	client, qrChan, err := h.engine.Connect(c.Context(), id)
	if err != nil {
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			return c.Status(fiber.StatusConflict).JSON(model.ErrorResp("Conflict", "A QR code connection is already pending for this session"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Connection Error", err.Error()))
	}

	status := "CONNECTED"
	if qrChan != nil {
		status = "PAIRING"
	} else if client != nil && !client.IsConnected() {
		status = "CONNECTING"
	}

	return c.JSON(model.SuccessResp(map[string]string{"status": status}, "Connection initiated"))
}

// Disconnect godoc
// @Summary     Disconnect current session
// @Description Disconnects the active WhatsApp session
// @Tags        Sessions
// @Produce     json
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
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /session/qr [get]
func (h *SessionHandler) QR(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}

	qrCode, err := h.engine.GetQRCode(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", err.Error()))
	}

	if qrCode == "" {
		return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", "No QR code available. Call /session/connect first, then poll this endpoint."))
	}

	imageBytes, imgErr := qrcode.Encode(qrCode, qrcode.Medium, 256)
	qrBase64 := ""
	if imgErr == nil {
		qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
	}

	return c.JSON(model.SuccessResp(map[string]interface{}{
		"qr":    qrCode,
		"image": qrBase64,
	}, "QR Code retrieved"))
}
