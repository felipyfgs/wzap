# AGENTS.md — wzap

## Idioma

- Sempre se comunique em Português do Brasil
- Sempre responda em Português do Brasil

## Project overview

**wzap** is a WhatsApp multi-session gateway with a Go REST API backend and a Nuxt 4 SPA dashboard.
It manages multiple WhatsApp sessions via [whatsmeow](https://github.com/tulir/whatsmeow), exposing messaging, contacts, groups, newsletters, labels, media, and webhook/WebSocket event delivery through a unified HTTP API.

## Tech stack

| Layer         | Technology                                                     |
| ------------- | -------------------------------------------------------------- |
| Language      | **Go 1.25** (module `wzap`)                                    |
| HTTP          | Fiber v2 (`gofiber/fiber`)                                     |
| WhatsApp      | whatsmeow (`go.mau.fi/whatsmeow`) + Cloud API emulator handler |
| Database      | PostgreSQL 16 via pgx v5 (`jackc/pgx`)                         |
| Migrations    | Embedded SQL files (`migrations/`) applied at startup           |
| Object store  | MinIO (`minio/minio-go`)                                       |
| Message queue | NATS JetStream (`nats-io/nats.go`)                             |
| Cache         | Redis (optional, falls back to in-memory)                      |
| Logging       | zerolog (`rs/zerolog`)                                         |
| Validation    | go-playground/validator v10                                    |
| Docs          | Swagger via swaggo (`/swagger/*`)                              |
| Frontend      | Nuxt 4 SPA (`pnpm`, `ssr: false`)                              |

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
  imgutil/                # Image conversion utilities (WebP→PNG/GIF)
  integrations/chatwoot/  # Chatwoot two-way integration (see below)
  logger/                 # zerolog singleton
  metrics/                # Prometheus metrics definitions
  middleware/              # Auth, rate-limit, recovery, logger, session, validate
  model/                  # Domain models (Session, Message, Webhook, Event envelopes)
  repo/                   # PostgreSQL repositories
  server/                 # Fiber app setup (server.go) + route registration (router.go)
  service/                # Business logic (one file per domain)
  storage/                # MinIO client wrapper
  testutil/               # Shared test helpers
  wa/                     # whatsmeow Manager — session lifecycle, events, QR, connect
  wautil/                 # WhatsApp protocol utilities (message extraction, etc.)
  webhook/                # Webhook dispatcher (HTTP + NATS + WebSocket broadcast)
  websocket/              # WebSocket hub for real-time events
migrations/               # Numbered SQL migrations (embedded via //go:embed)
docs/                     # Generated Swagger JSON/YAML
docker/chatwoot/          # Chatwoot stack (docker-compose.yml + trigger.rb)
Dockerfile                # Multi-stage: web-dev, web-prod, api-dev, api-prod, combined
docker-compose.yml        # Base infra: postgres + minio + nats + redes
docker-compose.dev.yml    # Overlay dev: api + web com hot reload
docker-compose.prod.yml   # Overlay prod: api + web compilados
scripts/setup.sh          # Build de imagens (combined/api-prod/web-prod/--split)
web/                      # Nuxt 4 SPA frontend
```

## Code style & conventions

### Architecture

- **Layered**: handler → service → repo. Handlers parse HTTP, services hold business logic, repos talk to the DB.
- **Dependency injection**: `server.New(cfg, db, nats, minio)` → `SetupRoutes()` wires repos → services → handlers. No DI framework, no global state besides the logger.
- **Router structure**: public routes (health, metrics, swagger, WS, Cloud API) have no auth. All `/sessions` routes require auth, with session-scoped routes nested under `/sessions/:sessionId`.

### Imports

Grouped with blank-line separators: (1) standard library, (2) external packages, (3) internal `wzap/internal/...` packages. Use aliases for disambiguation.

### Naming

Go standard — `MixedCaps`, short receiver names (1-2 chars), no `Get` prefix on getters. DTOs use `Req`/`Resp` suffixes (`SendTextReq`, `SessionResp`). Comments may be in Portuguese or English — preserve the original language when editing.

### Handler vs Service function naming

- **`Handle*`** (exported) = Fiber HTTP handlers (e.g., `HandleIncomingWebhook`)
- **`process*`** (unexported) = Service-layer business logic (e.g., `processMessage`, `processGroupInfo`, `processReaction`)

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
- **Cloud API paths** (`/v\d+\.\d+/...`) are skipped by auth middleware — these routes are public by design (registered before the auth group in router.go).

## Cloud API emulator

`internal/handler/cloud_api.go` emulates the Facebook WhatsApp Cloud API for Chatwoot's Cloud inbox mode. Routes are registered **before** the auth group:

- `GET /:version/debug_token` — fake token validation
- `GET /:version/:phone` — phone status
- `POST /:version/:phone/register` — register phone
- `POST /:version/:phone/subscribed_apps` — subscribe
- `GET /:version/:phone/messages` — webhook verification
- `POST /:version/:phone/messages` — send messages (text, media, contacts, reaction, template)
- `GET /:version/:phone/message_templates` — list templates
- `GET /:version/:phone/phone_numbers` — list phone numbers
- `GET /:version/:phone/:media_id` — get media

Never returns HTTP 401 to prevent Chatwoot from setting `reauthorization_required`. `warnTokenMismatch` logs mismatches but proceeds.

## Chatwoot integration

`internal/integrations/chatwoot/` — two-way Chatwoot sync with two inbox modes.

### File naming convention

Após o refactor `2edce63`, apenas dois prefixos sobrevivem — os demais arquivos usam nomes diretos (`webhook_outbound.go`, `conversation.go`, `bot.go`, `labels.go`, `backfill.go`, `mapping.go`, ...).

| Prefix | Origin | Purpose |
|--------|--------|---------|
| `wa_*` | WhatsApp inbound | Event pipeline (`wa_events.go`) |
| `inbox_*` | Inbox mode | Interface + router (`inbox.go`), API mode (`inbox_api.go`), Cloud mode (`inbox_cloud.go`), shared helpers (`inbox_common.go`) |

### InboxHandler interface

`InboxHandler` in `inbox.go` — `HandleMessage(ctx, cfg, payload)` + `UnlockWindow(ctx, cfg, chatJID)`. The `processMessage` router on Service delegates to `apiInboxHandler` (REST API) or `cloudInboxHandler` (Cloud webhook) based on `cfg.InboxType`.

### Cloud mapping (`database_uri`)

- For `inbox_type=cloud`, set `database_uri` in `wz_chatwoot` whenever possible.
- `database_uri` enables direct read-only lookup of Chatwoot `messages.source_id` (WAID mapping), avoiding dependence on webhook timing.
- Use `POST /sessions/{sessionId}/integrations/chatwoot/backfill` to retroactively fill `wz_messages.cw_message_id/cw_conversation_id/cw_source_id` for existing rows.
- Keep `database_uri` secret (never log full URI).

## Swagger docs

Every exported handler has godoc/Swagger annotations. Regenerate with `make docs`. The swag command scans `cmd/wzap,internal` with `--parseInternal --useStructName`.

## Testing

- Test files use `package xxx_test` (external test package) with standard `*testing.T`.
- Tests needing a DB use `DATABASE_URL` env var pointing to a test Postgres instance.
- Test helpers are in `internal/testutil/`.
- DTO validation tests create `validator.New()` directly.

## Docker

### Dockerfile (multi-stage, raiz único)

O `Dockerfile` na raiz consolida **API Go + Web Nuxt** em stages nomeados com BuildKit cache mounts (pnpm store, go mod, go build).

| Target     | Base                  | Uso                                  | Porta(s)    |
|------------|-----------------------|--------------------------------------|-------------|
| `web-dev`  | `node:22-alpine`      | Nuxt dev (hot reload)                | 3000        |
| `web-prod` | `node:22-alpine`      | Nitro node-server (`.output`)        | 3000        |
| `api-dev`  | `golang:1.25-alpine`  | API com air (hot reload)             | 8080        |
| `api-prod` | `alpine:3.21`         | Binário Go compilado                 | 8080        |
| `combined` | `node:22-alpine`+tini | API + Web numa única imagem          | 8080 + 3000 |

> Não existe `web/Dockerfile` — tudo vive no `Dockerfile` raiz. `tini` é usado no `combined` para propagação correta de sinais.

### Compose layering

- `docker-compose.yml` — **somente infra**: postgres + minio + nats + redes `wzap_net` e `wzap_chatwoot`
- `docker-compose.dev.yml` — **overlay dev**: serviços `api` (air) + `web` (nuxt dev), bind mount de código, volumes nomeados para caches
- `docker-compose.prod.yml` — **overlay prod**: serviços `api` + `web` compilados (imagens `wzap-api:latest` e `wzap-web:latest`)
- `docker/chatwoot/docker-compose.yml` — stack Chatwoot (rails + sidekiq + postgres + redis), usa `wzap_chatwoot` como `external`

Serviços `api` e `web` são **sempre separados** (dev e prod). A imagem `combined` existe apenas para deploys single-container (VPS simples) via `scripts/setup.sh`.

### Redes

- `wzap_net` — rede interna do wzap (postgres, minio, nats, api, web). Criada pelo compose base.
- `wzap_chatwoot` — rede compartilhada com Chatwoot. Criada pelo compose base do wzap (**não é `external`**), e referenciada como `external: true` pelo stack do Chatwoot.

### Comunicação entre serviços

- **Web → API** (server-side): `NUXT_API_URL=http://api:8080` (DNS interno). Usado pelo proxy Nitro em `web/server/api/[...].ts` e pelo WS bridge em `web/server/routes/ws.ts`.
- **Web → MinIO** (whitelist SSRF): `NUXT_MINIO_ENDPOINT` — dev: `http://localhost:9010`, prod: `http://minio:9000`.
- **API → DB/MinIO/NATS**: via DNS interno (`postgres:5432`, `minio:9000`, `nats:4222`).
- **Chatwoot → API (Cloud API)**: `WHATSAPP_CLOUD_BASE_URL=http://api:8080` via rede `wzap_chatwoot`.

### Makefile targets

| Comando                   | Ação                                                   |
|---------------------------|--------------------------------------------------------|
| `make docker-dev`         | sobe infra + api + web com hot reload                  |
| `make docker-prod`        | sobe infra + api + web em modo produção                |
| `make docker-build`       | builda imagem combinada (`wzap:latest`)                 |
| `make docker-build-split` | builda imagens separadas (`wzap-api`, `wzap-web`)       |
| `make push`               | build + push da imagem combinada ao Docker Hub         |
| `make logs-api`           | logs em tempo real do container `api`                  |
| `make logs-web`           | logs em tempo real do container `web`                  |
| `make docker-down`        | para todos os containers                               |
| `make docker-down-v`      | para + remove volumes (**destrutivo**)                 |
| `make chatwoot-up/down`   | sobe/para stack do Chatwoot                            |

### scripts/setup.sh

Builda imagens para deploy (tags locais + Docker Hub):

```bash
./scripts/setup.sh                     # combined (wzap:latest) [default]
./scripts/setup.sh --target=api-prod   # somente API (wzap-api:latest)
./scripts/setup.sh --target=web-prod   # somente Web (wzap-web:latest)
./scripts/setup.sh --split             # api-prod + web-prod separados
./scripts/setup.sh --push              # build + push ao Docker Hub
./scripts/setup.sh --no-cache          # force rebuild
```

## Security

- Never hardcode tokens, passwords, or API keys. `ADMIN_TOKEN` is required in production.
- Auth tokens compared with `crypto/subtle.ConstantTimeCompare`.
- The Nuxt frontend proxy (`web/server/api/[...].ts`) forwards `Authorization` headers — `NUXT_API_URL` is server-only.
