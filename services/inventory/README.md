# Inventory Service

Production-grade inventory management service built in Go. Manages the product catalogue, tracks stock levels, and handles concurrent stock reservations for the retail platform. Prevents overselling under high concurrent load using Postgres row-level locking.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Request & Response Lifecycle](#request--response-lifecycle)
- [Cache-aside Pattern](#cache-aside-pattern)
- [Concurrency & Overselling Prevention](#concurrency--overselling-prevention)
- [Low Stock Events](#low-stock-events)
- [Layer Breakdown](#layer-breakdown)
- [Database Schema](#database-schema)
- [API Reference](#api-reference)
- [Running Locally](#running-locally)
- [Running Tests](#running-tests)
- [Environment Variables](#environment-variables)
- [Key Design Decisions](#key-design-decisions)

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                      INVENTORY SERVICE                               │
│                        Port: 8082                                    │
│                                                                      │
│  Router → AuthMiddleware → Handler → Service                        │
│                                   │                                  │
│                         ┌─────────┴──────────┐                      │
│                         │                    │                       │
│                    Cache Layer          Repository Layer              │
│                    (cache.go)           (postgres.go)                │
│                    interface            interface                     │
│                         │                    │                       │
└─────────────────────────┼────────────────────┼───────────────────────┘
                          │                    │
               ┌──────────▼──────┐  ┌──────────▼──────────┐
               │      Redis      │  │      Postgres        │
               │  stock:uuid →   │  │    inventory_db      │
               │  available=145  │  │  ┌──────────────┐   │
               │  TTL: 5 minutes │  │  │   products   │   │
               └─────────────────┘  │  ├──────────────┤   │
                                    │  │ stock_levels │   │
                                    │  └──────────────┘   │
                                    └─────────────────────┘
```

---

## Request & Response Lifecycle

This section traces a complete request through every layer of the service.

### Example: POST /inventory/reserve

This is the most critical endpoint — reserving stock for an order while preventing overselling.

#### Phase 1 — Network & Router

```
CLIENT (Order Service or curl)
│
│  POST /inventory/reserve HTTP/1.1
│  Authorization: Bearer eyJhbGci...
│  Content-Type: application/json
│  {
│    "product_id": "a1b2c3d4-0001-...",
│    "quantity": 5,
│    "order_id": "order-uuid-here"
│  }
│
▼
TCP CONNECTION accepted on port 8082
│
▼
GIN ROUTER (internal/server/router.go)
│
│  Matches POST /inventory/reserve
│  Middleware chain: [RequestID] → [Recovery] → [AuthMiddleware] → [Handler]
│
▼
```

#### Phase 2 — Middleware

```
MIDDLEWARE 1: RequestID
│  Generates X-Request-ID: "f47ac10b-58cc-4372..."
│  Attached to context + response header
│  Every log line in this request includes this ID
▼

MIDDLEWARE 2: AuthMiddleware (pkg/middleware/auth.go)
│  Extracts Bearer token from Authorization header
│  Calls jwtManager.ValidateToken(token)
│  Checks HMAC signature (same secret as Auth Service)
│  Checks token expiry
│  Sets user_id and role in Gin context
│  If invalid → 401 Unauthorized, request stops here
▼
```

#### Phase 3 — Handler

```
HANDLER: InventoryHandler.Reserve (internal/handler/inventory_handler.go)
│
│  Step 1 — Bind JSON body into ReserveRequest struct
│  Step 2 — Validate fields:
│    product_id: required
│    quantity:   required, min=1
│    order_id:   required
│  Step 3 — Call service.Reserve(c.Request.Context(), req)
│  Step 4 — On success: fetch updated stock, return response
│  Step 5 — On error: map domain error to HTTP status
│
▼
```

#### Phase 4 — Service (Business Logic)

```
SERVICE: InventoryService.Reserve (internal/service/inventory_service.go)
│
│  Step 1 — Log the reservation attempt
│  │   log.Info: "reserving stock" {product_id, quantity, order_id}
│
│  Step 2 — Call stockRepo.Reserve(ctx, productID, quantity)
│  │   This is where SELECT FOR UPDATE happens (see Phase 5)
│  │   Returns nil on success
│  │   Returns ErrInsufficientStock if not enough available
│  │   Returns ErrProductNotFound if product doesn't exist
│
│  Step 3 — Invalidate Redis cache
│  │   cache.Delete(ctx, productID)
│  │   Removes "stock:product-uuid" key from Redis
│  │   Next GET /products/:id/stock will miss cache → hit Postgres
│  │   If cache.Delete fails → log warning, continue (not fatal)
│
│  Step 4 — Check if stock is now low
│  │   Read updated stock from Postgres
│  │   If available < LOW_STOCK_THRESHOLD (10):
│  │     publish StockEvent to events.Bus.Stock channel (non-blocking)
│  │     Notification Service will send alert to warehouse staff
│
│  Step 5 — Return nil (success)
│
▼
```

#### Phase 5 — Repository (Database Layer)

```
REPOSITORY: postgresStockRepo.Reserve (internal/repository/postgres.go)
│
│  This is the ONLY place in the entire service that contains SQL.
│
│  Step 1 — Begin transaction
│  │   tx, err := r.db.Begin(ctx)
│  │   defer func() { _ = tx.Rollback(ctx) }()  ← safety net, no-op if Commit succeeds
│
│  Step 2 — SELECT FOR UPDATE (row-level lock)
│  │
│  │   SELECT quantity, reserved
│  │   FROM stock_levels
│  │   WHERE product_id = $1
│  │   FOR UPDATE
│  │
│  │   What this does:
│  │   ├── Reads the current stock values
│  │   └── LOCKS this row — other transactions trying to
│  │       SELECT FOR UPDATE on the same product WAIT here
│  │       until we COMMIT or ROLLBACK
│
│  Step 3 — Check availability
│  │   available = quantity - reserved
│  │   if available < requested_quantity:
│  │     return ErrInsufficientStock (Rollback fires via defer → lock released)
│
│  Step 4 — UPDATE reserved count
│  │   UPDATE stock_levels
│  │   SET reserved = reserved + $1
│  │   WHERE product_id = $2
│
│  Step 5 — Commit
│  │   tx.Commit(ctx)
│  │   Lock is released here
│  │   Waiting transactions can now proceed
│  │   They will read updated reserved count and check availability
│
▼
```

#### Phase 6 — Postgres

```
POSTGRESQL (inventory_db)
│
│  Receives parameterised queries — $1, $2 are data, never SQL code
│  SQL injection is impossible with parameterised queries
│
│  For SELECT FOR UPDATE:
│  ├── Locates the stock_levels row using idx_stock_product_id index
│  ├── Acquires exclusive row lock
│  └── Returns quantity and reserved values
│
│  For UPDATE:
│  ├── CHECK (reserved >= 0) constraint verified
│  ├── Row updated atomically
│  └── Lock held until COMMIT
│
│  At COMMIT:
│  └── All changes made permanent
│      Lock released
│      Waiting transactions unblocked
│
▼
```

#### Phase 7 — Response flows back up

```
POSTGRES → returns success to Repository
│
REPOSITORY → returns nil to Service
│
SERVICE → invalidates cache, checks low stock, returns nil to Handler
│
HANDLER → fetches updated stock level, builds response:
│
│  HTTP/1.1 200 OK
│  X-Request-ID: f47ac10b-58cc-4372...
│  Content-Type: application/json
│
│  {
│    "product_id": "a1b2c3d4-0001-...",
│    "reserved": 5,
│    "available": 145,
│    "message": "stock reserved successfully"
│  }
│
▼
CLIENT receives confirmation — stock is locked for the order
```

---

## Cache-aside Pattern

The Inventory Service caches stock levels in Redis to avoid hitting Postgres on every product page view.

### Read path (cache-aside)

```
GET /products/:id/stock

┌─────────────────────────────────────────────────────────┐
│                   Service Layer                         │
│                                                         │
│  1. Check Redis: GET stock:product-uuid                 │
│     │                                                   │
│     ├── HIT  (key exists)                               │
│     │   └── Return cached available count (~0.1ms)      │
│     │       No database query needed                    │
│     │       NOTE: reserved is returned as 0 on a hit    │
│     │       — only available is cached, not the full    │
│     │         stock breakdown. Use GET /products/:id    │
│     │         for authoritative quantity + reserved.    │
│     │                                                   │
│     └── MISS (key doesn't exist)                        │
│         │                                               │
│         ├── Query Postgres: SELECT from stock_levels    │
│         │   (~5ms)                                      │
│         │                                               │
│         ├── Store in Redis: SET stock:uuid value 5m TTL │
│         │   (populates cache for next request)          │
│         │                                               │
│         └── Return full stock level to caller           │
└─────────────────────────────────────────────────────────┘
```

### Write path (cache invalidation)

```
Any stock CHANGE (reserve, release, adjust):

  1. Update Postgres (source of truth)
  2. DELETE stock:product-uuid from Redis
  3. Next read → cache miss → fresh Postgres query → re-cached

Why DELETE and not UPDATE?
─────────────────────────
  Updating is dangerous under concurrent load:
  - Goroutine A reads stock=10, computes new_value=7
  - Goroutine B reads stock=10, computes new_value=8
  - Both write their computed value — one overwrites the other

  Deleting is always safe:
  - Any goroutine that deletes the key is correct
  - The next read always gets authoritative data from Postgres
  - No computation, no race condition
```

### ListProducts bypasses cache

```
GET /products always queries Postgres directly — it does NOT use Redis.

Why?
  Caching a full product list is complex to invalidate correctly.
  Any stock change to any product would need to invalidate the list.
  Individual product caching (GET /products/:id/stock) is simpler
  and covers the hot path — the list endpoint is used less frequently.
```

### TTL as safety net

```
Cache TTL = 5 minutes (configurable via CACHE_TTL env var)

Purpose: prevents stale data from living forever if
         explicit invalidation somehow fails

Normal flow: explicit DELETE on every stock change
TTL flow:    backup — after 5 minutes, Redis auto-expires the key

This means: even in the worst case (bug prevents cache invalidation),
            data is at most 5 minutes stale
```

---

## Concurrency & Overselling Prevention

This is the most critical part of the Inventory Service. At scale, thousands of customers may simultaneously try to buy the last available unit.

### The problem without locking

```
Time →    T1          T2          T3
          ──────────────────────────────
          100 concurrent requests arrive for the last 1 unit of milk

          All 100 read:  quantity=1, reserved=0, available=1
          All 100 check: is 1 >= 1? YES
          All 100 run:   UPDATE SET reserved = reserved + 1
          All 100 get:   "stock reserved successfully"

          Result: reserved=100, available=-99
                  99 customers get confirmations but NO PRODUCT
                  This is the overselling problem
```

### The solution: SELECT FOR UPDATE

```
Time →    T1(first)   T2-T100(waiting)
          ──────────────────────────────────────────────
BEGIN
SELECT quantity, reserved
FROM stock_levels
WHERE product_id = $1
FOR UPDATE              ← T1 acquires row lock
                          T2-T100 PAUSE here, waiting

  available = 1 - 0 = 1
  1 >= 1? YES → proceed

UPDATE reserved = 0 + 1

COMMIT                  ← lock released
                          T2 acquires lock, reads reserved=1
                          T2: available = 1 - 1 = 0
                          T2: 0 >= 1? NO → ErrInsufficientStock
                          T2 ROLLBACK

                          T3-T100: same as T2
                          All return ErrInsufficientStock

Final state: reserved=1, available=0. Only 1 customer gets the product. ✓
```

### Why row-level locking scales

```
Product A: locked by transaction 1
Product B: locked by transaction 2    ← NO blocking between different products
Product C: locked by transaction 3

1000 different products = 1000 simultaneous reservations
Each locks its own row with no interference
```

---

## Low Stock Events

When a reservation brings available stock below `LOW_STOCK_THRESHOLD` (default: 10), the service publishes a `StockLow` event to the shared event bus. The Notification Service listens on this channel and alerts warehouse staff.

```
POST /inventory/reserve
│
│  Reservation succeeds
│
▼
SERVICE: checkAndPublishLowStock()
│
│  Reads updated stock from Postgres
│  available < LOW_STOCK_THRESHOLD (10)?
│
│  YES →  select {
│           case eventBus.Stock <- StockEvent{
│             Type:        "STOCK_LOW",
│             ProductID:   "...",
│             ProductName: "Full Cream Milk 1L",
│             StockLevel:  3,
│           }:
│           // published
│         default:
│           // channel full — skip, don't block the reservation
│         }
│
│  NO  →  nothing published
│
▼
NOTIFICATION SERVICE reads from eventBus.Stock channel
  → sends alert to warehouse staff
```

**Key design:** the event publish is non-blocking. If the notification channel is full, the event is dropped rather than slowing down the reservation. The reservation always completes — notifications are best-effort.

**Configuring the threshold:**
```bash
LOW_STOCK_THRESHOLD=20  # alert when stock drops below 20 units
```

---

## Layer Breakdown

| Layer | File | Knows About | Does NOT Know About |
|---|---|---|---|
| Router | `internal/server/router.go` | HTTP routes, middleware chain | Business logic, SQL, Redis |
| Middleware | `pkg/middleware/` | HTTP headers, JWT validation | Business logic, SQL |
| Handler | `internal/handler/inventory_handler.go` | HTTP, JSON, validation | SQL, Redis, transactions |
| Service | `internal/service/inventory_service.go` | Business rules, cache-aside, events | HTTP, SQL syntax |
| Cache | `internal/cache/redis.go` | Redis commands, key format | Postgres, business logic |
| Repository | `internal/repository/postgres.go` | SQL, transactions, SELECT FOR UPDATE | HTTP, Redis, business logic |
| Database | PostgreSQL `inventory_db` | Tables, indexes, constraints | Go code |

---

## Database Schema

### products

```sql
CREATE TABLE products (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    sku         VARCHAR(100)  NOT NULL UNIQUE,       -- retail identifier
    name        VARCHAR(255)  NOT NULL,
    description TEXT,
    price       NUMERIC(10,2) NOT NULL CHECK (price >= 0),  -- exact decimal, not float
    category    VARCHAR(100),
    is_active   BOOLEAN       NOT NULL DEFAULT true,  -- soft delete
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_sku      ON products(sku);       -- SKU lookups
CREATE INDEX idx_products_category ON products(category);  -- category filters
CREATE INDEX idx_products_active   ON products(is_active); -- active product queries
```

**Why NUMERIC(10,2) not FLOAT for price?**
Floating point cannot precisely represent most decimals. `0.1 + 0.2 = 0.30000000000000004`. For a retailer processing millions of transactions, this causes financial discrepancies. NUMERIC is exact.

### stock_levels

```sql
CREATE TABLE stock_levels (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id   UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity     INTEGER      NOT NULL DEFAULT 0 CHECK (quantity >= 0),  -- total units
    reserved     INTEGER      NOT NULL DEFAULT 0 CHECK (reserved >= 0),  -- locked units
    warehouse_id VARCHAR(100) NOT NULL DEFAULT 'main',
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(product_id, warehouse_id)  -- one stock row per product per warehouse
);

CREATE INDEX idx_stock_product_id ON stock_levels(product_id);
```

**quantity vs reserved vs available:**

| Field | Meaning | Changes when |
|---|---|---|
| `quantity` | Physical units in warehouse | Stock arrives or ships |
| `reserved` | Units locked by pending orders | Order placed or cancelled |
| `available` | `quantity - reserved` | Computed, not stored |

### Seed data (for development)

| SKU | Product | Quantity |
|---|---|---|
| OIL-SF-2L | Sunflower Oil 2L | 150 |
| RICE-BAS-5KG | Basmati Rice 5kg | 200 |
| DAIRY-MLK-FC-1L | Full Cream Milk 1L | 500 |
| BREAD-WH-700G | White Bread 700g | 300 |
| EGGS-FR-18PK | Free Range Eggs 18 Pack | 250 |

---

## API Reference

All endpoints require a valid JWT issued by Auth Service (`Authorization: Bearer <token>`).

### GET /products

Returns all active products with current stock levels. Always queries Postgres directly — bypasses Redis cache.

**Response 200:**
```json
{
  "products": [
    {
      "id": "a1b2c3d4-0001-0001-0001-000000000001",
      "sku": "OIL-SF-2L",
      "name": "Sunflower Oil 2L",
      "description": "Pure sunflower cooking oil",
      "price": 89.99,
      "category": "Groceries",
      "is_active": true,
      "quantity": 150,
      "reserved": 5,
      "available": 145
    }
  ],
  "total": 5
}
```

---

### GET /products/:id

Returns a single product with its full stock level. Uses Redis cache for the stock level.

**Response 200:**
```json
{
  "product": {
    "id": "a1b2c3d4-0001-0001-0001-000000000001",
    "sku": "OIL-SF-2L",
    "name": "Sunflower Oil 2L",
    "description": "Pure sunflower cooking oil",
    "price": 89.99,
    "category": "Groceries",
    "is_active": true,
    "quantity": 150,
    "reserved": 5,
    "available": 145
  }
}
```

**Errors:** `404` product not found

---

### GET /products/:id/stock

Returns just the stock level for a product. Uses Redis cache — fast path for Order Service.

**Note:** on a cache hit, `reserved` is returned as `0` and `quantity` reflects the cached `available` value. For the authoritative breakdown, use `GET /products/:id`.

**Response 200:**
```json
{
  "product_id": "a1b2c3d4-0001-0001-0001-000000000001",
  "quantity": 150,
  "reserved": 5,
  "available": 145
}
```

**Errors:** `404` product not found

---

### POST /inventory/reserve

Locks N units of a product for an order. Uses SELECT FOR UPDATE to prevent overselling.

**Request:**
```json
{
  "product_id": "a1b2c3d4-0001-0001-0001-000000000001",
  "quantity": 5,
  "order_id": "order-uuid-here"
}
```

**Response 200:**
```json
{
  "product_id": "a1b2c3d4-0001-0001-0001-000000000001",
  "reserved": 5,
  "available": 145,
  "message": "stock reserved successfully"
}
```

**Errors:** `409` insufficient stock · `404` product not found

---

### POST /inventory/release

Returns previously reserved units back to available. Called when an order is cancelled or fails.

**Request:**
```json
{
  "product_id": "a1b2c3d4-0001-0001-0001-000000000001",
  "quantity": 5,
  "order_id": "order-uuid-here"
}
```

**Response 200:**
```json
{
  "message": "stock released successfully"
}
```

**Errors:** `404` product not found

---

### PATCH /products/:id/stock 🔒 Admin only

Manually sets the total quantity. Used when new stock arrives at the warehouse.

**Request:**
```json
{
  "quantity": 200,
  "reason": "New delivery from supplier"
}
```

**Response 200:**
```json
{
  "message": "stock adjusted successfully"
}
```

**Errors:** `403` admin role required · `404` product not found

---

### GET /health · GET /ready

Kubernetes liveness and readiness probes.

---

## Running Locally

**Prerequisites:** Go 1.25+, Docker Desktop, golang-migrate

```bash
# 1. Start infrastructure
make infra-up

# 2. Run database migrations + seed data
make migrate-inventory

# 3. Copy environment file
cp services/inventory/.env.example services/inventory/.env

# 4. Start Auth Service (required for JWT validation)
make run-auth

# 5. Start Inventory Service (new terminal)
make run-inventory
```

Service starts on `http://localhost:8082`

---

## Running Tests

```bash
# Inventory service tests with race detector
make test-inventory

# All services
make test-race
```

Unit tests use mock repository and mock cache — no Postgres or Redis required. Tests run in milliseconds.

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | Postgres connection string for inventory_db |
| `JWT_SECRET` | ✅ | — | Must match Auth Service JWT_SECRET exactly |
| `REDIS_URL` | | `redis://localhost:6379/0` | Redis connection URL |
| `INVENTORY_PORT` | | `8082` | HTTP server port |
| `CACHE_TTL` | | `5m` | How long stock levels stay cached in Redis |
| `LOW_STOCK_THRESHOLD` | | `10` | Publish StockLow event when available drops below this |
| `APP_ENV` | | `development` | Environment name |
| `LOG_FORMAT` | | `pretty` | `pretty` for dev, `json` for production |

---

## Key Design Decisions

### Why SELECT FOR UPDATE instead of application-level locking?

Application-level locking (Go mutexes, Redis distributed locks) adds complexity and failure modes. `SELECT FOR UPDATE` leverages Postgres's battle-tested ACID transaction system. It's row-level — Product A's lock never blocks Product B. It's automatically released on connection failure. It's the industry standard for inventory reservation in relational databases.

### Why separate products and stock_levels tables?

Products change rarely (weekly). Stock changes constantly (every order). Separating them prevents high-frequency stock writes from locking the products table during reads. It also enables multi-warehouse support (one stock row per product per warehouse) and independent caching strategies for each table.

### Why cache-aside and not write-through?

Write-through updates cache on every write — adding latency to every reservation. Cache-aside only hits the cache on reads and invalidates on writes. For inventory: reads vastly outnumber writes. Cache-aside is optimal — writes stay fast, reads are cached after the first miss.

### Why only cache available and not the full stock breakdown?

Caching `available` (a single integer) is simple and covers the hot path — Order Service only needs to know if stock exists before reserving. Caching `quantity` and `reserved` separately introduces consistency risk: they could become out of sync under concurrent writes. One value, one key, one source of truth.

### Why NUMERIC(10,2) for price?

IEEE 754 floating point cannot precisely represent most decimal values. `0.1 + 0.2 = 0.30000000000000004`. For a retailer processing millions of transactions, floating point errors cause real financial discrepancies. NUMERIC stores exact decimals. Always use NUMERIC or DECIMAL for money — never FLOAT.

### Why is the cache layer a separate package with an interface?

The `StockCache` interface allows unit tests to inject a map-based mock instead of a real Redis. Tests run in milliseconds without any infrastructure. The same pattern as repository interfaces — depend on abstractions, not implementations.

### Why are low stock events non-blocking?

The reservation must always complete fast — a customer is waiting. Notification delivery is best-effort. Using a buffered channel with a `select/default` means if the notification channel is full, the event is dropped rather than blocking the reservation goroutine. Stock accuracy is critical; notification latency is acceptable.
