// Package validator wraps go-playground/validator to provide consistent
// request validation across all services.
//
// In Go, you bind an incoming JSON request body to a struct, then validate it.
// This package makes that a one-liner in any handler.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the underlying validator instance.
// We use a singleton pattern — one instance shared across the service.
type Validator struct {
	v *validator.Validate
}

// New creates a new Validator.
// Call this once at startup and inject it into handlers.
func New() *Validator {
	return &Validator{v: validator.New()}
}

// Validate checks a struct against its `validate` tags.
//
// Example struct:
//
//	type CreateOrderRequest struct {
//	    UserID string `json:"user_id" validate:"required,uuid4"`
//	    Items  []Item `json:"items"   validate:"required,min=1"`
//	}
//
// If validation fails, it returns a human-readable error message
// listing all the fields that failed — not just the first one.
func (val *Validator) Validate(s any) error {
	err := val.v.Struct(s)
	if err == nil {
		return nil
	}

	// Collect all validation errors into one readable message
	var errs []string
	for _, e := range err.(validator.ValidationErrors) {
		errs = append(errs, fmt.Sprintf(
			"field '%s' failed validation: %s",
			strings.ToLower(e.Field()),
			e.Tag(),
		))
	}
	return fmt.Errorf("%s", strings.Join(errs, "; "))
}
