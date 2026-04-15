# AGENTS.md — wzap

## Project overview

**wzap** is a WhatsApp multi-session gateway with a Go REST API backend and a Nuxt 4 SPA dashboard.
It manages multiple WhatsApp sessions via [whatsmeow](https://github.com/tulir/whatsmeow), exposing messaging, contacts, groups, newsletters, labels, media, and webhook/WebSocket event delivery through a unified HTTP API.

> **Sub-project docs**: the frontend has its own `web/AGENTS.md` with Nuxt-specific conventions.

## Tech stack

| Layer         | Technology                                                     |
| ------------- | -------------------------------------------------------------- |
| Language      | **Go 1.25** (module `wzap`)                                    |
| HTTP          | Fiber v2 (`gofiber/fiber`)                                     |
| WhatsApp      | whatsmeow (`go.mau.fi/whatsmeow`) + Cloud API provider        |
| Database      | PostgreSQL 16 via pgx v5 (`jackc/pgx`)                         |
| Migrations    | Embedded SQL files (`migrations/`) applied at startup           |
| Object store  | MinIO (`minio/minio-go`)                                       |
| Message queue | NATS JetStream (`nats-io/nats.go`)                             |
| Cache         | Redis (optional, falls back to in-memory)                      |
| Logging       | zerolog (`rs/zerolog`)                                         |
| Validation    | go-playground/validator v10                                    |
| Docs          | Swagger via swaggo (`/swagger/*`)                              |
| Frontend      | Nuxt 4 SPA — see `web/AGENTS.md`                              |

## Build, lint & test commands

```bash
make dev                # run API with go run
make build              # CGO_ENABLED=0 binary → bin/wzap
make tidy               # go mod tidy
make docs               # regenerate Swagger docs (swag init)
make install-tools      # install golangci-lint v2.11.4 + swag

# Testing
go test -v -race ./...                          # all tests
go test -v -race ./internal/service/...         # single package
go test -v -race -run TestFunctionName ./...    # single test by name
go test -v -race -run TestFunctionName ./internal/service/...  # single test in package

# Linting (no .golangci.yml — uses defaults)
golangci-lint run ./...
```

## Directory structure

```
cmd/wzap/main.go          # Entrypoint: config → DB → NATS → MinIO → Server
internal/
  async/                  # Worker pool (async.Pool, async.Runtime)
  broker/                 # NATS JetStream client
  config/                 # Env-based config (godotenv)
  database/               # pgxpool + embedded migration runner
  dto/                    # Request/response DTOs with validation tags
  handler/                # Fiber HTTP handlers (one file per domain)
  integrations/chatwoot/  # Chatwoot two-way integration
  logger/                 # zerolog singleton
  middleware/              # Auth, rate-limit, recovery, logger, session resolver
  model/                  # Domain models (Session, Message, Webhook, Event envelopes)
  provider/               # WhatsApp Cloud API provider
  repo/                   # PostgreSQL repositories
  server/                 # Fiber app setup (server.go) + route registration (router.go)
  service/                # Business logic (one file per domain)
  storage/                # MinIO client wrapper
  testutil/               # Shared test helpers
  wa/                     # whatsmeow Manager — session lifecycle, events, QR, connect
  webhook/                # Webhook dispatcher (HTTP + NATS + WebSocket broadcast)
  websocket/              # WebSocket hub for real-time events
migrations/               # Numbered SQL migrations (embedded via //go:embed)
docs/                     # Generated Swagger JSON/YAML
web/                      # Nuxt 4 frontend (see web/AGENTS.md)
```

## Code style & conventions

### Architecture

- **Layered**: handler → service → repo. Handlers parse HTTP, services hold business logic, repos talk to the DB.
- **Dependency injection**: `server.New(cfg, db, nats, minio)` → `SetupRoutes()` wires repos → services → handlers. No DI framework, no global state besides the logger.
- **Router structure**: public routes (health, metrics, swagger, WS) have no auth. All `/sessions` routes require auth, with session-scoped routes nested under `/sessions/:sessionId`.

### Imports

Grouped with blank-line separators: (1) standard library, (2) external packages, (3) internal `wzap/internal/...` packages. Use aliases for disambiguation: `cloudWA "wzap/internal/provider/whatsapp"`, `mw "wzap/internal/middleware"`.

### Naming

Go standard — `MixedCaps`, short receiver names (1-2 chars), no `Get` prefix on getters. DTOs use `Req`/`Resp` suffixes (`SendTextReq`, `SessionResp`). Comments may be in Portuguese or English — preserve the original language when editing.

### Error handling

- Wrap with `fmt.Errorf("descriptive context: %w", err)`. Handlers return `dto.ErrorResp(title, message)`.
- Internal errors: generic `"internal server error"` message (never leak details). Client errors (400): may include `err.Error()`.
- Custom error types for domain errors: `CapabilityError`, `LifecycleConflictError`, `LifecycleNotFoundError` (with `Unwrap()`).
- Sentinel errors in repos: `var ErrSessionNotFound = errors.New("session not found")`.
- Use `errors.As` to match custom error types in handlers.

### Handler patterns

```go
func (h *MessageHandler) SendText(c *fiber.Ctx) error {
    id, err := getSessionID(c)
    if err != nil { return err }

    var req dto.SendTextReq
    if err := parseAndValidate(c, &req); err != nil { return err }

    msgID, err := h.svc.SendText(c.Context(), id, req)
    if err != nil { return c.Status(500).JSON(dto.ErrorResp("Send Error", "internal server error")) }

    return c.JSON(dto.SuccessResp(dto.MidResp{Mid: msgID}))
}
```

- Use `parseAndValidate(c, &req)` for body parsing + validation.
- Use `dto.SuccessResp(data)` for success, `c.Status(code).JSON(dto.ErrorResp(title, msg))` for errors.
- Use `c.Status(fiber.StatusCreated)` for resource creation.
- Higher-order handler functions for shared logic (e.g., `sendMedia` for image/video/document/audio).

### Service patterns

- Always `context.Context` as first parameter. `sessionID string` as second for session-scoped ops.
- Generic runtime dispatch for dual-engine (whatsmeow + Cloud API): `runSessionRuntime[T any](...)`.
- Update DTOs use `*string`, `*bool` pointers for optional fields; apply with nil-checks.
- Setter injection for optional dependencies: `SetMessagePersist(fn)`, `SetMediaAutoUpload(fn)`.

### Repo patterns

- Raw SQL with positional params (`$1`, `$2`). Column lists as package-level constants.
- Dedicated `scanXxx(scanner, &m)` functions using a local `xxxScanner` interface for reuse across `pgx.Rows`/`pgx.Row`.
- Always `defer rows.Close()` and check `rows.Err()` after loops.
- Sentinel errors for not-found cases; double-wrap with `fmt.Errorf("%w: %w", ErrNotFound, err)`.

### Models & DTOs

- **Models** (`internal/model/`): plain data structs with `json:"camelCase"` and `json:"camelCase,omitempty"` tags. No validation tags.
- **DTOs** (`internal/dto/`): request DTOs have `validate:"required"` etc. tags; response DTOs do not.
- Typed string constants for domain concepts: `EventType`, `EventCategory`, `EngineCapability`.
- Mapper functions in dto package: `SessionToResp(s model.Session, ...) SessionResp`.

### Logging

Use the `logger` singleton — never `log.Print` or `fmt.Print`. Always include `.Str("component", "xxx")` as first field:

```go
logger.Warn().Str("component", "service").Err(err).Str("session", id).Msg("failed to connect")
logger.Info().Str("component", "server").Str("addr", addr).Msg("Starting API server")
```

### Concurrency

- `wa.Manager` holds a `sync.RWMutex` protecting the `clients` map.
- Use `async.Pool` for background work (webhooks, media, history). Pools drain gracefully on shutdown.
- Context-based runtime caching: `context.WithValue(ctx, runtimeCtxKey{}, r)`.

## Authentication

- **Admin token**: `ADMIN_TOKEN` env var. Sent as `Authorization` header (no `Bearer` prefix). Compared with `crypto/subtle.ConstantTimeCompare`.
- **Session tokens**: each session has its own API key. `middleware.Auth` checks admin first, then session token lookup.
- **Roles**: `admin` (full access) or `session` (scoped to one session via `RequiredSession` middleware).
- **WebSocket auth**: token via query param or `Authorization` header (configurable via `WS_AUTH_MODE`).

## Swagger docs

Every exported handler has godoc/Swagger annotations. Regenerate with `make docs`. The swag command scans `cmd/wzap,internal/handler,internal/dto,internal/model,internal/service,internal/repo` with `--parseInternal --useStructName`.

## Testing

- Test files use `package xxx_test` (external test package) with standard `*testing.T`.
- Tests needing a DB use `DATABASE_URL` env var pointing to a test Postgres instance.
- Test helpers are in `internal/testutil/`.
- DTO validation tests create `validator.New()` directly.

## Security

- Never hardcode tokens, passwords, or API keys. `ADMIN_TOKEN` is required in production.
- Auth tokens compared with `crypto/subtle.ConstantTimeCompare`.
- The Nuxt frontend proxy (`web/server/api/[...].ts`) forwards `Authorization` headers — `NUXT_API_URL` is server-only.
