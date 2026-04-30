// auth_service_test.go — unit tests for AuthService.
//
// Key concept: these tests use MOCK repositories that implement the
// UserRepository and TokenRepository interfaces. No real database.
// No Docker. Tests run in milliseconds.
//
// This is possible BECAUSE we designed the service to depend on interfaces,
// not concrete types. The payoff for that decision is right here.
package service

import (
	"context"
	"testing"
	"time"

	"retail-platform/auth/internal/config"
	"retail-platform/auth/internal/domain"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/logger"
)

// ── Mock repositories ─────────────────────────────────────────────────────────
// These structs implement UserRepository and TokenRepository interfaces.
// They store data in memory (maps) instead of a real database.
// Go automatically recognises them as implementations — no "implements" keyword.

type mockUserRepo struct {
	users map[string]*domain.User // keyed by email
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if _, exists := m.users[user.Email]; exists {
		return nil, domain.ErrEmailTaken
	}
	user.ID = "test-user-id-123"
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return user, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, exists := m.users[email]
	if !exists {
		return nil, domain.ErrInvalidCredentials
	}
	return user, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, domain.ErrInvalidCredentials
}

type mockTokenRepo struct {
	tokens map[string]string // token → userID
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: make(map[string]string)}
}

func (m *mockTokenRepo) StoreRefreshToken(ctx context.Context, userID, token string, expiry time.Time) error {
	m.tokens[token] = userID
	return nil
}

func (m *mockTokenRepo) FindRefreshToken(ctx context.Context, token string) (string, error) {
	userID, exists := m.tokens[token]
	if !exists {
		return "", domain.ErrInvalidToken
	}
	return userID, nil
}

func (m *mockTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	delete(m.tokens, token)
	return nil
}

func (m *mockTokenRepo) DeleteAllUserTokens(ctx context.Context, userID string) error {
	for token, uid := range m.tokens {
		if uid == userID {
			delete(m.tokens, token)
		}
	}
	return nil
}

// ── Test helper ───────────────────────────────────────────────────────────────

// newTestService creates an AuthService wired with mock dependencies.
// Called at the start of each test — fresh state every time.
func newTestService() (*AuthService, *mockUserRepo, *mockTokenRepo) {
	cfg := &config.Config{
		BcryptCost:      4, // cost 4 is minimum — makes tests fast (not 300ms each)
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
		JWTSecret:       "test-secret-key-for-unit-tests-only",
	}

	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	log := logger.New("auth-test")

	svc := NewAuthService(userRepo, tokenRepo, jwtManager, cfg, log)
	return svc, userRepo, tokenRepo
}

// ── Tests ─────────────────────────────────────────────────────────────────────
// Table-driven test pattern: define test cases as a slice of structs,
// loop over them, run each as a sub-test with t.Run().
// This is the idiomatic Go way — no Jest, no pytest, just the testing package.

func TestRegister(t *testing.T) {
	// Table of test cases
	tests := []struct {
		name    string
		req     domain.RegisterRequest
		wantErr bool
		errIs   error // the specific error we expect, if any
	}{
		{
			name:    "successful registration",
			req:     domain.RegisterRequest{Email: "micah@example.com", Password: "securepassword"},
			wantErr: false,
		},
		{
			name:    "duplicate email",
			req:     domain.RegisterRequest{Email: "micah@example.com", Password: "anotherpassword"},
			wantErr: true,
			errIs:   domain.ErrEmailTaken,
		},
	}

	for _, tc := range tests {
		// t.Run creates a named sub-test — shows up individually in test output.
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			ctx := context.Background()

			// For duplicate email test, pre-register the email first
			if tc.wantErr && tc.errIs == domain.ErrEmailTaken {
				_, _ = svc.Register(ctx, domain.RegisterRequest{
					Email: "micah@example.com", Password: "securepassword",
				})
			}

			// Now run the actual test case
			user, err := svc.Register(ctx, tc.req)

			if tc.wantErr {
				// We expect an error
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			// We expect success
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if user.ID == "" {
				t.Error("expected user to have an ID after registration")
			}

			if user.PasswordHash == "" {
				t.Error("expected password to be hashed")
			}

			// Critical: the hash must NOT equal the plain text password
			if user.PasswordHash == tc.req.Password {
				t.Error("password was stored as plain text — critical security bug")
			}

			if user.Role != domain.RoleCustomer {
				t.Errorf("expected role 'customer', got '%s'", user.Role)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.LoginRequest
		wantErr bool
	}{
		{
			name:    "successful login",
			req:     domain.LoginRequest{Email: "micah@example.com", Password: "securepassword"},
			wantErr: false,
		},
		{
			name:    "wrong password",
			req:     domain.LoginRequest{Email: "micah@example.com", Password: "wrongpassword"},
			wantErr: true,
		},
		{
			name:    "email not found",
			req:     domain.LoginRequest{Email: "nobody@example.com", Password: "anypassword"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			ctx := context.Background()

			// Register a user first so login has someone to find
			_, err := svc.Register(ctx, domain.RegisterRequest{
				Email: "micah@example.com", Password: "securepassword",
			})
			if err != nil {
				t.Fatalf("setup failed: could not register user: %v", err)
			}

			tokens, err := svc.Login(ctx, tc.req)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tokens.AccessToken == "" {
				t.Error("expected access token to be non-empty")
			}

			if tokens.RefreshToken == "" {
				t.Error("expected refresh token to be non-empty")
			}

			if tokens.ExpiresIn <= 0 {
				t.Error("expected expires_in to be positive")
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	t.Run("successful token refresh", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := context.Background()

		// Register and login to get initial tokens
		_, err := svc.Register(ctx, domain.RegisterRequest{
			Email: "micah@example.com", Password: "securepassword",
		})
		if err != nil {
			t.Fatalf("setup: register failed: %v", err)
		}

		tokens, err := svc.Login(ctx, domain.LoginRequest{
			Email: "micah@example.com", Password: "securepassword",
		})
		if err != nil {
			t.Fatalf("setup: login failed: %v", err)
		}

		// Refresh using the refresh token
		newTokens, err := svc.Refresh(ctx, domain.RefreshRequest{
			RefreshToken: tokens.RefreshToken,
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if newTokens.AccessToken == "" {
			t.Error("expected new access token")
		}

		// Token rotation: new refresh token must be different from the old one
		if newTokens.RefreshToken == tokens.RefreshToken {
			t.Error("refresh token should be rotated — old and new should differ")
		}
	})

	t.Run("invalid refresh token rejected", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := context.Background()

		_, err := svc.Refresh(ctx, domain.RefreshRequest{
			RefreshToken: "this-token-does-not-exist",
		})

		if err == nil {
			t.Error("expected error for invalid refresh token but got nil")
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("logout invalidates refresh token", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := context.Background()

		_, _ = svc.Register(ctx, domain.RegisterRequest{
			Email: "micah@example.com", Password: "securepassword",
		})

		tokens, _ := svc.Login(ctx, domain.LoginRequest{
			Email: "micah@example.com", Password: "securepassword",
		})

		// Logout
		err := svc.Logout(ctx, tokens.RefreshToken)
		if err != nil {
			t.Errorf("logout failed: %v", err)
			return
		}

		// Try to use the now-invalidated refresh token
		_, err = svc.Refresh(ctx, domain.RefreshRequest{
			RefreshToken: tokens.RefreshToken,
		})

		// Must fail — the token was deleted on logout
		if err == nil {
			t.Error("refresh should fail after logout but succeeded")
		}
	})
}
