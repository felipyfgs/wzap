package handler

import (
	"context"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/repo"
	"wzap/internal/webhook"

	"github.com/gofiber/fiber/v2"
)

type CloudWebhookHandler struct {
	sessRepo   *repo.SessionRepository
	provider   *cloudWA.Client
	dispatcher *webhook.Dispatcher
}

func NewCloudWebhookHandler(sessRepo *repo.SessionRepository, provider *cloudWA.Client, dispatcher *webhook.Dispatcher) *CloudWebhookHandler {
	return &CloudWebhookHandler{
		sessRepo:   sessRepo,
		provider:   provider,
		dispatcher: dispatcher,
	}
}

// Verify godoc
// @Summary     Verify Cloud API webhook
// @Description Verifies the webhook subscription from WhatsApp Cloud API
// @Tags        Cloud Webhooks
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       hub.mode query string true "Webhook mode (subscribe)"
// @Param       hub.verify_token query string true "Verification token"
// @Param       hub.challenge query string true "Challenge string"
// @Success     200 {string} string "Challenge string"
// @Failure     400 {object} dto.APIError
// @Failure     403 {object} dto.APIError
// @Failure     404 {object} dto.APIError
// @Router      /webhooks/cloud/{sessionId} [get]
func (h *CloudWebhookHandler) Verify(c *fiber.Ctx) error {
	id := c.Params("sessionId")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "sessionId is required"))
	}

	session, err := h.sessRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Session not found"))
	}

	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if session.Engine != "cloud_api" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "Session is not using cloud_api engine"))
	}

	result, err := cloudWA.VerifyWebhook(mode, token, challenge, session.WebhookVerifyToken)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", err.Error()))
	}

	return c.SendString(result)
}

// Handle godoc
// @Summary     Receive Cloud API webhook
// @Description Receives and processes webhooks from WhatsApp Cloud API
// @Tags        Cloud Webhooks
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       X-Hub-Signature-256 header string false "HMAC signature"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     401 {object} dto.APIError
// @Failure     404 {object} dto.APIError
// @Router      /webhooks/cloud/{sessionId} [post]
func (h *CloudWebhookHandler) Handle(c *fiber.Ctx) error {
	id := c.Params("sessionId")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "sessionId is required"))
	}

	session, err := h.sessRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Session not found"))
	}

	if session.Engine != "cloud_api" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "Session is not using cloud_api engine"))
	}

	body := c.Body()
	signature := c.Get("X-Hub-Signature-256")

	notification, err := cloudWA.ParseWebhook(body, signature, session.AppSecret)
	if err != nil {
		logger.Warn().Err(err).Str("session", id).Msg("Failed to parse cloud webhook")
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid webhook signature"))
	}

	for _, entry := range notification.Entry {
		for _, change := range entry.Changes {
			if change.Value == nil {
				continue
			}

			for _, msg := range change.Value.Messages {
				h.dispatchMessage(c.Context(), id, &msg, change.Value.Metadata)
			}

			for _, status := range change.Value.Statuses {
				h.dispatchStatus(c.Context(), id, &status)
			}
		}
	}

	return c.JSON(dto.SuccessResp(nil))
}

func (h *CloudWebhookHandler) dispatchMessage(ctx context.Context, sessionID string, msg *cloudWA.Message, metadata *cloudWA.Metadata) {
	data := map[string]any{
		"from":        msg.From,
		"id":          msg.ID,
		"type":        msg.Type,
		"timestamp":   msg.Timestamp,
		"displayName": metadata.DisplayPhoneNumber,
	}

	switch msg.Type {
	case "text":
		if msg.Text != nil {
			data["body"] = msg.Text.Body
		}
	case "image":
		if msg.Image != nil {
			data["mediaId"] = msg.Image.ID
			data["mimeType"] = msg.Image.MimeType
			data["caption"] = msg.Image.Caption
		}
	case "video":
		if msg.Video != nil {
			data["mediaId"] = msg.Video.ID
			data["mimeType"] = msg.Video.MimeType
			data["caption"] = msg.Video.Caption
		}
	case "audio":
		if msg.Audio != nil {
			data["mediaId"] = msg.Audio.ID
			data["mimeType"] = msg.Audio.MimeType
		}
	case "document":
		if msg.Document != nil {
			data["mediaId"] = msg.Document.ID
			data["mimeType"] = msg.Document.MimeType
			data["caption"] = msg.Document.Caption
			data["filename"] = msg.Document.Filename
		}
	case "sticker":
		if msg.Sticker != nil {
			data["mediaId"] = msg.Sticker.ID
			data["mimeType"] = msg.Sticker.MimeType
		}
	case "location":
		if msg.Location != nil {
			data["latitude"] = msg.Location.Latitude
			data["longitude"] = msg.Location.Longitude
			data["name"] = msg.Location.Name
			data["address"] = msg.Location.Address
		}
	case "contacts":
		data["contacts"] = msg.Contacts
	case "reaction":
		if msg.Reaction != nil {
			data["messageId"] = msg.Reaction.MessageID
			data["emoji"] = msg.Reaction.Emoji
		}
	case "interactive":
		if msg.Interactive != nil {
			data["interactive"] = msg.Interactive
		}
	case "button":
		if msg.Button != nil {
			data["buttonPayload"] = msg.Button
		}
	}

	bytes, err := model.BuildEventEnvelope(sessionID, "", model.EventMessage, data)
	if err != nil {
		logger.Error().Err(err).Str("session", sessionID).Msg("Failed to build cloud message envelope")
		return
	}

	if h.dispatcher != nil {
		h.dispatcher.Dispatch(sessionID, model.EventMessage, bytes)
	}
}

func (h *CloudWebhookHandler) dispatchStatus(ctx context.Context, sessionID string, status *cloudWA.Status) {
	data := map[string]any{
		"messageId":   status.ID,
		"recipientId": status.RecipientID,
		"status":      status.Status,
		"timestamp":   status.Timestamp,
	}

	if status.Conversation != nil {
		data["conversationId"] = status.Conversation.ID
	}
	if status.Pricing != nil {
		data["pricingCategory"] = status.Pricing.Category
		data["billable"] = status.Pricing.Billable
	}

	bytes, err := model.BuildEventEnvelope(sessionID, "", model.EventReceipt, data)
	if err != nil {
		logger.Error().Err(err).Str("session", sessionID).Msg("Failed to build cloud status envelope")
		return
	}

	if h.dispatcher != nil {
		h.dispatcher.Dispatch(sessionID, model.EventReceipt, bytes)
	}
}
