.PHONY: run build test test-unit test-integration mock swagger \
        migrate-up migrate-down migrate-create lint vet tidy docker-up docker-down

## ── Configuration ────────────────────────────────────────────────────────────
BINARY      := flowdo-api
BUILD_DIR   := ./bin
CMD_DIR     := ./cmd/api
MIGRATE_URL ?= $(shell grep DATABASE_DSN .env 2>/dev/null | cut -d= -f2-)
MIGRATIONS  := ./migrations

## ── Development ──────────────────────────────────────────────────────────────

# Run the API server (requires .env or env vars set)
run:
	go run $(CMD_DIR)/main.go

# Build a static binary
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

## ── Testing ──────────────────────────────────────────────────────────────────

# Run all tests
test: test-unit test-integration

# Run unit tests only (fast, no external deps)
test-unit:
	go test -v -race -count=1 ./internal/core/...

# Run integration tests (requires Docker for testcontainers)
test-integration:
	go test -v -race -count=1 -tags=integration ./internal/adapter/outbound/...

## ── Code generation ──────────────────────────────────────────────────────────

# Regenerate all mocks with mockery
mock:
	go generate ./internal/core/port/...

# Generate Swagger docs (requires swag CLI: go install github.com/swaggo/swag/cmd/swag@latest)
swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency

## ── Database migrations ──────────────────────────────────────────────────────

# Apply all pending migrations
migrate-up:
	migrate -path $(MIGRATIONS) -database "$(MIGRATE_URL)" up

# Roll back the last migration
migrate-down:
	migrate -path $(MIGRATIONS) -database "$(MIGRATE_URL)" down 1

# Create a new migration pair: make migrate-create name=add_index_to_flowdos
migrate-create:
	@[ -n "$(name)" ] || (echo "Usage: make migrate-create name=<migration_name>" && exit 1)
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

## ── Code quality ─────────────────────────────────────────────────────────────

# Run go vet
vet:
	go vet ./...

# Run golangci-lint (requires golangci-lint: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

# Tidy go.mod / go.sum
tidy:
	go mod tidy

## ── Docker ───────────────────────────────────────────────────────────────────

# Start all services (app + postgres + redis) in detached mode
docker-up:
	docker compose up --build -d

# Stop and remove all containers / volumes
docker-down:
	docker compose down -v
