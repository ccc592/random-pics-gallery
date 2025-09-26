package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/db/models"
	"github.com/randompic/api/internal/image"
	"gorm.io/gorm"
)

// RandomImagesHandler handles random image selection requests
type RandomImagesHandler struct {
	db           *gorm.DB
	imageService *image.Service
}

// NewRandomImagesHandler creates a new random images handler
func NewRandomImagesHandler(db *gorm.DB, imageService *image.Service) *RandomImagesHandler {
	return &RandomImagesHandler{
		db:           db,
		imageService: imageService,
	}
}

// RandomImagesResponse represents the response structure for random images
type RandomImagesResponse struct {
	Images []ImageResponse `json:"images"`
	Seed   int64           `json:"seed"`
	Total  int             `json:"total"`
}

// ImageResponse represents a single image in the response
type ImageResponse struct {
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
}

// Using common error response types from common.go

// createErrorResponse creates a standardized error response
func (h *RandomImagesHandler) createErrorResponse(code int, message, details string) ErrorResponse {
	return CreateErrorResponse(code, message, details)
}

// createValidationErrorResponse creates an error response with validation errors
func (h *RandomImagesHandler) createValidationErrorResponse(message string, validations []ValidationError) ErrorResponse {
	return CreateValidationErrorResponse(message, validations)
}

// convertImageToResponse converts a models.Image to ImageResponse
func (h *RandomImagesHandler) convertImageToResponse(img models.Image) ImageResponse {
	// Parse tags from comma-separated string
	var tags []string
	if img.Tags != nil && *img.Tags != "" {
		tagList := strings.Split(*img.Tags, ",")
		tags = make([]string, len(tagList))
		for i, tag := range tagList {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	return ImageResponse{
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
	}
}

// validateRandomImagesRequest validates the request parameters
func (h *RandomImagesHandler) validateRandomImagesRequest(c *gin.Context) (int, string, int64, []ValidationError) {
	var validationErrors []ValidationError

	// Parse and validate limit parameter
	limit := 4 // default value
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err != nil {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "limit",
				Value:   limitStr,
				Message: "limit must be a valid integer",
			})
		} else if parsedLimit < 3 || parsedLimit > 5 {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "limit",
				Value:   limitStr,
				Message: "limit must be between 3 and 5",
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

	// Parse and validate seed parameter
	var seed int64
	if seedStr := c.Query("seed"); seedStr != "" {
		if parsedSeed, err := strconv.ParseInt(seedStr, 10, 64); err != nil {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "seed",
				Value:   seedStr,
				Message: "seed must be a valid integer",
			})
		} else {
			seed = parsedSeed
		}
	} else {
		// Generate seed if not provided
		seed = time.Now().UnixNano()
	}

	return limit, tags, seed, validationErrors
}

// applyFisherYatesSelection applies Fisher-Yates shuffle with weighted selection to images
func (h *RandomImagesHandler) applyFisherYatesSelection(images []models.Image, limit int, seed int64) []models.Image {
	if len(images) == 0 {
		return []models.Image{}
	}

	// Use seed for reproducible randomization
	rand.Seed(seed)

	// Create a copy to avoid modifying the original slice
	imagesCopy := make([]models.Image, len(images))
	copy(imagesCopy, images)

	// Apply Fisher-Yates shuffle considering weights
	for i := len(imagesCopy) - 1; i > 0; i-- {
		// Weight-based selection probability
		totalWeight := 0
		for j := 0; j <= i; j++ {
			totalWeight += imagesCopy[j].Weight
		}

		if totalWeight == 0 {
			// If no weights, use uniform distribution
			j := rand.Intn(i + 1)
			imagesCopy[i], imagesCopy[j] = imagesCopy[j], imagesCopy[i]
		} else {
			// Weighted selection
			randWeight := rand.Intn(totalWeight)
			currentWeight := 0
			selectedIndex := 0

			for j := 0; j <= i; j++ {
				currentWeight += imagesCopy[j].Weight
				if currentWeight > randWeight {
					selectedIndex = j
					break
				}
			}

			imagesCopy[i], imagesCopy[selectedIndex] = imagesCopy[selectedIndex], imagesCopy[i]
		}
	}

	// Return the requested number of images
	if limit > len(imagesCopy) {
		limit = len(imagesCopy)
	}

	return imagesCopy[:limit]
}

// filterImagesByTags filters images based on tag criteria
func (h *RandomImagesHandler) filterImagesByTags(images []models.Image, tagFilter string) []models.Image {
	if tagFilter == "" {
		return images
	}

	// Parse comma-separated tags
	filterTags := strings.Split(tagFilter, ",")
	for i := range filterTags {
		filterTags[i] = strings.ToLower(strings.TrimSpace(filterTags[i]))
	}

	var filtered []models.Image
	for _, img := range images {
		if img.Tags == nil || *img.Tags == "" {
			continue
		}

		// Parse image tags
		imageTags := strings.Split(*img.Tags, ",")
		imageTagsLower := make([]string, len(imageTags))
		for i, tag := range imageTags {
			imageTagsLower[i] = strings.ToLower(strings.TrimSpace(tag))
		}

		// Check if any filter tag matches any image tag
		matchFound := false
		for _, filterTag := range filterTags {
			for _, imageTag := range imageTagsLower {
				if strings.Contains(imageTag, filterTag) {
					matchFound = true
					break
				}
			}
			if matchFound {
				break
			}
		}

		if matchFound {
			filtered = append(filtered, img)
		}
	}

	return filtered
}

// HandleRandomImages handles GET /api/random-images endpoint
// Implements Fisher-Yates randomization with weights, tag filtering, and seed reproducibility
func (h *RandomImagesHandler) HandleRandomImages(c *gin.Context) {
	// Validate request parameters
	limit, tags, seed, validationErrors := h.validateRandomImagesRequest(c)

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, h.createValidationErrorResponse(
			"Invalid request parameters",
			validationErrors,
		))
		return
	}

	// Use image service to get random images
	seedString := fmt.Sprintf("%d", seed)
	response, err := h.imageService.GetRandomImages(limit, seedString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to retrieve random images",
			err.Error(),
		))
		return
	}

	// If tag filtering is requested, we need to do it manually
	// since the image service doesn't support tag filtering yet
	if tags != "" {
		// Get all active images from database for filtering
		var allImages []models.Image
		if err := h.db.Where("status = ?", "active").Find(&allImages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, h.createErrorResponse(
				http.StatusInternalServerError,
				"Failed to retrieve images for filtering",
				err.Error(),
			))
			return
		}

		// Filter by tags
		filteredImages := h.filterImagesByTags(allImages, tags)

		if len(filteredImages) == 0 {
			// No images match the tag filter
			c.JSON(http.StatusOK, RandomImagesResponse{
				Images: []ImageResponse{},
				Seed:   seed,
				Total:  0,
			})
			return
		}

		// Apply Fisher-Yates randomization to filtered images
		selectedImages := h.applyFisherYatesSelection(filteredImages, limit, seed)

		// Convert filtered images to response format and use them instead of the unfiltered response
		imageResponses := make([]ImageResponse, len(selectedImages))
		for i, img := range selectedImages {
			imageResponses[i] = h.convertImageToResponse(img)
		}

		c.JSON(http.StatusOK, RandomImagesResponse{
			Images: imageResponses,
			Seed:   seed,
			Total:  len(selectedImages),
		})
		return
	}

	// Convert to API response format
	imageResponses := make([]ImageResponse, len(response.Images))
	for i, img := range response.Images {
		// Convert from models.ImageResponse to our ImageResponse
		imageResponses[i] = ImageResponse{
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
		}
	}

	// Create final response
	finalResponse := RandomImagesResponse{
		Images: imageResponses,
		Seed:   seed,
		Total:  len(imageResponses),
	}

	// Add caching headers for performance optimization
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes cache
	c.Header("ETag", fmt.Sprintf(`"%d-%d"`, seed, len(imageResponses)))

	c.JSON(http.StatusOK, finalResponse)
}

// Using common parseTagsFromString function from common.go

// HandleRandomImagesStats provides statistics about random image selection
// GET /api/random-images/stats (optional debugging endpoint)
func (h *RandomImagesHandler) HandleRandomImagesStats(c *gin.Context) {
	// Get basic image statistics
	stats, err := h.imageService.GetImageStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to retrieve image statistics",
			err.Error(),
		))
		return
	}

	// Add additional random selection statistics
	stats["endpoint"] = "random-images"
	stats["supported_limits"] = []int{3, 4, 5}
	stats["fisher_yates_enabled"] = true
	stats["seed_reproducible"] = true
	stats["tag_filtering_enabled"] = true

	c.JSON(http.StatusOK, stats)
}

// RegisterRoutes registers random images routes with the router
func (h *RandomImagesHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/api")
	{
		// Public random images endpoint (no authentication required)
		api.GET("/random-images", h.HandleRandomImages)
		api.GET("/random-images/stats", h.HandleRandomImagesStats) // Optional debug endpoint
	}
}