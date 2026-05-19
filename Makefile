.PHONY: build run-api run-worker run-migrate test test-unit test-api test-cover test-short lint tidy swag sqlc clean docker-up docker-down docker-logs docker-reset dev migrate seed

APP_NAME   := azzet
BUILD_DIR  := ./bin
CMD_DIR    := ./cmd

# ─────────────────────────────────────────────────────────────────
# Build
# ─────────────────────────────────────────────────────────────────

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/api     $(CMD_DIR)/api
	go build -o $(BUILD_DIR)/worker  $(CMD_DIR)/worker
	go build -o $(BUILD_DIR)/migrate $(CMD_DIR)/migrate
	@echo "Binaries placed in $(BUILD_DIR)/"

build-api:
	go build -o $(BUILD_DIR)/api $(CMD_DIR)/api

build-worker:
	go build -o $(BUILD_DIR)/worker $(CMD_DIR)/worker

# ─────────────────────────────────────────────────────────────────
# Run
# ─────────────────────────────────────────────────────────────────

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

run-migrate:
	go run ./cmd/migrate

# ─────────────────────────────────────────────────────────────────
# Test
# ─────────────────────────────────────────────────────────────────

# Run all tests
test:
	go test -race -count=1 ./tests/...

# Run unit tests only
test-unit:
	go test -race -count=1 ./tests/unit/...

# Run API tests only
test-api:
	go test -race -count=1 ./tests/api/...

# Run tests with short flag (skip integration tests)
test-short:
	go test -race -count=1 -short ./tests/...

# Run tests verbose
test-v:
	go test -v -race -count=1 ./tests/...

# Run tests with coverage report
test-cover:
	go test -v -race -count=1 -coverprofile=coverage.out ./tests/... ./internal/...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests for a specific package (usage: make test-pkg PKG=./tests/unit/shared)
test-pkg:
	go test -v -race -count=1 $(PKG)

# ─────────────────────────────────────────────────────────────────
# Code Quality
# ─────────────────────────────────────────────────────────────────

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

fmt:
	gofmt -w .

tidy:
	go mod tidy
	go mod verify

check: lint test
	@echo "All checks passed!"

# ─────────────────────────────────────────────────────────────────
# Code Generation
# ─────────────────────────────────────────────────────────────────

sqlc:
	sqlc generate
	@echo "SQLC code generated in internal/db/"

swag:
	swag init -g cmd/api/main.go -o docs
	@echo "Swagger docs generated in docs/"

generate: sqlc swag
	@echo "All code generation complete"

# ─────────────────────────────────────────────────────────────────
# Database
# ─────────────────────────────────────────────────────────────────

migrate:
	go run ./cmd/migrate

# ─────────────────────────────────────────────────────────────────
# Docker
# ─────────────────────────────────────────────────────────────────

docker-up:
	docker compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@docker compose ps

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-reset:
	docker compose down -v
	docker compose up -d
	@echo "All volumes removed and services restarted"

docker-ps:
	docker compose ps

# ─────────────────────────────────────────────────────────────────
# Development
# ─────────────────────────────────────────────────────────────────

# Start everything for development
dev: docker-up migrate run-api

# Start worker in development
dev-worker: docker-up run-worker

# Fresh start: reset docker, migrate, run
fresh: docker-reset migrate run-api

# Install development tools
tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Development tools installed"

# ─────────────────────────────────────────────────────────────────
# Clean
# ─────────────────────────────────────────────────────────────────

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# ─────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────

help:
	@echo ""
	@echo "  Azzet Backend - Available Commands"
	@echo "  ─────────────────────────────────────────────"
	@echo ""
	@echo "  Build:"
	@echo "    make build          Build all binaries"
	@echo "    make build-api      Build API binary only"
	@echo "    make build-worker   Build worker binary only"
	@echo ""
	@echo "  Run:"
	@echo "    make run-api        Run API server"
	@echo "    make run-worker     Run background worker"
	@echo "    make run-migrate    Run database migrations"
	@echo ""
	@echo "  Test:"
	@echo "    make test           Run all tests"
	@echo "    make test-unit      Run unit tests only"
	@echo "    make test-api       Run API tests only"
	@echo "    make test-cover     Run tests with coverage report"
	@echo "    make test-short     Run tests (skip integration)"
	@echo "    make test-pkg PKG=  Run tests for specific package"
	@echo ""
	@echo "  Code Quality:"
	@echo "    make lint           Run linter and format check"
	@echo "    make fmt            Format all Go files"
	@echo "    make tidy           Tidy and verify go.mod"
	@echo "    make check          Run lint + test"
	@echo ""
	@echo "  Code Generation:"
	@echo "    make sqlc           Generate SQLC code"
	@echo "    make swag           Generate Swagger docs"
	@echo "    make generate       Run all code generation"
	@echo ""
	@echo "  Database:"
	@echo "    make migrate        Run database migrations"
	@echo ""
	@echo "  Docker:"
	@echo "    make docker-up      Start all services"
	@echo "    make docker-down    Stop all services"
	@echo "    make docker-logs    Follow service logs"
	@echo "    make docker-reset   Reset volumes and restart"
	@echo "    make docker-ps      Show service status"
	@echo ""
	@echo "  Development:"
	@echo "    make dev            Start docker + migrate + API"
	@echo "    make dev-worker     Start docker + worker"
	@echo "    make fresh          Reset everything and start fresh"
	@echo "    make tools          Install dev tools (sqlc, swag)"
	@echo ""
	@echo "  Clean:"
	@echo "    make clean          Remove build artifacts"
	@echo ""
