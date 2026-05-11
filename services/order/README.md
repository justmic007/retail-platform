# Order Service

Production-grade async order processing service built in Go. Handles order creation, stock reservation via Inventory Service, price snapshots, idempotency, and graceful shutdown. Orders are processed asynchronously by a fixed worker pool — HTTP handlers return 202 immediately while workers confirm orders in the background.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Request & Response Lifecycle](#request--response-lifecycle)
- [Order Status Lifecycle](#order-status-lifecycle)
- [Worker Pool Design](#worker-pool-design)
- [Order Processor — Failure Handling](#order-processor--failure-handling)
- [Idempotency](#idempotency)
- [Service-to-Service Authentication](#service-to-service-authentication)
- [Price Snapshot](#price-snapshot)
- [Event Bus](#event-bus)
- [Layer Breakdown](#layer-breakdown)
- [Database Schema](#database-schema)
- [API Reference](#api-reference)
- [Security Design Decisions](#security-design-decisions)
- [Running Locally](#running-locally)
- [Postman Collection](#postman-collection)
- [Running Tests](#running-tests)
- [Environment Variables](#environment-variables)
- [Key Design Decisions](#key-design-decisions)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        ORDER SERVICE                            │
│                        Port: 8081                               │
│                                                                 │
│  ┌──────────┐   ┌────────────┐   ┌──────────┐   ┌──────────┐   │
│  │  Router  │ → │ Middleware │ → │ Handler  │ → │ Service  │   │
│  │router.go │   │request_id  │   │order_    │   │order_    │   │
│  │          │   │auth.go     │   │handler.go│   │service.go│   │
│  └──────────┘   └────────────┘   └──────────┘   └────┬─────┘   │
│                                                       │         │
│                                              ┌────────▼──────┐  │
│                                              │  Worker Pool  │  │
│                                              │   pool.go     │  │
│                                              │ processor.go  │  │
│                                              └────┬──────────┘  │
└───────────────────────────────────────────────────┼─────────────┘
                                                    │
                    ┌───────────────────────────────┼──────────────────┐
                    │                               │                  │
         ┌──────────▼──────────┐      ┌─────────────▼──────┐  ┌───────▼──────┐
         │     PostgreSQL      │      │  Inventory Service  │  │  Event Bus   │
         │     order_db        │      │  GET /products/:id  │  │ (Go channel) │
         │  ┌──────────────┐   │      │  POST /reserve      │  └──────────────┘
         │  │    orders    │   │      └────────────────────┘
         │  ├──────────────┤   │
         │  │ order_items  │   │
         │  └──────────────┘   │
         └─────────────────────┘
```

---

## Request & Response Lifecycle

This section traces a complete request from the moment the client sends it to the moment it receives a response, and then follows the async worker processing.

### Example: POST /orders

#### Phase 1 — Network & Router

```
CLIENT (Postman, mobile app, browser)
│
│  POST /orders HTTP/1.1
│  Authorization: Bearer eyJhbGci...
│  Content-Type: application/json
│  {
│    "idempotency_key": "order-001",
│    "items": [
│      {"product_id": "a1b2c3d4-...", "quantity": 2}
│    ]
│  }
│
▼
TCP CONNECTION accepted on port 8081
│
▼
GIN ROUTER (internal/server/router.go)
│
│  Matches POST /orders
│  Middleware chain: [RequestID] → [Recovery] → [AuthMiddleware] → [Handler]
│
▼
```

#### Phase 2 — Middleware

```
MIDDLEWARE 1: RequestID
│  Generates X-Request-ID: "f47ac10b-58cc-4372..."
│  Set in context + response header
▼

MIDDLEWARE 2: AuthMiddleware (pkg/middleware/auth.go)
│  Extracts Bearer token from Authorization header
│  Validates JWT signature and expiry
│  Sets user_id and role in Gin context
│  If invalid → 401 Unauthorized, request stops here
▼
```

#### Phase 3 — Handler

```
HANDLER: OrderHandler.CreateOrder (internal/handler/order_handler.go)
│
│  Step 1 — Extract user_id from JWT context
│  Step 2 — Bind JSON body into CreateOrderRequest struct
│  Step 3 — Validate fields:
│    idempotency_key: required
│    items:           required, min=1
│    items[].product_id: required
│    items[].quantity:   required, min=1
│  Step 4 — Call service.CreateOrder(ctx, userID, req)
│  Step 5 — Return 202 Accepted immediately
│
▼
```

#### Phase 4 — Service

```
SERVICE: OrderService.CreateOrder (internal/service/order_service.go)
│
│  Step 1 — Check idempotency key
│  │   repo.FindByIdempotencyKey(key, userID)
│  │   If found → return existing order (no duplicate processing)
│  │   If not found → proceed
│
│  Step 2 — Build order domain object
│  │   Status: PENDING
│  │   Items: product_id + quantity only (no prices yet)
│  │   TotalAmount: 0 (calculated by worker)
│
│  Step 3 — Save order to Postgres
│  │   repo.Create(ctx, order)
│  │   Order + items inserted in a single transaction
│
│  Step 4 — Submit to worker pool
│  │   pool.Submit(Job{OrderID: order.ID})
│  │   Non-blocking — returns ErrPoolFull if queue is full
│
│  Step 5 — Return order with PENDING status
│  │   Handler returns 202 immediately
│  │   Client polls GET /orders/:id for final status
│
▼
```

#### Phase 5 — Worker Pool

```
WORKER POOL (internal/worker/pool.go)
│
│  Job arrives on buffered channel
│  One of N worker goroutines picks it up
│  Calls processor.ProcessOrder(ctx, orderID)
│
▼
```

#### Phase 6 — Order Processor

```
PROCESSOR: OrderProcessor.ProcessOrder (internal/worker/processor.go)
│
│  Step 1 — Update status → PROCESSING
│
│  Step 2 — Fetch order from Postgres (FindByIDInternal — no ownership check)
│
│  Step 3 — For each item: call Inventory Service
│  │   GET /products/:id → fetches authoritative price + product name
│  │   Snapshots: item.ProductName = product.Name
│  │              item.UnitPrice   = product.Price
│  │              item.TotalPrice  = price × quantity
│  │   Accumulates totalAmount
│
│  Step 4 — Persist price snapshots
│  │   repo.UpdateItems(ctx, order.Items)
│  │   Writes product_name, unit_price, total_price to order_items
│
│  Step 5 — Reserve stock for each item
│  │   POST /inventory/reserve for each product
│  │   Tracks successfully reserved items
│  │   On failure → releaseAll() + failOrder()
│
│  Step 6 — Update status → CONFIRMED with total_amount + publish event
│  │   repo.UpdateStatusAndTotal(ctx, orderID, CONFIRMED, total)
│  │   Publish OrderConfirmed event
│
▼
```

#### Phase 7 — Repository

```
REPOSITORY (internal/repository/postgres.go)
│
│  This is the ONLY file that contains SQL.
│
│  UpdateStatusAndTotal:
│  │   UPDATE orders SET status = $1, total_amount = $2 WHERE id = $3
│
│  UpdateItems:
│  │   UPDATE order_items
│  │   SET product_name = $1, unit_price = $2, total_price = $3
│  │   WHERE id = $4
│  │   (one UPDATE per item)
│
▼
```

#### Phase 8 — Response flows back up

```
Client received 202 immediately after Phase 4.
Worker completes processing asynchronously.

Client polls GET /orders/:id:

  HTTP/1.1 200 OK
  {
    "order": {
      "id": "c5a465e1-...",
      "status": "CONFIRMED",
      "total_amount": "254.95",
      "items": [
        {
          "product_name": "Sunflower Oil 2L",
          "quantity": 2,
          "unit_price": "89.99",
          "total_price": "179.98"
        }
      ]
    }
  }
```

---

## Order Status Lifecycle

```
                    POST /orders
                         │
                         ▼
                      PENDING ──────────────────────► CANCELLED
                         │                         (PATCH /cancel)
                         │ worker picks up job
                         ▼
                     PROCESSING
                    /          \
     stock reserved             stock reservation fails
     prices fetched             or product not found
           │                           │
           ▼                           ▼
       CONFIRMED                     FAILED
    (terminal state)            (terminal state)
                                stock released
```

**Rules:**
- Only `PENDING` orders can be cancelled
- `CONFIRMED`, `FAILED`, and `CANCELLED` are terminal — no further transitions
- `PROCESSING` orders are mid-flight — cannot be cancelled

---

## Worker Pool Design

```
HTTP Handler
│
│  pool.Submit(Job{OrderID: "..."})
│  ├── channel not full → job queued, return nil
│  └── channel full    → return ErrPoolFull (503 to client)
│      Never blocks the HTTP handler goroutine
│
▼
Buffered Job Channel (size = WORKER_POOL_SIZE × 2)
│
├── Worker goroutine 0 ──► ProcessOrder()
├── Worker goroutine 1 ──► ProcessOrder()
├── Worker goroutine 2 ──► ProcessOrder()
│   ...
└── Worker goroutine N ──► ProcessOrder()

Shutdown sequence (on SIGTERM):
  1. HTTP server stops accepting requests
  2. pool.Shutdown() — closes job channel
  3. Workers drain remaining jobs
  4. WaitGroup waits for all workers to finish
  5. DB pool closed
  No orders lost mid-processing
```

**Why a fixed pool and not a goroutine per request?**
A goroutine per request means unbounded concurrency — under load, thousands of goroutines all call Inventory Service simultaneously, exhausting its connection pool. A fixed pool of N workers means at most N concurrent calls to Inventory Service, regardless of request volume. Backpressure is explicit via ErrPoolFull.

---

## Order Processor — Failure Handling

```
ProcessOrder fails at step 5 (Reserve stock for item 3 of 4):

  Item 1: reserved ✓
  Item 2: reserved ✓
  Item 3: reservation failed (insufficient stock)

  releaseAll() called:
    POST /inventory/release for item 1
    POST /inventory/release for item 2
    Item 3 was never reserved — nothing to release

  failOrder() called:
    UPDATE orders SET status = 'FAILED'
    Publish OrderFailed event

  Result: no stock permanently locked by a failed order
```

If `releaseAll` itself fails (Inventory Service down), the error is logged with `"manual intervention may be needed"` — the stock remains reserved until an operator or reconciliation job cleans it up. This is the correct trade-off: a stuck reservation is recoverable; silently losing a customer's order is not.

---

## Idempotency

Every order request requires an `idempotency_key`. This prevents duplicate orders from network retries or double-clicks.

```
First request:
  POST /orders {"idempotency_key": "order-001", ...}
  → order created, saved as PENDING, queued for processing
  → returns 202 with new order

Second request (same key, same user):
  POST /orders {"idempotency_key": "order-001", ...}
  → repo.FindByIdempotencyKey("order-001", userID) finds existing order
  → returns 202 with the SAME order (no duplicate created)
  → worker is NOT called again

Different user, same key:
  POST /orders {"idempotency_key": "order-001", ...} (different JWT)
  → idempotency key is scoped per user (WHERE idempotency_key=$1 AND user_id=$2)
  → treated as a new order — different user, different order
```

The idempotency key is enforced at the database level with a unique index on `(user_id, idempotency_key)`.

---

## Service-to-Service Authentication

Order Service calls Inventory Service on every order. Reserve and Release endpoints on Inventory Service require a JWT. Order Service authenticates using a dedicated service account.

```
Startup:
  ServiceTokenManager.NewServiceTokenManager(ctx, cfg)
  │
  │  POST /auth/login
  │  {"email": "order-service@internal.retailplatform.com",
  │   "password": "..."}
  │
  │  Stores: token + expiry
  │  Returns error if login fails — service refuses to start

Every Inventory Service call:
  tokenManager.GetToken(ctx)
  │
  ├── GetProduct  → GET /products/:id       (public route — token sent but not enforced)
  ├── Reserve     → POST /inventory/reserve (protected — token required)
  └── Release     → POST /inventory/release (protected — token required)

Token refresh:
  ├── token valid (> 1 minute remaining) → return cached token
  └── token expiring soon → POST /auth/login → refresh + return new token

Thread safety:
  sync.RWMutex protects token + expiry
  Multiple worker goroutines call GetToken() concurrently
  Read lock for cache check (non-blocking for concurrent reads)
  Write lock only during refresh
```

The service account is seeded in `infra/migrations/auth/004_seed_service_accounts.up.sql`. Its credentials are stored in environment variables — never hardcoded.

---

## Price Snapshot

```
Order placed at T=0:
  Sunflower Oil 2L → R89.99
  order_items.unit_price = 89.99  ← snapshot

Price changes at T=1 week:
  Sunflower Oil 2L → R99.99 (price increase)

Order viewed at T=2 weeks:
  order_items.unit_price = 89.99  ← still the original price
  Historical order is correct ✓

If we had joined to products table:
  order_items.product_id → products.price = 99.99
  Historical order shows wrong price ✗
```

**Why NUMERIC(10,2) not float64?**

```
float64:  89.99 × 3 = 269.97000000000003  ← floating point error
decimal:  89.99 × 3 = 269.97              ← exact
```

At millions of transactions, floating point errors accumulate into real financial discrepancies. `github.com/shopspring/decimal` is used throughout — domain types, repository, and processor all use `decimal.Decimal` for prices and totals.

---

## Event Bus

Order Service publishes events to the shared event bus for Notification Service to consume.

| Event | When published | Payload |
|---|---|---|
| `ORDER_CONFIRMED` | Worker confirms order | order_id, user_id, total |
| `ORDER_FAILED` | Worker fails order | order_id, user_id |
| `ORDER_CANCELLED` | User cancels order | order_id, user_id |

**Non-blocking publish:**
```go
select {
case eventBus.Orders <- event:
    // published
default:
    // channel full — skip, don't block the worker
}
```

Notifications are best-effort. If the channel is full, the event is dropped rather than blocking order processing. Stock accuracy is critical; notification latency is acceptable.

---

## Layer Breakdown

| Layer | File | Knows About | Does NOT Know About |
|---|---|---|---|
| Router | `internal/server/router.go` | HTTP routes, middleware chain | Business logic, SQL |
| Middleware | `pkg/middleware/` | HTTP headers, JWT validation | Business logic, SQL |
| Handler | `internal/handler/order_handler.go` | HTTP, JSON, validation | SQL, worker pool internals |
| Service | `internal/service/order_service.go` | Business rules, idempotency | HTTP, SQL syntax |
| Worker Pool | `internal/worker/pool.go` | Goroutines, channels, WaitGroup | Business logic, HTTP |
| Processor | `internal/worker/processor.go` | Order lifecycle, Inventory client | HTTP handlers, pool internals |
| Client | `internal/client/inventory_client.go` | HTTP calls to Inventory Service | Order business logic |
| Repository | `internal/repository/postgres.go` | SQL, pgx, Postgres | HTTP, worker pool |
| Database | PostgreSQL `order_db` | Tables, indexes, constraints | Go code |

**The rule:** each layer only communicates with the layer directly next to it. Handler never calls Repository. Service never reads HTTP headers. Processor never knows about Gin.

---

## Database Schema

### orders

```sql
CREATE TABLE IF NOT EXISTS orders (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID          NOT NULL,           -- no FK to users (cross-service)
    status          VARCHAR(50)   NOT NULL DEFAULT 'PENDING',
    total_amount    NUMERIC(10,2) NOT NULL DEFAULT 0, -- exact decimal, not float
    notes           TEXT,
    payment_status  VARCHAR(20)   NOT NULL DEFAULT 'UNPAID',
    idempotency_key VARCHAR(255)  NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
```

**Why no FK from user_id to users table?**
Order Service owns `order_db`. Auth Service owns `auth_db`. Cross-service FK constraints would couple two databases together — if Auth Service's database is unavailable, Order Service's migrations would fail. User identity is validated via JWT, not via database constraint.

### order_items

```sql
CREATE TABLE IF NOT EXISTS order_items (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID          NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   UUID          NOT NULL,              -- no FK to products (cross-service)
    product_name VARCHAR(255)  NOT NULL,              -- snapshot at order time
    quantity     INTEGER       NOT NULL CHECK (quantity > 0),
    unit_price   NUMERIC(10,2) NOT NULL CHECK (unit_price >= 0),  -- snapshot
    total_price  NUMERIC(10,2) NOT NULL CHECK (total_price >= 0), -- quantity × unit_price
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
```

### Indexes

| Index | Column(s) | Purpose |
|---|---|---|
| `idx_orders_user_id` | `user_id` | List user's orders |
| `idx_orders_status` | `status` | Filter by status |
| `idx_orders_created` | `created_at DESC` | Sort newest first |
| `idx_orders_user_created` | `user_id, created_at DESC` | User's orders sorted by date |
| `idx_orders_idempotency_key` | `user_id, idempotency_key` | Duplicate detection (UNIQUE) |
| `idx_order_items_order_id` | `order_id` | Fetch items for an order |
| `idx_order_items_product_id` | `product_id` | Analytics queries |

---

## API Reference

All endpoints require a valid JWT issued by Auth Service (`Authorization: Bearer <token>`).

### POST /orders

Creates a new order. Returns 202 immediately — processing is async.

**Request:**
```json
{
  "idempotency_key": "order-001",
  "items": [
    {"product_id": "a1b2c3d4-0001-0001-0001-000000000001", "quantity": 2},
    {"product_id": "a1b2c3d4-0003-0003-0003-000000000003", "quantity": 3}
  ],
  "notes": "Please handle with care"
}
```

**Response 202:**
```json
{
  "message": "order received and is being processed",
  "order": {
    "id": "c5a465e1-077c-4d55-8c59-8bb576a76b3c",
    "status": "PENDING",
    "total_amount": "0",
    "items": [...]
  }
}
```

**Errors:** `400` validation error · `401` missing or invalid token · `500` worker pool full

---

### GET /orders

Returns all orders for the authenticated user, newest first. Items are not included — use GET /orders/:id for full item details.

**Response 200:**
```json
{
  "orders": [
    {
      "id": "c5a465e1-...",
      "status": "CONFIRMED",
      "total_amount": "254.95",
      "created_at": "2026-05-11T07:38:09.568478+01:00"
    }
  ],
  "total": 1
}
```

---

### GET /orders/:id

Returns a single order with full item details including price snapshots.

**Response 200:**
```json
{
  "order": {
    "id": "c5a465e1-077c-4d55-8c59-8bb576a76b3c",
    "status": "CONFIRMED",
    "total_amount": "254.95",
    "items": [
      {
        "product_name": "Sunflower Oil 2L",
        "quantity": 2,
        "unit_price": "89.99",
        "total_price": "179.98"
      },
      {
        "product_name": "Full Cream Milk 1L",
        "quantity": 3,
        "unit_price": "24.99",
        "total_price": "74.97"
      }
    ]
  }
}
```

**Errors:** `404` order not found or belongs to different user · `401` missing or invalid token

---

### PATCH /orders/:id/cancel

Cancels a PENDING order. Only the order owner can cancel.

**Response 200:**
```json
{
  "message": "order cancelled successfully"
}
```

**Errors:** `400` order is not PENDING · `404` order not found or belongs to different user · `401` missing or invalid token

---

### GET /health · GET /ready

Kubernetes liveness and readiness probes.

---

## Security Design Decisions

### 1. Ownership enforced in SQL

```sql
SELECT ... FROM orders WHERE id = $1 AND user_id = $2
```

The `user_id` check is in the SQL `WHERE` clause — not in the service layer. A service-layer check would require two queries (find then check). The SQL clause does it in one query and prevents information leakage — user A doesn't even know user B's order exists (returns 404, not 403).

### 2. 404 not 403 for wrong-user access

Returning 403 Forbidden would confirm that the order exists but belongs to someone else. Returning 404 reveals nothing — the attacker cannot distinguish "order doesn't exist" from "order belongs to someone else".

### 3. No public routes

Every order endpoint requires a valid JWT. There is no unauthenticated access to any order data. Health and ready endpoints are the only public routes.

### 4. Service account scoped to minimum permissions

The Order Service service account has `role: customer` — the minimum role needed to call Inventory Service endpoints. It cannot access admin-only routes like `PATCH /products/:id/stock`.

---

## Running Locally

**Prerequisites:** Go 1.25+, Docker Desktop, golang-migrate

```bash
# 1. Start Postgres + Redis
make infra-up

# 2. Run all migrations (creates orders and order_items tables)
make migrate-all

# 3. Copy environment files
cp services/auth/.env.example services/auth/.env
cp services/inventory/.env.example services/inventory/.env
cp services/order/env.example services/order/.env

# 4. Start services (three terminals)
make run-auth         # terminal 1 — must start first
make run-inventory    # terminal 2
make run-order        # terminal 3
```

**Note:** Order Service logs in with its service account at startup. Auth Service must be running before Order Service starts.

---

## Postman Collection

A ready-to-use Postman collection is included at `postman_collection.json`.

**Import it:**
1. Open Postman → **Import** → select `postman_environment.json` from the repo root
2. Import `services/order/postman_collection.json`
3. Select **Retail Platform — Local** as the active environment
4. Run **Login** in the Auth Service collection first — `access_token` is saved automatically

**Recommended order:**
1. **Health Check** — verify service is up
2. **Create Order** — `order_id` is saved automatically
3. **Get Order by ID** — wait ~1 second, should show CONFIRMED with prices
4. **List Orders** — shows all your orders newest first
5. **Cancel Order** — returns 400 (order already CONFIRMED)
6. **Idempotency test** — same key returns same order
7. **Security Tests** folder — all should return errors

---

## Running Tests

```bash
# Order service tests with race detector
make test-order

# All services
make test-race
```

Unit tests use mock repository and mock inventory client — no Postgres or Inventory Service required. Tests run in milliseconds.

**What the tests cover:**
- `TestCreateOrder` — successful creation, idempotency
- `TestCancelOrder` — cancel PENDING, reject CONFIRMED, reject wrong user
- `TestGetOrder` — correct user, wrong user
- `TestWorkerPool_Submit` — jobs are processed
- `TestWorkerPool_ErrPoolFull` — full queue returns error
- `TestWorkerPool_GracefulShutdown` — all jobs complete before shutdown

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | Postgres connection string for order_db |
| `JWT_SECRET` | ✅ | — | Must match Auth Service JWT_SECRET exactly |
| `INVENTORY_SERVICE_URL` | ✅ | — | Base URL of Inventory Service |
| `AUTH_SERVICE_URL` | ✅ | — | Base URL of Auth Service |
| `ORDER_SERVICE_EMAIL` | ✅ | — | Service account email for inter-service auth |
| `ORDER_SERVICE_PASSWORD` | ✅ | — | Service account password |
| `ORDER_PORT` | | `8081` | HTTP server port |
| `WORKER_POOL_SIZE` | | `10` | Number of worker goroutines |
| `INVENTORY_CLIENT_TIMEOUT` | | `5s` | Timeout for Inventory Service HTTP calls |
| `APP_ENV` | | `development` | Environment name |
| `LOG_FORMAT` | | `pretty` | `pretty` for dev, `json` for production |

---

## Key Design Decisions

### Why 202 Accepted not 201 Created?

201 Created implies the resource is fully created and ready. An order at 202 is saved but not yet confirmed — stock hasn't been reserved, prices haven't been fetched. Returning 202 is honest: "we received your order and are working on it." The client polls GET /orders/:id for the final state.

### Why a fixed worker pool and not a goroutine per request?

A goroutine per request means unbounded concurrency. Under load, thousands of goroutines simultaneously call Inventory Service, exhausting its connection pool and causing cascading failures. A fixed pool of N workers means at most N concurrent calls to Inventory Service — explicit, predictable backpressure. When the pool is full, the client gets 503 immediately rather than the server silently degrading.

### Why graceful shutdown?

Kubernetes sends SIGTERM before terminating a pod during rolling deployments. Without graceful shutdown, in-flight orders get lost mid-processing — stock reserved but order never confirmed. With graceful shutdown: HTTP server stops accepting requests, worker pool drains all queued jobs, then the database pool closes. Zero orders lost during deployments.

### Why non-blocking Submit?

HTTP handlers must never block. If Submit blocked waiting for a worker to free up, a full pool would cause all HTTP handler goroutines to hang — the service would appear unresponsive even though it's just busy. Non-blocking Submit with `select/default` returns ErrPoolFull immediately, allowing the handler to return 503 to the client in milliseconds.

### Why payment_status is separate from status?

Order processing status (PENDING → CONFIRMED) and payment status (UNPAID → PAID) are independent lifecycles. An order can be CONFIRMED but UNPAID (stock reserved, payment not yet collected). Separating them avoids a combinatorial explosion of states and makes each lifecycle independently queryable.

### Why interface-driven repository and inventory client?

Both `OrderRepository` and `InventoryClientInterface` are interfaces. Unit tests inject lightweight mocks — no real Postgres or Inventory Service needed. Tests run in milliseconds. Swapping the HTTP inventory client for gRPC only requires a new implementation file, not changes to the processor or service.
