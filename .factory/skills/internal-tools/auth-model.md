# Auth Model — wzap

wzap uses a single HTTP header (`ApiKey`) with two distinct roles derived from the token value. There is no JWT, no OAuth, and no session cookie.

---

## middleware.Auth — how it works

File: `internal/middleware/auth.go`

```go
func Auth(cfg *config.Config, sessionRepo *repo.SessionRepository) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Dev mode: no API_KEY configured → grant admin to everyone
        if cfg.APIKey == "" {
            c.Locals("authRole", "admin")
            return c.Next()
        }

        token := c.Get("ApiKey")
        if token == "" {
            return c.Status(401).JSON(dto.ErrorResp("Unauthorized", "Missing ApiKey header"))
        }

        // Global admin token (env API_KEY)
        if token == cfg.APIKey {
            c.Locals("authRole", "admin")
            return c.Next()
        }

        // Per-session token (sk_<uuid> stored in "wzSessions"."apiKey")
        session, err := sessionRepo.FindByAPIKey(c.Context(), token)
        if err == nil {
            c.Locals("authRole", "session")
            c.Locals("sessionId", session.ID)
            return c.Next()
        }

        return c.Status(401).JSON(dto.ErrorResp("Unauthorized", "Invalid token"))
    }
}
```

---

## Roles summary

| Scenario | `c.Locals("authRole")` | `c.Locals("sessionId")` |
|---|---|---|
| `API_KEY` env empty (dev) | `"admin"` | not set |
| Header matches `cfg.APIKey` | `"admin"` | not set |
| Header matches a session `apiKey` | `"session"` | session UUID |
| No header / unknown token | — | 401 returned |

---

## Admin guard pattern

Use this check at the top of any admin-only handler body:

```go
func (h *SessionHandler) Create(c *fiber.Ctx) error {
    if c.Locals("authRole") != "admin" {
        return c.Status(fiber.StatusForbidden).JSON(
            dto.ErrorResp("Forbidden", "Admin access required"),
        )
    }
    // ... rest of handler
}
```

Currently admin-only routes:
- `POST /sessions` — create a new session
- `GET /sessions` — list all sessions

---

## RequiredSession middleware

File: `internal/middleware/session.go`

Runs after `Auth` on all session-scoped routes (`/sessions/:sessionId/*`). It resolves `:sessionId` (which can be the session UUID **or** the session name) to a canonical UUID and stores it in `c.Locals("sessionId")`.

```go
// Reading the resolved session ID inside a handler:
id := c.Locals("sessionId").(string)
```

Callers with the admin token (`authRole == "admin"`) can access any session ID. Callers with a session token (`authRole == "session"`) can only access the session whose `apiKey` they provided — the middleware enforces this automatically.

---

## Route groups in router.go

```go
// Auth applied to all routes in this group
grp := s.App.Group("/", middleware.Auth(s.Config, sessionRepo))

// Admin-only — no session resolution
grp.Post("/sessions", sessionHandler.Create)
grp.Get("/sessions", sessionHandler.List)

// Session-scoped — RequiredSession resolves :sessionId
reqSession := middleware.RequiredSession(sessionRepo)
sess := grp.Group("/sessions/:sessionId", reqSession)

sess.Get("/", sessionHandler.Get)
sess.Delete("/", sessionHandler.Delete)
// ... all other session-scoped routes
```

**Rule:** new admin-only routes go under `grp`; new session-scoped routes go under `sess`.

---

## Environment variables

| Variable | Effect |
|---|---|
| `API_KEY=""` | Dev mode — every caller is admin; **never use in production** |
| `API_KEY="<secret>"` | Only callers presenting this exact value are admin |

The API key is loaded in `internal/config/config.go` via `getEnv("API_KEY", "")`.

---

## Security invariants

1. `apiKey` values from `"wzSessions"` must **never** appear in response bodies of list endpoints or in log lines.
2. The `Auth` middleware must be applied to every route — the only exceptions are `/health` and `/swagger/*`.
3. A session-role caller must only be able to operate on their own session — `RequiredSession` enforces this by comparing the resolved session ID against `c.Locals("sessionId")`.
