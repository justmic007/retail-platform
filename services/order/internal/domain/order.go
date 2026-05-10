// Package domain contains the pure business types for the Order Service.
// No external dependencies - no Gin, no pgx, no HTTP.
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus represents the lifecycle of an order.
type OrderStatus string

const (
	StatusPending    OrderStatus = "PENDING"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusConfirmed  OrderStatus = "CONFIRMED"
	StatusFailed     OrderStatus = "FAILED"
	StatusCancelled  OrderStatus = "CANCELLED"
)

// PaymentStatus represents the payment lifecycle of an order.
type PaymentStatus string

const (
	PaymentUnpaid   PaymentStatus = "UNPAID"
	PaymentPaid     PaymentStatus = "PAID"
	PaymentRefunded PaymentStatus = "REFUNDED"
)

// Order is the core domain type
type Order struct {
	ID             string
	UserID         string
	Status         OrderStatus
	PaymentStatus  PaymentStatus
	TotalAmount    decimal.Decimal
	IdempotencyKey string
	Notes          string
	Items          []*OrderItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OrderItem represents a single product line in an order.
// product_name and unit_price are snapshots — copied from Inventory Service
// at order processing time. Historical orders are never affected by
// future price or name changes.
type OrderItem struct {
	ID          string
	OrderID     string
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   decimal.Decimal
	TotalPrice  decimal.Decimal
	CreatedAt   time.Time
}

// - Request types ------------------------------------------------------------------------------------

// CreateOrderRequest is the body for POST /orders
// Client sends product_id and quantiry only - unit_price is fetched
// from Inventory Service by the worker processor, never trusted from client.
type CreateOrderRequest struct {
	Items          []OrderItemRequest `json:"items" validate:"required,min=1"`
	IdempotencyKey string             `json:"idempotency_key" validate:"required"`
	Notes          string             `json:"notes"`
}

// OrderItemRequest is a single line item in the CreateOrderRequest
type OrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

// - Response types -----------------------------------------------------------------------------------

// OrderResponse is the API response shape - safe to return to clients
type OrderResponse struct {
	ID             string               `json:"id"`
	UserID         string               `json:"user_id"`
	Status         OrderStatus          `json:"status"`
	PaymentStatus  PaymentStatus        `json:"payment_status"`
	TotalAmount    decimal.Decimal      `json:"total_amount"`
	IdempotencyKey string               `json:"idempotency_key"`
	Notes          string               `json:"notes"`
	Items          []*OrderItemResponse `json:"items"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// OrderItemResponse is the API response shape for a single order item
type OrderItemResponse struct {
	ID          string          `json:"id"`
	ProductID   string          `json:"product_id"`
	ProductName string          `json:"product_name"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	TotalPrice  decimal.Decimal `json:"total_price"`
}

// ToResponse converts an Order domain model to an OrderResponse for API output
func (o *Order) ToResponse() *OrderResponse {
	resp := &OrderResponse{
		ID:             o.ID,
		UserID:         o.UserID,
		Status:         o.Status,
		TotalAmount:    o.TotalAmount,
		IdempotencyKey: o.IdempotencyKey,
		PaymentStatus:  o.PaymentStatus,
		Notes:          o.Notes,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}

	for _, item := range o.Items {
		resp.Items = append(resp.Items, &OrderItemResponse{
			ID:          item.ID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		})
	}
	return resp
}
