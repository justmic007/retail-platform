# Notification Service

Event-driven notification service built in Go. Subscribes to Redis Pub/Sub channels and delivers transactional emails to customers and warehouse staff via Brevo. Stateless by design — no database, no HTTP routes except health probes. All behaviour is driven by events published by Order Service and Inventory Service.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Event Flow](#event-flow)
- [Dispatcher Design](#dispatcher-design)
- [Handlers](#handlers)
- [Brevo Integration](#brevo-integration)
- [Layer Breakdown](#layer-breakdown)
- [Events Reference](#events-reference)
- [Running Locally](#running-locally)
- [Running Tests](#running-tests)
- [Environment Variables](#environment-variables)
- [Key Design Decisions](#key-design-decisions)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    NOTIFICATION SERVICE                         │
│                        Port: 8083                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     Dispatcher                          │    │
│  │                   dispatcher.go                         │    │
│  │                                                         │    │
│  │   fan-in select                                         │    │
│  │   ┌──────────────┐         ┌──────────────┐            │    │
│  │   │ orders chan   │         │  stock chan   │            │    │
│  │   └──────┬───────┘         └──────┬───────┘            │    │
│  │          │                        │                     │    │
│  │          ▼                        ▼                     │    │
│  │   ┌─────────────┐         ┌──────────────┐             │    │
│  │   │EmailHandler │         │InternalHandler│             │    │
│  │   │email_handler│         │internal_     │             │    │
│  │   │    .go      │         │handler.go    │             │    │
│  │   └──────┬──────┘         └──────┬───────┘             │    │
│  └──────────┼────────────────────────┼────────────────────┘    │
│             │                        │                          │
└─────────────┼────────────────────────┼──────────────────────────┘
              │                        │
              ▼                        ▼
     Customer Email            Warehouse Email
   (Brevo API call)           (Brevo API call)
```

```
Redis Pub/Sub
┌─────────────────────────────────────────────────────┐
│                                                     │
│  events:orders ──► Notification Service dispatcher  │
│  events:stock  ──► Notification Service dispatcher  │
│                                                     │
│  Publishers:                                        │
│    Order Service     → events:orders                │
│    Inventory Service → events:stock                 │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## Event Flow

### Order Confirmed

```
Customer places order
        │
        ▼
Order Service worker confirms order
        │
        ▼
Publishes to Redis: events:orders
{
  "type": "ORDER_CONFIRMED",
  "order_id": "c5a465e1-...",
  "user_id": "5d81fc13-...",
  "user_email": "customer@example.com",   ← from JWT, cryptographically verified
  "total": 254.95,
  "items": [
    {"product_name": "Sunflower Oil 2L", "quantity": 2, "unit_price": 89.99, "total_price": 179.98},
    {"product_name": "Full Cream Milk 1L", "quantity": 3, "unit_price": 24.99, "total_price": 74.97}
  ],
  "occurred_at": "2026-05-12T09:27:00Z"
}
        │
        ▼
Notification Service dispatcher receives event
        │
        ▼
EmailHandler.SendOrderConfirmation()
        │
        ▼
Brevo API → real email delivered to customer@example.com
```

### Stock Low

```
Order Service reserves stock
        │
        ▼
Inventory Service: available drops below LOW_STOCK_THRESHOLD (10)
        │
        ▼
Publishes to Redis: events:stock
{
  "type": "STOCK_LOW",
  "product_id": "a1b2c3d4-...",
  "product_name": "White Bread 700g",
  "stock_level": 4
}
        │
        ▼
Notification Service dispatcher receives event
        │
        ▼
InternalHandler.SendLowStockAlert()
        │
        ▼
Brevo API → real email delivered to warehouse@example.com
```

---

## Dispatcher Design

The dispatcher is the core of the notification service. It runs as a single goroutine, subscribing to both Redis channels and routing events to the correct handler.

```
func (d *Dispatcher) Run(ctx context.Context) {
    orders, _ := d.bus.SubscribeOrders(ctx)   // <-chan OrderEvent
    stock,  _ := d.bus.SubscribeStock(ctx)    // <-chan StockEvent

    for {
        select {
        case event := <-orders:
            d.safeHandle(func() { d.handleOrderEvent(ctx, event) })

        case event := <-stock:
            d.safeHandle(func() { d.handleStockEvent(ctx, event) })

        case <-ctx.Done():
            return   // graceful shutdown
        }
    }
}
```

**Fan-in select:** both channels are read simultaneously. Whichever event arrives first is processed. No event blocks the other channel.

**safeHandle:** every handler call is wrapped in `recover()`. A panicking handler cannot crash the dispatcher goroutine — the panic is logged and the next event is processed normally.

```
safeHandle wraps fn() with defer/recover:
  ├── fn() completes normally → continue
  └── fn() panics            → log panic, continue processing next event
                                dispatcher never crashes
```

---

## Handlers

### EmailHandler

Sends transactional emails to customers via Brevo. Three methods:

| Method | Trigger | Subject | Body |
|---|---|---|---|
| `SendOrderConfirmation` | `ORDER_CONFIRMED` | Your order has been confirmed | Order reference, items table (name, qty, unit price, total), grand total |
| `SendOrderFailed` | `ORDER_FAILED` | Your order could not be processed | Order reference, no payment taken message |
| `SendOrderCancelled` | `ORDER_CANCELLED` | Your order has been cancelled | Order reference, cancellation confirmation |

The `to` address is `event.UserEmail` — extracted from the JWT by Order Service at order creation time. The customer cannot supply their own email address.

The order confirmation email renders a bordered HTML table of line items:

```
Your order #dd952859 has been confirmed.

┌─────────────────────┬─────┬────────────┬──────────┐
│ Item                │ Qty │ Unit Price │ Total    │
├─────────────────────┼─────┼────────────┼──────────┤
│ Sunflower Oil 2L    │  2  │ $89.99     │ $179.98  │
│ Full Cream Milk 1L  │  3  │ $24.99     │ $74.97   │
└─────────────────────┴─────┴────────────┴──────────┘

Total: R254.95
```

### InternalHandler

Sends operational alerts to warehouse staff. One method:

| Method | Trigger | Subject |
|---|---|---|
| `SendLowStockAlert` | `STOCK_LOW` | Low stock alert: {product_name} |

The `to` address is `WAREHOUSE_EMAIL` from config — a fixed internal address set at deployment time.

---

## Brevo Integration

Both handlers use the [Brevo Go SDK](https://github.com/getbrevo/brevo-go) to send real transactional emails.

```
handler.send(ctx, to, subject, htmlContent)
│
│  ctx = context.WithValue(ctx, brevo.ContextAPIKey, brevo.APIKey{Key: h.apiKey})
│
▼
brevo.TransactionalEmailsApi.SendTransacEmail(ctx, SendSmtpEmail{
    Sender:      {Email: fromEmail, Name: fromName},
    To:          [{Email: to}],
    Subject:     subject,
    HtmlContent: htmlContent,
})
│
├── 2xx → email queued for delivery
└── error → logged, dispatcher continues (notifications are best-effort)
```

**Sender must be verified:** the `EMAIL_FROM` address must be verified in the Brevo dashboard under Senders & IP → Senders. Unverified senders return 401.

---

## Layer Breakdown

| Layer | File | Responsibility |
|---|---|---|
| Entry point | `cmd/server/main.go` | Wires all dependencies, starts dispatcher goroutine and HTTP server |
| Config | `internal/config/config.go` | Loads all env vars with defaults |
| Dispatcher | `internal/dispatcher/dispatcher.go` | Fan-in select, event routing, panic recovery |
| EmailHandler | `internal/handler/email_handler.go` | Customer-facing order emails via Brevo |
| InternalHandler | `internal/handler/internal_handler.go` | Warehouse low stock alerts via Brevo |
| Event Bus | `pkg/events/redis_bus.go` | Redis Pub/Sub subscriber |
| HTTP Server | `internal/server/` | Health + readiness probes only |

---

## Events Reference

### OrderEvent (channel: `events:orders`)

Published by Order Service.

| Field | Type | Description |
|---|---|---|
| `type` | string | `ORDER_CONFIRMED`, `ORDER_FAILED`, `ORDER_CANCELLED` |
| `order_id` | string | UUID of the order |
| `user_id` | string | UUID of the customer |
| `user_email` | string | Customer email — from JWT, not client input |
| `total` | float64 | Order total in USD |
| `items` | array | Line items — only populated on `ORDER_CONFIRMED` |
| `items[].product_name` | string | Product name snapshot at order time |
| `items[].quantity` | int | Units ordered |
| `items[].unit_price` | float64 | Price per unit at order time |
| `items[].total_price` | float64 | quantity × unit_price |
| `occurred_at` | time | When the event was published |

### StockEvent (channel: `events:stock`)

Published by Inventory Service.

| Field | Type | Description |
|---|---|---|
| `type` | string | `STOCK_LOW` |
| `product_id` | string | UUID of the product |
| `product_name` | string | Human-readable product name |
| `stock_level` | int | Current available units |

---

## Running Locally

**Prerequisites:** Go 1.22+, Docker Desktop, all other services running

```bash
# 1. Start infrastructure
make infra-up

# 2. Copy and configure environment
cp services/notification/env.example services/notification/.env
# Edit .env — add your BREVO_SEND_EMAIL_API_KEY and EMAIL_FROM

# 3. Start all services (notification depends on order + inventory for events)
make run-auth         # terminal 1
make run-inventory    # terminal 2
make run-order        # terminal 3
make run-notification # terminal 4
```

**To trigger ORDER_CONFIRMED:** place an order via Postman — confirmation email arrives within seconds.

**To trigger STOCK_LOW:** reserve stock until available drops below 10 — warehouse alert email arrives within seconds.

**To monitor Redis directly:**
```bash
redis-cli subscribe events:orders events:stock
```

---

## Running Tests

```bash
make test-notification

# or directly
cd services/notification && go test -race ./...
```

**What the tests cover:**

| Test | What it verifies |
|---|---|
| `TestDispatcher_OrderConfirmed` | ORDER_CONFIRMED event routes to EmailHandler |
| `TestDispatcher_StockLow` | STOCK_LOW event routes to InternalHandler |
| `TestDispatcher_PanicRecovery` | Panicking handler does not crash dispatcher |
| `TestDispatcher_GracefulShutdown` | ctx cancellation stops dispatcher cleanly |

Tests use an in-process `events.Bus` (Go channels) — no Redis required. Handlers are initialised with empty API keys so no real emails are sent during tests.

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `BREVO_SEND_EMAIL_API_KEY` | ✅ | — | Brevo API key for sending emails |
| `EMAIL_FROM` | ✅ | — | Verified sender email in Brevo dashboard |
| `EMAIL_FROM_NAME` | | `Retail Platform` | Display name for outgoing emails |
| `WAREHOUSE_EMAIL` | | `workfromhomenoni@gmail.com` | Recipient for low stock alerts |
| `REDIS_URL` | | `redis://localhost:6379/0` | Redis connection string |
| `NOTIFICATION_PORT` | | `8083` | HTTP server port |
| `APP_ENV` | | `development` | Environment name |
| `LOG_FORMAT` | | `pretty` | `pretty` for dev, `json` for production |

---

## Key Design Decisions

### Why no database?

The notification service is stateless by design. It receives an event, sends an email, and forgets. There is no state to persist — no retry queue, no delivery tracking, no user preferences. Adding a database would add operational complexity (migrations, connection pooling, backups) for no benefit at this stage.

### Why best-effort delivery?

Notifications are not critical path. If Brevo is down or the event channel is full, the order still completes correctly — stock is reserved, the order is confirmed. The customer may not receive an email, but their order is not affected. Blocking order processing to guarantee email delivery would be the wrong trade-off.

### Why is UserEmail in the JWT and not the event payload from the client?

The customer's email address is a sensitive field. If Order Service trusted the email from the request body, any authenticated user could send order confirmation emails to arbitrary addresses. By embedding the email in the JWT at login time (signed by Auth Service), the email is cryptographically verified — it cannot be tampered with by the client.

### Why a single dispatcher goroutine?

A single goroutine reading from both channels with `select` is simpler and sufficient. Events are low-volume (one per order, one per stock threshold breach). Parallel handlers would add complexity — shared state, race conditions — with no meaningful throughput benefit. If volume grows, the dispatcher can be scaled horizontally by running multiple notification service instances.

### Why panic recovery in safeHandle?

A bug in one handler (nil pointer, type assertion failure) must not crash the entire dispatcher. Without recovery, a single bad event would kill the goroutine and stop all notifications until the service restarts. With recovery, the bad event is logged and skipped — all subsequent events are processed normally.
