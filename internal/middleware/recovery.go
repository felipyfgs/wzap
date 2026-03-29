package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"wzap/internal/dto"
)

func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				log.Error().
					Err(err).
					Str("stack", string(debug.Stack())).
					Msg("panic recovered")

				c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Internal Server Error", err.Error()))
			}
		}()
		return c.Next()
	}
}
