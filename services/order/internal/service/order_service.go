// Package service contains the business logic for the Order Service.
// It orchestrates the repository, inventory client, and worker pool.
// No HTTP knowledge lives here — only domain logic.
package service

import (
	"context"
	"fmt"

	"retail-platform/order/internal/client"
	"retail-platform/order/internal/repository"
	"retail-platform/order/internal/domain"
	"retail-platform/order/internal/worker"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"

)

// OrderService contains all order business logic.
type OrderService struct {
	repo        repository.OrderRepository
	inventoryClient   client.InventoryClientInterface
	pool  *worker.WorkerPool
	eventBus	*events.Bus
	log      *logger.Logger
}

// NewOrderService creates a new OrderService with all dependencies injected.
func NewOrderService(
	repo repository.OrderRepository,
	inventoryClient client.InventoryClientInterface,
	pool *worker.WorkerPool,
	eventBus *events.Bus,
	log *logger.Logger,
) *OrderService {
	return &OrderService{
		repo:        repo,
		inventoryClient:   inventoryClient,
		pool:  pool,
		eventBus: eventBus,
		log:      log,
	}
}

// CreateOrder saves a new order as PENDING and submits it to the worker pool.
//
// Steps:
//  1. Check idempotency key — if order already exists, return it silently
//  2. Save order as PENDING with items (no prices yet — fetched by worker)
//  3. Submit to worker pool — returns ErrPoolFull if queue is full
//  4. Return order with PENDING status — handler returns 202
func (s *OrderService) CreateOrder(ctx context.Context, userID string req domain.CreateOrderRequest) (*domain.Order, error) {
	
	// Step 1: Check idempotency key
	existing, err := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey, userID)
	if err != nil {
		return nil, fmt.Errorf("check idempotency key: %w", err)
	}

	if existing != nil {
		// Duplicate request — return existing order silently
		s.log.Info().
			Str("user_id", userID).
			Str("idempotency_key", req.IdempotencyKey).
			Msg("duplicate order request — returning existing order")
		return existing, nil
	}

	// Step 2 — Build order domain object
	// Items have no price yet — worker fetches from Inventory Service
	order := &domain.Order{
		UserID:         userID,
		Status:         domain.StatusPending,
		IdempotencyKey: req.IdempotencyKey,
		Notes:          req.Notes,
	}

	for _, item := range req.Items {
		order.Items = append(order.Items, &domain.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	// Step 3 — Save order to database
	created, err := s.repo.Create(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Step 4 — Submit to worker pool for async processing
	if err := s.pool.Submit(worker.Job{
		OrderID: created.ID,
		UserID:  userID,
	}); err != nil {
		// Pool is full — order is saved as PENDING in DB but not queued
		// Client should retry — the idempotency key will return this order
		s.log.Warn().
			Str("order_id", created.ID).
			Msg("worker pool full — order saved but not queued")
		return nil, domain.ErrPoolFull
	}

	s.log.Info().
		Str("order_id", created.ID).
		Str("user_id", userID).
		Msg("order created and queued for processing")

	return created, nil
}

// GetOrder retrieves a single order by ID.
// Ownership is enforced — users can only see their own orders.
func (s *OrderService) GetOrder(ctx context.Context, orderID, userID string) (*domain.Order, error) {
	order, err := s.repo.FindByID(ctx, orderID, userID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return order, nil
}

// ListOrders returns all orders for the authenticated user.
func (s *OrderService) ListOrders(ctx context.Context, userID string) ([]*domain.Order, error) {
	orders, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}

// CancelOrder cancels a PENDING order.
// Only PENDING orders can be cancelled — PROCESSING orders are mid-flight.
func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID string) error {
	// Ownership check — user can only cancel their own orders
	order, err := s.repo.FindByID(ctx, orderID, userID)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}

	// Only PENDING orders can be cancelled
	if order.Status != domain.StatusPending {
		return domain.ErrCannotCancel
	}

	if err := s.repo.UpdateStatus(ctx, orderID, domain.StatusCancelled); err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}

	// Publish OrderCancelled event
	select {
	case s.eventBus.Orders <- events.OrderEvent{
		Type:    events.EventOrderCancelled,
		OrderID: orderID,
		UserID:  userID,
	}:
	default:
		// Channel full — skip, notifications are best-effort
	}

	s.log.Info().
		Str("order_id", orderID).
		Str("user_id", userID).
		Msg("order cancelled")

	return nil
}
