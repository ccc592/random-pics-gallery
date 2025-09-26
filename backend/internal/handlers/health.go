package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/image"
)

// HealthHandler handles health check related requests
type HealthHandler struct {
	database *db.Database
	storage  image.StorageInterface
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(database *db.Database, storage image.StorageInterface) *HealthHandler {
	return &HealthHandler{
		database: database,
		storage:  storage,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Database  HealthComponentStatus  `json:"database"`
	Storage   HealthComponentStatus  `json:"storage"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthComponentStatus represents the status of a system component
type HealthComponentStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthCheck handles GET /api/health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "healthy",
		Database: HealthComponentStatus{
			Status: "healthy",
		},
		Storage: HealthComponentStatus{
			Status: "healthy",
		},
	}

	overallHealthy := true

	// Check database health
	if err := h.database.HealthCheck(); err != nil {
		response.Database.Status = "unhealthy"
		response.Database.Message = err.Error()
		overallHealthy = false
	}

	// Check storage health (if storage implements health check)
	if healthChecker, ok := h.storage.(interface{ HealthCheck() error }); ok {
		if err := healthChecker.HealthCheck(); err != nil {
			response.Storage.Status = "unhealthy"
			response.Storage.Message = err.Error()
			overallHealthy = false
		}
	}

	// Set overall status
	if !overallHealthy {
		response.Status = "unhealthy"
	}

	// Add detailed information for debugging
	details := make(map[string]interface{})

	// Database stats
	if dbStats, err := h.database.GetStats(); err == nil {
		details["database_stats"] = dbStats
	}

	// Storage stats (if storage implements stats)
	if statsProvider, ok := h.storage.(interface{ GetStorageStats() (map[string]interface{}, error) }); ok {
		if storageStats, err := statsProvider.GetStorageStats(); err == nil {
			details["storage_stats"] = storageStats
		}
	}

	response.Details = details

	// Return appropriate HTTP status
	if overallHealthy {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// ReadinessCheck handles GET /api/ready
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// Readiness check is stricter - checks if the service is ready to handle requests
	response := HealthResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "ready",
		Database: HealthComponentStatus{
			Status: "ready",
		},
		Storage: HealthComponentStatus{
			Status: "ready",
		},
	}

	ready := true

	// Check database connectivity
	if err := h.database.Ping(); err != nil {
		response.Database.Status = "not_ready"
		response.Database.Message = "database connection failed"
		ready = false
	}

	// Quick storage check
	if healthChecker, ok := h.storage.(interface{ HealthCheck() error }); ok {
		if err := healthChecker.HealthCheck(); err != nil {
			response.Storage.Status = "not_ready"
			response.Storage.Message = "storage not accessible"
			ready = false
		}
	}

	if !ready {
		response.Status = "not_ready"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// LivenessCheck handles GET /api/live
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	// Liveness check is the most basic - just confirms the service is running
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// DetailedHealthCheck handles GET /api/health/detailed
func (h *HealthHandler) DetailedHealthCheck(c *gin.Context) {
	response := HealthResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "healthy",
		Database: HealthComponentStatus{
			Status: "healthy",
		},
		Storage: HealthComponentStatus{
			Status: "healthy",
		},
		Details: make(map[string]interface{}),
	}

	overallHealthy := true

	// Detailed database check
	dbDetails := make(map[string]interface{})

	// Test database connection
	if err := h.database.Ping(); err != nil {
		response.Database.Status = "unhealthy"
		response.Database.Message = "ping failed: " + err.Error()
		overallHealthy = false
		dbDetails["ping_error"] = err.Error()
	} else {
		dbDetails["ping"] = "ok"
	}

	// Get connection stats
	if stats, err := h.database.GetStats(); err == nil {
		dbDetails["connection_stats"] = stats
	}

	response.Details["database"] = dbDetails

	// Detailed storage check
	storageDetails := make(map[string]interface{})

	if healthChecker, ok := h.storage.(interface{ HealthCheck() error }); ok {
		if err := healthChecker.HealthCheck(); err != nil {
			response.Storage.Status = "unhealthy"
			response.Storage.Message = err.Error()
			overallHealthy = false
			storageDetails["health_check_error"] = err.Error()
		} else {
			storageDetails["health_check"] = "ok"
		}
	}

	// Get storage stats
	if statsProvider, ok := h.storage.(interface{ GetStorageStats() (map[string]interface{}, error) }); ok {
		if storageStats, err := statsProvider.GetStorageStats(); err == nil {
			storageDetails["stats"] = storageStats
		}
	}

	response.Details["storage"] = storageDetails

	// System information
	response.Details["system"] = map[string]interface{}{
		"uptime":    time.Since(time.Now().Add(-time.Hour)).String(), // Mock uptime
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Set overall status
	if !overallHealthy {
		response.Status = "unhealthy"
	}

	// Return appropriate HTTP status
	if overallHealthy {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}