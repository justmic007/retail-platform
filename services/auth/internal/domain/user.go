// Package domain contains the pure business types for the Auth Service.
// This package has ZERO external dependencies — no Gin, no pgx, no JWT library.
// If you changed from Postgres to MongoDB, or from Gin to Echo, this file
// would not change at all.

package domain

import "time"

// Role defines the access level of a user.
// Using a named type (type Role string) instead of plain string means
// the compiler catches mistakes like passing "admn" (typo) — it won't
// match RoleAdmin and the linter will flag it.

type Role string

const (
	RoleCustomer Role = "customer"
	RoleAdmin    Role = "admin"
)

// PromoteRequest is the body for PATCH /auth/users/:id/role
type PromoteRequest struct {
	Role string `json:"role" validate:"required,oneof=customer admin"`
}

// This struct is used INSIDE the service — it contains PasswordHash
// which must NEVER be sent to the client.
type User struct {
	ID           string
	Email        string
	PasswordHash string // internal only — never serialise this to JSON
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ChangePasswordRequest is the body for POST /auth/change-password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=8"`
}

// UserResponse is what gets sent to the client in API responses.
// It deliberately has no PasswordHash field — it physically cannot
// be included in a response because it doesn't exist on this struct.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a User domain struct to a UserResponse.
// Call this in the handler layer before returning JSON to the client.
// Usage: c.JSON(200, user.ToResponse())
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:    u.ID,
		Email: u.Email,
		// Role:  string(u.Role),
		Role: u.Role,

		CreatedAt: u.CreatedAt,
	}
}

// ── Request types ────────────────────────────────────────────────────────────
// These are what the client sends us. The `validate` tags are read by
// our pkg/validator to enforce rules before the service even runs.

// RegisterRequest is the JSON body for POST /auth/register

type RegisterRequest struct {
	Email string `json:"email" validate:"required,email"`
	// validate:"min=8" = password must be at least 8 characters
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest is the JSON body for POST /auth/login
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest is the JSON body for POST /auth/refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ── Response types ───────────────────────────────────────────────────────────

// TokenPair is returned after successful login or token refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	// ExpiresIn is seconds until the access token expires.
	// Clients use this to know when to call /auth/refresh.
	ExpiresIn int `json:"expires_in"`
}
