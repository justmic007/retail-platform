// Package handler contains the HTTP handlers for the Auth Service.
// Handlers are the thinnest possible layer — they translate HTTP
// into service calls and service results into HTTP responses.
// No business logic lives here.
package handler

import (
	"net/http"

	"retail-platform/auth/internal/domain"
	"retail-platform/auth/internal/service"
	"retail-platform/pkg/errors"
	"retail-platform/pkg/logger"
	"retail-platform/pkg/middleware"
	"retail-platform/pkg/validator"

	"github.com/gin-gonic/gin"
)

// AuthHandler holds the dependencies needed by all auth HTTP handlers.
type AuthHandler struct {
	service   *service.AuthService
	validator *validator.Validator
	log       *logger.Logger
}

// NewAuthHandler creates a new AuthHandler with injected dependencies.
func NewAuthHandler(
	svc *service.AuthService,
	v *validator.Validator,
	log *logger.Logger,
) *AuthHandler {
	return &AuthHandler{
		service:   svc,
		validator: v,
		log:       log,
	}
}

// Register handles POST /auth/register
// Creates a new user account.
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest

	// ShouldBindJSON reads the request body and decodes JSON into req.
	// Returns error if body is missing, not valid JSON, or wrong types.
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "BAD_REQUEST",
		})
		return
	}

	// Validate struct fields against their `validate` tags.
	// e.g. email format, password minimum length.
	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	// Call the service — all business logic happens there.
	user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// 201 Created — resource was created successfully.
	// user.ToResponse() strips the password hash before serialising to JSON.
	c.JSON(http.StatusCreated, gin.H{
		"user": user.ToResponse(),
	})
}

// Login handles POST /auth/login
// Authenticates a user and returns access + refresh tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	tokens, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Refresh handles POST /auth/refresh
// Issues a new access token using a valid refresh token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req domain.RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	tokens, err := h.service.Refresh(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Logout handles POST /auth/logout
// Invalidates the user's refresh token.
// Requires authentication — user must send their access token.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req domain.RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "refresh_token is required in request body",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// Me handles GET /auth/me
// Returns the currently authenticated user's profile.
// Requires authentication — AuthMiddleware sets user_id in context.
func (h *AuthHandler) Me(c *gin.Context) {
	// Read the user ID that AuthMiddleware set in the context.
	// If AuthMiddleware didn't run (route misconfiguration), this would be empty.
	userID := c.GetString(middleware.UserIDKey)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

// Health handles GET /health — liveness probe.
// Returns 200 if the process is running.
// Kubernetes uses this to know if the pod should be restarted.
func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "auth",
	})
}

// Ready handles GET /ready — readiness probe.
// Returns 200 only if the service is ready to handle traffic.
// Kubernetes uses this to know if the pod should receive traffic.
// Unlike /health, this checks actual dependencies (DB).
func (h *AuthHandler) Ready(c *gin.Context) {
	// In a full implementation, this would ping the DB pool.
	// For now it returns 200 — we'll enhance this in Day 5.
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "auth",
	})
}

// handleError is a shared error mapper used by all handlers.
// It converts domain/app errors into appropriate HTTP responses.
// This is the ONLY place in the handler layer that knows about error codes.
func (h *AuthHandler) handleError(c *gin.Context, err error) {
	// Check if it's one of our typed AppErrors from pkg/errors.
	if appErr, ok := errors.IsAppError(err); ok {
		c.JSON(appErr.HTTPStatus(), gin.H{
			"error": appErr.Message,
			"code":  string(appErr.Code),
		})
		return
	}

	// Unknown error — log it (it may contain sensitive info) and
	// return a generic 500 to the client (never expose internal errors).
	h.log.Error().Err(err).Msg("unhandled internal error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "an internal error occurred",
		"code":  "INTERNAL_ERROR",
	})
}
