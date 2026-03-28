package middleware

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
)

// RequiredSession ensures that a valid session ID is present in the request.
// It relies completely on the Auth middleware parsing the session.
func RequiredSession() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Enforce that the Auth middleware successfully matched the Bearer token to a session
		if val := c.Locals("session_id"); val != nil && val.(string) != "" {
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResp("Unauthorized", "Session token required in Authorization or Token header"))
	}
}
