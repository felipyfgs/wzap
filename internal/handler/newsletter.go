package handler

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
