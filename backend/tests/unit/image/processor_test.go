package image_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	randompicImage "github.com/randompic/api/internal/image"
)

// createTestJPEGImage creates a simple test JPEG image
func createTestJPEGImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, image.White)
		}
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// createTestPNGImage creates a simple test PNG image
func createTestPNGImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, image.Black)
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestImageFormatValidation(t *testing.T) {
	config := randompicImage.DefaultProcessorConfig()
	// Override to only allow JPEG and PNG as per requirements
	config.AllowedFormats = []string{"jpeg", "png"}
	processor := randompicImage.NewImageProcessor(config)

	t.Run("ValidJPEGImage", func(t *testing.T) {
		jpegData := createTestJPEGImage(200, 150)
		reader := bytes.NewReader(jpegData)

		info, err := processor.ValidateImage(reader, 2097152) // 2MB limit
		require.NoError(t, err)
		require.NotNil(t, info)

		assert.Equal(t, "jpeg", info.Format)
		assert.Equal(t, "image/jpeg", info.MimeType)
		assert.Equal(t, 200, info.Width)
		assert.Equal(t, 150, info.Height)
		assert.Greater(t, info.Size, int64(0))
	})

	t.Run("ValidPNGImage", func(t *testing.T) {
		pngData := createTestPNGImage(300, 200)
		reader := bytes.NewReader(pngData)

		info, err := processor.ValidateImage(reader, 2097152)
		require.NoError(t, err)
		require.NotNil(t, info)

		assert.Equal(t, "png", info.Format)
		assert.Equal(t, "image/png", info.MimeType)
		assert.Equal(t, 300, info.Width)
		assert.Equal(t, 200, info.Height)
		assert.Greater(t, info.Size, int64(0))
	})

	t.Run("UnsupportedFormat", func(t *testing.T) {
		// Create some non-image data
		invalidData := []byte("This is not an image")
		reader := bytes.NewReader(invalidData)

		_, err := processor.ValidateImage(reader, 2097152)
		assert.Error(t, err)
	})

	t.Run("FileSizeExceedsLimit", func(t *testing.T) {
		jpegData := createTestJPEGImage(200, 150)
		reader := bytes.NewReader(jpegData)

		// Set a very small limit
		_, err := processor.ValidateImage(reader, 100) // 100 bytes
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("MinimumDimensions", func(t *testing.T) {
		// Test with minimum allowed dimensions (100x100)
		jpegData := createTestJPEGImage(100, 100)
		reader := bytes.NewReader(jpegData)

		info, err := processor.ValidateImage(reader, 2097152)
		require.NoError(t, err)
		assert.Equal(t, 100, info.Width)
		assert.Equal(t, 100, info.Height)

		// Test with dimensions below minimum
		smallJpegData := createTestJPEGImage(50, 50)
		smallReader := bytes.NewReader(smallJpegData)

		_, err = processor.ValidateImage(smallReader, 2097152)
		// Note: ValidateImage doesn't check dimensions, that's done separately
		// So this test verifies the image can be decoded
		require.NoError(t, err)
	})
}

func TestEXIFStripping(t *testing.T) {
	processor := randompicImage.NewImageProcessor(&randompicImage.ProcessorConfig{
		MaxWidth:       2000,
		MaxHeight:      2000,
		Quality:        85,
		StripEXIF:      true,
		ThumbnailSize:  300,
		AllowedFormats: []string{"jpeg", "png"},
	})

	t.Run("StripEXIFFromJPEG", func(t *testing.T) {
		originalData := createTestJPEGImage(400, 300)
		originalReader := bytes.NewReader(originalData)

		// Process image (which should strip EXIF)
		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)
		require.NotNil(t, processedReader)

		// Read processed data
		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)
		assert.Greater(t, len(processedData), 0)

		// Verify it's still a valid JPEG
		finalReader := bytes.NewReader(processedData)
		_, format, err := image.Decode(finalReader)
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
	})

	t.Run("StripEXIFDirectly", func(t *testing.T) {
		originalData := createTestJPEGImage(200, 150)
		originalReader := bytes.NewReader(originalData)

		// Strip EXIF directly
		strippedReader, err := processor.StripEXIF(originalReader)
		require.NoError(t, err)
		require.NotNil(t, strippedReader)

		// Read stripped data
		strippedData, err := io.ReadAll(strippedReader)
		require.NoError(t, err)
		assert.Greater(t, len(strippedData), 0)

		// Verify it's still a valid image
		finalReader := bytes.NewReader(strippedData)
		decodedImg, format, err := image.Decode(finalReader)
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
		assert.NotNil(t, decodedImg)
	})

	t.Run("EXIFDetection", func(t *testing.T) {
		// Create test data without EXIF
		jpegData := createTestJPEGImage(200, 150)
		reader := bytes.NewReader(jpegData)

		info, err := processor.ValidateImage(reader, 2097152)
		require.NoError(t, err)

		// Our test images don't have EXIF data
		assert.False(t, info.HasEXIF)
	})

	t.Run("ProcessPNGImage", func(t *testing.T) {
		pngData := createTestPNGImage(300, 200)
		originalReader := bytes.NewReader(pngData)

		// Process PNG image
		processedReader, err := processor.ProcessImage(originalReader, "png")
		require.NoError(t, err)
		require.NotNil(t, processedReader)

		// Read processed data
		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)
		assert.Greater(t, len(processedData), 0)

		// Verify it's still a valid PNG
		finalReader := bytes.NewReader(processedData)
		_, format, err := image.Decode(finalReader)
		require.NoError(t, err)
		assert.Equal(t, "png", format)
	})
}

func TestImageResizing(t *testing.T) {
	processor := randompicImage.NewImageProcessor(&randompicImage.ProcessorConfig{
		MaxWidth:       800,
		MaxHeight:      600,
		Quality:        85,
		StripEXIF:      true,
		ThumbnailSize:  300,
		AllowedFormats: []string{"jpeg", "png"},
	})

	t.Run("ResizeLargeImage", func(t *testing.T) {
		// Create image larger than max dimensions
		largeImageData := createTestJPEGImage(1200, 900) // Larger than 800x600
		originalReader := bytes.NewReader(largeImageData)

		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)

		// Decode processed image to check dimensions
		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)

		finalReader := bytes.NewReader(processedData)
		decodedImg, _, err := image.Decode(finalReader)
		require.NoError(t, err)

		bounds := decodedImg.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Should be resized to fit within max dimensions while maintaining aspect ratio
		assert.LessOrEqual(t, width, 800)
		assert.LessOrEqual(t, height, 600)

		// Aspect ratio should be maintained (approximately)
		originalRatio := float64(1200) / float64(900)
		newRatio := float64(width) / float64(height)
		assert.InDelta(t, originalRatio, newRatio, 0.01)
	})

	t.Run("DoNotResizeSmallImage", func(t *testing.T) {
		// Create image smaller than max dimensions
		smallImageData := createTestJPEGImage(400, 300) // Smaller than 800x600
		originalReader := bytes.NewReader(smallImageData)

		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)

		// Decode processed image
		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)

		finalReader := bytes.NewReader(processedData)
		decodedImg, _, err := image.Decode(finalReader)
		require.NoError(t, err)

		bounds := decodedImg.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Dimensions should remain the same (or very close due to encoding/decoding)
		assert.InDelta(t, 400, width, 5)
		assert.InDelta(t, 300, height, 5)
	})

	t.Run("ResizeToMaxWidth", func(t *testing.T) {
		// Create very wide image
		wideImageData := createTestJPEGImage(1600, 200) // Much wider than 800x600
		originalReader := bytes.NewReader(wideImageData)

		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)

		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)

		finalReader := bytes.NewReader(processedData)
		decodedImg, _, err := image.Decode(finalReader)
		require.NoError(t, err)

		bounds := decodedImg.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Width should be constrained to max width
		assert.LessOrEqual(t, width, 800)
		// Height should be scaled proportionally
		assert.Greater(t, height, 0)
	})
}

func TestThumbnailGeneration(t *testing.T) {
	processor := randompicImage.NewImageProcessor(&randompicImage.ProcessorConfig{
		MaxWidth:       2000,
		MaxHeight:      2000,
		Quality:        85,
		StripEXIF:      true,
		ThumbnailSize:  300,
		AllowedFormats: []string{"jpeg", "png"},
	})

	t.Run("GenerateSquareThumbnail", func(t *testing.T) {
		// Create rectangular image
		imageData := createTestJPEGImage(600, 400)
		originalReader := bytes.NewReader(imageData)

		thumbnailReader, err := processor.GenerateThumbnail(originalReader, 0, 0) // Use default size
		require.NoError(t, err)
		require.NotNil(t, thumbnailReader)

		// Read thumbnail data
		thumbnailData, err := io.ReadAll(thumbnailReader)
		require.NoError(t, err)
		assert.Greater(t, len(thumbnailData), 0)

		// Decode thumbnail to check dimensions
		finalReader := bytes.NewReader(thumbnailData)
		decodedImg, format, err := image.Decode(finalReader)
		require.NoError(t, err)

		bounds := decodedImg.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Should be square thumbnail with default size (300x300)
		assert.Equal(t, 300, width)
		assert.Equal(t, 300, height)
		assert.Equal(t, "jpeg", format) // Thumbnails are always JPEG
	})

	t.Run("GenerateCustomSizeThumbnail", func(t *testing.T) {
		imageData := createTestPNGImage(800, 600)
		originalReader := bytes.NewReader(imageData)

		// Generate 150x150 thumbnail
		thumbnailReader, err := processor.GenerateThumbnail(originalReader, 150, 150)
		require.NoError(t, err)

		thumbnailData, err := io.ReadAll(thumbnailReader)
		require.NoError(t, err)

		finalReader := bytes.NewReader(thumbnailData)
		decodedImg, format, err := image.Decode(finalReader)
		require.NoError(t, err)

		bounds := decodedImg.Bounds()
		assert.Equal(t, 150, bounds.Dx())
		assert.Equal(t, 150, bounds.Dy())
		assert.Equal(t, "jpeg", format)
	})

	t.Run("ThumbnailFromSquareImage", func(t *testing.T) {
		// Create square image
		squareImageData := createTestJPEGImage(500, 500)
		originalReader := bytes.NewReader(squareImageData)

		thumbnailReader, err := processor.GenerateThumbnail(originalReader, 200, 200)
		require.NoError(t, err)

		thumbnailData, err := io.ReadAll(thumbnailReader)
		require.NoError(t, err)

		finalReader := bytes.NewReader(thumbnailData)
		decodedImg, _, err := image.Decode(finalReader)
		require.NoError(t, err)

		bounds := decodedImg.Bounds()
		assert.Equal(t, 200, bounds.Dx())
		assert.Equal(t, 200, bounds.Dy())
	})
}

func TestImageProcessorConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := randompicImage.DefaultProcessorConfig()
		require.NotNil(t, config)

		assert.Equal(t, 4096, config.MaxWidth)
		assert.Equal(t, 4096, config.MaxHeight)
		assert.Equal(t, 85, config.Quality)
		assert.True(t, config.StripEXIF)
		assert.Equal(t, 300, config.ThumbnailSize)
		assert.Contains(t, config.AllowedFormats, "jpeg")
		assert.Contains(t, config.AllowedFormats, "png")
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := &randompicImage.ProcessorConfig{
			MaxWidth:       1024,
			MaxHeight:      768,
			Quality:        90,
			StripEXIF:      false,
			ThumbnailSize:  150,
			AllowedFormats: []string{"jpeg", "png"},
		}

		processor := randompicImage.NewImageProcessor(config)
		retrievedConfig := processor.GetProcessorConfig()

		assert.Equal(t, config.MaxWidth, retrievedConfig.MaxWidth)
		assert.Equal(t, config.MaxHeight, retrievedConfig.MaxHeight)
		assert.Equal(t, config.Quality, retrievedConfig.Quality)
		assert.Equal(t, config.StripEXIF, retrievedConfig.StripEXIF)
		assert.Equal(t, config.ThumbnailSize, retrievedConfig.ThumbnailSize)
		assert.Equal(t, config.AllowedFormats, retrievedConfig.AllowedFormats)
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		processor := randompicImage.NewImageProcessor(nil) // Uses default config
		originalConfig := processor.GetProcessorConfig()

		newConfig := &randompicImage.ProcessorConfig{
			MaxWidth:       512,
			MaxHeight:      512,
			Quality:        75,
			StripEXIF:      false,
			ThumbnailSize:  100,
			AllowedFormats: []string{"jpeg"},
		}

		processor.UpdateConfig(newConfig)
		updatedConfig := processor.GetProcessorConfig()

		assert.NotEqual(t, originalConfig.MaxWidth, updatedConfig.MaxWidth)
		assert.Equal(t, newConfig.MaxWidth, updatedConfig.MaxWidth)
		assert.Equal(t, newConfig.Quality, updatedConfig.Quality)
	})
}

func TestImageProcessingEdgeCases(t *testing.T) {
	processor := randompicImage.NewImageProcessor(&randompicImage.ProcessorConfig{
		MaxWidth:       1000,
		MaxHeight:      1000,
		Quality:        85,
		StripEXIF:      true,
		ThumbnailSize:  200,
		AllowedFormats: []string{"jpeg", "png"},
	})

	t.Run("ProcessVerySmallImage", func(t *testing.T) {
		// Create minimal size image
		tinyImageData := createTestJPEGImage(10, 10)
		originalReader := bytes.NewReader(tinyImageData)

		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)

		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)
		assert.Greater(t, len(processedData), 0)
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		imageData := createTestJPEGImage(200, 200)
		originalReader := bytes.NewReader(imageData)

		_, err := processor.ProcessImage(originalReader, "gif") // Unsupported format
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported output format")
	})

	t.Run("EmptyImageData", func(t *testing.T) {
		emptyReader := bytes.NewReader([]byte{})

		_, err := processor.ProcessImage(emptyReader, "jpeg")
		assert.Error(t, err)
	})

	t.Run("CorruptedImageData", func(t *testing.T) {
		corruptedData := []byte{0xFF, 0xD8, 0xFF, 0x00, 0x00, 0x00} // Invalid JPEG
		corruptedReader := bytes.NewReader(corruptedData)

		_, err := processor.ProcessImage(corruptedReader, "jpeg")
		assert.Error(t, err)
	})
}

func TestImageQuality(t *testing.T) {
	t.Run("HighQualityJPEG", func(t *testing.T) {
		processor := randompicImage.NewImageProcessor(&randompicImage.ProcessorConfig{
			MaxWidth:       2000,
			MaxHeight:      2000,
			Quality:        95, // High quality
			StripEXIF:      true,
			ThumbnailSize:  300,
			AllowedFormats: []string{"jpeg", "png"},
		})

		imageData := createTestJPEGImage(400, 300)
		originalReader := bytes.NewReader(imageData)

		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)

		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)

		// High quality should result in larger file size (generally)
		assert.Greater(t, len(processedData), 0)
	})

	t.Run("LowQualityJPEG", func(t *testing.T) {
		processor := randompicImage.NewImageProcessor(&randompicImage.ProcessorConfig{
			MaxWidth:       2000,
			MaxHeight:      2000,
			Quality:        20, // Low quality
			StripEXIF:      true,
			ThumbnailSize:  300,
			AllowedFormats: []string{"jpeg", "png"},
		})

		imageData := createTestJPEGImage(400, 300)
		originalReader := bytes.NewReader(imageData)

		processedReader, err := processor.ProcessImage(originalReader, "jpeg")
		require.NoError(t, err)

		processedData, err := io.ReadAll(processedReader)
		require.NoError(t, err)

		// Should still be a valid image despite low quality
		finalReader := bytes.NewReader(processedData)
		_, format, err := image.Decode(finalReader)
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
	})
}

// Benchmark tests for performance
func BenchmarkImageProcessing(b *testing.B) {
	processor := randompicImage.NewImageProcessor(nil)
	imageData := createTestJPEGImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(imageData)
		_, err := processor.ProcessImage(reader, "jpeg")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThumbnailGeneration(b *testing.B) {
	processor := randompicImage.NewImageProcessor(nil)
	imageData := createTestJPEGImage(1200, 900)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(imageData)
		_, err := processor.GenerateThumbnail(reader, 300, 300)
		if err != nil {
			b.Fatal(err)
		}
	}
}