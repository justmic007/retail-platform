// Package jwt provides JWT token generation and validation for all services. Order Service and Inventory Service validate incoming tokens from clients, using the same secret that Auth Service used to sign them.

package jwt

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5" // golang-jwt handles signing, parsing, and validating JWTs.
)

// Claims defines what is encoded inside our JWTs.
// jwt.RegisteredClaims gives us the ExpiresAt, IssuedAt, Subject

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Manager holds the JWT configuration and handles all token operations.
// Created once in main.go and injected into AuthService.

type Manager struct {
	secret          []byte        // the signing secret — must stay private
	accessTokenTTL  time.Duration // how long access tokens are valid (15 minutes)
	refreshTokenTTL time.Duration // how long refresh tokens are valid (168h = 7d)
}

// NewManager creates a new JWT Manager.
// secret is the JWT_SECRET environment variable - used to sign and verify tokens.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// GenerateAccessToken creates a signed JWT access token for the given user.
func (m *Manager) GenerateAccessToken(userID, role string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		},
	}

	// jwt.NewWithClaims creates an unsigned token with the claims
	// SignedString(m.secret) signs it with our secret using HS256}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT string.
// Returns the Claims if valid, or an error if:
//   - the token is malformed
//   - the signature doesn't match our secret (tampered token)
//   - the token has expired
//
// This is called by AuthMiddleware in pkg/middleware/auth.go on every
// protected request. It's designed to be fast — just HMAC verification,
// no database lookup needed.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	// jwt.ParseWithClaims parses the token string and populates our Claims struct.
	// The key function returns our secret — jwt uses it to verify the signature.

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify the signing method is what we expect.
		// This prevents algorithm confusion attacks where an attacker sends
		// a token signed with "none" or a different algorithm.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}


// GenerateRefreshToken creates a cryptographically random refresh token.
//
// crypto/rand generates true randomness from the OS entropy source.
// math/rand is NOT suitable for security — it's predictable.
func (m *Manager) GenerateRefreshToken() (string, error) {
	// 32 bytes = 256 bits of entropy = practically unguessable
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes): err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	// hex.EncodeToString converts bytes to a hex string (64 characters)
	return hex.EncodeToString(bytes), nil
}

// RefreshTokenExpiry returns the expiry time for a new refresh token.
// Used when storing the token in the database.
func (m *Manager) RefreshTokenExpiry() time.Time {
	return time.Now().Add(m.refreshTokenTTL)
}