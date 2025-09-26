package models

import (
	"time"

	"github.com/google/uuid"
)

// Image represents the images table structure
type Image struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Filename       string     `gorm:"type:varchar(255);not null" json:"filename" validate:"required"`
	Alt            string     `gorm:"type:text;not null;check:length(alt) >= 10" json:"alt" validate:"required,min=10"`
	Title          *string    `gorm:"type:varchar(255)" json:"title,omitempty"`
	Tags           *string    `gorm:"type:text" json:"tags,omitempty"`
	Weight         int        `gorm:"type:integer;default:1;check:weight >= 1 AND weight <= 10" json:"weight" validate:"min=1,max=10"`
	StoragePath    string     `gorm:"type:varchar(500);not null" json:"storage_path" validate:"required"`
	MimeType       string     `gorm:"type:varchar(50);not null" json:"mime_type" validate:"required"`
	FileSize       int64      `gorm:"type:bigint" json:"file_size"`
	Width          *int       `gorm:"type:integer;check:width >= 100" json:"width,omitempty"`
	Height         *int       `gorm:"type:integer;check:height >= 100" json:"height,omitempty"`
	AspectRatio    *float64   `gorm:"type:decimal(5,3)" json:"aspect_ratio,omitempty"`
	DominantColors *string    `gorm:"type:text" json:"dominant_colors,omitempty"` // JSON array as string
	UploadDate     *time.Time `gorm:"type:timestamp with time zone" json:"upload_date,omitempty"`
	UploadedBy     *uuid.UUID `gorm:"type:uuid;foreignKey:UploadedBy;references:ID" json:"uploaded_by,omitempty"`
	Status         string     `gorm:"type:varchar(20);default:'active';check:status IN ('active', 'inactive', 'processing', 'failed')" json:"status" validate:"oneof=active inactive processing failed"`
	CreatedAt      time.Time  `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"type:timestamp with time zone;default:now()" json:"updated_at"`

	// Foreign key relationship
	UploadedByUser *User `gorm:"foreignKey:UploadedBy" json:"uploaded_by_user,omitempty"`
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
	ID             uuid.UUID `json:"id"`
	Filename       string    `json:"filename"`
	Alt            string    `json:"alt"`
	Title          *string   `json:"title,omitempty"`
	Tags           *string   `json:"tags,omitempty"`
	Weight         int       `json:"weight"`
	StoragePath    string    `json:"storage_path"`
	MimeType       string    `json:"mime_type"`
	FileSize       int64     `json:"file_size"`
	Width          *int      `json:"width,omitempty"`
	Height         *int      `json:"height,omitempty"`
	AspectRatio    *float64  `json:"aspect_ratio,omitempty"`
	DominantColors []string  `json:"dominant_colors,omitempty"` // Parsed from JSON string
	UploadDate     *string   `json:"upload_date,omitempty"`     // Formatted as ISO 8601
	UploadedBy     *string   `json:"uploaded_by,omitempty"`     // UUID as string
	Status         string    `json:"status"`
	CreatedAt      string    `json:"created_at"` // Formatted as ISO 8601
	UpdatedAt      string    `json:"updated_at"` // Formatted as ISO 8601
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
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
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
		Status:      i.Status,
		CreatedAt:   i.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   i.UpdatedAt.Format(time.RFC3339),
	}

	// Format upload date if present
	if i.UploadDate != nil {
		uploadDate := i.UploadDate.Format(time.RFC3339)
		response.UploadDate = &uploadDate
	}

	// Format uploaded by if present
	if i.UploadedBy != nil {
		uploadedBy := i.UploadedBy.String()
		response.UploadedBy = &uploadedBy
	}

	// Parse dominant colors JSON if present
	if i.DominantColors != nil && *i.DominantColors != "" {
		// This would be parsed from JSON in a real implementation
		// For now, return empty slice
		response.DominantColors = []string{}
	}

	return response
}