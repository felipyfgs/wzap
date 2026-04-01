package handler

import (
	"wzap/internal/config"
	"wzap/internal/logger"
	wsHub "wzap/internal/websocket"

	"github.com/gofiber/fiber/v2"
	ws "github.com/gofiber/contrib/websocket"
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
			token := c.Query("token")
			if token == "" {
				token = c.Get("Authorization")
			}
			if token == "" || token != h.cfg.APIKey {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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
