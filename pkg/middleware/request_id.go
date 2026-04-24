// Package middleware provides shared HTTP middleware for all services.
// Middleware sits between the router and the handler — every request
// passes through it before reaching the handler function.

package middleware

import (
	"github.com/gin-gonic/gin"
)

// RequestIDKey is the key used to store the request ID in Gin's context.
const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the client already sent a request ID.
		// Some API gateways (like AWS API Gateway) inject their own request IDs.
		// We honour it if present — maintains tracing continuity end to end.
		requestID := c.GetHeader("X-Request-ID")

		// If no incoming request ID, generate a fresh UUID v4.
		// uuid.New() uses crypto/rand internally — guaranteed unique.

		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in Gin context — accessible to all subsequent middleware
		// and handlers via c.GetString(middleware.RequestIDKey)
		c.Set(RequestIDKey, requestID)

		// Set on the response header — client receives it back.
		// This is what the client quotes in a support ticket.
		c.Header("X-Request-ID", requestID)

		// c.Next() passes control to the next middleware or handler.
		// Without this, the request stops here and never reaches your handler.
		// This is how Gin's middleware chain works — each middleware calls
		// c.Next() to continue, or returns early to abort the request.

		c.Next()
	}
}
