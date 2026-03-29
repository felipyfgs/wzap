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
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}

// AddToChat godoc
// @Summary     Add label to chat
// @Description Applies a label to an entire chat conversation
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     model.LabelChatReq true "Label chat payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /label/chat [post]
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

// RemoveFromChat godoc
// @Summary     Remove label from chat
// @Description Removes a label from a chat conversation
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     model.LabelChatReq true "Label chat payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /unlabel/chat [post]
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

// AddToMessage godoc
// @Summary     Add label to message
// @Description Applies a label to a specific message within a chat
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     model.LabelMessageReq true "Label message payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /label/message [post]
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

// RemoveFromMessage godoc
// @Summary     Remove label from message
// @Description Removes a label from a specific message
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     model.LabelMessageReq true "Label message payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /unlabel/message [post]
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

// EditLabel godoc
// @Summary     Edit a label
// @Description Edits an existing label's name, color, or marks it as deleted
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     model.EditLabelReq true "Edit label payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /label/edit [post]
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
