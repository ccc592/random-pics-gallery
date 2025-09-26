package handlers

import (
	"net/http"
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
)

// ImageCRUDHandler handles image CRUD operations (Create, Read, Update, Delete)
type ImageCRUDHandler struct {
	db           *gorm.DB
	imageService *imageService.Service
	authService  *auth.Service
}

// NewImageCRUDHandler creates a new image CRUD handler
func NewImageCRUDHandler(db *gorm.DB, imgService *imageService.Service, authService *auth.Service) *ImageCRUDHandler {
	return &ImageCRUDHandler{
		db:           db,
		imageService: imgService,
		authService:  authService,
	}
}

// ImageResponse represents a single image in the response
type ImageCRUDResponse struct {
	ID          string   `json:"id"`
	Alt         string   `json:"alt"`
	Title       *string  `json:"title,omitempty"`
	Tags        []string `json:"tags"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	AspectRatio float64  `json:"aspect_ratio"`
	Weight      int      `json:"weight"`
	UploadDate  string   `json:"upload_date"`
	StoragePath string   `json:"storage_path"`
	Status      string   `json:"status"`
	UploadedBy  string   `json:"uploaded_by"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// ListImagesResponse represents the response for paginated image list
type ListImagesResponse struct {
	Images  []ImageCRUDResponse `json:"images"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	Limit   int                 `json:"limit"`
	HasMore bool                `json:"has_more"`
}

// UpdateImageRequest represents the request payload for updating an image
type UpdateImageRequest struct {
	Alt    *string  `json:"alt,omitempty" binding:"omitempty,min=10,max=500"`
	Title  *string  `json:"title,omitempty" binding:"omitempty,max=255"`
	Tags   []string `json:"tags,omitempty"`
	Weight *int     `json:"weight,omitempty" binding:"omitempty,min=1,max=10"`
	Status *string  `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
}

// Using common error response types from common.go

// createErrorResponse creates a standardized error response
func (h *ImageCRUDHandler) createErrorResponse(code int, message, details string) ErrorResponse {
	return CreateErrorResponse(code, message, details)
}

// createValidationErrorResponse creates an error response with validation errors
func (h *ImageCRUDHandler) createValidationErrorResponse(message string, validations []ValidationError) ErrorResponse {
	return CreateValidationErrorResponse(message, validations)
}

// convertModelToResponse converts a models.Image to ImageCRUDResponse
func (h *ImageCRUDHandler) convertModelToResponse(img *models.Image) ImageCRUDResponse {
	// Parse tags from comma-separated string
	tags := ParseTagsFromString(img.Tags)

	return ImageCRUDResponse{
		ID:          img.ID,
		Alt:         img.Alt,
		Title:       img.Title,
		Tags:        tags,
		Width:       img.Width,
		Height:      img.Height,
		AspectRatio: img.AspectRatio,
		Weight:      img.Weight,
		UploadDate:  img.UploadDate.Format(time.RFC3339),
		StoragePath: img.StoragePath,
		Status:      img.Status,
		UploadedBy:  img.UploadedBy,
		CreatedAt:   img.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   img.UpdatedAt.Format(time.RFC3339),
	}
}

// validateUUID validates that a string is a valid UUID
func (h *ImageCRUDHandler) validateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return err
	}
	return nil
}

// validateListImagesParameters validates pagination and filtering parameters
func (h *ImageCRUDHandler) validateListImagesParameters(c *gin.Context) (int, int, string, string, string, bool, []ValidationError) {
	var validationErrors []ValidationError

	// Parse page parameter
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err != nil {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "page",
				Value:   pageStr,
				Message: "page must be a valid integer",
			})
		} else if parsedPage < 1 {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "page",
				Value:   pageStr,
				Message: "page must be greater than 0",
			})
		} else {
			page = parsedPage
		}
	}

	// Parse limit parameter
	limit := 20 // default value
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err != nil {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "limit",
				Value:   limitStr,
				Message: "limit must be a valid integer",
			})
		} else if parsedLimit < 1 || parsedLimit > 100 {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "limit",
				Value:   limitStr,
				Message: "limit must be between 1 and 100",
			})
		} else {
			limit = parsedLimit
		}
	}

	// Validate tags parameter
	tags := c.Query("tags")
	if len(tags) > 200 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "tags",
			Value:   tags,
			Message: "tags parameter cannot exceed 200 characters",
		})
	}

	// Validate status parameter
	status := c.Query("status")
	if status != "" && status != "active" && status != "inactive" && status != "processing" && status != "failed" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "status",
			Value:   status,
			Message: "status must be one of: active, inactive, processing, failed",
		})
	}

	// Validate sort_by parameter
	sortBy := c.Query("sort_by")
	if sortBy == "" {
		sortBy = "created_at" // default
	} else if sortBy != "created_at" && sortBy != "upload_date" && sortBy != "filename" && sortBy != "weight" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "sort_by",
			Value:   sortBy,
			Message: "sort_by must be one of: created_at, upload_date, filename, weight",
		})
	}

	// Parse sort_desc parameter
	sortDesc := true // default
	if sortDescStr := c.Query("sort_desc"); sortDescStr != "" {
		if sortDescStr == "true" {
			sortDesc = true
		} else if sortDescStr == "false" {
			sortDesc = false
		} else {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "sort_desc",
				Value:   sortDescStr,
				Message: "sort_desc must be true or false",
			})
		}
	}

	return page, limit, tags, status, sortBy, sortDesc, validationErrors
}

// HandleGetImage handles GET /api/images/{id} endpoint
// Retrieves a single image by its ID
func (h *ImageCRUDHandler) HandleGetImage(c *gin.Context) {
	// Get image ID from path parameter
	imageID := c.Param("id")

	// Validate UUID format
	if err := h.validateUUID(imageID); err != nil {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Invalid image ID format",
			"Image ID must be a valid UUID",
		))
		return
	}

	// Get image from service
	image, err := h.imageService.GetImageByID(imageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, h.createErrorResponse(
				http.StatusNotFound,
				"Image not found",
				"No image found with the specified ID",
			))
		} else {
			c.JSON(http.StatusInternalServerError, h.createErrorResponse(
				http.StatusInternalServerError,
				"Failed to retrieve image",
				err.Error(),
			))
		}
		return
	}

	// Convert to response format
	response := h.convertModelToResponse(image)

	// Add caching headers
	c.Header("Cache-Control", "public, max-age=3600") // 1 hour cache
	c.Header("ETag", `"`+image.ID+`_`+image.UpdatedAt.Format("20060102150405")+`"`)

	c.JSON(http.StatusOK, response)
}

// HandleListImages handles GET /api/images endpoint
// Lists images with pagination and filtering
func (h *ImageCRUDHandler) HandleListImages(c *gin.Context) {
	// Validate parameters
	page, limit, tags, status, sortBy, sortDesc, validationErrors := h.validateListImagesParameters(c)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, h.createValidationErrorResponse(
			"Invalid request parameters",
			validationErrors,
		))
		return
	}

	// Get images from service
	// TODO: Pass sortBy and sortDesc parameters to service when sorting is implemented
	_ = sortBy   // Currently unused - will be used when service supports sorting
	_ = sortDesc // Currently unused - will be used when service supports sorting
	imageListResponse, err := h.imageService.ListImages(page, limit, status, tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to retrieve images",
			err.Error(),
		))
		return
	}

	// Convert to API response format
	images := make([]ImageCRUDResponse, len(imageListResponse.Images))
	for i, img := range imageListResponse.Images {
		// Convert from models.ImageResponse to ImageCRUDResponse
		images[i] = ImageCRUDResponse{
			ID:          img.ID,
			Alt:         img.Alt,
			Title:       img.Title,
			Tags:        ParseTagsFromString(img.Tags),
			Width:       img.Width,
			Height:      img.Height,
			AspectRatio: img.AspectRatio,
			Weight:      img.Weight,
			UploadDate:  img.UploadDate,
			StoragePath: img.StoragePath,
			Status:      img.Status,
			UploadedBy:  img.UploadedBy,
			CreatedAt:   img.CreatedAt,
			UpdatedAt:   img.UpdatedAt,
		}
	}

	hasMore := page*limit < imageListResponse.Total

	response := ListImagesResponse{
		Images:  images,
		Total:   imageListResponse.Total,
		Page:    page,
		Limit:   limit,
		HasMore: hasMore,
	}

	// Add caching headers
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes cache for lists

	c.JSON(http.StatusOK, response)
}

// HandleUpdateImage handles PUT /api/images/{id} endpoint
// Updates image metadata (admin only)
func (h *ImageCRUDHandler) HandleUpdateImage(c *gin.Context) {
	// Get image ID from path parameter
	imageID := c.Param("id")

	// Validate UUID format
	if err := h.validateUUID(imageID); err != nil {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Invalid image ID format",
			"Image ID must be a valid UUID",
		))
		return
	}

	// Verify admin authorization
	user, err := middleware.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, h.createErrorResponse(
			http.StatusUnauthorized,
			"Authentication required",
			"Valid authentication token required for image updates",
		))
		return
	}

	if err := h.authService.AuthorizeAdmin(user); err != nil {
		c.JSON(http.StatusForbidden, h.createErrorResponse(
			http.StatusForbidden,
			"Admin access required",
			"Only administrators can update images",
		))
		return
	}

	// Parse request body
	var updateRequest UpdateImageRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate request
	var validationErrors []ValidationError

	if updateRequest.Alt != nil {
		if len(*updateRequest.Alt) < 10 || len(*updateRequest.Alt) > 500 {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "alt",
				Value:   *updateRequest.Alt,
				Message: "Alt text must be between 10 and 500 characters",
			})
		}
	}

	if updateRequest.Title != nil && len(*updateRequest.Title) > 255 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "title",
			Value:   *updateRequest.Title,
			Message: "Title cannot exceed 255 characters",
		})
	}

	if updateRequest.Weight != nil && (*updateRequest.Weight < 1 || *updateRequest.Weight > 10) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "weight",
			Value:   strconv.Itoa(*updateRequest.Weight),
			Message: "Weight must be between 1 and 10",
		})
	}

	if updateRequest.Status != nil && *updateRequest.Status != "active" && *updateRequest.Status != "inactive" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "status",
			Value:   *updateRequest.Status,
			Message: "Status must be either 'active' or 'inactive'",
		})
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusUnprocessableEntity, h.createValidationErrorResponse(
			"Validation failed",
			validationErrors,
		))
		return
	}

	// Convert to models.ImageUpdateRequest
	var tagsStr *string
	if updateRequest.Tags != nil {
		tagString := strings.Join(updateRequest.Tags, ",")
		tagsStr = &tagString
	}

	modelUpdateRequest := &models.ImageUpdateRequest{
		Alt:    updateRequest.Alt,
		Title:  updateRequest.Title,
		Tags:   tagsStr,
		Weight: updateRequest.Weight,
		Status: updateRequest.Status,
	}

	// Update image using service
	updatedImage, err := h.imageService.UpdateImage(imageID, modelUpdateRequest)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, h.createErrorResponse(
				http.StatusNotFound,
				"Image not found",
				"No image found with the specified ID",
			))
		} else {
			c.JSON(http.StatusInternalServerError, h.createErrorResponse(
				http.StatusInternalServerError,
				"Failed to update image",
				err.Error(),
			))
		}
		return
	}

	// Convert to response format
	response := h.convertModelToResponse(updatedImage)

	c.JSON(http.StatusOK, response)
}

// HandleDeleteImage handles DELETE /api/images/{id} endpoint
// Deletes an image (admin only)
func (h *ImageCRUDHandler) HandleDeleteImage(c *gin.Context) {
	// Get image ID from path parameter
	imageID := c.Param("id")

	// Validate UUID format
	if err := h.validateUUID(imageID); err != nil {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Invalid image ID format",
			"Image ID must be a valid UUID",
		))
		return
	}

	// Verify admin authorization
	user, err := middleware.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, h.createErrorResponse(
			http.StatusUnauthorized,
			"Authentication required",
			"Valid authentication token required for image deletion",
		))
		return
	}

	if err := h.authService.AuthorizeAdmin(user); err != nil {
		c.JSON(http.StatusForbidden, h.createErrorResponse(
			http.StatusForbidden,
			"Admin access required",
			"Only administrators can delete images",
		))
		return
	}

	// Get image first to check if it exists and get file size for quota adjustment
	image, err := h.imageService.GetImageByID(imageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, h.createErrorResponse(
				http.StatusNotFound,
				"Image not found",
				"No image found with the specified ID",
			))
		} else {
			c.JSON(http.StatusInternalServerError, h.createErrorResponse(
				http.StatusInternalServerError,
				"Failed to retrieve image",
				err.Error(),
			))
		}
		return
	}

	// Delete image using service
	if err := h.imageService.DeleteImage(imageID); err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to delete image",
			err.Error(),
		))
		return
	}

	// Update user storage usage (subtract deleted file size)
	if image.UploadedBy != "" {
		var uploader models.User
		if err := h.db.Where("id = ?", image.UploadedBy).First(&uploader).Error; err == nil {
			uploader.UpdateStorageUsage(-image.FileSize)
			h.db.Save(&uploader)
		}
	}

	c.Status(http.StatusNoContent)
}

// Using common ParseTagsFromString function from common.go

// RegisterRoutes registers image CRUD routes with the router
func (h *ImageCRUDHandler) RegisterRoutes(router *gin.RouterGroup, oauthMiddleware *middleware.OAuthMiddleware) {
	api := router.Group("/api")
	{
		// Public image retrieval endpoints
		api.GET("/images", h.HandleListImages)
		api.GET("/images/:id", h.HandleGetImage)

		// Protected image management endpoints (admin only)
		api.PUT("/images/:id",
			oauthMiddleware.RequireAuth(),
			oauthMiddleware.RequireRole("admin"),
			h.HandleUpdateImage)
		api.DELETE("/images/:id",
			oauthMiddleware.RequireAuth(),
			oauthMiddleware.RequireRole("admin"),
			h.HandleDeleteImage)
	}
}