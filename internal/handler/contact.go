package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type ContactHandler struct {
	contactSvc *service.ContactService
}

func NewContactHandler(contactSvc *service.ContactService) *ContactHandler {
	return &ContactHandler{contactSvc: contactSvc}
}

func (h *ContactHandler) getSessionID(c *fiber.Ctx) (string, error) {
	if id := c.Params("id"); id != "" {
		return id, nil
	}
	if val := c.Locals("session_id"); val != nil {
		return val.(string), nil
	}
	if id := c.Get("X-Session-ID"); id != "" {
		return id, nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required (auth token, or header X-Session-ID)")
}

// List godoc
// @Summary     List contacts
// @Description Returns all contacts from the WhatsApp session
// @Tags        Contacts
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /contacts [get]
func (h *ContactHandler) List(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	contacts, err := h.contactSvc.List(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("List Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(contacts, "Contacts retrieved"))
}

// Check godoc
// @Summary     Check contacts on WhatsApp
// @Description Checks if phone numbers are registered on WhatsApp
// @Tags        Contacts
// @Accept      json
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Param       body body     model.CheckContactReq true "Phone numbers"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /contacts/check [post]
func (h *ContactHandler) Check(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.CheckContactReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}

	results, err := h.contactSvc.CheckContacts(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Check Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(results, "Contacts checked"))
}
