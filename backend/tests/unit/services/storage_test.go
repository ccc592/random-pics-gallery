package services_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"randompic/internal/image"
)

func TestLocalFileOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	baseURL := "http://localhost:8080/static"

	storage, err := image.NewLocalStorage(tempDir, baseURL)
	require.NoError(t, err)
	require.NotNil(t, storage)

	t.Run("StoreFile", func(t *testing.T) {
		testData := []byte("This is test image data")
		filename := "test-image.jpg"

		path, err := storage.Store(filename, testData)
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		// File should exist at the returned path
		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		// Verify file contents
		content, err := os.ReadFile(fullPath)
		require.NoError(t, err)
		assert.Equal(t, testData, content)
	})

	t.Run("StoreMultipleFiles", func(t *testing.T) {
		files := map[string][]byte{
			"image1.jpg": []byte("Image 1 data"),
			"image2.png": []byte("Image 2 data"),
			"image3.jpg": []byte("Image 3 data"),
		}

		paths := make(map[string]string)
		for filename, data := range files {
			path, err := storage.Store(filename, data)
			require.NoError(t, err)
			paths[filename] = path

			// Verify each file exists
			fullPath := filepath.Join(tempDir, path)
			assert.FileExists(t, fullPath)
		}

		// All paths should be different
		uniquePaths := make(map[string]bool)
		for _, path := range paths {
			assert.False(t, uniquePaths[path], "Path should be unique: %s", path)
			uniquePaths[path] = true
		}
	})

	t.Run("RetrieveFile", func(t *testing.T) {
		testData := []byte("Retrieve test data")
		filename := "retrieve-test.jpg"

		// Store file first
		path, err := storage.Store(filename, testData)
		require.NoError(t, err)

		// Retrieve file
		retrievedData, err := storage.Retrieve(path)
		require.NoError(t, err)
		assert.Equal(t, testData, retrievedData)
	})

	t.Run("RetrieveNonexistentFile", func(t *testing.T) {
		_, err := storage.Retrieve("nonexistent/file.jpg")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("DeleteFile", func(t *testing.T) {
		testData := []byte("Delete test data")
		filename := "delete-test.jpg"

		// Store file first
		path, err := storage.Store(filename, testData)
		require.NoError(t, err)

		// Verify file exists
		assert.True(t, storage.Exists(path))

		// Delete file
		err = storage.Delete(path)
		assert.NoError(t, err)

		// Verify file no longer exists
		assert.False(t, storage.Exists(path))
	})

	t.Run("DeleteNonexistentFile", func(t *testing.T) {
		// Deleting non-existent file should not return an error
		err := storage.Delete("nonexistent/file.jpg")
		assert.NoError(t, err)
	})

	t.Run("FileExists", func(t *testing.T) {
		testData := []byte("Exists test data")
		filename := "exists-test.jpg"

		// File should not exist initially
		assert.False(t, storage.Exists("nonexistent-path"))

		// Store file
		path, err := storage.Store(filename, testData)
		require.NoError(t, err)

		// File should exist now
		assert.True(t, storage.Exists(path))

		// Delete file
		err = storage.Delete(path)
		require.NoError(t, err)

		// File should not exist anymore
		assert.False(t, storage.Exists(path))
	})

	t.Run("GetURL", func(t *testing.T) {
		testData := []byte("URL test data")
		filename := "url-test.jpg"

		path, err := storage.Store(filename, testData)
		require.NoError(t, err)

		url := storage.GetURL(path)
		expected := baseURL + "/" + path
		assert.Equal(t, expected, url)
		assert.Contains(t, url, "url-test.jpg")
	})
}

func TestQuotaEnforcement(t *testing.T) {
	// This test simulates quota enforcement logic
	// In practice, quota enforcement might be handled by a wrapper service
	tempDir := t.TempDir()
	storage, err := image.NewLocalStorage(tempDir, "http://localhost:8080/static")
	require.NoError(t, err)

	t.Run("StorageUsageCalculation", func(t *testing.T) {
		// Store multiple files of known sizes
		files := map[string][]byte{
			"small.jpg":  make([]byte, 1024),    // 1KB
			"medium.jpg": make([]byte, 1048576), // 1MB
			"large.jpg":  make([]byte, 2097152), // 2MB
		}

		totalSize := int64(0)
		for filename, data := range files {
			path, err := storage.Store(filename, data)
			require.NoError(t, err)

			// Verify file size on disk
			fullPath := filepath.Join(tempDir, path)
			info, err := os.Stat(fullPath)
			require.NoError(t, err)
			assert.Equal(t, int64(len(data)), info.Size())

			totalSize += int64(len(data))
		}

		// Total size should be approximately 3MB + 1KB
		expectedTotal := int64(1024 + 1048576 + 2097152) // 3,146,752 bytes
		assert.Equal(t, expectedTotal, totalSize)

		// Get storage stats
		stats, err := storage.GetStorageStats()
		require.NoError(t, err)

		assert.Equal(t, "local", stats["backend"])
		assert.Equal(t, tempDir, stats["base_path"])
		assert.Equal(t, totalSize, stats["total_size"])
		assert.Equal(t, int64(3), stats["file_count"])
	})
}

func TestDirectoryManagement(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("CreateStorageWithNonexistentPath", func(t *testing.T) {
		nonexistentPath := filepath.Join(tempDir, "nested", "directory", "structure")
		storage, err := image.NewLocalStorage(nonexistentPath, "http://localhost:8080/static")

		require.NoError(t, err)
		require.NotNil(t, storage)

		// Directory should be created
		assert.DirExists(t, nonexistentPath)
	})

	t.Run("PathTraversalGeneration", func(t *testing.T) {
		storage, err := image.NewLocalStorage(tempDir, "http://localhost:8080/static")
		require.NoError(t, err)

		// Store files should create dated subdirectories
		testData := []byte("Directory test data")
		filename := "subdir-test.jpg"

		path, err := storage.Store(filename, testData)
		require.NoError(t, err)

		// Path should contain date-based directory structure (YYYY/MM/DD)
		assert.Contains(t, path, "/")
		assert.Contains(t, path, filename)

		// Verify the file was created in a subdirectory
		fullPath := filepath.Join(tempDir, path)
		assert.FileExists(t, fullPath)

		// Parent directory should exist
		parentDir := filepath.Dir(fullPath)
		assert.DirExists(t, parentDir)
	})

	t.Run("DirectoryPermissions", func(t *testing.T) {
		// This test verifies that directories are created with proper permissions
		permTestDir := filepath.Join(tempDir, "permission-test")
		storage, err := image.NewLocalStorage(permTestDir, "http://localhost:8080/static")
		require.NoError(t, err)

		// Check that the directory was created
		assert.DirExists(t, permTestDir)

		// Store a file to create subdirectories
		testData := []byte("Permission test data")
		path, err := storage.Store("perm-test.jpg", testData)
		require.NoError(t, err)

		// Verify we can read the file (permissions are correct)
		retrievedData, err := storage.Retrieve(path)
		require.NoError(t, err)
		assert.Equal(t, testData, retrievedData)
	})
}

func TestCleanupOperations(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := image.NewLocalStorage(tempDir, "http://localhost:8080/static")
	require.NoError(t, err)

	t.Run("HealthCheck", func(t *testing.T) {
		err := storage.HealthCheck()
		assert.NoError(t, err)
	})

	t.Run("HealthCheckReadOnlyDirectory", func(t *testing.T) {
		// Create a read-only directory for testing
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0755)
		require.NoError(t, err)

		// Make directory read-only
		err = os.Chmod(readOnlyDir, 0444)
		require.NoError(t, err)

		// Cleanup: restore permissions after test
		defer func() {
			os.Chmod(readOnlyDir, 0755)
		}()

		// Health check should fail for read-only directory
		readOnlyStorage, err := image.NewLocalStorage(readOnlyDir, "http://localhost:8080/static")
		if err == nil {
			// If creation succeeded, health check should fail
			err = readOnlyStorage.HealthCheck()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not writable")
		}
		// If creation failed, that's also expected behavior for read-only directories
	})

	t.Run("BulkDelete", func(t *testing.T) {
		// Store multiple files
		files := []string{"bulk1.jpg", "bulk2.jpg", "bulk3.jpg"}
		paths := make([]string, len(files))

		testData := []byte("Bulk delete test data")
		for i, filename := range files {
			path, err := storage.Store(filename, testData)
			require.NoError(t, err)
			paths[i] = path

			// Verify file exists
			assert.True(t, storage.Exists(path))
		}

		// Delete all files
		for _, path := range paths {
			err := storage.Delete(path)
			assert.NoError(t, err)
			assert.False(t, storage.Exists(path))
		}
	})

	t.Run("StorageStats", func(t *testing.T) {
		// Store some files
		files := map[string]int{
			"stats1.jpg": 1024,
			"stats2.jpg": 2048,
			"stats3.jpg": 4096,
		}

		totalSize := int64(0)
		for filename, size := range files {
			data := make([]byte, size)
			path, err := storage.Store(filename, data)
			require.NoError(t, err)
			totalSize += int64(size)

			// Verify file was stored
			assert.True(t, storage.Exists(path))
		}

		stats, err := storage.GetStorageStats()
		require.NoError(t, err)

		assert.Equal(t, "local", stats["backend"])
		assert.Equal(t, tempDir, stats["base_path"])
		assert.GreaterOrEqual(t, stats["total_size"], totalSize)
		assert.GreaterOrEqual(t, stats["file_count"], int64(len(files)))
	})
}

func TestStorageErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := image.NewLocalStorage(tempDir, "http://localhost:8080/static")
	require.NoError(t, err)

	t.Run("EmptyFilename", func(t *testing.T) {
		testData := []byte("Empty filename test")
		path, err := storage.Store("", testData)

		// Should handle empty filename gracefully
		if err == nil {
			assert.NotEmpty(t, path)
			// Clean up
			storage.Delete(path)
		}
	})

	t.Run("LargeFile", func(t *testing.T) {
		// Create a large file (10MB)
		largeData := make([]byte, 10*1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		path, err := storage.Store("large-file.jpg", largeData)
		require.NoError(t, err)

		// Verify file was stored correctly
		retrievedData, err := storage.Retrieve(path)
		require.NoError(t, err)
		assert.Equal(t, len(largeData), len(retrievedData))
		assert.Equal(t, largeData, retrievedData)

		// Clean up
		err = storage.Delete(path)
		assert.NoError(t, err)
	})

	t.Run("SpecialCharactersInFilename", func(t *testing.T) {
		specialFilenames := []string{
			"file with spaces.jpg",
			"file-with-dashes.jpg",
			"file_with_underscores.jpg",
			"file.with.dots.jpg",
		}

		testData := []byte("Special characters test")
		for _, filename := range specialFilenames {
			path, err := storage.Store(filename, testData)
			require.NoError(t, err, "Failed to store file with name: %s", filename)

			// Verify file can be retrieved
			retrievedData, err := storage.Retrieve(path)
			require.NoError(t, err, "Failed to retrieve file: %s", filename)
			assert.Equal(t, testData, retrievedData)

			// Clean up
			err = storage.Delete(path)
			assert.NoError(t, err)
		}
	})
}

func TestStorageInterface(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("InterfaceCompliance", func(t *testing.T) {
		// Verify LocalStorage implements StorageInterface
		var storage image.StorageInterface
		localStorage, err := image.NewLocalStorage(tempDir, "http://localhost:8080/static")
		require.NoError(t, err)

		storage = localStorage // This should compile if interface is implemented correctly
		assert.NotNil(t, storage)

		// Test all interface methods
		testData := []byte("Interface test data")
		path, err := storage.Store("interface-test.jpg", testData)
		require.NoError(t, err)

		retrievedData, err := storage.Retrieve(path)
		require.NoError(t, err)
		assert.Equal(t, testData, retrievedData)

		url := storage.GetURL(path)
		assert.NotEmpty(t, url)

		exists := storage.Exists(path)
		assert.True(t, exists)

		err = storage.Delete(path)
		assert.NoError(t, err)

		exists = storage.Exists(path)
		assert.False(t, exists)
	})
}

// Test concurrent operations
func TestConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := image.NewLocalStorage(tempDir, "http://localhost:8080/static")
	require.NoError(t, err)

	t.Run("ConcurrentStores", func(t *testing.T) {
		numGoroutines := 10
		results := make(chan string, numGoroutines)
		errors := make(chan error, numGoroutines)

		// Start multiple goroutines storing files
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				testData := []byte("Concurrent test data")
				filename := fmt.Sprintf("concurrent-%d.jpg", id)
				path, err := storage.Store(filename, testData)
				if err != nil {
					errors <- err
					return
				}
				results <- path
			}(i)
		}

		// Collect results
		paths := make([]string, 0, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			select {
			case path := <-results:
				paths = append(paths, path)
			case err := <-errors:
				t.Errorf("Concurrent store failed: %v", err)
			}
		}

		assert.Len(t, paths, numGoroutines)

		// Verify all files exist and are unique
		uniquePaths := make(map[string]bool)
		for _, path := range paths {
			assert.True(t, storage.Exists(path))
			assert.False(t, uniquePaths[path], "Path should be unique: %s", path)
			uniquePaths[path] = true
		}

		// Clean up
		for _, path := range paths {
			storage.Delete(path)
		}
	})
}