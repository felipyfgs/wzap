package api

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
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}

// Create godoc
// @Summary     Create a community
// @Description Creates a new WhatsApp Community (a group of groups) with a name and optional description
// @Tags        Community
// @Accept      json
// @Produce     json
// @Param       body body     model.CreateCommunityReq true "Community payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /community/create [post]
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

// AddParticipant godoc
// @Summary     Add subgroup to community
// @Description Adds one or more subgroups (by JID) as participants to an existing community
// @Tags        Community
// @Accept      json
// @Produce     json
// @Param       body body     model.CommunityParticipantReq true "Community participant payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /community/participant/add [post]
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

// RemoveParticipant godoc
// @Summary     Remove subgroup from community
// @Description Removes one or more subgroups (by JID) from an existing community
// @Tags        Community
// @Accept      json
// @Produce     json
// @Param       body body     model.CommunityParticipantReq true "Community participant payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /community/participant/remove [post]
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
