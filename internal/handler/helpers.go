package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"wzap/internal/dto"
	mw "wzap/internal/middleware"
	"wzap/internal/model"
	"wzap/internal/service"
)

var errResponseSent = errors.New("response already sent")

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

func parseAndValidate(c *fiber.Ctx, req any) error {
	if err := c.BodyParser(req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", err.Error()))
		return errResponseSent
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
		return errResponseSent
	}
	return nil
}

func handleCapabilityError(c *fiber.Ctx, err error) bool {
	var capabilityErr *service.CapabilityError
	if !errors.As(err, &capabilityErr) {
		return false
	}

	title := "Not Supported"
	if capabilityErr.Support == model.SupportPartial {
		title = "Partial Support"
	}

	_ = c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp(title, capabilityErr.Error()))
	return true
}

func handleLifecycleError(c *fiber.Ctx, err error) bool {
	if handleCapabilityError(c, err) {
		return true
	}

	var conflictErr *service.ConflictError
	if errors.As(err, &conflictErr) {
		_ = c.Status(fiber.StatusConflict).JSON(dto.ErrorResp("Conflict", conflictErr.Error()))
		return true
	}

	var notFoundErr *service.NotFoundError
	if errors.As(err, &notFoundErr) {
		_ = c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", notFoundErr.Error()))
		return true
	}

	return false
}
