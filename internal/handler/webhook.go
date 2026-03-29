package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type WebhookHandler struct {
	webhookSvc *service.WebhookService
}

func NewWebhookHandler(webhookSvc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookSvc: webhookSvc}
}

func (h *WebhookHandler) getUserID(c *fiber.Ctx) string {
	if val := c.Locals("userId"); val != nil {
		return val.(string)
	}
	return ""
}

// Create godoc
// @Summary     Create a webhook
// @Description Registers a new webhook for a session
// @Tags        Webhooks
// @Accept      json
// @Produce     json
// @Param       id   path     string                 false "Session ID"
// @Param       body body     model.CreateWebhookReq  true "Webhook data"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /webhooks [post]
// @Router      /sessions/{id}/webhooks [post]
func (h *WebhookHandler) Create(c *fiber.Ctx) error {
	id := h.getUserID(c)
	var req model.CreateWebhookReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}

	webhook, err := h.webhookSvc.Create(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Create Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(webhook, "Webhook created"))
}

// List godoc
// @Summary     List webhooks
// @Description Returns all webhooks for a session
// @Tags        Webhooks
// @Produce     json
// @Param       id  path     string false "Session ID"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /webhooks [get]
// @Router      /sessions/{id}/webhooks [get]
func (h *WebhookHandler) List(c *fiber.Ctx) error {
	id := h.getUserID(c)
	webhooks, err := h.webhookSvc.List(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("List Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(webhooks, "Webhooks retrieved"))
}

// Delete godoc
// @Summary     Delete a webhook
// @Description Removes a webhook from a session
// @Tags        Webhooks
// @Produce     json
// @Param       id  path     string false "Session ID"
// @Param       wid path     string true "Webhook ID"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /webhooks/{wid} [delete]
// @Router      /sessions/{id}/webhooks/{wid} [delete]
func (h *WebhookHandler) Delete(c *fiber.Ctx) error {
	id := h.getUserID(c)
	webhookID := c.Params("wid")

	err := h.webhookSvc.Delete(c.Context(), id, webhookID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Delete Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(nil, "Webhook deleted"))
}
