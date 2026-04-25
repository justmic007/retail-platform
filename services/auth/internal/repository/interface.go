// Package repository defines the data access interfaces for the Auth Service.
//
// The golden rule: interfaces are defined by the CONSUMER, not the IMPLEMENTOR.
// AuthService (the consumer) defines exactly what database operations it needs.
// postgres.go (the implementor) satisfies those interfaces.
// This means AuthService has zero knowledge of Postgres — it only knows the interface.

package repository

import (
	"context"
	"retail-platform/auth/internal/domain"
)

// UserRepository defines all database operations related to users.
// AuthService depends on this interface — NOT on the concrete Postgres struct.
//
// Why interfaces?
// In unit tests, we create a mock struct that implements this interface.
// The mock returns whatever we tell it to — no real database needed.
// Tests run in milliseconds instead of seconds.
// Tests work in CI without a Postgres container.
type UserRepository interface {
	// Create inserts a new user into the database.
	// Returns the created user (with ID and timestamps populated by the DB).
	// Returns an error if the email already exists (unique constraint violation).
	Create(ctx context.Context, user *domain.user) (*domain.User, error)

	// FindByEmail retrieves a user by email address.
	// Used in Login — find the user then compare the password hash.
	// Returns domain.ErrNotFound if no user has that email.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindByID retrieves a user by their UUID.
	// Used in the /me endpoint — get the current user's profile.
	// Returns domain.ErrNotFound if no user has that ID.
	FindByID(ctx context.Context, id string) (*domain.User, error)
}
