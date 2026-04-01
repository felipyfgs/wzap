package handler

import "github.com/gofiber/fiber/v2"

func getSessionID(c *fiber.Ctx) (string, error) {
	val, ok := c.Locals("sessionId").(string)
	if !ok || val == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
	}
	return val, nil
}

func mustGetSessionID(c *fiber.Ctx) string {
	val, _ := c.Locals("sessionId").(string)
	return val
}
