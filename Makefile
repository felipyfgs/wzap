# wzap Makefile

.PHONY: help dev build run tidy up down down-clean clean install-tools docs \
        web-install web-dev web-build \
        docker-dev docker-prod docker-build docker-down docker-logs \
        chatwoot-up chatwoot-down chatwoot-logs \
        push deploy deploy-pull deploy-status

# Variables
APP_NAME=wzap
BUILD_DIR=bin

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Go ───────────────────────────────────────────────────────────────────────

tidy: ## Tidy go modules
	go mod tidy

dev: ## Run the API locally
	go run cmd/wzap/main.go

build: ## Build the Go binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) cmd/wzap/main.go

run: build ## Build and run the Go binary
	./$(BUILD_DIR)/$(APP_NAME)

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)

install-tools: ## Install dev tools (golangci-lint, swag)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.11.4
	go install github.com/swaggo/swag/cmd/swag@latest

docs: ## Generate Swagger documentation
	swag init -g main.go -o docs --parseInternal --useStructName \
		-d cmd/wzap,internal/handler,internal/dto,internal/model,internal/service,internal/repo

# ─── Frontend ─────────────────────────────────────────────────────────────────

web-install: ## Install frontend dependencies
	cd web && pnpm install

web-dev: ## Run the frontend dev server
	cd web && pnpm dev

web-build: ## Build the frontend for production
	cd web && pnpm build


# ─── Docker ───────────────────────────────────────────────────────────────────
# Infra (postgres, minio, nats) é sempre docker-compose.yml
# App: docker-compose.dev.yml  → hot reload (air + nuxt dev)
#      docker-compose.prod.yml → imagem combinada compilada

COMPOSE_INFRA = docker compose -f docker-compose.yml
COMPOSE_DEV   = $(COMPOSE_INFRA) -f docker-compose.dev.yml
COMPOSE_PROD  = $(COMPOSE_INFRA) -f docker-compose.prod.yml

docker-dev: ## Sobe infra + API (air) + Web (nuxt dev) com hot reload
	$(COMPOSE_DEV) up -d --build --remove-orphans

docker-prod: ## Sobe infra + imagem combinada (produção)
	$(COMPOSE_PROD) up -d --remove-orphans

docker-build: ## Builda a imagem combinada de produção
	./scripts/setup.sh

docker-down: ## Para todos os containers (dev ou prod)
	$(COMPOSE_DEV) down --remove-orphans 2>/dev/null; \
	$(COMPOSE_PROD) down --remove-orphans 2>/dev/null; true

docker-down-v: ## Para todos os containers e remove volumes (DESTRUTIVO)
	$(COMPOSE_DEV) down -v --remove-orphans 2>/dev/null; \
	$(COMPOSE_PROD) down -v --remove-orphans 2>/dev/null; true

logs: ## Logs do app em tempo real
	$(COMPOSE_DEV) logs -f api web 2>/dev/null || $(COMPOSE_PROD) logs -f app

# ─── Docker Push ──────────────────────────────────────────────────────────────

push: ## Builda e faz push da imagem combinada para Docker Hub
	./scripts/setup.sh --push

# ─── Docker Swarm Deploy ─────────────────────────────────────────────────────

deploy: ## Deploy stack to Docker Swarm
	./scripts/deploy.sh

deploy-pull: ## Pull latest images then deploy
	./scripts/deploy.sh --pull

deploy-status: ## Show current Swarm stack status
	./scripts/deploy.sh --status

# ─── Chatwoot ─────────────────────────────────────────────────────────────────

chatwoot-up: ## Start Chatwoot stack
	docker compose -f docker/chatwoot/docker-compose.yml up -d

chatwoot-down: ## Stop Chatwoot stack
	docker compose -f docker/chatwoot/docker-compose.yml down

chatwoot-logs: ## Tail Chatwoot logs
	docker compose -f docker/chatwoot/docker-compose.yml logs -f rails
