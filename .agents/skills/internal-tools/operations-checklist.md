# Operations Checklist — wzap

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `ADMIN_TOKEN` | `""` | Admin token master. **Never deploy empty.** |
| `PORT` | `8080` | HTTP port |
| `SERVER_HOST` | `0.0.0.0` | Bind address |
| `DATABASE_URL` | `postgres://wzap:wzap123@localhost:5435/wzap?sslmode=disable` | Postgres |
| `NATS_URL` | `nats://localhost:4222` | NATS JetStream |
| `MINIO_ENDPOINT` | `localhost:9010` | MinIO S3 |
| `MINIO_ACCESS_KEY` | `admin` | MinIO access key |
| `MINIO_SECRET_KEY` | `admin123` | MinIO secret |
| `MINIO_BUCKET` | `wzap-media` | Media bucket |
| `MINIO_USE_SSL` | `false` | TLS for MinIO |
| `GLOBAL_WEBHOOK_URL` | `""` | Catch-all webhook |
| `WA_LOG_LEVEL` | `INFO` | whatsmeow log level |
| `LOG_LEVEL` | `info` | zerolog level |
| `ENVIRONMENT` | `development` | Env tag in logs |

## Make commands

| Command | Action |
|---|---|
| `make up` | Start Postgres, MinIO, NATS |
| `make dev` | Run `go run cmd/wzap/main.go` |
| `make build` | Compile to `bin/wzap` |
| `make tidy` | `go mod tidy` |
| `make install-tools` | Install `golangci-lint` |
| `make docs` | Generate Swagger docs |

## Health check

```
GET /health  (no auth)
```

Returns status of database, nats, minio. Any `false` → check `make up` or env vars.

## Session lifecycle

| Status | Connected | Meaning |
|---|---|---|
| `disconnected` | `0` | No active WA connection |
| `connecting` | `0` | Handshake or QR in progress |
| `connected` | `1` | Paired and connected |

## Common admin operations

**Create:** `POST /sessions` with admin `Authorization` header.

**Connect:** `POST /sessions/:sessionId/connect` → returns `PAIRING` or `CONNECTED`.

**QR:** `GET /sessions/:sessionId/qr` → poll until connected.

**Disconnect:** `POST /sessions/:sessionId/disconnect`.

**Delete:** `DELETE /sessions/:sessionId` (irreversible — unpair + delete device + remove DB row + cascade delete webhooks).

## Common failures

| Symptom | Fix |
|---|---|
| `401` on every request | Check `ADMIN_TOKEN` env var |
| `session not found` | `GET /sessions` to list |
| QR returns empty | Call `POST /:id/connect` first |
| Webhook not firing | Check `enabled` and `events` via `GET /:id/webhooks` |
| `client not connected` | Reconnect via `POST /:id/connect` |
| Health `database: false` | `make up` or check `DATABASE_URL` |

## Orphan device cleanup

`ReconnectAll` on startup deletes whatsmeow devices whose JID doesn't match any `wz_sessions` row.
