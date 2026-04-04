package handler

import (
	"context"
	"encoding/json"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/webhook"

	"github.com/gofiber/fiber/v2"
)

type CloudWebhookHandler struct {
	sessRepo    *repo.SessionRepository
	provider    *cloudWA.Client
	dispatcher  *webhook.Dispatcher
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
	payload := map[string]any{
		"from":        msg.From,
		"id":          msg.ID,
		"type":        msg.Type,
		"timestamp":   msg.Timestamp,
		"displayName": metadata.DisplayPhoneNumber,
	}

	switch msg.Type {
	case "text":
		if msg.Text != nil {
			payload["body"] = msg.Text.Body
		}
	case "image":
		if msg.Image != nil {
			payload["mediaId"] = msg.Image.ID
			payload["mimeType"] = msg.Image.MimeType
			payload["caption"] = msg.Image.Caption
		}
	case "video":
		if msg.Video != nil {
			payload["mediaId"] = msg.Video.ID
			payload["mimeType"] = msg.Video.MimeType
			payload["caption"] = msg.Video.Caption
		}
	case "audio":
		if msg.Audio != nil {
			payload["mediaId"] = msg.Audio.ID
			payload["mimeType"] = msg.Audio.MimeType
		}
	case "document":
		if msg.Document != nil {
			payload["mediaId"] = msg.Document.ID
			payload["mimeType"] = msg.Document.MimeType
			payload["caption"] = msg.Document.Caption
			payload["filename"] = msg.Document.Filename
		}
	case "sticker":
		if msg.Sticker != nil {
			payload["mediaId"] = msg.Sticker.ID
			payload["mimeType"] = msg.Sticker.MimeType
		}
	case "location":
		if msg.Location != nil {
			payload["latitude"] = msg.Location.Latitude
			payload["longitude"] = msg.Location.Longitude
			payload["name"] = msg.Location.Name
			payload["address"] = msg.Location.Address
		}
	case "contacts":
		payload["contacts"] = msg.Contacts
	case "reaction":
		if msg.Reaction != nil {
			payload["messageId"] = msg.Reaction.MessageID
			payload["emoji"] = msg.Reaction.Emoji
		}
	case "interactive":
		if msg.Interactive != nil {
			payload["interactive"] = msg.Interactive
		}
	case "button":
		if msg.Button != nil {
			payload["buttonPayload"] = msg.Button
		}
	}

	data, _ := json.Marshal(payload)

	if h.dispatcher != nil {
		h.dispatcher.Dispatch(sessionID, model.EventMessage, data)
	}
}

func (h *CloudWebhookHandler) dispatchStatus(ctx context.Context, sessionID string, status *cloudWA.Status) {
	payload := map[string]any{
		"messageId":   status.ID,
		"recipientId": status.RecipientID,
		"status":      status.Status,
		"timestamp":   status.Timestamp,
	}

	if status.Conversation != nil {
		payload["conversationId"] = status.Conversation.ID
	}
	if status.Pricing != nil {
		payload["pricingCategory"] = status.Pricing.Category
		payload["billable"] = status.Pricing.Billable
	}

	data, _ := json.Marshal(payload)

	if h.dispatcher != nil {
		h.dispatcher.Dispatch(sessionID, model.EventReceipt, data)
	}
}
