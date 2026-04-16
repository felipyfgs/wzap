package handler

import (
	"context"
	"encoding/base64"

	"wzap/internal/dto"
	"wzap/internal/integrations/chatwoot"
	"wzap/internal/logger"
	"wzap/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/skip2/go-qrcode"
)

type sessionLifecycle interface {
	Connect(ctx context.Context, sessionID string) (*service.ConnectResult, error)
	Disconnect(ctx context.Context, sessionID string) (*service.DisconnectResult, error)
	QR(ctx context.Context, sessionID string) (*service.QRResult, error)
	Pair(ctx context.Context, sessionID, phone string) (*service.PairResult, error)
	Logout(ctx context.Context, sessionID string) (*service.LogoutResult, error)
	Reconnect(ctx context.Context, sessionID string) (*service.ReconnectResult, error)
	Restart(ctx context.Context, sessionID string) (*service.RestartResult, error)
}

type SessionHandler struct {
	sessionSvc   *service.SessionService
	lifecycleSvc sessionLifecycle
	chatwootRepo *chatwoot.Repository
}

func NewSessionHandler(sessionSvc *service.SessionService, lifecycleSvc sessionLifecycle, chatwootRepo *chatwoot.Repository) *SessionHandler {
	return &SessionHandler{
		sessionSvc:   sessionSvc,
		lifecycleSvc: lifecycleSvc,
		chatwootRepo: chatwootRepo,
	}
}

// Create godoc
// @Summary     Create a new session (Admin Only)
// @Description Creates a new session with an auto-generated or custom token. Returns the full session object including the token.
// @Tags        Sessions
// @Accept      json
// @Produce     json
// @Param       body body dto.SessionCreateReq true "Session data"
// @Success     201 {object} dto.APIResponse{Data=dto.SessionWithTokenResp}
// @Failure     400 {object} dto.APIError
// @Failure     403 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions [post]
func (h *SessionHandler) Create(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
	}

	var req dto.SessionCreateReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	session, err := h.sessionSvc.Create(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(session))
}

// List godoc
// @Summary     List sessions (Admin Only)
// @Description Returns all sessions. Token is never included in list responses.
// @Tags        Sessions
// @Produce     json
// @Success     200 {object} dto.APIResponse{Data=[]dto.SessionResp}
// @Failure     403 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions [get]
func (h *SessionHandler) List(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
	}

	sessions, err := h.sessionSvc.List(c.Context())
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Msg("failed to list sessions")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(sessions))
}

// Get godoc
// @Summary     Get session
// @Description Returns the session identified by :sessionId (name or id). Token is not included.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{Data=dto.SessionResp}
// @Failure     404 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId} [get]
func (h *SessionHandler) Get(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	session, err := h.sessionSvc.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	if h.chatwootRepo != nil {
		if _, err := h.chatwootRepo.FindBySessionID(c.Context(), id); err == nil {
			session.ChatwootEnabled = true
		}
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
// @Security    Authorization
// @Router      /sessions/{sessionId} [delete]
func (h *SessionHandler) Delete(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	if err := h.sessionSvc.Delete(c.Context(), id); err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to delete session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

// Update godoc
// @Summary     Update session
// @Description Updates session fields (name, proxy, settings). Only provided fields are changed.
// @Tags        Sessions
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SessionUpdateReq true "Session update data"
// @Success     200 {object} dto.APIResponse{Data=dto.SessionResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId} [put]
func (h *SessionHandler) Update(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	var req dto.SessionUpdateReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	session, err := h.sessionSvc.Update(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Update Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(session))
}

// Status godoc
// @Summary     Get session status
// @Description Returns detailed connection status (connected, loggedIn, JID)
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{Data=dto.SessionStatusResp}
// @Failure     404 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/status [get]
func (h *SessionHandler) Status(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	status, err := h.sessionSvc.Status(c.Context(), id)
	if err != nil {
		if handleCapabilityError(c, err) {
			return nil
		}
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(dto.SuccessResp(status))
}

// Connect godoc
// @Summary     Connect session
// @Description Connects a WhatsApp session. Returns status CONNECTED, PAIRING (QR required), or CONNECTING.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{Data=dto.ConnectResp}
// @Failure     409 {object} dto.APIError "QR pairing already pending"
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/connect [post]
func (h *SessionHandler) Connect(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	result, err := h.lifecycleSvc.Connect(c.Context(), id)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to connect session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Connection Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(dto.ConnectResp{Status: result.Status}))
}

// Disconnect godoc
// @Summary     Disconnect session
// @Description Disconnects the active WhatsApp session without removing the device (can reconnect later)
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/disconnect [post]
func (h *SessionHandler) Disconnect(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	_, err := h.lifecycleSvc.Disconnect(c.Context(), id)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to disconnect session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Disconnect Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

// QR godoc
// @Summary     Get QR code for pairing
// @Description Returns the current QR code string and a base64 PNG image. Call /connect first, then poll this endpoint until a code is available.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{Data=dto.QRResp}
// @Failure     404 {object} dto.APIError "No QR code available yet"
// @Security    Authorization
// @Router      /sessions/{sessionId}/qr [get]
func (h *SessionHandler) QR(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	result, err := h.lifecycleSvc.QR(c.Context(), id)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	imageBytes, imgErr := qrcode.Encode(result.QRCode, qrcode.Medium, 256)
	qrBase64 := ""
	if imgErr == nil {
		qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
	}

	return c.JSON(dto.SuccessResp(dto.QRResp{QRCode: result.QRCode, Image: qrBase64}))
}

// Pair godoc
// @Summary     Pair via phone number
// @Description Generates a pairing code for linking without QR scan. Call /connect first.
// @Tags        Sessions
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.PairPhoneReq true "Phone number"
// @Success     200 {object} dto.APIResponse{Data=dto.PairPhoneResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/pair [post]
func (h *SessionHandler) Pair(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	var req dto.PairPhoneReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	result, err := h.lifecycleSvc.Pair(c.Context(), id, req.Phone)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to pair phone")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Pair Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(dto.PairPhoneResp{PairingCode: result.PairingCode}))
}

// Logout godoc
// @Summary     Logout session
// @Description Logs out the WhatsApp device (requires re-scan), but keeps the session record
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/logout [post]
func (h *SessionHandler) Logout(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	_, err := h.lifecycleSvc.Logout(c.Context(), id)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to logout session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Logout Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

// Profile godoc
// @Summary     Get WhatsApp profile
// @Description Returns the WhatsApp profile info for this session (push name, picture, bio). Requires an active connection.
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{Data=dto.SessionProfileResp}
// @Failure     404 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/profile [get]
func (h *SessionHandler) Profile(c *fiber.Ctx) error {
	id := mustGetSessionID(c)
	profile, err := h.sessionSvc.Profile(c.Context(), id)
	if err != nil {
		if handleCapabilityError(c, err) {
			return nil
		}
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}
	return c.JSON(dto.SuccessResp(profile))
}

// Reconnect godoc
// @Summary     Reconnect session
// @Description Disconnects and reconnects the session
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/reconnect [post]
func (h *SessionHandler) Reconnect(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	_, err := h.lifecycleSvc.Reconnect(c.Context(), id)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to reconnect session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Reconnect Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}

// Restart godoc
// @Summary     Restart session
// @Description Disconnects and reconnects the session without losing state
// @Tags        Sessions
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse{Data=dto.SessionResp}
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/restart [post]
func (h *SessionHandler) Restart(c *fiber.Ctx) error {
	id := mustGetSessionID(c)

	result, err := h.lifecycleSvc.Restart(c.Context(), id)
	if err != nil {
		if handleLifecycleError(c, err) {
			return nil
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to restart session")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Restart Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(result.Session))
}
