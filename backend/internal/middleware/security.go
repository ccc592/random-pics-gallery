package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	AllowedOrigins       []string
	JWTSecret            string
	CSRFSecret           string
	ContentSecurityPolicy string
	StrictTransportSecurity string
	HSTSMaxAge           int
	EnableHSTS           bool
	TrustedProxies       []string
	AllowedHosts         []string
}

// SecurityHeadersMiddleware adds comprehensive security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy for API endpoints
		csp := "default-src 'none'; " +
			"script-src 'none'; " +
			"style-src 'none'; " +
			"img-src 'none'; " +
			"connect-src 'none'; " +
			"font-src 'none'; " +
			"object-src 'none'; " +
			"media-src 'none'; " +
			"child-src 'none'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'none'; " +
			"form-action 'none'"
		c.Header("Content-Security-Policy", csp)

		// HSTS (HTTP Strict Transport Security)
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Permissions Policy (formerly Feature Policy)
		permissions := "camera=(), microphone=(), geolocation=(), payment=(), " +
			"usb=(), magnetometer=(), gyroscope=(), accelerometer=(), " +
			"ambient-light-sensor=(), autoplay=(), encrypted-media=(), " +
			"fullscreen=(), picture-in-picture=()"
		c.Header("Permissions-Policy", permissions)

		// Server identification
		c.Header("Server", "randompic-api")

		c.Next()
	}
}

// InputValidationMiddleware validates and sanitizes input
func InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate Content-Length for POST/PUT/PATCH requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentLength := c.Request.ContentLength
			maxBodySize := int64(10 << 20) // 10MB default

			if contentLength > maxBodySize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "request body too large",
					"code":  "REQUEST_TOO_LARGE",
					"max_size": maxBodySize,
				})
				c.Abort()
				return
			}
		}

		// Validate Content-Type for JSON endpoints
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if strings.Contains(c.Request.URL.Path, "/api/") {
				contentType := c.GetHeader("Content-Type")
				if contentType != "" && !strings.Contains(contentType, "application/json") &&
				   !strings.Contains(contentType, "multipart/form-data") {
					c.JSON(http.StatusUnsupportedMediaType, gin.H{
						"error": "unsupported content type",
						"code":  "UNSUPPORTED_MEDIA_TYPE",
						"supported": []string{"application/json", "multipart/form-data"},
					})
					c.Abort()
					return
				}
			}
		}

		// Sanitize headers
		sanitizeHeaders(c)

		c.Next()
	}
}

// CSRFProtectionMiddleware provides CSRF protection
func CSRFProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF protection for GET, HEAD, OPTIONS requests
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip for API endpoints using Bearer token authentication
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		// For cookie-based authentication, require CSRF token
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = c.PostForm("csrf_token")
		}

		// Get stored CSRF token from session/cookie
		storedToken, err := c.Cookie("csrf_token")
		if err != nil || storedToken == "" || !isValidCSRFToken(csrfToken, storedToken) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token validation failed",
				"code":  "CSRF_INVALID",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HostValidationMiddleware validates the Host header
func HostValidationMiddleware(allowedHosts []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		if host == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Host header is required",
				"code":  "MISSING_HOST_HEADER",
			})
			c.Abort()
			return
		}

		// Remove port from host for validation
		hostWithoutPort := strings.Split(host, ":")[0]

		// Check if host is allowed
		allowed := false
		for _, allowedHost := range allowedHosts {
			if allowedHost == hostWithoutPort || allowedHost == "*" {
				allowed = true
				break
			}
			// Support wildcard subdomains
			if strings.HasPrefix(allowedHost, "*.") {
				domain := strings.TrimPrefix(allowedHost, "*.")
				if strings.HasSuffix(hostWithoutPort, "."+domain) || hostWithoutPort == domain {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid Host header",
				"code":  "INVALID_HOST",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestSizeMiddleware limits request size
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request too large",
				"code":  "REQUEST_TOO_LARGE",
				"max_size": maxSize,
			})
			c.Abort()
			return
		}

		// Set a http.MaxBytesReader to prevent reading more than maxSize
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}

// TrustedProxyMiddleware validates trusted proxies
func TrustedProxyMiddleware(trustedProxies []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If no trusted proxies configured, skip validation
		if len(trustedProxies) == 0 {
			c.Next()
			return
		}

		remoteIP := c.Request.RemoteAddr
		if remoteIP != "" {
			// Extract IP from remote address (remove port)
			ip := strings.Split(remoteIP, ":")[0]

			// Check if the request comes from a trusted proxy
			trusted := false
			for _, proxy := range trustedProxies {
				if proxy == ip || proxy == "*" {
					trusted = true
					break
				}
			}

			if !trusted && (c.GetHeader("X-Forwarded-For") != "" ||
				c.GetHeader("X-Real-IP") != "" ||
				c.GetHeader("X-Forwarded-Proto") != "") {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Untrusted proxy",
					"code":  "UNTRUSTED_PROXY",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// APIKeyMiddleware validates API keys (for public API access)
func APIKeyMiddleware(validKeys []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key required",
				"code":  "API_KEY_REQUIRED",
			})
			c.Abort()
			return
		}

		// Validate API key
		valid := false
		for _, key := range validKeys {
			if key == apiKey {
				valid = true
				break
			}
		}

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
				"code":  "INVALID_API_KEY",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Configuration functions

// Helper functions

func sanitizeHeaders(c *gin.Context) {
	// List of potentially dangerous headers to sanitize
	dangerousHeaders := []string{
		"X-Forwarded-Host",
		"X-Forwarded-Server",
		"X-Rewrite-URL",
		"X-Original-URL",
		"X-Forwarded-Prefix",
	}

	for _, header := range dangerousHeaders {
		c.Request.Header.Del(header)
	}

	// Sanitize User-Agent to prevent extremely long values
	userAgent := c.GetHeader("User-Agent")
	if len(userAgent) > 500 {
		c.Request.Header.Set("User-Agent", userAgent[:500])
	}
}

func isValidCSRFToken(provided, stored string) bool {
	// Simple CSRF token validation
	// In production, you might want to use a more sophisticated method
	return provided != "" && stored != "" && provided == stored
}

// GenerateCSRFToken generates a new CSRF token
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SetCSRFToken sets a CSRF token cookie
func SetCSRFToken(c *gin.Context) {
	token, err := GenerateCSRFToken()
	if err != nil {
		// Log error but don't fail the request
		return
	}

	c.SetCookie(
		"csrf_token",
		token,
		3600, // 1 hour
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HttpOnly
	)

	// Also set in response header for client-side access
	c.Header("X-CSRF-Token", token)
}

func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:4321",
			"https://*.vercel.app",
			"https://*.netlify.app",
		},
		JWTSecret:      "your-jwt-secret", // Should be loaded from environment
		CSRFSecret:     "your-csrf-secret", // Should be loaded from environment
		EnableHSTS:     true,
		HSTSMaxAge:     31536000, // 1 year
		TrustedProxies: []string{"127.0.0.1", "::1"},
		AllowedHosts: []string{
			"localhost",
			"127.0.0.1",
			"*.example.com", // Replace with your domain
		},
	}
}

// Security validation functions

var (
	// Common patterns for malicious input
	sqlInjectionPattern = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)`)
	xssPattern         = regexp.MustCompile(`(?i)(<script|javascript:|on\w+\s*=)`)
	pathTraversalPattern = regexp.MustCompile(`\.\.[\\/]`)
)

// ValidateInput performs basic input validation and sanitization
func ValidateInput(input string) bool {
	// Check for SQL injection patterns
	if sqlInjectionPattern.MatchString(input) {
		return false
	}

	// Check for XSS patterns
	if xssPattern.MatchString(input) {
		return false
	}

	// Check for path traversal
	if pathTraversalPattern.MatchString(input) {
		return false
	}

	return true
}

// SanitizeFilename sanitizes file names for safe storage
func SanitizeFilename(filename string) string {
	// Remove path components
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "..", "")

	// Limit length
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}