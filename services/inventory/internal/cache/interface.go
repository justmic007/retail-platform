// Package cache defines the caching interface for stock levels.
// Same pattern as repository — interface first, implementation second.
// The service depends on this interface, not the Redis implementation.
// This means unit tests can inject a mock cache without a real Redis.

package cache

import "context"

// StockCache defines the caching operations for stock levels.
// The key is always the product ID.
type StockCache interface {
	// Get retrieves the available stock count for a product.
	// Returns (quantity, true) on cache hit.
	// Returns (0, false) on cache miss — caller must query Postgres.
	Get(ctx context.Context, productID string) (available int, found bool, err error)

	// Set stores the available stock count for a product with TTL.
	// Called after a cache miss + Postgres query to populate the cache.
	Set(ctx context.Context, productID string, available int) error

	// Delete removes the cached stock for a product.
	// Called whenever stock changes (reserve, release, adjust).
	// Next read will be a cache miss → fresh Postgres query.
	Delete(ctx context.Context, productID string) error
}
