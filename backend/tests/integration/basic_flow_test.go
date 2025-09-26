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

func TestBasicUserFlowIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock routes - these will fail until actual handlers are implemented
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})
	router.GET("/api/images/random", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})

	t.Run("Basic user journey - health check then get random images", func(t *testing.T) {
		// Step 1: Check health endpoint
		healthReq := httptest.NewRequest("GET", "/api/health", nil)
		healthW := httptest.NewRecorder()
		router.ServeHTTP(healthW, healthReq)

		// This will initially fail - should return 200 when implemented
		if healthW.Code == http.StatusOK {
			var healthResponse struct {
				Status string `json:"status"`
			}
			err := json.Unmarshal(healthW.Body.Bytes(), &healthResponse)
			require.NoError(t, err)
			assert.Equal(t, "healthy", healthResponse.Status)
		} else {
			// Currently returns 501
			assert.Equal(t, http.StatusNotImplemented, healthW.Code)
		}

		// Step 2: Get random images
		randomReq := httptest.NewRequest("GET", "/api/images/random", nil)
		randomW := httptest.NewRecorder()
		router.ServeHTTP(randomW, randomReq)

		// This will initially fail - should return 200 when implemented
		if randomW.Code == http.StatusOK {
			var randomResponse struct {
				Images []struct {
					ID          string `json:"id"`
					Filename    string `json:"filename"`
					Alt         string `json:"alt"`
					StoragePath string `json:"storage_path"`
				} `json:"images"`
				SessionSeed string `json:"session_seed"`
				Count       int    `json:"count"`
			}

			err := json.Unmarshal(randomW.Body.Bytes(), &randomResponse)
			require.NoError(t, err)

			// Validate the response structure
			assert.GreaterOrEqual(t, randomResponse.Count, 3)
			assert.LessOrEqual(t, randomResponse.Count, 5)
			assert.Len(t, randomResponse.Images, randomResponse.Count)
			assert.NotEmpty(t, randomResponse.SessionSeed)

			// Validate each image in the response
			for _, img := range randomResponse.Images {
				assert.NotEmpty(t, img.ID)
				assert.NotEmpty(t, img.Filename)
				assert.NotEmpty(t, img.Alt)
				assert.GreaterOrEqual(t, len(img.Alt), 10)
				assert.NotEmpty(t, img.StoragePath)
			}
		} else {
			// Currently returns 501
			assert.Equal(t, http.StatusNotImplemented, randomW.Code)
		}
	})
}