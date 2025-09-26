package main

import (
	"log"

	"github.com/randompic/api/internal/api"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/config"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/image"
	"github.com/randompic/api/internal/middleware"
	"github.com/randompic/api/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database connection
	database, err := db.NewConnection(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate database schema
	if err := db.AutoMigrate(database); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize image processor
	imageProcessor := image.NewProcessor(&image.ProcessorConfig{
		MaxWidth:      2048,
		MaxHeight:     2048,
		Quality:       85,
		ThumbnailSize: 300,
	})

	// Initialize services
	servicesContainer := &services.Services{
		ImageService:      services.NewImageService(database, imageProcessor),
		RandomizerService: services.NewRandomizerService(database),
		UserService:       services.NewUserService(database),
		StorageService:    services.NewStorageService(database, cfg.Storage),
	}

	// Initialize auth service
	authService, err := auth.NewService(&cfg.Auth)
	if err != nil {
		log.Fatal("Failed to initialize auth service:", err)
	}

	// Set up router configuration
	routerConfig := &api.RouterConfig{
		Mode:          cfg.Server.Mode,
		TrustedProxies: cfg.Server.TrustedProxies,
		CORSConfig:    *cfg.Security.CORS,
		RateLimit:     middleware.RateLimitConfig{
			RequestsPerWindow: 60,
			WindowSize:        cfg.Security.RateLimit.Anonymous.WindowSize,
			CleanupInterval:   cfg.Security.RateLimit.Anonymous.CleanupInterval,
			AdminMultiplier:   cfg.Security.RateLimit.Anonymous.AdminMultiplier,
		},
		AuthConfig: cfg.Auth,
	}

	// Set up router with all handlers and middleware
	router := api.SetupRouter(database, servicesContainer, routerConfig)

	// Start server
	log.Printf("Starting server on %s", cfg.Server.Address)
	if err := router.Run(cfg.Server.Address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}