# wzap API

WhatsApp API - built in Go using Fiber, PostgreSQL, NATS, MinIO, and Whatsmeow.

## Structure

- `cmd/wzap/`: Application entry point.
- `internal/config/`: Loading `.env` into typed config.
- `internal/database/`: PostgreSQL pool wrapper & basic schema definitions.
- `internal/handler/`: HTTP standard controllers for API resources.
- `internal/middleware/`: Global HTTP interceptors (Logger, CORS, Auth, Recovery).
- `internal/model/`: Shared domain objects and payload structs.
- `internal/queue/`: Event dispatch broker (NATS Stream).
- `internal/server/`: HTTP Engine framework orchestrator + routes definitions.
- `internal/service/`: Business logical processors mapping APIs to whatsmeow engine logic.
- `internal/storage/`: S3 Bucket client for handling media parsing.

## Start

Make sure to provide all values at `.env` according to `.env.example`.

### Docker
```sh
make up # Stand up Postgres, MinIO, NATS
```

### Local
```sh
make dev # go run cmd/wzap/main.go
make build # compile
```

## Health

```
GET http://localhost:8080/health
```
