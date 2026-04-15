# ── web-builder: compila o frontend Nuxt ─────────────────────────────────────
FROM node:22-alpine AS web-builder
RUN npm install -g pnpm@10.33.0
WORKDIR /web
COPY web/pnpm-lock.yaml web/package.json ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

# ── dev: hot reload via air ──────────────────────────────────────────────────
FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
CMD ["air", "-c", ".air.toml"]

# ── dev-combined: API (air) + Web (nuxt dev) numa imagem só ──────────────────
FROM base AS dev-combined
RUN apk add --no-cache nodejs npm ffmpeg \
    && go install github.com/air-verse/air@latest \
    && npm install -g pnpm@10.33.0
COPY go.mod go.sum ./
RUN go mod download
COPY web/pnpm-lock.yaml web/package.json ./web/
RUN cd web && pnpm install --frozen-lockfile
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=debug \
    ENVIRONMENT=development \
    HOST=0.0.0.0 \
    WEB_PORT=3000
EXPOSE 8080 3000
CMD ["sh", "-c", "air -c .air.toml & API=$!; cd web && PORT=${WEB_PORT} pnpm dev --host 0.0.0.0 & WEB=$!; wait $API $WEB"]

# ── builder: generate docs + compile binary ───────────────────────────────────
FROM base AS builder
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN swag init -g main.go -o docs --parseInternal --useStructName \
    -d cmd/wzap,internal/handler,internal/dto,internal/model,internal/service,internal/repo
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /app/wzap cmd/wzap/main.go

# ── prod: minimal runtime image ───────────────────────────────────────────────
FROM alpine:3.21 AS prod
RUN apk add --no-cache ca-certificates tzdata wget ffmpeg \
    && addgroup -S wzap && adduser -S wzap -G wzap -h /app
WORKDIR /app
COPY --chown=wzap:wzap --from=builder /app/wzap /app/wzap
USER wzap
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:${PORT}/health || exit 1
ENTRYPOINT ["/app/wzap"]

# ── combined: API Go + Web Nuxt numa imagem só ────────────────────────────────
FROM node:22-alpine AS combined
RUN apk add --no-cache ca-certificates tzdata wget ffmpeg \
    && addgroup -S wzap && adduser -S wzap -G wzap -h /app
WORKDIR /app

COPY --chown=wzap:wzap --from=builder /app/wzap       ./api
COPY --chown=wzap:wzap --from=web-builder /web/.output ./web

USER wzap
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production \
    HOST=0.0.0.0 \
    WEB_PORT=3000

EXPOSE 8080 3000
HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD wget -qO- http://localhost:${PORT}/health || exit 1

CMD ["sh", "-c", "./api & API=$!; PORT=${WEB_PORT} node web/server/index.mjs & WEB=$!; wait $API $WEB"]
