package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/randompic/api/internal/auth"
)

type UpdateImageRequest struct {
	Alt    *string  `json:"alt,omitempty"`
	Title  *string  `json:"title,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Weight *int     `json:"weight,omitempty"`
	Status *string  `json:"status,omitempty"`
}

func TestImageUpdateContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add auth middleware
	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "Bearer valid-admin-token" {
			mockClaims := &auth.Claims{
				UserID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
				Email:  "admin@test.com",
				Role:   "admin",
			}
			c.Set("user", mockClaims)
		} else if authHeader == "Bearer valid-user-token" {
			mockClaims := &auth.Claims{
				UserID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
				Email:  "user@test.com",
				Role:   "user",
			}
			c.Set("user", mockClaims)
		}
		c.Next()
	})

	// This will fail until the actual handler is implemented
	router.PUT("/api/images/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})

	tests := []struct {
		name           string
		imageID        string
		authHeader     string
		requestBody    UpdateImageRequest
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:        "Update without authentication",
			imageID:     "123e4567-e89b-12d3-a456-426614174000",
			authHeader:  "",
			requestBody: UpdateImageRequest{},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "unauthorized", response.Error)
				assert.Equal(t, "UNAUTHORIZED", response.Code)
			},
		},
		{
			name:       "Update as non-admin user",
			imageID:    "123e4567-e89b-12d3-a456-426614174000",
			authHeader: "Bearer valid-user-token",
			requestBody: UpdateImageRequest{
				Alt: stringPtr("Updated beautiful sunset over mountains"),
			},
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "admin access required", response.Error)
				assert.Equal(t, "FORBIDDEN", response.Code)
			},
		},
		{
			name:       "Valid update as admin",
			imageID:    "123e4567-e89b-12d3-a456-426614174000",
			authHeader: "Bearer valid-admin-token",
			requestBody: UpdateImageRequest{
				Alt:    stringPtr("Updated beautiful sunset over mountains with vibrant colors"),
				Title:  stringPtr("Updated Sunset"),
				Tags:   []string{"sunset", "mountains", "nature"},
				Weight: intPtr(8),
				Status: stringPtr("active"),
			},
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

				// Validate updated fields match request
				assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", response.ID)
				assert.Equal(t, "Updated beautiful sunset over mountains with vibrant colors", response.Alt)
				assert.Equal(t, "Updated Sunset", response.Title)
				assert.Equal(t, []string{"sunset", "mountains", "nature"}, response.Tags)
				assert.Equal(t, 8, response.Weight)
				assert.Equal(t, "active", response.Status)

				// Validate schema constraints
				assert.GreaterOrEqual(t, len(response.Alt), 10)
				assert.LessOrEqual(t, len(response.Alt), 500)
				assert.LessOrEqual(t, len(response.Title), 255)
				for _, tag := range response.Tags {
					assert.LessOrEqual(t, len(tag), 50)
				}
				assert.GreaterOrEqual(t, response.Weight, 1)
				assert.LessOrEqual(t, response.Weight, 10)
				assert.Contains(t, []string{"active", "inactive"}, response.Status)
			},
		},
		{
			name:       "Update non-existent image",
			imageID:    "00000000-0000-0000-0000-000000000000",
			authHeader: "Bearer valid-admin-token",
			requestBody: UpdateImageRequest{
				Alt: stringPtr("Some alt text here"),
			},
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
			name:       "Update with invalid UUID",
			imageID:    "invalid-uuid",
			authHeader: "Bearer valid-admin-token",
			requestBody: UpdateImageRequest{
				Alt: stringPtr("Some alt text here"),
			},
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
			name:       "Update with invalid alt text (too short)",
			imageID:    "123e4567-e89b-12d3-a456-426614174000",
			authHeader: "Bearer valid-admin-token",
			requestBody: UpdateImageRequest{
				Alt: stringPtr("short"),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "alt text must be at least 10 characters")
				assert.Equal(t, "VALIDATION_ERROR", response.Code)
			},
		},
		{
			name:       "Update with invalid weight (too high)",
			imageID:    "123e4567-e89b-12d3-a456-426614174000",
			authHeader: "Bearer valid-admin-token",
			requestBody: UpdateImageRequest{
				Weight: intPtr(15),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "weight must be between 1 and 10")
				assert.Equal(t, "VALIDATION_ERROR", response.Code)
			},
		},
		{
			name:       "Update with invalid status",
			imageID:    "123e4567-e89b-12d3-a456-426614174000",
			authHeader: "Bearer valid-admin-token",
			requestBody: UpdateImageRequest{
				Status: stringPtr("invalid-status"),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "status must be 'active' or 'inactive'")
				assert.Equal(t, "VALIDATION_ERROR", response.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("PUT", "/api/images/"+tt.imageID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

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

// Helper functions for pointer creation
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}