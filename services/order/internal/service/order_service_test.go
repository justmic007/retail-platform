// Package service — unit tests for OrderService.
// Uses mock repository and mock inventory client — no real database or
// Inventory Service needed. Tests run in milliseconds.
package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"retail-platform/order/internal/client"
	"retail-platform/order/internal/domain"
	"retail-platform/order/internal/worker"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// ── Mock Repository ───────────────────────────────────────────────────────────

type mockOrderRepo struct {
	orders map[string]*domain.Order
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{orders: make(map[string]*domain.Order)}
}

func (m *mockOrderRepo) Create(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	order.ID = "test-order-id-123"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	m.orders[order.ID] = order
	return order, nil
}

func (m *mockOrderRepo) FindByID(ctx context.Context, orderID, userID string) (*domain.Order, error) {
	order, ok := m.orders[orderID]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	if userID != "" && order.UserID != userID {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}

func (m *mockOrderRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	var result []*domain.Order
	for _, o := range m.orders {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *mockOrderRepo) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	if order, ok := m.orders[orderID]; ok {
		order.Status = status
	}
	return nil
}

func (m *mockOrderRepo) UpdateStatusAndTotal(ctx context.Context, orderID string, status domain.OrderStatus, total decimal.Decimal) error {
	if order, ok := m.orders[orderID]; ok {
		order.Status = status
		order.TotalAmount = total
	}
	return nil
}

func (m *mockOrderRepo) FindByIdempotencyKey(ctx context.Context, key, userID string) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.IdempotencyKey == key && o.UserID == userID {
			return o, nil
		}
	}
	return nil, nil
}

// ── Mock Inventory Client ─────────────────────────────────────────────────────

type mockInventoryClient struct{}

func (m *mockInventoryClient) GetProduct(ctx context.Context, productID string) (*client.ProductResponse, error) {
	return &client.ProductResponse{
		ID:    productID,
		Name:  "Test Product",
		Price: decimal.NewFromFloat(99.99),
	}, nil
}

func (m *mockInventoryClient) Reserve(ctx context.Context, productID string, quantity int, orderID string) error {
	return nil
}

func (m *mockInventoryClient) Release(ctx context.Context, productID string, quantity int, orderID string) error {
	return nil
}

// ── Test helper ───────────────────────────────────────────────────────────────

func newTestService() (*OrderService, *mockOrderRepo) {
	log := logger.New("order-test")
	repo := newMockOrderRepo()
	eventBus := events.NewBus()
	inventoryClient := &mockInventoryClient{}

	processor := worker.NewOrderProcessor(repo, inventoryClient, eventBus, log)
	pool := worker.NewWorkerPool(2, processor, log)

	svc := NewOrderService(repo, inventoryClient, pool, eventBus, log)
	return svc, repo
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreateOrder(t *testing.T) {
	t.Run("creates order successfully", func(t *testing.T) {
		svc, _ := newTestService()
		ctx := context.Background()

		pool := svc.pool
		pool.Start(ctx)
		defer pool.Shutdown()

		req := domain.CreateOrderRequest{
			IdempotencyKey: "test-key-001",
			Items: []domain.OrderItemRequest{
				{ProductID: "product-1", Quantity: 2},
			},
		}

		order, err := svc.CreateOrder(ctx, "user-123", req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if order.ID == "" {
			t.Error("expected order to have an ID")
		}

		if order.Status != domain.StatusPending {
			t.Errorf("expected status PENDING, got %s", order.Status)
		}
	})

	t.Run("idempotency — same key returns same order", func(t *testing.T) {
		svc, _ := newTestService()
		ctx := context.Background()

		pool := svc.pool
		pool.Start(ctx)
		defer pool.Shutdown()

		req := domain.CreateOrderRequest{
			IdempotencyKey: "test-key-002",
			Items: []domain.OrderItemRequest{
				{ProductID: "product-1", Quantity: 1},
			},
		}

		order1, err := svc.CreateOrder(ctx, "user-123", req)
		if err != nil {
			t.Fatalf("first create failed: %v", err)
		}

		// Submit same request again
		order2, err := svc.CreateOrder(ctx, "user-123", req)
		if err != nil {
			t.Fatalf("second create failed: %v", err)
		}

		if order1.ID != order2.ID {
			t.Error("expected same order ID for duplicate request")
		}
	})
}

func TestCancelOrder(t *testing.T) {
	t.Run("cancels pending order", func(t *testing.T) {
		svc, repo := newTestService()
		ctx := context.Background()

		// Create an order directly in the mock repo
		order := &domain.Order{
			ID:     "order-to-cancel",
			UserID: "user-123",
			Status: domain.StatusPending,
		}
		repo.orders[order.ID] = order

		err := svc.CancelOrder(ctx, "order-to-cancel", "user-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if order.Status != domain.StatusCancelled {
			t.Errorf("expected status CANCELLED, got %s", order.Status)
		}
	})

	t.Run("cannot cancel non-pending order", func(t *testing.T) {
		svc, repo := newTestService()
		ctx := context.Background()

		order := &domain.Order{
			ID:     "confirmed-order",
			UserID: "user-123",
			Status: domain.StatusConfirmed,
		}
		repo.orders[order.ID] = order

		err := svc.CancelOrder(ctx, "confirmed-order", "user-123")
		if err != domain.ErrCannotCancel {
			t.Errorf("expected ErrCannotCancel, got %v", err)
		}
	})

	t.Run("cannot cancel another user's order", func(t *testing.T) {
		svc, repo := newTestService()
		ctx := context.Background()

		order := &domain.Order{
			ID:     "other-user-order",
			UserID: "user-456",
			Status: domain.StatusPending,
		}
		repo.orders[order.ID] = order

		err := svc.CancelOrder(ctx, "other-user-order", "user-123")
		if err != domain.ErrOrderNotFound {
			t.Errorf("expected ErrOrderNotFound, got %v", err)
		}
	})
}

func TestGetOrder(t *testing.T) {
	t.Run("returns order for correct user", func(t *testing.T) {
		svc, repo := newTestService()
		ctx := context.Background()

		order := &domain.Order{
			ID:     "order-123",
			UserID: "user-123",
			Status: domain.StatusPending,
		}
		repo.orders[order.ID] = order

		result, err := svc.GetOrder(ctx, "order-123", "user-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ID != "order-123" {
			t.Errorf("expected order ID order-123, got %s", result.ID)
		}
	})

	t.Run("returns error for wrong user", func(t *testing.T) {
		svc, repo := newTestService()
		ctx := context.Background()

		order := &domain.Order{
			ID:     "order-123",
			UserID: "user-456",
			Status: domain.StatusPending,
		}
		repo.orders[order.ID] = order

		_, err := svc.GetOrder(ctx, "order-123", "user-123")
		if err == nil {
			t.Error("expected error for wrong user but got nil")
		}
	})
}
