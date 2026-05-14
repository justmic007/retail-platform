# retail-platform

A production-grade microservices platform with a Next.js frontend.

## Structure

```
retail-platform/
├── backend/     # Go microservices — auth, order, inventory, notification
└── client/      # Next.js frontend
```

## Backend Services

| Service      | Port | Description                                      |
|--------------|------|--------------------------------------------------|
| Auth         | 8080 | JWT authentication, user management              |
| Order        | 8081 | Async order processing with worker pool          |
| Inventory    | 8082 | Product catalogue, stock reservation             |
| Notification | 8083 | Event-driven email notifications via Brevo       |

## Infrastructure

- **Databases** — Neon (serverless Postgres), one database per service
- **Cache / Events** — Upstash Redis (TLS), used for stock caching and Redis Pub/Sub event bus
- **Email** — Brevo transactional email API

## Quick Start

### Prerequisites

- Go 1.25+
- [overmind](https://github.com/DarthSim/overmind) — `brew install overmind`
- Environment files configured (see below)

### Run all services

```bash
cd backend
make run-all
```

This starts all four services via overmind with color-coded logs per service.

### Run services individually

```bash
cd backend
make run-auth         # terminal 1 — start first
make run-inventory    # terminal 2
make run-order        # terminal 3 (requires auth to be running)
make run-notification # terminal 4
```

### Environment setup

Each service has its own `.env` file:

```
backend/services/auth/.env
backend/services/inventory/.env
backend/services/order/.env
backend/services/notification/.env
```

Copy the example files and fill in your values:

```bash
cp services/auth/env.example services/auth/.env
cp services/inventory/env.example services/inventory/.env
cp services/order/env.example services/order/.env
cp services/notification/env.example services/notification/.env
```

### Database migrations

```bash
cd backend

# Local Postgres
make migrate-all

# Neon (production)
make migrate-neon-all \
  NEON_AUTH_URL=<auth-db-url> \
  NEON_ORDER_URL=<order-db-url> \
  NEON_INVENTORY_URL=<inventory-db-url>
```

## Frontend

```bash
cd client
npm install
npm run dev
```

## Testing

```bash
cd backend
make test           # all services
make test-race      # with race detector (recommended)
```

## Overmind tips

```bash
make run-all                  # start all services
overmind connect auth         # focus on auth logs
overmind connect order        # focus on order logs
overmind restart auth         # restart a single service
```

## Documentation

- [Backend README](./backend/README.md)
- [Auth Service](./backend/services/auth/README.md)
- [Order Service](./backend/services/order/README.md)
- [Inventory Service](./backend/services/inventory/README.md)
- [Notification Service](./backend/services/notification/README.md)
