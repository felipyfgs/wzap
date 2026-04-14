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
| WhatsApp      | whatsmeow (`go.mau.fi/whatsmeow`)                             |
| Database      | PostgreSQL 16 via pgx v5 (`jackc/pgx`)                         |
| Migrations    | Embedded SQL files (`migrations/`) applied at startup           |
| Object store  | MinIO (`minio/minio-go`)                                       |
| Message queue | NATS JetStream (`nats-io/nats.go`)                             |
| Cache         | Redis (optional, falls back to in-memory)                      |
| Logging       | zerolog (`rs/zerolog`)                                         |
| Observability | OpenTelemetry (OTLP traces), Prometheus metrics (`/metrics`)   |
| Validation    | go-playground/validator v10                                    |
| Docs          | Swagger via swaggo (`/swagger/*`)                              |
| Frontend      | Nuxt 4 SPA — see `web/AGENTS.md`                              |
| Docker        | Multi-stage Dockerfile, Docker Compose, Swarm deploy           |
| CI            | GitHub Actions (lint + test + Docker build/push)               |

## Build & dev commands

```bash
# Go backend
make dev              # run API with hot reload (go run)
make build            # CGO_ENABLED=0 binary → bin/wzap
make tidy             # go mod tidy
make docs             # regenerate Swagger docs (swag init)
make install-tools    # install golangci-lint + swag

# Frontend (from root)
make web-install      # pnpm install in web/
make web-dev          # pnpm dev in web/
make web-build        # pnpm build in web/
make dev-all          # backend + frontend concurrently

# Docker
make up               # docker compose up -d --build (dev stack)
make down             # stop dev stack
make prod             # production compose
make build-all        # build API + Web Docker images
make push             # build + push to Docker Hub
make deploy           # deploy to Docker Swarm

# Testing & linting
go test -v -race ./...                    # run all tests
go test -v -race -coverprofile=c.out ./...  # with coverage
golangci-lint run ./...                   # lint
```

## Directory structure

```
wzap/
├── cmd/wzap/main.go          # Entrypoint: config → DB → NATS → MinIO → Server
├── internal/
│   ├── async/                # Worker pool (async.Pool, async.Runtime)
│   ├── broker/               # NATS JetStream client
│   ├── config/               # Env-based config (godotenv)
│   ├── database/             # pgxpool + embedded migration runner
│   ├── dto/                  # Request/response DTOs with validation tags
│   ├── handler/              # Fiber HTTP handlers (one file per domain)
│   ├── integrations/
│   │   └── chatwoot/         # Chatwoot two-way integration (service, handler, consumer, repo)
│   ├── logger/               # zerolog singleton (Init, Info, Warn, Error, Fatal, Debug)
│   ├── metrics/              # Prometheus metrics
│   ├── middleware/            # Auth, rate-limit, recovery, logger, session resolver
│   ├── model/                # Domain models (Session, Message, Webhook, Event envelopes)
│   ├── provider/             # WhatsApp Cloud API provider
│   ├── repo/                 # PostgreSQL repositories (session, webhook, message, chat)
│   ├── server/               # Fiber app setup (server.go) + route registration (router.go)
│   ├── service/              # Business logic (one file per domain, lifecycle orchestrator)
│   ├── storage/              # MinIO client wrapper
│   ├── testutil/             # Shared test helpers
│   ├── wa/                   # whatsmeow Manager — session lifecycle, events, QR, connect
│   ├── webhook/              # Webhook dispatcher (HTTP + NATS + WebSocket broadcast)
│   └── websocket/            # WebSocket hub for real-time events
├── migrations/               # Numbered SQL migrations (001_schema, 002_messages, …)
├── docs/                     # Generated Swagger JSON/YAML
├── web/                      # Nuxt 4 frontend (see web/AGENTS.md)
├── scripts/                  # build.sh, deploy.sh
├── stacks/                   # Docker Swarm stack files
├── docker/chatwoot/          # Chatwoot Docker Compose + trigger script
├── docker-compose.yml        # Dev stack (API + Postgres + NATS + MinIO + Redis)
├── docker-compose.prod.yml   # Production compose
├── docker-compose.swarm.yml  # Swarm deploy
├── Dockerfile                # Multi-stage (dev + prod targets)
├── Makefile                  # All build/dev/deploy commands
└── .github/workflows/ci.yml  # CI: test → lint → build → push
```

## Code style & conventions

- **Go standard layout**: `cmd/` for entrypoints, `internal/` for all packages (not importable externally).
- **Layered architecture**: handler → service → repo. Handlers parse HTTP, services hold business logic, repos talk to the DB.
- **Error handling**: wrap with `fmt.Errorf("context: %w", err)`. Handlers return `dto.ErrorResp(title, message)`.
- **Logging**: use the `logger` singleton — never `log.Print` or `fmt.Print`. Always include `.Str("component", "xxx")`.
- **Context propagation**: pass `context.Context` as first parameter. Use `c.Context()` in Fiber handlers.
- **Concurrency**: the `wa.Manager` holds a `sync.RWMutex` protecting the `clients` map. Use `async.Pool` for background work (webhooks, media, history).
- **Dependency injection**: `server.New(cfg, db, nats, minio)` → `SetupRoutes()` wires repos → services → handlers. No global state besides the logger.
- **Naming**: Go standard — `MixedCaps`, short receiver names, no `Get` prefix on getters.
- Comments may be in Portuguese or English — preserve the original language when editing.

## Authentication

- **Admin token**: set via `ADMIN_TOKEN` env var. Sent as `Authorization` header (no `Bearer` prefix).
- **Session tokens**: each session has its own API key. The `middleware.Auth` checks admin token first (constant-time compare), then falls back to session token lookup.
- **Roles**: `admin` (full access) or `session` (scoped to one session via `RequiredSession` middleware).
- **WebSocket auth**: token via query param or `Authorization` header (configurable via `WS_AUTH_MODE`).

## Important patterns

1. **wa.Manager** — central singleton managing all whatsmeow clients. It handles connect/disconnect/reconnect, event routing, media download/retry, and exposes callback hooks (`OnMediaReceived`, `OnMessageReceived`, `OnHistorySyncReceived`).
2. **Webhook dispatcher** — `webhook.Dispatcher` fans out events to HTTP webhooks, NATS subjects, and the WebSocket hub. Listeners (like Chatwoot) register via `AddListener`.
3. **Embedded migrations** — SQL files in `migrations/` are embedded via `//go:embed` and applied at startup with advisory locks for safe concurrent deploys.
4. **Async pools** — `async.Pool` provides bounded goroutine pools for webhook delivery, media upload, and history sync. Pools drain gracefully on shutdown.
5. **Chatwoot integration** — full two-way sync: incoming WA messages → Chatwoot conversations, Chatwoot agent replies → WA messages. Uses NATS consumers for async processing.
6. **Router structure** — public routes (health, metrics, swagger, WS, chatwoot webhook) have no auth. All `/sessions` routes require auth middleware, with session-scoped routes nested under `/sessions/:sessionId`.

## Environment variables

| Variable                    | Purpose                                  | Default                  |
| --------------------------- | ---------------------------------------- | ------------------------ |
| `PORT`                      | API server port                          | `8080`                   |
| `SERVER_HOST`               | Bind address                             | `0.0.0.0`               |
| `ADMIN_TOKEN`               | Admin auth token                         | — (required)             |
| `LOG_LEVEL`                 | zerolog level                            | `info`                   |
| `ENVIRONMENT`               | `development` or `production`            | `development`            |
| `DATABASE_URL`              | PostgreSQL connection string             | — (required)             |
| `MINIO_ENDPOINT`            | MinIO host:port                          | `localhost:9010`         |
| `MINIO_ACCESS_KEY`          | MinIO access key                         | —                        |
| `MINIO_SECRET_KEY`          | MinIO secret key                         | —                        |
| `MINIO_BUCKET`              | Media bucket name                        | `wzap-media`             |
| `MINIO_USE_SSL`             | Use TLS for MinIO                        | `false`                  |
| `NATS_URL`                  | NATS server URL                          | `nats://localhost:4222`  |
| `REDIS_URL`                 | Redis URL (optional)                     | — (in-memory fallback)   |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector              | — (disabled by default)  |
| `OTEL_SDK_DISABLED`         | Disable OTel SDK                         | `true`                   |
| `WA_LOG_LEVEL`              | whatsmeow log level                      | `INFO`                   |
| `GLOBAL_WEBHOOK_URL`        | Global webhook endpoint                  | —                        |
| `SERVER_URL`                | Public server URL (for callbacks)        | `http://localhost:8080`  |
| `WS_AUTH_MODE`              | WebSocket auth mode                      | `token`                  |

## Testing

```bash
go test -v -race ./...                          # all tests
go test -v -race ./internal/service/...         # specific package
go test -v -race -run TestFunctionName ./...    # single test
```

- CI runs tests with a real Postgres service (see `.github/workflows/ci.yml`).
- Tests that need a DB use `DATABASE_URL` env var pointing to a test Postgres instance.
- Test helpers are in `internal/testutil/`.
- Linting: `golangci-lint run ./...` (CI uses `golangci/golangci-lint-action@v8`).

## Security considerations

- Never hardcode tokens, passwords, or API keys in source files.
- `ADMIN_TOKEN` must be set in production — the server rejects all requests if unset.
- Auth tokens are compared with `crypto/subtle.ConstantTimeCompare`.
- Database credentials and MinIO keys must stay in env vars or secrets, never in code.
- The Nuxt frontend proxy (`web/server/api/[...].ts`) forwards `Authorization` headers — `NUXT_API_URL` is server-only and never exposed to the browser.
