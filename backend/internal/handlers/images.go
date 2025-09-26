package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/db/models"
	"github.com/randompic/api/internal/image"
)

// ImageHandler handles image-related HTTP requests
type ImageHandler struct {
	imageService *image.Service
	authService  *auth.Service
}

// NewImageHandler creates a new image handler
func NewImageHandler(imageService *image.Service, authService *auth.Service) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
		authService:  authService,
	}
}

// GetRandomImages handles GET /api/images/random
func (h *ImageHandler) GetRandomImages(c *gin.Context) {
	// Parse query parameters
	countStr := c.DefaultQuery("count", "3")
	seed := c.Query("seed")

	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 || count > 5 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "count must be between 1 and 5",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Get random images
	response, err := h.imageService.GetRandomImages(count, seed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get random images",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetImageByID handles GET /api/images/:id
func (h *ImageHandler) GetImageByID(c *gin.Context) {
	idStr := c.Param("id")

	// Validate UUID format (but keep as string)
	if _, err := uuid.Parse(idStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid UUID format",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Get image
	image, err := h.imageService.GetImageByID(idStr)
	if err != nil {
		if err.Error() == "image not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "image not found",
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get image",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, image.ToResponse())
}

// ListImages handles GET /api/admin/images
func (h *ImageHandler) ListImages(c *gin.Context) {
	// This endpoint requires admin authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	authUser, ok := user.(*auth.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user context",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	// Check if user is admin
	userModel, err := h.authService.GetUserByID(authUser.UserID)
	if err != nil || !userModel.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin access required",
			"code":  "FORBIDDEN",
		})
		return
	}

	// Parse query parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	tags := c.Query("tags")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Get images
	response, err := h.imageService.ListImages(page, limit, status, tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list images",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UploadImage handles POST /api/admin/images/upload
func (h *ImageHandler) UploadImage(c *gin.Context) {
	// This endpoint requires admin authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	authUser, ok := user.(*auth.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user context",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	// Check if user is admin
	userModel, err := h.authService.GetUserByID(authUser.UserID)
	if err != nil || !userModel.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin access required",
			"code":  "FORBIDDEN",
		})
		return
	}

	// Get uploaded file
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file is required",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Validate file size (max 10MB)
	if fileHeader.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file size must be less than 10MB",
			"code":  "FILE_TOO_LARGE",
		})
		return
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to open uploaded file",
			"code":  "FILE_ERROR",
		})
		return
	}
	defer file.Close()

	// Read file content to validate type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read file content",
			"code":  "FILE_ERROR",
		})
		return
	}

	mimeType := http.DetectContentType(buffer)
	if !allowedTypes[mimeType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported file type. Only JPEG, PNG, WebP, and GIF are allowed",
			"code":  "INVALID_FILE_TYPE",
		})
		return
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Get form fields
	alt := c.PostForm("alt")
	if alt == "" || len(alt) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "alt text is required and must be at least 10 characters",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	title := c.PostForm("title")
	tags := c.PostForm("tags")
	weightStr := c.DefaultPostForm("weight", "1")

	weight, err := strconv.Atoi(weightStr)
	if err != nil || weight < 1 || weight > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "weight must be between 1 and 10",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Create a mock image record for now (in production, you'd store the file and extract metadata)
	mockID := uuid.New()
	mockStoragePath := fmt.Sprintf("/images/%s-%s", mockID.String(), fileHeader.Filename)

	// Create image record using the service
	createReq := &models.ImageCreateRequest{
		Filename:    fileHeader.Filename,
		Alt:         alt,
		Title:       &title,
		Tags:        &tags,
		Weight:      weight,
		StoragePath: mockStoragePath,
		MimeType:    mimeType,
		FileSize:    fileHeader.Size,
		Width:       nil, // Would extract from image in production
		Height:      nil, // Would extract from image in production
	}

	createdImage, err := h.imageService.CreateImage(createReq, &authUser.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create image record",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	// Update status to active (in production, this would happen after processing)
	err = h.imageService.UpdateImageStatus(createdImage.ID, "active")
	if err != nil {
		// Log warning but don't fail the request
		fmt.Printf("Warning: failed to update image status: %v\n", err)
	}

	c.JSON(http.StatusCreated, createdImage.ToResponse())
}

// UpdateImage handles PUT /api/admin/images/:id
func (h *ImageHandler) UpdateImage(c *gin.Context) {
	// This endpoint requires admin authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	authUser, ok := user.(*auth.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user context",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	// Check if user is admin
	userModel, err := h.authService.GetUserByID(authUser.UserID)
	if err != nil || !userModel.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin access required",
			"code":  "FORBIDDEN",
		})
		return
	}

	// Parse image ID
	idStr := c.Param("id")
	if _, err := uuid.Parse(idStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid UUID format",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Parse request body
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Build update request
	updateReq := &models.ImageUpdateRequest{}

	// Validate and set alt text
	if altValue, ok := updateData["alt"]; ok {
		if alt, ok := altValue.(string); ok {
			if len(alt) < 10 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "alt text must be at least 10 characters",
					"code":  "INVALID_REQUEST",
				})
				return
			}
			updateReq.Alt = &alt
		}
	}

	// Set title if provided
	if titleValue, ok := updateData["title"]; ok {
		if title, ok := titleValue.(string); ok {
			updateReq.Title = &title
		}
	}

	// Set tags if provided
	if tagsValue, ok := updateData["tags"]; ok {
		if tags, ok := tagsValue.(string); ok {
			updateReq.Tags = &tags
		}
	}

	// Validate and set weight
	if weightValue, ok := updateData["weight"]; ok {
		var weight int
		switch v := weightValue.(type) {
		case float64:
			weight = int(v)
		case int:
			weight = v
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "weight must be a number",
				"code":  "INVALID_REQUEST",
			})
			return
		}

		if weight < 1 || weight > 10 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "weight must be between 1 and 10",
				"code":  "INVALID_REQUEST",
			})
			return
		}
		updateReq.Weight = &weight
	}

	// Validate and set status
	if statusValue, ok := updateData["status"]; ok {
		if status, ok := statusValue.(string); ok {
			validStatuses := map[string]bool{
				"active":     true,
				"inactive":   true,
				"processing": true,
				"failed":     true,
			}
			if !validStatuses[status] {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "status must be one of: active, inactive, processing, failed",
					"code":  "INVALID_REQUEST",
				})
				return
			}
			updateReq.Status = &status
		}
	}

	// Update image using service
	updatedImage, err := h.imageService.UpdateImage(idStr, updateReq)
	if err != nil {
		if err.Error() == "image not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "image not found",
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update image",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, updatedImage.ToResponse())
}

// DeleteImage handles DELETE /api/admin/images/:id
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	// This endpoint requires admin authentication
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	authUser, ok := user.(*auth.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user context",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	// Check if user is admin
	userModel, err := h.authService.GetUserByID(authUser.UserID)
	if err != nil || !userModel.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "admin access required",
			"code":  "FORBIDDEN",
		})
		return
	}

	// Parse image ID
	idStr := c.Param("id")
	if _, err := uuid.Parse(idStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid UUID format",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Delete image
	err = h.imageService.DeleteImage(idStr)
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
		"id":      idStr,
	})
}