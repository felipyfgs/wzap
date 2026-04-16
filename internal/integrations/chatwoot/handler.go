package chatwoot

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	"wzap/internal/logger"
	mw "wzap/internal/middleware"
)

type Handler struct {
	service *Service
	repo    Repo
}

func NewHandler(service *Service, repo Repo) *Handler {
	return &Handler{
		service: service,
		repo:    repo,
	}
}

func validateReq(c *fiber.Ctx, req any) error {
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

func configToResp(cfg *Config, webhookURL string) dto.ChatwootConfigResp {
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
		InboxType:           cfg.InboxType,
		SignMsg:             cfg.SignMsg,
		SignDelimiter:       cfg.SignDelimiter,
		ReopenConversation:  cfg.ReopenConversation,
		MergeBRContacts:     cfg.MergeBRContacts,
		IgnoreGroups:        ignoreGroups,
		IgnoreJIDs:          cfg.IgnoreJIDs,
		ConversationPending: cfg.ConversationPending,
		Enabled:             cfg.Enabled,
		WebhookURL:          webhookURL,
		ImportOnConnect:     cfg.ImportOnConnect,
		ImportPeriod:        cfg.ImportPeriod,
		TimeoutTextSeconds:  cfg.TimeoutTextSeconds,
		TimeoutMediaSeconds: cfg.TimeoutMediaSeconds,
		TimeoutLargeSeconds: cfg.TimeoutLargeSeconds,
		MessageRead:         cfg.MessageRead,
		DatabaseURI:         maskDatabaseURI(cfg.DatabaseURI),
		RedisURL:            maskRedisURL(cfg.RedisURL),
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
// @Security Authorization
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

	timeoutText := 10
	if req.TimeoutTextSeconds != nil {
		timeoutText = *req.TimeoutTextSeconds
	}
	timeoutMedia := 60
	if req.TimeoutMediaSeconds != nil {
		timeoutMedia = *req.TimeoutMediaSeconds
	}
	timeoutLarge := 300
	if req.TimeoutLargeSeconds != nil {
		timeoutLarge = *req.TimeoutLargeSeconds
	}
	importPeriod := "7d"
	if req.ImportPeriod != "" {
		importPeriod = req.ImportPeriod
	}

	cfg := &Config{
		SessionID:           sessionID,
		URL:                 req.URL,
		AccountID:           req.AccountID,
		Token:               req.Token,
		WebhookToken:        req.WebhookToken,
		InboxID:             req.InboxID,
		InboxName:           req.InboxName,
		InboxType:           "api",
		SignMsg:             req.SignMsg != nil && *req.SignMsg,
		SignDelimiter:       req.SignDelimiter,
		ReopenConversation:  req.ReopenConversation == nil || *req.ReopenConversation,
		MergeBRContacts:     req.MergeBRContacts == nil || *req.MergeBRContacts,
		ConversationPending: req.ConversationPending != nil && *req.ConversationPending,
		ImportOnConnect:     req.ImportOnConnect != nil && *req.ImportOnConnect,
		ImportPeriod:        importPeriod,
		TimeoutTextSeconds:  timeoutText,
		TimeoutMediaSeconds: timeoutMedia,
		TimeoutLargeSeconds: timeoutLarge,
		MessageRead:         req.MessageRead != nil && *req.MessageRead,
		DatabaseURI:         req.DatabaseURI,
		RedisURL:            req.RedisURL,
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

	if req.InboxType != nil {
		cfg.InboxType = *req.InboxType
	}

	if err := h.service.Configure(c.Context(), cfg); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("Failed to configure Chatwoot")
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
// @Security Authorization
// @Failure 401 {object} dto.APIError
// @Failure 404 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/chatwoot [get]
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	cfg, err := h.repo.FindBySessionID(c.Context(), sessionID)
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
// @Security Authorization
// @Failure 401 {object} dto.APIError
// @Failure 500 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/chatwoot [delete]
func (h *Handler) DeleteConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	if err := h.repo.Delete(c.Context(), sessionID); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("Failed to delete Chatwoot config")
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

	cfg, err := h.repo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Chatwoot not configured for this session"))
	}

	hmacHeader := c.Get("X-Chatwoot-Hmac-Sha256")
	if cfg.WebhookToken != "" {
		if hmacHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing HMAC signature"))
		}
		body := c.Body()
		mac := hmac.New(sha256.New, []byte(cfg.WebhookToken))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(hmacHeader), []byte(expected)) {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid HMAC signature"))
		}
	}

	var body dto.ChatwootWebhookPayload
	rawBody := c.Body()
	if err := json.Unmarshal(rawBody, &body); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("contentType", c.Get("Content-Type")).Msg("failed to parse webhook payload")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "Failed to parse webhook payload"))
	}

	if h.service.js != nil {
		go func() {
			pCtx, pCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pCancel()
			if err := publishOutbound(pCtx, h.service.js, sessionID, rawBody); err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("failed to publish outbound webhook, falling back to sync")
				sCtx, sCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer sCancel()
				if err := h.service.HandleIncomingWebhook(sCtx, sessionID, body); err != nil {
					logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("Failed to handle Chatwoot webhook")
				}
				return
			}
		}()
	} else {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.service.HandleIncomingWebhook(ctx, sessionID, body); err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("Failed to handle Chatwoot webhook")
			}
		}()
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(nil))
}

// ImportHistory
// @Summary Trigger manual history import from WhatsApp to Chatwoot
// @Tags Chatwoot
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID"
// @Param body body dto.ImportHistoryReq true "Import configuration"
// @Success 202 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Security Authorization
// @Failure 401 {object} dto.APIError
// @Failure 404 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/chatwoot/import [post]
func (h *Handler) ImportHistory(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	_, err := h.repo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Chatwoot integration not configured for this session"))
	}

	var req dto.ImportHistoryReq
	if err := validateReq(c, &req); err != nil {
		return nil
	}

	go h.service.ImportHistoryAsync(context.Background(), sessionID, req.Period, req.CustomDays)

	return c.Status(fiber.StatusAccepted).JSON(dto.SuccessResp(dto.ImportHistoryResp{
		SessionID: sessionID,
		Period:    req.Period,
		Status:    "importing",
	}))
}

func mustGetSessionID(c *fiber.Ctx) string {
	if id, ok := c.Locals("sessionID").(string); ok {
		return id
	}
	return c.Params("sessionId")
}
