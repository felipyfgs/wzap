package chatwoot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	"wzap/internal/logger"
	mw "wzap/internal/middleware"
)

type Handler struct {
	service      *Service
	chatwootRepo Repo
}

func NewHandler(service *Service, chatwootRepo Repo) *Handler {
	return &Handler{
		service:      service,
		chatwootRepo: chatwootRepo,
	}
}

func validateReq(c *fiber.Ctx, req interface{}) error {
	if err := c.BodyParser(req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
		return fiber.ErrBadRequest
	}

	if err := mw.Validate.Struct(req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var msgs []string
			for _, e := range validationErrors {
				msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s'", e.Field(), e.Tag()))
			}
			_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", strings.Join(msgs, "; ")))
		} else {
			_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", err.Error()))
		}
		return fiber.ErrBadRequest
	}
	return nil
}

func configToResp(cfg *ChatwootConfig, webhookURL string) dto.ChatwootConfigResp {
	ignoreGroups := false
	for _, jid := range cfg.IgnoreJIDs {
		if jid == "@g.us" {
			ignoreGroups = true
			break
		}
	}

	return dto.ChatwootConfigResp{
		SessionID:           cfg.SessionID,
		URL:                 cfg.URL,
		AccountID:           cfg.AccountID,
		InboxID:             cfg.InboxID,
		InboxName:           cfg.InboxName,
		SignMsg:             cfg.SignMsg,
		SignDelimiter:       cfg.SignDelimiter,
		ReopenConversation:  cfg.ReopenConversation,
		MergeBRContacts:     cfg.MergeBRContacts,
		IgnoreGroups:        ignoreGroups,
		IgnoreJIDs:          cfg.IgnoreJIDs,
		ConversationPending: cfg.ConversationPending,
		Enabled:             cfg.Enabled,
		WebhookURL:          webhookURL,
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
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 401 {object} dto.APIError
// @Failure 500 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/chatwoot [put]
func (h *Handler) Configure(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	var req dto.ChatwootConfigReq
	if err := validateReq(c, &req); err != nil {
		return nil
	}

	cfg := &ChatwootConfig{
		SessionID:           sessionID,
		URL:                 req.URL,
		AccountID:           req.AccountID,
		Token:               req.Token,
		InboxID:             req.InboxID,
		InboxName:           req.InboxName,
		SignMsg:             req.SignMsg != nil && *req.SignMsg,
		SignDelimiter:       req.SignDelimiter,
		ReopenConversation:  req.ReopenConversation == nil || *req.ReopenConversation,
		MergeBRContacts:     req.MergeBRContacts == nil || *req.MergeBRContacts,
		ConversationPending: req.ConversationPending != nil && *req.ConversationPending,
	}

	ignoreJIDs := make([]string, 0, len(req.IgnoreJIDs))
	ignoreJIDs = append(ignoreJIDs, req.IgnoreJIDs...)
	if req.IgnoreGroups != nil && *req.IgnoreGroups {
		hasGroupMarker := false
		for _, jid := range ignoreJIDs {
			if jid == "@g.us" {
				hasGroupMarker = true
				break
			}
		}
		if !hasGroupMarker {
			ignoreJIDs = append(ignoreJIDs, "@g.us")
		}
	}
	cfg.IgnoreJIDs = ignoreJIDs

	autoCreate := req.AutoCreateInbox != nil && *req.AutoCreateInbox
	if autoCreate && cfg.InboxID == 0 {
		cfg.InboxID = 0
	}

	if err := h.service.Configure(c.Context(), cfg); err != nil {
		logger.Warn().Err(err).Str("session", sessionID).Msg("Failed to configure Chatwoot")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Configuration Error", "Failed to configure Chatwoot integration"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(configToResp(cfg, h.service.webhookURL(sessionID))))
}

// GetConfig
// @Summary Get Chatwoot configuration for a session
// @Description Returns the Chatwoot configuration for the specified session, or 404 if not configured
// @Tags Chatwoot
// @Produce json
// @Param sessionId path string true "Session ID or name"
// @Success 200 {object} dto.APIResponse
// @Failure 401 {object} dto.APIError
// @Failure 404 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/chatwoot [get]
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	cfg, err := h.chatwootRepo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Chatwoot integration not configured for this session"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(configToResp(cfg, h.service.webhookURL(cfg.SessionID))))
}

// DeleteConfig
// @Summary Delete Chatwoot configuration for a session
// @Description Removes the Chatwoot integration from the specified session
// @Tags Chatwoot
// @Param sessionId path string true "Session ID or name"
// @Success 204 "No Content"
// @Failure 401 {object} dto.APIError
// @Failure 500 {object} dto.APIError
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
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 401 {object} dto.APIError
// @Failure 500 {object} dto.APIError
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

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(nil))
}

func mustGetSessionID(c *fiber.Ctx) string {
	if id, ok := c.Locals("sessionID").(string); ok {
		return id
	}
	return c.Params("sessionId")
}
