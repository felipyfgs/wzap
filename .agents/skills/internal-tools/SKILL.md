---
name: internal-tools
description: Build or extend admin-only endpoints in wzap — session management, health monitoring, bulk operations, and operational dashboards restricted to the admin role. Use when the audience is an operator or engineer managing the wzap instance, not an end-user calling the WhatsApp API.
---

Build admin-only HTTP endpoints in wzap. Admin routes register under `grp` (not `sess`) in `internal/server/router.go`.

## Step 1 — Add admin guard

Every admin handler starts with:

```go
if c.Locals("authRole") != "admin" {
    return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
}
```

## Step 2 — Define DTO

File: `internal/dto/<domain>.go`. Follow standard DTO conventions (see service-integration skill).

## Step 3 — Implement service

File: `internal/service/<domain>.go`. Log every state-changing action:

```go
logger.Info().Str("session", id).Str("action", "force-disconnect").Msg("admin action")
```

Destructive operations must be preceded by a read to confirm the target exists.

## Step 4 — Write handler

File: `internal/handler/<domain>.go`. Include Swagger godoc with `@Description ... (Admin Only)`.

## Step 5 — Register route under `grp`

```go
// In internal/server/router.go
grp.Post("/sessions", sessionHandler.Create)  // admin-only
grp.Get("/sessions", sessionHandler.List)     // admin-only
```

Never put admin routes under `sess` (session-scoped group).

## Step 6 — Verify

```bash
go build ./...
go test -v -race ./...
golangci-lint run ./...
```

Manual verification:
- Admin `ADMIN_TOKEN` → operation succeeds.
- Session `sk_*` key → `403 Forbidden`.
- No key → `401 Unauthorized`.
- No `token` or `secret` in response body.

## Auth model

Read [auth-model.md](auth-model.md) for full details. Two roles:

| Role | How obtained | `c.Locals("authRole")` |
|---|---|---|
| `admin` | `Authorization` header == `cfg.AdminToken` (env `ADMIN_TOKEN`) | `"admin"` |
| `session` | `Authorization` header == session's `token` | `"session"` |

Header is `Authorization`, no "Bearer" prefix — raw token comparison.

Read [operations-checklist.md](operations-checklist.md) for env vars, session lifecycle, health check, and common failure patterns.

## Gotchas

- If `cfg.AdminToken` is empty, Auth middleware returns `503 Misconfigured` — never deploy with empty `ADMIN_TOKEN`.
- Never return `token` or `secret` in list responses — mask or omit.
- Session deletion and bulk disconnect are irreversible — log at Info level before executing.
- `wz_webhooks` has `ON DELETE CASCADE` on `session_id` — deleting a session removes all its webhooks.
