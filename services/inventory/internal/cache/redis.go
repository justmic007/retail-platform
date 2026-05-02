// Package cache — Redis implementation of StockCache.
package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"retail-platform/inventory/internal/config"

	// go-redis is the standard Redis client for Go.
	// v9 is the current version — it fully supports context cancellation.
	"github.com/redis/go-redis/v9"
)

// RedisCache implements StockCache using Redis.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache creates a new Redis-backed stock cache.
// Called once in main.go — the client is shared across all requests.
func NewRedisCache(cfg *config.Config) (StockCache, error) {
	// ParseURL is a helper function to parse Redis URL into connection options.
	// It handles redis://localhost:6379/0 format.
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Ping Redis at startup — fail fast if unreachable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Ping redis: %w", err)
	}

	return &redisCache{
		client: client,
		ttl:    cfg.CacheTTL,
	}, nil
}

// cacheKey builds the Redis key for a product's stock level.
// Namespacing with "stock:" prevents key collisions with other services
// that might use the same Redis instance.
func cacheKey(productID string) string {
	return fmt.Sprintf("stock:%s", productID)
}

// Get retrieves cached available stock for a product.
//
// Cache-aside read pattern:
//   - Hit: return cached value immediately (fast ~0.1ms)
//   - Miss: return found=false — caller queries Postgres and calls Set()
func (c *redisCache) Get(ctx context.Context, productID string) (int, bool, error) {
	val, err := c.client.Get(ctx, cacheKey(productID)).Result()
	if err != nil {
		// redis.Nil means the key doesn't exist — that's a cache miss, not an error
		if errors.Is(err, redis.Nil) {
			return 0, false, nil // cache miss - not an error
		}
		// Any other error (connection issue, timeout) — treat as miss
		// We don't want cache errors to break the user's request
		return 0, false, fmt.Errorf("cache get: %w", err)
	}

	// Redis stores everything as strings — convert back to int
	available, err := strconv.Atoi(val)
	if err != nil {
		// Corrupted cache value — treat as miss

		return 0, false, nil
	}

	return available, true, nil // cache hit
}

// Set stores the available stock count in Redis with TTL.
// Called after a cache miss + Postgres query.
// TTL = 5 minutes by default — after that, Redis expires the key automatically.
func (c *redisCache) Set(ctx context.Context, productID string, available int) error {
	err := c.client.Set(ctx, cacheKey(productID), available, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("Cache set: %w", err)
	}
	return nil
}

// Delete removes a product's cached stock level.
// Called whenever stock changes so the next read gets fresh data from Postgres.
//
// Why delete instead of update?
// Updating is error-prone — you might update with stale data if multiple
// goroutines are modifying stock simultaneously.
// Deleting is safe — the next read always gets the ground truth from Postgres.
func (c *redisCache) Delete(ctx context.Context, productID string) error {
	err := c.client.Del(ctx, cacheKey(productID)).Err()
	if err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	return nil
}
