package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"wzap/internal/config"
	"wzap/internal/model"
	"wzap/internal/repository"
)

func Auth(cfg *config.Config, repo *repository.SessionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If both are empty, no auth
		if cfg.APIKey == "" {
			c.Locals("auth_role", "admin")
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			authHeader = c.Get("Token")
		}

		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResp("Unauthorized", "Missing Authorization or Token header"))
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		// 1. Check Global Admin Key
		if token == cfg.APIKey {
			c.Locals("auth_role", "admin")
			return c.Next()
		}

		// 2. Check Session-specific Key
		session, err := repo.FindByAPIKey(c.Context(), token)
		if err == nil {
			c.Locals("auth_role", "session")
			c.Locals("session_id", session.ID)

			// Security check: if :id is in URL, it MUST match token's session ID
			pathID := c.Params("id")
			if pathID != "" && pathID != session.ID {
				return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Token not authorized for this session ID"))
			}

			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResp("Unauthorized", "Invalid API Key or Session Token"))
	}
}
