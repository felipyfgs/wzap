package middleware

import (
	"fmt"
	"runtime/debug"

	"wzap/internal/dto"
	"wzap/internal/logger"

	"github.com/gofiber/fiber/v2"
)

func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				logger.Error().
					Str("component", "http").
					Str("method", c.Method()).
					Str("path", c.Path()).
					Str("ip", c.IP()).
					Err(err).
					Str("stack", string(debug.Stack())).
					Msg("panic recovered")

				_ = c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", "internal server error"))
			}
		}()
		return c.Next()
	}
}
