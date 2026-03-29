package middleware

import (
	"wzap/internal/config"
	"wzap/internal/dto"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.APIToken == "" {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		token := c.Get("Token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing Token header"))
		}

		if token == cfg.APIToken {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		session, err := sessionRepo.FindByToken(c.Context(), token)
		if err == nil {
			c.Locals("authRole", "session")
			c.Locals("sessionId", session.ID)
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid API Key or Session Token"))
	}
}
