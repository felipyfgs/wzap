FROM golang:1.25-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

# ── dev: hot reload via air ──────────────────────────────────────────────────
FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
CMD ["air", "-c", ".air.toml"]

# ── builder: generate docs + compile binary ───────────────────────────────────
FROM base AS builder
RUN go install github.com/swaggo/swag/cmd/swag@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN swag init -g main.go -o docs --parseInternal --useStructName \
    -d cmd/wzap,internal/handler,internal/dto,internal/model,internal/service,internal/repo
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /app/wzap cmd/wzap/main.go

# ── prod: minimal runtime image ───────────────────────────────────────────────
FROM alpine:3.21 AS prod
RUN apk add --no-cache ca-certificates tzdata curl ffmpeg
RUN addgroup -S wzap && adduser -S wzap -G wzap -h /app
WORKDIR /app
COPY --from=builder /app/wzap /app/wzap
RUN chown -R wzap:wzap /app
USER wzap
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:${PORT}/health || exit 1
ENTRYPOINT ["/app/wzap"]
