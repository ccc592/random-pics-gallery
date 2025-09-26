package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	EnableRequestLogging  bool
	EnableResponseLogging bool
	EnablePerformanceLog  bool
	EnableErrorLogging    bool
	SlowThreshold         time.Duration
	LogLevel              string
	MaxBodySize           int
	SensitiveFields       []string
}

// APIRequestLog represents a structured API request log entry
type APIRequestLog struct {
	RequestID      string                 `json:"request_id"`
	Timestamp      time.Time              `json:"timestamp"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	Query          string                 `json:"query,omitempty"`
	StatusCode     int                    `json:"status_code"`
	ResponseTime   int                    `json:"response_time_ms"`
	RequestSize    int                    `json:"request_size"`
	ResponseSize   int                    `json:"response_size"`
	ClientIP       string                 `json:"client_ip"`
	UserAgent      string                 `json:"user_agent"`
	UserID         *string                `json:"user_id,omitempty"`
	UserRole       string                 `json:"user_role,omitempty"`
	Headers        map[string]interface{} `json:"headers,omitempty"`
	RequestBody    interface{}            `json:"request_body,omitempty"`
	ResponseBody   interface{}            `json:"response_body,omitempty"`
	Errors         []string               `json:"errors,omitempty"`
	IsSlowRequest  bool                   `json:"is_slow_request"`
	IsError        bool                   `json:"is_error"`
	RateLimited    bool                   `json:"rate_limited"`
}

// RequestLoggingMiddleware creates comprehensive request logging middleware
func RequestLoggingMiddleware() gin.HandlerFunc {
	config := DefaultLoggingConfig()
	return RequestLoggingMiddlewareWithConfig(nil, config)
}

// RequestLoggingMiddlewareWithConfig creates logging middleware with database integration
func RequestLoggingMiddlewareWithConfig(db *gorm.DB, config *LoggingConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Record start time
		startTime := time.Now()

		// Capture request body if enabled
		var requestBody []byte
		var requestSize int
		if config.EnableRequestLogging && shouldLogBody(c.Request.Method) {
			requestBody, _ = captureRequestBody(c, config.MaxBodySize)
			requestSize = len(requestBody)
		}

		// Wrap response writer to capture response
		responseCapture := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseCapture

		// Process request
		c.Next()

		// Calculate response time
		duration := time.Since(startTime)
		responseTime := int(duration.Milliseconds())

		// Create log entry
		logEntry := &APIRequestLog{
			RequestID:     requestID,
			Timestamp:     startTime,
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			Query:         c.Request.URL.RawQuery,
			StatusCode:    c.Writer.Status(),
			ResponseTime:  responseTime,
			RequestSize:   requestSize,
			ResponseSize:  responseCapture.body.Len(),
			ClientIP:      c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			IsSlowRequest: duration > config.SlowThreshold,
			IsError:       c.Writer.Status() >= 400,
			RateLimited:   c.Writer.Status() == http.StatusTooManyRequests,
		}

		// Add user information if authenticated
		if userID, exists := c.Get("user_id"); exists {
			if userIDStr, ok := userID.(string); ok {
				logEntry.UserID = &userIDStr
			}
		}

		if userRole, exists := c.Get("user_role"); exists {
			if roleStr, ok := userRole.(string); ok {
				logEntry.UserRole = roleStr
			}
		}

		// Add request headers (filtered)
		if config.EnableRequestLogging {
			logEntry.Headers = captureHeaders(c.Request.Header, config.SensitiveFields)
		}

		// Add request body
		if config.EnableRequestLogging && len(requestBody) > 0 {
			logEntry.RequestBody = sanitizeBody(requestBody, config.SensitiveFields)
		}

		// Add response body if enabled
		if config.EnableResponseLogging && (logEntry.IsError || gin.Mode() == gin.DebugMode) {
			logEntry.ResponseBody = sanitizeBody(responseCapture.body.Bytes(), config.SensitiveFields)
		}

		// Add errors
		if len(c.Errors) > 0 {
			errors := make([]string, len(c.Errors))
			for i, err := range c.Errors {
				errors[i] = err.Error()
			}
			logEntry.Errors = errors
		}

		// Log the request
		go logRequest(db, logEntry)
	}
}

// ResponseLoggingMiddleware captures response data
func ResponseLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if request logging is already handling this
		if _, exists := c.Get("request_logging_enabled"); exists {
			c.Next()
			return
		}

		responseCapture := &responseCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseCapture

		c.Next()

		// Log response if it's an error
		if c.Writer.Status() >= 400 {
			go func() {
				responseData := responseCapture.body.String()
				logData := map[string]interface{}{
					"request_id":    c.GetString("request_id"),
					"status_code":   c.Writer.Status(),
					"response_body": responseData,
					"path":          c.Request.URL.Path,
					"method":        c.Request.Method,
					"timestamp":     time.Now(),
				}
				// In production, send to logging service
				_ = logData
			}()
		}
	}
}

// PerformanceMiddleware logs performance metrics
func PerformanceMiddleware() gin.HandlerFunc {
	return PerformanceMiddlewareWithThreshold(500 * time.Millisecond)
}

// PerformanceMiddlewareWithThreshold creates performance logging with custom threshold
func PerformanceMiddlewareWithThreshold(slowThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		duration := time.Since(startTime)

		// Log slow requests
		if duration > slowThreshold {
			go func() {
				performanceLog := map[string]interface{}{
					"request_id":    c.GetString("request_id"),
					"method":        c.Request.Method,
					"path":          c.Request.URL.Path,
					"duration_ms":   duration.Milliseconds(),
					"status_code":   c.Writer.Status(),
					"client_ip":     c.ClientIP(),
					"user_agent":    c.Request.UserAgent(),
					"slow_request":  true,
					"timestamp":     startTime,
				}

				if userID, exists := c.Get("user_id"); exists {
					performanceLog["user_id"] = userID
				}

				// In production, send to monitoring service
				logPerformanceData(performanceLog)
			}()
		}
	}
}

// RecordAPIRequest records API request in database
func RecordAPIRequest(db *gorm.DB, req *APIRequestLog) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Convert to database model
	apiRequest := &models.APIRequest{
		UserID:       req.UserID,
		IPAddress:    req.ClientIP,
		Method:       req.Method,
		Endpoint:     req.Path,
		StatusCode:   req.StatusCode,
		ResponseTime: req.ResponseTime,
		RequestSize:  req.RequestSize,
		ResponseSize: req.ResponseSize,
		UserAgent:    req.UserAgent,
		CreatedAt:    req.Timestamp,
	}

	return db.Create(apiRequest).Error
}

// ErrorLoggingMiddleware logs errors with context
func ErrorLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				go func(ginError *gin.Error) {
					errorLog := map[string]interface{}{
						"request_id":  c.GetString("request_id"),
						"error":       ginError.Error(),
						"error_type":  ginError.Type,
						"method":      c.Request.Method,
						"path":        c.Request.URL.Path,
						"status_code": c.Writer.Status(),
						"client_ip":   c.ClientIP(),
						"user_agent":  c.Request.UserAgent(),
						"timestamp":   time.Now(),
					}

					if userID, exists := c.Get("user_id"); exists {
						errorLog["user_id"] = userID
					}

					// In production, send to error tracking service
					logErrorData(errorLog)
				}(err)
			}
		}

		// Log HTTP errors (4xx, 5xx)
		if c.Writer.Status() >= 400 {
			go func() {
				errorLog := map[string]interface{}{
					"request_id":  c.GetString("request_id"),
					"status_code": c.Writer.Status(),
					"method":      c.Request.Method,
					"path":        c.Request.URL.Path,
					"client_ip":   c.ClientIP(),
					"user_agent":  c.Request.UserAgent(),
					"query":       c.Request.URL.RawQuery,
					"timestamp":   time.Now(),
				}

				if userID, exists := c.Get("user_id"); exists {
					errorLog["user_id"] = userID
				}

				// Add response body for errors
				if responseCapture, ok := c.Writer.(*responseCapture); ok {
					errorLog["response_body"] = responseCapture.body.String()
				}

				logErrorData(errorLog)
			}()
		}
	}
}

// StructuredLoggingMiddleware provides structured JSON logging
func StructuredLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logEntry := map[string]interface{}{
			"timestamp":   param.TimeStamp.Format(time.RFC3339),
			"status":      param.StatusCode,
			"latency":     param.Latency.Milliseconds(),
			"client_ip":   param.ClientIP,
			"method":      param.Method,
			"path":        param.Path,
			"user_agent":  param.Request.UserAgent(),
			"body_size":   param.BodySize,
			"request_id":  param.Request.Header.Get("X-Request-ID"),
		}

		if param.ErrorMessage != "" {
			logEntry["error"] = param.ErrorMessage
		}

		// Marshal to JSON
		jsonData, _ := json.Marshal(logEntry)
		return string(jsonData) + "\n"
	})
}

// Helper types and functions

type responseCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func captureRequestBody(c *gin.Context, maxSize int) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}

	// Read body
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, int64(maxSize)))
	if err != nil {
		return nil, err
	}

	// Restore body for further processing
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

func captureHeaders(headers http.Header, sensitiveFields []string) map[string]interface{} {
	captured := make(map[string]interface{})

	for key, values := range headers {
		// Skip sensitive headers
		if isSensitiveField(key, sensitiveFields) {
			captured[key] = "[REDACTED]"
			continue
		}

		if len(values) == 1 {
			captured[key] = values[0]
		} else {
			captured[key] = values
		}
	}

	return captured
}

func sanitizeBody(bodyBytes []byte, sensitiveFields []string) interface{} {
	if len(bodyBytes) == 0 {
		return nil
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
		// Sanitize JSON data
		sanitizeJSON(jsonData, sensitiveFields)
		return jsonData
	}

	// Return as string if not JSON
	bodyStr := string(bodyBytes)
	if len(bodyStr) > 1000 {
		return bodyStr[:1000] + "..."
	}

	return bodyStr
}

func sanitizeJSON(data map[string]interface{}, sensitiveFields []string) {
	for key, value := range data {
		if isSensitiveField(key, sensitiveFields) {
			data[key] = "[REDACTED]"
			continue
		}

		// Recursively sanitize nested objects
		if nestedMap, ok := value.(map[string]interface{}); ok {
			sanitizeJSON(nestedMap, sensitiveFields)
		}
	}
}

func isSensitiveField(field string, sensitiveFields []string) bool {
	field = strings.ToLower(field)

	// Default sensitive fields
	defaultSensitive := []string{
		"password", "token", "authorization", "cookie", "secret", "key",
		"auth", "credentials", "access_token", "refresh_token", "api_key",
		"x-api-key", "x-auth-token", "csrf", "session",
	}

	allSensitive := append(defaultSensitive, sensitiveFields...)

	for _, sensitive := range allSensitive {
		if strings.Contains(field, strings.ToLower(sensitive)) {
			return true
		}
	}

	return false
}

func shouldLogBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

func logRequest(db *gorm.DB, logEntry *APIRequestLog) {
	// Log to database if available
	if db != nil {
		if err := RecordAPIRequest(db, logEntry); err != nil {
			// In production, you might want to log this error
			fmt.Printf("Failed to record API request: %v\n", err)
		}
	}

	// Also log structured data (in production, send to logging service)
	logStructuredData(logEntry)
}

func logStructuredData(data interface{}) {
	// In production, send to logging service like ELK, Splunk, etc.
	// For now, just marshal to JSON for debugging
	if gin.Mode() == gin.DebugMode {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("API Request Log: %s\n", jsonData)
	}
}

func logPerformanceData(data map[string]interface{}) {
	// In production, send to monitoring service like Prometheus, DataDog, etc.
	if gin.Mode() == gin.DebugMode {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("Performance Log: %s\n", jsonData)
	}
}

func logErrorData(data map[string]interface{}) {
	// In production, send to error tracking service like Sentry, Bugsnag, etc.
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("Error Log: %s\n", jsonData)
}

// Configuration functions

func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		EnableRequestLogging:  true,
		EnableResponseLogging: false,
		EnablePerformanceLog:  true,
		EnableErrorLogging:    true,
		SlowThreshold:         500 * time.Millisecond,
		LogLevel:              "info",
		MaxBodySize:           1024, // 1KB
		SensitiveFields: []string{
			"password", "token", "secret", "key", "auth",
		},
	}
}

func ProductionLoggingConfig() *LoggingConfig {
	config := DefaultLoggingConfig()
	config.EnableResponseLogging = false
	config.MaxBodySize = 512
	return config
}

func DevelopmentLoggingConfig() *LoggingConfig {
	config := DefaultLoggingConfig()
	config.EnableResponseLogging = true
	config.MaxBodySize = 2048
	return config
}

// Analytics functions

// GetRequestAnalytics returns analytics data for API requests
func GetRequestAnalytics(db *gorm.DB, hours int) (*models.APIAnalyticsResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var totalRequests int64
	if err := db.Model(&models.APIRequest{}).Where("created_at >= ?", since).Count(&totalRequests).Error; err != nil {
		return nil, err
	}

	var uniqueIPs int64
	if err := db.Model(&models.APIRequest{}).Where("created_at >= ?", since).
		Distinct("ip_address").Count(&uniqueIPs).Error; err != nil {
		return nil, err
	}

	var avgResponse float64
	if err := db.Model(&models.APIRequest{}).Where("created_at >= ?", since).
		Select("AVG(response_time)").Scan(&avgResponse).Error; err != nil {
		return nil, err
	}

	var rateLimited int64
	if err := db.Model(&models.APIRequest{}).Where("created_at >= ? AND status_code = ?", since, 429).
		Count(&rateLimited).Error; err != nil {
		return nil, err
	}

	analytics := &models.APIAnalyticsResponse{
		TotalRequests:    int(totalRequests),
		UniqueIPs:        int(uniqueIPs),
		AverageResponseMs: avgResponse,
		RateLimitedCount: int(rateLimited),
		Period:           fmt.Sprintf("last_%dh", hours),
	}

	return analytics, nil
}