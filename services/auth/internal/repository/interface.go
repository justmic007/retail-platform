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
	"time"
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
	Create(ctx context.Context, user *domain.User) (*domain.User, error)

	// FindByEmail retrieves a user by email address.
	// Used in Login — find the user then compare the password hash.
	// Returns domain.ErrNotFound if no user has that email.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindByID retrieves a user by their UUID.
	// Used in the /me endpoint — get the current user's profile.
	// Returns domain.ErrNotFound if no user has that ID.
	FindByID(ctx context.Context, id string) (*domain.User, error)

	// UpdateRole updates the role of a user.
	UpdateRole(ctx context.Context, userID, role string) error
}

// TokenRepository defines all database operations for refresh tokens.
// Refresh tokens are stored in the database so they can be invalidated
// server-side (unlike JWTs which are stateless and cannot be revoked).
type TokenRepository interface {
	// StoreRefreshToken saves a new refresh token for a user.
	// Called after successful login or token refresh.
	// expiry is the time at which this token becomes invalid.
	StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Time) error

	// FindRefreshToken looks up a refresh token and returns the associated userID.
	// Called when a client requests a new access token using their refresh token.
	// Returns an error if the token doesn't exist or has expired.
	FindRefreshToken(ctx context.Context, token string) (userID string, err error)

	// DeleteRefreshToken removes a single refresh token.
	// Called on logout — invalidates this specific session.
	DeleteRefreshToken(ctx context.Context, token string) error

	// DeleteAllUserTokens removes ALL refresh tokens for a user.
	// Called when a user changes their password or requests "logout all devices".
	// After this, the user must log in again on every device.
	DeleteAllUserTokens(ctx context.Context, userID string) error
}
