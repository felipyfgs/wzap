package handler

import (
	"wzap/internal/model"
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
// @Param       sessionName path     string                 true "Session name or ID"
// @Param       body        body     model.CreateWebhookReq true "Webhook data"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName}/webhooks [post]
func (h *WebhookHandler) Create(c *fiber.Ctx) error {
	sessionID := c.Locals("sessionId").(string)
	var req model.CreateWebhookReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}

	webhook, err := h.webhookSvc.Create(c.Context(), sessionID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Create Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(webhook, "Webhook created"))
}

// List godoc
// @Summary     List webhooks
// @Description Returns all webhooks for the session
// @Tags        Webhooks
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName}/webhooks [get]
func (h *WebhookHandler) List(c *fiber.Ctx) error {
	sessionID := c.Locals("sessionId").(string)
	webhooks, err := h.webhookSvc.List(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("List Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(webhooks, "Webhooks retrieved"))
}

// Delete godoc
// @Summary     Delete a webhook
// @Description Removes a webhook from the session
// @Tags        Webhooks
// @Produce     json
// @Param       sessionName path string true "Session name or ID"
// @Param       wid         path string true "Webhook ID"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /sessions/{sessionName}/webhooks/{wid} [delete]
func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	sessionID := c.Locals("sessionId").(string)
	webhookID := c.Params("wid")

	if err := h.webhookSvc.Delete(c.Context(), sessionID, webhookID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Delete Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(nil, "Webhook deleted"))
}
