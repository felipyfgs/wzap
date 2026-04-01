package handler

import (
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"
	"wzap/internal/dto"
	"wzap/internal/service"
)

type GroupHandler struct {
	groupSvc *service.GroupService
}

func NewGroupHandler(groupSvc *service.GroupService) *GroupHandler {
	return &GroupHandler{groupSvc: groupSvc}
}

// List godoc
// @Summary     List joined groups
// @Description Returns all WhatsApp groups the session is part of
// @Tags        Groups
// @Produce     json
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups [get]
func (h *GroupHandler) List(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	groups, err := h.groupSvc.List(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("List Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(groups))
}

// Create godoc
// @Summary     Create a new group
// @Description Creates a new WhatsApp group with the given participants
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.CreateGroupReq true "Group properties"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/create [post]
func (h *GroupHandler) Create(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.CreateGroupReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "name is required"))
	}

	group, err := h.groupSvc.CreateGroup(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Create Group Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(group))
}

// Info godoc
// @Summary     Get group info
// @Description Get detailed information about a group by JID
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupJIDReq true "Target Group JID Payload"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/info [post]
func (h *GroupHandler) Info(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupJIDReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if req.GroupJID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "groupJid is required"))
	}

	group, err := h.groupSvc.GetInfo(c.Context(), id, req.GroupJID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Get Group Info Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(group))
}

// GetInviteLink godoc
// @Summary     Get group invite link
// @Description Gets the invite link for a WhatsApp group, optionally resetting it
// @Tags        Groups
// @Produce     json
// @Param       request body dto.GroupJIDReq true "Target Group JID Payload"
// @Param       reset query bool false "Reset the invite link"
// @Success     200 {object} dto.APIResponse{Data=dto.GroupInviteLinkResp}
// @Security    Authorization
// @Router      /groups/invite-link [post]
func (h *GroupHandler) GetInviteLink(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GroupJIDReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID
	reset := c.QueryBool("reset", false)

	link, err := h.groupSvc.GetInviteLink(c.Context(), id, jid, reset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Get Invite Link Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.GroupInviteLinkResp{Link: link}))
}

// GetInfoFromLink godoc
// @Summary     Get group info from invite link
// @Description Previews a group's info using an invite code without joining
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupJoinReq true "Invite Code Payload"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/invite-info [post]
func (h *GroupHandler) GetInfoFromLink(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupJoinReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if req.InviteCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "inviteCode is required"))
	}

	group, err := h.groupSvc.GetInfoFromLink(c.Context(), id, req.InviteCode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Get Info From Link Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(group))
}

// JoinWithLink godoc
// @Summary     Join group via link
// @Description Joins a group using an invite code
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupJoinReq true "Invite Code"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/join [post]
func (h *GroupHandler) JoinWithLink(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupJoinReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	jid, err := h.groupSvc.JoinWithLink(c.Context(), id, req.InviteCode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Join Group Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(fiber.Map{"jid": jid}))
}

// Leave godoc
// @Summary     Leave group
// @Description Leaves a specified WhatsApp group
// @Tags        Groups
// @Produce     json
// @Param       request body dto.GroupJIDReq true "Target Group JID Payload"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/leave [post]
func (h *GroupHandler) Leave(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GroupJIDReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID

	err = h.groupSvc.LeaveGroup(c.Context(), id, jid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Leave Group Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// UpdateParticipants godoc
// @Summary     Update group participants
// @Description Add, remove, promote or demote participants in a group
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupParticipantReq true "Participants and action"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/participants [post]
func (h *GroupHandler) UpdateParticipants(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupParticipantReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID

	res, err := h.groupSvc.UpdateParticipants(c.Context(), id, jid, req.Participants, req.Action)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Update Participants Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(res))
}

// GetRequests godoc
// @Summary     Get group join requests
// @Description Get the list of participants that requested to join the group
// @Tags        Groups
// @Produce     json
// @Param       request body dto.GroupJIDReq true "Target Group JID Payload"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/requests [post]
func (h *GroupHandler) GetRequests(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GroupJIDReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID

	res, err := h.groupSvc.GetRequestParticipants(c.Context(), id, jid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Get Requests Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(res))
}

// UpdateRequests godoc
// @Summary     Approve/Reject group join requests
// @Description Approves or rejects participants that requested to join the group
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupRequestActionReq true "Participants and action (approve/reject)"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/requests/action [post]
func (h *GroupHandler) UpdateRequests(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupRequestActionReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID

	res, err := h.groupSvc.UpdateRequestParticipants(c.Context(), id, jid, req.Participants, req.Action)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Update Requests Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(res))
}

// UpdateName godoc
// @Summary     Update group name
// @Description Updates the name of the specified WhatsApp group
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupTextReq true "New name"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/name [post]
func (h *GroupHandler) UpdateName(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupTextReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "groupJid is required"))
	}

	if err := h.groupSvc.UpdateName(c.Context(), id, jid, req.Text); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Update Name Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// UpdateDescription godoc
// @Summary     Update group description
// @Description Updates the description of the specified WhatsApp group
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupTextReq true "New description"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/description [post]
func (h *GroupHandler) UpdateDescription(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupTextReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "groupJid is required"))
	}

	if err := h.groupSvc.UpdateDescription(c.Context(), id, jid, req.Text); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Update Description Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// UpdatePhoto godoc
// @Summary     Update group photo
// @Description Updates the profile picture of the specified WhatsApp group
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupPhotoReq true "Base64 encoded photo"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/photo [post]
func (h *GroupHandler) UpdatePhoto(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupPhotoReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID

	bytes, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Request", "failed to decode base64 photo"))
	}

	picID, err := h.groupSvc.UpdatePhoto(c.Context(), id, jid, bytes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Update Photo Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.PictureIDResp{PictureID: picID}))
}

// SetAnnounce godoc
// @Summary     Set group announce mode
// @Description Sets whether only admins can send messages in the group
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupSettingReq true "Enabled state"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/announce [post]
func (h *GroupHandler) SetAnnounce(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupSettingReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "groupJid is required"))
	}

	if err := h.groupSvc.SetAnnounce(c.Context(), id, jid, req.Enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Set Announce Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// SetLocked godoc
// @Summary     Set group locked mode
// @Description Sets whether only admins can edit group info
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupSettingReq true "Enabled state"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/locked [post]
func (h *GroupHandler) SetLocked(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupSettingReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "groupJid is required"))
	}

	if err := h.groupSvc.SetLocked(c.Context(), id, jid, req.Enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Set Locked Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// SetJoinApproval godoc
// @Summary     Set group join approval mode
// @Description Activates or deactivates the admin approval system for new members
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       request body dto.GroupSettingReq true "Enabled state"
// @Success     200 {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/join-approval [post]
func (h *GroupHandler) SetJoinApproval(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}

	var req dto.GroupSettingReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	jid := req.GroupJID
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "groupJid is required"))
	}

	if err := h.groupSvc.SetJoinApproval(c.Context(), id, jid, req.Enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Set Join Approval Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// RemovePhoto godoc
// @Summary     Remove group photo
// @Description Removes the current group profile photo
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       body body     dto.GroupJIDReq true "Group JID"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/photo/remove [post]
func (h *GroupHandler) RemovePhoto(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GroupJIDReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.groupSvc.RemovePhoto(c.Context(), id, req.GroupJID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Remove Photo Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// SetEphemeral godoc
// @Summary     Set ephemeral timer
// @Description Sets the disappearing messages timer for a group (in seconds: 0, 86400, 604800, 7776000)
// @Tags        Groups
// @Accept      json
// @Produce     json
// @Param       body body     dto.GroupEphemeralReq true "Ephemeral settings"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Router      /groups/ephemeral [post]
func (h *GroupHandler) SetEphemeral(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GroupEphemeralReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	duration := time.Duration(req.Duration) * time.Second
	if err := h.groupSvc.SetEphemeral(c.Context(), id, req.GroupJID, duration); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Ephemeral Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}
