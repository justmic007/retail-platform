// Package domain contains the pure business types for the Inventory Service.
// No external dependencies — no Gin, no pgx, no Redis.

package domain

import "time"

// Product represents a product in the catalog
type Product struct {
	ID          string
	SKU         string
	Name        string
	Description string
	Price       float64
	Category    string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StockLevel represent the stock for a product in a warehouse
type StockLevel struct {
	ID          string
	ProductID   string
	Quantity    int // total physical units
	Reserved    int // units reserved/locked by pending orders
	WarehouseID string
	UpdatedAt   time.Time
}

// Available returns how many units are actually available to purchase.
// available = quantity - reserved
func (s *StockLevel) Available() int {
	return s.Quantity - s.Reserved
}

// ProductWithStock combines a product and its current stock level.
// This is what most API responses return — product info + availability.
type ProductWithStock struct {
	Product    *Product
	StockLevel *StockLevel
}

// ProductResponse is the API response shape — safe to return to clients.
type ProductResponse struct {
	ID          string  `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	IsActive    bool    `json:"is_active"`
	// Stock info embedded directly so the client can have one response object
	Quantity  int `json:"quantity"`
	Reserved  int `json:"reserved"`
	Available int `json:"available"`
}

// ToResponse converts a ProductWithStock to the API response shape.
func (p *ProductWithStock) ToResponse() *ProductResponse {
	available := 0
	quantity := 0
	reserved := 0

	if p.StockLevel != nil {
		available = p.StockLevel.Available()
		quantity = p.StockLevel.Quantity
		reserved = p.StockLevel.Reserved
	}

	return &ProductResponse{
		ID:          p.Product.ID,
		SKU:         p.Product.SKU,
		Name:        p.Product.Name,
		Description: p.Product.Description,
		Price:       p.Product.Price,
		Category:    p.Product.Category,
		IsActive:    p.Product.IsActive,
		Quantity:    quantity,
		Reserved:    reserved,
		Available:   available,
	}
}

// ── Request types ─────────────────────────────────────────────────────────────

// ReserveRequest is the body for POST /inventory/reserve
// Called by Order Service when a customer places an order.
type ReserveRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity"   validate:"required,min=1"`
	OrderID   string `json:"order_id"   validate:"required"`
}

// ReleaseRequest is the body for POST /inventory/release
// Called by Order Service when an order is cancelled or fails.
type ReleaseRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity"   validate:"required,min=1"`
	OrderID   string `json:"order_id"   validate:"required"`
}

// StockAdjustRequest is the body for PATCH /products/:id/stock
// Used by warehouse staff to update stock after receiving new inventory.
type StockAdjustRequest struct {
	Quantity int    `json:"quantity" validate:"required,min=0"`
	Reason   string `json:"reason" validate:"required"` // e.g. "new shipment", "damaged goods"
}

// ── Response types ────────────────────────────────────────────────────────────

// StockResponse is returned by the stock check endpoint.
type StockResponse struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
}

// ReserveResponse is returned after a successful reservation.
type ReserveResponse struct {
	ProductID string `json:"product_id"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
	Message   string `json:"message"`
}
