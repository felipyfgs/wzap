package handler

import (
	"strconv"

	"wzap/internal/dto"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

type HistoryHandler struct {
	messageRepo *repo.MessageRepository
}

func NewHistoryHandler(messageRepo *repo.MessageRepository) *HistoryHandler {
	return &HistoryHandler{messageRepo: messageRepo}
}

// ListMessages godoc
// @Summary     List messages
// @Description Returns paginated message history for the session
// @Tags        Messages
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       chat query string false "Chat JID filter"
// @Param       limit query int false "Max results (default 50, max 100)"
// @Param       offset query int false "Offset for pagination"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages [get]
func (h *HistoryHandler) ListMessages(c *fiber.Ctx) error {
	sessionID := mustGetSessionID(c)
	chatJID := c.Query("chat")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var err error
	var msgs interface{}

	if chatJID != "" {
		msgs, err = h.messageRepo.FindByChat(c.Context(), sessionID, chatJID, limit, offset)
	} else {
		msgs, err = h.messageRepo.FindBySession(c.Context(), sessionID, limit, offset)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("History Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(msgs))
}
