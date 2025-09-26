package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"strings"
	"time"

	"github.com/randompic/api/internal/db/models"
)

// RandomParams represents parameters for random image selection
type RandomParams struct {
	Count       int      `json:"count" validate:"min=1,max=50"`
	Tags        []string `json:"tags,omitempty"`
	Seed        *int64   `json:"seed,omitempty"`
	MinWeight   *int     `json:"min_weight,omitempty" validate:"omitempty,min=1,max=10"`
	MaxWeight   *int     `json:"max_weight,omitempty" validate:"omitempty,min=1,max=10"`
	MimeType    *string  `json:"mime_type,omitempty"`
	UserID      *string  `json:"user_id,omitempty"` // For user-specific selections
}

// RandomResult represents the result of random image selection
type RandomResult struct {
	Images    []*models.Image `json:"images"`
	Seed      int64           `json:"seed"`
	Count     int             `json:"count"`
	Total     int             `json:"total"`     // Total available images
	Tags      []string        `json:"tags"`
	Generated time.Time       `json:"generated"`
}

// WeightedImage represents an image with computed selection weight
type WeightedImage struct {
	Image  *models.Image `json:"image"`
	Weight float64       `json:"weight"`
}

// RandomizerServiceInterface defines the contract for random image selection
type RandomizerServiceInterface interface {
	GetRandomImages(ctx context.Context, params *RandomParams) (*RandomResult, error)
	ShuffleWithWeights(images []*models.Image, seed int64) []*models.Image
	FilterByTags(images []*models.Image, tags []string) []*models.Image
	ValidateRandomParams(params *RandomParams) error
}

// RandomizerService implements the RandomizerServiceInterface
type RandomizerService struct {
	imageService ImageServiceInterface
}

// NewRandomizerService creates a new RandomizerService instance
func NewRandomizerService(imageService ImageServiceInterface) *RandomizerService {
	return &RandomizerService{
		imageService: imageService,
	}
}

// GetRandomImages selects random images based on parameters
func (s *RandomizerService) GetRandomImages(ctx context.Context, params *RandomParams) (*RandomResult, error) {
	if params == nil {
		return nil, fmt.Errorf("parameters cannot be nil")
	}

	// Validate parameters
	if err := s.ValidateRandomParams(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Generate seed if not provided
	seed := params.Seed
	if seed == nil {
		generatedSeed, err := s.generateSeed()
		if err != nil {
			return nil, fmt.Errorf("failed to generate seed: %w", err)
		}
		seed = &generatedSeed
	}

	// Get all active images
	allImages, err := s.imageService.GetActiveImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active images: %w", err)
	}

	if len(allImages) == 0 {
		return &RandomResult{
			Images:    []*models.Image{},
			Seed:      *seed,
			Count:     0,
			Total:     0,
			Tags:      params.Tags,
			Generated: time.Now(),
		}, nil
	}

	// Apply filters
	filteredImages := s.applyFilters(allImages, params)

	if len(filteredImages) == 0 {
		return &RandomResult{
			Images:    []*models.Image{},
			Seed:      *seed,
			Count:     0,
			Total:     len(allImages),
			Tags:      params.Tags,
			Generated: time.Now(),
		}, nil
	}

	// Shuffle with weights
	shuffledImages := s.ShuffleWithWeights(filteredImages, *seed)

	// Select requested count
	selectedCount := params.Count
	if selectedCount > len(shuffledImages) {
		selectedCount = len(shuffledImages)
	}

	selectedImages := shuffledImages[:selectedCount]

	return &RandomResult{
		Images:    selectedImages,
		Seed:      *seed,
		Count:     len(selectedImages),
		Total:     len(allImages),
		Tags:      params.Tags,
		Generated: time.Now(),
	}, nil
}

// ShuffleWithWeights shuffles images using Fisher-Yates algorithm with weights
func (s *RandomizerService) ShuffleWithWeights(images []*models.Image, seed int64) []*models.Image {
	if len(images) <= 1 {
		return images
	}

	// Convert to weighted images and calculate cumulative weights
	weightedImages := s.convertToWeightedImages(images)

	// Create random source with seed for reproducibility
	source := mathrand.NewSource(seed)
	rng := mathrand.New(source)

	// Use weighted Fisher-Yates shuffle
	shuffled := make([]*models.Image, len(images))
	copy(shuffled, images)

	for i := len(shuffled) - 1; i > 0; i-- {
		// Calculate weighted random index
		j := s.weightedRandomIndex(weightedImages[:i+1], rng)

		// Swap elements
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		weightedImages[i], weightedImages[j] = weightedImages[j], weightedImages[i]
	}

	return shuffled
}

// FilterByTags filters images by tags
func (s *RandomizerService) FilterByTags(images []*models.Image, tags []string) []*models.Image {
	if len(tags) == 0 {
		return images
	}

	var filtered []*models.Image
	for _, image := range images {
		if s.imageMatchesTags(image, tags) {
			filtered = append(filtered, image)
		}
	}

	return filtered
}

// ValidateRandomParams validates random selection parameters
func (s *RandomizerService) ValidateRandomParams(params *RandomParams) error {
	if params == nil {
		return fmt.Errorf("params cannot be nil")
	}

	// Validate count
	if params.Count < 1 {
		return fmt.Errorf("count must be at least 1")
	}
	if params.Count > 50 {
		return fmt.Errorf("count cannot exceed 50")
	}

	// Validate weight range
	if params.MinWeight != nil && (*params.MinWeight < 1 || *params.MinWeight > 10) {
		return fmt.Errorf("min_weight must be between 1 and 10")
	}
	if params.MaxWeight != nil && (*params.MaxWeight < 1 || *params.MaxWeight > 10) {
		return fmt.Errorf("max_weight must be between 1 and 10")
	}
	if params.MinWeight != nil && params.MaxWeight != nil && *params.MinWeight > *params.MaxWeight {
		return fmt.Errorf("min_weight cannot be greater than max_weight")
	}

	// Validate MIME type
	if params.MimeType != nil {
		validMimeTypes := map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
		}
		if !validMimeTypes[*params.MimeType] {
			return fmt.Errorf("invalid mime_type: %s, allowed: image/jpeg, image/png", *params.MimeType)
		}
	}

	// Validate tags
	for _, tag := range params.Tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("tags cannot contain empty values")
		}
	}

	return nil
}

// GetRandomImagesWithReproducibility returns random images with reproducible results
func (s *RandomizerService) GetRandomImagesWithReproducibility(ctx context.Context, params *RandomParams, sessionSeed string) (*RandomResult, error) {
	// Convert session seed to numeric seed
	seed := s.sessionSeedToNumeric(sessionSeed)
	params.Seed = &seed

	return s.GetRandomImages(ctx, params)
}

// GenerateSessionSeed creates a session-based seed for reproducible results
func (s *RandomizerService) GenerateSessionSeed() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("session_%d", timestamp)
}

// Helper functions

func (s *RandomizerService) applyFilters(images []*models.Image, params *RandomParams) []*models.Image {
	filtered := images

	// Filter by tags
	if len(params.Tags) > 0 {
		filtered = s.FilterByTags(filtered, params.Tags)
	}

	// Filter by weight range
	if params.MinWeight != nil || params.MaxWeight != nil {
		filtered = s.filterByWeightRange(filtered, params.MinWeight, params.MaxWeight)
	}

	// Filter by MIME type
	if params.MimeType != nil {
		filtered = s.filterByMimeType(filtered, *params.MimeType)
	}

	// Filter by user (if specified)
	if params.UserID != nil {
		filtered = s.filterByUser(filtered, *params.UserID)
	}

	return filtered
}

func (s *RandomizerService) filterByWeightRange(images []*models.Image, minWeight, maxWeight *int) []*models.Image {
	var filtered []*models.Image
	for _, image := range images {
		include := true

		if minWeight != nil && image.Weight < *minWeight {
			include = false
		}
		if maxWeight != nil && image.Weight > *maxWeight {
			include = false
		}

		if include {
			filtered = append(filtered, image)
		}
	}
	return filtered
}

func (s *RandomizerService) filterByMimeType(images []*models.Image, mimeType string) []*models.Image {
	var filtered []*models.Image
	for _, image := range images {
		if image.MimeType == mimeType {
			filtered = append(filtered, image)
		}
	}
	return filtered
}

func (s *RandomizerService) filterByUser(images []*models.Image, userID string) []*models.Image {
	var filtered []*models.Image
	for _, image := range images {
		if image.UploadedBy == userID {
			filtered = append(filtered, image)
		}
	}
	return filtered
}

func (s *RandomizerService) imageMatchesTags(image *models.Image, tags []string) bool {
	if image.Tags == nil {
		return false
	}

	imageTags := strings.ToLower(*image.Tags)
	for _, tag := range tags {
		if strings.Contains(imageTags, strings.ToLower(tag)) {
			return true
		}
	}
	return false
}

func (s *RandomizerService) convertToWeightedImages(images []*models.Image) []WeightedImage {
	weighted := make([]WeightedImage, len(images))
	for i, image := range images {
		// Convert integer weight (1-10) to floating point weight
		// Higher weight = higher probability of selection
		weight := float64(image.Weight)
		weighted[i] = WeightedImage{
			Image:  image,
			Weight: weight,
		}
	}
	return weighted
}

func (s *RandomizerService) weightedRandomIndex(weightedImages []WeightedImage, rng *mathrand.Rand) int {
	if len(weightedImages) == 0 {
		return 0
	}
	if len(weightedImages) == 1 {
		return 0
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, wi := range weightedImages {
		totalWeight += wi.Weight
	}

	if totalWeight == 0 {
		// If no weights, use uniform distribution
		return rng.Intn(len(weightedImages))
	}

	// Generate random value between 0 and totalWeight
	randomValue := rng.Float64() * totalWeight

	// Find the selected index
	currentWeight := 0.0
	for i, wi := range weightedImages {
		currentWeight += wi.Weight
		if randomValue <= currentWeight {
			return i
		}
	}

	// Fallback to last index (should not normally reach here)
	return len(weightedImages) - 1
}

func (s *RandomizerService) generateSeed() (int64, error) {
	// Generate cryptographically secure random seed
	max := big.NewInt(9223372036854775807) // max int64
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to time-based seed
		return time.Now().UnixNano(), nil
	}
	return n.Int64(), nil
}

func (s *RandomizerService) sessionSeedToNumeric(sessionSeed string) int64 {
	// Convert session seed string to numeric seed
	hash := int64(0)
	for _, c := range sessionSeed {
		hash = hash*31 + int64(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// GetImagesByWeight returns images grouped by weight for analysis
func (s *RandomizerService) GetImagesByWeight(ctx context.Context) (map[int][]*models.Image, error) {
	images, err := s.imageService.GetActiveImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active images: %w", err)
	}

	weightGroups := make(map[int][]*models.Image)
	for _, image := range images {
		weightGroups[image.Weight] = append(weightGroups[image.Weight], image)
	}

	return weightGroups, nil
}

// TestShuffleConsistency tests that the same seed produces consistent results
func (s *RandomizerService) TestShuffleConsistency(images []*models.Image, seed int64, iterations int) bool {
	if len(images) == 0 || iterations < 1 {
		return true
	}

	firstResult := s.ShuffleWithWeights(images, seed)

	for i := 1; i < iterations; i++ {
		result := s.ShuffleWithWeights(images, seed)

		// Compare results
		if len(result) != len(firstResult) {
			return false
		}

		for j := range result {
			if result[j].ID != firstResult[j].ID {
				return false
			}
		}
	}

	return true
}

// AnalyzeWeightDistribution analyzes the distribution of weights in random selection
func (s *RandomizerService) AnalyzeWeightDistribution(ctx context.Context, iterations int) (map[int]int, error) {
	if iterations < 1 {
		return nil, fmt.Errorf("iterations must be at least 1")
	}

	images, err := s.imageService.GetActiveImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active images: %w", err)
	}

	if len(images) == 0 {
		return make(map[int]int), nil
	}

	weightCounts := make(map[int]int)

	for i := 0; i < iterations; i++ {
		params := &RandomParams{
			Count: 1,
			Seed:  nil, // Use random seed each time
		}

		result, err := s.GetRandomImages(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get random images: %w", err)
		}

		if len(result.Images) > 0 {
			weight := result.Images[0].Weight
			weightCounts[weight]++
		}
	}

	return weightCounts, nil
}