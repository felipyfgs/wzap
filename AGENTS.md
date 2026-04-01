# AGENTS.md

Guide for agentic coding agents operating in the **wzap** repository (Go 1.25).

## 1 · Build / Lint / Test

| Action | Command |
|---|---|
| Install dependencies | `go mod download` |
| Run app locally | `make dev` (`go run cmd/wzap/main.go`) |
| Build binary | `make build` → `build/wzap` |
| **Run all tests** | `go test -v -race -coverprofile=coverage.out ./...` |
| **Run a single test** | `go test -v -race -run TestName ./path/to/pkg` |
| Run tests for one package | `go test -v -race ./internal/service/...` |
| Lint | `golangci-lint run ./...` (install via `make install-tools`) |
| Tidy modules | `make tidy` (`go mod tidy`) |
| Start services (Postgres, MinIO, NATS) | `make up` |

CI (`.github/workflows/ci.yml`) runs on every PR and push to `main`:
- **lint**: golangci-lint with 5min timeout
- **test**: `go test -v -race -coverprofile=coverage.out ./...`
- **build**: compile binary + upload artifact
- **docker**: build & push multi-arch image to GHCR (main only)

Docker image is pushed to `ghcr.io/<owner>/wzap` with tags `latest` and `<sha>`.

## 2 · Project Structure

```
cmd/wzap/          Entry point
internal/
  config/          .env → typed Config struct
  database/        pgx pool wrapper
  broker/          NATS connection
  storage/         MinIO S3 client
  dto/             Request / Response payloads
  model/           Domain objects (Session, Message, Webhook …)
  repo/            Data-access layer
  service/         Business logic
  handler/         HTTP controllers (Fiber)
  middleware/       Auth, Logger, Recovery, CORS
  server/          Fiber app bootstrap + routes
  wa/              whatsmeow engine integration
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
- **Constants / enum-like**: `PascalCase` prefix + `PascalCase` value — `MsgTypeText`, `MsgTypeImage`
- **Acronyms**: keep uppercase — `APIKey`, `ID`, `URL`, `NATS`, `S3`
- **Constructors**: `New<Type>(…)` — `NewServer`, `NewSessionService`

### Types & Structs
- JSON struct tags use `json:"camelCase"`, add `omitempty` where nil/zero is meaningful.
- Prefer concrete types over `interface{}`; use `any` only at API boundaries.
- Define request/response DTOs in `internal/dto/`; domain models in `internal/model/`.

### Error Handling
- Wrap errors: `fmt.Errorf("failed to create session: %w", err)`.
- Return errors up the call stack; handle at handler level.
- In handlers, respond with `dto.ErrorResp(title, message)` and appropriate HTTP status.
- Use `log.Warn().Err(err).Msg(…)` for non-fatal errors (e.g., inline webhook creation).
- Fiber-specific: return `fiber.NewError(code, msg)` for framework errors.

### Logging
- Use `github.com/rs/zerolog` (`log.Info()`, `log.Warn()`, `log.Error()`).
- Chain `.Str()`, `.Err()`, `.Int()` for structured fields; end with `.Msg(…)`.

### Context
- Pass `context.Context` as the **first** parameter to service/repo methods.
- Use `c.Context()` from Fiber handlers.

### HTTP Handlers
- Signature: `func (h *XxxHandler) MethodName(c *fiber.Ctx) error`
- Parse body: `c.BodyParser(&req)` → return 400 with `dto.ErrorResp` on error.
- Success: `c.JSON(dto.SuccessResp(data))`.
- Add Swagger godoc above each handler (`@Summary`, `@Router`, …).

## 4 · Conventions

- **No comments** unless explicitly requested.
- Follow existing patterns; check neighboring files before introducing new libraries.
- Never log or commit secrets; use `.env` (git-ignored).
- Run `go mod tidy` after adding/removing dependencies.
- Always run `golangci-lint run ./…` and `go test ./…` before considering a task done.

## 5 · Known Issues

- **Missing main entry point**: The project references `cmd/wzap/main.go` but the `cmd/` directory doesn't exist. The Makefile and Dockerfile expect this file.
- **Build will fail** until the main entry point is created (typically `cmd/wzap/main.go` containing `func main()`).
- **No tests exist** yet — all packages show `[no test files]`.
- **Linter shows undefined references** in some files (missing imports or build context).

## 6 · External Rules

- **`.factory/rules/language.md`**: "Always respond in Brazilian Portuguese (pt-BR), regardless of the input language."
- **No Cursor rules** found (`.cursor/` or `.cursorrules`).
- **No Copilot instructions** found (`.github/copilot-instructions.md`).
