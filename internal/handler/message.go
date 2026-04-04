package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
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
// @Description Sends a plain text message to a WhatsApp JID (user or group)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendTextReq true "Message payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/text [post]
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendTextReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	msgID, err := h.msgSvc.SendText(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendImage godoc
// @Summary     Send an image message
// @Description Sends a base64-encoded image to a WhatsApp JID with optional caption
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendMediaReq true "Media payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/image [post]
func (h *MessageHandler) SendImage(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendImage)
}

// SendVideo godoc
// @Summary     Send a video message
// @Description Sends a base64-encoded video to a WhatsApp JID with optional caption
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendMediaReq true "Media payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/video [post]
func (h *MessageHandler) SendVideo(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendVideo)
}

// SendDocument godoc
// @Summary     Send a document
// @Description Sends a base64-encoded file as a document. Use filename to set the display name.
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendMediaReq true "Media payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/document [post]
func (h *MessageHandler) SendDocument(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendDocument)
}

// SendAudio godoc
// @Summary     Send an audio message
// @Description Sends a base64-encoded audio file as a voice note (PTT)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendMediaReq true "Media payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/audio [post]
func (h *MessageHandler) SendAudio(c *fiber.Ctx) error {
	return h.sendMedia(c, h.msgSvc.SendAudio)
}

func (h *MessageHandler) sendMedia(c *fiber.Ctx, sendFunc func(context.Context, string, dto.SendMediaReq) (string, error)) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendMediaReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	if req.Base64 == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "base64 media data is required"))
	}

	msgID, err := sendFunc(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendContact godoc
// @Summary     Send a contact card
// @Description Sends a vCard contact message via WhatsApp to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendContactReq true "Contact payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/contact [post]
func (h *MessageHandler) SendContact(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendContactReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.SendContact(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendLocation godoc
// @Summary     Send a location message
// @Description Sends a GPS location message with optional name and address to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendLocationReq true "Location payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/location [post]
func (h *MessageHandler) SendLocation(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendLocationReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.SendLocation(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendPoll godoc
// @Summary     Send a poll message
// @Description Sends a poll with multiple choice options. selectableCount controls how many options a recipient may choose (0 = unlimited)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendPollReq true "Poll payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/poll [post]
func (h *MessageHandler) SendPoll(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendPollReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.SendPoll(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendSticker godoc
// @Summary     Send a sticker
// @Description Sends a base64-encoded sticker image (WebP) to the specified recipient
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendStickerReq true "Sticker payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/sticker [post]
func (h *MessageHandler) SendSticker(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendStickerReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.SendSticker(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendLink godoc
// @Summary     Send a link preview message
// @Description Sends a hyperlink with optional title and description as a rich preview message
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendLinkReq true "Link payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/link [post]
func (h *MessageHandler) SendLink(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendLinkReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.SendLink(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// EditMessage godoc
// @Summary     Edit a sent message
// @Description Edits a previously sent message by mid, replacing its text content
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.EditMessageReq true "Edit payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/edit [post]
func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.EditMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.EditMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// DeleteMessage godoc
// @Summary     Delete a sent message
// @Description Revokes a previously sent message for all recipients (unsend)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.DeleteMessageReq true "Delete payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/delete [post]
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.DeleteMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.DeleteMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// ReactMessage godoc
// @Summary     React to a message
// @Description Adds an emoji reaction to a message. Pass an empty string for reaction to remove an existing one.
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.ReactMessageReq true "Reaction payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/reaction [post]
func (h *MessageHandler) ReactMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ReactMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	msgID, err := h.msgSvc.ReactMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// MarkRead godoc
// @Summary     Mark a message as read
// @Description Sends a read receipt for a specific message (removes unread indicator)
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.MarkReadReq true "Mark read payload"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/read [post]
func (h *MessageHandler) MarkRead(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.MarkReadReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.msgSvc.MarkRead(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// SetPresence godoc
// @Summary     Set typing/recording presence
// @Description Sends a chat presence indicator. Accepted values for presence: typing, recording, paused
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SetPresenceReq true "Presence payload"
// @Success     200 {object} dto.APIResponse
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/presence [post]
func (h *MessageHandler) SetPresence(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SetPresenceReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.msgSvc.SetPresence(c.Context(), id, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}
	return c.JSON(dto.SuccessResp(nil))
}

// SendButton godoc
// @Summary     Send a button message
// @Description Sends a message with interactive buttons
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendButtonReq true "Button message payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/button [post]
func (h *MessageHandler) SendButton(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendButtonReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	msgID, err := h.msgSvc.SendButton(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendList godoc
// @Summary     Send a list message
// @Description Sends a message with interactive list sections
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendListReq true "List message payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/list [post]
func (h *MessageHandler) SendList(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendListReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	msgID, err := h.msgSvc.SendList(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendStatusText godoc
// @Summary     Send a text status
// @Description Sends a text message to WhatsApp Stories/Status
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendStatusTextReq true "Status text payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/status/text [post]
func (h *MessageHandler) SendStatusText(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendStatusTextReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	msgID, err := h.msgSvc.SendStatusText(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// SendStatusImage godoc
// @Summary     Send an image status
// @Description Sends an image to WhatsApp Stories/Status
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendStatusMediaReq true "Status image payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/status/image [post]
func (h *MessageHandler) SendStatusImage(c *fiber.Ctx) error {
	return h.sendStatusMedia(c, "image")
}

// SendStatusVideo godoc
// @Summary     Send a video status
// @Description Sends a video to WhatsApp Stories/Status
// @Tags        Status
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.SendStatusMediaReq true "Status video payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/status/video [post]
func (h *MessageHandler) SendStatusVideo(c *fiber.Ctx) error {
	return h.sendStatusMedia(c, "video")
}

func (h *MessageHandler) sendStatusMedia(c *fiber.Ctx, mediaType string) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.SendStatusMediaReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	var msgID string
	switch mediaType {
	case "image":
		msgID, err = h.msgSvc.SendStatusMedia(c.Context(), id, req, whatsmeow.MediaImage)
	case "video":
		msgID, err = h.msgSvc.SendStatusMedia(c.Context(), id, req, whatsmeow.MediaVideo)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Invalid Media Type", "media type must be image or video"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}

// ForwardMessage godoc
// @Summary     Forward a message
// @Description Forwards an existing message to another chat or group
// @Tags        Messages
// @Accept      json
// @Produce     json
// @Param       sessionId path string true "Session name or ID"
// @Param       body body dto.ForwardMessageReq true "Forward message payload"
// @Success     200 {object} dto.APIResponse{Data=dto.MidResp}
// @Failure     400 {object} dto.APIError
// @Failure     500 {object} dto.APIError
// @Security    Authorization
// @Router      /sessions/{sessionId}/messages/forward [post]
func (h *MessageHandler) ForwardMessage(c *fiber.Ctx) error {
	id, err := getSessionID(c)
	if err != nil {
		return err
	}
	var req dto.ForwardMessageReq
	if err := parseAndValidate(c, &req); err != nil {
		return err
	}

	msgID, err := h.msgSvc.ForwardMessage(c.Context(), id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Send Error", err.Error()))
	}

	return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}
