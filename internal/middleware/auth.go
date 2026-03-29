package middleware

import (
	"strings"

	"wzap/internal/config"
	"wzap/internal/dto"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.APIKey == "" {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			authHeader = c.Get("Token")
		}

		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing Authorization or Token header"))
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		if token == cfg.APIKey {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		session, err := sessionRepo.FindByAPIKey(c.Context(), token)
		if err == nil {
			c.Locals("authRole", "session")
			c.Locals("sessionId", session.ID)
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid API Key or Session Token"))
	}
}
