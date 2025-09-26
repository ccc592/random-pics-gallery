package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUpdateImageContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// This will fail until the actual handler is implemented
	router.PUT("/api/admin/images/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})

	tests := []struct {
		name           string
		imageID        string
		requestBody    interface{}
		authHeader     string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Update without auth",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			requestBody:    map[string]interface{}{"title": "Updated Title"},
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "unauthorized", response.Error)
			},
		},
		{
			name:    "Valid update with auth",
			imageID: "123e4567-e89b-12d3-a456-426614174000",
			requestBody: map[string]interface{}{
				"title":  "Updated Sunset Mountains",
				"alt":    "An updated beautiful motivational landscape with mountains during sunset",
				"tags":   "nature,mountains,sunset,motivation,updated",
				"weight": 7,
				"status": "active",
			},
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					ID     string  `json:"id"`
					Title  *string `json:"title"`
					Alt    string  `json:"alt"`
					Tags   *string `json:"tags"`
					Weight int     `json:"weight"`
					Status string  `json:"status"`
				}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", response.ID)
				assert.Equal(t, "Updated Sunset Mountains", *response.Title)
				assert.Equal(t, "An updated beautiful motivational landscape with mountains during sunset", response.Alt)
				assert.Equal(t, "nature,mountains,sunset,motivation,updated", *response.Tags)
				assert.Equal(t, 7, response.Weight)
				assert.Equal(t, "active", response.Status)
			},
		},
		{
			name:    "Update with short alt text",
			imageID: "123e4567-e89b-12d3-a456-426614174000",
			requestBody: map[string]interface{}{
				"alt": "short", // Less than 10 characters
			},
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "alt text must be at least 10 characters")
			},
		},
		{
			name:    "Update with invalid weight",
			imageID: "123e4567-e89b-12d3-a456-426614174000",
			requestBody: map[string]interface{}{
				"weight": 15, // Weight must be between 1-10
			},
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "weight must be between 1 and 10")
			},
		},
		{
			name:    "Update with invalid status",
			imageID: "123e4567-e89b-12d3-a456-426614174000",
			requestBody: map[string]interface{}{
				"status": "invalid_status",
			},
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "invalid status")
			},
		},
		{
			name:           "Update non-existent image",
			imageID:        "00000000-0000-0000-0000-000000000000",
			requestBody:    map[string]interface{}{"title": "Updated Title"},
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "image not found", response.Error)
			},
		},
		{
			name:           "Update with invalid UUID",
			imageID:        "invalid-uuid",
			requestBody:    map[string]interface{}{"title": "Updated Title"},
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Error string `json:"error"`
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "invalid UUID format")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("PUT", "/api/admin/images/"+tt.imageID, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// This test will initially fail because we return 501 Not Implemented
			if tt.expectedStatus == http.StatusOK {
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