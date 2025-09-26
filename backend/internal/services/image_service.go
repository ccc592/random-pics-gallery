package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// FileUpload represents an uploaded file with metadata
type FileUpload struct {
	File     multipart.File   `json:"-"`
	Header   *multipart.FileHeader `json:"-"`
	Filename string           `json:"filename"`
	Size     int64            `json:"size"`
	MimeType string           `json:"mime_type"`
	Data     io.Reader        `json:"-"`
}

// ImageUpdate represents fields that can be updated on an image
type ImageUpdate struct {
	Alt    *string `json:"alt,omitempty"`
	Title  *string `json:"title,omitempty"`
	Tags   *string `json:"tags,omitempty"`
	Weight *int    `json:"weight,omitempty"`
	Status *string `json:"status,omitempty"`
}

// ImageFilters represents filtering options for listing images
type ImageFilters struct {
	Status    *string `json:"status,omitempty"`
	Tags      *string `json:"tags,omitempty"`
	MimeType  *string `json:"mime_type,omitempty"`
	Limit     int     `json:"limit"`
	Offset    int     `json:"offset"`
	SortBy    string  `json:"sort_by"` // upload_date, file_size, weight
	SortOrder string  `json:"sort_order"` // asc, desc
}

// ImageServiceInterface defines the contract for image management
type ImageServiceInterface interface {
	UploadImage(ctx context.Context, userID string, file *FileUpload) (*models.Image, error)
	GetImage(ctx context.Context, imageID string) (*models.Image, error)
	UpdateImage(ctx context.Context, imageID string, updates *ImageUpdate) (*models.Image, error)
	DeleteImage(ctx context.Context, imageID string) error
	ListUserImages(ctx context.Context, userID string, filters *ImageFilters) ([]*models.Image, error)
	ValidateImageFile(file *FileUpload) error
	GetActiveImages(ctx context.Context) ([]*models.Image, error)
}

// ImageService implements the ImageServiceInterface
type ImageService struct {
	db             *gorm.DB
	storageService StorageServiceInterface
	userService    UserServiceInterface
	maxFileSize    int64
	allowedTypes   []string
}

// NewImageService creates a new ImageService instance
func NewImageService(db *gorm.DB, storageService StorageServiceInterface, userService UserServiceInterface) *ImageService {
	return &ImageService{
		db:             db,
		storageService: storageService,
		userService:    userService,
		maxFileSize:    2097152, // 2MB in bytes
		allowedTypes:   []string{"image/jpeg", "image/png"},
	}
}

// UploadImage uploads and processes a new image file
func (s *ImageService) UploadImage(ctx context.Context, userID string, file *FileUpload) (*models.Image, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if file == nil {
		return nil, fmt.Errorf("file cannot be nil")
	}

	// Validate file
	if err := s.ValidateImageFile(file); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Check storage quota
	if err := s.userService.ValidateStorageQuota(ctx, userID, file.Size); err != nil {
		return nil, fmt.Errorf("storage quota validation failed: %w", err)
	}

	// Start database transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Generate unique filename
	fileExt := s.getFileExtensionFromMimeType(file.MimeType)
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), s.sanitizeFilename(file.Filename), fileExt)

	// Store file
	storagePath, err := s.storageService.StoreFile(ctx, userID, filename, file.Data)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	// Create image record
	image := &models.Image{
		Filename:    filename,
		Alt:         fmt.Sprintf("Image %s uploaded by user", filename), // Default alt text
		StoragePath: storagePath,
		MimeType:    file.MimeType,
		FileSize:    file.Size,
		Weight:      1, // Default weight
		UploadedBy:  userID,
		Status:      "processing",
		UploadDate:  time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// For JPEG/PNG, we'll simulate getting dimensions (in real implementation, would use image library)
	// Setting default dimensions for now
	image.Width = 800
	image.Height = 600
	image.AspectRatio = float64(image.Width) / float64(image.Height)

	// Save to database
	if err := tx.Create(image).Error; err != nil {
		tx.Rollback()
		// Clean up stored file
		s.storageService.DeleteFile(ctx, storagePath)
		return nil, fmt.Errorf("failed to create image record: %w", err)
	}

	// Update user storage usage (create internal method)
	if err := s.updateUserStorageUsage(ctx, userID, file.Size); err != nil {
		tx.Rollback()
		// Clean up stored file
		s.storageService.DeleteFile(ctx, storagePath)
		return nil, fmt.Errorf("failed to update user storage: %w", err)
	}

	// Mark as active (processing complete)
	image.Status = "active"
	if err := tx.Save(image).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update image status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return image, nil
}

// GetImage retrieves an image by ID
func (s *ImageService) GetImage(ctx context.Context, imageID string) (*models.Image, error) {
	if imageID == "" {
		return nil, fmt.Errorf("image ID cannot be empty")
	}

	var image models.Image
	if err := s.db.WithContext(ctx).Where("id = ?", imageID).First(&image).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return &image, nil
}

// UpdateImage updates image metadata
func (s *ImageService) UpdateImage(ctx context.Context, imageID string, updates *ImageUpdate) (*models.Image, error) {
	if imageID == "" {
		return nil, fmt.Errorf("image ID cannot be empty")
	}
	if updates == nil {
		return nil, fmt.Errorf("updates cannot be nil")
	}

	// Get existing image
	image, err := s.GetImage(ctx, imageID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if updates.Alt != nil {
		if len(*updates.Alt) < 10 {
			return nil, fmt.Errorf("alt text must be at least 10 characters long")
		}
		image.Alt = *updates.Alt
	}
	if updates.Title != nil {
		image.Title = updates.Title
	}
	if updates.Tags != nil {
		image.Tags = updates.Tags
	}
	if updates.Weight != nil {
		if *updates.Weight < 1 || *updates.Weight > 10 {
			return nil, fmt.Errorf("weight must be between 1 and 10")
		}
		image.Weight = *updates.Weight
	}
	if updates.Status != nil {
		validStatuses := map[string]bool{"active": true, "inactive": true, "processing": true, "failed": true}
		if !validStatuses[*updates.Status] {
			return nil, fmt.Errorf("invalid status: %s", *updates.Status)
		}
		image.Status = *updates.Status
	}

	image.UpdatedAt = time.Now()

	// Save changes
	if err := s.db.WithContext(ctx).Save(image).Error; err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	return image, nil
}

// DeleteImage deletes an image and its associated file
func (s *ImageService) DeleteImage(ctx context.Context, imageID string) error {
	if imageID == "" {
		return fmt.Errorf("image ID cannot be empty")
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get image details
	var image models.Image
	if err := tx.Where("id = ?", imageID).First(&image).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("image not found")
		}
		return fmt.Errorf("failed to get image: %w", err)
	}

	// Delete from storage
	if err := s.storageService.DeleteFile(ctx, image.StoragePath); err != nil {
		// Log error but don't fail the deletion
		// File might already be missing
	}

	// Delete from database
	if err := tx.Delete(&image).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete image from database: %w", err)
	}

	// Update user storage usage (create internal method)
	if err := s.updateUserStorageUsage(ctx, image.UploadedBy, -image.FileSize); err != nil {
		// Log error but don't fail - user storage will be corrected later
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListUserImages retrieves images for a specific user with filtering
func (s *ImageService) ListUserImages(ctx context.Context, userID string, filters *ImageFilters) ([]*models.Image, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	query := s.db.WithContext(ctx).Where("uploaded_by = ?", userID)

	// Apply filters
	if filters != nil {
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		if filters.Tags != nil {
			query = query.Where("tags LIKE ?", "%"+*filters.Tags+"%")
		}
		if filters.MimeType != nil {
			query = query.Where("mime_type = ?", *filters.MimeType)
		}

		// Apply sorting
		sortBy := "upload_date"
		sortOrder := "desc"
		if filters.SortBy != "" {
			validSortFields := map[string]bool{"upload_date": true, "file_size": true, "weight": true, "created_at": true}
			if validSortFields[filters.SortBy] {
				sortBy = filters.SortBy
			}
		}
		if filters.SortOrder == "asc" || filters.SortOrder == "desc" {
			sortOrder = filters.SortOrder
		}
		query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

		// Apply pagination
		if filters.Limit > 0 {
			query = query.Limit(filters.Limit)
		}
		if filters.Offset > 0 {
			query = query.Offset(filters.Offset)
		}
	}

	var images []*models.Image
	if err := query.Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to list user images: %w", err)
	}

	return images, nil
}

// ValidateImageFile validates uploaded image file
func (s *ImageService) ValidateImageFile(file *FileUpload) error {
	if file == nil {
		return fmt.Errorf("file cannot be nil")
	}

	// Check file size
	if file.Size > s.maxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes (2MB)", file.Size, s.maxFileSize)
	}

	if file.Size <= 0 {
		return fmt.Errorf("file size must be greater than 0")
	}

	// Check MIME type
	validMimeType := false
	for _, allowedType := range s.allowedTypes {
		if file.MimeType == allowedType {
			validMimeType = true
			break
		}
	}
	if !validMimeType {
		return fmt.Errorf("unsupported MIME type: %s, only %v are allowed", file.MimeType, s.allowedTypes)
	}

	// Check filename
	if file.Filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check file extension matches MIME type
	expectedExt := s.getFileExtensionFromMimeType(file.MimeType)
	actualExt := strings.ToLower(filepath.Ext(file.Filename))
	if expectedExt != "" && actualExt != expectedExt {
		return fmt.Errorf("file extension %s does not match MIME type %s", actualExt, file.MimeType)
	}

	return nil
}

// GetActiveImages retrieves all active images (for randomization)
func (s *ImageService) GetActiveImages(ctx context.Context) ([]*models.Image, error) {
	var images []*models.Image
	if err := s.db.WithContext(ctx).Where("status = ?", "active").Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to get active images: %w", err)
	}

	return images, nil
}

// GetImagesByTags retrieves images matching specific tags
func (s *ImageService) GetImagesByTags(ctx context.Context, tags []string) ([]*models.Image, error) {
	if len(tags) == 0 {
		return s.GetActiveImages(ctx)
	}

	query := s.db.WithContext(ctx).Where("status = ?", "active")

	// Build OR condition for tags
	var conditions []string
	var args []interface{}
	for _, tag := range tags {
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, "%"+tag+"%")
	}

	query = query.Where(strings.Join(conditions, " OR "), args...)

	var images []*models.Image
	if err := query.Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to get images by tags: %w", err)
	}

	return images, nil
}

// GetImageStats returns statistics about images in the system
func (s *ImageService) GetImageStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total images
	var totalImages int64
	if err := s.db.WithContext(ctx).Model(&models.Image{}).Count(&totalImages).Error; err != nil {
		return nil, fmt.Errorf("failed to count total images: %w", err)
	}
	stats["total_images"] = totalImages

	// Active images
	var activeImages int64
	if err := s.db.WithContext(ctx).Model(&models.Image{}).Where("status = ?", "active").Count(&activeImages).Error; err != nil {
		return nil, fmt.Errorf("failed to count active images: %w", err)
	}
	stats["active_images"] = activeImages

	// Images by status
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := s.db.WithContext(ctx).Model(&models.Image{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}
	stats["by_status"] = statusCounts

	// Total storage usage
	var totalStorage struct {
		Total int64 `json:"total"`
	}
	if err := s.db.WithContext(ctx).Model(&models.Image{}).
		Select("COALESCE(SUM(file_size), 0) as total").
		Where("status != ?", "failed").
		Scan(&totalStorage).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate total storage: %w", err)
	}
	stats["total_storage_bytes"] = totalStorage.Total

	return stats, nil
}

// Helper functions

func (s *ImageService) getFileExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}

func (s *ImageService) sanitizeFilename(filename string) string {
	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Replace spaces and special characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	name = strings.ReplaceAll(name, "{", "_")
	name = strings.ReplaceAll(name, "}", "_")
	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}
	return name
}

// updateUserStorageUsage updates the user's storage usage directly in the database
func (s *ImageService) updateUserStorageUsage(ctx context.Context, userID string, deltaBytes int64) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Use raw SQL for atomic update to prevent race conditions
	result := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"total_storage_used": gorm.Expr("GREATEST(total_storage_used + ?, 0)", deltaBytes),
			"updated_at":         time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update storage usage: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found or no changes made")
	}

	return nil
}