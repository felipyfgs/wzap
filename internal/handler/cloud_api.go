package handler

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path"
	"strings"

	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	"wzap/internal/integrations/chatwoot"
	"wzap/internal/logger"
	"wzap/internal/repo"
	"wzap/internal/service"
)

// Package-level note: every handler in this file emulates a subset of the
// Facebook WhatsApp Cloud API for Chatwoot's "WhatsApp Cloud" inbox.
//
// Rule: we NEVER return HTTP 401 from these endpoints. Chatwoot's
// Reauthorizable concern increments an error counter on 401 responses and
// flips the channel to `reauthorization_required` after 2 failures, which
// silently drops every subsequent webhook. Token mismatches are logged via
// `warnTokenMismatch` and otherwise ignored.

type CloudPresigner interface {
	GetPresignedURL(ctx context.Context, key string) (string, error)
}

// CloudMediaStorage lets the Cloud API emulator stream object bytes back to
// the caller (Chatwoot) without redirecting to MinIO. Redirecting is fragile
// because Chatwoot's Down.download call always sends `Authorization: Bearer
// <api_key>` and MinIO rejects 400 when both a Bearer token and an AWS v4
// presigned signature are present on the same request.
type CloudMediaStorage interface {
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Stat(ctx context.Context, key string) (contentType string, size int64, err error)
	StatMeta(ctx context.Context, key string) (contentType string, size int64, userMeta map[string]string, err error)
}

type CloudWindowUnlocker interface {
	UnlockCloudWindow(ctx context.Context, cfg *chatwoot.Config, chatJID string)
}

type CloudIdempotencyMarker interface {
	MarkSourceIDIdempotent(ctx context.Context, sessionID, sourceID string)
}

type CloudAPIHandler struct {
	chatwootRepo   chatwoot.Repo
	messageSvc     *service.MessageService
	mediaPresigner CloudPresigner
	mediaStorage   CloudMediaStorage
	windowUnlocker CloudWindowUnlocker
	idempotency    CloudIdempotencyMarker
	msgRepo        repo.MessageRepo
	adminToken     string
	serverURL      string
}

func NewCloudAPIHandler(chatwootRepo chatwoot.Repo, messageSvc *service.MessageService, mediaPresigner CloudPresigner, msgRepo repo.MessageRepo, adminToken string) *CloudAPIHandler {
	return &CloudAPIHandler{
		chatwootRepo:   chatwootRepo,
		messageSvc:     messageSvc,
		mediaPresigner: mediaPresigner,
		msgRepo:        msgRepo,
		adminToken:     adminToken,
	}
}

// SetMediaStorage enables the inline streaming proxy for Chatwoot Cloud
// downloads. When set, GetMedia / GetMediaByID return a wzap URL instead of a
// MinIO presigned URL.
func (h *CloudAPIHandler) SetMediaStorage(s CloudMediaStorage) {
	h.mediaStorage = s
}

// SetWindowUnlocker configures an optional unlock strategy for Chatwoot Cloud
// 24h window. It is used only on template sends.
func (h *CloudAPIHandler) SetWindowUnlocker(u CloudWindowUnlocker) {
	h.windowUnlocker = u
}

func (h *CloudAPIHandler) SetIdempotencyMarker(m CloudIdempotencyMarker) {
	h.idempotency = m
}

// SetServerURL configures the base URL that will be embedded in media
// responses. It should be reachable by Chatwoot (typically the docker service
// name, e.g. http://wzap_app:8080).
func (h *CloudAPIHandler) SetServerURL(u string) {
	h.serverURL = strings.TrimRight(u, "/")
}

// isAdminToken returns true if the provided token matches the admin token
// configured in the server. Used as a bypass during Chatwoot inbox creation,
// when the wz_chatwoot config for the phone does not yet exist.
func (h *CloudAPIHandler) isAdminToken(token string) bool {
	if h.adminToken == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(h.adminToken)) == 1
}

// DebugToken returns a fake valid token response to make Chatwoot believe the
// WhatsApp Cloud API channel is authenticated.
// GET /{version}/debug_token
func (h *CloudAPIHandler) DebugToken(c *fiber.Ctx) error {
	_ = c.Query("access_token") // consumed but ignored
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"is_valid":    true,
			"app_id":      "wzap",
			"application": "wzap",
			"expires_at":  0,
			"granular_scopes": []fiber.Map{
				{"scope": "whatsapp_business_management"},
				{"scope": "whatsapp_business_messaging"},
			},
		},
	})
}

// PhoneNumbers lists the phone numbers associated with the session.
// GET /{version}/{phone}/phone_numbers
func (h *CloudAPIHandler) PhoneNumbers(c *fiber.Ctx) error {
	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		if h.isAdminToken(h.extractAccessToken(c)) {
			normalized := normalizePhone(phone)
			return c.JSON(fiber.Map{
				"data": []fiber.Map{
					{
						"verified_name":        "wzap",
						"display_phone_number": normalized,
						"id":                   normalized,
					},
				},
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

	normalized := normalizePhone(phone)
	return c.JSON(fiber.Map{
		"data": []fiber.Map{
			{
				"verified_name":        cfg.InboxName,
				"display_phone_number": normalized,
				"id":                   normalized,
			},
		},
	})
}

func (h *CloudAPIHandler) MessageTemplates(c *fiber.Ctx) error {
	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		// Unknown phone: return an empty list so Chatwoot's
		// validate_provider_config? and sync_templates see a 200 response.
		// Returning 401 would trigger the Reauthorizable flow — see the note
		// at the top of this file.
		logger.Debug().Str("component", "handler").Str("phone", phone).Err(err).Msg("Cloud API: MessageTemplates unknown phone, returning empty list")
		return c.JSON(fiber.Map{"data": []fiber.Map{}})
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

	// Template único "mensagem_inicial": BODY = {{1}}, onde {{1}} é o texto
	// digitado pelo agente na UI "Nova Conversa" do Chatwoot. Isso permite
	// iniciar conversas com qualquer texto (não precisa template aprovado
	// pela Meta, porque o wzap usa whatsmeow sob o capô).
	return c.JSON(fiber.Map{
		"data": []fiber.Map{
			{
				"name":              "mensagem_inicial",
				"language":          "pt_BR",
				"status":            "APPROVED",
				"category":          "UTILITY",
				"id":                "wzap_mensagem_inicial",
				"rejected_reason":   "NONE",
				"quality_score":     fiber.Map{"score": "GREEN"},
				"previous_category": "UTILITY",
				"parameter_format":  "POSITIONAL",
				"components": []fiber.Map{
					{
						"type": "BODY",
						"text": "{{1}}",
						"example": fiber.Map{
							"body_text": [][]string{{"Olá, tudo bem? Gostaria de falar com você."}},
						},
					},
				},
			},
		},
		"paging": fiber.Map{
			"cursors": fiber.Map{
				"before": "",
				"after":  "",
			},
		},
	})
}

func (h *CloudAPIHandler) VerifyWebhook(c *fiber.Ctx) error {
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

func (h *CloudAPIHandler) PhoneStatus(c *fiber.Ctx) error {
	phone := c.Params("phone")

	// Upstream Chatwoot calls GET /v{version}/{media_id} (no phone) for media
	// downloads. WhatsApp message ids contain letters ("A5...", "3EB0..."),
	// so if :phone is not purely numeric we delegate to the media handler.
	if !isNumeric(phone) {
		c.Locals("media_id", phone)
		return h.GetMediaByID(c)
	}

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		if h.isAdminToken(h.extractAccessToken(c)) {
			normalized := normalizePhone(phone)
			return c.JSON(fiber.Map{
				"id":                       normalized,
				"display_phone_number":     normalized,
				"verified_name":            "wzap",
				"code_verification_status": "VERIFIED",
				"quality_rating":           "GREEN",
				"platform_type":            "CLOUD_API",
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

	normalized := normalizePhone(phone)
	return c.JSON(fiber.Map{
		"id":                       normalized,
		"display_phone_number":     normalized,
		"verified_name":            cfg.InboxName,
		"code_verification_status": "VERIFIED",
		"quality_rating":           "GREEN",
		"messaging_limit_tier":     "TIER_100K",
		"account_mode":             "LIVE",
		"name_status":              "APPROVED",
		"platform_type":            "CLOUD_API",
		"throughput": fiber.Map{
			"level": "STANDARD",
		},
		"webhook_configuration": fiber.Map{
			"application_webhooks": []any{},
		},
	})
}

func (h *CloudAPIHandler) RegisterPhone(c *fiber.Ctx) error {
	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

	return c.JSON(fiber.Map{
		"success": true,
	})
}

func (h *CloudAPIHandler) SubscribeApps(c *fiber.Ctx) error {
	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

	return c.JSON(fiber.Map{
		"success": true,
	})
}

func (h *CloudAPIHandler) SendMessage(c *fiber.Ctx) error {
	phone := c.Params("phone")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

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
	case "sticker":
		return h.handleMediaSend(c, cfg, req, to, "sticker")
	case "document":
		return h.handleDocumentSend(c, cfg, req, to)
	case "location":
		return h.handleLocationSend(c, cfg, req, to)
	case "reaction":
		return h.handleReactionSend(c, cfg, req, to)
	case "contacts":
		return h.handleContactSend(c, cfg, req, to)
	case "template":
		return h.handleTemplateSend(c, cfg, req, to)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Unsupported message type", "OAuthException", 131009))
	}
}

func (h *CloudAPIHandler) GetMedia(c *fiber.Ctx) error {
	phone := c.Params("phone")
	mediaID := c.Params("media_id")

	cfg, err := h.resolveConfigByPhone(c.Context(), phone)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Phone number not found", "OAuthException", 100))
	}

	h.warnTokenMismatch(c, cfg.WebhookToken)

	if mediaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing media_id", "OAuthException", 100))
	}

	url, err := h.buildMediaURL(c.Context(), cfg.SessionID, mediaID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Media not found", "OAuthException", 100))
	}

	return c.JSON(h.mediaRespWithMeta(c.Context(), cfg.SessionID, mediaID, url))
}

// GetMediaByID handles the upstream Chatwoot request format:
//
//	GET /v{version}/{media_id}
//
// (without phone_number_id in the path). Chatwoot's WhatsappCloudService
// builds the URL as `{WHATSAPP_CLOUD_BASE_URL}/v13.0/{media_id}`, so this
// route is required to respond with the presigned URL and avoid marking the
// channel as reauthorization_required on download failures.
func (h *CloudAPIHandler) GetMediaByID(c *fiber.Ctx) error {
	mediaID := c.Params("media_id")
	if mediaID == "" {
		if v, ok := c.Locals("media_id").(string); ok {
			mediaID = v
		}
	}
	if mediaID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing media_id", "OAuthException", 100))
	}

	if h.msgRepo == nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Media not found", "OAuthException", 100))
	}

	sessionID, err := h.msgRepo.FindSessionByMessageID(c.Context(), mediaID)
	if err != nil || sessionID == "" {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Media not found", "OAuthException", 100))
	}

	url, err := h.buildMediaURL(c.Context(), sessionID, mediaID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(cloudAPIError("Media not found", "OAuthException", 100))
	}

	return c.JSON(h.mediaRespWithMeta(c.Context(), sessionID, mediaID, url))
}

// mediaRespWithMeta builds a CloudAPIMediaResp populated with `mime_type` and
// `file_size` read from MinIO (when available). These fields are what Chatwoot
// uses to pick the right attachment renderer (e.g., PDF preview vs. generic
// file). Without them, the UI falls back to a bare filename/id display.
func (h *CloudAPIHandler) mediaRespWithMeta(ctx context.Context, sessionID, mediaID, url string) dto.CloudAPIMediaResp {
	resp := dto.CloudAPIMediaResp{
		URL:              url,
		MessagingProduct: "whatsapp",
		ID:               mediaID,
	}
	if h.mediaStorage == nil {
		return resp
	}
	key := fmt.Sprintf("chatwoot/%s/%s", sessionID, mediaID)
	if ct, size, err := h.mediaStorage.Stat(ctx, key); err == nil {
		resp.MimeType = ct
		resp.FileSize = size
	}
	return resp
}

// buildMediaURL returns the URL that Chatwoot should use to download the
// media. Two modes are supported:
//
//  1. Inline proxy (preferred): when `mediaStorage` is configured, we return
//     a wzap URL that streams the object bytes (see DownloadCloudMedia).
//     This avoids the MinIO 400 error caused by Chatwoot's Down.download
//     always attaching an `Authorization: Bearer` header that conflicts with
//     AWS v4 presigned signatures.
//  2. Presigned fallback: when no proxy is available, return the MinIO
//     presigned URL directly. This only works when nothing else adds a
//     Bearer header.
func (h *CloudAPIHandler) buildMediaURL(ctx context.Context, sessionID, mediaID string) (string, error) {
	if h.mediaStorage != nil && h.serverURL != "" {
		return fmt.Sprintf("%s/cloud-media/%s", h.serverURL, mediaID), nil
	}
	if h.mediaPresigner == nil {
		return "", fmt.Errorf("no media backend configured")
	}
	key := fmt.Sprintf("chatwoot/%s/%s", sessionID, mediaID)
	return h.mediaPresigner.GetPresignedURL(ctx, key)
}

// DownloadCloudMedia streams the raw bytes of a Cloud-API media object to the
// caller (Chatwoot). It is registered WITHOUT authentication because:
//
//   - The media id itself is a high-entropy WhatsApp stanza id that is only
//     known to the real sender/recipient and the Chatwoot worker.
//   - Chatwoot always sends `Authorization: Bearer <api_key>` on this call,
//     which we must accept silently (NEVER 401, see Reauthorizable note).
func (h *CloudAPIHandler) DownloadCloudMedia(c *fiber.Ctx) error {
	mediaID := c.Params("media_id")
	if mediaID == "" {
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}
	if h.mediaStorage == nil || h.msgRepo == nil {
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}

	sessionID, err := h.msgRepo.FindSessionByMessageID(c.Context(), mediaID)
	if err != nil || sessionID == "" {
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}

	key := fmt.Sprintf("chatwoot/%s/%s", sessionID, mediaID)
	contentType, size, userMeta, err := h.mediaStorage.StatMeta(c.Context(), key)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("key", key).Msg("DownloadCloudMedia: stat failed")
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}

	reader, err := h.mediaStorage.Download(c.Context(), key)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("key", key).Msg("DownloadCloudMedia: storage read failed")
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}
	defer func() { _ = reader.Close() }()

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Set("Content-Type", contentType)
	if size > 0 {
		c.Set("Content-Length", fmt.Sprintf("%d", size))
	}
	// Preserva o nome original do arquivo no header Content-Disposition.
	// Sem isso, o Chatwoot salva o anexo usando o próprio mediaID como nome
	// (o último segmento da URL), o que aparece como um ID aleatório feio
	// na UI em vez do nome real do arquivo (ex.: "report.pdf").
	if filename := pickFilenameFromMeta(userMeta); filename != "" {
		c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, sanitizeFilename(filename)))
	}
	_, err = io.Copy(c.Response().BodyWriter(), reader)
	return err
}

// pickFilenameFromMeta extrai o filename do metadata (case-insensitive — alguns
// backends normalizam as keys para lower, outros preservam case).
func pickFilenameFromMeta(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	for _, k := range []string{"filename", "Filename", "FileName", "X-Amz-Meta-Filename"} {
		if v, ok := meta[k]; ok && v != "" {
			return v
		}
	}
	for k, v := range meta {
		if strings.EqualFold(k, "filename") && v != "" {
			return v
		}
	}
	return ""
}

// sanitizeFilename remove aspas e caracteres de controle que quebrariam o
// header Content-Disposition.
func sanitizeFilename(name string) string {
	r := strings.NewReplacer(`"`, "", "\r", "", "\n", "", "\\", "")
	return r.Replace(name)
}

// inferMimeType devolve o primeiro mime não vazio entre o explícito, a
// extensão do filename e a extensão da URL. Usa o stdlib `mime.TypeByExtension`
// (que cobre os formatos comuns). Retorna "" quando não consegue resolver —
// whatsmeow aceita mime vazio para alguns tipos e detecta via magic bytes.
func inferMimeType(explicit, link, filename string) string {
	if explicit != "" {
		return explicit
	}
	if filename != "" {
		if mt := mime.TypeByExtension(strings.ToLower(path.Ext(filename))); mt != "" {
			return mt
		}
	}
	if link != "" {
		if u, err := url.Parse(link); err == nil {
			if mt := mime.TypeByExtension(strings.ToLower(path.Ext(u.Path))); mt != "" {
				return mt
			}
		}
	}
	return ""
}

func (h *CloudAPIHandler) handleTextSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
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

	h.markOutboundWAID(c.Context(), sessionID, msgID)
	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudAPIHandler) handleMediaSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to, mediaType string) error {
	var media *dto.CloudAPIMedia
	switch mediaType {
	case "image":
		media = req.Image
	case "video":
		media = req.Video
	case "audio":
		media = req.Audio
	case "sticker":
		media = req.Sticker
	}
	if media == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing '"+mediaType+"' field", "OAuthException", 131000))
	}
	if media.Link == "" {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing '"+mediaType+".link' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)
	link := rewriteCloudAssetURL(media.Link, cfg.URL)

	sendReq := dto.SendMediaReq{
		Phone:    to,
		URL:      link,
		MimeType: inferMimeType(media.MimeType, link, ""),
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
	case "sticker":
		// whatsmeow não expõe SendSticker via URL; usamos o caminho de
		// documento preservando o mime type para render correto no cliente.
		msgID, err = h.messageSvc.SendDocument(c.Context(), sessionID, sendReq)
	}

	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send media")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	h.markOutboundWAID(c.Context(), sessionID, msgID)
	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudAPIHandler) handleDocumentSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if req.Document == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'document' field", "OAuthException", 131000))
	}
	if req.Document.Link == "" {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'document.link' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)
	link := rewriteCloudAssetURL(req.Document.Link, cfg.URL)

	sendReq := dto.SendMediaReq{
		Phone:    to,
		URL:      link,
		MimeType: inferMimeType(req.Document.MimeType, link, req.Document.Filename),
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

	h.markOutboundWAID(c.Context(), sessionID, msgID)
	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudAPIHandler) handleLocationSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
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

	h.markOutboundWAID(c.Context(), sessionID, msgID)
	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudAPIHandler) handleContactSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
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

		msgID, err := h.messageSvc.SendContact(c.Context(), sessionID, sendReq)
		if err != nil {
			logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send contact")
			return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
		}
		h.markOutboundWAID(c.Context(), sessionID, msgID)
	}

	return c.JSON(cloudAPISuccess(normalizePhone(req.To), ""))
}

func (h *CloudAPIHandler) handleReactionSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
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

	h.markOutboundWAID(c.Context(), sessionID, msgID)
	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudAPIHandler) handleTemplateSend(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq, to string) error {
	if req.Template == nil {
		return c.Status(fiber.StatusBadRequest).JSON(cloudAPIError("Missing 'template' field", "OAuthException", 131000))
	}

	sessionID := resolveSessionIDFromConfig(cfg)

	body := ""
	for _, comp := range req.Template.Components {
		if comp.Type == "body" || comp.Type == "BODY" {
			for _, p := range comp.Parameters {
				if p.Text != "" {
					body = p.Text
					break
				}
			}
		}
	}
	if body == "" {
		body = req.Template.Name
	}

	sendReq := dto.SendTextReq{
		Phone: to,
		Body:  body,
	}

	msgID, err := h.messageSvc.SendText(c.Context(), sessionID, sendReq)
	if err != nil {
		logger.Warn().Str("component", "handler").Err(err).Str("session", sessionID).Msg("Cloud API: failed to send template")
		return c.Status(fiber.StatusInternalServerError).JSON(cloudAPIError("internal server error", "OAuthException", 131000))
	}

	if h.windowUnlocker != nil {
		h.windowUnlocker.UnlockCloudWindow(c.Context(), cfg, to)
	}

	h.markOutboundWAID(c.Context(), sessionID, msgID)
	return c.JSON(cloudAPISuccess(normalizePhone(req.To), msgID))
}

func (h *CloudAPIHandler) handleMarkRead(c *fiber.Ctx, cfg *chatwoot.Config, req dto.CloudAPIMessageReq) error {
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

func (h *CloudAPIHandler) warnTokenMismatch(c *fiber.Ctx, expectedToken string) {
	token := h.extractAccessToken(c)

	if expectedToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
		logger.Warn().Str("component", "handler").Str("path", c.Path()).Msg("Cloud API: bearer token mismatch (ignored)")
	}
}

// extractAccessToken extracts the token from the Authorization header
// (supports "Bearer " prefix) or the access_token query string.
func (h *CloudAPIHandler) extractAccessToken(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		token = strings.TrimPrefix(authHeader, "bearer ")
	}
	if token == "" {
		token = c.Query("access_token")
	}
	return token
}

func (h *CloudAPIHandler) resolveConfigByPhone(ctx context.Context, phone string) (*chatwoot.Config, error) {
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

func (h *CloudAPIHandler) markOutboundWAID(ctx context.Context, sessionID, msgID string) {
	if h.idempotency == nil || sessionID == "" || msgID == "" {
		return
	}
	h.idempotency.MarkSourceIDIdempotent(ctx, sessionID, "WAID:"+msgID)
}

func normalizePhone(phone string) string {
	return strings.TrimLeft(strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone), "0")
}

// isNumeric reports whether s is a non-empty sequence of ASCII digits.
// Used to distinguish phone numbers from WhatsApp media ids in shared routes.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func rewriteCloudAssetURL(rawURL, chatwootBaseURL string) string {
	if rawURL == "" || chatwootBaseURL == "" {
		return rawURL
	}

	assetURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	baseURL, err := url.Parse(strings.TrimRight(chatwootBaseURL, "/"))
	if err != nil {
		return rawURL
	}

	if assetURL.Host == "" {
		assetURL.Scheme = baseURL.Scheme
		assetURL.Host = baseURL.Host
		return assetURL.String()
	}

	if strings.EqualFold(assetURL.Hostname(), "localhost") || assetURL.Hostname() == "127.0.0.1" || assetURL.Hostname() == "::1" {
		assetURL.Scheme = baseURL.Scheme
		assetURL.Host = baseURL.Host
	}

	return assetURL.String()
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
		Contacts: []dto.CloudAPIContactRef{
			{Input: to, WaID: to},
		},
		Messages: []dto.CloudAPIMsgRef{},
	}
	if msgID != "" {
		resp.Messages = append(resp.Messages, dto.CloudAPIMsgRef{ID: msgID})
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
