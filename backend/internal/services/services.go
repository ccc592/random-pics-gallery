package services

import (
	"gorm.io/gorm"
)

// Services holds all application services
type Services struct {
	UserService       UserServiceInterface
	ImageService      ImageServiceInterface
	RandomizerService RandomizerServiceInterface
	StorageService    StorageServiceInterface
}

// NewServices creates and initializes all services with their dependencies
func NewServices(db *gorm.DB, storagePath string) *Services {
	// Initialize storage service first (needed by image service)
	storageService := NewStorageService(storagePath, db)

	// Initialize user service
	userService := NewUserService(db)

	// Initialize image service with dependencies
	imageService := NewImageService(db, storageService, userService)

	// Initialize randomizer service with image service dependency
	randomizerService := NewRandomizerService(imageService)

	return &Services{
		UserService:       userService,
		ImageService:      imageService,
		RandomizerService: randomizerService,
		StorageService:    storageService,
	}
}

// ServiceConfig holds configuration for services
type ServiceConfig struct {
	StoragePath        string `json:"storage_path"`
	MaxFileSize        int64  `json:"max_file_size"`
	DefaultQuotaBytes  int64  `json:"default_quota_bytes"`
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		StoragePath:       "./storage",
		MaxFileSize:       2097152,     // 2MB
		DefaultQuotaBytes: 10737418240, // 10GB
	}
}