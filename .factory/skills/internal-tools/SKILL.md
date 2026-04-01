---
name: internal-tools
description: Build or extend admin and operational endpoints in wzap that are restricted to the admin role — session management, health monitoring, bulk operations, and operational dashboards. Use when the audience is an operator or engineer managing the wzap instance, not an end-user calling the WhatsApp API.
---
# Skill: Internal tools — wzap

## Purpose

Build or extend admin-only HTTP endpoints, operational scripts, or monitoring surfaces in wzap, while enforcing the existing `"admin"` role gate, keeping `apiKey` and `secret` fields secure, and leaving every state change traceable in zerolog output.

## When to use this skill

- Adding admin-only routes (session provisioning, bulk disconnect, status dashboard).
- Building operational tooling that reads or writes `"wzSessions"` or `"wzWebhooks"` with elevated privileges.
- Extending the `/health` endpoint or adding metrics/diagnostics endpoints.
- Creating scripts or API clients that automate session lifecycle management.

## Auth model summary

wzap has **two roles** — see `auth-model.md` for full details:

| Role | How obtained | `c.Locals("authRole")` |
|---|---|---|
| `admin` | `ApiKey` header == `cfg.APIKey` (env `API_KEY`) | `"admin"` |
| `session` | `ApiKey` header == a session's `apiKey` (`sk_*`) | `"session"` |

**Admin guard pattern** (place at top of every admin-only handler body):
```go
if c.Locals("authRole") != "admin" {
    return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "Admin access required"))
}
```

## Inputs

- **Operator persona**: engineer, SRE, or support staff using the tool.
- **Workflow**: what action needs to happen (create session, force disconnect, list all sessions, etc.).
- **Risk level**: read-only (low), state change (medium), irreversible (high — requires explicit confirmation).
- **Data touched**: which tables and which sensitive fields are involved.

## Out of scope

- Endpoints accessible by session-role callers (those belong to the standard API layer).
- Changes that bypass the `Auth` middleware.
- New identity providers or SSO — use the existing `API_KEY` env variable mechanism.

## Conventions

- Use `internal/handler/`, `internal/service/`, `internal/repo/` — do not put business logic in the route registration file.
- Admin routes register under `grp` (not `sess`) in `internal/server/router.go`.
- Never return `"apiKey"` or `"secret"` in list responses — mask or omit them.
- Log every state-changing admin action: `log.Info().Str("session", id).Str("action", "force-disconnect").Msg("admin action")`.
- Destructive operations (delete, bulk disconnect) must be preceded by a read to confirm the target exists.

## Required artifacts

- Handler in `internal/handler/<domain>.go` with Swagger godoc.
- Service method in `internal/service/<domain>.go`.
- Repo method if DB access is needed (follow `data-querying/query-patterns.md`).
- Route registered in `internal/server/router.go` under `grp`.

## Implementation checklist

1. Confirm the operation is admin-only and add the role guard at the top of the handler.
2. Define or reuse a DTO in `internal/dto/`.
3. Implement the service method; log the action with zerolog before executing it.
4. If DB access is needed, add a typed repo method following `data-querying/query-patterns.md`.
5. Register the route under `grp` in `router.go`.
6. Verify that `apiKey` and `secret` are not returned in any response payload.
7. Run verification commands.

## Verification

```bash
go build ./...
go test -v -race ./...
golangci-lint run ./...
```

Manual verification:
- Call the endpoint with the admin `API_KEY` — expect the operation to succeed.
- Call it with a session `sk_*` key — expect `403 Forbidden`.
- Call it with no key — expect `401 Unauthorized`.

The skill is complete when:

- All three auth scenarios return the correct status codes.
- The zerolog output shows the action with the relevant IDs.
- No `apiKey` or `secret` appears in the response body.

## Safety and escalation

- **Bulk disconnects** and **session deletions** are irreversible from the database perspective — log them at `Info` level before execution and confirm the session exists first.
- If `cfg.APIKey` is empty (dev mode), the `Auth` middleware grants admin to every caller — never deploy to production with an empty `API_KEY`.
- If a bug allows session-role callers to reach an admin endpoint, treat it as a security incident and patch immediately.

## Companion files

- `auth-model.md` — how `middleware.Auth` works, role locals, and the `RequiredSession` middleware.
- `operations-checklist.md` — health endpoint, session statuses, env vars, and `make` commands.
