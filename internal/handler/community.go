package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type CommunityHandler struct {
	communitySvc *service.CommunityService
}

func NewCommunityHandler(communitySvc *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{communitySvc: communitySvc}
}

func (h *CommunityHandler) getSessionID(c *fiber.Ctx) (string, error) {
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

func (h *CommunityHandler) Create(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.CreateCommunityReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	group, err := h.communitySvc.Create(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(group, "Community created"))
}

func (h *CommunityHandler) AddParticipant(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.CommunityParticipantReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	participants, err := h.communitySvc.AddParticipant(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(participants, "Participant added to community"))
}

func (h *CommunityHandler) RemoveParticipant(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.CommunityParticipantReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	participants, err := h.communitySvc.RemoveParticipant(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(participants, "Participant removed from community"))
}
