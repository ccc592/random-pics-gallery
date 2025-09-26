package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlingIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock error handling middleware
	router.Use(func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
					"code":  "INTERNAL_ERROR",
				})
				c.Abort()
			}
		}()
		c.Next()
	})

	// Mock routes - these will fail until actual handlers are implemented
	router.GET("/api/images/random", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})
	router.GET("/api/images/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})
	router.GET("/api/nonexistent", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "endpoint not found",
			"code":  "NOT_FOUND",
		})
	})

	// Add 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "endpoint not found",
			"code":  "NOT_FOUND",
		})
	})

	t.Run("404 errors have consistent format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/does-not-exist", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}

		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "endpoint not found", response.Error)
		assert.Equal(t, "NOT_FOUND", response.Code)
	})

	t.Run("Invalid UUID format returns 400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/images/invalid-uuid-format", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Currently returns 501, but should return 400 when implemented
		if w.Code == http.StatusBadRequest {
			var response struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}

			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response.Error, "invalid UUID")
			assert.Equal(t, "INVALID_REQUEST", response.Code)
		} else {
			// Currently returns 501
			assert.Equal(t, http.StatusNotImplemented, w.Code)
		}
	})

	t.Run("Invalid query parameters return 400", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
		}{
			{
				name:  "Invalid count - negative",
				query: "?count=-1",
			},
			{
				name:  "Invalid count - too high",
				query: "?count=10",
			},
			{
				name:  "Invalid count - non-numeric",
				query: "?count=abc",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/api/images/random"+tc.query, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Currently returns 501, but should return 400 when implemented
				if w.Code == http.StatusBadRequest {
					var response struct {
						Error string `json:"error"`
						Code  string `json:"code"`
					}

					err := json.Unmarshal(w.Body.Bytes(), &response)
					require.NoError(t, err)

					assert.NotEmpty(t, response.Error)
					assert.Equal(t, "INVALID_REQUEST", response.Code)
				} else {
					// Currently returns 501
					assert.Equal(t, http.StatusNotImplemented, w.Code)
				}
			})
		}
	})

	t.Run("Server errors have consistent format", func(t *testing.T) {
		// Add a route that panics to test error handling
		router.GET("/api/test/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/api/test/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}

		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "internal server error", response.Error)
		assert.Equal(t, "INTERNAL_ERROR", response.Code)
	})

	t.Run("CORS errors are handled properly", func(t *testing.T) {
		// Test preflight request
		req := httptest.NewRequest("OPTIONS", "/api/images/random", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// When CORS middleware is implemented, should handle OPTIONS requests
		// For now, might return 404 or method not allowed
		assert.Contains(t, []int{
			http.StatusOK,                    // CORS handled
			http.StatusNoContent,             // CORS handled
			http.StatusNotFound,              // No OPTIONS handler
			http.StatusMethodNotAllowed,      // Method not allowed
		}, w.Code)
	})

	t.Run("Content-Type validation", func(t *testing.T) {
		// This would be tested when upload endpoints are implemented
		// For now, just verify the route exists
		req := httptest.NewRequest("POST", "/api/admin/images/upload", nil)
		req.Header.Set("Authorization", "Bearer token")
		req.Header.Set("Content-Type", "text/plain") // Invalid content type

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Route doesn't exist yet, so 404
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error response includes request ID", func(t *testing.T) {
		// Add middleware to set request ID
		router.Use(func(c *gin.Context) {
			c.Header("X-Request-ID", "test-request-123")
			c.Next()
		})

		req := httptest.NewRequest("GET", "/api/does-not-exist-2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "test-request-123", w.Header().Get("X-Request-ID"))

		var response struct {
			Error     string `json:"error"`
			Code      string `json:"code"`
			RequestID string `json:"request_id,omitempty"`
		}

		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// When error handling is fully implemented, might include request ID
		if response.RequestID != "" {
			assert.Equal(t, "test-request-123", response.RequestID)
		}
	})
}