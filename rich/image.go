// Package rich provides image loading and management for Markdeep rendering.
package rich

import (
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"os"
)

// Image size limits to prevent memory exhaustion.
const (
	MaxImageWidth  = 4096              // Maximum width in pixels
	MaxImageHeight = 4096              // Maximum height in pixels
	MaxImageBytes  = 16 * 1024 * 1024  // 16MB uncompressed (RGBA at 4 bytes/pixel)
)

// LoadImage loads an image from a file path.
// Supports PNG, JPEG, and GIF (first frame only for GIF).
// Returns the decoded image or an error if the file cannot be read,
// the format is not supported, or the image exceeds size limits.
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer f.Close()

	// image.Decode auto-detects format from registered decoders
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Validate image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width > MaxImageWidth || height > MaxImageHeight {
		return nil, fmt.Errorf("image too large: %dx%d (max %dx%d)",
			width, height, MaxImageWidth, MaxImageHeight)
	}

	// Check uncompressed size (assuming RGBA at 4 bytes per pixel)
	uncompressedSize := width * height * 4
	if uncompressedSize > MaxImageBytes {
		return nil, fmt.Errorf("image uncompressed size exceeds limit: %d bytes (max %d bytes)",
			uncompressedSize, MaxImageBytes)
	}

	return img, nil
}
