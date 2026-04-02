package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/dto"
	"wzap/internal/service"
)

type ChatHandler struct {
	chatSvc *service.ChatService
}

func NewChatHandler(chatSvc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

// Archive godoc
// @Summary     Archive a chat
// @Description Archives a chat identified by JID, moving it out of the main chat list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/archive [post]
func (h *ChatHandler) Archive(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.Archive(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Mute godoc
// @Summary     Mute a chat
// @Description Mutes notifications for a chat identified by JID
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/mute [post]
func (h *ChatHandler) Mute(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.Mute(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Pin godoc
// @Summary     Pin a chat
// @Description Pins a chat to the top of the chat list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/pin [post]
func (h *ChatHandler) Pin(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.Pin(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Unpin godoc
// @Summary     Unpin a chat
// @Description Removes a chat from the pinned position at the top of the chat list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/unpin [post]
func (h *ChatHandler) Unpin(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.Unpin(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Unarchive godoc
// @Summary     Unarchive a chat
// @Description Removes a chat from the archived list
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/unarchive [post]
func (h *ChatHandler) Unarchive(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.Unarchive(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Unmute godoc
// @Summary     Unmute a chat
// @Description Removes mute from a chat
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/unmute [post]
func (h *ChatHandler) Unmute(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.Unmute(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// DeleteChat godoc
// @Summary     Delete a chat
// @Description Permanently deletes a chat (including media)
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatActionReq true "Chat JID"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/delete [post]
func (h *ChatHandler) DeleteChat(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.DeleteChat(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// MarkRead godoc
// @Summary     Mark chat as read
// @Description Marks specific messages in a chat as read
// @Tags        Chat
// @Accept      json
// @Produce     json
// @Param       body body     dto.ChatMarkReadReq true "Chat JID + message IDs"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/chat/read [post]
func (h *ChatHandler) MarkRead(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ChatMarkReadReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.chatSvc.MarkRead(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}
