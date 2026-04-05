package handler

import (
	"crypto/subtle"

	"wzap/internal/config"
	"wzap/internal/dto"
	"wzap/internal/logger"
	wsHub "wzap/internal/websocket"

	ws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type WebSocketHandler struct {
	hub *wsHub.Hub
	cfg *config.Config
}

func NewWebSocketHandler(hub *wsHub.Hub, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{hub: hub, cfg: cfg}
}

func (h *WebSocketHandler) Upgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if ws.IsWebSocketUpgrade(c) {
			var token string
			switch h.cfg.WSAuthMode {
			case "header":
				token = c.Get("Authorization")
			case "subprotocol":
				token = c.Get("Sec-WebSocket-Protocol")
			default:
				token = c.Query("token")
				if token == "" {
					token = c.Get("Authorization")
				}
			}
			if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(h.cfg.AdminToken)) != 1 {
				return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid or missing token"))
			}
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

func (h *WebSocketHandler) Handle() func(*ws.Conn) {
	return func(c *ws.Conn) {
		sessionID := c.Params("sessionId", "*")

		h.hub.Register(sessionID, c)
		defer h.hub.Unregister(sessionID, c)

		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				logger.Debug().Err(err).Str("session", sessionID).Msg("WebSocket read error")
				break
			}
		}
	}
}
