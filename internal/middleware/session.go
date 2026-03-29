package middleware

import (
	"wzap/internal/model"
	"wzap/internal/repository"

	"github.com/gofiber/fiber/v2"
)

func RequiredSession(sessionRepo *repository.SessionRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionName := c.Params("sessionName")
		if sessionName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "sessionName is required in URL path"))
		}

		session, err := sessionRepo.FindByName(c.Context(), sessionName)
		if err != nil {
			session, err = sessionRepo.FindByID(c.Context(), sessionName)
			if err != nil {
				return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", "Session not found"))
			}
		}

		if c.Locals("authRole") == "session" {
			if c.Locals("sessionId") != session.ID {
				return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Token not authorized for this session"))
			}
		}

		c.Locals("sessionId", session.ID)
		return c.Next()
	}
}
