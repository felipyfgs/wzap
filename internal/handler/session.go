package handler

import (
	"encoding/base64"
	"errors"

	"wzap/internal/dto"
	"wzap/internal/service"
	"wzap/internal/wa"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
)

type SessionHandler struct {
	sessionSvc *service.SessionService
	engine     *wa.Manager
}

func NewSessionHandler(sessionSvc *service.SessionService, engine *wa.Manager) *SessionHandler {
	return &SessionHandler{
		sessionSvc: sessionSvc,
		engine:     engine,
	}
}

// Create godoc
// @Summary     Create a new session (Admin Only)
// @Description Creates a new session with an auto-generated or custom apiKey. Returns the full session object including the apiKey.
// @Tags        Sessions
// @Accept      json
// @Produce     json
// @Param       body body     dto.SessionCreateReq true "Session data"
// @Success     201  {object} dto.APIResponse{data=dto.SessionCreatedResp}
// @Failure     400  {object} dto.APIError
// @Failure     403  {object} dto.APIError
// @Security    ApiKey
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

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(session))
}

// List godoc
// @Summary     List sessions (Admin Only)
// @Description Returns all sessions. APIKey is never included in responses.
// @Tags        Sessions
// @Produce     json
// @Success     200 {object} dto.APIResponse{data=[]dto.SessionResp}
// @Failure     403 {object} dto.APIError
// @Security    ApiKey
// @Router      /sessions [get]
func (h *SessionHandler) List(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
	}

	sessions, err := h.sessionSvc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(sessions))
}

// Get godoc
// @Summary     Get session
// @Description Returns the session identified by :sessionId (name or id). APIKey is not included.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{data=dto.SessionResp}
// @Failure     404 {object} dto.APIError
// @Security    ApiKey
// @Router      /sessions/{sessionId} [get]
func (h *SessionHandler) Get(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	session, err := h.sessionSvc.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(dto.SuccessResp(session))
}

// Delete godoc
// @Summary     Delete session
// @Description Disconnects and permanently deletes the session and its device from the store
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    ApiKey
// @Router      /sessions/{sessionId} [delete]
func (h *SessionHandler) Delete(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	if err := h.sessionSvc.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(nil))
}

// Connect godoc
// @Summary     Connect session
// @Description Connects a WhatsApp session. Returns status CONNECTED, PAIRING (QR required), or CONNECTING.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{data=dto.ConnectResp}
// @Failure     409 {object} dto.APIError "QR pairing already pending"
// @Failure     500 {object} dto.APIError
// @Security    ApiKey
// @Router      /sessions/{sessionId}/connect [post]
func (h *SessionHandler) Connect(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

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

	return c.JSON(dto.SuccessResp(dto.ConnectResp{Status: status}))
}

// Disconnect godoc
// @Summary     Disconnect session
// @Description Disconnects the active WhatsApp session without removing the device (can reconnect later)
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    ApiKey
// @Router      /sessions/{sessionId}/disconnect [post]
func (h *SessionHandler) Disconnect(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	if err := h.engine.Disconnect(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Disconnect Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(nil))
}

// QR godoc
// @Summary     Get QR code for pairing
// @Description Returns the current QR code string and a base64 PNG image. Call /connect first, then poll this endpoint until a code is available.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{data=dto.QRResp}
// @Failure     404 {object} dto.APIError "No QR code available yet"
// @Security    ApiKey
// @Router      /sessions/{sessionId}/qr [get]
func (h *SessionHandler) QR(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

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

	return c.JSON(dto.SuccessResp(dto.QRResp{QRCode: qrCode, Image: qrBase64}))
}
