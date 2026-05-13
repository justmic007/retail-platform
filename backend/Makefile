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
	docker compose down

# WARNING: this deletes all data
infra-reset:
	docker compose down -v
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

POSTGRES_BASE=postgres://retail:retailsecret@localhost:5433

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

rollback-inventory:
	migrate -path infra/migrations/inventory -database "$(POSTGRES_BASE)/inventory_db?sslmode=disable" down 1

# ── Neon migrations (production) ──────────────────────────────────────────────
# Usage: make migrate-neon-auth NEON_AUTH_URL=postgres://...
#        make migrate-neon-all NEON_AUTH_URL=postgres://... NEON_ORDER_URL=postgres://... NEON_INVENTORY_URL=postgres://...

migrate-neon-auth:
	migrate -path infra/migrations/auth -database "$(NEON_AUTH_URL)" up

migrate-neon-order:
	migrate -path infra/migrations/order -database "$(NEON_ORDER_URL)" up

migrate-neon-inventory:
	migrate -path infra/migrations/inventory -database "$(NEON_INVENTORY_URL)" up

migrate-neon-all: migrate-neon-auth migrate-neon-order migrate-neon-inventory
	@echo "All Neon migrations applied."

# ── Testing ───────────────────────────────────────────────────────────────────

test-auth:
	cd services/auth && go test -race ./... -v
	
test-inventory:
	cd services/inventory && go test -race ./... -v

test-order:
	cd services/order && go test -race ./... -v

test-notification:
	cd services/notification && go test -race ./... -v

test:
	cd pkg && go test ./...
	cd services/auth && go test ./...
	cd services/order && go test ./...
	cd services/inventory && go test ./...
	cd services/notification && go test ./...

# -race detects race conditions in concurrent code — ALWAYS run this
test-race:
	cd pkg && go test -race ./...
	cd services/auth && go test -race ./...
	cd services/order && go test -race ./...
	cd services/inventory && go test -race ./...
	cd services/notification && go test -race ./...


test-coverage:
	cd services/auth && go test -race -coverprofile=coverage.out ./...
	cd services/auth && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: open services/auth/coverage.html"


# ── Code quality ──────────────────────────────────────────────────────────────

lint:
	@grep -E '^\s+\.' go.work | tr -d ' \t' | sed 's|^\./||' | while read dir; do \
		echo "--- Linting $$dir ---"; \
		(cd $$dir && golangci-lint run --timeout=5m ./...); \
	done

# gofmt formats code. In Go, formatting is not a style preference — it's enforced.
fmt:
	gofmt -s -w .

# Add a dependency to the shared pkg/ module
# Usage: make get-pkg-dep DEP=github.com/google/uuid
get-pkg-dep:
	cd pkg && go get $(DEP) && go mod tidy && cd ..
	@echo "Run 'make tidy' to update all services"

# Add a dependency to a specific service
# Usage: make get-dep SERVICE=auth DEP=github.com/joho/godotenv
get-dep:
	cd services/$(SERVICE) && go get $(DEP) && go mod tidy && cd ../..
	@echo "Added $(DEP) to $(SERVICE) service"

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