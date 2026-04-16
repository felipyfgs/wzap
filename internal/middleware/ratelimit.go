package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"wzap/internal/dto"
)

// RateLimit applies per-IP rate limiting. This middleware runs before auth,
// so session-scoped keys are not available here.
func RateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(dto.ErrorResp("Rate Limit", "Too many requests. Please slow down."))
		},
	})
}
