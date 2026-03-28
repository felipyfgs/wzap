package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type MessageHandler struct {
	msgSvc *service.MessageService
}

func NewMessageHandler(msgSvc *service.MessageService) *MessageHandler {
	return &MessageHandler{msgSvc: msgSvc}
}

func (h *MessageHandler) getSessionID(c *fiber.Ctx) (string, error) {
	if id := c.Params("id"); id != "" {
		return id, nil
	}
	if val := c.Locals("session_id"); val != nil {
		return val.(string), nil
	}
	if id := c.Get("X-Session-ID"); id != "" {
		return id, nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required (path :id, auth token, or header X-Session-ID)")
}

// SendText godoc
// @Summary     Send a text message
// @Description Sends a text message via WhatsApp. If :id is omitted, session is identified from Bearer token.
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Param       body body     model.SendTextReq true "Message payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendTextReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}

	msgID, err := h.msgSvc.SendText(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Text message sent"))
}

// SendImage godoc
// @Summary     Send an image message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Param       body body     model.SendMediaReq true "Media payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/image [post]
func (h *MessageHandler) SendImage(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendImage)
}

// SendVideo godoc
// @Summary     Send a video message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Param       body body     model.SendMediaReq true "Media payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/video [post]
func (h *MessageHandler) SendVideo(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendVideo)
}

// SendDocument godoc
// @Summary     Send a document
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Param       body body     model.SendMediaReq true "Media payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/document [post]
func (h *MessageHandler) SendDocument(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendDocument)
}

// SendAudio godoc
// @Summary     Send an audio message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       X-Session-ID header string false "Session ID (Admin fallback)"
// @Param       body body     model.SendMediaReq true "Media payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/audio [post]
func (h *MessageHandler) SendAudio(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendAudio)
}

func (h *MessageHandler) sendMedia(c *fiber.Ctx, sendFunc func(context.Context, string, model.SendMediaReq) (string, error)) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendMediaReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}

	if req.Base64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "base64 media data is required"))
	}

	msgID, err := sendFunc(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Media message sent"))
}
