package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
			ev = log.Error()
		case status >= 400:
			ev = log.Warn()
		default:
			ev = log.Info()
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
