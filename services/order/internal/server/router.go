// Package server — router configuration for Order Service.
package server

import (
	"retail-platform/order/internal/handler"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures the Gin router with all routes.
// All order routes require authentication — no public routes except health.
func NewRouter(h *handler.OrderHandler, jwtManager *jwt.Manager) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(gin.Recovery())

	// Health endpoints — no auth required
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	// All order routes require a valid JWT and a verified email
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	protected.Use(middleware.RequireEmailVerified())
	{
		protected.POST("/orders", h.CreateOrder)
		protected.GET("/orders", h.ListOrders)
		protected.GET("/orders/:id", h.GetOrder)
		protected.PATCH("/orders/:id/cancel", h.CancelOrder)
	}

	return r
}
