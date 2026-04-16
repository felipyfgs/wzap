package handler

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	"wzap/internal/integrations/chatwoot"
	"wzap/internal/logger"
	"wzap/internal/repo"
	"wzap/internal/service"
)

type CloudWAAPIPresigner interface {
	GetPresignedURL(ctx context.Context, key string) (string, error)
}

type CloudWAAPIHandler struct {
	chatwootRepo   chatwoot.Repo
	messageSvc     *service.MessageService
	mediaPresigner CloudWAAPIPresigner
	msgRepo        repo.MessageRepo
}

func NewCloudWAAPIHandler(chatwootRepo chatwoot.Repo, messageSvc *service.MessageService, mediaPresigner CloudWAAPIPresigner, msgRepo repo.MessageRepo) *CloudWAAPIHandler {
	return &CloudWAAPIHandler{
		chatwootRepo:   chatwootRepo,
		messageSvc:     messageSvc,
		mediaPresigner: mediaPresigner,
		msgRepo:        msgRepo,
	}
}

func (h *CloudWAAPIHandler) VerifyWebhook(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode != "subscribe" {
		return c.SendStatus(fiber.StatusForbidden)
	}

	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.SendStatus(fiber.StatusForbidden)
	}

	if cfg.WebhookToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.WebhookToken)) != 1 {
		return c.SendStatus(fiber.StatusForbidden)
	}

	return c.SendString(challenge)
}

func (h *CloudWAAPIHandler) SendMessage(c *fiber.Ctx) error {
	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	if !h.validateBearerToken(c, cfg.WebhookToken) {
		return nil
	}

	var req dto.CloudAPIMessageReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Invalid JSON payload", "OAuthException", 131000))
	}

	if req.Status == "read" && req.MessageID != "" {
		return h.handleMarkRead(c, cfg, req)
	}

	if req.To == "" {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'to' field", "OAuthException", 131000))
	}

	to := normalizePhone(req.To) + "@s.whatsapp.net"

	switch req.Type {
	case "text":
		return h.handleTextSend(c, cfg, req, to)
	case "image":
		return h.handleMediaSend(c, cfg, req, to, "image")
	case "video":
		return h.handleMediaSend(c, cfg, req, to, "video")
	case "audio":
		return h.handleMediaSend(c, cfg, req, to, "audio")
	case "document":
		return h.handleDocumentSend(c, cfg, req, to)
	case "location":
		return h.handleLocationSend(c, cfg, req, to)
	case "reaction":
		return h.handleReactionSend(c, cfg, req, to)
	case "contacts":
		return h.handleContactSend(c, cfg, req, to)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Unsupported message type", "OAuthException", 131009))
	}
}

func (h *CloudWAAPIHandler) GetMedia(c *fiber.Ctx) error {
	phone := c.Params("phone")
	mediaID := c.Params("media_id")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	if !h.validateBearerToken(c, cfg.WebhookToken) {
		return nil
	}

	if mediaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing media_id", "OAuthException", 100))
	}

	if h.mediaPresigner == nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Media not found", "OAuthException", 100))
	}

	key := fmt.Sprintf("chatwoot/%s/%s", cfg.SessionID, mediaID)
	presignedURL, err := h.mediaPresigner.GetPresignedURL(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Media not found", "OAuthException", 100))
	}

	return c.JSON(dto.CloudAPIMediaResp{
		URL:              presignedURL,
		MessagingProduct: "whatsapp",
		ID:               mediaID,
	})
}

func (h *CloudWAAPIHandler) handleTextSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if req.Text == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'text' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	sendReq := dto.SendTextReq{
		Phone: to,
		Body:  req.Text.Body,
	}
	if req.Context != nil && req.Context.MessageID != "" {
		sendReq.ReplyTo = &dto.ReplyContext{MessageID: req.Context.MessageID}
	}

	msgID, err := h.messageSvc.SendText(c.Context(), sessionID, sendReq)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send text")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudWAAPIHandler) handleMediaSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to, mediaType string) error {
	var media *dto.CloudAPIMedia
	switch mediaType {
	case "image":
		media = req.Image
	case "video":
		media = req.Video
	case "audio":
		media = req.Audio
	}
	if media == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError(fmt.Sprintf("Missing '%s' field", mediaType), "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	sendReq := dto.SendMediaReq{
		Phone:    to,
		URL:      media.Link,
		MimeType: media.MimeType,
		Caption:  media.Caption,
	}

	if req.Context != nil && req.Context.MessageID != "" {
		sendReq.ReplyTo = &dto.ReplyContext{MessageID: req.Context.MessageID}
	}

	var msgID string
	var err error

	switch mediaType {
	case "image":
		msgID, err = h.messageSvc.SendImage(c.Context(), sessionID, sendReq)
	case "video":
		msgID, err = h.messageSvc.SendVideo(c.Context(), sessionID, sendReq)
	case "audio":
		msgID, err = h.messageSvc.SendAudio(c.Context(), sessionID, sendReq)
	}

	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send media")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudWAAPIHandler) handleDocumentSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if req.Document == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'document' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	sendReq := dto.SendMediaReq{
		Phone:    to,
		URL:      req.Document.Link,
		MimeType: req.Document.MimeType,
		Caption:  req.Document.Caption,
		FileName: req.Document.Filename,
	}

	if req.Context != nil && req.Context.MessageID != "" {
		sendReq.ReplyTo = &dto.ReplyContext{MessageID: req.Context.MessageID}
	}

	msgID, err := h.messageSvc.SendDocument(c.Context(), sessionID, sendReq)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send document")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudWAAPIHandler) handleLocationSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if req.Location == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'location' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	sendReq := dto.SendLocationReq{
		Phone:     to,
		Latitude:  req.Location.Latitude,
		Longitude: req.Location.Longitude,
		Name:      req.Location.Name,
		Address:   req.Location.Address,
	}

	msgID, err := h.messageSvc.SendLocation(c.Context(), sessionID, sendReq)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send location")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudWAAPIHandler) handleContactSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if len(req.Contacts) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'contacts' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	for _, contact := range req.Contacts {
		name := contact.Name.FormattedName
		vcard := buildVCard(contact)

		sendReq := dto.SendContactReq{
			Phone: to,
			Name:  name,
			Vcard: vcard,
		}

		_, err := h.messageSvc.SendContact(c.Context(), sessionID, sendReq)
		if err != nil {
			logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send contact")
			return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
		}
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), ""))
}

func (h *CloudWAAPIHandler) handleReactionSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if req.Reaction == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'reaction' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	sendReq := dto.ReactMessageReq{
		Phone:     to,
		MessageID: req.Reaction.MessageID,
		Reaction:  req.Reaction.Emoji,
	}

	msgID, err := h.messageSvc.ReactMessage(c.Context(), sessionID, sendReq)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send reaction")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudWAAPIHandler) handleMarkRead(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq) error {
	sessionID := resolveSessionIDFromConfig(cfg)

	phone := ""
	if h.msgRepo != nil {
		if msg, err := h.msgRepo.FindByID(c.Context(), sessionID, req.MessageID); err == nil && msg.ChatJID != "" {
			phone = msg.ChatJID
		}
	}
	if phone == "" {
		logger.Warn().Str("component", "handler").Str("session", sessionID).Str("messageId", req.MessageID).Msg("Cloud API: mark read — could not resolve chat JID, skipping")
		return c.JSON(fiber.Map{"success": true})
	}

	markReq := dto.MarkReadReq{
		Phone:     phone,
		MessageID: req.MessageID,
	}

	if err := h.messageSvc.MarkRead(c.Context(), sessionID, markReq); err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to mark read")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *CloudWAAPIHandler) validateBearerToken(c *fiber.Ctx, expectedToken string) bool {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		_ = c.Status(fiber.StatusUnauthorized).JSON(cloudAPIError("Invalid access token", "OAuthException", 190))
		return false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		token = strings.TrimPrefix(authHeader, "bearer ")
	}

	if expectedToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
		_ = c.Status(fiber.StatusUnauthorized).JSON(cloudAPIError("Invalid access token", "OAuthException", 190))
		return false
	}

	return true
}

func (h *CloudWAAPIHandler) resolveConfigByPhone(ctx context.Context, phone string) (*chatwoot.Config, error) {
	normalized := normalizePhone(phone)
	cfg, err := h.chatwootRepo.FindByPhoneAndInboxType(ctx, normalized, "cloud")
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("cloud integration disabled for session %s", cfg.SessionID)
	}
	return cfg, nil
}

func resolveSessionIDFromConfig(cfg *chatwoot.Config) string {
	return cfg.SessionID
}

func normalizePhone(phone string) string {
	return strings.TrimLeft(strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone), "0")
}

func cloudAPIError(message, errType string, code int) dto.CloudAPIErrorResp {
	return dto.CloudAPIErrorResp{
		Error: dto.CloudAPIErrorDetail{
			Message: message,
			Type:    errType,
			Code:    code,
		},
	}
}

func cloudAPISuccess(to, msgID string) dto.CloudAPIMessageResp {
	resp := dto.CloudAPIMessageResp{
		MessagingProduct: "whatsapp",
		Contacts: []dto.CloudAPIRespContact{
			{Input: to, WaID: to},
		},
		Messages: []dto.CloudAPIRespMessage{},
	}
	if msgID != "" {
		resp.Messages = append(resp.Messages, dto.CloudAPIRespMessage{ID: msgID})
	}
	return resp
}

func buildVCard(contact dto.CloudAPIContact) string {
	var sb strings.Builder
	sb.WriteString("BEGIN:VCARD\n")
	sb.WriteString("VERSION:3.0\n")
	sb.WriteString("FN:" + contact.Name.FormattedName + "\n")
	if contact.Name.FirstName != "" {
		sb.WriteString("N:" + contact.Name.LastName + ";" + contact.Name.FirstName + ";;;\n")
	}
	for _, p := range contact.Phones {
		sb.WriteString("TEL;TYPE=CELL:" + p.Phone + "\n")
	}
	for _, e := range contact.Emails {
		sb.WriteString("EMAIL;TYPE=INTERNET:" + e.Email + "\n")
	}
	if contact.Org.Company != "" {
		sb.WriteString("ORG:" + contact.Org.Company + "\n")
	}
	sb.WriteString("END:VCARD")
	return sb.String()
}
