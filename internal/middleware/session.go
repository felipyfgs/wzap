package middleware

import (
	"wzap/internal/dto"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

func RequiredSession(sessionRepo *repo.SessionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Params("sessionId")
		if sessionID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "sessionId is required in URL path"))
		}

		session, err := sessionRepo.FindByName(c.Context(), sessionID)
		if err != nil {
			session, err = sessionRepo.FindByID(c.Context(), sessionID)
			if err != nil {
				return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "Session not found"))
			}
		}

		if c.Locals("authRole") == "session" {
			if c.Locals("sessionID") != session.ID {
				return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Token not authorized for this session"))
			}
		}

		c.Locals("sessionID", session.ID)
		return c.Next()
	}
}
