# wzap Makefile

.PHONY: help dev build run tidy up down clean install-tools

# Variables
APP_NAME=wzap
BUILD_DIR=bin
DOCKER_IMAGE=wzap:latest

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## Tidy go modules
	go mod tidy

dev: ## Run the application
	go run cmd/wzap/main.go

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) cmd/wzap/main.go

run: build ## Build and run the application
	./$(BUILD_DIR)/$(APP_NAME)

up: ## Start all services via docker-compose
	docker compose up -d

down: ## Stop all services
	docker compose down

down-clean: ## Stop all services and remove volumes (DESTRUCTIVE)
	docker compose down -v

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)

install-tools: ## Install development tools (golangci-lint)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.11.4
