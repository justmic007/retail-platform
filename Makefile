# ── retail-platform Makefile ──────────────────────────────────────────────────
# Run `make help` to see all available commands
.PHONY: help infra-up infra-down run-auth run-order run-inventory run-notification \
        test test-race lint migrate-auth migrate-order migrate-inventory tidy

# Default target — show help
help:
	@echo ""
	@echo "retail-platform — available commands:"
	@echo ""
	@echo "  Infrastructure:"
	@echo "    make infra-up         Start Postgres + Redis (without services)"
	@echo "    make infra-down       Stop all containers"
	@echo "    make infra-reset      Stop + delete volumes (fresh database)"
	@echo ""
	@echo "  Run services locally (requires infra-up first):"
	@echo "    make run-auth         Start auth service"
	@echo "    make run-order        Start order service"
	@echo "    make run-inventory    Start inventory service"
	@echo "    make run-notification Start notification service"
	@echo ""
	@echo "  Database migrations:"
	@echo "    make migrate-auth     Run auth DB migrations"
	@echo "    make migrate-order    Run order DB migrations"
	@echo "    make migrate-inventory Run inventory DB migrations"
	@echo "    make migrate-all      Run all migrations"
	@echo ""
	@echo "  Testing:"
	@echo "    make test             Run all tests"
	@echo "    make test-race        Run tests with race detector (IMPORTANT)"
	@echo "    make test-coverage    Run tests with coverage report"
	@echo ""
	@echo "  Code quality:"
	@echo "    make lint             Run golangci-lint"
	@echo "    make tidy             Run go mod tidy on all modules"
	@echo "    make fmt              Format all Go files"
	@echo ""

# ── Infrastructure ────────────────────────────────────────────────────────────

# Start only Postgres and Redis — not the Go services (we run those locally)
infra-up:
	docker compose up -d postgres redis
	@echo "Waiting for Postgres to be ready..."
	@sleep 3
	@echo "Infrastructure is up. Run 'make migrate-all' next."

infra-down:
	docker-compose down

# WARNING: this deletes all data
infra-reset:
	docker-compose down -v
	@echo "All volumes deleted. Fresh start."

# ── Run services ──────────────────────────────────────────────────────────────
# We use -race flag even in development to catch race conditions early

run-auth:
	cd services/auth && go run -race ./cmd/server/...

run-order:
	cd services/order && go run -race ./cmd/server/...

run-inventory:
	cd services/inventory && go run -race ./cmd/server/...

run-notification:
	cd services/notification && go run -race ./cmd/server/...

# ── Migrations ────────────────────────────────────────────────────────────────
# golang-migrate reads SQL files from infra/migrations/<service>/
# and applies them in order (001_create_users.up.sql, 002_add_index.up.sql, etc.)

POSTGRES_BASE=postgres://retail:retailsecret@localhost:5432

migrate-auth:
	migrate -path infra/migrations/auth -database "$(POSTGRES_BASE)/auth_db?sslmode=disable" up

migrate-order:
	migrate -path infra/migrations/order -database "$(POSTGRES_BASE)/order_db?sslmode=disable" up

migrate-inventory:
	migrate -path infra/migrations/inventory -database "$(POSTGRES_BASE)/inventory_db?sslmode=disable" up

migrate-all: migrate-auth migrate-order migrate-inventory
	@echo "All migrations applied."

# Rollback last migration (useful during development)
rollback-auth:
	migrate -path infra/migrations/auth -database "$(POSTGRES_BASE)/auth_db?sslmode=disable" down 1

# ── Testing ───────────────────────────────────────────────────────────────────

test:
	go test ./...

# -race detects race conditions in concurrent code — ALWAYS run this
# It's slower but catches bugs that only appear under concurrency
test-race:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: open coverage.html"

# ── Code quality ──────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

# gofmt formats code. In Go, formatting is not a style preference — it's enforced.
fmt:
	gofmt -s -w .

# go mod tidy removes unused dependencies and adds missing ones
tidy:
	cd pkg && go mod tidy
	cd services/auth && go mod tidy
	cd services/order && go mod tidy
	cd services/inventory && go mod tidy
	cd services/notification && go mod tidy

# ── Build ─────────────────────────────────────────────────────────────────────

build-all:
	cd services/auth && go build -o ../../bin/auth ./cmd/server/...
	cd services/order && go build -o ../../bin/order ./cmd/server/...
	cd services/inventory && go build -o ../../bin/inventory ./cmd/server/...
	cd services/notification && go build -o ../../bin/notification ./cmd/server/...
	@echo "All binaries built in bin/"