# wzap Makefile

.PHONY: $(shell grep -E '^[a-zA-Z_-]+:' Makefile | cut -d: -f1)

APP_NAME = wzap
BUILD_DIR = bin

COMPOSE_INFRA = docker compose -f docker-compose.yml
COMPOSE_DEV   = $(COMPOSE_INFRA) -f docker-compose.dev.yml
COMPOSE_PROD  = $(COMPOSE_INFRA) -f docker-compose.prod.yml

help: ## Mostra este menu
	@echo ""
	@echo "  \033[1mGo\033[0m"
	@grep -E '^(dev|build|tidy|clean|docs|install-tools):.*## ' Makefile | awk 'BEGIN{FS=":.*## "}{printf "    \033[36m%-18s\033[0m %s\n",$$1,$$2}'
	@echo ""
	@echo "  \033[1mFrontend\033[0m"
	@grep -E '^web-.*:.*## ' Makefile | awk 'BEGIN{FS=":.*## "}{printf "    \033[36m%-18s\033[0m %s\n",$$1,$$2}'
	@echo ""
	@echo "  \033[1mDocker\033[0m"
	@grep -E '^(docker-.*|logs.*|push):.*## ' Makefile | awk 'BEGIN{FS=":.*## "}{printf "    \033[36m%-18s\033[0m %s\n",$$1,$$2}'
	@echo ""
	@echo "  \033[1mChatwoot\033[0m"
	@grep -E '^chatwoot-.*:.*## ' Makefile | awk 'BEGIN{FS=":.*## "}{printf "    \033[36m%-18s\033[0m %s\n",$$1,$$2}'
	@echo ""

# ─── Go ───────────────────────────────────────────────────────────────────────

dev: ## Roda a API localmente (go run)
	go run cmd/wzap/main.go

build: ## Compila o binário Go
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) cmd/wzap/main.go

tidy: ## go mod tidy
	go mod tidy

clean: ## Remove artefatos de build
	@rm -rf $(BUILD_DIR)

docs: ## Gera documentação Swagger
	swag init -g main.go -o docs --parseInternal --useStructName \
		-d cmd/wzap,internal

install-tools: ## Instala ferramentas de dev (golangci-lint, swag)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.11.4
	go install github.com/swaggo/swag/cmd/swag@latest

# ─── Frontend ─────────────────────────────────────────────────────────────────

web-install: ## Instala dependências do frontend
	cd web && pnpm install

web-dev: ## Roda o frontend em modo dev (fora do Docker)
	cd web && pnpm dev

web-build: ## Builda o frontend para produção
	cd web && pnpm build

# ─── Docker ───────────────────────────────────────────────────────────────────

docker-network-init: ## Cria rede compartilhada `wzap_chatwoot` (idempotente)
	@if docker network inspect wzap_chatwoot >/dev/null 2>&1; then \
		echo "✔ Rede wzap_chatwoot já existe"; \
	else \
		docker network create wzap_chatwoot >/dev/null && \
		echo "✔ Rede wzap_chatwoot criada"; \
	fi

docker-dev: docker-network-init ## Sobe infra + api + web com hot reload (air + nuxt dev)
	$(COMPOSE_DEV) up -d --build --remove-orphans

docker-prod: docker-network-init ## Sobe infra + api + web em modo produção
	$(COMPOSE_PROD) up -d --build --remove-orphans

docker-build: ## Builda a imagem combinada (wzap:latest)
	./scripts/setup.sh

docker-build-split: ## Builda imagens separadas (wzap-api:latest + wzap-web:latest)
	./scripts/setup.sh --split

push: ## Build + push da imagem combinada para Docker Hub
	./scripts/setup.sh --push

logs: ## Logs em tempo real dos containers api+web (dev)
	$(COMPOSE_DEV) logs -f api web 2>/dev/null || $(COMPOSE_PROD) logs -f api web

logs-api: ## Logs em tempo real da API
	$(COMPOSE_DEV) logs -f api 2>/dev/null || $(COMPOSE_PROD) logs -f api

logs-web: ## Logs em tempo real do Web
	$(COMPOSE_DEV) logs -f web 2>/dev/null || $(COMPOSE_PROD) logs -f web

docker-down: ## Para todos os containers
	$(COMPOSE_DEV) down --remove-orphans 2>/dev/null; \
	$(COMPOSE_PROD) down --remove-orphans 2>/dev/null; true

docker-down-v: ## Para containers e remove volumes (DESTRUTIVO)
	$(COMPOSE_DEV) down -v --remove-orphans 2>/dev/null; \
	$(COMPOSE_PROD) down -v --remove-orphans 2>/dev/null; true

# ─── Chatwoot ─────────────────────────────────────────────────────────────────

chatwoot-up: docker-network-init ## Sobe o stack do Chatwoot
	docker compose -f docker/chatwoot/docker-compose.yml up -d

chatwoot-down: ## Para o stack do Chatwoot
	docker compose -f docker/chatwoot/docker-compose.yml down

chatwoot-logs: ## Logs do Chatwoot (rails)
	docker compose -f docker/chatwoot/docker-compose.yml logs -f rails
