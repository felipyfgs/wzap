# wzap Makefile

.PHONY: help dev build run tidy up down down-clean clean install-tools docs \
        web-install web-dev web-build dev-all prod logs \
        chatwoot-up chatwoot-down chatwoot-logs \
        build-api build-web build-all push deploy deploy-pull deploy-status

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

dev-all: ## Run backend and frontend concurrently
	make dev & make web-dev

# ─── Docker (local) ──────────────────────────────────────────────────────────

up: ## Start dev stack (hot reload)
	docker compose up -d --build

down: ## Stop dev stack
	docker compose down

down-clean: ## Stop dev stack and remove volumes (DESTRUCTIVE)
	docker compose down -v

prod: ## Start prod stack (compiled image)
	docker compose -f docker-compose.prod.yml up -d --build

logs: ## Tail wzap container logs
	docker compose logs -f wzap

# ─── Docker Build & Push ─────────────────────────────────────────────────────

build-api: ## Build API Docker image
	./scripts/build.sh api

build-web: ## Build Web Docker image
	./scripts/build.sh web

build-all: ## Build API + Web Docker images
	./scripts/build.sh all

push: ## Build and push all images to Docker Hub
	./scripts/build.sh all --push

push-api: ## Build and push API image to Docker Hub
	./scripts/build.sh api --push

push-web: ## Build and push Web image to Docker Hub
	./scripts/build.sh web --push

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
