package middleware

import (
	"wzap/internal/config"
	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
	if cfg.AdminToken == "" {
		logger.Warn().Msg("ADMIN_TOKEN not set: all requests will be rejected")
	}
	return func(c *fiber.Ctx) error {
		if cfg.AdminToken == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResp("Misconfigured", "ADMIN_TOKEN is not set"))
		}

		token := c.Get("Authorization")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing Authorization header"))
		}

		if token == cfg.AdminToken {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		session, err := sessionRepo.FindByToken(c.Context(), token)
		if err == nil {
			c.Locals("authRole", "session")
			c.Locals("sessionID", session.ID)
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid token"))
	}
}
