# Operations Checklist — wzap

Reference for engineers operating a wzap instance: startup, health verification, session lifecycle management, and common admin tasks.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `API_KEY` | `""` (dev mode — admin open) | Global admin bearer token |
| `PORT` | `8080` | HTTP server port |
| `SERVER_HOST` | `0.0.0.0` | Bind address |
| `DATABASE_URL` | `postgres://wzap:wzap123@localhost:5435/wzap?sslmode=disable` | pgx connection string |
| `NATS_URL` | `nats://localhost:4222` | NATS server for event fan-out |
| `MINIO_ENDPOINT` | `localhost:9010` | MinIO S3-compatible storage |
| `MINIO_ACCESS_KEY` | `admin` | MinIO access key |
| `MINIO_SECRET_KEY` | `admin123` | MinIO secret — **treat as secret** |
| `MINIO_BUCKET` | `wzap-media` | Bucket for uploaded media |
| `MINIO_USE_SSL` | `false` | Enable TLS for MinIO |
| `GLOBAL_WEBHOOK_URL` | `""` | Optional catch-all webhook for all sessions |
| `WA_LOG_LEVEL` | `INFO` | whatsmeow internal log level |
| `LOG_LEVEL` | `info` | zerolog level (`debug`, `info`, `warn`, `error`) |
| `ENVIRONMENT` | `development` | Environment tag in logs |

---

## Make commands

| Command | Action |
|---|---|
| `make up` | Start Postgres, MinIO, NATS via Docker Compose |
| `make dev` | Run `go run cmd/wzap/main.go` |
| `make build` | Compile to `build/wzap` |
| `make tidy` | `go mod tidy` |
| `make install-tools` | Install `golangci-lint` |

---

## Health check

```
GET /health
```

No auth required. Returns the status of the three infrastructure dependencies:

```json
{
  "success": true,
  "data": {
    "status": "UP",
    "services": {
      "database": true,
      "nats": true,
      "minio": true
    }
  }
}
```

If any value is `false`, the corresponding service is unreachable. Check Docker Compose (`make up`) or the environment variables.

---

## Session lifecycle

### States

| Status | `connected` | Meaning |
|---|---|---|
| `"disconnected"` | `0` | Session exists, no active WA connection |
| `"connecting"` | `0` | `POST /sessions/:id/connect` called, handshake or QR scan in progress |
| `"connected"` | `1` | Paired and fully connected; `jid` is populated |

### Common admin operations

**Create a session:**
```
POST /sessions
ApiKey: <ADMIN_API_KEY>

{
  "name": "my-bot",
  "webhook": {
    "url": "https://my-service.com/wh",
    "events": ["Message", "Connected"]
  }
}
```
Response includes the session's `apiKey` (`sk_<uuid>`) — store it securely.

**Connect / start pairing:**
```
POST /sessions/:sessionId/connect
ApiKey: <ADMIN_API_KEY>
```
Returns `{"status": "PAIRING"}` if the device needs to be linked, `{"status": "CONNECTED"}` if already linked, or `{"status": "CONNECTING"}` if the client exists but is not yet connected.

**Get QR code (poll until connected):**
```
GET /sessions/:sessionId/qr
ApiKey: <ADMIN_API_KEY>
```
Returns `{"qr": "<raw>", "image": "data:image/png;base64,..."}`.

**Force disconnect:**
```
POST /sessions/:sessionId/disconnect
ApiKey: <ADMIN_API_KEY>
```

**Delete session (irreversible):**
```
DELETE /sessions/:sessionId
ApiKey: <ADMIN_API_KEY>
```
Calls `engine.Logout(ctx, id)` which sends an unpair request to WhatsApp, deletes the device from the whatsmeow sqlstore, clears device fields in the DB, then removes the session row.

---

## Webhook management

**List webhooks for a session:**
```
GET /sessions/:sessionId/webhooks
ApiKey: <SESSION_API_KEY>
```

**Create a webhook:**
```
POST /sessions/:sessionId/webhooks
ApiKey: <SESSION_API_KEY>

{
  "url": "https://my-service.com/wh",
  "events": ["All"],
  "secret": "optional-hmac-secret"
}
```

**Delete a webhook:**
```
DELETE /sessions/:sessionId/webhooks/:wid
ApiKey: <SESSION_API_KEY>
```

---

## Observability

All structured log lines are emitted via `github.com/rs/zerolog/log`. Key fields:

| Field | Type | Example |
|---|---|---|
| `session` | string | `"abc-123"` (session ID) |
| `action` | string | `"force-disconnect"` |
| `err` | error | wrapped error message |
| `addr` | string | `"0.0.0.0:8080"` |

**Log level guidance:**
- `Info` — normal state changes (session connected, webhook created).
- `Warn` — non-fatal failures (inline webhook creation failed during session create).
- `Error` — unexpected failures that may require intervention.
- `Debug` — verbose tracing (enabled with `LOG_LEVEL=debug`).

---

## Common failure patterns

| Symptom | Likely cause | Fix |
|---|---|---|
| `401 Unauthorized` on every request | `API_KEY` not set or mismatch | Check `API_KEY` env var |
| `session not found` on connect | Session deleted or wrong ID | `GET /sessions` to list |
| QR endpoint returns 404 | `connect` not called first | Call `POST /:id/connect` then poll `/qr` |
| Webhook not firing | `enabled=false` or wrong event | Check webhook via `GET /:id/webhooks` |
| `client not connected` error | Session disconnected mid-operation | Reconnect via `POST /:id/connect` |
| Health check shows `"database": false` | Postgres down or wrong `DATABASE_URL` | `make up` or check connection string |

---

## Orphan device cleanup

On startup, `ReconnectAll` iterates all devices in the whatsmeow sqlstore. Any device whose JID does not match a session in `"wzSessions"` is logged as orphan and deleted from the sqlstore automatically. This prevents stale devices from accumulating after unclean shutdowns or failed deletions.
