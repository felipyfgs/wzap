package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	mw "wzap/internal/middleware"
)

// Ensure Validate is initialized at package load
var _ = mw.Validate

func getSessionID(c *fiber.Ctx) (string, error) {
	val, ok := c.Locals("sessionID").(string)
	if !ok || val == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "session identification is required")
	}
	return val, nil
}

func mustGetSessionID(c *fiber.Ctx) string {
	val, _ := c.Locals("sessionID").(string)
	return val
}

func parseAndValidate(c *fiber.Ctx, req interface{}) error {
	if err := c.BodyParser(req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
		return fiber.ErrBadRequest
	}

	if err := mw.Validate.Struct(req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var msgs []string
			for _, e := range validationErrors {
				msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s'", e.Field(), e.Tag()))
			}
			_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", strings.Join(msgs, "; ")))
		} else {
			_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Validation Error", err.Error()))
		}
		return fiber.ErrBadRequest
	}
	return nil
}
