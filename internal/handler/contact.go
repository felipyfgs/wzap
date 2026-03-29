package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/dto"
	"wzap/internal/service"
)

type ContactHandler struct {
	contactSvc *service.ContactService
}

func NewContactHandler(contactSvc *service.ContactService) *ContactHandler {
	return &ContactHandler{contactSvc: contactSvc}
}


// List godoc
// @Summary     List contacts
// @Description Returns all contacts from the WhatsApp session
// @Tags        Contacts
// @Produce     json
// @Success     200 {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts [get]
func (h *ContactHandler) List(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	contacts, err := h.contactSvc.List(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("List Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(contacts))
}

// Check godoc
// @Summary     Check contacts on WhatsApp
// @Description Checks if phone numbers are registered on WhatsApp
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       body body     dto.CheckContactReq true "Phone numbers"
// @Success     200  {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts/check [post]
func (h *ContactHandler) Check(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.CheckContactReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	results, err := h.contactSvc.CheckContacts(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Check Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(results))
}

// GetAvatar godoc
// @Summary     Get contact avatar
// @Description Fetches the profile picture URL and picture ID for the given WhatsApp JID
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       body body     dto.GetAvatarReq true "JID payload"
// @Success     200  {object} dto.APIResponse{data=dto.GetAvatarResp}
// @Security    ApiKeyAuth
// @Router      /contacts/avatar [post]
func (h *ContactHandler) GetAvatar(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GetAvatarReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	resp, err := h.contactSvc.GetAvatar(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(resp))
}

// Block godoc
// @Summary     Block a contact
// @Description Blocks a WhatsApp contact by JID, preventing them from sending messages
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       body body     dto.BlockContactReq true "JID payload"
// @Success     200  {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts/block [post]
func (h *ContactHandler) Block(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.BlockContactReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	resp, err := h.contactSvc.Block(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(resp))
}

// Unblock godoc
// @Summary     Unblock a contact
// @Description Unblocks a previously blocked WhatsApp contact by JID
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       body body     dto.BlockContactReq true "JID payload"
// @Success     200  {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts/unblock [post]
func (h *ContactHandler) Unblock(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.BlockContactReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	resp, err := h.contactSvc.Unblock(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(resp))
}

// GetBlocklist godoc
// @Summary     Get blocked contacts list
// @Description Returns the full list of JIDs currently blocked by the session
// @Tags        Contacts
// @Produce     json
// @Success     200  {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts/blocklist [get]
func (h *ContactHandler) GetBlocklist(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	resp, err := h.contactSvc.GetBlocklist(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(resp))
}

// GetUserInfo godoc
// @Summary     Get user info for JIDs
// @Description Fetches detailed user info (status, profile picture, devices) for one or more WhatsApp JIDs
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       body body     dto.GetUserInfoReq true "JIDs payload"
// @Success     200  {object} dto.APIResponse{data=[]dto.UserInfoResp}
// @Security    ApiKeyAuth
// @Router      /contacts/info [post]
func (h *ContactHandler) GetUserInfo(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.GetUserInfoReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	resp, err := h.contactSvc.GetUserInfo(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(resp))
}

// GetPrivacySettings godoc
// @Summary     Get privacy settings
// @Description Retrieves the current session's WhatsApp privacy settings (last-seen, profile photo, status visibility)
// @Tags        Contacts
// @Produce     json
// @Success     200  {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts/privacy [get]
func (h *ContactHandler) GetPrivacySettings(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	resp, err := h.contactSvc.GetPrivacySettings(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(resp))
}

// SetProfilePicture godoc
// @Summary     Set profile picture
// @Description Updates the session account's WhatsApp profile picture with a base64-encoded image
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       body body     dto.SetProfilePictureReq true "Base64 image payload"
// @Success     200  {object} dto.APIResponse
// @Security    ApiKeyAuth
// @Router      /contacts/profile-picture [post]
func (h *ContactHandler) SetProfilePicture(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SetProfilePictureReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	resp, err := h.contactSvc.SetProfilePicture(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"pictureId": resp}))
}
