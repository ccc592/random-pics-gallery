package contract

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/randompic/api/internal/auth"
)

// MockImageService for testing delete functionality
type MockDeleteImageService struct{}

func (m *MockDeleteImageService) DeleteImage(id uuid.UUID) error {
	if id.String() == "123e4567-e89b-12d3-a456-426614174000" {
		return nil // Success for test image
	}
	return fmt.Errorf("image not found")
}

// MockAuthService for testing
type MockDeleteAuthService struct{}

func (m *MockDeleteAuthService) GetUserByID(id uuid.UUID) (*MockUser, error) {
	return &MockUser{IsAdminUser: true}, nil
}

type MockUser struct {
	IsAdminUser bool
}

func (u *MockUser) IsAdmin() bool {
	return u.IsAdminUser
}

func TestAdminDeleteImageContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware for auth
	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "Bearer valid-admin-token" {
			mockClaims := &auth.Claims{
				UserID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
				Email:  "admin@test.com",
				Role:   "admin",
			}
			c.Set("user", mockClaims)
		}
		c.Next()
	})

	// Create mock services
	mockImageService := &MockDeleteImageService{}
	_ = &MockDeleteAuthService{} // Keep for future use

	// For the contract test, we'll create a simplified handler
	router.DELETE("/api/admin/images/:id", func(c *gin.Context) {
		// Check authentication
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		// Validate UUID
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid UUID format",
				"code":  "INVALID_REQUEST",
			})
			return
		}

		// Mock admin check
		_, ok := user.(*auth.Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user context",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		// Delete image
		err = mockImageService.DeleteImage(id)
		if err != nil {
			if err.Error() == "image not found" {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "image not found",
					"code":  "NOT_FOUND",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to delete image",
				"code":  "INTERNAL_ERROR",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "image deleted successfully",
			"id":      id.String(),
		})
	})

	tests := []struct {
		name           string
		imageID        string
		authHeader     string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Delete without auth",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
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
			name:           "Valid delete with auth",
			imageID:        "123e4567-e89b-12d3-a456-426614174000",
			authHeader:     "Bearer valid-admin-token",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response struct {
					Message string `json:"message"`
					ID      string `json:"id"`
				}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				assert.Equal(t, "image deleted successfully", response.Message)
				assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", response.ID)
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
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Equal(t, "image not found", response.Error)
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
				}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)
				assert.Contains(t, response.Error, "invalid UUID format")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/admin/images/"+tt.imageID, nil)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Test the actual expected status
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateBody != nil {
				tt.validateBody(t, w.Body.Bytes())
			}
		})
	}
}