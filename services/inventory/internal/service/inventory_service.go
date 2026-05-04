// Package service contains the business logic for the Inventory Service.
// It orchestrates the repository (Postgres) and cache (Redis) layers.
// The cache-aside pattern lives here — service checks cache first,
// falls back to Postgres on miss, then populates the cache.

package service

import (
	"context"
	"fmt"

	"retail-platform/inventory/internal/cache"
	"retail-platform/inventory/internal/config"
	"retail-platform/inventory/internal/domain"
	"retail-platform/inventory/internal/repository"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// InventoryService contains all inventory business logic.
type InventoryService struct {
	productRepo repository.ProductRepository
	stockRepo   repository.StockRepository
	cache       cache.StockCache
	eventBus    *events.Bus
	cfg         *config.Config
	log         *logger.Logger
}

// NewInventoryService creates a new InventoryService with all dependencies injected.
func NewInventoryService(
	productRepo repository.ProductRepository,
	stockRepo repository.StockRepository,
	stockCache cache.StockCache,
	eventBus *events.Bus,
	cfg *config.Config,
	log *logger.Logger,
) *InventoryService {
	return &InventoryService{
		productRepo: productRepo,
		stockRepo:   stockRepo,
		cache:       stockCache,
		eventBus:    eventBus,
		cfg:         cfg,
		log:         log,
	}
}

// ListProducts returns all active products with their stock levels.
// Stock levels are read from Postgres directly — list queries bypass cache
// because caching a full list is complex to invalidate correctly.
func (s *InventoryService) ListProducts(ctx context.Context) ([]*domain.ProductWithStock, error) {
	products, err := s.productRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	// Fetch stock for each product
	result := make([]*domain.ProductWithStock, 0, len(products))
	for _, p := range products {
		stock, err := s.stockRepo.GetByProductID(ctx, p.ID)
		if err != nil {
			// Log but don't fail — show product with zero stock
			s.log.Warn().Str("product_id", p.ID).Err(err).Msg("failed to get stock for product")
			stock = &domain.StockLevel{ProductID: p.ID}
		}
		result = append(result, &domain.ProductWithStock{
			Product:    p,
			StockLevel: stock,
		})
	}

	return result, nil
}

// GetProduct returns a single product with its stock level.
// Uses cache-aside for the stock level — fast for frequently viewed products.
func (s *InventoryService) GetProduct(ctx context.Context, productID string) (*domain.ProductWithStock, error) {
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}

	stock, err := s.getStockWithCache(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}

	return &domain.ProductWithStock{
		Product:    product,
		StockLevel: stock,
	}, nil
}

// GetStockLevel returns just the stock level for a product.
// This is the hot path — called by Order Service before every reservation.
// Cache-aside: Redis first → Postgres on miss → re-cache.
func (s *InventoryService) GetStockLevel(ctx context.Context, productID string) (*domain.StockLevel, error) {
	return s.getStockWithCache(ctx, productID)
}

// Reserve locks N units of a product for an order.
// This is the most critical operation — uses SELECT FOR UPDATE in Postgres
// to prevent overselling under concurrent load.
//
// After successful reservation:
//   - Redis cache is invalidated (deleted) — next read gets fresh data
//   - If stock falls below threshold, publishes StockLow event
func (s *InventoryService) Reserve(ctx context.Context, req domain.ReserveRequest) error {
	s.log.Info().
		Str("product_id", req.ProductID).
		Int("quantity", req.Quantity).
		Str("order_id", req.OrderID).
		Msg("reserving stock")

	// Reserve in Postgres using SELECT FOR UPDATE
	// This is atomic — either all N units are reserved or none are
	if err := s.stockRepo.Reserve(ctx, req.ProductID, req.Quantity); err != nil {
		return fmt.Errorf("reserve stock: %w", err)
	}

	// Invalidate cache — stock has changed, cached value is now stale
	// Next read will miss cache → query Postgres → get fresh value
	if err := s.cache.Delete(ctx, req.ProductID); err != nil {
		// Cache invalidation failure is NOT fatal — log and continue
		// Worst case: stale cache for up to CacheTTL (5 minutes)
		// The stock in Postgres is correct — cache is just an optimisation
		s.log.Warn().Err(err).Str("product_id", req.ProductID).
			Msg("failed to invalidate cache after reserve — cache may be stale")
	}

	// Check if stock is low — publish event for Notification Service
	s.checkAndPublishLowStock(ctx, req.ProductID)

	s.log.Info().
		Str("product_id", req.ProductID).
		Int("quantity", req.Quantity).
		Str("order_id", req.OrderID).
		Msg("stock reserved successfully")

	return nil
}

// Release returns N previously reserved units back to available.
// Called when an order is cancelled or payment fails.
func (s *InventoryService) Release(ctx context.Context, req domain.ReleaseRequest) error {
	s.log.Info().
		Str("product_id", req.ProductID).
		Int("quantity", req.Quantity).
		Str("order_id", req.OrderID).
		Msg("releasing stock")

	if err := s.stockRepo.Release(ctx, req.ProductID, req.Quantity); err != nil {
		return fmt.Errorf("release stock: %w", err)
	}

	// Invalidate cache — stock has changed
	if err := s.cache.Delete(ctx, req.ProductID); err != nil {
		s.log.Warn().Err(err).Str("product_id", req.ProductID).
			Msg("failed to invalidate cache after release")
	}

	s.log.Info().
		Str("product_id", req.ProductID).
		Int("quantity", req.Quantity).
		Msg("stock released successfully")

	return nil
}

// AdjustStock sets the total quantity for a product.
// Used by warehouse staff when new inventory arrives.
// Admin only — enforced at the handler/middleware level.
func (s *InventoryService) AdjustStock(ctx context.Context, productID string, req domain.StockAdjustRequest) error {
	s.log.Info().
		Str("product_id", productID).
		Int("quantity", req.Quantity).
		Str("reason", req.Reason).
		Msg("adjusting stock")

	if err := s.stockRepo.Adjust(ctx, productID, req.Quantity); err != nil {
		return fmt.Errorf("adjust stock: %w", err)
	}

	// Invalidate cache — stock has changed
	if err := s.cache.Delete(ctx, productID); err != nil {
		s.log.Warn().Err(err).Str("product_id", productID).
			Msg("failed to invalidate cache after adjustment")
	}

	return nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

// getStockWithCache implements the cache-aside pattern for stock levels.
//
// Cache-aside flow:
//  1. Check Redis — if found, return immediately (fast path ~0.1ms)
//  2. Cache miss — query Postgres (~5ms)
//  3. Store result in Redis with TTL
//  4. Return result
//
// This is called on every product view and every stock check.
// The cache absorbs the majority of read traffic.
func (s *InventoryService) getStockWithCache(ctx context.Context, productID string) (*domain.StockLevel, error) {
	// Step 1: Check Redis cache
	available, found, err := s.cache.Get(ctx, productID)
	if err != nil {
		// Cache error — log but fall through to Postgres
		// Never let cache errors break the user's request
		s.log.Warn().Err(err).Str("product_id", productID).Msg("cache get error — falling back to database")
	}

	if found {
		// Cache HIT — return immediately without hitting Postgres
		s.log.Debug().Str("product_id", productID).Int("available", available).Msg("cache hit")
		// Build a StockLevel from cached available count
		// We only cache available — fetch full stock from DB if needed
		return &domain.StockLevel{
			ProductID: productID,
			Quantity:  available, // approximation from cache
			Reserved:  0,
		}, nil
	}

	// Step 2: Cache MISS — query Postgres for authoritative data
	s.log.Debug().Str("product_id", productID).Msg("cache miss — querying database")
	stock, err := s.stockRepo.GetByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get stock from database: %w", err)
	}

	// Step 3: Populate cache for next request
	if cacheErr := s.cache.Set(ctx, productID, stock.Available()); cacheErr != nil {
		// Cache write failure is not fatal — log and continue
		s.log.Warn().Err(cacheErr).Str("product_id", productID).Msg("failed to populate cache")
	}

	return stock, nil
}

// checkAndPublishLowStock checks if stock is below threshold
// and publishes a StockLow event if so.
// Runs non-blocking — does not affect the reservation response.
func (s *InventoryService) checkAndPublishLowStock(ctx context.Context, productID string) {
	stock, err := s.stockRepo.GetByProductID(ctx, productID)
	if err != nil {
		return
	}

	if stock.Available() < s.cfg.LowStockThreshold {
		// Get product name for the event
		product, err := s.productRepo.FindByID(ctx, productID)
		if err != nil {
			return
		}

		// Publish to event bus — non-blocking send
		// If channel is full, skip rather than blocking the reservation
		select {
		case s.eventBus.Stock <- events.StockEvent{
			Type:        events.EventStockLow,
			ProductID:   productID,
			ProductName: product.Name,
			StockLevel:  stock.Available(),
		}:
			s.log.Info().
				Str("product_id", productID).
				Int("available", stock.Available()).
				Msg("low stock event published")
		default:
			// Channel full — skip event, don't block
		}
	}
}
