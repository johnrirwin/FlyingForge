# FlyingForge Makefile
# Run `make help` to see available commands

.PHONY: help test test-go test-web lint lint-go lint-web build build-go build-web run clean install rekognition-test

# Default target
.DEFAULT_GOAL := help

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

## Help
help: ## Show this help message
	@echo "$(CYAN)FlyingForge Development Commands$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'

## Installation
install: install-go install-web ## Install all dependencies

install-go: ## Install Go dependencies
	@echo "$(CYAN)Installing Go dependencies...$(RESET)"
	cd server && go mod download

install-web: ## Install frontend dependencies
	@echo "$(CYAN)Installing frontend dependencies...$(RESET)"
	cd web && npm install

## Testing
test: test-go test-web ## Run all tests

test-go: ## Run Go tests
	@echo "$(CYAN)Running Go tests...$(RESET)"
	cd server && go test -v ./...

test-go-cover: ## Run Go tests with coverage
	@echo "$(CYAN)Running Go tests with coverage...$(RESET)"
	cd server && go test -v -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: server/coverage.html$(RESET)"

test-web: ## Run frontend tests
	@echo "$(CYAN)Running frontend tests...$(RESET)"
	cd web && npm run test -- --run

test-web-watch: ## Run frontend tests in watch mode
	cd web && npm run test

test-web-cover: ## Run frontend tests with coverage
	@echo "$(CYAN)Running frontend tests with coverage...$(RESET)"
	cd web && npm run test:ci

rekognition-test: ## Run Rekognition moderation test against a local image (IMAGE=./path.jpg)
	@echo "$(CYAN)Running Rekognition moderation test...$(RESET)"
	cd server && IMAGE=$(IMAGE) go run ./cmd/rekognition-test -image "$(IMAGE)"

## Linting
lint: lint-go lint-web ## Run all linters

lint-go: ## Run Go linter
	@echo "$(CYAN)Running golangci-lint...$(RESET)"
	cd server && golangci-lint run

lint-go-fix: ## Run Go linter with auto-fix
	@echo "$(CYAN)Running golangci-lint with auto-fix...$(RESET)"
	cd server && golangci-lint run --fix

lint-web: ## Run frontend linter
	@echo "$(CYAN)Running ESLint...$(RESET)"
	cd web && npm run lint

## Formatting
fmt: fmt-go ## Format all code

fmt-go: ## Format Go code
	@echo "$(CYAN)Formatting Go code...$(RESET)"
	cd server && go fmt ./...

## Building
build: build-go build-web ## Build all

build-go: ## Build Go server
	@echo "$(CYAN)Building Go server...$(RESET)"
	cd server && go build -o bin/server ./cmd/server

build-web: ## Build frontend
	@echo "$(CYAN)Building frontend...$(RESET)"
	cd web && npm run build

## Running
run-server: ## Run the Go server
	@echo "$(CYAN)Starting Go server...$(RESET)"
	cd server && go run ./cmd/server

run-web: ## Run the frontend dev server
	@echo "$(CYAN)Starting frontend dev server...$(RESET)"
	cd web && npm run dev

## Docker
docker-up: ## Start all services with Docker Compose
	@echo "$(CYAN)Starting Docker services...$(RESET)"
	docker-compose up -d

docker-down: ## Stop all Docker services
	@echo "$(CYAN)Stopping Docker services...$(RESET)"
	docker-compose down

docker-build: ## Build Docker images
	@echo "$(CYAN)Building Docker images...$(RESET)"
	docker-compose build

docker-logs: ## Show Docker logs
	docker-compose logs -f

## Cleanup
clean: ## Clean build artifacts
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	rm -rf server/bin
	rm -rf server/coverage.out server/coverage.html
	rm -rf web/dist
	rm -rf web/coverage

## Type checking
typecheck: ## Run TypeScript type checking
	@echo "$(CYAN)Running TypeScript type check...$(RESET)"
	cd web && npx tsc --noEmit

## CI simulation
ci: lint test build ## Run full CI pipeline locally
	@echo "$(GREEN)✓ CI pipeline passed$(RESET)"
