package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"wzap/internal/logger"
)

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		var ev *zerolog.Event
		switch {
		case status >= 500:
			ev = logger.Error()
		case status >= 400:
			ev = logger.Warn()
		default:
			ev = logger.Info()
		}

		ev.
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Str("latency", duration.String()).
			Str("ip", c.IP()).
			Msg("HTTP Request")

		return err
	}
}
