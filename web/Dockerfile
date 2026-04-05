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
WORKDIR /app
COPY --from=web-builder /app/.output ./
ENV HOST=0.0.0.0 PORT=3000
EXPOSE 3000
CMD ["node", "server/index.mjs"]
