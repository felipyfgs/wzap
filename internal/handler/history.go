package handler

import (
	"errors"
	"strconv"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

type HistoryHandler struct {
	messageRepo repo.MessageRepo
}

func NewHistoryHandler(messageRepo repo.MessageRepo) *HistoryHandler {
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
	sessionID, err := getSessionID(c)
	if err != nil {
		return err
	}
	chatJID := c.Query("chat")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var msgs any

	if chatJID != "" {
		msgs, err = h.messageRepo.FindByChat(c.Context(), sessionID, chatJID, limit, offset)
	} else {
		msgs, err = h.messageRepo.FindBySession(c.Context(), sessionID, limit, offset)
	}

	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", sessionID).Msg("failed to list messages")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("History Error", "internal server error"))
	}

	return c.JSON(dto.SuccessResp(msgs))
}

// ListMedia godoc
// @Summary     List media
// @Description Returns paginated media messages with server-side filtering
// @Tags        Media
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       type query string false "Media type filter (image, video, document, audio, sticker)"
// @Param       chat query string false "Chat JID filter"
// @Param       search query string false "Search in body or chat JID"
// @Param       since query string false "ISO date string (start of range)"
// @Param       until query string false "ISO date string (end of range)"
// @Param       cursor query string false "Cursor for pagination (RFC3339 timestamp of last item)"
// @Param       sort query string false "Sort order: desc (default) or asc"
// @Param       limit query int false "Max results (default 100, max 200)"
// @Success     200 {object} dto.APIResponse
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/media [get]
func (h *HistoryHandler) ListMedia(c *fiber.Ctx) error {
	sessionID, err := getSessionID(c)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	filter := repo.MediaFilter{
		MsgType: c.Query("type"),
		Chat:    c.Query("chat"),
		Search:  c.Query("search"),
		Since:   c.Query("since"),
		Until:   c.Query("until"),
		Cursor:  c.Query("cursor"),
		Sort:    c.Query("sort"),
		Limit:   limit,
	}

	msgs, total, err := h.messageRepo.FindMedia(c.Context(), sessionID, filter)
	if err != nil {
		if errors.Is(err, repo.ErrInvalidCursor) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Cursor", err.Error()))
		}
		logger.Warn().Err(err).Str("component", "handler").Str("session", sessionID).Msg("failed to list media")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Media Error", "internal server error"))
	}

	var nextCursor *string
	if len(msgs) >= limit {
		last := msgs[len(msgs)-1]
		cursor := last.Timestamp.Format(time.RFC3339Nano) + "|" + last.ID
		nextCursor = &cursor
	}

	return c.JSON(dto.SuccessResp(fiber.Map{
		"items":      msgs,
		"total":      total,
		"nextCursor": nextCursor,
	}))
}
