package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageConfig holds configuration for different storage backends
type StorageConfig struct {
	Backend   string // "local" or "s3"
	LocalPath string // For local backend

	// S3 configuration
	S3Bucket    string
	S3Region    string
	S3Endpoint  string // For S3-compatible services like MinIO
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
}

// LocalStorage implements local filesystem storage
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Store saves data to local filesystem
func (ls *LocalStorage) Store(filename string, data []byte) (string, error) {
	// Generate unique path to avoid conflicts
	timestamp := time.Now().Format("2006/01/02")
	dir := filepath.Join(ls.basePath, timestamp)

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Full file path
	fullPath := filepath.Join(dir, filename)

	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return relative path for storage
	relativePath := filepath.Join(timestamp, filename)
	return strings.ReplaceAll(relativePath, "\\", "/"), nil // Ensure forward slashes
}

// Retrieve reads data from local filesystem
func (ls *LocalStorage) Retrieve(path string) ([]byte, error) {
	fullPath := filepath.Join(ls.basePath, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Delete removes a file from local filesystem
func (ls *LocalStorage) Delete(path string) error {
	fullPath := filepath.Join(ls.basePath, path)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetURL returns the public URL for a file
func (ls *LocalStorage) GetURL(path string) string {
	return ls.baseURL + "/" + path
}

// Exists checks if a file exists in local storage
func (ls *LocalStorage) Exists(path string) bool {
	fullPath := filepath.Join(ls.basePath, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// S3Storage implements S3-compatible storage
type S3Storage struct {
	client   *s3.Client
	bucket   string
	baseURL  string
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(cfg *StorageConfig) (*S3Storage, error) {
	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.S3Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override credentials if provided
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		awsCfg.Credentials = aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.S3AccessKey,
				SecretAccessKey: cfg.S3SecretKey,
			}, nil
		})
	}

	// Create S3 client with custom endpoint if provided (for MinIO/compatible services)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.EndpointResolver = s3.EndpointResolverFromURL(cfg.S3Endpoint)
		}
		if !cfg.S3UseSSL {
			o.UsePathStyle = true
		}
	})

	// Generate base URL for public access
	baseURL := cfg.S3Endpoint
	if baseURL == "" {
		if cfg.S3UseSSL {
			baseURL = fmt.Sprintf("https://s3.%s.amazonaws.com", cfg.S3Region)
		} else {
			baseURL = fmt.Sprintf("http://s3.%s.amazonaws.com", cfg.S3Region)
		}
	}

	return &S3Storage{
		client:  client,
		bucket:  cfg.S3Bucket,
		baseURL: baseURL,
	}, nil
}

// Store uploads data to S3
func (s3s *S3Storage) Store(filename string, data []byte) (string, error) {
	// Generate unique key with timestamp prefix
	timestamp := time.Now().Format("2006/01/02")
	key := fmt.Sprintf("%s/%s", timestamp, filename)

	// Upload to S3
	_, err := s3s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(string(data)),
		ContentType: aws.String("application/octet-stream"), // Will be overridden by actual MIME type
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return key, nil
}

// Retrieve downloads data from S3
func (s3s *S3Storage) Retrieve(path string) ([]byte, error) {
	result, err := s3s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return data, nil
}

// Delete removes an object from S3
func (s3s *S3Storage) Delete(path string) error {
	_, err := s3s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// GetURL returns the public URL for an S3 object
func (s3s *S3Storage) GetURL(path string) string {
	return fmt.Sprintf("%s/%s/%s", s3s.baseURL, s3s.bucket, path)
}

// Exists checks if an object exists in S3
func (s3s *S3Storage) Exists(path string) bool {
	_, err := s3s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(path),
	})
	return err == nil
}

// NewStorageFromConfig creates a storage instance based on configuration
func NewStorageFromConfig(cfg *StorageConfig) (StorageInterface, error) {
	switch strings.ToLower(cfg.Backend) {
	case "local":
		return NewLocalStorage(cfg.LocalPath, "http://localhost:8080/static")
	case "s3":
		return NewS3Storage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.Backend)
	}
}

// StorageHealthCheck checks if the storage backend is accessible
func (ls *LocalStorage) HealthCheck() error {
	// Check if base directory is writable
	testFile := filepath.Join(ls.basePath, ".health_check")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("local storage not writable: %w", err)
	}

	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("local storage cleanup failed: %w", err)
	}

	return nil
}

// HealthCheck checks if S3 is accessible
func (s3s *S3Storage) HealthCheck() error {
	// Try to list objects in bucket (with limit 1)
	_, err := s3s.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(s3s.bucket),
		MaxKeys: int32(1),
	})

	if err != nil {
		return fmt.Errorf("S3 storage not accessible: %w", err)
	}

	return nil
}

// GetStorageStats returns statistics about storage usage
func (ls *LocalStorage) GetStorageStats() (map[string]interface{}, error) {
	var totalSize int64
	var fileCount int64

	err := filepath.Walk(ls.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate storage stats: %w", err)
	}

	return map[string]interface{}{
		"backend":    "local",
		"total_size": totalSize,
		"file_count": fileCount,
		"base_path":  ls.basePath,
	}, nil
}

// GetStorageStats returns statistics about S3 storage usage
func (s3s *S3Storage) GetStorageStats() (map[string]interface{}, error) {
	// Note: Getting exact size from S3 requires listing all objects
	// For now, return basic info
	return map[string]interface{}{
		"backend": "s3",
		"bucket":  s3s.bucket,
	}, nil
}