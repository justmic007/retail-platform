# Auth Service

Production-grade authentication service built in Go. Issues and validates JWT tokens for the retail platform. All other services validate tokens issued by this service.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Request & Response Lifecycle](#request--response-lifecycle)
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
│                        AUTH SERVICE                             │
│                        Port: 8080                               │
│                                                                 │
│  ┌──────────┐   ┌────────────┐   ┌──────────┐   ┌──────────┐    │
│  │  Router  │ → │ Middleware │ → │ Handler  │ → │ Service  │    │
│  │router.go │   │request_id  │   │auth_     │   │auth_     │    │
│  │          │   │auth.go     │   │handler.go│   │service.go│    │
│  └──────────┘   └────────────┘   └──────────┘   └────────┬─┘    │
│                                                          │      │
│                                              ┌───────────▼─-─┐  │
│                                              │  Repository   │  │
│                                              │ interface.go  │  │
│                                              │ postgres.go   │  │
│                                              └────────────┬──┘  │
└───────────────────────────────────────────────────────────┼─────┘
                                                            │
                                              ┌─────────────▼──────┐
                                              │     PostgreSQL     │
                                              │     auth_db        │
                                              │  ┌──────────────┐  │
                                              │  │    users     │  │
                                              │  ├──────────────┤  │
                                              │  │refresh_tokens│  │
                                              │  └──────────────┘  │
                                              └────────────────────┘
```

---

## Request & Response Lifecycle

This section traces a complete request from the moment the client sends it to the moment it receives a response. Every layer is explained.

### Example: POST /auth/login

#### Phase 1 — Network & Router

```
CLIENT (curl, browser, mobile app)
│
│  POST /auth/login HTTP/1.1
│  Host: localhost:8080
│  Content-Type: application/json
│  {"email":"micah@example.com","password":"securepassword123"}
│
▼
TCP CONNECTION ACCEPTED by Go's net/http on port 8080
│
│  Go's HTTP server reads the raw bytes off the socket.
│  It parses the HTTP request line, headers, and body.
│
▼
GIN ROUTER (internal/server/router.go)
│
│  Gin matches "POST" + "/auth/login" against registered routes.
│  It finds: public.POST("/auth/login", h.Login)
│  It builds the middleware chain for this route.
│
│  Middleware chain for public routes:
│  [RequestID] → [Recovery] → [Login Handler]
│
│  Middleware chain for protected routes:
│  [RequestID] → [Recovery] → [AuthMiddleware] → [Handler]
│
▼
```

#### Phase 2 — Middleware Execution

```
MIDDLEWARE 1: RequestID (pkg/middleware/request_id.go)
│
│  Checks if client sent X-Request-ID header.
│  If not, generates a new UUID v4:
│    requestID = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
│
│  Sets it in two places:
│    1. Gin context: c.Set("request_id", requestID)
│       → Every log line in this request includes this ID
│    2. Response header: X-Request-ID: f47ac10b-...
│       → Client receives it back for support tickets
│
│  Calls c.Next() → passes control to next middleware
│
▼
MIDDLEWARE 2: Recovery (built into Gin)
│
│  Wraps the remaining chain in a recover() call.
│  If any handler panics (unexpected crash), Recovery:
│    - Catches the panic
│    - Logs the stack trace
│    - Returns HTTP 500 to the client
│    - Does NOT crash the server
│
│  Calls c.Next() → passes control to the Login Handler
│
▼
```

#### Phase 3 — Handler Layer

```
HANDLER: AuthHandler.Login (internal/handler/auth_handler.go)
│
│  Step 1 — Bind JSON
│  ├─ c.ShouldBindJSON(&req) reads the request body
│  ├─ Decodes JSON into domain.LoginRequest struct:
│  │    LoginRequest{
│  │      Email:    "micah@example.com",
│  │      Password: "securepassword123"
│  │    }
│  └─ If body is missing or malformed JSON → returns 400 immediately
│
│  Step 2 — Validate
│  ├─ h.validator.Validate(req) checks struct tags:
│  │    Email:    validate:"required,email"  → must be valid email format
│  │    Password: validate:"required"        → must not be empty
│  └─ If validation fails → returns 400 with field errors
│
│  Step 3 — Call Service
│  ├─ h.service.Login(c.Request.Context(), req)
│  │    Note: c.Request.Context() passes the HTTP request's context
│  │    This context cancels if the client disconnects
│  └─ Handler waits for service to return
│
│  Step 4 — Map result to HTTP response (after service returns)
│  ├─ If error → h.handleError(c, err) maps domain error to HTTP status
│  └─ If success → c.JSON(200, tokens)
│
▼
```

#### Phase 4 — Service Layer

```
SERVICE: AuthService.Login (internal/service/auth_service.go)
│
│  The service knows NOTHING about HTTP. No c *gin.Context here.
│  It works purely with Go types and domain concepts.
│
│  Step 1 — Find user by email
│  ├─ s.userRepo.FindByEmail(ctx, req.Email)
│  ├─ Calls the repository (see Phase 5)
│  ├─ If not found → returns domain.ErrInvalidCredentials
│  │    (NOT "email not found" — prevents user enumeration)
│  └─ If found → receives *domain.User with PasswordHash
│
│  Step 2 — Verify password
│  ├─ bcrypt.CompareHashAndPassword(
│  │      []byte(user.PasswordHash),
│  │      []byte(req.Password)
│  │  )
│  ├─ bcrypt re-hashes the input with the stored salt and compares
│  ├─ Takes ~300ms intentionally (brute force protection)
│  ├─ If mismatch → returns domain.ErrInvalidCredentials
│  │    (same error as "email not found" — attacker learns nothing)
│  └─ If match → password is correct, proceed
│
│  Step 3 — Generate access token (JWT)
│  ├─ s.jwt.GenerateAccessToken(user.ID, string(user.Role))
│  ├─ Creates JWT Claims:
│  │    {user_id: "uuid", role: "customer", exp: now+15m, iat: now}
│  ├─ Signs with HMAC-SHA256 using JWT_SECRET
│  └─ Returns: "eyJhbGci..." (base64-encoded JWT string)
│
│  Step 4 — Generate refresh token
│  ├─ s.jwt.GenerateRefreshToken()
│  ├─ Generates 32 cryptographically random bytes
│  ├─ Hex-encodes to 64-character string
│  └─ Returns: "58f1dbdb8d7bd42c..." (not a JWT — just random)
│
│  Step 5 — Store refresh token
│  ├─ s.tokenRepo.StoreRefreshToken(ctx, user.ID, refreshToken, expiry)
│  ├─ expiry = now + 168h (7 days)
│  └─ Calls repository (see Phase 5)
│
│  Step 6 — Return token pair to handler
│  └─ Returns: &domain.TokenPair{
│         AccessToken:  "eyJhbGci...",
│         RefreshToken: "58f1dbdb...",
│         ExpiresIn:    900,  // 15 minutes in seconds
│     }
│
▼
```

#### Phase 5 — Repository Layer

```
REPOSITORY (internal/repository/postgres.go)
│
│  This is the ONLY layer that knows SQL exists.
│  Handler doesn't know SQL. Service doesn't know SQL.
│  Only repository/postgres.go contains SQL.
│
│  FindByEmail — called by service in Step 1:
│  ├─ Executes parameterised query:
│  │    SELECT id, email, password_hash, role, created_at, updated_at
│  │    FROM users
│  │    WHERE email = $1          ← $1 is replaced with actual email
│  │                                 by Postgres — NOT string concat
│  ├─ pgx sends query to Postgres over the connection pool
│  ├─ If no rows → returns domain.ErrInvalidCredentials
│  └─ If found → scans row into domain.User struct
│
│  StoreRefreshToken — called by service in Step 5:
│  ├─ Executes:
│  │    INSERT INTO refresh_tokens (user_id, token, expires_at)
│  │    VALUES ($1, $2, $3)
│  └─ pgx sends to Postgres, confirms INSERT succeeded
│
▼
```

#### Phase 6 — Database

```
POSTGRESQL (auth_db)
│
│  Receives the parameterised query from pgx.
│  The $1 placeholder is treated as DATA, never as SQL code.
│  This is why SQL injection is impossible with parameterised queries.
│
│  For FindByEmail:
│  ├─ Uses idx_users_email index to find the row in O(log n)
│  │    Without this index: full table scan O(n) — slow at scale
│  ├─ Returns the row with all columns
│  └─ pgx scans the result into Go variables
│
│  For StoreRefreshToken:
│  ├─ Inserts new row into refresh_tokens table
│  ├─ idx_refresh_tokens_token index updated
│  └─ Confirms success (rowsAffected = 1)
│
▼
```

#### Phase 7 — Response flows back up

```
DATABASE → returns row to Repository
│
REPOSITORY → scans into domain.User, returns to Service
│
SERVICE → generates tokens, returns *domain.TokenPair to Handler
│
HANDLER → calls c.JSON(200, tokens):
│
│  HTTP/1.1 200 OK
│  Content-Type: application/json
│  X-Request-ID: f47ac10b-58cc-4372-a567-0e02b2c3d479
│
│  {
│    "access_token":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
│    "refresh_token": "58f1dbdb8d7bd42c7d02c87562fdf8d58ef4...",
│    "expires_in":    900
│  }
│
▼
CLIENT receives the response
```

---

## Layer Breakdown

| Layer | File | Knows About | Does NOT Know About |
|---|---|---|---|
| Router | `internal/server/router.go` | HTTP routes, middleware chain | Business logic, SQL |
| Middleware | `pkg/middleware/` | HTTP headers, Gin context | Business logic, SQL |
| Handler | `internal/handler/auth_handler.go` | HTTP, JSON, validation | SQL, bcrypt, JWT signing |
| Service | `internal/service/auth_service.go` | Business rules, bcrypt, JWT | HTTP, SQL |
| Repository | `internal/repository/postgres.go` | SQL, pgx, Postgres errors | HTTP, bcrypt, JWT |
| Database | PostgreSQL `auth_db` | Tables, indexes, constraints | Go code |

**The rule:** each layer only communicates with the layer directly next to it. Handler never calls Repository. Service never reads HTTP headers.

---

## Database Schema

### users

```sql
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,           -- bcrypt hash, never plain text
    role          VARCHAR(50)  NOT NULL DEFAULT 'customer',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);      -- every login queries by email
```

### refresh_tokens

```sql
CREATE TABLE refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      VARCHAR(500) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_token      ON refresh_tokens(token);      -- hot path
CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens(user_id);    -- logout all
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at); -- cleanup job
```

**Why ON DELETE CASCADE?** If a user is deleted, all their refresh tokens are deleted automatically — no orphaned rows.

**Why three indexes on refresh_tokens?** Each index serves a different query pattern: token lookup (every refresh), user_id lookup (logout all devices), expires_at (periodic cleanup of expired tokens).

---

## API Reference

### POST /auth/register

Creates a new user account.

**Request:**
```json
{
  "email": "micah@example.com",
  "password": "securepassword123"
}
```

**Response 201:**
```json
{
  "user": {
    "id": "5fc1d649-2a76-49f5-94a8-54a4bc2e7873",
    "email": "micah@example.com",
    "role": "customer",
    "created_at": "2026-04-26T03:44:29.059418+01:00"
  }
}
```

**Errors:** `400` invalid input · `409` email already exists

---

### POST /auth/login

Authenticates a user and returns tokens.

**Request:**
```json
{
  "email": "micah@example.com",
  "password": "securepassword123"
}
```

**Response 200:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "58f1dbdb8d7bd42c7d02c87562fdf8d5...",
  "expires_in": 900
}
```

**Errors:** `400` invalid input · `401` invalid credentials

---

### POST /auth/refresh

Issues a new access token using a refresh token. Old refresh token is invalidated (token rotation).

**Request:**
```json
{
  "refresh_token": "58f1dbdb8d7bd42c..."
}
```

**Response 200:** Same shape as login response with new tokens.

**Errors:** `401` invalid or expired refresh token

---

### GET /auth/me 🔒

Returns the authenticated user's profile.

**Headers:** `Authorization: Bearer <access_token>`

**Response 200:**
```json
{
  "user": {
    "id": "5fc1d649-2a76-49f5-94a8-54a4bc2e7873",
    "email": "micah@example.com",
    "role": "customer",
    "created_at": "2026-04-26T03:44:29.059418+01:00"
  }
}
```

**Errors:** `401` missing or invalid token

---

### POST /auth/logout 🔒

Invalidates the refresh token. Access token remains valid until expiry (15m).

**Headers:** `Authorization: Bearer <access_token>`

**Request:**
```json
{
  "refresh_token": "58f1dbdb8d7bd42c..."
}
```

**Response 200:**
```json
{
  "message": "logged out successfully"
}
```

---

### GET /health

Liveness probe — returns 200 if the process is running.

### GET /ready

Readiness probe — returns 200 if the service is ready to handle traffic.

---

## Security Design Decisions

### 1. Password hashing with bcrypt cost 12

bcrypt with cost 12 produces ~300ms per hash. This is intentional — it limits brute force attacks to ~3 attempts per second. Each hash includes a unique random salt, making rainbow table attacks useless.

### 2. User enumeration prevention

The login endpoint returns `invalid email or password` for both wrong email and wrong password. Returning different messages for each case would allow attackers to discover which emails are registered.

### 3. Two-token strategy

- **Access token (JWT, 15m):** Stateless, verified by signature alone, no database lookup. Cannot be revoked — if stolen, it works for up to 15 minutes.
- **Refresh token (random, 7d):** Stored in database. Can be deleted immediately on logout or compromise. Used only to obtain new access tokens.

### 4. Token rotation

Every token refresh invalidates the old refresh token and issues a new one. If an attacker steals a refresh token, their session dies the moment the legitimate user next refreshes.

### 5. SQL injection prevention

All queries use parameterised placeholders (`$1`, `$2`). No string concatenation in SQL. Parameterised queries pass values separately to Postgres — they are treated as data, never as SQL code.

### 6. Distroless Docker image

The production image has no shell, no package manager, no compiler — only the compiled Go binary. Attack surface is minimal. Even if exploited, an attacker cannot execute shell commands.

---

## Running Locally

**Prerequisites:** Go 1.22+, Docker Desktop, golang-migrate

```bash
# 1. Start infrastructure
make infra-up

# 2. Run database migrations
make migrate-auth

# 3. Copy environment file
cp services/auth/.env.example services/auth/.env

# 4. Start the service
make run-auth
```

Service starts on `http://localhost:8080`

---

## Postman Collection

A ready-to-use Postman collection is included at `postman_collection.json`.

**Import it:**
1. Open Postman → **Import** → select `postman_environment.json` from the repo root
2. Import `services/auth/postman_collection.json`
3. Select **Retail Platform — Local** as the active environment
4. The `auth_base_url` is pre-set to `http://localhost:8080`

**Recommended order:**
1. **Register** — creates a user
2. **Login** — tokens are auto-saved to collection variables
3. **Get Current User** — uses the saved token automatically
4. **Security Tests** folder — verifies auth behaviour (wrong password, missing token, etc.)

---

## Running Tests

```bash
# Auth service tests only (recommended during development)
make test-auth

# All services with race detector
make test-race
```

Tests use mock repositories — no database required. The race detector (`-race`) detects concurrent memory access bugs.

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | Postgres connection string |
| `JWT_SECRET` | ✅ | — | JWT signing secret (min 32 chars recommended) |
| `AUTH_PORT` | | `8080` | HTTP server port |
| `APP_ENV` | | `development` | Environment name |
| `JWT_ACCESS_TOKEN_TTL` | | `15m` | Access token lifetime |
| `JWT_REFRESH_TOKEN_TTL` | | `168h` | Refresh token lifetime (7 days) |
| `BCRYPT_COST` | | `12` | bcrypt cost factor (4–31, higher = slower) |
| `RATE_LIMIT_REQUESTS` | | `10` | Max login attempts per period |
| `LOG_FORMAT` | | `pretty` | `pretty` for dev, `json` for production |

---

## Key Design Decisions

### Why no ORM?

Raw SQL with pgx gives full control over queries, indexes, and performance. ORMs hide what hits the database — at scale, that's dangerous. Every SQL statement in this service is in one file (`repository/postgres.go`) and is exactly what you expect it to be.

### Why interface-driven repositories?

AuthService depends on `UserRepository` and `TokenRepository` interfaces — not concrete Postgres structs. This means unit tests inject lightweight mock structs that satisfy the same interfaces. Tests run in milliseconds without a database. Swapping Postgres for a different database only requires a new implementation file, not changes to service logic.

### Why separate UserResponse from User?

The `User` domain struct contains `PasswordHash`. The `UserResponse` struct does not. Having two separate structs makes it physically impossible to accidentally return the password hash in an API response — it's structural safety, not a comment or convention.

### Why graceful shutdown?

Kubernetes sends `SIGTERM` before terminating a pod during rolling deployments. Without graceful shutdown, in-flight requests get `502` errors. With it, the server stops accepting new connections, finishes all active requests (up to 30s), closes the database pool, then exits. Zero dropped requests during deployments.
