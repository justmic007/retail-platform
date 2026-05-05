# retail-platform

A production-grade microservices platform built in Go. Covers authentication, order management, inventory, and notifications — each as an independent service with its own database, deployed via Docker.

---

## Services

| Service | Port | Status | Description |
|---|---|---|---|
| [auth](./services/auth/README.md) | 8080 | ✅ Complete | JWT authentication — register, login, refresh, logout |
| order | 8081 | 🚧 In progress | Order creation and management |
| inventory | 8082 | 🚧 In progress | Stock levels with Redis caching |
| notification | 8083 | 🚧 In progress | Email/SMS notifications via event bus |

---

## Architecture

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│    auth     │   │    order    │   │  inventory  │   │notification │
│  :8080      │   │  :8081      │   │  :8082      │   │  :8083      │
└──────┬──────┘   └──────┬──────┘   └──────┬──────┘   └──────┬──────┘
       │                 │                 │                 │
       └─────────────────┴────────┬────────┘                 │
                                  │                          │
                         ┌────────▼────────┐        ┌────────▼────────┐
                         │   PostgreSQL    │        │   Event Bus     │
                         │  (3 databases) │        │ (Go channels)   │
                         └─────────────────┘        └─────────────────┘
                                  │
                         ┌────────▼────────┐
                         │     Redis       │
                         │  (stock cache)  │
                         └─────────────────┘
```

All services share a `pkg/` module for common code — JWT, middleware, logging, validation, and error types.

---

## Quick Start

**Prerequisites:** Go 1.25+, Docker Desktop, [golang-migrate](https://github.com/golang-migrate/migrate)

```bash
# 1. Start Postgres + Redis
make infra-up

# 2. Run migrations
make migrate-all

# 3. Copy environment files
cp services/auth/.env.example services/auth/.env

# 4. Start a service
make run-auth
```

---

## Testing the API

A shared Postman environment and per-service collections are included.

**To use them:**
1. Open Postman → **Import** → select `postman_environment.json`
2. Import `services/auth/postman_collection.json`
3. Import `services/inventory/postman_collection.json`
4. Select **Retail Platform — Local** as the active environment
5. Start the services, run **Register** then **Login** in the auth collection — tokens are saved to the environment automatically and all other collections pick them up

---

## Development

```bash
make test-race       # run all tests with race detector
make lint            # run golangci-lint across all services
make build-all       # build all service binaries to bin/
make infra-reset     # wipe all data and start fresh
```

---

## Project Structure

```
retail-platform/
├── pkg/                    # shared code used by all services
│   ├── errors/             # domain error types
│   ├── events/             # event bus (order → notification)
│   ├── jwt/                # token generation and validation
│   ├── logger/             # structured logging (zerolog)
│   ├── middleware/         # auth + request ID middleware
│   └── validator/          # request validation
├── services/
│   ├── auth/               # authentication service
│   ├── order/              # order service
│   ├── inventory/          # inventory service
│   └── notification/       # notification service
├── infra/
│   ├── docker/             # Dockerfiles + init scripts
│   └── migrations/         # SQL migrations per service
├── go.work                 # Go workspace linking all modules
└── docker-compose.yml      # local infrastructure
```
