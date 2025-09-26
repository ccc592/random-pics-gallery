package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/db/models"
	"github.com/randompic/api/internal/randomizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

// TestRandomizerIntegration contains all integration tests for the Fisher-Yates randomization service
func TestRandomizerIntegration(t *testing.T) {
	// Skip if integration test flag is not set
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test database
	testDB, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Initialize randomizer service
	randomizerSvc := randomizer.NewService()

	t.Run("BasicRandomizationFunctionality", func(t *testing.T) {
		testBasicRandomization(t, testDB, randomizerSvc)
	})

	t.Run("WeightedSelectionDistribution", func(t *testing.T) {
		testWeightedSelectionDistribution(t, testDB, randomizerSvc)
	})

	t.Run("SeedReproducibility", func(t *testing.T) {
		testSeedReproducibility(t, testDB, randomizerSvc)
	})

	t.Run("StatisticalValidation", func(t *testing.T) {
		testStatisticalValidation(t, testDB, randomizerSvc)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		testEdgeCases(t, testDB, randomizerSvc)
	})

	t.Run("PerformanceWithLargeDatasets", func(t *testing.T) {
		testPerformanceWithLargeDatasets(t, testDB, randomizerSvc)
	})

	t.Run("DatabaseIntegrationQueries", func(t *testing.T) {
		testDatabaseIntegrationQueries(t, testDB, randomizerSvc)
	})

	t.Run("UserSpecificImageCollections", func(t *testing.T) {
		testUserSpecificImageCollections(t, testDB, randomizerSvc)
	})

	t.Run("TagFilteringIntegration", func(t *testing.T) {
		testTagFilteringIntegration(t, testDB, randomizerSvc)
	})
}

// setupTestDatabase creates a test database with sample data
func setupTestDatabase(t *testing.T) (*db.Database, func()) {
	// Use SQLite for integration tests to avoid dependency on external PostgreSQL
	config := db.SQLiteConfig(":memory:")
	config.LogLevel = logger.Silent // Reduce test output noise

	testDB, err := db.NewDatabase(config)
	require.NoError(t, err, "Failed to create test database")

	// Run migrations
	err = testDB.AutoMigrate()
	require.NoError(t, err, "Failed to run migrations")

	// Setup test data
	setupTestData(t, testDB)

	cleanup := func() {
		err := testDB.Close()
		if err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	}

	return testDB, cleanup
}

// setupTestData creates test users and images with various weights
func setupTestData(t *testing.T, testDB *db.Database) {
	// Create test users
	users := []models.User{
		{
			ID:            uuid.New(),
			KindeUserID:   "kinde_user_1",
			Email:         "user1@example.com",
			DisplayName:   stringPtrRand("User One"),
			AvatarURL:     stringPtrRand("https://example.com/user1.jpg"),
			IsActive:      true,
			EmailVerified: true,
		},
		{
			ID:            uuid.New(),
			KindeUserID:   "kinde_user_2",
			Email:         "user2@example.com",
			DisplayName:   stringPtrRand("User Two"),
			AvatarURL:     stringPtrRand("https://example.com/user2.jpg"),
			IsActive:      true,
			EmailVerified: true,
		},
	}

	for _, user := range users {
		err := testDB.DB.Create(&user).Error
		require.NoError(t, err, "Failed to create test user")
	}

	// Create test images with various weights (1-10)
	now := time.Now()
	images := []models.Image{
		// Weight 1 images (low priority)
		createTestImage(users[0].ID, "low_weight_1.jpg", "Beautiful mountain landscape with snow", 1, "nature,mountain"),
		createTestImage(users[0].ID, "low_weight_2.jpg", "Peaceful lake reflection at sunset", 1, "nature,water"),

		// Weight 5 images (medium priority)
		createTestImage(users[0].ID, "med_weight_1.jpg", "Colorful flower garden in spring", 5, "flowers,garden"),
		createTestImage(users[1].ID, "med_weight_2.jpg", "Urban cityscape at nighttime", 5, "city,night"),
		createTestImage(users[1].ID, "med_weight_3.jpg", "Abstract geometric art patterns", 5, "abstract,art"),

		// Weight 10 images (high priority)
		createTestImage(users[0].ID, "high_weight_1.jpg", "Stunning aurora borealis display", 10, "nature,aurora"),
		createTestImage(users[1].ID, "high_weight_2.jpg", "Majestic eagle soaring through clouds", 10, "wildlife,bird"),

		// Images with different statuses for testing
		createTestImageWithStatus(users[0].ID, "inactive_image.jpg", "This image is inactive", 8, "test,inactive", "inactive"),
		createTestImageWithStatus(users[1].ID, "processing_image.jpg", "This image is processing", 6, "test,processing", "processing"),

		// Edge case images for testing
		createTestImage(users[0].ID, "edge_case_1.jpg", "Edge case test image number one", 3, "test,edge"),
		createTestImage(users[1].ID, "edge_case_2.jpg", "Edge case test image number two", 7, "test,edge"),
	}

	// Set upload dates
	for i := range images {
		images[i].UploadDate = &now
		images[i].CreatedAt = now
		images[i].UpdatedAt = now
	}

	for _, img := range images {
		err := testDB.DB.Create(&img).Error
		require.NoError(t, err, "Failed to create test image")
	}
}

// Helper functions to create test data
func stringPtrRand(s string) *string {
	return &s
}

func createTestImage(userID uuid.UUID, filename, alt string, weight int, tags string) models.Image {
	return createTestImageWithStatus(userID, filename, alt, weight, tags, "active")
}

func createTestImageWithStatus(userID uuid.UUID, filename, alt string, weight int, tags, status string) models.Image {
	width := 1920
	height := 1080
	return models.Image{
		ID:          uuid.New(),
		Filename:    filename,
		Alt:         alt,
		Weight:      weight,
		StoragePath: fmt.Sprintf("/storage/%s", filename),
		MimeType:    "image/jpeg",
		FileSize:    int64(1024 * 1024), // 1MB
		Width:       &width,
		Height:      &height,
		Tags:        &tags,
		UploadedBy:  &userID,
		Status:      status,
	}
}

// testBasicRandomization tests the core Fisher-Yates functionality
func testBasicRandomization(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("NoEmptyResults", func(t *testing.T) {
		// Get active images from database
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)
		require.NotEmpty(t, images, "Should have active images in test database")

		// Convert to weighted items
		weightedItems := convertImagesToWeightedItems(images)

		// Test selection of 3 images
		result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 3, "test_seed_123")
		require.NoError(t, err)
		assert.Len(t, result, 3, "Should return exactly 3 images")

		// Verify no duplicates in single selection
		seenIDs := make(map[string]bool)
		for _, item := range result {
			assert.False(t, seenIDs[item.ID], "Should not have duplicate images in result")
			seenIDs[item.ID] = true
		}
	})

	t.Run("FisherYatesShuffleCorrectness", func(t *testing.T) {
		// Create a small controlled set for testing
		items := []randomizer.WeightedItem{
			{ID: "img1", Weight: 1},
			{ID: "img2", Weight: 1},
			{ID: "img3", Weight: 1},
			{ID: "img4", Weight: 1},
			{ID: "img5", Weight: 1},
		}

		// Test multiple runs with different seeds should produce different orders
		seeds := []string{"seed1", "seed2", "seed3", "seed4", "seed5"}
		results := make([][]randomizer.WeightedItem, len(seeds))

		for i, seed := range seeds {
			result, err := randomizerSvc.SelectWeightedRandom(items, 5, seed)
			require.NoError(t, err)
			results[i] = result
		}

		// At least some results should be in different orders
		differentOrderCount := 0
		for i := 1; i < len(results); i++ {
			if !areItemOrdersEqual(results[0], results[i]) {
				differentOrderCount++
			}
		}

		assert.Greater(t, differentOrderCount, 0, "Different seeds should produce different orderings")
	})
}

// testWeightedSelectionDistribution verifies that weights affect selection probability
func testWeightedSelectionDistribution(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("WeightInfluencesSelection", func(t *testing.T) {
		// Create controlled test items with known weights
		items := []randomizer.WeightedItem{
			{ID: "low_weight", Weight: 1, Data: "weight_1"},
			{ID: "med_weight", Weight: 5, Data: "weight_5"},
			{ID: "high_weight", Weight: 10, Data: "weight_10"},
		}

		// Run many selections and count how often each weight appears
		const iterations = 1000
		selections := make(map[string]int)

		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("weight_test_%d", i)
			result, err := randomizerSvc.SelectWeightedRandom(items, 1, seed)
			require.NoError(t, err)
			require.Len(t, result, 1)

			selections[result[0].ID]++
		}

		// Verify that higher weights are selected more frequently
		assert.Greater(t, selections["high_weight"], selections["med_weight"],
			"Weight 10 items should be selected more than weight 5")
		assert.Greater(t, selections["med_weight"], selections["low_weight"],
			"Weight 5 items should be selected more than weight 1")

		// Log the actual distribution for debugging
		t.Logf("Selection distribution: low=%d, med=%d, high=%d",
			selections["low_weight"], selections["med_weight"], selections["high_weight"])
	})

	t.Run("RealDatabaseWeightDistribution", func(t *testing.T) {
		// Get images from database with different weights
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)
		require.NotEmpty(t, images)

		weightedItems := convertImagesToWeightedItems(images)

		// Group by weight for analysis
		weightGroups := make(map[int][]randomizer.WeightedItem)
		for _, item := range weightedItems {
			weightGroups[item.Weight] = append(weightGroups[item.Weight], item)
		}

		// Run selections and analyze distribution
		const iterations = 500
		selections := make(map[int]int)

		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("db_weight_test_%d", i)
			result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 1, seed)
			require.NoError(t, err)
			require.Len(t, result, 1)

			selections[result[0].Weight]++
		}

		// Verify distribution makes sense relative to weights
		weights := []int{1, 5, 10}
		for i := 1; i < len(weights); i++ {
			if selections[weights[i-1]] > 0 && selections[weights[i]] > 0 {
				ratio := float64(selections[weights[i]]) / float64(selections[weights[i-1]])
				assert.Greater(t, ratio, 1.0,
					"Higher weights should have higher selection frequency")
			}
		}

		t.Logf("Weight distribution from database: %+v", selections)
	})
}

// testSeedReproducibility ensures same seed produces same results
func testSeedReproducibility(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("IdenticalSeedsProduceIdenticalResults", func(t *testing.T) {
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)
		require.NotEmpty(t, images)

		weightedItems := convertImagesToWeightedItems(images)
		seed := "reproducibility_test_seed_12345"

		// Run selection multiple times with same seed
		const iterations = 10
		results := make([][]randomizer.WeightedItem, iterations)

		for i := 0; i < iterations; i++ {
			result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 3, seed)
			require.NoError(t, err)
			results[i] = result
		}

		// All results should be identical
		for i := 1; i < iterations; i++ {
			assert.True(t, areItemOrdersEqual(results[0], results[i]),
				"Same seed should produce identical results (iteration %d)", i)
		}
	})

	t.Run("DifferentSeedsProduceDifferentResults", func(t *testing.T) {
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)
		require.NotEmpty(t, images)

		weightedItems := convertImagesToWeightedItems(images)

		// Generate results with different seeds
		seeds := []string{"seed_a", "seed_b", "seed_c", "seed_d", "seed_e"}
		results := make([][]randomizer.WeightedItem, len(seeds))

		for i, seed := range seeds {
			result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 3, seed)
			require.NoError(t, err)
			results[i] = result
		}

		// At least some results should be different
		differentResults := 0
		for i := 1; i < len(results); i++ {
			if !areItemOrdersEqual(results[0], results[i]) {
				differentResults++
			}
		}

		assert.Greater(t, differentResults, 0, "Different seeds should produce different results")
	})

	t.Run("ValidateSeedFunction", func(t *testing.T) {
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)
		require.NotEmpty(t, images)

		weightedItems := convertImagesToWeightedItems(images)
		seed := "validation_test_seed"

		// Test the built-in validation function
		err = randomizerSvc.ValidateSeed(weightedItems, 3, seed, 5)
		assert.NoError(t, err, "Seed validation should pass for consistent seed")
	})
}

// testStatisticalValidation performs comprehensive statistical analysis
func testStatisticalValidation(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("DistributionFairnessAnalysis", func(t *testing.T) {
		// Create test set with known distribution
		items := []randomizer.WeightedItem{
			{ID: "item1", Weight: 2},
			{ID: "item2", Weight: 2},
			{ID: "item3", Weight: 6}, // Should appear ~3x more than items 1&2
		}

		const iterations = 2000
		selections := make(map[string]int)

		// Run many selections
		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("fairness_test_%d", i)
			result, err := randomizerSvc.SelectWeightedRandom(items, 1, seed)
			require.NoError(t, err)
			require.Len(t, result, 1)

			selections[result[0].ID]++
		}

		// Calculate expected vs actual distribution
		totalWeight := 10.0 // 2 + 2 + 6
		expectedItem1 := iterations * (2.0 / totalWeight)
		expectedItem2 := iterations * (2.0 / totalWeight)
		expectedItem3 := iterations * (6.0 / totalWeight)

		// Allow for statistical variance (within 15% of expected)
		tolerance := 0.15
		assert.InDelta(t, expectedItem1, selections["item1"], expectedItem1*tolerance,
			"Item1 selection frequency should be within tolerance")
		assert.InDelta(t, expectedItem2, selections["item2"], expectedItem2*tolerance,
			"Item2 selection frequency should be within tolerance")
		assert.InDelta(t, expectedItem3, selections["item3"], expectedItem3*tolerance,
			"Item3 selection frequency should be within tolerance")

		t.Logf("Expected: item1=%.0f, item2=%.0f, item3=%.0f",
			expectedItem1, expectedItem2, expectedItem3)
		t.Logf("Actual: item1=%d, item2=%d, item3=%d",
			selections["item1"], selections["item2"], selections["item3"])
	})

	t.Run("ChiSquareGoodnessOfFit", func(t *testing.T) {
		// Test using chi-square to validate distribution quality
		items := []randomizer.WeightedItem{
			{ID: "a", Weight: 1},
			{ID: "b", Weight: 1},
			{ID: "c", Weight: 1},
			{ID: "d", Weight: 1},
		}

		const iterations = 1000
		selections := make(map[string]int)

		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("chi_square_%d", i)
			result, err := randomizerSvc.SelectWeightedRandom(items, 1, seed)
			require.NoError(t, err)
			selections[result[0].ID]++
		}

		// For uniform distribution, expected frequency is iterations/4
		expectedFreq := float64(iterations) / 4.0
		chiSquare := 0.0

		for _, observed := range selections {
			diff := float64(observed) - expectedFreq
			chiSquare += (diff * diff) / expectedFreq
		}

		// For 3 degrees of freedom and 95% confidence, critical value is ~7.815
		// We use a more lenient threshold since this is a randomization test
		assert.Less(t, chiSquare, 12.0, "Chi-square value suggests distribution is reasonably uniform")

		t.Logf("Chi-square value: %.2f (expected ~7.815 for uniform distribution)", chiSquare)
	})
}

// testEdgeCases validates handling of edge cases
func testEdgeCases(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("EmptyDatabase", func(t *testing.T) {
		emptyItems := []randomizer.WeightedItem{}
		result, err := randomizerSvc.SelectWeightedRandom(emptyItems, 3, "empty_test")

		require.NoError(t, err)
		assert.Empty(t, result, "Should return empty result for empty input")
	})

	t.Run("SingleImage", func(t *testing.T) {
		singleItem := []randomizer.WeightedItem{
			{ID: "only_image", Weight: 5, Data: "single"},
		}

		result, err := randomizerSvc.SelectWeightedRandom(singleItem, 3, "single_test")
		require.NoError(t, err)
		assert.Len(t, result, 1, "Should return single item when only one available")
		assert.Equal(t, "only_image", result[0].ID)
	})

	t.Run("RequestMoreThanAvailable", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "img1", Weight: 1},
			{ID: "img2", Weight: 1},
		}

		result, err := randomizerSvc.SelectWeightedRandom(items, 5, "more_than_available")
		require.NoError(t, err)
		assert.Len(t, result, 2, "Should return all available items when requesting more than available")
	})

	t.Run("ZeroCount", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "img1", Weight: 5},
		}

		result, err := randomizerSvc.SelectWeightedRandom(items, 0, "zero_count")
		require.NoError(t, err)
		assert.Empty(t, result, "Should return empty result for zero count")
	})

	t.Run("AllSameWeight", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "img1", Weight: 5},
			{ID: "img2", Weight: 5},
			{ID: "img3", Weight: 5},
			{ID: "img4", Weight: 5},
		}

		// Should still work with uniform weights
		result, err := randomizerSvc.SelectWeightedRandom(items, 2, "same_weight")
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Verify no duplicates
		seenIDs := make(map[string]bool)
		for _, item := range result {
			assert.False(t, seenIDs[item.ID], "Should not have duplicates")
			seenIDs[item.ID] = true
		}
	})

	t.Run("OnlyInactiveImages", func(t *testing.T) {
		// Test database query filtering for active images only
		var inactiveImages []models.Image
		err := testDB.DB.Where("status != ?", "active").Find(&inactiveImages).Error
		require.NoError(t, err)

		if len(inactiveImages) > 0 {
			weightedItems := convertImagesToWeightedItems(inactiveImages)
			result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 3, "inactive_test")
			require.NoError(t, err)
			// Should still work, just with inactive images
			assert.LessOrEqual(t, len(result), len(inactiveImages))
		}
	})
}

// testPerformanceWithLargeDatasets validates performance characteristics
func testPerformanceWithLargeDatasets(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("LargeDatasetPerformance", func(t *testing.T) {
		// Create a large dataset for performance testing
		const imageCount = 1000
		largeDataset := make([]randomizer.WeightedItem, imageCount)

		for i := 0; i < imageCount; i++ {
			largeDataset[i] = randomizer.WeightedItem{
				ID:     fmt.Sprintf("perf_img_%d", i),
				Weight: (i % 10) + 1, // Weights 1-10
				Data:   fmt.Sprintf("performance_test_image_%d", i),
			}
		}

		// Measure selection time
		start := time.Now()
		result, err := randomizerSvc.SelectWeightedRandom(largeDataset, 5, "performance_test")
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, result, 5)

		// Should complete within reasonable time (< 10ms for 1000 images)
		assert.Less(t, duration.Milliseconds(), int64(10),
			"Selection should complete quickly even with large dataset")

		t.Logf("Performance test: %d images selected in %v",
			len(result), duration)
	})

	t.Run("MultipleSelectionsPerformance", func(t *testing.T) {
		// Test multiple selections performance
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)

		if len(images) == 0 {
			t.Skip("No active images for performance test")
		}

		weightedItems := convertImagesToWeightedItems(images)

		// Run multiple selections and measure average time
		const iterations = 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("multi_perf_%d", i)
			_, err := randomizerSvc.SelectWeightedRandom(weightedItems, 3, seed)
			require.NoError(t, err)
		}

		totalDuration := time.Since(start)
		averageDuration := totalDuration / iterations

		t.Logf("Average selection time over %d iterations: %v", iterations, averageDuration)

		// Should average less than 1ms per selection
		assert.Less(t, averageDuration.Milliseconds(), int64(1),
			"Average selection time should be very fast")
	})
}

// testDatabaseIntegrationQueries tests real database integration
func testDatabaseIntegrationQueries(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("ActiveImagesOnlyQuery", func(t *testing.T) {
		var activeImages []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&activeImages).Error
		require.NoError(t, err)

		var allImages []models.Image
		err = testDB.DB.Find(&allImages).Error
		require.NoError(t, err)

		// Should have fewer active images than total images
		assert.Less(t, len(activeImages), len(allImages),
			"Should filter out inactive images")

		// Convert to weighted items and test selection
		weightedItems := convertImagesToWeightedItems(activeImages)
		result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 3, "active_only")
		require.NoError(t, err)

		// All results should be from active images
		for _, item := range result {
			found := false
			for _, activeImg := range activeImages {
				if activeImg.ID.String() == item.ID {
					found = true
					assert.Equal(t, "active", activeImg.Status)
					break
				}
			}
			assert.True(t, found, "Selected image should be from active images list")
		}
	})

	t.Run("WeightRangeValidation", func(t *testing.T) {
		var images []models.Image
		err := testDB.DB.Where("status = ?", "active").Find(&images).Error
		require.NoError(t, err)

		// All weights should be in valid range (1-10)
		for _, img := range images {
			assert.GreaterOrEqual(t, img.Weight, 1, "Weight should be at least 1")
			assert.LessOrEqual(t, img.Weight, 10, "Weight should be at most 10")
		}

		// Test that randomizer handles these properly
		weightedItems := convertImagesToWeightedItems(images)
		result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 2, "weight_validation")
		require.NoError(t, err)

		for _, item := range result {
			assert.GreaterOrEqual(t, item.Weight, 1)
			assert.LessOrEqual(t, item.Weight, 10)
		}
	})
}

// testUserSpecificImageCollections tests user-based filtering
func testUserSpecificImageCollections(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("UserSpecificFiltering", func(t *testing.T) {
		// Get images for specific user
		var user models.User
		err := testDB.DB.First(&user).Error
		require.NoError(t, err)

		var userImages []models.Image
		err = testDB.DB.Where("uploaded_by = ? AND status = ?", user.ID, "active").Find(&userImages).Error
		require.NoError(t, err)

		if len(userImages) == 0 {
			t.Skip("No images for user to test with")
		}

		// Convert to weighted items and test selection
		weightedItems := convertImagesToWeightedItems(userImages)
		result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 2, "user_specific")
		require.NoError(t, err)

		// Verify all results belong to the specific user
		for _, item := range result {
			found := false
			for _, userImg := range userImages {
				if userImg.ID.String() == item.ID {
					found = true
					assert.Equal(t, user.ID, *userImg.UploadedBy)
					break
				}
			}
			assert.True(t, found, "Selected image should belong to specific user")
		}
	})

	t.Run("MultiUserDistribution", func(t *testing.T) {
		// Test selection across multiple users
		var images []models.Image
		err := testDB.DB.Where("status = ? AND uploaded_by IS NOT NULL", "active").Find(&images).Error
		require.NoError(t, err)

		if len(images) < 2 {
			t.Skip("Need at least 2 images from different users")
		}

		weightedItems := convertImagesToWeightedItems(images)

		// Run multiple selections to see if we get images from different users
		const iterations = 50
		userSet := make(map[string]bool)

		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("multi_user_%d", i)
			result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 1, seed)
			require.NoError(t, err)
			require.Len(t, result, 1)

			// Find the user for this image
			var img models.Image
			err = testDB.DB.Where("id = ?", result[0].ID).First(&img).Error
			require.NoError(t, err)

			if img.UploadedBy != nil {
				userSet[img.UploadedBy.String()] = true
			}
		}

		// Should have selected images from multiple users over many iterations
		assert.Greater(t, len(userSet), 1, "Should select images from multiple users over iterations")
		t.Logf("Selected images from %d different users", len(userSet))
	})
}

// testTagFilteringIntegration tests tag-based filtering
func testTagFilteringIntegration(t *testing.T, testDB *db.Database, randomizerSvc *randomizer.Service) {
	t.Run("TagBasedFiltering", func(t *testing.T) {
		// Test with nature-tagged images
		var natureImages []models.Image
		err := testDB.DB.Where("status = ? AND tags LIKE ?", "active", "%nature%").Find(&natureImages).Error
		require.NoError(t, err)

		if len(natureImages) == 0 {
			t.Skip("No nature-tagged images to test with")
		}

		weightedItems := convertImagesToWeightedItems(natureImages)
		result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 2, "nature_filter")
		require.NoError(t, err)

		// Verify all results have nature tag
		for _, item := range result {
			var img models.Image
			err = testDB.DB.Where("id = ?", item.ID).First(&img).Error
			require.NoError(t, err)

			if img.Tags != nil {
				assert.Contains(t, *img.Tags, "nature", "Selected image should have nature tag")
			}
		}
	})

	t.Run("MultipleTagFiltering", func(t *testing.T) {
		// Test with edge case tagged images
		var edgeImages []models.Image
		err := testDB.DB.Where("status = ? AND tags LIKE ?", "active", "%edge%").Find(&edgeImages).Error
		require.NoError(t, err)

		if len(edgeImages) > 0 {
			weightedItems := convertImagesToWeightedItems(edgeImages)
			result, err := randomizerSvc.SelectWeightedRandom(weightedItems, 1, "edge_filter")
			require.NoError(t, err)

			if len(result) > 0 {
				// Verify tag filtering worked
				var img models.Image
				err = testDB.DB.Where("id = ?", result[0].ID).First(&img).Error
				require.NoError(t, err)

				if img.Tags != nil {
					assert.Contains(t, *img.Tags, "edge", "Selected image should have edge tag")
				}
			}
		}
	})
}

// Helper functions

// convertImagesToWeightedItems converts database Image models to randomizer WeightedItems
func convertImagesToWeightedItems(images []models.Image) []randomizer.WeightedItem {
	items := make([]randomizer.WeightedItem, len(images))
	for i, img := range images {
		items[i] = randomizer.WeightedItem{
			ID:     img.ID.String(),
			Weight: img.Weight,
			Data:   img, // Store full image model in Data field
		}
	}
	return items
}

// areItemOrdersEqual compares two slices of WeightedItem for identical order
func areItemOrdersEqual(a, b []randomizer.WeightedItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			return false
		}
	}
	return true
}