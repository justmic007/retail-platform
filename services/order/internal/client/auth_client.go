// Package client provides inter-service HTTP clients for the order service.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"retail-platform/order/internal/config"
)

// loginRequest is the body sent to Auth Service to get a JWT token.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the response from Auth Service containing the JWT token.
type loginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// ServiceTokenManager logs in with a service account at startup and
// refreshes the token before it expires. Thread-safe - multiple worker
// goroutes call GetToken() concurrently.
type ServiceTokenManager struct {
	authURL  string
	email    string
	password string
	token    string
	expiry   time.Time
	mu       sync.RWMutex
	client   *http.Client
}

// NewServiceTokenManager creates a new ServiceTokenManager with the given credentials
// and logs in immediately. Returns an error if the initial login fails - the service should not start
// without a valid token.
func NewServiceTokenManager(ctx context.Context, cfg *config.Config) (*ServiceTokenManager, error) {
	m := &ServiceTokenManager{
		authURL:  cfg.AuthServiceURL,
		email:    cfg.OrderServiceEmail,
		password: cfg.OrderServicePassword,
		client:   &http.Client{Timeout: 10 * time.Second},
	}

	if _, err := m.refreshToken(ctx); err != nil {
		return nil, fmt.Errorf("initial service account login failed: %w", err)
	}

	return m, nil
}

// GetToken returns a valid JWT for the service account
// If the token is within 1 minute of expiry, it refreshes first.
// Safe to call from multiple goroutines concurrently.
func (m *ServiceTokenManager) GetToken(ctx context.Context) (string, error) {
	m.mu.RLock() // Aquires a read lock to check token expiry. Multiple goroutines can read concurrently.
	if time.Now().Before(m.expiry.Add(-1 * time.Minute)) {
		token := m.token //Copies the token string into a local variable before releasing the lock
		m.mu.RUnlock()   // Releases the read lock and returns the cached token
		return token, nil
	}
	m.mu.RUnlock() // Token is expired or close to expiry — release the read lock before trying to refresh

	return m.refreshToken(ctx)
}

// refreshToken logs in with the service account and stores the new token.
func (m *ServiceTokenManager) refreshToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	body, err := json.Marshal(loginRequest{
		Email:    m.email,
		Password: m.password,
	})
	if err != nil {
		return "", fmt.Errorf("marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		m.authURL+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}

	m.token = loginResp.AccessToken
	m.expiry = time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second)

	return m.token, nil
}
