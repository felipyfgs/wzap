package handler

import "github.com/gofiber/fiber/v2"

func getSessionID(c *fiber.Ctx) (string, error) {
	if val := c.Locals("sessionId"); val != nil {
		return val.(string), nil
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
}
