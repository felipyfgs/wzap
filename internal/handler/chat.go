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
	if id := c.Params("id"); id != "" {
		return id, nil
	}
	if val := c.Locals("session_id"); val != nil {
		return val.(string), nil
	}
	if id := c.Get("X-Session-ID"); id != "" {
		return id, nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}

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
