// Package middleware — CORS middleware for all services.
// Allows the Next.js frontend (localhost:3000) to call the backend APIs
// from the browser without being blocked by the same-origin policy.
package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// CORS returns a Gin middleware that sets the appropriate CORS headers.
// In development it allows localhost:3000 (Next.js dev server).
// In production it reads ALLOWED_ORIGIN from the environment.
//
// Why CORS?
// Browsers enforce the same-origin policy — a page on localhost:3000
// cannot call an API on localhost:8080 unless the API explicitly allows it.
// CORS headers tell the browser: "yes, this origin is allowed".
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := allowedOrigin()

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24h preflight cache

		// Handle preflight OPTIONS request — browser sends this before
		// the actual request to check if CORS is allowed.
		// Must return 204 No Content immediately — no handler needed.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// allowedOrigin returns the allowed origin from env or defaults to localhost:3000.
func allowedOrigin() string {
	if origin := os.Getenv("ALLOWED_ORIGIN"); origin != "" {
		return origin
	}
	return "http://localhost:3000"
}
