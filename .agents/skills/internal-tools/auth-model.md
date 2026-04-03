# Auth Model — wzap

Two roles derived from the `Authorization` header. No JWT, no OAuth, no session cookie.

## middleware.Auth

File: `internal/middleware/auth.go`

```go
token := c.Get("Authorization")  // raw header, NO "Bearer" prefix

// 1. Admin: token == cfg.AdminToken (env ADMIN_TOKEN)
c.Locals("authRole", "admin")

// 2. Session: sessionRepo.FindByToken(ctx, token) — matches token column in wz_sessions
c.Locals("authRole", "session")
c.Locals("sessionID", session.ID)
```

If `cfg.AdminToken` is empty → returns `503 Misconfigured`.

## Roles

| Scenario | `authRole` | `sessionID` |
|---|---|---|
| `ADMIN_TOKEN` env empty (dev) | `503 Misconfigured` | — |
| Header matches `cfg.AdminToken` | `"admin"` | not set |
| Header matches session `token` | `"session"` | session UUID |
| No/invalid header | — | 401 returned |

## Admin guard

```go
if c.Locals("authRole") != "admin" {
    return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
}
```

Currently admin-only routes: `POST /sessions`, `GET /sessions`.

## RequiredSession middleware

File: `internal/middleware/session.go`

Resolves `:sessionId` (name OR UUID):
1. Try `sessionRepo.FindByName(ctx, param)`.
2. Fallback `sessionRepo.FindByID(ctx, param)`.
3. Row-level security: if `authRole == "session"`, token's session ID must match resolved session.
4. Overwrites `c.Locals("sessionID", session.ID)` — always canonical UUID.

## Route groups

```go
grp := s.App.Group("/", middleware.Auth(s.Config, sessionRepo))

// Admin-only
grp.Post("/sessions", sessionHandler.Create)
grp.Get("/sessions", sessionHandler.List)

// Session-scoped
reqSession := middleware.RequiredSession(sessionRepo)
sess := grp.Group("/sessions/:sessionId", reqSession)
sess.Get("/", sessionHandler.Get)
sess.Post("/messages/text", messageHandler.SendText)
// ... all session-scoped routes
```

New admin routes → `grp`. New session-scoped routes → `sess`.

## Security invariants

1. `token` from `wz_sessions` must never appear in response bodies or log lines.
2. Auth middleware must be on every route except `/health` and `/swagger/*`.
3. Session-role callers can only access their own session — `RequiredSession` enforces this.
