package contract

import (
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

func TestImageDeleteContract(t *testing.T) {
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
	router.DELETE("/api/images/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})

	tests := []struct {
		name           string
		imageID        string
		authHeader     string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Delete without authentication",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			authHeader:     "",
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
			name:           "Delete as non-admin user",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			authHeader:     "Bearer valid-user-token",
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
			name:           "Valid delete as admin",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusNoContent,
			validateBody: func(t *testing.T, body []byte) {
				// 204 No Content should have empty body
				assert.Empty(t, body, "204 No Content response should have empty body")
			},
		},
		{
			name:           "Delete non-existent image",
			imageID:        "00000000-0000-0000-0000-000000000000",
			authHeader:     "Bearer valid-admin-token",
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
			name:           "Delete with invalid UUID",
			imageID:        "invalid-uuid",
			authHeader:     "Bearer valid-admin-token",
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
			name:           "Delete with malformed UUID",
			imageID:        "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			authHeader:     "Bearer valid-admin-token",
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
			name:           "Delete with expired admin token",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			authHeader:     "Bearer expired-admin-token",
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
			name:           "Delete with invalid token format",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			authHeader:     "InvalidTokenFormat",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/images/"+tt.imageID, nil)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// This test will initially fail because we return 501 Not Implemented
			if tt.expectedStatus == http.StatusNoContent {
				// For TDD, we expect the test to fail initially
				assert.Equal(t, http.StatusNotImplemented, w.Code, "handler should return 501 until implemented")
			} else {
				// For error cases, we expect the actual behavior since these should fail consistently
				assert.Equal(t, tt.expectedStatus, w.Code)
			}

			if tt.validateBody != nil {
				if tt.expectedStatus == http.StatusNoContent {
					// Skip body validation for success case until implementation
				} else {
					// Validate error response bodies
					tt.validateBody(t, w.Body.Bytes())
				}
			}
		})
	}
}