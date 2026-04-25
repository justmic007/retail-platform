// Package server wires together the HTTP server, router, and middleware chain.
package server

import (
	"retail-platform/auth/internal/handler"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures the Gin router with all routes
// and middleware attached. Keeping this separate from server.go means
// you can read all routing configuration in one place without noise
// from server lifecycle code.
func NewRouter(h *handler.AuthHandler, jwtManager *jwt.Manager) *gin.Engine {
	// gin.New() creates a router with NO default middleware.
	// gin.Default() adds Logger and Recovery middleware automatically —
	// we don't want that because we use our own structured logger.
	r := gin.New()

	// ── Global middleware — runs on EVERY request ─────────────────────────
	// Order matters: RequestID first so every subsequent log line has the ID.

	// 1. RequestID — generate unique ID per request
	r.Use(middleware.RequestID())

	// 2. Recovery — catches panics and returns 500 instead of crashing.
	// Must be after RequestID so recovery logs include the request ID.
	r.Use(gin.Recovery())

	// ── Health endpoints — no auth, no rate limiting ───────────────────────
	// These must be reachable by Kubernetes probes without any credentials.
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	// ── Public auth routes — no JWT required ──────────────────────────────
	// Anyone can call these — they're how you get a token in the first place.
	public := r.Group("/auth")
	{
		public.POST("/register", h.Register)
		public.POST("/login", h.Login)
		public.POST("/refresh", h.Refresh)
	}

	// ── Protected auth routes — valid JWT required ─────────────────────────
	// AuthMiddleware validates the JWT on every request in this group.
	// If the token is missing or invalid, the middleware returns 401
	// and the handler never runs.
	protected := r.Group("/auth")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		protected.GET("/me", h.Me)
		protected.POST("/logout", h.Logout)
	}

	return r
}
