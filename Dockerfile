# syntax=docker/dockerfile:1.7
# =============================================================================
# wzap — Dockerfile multi-stage (API Go + Web Nuxt)
# =============================================================================
# Targets:
#   web-dev      Nuxt dev server (hot reload)          → porta 3000
#   web-prod     Nuxt Nitro output (node server)       → porta 3000
#   api-dev      API Go com air (hot reload)           → porta 8080
#   api-prod     API Go compilada (runtime mínimo)     → porta 8080
#   combined     API + Web numa imagem só              → portas 8080 + 3000
# =============================================================================

ARG GO_VERSION=1.25
ARG NODE_VERSION=22
ARG PNPM_VERSION=10.33.0
ARG ALPINE_VERSION=3.21

# =============================================================================
#                                  WEB
# =============================================================================

# ── web-base: Node + pnpm ────────────────────────────────────────────────────
FROM node:${NODE_VERSION}-alpine AS web-base
ARG PNPM_VERSION
ENV CI=1 \
    PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH
RUN npm install -g pnpm@${PNPM_VERSION}
WORKDIR /web

# ── web-deps: resolve dependências (cache layer) ─────────────────────────────
FROM web-base AS web-deps
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN --mount=type=cache,id=wzap-pnpm,target=/pnpm/store \
    pnpm install --frozen-lockfile

# ── web-dev: Nuxt dev server (hot reload) ────────────────────────────────────
FROM web-deps AS web-dev
ENV NODE_ENV=development \
    HOST=0.0.0.0 \
    PORT=3000
EXPOSE 3000
# usa o binário do nuxt diretamente p/ evitar o `--dotenv ../.env` do script npm
CMD ["pnpm", "exec", "nuxt", "dev", "--host", "0.0.0.0"]

# ── web-builder: build de produção (.output) ─────────────────────────────────
FROM web-deps AS web-builder
COPY web/ ./
# nota: NÃO usar cache mount em /web/.nuxt — o postinstall já gerou tsconfigs
# necessários ali e um cache mount os sobrescreveria com um volume vazio.
RUN pnpm exec nuxt build

# ── web-prod: runtime mínimo (node server) ───────────────────────────────────
FROM node:${NODE_VERSION}-alpine AS web-prod
RUN apk add --no-cache wget \
    && addgroup -S wzap && adduser -S wzap -G wzap -h /app
WORKDIR /app
COPY --chown=wzap:wzap --from=web-builder /web/.output ./
USER wzap
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000 \
    NITRO_PORT=3000
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${PORT}/health || exit 1
CMD ["node", "server/index.mjs"]

# =============================================================================
#                                   API
# =============================================================================

# ── api-base: toolchain Go ───────────────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine AS api-base
# ffmpeg é necessário em runtime (conversão de áudio p/ OGG Opus antes
# de enviar ao WhatsApp). Adicionado aqui para cobrir tanto o api-dev
# (que herda de api-deps → api-base) quanto o api-builder.
RUN apk add --no-cache git ca-certificates tzdata ffmpeg
WORKDIR /app

# ── api-deps: baixa módulos Go (cache layer) ─────────────────────────────────
FROM api-base AS api-deps
COPY go.mod go.sum ./
RUN --mount=type=cache,id=wzap-gomod,target=/go/pkg/mod \
    go mod download

# ── api-dev: hot reload via air ──────────────────────────────────────────────
FROM api-deps AS api-dev
RUN --mount=type=cache,id=wzap-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=wzap-gobuild,target=/root/.cache/go-build \
    go install github.com/air-verse/air@latest
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=debug \
    ENVIRONMENT=development
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ── api-builder: gera docs Swagger + compila binário ────────────────────────
FROM api-deps AS api-builder
RUN --mount=type=cache,id=wzap-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=wzap-gobuild,target=/root/.cache/go-build \
    go install github.com/swaggo/swag/cmd/swag@v1.16.6
COPY . .
RUN swag init -g main.go -o docs --parseInternal --useStructName \
        -d cmd/wzap,internal/handler,internal/dto,internal/model,internal/service,internal/repo
RUN --mount=type=cache,id=wzap-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=wzap-gobuild,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /out/wzap cmd/wzap/main.go

# ── api-prod: runtime mínimo ────────────────────────────────────────────────
FROM alpine:${ALPINE_VERSION} AS api-prod
RUN apk add --no-cache ca-certificates tzdata wget ffmpeg \
    && addgroup -S wzap && adduser -S wzap -G wzap -h /app
WORKDIR /app
COPY --chown=wzap:wzap --from=api-builder /out/wzap /app/wzap
USER wzap
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${PORT}/health || exit 1
ENTRYPOINT ["/app/wzap"]

# =============================================================================
#                        COMBINED (API + WEB numa imagem)
# =============================================================================

FROM node:${NODE_VERSION}-alpine AS combined
RUN apk add --no-cache ca-certificates tzdata wget ffmpeg tini \
    && addgroup -S wzap && adduser -S wzap -G wzap -h /app
WORKDIR /app
COPY --chown=wzap:wzap --from=api-builder  /out/wzap      ./api
COPY --chown=wzap:wzap --from=web-builder  /web/.output   ./web
USER wzap
ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production \
    HOST=0.0.0.0 \
    WEB_PORT=3000 \
    NITRO_PORT=3000
EXPOSE 8080 3000
HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${PORT}/health || exit 1
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["sh", "-c", "./api & API=$!; PORT=${WEB_PORT} node web/server/index.mjs & WEB=$!; trap 'kill $API $WEB 2>/dev/null' TERM INT; wait -n $API $WEB; kill $API $WEB 2>/dev/null; wait"]
