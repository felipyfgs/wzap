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

## Docker Image

### Build local image
```sh
make build
docker build -t wzap:latest .
```

### Pull from GitHub Container Registry
```sh
docker pull ghcr.io/<owner>/wzap:latest
```

### Run with Docker Compose (recommended)
```sh
make up # Start Postgres, MinIO, NATS and wzap API
```

### Run standalone container
```sh
docker run -d \
  --name wzap \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://USER:PASSWORD@host.docker.internal:5435/wzap?sslmode=disable" \
  -e NATS_URL="nats://host.docker.internal:4222" \
  -e MINIO_ENDPOINT="host.docker.internal:9010" \
  ghcr.io/<owner>/wzap:latest
```

## Health

```
GET http://localhost:8080/health
```
