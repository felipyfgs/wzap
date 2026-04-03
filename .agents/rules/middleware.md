# Middleware Rules

## Pattern: Factory Functions Returning `fiber.Handler`

Every middleware is a free function (never a struct method) returning `fiber.Handler`:

```go
func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
    return func(c *fiber.Ctx) error { ... }
}
```

Dependencies are captured via closure at construction time.

## Global Middleware Registration Order (in `server.go`)

```
Recovery → Logger → CORS → RateLimit
```

- Recovery is outermost (catches panics in everything).
- Logger wraps business logic (captures status/latency).
- Rate limiting is innermost among globals.

## Auth Middleware (Two-Tier Token)

```go
token := c.Get("Authorization")  // Raw header, NO "Bearer" prefix
```

1. **Admin**: `token == cfg.AdminToken` → `c.Locals("authRole", "admin")`
2. **Session**: `sessionRepo.FindByToken(ctx, token)` → `c.Locals("authRole", "session")` + `c.Locals("sessionID", session.ID)`

If `cfg.AdminToken` is empty → returns `503 Misconfigured`. Env var: `ADMIN_TOKEN`.

## RequiredSession Middleware

Resolves `:sessionId` param (name OR UUID):

```go
session, err := sessionRepo.FindByName(ctx, sessionID)  // try name first
if err != nil {
    session, err = sessionRepo.FindByID(ctx, sessionID)  // fallback to UUID
}
```

Then overwrites: `c.Locals("sessionID", session.ID)` — always the canonical UUID.

Row-level security: if `authRole == "session"`, the token's session ID must match the resolved session ID.

## Context Locals Contract

| Key | Type | Set By |
|---|---|---|
| `"authRole"` | `string` | `Auth` middleware |
| `"sessionID"` | `string` | `Auth` (for session tokens) + `RequiredSession` (always overwrites) |
| `"allowed"` | `bool` | `WebSocketHandler.Upgrade()` |

Reading `authRole` in handlers:
```go
if c.Locals("authRole") != "admin" { ... }  // direct comparison, no type assertion
```

Reading `sessionID` in handlers:
```go
mustGetSessionID(c)  // safe, returns "" if unset
getSessionID(c)      // returns error if unset
```

## Error Responses

Every middleware error uses `dto.ErrorResp(title, msg)`:

| Middleware | Status | Title |
|---|---|---|
| Auth (no admin token) | 503 | `"Misconfigured"` |
| Auth (invalid token) | 401 | `"Unauthorized"` |
| Recovery | 500 | `"Internal Server Error"` |
| RateLimit | 429 | `"Rate Limit"` |
| RequiredSession (missing param) | 400 | `"Bad Request"` |
| RequiredSession (not found) | 404 | `"Not Found"` |
| RequiredSession (wrong session) | 403 | `"Forbidden"` |

## Logger Middleware

Post-processing: calls `c.Next()` first, then logs method, path, status, latency, IP. Log level based on status code range (500+ → Error, 400+ → Warn, else Info).

## Rate Limiting

- Parameters: `max int, window time.Duration` (currently 120 req/min).
- Key: session ID (from locals) when authenticated, IP otherwise.
- Custom `LimitReached` handler returns consistent error format.

## Recovery Middleware

Uses `defer`/`recover` pattern, logs full stack trace via `debug.Stack()`, coerces non-error panic values with `fmt.Errorf("%v", r)`.

## Global Validator

```go
var Validate = validator.New()  // in middleware/validate.go
```

Singleton instance used by `parseAndValidate` in `handler/helpers.go`. Handler helpers reference it as `mw.Validate`.
