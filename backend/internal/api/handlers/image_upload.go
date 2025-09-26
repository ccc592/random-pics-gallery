package handlers

import (
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/db/models"
	imageService "github.com/randompic/api/internal/image"
	"github.com/randompic/api/internal/middleware"
	"gorm.io/gorm"

	// Import image decoders for validation
	_ "image/jpeg"
	_ "image/png"
)

// ImageUploadHandler handles image upload requests
type ImageUploadHandler struct {
	db           *gorm.DB
	imageService *imageService.Service
	authService  *auth.Service
}

// NewImageUploadHandler creates a new image upload handler
func NewImageUploadHandler(db *gorm.DB, imgService *imageService.Service, authService *auth.Service) *ImageUploadHandler {
	return &ImageUploadHandler{
		db:           db,
		imageService: imgService,
		authService:  authService,
	}
}

// ImageUploadRequest represents the multipart form data for image upload
type ImageUploadRequest struct {
	File   *multipart.FileHeader `form:"file" binding:"required"`
	Alt    string                `form:"alt" binding:"required,min=10,max=500"`
	Title  string                `form:"title" binding:"max=255"`
	Tags   string                `form:"tags" binding:"max=200"`
	Weight int                   `form:"weight" binding:"min=1,max=10"`
}

// ImageUploadResponse represents the response after successful upload
type ImageUploadResponse struct {
	ID          string   `json:"id"`
	Filename    string   `json:"filename"`
	Alt         string   `json:"alt"`
	Title       *string  `json:"title,omitempty"`
	Tags        []string `json:"tags"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	AspectRatio float64  `json:"aspect_ratio"`
	Weight      int      `json:"weight"`
	UploadDate  string   `json:"upload_date"`
	StoragePath string   `json:"storage_path"`
}

// Using common error response types from common.go

// createErrorResponse creates a standardized error response
func (h *ImageUploadHandler) createErrorResponse(code int, message, details string) ErrorResponse {
	return CreateErrorResponse(code, message, details)
}

// createValidationErrorResponse creates an error response with validation errors
func (h *ImageUploadHandler) createValidationErrorResponse(message string, validations []ValidationError) ErrorResponse {
	return CreateValidationErrorResponse(message, validations)
}

// validateImageFile validates the uploaded image file
func (h *ImageUploadHandler) validateImageFile(fileHeader *multipart.FileHeader) (int, int, []ValidationError) {
	var validationErrors []ValidationError
	var width, height int

	// Check file size (2MB limit)
	const maxFileSize = 2 * 1024 * 1024 // 2MB
	if fileHeader.Size > maxFileSize {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   fmt.Sprintf("%d bytes", fileHeader.Size),
			Message: fmt.Sprintf("File size exceeds maximum limit of 2MB (current: %.2f MB)", float64(fileHeader.Size)/(1024*1024)),
		})
		return 0, 0, validationErrors
	}

	// Check file extension and MIME type
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   ext,
			Message: "Only JPEG and PNG image formats are supported",
		})
		return 0, 0, validationErrors
	}

	// Open and validate the actual image
	file, err := fileHeader.Open()
	if err != nil {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   fileHeader.Filename,
			Message: "Unable to open uploaded file",
		})
		return 0, 0, validationErrors
	}
	defer file.Close()

	// Decode image to get dimensions and validate format
	img, format, err := image.DecodeConfig(file)
	if err != nil {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   fileHeader.Filename,
			Message: "Invalid image format or corrupted file",
		})
		return 0, 0, validationErrors
	}

	// Validate image format
	if format != "jpeg" && format != "png" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   format,
			Message: "Only JPEG and PNG formats are supported",
		})
		return 0, 0, validationErrors
	}

	// Check minimum dimensions
	width, height = img.Width, img.Height
	const minDimension = 100
	if width < minDimension || height < minDimension {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   fmt.Sprintf("%dx%d", width, height),
			Message: fmt.Sprintf("Image dimensions must be at least %dx%d pixels", minDimension, minDimension),
		})
	}

	return width, height, validationErrors
}

// validateUploadRequest validates the complete upload request
func (h *ImageUploadHandler) validateUploadRequest(c *gin.Context) (*ImageUploadRequest, []ValidationError) {
	var validationErrors []ValidationError

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(2 * 1024 * 1024); err != nil { // 2MB limit
		validationErrors = append(validationErrors, ValidationError{
			Field:   "request",
			Value:   "",
			Message: "Failed to parse multipart form data",
		})
		return nil, validationErrors
	}

	// Get file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "file",
			Value:   "",
			Message: "File is required",
		})
		return nil, validationErrors
	}

	// Get and validate alt text
	alt := strings.TrimSpace(c.PostForm("alt"))
	if alt == "" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "alt",
			Value:   alt,
			Message: "Alt text is required",
		})
	} else if len(alt) < 10 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "alt",
			Value:   alt,
			Message: "Alt text must be at least 10 characters",
		})
	} else if len(alt) > 500 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "alt",
			Value:   alt,
			Message: "Alt text cannot exceed 500 characters",
		})
	}

	// Get and validate title (optional)
	title := strings.TrimSpace(c.PostForm("title"))
	if len(title) > 255 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "title",
			Value:   title,
			Message: "Title cannot exceed 255 characters",
		})
	}

	// Get and validate tags (optional)
	tags := strings.TrimSpace(c.PostForm("tags"))
	if len(tags) > 200 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "tags",
			Value:   tags,
			Message: "Tags cannot exceed 200 characters",
		})
	}

	// Get and validate weight (optional, default 1)
	weight := 1 // default value
	if weightStr := strings.TrimSpace(c.PostForm("weight")); weightStr != "" {
		if parsedWeight, err := strconv.Atoi(weightStr); err != nil {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "weight",
				Value:   weightStr,
				Message: "Weight must be a valid integer",
			})
		} else if parsedWeight < 1 || parsedWeight > 10 {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "weight",
				Value:   weightStr,
				Message: "Weight must be between 1 and 10",
			})
		} else {
			weight = parsedWeight
		}
	}

	request := &ImageUploadRequest{
		File:   fileHeader,
		Alt:    alt,
		Title:  title,
		Tags:   tags,
		Weight: weight,
	}

	return request, validationErrors
}

// generateStoragePath generates a unique storage path for the uploaded file
func (h *ImageUploadHandler) generateStoragePath(userID, filename string) string {
	// Generate unique filename to avoid collisions
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%s_%s%s", uuid.New().String(), time.Now().Format("20060102150405"), ext)

	return filepath.Join("storage", "images", userID, uniqueFilename)
}

// getMimeTypeFromExtension returns the MIME type based on file extension
func (h *ImageUploadHandler) getMimeTypeFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

// HandleImageUpload handles POST /api/images/upload endpoint
// Implements multipart form data handling, JPEG/PNG validation, and local storage with quota enforcement
func (h *ImageUploadHandler) HandleImageUpload(c *gin.Context) {
	// Verify admin authorization
	user, err := middleware.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, h.createErrorResponse(
			http.StatusUnauthorized,
			"Authentication required",
			"Valid authentication token required for image upload",
		))
		return
	}

	// Check if user has upload permissions (admin only)
	if err := h.authService.AuthorizeAdmin(user); err != nil {
		c.JSON(http.StatusForbidden, h.createErrorResponse(
			http.StatusForbidden,
			"Admin access required",
			"Only administrators can upload images",
		))
		return
	}

	// Validate upload request
	uploadRequest, validationErrors := h.validateUploadRequest(c)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, h.createValidationErrorResponse(
			"Invalid upload request",
			validationErrors,
		))
		return
	}

	// Validate image file
	width, height, fileValidationErrors := h.validateImageFile(uploadRequest.File)
	if len(fileValidationErrors) > 0 {
		validationErrors = append(validationErrors, fileValidationErrors...)
		c.JSON(http.StatusUnprocessableEntity, h.createValidationErrorResponse(
			"Invalid image file",
			validationErrors,
		))
		return
	}

	// Check user storage quota
	if !user.CanUploadFile(uploadRequest.File.Size) {
		c.JSON(http.StatusRequestEntityTooLarge, h.createErrorResponse(
			http.StatusRequestEntityTooLarge,
			"Storage quota exceeded",
			fmt.Sprintf("Upload would exceed storage quota. Current usage: %.2f MB, Quota: %.2f MB",
				float64(user.TotalStorageUsed)/(1024*1024),
				float64(user.StorageQuotaBytes)/(1024*1024)),
		))
		return
	}

	// Generate unique storage path
	storagePath := h.generateStoragePath(user.ID, uploadRequest.File.Filename)

	// Read file contents
	file, err := uploadRequest.File.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to read uploaded file",
			err.Error(),
		))
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to read file data",
			err.Error(),
		))
		return
	}

	// Create image record in database
	var title *string
	if uploadRequest.Title != "" {
		title = &uploadRequest.Title
	}

	var tags *string
	if uploadRequest.Tags != "" {
		tags = &uploadRequest.Tags
	}

	imageCreateRequest := &models.ImageCreateRequest{
		Filename:    uploadRequest.File.Filename,
		Alt:         uploadRequest.Alt,
		Title:       title,
		Tags:        tags,
		Weight:      uploadRequest.Weight,
		StoragePath: storagePath,
		MimeType:    h.getMimeTypeFromExtension(uploadRequest.File.Filename),
		FileSize:    uploadRequest.File.Size,
		Width:       &width,
		Height:      &height,
	}

	createdImage, err := h.imageService.CreateImage(imageCreateRequest, &user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to create image record",
			err.Error(),
		))
		return
	}

	// Store file in storage system (local filesystem)
	// This is a simplified implementation - in production you'd use the storage interface
	// TODO: Use the actual storage service from imageService
	if err := h.storeFile(storagePath, fileData); err != nil {
		// Clean up database record if file storage fails
		h.imageService.DeleteImage(createdImage.ID)
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to store image file",
			err.Error(),
		))
		return
	}

	// Update user storage usage
	user.UpdateStorageUsage(uploadRequest.File.Size)
	if err := h.db.Save(user).Error; err != nil {
		// Log error but don't fail the upload
		fmt.Printf("Warning: failed to update user storage usage: %v\n", err)
	}

	// Update image status to active
	if err := h.imageService.UpdateImageStatus(createdImage.ID, "active"); err != nil {
		// Log error but don't fail the upload
		fmt.Printf("Warning: failed to update image status: %v\n", err)
	}

	// Convert to response format
	var titlePtr *string
	if createdImage.Title != nil {
		titlePtr = createdImage.Title
	}

	response := ImageUploadResponse{
		ID:          createdImage.ID,
		Filename:    createdImage.Filename,
		Alt:         createdImage.Alt,
		Title:       titlePtr,
		Tags:        ParseTagsFromString(createdImage.Tags),
		Width:       createdImage.Width,
		Height:      createdImage.Height,
		AspectRatio: createdImage.AspectRatio,
		Weight:      createdImage.Weight,
		UploadDate:  createdImage.UploadDate.Format(time.RFC3339),
		StoragePath: createdImage.StoragePath,
	}

	c.JSON(http.StatusCreated, response)
}

// storeFile stores the file data to the local filesystem
func (h *ImageUploadHandler) storeFile(storagePath string, data []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(storagePath)
	if err := createDirIfNotExists(dir); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Write file to disk
	if err := os.WriteFile(storagePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file to %s: %w", storagePath, err)
	}

	return nil
}

// createDirIfNotExists creates a directory if it doesn't exist
func createDirIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

// Using common ParseTagsFromString function from common.go

// RegisterRoutes registers image upload routes with the router
func (h *ImageUploadHandler) RegisterRoutes(router *gin.RouterGroup, oauthMiddleware *middleware.OAuthMiddleware) {
	api := router.Group("/api")
	{
		// Protected image upload endpoint (admin only)
		api.POST("/images/upload",
			oauthMiddleware.RequireAuth(),
			oauthMiddleware.RequireRole("admin"),
			h.HandleImageUpload)
	}
}