// Package repository defines the data access interfaces for the Order Service
package repository

import (
	"context"
	"retail-platform/order/internal/domain"

	"github.com/shopspring/decimal"
)

// OrderRepository defines all database operations for orders
// The service depends on this interface - not the concrete Postgres struct
// This means that unit tests can inject a mock without a real database
type OrderRepository interface {
	// Create inserts a new order and its items in a single transaction
	// It returns the created order with DB-generated ID and timestamps
	Create(ctx context.Context, order *domain.Order) (*domain.Order, error)

	// FindByID retrieves an order by its ID and userID
	FindByID(ctx context.Context, orderID, userID string) (*domain.Order, error)

	// FindByUserID retrieves all orders for a given userID, newest first
	FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error)

	// UpdateStatus updates the status of an order
	// Called by the worker processor as the order moves through its lifecycle
	UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error

	// UpdateStatusAndTotal updates status and total_amount together.
	// Called when order is CONFIRMED — sets the final calculated total.
	UpdateStatusAndTotal(ctx context.Context, orderID string, status domain.OrderStatus, total decimal.Decimal) error

	// FindByIdempotencyKey looks up an order by its idempotency key and UserID
	// Returns the existing order if found - used to detect duplicate submissions
	// Returns nil, nil if not found - is not an error
	FindByIdempotencyKey(ctx context.Context, key, userID string) (*domain.Order, error)
}
