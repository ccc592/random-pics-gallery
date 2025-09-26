package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/handlers"
	"github.com/randompic/api/internal/image"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

func TestHealthContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup test database and storage
	// Using PostgreSQL test database or skip if not available
	dbConfig := &db.Config{
		Driver:   "postgres",
		DSN:      "host=localhost port=5432 user=postgres password=password dbname=randompic_test sslmode=disable",
		LogLevel: logger.Silent,
	}
	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		t.Fatal("Failed to setup test database:", err)
	}
	defer database.Close()

	// Auto-migrate tables
	database.AutoMigrate()

	// Setup test storage
	storageConfig := &image.StorageConfig{
		Backend:   "local",
		LocalPath: "./test_uploads",
	}
	storage, err := image.NewStorageFromConfig(storageConfig)
	if err != nil {
		t.Fatal("Failed to setup test storage:", err)
	}

	// Setup actual handler
	healthHandler := handlers.NewHealthHandler(database, storage)
	router.GET("/api/health", healthHandler.HealthCheck)

	t.Run("Health check endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should now return 200 OK with actual implementation
		assert.Equal(t, http.StatusOK, w.Code)

		// Should validate this structure:
		var response struct {
			Status   string `json:"status"`
			Database struct {
				Status string `json:"status"`
			} `json:"database"`
			Storage struct {
				Status string `json:"status"`
			} `json:"storage"`
			Timestamp string `json:"timestamp"`
		}

		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// These assertions should now pass
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "healthy", response.Database.Status)
		assert.Equal(t, "healthy", response.Storage.Status)
		assert.NotEmpty(t, response.Timestamp)
	})
}