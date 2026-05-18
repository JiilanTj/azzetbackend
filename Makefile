.PHONY: build run-api run-worker run-migrate test lint tidy swag clean docker-up docker-down

APP_NAME   := azzet
BUILD_DIR  := ./bin
CMD_DIR    := ./cmd

# Build all binaries
build:
	@echo "Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/api    $(CMD_DIR)/api
	go build -o $(BUILD_DIR)/worker $(CMD_DIR)/worker
	go build -o $(BUILD_DIR)/migrate $(CMD_DIR)/migrate
	@echo "Binaries placed in $(BUILD_DIR)/"

# Run individual services
run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

run-migrate:
	go run ./cmd/migrate

# Test
test:
	go test -v -race -count=1 ./...

test-cover:
	go test -v -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

# Dependency management
tidy:
	go mod tidy
	go mod verify

# Swagger
swag:
	swag init -g cmd/api/main.go -o docs

# Clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# Development helpers
dev: docker-up run-api

.PHONY: $(MAKECMDGOALS)
