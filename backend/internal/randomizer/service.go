package randomizer

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
)

// WeightedItem represents an item with a weight for random selection
type WeightedItem struct {
	ID     string      `json:"id"`
	Weight int         `json:"weight"`
	Data   interface{} `json:"data,omitempty"` // Can hold any additional data
}

// Service handles weighted random selection using Fisher-Yates algorithm
type Service struct {
	// No fields needed for stateless service
}

// NewService creates a new randomizer service
func NewService() *Service {
	return &Service{}
}

// SelectWeightedRandom selects a specified count of items using weighted random selection
// with Fisher-Yates shuffle algorithm and deterministic seeding
func (s *Service) SelectWeightedRandom(items []WeightedItem, count int, seed string) ([]WeightedItem, error) {
	if len(items) == 0 {
		return []WeightedItem{}, nil
	}

	if count <= 0 {
		return []WeightedItem{}, nil
	}

	if count >= len(items) {
		// If requesting all or more items, return all items shuffled
		return s.shuffleItems(items, seed), nil
	}

	// Create weighted pool based on item weights
	weightedPool := s.createWeightedPool(items)

	// Create deterministic random source from seed
	rng := s.createSeededRNG(seed)

	// Select items using weighted selection
	selectedItems := make([]WeightedItem, 0, count)
	usedIndices := make(map[int]bool)

	for len(selectedItems) < count {
		// Select random index from weighted pool
		selectedIndex := s.selectWeightedIndex(weightedPool, rng)

		// Find the original item this index corresponds to
		originalIndex := s.findOriginalIndex(weightedPool, selectedIndex)

		// Skip if already selected
		if usedIndices[originalIndex] {
			continue
		}

		usedIndices[originalIndex] = true
		selectedItems = append(selectedItems, items[originalIndex])
	}

	// Apply Fisher-Yates shuffle to the selected items for final order
	return s.shuffleItems(selectedItems, seed), nil
}

// createWeightedPool creates a weighted pool where each item appears according to its weight
func (s *Service) createWeightedPool(items []WeightedItem) []int {
	var pool []int

	for i, item := range items {
		weight := item.Weight
		if weight < 1 {
			weight = 1 // Minimum weight of 1
		}

		// Add the item index to pool 'weight' number of times
		for j := 0; j < weight; j++ {
			pool = append(pool, i)
		}
	}

	return pool
}

// createSeededRNG creates a deterministic random number generator from a seed string
func (s *Service) createSeededRNG(seed string) *rand.Rand {
	// Hash the seed string to get consistent numeric seed
	hash := sha256.Sum256([]byte(seed))

	// Convert first 8 bytes of hash to int64 for seed
	numericSeed := int64(binary.BigEndian.Uint64(hash[:8]))

	// Create new random source with the seed
	source := rand.NewSource(numericSeed)
	return rand.New(source)
}

// selectWeightedIndex selects a random index from the weighted pool
func (s *Service) selectWeightedIndex(pool []int, rng *rand.Rand) int {
	if len(pool) == 0 {
		return 0
	}
	return rng.Intn(len(pool))
}

// findOriginalIndex finds which original item index the pool index corresponds to
func (s *Service) findOriginalIndex(pool []int, poolIndex int) int {
	if poolIndex >= len(pool) {
		return 0
	}
	return pool[poolIndex]
}

// shuffleItems applies Fisher-Yates shuffle to items using deterministic seed
func (s *Service) shuffleItems(items []WeightedItem, seed string) []WeightedItem {
	if len(items) <= 1 {
		return items
	}

	// Create a copy to avoid modifying original slice
	shuffled := make([]WeightedItem, len(items))
	copy(shuffled, items)

	// Create deterministic RNG
	rng := s.createSeededRNG(seed + "_shuffle") // Different seed for shuffle

	// Apply Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

// SelectUniformRandom selects items with uniform probability (ignoring weights)
func (s *Service) SelectUniformRandom(items []WeightedItem, count int, seed string) ([]WeightedItem, error) {
	if len(items) == 0 {
		return []WeightedItem{}, nil
	}

	if count <= 0 {
		return []WeightedItem{}, nil
	}

	if count >= len(items) {
		return s.shuffleItems(items, seed), nil
	}

	// Shuffle all items first
	shuffled := s.shuffleItems(items, seed)

	// Return first 'count' items
	return shuffled[:count], nil
}

// GetWeightDistribution returns statistics about weight distribution
func (s *Service) GetWeightDistribution(items []WeightedItem) map[string]interface{} {
	if len(items) == 0 {
		return map[string]interface{}{
			"total_items":     0,
			"total_weight":    0,
			"average_weight":  0.0,
			"weight_histogram": map[int]int{},
		}
	}

	totalWeight := 0
	weightHistogram := make(map[int]int)
	weights := make([]int, len(items))

	for i, item := range items {
		weight := item.Weight
		if weight < 1 {
			weight = 1
		}

		weights[i] = weight
		totalWeight += weight
		weightHistogram[weight]++
	}

	// Sort weights for percentile calculations
	sort.Ints(weights)

	averageWeight := float64(totalWeight) / float64(len(items))

	return map[string]interface{}{
		"total_items":      len(items),
		"total_weight":     totalWeight,
		"average_weight":   averageWeight,
		"min_weight":       weights[0],
		"max_weight":       weights[len(weights)-1],
		"median_weight":    s.calculateMedian(weights),
		"weight_histogram": weightHistogram,
	}
}

// calculateMedian calculates the median of a sorted slice of integers
func (s *Service) calculateMedian(sortedWeights []int) float64 {
	n := len(sortedWeights)
	if n%2 == 0 {
		// Even number of elements
		return float64(sortedWeights[n/2-1]+sortedWeights[n/2]) / 2.0
	}
	// Odd number of elements
	return float64(sortedWeights[n/2])
}

// ValidateSeed checks if a seed produces consistent results
func (s *Service) ValidateSeed(items []WeightedItem, count int, seed string, iterations int) error {
	if iterations < 2 {
		return fmt.Errorf("need at least 2 iterations to validate consistency")
	}

	// Get first result
	firstResult, err := s.SelectWeightedRandom(items, count, seed)
	if err != nil {
		return fmt.Errorf("failed to get first result: %w", err)
	}

	// Test additional iterations
	for i := 1; i < iterations; i++ {
		result, err := s.SelectWeightedRandom(items, count, seed)
		if err != nil {
			return fmt.Errorf("failed to get result for iteration %d: %w", i, err)
		}

		// Check if results are identical
		if !s.areResultsEqual(firstResult, result) {
			return fmt.Errorf("seed validation failed: iteration %d produced different result", i)
		}
	}

	return nil
}

// areResultsEqual checks if two selection results are identical
func (s *Service) areResultsEqual(a, b []WeightedItem) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].ID != b[i].ID || a[i].Weight != b[i].Weight {
			return false
		}
	}

	return true
}

// GenerateSessionSeed generates a new session seed based on current timestamp and random data
func (s *Service) GenerateSessionSeed() string {
	// Use crypto/rand for true randomness when generating new seeds
	rng := rand.New(rand.NewSource(rand.Int63()))
	return fmt.Sprintf("%d_%d", rng.Int63(), rng.Int63())
}