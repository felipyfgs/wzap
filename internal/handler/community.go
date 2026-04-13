package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/service"
)

type CommunityHandler struct {
	communitySvc *service.CommunityService
}

func NewCommunityHandler(communitySvc *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{communitySvc: communitySvc}
}

// Create godoc
// @Summary     Create a community
// @Description Creates a new WhatsApp Community (a group of groups) with a name and optional description
// @Tags        Community
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.CreateCommunityReq true "Community payload"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/community/create [post]
func (h *CommunityHandler) Create(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.CreateCommunityReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	group, err := h.communitySvc.Create(c.Context(), id, req)
	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to create community")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", "internal server error"))
	}
	return c.JSON(dto.SuccessResp(group))
}

// AddParticipant godoc
// @Summary     Add subgroup to community
// @Description Adds one or more subgroups (by JID) as participants to an existing community
// @Tags        Community
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.CommunityParticipantReq true "Community participant payload"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/community/participant/add [post]
func (h *CommunityHandler) AddParticipant(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.CommunityParticipantReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	participants, err := h.communitySvc.AddParticipant(c.Context(), id, req)
	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to add community participant")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", "internal server error"))
	}
	return c.JSON(dto.SuccessResp(participants))
}

// RemoveParticipant godoc
// @Summary     Remove subgroup from community
// @Description Removes one or more subgroups (by JID) from an existing community
// @Tags        Community
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.CommunityParticipantReq true "Community participant payload"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/community/participant/remove [post]
func (h *CommunityHandler) RemoveParticipant(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.CommunityParticipantReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	participants, err := h.communitySvc.RemoveParticipant(c.Context(), id, req)
	if err != nil {
		logger.Warn().Err(err).Str("component", "handler").Str("session", id).Msg("failed to remove community participant")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", "internal server error"))
	}
	return c.JSON(dto.SuccessResp(participants))
}
