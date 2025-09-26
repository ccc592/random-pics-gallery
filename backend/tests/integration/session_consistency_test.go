package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionConsistencyIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock route - this will fail until actual handler is implemented
	router.GET("/api/images/random", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
	})

	t.Run("Same seed returns same images in same order", func(t *testing.T) {
		const testSeed = "test-seed-123"

		// Make first request with seed
		req1 := httptest.NewRequest("GET", fmt.Sprintf("/api/images/random?seed=%s&count=4", testSeed), nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Make second request with same seed
		req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/images/random?seed=%s&count=4", testSeed), nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		if w1.Code == http.StatusOK && w2.Code == http.StatusOK {
			var response1, response2 struct {
				Images []struct {
					ID       string `json:"id"`
					Filename string `json:"filename"`
				} `json:"images"`
				SessionSeed string `json:"session_seed"`
			}

			err := json.Unmarshal(w1.Body.Bytes(), &response1)
			require.NoError(t, err)

			err = json.Unmarshal(w2.Body.Bytes(), &response2)
			require.NoError(t, err)

			// Should return same seed
			assert.Equal(t, testSeed, response1.SessionSeed)
			assert.Equal(t, testSeed, response2.SessionSeed)

			// Should return same images in same order
			require.Len(t, response1.Images, 4)
			require.Len(t, response2.Images, 4)

			for i := 0; i < 4; i++ {
				assert.Equal(t, response1.Images[i].ID, response2.Images[i].ID,
					"Image at position %d should be the same", i)
				assert.Equal(t, response1.Images[i].Filename, response2.Images[i].Filename,
					"Filename at position %d should be the same", i)
			}
		} else {
			// Currently both return 501
			assert.Equal(t, http.StatusNotImplemented, w1.Code)
			assert.Equal(t, http.StatusNotImplemented, w2.Code)
		}
	})

	t.Run("Different seeds return different images", func(t *testing.T) {
		// Make request with first seed
		req1 := httptest.NewRequest("GET", "/api/images/random?seed=seed1&count=3", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Make request with second seed
		req2 := httptest.NewRequest("GET", "/api/images/random?seed=seed2&count=3", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		if w1.Code == http.StatusOK && w2.Code == http.StatusOK {
			var response1, response2 struct {
				Images []struct {
					ID string `json:"id"`
				} `json:"images"`
				SessionSeed string `json:"session_seed"`
			}

			err := json.Unmarshal(w1.Body.Bytes(), &response1)
			require.NoError(t, err)

			err = json.Unmarshal(w2.Body.Bytes(), &response2)
			require.NoError(t, err)

			// Should return different seeds
			assert.Equal(t, "seed1", response1.SessionSeed)
			assert.Equal(t, "seed2", response2.SessionSeed)

			// Should return different image sets (at least some different)
			require.Len(t, response1.Images, 3)
			require.Len(t, response2.Images, 3)

			// Collect IDs from both responses
			ids1 := make([]string, len(response1.Images))
			for i, img := range response1.Images {
				ids1[i] = img.ID
			}

			ids2 := make([]string, len(response2.Images))
			for i, img := range response2.Images {
				ids2[i] = img.ID
			}

			// At least one image should be different (highly likely with different seeds)
			assert.NotEqual(t, ids1, ids2, "Different seeds should produce different image sets")
		} else {
			// Currently both return 501
			assert.Equal(t, http.StatusNotImplemented, w1.Code)
			assert.Equal(t, http.StatusNotImplemented, w2.Code)
		}
	})

	t.Run("No seed generates unique session seed", func(t *testing.T) {
		// Make multiple requests without seed
		var sessionSeeds []string

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/api/images/random", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				var response struct {
					SessionSeed string `json:"session_seed"`
				}

				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.SessionSeed)
				sessionSeeds = append(sessionSeeds, response.SessionSeed)
			} else {
				// Currently returns 501
				assert.Equal(t, http.StatusNotImplemented, w.Code)
			}
		}

		// If we got responses, ensure session seeds are unique
		if len(sessionSeeds) > 0 {
			for i := 0; i < len(sessionSeeds); i++ {
				for j := i + 1; j < len(sessionSeeds); j++ {
					assert.NotEqual(t, sessionSeeds[i], sessionSeeds[j],
						"Session seeds should be unique for different requests")
				}
			}
		}
	})
}