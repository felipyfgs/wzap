package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"wzap/internal/config"
	"wzap/internal/model"
	"wzap/internal/repository"
)

func Auth(cfg *config.Config, userRepo *repository.UserRepository) fiber.Handler {
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
			return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResp("Unauthorized", "Missing Authorization or Token header"))
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		if token == cfg.APIKey {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		user, err := userRepo.FindByToken(c.Context(), token)
		if err == nil {
			c.Locals("authRole", "user")
			c.Locals("userId", user.ID)

			pathID := c.Params("id")
			if pathID != "" && pathID != user.ID {
				return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Token not authorized for this user ID"))
			}

			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResp("Unauthorized", "Invalid API Key or User Token"))
	}
}
