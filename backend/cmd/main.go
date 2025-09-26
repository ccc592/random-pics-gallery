package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/config"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/handlers"
	"github.com/randompic/api/internal/image"
	"github.com/randompic/api/internal/middleware"
	"github.com/randompic/api/internal/randomizer"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration validation failed:", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database
	database, err := db.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.AutoMigrate(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize services
	randomizerService := randomizer.NewService()

	// Initialize storage
	storage, err := image.NewStorageFromConfig(cfg.Storage)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	imageService := image.NewService(database.DB, randomizerService, storage)

	// Initialize auth service
	authService := auth.NewService(database.DB, cfg.Auth)

	// Initialize rate limiters
	var anonymousLimiter, authenticatedLimiter *middleware.RateLimiter
	if cfg.RateLimit.Enabled {
		anonymousLimiter = middleware.NewRateLimiter(cfg.RateLimit.Anonymous)
		authenticatedLimiter = middleware.NewRateLimiter(cfg.RateLimit.Authenticated)
		defer anonymousLimiter.Stop()
		defer authenticatedLimiter.Stop()
	}

	// Initialize handlers
	imageHandler := handlers.NewImageHandler(imageService, authService)
	healthHandler := handlers.NewHealthHandler(database, storage)
	metricsHandler := handlers.NewMetricsHandler(database, imageService)

	// Setup router with middleware
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery()) // Panic recovery
	router.Use(middleware.RequestIDMiddleware()) // Request ID
	router.Use(middleware.CORSMiddleware(cfg.CORS)) // CORS
	router.Use(middleware.SecurityHeaders()) // Security headers

	// Logging middleware
	if cfg.Logging.RequestLogging {
		router.Use(middleware.RequestLoggingMiddleware(database.DB))
		router.Use(middleware.PerformanceLoggingMiddleware(cfg.Logging.SlowThreshold))
		router.Use(middleware.ErrorLoggingMiddleware())
	}

	// JSON logging in production
	if cfg.Server.IsProduction() {
		router.Use(middleware.JSONLoggingMiddleware())
	} else {
		router.Use(gin.Logger()) // Standard Gin logger for development
	}

	// Request size limiting
	router.Use(middleware.RequestSizeMiddleware(cfg.Server.MaxBodySize))

	// API-specific middleware for /api routes
	api := router.Group("/api")
	api.Use(middleware.APISecurityHeaders())

	// Optional auth for some endpoints
	api.Use(middleware.OptionalAuthMiddleware(authService))

	// Rate limiting for API endpoints
	if cfg.RateLimit.Enabled {
		api.Use(middleware.DifferentialRateLimitMiddleware(anonymousLimiter, authenticatedLimiter))
	}

	// Health endpoints (no additional middleware)
	api.GET("/health", healthHandler.HealthCheck)
	api.GET("/health/detailed", healthHandler.DetailedHealthCheck)
	api.GET("/ready", healthHandler.ReadinessCheck)
	api.GET("/live", healthHandler.LivenessCheck)

	// Metrics endpoints (no additional middleware)
	api.GET("/metrics", metricsHandler.PrometheusMetrics)
	api.GET("/metrics/json", metricsHandler.JSONMetrics)
	api.GET("/metrics/health", metricsHandler.HealthMetrics)
	api.GET("/metrics/custom", metricsHandler.CustomMetrics)

	// Public image endpoints (with optional auth already applied)
	api.GET("/images/random", imageHandler.GetRandomImages)
	api.GET("/images/:id", imageHandler.GetImageByID)

	// Admin endpoints (require authentication and admin role)
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(authService))
	admin.Use(middleware.AdminOnlyMiddleware(authService))
	{
		admin.GET("/images", imageHandler.ListImages)
		admin.POST("/images/upload", imageHandler.UploadImage)
		admin.PUT("/images/:id", imageHandler.UpdateImage)
		admin.DELETE("/images/:id", imageHandler.DeleteImage)
	}

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         cfg.Server.GetAddress(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s (mode: %s)", cfg.Server.GetAddress(), cfg.Server.Mode)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}