package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	"wzap/internal/repo"
	"wzap/internal/service"
	"wzap/internal/storage"
)

type MediaHandler struct {
	mediaSvc    *service.MediaService
	messageRepo repo.MessageRepo
}

func NewMediaHandler(mediaSvc *service.MediaService, messageRepo repo.MessageRepo) *MediaHandler {
	return &MediaHandler{mediaSvc: mediaSvc, messageRepo: messageRepo}
}

// GetMedia godoc
// @Summary     Get media URL
// @Description Returns a presigned S3 URL for a previously stored media message
// @Tags        Media
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       messageId path string true "Message ID"
// @Success     200 {object} dto.APIResponse
// @Failure     404 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/media/{messageId} [get]
func (h *MediaHandler) GetMedia(c *fiber.Ctx) error {
	sessionID, err := getSessionID(c)
	if err != nil {
		return err
	}
	messageID := c.Params("messageId")

	msg, err := h.messageRepo.FindByID(c.Context(), sessionID, messageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "message not found"))
	}

	key := msg.MediaURL
	if strings.HasPrefix(key, "http") || key == "" {
		key = storage.MediaObjectKey(storage.MediaKeyParams{
			SessionID: msg.SessionID,
			ChatJID:   msg.ChatJID,
			SenderJID: msg.SenderJID,
			FromMe:    msg.FromMe,
			MessageID: msg.ID,
			MimeType:  msg.MediaType,
			Timestamp: msg.Timestamp,
		})
	}

	url, err := h.mediaSvc.GetPresignedURL(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"url": url}))
}
