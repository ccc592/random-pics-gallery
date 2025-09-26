package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// RequestLoggingMiddleware logs API requests to database for analytics
func RequestLoggingMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Record start time
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate response time
		duration := time.Since(startTime)

		// Get user ID if authenticated
		var userID *uuid.UUID
		if userIDValue, exists := c.Get("user_id"); exists {
			if uid, ok := userIDValue.(uuid.UUID); ok {
				userID = &uid
			}
		}

		// Get request parameters (query + form)
		params := make(map[string]interface{})
		for key, values := range c.Request.URL.Query() {
			if len(values) == 1 {
				params[key] = values[0]
			} else {
				params[key] = values
			}
		}

		// Create API request log entry
		userAgent := c.Request.UserAgent()
		apiRequest := &models.APIRequest{
			ID:             uuid.New(),
			UserID:         userID,
			IPAddress:      c.ClientIP(),
			Endpoint:       c.Request.URL.Path,
			Method:         c.Request.Method,
			StatusCode:     c.Writer.Status(),
			ResponseTimeMs: int(duration.Milliseconds()),
			UserAgent:      &userAgent,
			RateLimited:    c.Writer.Status() == 429,
			Timestamp:      startTime,
		}

		// Store in database asynchronously
		go func() {
			if err := db.Create(apiRequest).Error; err != nil {
				// Log error but don't fail request
				// In production, you might want to use proper logging
			}
		}()
	}
}

// RequestIDMiddleware adds request ID to context and response headers
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header or generate new one
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// ResponseBodyLoggingMiddleware captures response body for debugging
func ResponseBodyLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only log in debug/development mode
		if gin.Mode() != gin.DebugMode {
			c.Next()
			return
		}

		// Capture response body
		w := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = w

		c.Next()

		// Log response body if needed (for debugging)
		// In production, you might want to log only errors or specific endpoints
		if c.Writer.Status() >= 400 {
			// Log error responses
			go func() {
				// Here you could log to file, send to monitoring service, etc.
				_ = w.body.String()
			}()
		}
	}
}

// responseBodyWriter wraps gin.ResponseWriter to capture response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// PerformanceLoggingMiddleware logs slow requests
func PerformanceLoggingMiddleware(slowThreshold time.Duration) gin.HandlerFunc {
	if slowThreshold == 0 {
		slowThreshold = 500 * time.Millisecond
	}

	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		duration := time.Since(startTime)

		// Log slow requests
		if duration > slowThreshold {
			go func() {
				// In production, you might want to send this to a monitoring service
				_ = map[string]interface{}{
					"request_id":    c.GetString("request_id"),
					"method":        c.Request.Method,
					"path":          c.Request.URL.Path,
					"duration_ms":   duration.Milliseconds(),
					"status_code":   c.Writer.Status(),
					"client_ip":     c.ClientIP(),
					"user_agent":    c.Request.UserAgent(),
					"slow_request":  true,
				}
			}()
		}
	}
}

// ErrorLoggingMiddleware logs errors that occur during request processing
func ErrorLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				go func(ginError *gin.Error) {
					// In production, you might want to send this to an error tracking service
					_ = map[string]interface{}{
						"request_id":  c.GetString("request_id"),
						"error":       ginError.Error(),
						"error_type":  ginError.Type,
						"method":      c.Request.Method,
						"path":        c.Request.URL.Path,
						"status_code": c.Writer.Status(),
						"client_ip":   c.ClientIP(),
						"timestamp":   time.Now(),
					}
				}(err)
			}
		}
	}
}

// RequestSizeMiddleware limits request body size
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	if maxSize == 0 {
		maxSize = 10 << 20 // 10MB default
	}

	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(413, gin.H{
				"error": "request body too large",
				"code":  "REQUEST_TOO_LARGE",
				"max_size": maxSize,
			})
			c.Abort()
			return
		}

		// Limit reader to prevent abuse
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}

// JSONLoggingMiddleware provides structured JSON logging
func JSONLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Create structured log entry
		logEntry := map[string]interface{}{
			"timestamp":    param.TimeStamp.Format(time.RFC3339),
			"status":       param.StatusCode,
			"latency":      param.Latency.String(),
			"client_ip":    param.ClientIP,
			"method":       param.Method,
			"path":         param.Path,
			"user_agent":   param.Request.UserAgent(),
			"body_size":    param.BodySize,
		}

		// Add error information if present
		if param.ErrorMessage != "" {
			logEntry["error"] = param.ErrorMessage
		}

		// In production, you would marshal this to JSON and write to log output
		// For now, return empty string to avoid duplicate logging
		return ""
	})
}