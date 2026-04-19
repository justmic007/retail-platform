// Package errors defines domain-level error types used across all services.
// The key idea: business logic returns domain errors (ErrNotFound, ErrUnauthorized).
// HTTP handlers map those to status codes. The two layers never mix.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode is a machine-readable error identifier.
// Frontend apps and API clients can use these to show the right message.
type ErrorCode string

const (
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"
	ErrCodeInsufficientStock ErrorCode = "INSUFFICIENT_STOCK"
	ErrCodeInvalidToken     ErrorCode = "INVALID_TOKEN"
)

// AppError is our custom error type.
// It carries both a human-readable message and a machine-readable code.
// It also implements the standard Go error interface.
type AppError struct {
	Code    ErrorCode // machine-readable
	Message string    // human-readable
	Err     error     // original underlying error (for logging)
}

// Error implements the error interface.
// In Go, any type with an Error() string method IS an error.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap allows errors.Is() and errors.As() to work through our wrapper.
// This is standard Go error wrapping — very important to understand.
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus maps our domain error codes to HTTP status codes.
// This mapping lives here — NOT in handlers, NOT in services.
// Centralised so every service uses the same mapping.
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeNotFound:
		return http.StatusNotFound // 404
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized // 401
	case ErrCodeForbidden:
		return http.StatusForbidden // 403
	case ErrCodeBadRequest:
		return http.StatusBadRequest // 400
	case ErrCodeConflict, ErrCodeInsufficientStock:
		return http.StatusConflict // 409
	default:
		return http.StatusInternalServerError // 500
	}
}

// ── Constructors ──────────────────────────────────────────────────────────────
// These are convenience functions so services don't build AppError structs manually.

func NewNotFound(resource string, err error) *AppError {
	return &AppError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Err:     err,
	}
}

func NewUnauthorized(message string) *AppError {
	return &AppError{Code: ErrCodeUnauthorized, Message: message}
}

func NewBadRequest(message string, err error) *AppError {
	return &AppError{Code: ErrCodeBadRequest, Message: message, Err: err}
}

func NewInsufficientStock(productID string) *AppError {
	return &AppError{
		Code:    ErrCodeInsufficientStock,
		Message: fmt.Sprintf("insufficient stock for product %s", productID),
	}
}

func NewInternal(err error) *AppError {
	return &AppError{
		Code:    ErrCodeInternal,
		Message: "an internal error occurred",
		Err:     err,
	}
}

// IsAppError checks if an error is one of our AppErrors.
// Uses standard Go errors.As — works through wrapping chains.
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
