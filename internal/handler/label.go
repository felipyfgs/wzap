package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/dto"
	"wzap/internal/service"
)

type LabelHandler struct {
	labelSvc *service.LabelService
}

func NewLabelHandler(labelSvc *service.LabelService) *LabelHandler {
	return &LabelHandler{labelSvc: labelSvc}
}

// AddToChat godoc
// @Summary     Add label to chat
// @Description Applies a label to an entire chat conversation
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     dto.LabelChatReq true "Label chat payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/label/chat [post]
func (h *LabelHandler) AddToChat(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.LabelChatReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.labelSvc.AddToChat(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// RemoveFromChat godoc
// @Summary     Remove label from chat
// @Description Removes a label from a chat conversation
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     dto.LabelChatReq true "Label chat payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/unlabel/chat [post]
func (h *LabelHandler) RemoveFromChat(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.LabelChatReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.labelSvc.RemoveFromChat(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// AddToMessage godoc
// @Summary     Add label to message
// @Description Applies a label to a specific message within a chat
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     dto.LabelMessageReq true "Label message payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/label/message [post]
func (h *LabelHandler) AddToMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.LabelMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.labelSvc.AddToMessage(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// RemoveFromMessage godoc
// @Summary     Remove label from message
// @Description Removes a label from a specific message
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     dto.LabelMessageReq true "Label message payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/unlabel/message [post]
func (h *LabelHandler) RemoveFromMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.LabelMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.labelSvc.RemoveFromMessage(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// EditLabel godoc
// @Summary     Edit a label
// @Description Edits an existing label's name, color, or marks it as deleted
// @Tags        Labels
// @Accept      json
// @Produce     json
// @Param       body body     dto.EditLabelReq true "Edit label payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/label/edit [post]
func (h *LabelHandler) EditLabel(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.EditLabelReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.labelSvc.EditLabel(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}
