---
name: service-integration
description: Add or modify HTTP endpoints, WhatsApp capabilities, webhook events, or repo queries in wzap. Use when adding a new route, a service method, a whatsmeow feature, or a repo query — following the handler/service/repo layering with Fiber + whatsmeow patterns.
---

Add or modify backend capabilities in wzap. Follow the step-by-step workflow below.

## Step 1 — Identify the layer

| If you need to... | Start at |
|---|---|
| Add a new REST endpoint | DTO → Handler → Route |
| Expose a whatsmeow feature | Service → Handler → Route |
| Add a DB query | Repo → (Service) → Handler |
| Emit a new event type | model/events.go → wa/events.go |

Read [patterns.md](patterns.md) for copy-paste code snippets. Read [whatsmeow-integration.md](whatsmeow-integration.md) for the wa.Manager API.

## Step 2 — Create or update the DTO

File: `internal/dto/<domain>.go`

- Request DTOs: `<Action><Resource>Req` with `validate:"required"` tags.
- Response DTOs: `<Resource>Resp`.
- JSON tags: `camelCase` with `omitempty` on optional fields.
- Update DTOs: pointer fields (`*string`, `*bool`) for partial updates.
- Slice fields: `validate:"required,min=1"` to reject empty arrays.

## Step 3 — Add the repo method (if DB access needed)

File: `internal/repo/<entity>.go`

- Tables are `snake_case`: `wz_sessions`, `wz_webhooks`, `wz_messages`.
- Nullable text columns: `COALESCE(col, '')` in SELECT.
- Parameters: positional `$1`, `$2`, …
- JSONB containment: `events @> $1::jsonb`.
- Wrap errors: `fmt.Errorf("failed to <verb> <noun>: %w", err)`.
- `context.Context` always first param. Receiver is `r`.

## Step 4 — Add the service method

File: `internal/service/<domain>.go`

For WhatsApp operations, follow the GetClient + Guard pattern:

```go
client, err := s.engine.GetClient(sessionID)
if err != nil { return zero, err }
if !client.IsConnected() { return zero, fmt.Errorf("client not connected") }
```

- Send operations return `(string, error)` where string is the WhatsApp message ID.
- Use `parseJID(target)` (unexported, in `service/message.go`) for phone/JID normalization.
- Wrap errors: `fmt.Errorf("failed to <action>: %w", err)`.
- Non-fatal errors: log with `logger.Warn()` but don't return.

## Step 5 — Write the handler

File: `internal/handler/<domain>.go`

```go
func (h *XxxHandler) Method(c *fiber.Ctx) error {
    id, err := getSessionID(c)
    if err != nil { return err }

    var req dto.XxxReq
    if err := parseAndValidate(c, &req); err != nil { return err }

    result, err := h.xxxSvc.Method(c.Context(), id, req)
    if err != nil {
        return c.Status(500).JSON(dto.ErrorResp("Error Title", err.Error()))
    }
    return c.JSON(dto.SuccessResp(result))
}
```

- Admin-only: add `if c.Locals("authRole") != "admin" { return 403 }` at top.
- Swagger godoc: `@Summary`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Security Authorization`, `@Router`.
- `@Router` uses `{sessionId}` with curly braces.

## Step 6 — Register the route

File: `internal/server/router.go`

- Admin routes: `grp.Post("/sessions", ...)`
- Session-scoped: `sess.Post("/messages/text", ...)`

## Step 7 — New event type (if needed)

1. Add constant to `internal/model/events.go` and the `ValidEventTypes` slice.
2. Handle it in `internal/wa/events.go` in the `handleEvent` type switch.

## Step 8 — Verify

```bash
go build ./...
go test -v -race ./...
golangci-lint run ./...
```

## Gotchas

- Table names are `wz_sessions`, `wz_webhooks`, `wz_messages` — all `snake_case`, no double quotes needed.
- `parseAndValidate` already writes the error response and returns `fiber.ErrBadRequest` — just `return err`.
- `getSessionID(c)` returns error for session-scoped routes; `mustGetSessionID(c)` returns empty string (safe when behind `RequiredSession` middleware).
- `Authorization` header has NO "Bearer" prefix — raw token comparison.
- Never log or return `token` or `secret` fields.
- Import groups: stdlib → blank line → third-party → blank line → `wzap/internal/...`.
