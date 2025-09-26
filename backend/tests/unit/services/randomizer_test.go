package services_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/randompic/api/internal/randomizer"
)

func TestFisherYatesAlgorithm(t *testing.T) {
	service := randomizer.NewService()

	t.Run("BasicFisherYatesSelection", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1, Data: "Item 1"},
			{ID: "2", Weight: 1, Data: "Item 2"},
			{ID: "3", Weight: 1, Data: "Item 3"},
			{ID: "4", Weight: 1, Data: "Item 4"},
			{ID: "5", Weight: 1, Data: "Item 5"},
		}

		// Select 3 items
		result, err := service.SelectWeightedRandom(items, 3, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 3)

		// Ensure all selected items are from the original set
		originalIDs := map[string]bool{"1": true, "2": true, "3": true, "4": true, "5": true}
		for _, item := range result {
			assert.True(t, originalIDs[item.ID], "Selected item %s should be from original set", item.ID)
		}

		// Ensure no duplicates
		selectedIDs := make(map[string]bool)
		for _, item := range result {
			assert.False(t, selectedIDs[item.ID], "Item %s should not be selected twice", item.ID)
			selectedIDs[item.ID] = true
		}
	})

	t.Run("EmptyItemSet", func(t *testing.T) {
		items := []randomizer.WeightedItem{}
		result, err := service.SelectWeightedRandom(items, 3, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("ZeroCount", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
		}
		result, err := service.SelectWeightedRandom(items, 0, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("NegativeCount", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
		}
		result, err := service.SelectWeightedRandom(items, -1, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("CountExceedsItemCount", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
		}
		result, err := service.SelectWeightedRandom(items, 5, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 2) // Should return all available items
	})

	t.Run("UniformRandomSelection", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
			{ID: "3", Weight: 1},
			{ID: "4", Weight: 1},
			{ID: "5", Weight: 1},
		}

		result, err := service.SelectUniformRandom(items, 3, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 3)

		// Ensure no duplicates
		selectedIDs := make(map[string]bool)
		for _, item := range result {
			assert.False(t, selectedIDs[item.ID], "Item %s should not be selected twice", item.ID)
			selectedIDs[item.ID] = true
		}
	})
}

func TestWeightedSelection(t *testing.T) {
	service := randomizer.NewService()

	t.Run("WeightedProbabilityDistribution", func(t *testing.T) {
		// Items with different weights
		items := []randomizer.WeightedItem{
			{ID: "heavy", Weight: 10},   // Should be selected more often
			{ID: "medium", Weight: 5},   // Medium frequency
			{ID: "light", Weight: 1},    // Should be selected less often
		}

		// Run multiple selections to test distribution
		selections := make(map[string]int)
		iterations := 1000

		for i := 0; i < iterations; i++ {
			seed := fmt.Sprintf("seed_%d", i)
			result, err := service.SelectWeightedRandom(items, 1, seed)
			require.NoError(t, err)
			require.Len(t, result, 1)
			selections[result[0].ID]++
		}

		// Heavy weight item should be selected most frequently
		// With weights 10:5:1, expected ratio is roughly 10:5:1
		assert.True(t, selections["heavy"] > selections["medium"],
			"Heavy item should be selected more than medium (heavy: %d, medium: %d)",
			selections["heavy"], selections["medium"])
		assert.True(t, selections["medium"] > selections["light"],
			"Medium item should be selected more than light (medium: %d, light: %d)",
			selections["medium"], selections["light"])

		// Rough distribution check (allowing for randomness)
		totalSelections := selections["heavy"] + selections["medium"] + selections["light"]
		heavyPercentage := float64(selections["heavy"]) / float64(totalSelections) * 100

		// Heavy item (weight 10 out of 16 total) should be around 62.5%
		assert.InDelta(t, 62.5, heavyPercentage, 15.0, // Allow 15% deviation due to randomness
			"Heavy item percentage should be around 62.5%%, got %.1f%%", heavyPercentage)
	})

	t.Run("ZeroWeightHandling", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 0}, // Should be treated as weight 1
			{ID: "2", Weight: 1},
		}

		result, err := service.SelectWeightedRandom(items, 2, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Both items should be selectable
		ids := make([]string, len(result))
		for i, item := range result {
			ids[i] = item.ID
		}
		assert.Contains(t, ids, "1")
		assert.Contains(t, ids, "2")
	})

	t.Run("NegativeWeightHandling", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: -5}, // Should be treated as weight 1
			{ID: "2", Weight: 3},
		}

		result, err := service.SelectWeightedRandom(items, 2, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("HighWeightItems", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "max_weight", Weight: 10}, // Maximum allowed weight
			{ID: "min_weight", Weight: 1},  // Minimum weight
		}

		result, err := service.SelectWeightedRandom(items, 1, "test_seed")
		require.NoError(t, err)
		assert.Len(t, result, 1)

		// Should be able to select either item
		assert.Contains(t, []string{"max_weight", "min_weight"}, result[0].ID)
	})
}

func TestSeedReproducibility(t *testing.T) {
	service := randomizer.NewService()

	t.Run("DeterministicResults", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 2},
			{ID: "3", Weight: 3},
			{ID: "4", Weight: 4},
			{ID: "5", Weight: 5},
		}

		seed := "consistent_seed"
		count := 3

		// Get first result
		result1, err := service.SelectWeightedRandom(items, count, seed)
		require.NoError(t, err)

		// Get second result with same seed
		result2, err := service.SelectWeightedRandom(items, count, seed)
		require.NoError(t, err)

		// Results should be identical
		require.Len(t, result1, len(result2))
		for i, item := range result1 {
			assert.Equal(t, item.ID, result2[i].ID, "Item at position %d should be the same", i)
			assert.Equal(t, item.Weight, result2[i].Weight, "Weight at position %d should be the same", i)
		}
	})

	t.Run("DifferentSeedsDifferentResults", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
			{ID: "3", Weight: 1},
			{ID: "4", Weight: 1},
			{ID: "5", Weight: 1},
		}

		result1, err := service.SelectWeightedRandom(items, 3, "seed1")
		require.NoError(t, err)

		result2, err := service.SelectWeightedRandom(items, 3, "seed2")
		require.NoError(t, err)

		// Results should likely be different (though theoretically could be same)
		// Check at least that we got valid results
		assert.Len(t, result1, 3)
		assert.Len(t, result2, 3)
	})

	t.Run("SeedValidation", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 2},
			{ID: "3", Weight: 3},
		}

		err := service.ValidateSeed(items, 2, "test_seed", 5)
		assert.NoError(t, err, "Seed validation should pass for deterministic algorithm")
	})

	t.Run("SeedValidationInvalidIterations", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
		}

		err := service.ValidateSeed(items, 1, "test_seed", 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "need at least 2 iterations")
	})

	t.Run("GenerateSessionSeed", func(t *testing.T) {
		seed1 := service.GenerateSessionSeed()
		seed2 := service.GenerateSessionSeed()

		// Seeds should be non-empty and different
		assert.NotEmpty(t, seed1)
		assert.NotEmpty(t, seed2)
		assert.NotEqual(t, seed1, seed2, "Generated seeds should be different")

		// Seeds should contain underscores (based on format)
		assert.Contains(t, seed1, "_")
		assert.Contains(t, seed2, "_")
	})
}

func TestTagFiltering(t *testing.T) {
	service := randomizer.NewService()

	t.Run("WeightDistribution", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1, Data: "landscape"},
			{ID: "2", Weight: 5, Data: "portrait"},
			{ID: "3", Weight: 3, Data: "nature"},
			{ID: "4", Weight: 2, Data: "city"},
			{ID: "5", Weight: 1, Data: "animal"},
		}

		stats := service.GetWeightDistribution(items)

		require.NotNil(t, stats)
		assert.Equal(t, 5, stats["total_items"])
		assert.Equal(t, 12, stats["total_weight"]) // 1+5+3+2+1
		assert.InDelta(t, 2.4, stats["average_weight"], 0.1) // 12/5
		assert.Equal(t, 1, stats["min_weight"])
		assert.Equal(t, 5, stats["max_weight"])
		assert.InDelta(t, 2.0, stats["median_weight"], 0.1)

		histogram := stats["weight_histogram"].(map[int]int)
		assert.Equal(t, 2, histogram[1]) // Two items with weight 1
		assert.Equal(t, 1, histogram[2]) // One item with weight 2
		assert.Equal(t, 1, histogram[3]) // One item with weight 3
		assert.Equal(t, 1, histogram[5]) // One item with weight 5
	})

	t.Run("EmptyItemsDistribution", func(t *testing.T) {
		items := []randomizer.WeightedItem{}
		stats := service.GetWeightDistribution(items)

		assert.Equal(t, 0, stats["total_items"])
		assert.Equal(t, 0, stats["total_weight"])
		assert.Equal(t, 0.0, stats["average_weight"])

		histogram := stats["weight_histogram"].(map[int]int)
		assert.Empty(t, histogram)
	})

	t.Run("SingleItemDistribution", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 7},
		}

		stats := service.GetWeightDistribution(items)
		assert.Equal(t, 1, stats["total_items"])
		assert.Equal(t, 7, stats["total_weight"])
		assert.Equal(t, 7.0, stats["average_weight"])
		assert.Equal(t, 7, stats["min_weight"])
		assert.Equal(t, 7, stats["max_weight"])
		assert.Equal(t, 7.0, stats["median_weight"])
	})

	t.Run("EvenNumberMedian", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 2},
			{ID: "2", Weight: 4},
			{ID: "3", Weight: 6},
			{ID: "4", Weight: 8},
		}

		stats := service.GetWeightDistribution(items)
		// Median of [2,4,6,8] should be (4+6)/2 = 5.0
		assert.Equal(t, 5.0, stats["median_weight"])
	})
}

func TestRandomizerEdgeCases(t *testing.T) {
	service := randomizer.NewService()

	t.Run("SingleItem", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "only", Weight: 5},
		}

		result, err := service.SelectWeightedRandom(items, 1, "test")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "only", result[0].ID)
	})

	t.Run("RequestMoreThanAvailable", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
		}

		result, err := service.SelectWeightedRandom(items, 10, "test")
		require.NoError(t, err)
		assert.Len(t, result, 2) // Should return all available
	})

	t.Run("ExactCount", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1},
			{ID: "2", Weight: 1},
			{ID: "3", Weight: 1},
		}

		result, err := service.SelectWeightedRandom(items, 3, "test")
		require.NoError(t, err)
		assert.Len(t, result, 3) // Should return exactly 3
	})

	t.Run("LargeDataset", func(t *testing.T) {
		// Test with larger dataset
		items := make([]randomizer.WeightedItem, 100)
		for i := 0; i < 100; i++ {
			items[i] = randomizer.WeightedItem{
				ID:     fmt.Sprintf("item_%d", i),
				Weight: (i % 10) + 1, // Weights from 1 to 10
			}
		}

		result, err := service.SelectWeightedRandom(items, 20, "large_test")
		require.NoError(t, err)
		assert.Len(t, result, 20)

		// Ensure no duplicates
		seen := make(map[string]bool)
		for _, item := range result {
			assert.False(t, seen[item.ID], "No duplicates should exist")
			seen[item.ID] = true
		}
	})

	t.Run("WeightConsistency", func(t *testing.T) {
		items := []randomizer.WeightedItem{
			{ID: "1", Weight: 1, Data: "extra_data_1"},
			{ID: "2", Weight: 2, Data: "extra_data_2"},
		}

		result, err := service.SelectWeightedRandom(items, 2, "test")
		require.NoError(t, err)

		for _, item := range result {
			// Verify that weights are preserved
			if item.ID == "1" {
				assert.Equal(t, 1, item.Weight)
				assert.Equal(t, "extra_data_1", item.Data)
			} else if item.ID == "2" {
				assert.Equal(t, 2, item.Weight)
				assert.Equal(t, "extra_data_2", item.Data)
			}
		}
	})
}

func TestRandomizerServiceCreation(t *testing.T) {
	t.Run("NewService", func(t *testing.T) {
		service := randomizer.NewService()
		assert.NotNil(t, service)

		// Service should be usable immediately
		items := []randomizer.WeightedItem{
			{ID: "test", Weight: 1},
		}

		result, err := service.SelectWeightedRandom(items, 1, "test")
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

// Benchmark tests for performance validation
func BenchmarkSelectWeightedRandom(b *testing.B) {
	service := randomizer.NewService()
	items := make([]randomizer.WeightedItem, 1000)

	for i := 0; i < 1000; i++ {
		items[i] = randomizer.WeightedItem{
			ID:     fmt.Sprintf("item_%d", i),
			Weight: (i % 10) + 1,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.SelectWeightedRandom(items, 10, fmt.Sprintf("seed_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectUniformRandom(b *testing.B) {
	service := randomizer.NewService()
	items := make([]randomizer.WeightedItem, 1000)

	for i := 0; i < 1000; i++ {
		items[i] = randomizer.WeightedItem{
			ID:     fmt.Sprintf("item_%d", i),
			Weight: 1,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.SelectUniformRandom(items, 10, fmt.Sprintf("seed_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}
}