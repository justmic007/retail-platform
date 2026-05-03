// Package repository defines the data access interfaces for the Inventory Service.
package repository

import (
	"context"

	"retail-platform/inventory/internal/domain"
)

// ProductRepository defines the database operations for managing products
type ProductRepository interface {
	// List returns all active products
	List(ctx context.Context) ([]*domain.Product, error)

	// FindByID returns a single product by UUID.
	// Returns ErrProductNotFound if not found.
	FindByID(ctx context.Context, id string) (*domain.Product, error)

	// FindBySKU returns a product by its SKU code.
	// Returns ErrProductNotFound if not found.
	FindBySKU(ctx context.Context, sku string) (*domain.Product, error)
}

// StockRepository defines database operations for stock levels.
type StockRepository interface {
	// GetByProductID returns the stock level for a product.
	GetByProductID(ctx context.Context, productID string) (*domain.StockLevel, error)

	// Reserve atomically locks N units for an order.
	// Uses SELECT FOR UPDATE inside a transaction to prevent overselling.
	// Returns ErrInsufficientStock if available < quantity.
	Reserve(ctx context.Context, productID string, quantity int) error

	// Release returns N previously reserved units back to available.
	// Called when an order is cancelled or fails.
	Release(ctx context.Context, productID string, quantity int) error

	// Adjust sets the total quantity for a product.
	// Called when new stock arrives at the warehouse.
	Adjust(ctx context.Context, productID string, quantity int) error
}
