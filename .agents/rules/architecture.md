# Architecture Rules

## Layered Architecture

All code follows a strict layered dependency flow:

```
Handler → Service → Repo / wa.Manager
                  ↘ storage.Minio (only MediaService)
```

- **Handlers** never import `repo` directly (exception: `HistoryHandler` uses `messageRepo`).
- **Services** never import `handler`.
- **Repos** never import `service` or `handler`.
- **No interfaces** — all dependencies are concrete struct pointers.
- **No transactions** — every DB operation is a single auto-committed statement.

## Dependency Wiring Order (in `internal/server/router.go`)

```
Repositories → Hub + Dispatcher → wa.Manager → Services → Callbacks → Handlers
```

Callbacks (`SetMediaAutoUpload`, `SetMessagePersist`) are wired after services to break dependency cycles.

## Route Tiers

| Tier | Auth | Registration |
|---|---|---|
| Public (`/health`, `/swagger/*`) | None | `s.App.Get(...)` |
| WebSocket (`/ws`) | Custom token check | `s.App.Use("/ws", wsHandler.Upgrade())` |
| Admin API (`POST /sessions`, `GET /sessions`) | `middleware.Auth` + handler-level admin check | `grp.Post(...)` |
| Session-scoped API | `middleware.Auth` + `middleware.RequiredSession` | `sess.Post(...)` |

Admin-only routes are inside the authenticated group but check `c.Locals("authRole") != "admin"` inside the handler body — there is no admin middleware.

## Startup and Shutdown

- Startup is linear and sequential: `config → logger → database → broker → storage → server → routes`.
- Every startup failure calls `logger.Fatal()`.
- Shutdown: `SIGINT`/`SIGTERM` → `srv.Shutdown(ctx)` → `cancel()` (stops background consumers) → Fiber drain with 10s timeout.

## Context Flow

- `context.Context` originates from `c.Context()` in Fiber handlers.
- Passed as first parameter to all service and repo methods.
- Fire-and-forget operations (`PersistMessage`, `AutoUploadMedia`) create their own `context.WithTimeout` internally.
