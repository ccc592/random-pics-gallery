package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"

	"golang.org/x/image/draw"
)

// ImageInfo represents processed image information
type ImageInfo struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	HasEXIF  bool   `json:"has_exif"`
}

// ProcessorConfig holds image processing configuration
type ProcessorConfig struct {
	MaxWidth      int     `json:"max_width"`
	MaxHeight     int     `json:"max_height"`
	Quality       int     `json:"quality"`        // JPEG quality (1-100)
	StripEXIF     bool    `json:"strip_exif"`     // Whether to strip EXIF data
	ThumbnailSize int     `json:"thumbnail_size"` // Thumbnail size (square)
	AllowedFormats []string `json:"allowed_formats"`
}

// ImageProcessor handles image processing operations
type ImageProcessor struct {
	config *ProcessorConfig
}

// DefaultProcessorConfig returns default processor configuration
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		MaxWidth:      4096,
		MaxHeight:     4096,
		Quality:       85,
		StripEXIF:     true,
		ThumbnailSize: 300,
		AllowedFormats: []string{"jpeg", "png", "webp"},
	}
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(config *ProcessorConfig) *ImageProcessor {
	if config == nil {
		config = DefaultProcessorConfig()
	}

	return &ImageProcessor{
		config: config,
	}
}

// ValidateImage validates an image file and returns its information
func (p *ImageProcessor) ValidateImage(file io.Reader, maxSize int64) (*ImageInfo, error) {
	// Read the file data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Check file size
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", len(data), maxSize)
	}

	// Detect content type
	mimeType := http.DetectContentType(data)
	format := p.mimeTypeToFormat(mimeType)

	// Check if format is supported
	if !p.isFormatAllowed(format) {
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	// Decode image to get dimensions
	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()

	return &ImageInfo{
		Width:    bounds.Dx(),
		Height:   bounds.Dy(),
		Format:   format,
		Size:     int64(len(data)),
		MimeType: mimeType,
		HasEXIF:  p.detectEXIF(data),
	}, nil
}

// ProcessImage processes an image according to configuration
func (p *ImageProcessor) ProcessImage(input io.Reader, format string) (io.Reader, error) {
	// Read input data
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Decode image
	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if needed
	processedImg := p.resizeIfNeeded(img)

	// Strip EXIF if configured
	var output bytes.Buffer
	if p.config.StripEXIF {
		err = p.encodeWithoutEXIF(processedImg, &output, format)
	} else {
		err = p.encode(processedImg, &output, format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode processed image: %w", err)
	}

	return bytes.NewReader(output.Bytes()), nil
}

// StripEXIF removes EXIF data from an image
func (p *ImageProcessor) StripEXIF(input io.Reader) (io.Reader, error) {
	// Read input data
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Decode image
	reader := bytes.NewReader(data)
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Re-encode without EXIF
	var output bytes.Buffer
	err = p.encodeWithoutEXIF(img, &output, format)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image without EXIF: %w", err)
	}

	return bytes.NewReader(output.Bytes()), nil
}

// GenerateThumbnail generates a square thumbnail of specified size
func (p *ImageProcessor) GenerateThumbnail(input io.Reader, width, height int) (io.Reader, error) {
	// Use configured thumbnail size if width/height are 0
	if width == 0 || height == 0 {
		width = p.config.ThumbnailSize
		height = p.config.ThumbnailSize
	}

	// Read input data
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Decode image
	reader := bytes.NewReader(data)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Create thumbnail
	thumbnail := p.createThumbnail(img, width, height)

	// Encode as JPEG for thumbnails (smaller file size)
	var output bytes.Buffer
	err = jpeg.Encode(&output, thumbnail, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return bytes.NewReader(output.Bytes()), nil
}

// resizeIfNeeded resizes the image if it exceeds maximum dimensions
func (p *ImageProcessor) resizeIfNeeded(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if resizing is needed
	if width <= p.config.MaxWidth && height <= p.config.MaxHeight {
		return img
	}

	// Calculate new dimensions maintaining aspect ratio
	newWidth, newHeight := p.calculateResizedDimensions(width, height)

	// Create new image with resized dimensions
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)

	return resized
}

// calculateResizedDimensions calculates new dimensions maintaining aspect ratio
func (p *ImageProcessor) calculateResizedDimensions(width, height int) (int, int) {
	maxWidth := float64(p.config.MaxWidth)
	maxHeight := float64(p.config.MaxHeight)

	widthRatio := maxWidth / float64(width)
	heightRatio := maxHeight / float64(height)

	// Use the smaller ratio to maintain aspect ratio
	ratio := widthRatio
	if heightRatio < widthRatio {
		ratio = heightRatio
	}

	return int(float64(width) * ratio), int(float64(height) * ratio)
}

// createThumbnail creates a square thumbnail with center cropping
func (p *ImageProcessor) createThumbnail(img image.Image, size int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate square crop dimensions
	cropSize := width
	if height < width {
		cropSize = height
	}

	// Calculate crop offsets for center cropping
	offsetX := (width - cropSize) / 2
	offsetY := (height - cropSize) / 2

	// Crop to square
	cropRect := image.Rect(offsetX, offsetY, offsetX+cropSize, offsetY+cropSize)
	cropped := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(cropRect)

	// Resize to thumbnail size
	thumbnail := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	return thumbnail
}

// encode encodes an image in the specified format
func (p *ImageProcessor) encode(img image.Image, output io.Writer, format string) error {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(output, img, &jpeg.Options{Quality: p.config.Quality})
	case "png":
		return png.Encode(output, img)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// encodeWithoutEXIF encodes an image without EXIF data (always strips)
func (p *ImageProcessor) encodeWithoutEXIF(img image.Image, output io.Writer, format string) error {
	// This method always creates a new image without any metadata
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(output, img, &jpeg.Options{Quality: p.config.Quality})
	case "png":
		return png.Encode(output, img)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// mimeTypeToFormat converts MIME type to format string
func (p *ImageProcessor) mimeTypeToFormat(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "unknown"
	}
}

// isFormatAllowed checks if the format is in the allowed list
func (p *ImageProcessor) isFormatAllowed(format string) bool {
	for _, allowed := range p.config.AllowedFormats {
		if strings.EqualFold(format, allowed) {
			return true
		}
	}
	return false
}

// detectEXIF performs basic EXIF detection (simplified implementation)
func (p *ImageProcessor) detectEXIF(data []byte) bool {
	// Look for EXIF header in JPEG files
	if len(data) > 10 {
		// Check for JPEG magic number and EXIF marker
		if data[0] == 0xFF && data[1] == 0xD8 {
			// Look for EXIF marker (0xFF 0xE1 followed by "Exif")
			for i := 2; i < len(data)-4; i++ {
				if data[i] == 0xFF && data[i+1] == 0xE1 {
					if i+8 < len(data) && string(data[i+4:i+8]) == "Exif" {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetProcessorConfig returns the current processor configuration
func (p *ImageProcessor) GetProcessorConfig() *ProcessorConfig {
	return p.config
}

// UpdateConfig updates the processor configuration
func (p *ImageProcessor) UpdateConfig(config *ProcessorConfig) {
	p.config = config
}