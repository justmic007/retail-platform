// Package service contains the business logic for the Auth Service.
// This is the most important layer — it orchestrates everything.
// It knows about domain types, repositories, and JWT — but NOT about HTTP.

package service

import (
	"context"
	"fmt"

	"retail-platform/auth/internal/config"
	"retail-platform/auth/internal/domain"
	"retail-platform/auth/internal/repository"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/logger"

	// bcrypt is Go's standard password hashing library.
	// It is intentionally slow (by design) to resist brute-force attacks.
	"golang.org/x/crypto/bcrypt"
)

// AuthService contains all authentication business logic.
// Every field is an interface or a value type — no concrete implementations.
// This means AuthService can be tested without a real database or JWT library.
type AuthService struct {
	userRepo  repository.UserRepository  // interface - not *postgresUserRepo
	tokenRepo repository.TokenRepository // interface - not *postgresUserRepo
	jwt       *jwt.Manager
	cfg       *config.Config
	log       *logger.Logger
}

// NewAuthService creates a new AuthService with all dependencies injected.
// This is called once in main.go — the "composition root".
//
// Dependency injection pattern: the service receives its dependencies,
// it does NOT create them. This means:
//   - Tests can inject mocks
//   - Production injects real implementations
//   - The service itself never cares which is which
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtManager *jwt.Manager,
	cfg *config.Config,
	log *logger.Logger,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwt:       jwtManager,
		cfg:       cfg,
		log:       log,
	}
}

// Register creates a new user account.
//
// Steps:
//  1. Hash the password with bcrypt (NEVER store plain text)
//  2. Create the user in the database
//  3. Return the created user (repository handles duplicate email error)
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, error) {
	// bcrypt.GenerateFromPassword hashes the password.
	// s.cfg.BcryptCost = 12 → 2^12 = 4096 iterations → ~300ms per hash.
	// The hash includes a random salt — two users with the same password
	// get completely different hashes. Rainbow table attacks are useless.
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Build the domain User to pass to the repository.
	// We set defaults here — role is always "customer" on registration.
	// Admins can only be created directly in the database or via a
	// separate admin-only endpoint.
	user := &domain.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         domain.RoleCustomer,
	}

	// The repository handles the INSERT and returns the user with
	// DB-generated ID and timestamps populated.
	// If the email already exists, repository returns domain.ErrEmailTaken —
	// we let that propagate up to the handler unchanged.
	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	s.log.Info().
		Str("user_id", created.ID).
		Str("email", created.Email).
		Msg("new user registered")

	return created, nil
}

// Login authenticates a user and issues access + refresh tokens.
//
// Steps:
//  1. Find user by email (returns ErrInvalidCredentials if not found)
//  2. Compare provided password against stored hash
//  3. Generate access token (JWT, 15m TTL)
//  4. Generate refresh token (random string, 7d TTL)
//  5. Store refresh token in database
//  6. Return both tokens
//
// Security note: steps 1 and 2 return the SAME error (ErrInvalidCredentials)
// regardless of whether the email doesn't exist or the password is wrong.
// This prevents user enumeration — attacker learns nothing.
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenPair, error) {
	// Step 1: Find user by email.
	// If not found, repository returns ErrInvalidCredentials (not ErrNotFound).
	// This is a deliberate security decision in the repository layer.
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Don't log the email here — logs may be accessible to attackers.
		// Log the attempt without revealing which email failed.
		s.log.Warn().Msg("login attempt with unknown email")
		return nil, err // already ErrInvalidCredentials
	}

	// Step 2: Compare the provided password against the stored bcrypt hash.
	// bcrypt.CompareHashAndPassword is the correct way — never decrypt,
	// never compare plain text. bcrypt hashes are one-way.
	//
	// This takes ~300ms intentionally — the same cost as generating the hash.
	// This is what makes brute force slow: 300ms per attempt means
	// only ~3 attempts per second, not millions.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.log.Warn().Str("user_id", user.ID).Msg("login attempt with wrong password")
		// Return the same error as "user not found" — no difference to the caller.
		return nil, domain.ErrInvalidCredentials
	}

	// Step 3: Generate access token (JWT).
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Step 4: Generate refresh token (cryptographically random string).
	refreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Step 5: Store refresh token in database with its expiry time.
	expiry := s.jwt.RefreshTokenExpiry()
	if err := s.tokenRepo.StoreRefreshToken(ctx, user.ID, refreshToken, expiry); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	s.log.Info().Str("user_id", user.ID).Msg("user logged in")

	// Step 6: Return the token pair to the handler.
	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// Refresh issues a new access token using a valid refresh token.
// This is called when the access token expires — client sends refresh token,
// gets a new access token back without needing to log in again.
//
// Token rotation: every refresh call invalidates the old refresh token
// and issues a new one. This limits the damage if a refresh token is stolen —
// the attacker's token becomes invalid as soon as the real user refreshes.
func (s *AuthService) Refresh(ctx context.Context, req domain.RefreshRequest) (*domain.TokenPair, error) {
	// Validate the refresh token — look it up in the database.
	// If expired or not found, returns ErrInvalidToken.
	userID, err := s.tokenRepo.FindRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}

	// Get the user to include their current role in the new token.
	// Role may have changed since the last login (e.g. promoted to admin).
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("refresh - find user: %w", err)
	}

	// Delete the old refresh token — it is now invalid.
	// If an attacker stole this token and tries to use it after the real
	// user has refreshed, they'll find it's already deleted.
	if err := s.tokenRepo.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		return nil, fmt.Errorf("refresh - delete old token: %w", err)
	}

	// Generate new token pair.
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("refresh - generate access token: %w", err)
	}

	newRefreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("refresh - generate refresh token: %w", err)
	}

	// Store the new refresh token.
	expiry := s.jwt.RefreshTokenExpiry()
	if err := s.tokenRepo.StoreRefreshToken(ctx, user.ID, newRefreshToken, expiry); err != nil {
		return nil, fmt.Errorf("refresh - store new token: %w", err)
	}

	s.log.Info().Str("user_id", user.ID).Msg("tokens refreshed")

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// Logout invalidates a user's refresh token.
// After this, the user must log in again to get new tokens.
// Their access token still works until it expires (15m) — this is
// acceptable for most use cases. For immediate invalidation, you'd
// need a token blacklist (Redis-based), which is a Day 5 enhancement.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	s.log.Info().Msg("user logged out")
	return nil
}

// GetUserByID retrieves a user's profile by their ID.
// Used by the /me endpoint — the authenticated user requesting their own profile.
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}
