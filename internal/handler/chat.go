package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type ChatHandler struct {
	chatSvc *service.ChatService
}

func NewChatHandler(chatSvc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

func (h *ChatHandler) getSessionID(c *fiber.Ctx) (string, error) {
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}

// Archive godoc
// @Summary     Archive a chat
// @Description Archives a chat identified by JID, moving it out of the main chat list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     model.ChatActionReq true "Chat JID payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /chat/archive [post]
func (h *ChatHandler) Archive(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.ChatActionReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.chatSvc.Archive(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Chat archived"))
}

// Mute godoc
// @Summary     Mute a chat
// @Description Mutes notifications for a chat identified by JID
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     model.ChatActionReq true "Chat JID payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /chat/mute [post]
func (h *ChatHandler) Mute(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.ChatActionReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.chatSvc.Mute(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Chat muted"))
}

// Pin godoc
// @Summary     Pin a chat
// @Description Pins a chat to the top of the chat list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     model.ChatActionReq true "Chat JID payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /chat/pin [post]
func (h *ChatHandler) Pin(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.ChatActionReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.chatSvc.Pin(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Chat pinned"))
}

// Unpin godoc
// @Summary     Unpin a chat
// @Description Removes a chat from the pinned position at the top of the chat list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     model.ChatActionReq true "Chat JID payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /chat/unpin [post]
func (h *ChatHandler) Unpin(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.ChatActionReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.chatSvc.Unpin(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Chat unpinned"))
}
