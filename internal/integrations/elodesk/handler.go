package elodesk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
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

func configToResp(cfg *Config, webhookURL string) dto.ElodeskConfigResp {
	ignoreGroups := cfg.IgnoreGroups || slices.Contains(cfg.IgnoreJIDs, "@g.us")

	return dto.ElodeskConfigResp{
		SessionID:          cfg.SessionID,
		URL:                cfg.URL,
		InboxIdentifier:    cfg.InboxIdentifier,
		HasAPIToken:        cfg.APIToken != "",
		HasHMACToken:       cfg.HMACToken != "",
		HasUserAccessToken: cfg.UserAccessToken != "",
		AccountID:          cfg.AccountID,
		SignMsg:            cfg.SignMsg,
		SignDelimiter:      cfg.SignDelimiter,
		ReopenConv:         cfg.ReopenConv,
		MergeBRContacts:    cfg.MergeBRContacts,
		IgnoreGroups:       ignoreGroups,
		IgnoreJIDs:         cfg.IgnoreJIDs,
		PendingConv:        cfg.PendingConv,
		Enabled:            cfg.Enabled,
		WebhookURL:         webhookURL,
		ImportOnConnect:    cfg.ImportOnConnect,
		ImportPeriod:       cfg.ImportPeriod,
		TextTimeout:        cfg.TextTimeout,
		MediaTimeout:       cfg.MediaTimeout,
		LargeTimeout:       cfg.LargeTimeout,
		MessageRead:        cfg.MessageRead,
	}
}

// Configure
// @Summary Configure Elodesk integration for a session
// @Description Upserts Elodesk configuration for the session
// @Tags Elodesk
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID or name"
// @Param body body dto.ElodeskConfigReq true "Elodesk configuration"
// @Success 200 {object} dto.APIResponse
// @Security Authorization
// @Failure 400 {object} dto.APIError
// @Failure 401 {object} dto.APIError
// @Failure 500 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/elodesk [put]
func (h *Handler) Configure(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	var req dto.ElodeskConfigReq
	if err := validateReq(c, &req); err != nil {
		return nil
	}

	timeoutText := 10
	if req.TextTimeout != nil {
		timeoutText = *req.TextTimeout
	}
	timeoutMedia := 60
	if req.MediaTimeout != nil {
		timeoutMedia = *req.MediaTimeout
	}
	timeoutLarge := 300
	if req.LargeTimeout != nil {
		timeoutLarge = *req.LargeTimeout
	}
	importPeriod := "7d"
	if req.ImportPeriod != "" {
		importPeriod = req.ImportPeriod
	}

	accountID := 1
	if req.AccountID != nil {
		accountID = *req.AccountID
	}

	cfg := &Config{
		SessionID:       sessionID,
		URL:             req.URL,
		InboxIdentifier: req.InboxIdentifier,
		APIToken:        req.APIToken,
		HMACToken:       req.HMACToken,
		WebhookSecret:   req.WebhookSecret,
		UserAccessToken: req.UserAccessToken,
		AccountID:       accountID,
		InboxName:       req.InboxName,
		SignMsg:         req.SignMsg != nil && *req.SignMsg,
		SignDelimiter:   req.SignDelimiter,
		ReopenConv:      req.ReopenConv == nil || *req.ReopenConv,
		MergeBRContacts: req.MergeBRContacts == nil || *req.MergeBRContacts,
		PendingConv:     req.PendingConv != nil && *req.PendingConv,
		ImportOnConnect: req.ImportOnConnect != nil && *req.ImportOnConnect,
		ImportPeriod:    importPeriod,
		TextTimeout:     timeoutText,
		MediaTimeout:    timeoutMedia,
		LargeTimeout:    timeoutLarge,
		MessageRead:     req.MessageRead != nil && *req.MessageRead,
		Enabled:         req.Enabled == nil || *req.Enabled,
	}

	// Preserve secrets/identifiers já persistidos quando o request vier sem
	// eles. Sem esse merge um save parcial do frontend (que não reenvia
	// api_token/hmac_token) zeraria os tokens da inbox auto-provisionada.
	if existing, err := h.repo.FindBySessionID(c.Context(), sessionID); err == nil {
		if cfg.APIToken == "" {
			cfg.APIToken = existing.APIToken
		}
		if cfg.HMACToken == "" {
			cfg.HMACToken = existing.HMACToken
		}
		if cfg.UserAccessToken == "" {
			cfg.UserAccessToken = existing.UserAccessToken
		}
		if cfg.InboxIdentifier == "" {
			cfg.InboxIdentifier = existing.InboxIdentifier
		}
		if cfg.WebhookSecret == "" {
			cfg.WebhookSecret = existing.WebhookSecret
		}
		cfg.ChannelID = existing.ChannelID
	}

	ignoreJIDs := make([]string, 0, len(req.IgnoreJIDs))
	ignoreJIDs = append(ignoreJIDs, req.IgnoreJIDs...)
	if req.IgnoreGroups != nil && *req.IgnoreGroups {
		if !slices.Contains(ignoreJIDs, "@g.us") {
			ignoreJIDs = append(ignoreJIDs, "@g.us")
		}
		cfg.IgnoreGroups = true
	}
	cfg.IgnoreJIDs = ignoreJIDs

	if err := h.service.Configure(c.Context(), cfg); err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("Failed to configure Elodesk")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Configuration Error", "Failed to configure Elodesk integration"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(configToResp(cfg, h.service.webhookURL(sessionID))))
}

// GetConfig
// @Summary Get Elodesk configuration for a session
// @Tags Elodesk
// @Produce json
// @Param sessionId path string true "Session ID or name"
// @Success 200 {object} dto.APIResponse
// @Security Authorization
// @Failure 401 {object} dto.APIError
// @Failure 404 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/elodesk [get]
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	cfg, err := h.repo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Elodesk integration not configured for this session"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(configToResp(cfg, h.service.webhookURL(cfg.SessionID))))
}

// DeleteConfig
// @Summary Delete Elodesk configuration for a session
// @Tags Elodesk
// @Param sessionId path string true "Session ID or name"
// @Success 204 "No Content"
// @Security Authorization
// @Failure 401 {object} dto.APIError
// @Failure 500 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/elodesk [delete]
func (h *Handler) DeleteConfig(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	if err := h.repo.Delete(c.Context(), sessionID); err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("Failed to delete Elodesk config")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Delete Error", "Failed to delete Elodesk configuration"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// IncomingWebhook
// @Summary Receive webhook events from Elodesk
// @Description Handles incoming Elodesk webhook events (agent replies, message updates, conversation status changes)
// @Tags Elodesk
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID"
// @Param X-Chatwoot-Hmac-Sha256 header string false "HMAC-SHA256 signature (legacy alias)"
// @Param X-Elodesk-Signature header string false "HMAC-SHA256 signature (preferred)"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Failure 401 {object} dto.APIError
// @Router /elodesk/webhook/{sessionId} [post]
func (h *Handler) IncomingWebhook(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")

	cfg, err := h.repo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Elodesk not configured for this session"))
	}

	// HMAC: preferir X-Chatwoot-Hmac-Sha256 (body direto, compatível),
	// fallback X-Elodesk-Signature (com prefixo sha256= e assinado sobre ts.body).
	hmacHeader := c.Get("X-Chatwoot-Hmac-Sha256")
	if hmacHeader == "" {
		hmacHeader = strings.TrimPrefix(c.Get("X-Elodesk-Signature"), "sha256=")
	}
	if cfg.HMACToken != "" {
		if hmacHeader == "" {
			logger.Warn().Str("component", "elodesk").Str("session", sessionID).Str("reason", "missing_signature").Msg("webhook rejected")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing HMAC signature"))
		}
		body := c.Body()
		mac := hmac.New(sha256.New, []byte(cfg.HMACToken))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(hmacHeader), []byte(expected)) {
			logger.Warn().Str("component", "elodesk").Str("session", sessionID).Str("reason", "invalid_signature").Msg("webhook rejected")
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid HMAC signature"))
		}
	}

	var body dto.ElodeskWebhookPayload
	rawBody := c.Body()
	if err := json.Unmarshal(rawBody, &body); err != nil {
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("failed to parse webhook payload")
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "Failed to parse webhook payload"))
	}

	if h.service.js != nil {
		go func() {
			pCtx, pCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pCancel()
			if err := publishOutbound(pCtx, h.service.js, sessionID, rawBody); err != nil {
				logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("failed to publish outbound webhook, falling back to sync")
				sCtx, sCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer sCancel()
				if err := h.service.HandleIncomingWebhook(sCtx, sessionID, body); err != nil {
					logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("Failed to handle Elodesk webhook")
				}
			}
		}()
	} else {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.service.HandleIncomingWebhook(ctx, sessionID, body); err != nil {
				logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("Failed to handle Elodesk webhook")
			}
		}()
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResp(nil))
}

// ImportHistory
// @Summary Trigger manual history import from WhatsApp to Elodesk
// @Tags Elodesk
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID"
// @Param body body dto.ElodeskImportReq true "Import configuration"
// @Success 202 {object} dto.APIResponse
// @Failure 400 {object} dto.APIError
// @Security Authorization
// @Failure 401 {object} dto.APIError
// @Failure 404 {object} dto.APIError
// @Router /sessions/{sessionId}/integrations/elodesk/import [post]
func (h *Handler) ImportHistory(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)

	_, err := h.repo.FindBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Elodesk integration not configured for this session"))
	}

	var req dto.ElodeskImportReq
	if err := validateReq(c, &req); err != nil {
		return nil
	}

	go h.service.ImportHistoryAsync(context.Background(), sessionID, req.Period, req.CustomDays)

	return c.Status(fiber.StatusAccepted).JSON(dto.SuccessResp(dto.ElodeskImportResp{
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
