FROM node:22-alpine AS web-base
WORKDIR /app
RUN npm install -g pnpm@10.33.0

# ── dev: hot reload via nuxt dev ─────────────────────────────────────────────
FROM web-base AS web-dev
COPY pnpm-lock.yaml package.json ./
RUN pnpm install
CMD ["pnpm", "dev", "--host", "0.0.0.0"]

# ── builder: instala deps e compila ──────────────────────────────────────────
FROM web-base AS web-builder
COPY pnpm-lock.yaml package.json ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

# ── prod: serve o output estático via node ────────────────────────────────────
FROM node:22-alpine AS web-prod
RUN addgroup -S wzap && adduser -S wzap -G wzap
WORKDIR /app
COPY --chown=wzap:wzap --from=web-builder /app/.output ./
USER wzap
ENV HOST=0.0.0.0 PORT=3000
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:${PORT}/health 2>/dev/null || exit 1
CMD ["node", "server/index.mjs"]
