package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerWindow int           // Number of requests allowed per window
	WindowSize        time.Duration // Size of the rate limiting window
	CleanupInterval   time.Duration // How often to clean up expired entries
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	config      *RateLimitConfig
	buckets     map[string]*TokenBucket
	mutex       sync.RWMutex
	stopCleaner chan bool
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	Tokens        int       // Current number of tokens
	MaxTokens     int       // Maximum number of tokens
	RefillRate    int       // Tokens added per window
	LastRefill    time.Time // Last time tokens were added
	WindowStart   time.Time // Start of current window
	RequestCount  int       // Requests in current window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      config,
		buckets:     make(map[string]*TokenBucket),
		stopCleaner: make(chan bool),
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredBuckets()

	return rl
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 100,               // 100 requests per window
		WindowSize:        15 * time.Minute,  // 15 minute window
		CleanupInterval:   5 * time.Minute,   // Clean up every 5 minutes
	}
}

// AnonymousRateLimitConfig returns rate limiting for anonymous users
func AnonymousRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 60,                // 60 requests per window
		WindowSize:        15 * time.Minute,  // 15 minute window
		CleanupInterval:   5 * time.Minute,
	}
}

// AuthenticatedRateLimitConfig returns rate limiting for authenticated users
func AuthenticatedRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 300,               // 300 requests per window
		WindowSize:        15 * time.Minute,  // 15 minute window
		CleanupInterval:   5 * time.Minute,
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier
		clientID := getClientIdentifier(c)

		// Check rate limit
		allowed, resetTime := limiter.Allow(clientID)
		if !allowed {
			// Add rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerWindow))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
				"retry_after": resetTime.Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		bucket := limiter.getBucket(clientID)
		if bucket != nil {
			remaining := max(0, limiter.config.RequestsPerWindow - bucket.RequestCount)
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerWindow))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
		}

		c.Next()
	}
}

// Allow checks if a request should be allowed for the given client
func (rl *RateLimiter) Allow(clientID string) (bool, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	bucket := rl.getOrCreateBucket(clientID, now)

	// Check if we need to reset the window
	if now.Sub(bucket.WindowStart) >= rl.config.WindowSize {
		bucket.WindowStart = now
		bucket.RequestCount = 0
	}

	// Check if request is allowed
	if bucket.RequestCount >= rl.config.RequestsPerWindow {
		resetTime := bucket.WindowStart.Add(rl.config.WindowSize)
		return false, resetTime
	}

	// Allow request and increment counter
	bucket.RequestCount++
	resetTime := bucket.WindowStart.Add(rl.config.WindowSize)
	return true, resetTime
}

// getClientIdentifier gets a unique identifier for the client
func getClientIdentifier(c *gin.Context) string {
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

// getBucket gets an existing bucket for the client
func (rl *RateLimiter) getBucket(clientID string) *TokenBucket {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	return rl.buckets[clientID]
}

// getOrCreateBucket gets or creates a bucket for the client
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

// cleanupExpiredBuckets removes expired rate limit buckets
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

// cleanup removes expired buckets
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expireTime := rl.config.WindowSize * 2 // Keep buckets for 2 windows

	for clientID, bucket := range rl.buckets {
		if now.Sub(bucket.WindowStart) > expireTime {
			delete(rl.buckets, clientID)
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleaner)
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	totalBuckets := len(rl.buckets)
	activeBuckets := 0
	totalRequests := 0

	now := time.Now()
	for _, bucket := range rl.buckets {
		if now.Sub(bucket.WindowStart) < rl.config.WindowSize {
			activeBuckets++
		}
		totalRequests += bucket.RequestCount
	}

	return map[string]interface{}{
		"total_buckets":       totalBuckets,
		"active_buckets":      activeBuckets,
		"total_requests":      totalRequests,
		"requests_per_window": rl.config.RequestsPerWindow,
		"window_size":         rl.config.WindowSize.String(),
	}
}

// DifferentialRateLimitMiddleware applies different limits based on user type
func DifferentialRateLimitMiddleware(anonymousLimiter, authenticatedLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Choose limiter based on authentication status
		var limiter *RateLimiter
		if _, exists := c.Get("user"); exists {
			limiter = authenticatedLimiter
		} else {
			limiter = anonymousLimiter
		}

		// Apply rate limiting
		clientID := getClientIdentifier(c)
		allowed, resetTime := limiter.Allow(clientID)

		if !allowed {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerWindow))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
				"retry_after": resetTime.Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		bucket := limiter.getBucket(clientID)
		if bucket != nil {
			remaining := max(0, limiter.config.RequestsPerWindow - bucket.RequestCount)
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerWindow))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
		}

		c.Next()
	}
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}