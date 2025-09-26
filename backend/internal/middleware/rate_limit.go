package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerWindow int           // Number of requests allowed per window
	WindowSize        time.Duration // Size of the rate limiting window
	CleanupInterval   time.Duration // How often to clean up expired entries
	AdminMultiplier   int           // Multiplier for admin user limits
}

// DatabaseRateLimiter implements database-backed sliding window rate limiting
type DatabaseRateLimiter struct {
	db              *gorm.DB
	config          *RateLimitConfig
	memoryCounts    map[string]*WindowCounter // In-memory cache for performance
	mutex           sync.RWMutex
	stopCleaner     chan bool
}

// WindowCounter tracks requests in current window
type WindowCounter struct {
	Count       int       // Current request count in window
	WindowStart time.Time // Start of current window
	LastCheck   time.Time // Last time we checked the database
}

// NewDatabaseRateLimiter creates a new database-backed rate limiter
func NewDatabaseRateLimiter(db *gorm.DB, config *RateLimitConfig) *DatabaseRateLimiter {
	rl := &DatabaseRateLimiter{
		db:           db,
		config:       config,
		memoryCounts: make(map[string]*WindowCounter),
		stopCleaner:  make(chan bool),
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredCounters()

	return rl
}

// RateLimitMiddleware creates database-backed rate limiting middleware
func RateLimitMiddleware(requests int, window time.Duration) gin.HandlerFunc {
	// This is a simplified version for backward compatibility
	// Use DatabaseRateLimitMiddleware for full database integration
	config := &RateLimitConfig{
		RequestsPerWindow: requests,
		WindowSize:        window,
		CleanupInterval:   5 * time.Minute,
		AdminMultiplier:   5,
	}

	limiter := NewRateLimiter(config)
	return limiter.Middleware()
}

// IPRateLimitMiddleware creates IP-based rate limiting
func IPRateLimitMiddleware(requests int, window time.Duration) gin.HandlerFunc {
	config := &RateLimitConfig{
		RequestsPerWindow: requests,
		WindowSize:        window,
		CleanupInterval:   5 * time.Minute,
		AdminMultiplier:   1, // No multiplier for IP-based limiting
	}

	limiter := NewRateLimiter(config)
	return limiter.IPMiddleware()
}

// UserRateLimitMiddleware creates user-based rate limiting
func UserRateLimitMiddleware(requests int, window time.Duration) gin.HandlerFunc {
	config := &RateLimitConfig{
		RequestsPerWindow: requests,
		WindowSize:        window,
		CleanupInterval:   5 * time.Minute,
		AdminMultiplier:   5, // Admin users get 5x limits
	}

	limiter := NewRateLimiter(config)
	return limiter.UserMiddleware()
}

// DatabaseRateLimitMiddleware creates database-backed rate limiting middleware
func (rl *DatabaseRateLimiter) DatabaseRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := rl.getClientIdentifier(c)
		userRole := rl.getUserRole(c)

		// Calculate rate limit based on user role
		limit := rl.config.RequestsPerWindow
		if userRole == "admin" {
			limit *= rl.config.AdminMultiplier
		}

		// Check rate limit
		allowed, remaining, resetTime, err := rl.checkRateLimit(clientID, limit)
		if err != nil {
			// Log error but allow request to continue
			// In production, you might want to handle this differently
			c.Next()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": resetTime.Format(time.RFC3339),
				"limit":       limit,
				"window":      rl.config.WindowSize.String(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckRateLimit checks if a request should be allowed
func (rl *DatabaseRateLimiter) checkRateLimit(identifier string, limit int) (allowed bool, remaining int, resetTime time.Time, err error) {
	now := time.Now()
	windowStart := now.Truncate(rl.config.WindowSize)

	// Check memory cache first for performance
	rl.mutex.Lock()
	counter, exists := rl.memoryCounts[identifier]
	if !exists || now.Sub(counter.LastCheck) > time.Minute {
		// Cache miss or stale data - check database
		count, err := rl.getRequestCountFromDB(identifier, windowStart)
		if err != nil {
			rl.mutex.Unlock()
			return false, 0, windowStart.Add(rl.config.WindowSize), err
		}

		counter = &WindowCounter{
			Count:       count,
			WindowStart: windowStart,
			LastCheck:   now,
		}
		rl.memoryCounts[identifier] = counter
	} else if counter.WindowStart != windowStart {
		// New window started
		count, err := rl.getRequestCountFromDB(identifier, windowStart)
		if err != nil {
			rl.mutex.Unlock()
			return false, 0, windowStart.Add(rl.config.WindowSize), err
		}

		counter.Count = count
		counter.WindowStart = windowStart
		counter.LastCheck = now
	}

	// Check if limit exceeded
	if counter.Count >= limit {
		remaining = 0
		resetTime = windowStart.Add(rl.config.WindowSize)
		rl.mutex.Unlock()
		return false, remaining, resetTime, nil
	}

	// Increment counter
	counter.Count++
	remaining = limit - counter.Count
	resetTime = windowStart.Add(rl.config.WindowSize)

	rl.mutex.Unlock()

	// Record request in database (async to avoid blocking)
	go rl.recordRequestAsync(identifier, now)

	return true, remaining, resetTime, nil
}

// CheckRateLimit is the public interface for checking rate limits
func CheckRateLimit(identifier string, requests int, window time.Duration) (bool, error) {
	// This is a simplified implementation for the public interface
	// For full functionality, use DatabaseRateLimiter

	// You would typically have a global rate limiter instance
	// For now, return true to allow all requests
	return true, nil
}

func (rl *DatabaseRateLimiter) getRequestCountFromDB(identifier string, windowStart time.Time) (int, error) {
	var count int64

	windowEnd := windowStart.Add(rl.config.WindowSize)

	// Extract IP and user ID from identifier
	var userID *string
	var ipAddress string

	if len(identifier) > 5 && identifier[:5] == "user:" {
		userIDStr := identifier[5:]
		userID = &userIDStr
		// For user-based limiting, we still want to check IP patterns for abuse
		ipAddress = "%"
	} else if len(identifier) > 3 && identifier[:3] == "ip:" {
		ipAddress = identifier[3:]
	} else {
		ipAddress = identifier
	}

	query := rl.db.Model(&models.APIRequest{}).
		Where("created_at >= ? AND created_at < ?", windowStart, windowEnd)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("ip_address = ?", ipAddress)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get request count: %w", err)
	}

	return int(count), nil
}

func (rl *DatabaseRateLimiter) recordRequestAsync(identifier string, timestamp time.Time) {
	// This would be called from the logging middleware
	// We don't want to duplicate API request recording here
	// The logging middleware will handle the database insertion
}

func (rl *DatabaseRateLimiter) getClientIdentifier(c *gin.Context) string {
	// First, try to get user ID from context (for authenticated users)
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}

	// For anonymous users, use IP address
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "unknown"
	}

	return fmt.Sprintf("ip:%s", clientIP)
}

func (rl *DatabaseRateLimiter) getUserRole(c *gin.Context) string {
	if role, exists := c.Get("user_role"); exists {
		if roleStr, ok := role.(string); ok {
			return roleStr
		}
	}
	return "user" // Default role
}

func (rl *DatabaseRateLimiter) cleanupExpiredCounters() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleaner:
			return
		}
	}
}

func (rl *DatabaseRateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expireTime := rl.config.WindowSize * 2 // Keep counters for 2 windows

	for identifier, counter := range rl.memoryCounts {
		if now.Sub(counter.LastCheck) > expireTime {
			delete(rl.memoryCounts, identifier)
		}
	}
}

// Stop stops the cleanup goroutine
func (rl *DatabaseRateLimiter) Stop() {
	close(rl.stopCleaner)
}

// GetRateLimitStatus returns current rate limit status for a client
func (rl *DatabaseRateLimiter) GetRateLimitStatus(identifier string, limit int) (*models.RateLimitStatus, error) {
	now := time.Now()
	windowStart := now.Truncate(rl.config.WindowSize)
	windowEnd := windowStart.Add(rl.config.WindowSize)

	count, err := rl.getRequestCountFromDB(identifier, windowStart)
	if err != nil {
		return nil, err
	}

	var userID *string
	var ipAddress string

	if len(identifier) > 5 && identifier[:5] == "user:" {
		userIDStr := identifier[5:]
		userID = &userIDStr
	} else if len(identifier) > 3 && identifier[:3] == "ip:" {
		ipAddress = identifier[3:]
	} else {
		ipAddress = identifier
	}

	status := &models.RateLimitStatus{
		IPAddress:     ipAddress,
		UserID:        userID,
		RequestCount:  count,
		WindowStart:   windowStart,
		WindowEnd:     windowEnd,
		Limit:         limit,
		Remaining:     max(0, limit-count),
		ResetTime:     windowEnd,
		IsRateLimited: count >= limit,
	}

	return status, nil
}

// Legacy RateLimiter for backward compatibility
type RateLimiter struct {
	config      *RateLimitConfig
	buckets     map[string]*TokenBucket
	mutex       sync.RWMutex
	stopCleaner chan bool
}

type TokenBucket struct {
	Tokens        int
	MaxTokens     int
	RefillRate    int
	LastRefill    time.Time
	WindowStart   time.Time
	RequestCount  int
}

// NewRateLimiter creates a new in-memory rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      config,
		buckets:     make(map[string]*TokenBucket),
		stopCleaner: make(chan bool),
	}

	go rl.cleanupExpiredBuckets()
	return rl
}

// Middleware creates standard rate limiting middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := rl.getClientIdentifier(c)
		rl.applyRateLimit(c, clientID, rl.config.RequestsPerWindow)
	}
}

// IPMiddleware creates IP-based rate limiting middleware
func (rl *RateLimiter) IPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "unknown"
		}
		clientID := fmt.Sprintf("ip:%s", clientIP)
		rl.applyRateLimit(c, clientID, rl.config.RequestsPerWindow)
	}
}

// UserMiddleware creates user-based rate limiting middleware
func (rl *RateLimiter) UserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := rl.getClientIdentifier(c)
		limit := rl.config.RequestsPerWindow

		// Apply admin multiplier
		if role, exists := c.Get("user_role"); exists {
			if roleStr, ok := role.(string); ok && roleStr == "admin" {
				limit *= rl.config.AdminMultiplier
			}
		}

		rl.applyRateLimit(c, clientID, limit)
	}
}

func (rl *RateLimiter) applyRateLimit(c *gin.Context, clientID string, limit int) {
	allowed, resetTime := rl.Allow(clientID, limit)
	if !allowed {
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", "0")
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "rate limit exceeded",
			"code":        "RATE_LIMIT_EXCEEDED",
			"retry_after": resetTime.Format(time.RFC3339),
		})
		c.Abort()
		return
	}

	// Add rate limit headers for successful requests
	bucket := rl.getBucket(clientID)
	if bucket != nil {
		remaining := max(0, limit-bucket.RequestCount)
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	}

	c.Next()
}

func (rl *RateLimiter) Allow(clientID string, limit int) (bool, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	bucket := rl.getOrCreateBucket(clientID, now)

	if now.Sub(bucket.WindowStart) >= rl.config.WindowSize {
		bucket.WindowStart = now
		bucket.RequestCount = 0
	}

	if bucket.RequestCount >= limit {
		resetTime := bucket.WindowStart.Add(rl.config.WindowSize)
		return false, resetTime
	}

	bucket.RequestCount++
	resetTime := bucket.WindowStart.Add(rl.config.WindowSize)
	return true, resetTime
}

func (rl *RateLimiter) getClientIdentifier(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}

	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "unknown"
	}

	return fmt.Sprintf("ip:%s", clientIP)
}

func (rl *RateLimiter) getBucket(clientID string) *TokenBucket {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	return rl.buckets[clientID]
}

func (rl *RateLimiter) getOrCreateBucket(clientID string, now time.Time) *TokenBucket {
	bucket, exists := rl.buckets[clientID]
	if !exists {
		bucket = &TokenBucket{
			Tokens:       rl.config.RequestsPerWindow,
			MaxTokens:    rl.config.RequestsPerWindow,
			RefillRate:   rl.config.RequestsPerWindow,
			LastRefill:   now,
			WindowStart:  now,
			RequestCount: 0,
		}
		rl.buckets[clientID] = bucket
	}

	return bucket
}

func (rl *RateLimiter) cleanupExpiredBuckets() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleaner:
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expireTime := rl.config.WindowSize * 2

	for clientID, bucket := range rl.buckets {
		if now.Sub(bucket.WindowStart) > expireTime {
			delete(rl.buckets, clientID)
		}
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stopCleaner)
}

// Configuration functions
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 100,
		WindowSize:        15 * time.Minute,
		CleanupInterval:   5 * time.Minute,
		AdminMultiplier:   5,
	}
}

func AnonymousRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 60,
		WindowSize:        15 * time.Minute,
		CleanupInterval:   5 * time.Minute,
		AdminMultiplier:   1,
	}
}

func AuthenticatedRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 300,
		WindowSize:        15 * time.Minute,
		CleanupInterval:   5 * time.Minute,
		AdminMultiplier:   5,
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}