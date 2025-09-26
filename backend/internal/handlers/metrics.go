package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/image"
)

// MetricsHandler handles metrics and observability endpoints
type MetricsHandler struct {
	database     *db.Database
	imageService *image.Service
	startTime    time.Time
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(database *db.Database, imageService *image.Service) *MetricsHandler {
	return &MetricsHandler{
		database:     database,
		imageService: imageService,
		startTime:    time.Now(),
	}
}

// PrometheusMetrics handles GET /api/metrics (Prometheus format)
func (h *MetricsHandler) PrometheusMetrics(c *gin.Context) {
	// Set correct content type for Prometheus
	c.Header("Content-Type", "text/plain; charset=utf-8")

	var metrics []string

	// System metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics = append(metrics, []string{
		"# HELP process_start_time_seconds Start time of the process since unix epoch in seconds",
		"# TYPE process_start_time_seconds gauge",
		fmt.Sprintf("process_start_time_seconds %d", h.startTime.Unix()),
		"",
		"# HELP process_uptime_seconds Total uptime of the process in seconds",
		"# TYPE process_uptime_seconds counter",
		fmt.Sprintf("process_uptime_seconds %d", int64(time.Since(h.startTime).Seconds())),
		"",
		"# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use",
		"# TYPE go_memstats_alloc_bytes gauge",
		fmt.Sprintf("go_memstats_alloc_bytes %d", memStats.Alloc),
		"",
		"# HELP go_memstats_sys_bytes Number of bytes obtained from system",
		"# TYPE go_memstats_sys_bytes gauge",
		fmt.Sprintf("go_memstats_sys_bytes %d", memStats.Sys),
		"",
		"# HELP go_goroutines Number of goroutines that currently exist",
		"# TYPE go_goroutines gauge",
		fmt.Sprintf("go_goroutines %d", runtime.NumGoroutine()),
		"",
	}...)

	// HTTP metrics (would be collected by middleware in real implementation)
	metrics = append(metrics, []string{
		"# HELP http_requests_total Total number of HTTP requests",
		"# TYPE http_requests_total counter",
		"http_requests_total{method=\"GET\",status=\"200\"} 0",
		"http_requests_total{method=\"POST\",status=\"201\"} 0",
		"http_requests_total{method=\"PUT\",status=\"200\"} 0",
		"http_requests_total{method=\"DELETE\",status=\"200\"} 0",
		"",
		"# HELP http_request_duration_seconds HTTP request duration in seconds",
		"# TYPE http_request_duration_seconds histogram",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"0.1\"} 0",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"0.25\"} 0",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"0.5\"} 0",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"1.0\"} 0",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"2.5\"} 0",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"5.0\"} 0",
		"http_request_duration_seconds_bucket{method=\"GET\",le=\"+Inf\"} 0",
		"http_request_duration_seconds_sum{method=\"GET\"} 0",
		"http_request_duration_seconds_count{method=\"GET\"} 0",
		"",
	}...)

	// Database metrics
	if dbStats, err := h.database.GetStats(); err == nil {
		metrics = append(metrics, []string{
			"# HELP database_connections_open Current number of open database connections",
			"# TYPE database_connections_open gauge",
			fmt.Sprintf("database_connections_open %v", dbStats["open_connections"]),
			"",
			"# HELP database_connections_in_use Current number of database connections in use",
			"# TYPE database_connections_in_use gauge",
			fmt.Sprintf("database_connections_in_use %v", dbStats["in_use"]),
			"",
			"# HELP database_connections_idle Current number of idle database connections",
			"# TYPE database_connections_idle gauge",
			fmt.Sprintf("database_connections_idle %v", dbStats["idle"]),
			"",
		}...)
	}

	// Image service metrics
	if imageStats, err := h.imageService.GetImageStats(); err == nil {
		metrics = append(metrics, []string{
			"# HELP images_total Total number of images in the system",
			"# TYPE images_total gauge",
			fmt.Sprintf("images_total %v", imageStats["total_images"]),
			"",
			"# HELP images_active Number of active images",
			"# TYPE images_active gauge",
			fmt.Sprintf("images_active %v", imageStats["active_images"]),
			"",
			"# HELP images_processing Number of images currently being processed",
			"# TYPE images_processing gauge",
			fmt.Sprintf("images_processing %v", imageStats["processing_images"]),
			"",
			"# HELP images_failed Number of failed images",
			"# TYPE images_failed gauge",
			fmt.Sprintf("images_failed %v", imageStats["failed_images"]),
			"",
		}...)
	}

	// Custom business metrics
	metrics = append(metrics, []string{
		"# HELP images_served_total Total number of images served via random endpoint",
		"# TYPE images_served_total counter",
		"images_served_total 0",
		"",
		"# HELP random_requests_total Total number of random image requests",
		"# TYPE random_requests_total counter",
		"random_requests_total{count=\"3\"} 0",
		"random_requests_total{count=\"4\"} 0",
		"random_requests_total{count=\"5\"} 0",
		"",
		"# HELP upload_requests_total Total number of image upload requests",
		"# TYPE upload_requests_total counter",
		"upload_requests_total{status=\"success\"} 0",
		"upload_requests_total{status=\"failed\"} 0",
		"",
	}...)

	// Join all metrics
	output := ""
	for _, metric := range metrics {
		output += metric + "\n"
	}

	c.String(http.StatusOK, output)
}

// JSONMetrics handles GET /api/metrics/json (JSON format)
func (h *MetricsHandler) JSONMetrics(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	response := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"system": map[string]interface{}{
			"uptime_seconds":     int64(time.Since(h.startTime).Seconds()),
			"start_time":         h.startTime.Format(time.RFC3339),
			"memory_alloc":       memStats.Alloc,
			"memory_sys":         memStats.Sys,
			"goroutines":         runtime.NumGoroutine(),
			"gc_runs":            memStats.NumGC,
			"next_gc":            memStats.NextGC,
		},
		"http": map[string]interface{}{
			"requests_total": map[string]interface{}{
				"GET":    0,
				"POST":   0,
				"PUT":    0,
				"DELETE": 0,
			},
			"response_time_avg_ms": 0,
			"errors_total":         0,
		},
	}

	// Database metrics
	if dbStats, err := h.database.GetStats(); err == nil {
		response["database"] = dbStats
	}

	// Image metrics
	if imageStats, err := h.imageService.GetImageStats(); err == nil {
		response["images"] = imageStats
	}

	// Business metrics
	response["business"] = map[string]interface{}{
		"images_served_total":    0,
		"random_requests_total":  0,
		"upload_requests_total":  0,
		"download_requests_total": 0,
	}

	c.JSON(http.StatusOK, response)
}

// HealthMetrics handles GET /api/metrics/health
func (h *MetricsHandler) HealthMetrics(c *gin.Context) {
	response := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    "healthy",
		"checks": map[string]interface{}{
			"database": map[string]interface{}{
				"status": "healthy",
				"latency_ms": 0, // Would measure actual DB query time
			},
			"storage": map[string]interface{}{
				"status": "healthy",
				"latency_ms": 0, // Would measure actual storage operation time
			},
		},
		"performance": map[string]interface{}{
			"cpu_usage_percent":    0.0, // Would get actual CPU usage
			"memory_usage_percent": float64(runtime.NumGoroutine()) / 1000 * 100, // Mock calculation
			"disk_usage_percent":   0.0, // Would get actual disk usage
		},
	}

	// Check database health
	if err := h.database.Ping(); err != nil {
		response["status"] = "degraded"
		checks := response["checks"].(map[string]interface{})
		dbCheck := checks["database"].(map[string]interface{})
		dbCheck["status"] = "unhealthy"
		dbCheck["error"] = err.Error()
	}

	c.JSON(http.StatusOK, response)
}

// CustomMetrics handles GET /api/metrics/custom
func (h *MetricsHandler) CustomMetrics(c *gin.Context) {
	// This endpoint could provide application-specific metrics
	response := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"metrics": map[string]interface{}{
			"fisher_yates_selections": map[string]interface{}{
				"total_calls":       0,
				"avg_duration_ms":   0,
				"seed_cache_hits":   0,
				"seed_cache_misses": 0,
			},
			"image_processing": map[string]interface{}{
				"resizes_completed": 0,
				"format_conversions": map[string]interface{}{
					"to_webp": 0,
					"to_avif": 0,
					"to_jpeg": 0,
				},
				"metadata_extractions": 0,
			},
			"storage_operations": map[string]interface{}{
				"writes_total":       0,
				"reads_total":        0,
				"deletes_total":      0,
				"errors_total":       0,
				"avg_write_time_ms":  0,
				"avg_read_time_ms":   0,
			},
		},
	}

	c.JSON(http.StatusOK, response)
}