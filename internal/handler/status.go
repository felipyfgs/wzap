package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/service"
)

type StatusHandler struct {
	statusSvc *service.StatusService
}

func NewStatusHandler(statusSvc *service.StatusService) *StatusHandler {
	return &StatusHandler{statusSvc: statusSvc}
}

func (h *StatusHandler) respondStatusError(c *fiber.Ctx, err error, sessionID, logMsg string) error {
	if handleCapabilityError(c, err) {
		return nil
	}

	logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg(logMsg)
	return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Status Error", "internal server error"))
}

// SendText godoc
// @Summary     Send a text status
// @Description Sends a text message to WhatsApp Stories/Status and persists it
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.StatusTextReq true "Status text payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/status/text [post]
func (h *StatusHandler) SendText(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.StatusTextReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	msgID, err := h.statusSvc.SendStatusText(c.Context(), id, req)
	if err != nil {
		return h.respondStatusError(c, err, id, "failed to send status text")
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendImage godoc
// @Summary     Send an image status
// @Description Sends an image to WhatsApp Stories/Status and persists it
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.StatusMediaReq true "Status image payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/status/image [post]
func (h *StatusHandler) SendImage(c *fiber.Ctx) error {
	return h.sendStatusMedia(c, whatsmeow.MediaImage)
}

// SendVideo godoc
// @Summary     Send a video status
// @Description Sends a video to WhatsApp Stories/Status and persists it
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.StatusMediaReq true "Status video payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/status/video [post]
func (h *StatusHandler) SendVideo(c *fiber.Ctx) error {
	return h.sendStatusMedia(c, whatsmeow.MediaVideo)
}

func (h *StatusHandler) sendStatusMedia(c *fiber.Ctx, mediaType whatsmeow.MediaType) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.StatusMediaReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	if req.Base64 == "" && req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "either base64 or url is required"))
	}

	msgID, err := h.statusSvc.SendStatusMedia(c.Context(), id, req, mediaType)
	if err != nil {
		return h.respondStatusError(c, err, id, "failed to send status media")
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// ListStatus godoc
// @Summary     List statuses
// @Description Returns the list of active (non-expired) statuses for a session
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       limit query int false "Limit (default 50, max 200)" default(50)
// @Param       offset query int false "Offset" default(0)
// @Success     200 {object} dto.APIResponse{Data=[]model.Status}
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/status [get]
func (h *StatusHandler) ListStatus(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	statuses, err := h.statusSvc.ListStatus(c.Context(), id, limit, offset)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", id).Msg("failed to list statuses")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("List Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(statuses))
}

// ListContactStatus godoc
// @Summary     List contact statuses
// @Description Returns the list of active statuses from a specific sender JID
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       senderJid path string true "Sender JID"
// @Success     200 {object} dto.APIResponse{Data=[]model.Status}
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/status/{senderJid} [get]
func (h *StatusHandler) ListContactStatus(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	senderJID := c.Params("senderJid")
	if senderJID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "senderJid is required"))
	}

	statuses, err := h.statusSvc.ListContactStatus(c.Context(), id, senderJID)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", id).Msg("failed to list contact statuses")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("List Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(statuses))
}
