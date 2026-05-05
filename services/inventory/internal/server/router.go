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

	// PUBLIC — no auth required (browsing the product catalogue)
	r.GET("/products", h.ListProducts)
	r.GET("/products/:id", h.GetProduct)
	r.GET("/products/:id/stock", h.GetStockLevel)

	// All inventory routes require a valid JWT
	// The JWT is issued by Auth Service — same secret
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		// Stock operations — any authenticated user (Order Service calls these)
		protected.POST("/inventory/reserve", h.Reserve)
		protected.POST("/inventory/release", h.Release)

		// ADMIN — JWT + admin role required
		admin := r.Group("/")
		admin.Use(middleware.AuthMiddleware(jwtManager))
		admin.Use(middleware.RequireRole("admin"))
		{
			admin.PATCH("/products/:id/stock", h.AdjustStock)
		}
	}

	return r
}
