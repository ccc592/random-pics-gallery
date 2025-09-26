package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// FileInfo represents information about a stored file
type FileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsDir        bool      `json:"is_dir"`
	Exists       bool      `json:"exists"`
	MimeType     string    `json:"mime_type,omitempty"`
	UserID       string    `json:"user_id,omitempty"`
}

// StorageStats represents storage usage statistics
type StorageStats struct {
	TotalFiles      int64            `json:"total_files"`
	TotalSizeBytes  int64            `json:"total_size_bytes"`
	UserStats       map[string]int64 `json:"user_stats"`       // userID -> bytes used
	OrphanedFiles   []string         `json:"orphaned_files"`   // files without database records
	DirectoryCount  int              `json:"directory_count"`
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	RemovedFiles     []string `json:"removed_files"`
	RemovedDirs      []string `json:"removed_dirs"`
	FreedBytes       int64    `json:"freed_bytes"`
	Errors           []string `json:"errors"`
	ProcessedFiles   int      `json:"processed_files"`
	ProcessedDirs    int      `json:"processed_dirs"`
}

// StorageServiceInterface defines the contract for file storage operations
type StorageServiceInterface interface {
	StoreFile(ctx context.Context, userID string, filename string, data io.Reader) (string, error)
	RetrieveFile(ctx context.Context, path string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, path string) error
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
	CalculateUserStorageUsage(ctx context.Context, userID string) (int64, error)
	CleanupOrphanedFiles(ctx context.Context) error
}

// StorageService implements the StorageServiceInterface for local filesystem storage
type StorageService struct {
	basePath string
	db       *gorm.DB
}

// NewStorageService creates a new StorageService instance
func NewStorageService(basePath string, db *gorm.DB) *StorageService {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create storage directory: %v", err))
	}

	return &StorageService{
		basePath: basePath,
		db:       db,
	}
}

// StoreFile stores a file in the local filesystem with user-based organization
func (s *StorageService) StoreFile(ctx context.Context, userID string, filename string, data io.Reader) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user ID cannot be empty")
	}
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	if data == nil {
		return "", fmt.Errorf("data cannot be nil")
	}

	// Sanitize filename to prevent directory traversal
	filename = s.sanitizeFilename(filename)
	if filename == "" {
		return "", fmt.Errorf("filename is invalid after sanitization")
	}

	// Create user directory structure: basePath/userID/YYYY/MM/
	now := time.Now()
	userDir := filepath.Join(s.basePath, userID, fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()))

	// Create directory structure if it doesn't exist
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create user directory: %w", err)
	}

	// Create full file path
	fullPath := filepath.Join(userDir, filename)

	// Check if file already exists and generate unique name if needed
	uniquePath, err := s.getUniqueFilePath(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get unique file path: %w", err)
	}

	// Create the file
	file, err := os.Create(uniquePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data to file
	_, err = io.Copy(file, data)
	if err != nil {
		// Clean up the file if copy failed
		os.Remove(uniquePath)
		return "", fmt.Errorf("failed to write file data: %w", err)
	}

	// Ensure file is flushed to disk
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync file: %w", err)
	}

	// Return the relative path from base path
	relativePath, err := filepath.Rel(s.basePath, uniquePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return relativePath, nil
}

// RetrieveFile retrieves a file from the local filesystem
func (s *StorageService) RetrieveFile(ctx context.Context, path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Resolve absolute path and ensure it's within base path
	fullPath, err := s.resolveAndValidatePath(path)
	if err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	// Check if file exists and is readable
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Open file for reading
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// DeleteFile deletes a file from the local filesystem
func (s *StorageService) DeleteFile(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Resolve absolute path and ensure it's within base path
	fullPath, err := s.resolveAndValidatePath(path)
	if err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, consider it already deleted
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Delete the file
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to clean up empty parent directories (user/year/month structure)
	s.cleanupEmptyDirectories(filepath.Dir(fullPath))

	return nil
}

// GetFileInfo returns information about a file
func (s *StorageService) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Resolve absolute path and ensure it's within base path
	fullPath, err := s.resolveAndValidatePath(path)
	if err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileInfo{
				Path:   path,
				Exists: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Extract user ID from path
	userID := s.extractUserIDFromPath(path)

	// Determine MIME type from file extension
	mimeType := s.getMimeTypeFromPath(path)

	fileInfo := &FileInfo{
		Path:     path,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		IsDir:    info.IsDir(),
		Exists:   true,
		MimeType: mimeType,
		UserID:   userID,
	}

	return fileInfo, nil
}

// CalculateUserStorageUsage calculates total storage usage for a user
func (s *StorageService) CalculateUserStorageUsage(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, fmt.Errorf("user ID cannot be empty")
	}

	userPath := filepath.Join(s.basePath, userID)

	// Check if user directory exists
	if _, err := os.Stat(userPath); err != nil {
		if os.IsNotExist(err) {
			return 0, nil // User has no files
		}
		return 0, fmt.Errorf("failed to stat user directory: %w", err)
	}

	var totalSize int64
	err := filepath.Walk(userPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to walk user directory: %w", err)
	}

	return totalSize, nil
}

// CleanupOrphanedFiles removes files that don't have corresponding database records
func (s *StorageService) CleanupOrphanedFiles(ctx context.Context) error {
	result := &CleanupResult{
		RemovedFiles: []string{},
		RemovedDirs:  []string{},
		Errors:       []string{},
	}

	// Get all image records from database
	var images []models.Image
	if err := s.db.WithContext(ctx).Select("storage_path").Find(&images).Error; err != nil {
		return fmt.Errorf("failed to get image records: %w", err)
	}

	// Create a map of valid storage paths
	validPaths := make(map[string]bool)
	for _, image := range images {
		validPaths[image.StoragePath] = true
	}

	// Walk through storage directory
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Walk error for %s: %v", path, err))
			return nil // Continue walking
		}

		// Skip the base directory itself
		if path == s.basePath {
			return nil
		}

		if info.IsDir() {
			result.ProcessedDirs++
			return nil
		}

		result.ProcessedFiles++

		// Get relative path from base
		relativePath, err := filepath.Rel(s.basePath, path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to get relative path for %s: %v", path, err))
			return nil
		}

		// Check if this path has a corresponding database record
		if !validPaths[relativePath] {
			// This is an orphaned file
			if err := os.Remove(path); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to remove orphaned file %s: %v", path, err))
			} else {
				result.RemovedFiles = append(result.RemovedFiles, relativePath)
				result.FreedBytes += info.Size()
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk storage directory: %w", err)
	}

	// Clean up empty directories
	s.cleanupEmptyDirectoriesRecursive(s.basePath, result)

	return nil
}

// GetStorageStats returns comprehensive storage statistics
func (s *StorageService) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	stats := &StorageStats{
		UserStats:      make(map[string]int64),
		OrphanedFiles:  []string{},
	}

	// Get all users from database
	var users []models.User
	if err := s.db.WithContext(ctx).Select("id").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Calculate storage usage for each user
	for _, user := range users {
		usage, err := s.CalculateUserStorageUsage(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate usage for user %s: %w", user.ID, err)
		}
		stats.UserStats[user.ID] = usage
		stats.TotalSizeBytes += usage
	}

	// Get all image records to find orphaned files
	var images []models.Image
	if err := s.db.WithContext(ctx).Select("storage_path").Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to get image records: %w", err)
	}

	validPaths := make(map[string]bool)
	for _, image := range images {
		validPaths[image.StoragePath] = true
	}

	// Count files and identify orphans
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		if info.IsDir() {
			stats.DirectoryCount++
			return nil
		}

		stats.TotalFiles++

		// Get relative path
		relativePath, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return nil
		}

		// Check if orphaned
		if !validPaths[relativePath] {
			stats.OrphanedFiles = append(stats.OrphanedFiles, relativePath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk storage directory: %w", err)
	}

	return stats, nil
}

// Helper functions

func (s *StorageService) sanitizeFilename(filename string) string {
	// Remove path separators and dangerous characters
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")
	filename = strings.ReplaceAll(filename, "\x00", "_")

	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Ensure filename is not empty or just dots
	if filename == "" || filename == "." || filename == ".." {
		filename = fmt.Sprintf("file_%d", time.Now().UnixNano())
	}

	return filename
}

func (s *StorageService) resolveAndValidatePath(path string) (string, error) {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)

	// Resolve to absolute path within base path
	fullPath := filepath.Join(s.basePath, cleanPath)

	// Ensure the resolved path is still within base path
	if !strings.HasPrefix(fullPath, s.basePath) {
		return "", fmt.Errorf("path escapes base directory: %s", path)
	}

	return fullPath, nil
}

func (s *StorageService) getUniqueFilePath(path string) (string, error) {
	// If file doesn't exist, use the original path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}

	// File exists, generate a unique name
	dir := filepath.Dir(path)
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	for i := 1; i < 1000; i++ { // Limit attempts to prevent infinite loop
		newFilename := fmt.Sprintf("%s_%d%s", name, i, ext)
		newPath := filepath.Join(dir, newFilename)

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, nil
		}
	}

	return "", fmt.Errorf("unable to generate unique filename after 1000 attempts")
}

func (s *StorageService) extractUserIDFromPath(path string) string {
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	if len(parts) > 0 {
		return parts[0] // First part should be user ID
	}
	return ""
}

func (s *StorageService) getMimeTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return ""
	}
}

func (s *StorageService) cleanupEmptyDirectories(dir string) {
	// Don't remove directories above user level
	if !strings.HasPrefix(dir, s.basePath) {
		return
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return
	}

	// Remove empty directory
	if err := os.Remove(dir); err != nil {
		return
	}

	// Recursively clean parent directory
	parent := filepath.Dir(dir)
	if parent != dir && parent != s.basePath {
		s.cleanupEmptyDirectories(parent)
	}
}

func (s *StorageService) cleanupEmptyDirectoriesRecursive(rootDir string, result *CleanupResult) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read directory %s: %v", rootDir, err))
		return
	}

	// Process subdirectories first
	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(rootDir, entry.Name())
			s.cleanupEmptyDirectoriesRecursive(subDir, result)
		}
	}

	// Check if directory is now empty (after processing subdirectories)
	if rootDir != s.basePath {
		entries, err = os.ReadDir(rootDir)
		if err == nil && len(entries) == 0 {
			if err := os.Remove(rootDir); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to remove empty directory %s: %v", rootDir, err))
			} else {
				relativePath, _ := filepath.Rel(s.basePath, rootDir)
				result.RemovedDirs = append(result.RemovedDirs, relativePath)
			}
		}
	}
}