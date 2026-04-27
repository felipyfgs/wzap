package middleware

import (
	"crypto/subtle"
	"regexp"
	"strings"

	"wzap/internal/config"
	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/repo"

	"github.com/gofiber/fiber/v2"
)

// cloudAPIPathRegex matches WhatsApp Cloud API paths (e.g. /v1.0/12345).
// These paths bypass auth intentionally — Meta sends requests directly here.
// Ensure no internal handlers are registered under this pattern.
var cloudAPIPathRegex = regexp.MustCompile(`^/v\d+\.\d+/(?:\d+|debug_token)`)

// stripBearer remove o prefixo "Bearer " (case-insensitive) do header. Usa
// HasPrefix em vez de Index para não casar a substring no meio do header
// (ex.: "X-fake bearer realtoken" não deve virar "realtoken").
func stripBearer(token string) string {
	const prefix = "bearer "
	if len(token) >= len(prefix) && strings.EqualFold(token[:len(prefix)], prefix) {
		token = token[len(prefix):]
	}
	return strings.TrimSpace(token)
}

func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
	if cfg.AdminToken == "" {
		logger.Warn().Str("component", "http").Msg("ADMIN_TOKEN not set: all requests will be rejected")
	}
	return func(c *fiber.Ctx) error {
		if cloudAPIPathRegex.MatchString(c.Path()) {
			return c.Next()
		}

		if cfg.AdminToken == "" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResp("Misconfigured", "ADMIN_TOKEN is not set"))
		}

		rawToken := c.Get("Authorization")
		if rawToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Missing Authorization header"))
		}

		token := stripBearer(rawToken)

		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.AdminToken)) == 1 {
			c.Locals("authRole", "admin")
			return c.Next()
		}

		session, err := sessionRepo.FindByToken(c.Context(), token)
		if err == nil {
			c.Locals("authRole", "session")
			c.Locals("sessionID", session.ID)
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResp("Unauthorized", "Invalid token"))
	}
}
