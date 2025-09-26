package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type            string        `json:"type" yaml:"type"`                         // "local" or "s3"
	BasePath        string        `json:"base_path" yaml:"base_path"`               // Base storage path
	MaxUserQuota    int64         `json:"max_user_quota" yaml:"max_user_quota"`     // Max quota per user in bytes (10GB default)
	MaxFileSize     int64         `json:"max_file_size" yaml:"max_file_size"`       // Max file size in bytes
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"` // Cleanup interval
	TempDir         string        `json:"temp_dir" yaml:"temp_dir"`                 // Temporary directory
	AllowedMimeTypes []string     `json:"allowed_mime_types" yaml:"allowed_mime_types"` // Allowed MIME types
}

// StorageService manages local filesystem storage with quota tracking
type StorageService struct {
	config      *StorageConfig
	db          *gorm.DB
	quotaCache  map[string]int64 // Cache for user quotas
	quotaMutex  sync.RWMutex
	cleanupStop chan struct{}
	cleanupWg   sync.WaitGroup
}

// StorageStats represents storage statistics
type StorageStats struct {
	UserID          string    `json:"user_id"`
	TotalFiles      int64     `json:"total_files"`
	TotalSize       int64     `json:"total_size"`
	QuotaUsed       int64     `json:"quota_used"`
	QuotaLimit      int64     `json:"quota_limit"`
	QuotaPercentage float64   `json:"quota_percentage"`
	LastUpdated     time.Time `json:"last_updated"`
}

// FileInfo represents file information
type FileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	LastModified time.Time `json:"last_modified"`
}

// DefaultStorageConfig returns default storage configuration
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type:            "local",
		BasePath:        "./storage",
		MaxUserQuota:    10 * 1024 * 1024 * 1024, // 10GB
		MaxFileSize:     50 * 1024 * 1024,        // 50MB
		CleanupInterval: 24 * time.Hour,
		TempDir:         "./storage/temp",
		AllowedMimeTypes: []string{
			"image/jpeg",
			"image/png",
			"image/webp",
			"image/gif",
		},
	}
}

// NewStorageService creates a new storage service
func NewStorageService(config *StorageConfig, db *gorm.DB) (*StorageService, error) {
	service := &StorageService{
		config:      config,
		db:          db,
		quotaCache:  make(map[string]int64),
		cleanupStop: make(chan struct{}),
	}

	// Initialize storage directories
	if err := service.initDirectories(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage directories: %w", err)
	}

	// Start cleanup routine
	service.startCleanupRoutine()

	return service, nil
}

// InitStorage initializes and returns a storage service
func InitStorage(config *StorageConfig, db *gorm.DB) (*StorageService, error) {
	return NewStorageService(config, db)
}

// initDirectories creates necessary storage directories
func (s *StorageService) initDirectories() error {
	dirs := []string{
		s.config.BasePath,
		s.config.TempDir,
		filepath.Join(s.config.BasePath, "images"),
		filepath.Join(s.config.BasePath, "thumbnails"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetUserPath returns the storage path for a specific user
func (s *StorageService) GetUserPath(userID string) string {
	return filepath.Join(s.config.BasePath, "images", userID)
}

// GetThumbnailPath returns the thumbnail path for a specific user
func (s *StorageService) GetThumbnailPath(userID string) string {
	return filepath.Join(s.config.BasePath, "thumbnails", userID)
}

// CheckUserQuota checks if a user has enough quota for a file
func (s *StorageService) CheckUserQuota(userID string, fileSize int64) error {
	currentUsage, err := s.GetUserQuotaUsage(userID)
	if err != nil {
		return fmt.Errorf("failed to get user quota usage: %w", err)
	}

	if currentUsage+fileSize > s.config.MaxUserQuota {
		return fmt.Errorf("quota exceeded: current usage %d + file size %d > limit %d",
			currentUsage, fileSize, s.config.MaxUserQuota)
	}

	return nil
}

// GetUserQuotaUsage returns the current quota usage for a user
func (s *StorageService) GetUserQuotaUsage(userID string) (int64, error) {
	// Check cache first
	s.quotaMutex.RLock()
	if usage, exists := s.quotaCache[userID]; exists {
		s.quotaMutex.RUnlock()
		return usage, nil
	}
	s.quotaMutex.RUnlock()

	// Calculate usage from database
	var totalSize int64
	err := s.db.Model(&models.Image{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalSize).Error

	if err != nil {
		return 0, fmt.Errorf("failed to calculate user quota usage: %w", err)
	}

	// Update cache
	s.quotaMutex.Lock()
	s.quotaCache[userID] = totalSize
	s.quotaMutex.Unlock()

	return totalSize, nil
}

// UpdateUserQuotaUsage updates the cached quota usage for a user
func (s *StorageService) UpdateUserQuotaUsage(userID string, delta int64) {
	s.quotaMutex.Lock()
	defer s.quotaMutex.Unlock()

	if currentUsage, exists := s.quotaCache[userID]; exists {
		s.quotaCache[userID] = currentUsage + delta
	} else {
		// If not in cache, we'll recalculate on next request
		delete(s.quotaCache, userID)
	}
}

// GetStorageStats returns storage statistics for a user
func (s *StorageService) GetStorageStats(userID string) (*StorageStats, error) {
	var stats struct {
		Count int64 `json:"count"`
		Size  int64 `json:"size"`
	}

	err := s.db.Model(&models.Image{}).
		Where("user_id = ?", userID).
		Select("COUNT(*) as count, COALESCE(SUM(file_size), 0) as size").
		Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	quotaPercentage := float64(stats.Size) / float64(s.config.MaxUserQuota) * 100

	return &StorageStats{
		UserID:          userID,
		TotalFiles:      stats.Count,
		TotalSize:       stats.Size,
		QuotaUsed:       stats.Size,
		QuotaLimit:      s.config.MaxUserQuota,
		QuotaPercentage: quotaPercentage,
		LastUpdated:     time.Now().UTC(),
	}, nil
}

// SaveFile saves a file to user's storage directory
func (s *StorageService) SaveFile(userID, fileName string, data []byte) (string, error) {
	// Check quota
	if err := s.CheckUserQuota(userID, int64(len(data))); err != nil {
		return "", err
	}

	// Create user directory if it doesn't exist
	userPath := s.GetUserPath(userID)
	if err := os.MkdirAll(userPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create user directory: %w", err)
	}

	// Generate full file path
	filePath := filepath.Join(userPath, fileName)

	// Save file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Update quota cache
	s.UpdateUserQuotaUsage(userID, int64(len(data)))

	return filePath, nil
}

// DeleteFile deletes a file from storage
func (s *StorageService) DeleteFile(filePath string, userID string, fileSize int64) error {
	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// Update quota cache
	s.UpdateUserQuotaUsage(userID, -fileSize)

	return nil
}

// ListUserFiles lists all files for a user
func (s *StorageService) ListUserFiles(userID string) ([]FileInfo, error) {
	userPath := s.GetUserPath(userID)

	// Check if directory exists
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		return []FileInfo{}, nil
	}

	var files []FileInfo
	err := filepath.Walk(userPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			files = append(files, FileInfo{
				Path:         path,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list user files: %w", err)
	}

	return files, nil
}

// CleanupExpiredFiles removes expired temporary files
func (s *StorageService) CleanupExpiredFiles() error {
	tempDir := s.config.TempDir

	// Check if temp directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return nil
	}

	cutoff := time.Now().Add(-24 * time.Hour) // Remove files older than 24 hours

	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				// Log error but continue cleanup
				return nil
			}
		}

		return nil
	})
}

// MonitorStorageUsage monitors storage usage and sends alerts if needed
func (s *StorageService) MonitorStorageUsage() error {
	// This would typically integrate with monitoring systems
	// For now, we'll just validate the storage health

	// Check if base directory is accessible
	if _, err := os.Stat(s.config.BasePath); err != nil {
		return fmt.Errorf("storage base path not accessible: %w", err)
	}

	// Check available disk space
	// This would require platform-specific code or third-party library
	// For now, we'll just return success
	return nil
}

// startCleanupRoutine starts the background cleanup routine
func (s *StorageService) startCleanupRoutine() {
	s.cleanupWg.Add(1)
	go func() {
		defer s.cleanupWg.Done()

		ticker := time.NewTicker(s.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.CleanupExpiredFiles(); err != nil {
					// Log error but continue
					continue
				}
			case <-s.cleanupStop:
				return
			}
		}
	}()
}

// Close closes the storage service and stops background routines
func (s *StorageService) Close() error {
	close(s.cleanupStop)
	s.cleanupWg.Wait()
	return nil
}

// GetConfig returns the storage configuration
func (s *StorageService) GetConfig() *StorageConfig {
	return s.config
}

// IsAllowedMimeType checks if a MIME type is allowed
func (s *StorageService) IsAllowedMimeType(mimeType string) bool {
	for _, allowed := range s.config.AllowedMimeTypes {
		if mimeType == allowed {
			return true
		}
	}
	return false
}

// GetMaxFileSize returns the maximum allowed file size
func (s *StorageService) GetMaxFileSize() int64 {
	return s.config.MaxFileSize
}