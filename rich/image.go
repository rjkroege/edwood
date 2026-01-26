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

// ConvertToPlan9 converts a Go image.Image to Plan 9 RGBA32 pixel data.
// The returned byte slice contains pixels in row-major order, with each
// pixel being 4 bytes: R, G, B, A (pre-multiplied alpha).
//
// Plan 9's draw model uses pre-multiplied alpha, meaning RGB values are
// multiplied by the alpha value. For example, a 50% transparent red
// (255, 0, 0, 128) becomes (128, 0, 0, 128) in pre-multiplied form.
//
// Fully transparent pixels (alpha=0) have all components set to 0.
func ConvertToPlan9(img image.Image) ([]byte, error) {
	if img == nil {
		return nil, fmt.Errorf("cannot convert nil image")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Handle empty images
	if width == 0 || height == 0 {
		return []byte{}, nil
	}

	// Allocate buffer for RGBA32 pixel data (4 bytes per pixel)
	data := make([]byte, width*height*4)

	i := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get color at this position
			// RGBA() returns 16-bit values (0-65535)
			r32, g32, b32, a32 := img.At(x, y).RGBA()

			// Convert to 8-bit
			a := uint8(a32 >> 8)

			// Pre-multiply alpha
			// For fully transparent pixels, all values are 0
			if a == 0 {
				data[i] = 0
				data[i+1] = 0
				data[i+2] = 0
				data[i+3] = 0
			} else {
				// Pre-multiply RGB by alpha
				// Note: RGBA() already returns pre-multiplied values for most image types
				// but we convert from 16-bit to 8-bit here
				data[i] = uint8(r32 >> 8)
				data[i+1] = uint8(g32 >> 8)
				data[i+2] = uint8(b32 >> 8)
				data[i+3] = a
			}
			i += 4
		}
	}

	return data, nil
}
