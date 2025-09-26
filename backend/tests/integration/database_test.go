package integration

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/db/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testDatabaseDSN = "host=localhost port=5432 user=postgres password=postgres dbname=randompic_test sslmode=disable TimeZone=UTC"
	maxTestRetries  = 5
	retryDelay      = time.Second * 2
)

// TestDatabase represents the test database instance
type TestDatabase struct {
	DB     *db.Database
	Config *db.Config
}

// newTestDatabase creates a new test database connection
func newTestDatabase(t *testing.T) *TestDatabase {
	// Use environment variable if available, otherwise use default
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = testDatabaseDSN
	}

	config := &db.Config{
		Driver:          "postgres",
		DSN:             dsn,
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
		LogLevel:        logger.Info,
	}

	var database *db.Database
	var err error

	// Retry connection with backoff for CI environments
	for i := 0; i < maxTestRetries; i++ {
		database, err = db.NewDatabase(config)
		if err == nil {
			break
		}

		if i < maxTestRetries-1 {
			t.Logf("Database connection attempt %d failed: %v. Retrying in %v...", i+1, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	require.NoError(t, err, "Failed to connect to test database after %d attempts", maxTestRetries)
	require.NotNil(t, database, "Database instance should not be nil")

	return &TestDatabase{
		DB:     database,
		Config: config,
	}
}

// cleanup removes all test data from the database
func (td *TestDatabase) cleanup(t *testing.T) {
	// Delete in reverse order due to foreign key constraints
	err := td.DB.DB.Exec("DELETE FROM images").Error
	require.NoError(t, err, "Failed to cleanup images table")

	err = td.DB.DB.Exec("DELETE FROM users").Error
	require.NoError(t, err, "Failed to cleanup users table")

	err = td.DB.DB.Exec("DELETE FROM api_requests").Error
	require.NoError(t, err, "Failed to cleanup api_requests table")
}

// close closes the test database connection
func (td *TestDatabase) close(t *testing.T) {
	if td.DB != nil {
		err := td.DB.Close()
		assert.NoError(t, err, "Failed to close database connection")
	}
}

func TestDatabaseConnection(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)

	t.Run("Connection Health Check", func(t *testing.T) {
		err := testDB.DB.Ping()
		assert.NoError(t, err, "Database ping should succeed")
	})

	t.Run("Connection Pool Stats", func(t *testing.T) {
		stats, err := testDB.DB.GetStats()
		require.NoError(t, err, "Should retrieve connection stats")

		assert.Contains(t, stats, "max_open_connections", "Stats should contain max_open_connections")
		assert.Contains(t, stats, "open_connections", "Stats should contain open_connections")

		maxOpen, ok := stats["max_open_connections"].(int)
		require.True(t, ok, "max_open_connections should be an integer")
		assert.Equal(t, testDB.Config.MaxOpenConns, maxOpen, "Max open connections should match config")
	})

	t.Run("Comprehensive Health Check", func(t *testing.T) {
		err := testDB.DB.HealthCheck()
		assert.NoError(t, err, "Comprehensive health check should pass")
	})
}

func TestDatabaseMigrations(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)

	t.Run("Auto Migration", func(t *testing.T) {
		err := testDB.DB.AutoMigrate()
		assert.NoError(t, err, "Auto migration should succeed")

		// Verify all tables exist
		tables := []string{"users", "images", "api_requests"}
		for _, table := range tables {
			var exists bool
			err := testDB.DB.DB.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = ?)", table).Scan(&exists).Error
			require.NoError(t, err, "Failed to check if table %s exists", table)
			assert.True(t, exists, "Table %s should exist after migration", table)
		}
	})
}

func TestUserCRUDOperations(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)
	defer testDB.cleanup(t)

	// Run migrations first
	err := testDB.DB.AutoMigrate()
	require.NoError(t, err, "Migration should succeed")

	t.Run("Create User", func(t *testing.T) {
		user := &models.User{
			Email:       "test@example.com",
			Username:    dbStringPtr("testuser"),
			DisplayName: dbStringPtr("Test User"),
			Role:        "user",
			KindeUserID: "kinde_test_123",
			EmailVerified: true,
		}

		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")
		assert.NotEqual(t, uuid.Nil, user.ID, "User ID should be generated")
		assert.NotZero(t, user.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, user.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Read User", func(t *testing.T) {
		// Create user first
		user := &models.User{
			Email:       "read@example.com",
			KindeUserID: "kinde_read_123",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		// Read by ID
		var foundUser models.User
		err = testDB.DB.DB.First(&foundUser, user.ID).Error
		require.NoError(t, err, "User retrieval by ID should succeed")
		assert.Equal(t, user.Email, foundUser.Email, "Email should match")

		// Read by email
		var foundByEmail models.User
		err = testDB.DB.DB.Where("email = ?", user.Email).First(&foundByEmail).Error
		require.NoError(t, err, "User retrieval by email should succeed")
		assert.Equal(t, user.ID, foundByEmail.ID, "ID should match")

		// Read by KindeUserID
		var foundByKinde models.User
		err = testDB.DB.DB.Where("kinde_user_id = ?", user.KindeUserID).First(&foundByKinde).Error
		require.NoError(t, err, "User retrieval by KindeUserID should succeed")
		assert.Equal(t, user.ID, foundByKinde.ID, "ID should match")
	})

	t.Run("Update User", func(t *testing.T) {
		// Create user first
		user := &models.User{
			Email:       "update@example.com",
			KindeUserID: "kinde_update_123",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		originalUpdatedAt := user.UpdatedAt

		// Update user
		time.Sleep(time.Millisecond * 10) // Ensure timestamp difference
		user.DisplayName = dbStringPtr("Updated Name")
		user.Role = "admin"

		err = testDB.DB.DB.Save(user).Error
		require.NoError(t, err, "User update should succeed")
		assert.Equal(t, "Updated Name", *user.DisplayName, "DisplayName should be updated")
		assert.Equal(t, "admin", user.Role, "Role should be updated")
		assert.True(t, user.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be updated")
	})

	t.Run("Delete User", func(t *testing.T) {
		// Create user first
		user := &models.User{
			Email:       "delete@example.com",
			KindeUserID: "kinde_delete_123",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		// Delete user
		err = testDB.DB.DB.Delete(user).Error
		require.NoError(t, err, "User deletion should succeed")

		// Verify deletion
		var deletedUser models.User
		err = testDB.DB.DB.First(&deletedUser, user.ID).Error
		assert.Error(t, err, "User should not be found after deletion")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Error should be record not found")
	})
}

func TestImageCRUDOperations(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)
	defer testDB.cleanup(t)

	// Run migrations first
	err := testDB.DB.AutoMigrate()
	require.NoError(t, err, "Migration should succeed")

	// Create a test user for foreign key relationships
	user := &models.User{
		Email:       "imageowner@example.com",
		KindeUserID: "kinde_owner_123",
		Role:        "admin",
	}
	err = testDB.DB.DB.Create(user).Error
	require.NoError(t, err, "User creation should succeed")

	t.Run("Create Image", func(t *testing.T) {
		image := &models.Image{
			Filename:    "test-image.jpg",
			Alt:         "A beautiful test image for testing purposes",
			Title:       dbStringPtr("Test Image"),
			Tags:        dbStringPtr("test,nature,beautiful"),
			Weight:      5,
			StoragePath: "/storage/2024/01/01/test-image.jpg",
			MimeType:    "image/jpeg",
			FileSize:    1024000,
			Width:       dbIntPtr(1920),
			Height:      dbIntPtr(1080),
			AspectRatio: dbFloat64Ptr(1.778),
			Status:      "active",
			UploadedBy:  &user.ID,
		}

		err := testDB.DB.DB.Create(image).Error
		require.NoError(t, err, "Image creation should succeed")
		assert.NotEqual(t, uuid.Nil, image.ID, "Image ID should be generated")
		assert.NotZero(t, image.CreatedAt, "CreatedAt should be set")
		assert.NotZero(t, image.UpdatedAt, "UpdatedAt should be set")
	})

	t.Run("Read Image with User Relationship", func(t *testing.T) {
		// Create image first
		image := &models.Image{
			Filename:    "read-test.jpg",
			Alt:         "A readable test image for verification",
			StoragePath: "/storage/read-test.jpg",
			MimeType:    "image/jpeg",
			FileSize:    512000,
			Status:      "active",
			UploadedBy:  &user.ID,
		}
		err := testDB.DB.DB.Create(image).Error
		require.NoError(t, err, "Image creation should succeed")

		// Read with user relationship
		var foundImage models.Image
		err = testDB.DB.DB.Preload("UploadedByUser").First(&foundImage, image.ID).Error
		require.NoError(t, err, "Image retrieval should succeed")
		assert.Equal(t, image.Filename, foundImage.Filename, "Filename should match")
		assert.NotNil(t, foundImage.UploadedByUser, "User relationship should be loaded")
		assert.Equal(t, user.Email, foundImage.UploadedByUser.Email, "User email should match")
	})

	t.Run("Update Image", func(t *testing.T) {
		// Create image first
		image := &models.Image{
			Filename:    "update-test.jpg",
			Alt:         "An updatable test image for modification",
			StoragePath: "/storage/update-test.jpg",
			MimeType:    "image/jpeg",
			FileSize:    256000,
			Status:      "processing",
			UploadedBy:  &user.ID,
		}
		err := testDB.DB.DB.Create(image).Error
		require.NoError(t, err, "Image creation should succeed")

		originalUpdatedAt := image.UpdatedAt

		// Update image
		time.Sleep(time.Millisecond * 10) // Ensure timestamp difference
		image.Alt = "Updated test image with new description"
		image.Status = "active"
		image.Weight = 8

		err = testDB.DB.DB.Save(image).Error
		require.NoError(t, err, "Image update should succeed")
		assert.Equal(t, "Updated test image with new description", image.Alt, "Alt should be updated")
		assert.Equal(t, "active", image.Status, "Status should be updated")
		assert.Equal(t, 8, image.Weight, "Weight should be updated")
		assert.True(t, image.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be updated")
	})

	t.Run("Delete Image", func(t *testing.T) {
		// Create image first
		image := &models.Image{
			Filename:    "delete-test.jpg",
			Alt:         "A deletable test image for removal",
			StoragePath: "/storage/delete-test.jpg",
			MimeType:    "image/jpeg",
			FileSize:    128000,
			Status:      "active",
			UploadedBy:  &user.ID,
		}
		err := testDB.DB.DB.Create(image).Error
		require.NoError(t, err, "Image creation should succeed")

		// Delete image
		err = testDB.DB.DB.Delete(image).Error
		require.NoError(t, err, "Image deletion should succeed")

		// Verify deletion
		var deletedImage models.Image
		err = testDB.DB.DB.First(&deletedImage, image.ID).Error
		assert.Error(t, err, "Image should not be found after deletion")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Error should be record not found")
	})
}

func TestDatabaseConstraints(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)
	defer testDB.cleanup(t)

	// Run migrations first
	err := testDB.DB.AutoMigrate()
	require.NoError(t, err, "Migration should succeed")

	t.Run("User Email Uniqueness", func(t *testing.T) {
		user1 := &models.User{
			Email:       "unique@example.com",
			KindeUserID: "kinde_unique_1",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user1).Error
		require.NoError(t, err, "First user creation should succeed")

		user2 := &models.User{
			Email:       "unique@example.com", // Same email
			KindeUserID: "kinde_unique_2",     // Different KindeUserID
			Role:        "user",
		}
		err = testDB.DB.DB.Create(user2).Error
		assert.Error(t, err, "Second user with same email should fail")
		assert.Contains(t, err.Error(), "duplicate", "Error should mention duplicate constraint")
	})

	t.Run("User KindeUserID Uniqueness", func(t *testing.T) {
		user1 := &models.User{
			Email:       "kinde1@example.com",
			KindeUserID: "kinde_unique_constraint",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user1).Error
		require.NoError(t, err, "First user creation should succeed")

		user2 := &models.User{
			Email:       "kinde2@example.com", // Different email
			KindeUserID: "kinde_unique_constraint", // Same KindeUserID
			Role:        "user",
		}
		err = testDB.DB.DB.Create(user2).Error
		assert.Error(t, err, "Second user with same KindeUserID should fail")
		assert.Contains(t, err.Error(), "duplicate", "Error should mention duplicate constraint")
	})

	t.Run("Image Alt Text Minimum Length", func(t *testing.T) {
		// Create user for foreign key
		user := &models.User{
			Email:       "altowner@example.com",
			KindeUserID: "kinde_alt_owner",
			Role:        "admin",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		image := &models.Image{
			Filename:    "short-alt.jpg",
			Alt:         "Short", // Less than 10 characters
			StoragePath: "/storage/short-alt.jpg",
			MimeType:    "image/jpeg",
			FileSize:    100000,
			Status:      "active",
			UploadedBy:  &user.ID,
		}
		err = testDB.DB.DB.Create(image).Error
		assert.Error(t, err, "Image with short alt text should fail")
		assert.Contains(t, err.Error(), "check", "Error should mention check constraint")
	})

	t.Run("Image Weight Range", func(t *testing.T) {
		// Create user for foreign key
		user := &models.User{
			Email:       "weightowner@example.com",
			KindeUserID: "kinde_weight_owner",
			Role:        "admin",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		// Test weight too high
		imageHigh := &models.Image{
			Filename:    "high-weight.jpg",
			Alt:         "Image with weight too high for testing",
			Weight:      15, // Above maximum of 10
			StoragePath: "/storage/high-weight.jpg",
			MimeType:    "image/jpeg",
			FileSize:    100000,
			Status:      "active",
			UploadedBy:  &user.ID,
		}
		err = testDB.DB.DB.Create(imageHigh).Error
		assert.Error(t, err, "Image with weight > 10 should fail")

		// Test weight too low
		imageLow := &models.Image{
			Filename:    "low-weight.jpg",
			Alt:         "Image with weight too low for testing",
			Weight:      0, // Below minimum of 1
			StoragePath: "/storage/low-weight.jpg",
			MimeType:    "image/jpeg",
			FileSize:    100000,
			Status:      "active",
			UploadedBy:  &user.ID,
		}
		err = testDB.DB.DB.Create(imageLow).Error
		assert.Error(t, err, "Image with weight < 1 should fail")
	})

	t.Run("Foreign Key Constraint", func(t *testing.T) {
		nonExistentUserID := uuid.New()

		image := &models.Image{
			Filename:    "orphan.jpg",
			Alt:         "Orphaned image without valid user",
			StoragePath: "/storage/orphan.jpg",
			MimeType:    "image/jpeg",
			FileSize:    100000,
			Status:      "active",
			UploadedBy:  &nonExistentUserID,
		}
		err = testDB.DB.DB.Create(image).Error
		assert.Error(t, err, "Image with non-existent user should fail")
		assert.Contains(t, err.Error(), "violates foreign key constraint", "Error should mention foreign key constraint")
	})
}

func TestTransactionHandling(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)
	defer testDB.cleanup(t)

	// Run migrations first
	err := testDB.DB.AutoMigrate()
	require.NoError(t, err, "Migration should succeed")

	t.Run("Transaction Commit", func(t *testing.T) {
		err := testDB.DB.WithTransaction(func(tx *gorm.DB) error {
			user := &models.User{
				Email:       "commit@example.com",
				KindeUserID: "kinde_commit_123",
				Role:        "user",
			}
			if err := tx.Create(user).Error; err != nil {
				return err
			}

			image := &models.Image{
				Filename:    "commit-test.jpg",
				Alt:         "Transaction commit test image",
				StoragePath: "/storage/commit-test.jpg",
				MimeType:    "image/jpeg",
				FileSize:    100000,
				Status:      "active",
				UploadedBy:  &user.ID,
			}
			return tx.Create(image).Error
		})
		require.NoError(t, err, "Transaction should commit successfully")

		// Verify data exists
		var userCount, imageCount int64
		testDB.DB.DB.Model(&models.User{}).Where("email = ?", "commit@example.com").Count(&userCount)
		testDB.DB.DB.Model(&models.Image{}).Where("filename = ?", "commit-test.jpg").Count(&imageCount)

		assert.Equal(t, int64(1), userCount, "User should exist after commit")
		assert.Equal(t, int64(1), imageCount, "Image should exist after commit")
	})

	t.Run("Transaction Rollback", func(t *testing.T) {
		err := testDB.DB.WithTransaction(func(tx *gorm.DB) error {
			user := &models.User{
				Email:       "rollback@example.com",
				KindeUserID: "kinde_rollback_123",
				Role:        "user",
			}
			if err := tx.Create(user).Error; err != nil {
				return err
			}

			// Intentionally cause an error to trigger rollback
			image := &models.Image{
				Filename:    "rollback-test.jpg",
				Alt:         "Short", // This will fail due to constraint
				StoragePath: "/storage/rollback-test.jpg",
				MimeType:    "image/jpeg",
				FileSize:    100000,
				Status:      "active",
				UploadedBy:  &user.ID,
			}
			return tx.Create(image).Error
		})
		assert.Error(t, err, "Transaction should fail and rollback")

		// Verify no data exists
		var userCount, imageCount int64
		testDB.DB.DB.Model(&models.User{}).Where("email = ?", "rollback@example.com").Count(&userCount)
		testDB.DB.DB.Model(&models.Image{}).Where("filename = ?", "rollback-test.jpg").Count(&imageCount)

		assert.Equal(t, int64(0), userCount, "User should not exist after rollback")
		assert.Equal(t, int64(0), imageCount, "Image should not exist after rollback")
	})

	t.Run("Manual Transaction Control", func(t *testing.T) {
		tx := testDB.DB.BeginTransaction()

		user := &models.User{
			Email:       "manual@example.com",
			KindeUserID: "kinde_manual_123",
			Role:        "user",
		}
		err := tx.Create(user).Error
		require.NoError(t, err, "User creation in transaction should succeed")

		// Before commit, data should not be visible outside transaction
		var count int64
		testDB.DB.DB.Model(&models.User{}).Where("email = ?", "manual@example.com").Count(&count)
		assert.Equal(t, int64(0), count, "User should not be visible before commit")

		// Commit transaction
		err = tx.Commit().Error
		require.NoError(t, err, "Transaction commit should succeed")

		// After commit, data should be visible
		testDB.DB.DB.Model(&models.User{}).Where("email = ?", "manual@example.com").Count(&count)
		assert.Equal(t, int64(1), count, "User should be visible after commit")
	})
}

func TestConcurrentDatabaseAccess(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)
	defer testDB.cleanup(t)

	// Run migrations first
	err := testDB.DB.AutoMigrate()
	require.NoError(t, err, "Migration should succeed")

	t.Run("Concurrent User Creation", func(t *testing.T) {
		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				user := &models.User{
					Email:       fmt.Sprintf("concurrent%d@example.com", id),
					KindeUserID: fmt.Sprintf("kinde_concurrent_%d", id),
					Role:        "user",
				}

				if err := testDB.DB.DB.Create(user).Error; err != nil {
					errors <- err
					return
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent user creation failed: %v", err)
		}

		// Verify all users were created
		var count int64
		testDB.DB.DB.Model(&models.User{}).Where("email LIKE 'concurrent%@example.com'").Count(&count)
		assert.Equal(t, int64(numGoroutines), count, "All concurrent users should be created")
	})

	t.Run("Concurrent Read Operations", func(t *testing.T) {
		// Create a test user first
		user := &models.User{
			Email:       "readtest@example.com",
			KindeUserID: "kinde_readtest_123",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		const numReads = 20
		var wg sync.WaitGroup
		errors := make(chan error, numReads)
		results := make(chan models.User, numReads)

		for i := 0; i < numReads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var foundUser models.User
				if err := testDB.DB.DB.Where("email = ?", "readtest@example.com").First(&foundUser).Error; err != nil {
					errors <- err
					return
				}
				results <- foundUser
			}()
		}

		wg.Wait()
		close(errors)
		close(results)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent read failed: %v", err)
		}

		// Verify all reads returned the same user
		readCount := 0
		for result := range results {
			assert.Equal(t, user.ID, result.ID, "All reads should return the same user")
			readCount++
		}
		assert.Equal(t, numReads, readCount, "All reads should complete successfully")
	})
}

func TestOAuthUserOperations(t *testing.T) {
	t.Parallel()

	testDB := newTestDatabase(t)
	defer testDB.close(t)
	defer testDB.cleanup(t)

	// Run migrations first
	err := testDB.DB.AutoMigrate()
	require.NoError(t, err, "Migration should succeed")

	t.Run("OAuth User Creation", func(t *testing.T) {
		user := &models.User{
			Email:               "oauth@example.com",
			Username:            dbStringPtr("oauthuser"),
			DisplayName:         dbStringPtr("OAuth User"),
			AvatarURL:           dbStringPtr("https://example.com/avatar.jpg"),
			Role:                "user",
			KindeUserID:         "kinde_oauth_123",
			KindeOrganizationID: dbStringPtr("org_123"),
			JWTSubject:          dbStringPtr("sub_123"),
			EmailVerified:       true,
		}

		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "OAuth user creation should succeed")

		assert.NotEqual(t, uuid.Nil, user.ID, "User ID should be generated")
		assert.Equal(t, "oauth@example.com", user.Email, "Email should be set")
		assert.Equal(t, "kinde_oauth_123", user.KindeUserID, "KindeUserID should be set")
		assert.True(t, user.EmailVerified, "EmailVerified should be true")
	})

	t.Run("OAuth User Update", func(t *testing.T) {
		// Create user
		user := &models.User{
			Email:         "oauthupdate@example.com",
			KindeUserID:   "kinde_oauth_update_123",
			Role:          "user",
			EmailVerified: false,
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		// Simulate OAuth profile sync
		user.DisplayName = dbStringPtr("Updated OAuth User")
		user.AvatarURL = dbStringPtr("https://example.com/new-avatar.jpg")
		user.EmailVerified = true
		user.UpdateLastLogin()
		user.JWTSubject = dbStringPtr("updated_sub_123")

		err = testDB.DB.DB.Save(user).Error
		require.NoError(t, err, "OAuth user update should succeed")

		// Verify updates
		var updatedUser models.User
		err = testDB.DB.DB.First(&updatedUser, user.ID).Error
		require.NoError(t, err, "User retrieval should succeed")

		assert.Equal(t, "Updated OAuth User", *updatedUser.DisplayName, "DisplayName should be updated")
		assert.Equal(t, "https://example.com/new-avatar.jpg", *updatedUser.AvatarURL, "AvatarURL should be updated")
		assert.True(t, updatedUser.EmailVerified, "EmailVerified should be updated")
		assert.NotNil(t, updatedUser.LastLogin, "LastLogin should be set")
		assert.Equal(t, "updated_sub_123", *updatedUser.JWTSubject, "JWTSubject should be updated")
	})

	t.Run("Rate Limit Operations", func(t *testing.T) {
		// Create user
		user := &models.User{
			Email:       "ratelimit@example.com",
			KindeUserID: "kinde_ratelimit_123",
			Role:        "user",
		}
		err := testDB.DB.DB.Create(user).Error
		require.NoError(t, err, "User creation should succeed")

		// Test rate limit increment
		user.IncrementRateLimit()
		err = testDB.DB.DB.Save(user).Error
		require.NoError(t, err, "Rate limit increment should succeed")
		assert.Equal(t, 1, user.RateLimitCount, "Rate limit count should be incremented")

		// Test multiple increments
		for i := 0; i < 5; i++ {
			user.IncrementRateLimit()
		}
		err = testDB.DB.DB.Save(user).Error
		require.NoError(t, err, "Multiple rate limit increments should succeed")
		assert.Equal(t, 6, user.RateLimitCount, "Rate limit count should be 6")

		// Test rate limit reset
		user.ResetRateLimit()
		err = testDB.DB.DB.Save(user).Error
		require.NoError(t, err, "Rate limit reset should succeed")
		assert.Equal(t, 0, user.RateLimitCount, "Rate limit count should be reset")
		assert.NotNil(t, user.RateLimitReset, "Rate limit reset time should be set")
	})
}

// Helper functions for pointer types
func dbStringPtr(s string) *string {
	return &s
}

func dbIntPtr(i int) *int {
	return &i
}

func dbFloat64Ptr(f float64) *float64 {
	return &f
}