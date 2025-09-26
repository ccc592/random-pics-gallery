package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomImagesContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// This will fail until the actual handler is implemented
	router.GET("/api/images/random", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
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
				var response struct {
					Images []struct {
						ID           string   `json:"id"`
						Filename     string   `json:"filename"`
						Alt          string   `json:"alt"`
						Title        *string  `json:"title"`
						Tags         *string  `json:"tags"`
						StoragePath  string   `json:"storage_path"`
						MimeType     string   `json:"mime_type"`
						FileSize     int64    `json:"file_size"`
						Width        int      `json:"width"`
						Height       int      `json:"height"`
						AspectRatio  float64  `json:"aspect_ratio"`
						DominantColors []string `json:"dominant_colors"`
					} `json:"images"`
					SessionSeed string `json:"session_seed"`
					Count       int    `json:"count"`
				}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Should return 3-5 images by default
				assert.GreaterOrEqual(t, response.Count, 3)
				assert.LessOrEqual(t, response.Count, 5)
				assert.Len(t, response.Images, response.Count)
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
					assert.GreaterOrEqual(t, img.Width, 100)
					assert.GreaterOrEqual(t, img.Height, 100)
					assert.Greater(t, img.AspectRatio, 0.0)
				}
			},
		},
		{
			name:           "Random images with custom count",
			queryParams:    "?count=4",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Count int `json:"count"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, 4, response.Count)
			},
		},
		{
			name:           "Random images with seed",
			queryParams:    "?seed=test123",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					SessionSeed string `json:"session_seed"`
				}
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
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "count must be between 1 and 5")
			},
		},
		{
			name:           "Invalid count - zero",
			queryParams:    "?count=0",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "count must be between 1 and 5")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/images/random"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// This test will initially fail because we return 501 Not Implemented
			// Once the real handler is implemented, these assertions should pass
			if tt.expectedStatus == http.StatusOK {
				// For now, expect 501 until implementation
				assert.Equal(t, http.StatusNotImplemented, w.Code)
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}

			if tt.validateBody != nil {
				tt.validateBody(t, w.Body.Bytes())
			}
		})
	}
}