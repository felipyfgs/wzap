package handler

import (
	"wzap/internal/dto"
	"wzap/internal/service"

	"github.com/gofiber/fiber/v2"
)

type MediaHandler struct {
	mediaSvc *service.MediaService
}

func NewMediaHandler(mediaSvc *service.MediaService) *MediaHandler {
	return &MediaHandler{mediaSvc: mediaSvc}
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
	sessionID := mustGetSessionID(c)
	messageID := c.Params("messageId")

	url, err := h.mediaSvc.GetPresignedURL(c.Context(), sessionID, messageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(dto.SuccessResp(fiber.Map{"url": url}))
}
