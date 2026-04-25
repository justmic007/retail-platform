// postgres.go — Postgres implementations of UserRepository and TokenRepository.
//
// This is the ONLY file in the entire Auth Service that contains SQL.
// If you search the codebase for "SELECT", "INSERT", "UPDATE", "DELETE" —
// they all live here. Nowhere else.
//
// Why centralise SQL?
// When your DBA says "that query is causing a table scan, add this index",
// you know exactly where to look. When you need to audit data access patterns,
// one file tells the whole story.

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"retail-platform/auth/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// -- User Repository --------------------------------------------------

// postgresUserRepo is the concrete Postgres implementation of UserRepository.
// The lowercase name means it's unexported — nothing outside this package
// can create one directly. They must use NewPostgresUserRepo() below.
type postgresUserRepo struct {
	db *pgxpool.Pool
}

// NewPostgresUserRepo creates a new PostgreSQL user repository.
// Returns UserRepository (the interface) not *postgresUserRepo (the struct).
// This enforces that callers depend on the interface, not the implementation.
func NewPostgresUserRepo(db *pgxpool.Pool) UserRepository {
	return &postgresUserRepo{db: db}
}

// Create inserts a new user into the users table.
//
// Key patterns to notice:
//  1. Parameterised queries ($1, $2, ...) — NEVER string concatenation.
//     String concatenation = SQL injection vulnerability.
//     Parameterised queries let Postgres safely handle any input.
//  2. RETURNING clause — gets the DB-generated id and timestamps back
//     in one round trip instead of doing INSERT then SELECT.
//  3. ctx is passed to every DB call — if the HTTP request is cancelled,
//     the DB query is cancelled too. No wasted DB resources.

func (r *postgresUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash role created_at, updated_at
	`

	created := &domain.User{}

	// QueryRow executes a query expected to return one row.
	// Scan reads the returned columns into Go variables.
	// The order of Scan arguments MUST match the RETURNING column order.
	err := r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		string(user.Role),
	).Scan(
		&created.ID,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation (duplicate email).
		// pgx returns a *pgconn.PgError with Code "23505" for unique violations.
		// We convert this to our domain error so the service layer
		// doesn't need to know about Postgres error codes.
		if isPgUniqueViolation(err) {
			return nil, domain.ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return created, nil
}

// FindByEmail retrieves a user by their email address.
// Used on every login attempt.
func (r *postgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		// pgx.ErrNoRows is returned when SELECT finds no matching row.
		// We convert it to ErrInvalidCredentials (not ErrNotFound) because
		// we never want to tell the caller "that email doesn't exist" —
		// that's a user enumeration vulnerability.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

// FindByID retrieves a user by their UUID primary key.
// Used by the /me endpoint to get the current user's profile.
func (r *postgresUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

// ── Token Repository ──────────────────────────────────────────────────────────

type postgresTokenRepo struct {
	db *pgxpool.Pool
}

// NewPostgresTokenRepo creates a new PostgreSQL token repository.
func NewPostgresTokenRepo(db *pgxpool.Pool) TokenRepository {
	return &postgresTokenRepo{db: db}
}

// StoreRefreshToken saves a new refresh token to the database.
func (r *postgresTokenRepo) StoreRefreshToken(
	ctx context.Context,
	userID, token string,
	expiry time.Time,
) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`

	// Exec is used for queries that don't return rows (INSERT, UPDATE, DELETE).
	_, err := r.db.Exec(ctx, query, userID, token, expiry)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}

	return nil
}

// FindRefreshToken looks up a refresh token and returns the owning user's ID.
// Also checks that the token hasn't expired — expired tokens are rejected
// even if they exist in the database.
func (r *postgresTokenRepo) FindRefreshToken(ctx context.Context, token string) (string, error) {
	query := `
		SELECT user_id
		FROM refresh_tokens
		WHERE token = $1
		  AND expires_at > NOW()
	`
	// AND expires_at > NOW() means expired tokens return no rows —
	// treated the same as a non-existent token. No separate expiry check needed.

	var userID string
	err := r.db.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrInvalidToken
		}
		return "", fmt.Errorf("find refresh token: %w", err)
	}

	return userID, nil
}

// DeleteRefreshToken removes a specific refresh token (single device logout).
func (r *postgresTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`

	_, err := r.db.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}

	return nil
}

// DeleteAllUserTokens removes all refresh tokens for a user (logout all devices).
func (r *postgresTokenRepo) DeleteAllUserTokens(ctx context.Context, userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete all user tokens: %w", err)
	}

	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// isPgUniqueViolation checks if an error is a Postgres unique constraint violation.
// Postgres error code 23505 = unique_violation.
// We check for this to convert DB-level errors into domain errors.
func isPgUniqueViolation(err error) bool {
	// pgconn.PgError is the Postgres-specific error type from pgx.
	// errors.As unwraps the error chain looking for this type.
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
