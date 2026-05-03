// Package server — router configuration for Inventory Service.
package server

import (
	"retail-platform/inventory/internal/handler"
	"retail-platform/pkg/jwt"
	"retail-platform/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures the Gin router with all routes.
func NewRouter(h *handler.InventoryHandler, jwtManager *jwt.Manager) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(middleware.RequestID())
	r.Use(gin.Recovery())

	// Health endpoints — no auth required
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	// All inventory routes require a valid JWT
	// The JWT is issued by Auth Service — same secret
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		// Product catalogue — any authenticated user
		protected.GET("/products", h.ListProducts)
		protected.GET("/products/:id", h.GetProduct)
		protected.GET("/products/:id/stock", h.GetStockLevel)

		// Stock operations — any authenticated user (Order Service calls these)
		protected.POST("/inventory/reserve", h.Reserve)
		protected.POST("/inventory/release", h.Release)

		// Stock adjustment — admin only
		// RequireRole must come AFTER AuthMiddleware
		protected.PATCH("/products/:id/stock",
			middleware.RequireRole("admin"),
			h.AdjustStock,
		)
	}

	return r
}
