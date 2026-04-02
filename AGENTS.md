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
| Tidy modules | `make tidy` (`go mod tidy`) |
| Start services (Postgres, MinIO, NATS) | `make up` |
| Generate Swagger docs | `make docs` |

CI (`.github/workflows/ci.yml`) runs on every PR and push to `main`:
- **lint**: golangci-lint with 5min timeout
- **test**: `go test -v -race -coverprofile=coverage.out ./...`
- **build**: compile binary + upload artifact
- **docker**: build & push multi-arch image to GHCR (main only)

Docker image is pushed to `ghcr.io/<owner>/wzap` with tags `latest` and `<sha>`.

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
```

## 3 · Code Style

### Imports
Group imports in this order, separated by blank lines:
1. Standard library
2. Third-party packages (`github.com/…`, `go.mau.fi/…`)
3. Internal packages (`wzap/internal/…`)

```go
import (
    "context"
    "fmt"

    "github.com/gofiber/fiber/v2"
    "github.com/rs/zerolog/log"

    "wzap/internal/dto"
    "wzap/internal/service"
)
```

### Naming
- **Exported**: `PascalCase` — `SessionService`, `NewHealthHandler`, `SendText`
- **Unexported**: `camelCase` — `getSessionID`, `sessionNameRegex`
- **Constants / enum-like**: `PascalCase` prefix + `PascalCase` value — `MsgTypeText`, `EventMessage`
- **Acronyms**: keep uppercase — `APIKey`, `ID`, `URL`, `JID`, `NATS`, `S3`
- **Constructors**: `New<Type>(…)` — `NewServer`, `NewSessionService`
- **Private helpers in handlers**: lowercase — `parseAndValidate`, `mustGetSessionID`

### Types & Structs
- JSON struct tags use `json:"camelCase"`, add `omitempty` where nil/zero is meaningful.
- Use `validate:"required"` tags on DTO fields for request validation via `go-playground/validator`.
- Prefer concrete types over `interface{}`; use `any` only at API boundaries (e.g. `dto.SuccessResp(data interface{})`).
- Define request/response DTOs in `internal/dto/`; domain models in `internal/model/`.
- Use pointers for optional update fields: `*string`, `*SessionProxy` (nil = not provided).

### Error Handling
- Wrap errors: `fmt.Errorf("failed to create session: %w", err)`.
- Return errors up the call stack; handle at handler level.
- In handlers, respond with `dto.ErrorResp(title, message)` and appropriate HTTP status.
- Use `logger.Warn().Err(err).Msg(…)` for non-fatal errors (e.g., inline webhook creation).
- For fatal startup errors, use `logger.Fatal().Err(err).Msg(…)`.
- Fiber-specific: return `fiber.NewError(code, msg)` for framework-level errors.

### Logging
- Use the `internal/logger` wrapper (not `zerolog/log` directly): `logger.Info()`, `logger.Warn()`, etc.
- Chain `.Str()`, `.Err()`, `.Int()` for structured fields; end with `.Msg(…)`.
- Example: `logger.Info().Str("addr", addr).Msg("Starting API server")`.

### Context
- Pass `context.Context` as the **first** parameter to service/repo methods.
- Use `c.Context()` from Fiber handlers to pass the request context.

### HTTP Handlers
- Signature: `func (h *XxxHandler) MethodName(c *fiber.Ctx) error`
- Parse + validate: `parseAndValidate(c, &req)` — handles BodyParser and struct validation.
- Success: `c.JSON(dto.SuccessResp(data))` with appropriate status code.
- Error: `c.Status(code).JSON(dto.ErrorResp(title, msg))`.
- Get session ID: `getSessionID(c)` (returns error) or `mustGetSessionID(c)` (panics-safe, for admin routes).
- Add Swagger godoc above each handler (`@Summary`, `@Router`, `@Tags`, etc.).

### Authentication Flow
- Auth middleware (`middleware.Auth`) reads `Authorization` header (token only, no Bearer prefix).
- Sets `c.Locals("authRole", "admin"|"session")` and `c.Locals("sessionID", id)`.
- `middleware.RequiredSession` resolves `:sessionId` param (name or UUID → session.ID).

### Validation
- Global validator instance: `middleware.Validate` (initialized in `middleware/validate.go`).
- `parseAndValidate` helper in `handler/helpers.go` parses body + runs struct validation.
- Validation errors returned as 400 with field-level detail messages.

## 4 · Testing Conventions

- Use external test packages: `package handler_test`, `package dto_test`.
- Use standard `testing.T` — no assertion libraries (no testify).
- Create Fiber app per test group via helper functions (e.g., `newSessionApp()`, `newMessageApp()`).
- Use `fiber.New(fiber.Config{DisableStartupMessage: true})` in tests.
- Use `httptest.NewRequest` + `app.Test(req, -1)` for HTTP testing.
- Test file `internal/testutil/fiber.go` provides `NewApp()`, `DoRequest()`, `ParseResp()`.
- Stub services with nil when only testing handler validation (nil repo causes 500, not panic).

## 5 · Conventions

- **No comments** unless explicitly requested (godoc on handlers is acceptable).
- Follow existing patterns; check neighboring files before introducing new libraries.
- Never log or commit secrets; use `.env` (git-ignored, see `.env.example`).
- Run `go mod tidy` after adding/removing dependencies.
- Always run `golangci-lint run ./…` and `go test ./…` before considering a task done.
- Swagger docs are auto-generated; run `make docs` after changing handler annotations.

## 6 · Agent Configuration

This project is dual-compatible with **OpenCode** (`.opencode/`) and **Cursor/Factory** (`.factory/`).

### Directory layout

```
.factory/
  rules/                  Global rules (language.md)
  skills/                 Reusable agent skills (SKILL.md + companion docs)
    data-querying/        Postgres query patterns & schema reference
    internal-tools/       Admin-only endpoint creation
    service-integration/  Handler/service/repo layer patterns & whatsmeow API
.opencode/                OpenCode-compatible directory
  skills/                 Symlinks → ../../.factory/skills/<name>/
opencode.json             OpenCode config (loads AGENTS.md + .factory/rules/*.md)
AGENTS.md                 This file — main rules (read by both OpenCode and Factory)
```

### How it works

| Tool | Rules source | Skills source |
|---|---|---|
| **OpenCode** | `AGENTS.md` + `opencode.json` instructions | `.opencode/skills/*/SKILL.md` (symlinks) |
| **Cursor / Factory** | `.factory/rules/*.md` | `.factory/skills/*/SKILL.md` |

- Skills are defined **once** in `.factory/skills/` and symlinked into `.opencode/skills/`.
- `opencode.json` loads additional instructions from `.factory/rules/*.md`.
- Both systems share the same `AGENTS.md` as the primary rules file.

### Available skills

| Skill | Description |
|---|---|
| `data-querying` | Add/extend Postgres queries in the repo layer |
| `internal-tools` | Build admin-only operational endpoints |
| `service-integration` | Add HTTP endpoints, WhatsApp features, or repo methods |

## 7 · External Rules

- **`.factory/rules/language.md`**: "Always respond in Brazilian Portuguese (pt-BR), regardless of the input language."
- **No Cursor rules** found (`.cursor/` or `.cursorrules`).
- **No Copilot instructions** found (`.github/copilot-instructions.md`).
