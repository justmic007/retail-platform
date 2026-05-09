// Package domain — order-specific domain errors.
package domain

import "retail-platform/pkg/errors"

var (
	// ErrOrderNotFound is returned when an order ID doesn't exist
	// or belongs to a different user.
	ErrOrderNotFound = errors.NewNotFound("order", nil)

	// ErrOrderNotOwned is returned when a user tries to access
	// an order that belongs to a different user.
	ErrOrderNotOwned = &errors.AppError{
		Code:    errors.ErrCodeForbidden,
		Message: "order not found",
	}

	// ErrCannotCancel is returned when trying to cancel an order
	// that is not in PENDING status.
	ErrCannotCancel = &errors.AppError{
		Code:    errors.ErrCodeBadRequest,
		Message: "only pending orders can be cancelled",
	}

	// ErrPoolFull is returned when the worker pool job channel is full.
	// The caller should return 503 Service Unavailable.
	ErrPoolFull = &errors.AppError{
		Code:    errors.ErrCodeInternal,
		Message: "server is busy, please try again",
	}

	// ErrDuplicateOrder is never returned as an error — when an idempotency
	// key matches, the existing order is returned silently. This sentinel
	// is used internally to signal that case.
	ErrDuplicateOrder = &errors.AppError{
		Code:    errors.ErrCodeConflict,
		Message: "order with this idempotency key already exists",
	}
)
