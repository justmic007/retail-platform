// Package domain — inventory-specific domain errors.
package domain

import "retail-platform/pkg/errors"

var (
	// ErrProductNotFound is returned when a product ID doesn't exist.
	ErrProductNotFound = errors.NewNotFound("product", nil)

	// ErrInsufficientStock is returned when available stock < requested quantity.
	// This is the most critical error — prevents overselling.
	ErrInsufficientStock = &errors.AppError{
		Code:    errors.ErrCodeInsufficientStock,
		Message: "Insufficient stock available for this product",
	}

	// ErrInvalidQuantity is returned when quantity <= 0.
	ErrInvalidQuantity = errors.NewBadRequest("quantity must be greater than zero", nil)
)
