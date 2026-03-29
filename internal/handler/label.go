package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type LabelHandler struct {
	labelSvc *service.LabelService
}

func NewLabelHandler(labelSvc *service.LabelService) *LabelHandler {
	return &LabelHandler{labelSvc: labelSvc}
}

func (h *LabelHandler) getSessionID(c *fiber.Ctx) (string, error) {
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

func (h *LabelHandler) AddToChat(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.LabelChatReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.labelSvc.AddToChat(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Label added to chat"))
}

func (h *LabelHandler) RemoveFromChat(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.LabelChatReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.labelSvc.RemoveFromChat(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Label removed from chat"))
}

func (h *LabelHandler) AddToMessage(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.LabelMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.labelSvc.AddToMessage(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Label added to message"))
}

func (h *LabelHandler) RemoveFromMessage(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.LabelMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.labelSvc.RemoveFromMessage(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Label removed from message"))
}

func (h *LabelHandler) EditLabel(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.EditLabelReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.labelSvc.EditLabel(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Label edited"))
}
