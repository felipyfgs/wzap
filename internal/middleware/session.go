package middleware

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/repository"
)

func RequiredSession(sessionRepo *repository.SessionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		val := c.Locals("userId")
		if val == nil || val.(string) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResp("Unauthorized", "User token required in Authorization or Token header"))
		}

		userID := val.(string)
		session, err := sessionRepo.FindByUserID(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", "No session found for this user"))
		}

		c.Locals("sessionId", session.ID)
		return c.Next()
	}
}
