package chatwoot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	"wzap/internal/logger"
)

type Handler struct {
	service     *Service
	chatwootRepo *Repository
}

func NewHandler(service *Service, chatwootRepo *Repository) *Handler {
	return &Handler{
		service:     service,
		chatwootRepo: chatwootRepo,
	}
}

// Configure
// @Summary Configure Chatwoot integration for a session
// @Description Upserts Chatwoot configuration and optionally auto-creates an inbox
// @Tags Chatwoot
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID or name"
// @Param body body dto.ChatwootConfigReq true "Chatwoot configuration"
// @Success 200 {object} dto.ChatwootConfigResp
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /sessions/{sessionId}/integrations/chatwoot [put]
func (h *Handler) Configure(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	var req dto.ChatwootConfigReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "Failed to parse request body"))
	}

	cfg := &ChatwootConfig{
		SessionID:        sessionID,
		URL:              req.URL,
		AccountID:        req.AccountID,
		Token:            req.Token,
		InboxName:        req.InboxName,
		SignMsg:          req.SignMsg != nil && *req.SignMsg,
		SignDelimiter:    req.SignDelimiter,
		ReopenConversation: req.ReopenConversation == nil || *req.ReopenConversation,
		MergeBRContacts:  req.MergeBRContacts == nil || *req.MergeBRContacts,
		IgnoreGroups:     req.IgnoreGroups != nil && *req.IgnoreGroups,
	}

	if err := h.service.Configure(c.Context(), cfg); err != nil {
		logger.Warn().Err(err).Str("session", sessionID).Msg("Failed to configure Chatwoot")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Configuration Error", "Failed to configure Chatwoot integration"))
	}

	webhookURL := fmt.Sprintf("%s/chatwoot/webhook/%s", cfg.URL, sessionID)

	resp := dto.ChatwootConfigResp{
		SessionID:        cfg.SessionID,
		URL:              cfg.URL,
		AccountID:        cfg.AccountID,
		InboxID:          cfg.InboxID,
		InboxName:        cfg.InboxName,
		SignMsg:          cfg.SignMsg,
		SignDelimiter:    cfg.SignDelimiter,
		ReopenConversation: cfg.ReopenConversation,
		MergeBRContacts:  cfg.MergeBRContacts,
		IgnoreGroups:     cfg.IgnoreGroups,
		Enabled:          cfg.Enabled,
		WebhookURL:       webhookURL,
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// GetConfig
// @Summary Get Chatwoot configuration for a session
// @Description Returns the Chatwoot configuration for the specified session, or 404 if not configured
// @Tags Chatwoot
// @Produce json
// @Param sessionId path string true "Session ID or name"
// @Success 200 {object} dto.ChatwootConfigResp
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /sessions/{sessionId}/integrations/chatwoot [get]
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	cfg, err := h.chatwootRepo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Chatwoot integration not configured for this session"))
	}

	webhookURL := fmt.Sprintf("%s/chatwoot/webhook/%s", cfg.URL, sessionID)

	resp := dto.ChatwootConfigResp{
		SessionID:        cfg.SessionID,
		URL:              cfg.URL,
		AccountID:        cfg.AccountID,
		InboxID:          cfg.InboxID,
		InboxName:        cfg.InboxName,
		SignMsg:          cfg.SignMsg,
		SignDelimiter:    cfg.SignDelimiter,
		ReopenConversation: cfg.ReopenConversation,
		MergeBRContacts:  cfg.MergeBRContacts,
		IgnoreGroups:     cfg.IgnoreGroups,
		Enabled:          cfg.Enabled,
		WebhookURL:       webhookURL,
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// DeleteConfig
// @Summary Delete Chatwoot configuration for a session
// @Description Removes the Chatwoot integration from the specified session
// @Tags Chatwoot
// @Param sessionId path string true "Session ID or name"
// @Success 204 "No Content"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /sessions/{sessionId}/integrations/chatwoot [delete]
func (h *Handler) DeleteConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	if err := h.chatwootRepo.Delete(c.Context(), sessionID); err != nil {
		logger.Warn().Err(err).Str("session", sessionID).Msg("Failed to delete Chatwoot config")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Delete Error", "Failed to delete Chatwoot configuration"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// IncomingWebhook
// @Summary Receive webhook events from Chatwoot
// @Description Handles incoming Chatwoot webhook events (agent replies, message updates, conversation status changes)
// @Tags Chatwoot
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID"
// @Param X-Chatwoot-Hmac-Sha256 header string false "HMAC-SHA256 signature"
// @Success 200 {object} map[string]string
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /chatwoot/webhook/{sessionId} [post]
func (h *Handler) IncomingWebhook(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	cfg, err := h.chatwootRepo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Chatwoot not configured for this session"))
	}

	hmacHeader := c.Get("X-Chatwoot-Hmac-Sha256")
	if hmacHeader != "" {
		body := c.Body()
		mac := hmac.New(sha256.New, []byte(cfg.Token))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(hmacHeader), []byte(expected)) {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid HMAC signature"))
		}
	}

	var body dto.ChatwootWebhookPayload
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "Failed to parse webhook payload"))
	}

	if err := h.service.HandleIncomingWebhook(c.Context(), sessionID, body); err != nil {
		logger.Warn().Err(err).Str("session", sessionID).Msg("Failed to handle Chatwoot webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Webhook Error", "Failed to process webhook"))
	}

	return c.Status(fiber.StatusOK).JSON(map[string]string{"message": "ok"})
}

func mustGetSessionID(c *fiber.Ctx) string {
	if id, ok := c.Locals("sessionID").(string); ok {
		return id
	}
	return c.Params("sessionId")
}
