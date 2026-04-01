---
name: service-integration
description: Extend or integrate with existing services in the wzap codebase — adding new HTTP endpoints, WhatsApp capabilities, webhook events, or repo methods — while following the established handler/service/repo layering and Fiber + whatsmeow patterns. Use when the task requires adding or modifying a route, a service method, a repo query, or a whatsmeow feature.
---
# Skill: Service integration — wzap

## Purpose

Add or modify backend capabilities in wzap while preserving the layered architecture (`handler → service → repo / wa.Manager`), the Fiber + zerolog conventions, and the session-scoped auth model.

## When to use this skill

- Adding a new HTTP endpoint (message type, contact action, group operation, etc.).
- Exposing a new whatsmeow feature via the REST API.
- Adding or modifying a repo query against `"wzSessions"` or `"wzWebhooks"`.
- Publishing a new event type through the dispatcher to webhooks / NATS.
- Changing the session lifecycle (connect, disconnect, QR flow).

## Layer map

| Layer | Package | Responsibility |
|---|---|---|
| HTTP handler | `internal/handler/` | Parse request, call service, return `dto.SuccessResp` / `dto.ErrorResp` |
| Business logic | `internal/service/` | Orchestrate whatsmeow client + repo calls |
| Data access | `internal/repo/` | Raw pgx SQL against Postgres |
| WA engine | `internal/wa/` | whatsmeow client lifecycle, QR, events |
| Event fan-out | `internal/dispatcher/` | Deliver events to webhooks + NATS subjects |
| DTOs | `internal/dto/` | Request / response payloads (camelCase JSON tags) |
| Domain models | `internal/model/` | Persistent entities (`Session`, `Webhook`, event constants) |
| Routes | `internal/server/router.go` | Fiber group registration + middleware wiring |

## Inputs

- **Feature description**: what the user or caller needs.
- **WhatsApp capability**: which whatsmeow method or event is involved (see `whatsmeow-integration.md`).
- **Auth scope**: admin-only (`cfg.APIKey`) or session-scoped (`sk_*` apiKey).
- **DTO shape**: request and response field names and types.
- **DB change required**: new table, column, or query (see `data-querying/schema.md`).

## Out of scope

- New infrastructure (databases, queues) without an infra PR.
- Changes to the whatsmeow library itself.
- Breaking changes to existing response shapes without a migration plan.

## Conventions

- Handler signature: `func (h *XxxHandler) Method(c *fiber.Ctx) error` — see `patterns.md`.
- Error wrapping: `fmt.Errorf("failed to <action>: %w", err)`.
- HTTP errors: `c.Status(code).JSON(dto.ErrorResp("Title", err.Error()))`.
- Logging: `log.Warn().Err(err).Str("session", id).Msg("...")` — never log `apiKey`.
- Import order: stdlib → third-party → `wzap/internal/...`.
- Session ID from context: `c.Locals("sessionId").(string)` (set by `RequiredSession` middleware).
- Admin guard: `if c.Locals("authRole") != "admin" { return 403 }`.

## Implementation checklist

1. Define or update DTOs in `internal/dto/<domain>.go`.
2. Add the service method in `internal/service/<domain>.go` — call `engine.GetClient(sessionID)` and verify `client.IsConnected()`.
3. If DB access is needed, add a repo method in `internal/repo/<entity>.go` following the pgx patterns in `data-querying/query-patterns.md`.
4. Write the handler in `internal/handler/<domain>.go` with Swagger godoc (`@Summary`, `@Router`, `@Security ApiKey`).
5. Register the route in `internal/server/router.go` under the correct group (`grp` for admin, `sess` for session-scoped).
6. If the feature emits a new event, add the `EventType` constant in `internal/model/events.go` and update `ValidEventTypes`.
7. Run verification commands.

## Verification

```bash
go build ./...
go test -v -race ./...
golangci-lint run ./...
```

The skill is complete when:

- Build and lint pass with zero errors.
- The new endpoint behaves correctly against a local stack (`make up && make dev`).
- Zerolog output for the new path is clean (no unexpected warnings or panics).

## Safety and escalation

- Never log or return the `apiKey` field in any response or log line.
- If a new route bypasses the `Auth` middleware, stop and confirm it is intentional.
- Session deletion / disconnection is irreversible — guard with explicit confirmation in the calling layer.

## Companion files

- `patterns.md` — concrete code snippets for handler, service, and repo layers.
- `whatsmeow-integration.md` — `wa.Manager` API, JID parsing, media upload, event types.
