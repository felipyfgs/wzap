package api

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
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}

// SendText godoc
// @Summary     Send a text message
// @Description Sends a text message via WhatsApp. If :id is omitted, session is identified from Bearer token.
// @Tags        Messages
// @Accept      json
// @Produce     json
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

// SendContact godoc
// @Summary     Send a contact card
// @Description Sends a vCard contact message via WhatsApp to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.SendContactReq true "Contact payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/contact [post]
func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendContactReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendContact(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Contact message sent"))
}

// SendLocation godoc
// @Summary     Send a location message
// @Description Sends a GPS location message with optional name and address to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.SendLocationReq true "Location payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/location [post]
func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendLocationReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendLocation(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Location message sent"))
}

// SendPoll godoc
// @Summary     Send a poll message
// @Description Sends a poll with multiple choice options; selectable_count controls how many options a recipient may choose (0 = unlimited)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.SendPollReq true "Poll payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/poll [post]
func (h *MessageHandler) SendPoll(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendPollReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendPoll(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Poll message sent"))
}

// SendSticker godoc
// @Summary     Send a sticker
// @Description Sends a base64-encoded sticker image to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.SendStickerReq true "Sticker payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/sticker [post]
func (h *MessageHandler) SendSticker(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendStickerReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendSticker(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Sticker message sent"))
}

// SendLink godoc
// @Summary     Send a link preview message
// @Description Sends a hyperlink preview message with optional title and description to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.SendLinkReq true "Link payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/link [post]
func (h *MessageHandler) SendLink(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SendLinkReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendLink(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Link message sent"))
}

// EditMessage godoc
// @Summary     Edit a sent message
// @Description Edits an existing sent message by ID, replacing its text content
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.EditMessageReq true "Edit payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/edit [post]
func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.EditMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.EditMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Message edited"))
}

// DeleteMessage godoc
// @Summary     Delete a sent message
// @Description Revokes a previously sent message for all recipients
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.DeleteMessageReq true "Delete payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/delete [post]
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.DeleteMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.DeleteMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Message deleted"))
}

// ReactMessage godoc
// @Summary     React to a message
// @Description Adds an emoji reaction to a message; pass an empty string for reaction to remove an existing reaction
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.ReactMessageReq true "Reaction payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/reaction [post]
func (h *MessageHandler) ReactMessage(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.ReactMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.ReactMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(map[string]string{"message_id": msgID}, "Message reacted"))
}

// MarkRead godoc
// @Summary     Mark a message as read
// @Description Sends a read receipt for a specific message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.MarkReadReq true "Mark read payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/read [post]
func (h *MessageHandler) MarkRead(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.MarkReadReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.msgSvc.MarkRead(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Message marked as read"))
}

// SetPresence godoc
// @Summary     Set typing/recording presence
// @Description Sends a typing, recording, or paused presence indicator to a specific chat; presence values: typing, recording, paused
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     model.SetPresenceReq true "Presence payload"
// @Success     200  {object} model.APIResponse
// @Security    BearerAuth
// @Router      /messages/presence [post]
func (h *MessageHandler) SetPresence(c *fiber.Ctx) error {
	id, err := h.getSessionID(c)
	if err != nil {
		return err
	}
	var req model.SetPresenceReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.msgSvc.SetPresence(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(model.SuccessResp(nil, "Presence set"))
}
