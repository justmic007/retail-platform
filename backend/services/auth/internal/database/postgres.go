package database

import (
	"context"
	"fmt"
	"time"

	"retail-platform/auth/internal/config"

	// pgxpool is the connection pool package from the pgx Postgres driver.
	// pgx is the best Go Postgres driver — faster than database/sql,
	// supports Postgres-specific types, and has excellent context support.
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates and returns a configured Postgres connection pool.
//
// Why a pool and not a single connection?
// A web server handles many concurrent requests. A single connection handles
// one query at a time — requests queue up waiting. A pool maintains N ready
// connections, one per concurrent request. With pool size 25, your service
// handles 25 simultaneous database operations without any waiting.
//
// Why ping at startup?
// Without the ping, the service starts successfully even if Postgres is down.
// The first user request then fails with a connection error — a confusing
// experience. With ping, the service refuses to start if the DB is unreachable.
// "Fail fast" is always better than "fail on the first user request".
func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	// pgxpool.ParseConfig parses a Postgres connection string into a config struct
	// that we can then customise before creating the pool.
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	// ── Pool sizing ──────────────────────────────────────────────────────────

	// MaxConns: maximum number of connections in the pool.
	// If all 25 are busy and a 26th request arrives, it waits for one to free up.
	// Too high: Postgres runs out of connections (default max is 100).
	// Too low: requests queue up unnecessarily.
	// 25 is appropriate for a service handling hundreds of concurrent requests.
	poolConfig.MaxConns = 25

	// MinConns: connections kept open even when idle.
	// Prevents cold-start latency when traffic spikes after a quiet period.
	// Opening a new Postgres connection takes ~50ms — pre-warming avoids this.
	poolConfig.MinConns = 5

	// ── Connection lifetime ──────────────────────────────────────────────────

	// MaxConnLifetime: maximum time a connection lives before being replaced.
	// Prevents issues with load balancers that drop long-lived connections silently.
	poolConfig.MaxConnLifetime = 1 * time.Hour

	// MaxConnIdleTime: how long an idle connection waits before being closed.
	// Keeps the pool lean during quiet periods.
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// ── Create the pool ──────────────────────────────────────────────────────
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// ── Ping to verify the connection ────────────────────────────────────────
	// Acquire a connection from the pool and immediately release it.
	// This verifies that:
	//   1. The DATABASE_URL is correct
	//   2. Postgres is actually running and reachable
	//   3. The credentials are valid
	//   4. The database exists
	// If any of these fail, we get a clear error at startup — not on user request.
	if err := pool.Ping(ctx); err != nil {
		pool.Close() // don't leak the pool if ping fails
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
