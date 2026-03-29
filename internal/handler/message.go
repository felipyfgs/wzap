package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"wzap/internal/dto"
	"wzap/internal/service"
)

type MessageHandler struct {
	msgSvc *service.MessageService
}

func NewMessageHandler(msgSvc *service.MessageService) *MessageHandler {
	return &MessageHandler{msgSvc: msgSvc}
}


// SendText godoc
// @Summary     Send a text message
// @Description Sends a text message via WhatsApp. If :id is omitted, session is identified from Bearer token.
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendTextReq true "Message payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendTextReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	msgID, err := h.msgSvc.SendText(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Text message sent"))
}

// SendImage godoc
// @Summary     Send an image message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendMediaReq true "Media payload"
// @Success     200  {object} dto.APIResponse
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
// @Param       body body     dto.SendMediaReq true "Media payload"
// @Success     200  {object} dto.APIResponse
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
// @Param       body body     dto.SendMediaReq true "Media payload"
// @Success     200  {object} dto.APIResponse
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
// @Param       body body     dto.SendMediaReq true "Media payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/audio [post]
func (h *MessageHandler) SendAudio(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendAudio)
}

func (h *MessageHandler) sendMedia(c *fiber.Ctx, sendFunc func(context.Context, string, dto.SendMediaReq) (string, error)) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendMediaReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}

	if req.Base64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "base64 media data is required"))
	}

	msgID, err := sendFunc(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Media message sent"))
}

// SendContact godoc
// @Summary     Send a contact card
// @Description Sends a vCard contact message via WhatsApp to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendContactReq true "Contact payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/contact [post]
func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendContactReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendContact(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Contact message sent"))
}

// SendLocation godoc
// @Summary     Send a location message
// @Description Sends a GPS location message with optional name and address to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendLocationReq true "Location payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/location [post]
func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendLocationReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendLocation(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Location message sent"))
}

// SendPoll godoc
// @Summary     Send a poll message
// @Description Sends a poll with multiple choice options; selectable_count controls how many options a recipient may choose (0 = unlimited)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendPollReq true "Poll payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/poll [post]
func (h *MessageHandler) SendPoll(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendPollReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendPoll(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Poll message sent"))
}

// SendSticker godoc
// @Summary     Send a sticker
// @Description Sends a base64-encoded sticker image to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendStickerReq true "Sticker payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/sticker [post]
func (h *MessageHandler) SendSticker(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendStickerReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendSticker(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Sticker message sent"))
}

// SendLink godoc
// @Summary     Send a link preview message
// @Description Sends a hyperlink preview message with optional title and description to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SendLinkReq true "Link payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/link [post]
func (h *MessageHandler) SendLink(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendLinkReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.SendLink(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Link message sent"))
}

// EditMessage godoc
// @Summary     Edit a sent message
// @Description Edits an existing sent message by ID, replacing its text content
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.EditMessageReq true "Edit payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/edit [post]
func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.EditMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.EditMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Message edited"))
}

// DeleteMessage godoc
// @Summary     Delete a sent message
// @Description Revokes a previously sent message for all recipients
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.DeleteMessageReq true "Delete payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/delete [post]
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.DeleteMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.DeleteMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Message deleted"))
}

// ReactMessage godoc
// @Summary     React to a message
// @Description Adds an emoji reaction to a message; pass an empty string for reaction to remove an existing reaction
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.ReactMessageReq true "Reaction payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/reaction [post]
func (h *MessageHandler) ReactMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ReactMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	msgID, err := h.msgSvc.ReactMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(map[string]string{"messageId": msgID}, "Message reacted"))
}

// MarkRead godoc
// @Summary     Mark a message as read
// @Description Sends a read receipt for a specific message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.MarkReadReq true "Mark read payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/read [post]
func (h *MessageHandler) MarkRead(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.MarkReadReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.msgSvc.MarkRead(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil, "Message marked as read"))
}

// SetPresence godoc
// @Summary     Set typing/recording presence
// @Description Sends a typing, recording, or paused presence indicator to a specific chat; presence values: typing, recording, paused
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       body body     dto.SetPresenceReq true "Presence payload"
// @Success     200  {object} dto.APIResponse
// @Security    BearerAuth
// @Router      /messages/presence [post]
func (h *MessageHandler) SetPresence(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SetPresenceReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
	}
	if err := h.msgSvc.SetPresence(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil, "Presence set"))
}
