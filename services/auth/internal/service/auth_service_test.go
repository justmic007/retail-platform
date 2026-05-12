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

func (m *mockUserRepo) UpdateRole(ctx context.Context, userID, role string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.Role = domain.Role(role)
		}
	}
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.PasswordHash = passwordHash
		}
	}
	return nil
}

func (m *mockUserRepo) MarkEmailVerified(ctx context.Context, userID string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.EmailVerified = true
		}
	}
	return nil
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

type mockVerificationTokenRepo struct {
	tokens map[string]string // token → userID
}

func newMockVerificationTokenRepo() *mockVerificationTokenRepo {
	return &mockVerificationTokenRepo{tokens: make(map[string]string)}
}

func (m *mockVerificationTokenRepo) Store(ctx context.Context, userID, token string, expiry time.Time) error {
	m.tokens[token] = userID
	return nil
}

func (m *mockVerificationTokenRepo) FindUserID(ctx context.Context, token string) (string, error) {
	userID, exists := m.tokens[token]
	if !exists {
		return "", domain.ErrInvalidVerificationToken
	}
	return userID, nil
}

func (m *mockVerificationTokenRepo) Delete(ctx context.Context, token string) error {
	delete(m.tokens, token)
	return nil
}

// ── Test helper ───────────────────────────────────────────────────────────────

func newTestService() (*AuthService, *mockUserRepo, *mockTokenRepo) {
	cfg := &config.Config{
		BcryptCost:      4,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
		JWTSecret:       "test-secret-key-for-unit-tests-only",
		AppBaseURL:      "http://localhost:8080",
	}
	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	verificationRepo := newMockVerificationTokenRepo()
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	log := logger.New("auth-test")
	svc := NewAuthService(userRepo, tokenRepo, verificationRepo, jwtManager, nil, cfg, log)
	return svc, userRepo, tokenRepo
}

// registerAndVerify is a test helper that registers a user and marks them verified.
func registerAndVerify(t *testing.T, svc *AuthService, userRepo *mockUserRepo, email, password string) {
	t.Helper()
	ctx := context.Background()
	_, err := svc.Register(ctx, domain.RegisterRequest{Email: email, Password: password})
	if err != nil {
		t.Fatalf("setup: register failed: %v", err)
	}
	_ = userRepo.MarkEmailVerified(ctx, "test-user-id-123")
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.RegisterRequest
		wantErr bool
		errIs   error
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
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService()
			ctx := context.Background()

			if tc.wantErr && tc.errIs == domain.ErrEmailTaken {
				_, _ = svc.Register(ctx, domain.RegisterRequest{Email: "micah@example.com", Password: "securepassword"})
			}

			user, err := svc.Register(ctx, tc.req)

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
			if user.ID == "" {
				t.Error("expected user to have an ID after registration")
			}
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
			svc, userRepo, _ := newTestService()
			ctx := context.Background()
			registerAndVerify(t, svc, userRepo, "micah@example.com", "securepassword")

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

func TestLoginUnverifiedEmail(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, domain.RegisterRequest{Email: "micah@example.com", Password: "securepassword"})
	if err != nil {
		t.Fatalf("setup: register failed: %v", err)
	}

	// Do NOT verify email — login should be blocked
	_, err = svc.Login(ctx, domain.LoginRequest{Email: "micah@example.com", Password: "securepassword"})
	if err == nil {
		t.Error("expected error for unverified email but got nil")
	}
}

func TestRefresh(t *testing.T) {
	t.Run("successful token refresh", func(t *testing.T) {
		svc, userRepo, _ := newTestService()
		ctx := context.Background()
		registerAndVerify(t, svc, userRepo, "micah@example.com", "securepassword")

		tokens, err := svc.Login(ctx, domain.LoginRequest{Email: "micah@example.com", Password: "securepassword"})
		if err != nil {
			t.Fatalf("setup: login failed: %v", err)
		}

		newTokens, err := svc.Refresh(ctx, domain.RefreshRequest{RefreshToken: tokens.RefreshToken})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if newTokens.AccessToken == "" {
			t.Error("expected new access token")
		}
		if newTokens.RefreshToken == tokens.RefreshToken {
			t.Error("refresh token should be rotated")
		}
	})

	t.Run("invalid refresh token rejected", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := context.Background()
		_, err := svc.Refresh(ctx, domain.RefreshRequest{RefreshToken: "this-token-does-not-exist"})
		if err == nil {
			t.Error("expected error for invalid refresh token but got nil")
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("logout invalidates refresh token", func(t *testing.T) {
		svc, userRepo, _ := newTestService()
		ctx := context.Background()
		registerAndVerify(t, svc, userRepo, "micah@example.com", "securepassword")

		tokens, err := svc.Login(ctx, domain.LoginRequest{Email: "micah@example.com", Password: "securepassword"})
		if err != nil {
			t.Fatalf("setup: login failed: %v", err)
		}

		if err := svc.Logout(ctx, tokens.RefreshToken); err != nil {
			t.Errorf("logout failed: %v", err)
			return
		}

		_, err = svc.Refresh(ctx, domain.RefreshRequest{RefreshToken: tokens.RefreshToken})
		if err == nil {
			t.Error("refresh should fail after logout but succeeded")
		}
	})
}
