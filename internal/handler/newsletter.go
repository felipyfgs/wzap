package handler

import (
	"wzap/internal/dto"
	"wzap/internal/service"

	"github.com/gofiber/fiber/v2"
)

type NewsletterHandler struct {
	newsletterSvc *service.NewsletterService
}

func NewNewsletterHandler(newsletterSvc *service.NewsletterService) *NewsletterHandler {
	return &NewsletterHandler{newsletterSvc: newsletterSvc}
}

// Create godoc
// @Summary     Create a newsletter
// @Description Creates a new WhatsApp Newsletter (channel) with optional description and profile picture
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.CreateNewsletterReq true "Newsletter payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/create [post]
func (h *NewsletterHandler) Create(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.CreateNewsletterReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	meta, err := h.newsletterSvc.Create(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(meta))
}

// Info godoc
// @Summary     Get newsletter info
// @Description Retrieves metadata about a newsletter by its JID
// @Tags        Newsletter
// @Produce     json
// @Param       jid query string true "Newsletter JID"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/info [post]
func (h *NewsletterHandler) Info(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	jid := c.Query("jid")
	if jid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "jid is required"))
	}
	meta, err := h.newsletterSvc.GetInfo(c.Context(), id, jid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(meta))
}

// Invite godoc
// @Summary     Get newsletter info from invite code
// @Description Retrieves newsletter metadata from an invite code without subscribing
// @Tags        Newsletter
// @Produce     json
// @Param       code query string true "Newsletter invite code"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/invite [post]
func (h *NewsletterHandler) Invite(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "code is required"))
	}
	meta, err := h.newsletterSvc.GetInvite(c.Context(), id, code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(meta))
}

// List godoc
// @Summary     List subscribed newsletters
// @Description Returns all newsletters the current session is subscribed to
// @Tags        Newsletter
// @Produce     json
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/list [get]
func (h *NewsletterHandler) List(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	newsletters, err := h.newsletterSvc.List(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(newsletters))
}

// Messages godoc
// @Summary     Get newsletter messages
// @Description Fetches messages from a newsletter; supports pagination via before_id cursor and count
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.NewsletterMessageReq true "Messages pagination payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/messages [post]
func (h *NewsletterHandler) Messages(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.NewsletterMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgs, err := h.newsletterSvc.Messages(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(msgs))
}

// Subscribe godoc
// @Summary     Subscribe to a newsletter
// @Description Subscribes the current session to a newsletter identified by its JID
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.NewsletterSubscribeReq true "Newsletter JID payload"
// @Success     200  {object} dto.APIResponse
// @Failure     400  {object} dto.APIError
// @Failure     500  {object} dto.APIError
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/subscribe [post]
func (h *NewsletterHandler) Subscribe(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.NewsletterSubscribeReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.newsletterSvc.Subscribe(c.Context(), id, req.NewsletterJID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Unsubscribe godoc
// @Summary     Unsubscribe from a newsletter
// @Description Unfollows a newsletter
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.NewsletterSubscribeReq true "Newsletter JID"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/unsubscribe [post]
func (h *NewsletterHandler) Unsubscribe(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.NewsletterSubscribeReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.newsletterSvc.Unsubscribe(c.Context(), id, req.NewsletterJID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// Mute godoc
// @Summary     Mute/unmute a newsletter
// @Description Toggles mute status for a newsletter
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.NewsletterMuteReq true "Mute payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/mute [post]
func (h *NewsletterHandler) Mute(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.NewsletterMuteReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.newsletterSvc.Mute(c.Context(), id, req.NewsletterJID, req.Mute); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// React godoc
// @Summary     React to a newsletter message
// @Description Sends a reaction to a newsletter message
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.NewsletterReactReq true "Reaction payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/react [post]
func (h *NewsletterHandler) React(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.NewsletterReactReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.newsletterSvc.React(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// MarkViewed godoc
// @Summary     Mark newsletter messages as viewed
// @Description Marks newsletter messages as viewed by server IDs
// @Tags        Newsletter
// @Accept      json
// @Produce     json
// @Param       body body     dto.NewsletterMarkViewedReq true "Mark viewed payload"
// @Success     200  {object} dto.APIResponse
// @Security    Authorization
// @Param       sessionId path string true "Session name or ID"
// @Router      /sessions/{sessionId}/newsletter/viewed [post]
func (h *NewsletterHandler) MarkViewed(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.NewsletterMarkViewedReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.newsletterSvc.MarkViewed(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}
