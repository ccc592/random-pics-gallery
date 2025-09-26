package models

import (
	"fmt"
	"path/filepath"
	"time"
)

// Image represents the images table structure
type Image struct {
	ID           string     `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Filename     string     `gorm:"type:varchar(255);not null" json:"filename" validate:"required"`
	Alt          string     `gorm:"type:text;not null;check:length(alt) >= 10" json:"alt" validate:"required,min=10"`
	Title        *string    `gorm:"type:varchar(255)" json:"title,omitempty"`
	Tags         *string    `gorm:"type:text;index:idx_images_tags_gin,type:gin" json:"tags,omitempty"`
	Weight       int        `gorm:"not null;default:1;check:weight >= 1 AND weight <= 10" json:"weight" validate:"min=1,max=10"`
	StoragePath  string     `gorm:"type:varchar(500);not null" json:"storage_path" validate:"required"`
	MimeType     string     `gorm:"type:varchar(50);not null;check:mime_type IN ('image/jpeg','image/png')" json:"mime_type" validate:"required,oneof=image/jpeg image/png"`
	FileSize     int64      `gorm:"not null;check:file_size <= 2097152" json:"file_size"` // 2MB
	Width        int        `gorm:"not null;check:width >= 100" json:"width" validate:"min=100"`
	Height       int        `gorm:"not null;check:height >= 100" json:"height" validate:"min=100"`
	AspectRatio  float64    `gorm:"type:decimal(5,3)" json:"aspect_ratio"`
	UploadDate   time.Time  `gorm:"not null" json:"upload_date"`
	UploadedBy   string     `gorm:"type:uuid;not null" json:"uploaded_by"`
	Status       string     `gorm:"type:varchar(50);not null;default:'active';check:status IN ('active','inactive','processing','failed')" json:"status" validate:"oneof=active inactive processing failed"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UploadedBy;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// TableName specifies the table name for GORM
func (Image) TableName() string {
	return "images"
}

// ImageCreateRequest represents the request payload for creating an image
type ImageCreateRequest struct {
	Filename    string  `json:"filename" validate:"required"`
	Alt         string  `json:"alt" validate:"required,min=10"`
	Title       *string `json:"title,omitempty"`
	Tags        *string `json:"tags,omitempty"`
	Weight      int     `json:"weight" validate:"min=1,max=10"`
	StoragePath string  `json:"storage_path" validate:"required"`
	MimeType    string  `json:"mime_type" validate:"required"`
	FileSize    int64   `json:"file_size"`
	Width       *int    `json:"width,omitempty"`
	Height      *int    `json:"height,omitempty"`
}

// ImageUpdateRequest represents the request payload for updating an image
type ImageUpdateRequest struct {
	Alt    *string `json:"alt,omitempty" validate:"omitempty,min=10"`
	Title  *string `json:"title,omitempty"`
	Tags   *string `json:"tags,omitempty"`
	Weight *int    `json:"weight,omitempty" validate:"omitempty,min=1,max=10"`
	Status *string `json:"status,omitempty" validate:"omitempty,oneof=active inactive processing failed"`
}

// ImageResponse represents the response payload for image operations
type ImageResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Alt         string    `json:"alt"`
	Title       *string   `json:"title,omitempty"`
	Tags        *string   `json:"tags,omitempty"`
	Weight      int       `json:"weight"`
	StoragePath string    `json:"storage_path"`
	MimeType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	AspectRatio float64   `json:"aspect_ratio"`
	UploadDate  string    `json:"upload_date"`  // Formatted as ISO 8601
	UploadedBy  string    `json:"uploaded_by"`  // User UUID
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"` // Formatted as ISO 8601
	UpdatedAt   string    `json:"updated_at"` // Formatted as ISO 8601
}

// RandomImagesResponse represents the response for random images endpoint
type RandomImagesResponse struct {
	Images      []ImageResponse `json:"images"`
	SessionSeed string          `json:"session_seed"`
	Count       int             `json:"count"`
}

// ImageListResponse represents the response for paginated image list
type ImageListResponse struct {
	Images []ImageResponse `json:"images"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
	Pages  int             `json:"pages"`
}

// BeforeCreate is a GORM hook that runs before creating a record
func (i *Image) BeforeCreate() error {
	// GORM will handle UUID generation via default:uuid_generate_v4()
	// Calculate aspect ratio if not set
	if i.Width > 0 && i.Height > 0 {
		i.AspectRatio = float64(i.Width) / float64(i.Height)
	}
	// Set upload date if not set
	if i.UploadDate.IsZero() {
		i.UploadDate = time.Now()
	}
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a record
func (i *Image) BeforeUpdate() error {
	i.UpdatedAt = time.Now()
	return nil
}

// ToResponse converts Image model to ImageResponse
func (i *Image) ToResponse() ImageResponse {
	response := ImageResponse{
		ID:          i.ID,
		Filename:    i.Filename,
		Alt:         i.Alt,
		Title:       i.Title,
		Tags:        i.Tags,
		Weight:      i.Weight,
		StoragePath: i.StoragePath,
		MimeType:    i.MimeType,
		FileSize:    i.FileSize,
		Width:       i.Width,
		Height:      i.Height,
		AspectRatio: i.AspectRatio,
		UploadDate:  i.UploadDate.Format(time.RFC3339),
		UploadedBy:  i.UploadedBy,
		Status:      i.Status,
		CreatedAt:   i.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   i.UpdatedAt.Format(time.RFC3339),
	}

	return response
}

// ValidateMimeType checks if the MIME type is supported
func (i *Image) ValidateMimeType() error {
	if i.MimeType != "image/jpeg" && i.MimeType != "image/png" {
		return fmt.Errorf("unsupported MIME type: %s, only image/jpeg and image/png are allowed", i.MimeType)
	}
	return nil
}

// ValidateFileSize checks if the file size is within limits
func (i *Image) ValidateFileSize() error {
	const maxFileSize = 2097152 // 2MB
	if i.FileSize > maxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes (2MB)", i.FileSize, maxFileSize)
	}
	return nil
}

// ValidateDimensions checks if the image dimensions meet requirements
func (i *Image) ValidateDimensions() error {
	const minDimension = 100
	if i.Width < minDimension {
		return fmt.Errorf("image width %d is less than minimum required %d pixels", i.Width, minDimension)
	}
	if i.Height < minDimension {
		return fmt.Errorf("image height %d is less than minimum required %d pixels", i.Height, minDimension)
	}
	return nil
}

// GenerateStoragePath creates the storage path based on user ID and filename
func (i *Image) GenerateStoragePath() {
	if i.UploadedBy != "" && i.Filename != "" {
		i.StoragePath = filepath.Join("./storage/images", i.UploadedBy, i.Filename)
	}
}

// IsProcessingComplete checks if the image has finished processing
func (i *Image) IsProcessingComplete() bool {
	return i.Status == "active" || i.Status == "inactive" || i.Status == "failed"
}

// IsActive checks if the image is active and ready for selection
func (i *Image) IsActive() bool {
	return i.Status == "active"
}

// GetFileExtension returns the file extension based on MIME type
func (i *Image) GetFileExtension() string {
	switch i.MimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}