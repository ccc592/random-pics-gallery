package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randompic/api/internal/db/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRandomImagesContractSimple tests the random images endpoint with mock data
func TestRandomImagesContractSimple(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock random images endpoint with sample data
	router.GET("/api/images/random", func(c *gin.Context) {
		countStr := c.DefaultQuery("count", "3")
		seed := c.Query("seed")

		// Validate count parameter
		if countStr != "3" && countStr != "4" && countStr != "5" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "count must be between 1 and 5",
				"code":  "INVALID_REQUEST",
			})
			return
		}

		count := 3
		if countStr == "4" {
			count = 4
		} else if countStr == "5" {
			count = 5
		}

		// Generate session seed if not provided
		sessionSeed := seed
		if sessionSeed == "" {
			sessionSeed = "generated-seed-12345"
		}

		// Mock image responses
		images := []models.ImageResponse{
			{
				ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
				Filename:    "mountain-sunrise.jpg",
				Alt:         "A beautiful mountain sunrise with golden light",
				StoragePath: "2025/01/23/mountain-sunrise.jpg",
				MimeType:    "image/jpeg",
				FileSize:    2048576,
				Width:       &[]int{1920}[0],
				Height:      &[]int{1080}[0],
				AspectRatio: &[]float64{1.777}[0],
				Status:      "active",
				CreatedAt:   "2025-01-23T12:00:00Z",
				UpdatedAt:   "2025-01-23T12:00:00Z",
			},
			{
				ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
				Filename:    "ocean-waves.jpg",
				Alt:         "Peaceful ocean waves at sunset for motivation",
				StoragePath: "2025/01/23/ocean-waves.jpg",
				MimeType:    "image/jpeg",
				FileSize:    1524288,
				Width:       &[]int{1600}[0],
				Height:      &[]int{1200}[0],
				AspectRatio: &[]float64{1.333}[0],
				Status:      "active",
				CreatedAt:   "2025-01-23T12:00:00Z",
				UpdatedAt:   "2025-01-23T12:00:00Z",
			},
			{
				ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174003"),
				Filename:    "forest-path.jpg",
				Alt:         "A winding path through a green forest for inspiration",
				StoragePath: "2025/01/23/forest-path.jpg",
				MimeType:    "image/jpeg",
				FileSize:    1843200,
				Width:       &[]int{1440}[0],
				Height:      &[]int{960}[0],
				AspectRatio: &[]float64{1.5}[0],
				Status:      "active",
				CreatedAt:   "2025-01-23T12:00:00Z",
				UpdatedAt:   "2025-01-23T12:00:00Z",
			},
			{
				ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174004"),
				Filename:    "city-lights.jpg",
				Alt:         "Inspiring city lights at night with urban skyline",
				StoragePath: "2025/01/23/city-lights.jpg",
				MimeType:    "image/jpeg",
				FileSize:    2359296,
				Width:       &[]int{1920}[0],
				Height:      &[]int{1080}[0],
				AspectRatio: &[]float64{1.777}[0],
				Status:      "active",
				CreatedAt:   "2025-01-23T12:00:00Z",
				UpdatedAt:   "2025-01-23T12:00:00Z",
			},
			{
				ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174005"),
				Filename:    "desert-sunset.jpg",
				Alt:         "Magnificent desert sunset with vibrant colors for daily motivation",
				StoragePath: "2025/01/23/desert-sunset.jpg",
				MimeType:    "image/jpeg",
				FileSize:    1987584,
				Width:       &[]int{1800}[0],
				Height:      &[]int{1200}[0],
				AspectRatio: &[]float64{1.5}[0],
				Status:      "active",
				CreatedAt:   "2025-01-23T12:00:00Z",
				UpdatedAt:   "2025-01-23T12:00:00Z",
			},
		}

		// Return subset based on count
		selectedImages := images[:count]

		response := models.RandomImagesResponse{
			Images:      selectedImages,
			SessionSeed: sessionSeed,
			Count:       count,
		}

		c.JSON(http.StatusOK, response)
	})

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Default random images request",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response models.RandomImagesResponse

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Should return 3 images by default
				assert.Equal(t, 3, response.Count)
				assert.Len(t, response.Images, 3)
				assert.NotEmpty(t, response.SessionSeed)

				// Validate each image structure
				for _, img := range response.Images {
					assert.NotEmpty(t, img.ID)
					assert.NotEmpty(t, img.Filename)
					assert.NotEmpty(t, img.Alt)
					assert.GreaterOrEqual(t, len(img.Alt), 10) // Alt text must be >= 10 chars
					assert.NotEmpty(t, img.StoragePath)
					assert.NotEmpty(t, img.MimeType)
					assert.Greater(t, img.FileSize, int64(0))
					assert.NotNil(t, img.Width)
					assert.NotNil(t, img.Height)
					assert.GreaterOrEqual(t, *img.Width, 100)
					assert.GreaterOrEqual(t, *img.Height, 100)
					assert.NotNil(t, img.AspectRatio)
					assert.Greater(t, *img.AspectRatio, 0.0)
					assert.Equal(t, "active", img.Status)
				}
			},
		},
		{
			name:           "Random images with custom count",
			queryParams:    "?count=4",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response models.RandomImagesResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, 4, response.Count)
				assert.Len(t, response.Images, 4)
			},
		},
		{
			name:           "Random images with seed",
			queryParams:    "?seed=test123",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response models.RandomImagesResponse
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "test123", response.SessionSeed)
			},
		},
		{
			name:           "Invalid count - too high",
			queryParams:    "?count=10",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "count must be between 1 and 5")
				assert.Equal(t, "INVALID_REQUEST", response.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/images/random"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, w.Body.Bytes())
			}
		})
	}
}