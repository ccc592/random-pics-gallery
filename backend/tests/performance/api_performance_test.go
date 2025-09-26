package performance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/randompic/api/internal/api/handlers"
	"github.com/randompic/api/internal/db/models"
	"github.com/randompic/api/internal/services"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.User{}, &models.Image{}, &models.APIRequest{})
	if err != nil {
		panic("failed to migrate test database")
	}

	return db
}

// seedTestImages creates test images in the database
func seedTestImages(db *gorm.DB, count int) {
	user := &models.User{
		ID:            "test-user-123",
		Email:         "test@example.com",
		Role:          "admin",
		OAuthProvider: "google",
		OAuthUserID:   "oauth123",
		IsActive:      true,
	}
	db.Create(user)

	for i := 0; i < count; i++ {
		image := &models.Image{
			ID:          fmt.Sprintf("img-%d", i),
			Filename:    fmt.Sprintf("image-%d.jpg", i),
			Alt:         fmt.Sprintf("Test image %d description", i),
			MimeType:    "image/jpeg",
			FileSize:    1024 * 1024, // 1MB
			Width:       800,
			Height:      600,
			Weight:      (i%10 + 1), // Weight from 1-10
			StoragePath: fmt.Sprintf("./storage/images/test-user-123/image-%d.jpg", i),
			UploadedBy:  "test-user-123",
			Status:      "active",
		}
		db.Create(image)
	}
}

// setupTestRouter creates a test Gin router with random images endpoint
func setupTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	randomizerService := randomizer.NewService()

	// Mock random images handler
	router.GET("/api/random-images", func(c *gin.Context) {
		count := 10 // Default count
		if countParam := c.Query("count"); countParam != "" {
			// Parse count parameter if provided
		}

		sessionSeed := c.Query("session_seed")
		if sessionSeed == "" {
			sessionSeed = randomizerService.GenerateSessionSeed()
		}

		// Query active images from database
		var images []models.Image
		if err := db.Where("status = ?", "active").Find(&images).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		// Convert to weighted items
		weightedItems := make([]randomizer.WeightedItem, len(images))
		for i, img := range images {
			weightedItems[i] = randomizer.WeightedItem{
				ID:     img.ID,
				Weight: img.Weight,
				Data:   img,
			}
		}

		// Select random images
		selectedItems, err := randomizerService.SelectWeightedRandom(weightedItems, count, sessionSeed)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Randomization error"})
			return
		}

		// Convert to response format
		imageResponses := make([]models.ImageResponse, len(selectedItems))
		for i, item := range selectedItems {
			img := item.Data.(models.Image)
			imageResponses[i] = img.ToResponse()
		}

		response := models.RandomImagesResponse{
			Images:      imageResponses,
			SessionSeed: sessionSeed,
			Count:       len(imageResponses),
		}

		c.JSON(http.StatusOK, response)
	})

	return router
}

// PerformanceMetrics holds performance test results
type PerformanceMetrics struct {
	RequestCount      int
	TotalDuration     time.Duration
	AverageLatency    time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	SuccessfulRequests int
	FailedRequests     int
	RequestsPerSecond  float64
}

// measureLatency measures request latency and collects metrics
func measureLatency(router *gin.Engine, requests int, concurrent int) *PerformanceMetrics {
	latencies := make([]time.Duration, 0, requests)
	var mu sync.Mutex
	var wg sync.WaitGroup

	startTime := time.Now()
	semaphore := make(chan struct{}, concurrent) // Limit concurrent requests
	successCount := 0
	failureCount := 0

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			reqStart := time.Now()

			// Create test request
			req, _ := http.NewRequest("GET", "/api/random-images?count=5", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			latency := time.Since(reqStart)

			mu.Lock()
			latencies = append(latencies, latency)
			if w.Code == 200 {
				successCount++
			} else {
				failureCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Calculate percentiles
	if len(latencies) == 0 {
		return &PerformanceMetrics{}
	}

	// Sort latencies for percentile calculation
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	var totalLatency time.Duration
	for _, lat := range latencies {
		totalLatency += lat
	}

	avgLatency := totalLatency / time.Duration(len(latencies))
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)

	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	if p99Index >= len(latencies) {
		p99Index = len(latencies) - 1
	}

	return &PerformanceMetrics{
		RequestCount:       requests,
		TotalDuration:      totalDuration,
		AverageLatency:     avgLatency,
		P95Latency:         latencies[p95Index],
		P99Latency:         latencies[p99Index],
		MinLatency:         latencies[0],
		MaxLatency:         latencies[len(latencies)-1],
		SuccessfulRequests: successCount,
		FailedRequests:     failureCount,
		RequestsPerSecond:  float64(requests) / totalDuration.Seconds(),
	}
}

func TestRandomImageAPIPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	db := setupTestDB()
	seedTestImages(db, 1000) // Seed with 1000 test images
	router := setupTestRouter(db)

	t.Run("P95LatencyUnder150ms", func(t *testing.T) {
		// Test with moderate load
		metrics := measureLatency(router, 100, 10)

		t.Logf("Performance Metrics:")
		t.Logf("  Total Requests: %d", metrics.RequestCount)
		t.Logf("  Successful: %d, Failed: %d", metrics.SuccessfulRequests, metrics.FailedRequests)
		t.Logf("  Average Latency: %v", metrics.AverageLatency)
		t.Logf("  P95 Latency: %v", metrics.P95Latency)
		t.Logf("  P99 Latency: %v", metrics.P99Latency)
		t.Logf("  Min Latency: %v", metrics.MinLatency)
		t.Logf("  Max Latency: %v", metrics.MaxLatency)
		t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)

		// Verify performance targets
		assert.Less(t, metrics.P95Latency, 150*time.Millisecond, "P95 latency should be under 150ms")
		assert.Equal(t, 0, metrics.FailedRequests, "Should have no failed requests")
		assert.Greater(t, metrics.RequestsPerSecond, 10.0, "Should handle at least 10 requests per second")
	})

	t.Run("ColdStartLatencyUnder350ms", func(t *testing.T) {
		// Test cold start performance
		req, _ := http.NewRequest("GET", "/api/random-images?count=10", nil)
		w := httptest.NewRecorder()

		startTime := time.Now()
		router.ServeHTTP(w, req)
		latency := time.Since(startTime)

		t.Logf("Cold start latency: %v", latency)

		assert.Equal(t, 200, w.Code)
		assert.Less(t, latency, 350*time.Millisecond, "Cold start latency should be under 350ms")

		// Verify response structure
		var response models.RandomImagesResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Images, 10)
		assert.NotEmpty(t, response.SessionSeed)
	})

	t.Run("DatabaseQueryPerformanceUnder50ms", func(t *testing.T) {
		// Test direct database query performance
		var images []models.Image

		startTime := time.Now()
		err := db.Where("status = ?", "active").Limit(100).Find(&images).Error
		queryTime := time.Since(startTime)

		t.Logf("Database query time for 100 images: %v", queryTime)

		require.NoError(t, err)
		assert.Less(t, queryTime, 50*time.Millisecond, "Database query should be under 50ms")
		assert.Len(t, images, 100)
	})

	t.Run("ConcurrentRequestHandling", func(t *testing.T) {
		// Test handling 100+ simultaneous users
		metrics := measureLatency(router, 200, 50) // 200 requests, 50 concurrent

		t.Logf("Concurrent Request Metrics:")
		t.Logf("  Total Requests: %d (50 concurrent)", metrics.RequestCount)
		t.Logf("  Successful: %d, Failed: %d", metrics.SuccessfulRequests, metrics.FailedRequests)
		t.Logf("  Average Latency: %v", metrics.AverageLatency)
		t.Logf("  P95 Latency: %v", metrics.P95Latency)
		t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)

		assert.Equal(t, 0, metrics.FailedRequests, "Should handle concurrent requests without failures")
		assert.Less(t, metrics.P95Latency, 500*time.Millisecond, "P95 latency under load should be reasonable")
		assert.Greater(t, metrics.RequestsPerSecond, 20.0, "Should maintain good throughput under load")
	})

	t.Run("MemoryUsageStability", func(t *testing.T) {
		// Test multiple requests to ensure no memory leaks
		for i := 0; i < 50; i++ {
			req, _ := http.NewRequest("GET", "/api/random-images?count=20", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code, "Request %d should succeed", i)
		}

		// This test mainly ensures the handler can be called multiple times
		// without crashing or obvious memory issues
	})

	t.Run("DifferentCountParameters", func(t *testing.T) {
		counts := []int{1, 5, 10, 25, 50}

		for _, count := range counts {
			t.Run(fmt.Sprintf("Count_%d", count), func(t *testing.T) {
				req, _ := http.NewRequest("GET", fmt.Sprintf("/api/random-images?count=%d", count), nil)
				w := httptest.NewRecorder()

				startTime := time.Now()
				router.ServeHTTP(w, req)
				latency := time.Since(startTime)

				t.Logf("Latency for count %d: %v", count, latency)

				assert.Equal(t, 200, w.Code)
				assert.Less(t, latency, 200*time.Millisecond, "Should be fast regardless of count")

				var response models.RandomImagesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				expectedCount := count
				if count > 1000 { // More than available images
					expectedCount = 1000
				}
				assert.LessOrEqual(t, len(response.Images), expectedCount)
			})
		}
	})
}

func TestCachedResponsePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	db := setupTestDB()
	seedTestImages(db, 100)
	router := setupTestRouter(db)

	t.Run("SameSeedConsistentPerformance", func(t *testing.T) {
		seed := "consistent_test_seed"

		// First request
		req1, _ := http.NewRequest("GET", fmt.Sprintf("/api/random-images?count=10&session_seed=%s", seed), nil)
		w1 := httptest.NewRecorder()

		start1 := time.Now()
		router.ServeHTTP(w1, req1)
		latency1 := time.Since(start1)

		// Second request with same seed
		req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/random-images?count=10&session_seed=%s", seed), nil)
		w2 := httptest.NewRecorder()

		start2 := time.Now()
		router.ServeHTTP(w2, req2)
		latency2 := time.Since(start2)

		t.Logf("First request latency: %v", latency1)
		t.Logf("Second request latency: %v", latency2)

		assert.Equal(t, 200, w1.Code)
		assert.Equal(t, 200, w2.Code)

		// Both requests should be fast (no significant performance difference)
		assert.Less(t, latency1, 150*time.Millisecond)
		assert.Less(t, latency2, 150*time.Millisecond)

		// Results should be identical
		var response1, response2 models.RandomImagesResponse
		json.Unmarshal(w1.Body.Bytes(), &response1)
		json.Unmarshal(w2.Body.Bytes(), &response2)

		assert.Equal(t, response1.SessionSeed, response2.SessionSeed)
		assert.Len(t, response1.Images, len(response2.Images))

		// Images should be in the same order
		for i, img1 := range response1.Images {
			assert.Equal(t, img1.ID, response2.Images[i].ID)
		}
	})
}

// Benchmark tests
func BenchmarkRandomImagesAPI(b *testing.B) {
	db := setupTestDB()
	seedTestImages(db, 1000)
	router := setupTestRouter(db)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/random-images?count=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Request failed with status %d", w.Code)
		}
	}
}

func BenchmarkRandomImagesAPIWithSeed(b *testing.B) {
	db := setupTestDB()
	seedTestImages(db, 1000)
	router := setupTestRouter(db)

	seed := "benchmark_seed"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/random-images?count=10&session_seed=%s", seed), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			b.Fatalf("Request failed with status %d", w.Code)
		}
	}
}

func BenchmarkDatabaseQuery(b *testing.B) {
	db := setupTestDB()
	seedTestImages(db, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var images []models.Image
		err := db.Where("status = ?", "active").Find(&images).Error
		if err != nil {
			b.Fatalf("Database query failed: %v", err)
		}
	}
}

func BenchmarkRandomizerService(b *testing.B) {
	service := randomizer.NewService()

	// Create test items
	items := make([]randomizer.WeightedItem, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = randomizer.WeightedItem{
			ID:     fmt.Sprintf("item_%d", i),
			Weight: (i % 10) + 1,
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.SelectWeightedRandom(items, 10, fmt.Sprintf("seed_%d", i))
		if err != nil {
			b.Fatalf("Randomizer service failed: %v", err)
		}
	}
}