package handler

import (
	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/service"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct {
	webhookSvc *service.WebhookService
}

func NewWebhookHandler(webhookSvc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookSvc: webhookSvc}
}

// Create godoc
// @Summary     Create a webhook
// @Description Registers a new webhook for the session
// @Tags        Webhooks
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.CreateWebhookReq true "Webhook data"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/webhooks [post]
func (h *WebhookHandler) Create(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)
	var req dto.CreateWebhookReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	webhook, err := h.webhookSvc.Create(c.Context(), sessionID, req)
	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", sessionID).Msg("failed to create webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Create Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(webhook))
}

// List godoc
// @Summary     List webhooks
// @Description Returns all webhooks for the session
// @Tags        Webhooks
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/webhooks [get]
func (h *WebhookHandler) List(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)
	webhooks, err := h.webhookSvc.List(c.Context(), sessionID)
	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", sessionID).Msg("failed to list webhooks")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("List Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(webhooks))
}

// Update godoc
// @Summary     Update a webhook
// @Description Updates webhook fields (URL, secret, events, enabled, natsEnabled). Only provided fields are changed.
// @Tags        Webhooks
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       wid path string true "Webhook ID"
// @Param       body body dto.UpdateWebhookReq true "Webhook update data"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/webhooks/{wid} [put]
func (h *WebhookHandler) Update(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)
	webhookID := c.Params("wid")

	var req dto.UpdateWebhookReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	webhook, err := h.webhookSvc.Update(c.Context(), sessionID, webhookID, req)
	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", sessionID).Msg("failed to update webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Update Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(webhook))
}

// Delete godoc
// @Summary     Delete a webhook
// @Description Removes a webhook from the session
// @Tags        Webhooks
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       wid path string true "Webhook ID"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/webhooks/{wid} [delete]
func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)
	webhookID := c.Params("wid")

	if err := h.webhookSvc.Delete(c.Context(), sessionID, webhookID); err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", sessionID).Msg("failed to delete webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Delete Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(nil))
}
