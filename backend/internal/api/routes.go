package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/randompic/api/internal/api/handlers"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/image"
	"github.com/randompic/api/internal/middleware"
	"github.com/randompic/api/internal/services"
)

// RouterConfig holds configuration for the API router
type RouterConfig struct {
	Mode          string
	TrustedProxies []string
	CORSConfig    middleware.CORSConfig
	RateLimit     middleware.RateLimitConfig
	AuthConfig    auth.Config
}

// SetupRouter initializes and configures the Gin router with all routes and middleware
func SetupRouter(db *gorm.DB, services *services.Services, config *RouterConfig) *gin.Engine {
	// Set Gin mode
	gin.SetMode(config.Mode)

	// Create router
	router := gin.New()

	// Set trusted proxies
	if len(config.TrustedProxies) > 0 {
		router.SetTrustedProxies(config.TrustedProxies)
	}

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS(config.CORSConfig))

	// Health check endpoint (no middleware required)
	router.GET("/health", handlers.HealthCheck(db))
	router.GET("/metrics", handlers.Metrics(db))

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Apply rate limiting to all API routes
		v1.Use(middleware.RateLimit(config.RateLimit))

		// Initialize handlers
		authHandlers := handlers.NewAuthHandlers(services.AuthService, services.UserService)
		imageHandlers := handlers.NewImageHandlers(services.ImageService, services.UserService)
		randomHandlers := handlers.NewRandomImageHandlers(services.ImageService, services.RandomizerService)

		// Setup route groups
		setupPublicRoutes(v1, randomHandlers, imageHandlers)
		setupAuthRoutes(v1, authHandlers, config.AuthConfig)
		setupProtectedRoutes(v1, imageHandlers, services.AuthService)
	}

	return router
}

// setupPublicRoutes configures public API routes (no authentication required)
func setupPublicRoutes(router *gin.RouterGroup, randomHandlers *handlers.RandomImageHandlers, imageHandlers *handlers.ImageHandlers) {
	public := router.Group("/public")
	{
		// Random images endpoint
		public.GET("/random-images", randomHandlers.GetRandomImages)
		public.GET("/random-images/:seed", randomHandlers.GetRandomImagesWithSeed)

		// Public image metadata (for public images only)
		public.GET("/images/:id/metadata", imageHandlers.GetPublicImageMetadata)

		// Public image gallery (paginated)
		public.GET("/gallery", imageHandlers.GetPublicGallery)
	}
}

// setupAuthRoutes configures authentication-related routes
func setupAuthRoutes(router *gin.RouterGroup, authHandlers *handlers.AuthHandlers, config auth.Config) {
	auth := router.Group("/auth")
	{
		// OAuth 2.0 login endpoints
		auth.GET("/login", authHandlers.Login)
		auth.GET("/login/:provider", authHandlers.LoginWithProvider)

		// OAuth 2.0 callback
		auth.GET("/callback", authHandlers.Callback)

		// Logout (can be called with or without authentication)
		auth.POST("/logout", authHandlers.Logout)

		// Token refresh (requires valid refresh token)
		auth.POST("/refresh", authHandlers.RefreshToken)

		// Protected auth routes (require authentication)
		authProtected := auth.Group("/")
		authProtected.Use(middleware.RequireAuth(config))
		{
			// User profile
			authProtected.GET("/me", authHandlers.GetProfile)
			authProtected.PUT("/me", authHandlers.UpdateProfile)
			authProtected.DELETE("/me", authHandlers.DeleteAccount)
		}
	}
}

// setupProtectedRoutes configures routes that require authentication
func setupProtectedRoutes(router *gin.RouterGroup, imageHandlers *handlers.ImageHandlers, authService *auth.Service) {
	protected := router.Group("/")
	protected.Use(middleware.RequireAuth(authService.GetConfig()))
	{
		// Image management
		images := protected.Group("/images")
		{
			// CRUD operations
			images.GET("/", imageHandlers.ListUserImages)
			images.POST("/", imageHandlers.CreateImage) // Upload
			images.GET("/:id", imageHandlers.GetImage)
			images.PUT("/:id", imageHandlers.UpdateImage)
			images.DELETE("/:id", imageHandlers.DeleteImage)

			// Batch operations
			images.POST("/batch", imageHandlers.BatchUpload)
			images.DELETE("/batch", imageHandlers.BatchDelete)

			// Image metadata and statistics
			images.GET("/:id/metadata", imageHandlers.GetImageMetadata)
			images.GET("/:id/stats", imageHandlers.GetImageStats)
		}

		// User management
		user := protected.Group("/user")
		{
			// Storage information
			user.GET("/storage", imageHandlers.GetStorageInfo)
			user.GET("/storage/usage", imageHandlers.GetStorageUsage)

			// User statistics
			user.GET("/stats", imageHandlers.GetUserStats)

			// User preferences
			user.GET("/preferences", imageHandlers.GetUserPreferences)
			user.PUT("/preferences", imageHandlers.UpdateUserPreferences)
		}

		// Admin routes (require admin role)
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRole("admin"))
		{
			// User management
			admin.GET("/users", imageHandlers.ListAllUsers)
			admin.GET("/users/:id", imageHandlers.GetUserAdmin)
			admin.PUT("/users/:id", imageHandlers.UpdateUserAdmin)
			admin.DELETE("/users/:id", imageHandlers.DeleteUserAdmin)

			// System statistics
			admin.GET("/stats", imageHandlers.GetSystemStats)
			admin.GET("/storage/stats", imageHandlers.GetSystemStorageStats)
		}
	}
}

// SetupTestRouter creates a router for testing with minimal middleware
func SetupTestRouter(db *gorm.DB, services *services.Services) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())

	// Basic CORS for testing
	corsConfig := middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
		MaxAge:       12 * time.Hour,
	}
	router.Use(middleware.CORS(corsConfig))

	// Simple health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// Setup API routes
	config := &RouterConfig{
		Mode: gin.TestMode,
		CORSConfig: corsConfig,
		RateLimit: middleware.RateLimitConfig{
			RequestsPerMinute: 1000, // Higher limit for testing
			BurstSize:        100,
		},
	}

	return SetupRouter(db, services, config)
}