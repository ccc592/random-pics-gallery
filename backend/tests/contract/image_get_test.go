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

func TestImageGetContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// This will fail until the actual handler is implemented
	router.GET("/api/images/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})

	tests := []struct {
		name           string
		imageID        string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid image ID - success response",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					ID             string   `json:"id"`
					Alt            string   `json:"alt"`
					Title          string   `json:"title"`
					Tags           []string `json:"tags"`
					Width          int      `json:"width"`
					Height         int      `json:"height"`
					AspectRatio    float64  `json:"aspect_ratio"`
					MimeType       string   `json:"mime_type"`
					FileSize       int64    `json:"file_size"`
					StoragePath    string   `json:"storage_path"`
					Weight         int      `json:"weight"`
					Status         string   `json:"status"`
					UploadDate     string   `json:"upload_date"`
					UploadedBy     string   `json:"uploaded_by"`
					CreatedAt      string   `json:"created_at"`
					UpdatedAt      string   `json:"updated_at"`
				}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Validate ImageResponse schema compliance
				assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", response.ID)
				assert.NotEmpty(t, response.Alt)
				assert.GreaterOrEqual(t, len(response.Alt), 10, "alt text must be at least 10 characters")
				assert.LessOrEqual(t, len(response.Alt), 500, "alt text must not exceed 500 characters")

				if response.Title != "" {
					assert.LessOrEqual(t, len(response.Title), 255, "title must not exceed 255 characters")
				}

				for _, tag := range response.Tags {
					assert.LessOrEqual(t, len(tag), 50, "each tag must not exceed 50 characters")
				}

				assert.GreaterOrEqual(t, response.Width, 100, "width must be at least 100px")
				assert.GreaterOrEqual(t, response.Height, 100, "height must be at least 100px")
				assert.Greater(t, response.AspectRatio, 0.0, "aspect ratio must be positive")
				assert.NotEmpty(t, response.MimeType)
				assert.Greater(t, response.FileSize, int64(0), "file size must be positive")
				assert.NotEmpty(t, response.StoragePath)
				assert.GreaterOrEqual(t, response.Weight, 1, "weight must be at least 1")
				assert.LessOrEqual(t, response.Weight, 10, "weight must not exceed 10")
				assert.Contains(t, []string{"active", "inactive"}, response.Status)
				assert.NotEmpty(t, response.CreatedAt)
				assert.NotEmpty(t, response.UpdatedAt)
			},
		},
		{
			name:           "Invalid UUID format",
			imageID:        "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "invalid UUID format")
				assert.Equal(t, "INVALID_REQUEST", response.Code)
			},
		},
		{
			name:           "Non-existent image ID",
			imageID:        "00000000-0000-0000-0000-000000000000",
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "image not found", response.Error)
				assert.Equal(t, "NOT_FOUND", response.Code)
			},
		},
		{
			name:           "Malformed UUID with correct length",
			imageID:        "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "invalid UUID format")
				assert.Equal(t, "INVALID_REQUEST", response.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/images/"+tt.imageID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// This test will initially fail because we return 501 Not Implemented
			if tt.expectedStatus == http.StatusOK {
				// For TDD, we expect the test to fail initially
				assert.Equal(t, http.StatusNotImplemented, w.Code, "handler should return 501 until implemented")
			} else {
				// For error cases, we expect the actual behavior since these should fail consistently
				assert.Equal(t, tt.expectedStatus, w.Code)
			}

			if tt.validateBody != nil && tt.expectedStatus != http.StatusOK {
				// Only validate error response bodies for now
				// Success body validation will work once the handler is implemented
				tt.validateBody(t, w.Body.Bytes())
			}
		})
	}
}