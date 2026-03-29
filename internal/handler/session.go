package handler

import (
	"encoding/base64"
	"errors"

	"wzap/internal/dto"
	"wzap/internal/service"
	"wzap/internal/whatsapp"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
)

type SessionHandler struct {
	sessionSvc *service.SessionService
	engine     *whatsapp.Engine
}

func NewSessionHandler(sessionSvc *service.SessionService, engine *whatsapp.Engine) *SessionHandler {
	return &SessionHandler{
		sessionSvc: sessionSvc,
		engine:     engine,
	}
}

// Create godoc
// @Summary     Create a new session (Admin Only)
// @Description Creates a new session with an auto-generated or custom apiKey
// @Tags        Sessions
// @Accept      json
// @Produce     json
// @Param       body body     dto.SessionCreateReq true "Session data"
// @Success     200  {object} dto.APIResponse
// @Failure     400  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions [post]
func (h *SessionHandler) Create(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
	}

	var req dto.SessionCreateReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	session, err := h.sessionSvc.Create(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(session, "Session created"))
}

// List godoc
// @Summary     List sessions (Admin Only)
// @Description Returns all sessions
// @Tags        Sessions
// @Produce     json
// @Success     200 {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions [get]
func (h *SessionHandler) List(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
	}

	sessions, err := h.sessionSvc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(sessions, "Sessions retrieved"))
}

// Get godoc
// @Summary     Get session
// @Description Returns the session identified by :sessionName (name or id)
// @Tags        Sessions
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName} [get]
func (h *SessionHandler) Get(c *fiber.Ctx) error {
	id := c.Locals("sessionId").(string)
	session, err := h.sessionSvc.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(dto.SuccessResp(session, "Session retrieved"))
}

// Delete godoc
// @Summary     Delete session
// @Description Disconnects and deletes the session
// @Tags        Sessions
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName} [delete]
func (h *SessionHandler) Delete(c *fiber.Ctx) error {
	id := c.Locals("sessionId").(string)
	if err := h.sessionSvc.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(nil, "Session deleted"))
}

// Connect godoc
// @Summary     Connect session
// @Description Connects a WhatsApp session (starts pairing if new)
// @Tags        Sessions
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName}/connect [post]
func (h *SessionHandler) Connect(c *fiber.Ctx) error {
	id := c.Locals("sessionId").(string)

	client, qrChan, err := h.engine.Connect(c.Context(), id)
	if err != nil {
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", "A QR code connection is already pending for this session"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Connection Error", err.Error()))
	}

	status := "CONNECTED"
	if qrChan != nil {
		status = "PAIRING"
	} else if client != nil && !client.IsConnected() {
		status = "CONNECTING"
	}

	return c.JSON(dto.SuccessResp(map[string]string{"status": status}, "Connection initiated"))
}

// Disconnect godoc
// @Summary     Disconnect session
// @Description Disconnects the active WhatsApp session
// @Tags        Sessions
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName}/disconnect [post]
func (h *SessionHandler) Disconnect(c *fiber.Ctx) error {
	id := c.Locals("sessionId").(string)
	if err := h.engine.Disconnect(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Disconnect Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(nil, "Disconnected successfully"))
}

// QR godoc
// @Summary     Get QR code for pairing
// @Description Returns a QR code for pairing a new WhatsApp device
// @Tags        Sessions
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName}/qr [get]
func (h *SessionHandler) QR(c *fiber.Ctx) error {
	id := c.Locals("sessionId").(string)

	qrCode, err := h.engine.GetQRCode(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	if qrCode == "" {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "No QR code available. Call connect first, then poll this endpoint."))
	}

	imageBytes, imgErr := qrcode.Encode(qrCode, qrcode.Medium, 256)
	qrBase64 := ""
	if imgErr == nil {
		qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
	}

	return c.JSON(dto.SuccessResp(map[string]interface{}{
		"qr":    qrCode,
		"image": qrBase64,
	}, "QR Code retrieved"))
}
