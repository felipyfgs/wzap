package middleware

import (
	"wzap/internal/config"
	"wzap/internal/dto"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
	if cfg.APIKey == "" {
		log.Warn().Msg("API_KEY not set: all requests will be rejected")
	}
	return func(c *fiber.Ctx) error {
		if cfg.APIKey == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResp("Misconfigured", "API_KEY is not set"))
		}

		token := c.Get("ApiKey")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing ApiKey header"))
		}

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

		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid token"))
	}
}
