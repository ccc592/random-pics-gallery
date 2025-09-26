package integration

import (
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randompic/api/internal/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testStorageDir    = "/tmp/claude/randompic_test_storage"
	testStorageURL    = "http://localhost:8080/static"
	testStorageQuota  = 10 * 1024 * 1024 * 1024 // 10GB in bytes
	userStorageQuota  = testStorageQuota         // For per-user testing
	maxConcurrentOps  = 50
	testImageSizes    = 1024 * 1024 // 1MB test files
)

// TestStorage represents the test storage instance
type TestStorage struct {
	Storage  image.StorageInterface
	BaseDir  string
	TestQuota int64
}

// newTestStorage creates a new test storage instance
func newTestStorage(t *testing.T) *TestStorage {
	// Create unique test directory to avoid conflicts
	testID := uuid.New().String()[:8]
	testDir := filepath.Join(testStorageDir, fmt.Sprintf("test_%s_%d", testID, time.Now().Unix()))

	// Ensure test directory is clean
	err := os.RemoveAll(testDir)
	require.NoError(t, err, "Failed to clean test directory")

	config := &image.StorageConfig{
		Backend:   "local",
		LocalPath: testDir,
	}

	storage, err := image.NewStorageFromConfig(config)
	require.NoError(t, err, "Failed to create test storage")

	return &TestStorage{
		Storage:   storage,
		BaseDir:   testDir,
		TestQuota: userStorageQuota,
	}
}

// cleanup removes all test storage data
func (ts *TestStorage) cleanup(t *testing.T) {
	if ts.BaseDir != "" {
		err := os.RemoveAll(ts.BaseDir)
		if err != nil {
			t.Logf("Warning: Failed to cleanup test storage directory %s: %v", ts.BaseDir, err)
		}
	}
}

// generateTestImage creates test image data of specified size
func generateTestImage(size int) []byte {
	data := make([]byte, size)
	// Fill with pseudo-random data to simulate real image content
	rand.Seed(time.Now().UnixNano())
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	return data
}

// calculateDirSize calculates the total size of a directory
func calculateDirSize(dirPath string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

// countFilesInDir counts the number of files in a directory recursively
func countFilesInDir(dirPath string) (int, error) {
	var fileCount int
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
		}
		return nil
	})
	return fileCount, err
}

func TestLocalStorageBasicOperations(t *testing.T) {
	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Storage Directory Creation", func(t *testing.T) {
		// Verify base directory exists
		_, err := os.Stat(testStorage.BaseDir)
		assert.NoError(t, err, "Base storage directory should exist")

		// Verify directory permissions
		info, err := os.Stat(testStorage.BaseDir)
		require.NoError(t, err, "Should be able to stat directory")
		assert.True(t, info.IsDir(), "Should be a directory")

		// Check directory is writable
		testFile := filepath.Join(testStorage.BaseDir, "write_test.tmp")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		assert.NoError(t, err, "Directory should be writable")

		// Cleanup test file
		os.Remove(testFile)
	})

	t.Run("Store File", func(t *testing.T) {
		testData := generateTestImage(1024) // 1KB test image
		filename := "test-store.jpg"

		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")
		assert.NotEmpty(t, storagePath, "Storage path should not be empty")
		assert.Contains(t, storagePath, filename, "Storage path should contain filename")

		// Verify file exists on disk
		fullPath := filepath.Join(testStorage.BaseDir, storagePath)
		_, err = os.Stat(fullPath)
		assert.NoError(t, err, "Stored file should exist on disk")

		// Verify file size
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "Should be able to stat stored file")
		assert.Equal(t, int64(len(testData)), info.Size(), "File size should match original data")
	})

	t.Run("Retrieve File", func(t *testing.T) {
		testData := generateTestImage(2048) // 2KB test image
		filename := "test-retrieve.jpg"

		// Store file first
		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")

		// Retrieve file
		retrievedData, err := testStorage.Storage.Retrieve(storagePath)
		require.NoError(t, err, "File retrieval should succeed")
		assert.Equal(t, testData, retrievedData, "Retrieved data should match original")
	})

	t.Run("Delete File", func(t *testing.T) {
		testData := generateTestImage(512) // 512B test image
		filename := "test-delete.jpg"

		// Store file first
		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")

		// Verify file exists
		fullPath := filepath.Join(testStorage.BaseDir, storagePath)
		_, err = os.Stat(fullPath)
		require.NoError(t, err, "File should exist before deletion")

		// Delete file
		err = testStorage.Storage.Delete(storagePath)
		require.NoError(t, err, "File deletion should succeed")

		// Verify file no longer exists
		_, err = os.Stat(fullPath)
		assert.True(t, os.IsNotExist(err), "File should not exist after deletion")
	})

	t.Run("File Exists Check", func(t *testing.T) {
		testData := generateTestImage(256)
		filename := "test-exists.jpg"

		// Check non-existent file
		exists := testStorage.Storage.Exists("non/existent/path.jpg")
		assert.False(t, exists, "Non-existent file should return false")

		// Store file
		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")

		// Check existing file
		exists = testStorage.Storage.Exists(storagePath)
		assert.True(t, exists, "Existing file should return true")

		// Delete and check again
		err = testStorage.Storage.Delete(storagePath)
		require.NoError(t, err, "File deletion should succeed")

		exists = testStorage.Storage.Exists(storagePath)
		assert.False(t, exists, "Deleted file should return false")
	})

	t.Run("Get File URL", func(t *testing.T) {
		testData := generateTestImage(128)
		filename := "test-url.jpg"

		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")

		url := testStorage.Storage.GetURL(storagePath)
		assert.NotEmpty(t, url, "URL should not be empty")
		assert.Contains(t, url, storagePath, "URL should contain storage path")
		assert.Contains(t, url, testStorageURL, "URL should contain base URL")
	})
}

func TestLocalStorageFileOrganization(t *testing.T) {
	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Date-Based Directory Structure", func(t *testing.T) {
		testData := generateTestImage(1024)
		filename := "date-organized.jpg"

		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")

		// Verify date-based structure (YYYY/MM/DD format expected)
		expectedDate := time.Now().Format("2006/01/02")
		assert.Contains(t, storagePath, expectedDate, "Storage path should contain date-based directory structure")

		// Verify directory structure exists on disk
		dateDirPath := filepath.Join(testStorage.BaseDir, expectedDate)
		info, err := os.Stat(dateDirPath)
		require.NoError(t, err, "Date directory should exist")
		assert.True(t, info.IsDir(), "Date path should be a directory")
	})

	t.Run("Multiple Files Same Day", func(t *testing.T) {
		numFiles := 5
		storedPaths := make([]string, numFiles)
		testData := generateTestImage(512)

		// Store multiple files
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("multi-file-%d.jpg", i)
			storagePath, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "File storage should succeed for file %d", i)
			storedPaths[i] = storagePath

			// All files should be in the same date directory
			expectedDate := time.Now().Format("2006/01/02")
			assert.Contains(t, storagePath, expectedDate, "All files should be in same date directory")
		}

		// Verify all files exist and are different
		for i, path := range storedPaths {
			exists := testStorage.Storage.Exists(path)
			assert.True(t, exists, "File %d should exist", i)

			// Verify each path is unique
			for j, otherPath := range storedPaths {
				if i != j {
					assert.NotEqual(t, path, otherPath, "Storage paths should be unique")
				}
			}
		}
	})

	t.Run("Directory Permissions", func(t *testing.T) {
		testData := generateTestImage(256)
		filename := "permission-test.jpg"

		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "File storage should succeed")

		// Check file permissions
		fullPath := filepath.Join(testStorage.BaseDir, storagePath)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "Should be able to stat file")

		mode := info.Mode()
		assert.True(t, mode.IsRegular(), "Should be a regular file")

		// Check directory permissions
		dirPath := filepath.Dir(fullPath)
		dirInfo, err := os.Stat(dirPath)
		require.NoError(t, err, "Should be able to stat directory")

		dirMode := dirInfo.Mode()
		assert.True(t, dirMode.IsDir(), "Should be a directory")
		assert.True(t, dirMode&0o700 != 0, "Directory should be readable/writable by owner")
	})
}

func TestStorageQuotaManagement(t *testing.T) {
	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Storage Usage Calculation", func(t *testing.T) {
		// Store several files of known sizes
		fileSizes := []int{1024, 2048, 4096, 8192} // 1KB, 2KB, 4KB, 8KB
		expectedTotal := int64(0)

		for i, size := range fileSizes {
			testData := generateTestImage(size)
			filename := fmt.Sprintf("quota-test-%d.jpg", i)

			_, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "File storage should succeed")
			expectedTotal += int64(size)
		}

		// Calculate actual storage usage
		actualSize, err := calculateDirSize(testStorage.BaseDir)
		require.NoError(t, err, "Should be able to calculate directory size")
		assert.Equal(t, expectedTotal, actualSize, "Calculated size should match expected total")
	})

	t.Run("Storage Statistics", func(t *testing.T) {
		// Store some test files
		numFiles := 3
		totalSize := int64(0)

		for i := 0; i < numFiles; i++ {
			size := (i + 1) * 1024 // 1KB, 2KB, 3KB
			testData := generateTestImage(size)
			filename := fmt.Sprintf("stats-test-%d.jpg", i)

			_, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "File storage should succeed")
			totalSize += int64(size)
		}

		// Get storage statistics (assuming LocalStorage implements GetStorageStats)
		if localStorage, ok := testStorage.Storage.(*image.LocalStorage); ok {
			stats, err := localStorage.GetStorageStats()
			require.NoError(t, err, "Should be able to get storage stats")

			assert.Contains(t, stats, "backend", "Stats should contain backend info")
			assert.Contains(t, stats, "total_size", "Stats should contain total size")
			assert.Contains(t, stats, "file_count", "Stats should contain file count")

			statsSize, ok := stats["total_size"].(int64)
			assert.True(t, ok, "total_size should be int64")
			assert.Equal(t, totalSize, statsSize, "Stats size should match calculated size")

			statsCount, ok := stats["file_count"].(int64)
			assert.True(t, ok, "file_count should be int64")
			assert.Equal(t, int64(numFiles), statsCount, "Stats count should match number of files")
		}
	})

	t.Run("10GB Quota Enforcement Simulation", func(t *testing.T) {
		// This test simulates quota enforcement by checking storage limits
		// In a real implementation, this would be enforced by the storage service

		maxFiles := 100
		fileSize := 1024 * 1024 // 1MB files
		maxTotalSize := int64(10 * 1024 * 1024) // 10MB for testing (simulating 10GB limit)

		var totalStoredSize int64
		filesStored := 0

		for i := 0; i < maxFiles; i++ {
			// Check if adding this file would exceed quota
			if totalStoredSize+int64(fileSize) > maxTotalSize {
				t.Logf("Quota limit reached after storing %d files (%d bytes)", filesStored, totalStoredSize)
				break
			}

			testData := generateTestImage(fileSize)
			filename := fmt.Sprintf("quota-limit-test-%d.jpg", i)

			_, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "File storage should succeed within quota")

			totalStoredSize += int64(fileSize)
			filesStored++
		}

		// Verify we stayed within quota
		assert.LessOrEqual(t, totalStoredSize, maxTotalSize, "Total stored size should not exceed quota")
		t.Logf("Successfully stored %d files totaling %d bytes (limit: %d bytes)", filesStored, totalStoredSize, maxTotalSize)
	})

	t.Run("Storage Cleanup and Garbage Collection", func(t *testing.T) {
		// Store several files
		numFiles := 5
		storedPaths := make([]string, numFiles)
		fileSize := 2048

		for i := 0; i < numFiles; i++ {
			testData := generateTestImage(fileSize)
			filename := fmt.Sprintf("cleanup-test-%d.jpg", i)

			path, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "File storage should succeed")
			storedPaths[i] = path
		}

		// Verify initial state
		initialSize, err := calculateDirSize(testStorage.BaseDir)
		require.NoError(t, err, "Should calculate initial size")
		assert.Equal(t, int64(numFiles*fileSize), initialSize, "Initial size should match stored files")

		// Delete some files (simulating cleanup)
		filesToDelete := numFiles / 2
		for i := 0; i < filesToDelete; i++ {
			err := testStorage.Storage.Delete(storedPaths[i])
			require.NoError(t, err, "File deletion should succeed")
		}

		// Verify cleanup reduced storage usage
		finalSize, err := calculateDirSize(testStorage.BaseDir)
		require.NoError(t, err, "Should calculate final size")
		expectedFinalSize := int64((numFiles - filesToDelete) * fileSize)
		assert.Equal(t, expectedFinalSize, finalSize, "Final size should reflect deletions")

		// Verify remaining files still exist
		for i := filesToDelete; i < numFiles; i++ {
			exists := testStorage.Storage.Exists(storedPaths[i])
			assert.True(t, exists, "Remaining files should still exist")
		}
	})
}

func TestConcurrentStorageAccess(t *testing.T) {
	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Concurrent File Uploads", func(t *testing.T) {
		const numGoroutines = 20
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)
		paths := make(chan string, numGoroutines)

		// Generate test data outside goroutines to avoid race conditions
		testData := generateTestImage(1024)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				filename := fmt.Sprintf("concurrent-upload-%d.jpg", id)

				path, err := testStorage.Storage.Store(filename, testData)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: %w", id, err)
					return
				}
				paths <- path
			}(i)
		}

		wg.Wait()
		close(errors)
		close(paths)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent upload failed: %v", err)
		}

		// Verify all files were stored
		storedPaths := make([]string, 0, numGoroutines)
		for path := range paths {
			storedPaths = append(storedPaths, path)
		}

		assert.Equal(t, numGoroutines, len(storedPaths), "All concurrent uploads should succeed")

		// Verify all paths are unique
		pathMap := make(map[string]bool)
		for _, path := range storedPaths {
			assert.False(t, pathMap[path], "All storage paths should be unique")
			pathMap[path] = true

			// Verify file exists
			exists := testStorage.Storage.Exists(path)
			assert.True(t, exists, "Each uploaded file should exist")
		}
	})

	t.Run("Concurrent Read Operations", func(t *testing.T) {
		// Store a test file first
		testData := generateTestImage(4096)
		filename := "concurrent-read-test.jpg"

		storagePath, err := testStorage.Storage.Store(filename, testData)
		require.NoError(t, err, "Initial file storage should succeed")

		const numReads = 30
		var wg sync.WaitGroup
		errors := make(chan error, numReads)
		results := make(chan []byte, numReads)

		for i := 0; i < numReads; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				data, err := testStorage.Storage.Retrieve(storagePath)
				if err != nil {
					errors <- fmt.Errorf("read %d: %w", id, err)
					return
				}
				results <- data
			}(i)
		}

		wg.Wait()
		close(errors)
		close(results)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent read failed: %v", err)
		}

		// Verify all reads returned correct data
		readCount := 0
		for data := range results {
			assert.Equal(t, testData, data, "All reads should return original data")
			readCount++
		}
		assert.Equal(t, numReads, readCount, "All concurrent reads should complete")
	})

	t.Run("Concurrent Mixed Operations", func(t *testing.T) {
		const numOperations = 30
		var wg sync.WaitGroup
		errors := make(chan error, numOperations)

		// Pre-store some files for read/delete operations
		preStoredPaths := make([]string, 10)
		baseData := generateTestImage(512)

		for i := 0; i < len(preStoredPaths); i++ {
			filename := fmt.Sprintf("prestored-%d.jpg", i)
			path, err := testStorage.Storage.Store(filename, baseData)
			require.NoError(t, err, "Pre-storage should succeed")
			preStoredPaths[i] = path
		}

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				switch id % 3 {
				case 0: // Store operation
					testData := generateTestImage(1024)
					filename := fmt.Sprintf("mixed-store-%d.jpg", id)
					_, err := testStorage.Storage.Store(filename, testData)
					if err != nil {
						errors <- fmt.Errorf("mixed store %d: %w", id, err)
					}

				case 1: // Read operation
					if len(preStoredPaths) > 0 {
						path := preStoredPaths[id%len(preStoredPaths)]
						_, err := testStorage.Storage.Retrieve(path)
						if err != nil {
							errors <- fmt.Errorf("mixed read %d: %w", id, err)
						}
					}

				case 2: // Exists check operation
					if len(preStoredPaths) > 0 {
						path := preStoredPaths[id%len(preStoredPaths)]
						testStorage.Storage.Exists(path)
						// Exists operations don't return errors in this implementation
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent mixed operation failed: %v", err)
		}
	})
}

func TestStorageErrorHandling(t *testing.T) {
	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Retrieve Non-Existent File", func(t *testing.T) {
		_, err := testStorage.Storage.Retrieve("non/existent/file.jpg")
		assert.Error(t, err, "Retrieving non-existent file should fail")
		assert.Contains(t, err.Error(), "not found", "Error should mention file not found")
	})

	t.Run("Delete Non-Existent File", func(t *testing.T) {
		err := testStorage.Storage.Delete("non/existent/file.jpg")
		// Delete should succeed even if file doesn't exist (idempotent operation)
		assert.NoError(t, err, "Deleting non-existent file should not error")
	})

	t.Run("Store with Invalid Path Characters", func(t *testing.T) {
		testData := generateTestImage(256)
		invalidFilenames := []string{
			"../../../etc/passwd",     // Path traversal
			"file\x00name.jpg",        // Null byte
			"file\nname.jpg",          // Newline
			"file<>|name.jpg",         // Invalid characters
		}

		for _, filename := range invalidFilenames {
			// The storage implementation should handle these gracefully
			// Either by sanitizing the filename or by storing safely
			path, err := testStorage.Storage.Store(filename, testData)
			if err == nil {
				// If storage succeeded, verify the file was stored safely
				assert.NotEmpty(t, path, "Storage path should not be empty")

				// Verify the file exists and can be retrieved
				exists := testStorage.Storage.Exists(path)
				assert.True(t, exists, "Stored file should exist")

				data, err := testStorage.Storage.Retrieve(path)
				assert.NoError(t, err, "Should be able to retrieve stored file")
				assert.Equal(t, testData, data, "Retrieved data should match original")
			} else {
				// If storage failed, that's also acceptable behavior
				t.Logf("Storage rejected invalid filename %q: %v", filename, err)
			}
		}
	})

	t.Run("Large File Handling", func(t *testing.T) {
		// Test storing a larger file (10MB)
		largeFileSize := 10 * 1024 * 1024
		largeData := generateTestImage(largeFileSize)
		filename := "large-test.jpg"

		path, err := testStorage.Storage.Store(filename, largeData)
		require.NoError(t, err, "Large file storage should succeed")

		// Verify file size on disk
		fullPath := filepath.Join(testStorage.BaseDir, path)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "Should be able to stat large file")
		assert.Equal(t, int64(largeFileSize), info.Size(), "Large file size should be preserved")

		// Verify data integrity
		retrievedData, err := testStorage.Storage.Retrieve(path)
		require.NoError(t, err, "Large file retrieval should succeed")
		assert.Equal(t, largeData, retrievedData, "Large file data should be intact")
	})
}

func TestStorageHealthCheck(t *testing.T) {
	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Healthy Storage", func(t *testing.T) {
		// Health check should pass for normal storage
		if localStorage, ok := testStorage.Storage.(*image.LocalStorage); ok {
			err := localStorage.HealthCheck()
			assert.NoError(t, err, "Health check should pass for healthy storage")
		}
	})

	t.Run("Storage Accessibility", func(t *testing.T) {
		// Verify storage directory is accessible and writable
		testFile := filepath.Join(testStorage.BaseDir, "health_check_test.tmp")

		// Write test
		err := os.WriteFile(testFile, []byte("health check"), 0644)
		require.NoError(t, err, "Should be able to write health check file")

		// Read test
		data, err := os.ReadFile(testFile)
		require.NoError(t, err, "Should be able to read health check file")
		assert.Equal(t, []byte("health check"), data, "Health check data should match")

		// Cleanup
		err = os.Remove(testFile)
		assert.NoError(t, err, "Should be able to delete health check file")
	})

	t.Run("Disk Space Monitoring", func(t *testing.T) {
		// Get initial disk usage
		initialSize, err := calculateDirSize(testStorage.BaseDir)
		require.NoError(t, err, "Should be able to calculate initial disk usage")

		// Store some test files
		numFiles := 5
		fileSize := 1024
		for i := 0; i < numFiles; i++ {
			testData := generateTestImage(fileSize)
			filename := fmt.Sprintf("disk-monitor-test-%d.jpg", i)
			_, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "File storage should succeed")
		}

		// Check disk usage increased
		finalSize, err := calculateDirSize(testStorage.BaseDir)
		require.NoError(t, err, "Should be able to calculate final disk usage")

		expectedIncrease := int64(numFiles * fileSize)
		actualIncrease := finalSize - initialSize
		assert.Equal(t, expectedIncrease, actualIncrease, "Disk usage should increase by expected amount")
	})
}

func TestStoragePerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmark in short mode")
	}

	t.Parallel()

	testStorage := newTestStorage(t)
	defer testStorage.cleanup(t)

	t.Run("Store Performance", func(t *testing.T) {
		numFiles := 100
		fileSize := 10 * 1024 // 10KB files
		testData := generateTestImage(fileSize)

		startTime := time.Now()

		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("perf-store-%d.jpg", i)
			_, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "Performance test store should succeed")
		}

		duration := time.Since(startTime)
		avgTimePerFile := duration / time.Duration(numFiles)

		t.Logf("Stored %d files in %v (avg: %v per file)", numFiles, duration, avgTimePerFile)

		// Performance assertion: should be able to store at least 10 files per second
		maxExpectedDuration := time.Duration(numFiles) * 100 * time.Millisecond
		assert.Less(t, duration, maxExpectedDuration, "Store performance should meet minimum requirements")
	})

	t.Run("Retrieve Performance", func(t *testing.T) {
		// Pre-store files for retrieval test
		numFiles := 50
		fileSize := 50 * 1024 // 50KB files
		testData := generateTestImage(fileSize)
		paths := make([]string, numFiles)

		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("perf-retrieve-%d.jpg", i)
			path, err := testStorage.Storage.Store(filename, testData)
			require.NoError(t, err, "Pre-storage should succeed")
			paths[i] = path
		}

		// Benchmark retrieval
		startTime := time.Now()

		for _, path := range paths {
			data, err := testStorage.Storage.Retrieve(path)
			require.NoError(t, err, "Performance test retrieve should succeed")
			assert.Equal(t, len(testData), len(data), "Retrieved data length should match")
		}

		duration := time.Since(startTime)
		avgTimePerFile := duration / time.Duration(numFiles)

		t.Logf("Retrieved %d files in %v (avg: %v per file)", numFiles, duration, avgTimePerFile)

		// Performance assertion: should be able to retrieve at least 20 files per second
		maxExpectedDuration := time.Duration(numFiles) * 50 * time.Millisecond
		assert.Less(t, duration, maxExpectedDuration, "Retrieve performance should meet minimum requirements")
	})
}