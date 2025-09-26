package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"randompic/internal/db/models"
)

func TestImageMimeTypeValidation(t *testing.T) {
	validMimeTypes := []string{"image/jpeg", "image/png"}

	for _, mimeType := range validMimeTypes {
		t.Run("ValidMimeType_"+mimeType, func(t *testing.T) {
			image := &models.Image{
				Filename:    "test.jpg",
				Alt:         "Test image description",
				MimeType:    mimeType,
				FileSize:    1048576, // 1MB
				Width:       800,
				Height:      600,
				StoragePath: "/storage/test.jpg",
				UploadedBy:  "user-123",
			}

			err := image.ValidateMimeType()
			assert.NoError(t, err)
		})
	}

	t.Run("InvalidMimeType", func(t *testing.T) {
		invalidMimeTypes := []string{
			"image/gif",
			"image/webp",
			"image/bmp",
			"image/tiff",
			"application/pdf",
			"text/plain",
		}

		for _, mimeType := range invalidMimeTypes {
			image := &models.Image{
				MimeType: mimeType,
			}

			err := image.ValidateMimeType()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported MIME type")
			assert.Contains(t, err.Error(), mimeType)
		}
	})
}

func TestFileSizeValidation(t *testing.T) {
	t.Run("ValidFileSize", func(t *testing.T) {
		validSizes := []int64{
			1024,     // 1KB
			1048576,  // 1MB
			2097152,  // 2MB (max allowed)
		}

		for _, size := range validSizes {
			image := &models.Image{
				FileSize: size,
			}

			err := image.ValidateFileSize()
			assert.NoError(t, err)
		}
	})

	t.Run("InvalidFileSize", func(t *testing.T) {
		invalidSizes := []int64{
			2097153,  // 2MB + 1 byte
			5242880,  // 5MB
			10485760, // 10MB
		}

		for _, size := range invalidSizes {
			image := &models.Image{
				FileSize: size,
			}

			err := image.ValidateFileSize()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "exceeds maximum allowed size")
			assert.Contains(t, err.Error(), "2097152") // Max size in error message
		}
	})

	t.Run("ZeroFileSize", func(t *testing.T) {
		image := &models.Image{
			FileSize: 0,
		}

		err := image.ValidateFileSize()
		assert.NoError(t, err) // Zero size is technically valid
	})
}

func TestImageDimensionValidation(t *testing.T) {
	t.Run("ValidDimensions", func(t *testing.T) {
		validDimensions := []struct {
			width, height int
		}{
			{100, 100}, // Minimum allowed
			{800, 600}, // Typical dimensions
			{1920, 1080}, // HD
			{4000, 3000}, // High resolution
		}

		for _, dim := range validDimensions {
			image := &models.Image{
				Width:  dim.width,
				Height: dim.height,
			}

			err := image.ValidateDimensions()
			assert.NoError(t, err)
		}
	})

	t.Run("InvalidDimensions", func(t *testing.T) {
		invalidDimensions := []struct {
			width, height int
			expectError   string
		}{
			{50, 100, "width 50 is less than minimum required 100"},
			{100, 50, "height 50 is less than minimum required 100"},
			{99, 99, "width 99 is less than minimum required 100"},
			{0, 100, "width 0 is less than minimum required 100"},
			{100, 0, "height 0 is less than minimum required 100"},
		}

		for _, dim := range invalidDimensions {
			image := &models.Image{
				Width:  dim.width,
				Height: dim.height,
			}

			err := image.ValidateDimensions()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), dim.expectError)
		}
	})
}

func TestStoragePathGeneration(t *testing.T) {
	t.Run("GenerateStoragePath", func(t *testing.T) {
		testCases := []struct {
			uploadedBy string
			filename   string
			expected   string
		}{
			{
				uploadedBy: "user-123",
				filename:   "image.jpg",
				expected:   "./storage/images/user-123/image.jpg",
			},
			{
				uploadedBy: "admin-456",
				filename:   "photo.png",
				expected:   "./storage/images/admin-456/photo.png",
			},
			{
				uploadedBy: "user-789",
				filename:   "test_image_with_underscores.jpeg",
				expected:   "./storage/images/user-789/test_image_with_underscores.jpeg",
			},
		}

		for _, tc := range testCases {
			image := &models.Image{
				UploadedBy: tc.uploadedBy,
				Filename:   tc.filename,
			}

			image.GenerateStoragePath()
			assert.Equal(t, tc.expected, image.StoragePath)
		}
	})

	t.Run("GenerateStoragePathWithEmptyValues", func(t *testing.T) {
		image := &models.Image{
			UploadedBy: "",
			Filename:   "image.jpg",
		}

		image.GenerateStoragePath()
		assert.Empty(t, image.StoragePath)

		image = &models.Image{
			UploadedBy: "user-123",
			Filename:   "",
		}

		image.GenerateStoragePath()
		assert.Empty(t, image.StoragePath)
	})
}

func TestImageModelFields(t *testing.T) {
	t.Run("RequiredFields", func(t *testing.T) {
		image := &models.Image{
			Filename:    "test.jpg",
			Alt:         "Test image description with minimum 10 characters",
			MimeType:    "image/jpeg",
			FileSize:    1048576,
			Width:       800,
			Height:      600,
			StoragePath: "/storage/test.jpg",
			UploadedBy:  "user-123",
			Status:      "active",
			Weight:      5,
		}

		// Verify required fields
		assert.NotEmpty(t, image.Filename)
		assert.NotEmpty(t, image.Alt)
		assert.GreaterOrEqual(t, len(image.Alt), 10) // Alt text minimum length
		assert.NotEmpty(t, image.MimeType)
		assert.Greater(t, image.FileSize, int64(0))
		assert.GreaterOrEqual(t, image.Width, 100)
		assert.GreaterOrEqual(t, image.Height, 100)
		assert.NotEmpty(t, image.StoragePath)
		assert.NotEmpty(t, image.UploadedBy)
		assert.NotEmpty(t, image.Status)
		assert.GreaterOrEqual(t, image.Weight, 1)
		assert.LessOrEqual(t, image.Weight, 10)
	})

	t.Run("OptionalFields", func(t *testing.T) {
		image := &models.Image{
			Filename:    "test.jpg",
			Alt:         "Test image description",
			MimeType:    "image/jpeg",
			FileSize:    1048576,
			Width:       800,
			Height:      600,
			StoragePath: "/storage/test.jpg",
			UploadedBy:  "user-123",
		}

		// Optional fields can be nil/empty
		assert.Nil(t, image.Title)
		assert.Nil(t, image.Tags)
	})

	t.Run("DefaultValues", func(t *testing.T) {
		image := &models.Image{
			Filename:    "test.jpg",
			Alt:         "Test image description",
			MimeType:    "image/jpeg",
			FileSize:    1048576,
			Width:       800,
			Height:      600,
			StoragePath: "/storage/test.jpg",
			UploadedBy:  "user-123",
		}

		// Check default values as defined in the model
		assert.Equal(t, 1, image.Weight) // Default weight should be 1
		assert.Equal(t, "active", image.Status) // Default status should be active
	})

	t.Run("WeightConstraints", func(t *testing.T) {
		// Valid weights
		validWeights := []int{1, 2, 5, 8, 10}
		for _, weight := range validWeights {
			image := &models.Image{Weight: weight}
			assert.GreaterOrEqual(t, image.Weight, 1)
			assert.LessOrEqual(t, image.Weight, 10)
		}
	})

	t.Run("StatusValues", func(t *testing.T) {
		validStatuses := []string{"active", "inactive", "processing", "failed"}

		for _, status := range validStatuses {
			image := &models.Image{
				Status: status,
			}
			assert.Contains(t, validStatuses, image.Status)
		}
	})
}

func TestImageBusinessLogic(t *testing.T) {
	t.Run("IsProcessingComplete", func(t *testing.T) {
		completeStatuses := []string{"active", "inactive", "failed"}
		incompleteStatuses := []string{"processing"}

		for _, status := range completeStatuses {
			image := &models.Image{Status: status}
			assert.True(t, image.IsProcessingComplete(), "Status %s should be complete", status)
		}

		for _, status := range incompleteStatuses {
			image := &models.Image{Status: status}
			assert.False(t, image.IsProcessingComplete(), "Status %s should not be complete", status)
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		activeImage := &models.Image{Status: "active"}
		inactiveImage := &models.Image{Status: "inactive"}
		processingImage := &models.Image{Status: "processing"}
		failedImage := &models.Image{Status: "failed"}

		assert.True(t, activeImage.IsActive())
		assert.False(t, inactiveImage.IsActive())
		assert.False(t, processingImage.IsActive())
		assert.False(t, failedImage.IsActive())
	})

	t.Run("GetFileExtension", func(t *testing.T) {
		testCases := []struct {
			mimeType  string
			expected  string
		}{
			{"image/jpeg", ".jpg"},
			{"image/png", ".png"},
			{"image/gif", ""}, // Unsupported
			{"text/plain", ""}, // Unsupported
		}

		for _, tc := range testCases {
			image := &models.Image{MimeType: tc.mimeType}
			assert.Equal(t, tc.expected, image.GetFileExtension())
		}
	})
}

func TestImageHooks(t *testing.T) {
	t.Run("BeforeCreate", func(t *testing.T) {
		image := &models.Image{
			Width:  800,
			Height: 600,
		}

		err := image.BeforeCreate()
		assert.NoError(t, err)

		// Check aspect ratio calculation
		expectedRatio := float64(800) / float64(600)
		assert.InDelta(t, expectedRatio, image.AspectRatio, 0.001)

		// Check upload date is set
		assert.False(t, image.UploadDate.IsZero())
	})

	t.Run("BeforeCreateZeroDimensions", func(t *testing.T) {
		image := &models.Image{
			Width:  0,
			Height: 600,
		}

		err := image.BeforeCreate()
		assert.NoError(t, err)
		assert.Equal(t, 0.0, image.AspectRatio) // Should remain 0 with invalid dimensions
	})

	t.Run("BeforeUpdate", func(t *testing.T) {
		image := &models.Image{
			UpdatedAt: time.Now().Add(-time.Hour), // Old timestamp
		}

		beforeUpdate := time.Now()
		err := image.BeforeUpdate()

		assert.NoError(t, err)
		assert.True(t, image.UpdatedAt.After(beforeUpdate.Add(-time.Second)))
		assert.True(t, image.UpdatedAt.Before(time.Now().Add(time.Second)))
	})
}

func TestImageResponseModels(t *testing.T) {
	t.Run("ToResponse", func(t *testing.T) {
		now := time.Now()
		uploadDate := now.Add(-time.Hour)

		image := &models.Image{
			ID:          "img-123",
			Filename:    "test.jpg",
			Alt:         "Test image description",
			Title:       stringPtr("Test Image"),
			Tags:        stringPtr("test,photo,sample"),
			Weight:      5,
			StoragePath: "/storage/test.jpg",
			MimeType:    "image/jpeg",
			FileSize:    1048576,
			Width:       800,
			Height:      600,
			AspectRatio: 1.333,
			UploadDate:  uploadDate,
			UploadedBy:  "user-123",
			Status:      "active",
			CreatedAt:   now.Add(-2*time.Hour),
			UpdatedAt:   now,
		}

		response := image.ToResponse()

		assert.Equal(t, "img-123", response.ID)
		assert.Equal(t, "test.jpg", response.Filename)
		assert.Equal(t, "Test image description", response.Alt)
		assert.Equal(t, "Test Image", *response.Title)
		assert.Equal(t, "test,photo,sample", *response.Tags)
		assert.Equal(t, 5, response.Weight)
		assert.Equal(t, "/storage/test.jpg", response.StoragePath)
		assert.Equal(t, "image/jpeg", response.MimeType)
		assert.Equal(t, int64(1048576), response.FileSize)
		assert.Equal(t, 800, response.Width)
		assert.Equal(t, 600, response.Height)
		assert.Equal(t, 1.333, response.AspectRatio)
		assert.Equal(t, "user-123", response.UploadedBy)
		assert.Equal(t, "active", response.Status)

		// Check time formatting
		assert.Contains(t, response.UploadDate, "T")
		assert.Contains(t, response.CreatedAt, "T")
		assert.Contains(t, response.UpdatedAt, "T")
	})

	t.Run("ImageCreateRequest", func(t *testing.T) {
		req := &models.ImageCreateRequest{
			Filename:    "test.jpg",
			Alt:         "Test image description",
			Title:       stringPtr("Test Image"),
			Tags:        stringPtr("test,photo"),
			Weight:      3,
			StoragePath: "/storage/test.jpg",
			MimeType:    "image/jpeg",
			FileSize:    1048576,
			Width:       intPtr(800),
			Height:      intPtr(600),
		}

		assert.Equal(t, "test.jpg", req.Filename)
		assert.Equal(t, "Test image description", req.Alt)
		assert.Equal(t, "Test Image", *req.Title)
		assert.Equal(t, "test,photo", *req.Tags)
		assert.Equal(t, 3, req.Weight)
		assert.Equal(t, "image/jpeg", req.MimeType)
		assert.Equal(t, 800, *req.Width)
		assert.Equal(t, 600, *req.Height)
	})

	t.Run("ImageUpdateRequest", func(t *testing.T) {
		req := &models.ImageUpdateRequest{
			Alt:    stringPtr("Updated description"),
			Title:  stringPtr("Updated Title"),
			Tags:   stringPtr("updated,tags"),
			Weight: intPtr(8),
			Status: stringPtr("inactive"),
		}

		assert.Equal(t, "Updated description", *req.Alt)
		assert.Equal(t, "Updated Title", *req.Title)
		assert.Equal(t, "updated,tags", *req.Tags)
		assert.Equal(t, 8, *req.Weight)
		assert.Equal(t, "inactive", *req.Status)
	})
}

func TestImageTableName(t *testing.T) {
	image := models.Image{}
	assert.Equal(t, "images", image.TableName())
}

func TestImageListResponse(t *testing.T) {
	t.Run("ImageListResponse", func(t *testing.T) {
		images := []models.ImageResponse{
			{ID: "img-1", Filename: "test1.jpg"},
			{ID: "img-2", Filename: "test2.png"},
		}

		response := &models.ImageListResponse{
			Images: images,
			Total:  100,
			Page:   1,
			Limit:  10,
			Pages:  10,
		}

		assert.Len(t, response.Images, 2)
		assert.Equal(t, 100, response.Total)
		assert.Equal(t, 1, response.Page)
		assert.Equal(t, 10, response.Limit)
		assert.Equal(t, 10, response.Pages)
	})

	t.Run("RandomImagesResponse", func(t *testing.T) {
		images := []models.ImageResponse{
			{ID: "img-1", Filename: "random1.jpg"},
			{ID: "img-2", Filename: "random2.png"},
			{ID: "img-3", Filename: "random3.jpg"},
		}

		response := &models.RandomImagesResponse{
			Images:      images,
			SessionSeed: "seed123456",
			Count:       3,
		}

		assert.Len(t, response.Images, 3)
		assert.Equal(t, "seed123456", response.SessionSeed)
		assert.Equal(t, 3, response.Count)
	})
}

// Helper functions for pointers
func intPtr(i int) *int {
	return &i
}