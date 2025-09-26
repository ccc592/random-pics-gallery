package image

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/randompic/api/internal/db/models"
	"github.com/randompic/api/internal/randomizer"
	"gorm.io/gorm"
)

// Service handles image-related business logic
type Service struct {
	db         *gorm.DB
	randomizer *randomizer.Service
	storage    StorageInterface
}

// StorageInterface defines the interface for image storage backends
type StorageInterface interface {
	Store(filename string, data []byte) (string, error)
	Retrieve(path string) ([]byte, error)
	Delete(path string) error
	GetURL(path string) string
	Exists(path string) bool
}

// NewService creates a new image service
func NewService(db *gorm.DB, randomizer *randomizer.Service, storage StorageInterface) *Service {
	return &Service{
		db:         db,
		randomizer: randomizer,
		storage:    storage,
	}
}

// CreateImage creates a new image record in the database
func (s *Service) CreateImage(req *models.ImageCreateRequest, uploadedBy *uuid.UUID) (*models.Image, error) {
	now := time.Now().UTC()

	image := &models.Image{
		ID:          uuid.New(),
		Filename:    req.Filename,
		Alt:         req.Alt,
		Title:       req.Title,
		Tags:        req.Tags,
		Weight:      req.Weight,
		StoragePath: req.StoragePath,
		MimeType:    req.MimeType,
		FileSize:    req.FileSize,
		Width:       req.Width,
		Height:      req.Height,
		UploadDate:  &now,
		UploadedBy:  uploadedBy,
		Status:      "processing", // Start in processing status
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Calculate aspect ratio if dimensions are provided
	if req.Width != nil && req.Height != nil && *req.Height != 0 {
		aspectRatio := float64(*req.Width) / float64(*req.Height)
		image.AspectRatio = &aspectRatio
	}

	if err := s.db.Create(image).Error; err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	return image, nil
}

// GetImageByID retrieves an image by its ID
func (s *Service) GetImageByID(id uuid.UUID) (*models.Image, error) {
	var image models.Image

	err := s.db.Where("id = ?", id).First(&image).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return &image, nil
}

// UpdateImage updates an existing image
func (s *Service) UpdateImage(id uuid.UUID, req *models.ImageUpdateRequest) (*models.Image, error) {
	var image models.Image

	// First, get the existing image
	err := s.db.Where("id = ?", id).First(&image).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image not found")
		}
		return nil, fmt.Errorf("failed to find image: %w", err)
	}

	// Update only provided fields
	updates := make(map[string]interface{})

	if req.Alt != nil {
		updates["alt"] = *req.Alt
	}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	updates["updated_at"] = time.Now().UTC()

	// Perform the update
	err = s.db.Model(&image).Updates(updates).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	// Return updated image
	return s.GetImageByID(id)
}

// DeleteImage deletes an image by its ID
func (s *Service) DeleteImage(id uuid.UUID) error {
	// First, get the image to check if it exists and get storage path
	image, err := s.GetImageByID(id)
	if err != nil {
		return err
	}

	// Delete from storage
	if err := s.storage.Delete(image.StoragePath); err != nil {
		// Log the error but don't fail the deletion if storage delete fails
		// This allows cleanup of orphaned database records
		fmt.Printf("Warning: failed to delete image from storage: %v\n", err)
	}

	// Delete from database
	err = s.db.Delete(&models.Image{}, "id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to delete image from database: %w", err)
	}

	return nil
}

// ListImages retrieves a paginated list of images
func (s *Service) ListImages(page, limit int, status string, tags string) (*models.ImageListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	var images []models.Image
	var total int64

	// Build query
	query := s.db.Model(&models.Image{})

	// Filter by status if provided
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by tags if provided (simple contains search)
	if tags != "" {
		query = query.Where("tags ILIKE ?", "%"+tags+"%")
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count images: %w", err)
	}

	// Get images
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	// Convert to response format
	imageResponses := make([]models.ImageResponse, len(images))
	for i, img := range images {
		imageResponses[i] = img.ToResponse()
	}

	pages := int((total + int64(limit) - 1) / int64(limit)) // Ceiling division

	return &models.ImageListResponse{
		Images: imageResponses,
		Total:  int(total),
		Page:   page,
		Limit:  limit,
		Pages:  pages,
	}, nil
}

// GetRandomImages retrieves random images using the randomizer service
func (s *Service) GetRandomImages(count int, seed string) (*models.RandomImagesResponse, error) {
	if count < 1 || count > 5 {
		return nil, fmt.Errorf("count must be between 1 and 5")
	}

	// Get all active images with their weights
	var images []models.Image
	err := s.db.Where("status = ?", "active").Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve images: %w", err)
	}

	if len(images) == 0 {
		return &models.RandomImagesResponse{
			Images:      []models.ImageResponse{},
			SessionSeed: seed,
			Count:       0,
		}, nil
	}

	// Convert images to weighted items for randomizer
	items := make([]randomizer.WeightedItem, len(images))
	for i, img := range images {
		items[i] = randomizer.WeightedItem{
			ID:     img.ID.String(),
			Weight: img.Weight,
			Data:   img, // Store the full image object
		}
	}

	// Generate session seed if not provided
	sessionSeed := seed
	if sessionSeed == "" {
		sessionSeed = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Use randomizer to select images
	selectedItems, err := s.randomizer.SelectWeightedRandom(items, count, sessionSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to select random images: %w", err)
	}

	// Convert selected items back to image responses
	selectedImages := make([]models.ImageResponse, len(selectedItems))
	for i, item := range selectedItems {
		img, ok := item.Data.(models.Image)
		if !ok {
			return nil, fmt.Errorf("invalid image data in selected item")
		}
		selectedImages[i] = img.ToResponse()
	}

	return &models.RandomImagesResponse{
		Images:      selectedImages,
		SessionSeed: sessionSeed,
		Count:       len(selectedImages),
	}, nil
}

// UpdateImageStatus updates the status of an image
func (s *Service) UpdateImageStatus(id uuid.UUID, status string) error {
	err := s.db.Model(&models.Image{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		return fmt.Errorf("failed to update image status: %w", err)
	}
	return nil
}

// GetImagesByUploader retrieves images uploaded by a specific user
func (s *Service) GetImagesByUploader(uploaderID uuid.UUID, page, limit int) (*models.ImageListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	var images []models.Image
	var total int64

	// Get total count
	err := s.db.Model(&models.Image{}).Where("uploaded_by = ?", uploaderID).Count(&total).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count user images: %w", err)
	}

	// Get images
	err = s.db.Where("uploaded_by = ?", uploaderID).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list user images: %w", err)
	}

	// Convert to response format
	imageResponses := make([]models.ImageResponse, len(images))
	for i, img := range images {
		imageResponses[i] = img.ToResponse()
	}

	pages := int((total + int64(limit) - 1) / int64(limit))

	return &models.ImageListResponse{
		Images: imageResponses,
		Total:  int(total),
		Page:   page,
		Limit:  limit,
		Pages:  pages,
	}, nil
}

// GetImageStats returns statistics about images
func (s *Service) GetImageStats() (map[string]interface{}, error) {
	var totalImages int64
	var activeImages int64
	var processingImages int64
	var failedImages int64

	// Get total count
	if err := s.db.Model(&models.Image{}).Count(&totalImages).Error; err != nil {
		return nil, fmt.Errorf("failed to count total images: %w", err)
	}

	// Get count by status
	if err := s.db.Model(&models.Image{}).Where("status = ?", "active").Count(&activeImages).Error; err != nil {
		return nil, fmt.Errorf("failed to count active images: %w", err)
	}

	if err := s.db.Model(&models.Image{}).Where("status = ?", "processing").Count(&processingImages).Error; err != nil {
		return nil, fmt.Errorf("failed to count processing images: %w", err)
	}

	if err := s.db.Model(&models.Image{}).Where("status = ?", "failed").Count(&failedImages).Error; err != nil {
		return nil, fmt.Errorf("failed to count failed images: %w", err)
	}

	return map[string]interface{}{
		"total_images":      totalImages,
		"active_images":     activeImages,
		"processing_images": processingImages,
		"failed_images":     failedImages,
		"inactive_images":   totalImages - activeImages - processingImages - failedImages,
	}, nil
}

// SearchImages searches images by various criteria
func (s *Service) SearchImages(query string, tags string, status string, page, limit int) (*models.ImageListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	var images []models.Image
	var total int64

	// Build search query
	dbQuery := s.db.Model(&models.Image{})

	if query != "" {
		// Search in alt text, title, and tags
		searchPattern := "%" + query + "%"
		dbQuery = dbQuery.Where("alt ILIKE ? OR title ILIKE ? OR tags ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if tags != "" {
		dbQuery = dbQuery.Where("tags ILIKE ?", "%"+tags+"%")
	}

	if status != "" {
		dbQuery = dbQuery.Where("status = ?", status)
	}

	// Get total count
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Get images
	err := dbQuery.Offset(offset).Limit(limit).Order("created_at DESC").Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to search images: %w", err)
	}

	// Convert to response format
	imageResponses := make([]models.ImageResponse, len(images))
	for i, img := range images {
		imageResponses[i] = img.ToResponse()
	}

	pages := int((total + int64(limit) - 1) / int64(limit))

	return &models.ImageListResponse{
		Images: imageResponses,
		Total:  int(total),
		Page:   page,
		Limit:  limit,
		Pages:  pages,
	}, nil
}