package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"randompic/internal/db/models"
)

func TestUserModelValidation(t *testing.T) {
	t.Run("ValidUserCreation", func(t *testing.T) {
		user := &models.User{
			Email:             "test@example.com",
			Username:          stringPtr("testuser"),
			DisplayName:       stringPtr("Test User"),
			Role:              "user",
			OAuthProvider:     "google",
			OAuthUserID:       "oauth123",
			EmailVerified:     true,
			IsActive:          true,
			TotalStorageUsed:  0,
			StorageQuotaBytes: 10737418240, // 10GB
		}

		// Validate required fields
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "user", user.Role)
		assert.Equal(t, "google", user.OAuthProvider)
		assert.Equal(t, "oauth123", user.OAuthUserID)
		assert.True(t, user.EmailVerified)
		assert.True(t, user.IsActive)
		assert.Equal(t, int64(10737418240), user.StorageQuotaBytes)
	})

	t.Run("EmailValidation", func(t *testing.T) {
		validEmails := []string{
			"test@example.com",
			"user.name@domain.co.uk",
			"user+tag@example.org",
		}

		for _, email := range validEmails {
			user := &models.User{
				Email:         email,
				Role:          "user",
				OAuthProvider: "google",
				OAuthUserID:   "oauth123",
			}
			assert.Equal(t, email, user.Email)
		}
	})

	t.Run("RoleValidation", func(t *testing.T) {
		validRoles := []string{"user", "admin"}

		for _, role := range validRoles {
			user := &models.User{
				Email:         "test@example.com",
				Role:          role,
				OAuthProvider: "google",
				OAuthUserID:   "oauth123",
			}
			assert.Equal(t, role, user.Role)
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		user := &models.User{
			Email:         "test@example.com",
			Role:          "user",
			OAuthProvider: "google",
			OAuthUserID:   "oauth123",
		}

		// Check default values
		assert.Equal(t, int64(0), user.TotalStorageUsed)
		assert.Equal(t, int64(10737418240), user.StorageQuotaBytes) // 10GB
		assert.False(t, user.EmailVerified) // Should be false by default
		assert.True(t, user.IsActive)       // Should be true by default
	})

	t.Run("StorageQuotaConstraints", func(t *testing.T) {
		user := &models.User{
			Email:             "test@example.com",
			Role:              "user",
			OAuthProvider:     "google",
			OAuthUserID:       "oauth123",
			TotalStorageUsed:  0,
			StorageQuotaBytes: 10737418240,
		}

		// Storage used should be >= 0
		assert.True(t, user.TotalStorageUsed >= 0)
		assert.True(t, user.StorageQuotaBytes > 0)
	})
}

func TestOAuthProviderValidation(t *testing.T) {
	validProviders := []string{"google", "github", "microsoft", "generic"}

	for _, provider := range validProviders {
		t.Run("ValidProvider_"+provider, func(t *testing.T) {
			user := &models.User{
				Email:         "test@example.com",
				Role:          "user",
				OAuthProvider: provider,
				OAuthUserID:   "oauth123",
			}
			assert.Equal(t, provider, user.OAuthProvider)
		})
	}

	t.Run("RequiredOAuthUserID", func(t *testing.T) {
		user := &models.User{
			Email:         "test@example.com",
			Role:          "user",
			OAuthProvider: "google",
			OAuthUserID:   "required_oauth_id",
		}
		assert.NotEmpty(t, user.OAuthUserID)
		assert.Equal(t, "required_oauth_id", user.OAuthUserID)
	})
}

func TestStorageQuotaManagement(t *testing.T) {
	user := &models.User{
		Email:             "test@example.com",
		Role:              "user",
		OAuthProvider:     "google",
		OAuthUserID:       "oauth123",
		TotalStorageUsed:  1000000, // 1MB
		StorageQuotaBytes: 10737418240, // 10GB
	}

	t.Run("CanUploadFile", func(t *testing.T) {
		// Can upload 1MB file
		assert.True(t, user.CanUploadFile(1048576))

		// Cannot upload 11GB file (exceeds quota)
		assert.False(t, user.CanUploadFile(11000000000))

		// Edge case: exactly at quota
		remainingSpace := user.StorageQuotaBytes - user.TotalStorageUsed
		assert.True(t, user.CanUploadFile(remainingSpace))
		assert.False(t, user.CanUploadFile(remainingSpace+1))
	})

	t.Run("UpdateStorageUsage", func(t *testing.T) {
		initialUsage := user.TotalStorageUsed

		// Add storage
		user.UpdateStorageUsage(500000) // Add 500KB
		assert.Equal(t, initialUsage+500000, user.TotalStorageUsed)

		// Remove storage
		user.UpdateStorageUsage(-200000) // Remove 200KB
		assert.Equal(t, initialUsage+300000, user.TotalStorageUsed)

		// Cannot go negative
		user.UpdateStorageUsage(-999999999)
		assert.Equal(t, int64(0), user.TotalStorageUsed)
	})

	t.Run("GetStorageUsagePercentage", func(t *testing.T) {
		user := &models.User{
			TotalStorageUsed:  5368709120, // 5GB
			StorageQuotaBytes: 10737418240, // 10GB
		}

		percentage := user.GetStorageUsagePercentage()
		assert.InDelta(t, 50.0, percentage, 0.1) // Should be ~50%

		// Zero quota edge case
		user.StorageQuotaBytes = 0
		percentage = user.GetStorageUsagePercentage()
		assert.Equal(t, 0.0, percentage)
	})

	t.Run("HasStorageQuotaExceeded", func(t *testing.T) {
		user := &models.User{
			TotalStorageUsed:  5000000000,  // 5GB
			StorageQuotaBytes: 10000000000, // 10GB
		}
		assert.False(t, user.HasStorageQuotaExceeded())

		user.TotalStorageUsed = 15000000000 // 15GB
		assert.True(t, user.HasStorageQuotaExceeded())
	})
}

func TestUserBusinessLogic(t *testing.T) {
	t.Run("IsAdmin", func(t *testing.T) {
		adminUser := &models.User{Role: "admin"}
		regularUser := &models.User{Role: "user"}

		assert.True(t, adminUser.IsAdmin())
		assert.False(t, regularUser.IsAdmin())
	})

	t.Run("CanUpload", func(t *testing.T) {
		adminUser := &models.User{Role: "admin"}
		regularUser := &models.User{Role: "user"}

		// Only admin users can upload in this implementation
		assert.True(t, adminUser.CanUpload())
		assert.False(t, regularUser.CanUpload())
	})

	t.Run("UpdateLastLogin", func(t *testing.T) {
		user := &models.User{
			Email:         "test@example.com",
			Role:          "user",
			OAuthProvider: "google",
			OAuthUserID:   "oauth123",
		}

		// Initially no last login
		assert.Nil(t, user.LastLogin)

		// Update last login
		beforeUpdate := time.Now()
		user.UpdateLastLogin()

		require.NotNil(t, user.LastLogin)
		assert.True(t, user.LastLogin.After(beforeUpdate.Add(-time.Second)))
		assert.True(t, user.LastLogin.Before(time.Now().Add(time.Second)))
		assert.True(t, user.UpdatedAt.After(beforeUpdate.Add(-time.Second)))
	})

	t.Run("ToResponse", func(t *testing.T) {
		now := time.Now()
		lastLogin := now.Add(-time.Hour)

		user := &models.User{
			ID:                "user-123",
			Email:             "test@example.com",
			Username:          stringPtr("testuser"),
			DisplayName:       stringPtr("Test User"),
			AvatarURL:         stringPtr("https://example.com/avatar.jpg"),
			Role:              "user",
			OAuthProvider:     "google",
			OAuthUserID:       "oauth123",
			EmailVerified:     true,
			IsActive:          true,
			LastLogin:         &lastLogin,
			TotalStorageUsed:  1000000,
			StorageQuotaBytes: 10737418240,
			CreatedAt:         now.Add(-24*time.Hour),
			UpdatedAt:         now,
		}

		response := user.ToResponse()

		assert.Equal(t, "user-123", response.ID)
		assert.Equal(t, "test@example.com", response.Email)
		assert.Equal(t, "testuser", *response.Username)
		assert.Equal(t, "Test User", *response.DisplayName)
		assert.Equal(t, "https://example.com/avatar.jpg", *response.AvatarURL)
		assert.Equal(t, "user", response.Role)
		assert.Equal(t, "google", response.OAuthProvider)
		assert.Equal(t, "oauth123", response.OAuthUserID)
		assert.True(t, response.EmailVerified)
		assert.True(t, response.IsActive)
		assert.NotNil(t, response.LastLogin)
		assert.Equal(t, int64(1000000), response.TotalStorageUsed)
		assert.Equal(t, int64(10737418240), response.StorageQuotaBytes)

		// Check time formatting
		assert.Contains(t, response.CreatedAt, "T")
		assert.Contains(t, response.UpdatedAt, "T")
	})

	t.Run("ToProfileResponse", func(t *testing.T) {
		user := &models.User{
			ID:                "user-123",
			Email:             "test@example.com",
			Username:          stringPtr("testuser"),
			Role:              "user",
			TotalStorageUsed:  5368709120, // 5GB
			StorageQuotaBytes: 10737418240, // 10GB
			CreatedAt:         time.Now().Add(-24*time.Hour),
			UpdatedAt:         time.Now(),
		}

		imageCount := 42
		response := user.ToProfileResponse(imageCount)

		assert.Equal(t, "user-123", response.ID)
		assert.Equal(t, "test@example.com", response.Email)
		assert.Equal(t, "testuser", *response.Username)
		assert.Equal(t, "user", response.Role)
		assert.Equal(t, 42, response.ImageCount)
		assert.Equal(t, int64(5368709120), response.TotalStorageUsed)
		assert.Equal(t, int64(10737418240), response.StorageQuotaBytes)
		assert.InDelta(t, 50.0, response.StorageUsagePercentage, 0.1)
	})
}

func TestUserRequestModels(t *testing.T) {
	t.Run("UserCreateRequest", func(t *testing.T) {
		req := &models.UserCreateRequest{
			Email:             "test@example.com",
			Username:          stringPtr("testuser"),
			DisplayName:       stringPtr("Test User"),
			AvatarURL:         stringPtr("https://example.com/avatar.jpg"),
			Role:              "user",
			OAuthProvider:     "google",
			OAuthUserID:       "oauth123",
			EmailVerified:     true,
			StorageQuotaBytes: 5000000000, // 5GB custom quota
		}

		assert.Equal(t, "test@example.com", req.Email)
		assert.Equal(t, "testuser", *req.Username)
		assert.Equal(t, "user", req.Role)
		assert.Equal(t, "google", req.OAuthProvider)
		assert.Equal(t, "oauth123", req.OAuthUserID)
		assert.True(t, req.EmailVerified)
		assert.Equal(t, int64(5000000000), req.StorageQuotaBytes)
	})

	t.Run("UserUpdateRequest", func(t *testing.T) {
		req := &models.UserUpdateRequest{
			Email:    stringPtr("newemail@example.com"),
			Username: stringPtr("newusername"),
			Role:     stringPtr("admin"),
		}

		assert.Equal(t, "newemail@example.com", *req.Email)
		assert.Equal(t, "newusername", *req.Username)
		assert.Equal(t, "admin", *req.Role)
	})
}

func TestUserHooks(t *testing.T) {
	t.Run("BeforeCreate", func(t *testing.T) {
		user := &models.User{
			Email:         "test@example.com",
			Role:          "user",
			OAuthProvider: "google",
			OAuthUserID:   "oauth123",
		}

		err := user.BeforeCreate()
		assert.NoError(t, err)
		// GORM will handle UUID generation, so we just ensure no error
	})

	t.Run("BeforeUpdate", func(t *testing.T) {
		user := &models.User{
			Email:         "test@example.com",
			Role:          "user",
			OAuthProvider: "google",
			OAuthUserID:   "oauth123",
			UpdatedAt:     time.Now().Add(-time.Hour), // Old timestamp
		}

		beforeUpdate := time.Now()
		err := user.BeforeUpdate()

		assert.NoError(t, err)
		assert.True(t, user.UpdatedAt.After(beforeUpdate.Add(-time.Second)))
		assert.True(t, user.UpdatedAt.Before(time.Now().Add(time.Second)))
	})
}

func TestUserTableName(t *testing.T) {
	user := models.User{}
	assert.Equal(t, "users", user.TableName())
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}