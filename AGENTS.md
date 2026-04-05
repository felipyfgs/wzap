# AGENTS.md

Guide for agentic coding agents operating in the **wzap** repository (Go 1.25).

## 1 · Build / Lint / Test

| Action | Command |
|---|---|
| Install dependencies | `go mod download` |
| Run app locally | `make dev` (`go run cmd/wzap/main.go`) |
| Build binary | `make build` → `bin/wzap` |
| **Run all tests** | `go test -v -race -coverprofile=coverage.out ./...` |
| **Run a single test** | `go test -v -race -run TestName ./path/to/pkg` |
| Run tests for one package | `go test -v -race ./internal/service/...` |
| Lint | `golangci-lint run ./...` (install via `make install-tools`) |
| Tidy modules | `go mod tidy` |
| Start services (Postgres, MinIO, NATS) | `make up` |
| Generate Swagger docs | `make docs` |

## 2 · Project Structure

```
cmd/wzap/            Entry point (main.go)
internal/
  config/            .env → typed Config struct
  database/          pgx pool wrapper + migrations
  broker/            NATS JetStream connection
  storage/           MinIO S3 client
  logger/            zerolog wrapper (Info, Warn, Error, Fatal, Debug)
  dto/               Request / Response payloads + validation
  model/             Domain objects (Session, Message, Webhook, Events …)
  repo/              Data-access layer (pgx queries)
  service/           Business logic
  handler/           HTTP controllers (Fiber)
  middleware/        Auth, Logger, Recovery, CORS, RateLimit, Validate
  server/            Fiber app bootstrap + routes
  metrics/           Prometheus metrics (counters, histograms, collectors)
  wa/                whatsmeow engine integration (Manager, events, JID helpers)
  webhook/           Webhook dispatcher (NATS consumer, WS broadcaster)
  websocket/         WebSocket hub for real-time events
  provider/          External API clients (e.g. WhatsApp Cloud API)
  integrations/      Third-party integrations (e.g. chatwoot)
  testutil/          Shared test helpers (NewApp, DoRequest, ParseResp)
migrations/          SQL migration files
docs/                Generated Swagger docs
web/                 Nuxt UI dashboard (Vue 3 + Nuxt UI Pro)
```

## 3 · Code Style

### Imports
Group imports in **three** groups separated by blank lines:
1. Standard library (`"context"`, `"fmt"`, …)
2. Third-party packages (`"github.com/…"`, `"go.mau.fi/…"`, `"google.golang.org/…"`)
3. Internal packages (`"wzap/internal/…"`)

Use aliases when needed to avoid collisions: `mw "wzap/internal/middleware"`, `ws "github.com/gofiber/contrib/websocket"`, `cloudWA "wzap/internal/provider/whatsapp"`, `wsHub "wzap/internal/websocket"`.

### Naming
- **Exported**: `PascalCase` — `SessionService`, `NewHealthHandler`, `SendText`
- **Unexported**: `camelCase` — `getSessionID`, `sessionNameRegex`, `sendMedia`
- **Acronyms**: keep uppercase — `ID`, `URL`, `JID`, `NATS`, `S3`, `API`
- **Constructors**: `New<Type>(…)` — `NewServer`, `NewSessionService`, `NewSessionRepository`
- **Interfaces**: prefer small, single-method interfaces defined where they're consumed

### Types & Structs
- JSON struct tags use `json:"camelCase"`, add `omitempty` where nil/zero is meaningful.
- Use `validate:"required"` tags on DTO fields for request validation via `go-playground/validator`.
- Use `validate:"required,min=1"` on slice fields to reject empty arrays.
- Prefer concrete types; use `interface{}` only at API boundaries (e.g. `dto.SuccessResp(data interface{})`).
- Define request/response DTOs in `internal/dto/`; domain models in `internal/model/`.
- Use pointers for optional update fields: `*string`, `*SessionProxy` (nil = not provided).
- Initialize slices with `make([]T, 0)` to avoid JSON `null` instead of `[]`.

### Error Handling
- Wrap errors: `fmt.Errorf("failed to create session: %w", err)`.
- Return errors up the call stack; handle at handler level.
- In handlers, respond with `dto.ErrorResp(title, msg)` and appropriate HTTP status.
- **Never expose raw `err.Error()` from service/database layer for 500 errors** — log internally with `logger.Warn().Err(err).Msg(…)` and return a generic message like `"internal server error"`.
- For 400-level errors where the message is user-facing (validation, not found), `err.Error()` is acceptable.
- Use `logger.Warn().Err(err).Msg(…)` for non-fatal errors.
- For fatal startup errors, use `logger.Fatal().Err(err).Msg(…)`.

### Logging
- Use the `internal/logger` wrapper (not `zerolog/log` directly): `logger.Info()`, `logger.Warn()`, etc.
- Chain `.Str()`, `.Err()`, `.Int()` for structured fields; end with `.Msg(…)`.

### Context
- Pass `context.Context` as the **first** parameter to service/repo methods.
- Use `c.Context()` from Fiber handlers to pass the request context.

### HTTP Handlers
- Signature: `func (h *XxxHandler) MethodName(c *fiber.Ctx) error`
- Parse + validate: `parseAndValidate(c, &req)` — handles BodyParser and struct validation.
- Success: `c.JSON(dto.SuccessResp(data))` with appropriate status code.
- Error: `c.Status(code).JSON(dto.ErrorResp(title, msg))`.
- Session ID: use `mustGetSessionID(c)` behind `RequiredSession` middleware; use `getSessionID(c)` when no middleware guarantee.
- Add Swagger godoc above each handler (`@Summary`, `@Router`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Security`).

### Authentication Flow
- Auth middleware (`middleware.Auth`) reads `Authorization` header (token only, no Bearer prefix). Uses constant-time comparison.
- Sets `c.Locals("authRole", "admin"|"session")` and `c.Locals("sessionID", id)`.
- `middleware.RequiredSession` resolves `:sessionId` param (name or UUID → session.ID).

### Dependency Injection
- Handlers receive concrete `*service.XxxService` and `*repo.XxxRepository` via constructor injection.
- Services receive `*repo.XxxRepository` and other dependencies (engine, providers) via constructor.
- Repos receive `*pgxpool.Pool` via constructor.
- Constructors return pointers: `NewSessionHandler(…) *SessionHandler`.

### Database (Repo Layer)
- Use `pgxpool.Pool` directly; write raw SQL with positional parameters (`$1`, `$2`, …).
- Use `COALESCE` for nullable columns in SELECT queries.
- Wrap all errors with `fmt.Errorf("failed to <verb> <entity>: %w", err)`.
- Return `(*model.T, error)` for single-record lookups, `([]model.T, error)` for lists.

## 4 · Testing Conventions

- Use **external test packages** (`package handler_test`, `package dto_test`) for public API tests.
- Use **internal test packages** (`package service`, `package wa`) only when testing unexported functions.
- Use standard `testing.T` — **no assertion libraries** (no testify).
- Create a Fiber app per test group via helper functions (e.g., `newSessionApp()`, `newMessageApp()`, `newWebhookApp()`).
- Use `fiber.New(fiber.Config{DisableStartupMessage: true})` in tests.
- Use `fiber/middleware/recover.New()` in test apps.
- Use `httptest.NewRequest` + `app.Test(req, -1)` for HTTP testing.
- Mock dependencies by passing `nil` for services/repos when testing validation/error paths.
- Simulate middleware in tests by setting `c.Locals("authRole", "admin")` and `c.Locals("sessionID", c.Params("sessionId"))` inline via middleware closures.
- Error assertions: check status codes directly, use `t.Errorf`/`t.Fatalf`.

## 5 · Commit & Pull Request Conventions

### Commit Messages
Follow **Conventional Commits** format:
- `feat:` / `feat(scope):` — new feature (`feat(chatwoot): add message sync`)
- `fix:` / `fix(scope):` — bug fix (`fix: improve error handling in handlers`)
- `chore:` — maintenance tasks (`chore: add .agent/ to .gitignore`)
- `refactor:` — code restructuring without behavior change
- `docs:` — documentation updates
- `test:` — adding or updating tests

Keep the subject line concise (≤72 chars), imperative mood, lowercase.

### Pull Requests
- Reference related issues when applicable.
- Keep PRs focused on a single concern; avoid mixing refactors with features.
- Ensure `golangci-lint run ./…` and `go test ./…` pass before opening a PR.

## 6 · Conventions

- **No comments** unless explicitly requested (godoc on handlers is acceptable).
- Follow existing patterns; check neighboring files before introducing new libraries.
- Never log or commit secrets; use `.env` (git-ignored, see `.env.example`).
- Run `go mod tidy` after adding/removing dependencies.
- Always run `golangci-lint run ./…` and `go test ./…` before considering a task done.
- Swagger docs are auto-generated; run `make docs` after changing handler annotations.

## 7 · Agent Configuration

- Always respond in Brazilian Portuguese (pt-BR), regardless of the input language.
- Read this file before starting any task.
- Check neighboring files for patterns before introducing new libraries or structures.
