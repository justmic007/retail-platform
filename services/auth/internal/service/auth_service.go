// Package service contains the business logic for the Auth Service.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"retail-platform/auth/internal/config"
	"retail-platform/auth/internal/domain"
	"retail-platform/auth/internal/repository"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/logger"
	"retail-platform/pkg/mailer"

	"golang.org/x/crypto/bcrypt"
)

// AuthService contains all authentication business logic.
type AuthService struct {
	userRepo          repository.UserRepository
	tokenRepo         repository.TokenRepository
	verificationRepo  repository.VerificationTokenRepository
	jwt               *jwt.Manager
	mailer            *mailer.Mailer
	cfg               *config.Config
	log               *logger.Logger
}

// NewAuthService creates a new AuthService with all dependencies injected.
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	verificationRepo repository.VerificationTokenRepository,
	jwtManager *jwt.Manager,
	m *mailer.Mailer,
	cfg *config.Config,
	log *logger.Logger,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		verificationRepo: verificationRepo,
		jwt:              jwtManager,
		mailer:           m,
		cfg:              cfg,
		log:              log,
	}
}

// Register creates a new user account and sends a verification email.
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         domain.RoleCustomer,
	}

	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	s.log.Info().Str("user_id", created.ID).Str("email", created.Email).Msg("new user registered")

	// Generate and store verification token — best effort, don't fail registration
	s.sendVerificationEmail(ctx, created)

	return created, nil
}

// Login authenticates a user and issues access + refresh tokens.
// Blocks login if email is not verified.
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		s.log.Warn().Msg("login attempt with unknown email")
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.log.Warn().Str("user_id", user.ID).Msg("login attempt with wrong password")
		return nil, domain.ErrInvalidCredentials
	}

	// Block login for unverified accounts
	if !user.EmailVerified {
		s.log.Warn().Str("user_id", user.ID).Msg("login attempt with unverified email")
		return nil, domain.ErrEmailNotVerified
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	expiry := s.jwt.RefreshTokenExpiry()
	if err := s.tokenRepo.StoreRefreshToken(ctx, user.ID, refreshToken, expiry); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	s.log.Info().Str("user_id", user.ID).Msg("user logged in")

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// VerifyEmail marks a user's email as verified using the token from the email link.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	userID, err := s.verificationRepo.FindUserID(ctx, token)
	if err != nil {
		return err // already ErrInvalidVerificationToken
	}

	if err := s.userRepo.MarkEmailVerified(ctx, userID); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}

	// Delete the token — single use only
	if err := s.verificationRepo.Delete(ctx, token); err != nil {
		s.log.Warn().Err(err).Str("user_id", userID).Msg("failed to delete used verification token")
	}

	s.log.Info().Str("user_id", userID).Msg("email verified")
	return nil
}

// ResendVerification sends a new verification email to an unverified user.
func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists
		return nil
	}

	if user.EmailVerified {
		return nil // already verified — silently succeed
	}

	s.sendVerificationEmail(ctx, user)
	return nil
}

// Refresh issues a new access token using a valid refresh token.
func (s *AuthService) Refresh(ctx context.Context, req domain.RefreshRequest) (*domain.TokenPair, error) {
	userID, err := s.tokenRepo.FindRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("refresh - find user: %w", err)
	}

	if err := s.tokenRepo.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		return nil, fmt.Errorf("refresh - delete old token: %w", err)
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("refresh - generate access token: %w", err)
	}

	newRefreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("refresh - generate refresh token: %w", err)
	}

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
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	s.log.Info().Msg("user logged out")
	return nil
}

// GetUserByID retrieves a user's profile by their ID.
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// PromoteUser updates a user's role.
func (s *AuthService) PromoteUser(ctx context.Context, userID, role string) error {
	if err := s.userRepo.UpdateRole(ctx, userID, role); err != nil {
		return fmt.Errorf("promote user: %w", err)
	}
	s.log.Info().Str("user_id", userID).Str("role", role).Msg("user role updated")
	return nil
}

// ChangePassword updates a user's password and invalidates all sessions.
func (s *AuthService) ChangePassword(ctx context.Context, userID string, req domain.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := s.tokenRepo.DeleteAllUserTokens(ctx, userID); err != nil {
		s.log.Warn().Err(err).Str("user_id", userID).Msg("failed to delete tokens after password change")
	}

	s.log.Info().Str("user_id", userID).Msg("password changed — all sessions terminated")
	return nil
}

// sendVerificationEmail generates a token, stores it, and sends the email.
// Best-effort — errors are logged but never returned to the caller.
func (s *AuthService) sendVerificationEmail(ctx context.Context, user *domain.User) {
	if s.mailer == nil {
		s.log.Warn().Str("user_id", user.ID).Msg("mailer not configured — skipping verification email")
		return
	}
	token, err := generateToken()
	if err != nil {
		s.log.Error().Err(err).Str("user_id", user.ID).Msg("failed to generate verification token")
		return
	}

	expiry := time.Now().Add(24 * time.Hour)
	if err := s.verificationRepo.Store(ctx, user.ID, token, expiry); err != nil {
		s.log.Error().Err(err).Str("user_id", user.ID).Msg("failed to store verification token")
		return
	}

	link := fmt.Sprintf("%s/auth/verify?token=%s", s.cfg.AppBaseURL, token)
	subject := "Verify your email address"
	body := fmt.Sprintf(
		"Hi,<br><br>Thanks for signing up. Please verify your email address by clicking the link below:<br><br>"+
			"<a href=\"%s\">Verify Email</a><br><br>"+
			"This link expires in 24 hours.<br><br>"+
			"If you did not create an account, you can ignore this email.",
		link,
	)

	if err := s.mailer.Send(ctx, user.Email, subject, body); err != nil {
		s.log.Error().Err(err).Str("user_id", user.ID).Msg("failed to send verification email")
		return
	}

	s.log.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("verification email sent")
}

// generateToken creates a cryptographically random 32-byte hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
