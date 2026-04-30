// auth.go — JWT validation middleware shared across all services.
// Order Service and Inventory Service import and use this same middleware.
// Written once in pkg/, used everywhere.

package middleware

import (
	"net/http"
	"strings"

	"retail-platform/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// UserIDKey and RoleKey are context keys set by AuthMiddleware.
// Handlers retrieve the authenticated user's identity via:
//
//	userID := c.GetString(middleware.UserIDKey)
//	role   := c.GetString(middleware.RoleKey)

const (
	UserIDKey = "user_id"
	RoleKey   = "role"
)

// AuthMiddleware returns a Gin middleware function that validates JWTs.
// It takes a *jwt.Manager so it can validate tokens using the same
// secret that Auth Service used to sign them.
//
// How it works on every protected request:
//  1. Extracts the Authorization header
//  2. Strips the "Bearer " prefix to get the raw token string
//  3. Calls jwtManager.ValidateToken() — checks signature + expiry
//  4. If valid: sets user_id and role in context, calls c.Next()
//  5. If invalid: returns 401 and ABORTS — handler never runs
//
// Usage in router.go:
//
//	protected := r.Group("/orders")
//	protected.Use(middleware.AuthMiddleware(jwtManager))

func AuthMiddleware(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── Step 1: Extract the Authorization header ──────────────────────
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		// ── Step 2: Validate format and strip "Bearer " prefix ────────────
		// The Authorization header must be in the format: "Bearer <token>"
		// strings.HasPrefix checks it starts with "Bearer "
		// strings.TrimPrefix removes that prefix leaving just the token.
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header must start with 'Bearer '",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token is required",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		// ── Step 3: Validate the token ────────────────────────────────────
		// ValidateToken checks:
		//   - the signature (was it signed with our secret?)
		//   - the expiry (has it expired?)
		//   - the algorithm (is it HS256 as expected?)
		// No database lookup — pure cryptographic verification.
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		// ── Step 4: Set user identity in context ──────────────────────────
		// Any handler after this middleware can read the authenticated user's
		// identity without touching the database — it's all from the JWT.
		c.Set(UserIDKey, claims.UserID)
		c.Set(RoleKey, claims.Role)

		// ── Step 5: Continue to the handler ──────────────────────────────
		// c.Next() passes control to the next middleware or handler.
		// c.Abort() (used above in error cases) stops the chain here —
		// the handler never runs if the token is invalid.
		c.Next()
	}
}

// RequireRole returns a middleware that checks the authenticated user's role.
// Use AFTER AuthMiddleware — it assumes user_id and role are already in context.
//
// Usage: admin-only route
//
//	admin := r.Group("/admin")
//	admin.Use(middleware.AuthMiddleware(jwtManager))
//	admin.Use(middleware.RequireRole("admin"))
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString(RoleKey)

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		// User is authenticated but doesn't have the required role.
		// 403 Forbidden (not 401 Unauthorized) — they ARE logged in,
		// they just don't have permission.
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions",
			"code":  "FORBIDDEN",
		})
	}
}
