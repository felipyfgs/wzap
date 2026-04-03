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

CI (`.github/workflows/ci.yml`) runs lint + test + build on every PR.

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
  wa/                whatsmeow engine integration (Manager, events, JID helpers)
  webhook/           Webhook dispatcher (NATS consumer, WS broadcaster)
  websocket/         WebSocket hub for real-time events
  testutil/          Shared test helpers (NewApp, DoRequest, ParseResp)
docs/                Generated Swagger docs
web/                 Nuxt UI dashboard (Vue 3 + Nuxt UI Pro)
```

## 3 · Code Style

### Imports
Group imports in **three** groups separated by blank lines:
1. Standard library (`"context"`, `"fmt"`, …)
2. Third-party packages (`"github.com/…"`, `"go.mau.fi/…"`, `"google.golang.org/…"`)
3. Internal packages (`"wzap/internal/…"`)

### Naming
- **Exported**: `PascalCase` — `SessionService`, `NewHealthHandler`, `SendText`
- **Unexported**: `camelCase` — `getSessionID`, `sessionNameRegex`
- **Acronyms**: keep uppercase — `APIKey`, `ID`, `URL`, `JID`, `NATS`, `S3`
- **Constructors**: `New<Type>(…)` — `NewServer`, `NewSessionService`

### Types & Structs
- JSON struct tags use `json:"camelCase"`, add `omitempty` where nil/zero is meaningful.
- Use `validate:"required"` tags on DTO fields for request validation via `go-playground/validator`.
- Use `validate:"required,min=1"` on slice fields to reject empty arrays.
- Prefer concrete types over `interface{}`; use `any` only at API boundaries (e.g. `dto.SuccessResp`).
- Define request/response DTOs in `internal/dto/`; domain models in `internal/model/`.
- Use pointers for optional update fields: `*string`, `*SessionProxy` (nil = not provided).
- Initialize slices with `make([]T, 0)` to avoid JSON `null` instead of `[]`.

### Error Handling
- Wrap errors: `fmt.Errorf("failed to create session: %w", err)`.
- Return errors up the call stack; handle at handler level.
- In handlers, respond with `dto.ErrorResp(title, msg)` and appropriate HTTP status. Never expose raw `err.Error()` from service/database layer — log internally instead.
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
- Get session ID: `mustGetSessionID(c)` (for routes behind `RequiredSession` middleware).
- Add Swagger godoc above each handler (`@Summary`, `@Router`, `@Tags`, etc.).

### Authentication Flow
- Auth middleware (`middleware.Auth`) reads `Authorization` header (token only, no Bearer prefix). Uses constant-time comparison.
- Sets `c.Locals("authRole", "admin"|"session")` and `c.Locals("sessionID", id)`.
- `middleware.RequiredSession` resolves `:sessionId` param (name or UUID → session.ID).

## 4 · Testing Conventions

- Use external test packages: `package handler_test`, `package dto_test`.
- Use standard `testing.T` — no assertion libraries (no testify).
- Create Fiber app per test group via helper functions (e.g., `newSessionApp()`, `newMessageApp()`).
- Use `fiber.New(fiber.Config{DisableStartupMessage: true})` in tests.
- Use `httptest.NewRequest` + `app.Test(req, -1)` for HTTP testing.
- Test file `internal/testutil/fiber.go` provides `NewApp()`, `DoRequest()`, `ParseResp()`.

## 5 · Conventions

- **No comments** unless explicitly requested (godoc on handlers is acceptable).
- Follow existing patterns; check neighboring files before introducing new libraries.
- Never log or commit secrets; use `.env` (git-ignored, see `.env.example`).
- Run `go mod tidy` after adding/removing dependencies.
- Always run `golangci-lint run ./…` and `go test ./…` before considering a task done.
- Swagger docs are auto-generated; run `make docs` after changing handler annotations.

## 6 · Agent Configuration

```
.agents/
  rules/                  Global rules (language.md)
  skills/                 Reusable agent skills (SKILL.md + companion docs)
    data-querying/        Postgres query patterns & schema reference
    internal-tools/       Admin-only endpoint creation
    service-integration/  Handler/service/repo layer patterns & whatsmeow API
  mcp.json                MCP servers (Nuxt UI, Nuxt, Postgres)
```

### Available skills

| Skill | Description |
|---|---|
| `data-querying` | Add/extend Postgres queries in the repo layer |
| `internal-tools` | Build admin-only operational endpoints |
| `service-integration` | Add HTTP endpoints, WhatsApp features, or repo methods |

### External Rules

- **`.agents/rules/language.md`**: "Always respond in Brazilian Portuguese (pt-BR), regardless of the input language."
