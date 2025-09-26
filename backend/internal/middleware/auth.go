package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/auth"
)

// AuthMiddleware creates JWT authentication middleware
func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate JWT token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Store user claims in context
		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// OptionalAuthMiddleware allows both authenticated and anonymous requests
func OptionalAuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header - continue as anonymous user
			c.Next()
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format - continue as anonymous user
			c.Next()
			return
		}

		token := parts[1]

		// Try to validate JWT token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			// Invalid token - continue as anonymous user
			c.Next()
			return
		}

		// Store user claims in context if valid
		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// AdminOnlyMiddleware ensures only admin users can access the endpoint
func AdminOnlyMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated first
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		claims, ok := user.(*auth.Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user context",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Check if user has admin role
		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "admin access required",
				"code":  "FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserOnlyMiddleware ensures only authenticated users can access the endpoint
func UserOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated
		_, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}