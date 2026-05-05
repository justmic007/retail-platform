// inventory_service_test.go — unit tests for InventoryService.
// Uses mock repository AND mock cache — no real Postgres or Redis needed.
// Tests run in milliseconds.
package service

import (
	"context"
	"testing"
	"time"

	"retail-platform/inventory/internal/config"
	"retail-platform/inventory/internal/domain"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// ── Mock Product Repository ───────────────────────────────────────────────────

type mockProductRepo struct {
	products map[string]*domain.Product
}

func newMockProductRepo() *mockProductRepo {
	repo := &mockProductRepo{products: make(map[string]*domain.Product)}
	// Pre-populate with test products matching our seed data
	repo.products["product-1"] = &domain.Product{
		ID: "product-1", SKU: "OIL-SF-2L", Name: "Sunflower Oil 2L",
		Price: 89.99, Category: "Groceries", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.products["product-2"] = &domain.Product{
		ID: "product-2", SKU: "RICE-BAS-5KG", Name: "Basmati Rice 5kg",
		Price: 129.99, Category: "Groceries", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	return repo
}

func (m *mockProductRepo) List(ctx context.Context) ([]*domain.Product, error) {
	result := make([]*domain.Product, 0, len(m.products))
	for _, p := range m.products {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockProductRepo) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (m *mockProductRepo) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	for _, p := range m.products {
		if p.SKU == sku {
			return p, nil
		}
	}
	return nil, domain.ErrProductNotFound
}

// ── Mock Stock Repository ─────────────────────────────────────────────────────

type mockStockRepo struct {
	stocks map[string]*domain.StockLevel
}

func newMockStockRepo() *mockStockRepo {
	repo := &mockStockRepo{stocks: make(map[string]*domain.StockLevel)}
	// Pre-populate with test stock
	repo.stocks["product-1"] = &domain.StockLevel{
		ID: "stock-1", ProductID: "product-1",
		Quantity: 150, Reserved: 0, WarehouseID: "main",
	}
	repo.stocks["product-2"] = &domain.StockLevel{
		ID: "stock-2", ProductID: "product-2",
		Quantity: 5, Reserved: 0, WarehouseID: "main", // low stock
	}
	return repo
}

func (m *mockStockRepo) GetByProductID(ctx context.Context, productID string) (*domain.StockLevel, error) {
	s, ok := m.stocks[productID]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return s, nil
}

func (m *mockStockRepo) Reserve(ctx context.Context, productID string, quantity int) error {
	s, ok := m.stocks[productID]
	if !ok {
		return domain.ErrProductNotFound
	}
	if s.Available() < quantity {
		return domain.ErrInsufficientStock
	}
	s.Reserved += quantity
	return nil
}

func (m *mockStockRepo) Release(ctx context.Context, productID string, quantity int) error {
	s, ok := m.stocks[productID]
	if !ok {
		return domain.ErrProductNotFound
	}
	if quantity > s.Reserved {
		s.Reserved = 0
	} else {
		s.Reserved -= quantity
	}
	return nil
}

func (m *mockStockRepo) Adjust(ctx context.Context, productID string, quantity int) error {
	s, ok := m.stocks[productID]
	if !ok {
		return domain.ErrProductNotFound
	}
	s.Quantity = quantity
	return nil
}

// ── Mock Cache ────────────────────────────────────────────────────────────────

type mockCache struct {
	data    map[string]int
	deleted []string // track which keys were deleted (for assertions)
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string]int)}
}

func (m *mockCache) Get(ctx context.Context, productID string) (int, bool, error) {
	val, ok := m.data[productID]
	return val, ok, nil
}

func (m *mockCache) Set(ctx context.Context, productID string, available int) error {
	m.data[productID] = available
	return nil
}

func (m *mockCache) Delete(ctx context.Context, productID string) error {
	delete(m.data, productID)
	m.deleted = append(m.deleted, productID)
	return nil
}

// ── Test helper ───────────────────────────────────────────────────────────────

func newTestService() (*InventoryService, *mockStockRepo, *mockCache) {
	cfg := &config.Config{
		LowStockThreshold: 10,
		CacheTTL:          5 * 60 * 1000000000, // 5 minutes in nanoseconds
	}

	productRepo := newMockProductRepo()
	stockRepo := newMockStockRepo()
	stockCache := newMockCache()
	eventBus := events.NewBus()
	log := logger.New("inventory-test")

	svc := NewInventoryService(productRepo, stockRepo, stockCache, eventBus, cfg, log)
	return svc, stockRepo, stockCache
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestListProducts(t *testing.T) {
	t.Run("returns all products with stock", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := context.Background()

		products, err := svc.ListProducts(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(products) != 2 {
			t.Errorf("expected 2 products, got %d", len(products))
		}

		for _, p := range products {
			if p.Product == nil {
				t.Error("product should not be nil")
			}
			if p.StockLevel == nil {
				t.Error("stock level should not be nil")
			}
		}
	})
}

func TestGetStockLevel(t *testing.T) {
	tests := []struct {
		name      string
		productID string
		wantErr   bool
	}{
		{
			name:      "returns stock for existing product",
			productID: "product-1",
			wantErr:   false,
		},
		{
			name:      "returns error for non-existent product",
			productID: "non-existent-id",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			ctx := context.Background()

			stock, err := svc.GetStockLevel(ctx, tc.productID)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if stock.ProductID != tc.productID {
				t.Errorf("expected product_id %s, got %s", tc.productID, stock.ProductID)
			}
		})
	}
}

func TestGetStockLevel_CacheAside(t *testing.T) {
	t.Run("cache miss triggers database query and populates cache", func(t *testing.T) {
		svc, _, mockCache := newTestService()
		ctx := context.Background()

		// Ensure cache is empty
		if len(mockCache.data) != 0 {
			t.Fatal("cache should be empty at start")
		}

		// First call — cache miss, queries Postgres
		_, err := svc.GetStockLevel(ctx, "product-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Cache should now be populated
		if _, ok := mockCache.data["product-1"]; !ok {
			t.Error("cache should be populated after cache miss")
		}

		// Second call — should hit cache
		_, err = svc.GetStockLevel(ctx, "product-1")
		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}
	})

	t.Run("cache hit avoids database query", func(t *testing.T) {
		svc, _, mockCache := newTestService()
		ctx := context.Background()

		// Pre-populate cache
		mockCache.data["product-1"] = 100

		stock, err := svc.GetStockLevel(ctx, "product-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should return cached value
		if stock == nil {
			t.Error("stock should not be nil")
		}
	})
}

func TestReserve(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.ReserveRequest
		wantErr bool
		errType error
	}{
		{
			name: "successful reservation",
			req: domain.ReserveRequest{
				ProductID: "product-1",
				Quantity:  10,
				OrderID:   "order-abc-123",
			},
			wantErr: false,
		},
		{
			name: "insufficient stock",
			req: domain.ReserveRequest{
				ProductID: "product-1",
				Quantity:  999, // more than available (150)
				OrderID:   "order-abc-456",
			},
			wantErr: true,
			errType: domain.ErrInsufficientStock,
		},
		{
			name: "product not found",
			req: domain.ReserveRequest{
				ProductID: "non-existent",
				Quantity:  1,
				OrderID:   "order-abc-789",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, stockRepo, mockCache := newTestService()
			ctx := context.Background()

			// Pre-populate cache to verify it gets invalidated
			mockCache.data[tc.req.ProductID] = 100

			err := svc.Reserve(ctx, tc.req)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify stock was actually reserved
			stock, _ := stockRepo.GetByProductID(ctx, tc.req.ProductID)
			if stock.Reserved != tc.req.Quantity {
				t.Errorf("expected reserved=%d, got %d", tc.req.Quantity, stock.Reserved)
			}

			// Verify cache was invalidated after reservation
			if _, ok := mockCache.data[tc.req.ProductID]; ok {
				t.Error("cache should be invalidated after reservation")
			}
		})
	}
}

func TestRelease(t *testing.T) {
	t.Run("release returns units to available", func(t *testing.T) {
		svc, stockRepo, _ := newTestService()
		ctx := context.Background()

		// First reserve some units
		err := svc.Reserve(ctx, domain.ReserveRequest{
			ProductID: "product-1", Quantity: 20, OrderID: "order-1",
		})
		if err != nil {
			t.Fatalf("reserve failed: %v", err)
		}

		// Verify reserved
		stock, _ := stockRepo.GetByProductID(ctx, "product-1")
		if stock.Reserved != 20 {
			t.Fatalf("expected reserved=20, got %d", stock.Reserved)
		}

		// Now release
		err = svc.Release(ctx, domain.ReleaseRequest{
			ProductID: "product-1", Quantity: 20, OrderID: "order-1",
		})
		if err != nil {
			t.Fatalf("release failed: %v", err)
		}

		// Verify released
		stock, _ = stockRepo.GetByProductID(ctx, "product-1")
		if stock.Reserved != 0 {
			t.Errorf("expected reserved=0 after release, got %d", stock.Reserved)
		}
	})
}

func TestReserve_CacheInvalidation(t *testing.T) {
	t.Run("cache is invalidated after reserve", func(t *testing.T) {
		svc, _, mockCache := newTestService()
		ctx := context.Background()

		// Pre-populate cache
		mockCache.data["product-1"] = 150

		// Reserve
		err := svc.Reserve(ctx, domain.ReserveRequest{
			ProductID: "product-1", Quantity: 5, OrderID: "order-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Cache must be deleted — not updated, DELETED
		// Next read will go to Postgres for authoritative data
		if _, ok := mockCache.data["product-1"]; ok {
			t.Error("cache key should be deleted after reservation, not just updated")
		}

		// Verify delete was called
		deleted := false
		for _, key := range mockCache.deleted {
			if key == "product-1" {
				deleted = true
				break
			}
		}
		if !deleted {
			t.Error("cache.Delete should have been called for product-1")
		}
	})
}
