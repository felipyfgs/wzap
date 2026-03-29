package api

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type NewsletterHandler struct {
	newsletterSvc *service.NewsletterService
}

func NewNewsletterHandler(newsletterSvc *service.NewsletterService) *NewsletterHandler {
	return &NewsletterHandler{newsletterSvc: newsletterSvc}
}

func (h *NewsletterHandler) getSessionID(c *fiber.Ctx) (string, error) {
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}

// Create godoc
// @Summary     Create a newsletter
// @Description Creates a new WhatsApp Newsletter (channel) with optional description and profile picture
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     model.CreateNewsletterReq true "Newsletter payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /newsletter/create [post]
func (h *NewsletterHandler) Create(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.CreateNewsletterReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	meta, err := h.newsletterSvc.Create(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(meta, "Newsletter created"))
}

// Info godoc
// @Summary     Get newsletter info
// @Description Retrieves metadata about a newsletter by its JID
// @Tags        Newsletter
// @Produce     json
// @Param       jid query string true "Newsletter JID"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /newsletter/info [post]
func (h *NewsletterHandler) Info(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	jid := c.Query("jid")
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "jid is required"))
	}
	meta, err := h.newsletterSvc.GetInfo(c.Context(), id, jid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(meta, "Newsletter info retrieved"))
}

// Invite godoc
// @Summary     Get newsletter info from invite code
// @Description Retrieves newsletter metadata from an invite code without subscribing
// @Tags        Newsletter
// @Produce     json
// @Param       code query string true "Newsletter invite code"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /newsletter/invite [post]
func (h *NewsletterHandler) Invite(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "code is required"))
	}
	meta, err := h.newsletterSvc.GetInvite(c.Context(), id, code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(meta, "Newsletter invite info retrieved"))
}

// List godoc
// @Summary     List subscribed newsletters
// @Description Returns all newsletters the current session is subscribed to
// @Tags        Newsletter
// @Produce     json
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /newsletter/list [get]
func (h *NewsletterHandler) List(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	newsletters, err := h.newsletterSvc.List(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(newsletters, "Subscribed newsletters retrieved"))
}

// Messages godoc
// @Summary     Get newsletter messages
// @Description Fetches messages from a newsletter; supports pagination via before_id cursor and count
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     model.NewsletterMessageReq true "Messages pagination payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /newsletter/messages [post]
func (h *NewsletterHandler) Messages(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.NewsletterMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgs, err := h.newsletterSvc.Messages(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(msgs, "Newsletter messages retrieved"))
}

// Subscribe godoc
// @Summary     Subscribe to a newsletter
// @Description Subscribes the current session to a newsletter identified by its JID
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     object{jid=string} true "Newsletter JID payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /newsletter/subscribe [post]
func (h *NewsletterHandler) Subscribe(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req struct {
		JID string `json:"jid"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.newsletterSvc.Subscribe(c.Context(), id, req.JID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Subscribed to newsletter"))
}
