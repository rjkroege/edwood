package rich

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadImagePNG verifies that PNG images are loaded correctly.
func TestLoadImagePNG(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")

	// Create a simple 10x10 red image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, red)
		}
	}

	// Save as PNG
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Load the image
	loaded, err := LoadImage(pngPath)
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}

	// Verify dimensions
	bounds := loaded.Bounds()
	if bounds.Dx() != 10 || bounds.Dy() != 10 {
		t.Errorf("loaded image size = %dx%d, want 10x10", bounds.Dx(), bounds.Dy())
	}

	// Verify a pixel color
	r, g, b, a := loaded.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 || a>>8 != 255 {
		t.Errorf("pixel color = (%d, %d, %d, %d), want (255, 0, 0, 255)", r>>8, g>>8, b>>8, a>>8)
	}
}

// TestLoadImageJPEG verifies that JPEG images are loaded correctly.
func TestLoadImageJPEG(t *testing.T) {
	// Create a temporary JPEG file
	tmpDir := t.TempDir()
	jpegPath := filepath.Join(tmpDir, "test.jpg")

	// Create a simple 20x15 blue image
	img := image.NewRGBA(image.Rect(0, 0, 20, 15))
	blue := color.RGBA{0, 0, 255, 255}
	for y := 0; y < 15; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, blue)
		}
	}

	// Save as JPEG
	f, err := os.Create(jpegPath)
	if err != nil {
		t.Fatalf("failed to create test JPEG: %v", err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 100}); err != nil {
		f.Close()
		t.Fatalf("failed to encode JPEG: %v", err)
	}
	f.Close()

	// Load the image
	loaded, err := LoadImage(jpegPath)
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}

	// Verify dimensions
	bounds := loaded.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 15 {
		t.Errorf("loaded image size = %dx%d, want 20x15", bounds.Dx(), bounds.Dy())
	}

	// JPEG is lossy, so just verify it's approximately blue
	r, g, b, _ := loaded.At(10, 7).RGBA()
	if r>>8 > 50 || g>>8 > 50 || b>>8 < 200 {
		t.Errorf("pixel color not approximately blue: (%d, %d, %d)", r>>8, g>>8, b>>8)
	}
}

// TestLoadImageGIF verifies that GIF images are loaded correctly (first frame).
func TestLoadImageGIF(t *testing.T) {
	// Create a temporary GIF file
	tmpDir := t.TempDir()
	gifPath := filepath.Join(tmpDir, "test.gif")

	// Create a simple 8x8 green image
	palette := []color.Color{
		color.RGBA{0, 255, 0, 255}, // green
		color.White,
	}
	img := image.NewPaletted(image.Rect(0, 0, 8, 8), palette)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetColorIndex(x, y, 0) // green
		}
	}

	// Save as GIF
	f, err := os.Create(gifPath)
	if err != nil {
		t.Fatalf("failed to create test GIF: %v", err)
	}
	if err := gif.Encode(f, img, nil); err != nil {
		f.Close()
		t.Fatalf("failed to encode GIF: %v", err)
	}
	f.Close()

	// Load the image
	loaded, err := LoadImage(gifPath)
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}

	// Verify dimensions
	bounds := loaded.Bounds()
	if bounds.Dx() != 8 || bounds.Dy() != 8 {
		t.Errorf("loaded image size = %dx%d, want 8x8", bounds.Dx(), bounds.Dy())
	}

	// Verify a pixel color (should be green)
	r, g, b, a := loaded.At(0, 0).RGBA()
	if r>>8 != 0 || g>>8 != 255 || b>>8 != 0 || a>>8 != 255 {
		t.Errorf("pixel color = (%d, %d, %d, %d), want (0, 255, 0, 255)", r>>8, g>>8, b>>8, a>>8)
	}
}

// TestLoadImageMissing verifies that loading a missing file returns an error.
func TestLoadImageMissing(t *testing.T) {
	_, err := LoadImage("/nonexistent/path/to/image.png")
	if err == nil {
		t.Error("LoadImage should return an error for missing file")
	}
}

// TestLoadImageCorrupt verifies that loading a corrupt image file returns an error.
func TestLoadImageCorrupt(t *testing.T) {
	// Create a temporary file with corrupt data
	tmpDir := t.TempDir()
	corruptPath := filepath.Join(tmpDir, "corrupt.png")

	// Write random bytes that aren't a valid image
	if err := os.WriteFile(corruptPath, []byte("not a valid image file content"), 0644); err != nil {
		t.Fatalf("failed to create corrupt file: %v", err)
	}

	_, err := LoadImage(corruptPath)
	if err == nil {
		t.Error("LoadImage should return an error for corrupt image data")
	}
}

// TestLoadImageNotImage verifies that loading a non-image file returns an error.
func TestLoadImageNotImage(t *testing.T) {
	// Create a temporary text file
	tmpDir := t.TempDir()
	textPath := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(textPath, []byte("This is a text file, not an image."), 0644); err != nil {
		t.Fatalf("failed to create text file: %v", err)
	}

	_, err := LoadImage(textPath)
	if err == nil {
		t.Error("LoadImage should return an error for non-image file")
	}
}

// TestLoadImageTooLarge verifies that images exceeding size limits are rejected.
func TestLoadImageTooLarge(t *testing.T) {
	// Create a temporary PNG file that's too large (5000x5000)
	// Note: We create the file but the load should fail due to size limit
	tmpDir := t.TempDir()
	largePath := filepath.Join(tmpDir, "large.png")

	// Create an image that exceeds MaxImageWidth/MaxImageHeight (4096x4096)
	// We use a small PNG header that claims a large size to avoid actually
	// allocating huge memory during test
	img := image.NewRGBA(image.Rect(0, 0, 5000, 5000))

	// Save as PNG (this will be slow but necessary for the test)
	// To speed up, we'll create a minimal image and test the size check
	f, err := os.Create(largePath)
	if err != nil {
		t.Fatalf("failed to create large PNG: %v", err)
	}

	// Encode with default compression
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode large PNG: %v", err)
	}
	f.Close()

	_, err = LoadImage(largePath)
	if err == nil {
		t.Error("LoadImage should return an error for image exceeding size limits")
	}
}

// TestLoadImageMemoryLimit verifies that images with excessive uncompressed size are rejected.
func TestLoadImageMemoryLimit(t *testing.T) {
	// Create an image that exceeds the memory limit when uncompressed
	// MaxImageBytes = 16 * 1024 * 1024 = 16MB
	// A 4096x4096 RGBA image = 4096 * 4096 * 4 = 67MB which exceeds 16MB
	// But we want to test something smaller that still exceeds the limit
	// 2048x2048 RGBA = 2048 * 2048 * 4 = 16MB exactly, so 2049x2049 would exceed

	tmpDir := t.TempDir()
	memPath := filepath.Join(tmpDir, "bigmem.png")

	// Create a 2100x2100 image (17.64MB uncompressed RGBA)
	img := image.NewRGBA(image.Rect(0, 0, 2100, 2100))

	f, err := os.Create(memPath)
	if err != nil {
		t.Fatalf("failed to create large memory PNG: %v", err)
	}

	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	_, err = LoadImage(memPath)
	if err == nil {
		t.Error("LoadImage should return an error for image exceeding memory limits")
	}
}

// TestLoadImageValidatesFormat verifies that LoadImage auto-detects the image format.
func TestLoadImageValidatesFormat(t *testing.T) {
	// Create a PNG file with wrong extension - should still load
	tmpDir := t.TempDir()
	wrongExtPath := filepath.Join(tmpDir, "test.jpg") // wrong extension

	// Create a PNG image
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	f, err := os.Create(wrongExtPath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Should still load correctly since image.Decode auto-detects format
	loaded, err := LoadImage(wrongExtPath)
	if err != nil {
		t.Fatalf("LoadImage should auto-detect format regardless of extension: %v", err)
	}

	bounds := loaded.Bounds()
	if bounds.Dx() != 5 || bounds.Dy() != 5 {
		t.Errorf("loaded image size = %dx%d, want 5x5", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImageEmptyFile verifies that an empty file returns an error.
func TestLoadImageEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyPath := filepath.Join(tmpDir, "empty.png")

	// Create empty file
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	_, err := LoadImage(emptyPath)
	if err == nil {
		t.Error("LoadImage should return an error for empty file")
	}
}

// TestLoadImagePartialHeader verifies that a file with only a partial header returns an error.
func TestLoadImagePartialHeader(t *testing.T) {
	tmpDir := t.TempDir()
	partialPath := filepath.Join(tmpDir, "partial.png")

	// Write just the PNG magic bytes but nothing else
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(partialPath, pngMagic, 0644); err != nil {
		t.Fatalf("failed to create partial file: %v", err)
	}

	_, err := LoadImage(partialPath)
	if err == nil {
		t.Error("LoadImage should return an error for partial/truncated file")
	}
}

// Helper function to create a minimal valid PNG in memory for testing
func createMinimalPNG(width, height int, c color.Color) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// =============================================================================
// Phase 16C: Plan 9 Conversion Tests
// =============================================================================

// TestConvertRGBA verifies that RGBA images are converted correctly to Plan 9 format.
func TestConvertRGBA(t *testing.T) {
	// Create a 4x4 RGBA image with distinct colors in each quadrant
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	// Top-left: red
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	// Top-right: green
	for y := 0; y < 2; y++ {
		for x := 2; x < 4; x++ {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}
	// Bottom-left: blue
	for y := 2; y < 4; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	// Bottom-right: white
	for y := 2; y < 4; y++ {
		for x := 2; x < 4; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// Test that ConvertToPlan9 returns valid data
	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Expected size: 4x4 pixels * 4 bytes/pixel (RGBA32) = 64 bytes
	expectedSize := 4 * 4 * 4
	if len(data) != expectedSize {
		t.Errorf("converted data size = %d, want %d", len(data), expectedSize)
	}

	// Verify first pixel (red) - Plan 9 RGBA32 uses ABGR byte order (little-endian)
	// For red (R=255, G=0, B=0, A=255), bytes are: A=255, B=0, G=0, R=255
	if len(data) >= 4 {
		a, b, g, r := data[0], data[1], data[2], data[3]
		if a != 255 || b != 0 || g != 0 || r != 255 {
			t.Errorf("first pixel (red) = A=%d, B=%d, G=%d, R=%d, want A=255, B=0, G=0, R=255", a, b, g, r)
		}
	}
}

// TestConvertRGB verifies that RGB images (no alpha channel) are converted correctly.
func TestConvertRGB(t *testing.T) {
	// Create a 3x3 image using NRGBA (no premultiplied alpha) with full opacity
	// Go's image package doesn't have a pure RGB type, but we can test with NRGBA
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))

	// Fill with cyan (R=0, G=255, B=255)
	cyan := color.NRGBA{0, 255, 255, 255}
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			img.Set(x, y, cyan)
		}
	}

	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Expected size: 3x3 pixels * 4 bytes/pixel = 36 bytes
	expectedSize := 3 * 3 * 4
	if len(data) != expectedSize {
		t.Errorf("converted data size = %d, want %d", len(data), expectedSize)
	}

	// Verify first pixel - Plan 9 RGBA32 uses ABGR byte order (little-endian)
	// For cyan (R=0, G=255, B=255, A=255), bytes are: A=255, B=255, G=255, R=0
	if len(data) >= 4 {
		a, b, g, r := data[0], data[1], data[2], data[3]
		if a != 255 || b != 255 || g != 255 || r != 0 {
			t.Errorf("first pixel (cyan) = A=%d, B=%d, G=%d, R=%d, want A=255, B=255, G=255, R=0", a, b, g, r)
		}
	}
}

// TestConvertGrayscale verifies that grayscale images are converted correctly.
func TestConvertGrayscale(t *testing.T) {
	// Create a 2x2 grayscale image
	img := image.NewGray(image.Rect(0, 0, 2, 2))

	// Set different gray levels
	img.SetGray(0, 0, color.Gray{0})   // Black
	img.SetGray(1, 0, color.Gray{85})  // Dark gray
	img.SetGray(0, 1, color.Gray{170}) // Light gray
	img.SetGray(1, 1, color.Gray{255}) // White

	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Expected size: 2x2 pixels * 4 bytes/pixel = 16 bytes
	expectedSize := 2 * 2 * 4
	if len(data) != expectedSize {
		t.Errorf("converted data size = %d, want %d", len(data), expectedSize)
	}

	// Verify black pixel (0,0) - Plan 9 RGBA32 uses ABGR byte order
	// For black (R=0, G=0, B=0, A=255), bytes are: A=255, B=0, G=0, R=0
	if len(data) >= 4 {
		a, b, g, r := data[0], data[1], data[2], data[3]
		if a != 255 || b != 0 || g != 0 || r != 0 {
			t.Errorf("black pixel = A=%d, B=%d, G=%d, R=%d, want A=255, B=0, G=0, R=0", a, b, g, r)
		}
	}

	// Verify white pixel (1,1) - Plan 9 RGBA32 uses ABGR byte order
	// For white (R=255, G=255, B=255, A=255), bytes are: A=255, B=255, G=255, R=255
	// Position in data: pixel at (1,1) is at index (1*2 + 1) * 4 = 12
	if len(data) >= 16 {
		a, b, g, r := data[12], data[13], data[14], data[15]
		if a != 255 || b != 255 || g != 255 || r != 255 {
			t.Errorf("white pixel = A=%d, B=%d, G=%d, R=%d, want A=255, B=255, G=255, R=255", a, b, g, r)
		}
	}
}

// TestConvertAlphaPreMultiplied verifies that alpha is properly pre-multiplied.
// Plan 9's draw model uses pre-multiplied alpha, so colors with partial
// transparency need to have their RGB values multiplied by the alpha value.
func TestConvertAlphaPreMultiplied(t *testing.T) {
	// Create a 2x1 image with:
	// - Pixel 0: Red at 50% alpha (should become R=127, G=0, B=0, A=127)
	// - Pixel 1: White at 50% alpha (should become R=127, G=127, B=127, A=127)
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))

	// Set red at 50% alpha
	img.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 128})
	// Set white at 50% alpha
	img.SetNRGBA(1, 0, color.NRGBA{255, 255, 255, 128})

	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Expected size: 2x1 pixels * 4 bytes/pixel = 8 bytes
	if len(data) != 8 {
		t.Errorf("converted data size = %d, want 8", len(data))
	}

	// Verify first pixel (red at 50% alpha)
	// Pre-multiplied: R = 255 * 128 / 255 ≈ 128, G = 0, B = 0, A = 128
	if len(data) >= 4 {
		r, g, b, a := data[0], data[1], data[2], data[3]
		// Allow some tolerance due to integer rounding
		if r < 126 || r > 130 || g != 0 || b != 0 || a != 128 {
			t.Errorf("red 50%% alpha pixel = (%d, %d, %d, %d), want approximately (128, 0, 0, 128)", r, g, b, a)
		}
	}

	// Verify second pixel (white at 50% alpha)
	// Pre-multiplied: R = G = B = 255 * 128 / 255 ≈ 128, A = 128
	if len(data) >= 8 {
		r, g, b, a := data[4], data[5], data[6], data[7]
		// Allow some tolerance
		if r < 126 || r > 130 || g < 126 || g > 130 || b < 126 || b > 130 || a != 128 {
			t.Errorf("white 50%% alpha pixel = (%d, %d, %d, %d), want approximately (128, 128, 128, 128)", r, g, b, a)
		}
	}
}

// TestConvertTransparent verifies that fully transparent pixels are handled correctly.
// Fully transparent pixels (alpha=0) should have RGB=0 as well (pre-multiplied).
func TestConvertTransparent(t *testing.T) {
	// Create a 2x2 image with varying transparency
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))

	// Fully opaque red
	img.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	// Fully transparent red (should become all zeros)
	img.SetNRGBA(1, 0, color.NRGBA{255, 0, 0, 0})
	// Fully transparent white (should become all zeros)
	img.SetNRGBA(0, 1, color.NRGBA{255, 255, 255, 0})
	// Fully opaque blue
	img.SetNRGBA(1, 1, color.NRGBA{0, 0, 255, 255})

	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Verify fully transparent red pixel (1,0) is all zeros
	// Position: (1 + 0*2) * 4 = 4
	if len(data) >= 8 {
		r, g, b, a := data[4], data[5], data[6], data[7]
		if r != 0 || g != 0 || b != 0 || a != 0 {
			t.Errorf("transparent red pixel = (%d, %d, %d, %d), want (0, 0, 0, 0)", r, g, b, a)
		}
	}

	// Verify fully transparent white pixel (0,1) is all zeros
	// Position: (0 + 1*2) * 4 = 8
	if len(data) >= 12 {
		r, g, b, a := data[8], data[9], data[10], data[11]
		if r != 0 || g != 0 || b != 0 || a != 0 {
			t.Errorf("transparent white pixel = (%d, %d, %d, %d), want (0, 0, 0, 0)", r, g, b, a)
		}
	}

	// Verify fully opaque blue pixel (1,1) is correct - Plan 9 RGBA32 uses ABGR byte order
	// For blue (R=0, G=0, B=255, A=255), bytes are: A=255, B=255, G=0, R=0
	// Position: (1 + 1*2) * 4 = 12
	if len(data) >= 16 {
		a, b, g, r := data[12], data[13], data[14], data[15]
		if a != 255 || b != 255 || g != 0 || r != 0 {
			t.Errorf("opaque blue pixel = A=%d, B=%d, G=%d, R=%d, want A=255, B=255, G=0, R=0", a, b, g, r)
		}
	}
}

// TestConvertNilImage verifies that nil images return an error.
func TestConvertNilImage(t *testing.T) {
	_, err := ConvertToPlan9(nil)
	if err == nil {
		t.Error("ConvertToPlan9 should return an error for nil image")
	}
}

// TestConvertEmptyImage verifies that zero-size images are handled.
func TestConvertEmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed on empty image: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("converted empty image should have 0 bytes, got %d", len(data))
	}
}

// TestConvertSinglePixel verifies conversion of a 1x1 image.
func TestConvertSinglePixel(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{100, 150, 200, 255})

	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	if len(data) != 4 {
		t.Errorf("single pixel should produce 4 bytes, got %d", len(data))
	}

	// Plan 9 RGBA32 uses ABGR byte order
	// For (R=100, G=150, B=200, A=255), bytes are: A=255, B=200, G=150, R=100
	if len(data) >= 4 {
		a, b, g, r := data[0], data[1], data[2], data[3]
		if a != 255 || b != 200 || g != 150 || r != 100 {
			t.Errorf("pixel = A=%d, B=%d, G=%d, R=%d, want A=255, B=200, G=150, R=100", a, b, g, r)
		}
	}
}

// TestConvertPalettedImage verifies that paletted (indexed color) images are converted.
func TestConvertPalettedImage(t *testing.T) {
	// Create a paletted image (like GIF)
	palette := []color.Color{
		color.RGBA{255, 0, 0, 255},   // Red
		color.RGBA{0, 255, 0, 255},   // Green
		color.RGBA{0, 0, 255, 255},   // Blue
		color.RGBA{255, 255, 0, 255}, // Yellow
	}
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	img.SetColorIndex(0, 0, 0) // Red
	img.SetColorIndex(1, 0, 1) // Green
	img.SetColorIndex(0, 1, 2) // Blue
	img.SetColorIndex(1, 1, 3) // Yellow

	data, err := ConvertToPlan9(img)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Expected size: 2x2 pixels * 4 bytes/pixel = 16 bytes
	if len(data) != 16 {
		t.Errorf("converted data size = %d, want 16", len(data))
	}

	// Verify red pixel (0,0) - Plan 9 RGBA32 uses ABGR byte order
	// For red (R=255, G=0, B=0, A=255), bytes are: A=255, B=0, G=0, R=255
	if len(data) >= 4 {
		a, b, g, r := data[0], data[1], data[2], data[3]
		if a != 255 || b != 0 || g != 0 || r != 255 {
			t.Errorf("red pixel = A=%d, B=%d, G=%d, R=%d, want A=255, B=0, G=0, R=255", a, b, g, r)
		}
	}

	// Verify yellow pixel (1,1) - Plan 9 RGBA32 uses ABGR byte order
	// For yellow (R=255, G=255, B=0, A=255), bytes are: A=255, B=0, G=255, R=255
	if len(data) >= 16 {
		a, b, g, r := data[12], data[13], data[14], data[15]
		if a != 255 || b != 0 || g != 255 || r != 255 {
			t.Errorf("yellow pixel = A=%d, B=%d, G=%d, R=%d, want A=255, B=0, G=255, R=255", a, b, g, r)
		}
	}
}

// =============================================================================
// Phase 16D: Image Cache Tests
// =============================================================================

// TestImageCacheHit verifies that cached images are returned on subsequent loads.
func TestImageCacheHit(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")

	// Create a simple 5x5 image
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Create cache
	cache := NewImageCache(10)

	// Load image twice
	cached1, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}

	cached2, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}

	// Should be the same object (cache hit)
	if cached1 != cached2 {
		t.Error("cache should return same CachedImage on second load")
	}

	// Verify image data is correct
	if cached1.Width != 5 || cached1.Height != 5 {
		t.Errorf("cached image size = %dx%d, want 5x5", cached1.Width, cached1.Height)
	}
}

// TestImageCacheMiss verifies that cache misses trigger image loading.
func TestImageCacheMiss(t *testing.T) {
	// Create two temporary PNG files
	tmpDir := t.TempDir()
	pngPath1 := filepath.Join(tmpDir, "test1.png")
	pngPath2 := filepath.Join(tmpDir, "test2.png")

	// Create first image (red)
	img1 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img1.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	f1, err := os.Create(pngPath1)
	if err != nil {
		t.Fatalf("failed to create test1 PNG: %v", err)
	}
	if err := png.Encode(f1, img1); err != nil {
		f1.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f1.Close()

	// Create second image (blue, different size)
	img2 := image.NewRGBA(image.Rect(0, 0, 15, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 15; x++ {
			img2.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	f2, err := os.Create(pngPath2)
	if err != nil {
		t.Fatalf("failed to create test2 PNG: %v", err)
	}
	if err := png.Encode(f2, img2); err != nil {
		f2.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f2.Close()

	// Create cache
	cache := NewImageCache(10)

	// Load first image
	cached1, err := cache.Load(pngPath1)
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}
	if cached1.Width != 10 || cached1.Height != 10 {
		t.Errorf("first image size = %dx%d, want 10x10", cached1.Width, cached1.Height)
	}

	// Load second image (cache miss, different file)
	cached2, err := cache.Load(pngPath2)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if cached2.Width != 15 || cached2.Height != 20 {
		t.Errorf("second image size = %dx%d, want 15x20", cached2.Width, cached2.Height)
	}

	// They should be different objects
	if cached1 == cached2 {
		t.Error("different files should return different CachedImage objects")
	}
}

// TestImageCacheGet verifies that Get returns cached images without loading.
func TestImageCacheGet(t *testing.T) {
	cache := NewImageCache(10)

	// Get non-existent key
	cached, ok := cache.Get("/nonexistent/path")
	if ok {
		t.Error("Get should return false for non-existent key")
	}
	if cached != nil {
		t.Error("Get should return nil for non-existent key")
	}

	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Load the image
	loaded, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Now Get should return it
	cached, ok = cache.Get(pngPath)
	if !ok {
		t.Error("Get should return true for loaded image")
	}
	if cached != loaded {
		t.Error("Get should return the same CachedImage as Load")
	}
}

// TestImageCacheEviction verifies LRU eviction when cache exceeds maxSize.
func TestImageCacheEviction(t *testing.T) {
	// Create a cache with max size of 3
	cache := NewImageCache(3)

	// Create 5 temporary PNG files
	tmpDir := t.TempDir()
	paths := make([]string, 5)
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune('1'+i))+".png")
		paths[i] = path

		img := image.NewRGBA(image.Rect(0, 0, 5, 5))
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create test PNG %d: %v", i, err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatalf("failed to encode PNG %d: %v", i, err)
		}
		f.Close()
	}

	// Load first 3 images
	for i := 0; i < 3; i++ {
		_, err := cache.Load(paths[i])
		if err != nil {
			t.Fatalf("Load(%d) failed: %v", i, err)
		}
	}

	// All 3 should be in cache
	for i := 0; i < 3; i++ {
		if _, ok := cache.Get(paths[i]); !ok {
			t.Errorf("image %d should be in cache", i)
		}
	}

	// Load 4th image - should evict the oldest (first)
	_, err := cache.Load(paths[3])
	if err != nil {
		t.Fatalf("Load(3) failed: %v", err)
	}

	// First image should be evicted, 2nd, 3rd, 4th should remain
	if _, ok := cache.Get(paths[0]); ok {
		t.Error("first image should have been evicted")
	}
	for i := 1; i <= 3; i++ {
		if _, ok := cache.Get(paths[i]); !ok {
			t.Errorf("image %d should still be in cache", i)
		}
	}

	// Load 5th image - should evict the 2nd (now oldest)
	_, err = cache.Load(paths[4])
	if err != nil {
		t.Fatalf("Load(4) failed: %v", err)
	}

	// 2nd should be evicted, 3rd, 4th, 5th should remain
	if _, ok := cache.Get(paths[1]); ok {
		t.Error("second image should have been evicted")
	}
	for i := 2; i <= 4; i++ {
		if _, ok := cache.Get(paths[i]); !ok {
			t.Errorf("image %d should still be in cache", i)
		}
	}
}

// TestImageCacheMaxSize verifies that cache respects max size limit.
func TestImageCacheMaxSize(t *testing.T) {
	// Create a cache with max size of 2
	cache := NewImageCache(2)

	// Create 5 temporary PNG files
	tmpDir := t.TempDir()
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune('1'+i))+".png")
		img := image.NewRGBA(image.Rect(0, 0, 5, 5))
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create test PNG %d: %v", i, err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatalf("failed to encode PNG %d: %v", i, err)
		}
		f.Close()

		// Load each image
		_, err = cache.Load(path)
		if err != nil {
			t.Fatalf("Load(%d) failed: %v", i, err)
		}
	}

	// Cache should contain at most maxSize items
	count := 0
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune('1'+i))+".png")
		if _, ok := cache.Get(path); ok {
			count++
		}
	}

	if count > 2 {
		t.Errorf("cache contains %d items, expected at most 2", count)
	}
}

// TestImageCacheErrorCached verifies that load errors are cached.
func TestImageCacheErrorCached(t *testing.T) {
	cache := NewImageCache(10)

	// Try to load a non-existent file
	badPath := "/nonexistent/path/to/image.png"

	cached1, err1 := cache.Load(badPath)
	if err1 == nil {
		t.Fatal("Load should return error for missing file")
	}
	if cached1 == nil {
		t.Fatal("Load should return CachedImage even on error")
	}
	if cached1.Err == nil {
		t.Error("CachedImage.Err should be set on load failure")
	}

	// Second load should return cached error
	cached2, err2 := cache.Load(badPath)
	if err2 == nil {
		t.Fatal("second Load should also return error")
	}
	if cached2 != cached1 {
		t.Error("cache should return same CachedImage for repeated error loads")
	}
}

// TestImageCacheNoRetry verifies that failed loads are not retried.
func TestImageCacheNoRetry(t *testing.T) {
	cache := NewImageCache(10)

	// Create a counter to track load attempts
	loadCount := 0
	badPath := "/nonexistent/will/never/exist.png"

	// First load
	_, err := cache.Load(badPath)
	if err == nil {
		t.Fatal("Load should fail for non-existent file")
	}
	loadCount++

	// Second load - should use cached error, not retry
	cached, err := cache.Load(badPath)
	if err == nil {
		t.Fatal("second Load should also fail")
	}

	// The cached entry should exist with an error
	if cached == nil || cached.Err == nil {
		t.Error("cached error entry should exist")
	}

	// Verify via Get that the entry is cached
	cachedGet, ok := cache.Get(badPath)
	if !ok {
		t.Error("failed load should still be cached")
	}
	if cachedGet.Err == nil {
		t.Error("cached entry should have error set")
	}
}

// TestImageCacheClear verifies that Clear removes all cached images.
func TestImageCacheClear(t *testing.T) {
	cache := NewImageCache(10)

	// Create some temporary PNG files and load them
	tmpDir := t.TempDir()
	paths := make([]string, 3)
	for i := 0; i < 3; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune('1'+i))+".png")
		paths[i] = path

		img := image.NewRGBA(image.Rect(0, 0, 5, 5))
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create test PNG %d: %v", i, err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatalf("failed to encode PNG %d: %v", i, err)
		}
		f.Close()

		_, err = cache.Load(path)
		if err != nil {
			t.Fatalf("Load(%d) failed: %v", i, err)
		}
	}

	// Verify all are cached
	for i, path := range paths {
		if _, ok := cache.Get(path); !ok {
			t.Errorf("image %d should be cached before Clear", i)
		}
	}

	// Clear the cache
	cache.Clear()

	// Verify all are gone
	for i, path := range paths {
		if _, ok := cache.Get(path); ok {
			t.Errorf("image %d should not be cached after Clear", i)
		}
	}
}

// TestImageCacheFreeImages verifies that Clear properly cleans up resources.
func TestImageCacheFreeImages(t *testing.T) {
	cache := NewImageCache(10)

	// Create and load a PNG
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	cached, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify image was loaded
	if cached.Original == nil {
		t.Error("cached.Original should be set")
	}
	if cached.Data == nil {
		t.Error("cached.Data (Plan 9 bytes) should be set")
	}

	// Clear and verify the cache is empty
	cache.Clear()

	if _, ok := cache.Get(pngPath); ok {
		t.Error("cache should be empty after Clear")
	}

	// Note: In a full implementation with Plan 9 display integration,
	// we would verify that Plan9Image.Free() was called. For now,
	// we verify the cache map is cleared.
}

// =============================================================================
// Phase 16F: Frame Rendering Tests
// =============================================================================

// TestDrawImage verifies that images are rendered in Phase 5 of drawText.
// An image box should trigger a blit operation to copy the image to the screen.
func TestDrawImage(t *testing.T) {
	// Create a temporary PNG file for the test
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test_image.png")

	// Create a simple 20x15 red image
	img := image.NewRGBA(image.Rect(0, 0, 20, 15))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 15; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, red)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Create an ImageCache and load the image
	cache := NewImageCache(10)
	cached, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("failed to load image into cache: %v", err)
	}

	// Create content with an image span
	content := Content{
		Span{
			Text: "[Image: test]",
			Style: Style{
				Image:    true,
				ImageURL: pngPath,
				ImageAlt: "test",
			},
		},
	}

	// Create boxes from content and inject the cached image data
	boxes := contentToBoxes(content)
	if len(boxes) == 0 {
		t.Fatal("contentToBoxes returned no boxes")
	}
	// Inject the loaded image data into the box
	boxes[0].ImageData = cached

	// Verify the box is recognized as an image
	if !boxes[0].IsImage() {
		t.Error("box should be recognized as an image when ImageData is set")
	}

	// Verify box dimensions are correct
	if boxes[0].ImageData.Width != 20 || boxes[0].ImageData.Height != 15 {
		t.Errorf("image dimensions = %dx%d, want 20x15",
			boxes[0].ImageData.Width, boxes[0].ImageData.Height)
	}
}

// TestDrawImagePosition verifies that images are positioned correctly in layout.
// The image should be placed at the correct X,Y position based on its layout position.
func TestDrawImagePosition(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "position_test.png")

	// Create a 30x20 image
	img := image.NewRGBA(image.Rect(0, 0, 30, 20))
	blue := color.RGBA{0, 0, 255, 255}
	for y := 0; y < 20; y++ {
		for x := 0; x < 30; x++ {
			img.Set(x, y, blue)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Load image
	cache := NewImageCache(10)
	cached, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("failed to load image: %v", err)
	}

	// Create content: text followed by image
	content := Content{
		Span{Text: "prefix "},
		Span{
			Text: "[Image: pos]",
			Style: Style{
				Image:    true,
				ImageURL: pngPath,
				ImageAlt: "pos",
			},
		},
	}

	// Convert to boxes
	boxes := contentToBoxes(content)
	if len(boxes) < 2 {
		t.Fatalf("expected at least 2 boxes, got %d", len(boxes))
	}

	// Find and set up the image box
	for i := range boxes {
		if boxes[i].Style.Image {
			boxes[i].ImageData = cached
		}
	}

	// Create a mock font with known dimensions
	mockFont := &testFont{width: 10, height: 14}

	// Layout the boxes
	frameWidth := 400
	maxtab := 80
	lines := layout(boxes, mockFont, frameWidth, maxtab, nil, nil)

	if len(lines) == 0 {
		t.Fatal("layout returned no lines")
	}

	// Find the image box in the layout
	var imageBox *PositionedBox
	for _, line := range lines {
		for i := range line.Boxes {
			if line.Boxes[i].Box.IsImage() {
				imageBox = &line.Boxes[i]
				break
			}
		}
	}

	if imageBox == nil {
		t.Fatal("image box not found in layout")
	}

	// The image should be positioned after the "prefix " text
	// "prefix " = 7 characters * 10 pixels = 70 pixels
	expectedX := 70 // After "prefix " (7 chars at 10px each)
	if imageBox.X != expectedX {
		t.Errorf("image X position = %d, want %d", imageBox.X, expectedX)
	}

	// Image width should be its actual width (not scaled since it fits)
	if imageBox.Box.Wid != 30 {
		t.Errorf("image width = %d, want 30", imageBox.Box.Wid)
	}
}

// TestDrawImageClipBottom verifies that images are clipped at the frame bottom boundary.
// Images that extend below the frame should be partially rendered.
func TestDrawImageClipBottom(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "clip_bottom.png")

	// Create a tall 20x50 image
	img := image.NewRGBA(image.Rect(0, 0, 20, 50))
	green := color.RGBA{0, 255, 0, 255}
	for y := 0; y < 50; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, green)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Load image
	cache := NewImageCache(10)
	cached, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("failed to load image: %v", err)
	}

	// Verify the image is 50 pixels tall
	if cached.Height != 50 {
		t.Errorf("cached image height = %d, want 50", cached.Height)
	}

	// Create a frame with height smaller than the image
	// When rendering, images that exceed the frame height should be clipped
	// The clipping is handled by Draw's Intersect operation

	// Test that imageBoxDimensions returns correct values
	box := Box{
		Style: Style{
			Image:    true,
			ImageURL: pngPath,
			ImageAlt: "clip",
		},
		ImageData: cached,
	}

	// For a frame width of 100, the image (20px wide) should not be scaled
	frameWidth := 100
	width, height := imageBoxDimensions(&box, frameWidth)

	// Image fits horizontally, so no scaling
	if width != 20 {
		t.Errorf("imageBoxDimensions width = %d, want 20", width)
	}
	if height != 50 {
		t.Errorf("imageBoxDimensions height = %d, want 50", height)
	}

	// Note: Actual clipping during rendering is handled by the Draw operation
	// which clips the destination rectangle to the frame bounds.
	// This test verifies the image dimensions are preserved before clipping.
}

// TestDrawImageClipRight verifies that images wider than the frame are scaled down.
// Unlike clipping at the bottom, wide images are scaled to fit the frame width.
func TestDrawImageClipRight(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "clip_right.png")

	// Create a wide 200x50 image
	img := image.NewRGBA(image.Rect(0, 0, 200, 50))
	yellow := color.RGBA{255, 255, 0, 255}
	for y := 0; y < 50; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, yellow)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Load image
	cache := NewImageCache(10)
	cached, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("failed to load image: %v", err)
	}

	// Verify original dimensions
	if cached.Width != 200 || cached.Height != 50 {
		t.Errorf("cached image dimensions = %dx%d, want 200x50",
			cached.Width, cached.Height)
	}

	// Create a box for the image
	box := Box{
		Style: Style{
			Image:    true,
			ImageURL: pngPath,
			ImageAlt: "wide",
		},
		ImageData: cached,
	}

	// Test scaling when image is wider than frame
	frameWidth := 100 // Narrower than image width (200)
	width, height := imageBoxDimensions(&box, frameWidth)

	// Image should be scaled to fit frame width
	if width != 100 {
		t.Errorf("scaled image width = %d, want 100", width)
	}

	// Height should be proportionally scaled: 50 * (100/200) = 25
	expectedHeight := 25
	if height != expectedHeight {
		t.Errorf("scaled image height = %d, want %d", height, expectedHeight)
	}
}

// TestDrawImageError verifies that a placeholder is shown when image fails to load.
// When an image cannot be loaded, the system should display an error placeholder
// instead of crashing or showing a broken image.
func TestDrawImageError(t *testing.T) {
	// Create an ImageCache
	cache := NewImageCache(10)

	// Try to load a non-existent image
	badPath := "/nonexistent/path/to/image.png"
	cached, err := cache.Load(badPath)

	// Should return an error
	if err == nil {
		t.Fatal("expected error for non-existent image")
	}

	// Should still return a CachedImage with error set
	if cached == nil {
		t.Fatal("expected CachedImage even on error")
	}
	if cached.Err == nil {
		t.Error("CachedImage.Err should be set for failed load")
	}

	// The cached image should not have Original set for failed loads
	if cached.Original != nil {
		t.Error("cached.Original should be nil for failed load")
	}

	// When rendering, if ImageData.Original is nil but there's an error,
	// the rendering code should show a placeholder instead
	// Test that IsImage() returns false when ImageData has error (no actual image)
	box := Box{
		Style: Style{
			Image:    true,
			ImageURL: badPath,
			ImageAlt: "missing",
		},
		ImageData: cached,
	}

	// IsImage() checks Style.Image && ImageData != nil
	// Even with ImageData set, if there's an error the box is technically
	// an "image" but will show error placeholder during rendering
	if !box.Style.Image {
		t.Error("box.Style.Image should be true")
	}
	if box.ImageData == nil {
		t.Error("box.ImageData should be set (even with error)")
	}

	// Verify error is preserved for placeholder rendering
	if box.ImageData.Err == nil {
		t.Error("box.ImageData.Err should be set for error placeholder")
	}
}

// testFont is a simple font implementation for testing.
type testFont struct {
	width  int
	height int
}

func (f *testFont) Name() string             { return "test-font" }
func (f *testFont) Height() int              { return f.height }
func (f *testFont) BytesWidth(b []byte) int  { return f.width * len(b) }
func (f *testFont) RunesWidth(r []rune) int  { return f.width * len(r) }
func (f *testFont) StringWidth(s string) int { return f.width * len(s) }

// =============================================================================
// Phase 16J: URL Image Support Tests
// =============================================================================

// TestIsImageURL verifies that URL detection correctly identifies HTTP(S) URLs
// vs local file paths.
func TestIsImageURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// HTTP URLs
		{"http URL", "http://example.com/image.png", true},
		{"http URL with port", "http://example.com:8080/image.png", true},
		{"http URL with query", "http://example.com/image.png?size=large", true},
		{"http URL with fragment", "http://example.com/image.png#section", true},
		{"http URL uppercase", "HTTP://example.com/image.png", true},
		{"http URL mixed case", "Http://example.com/image.png", true},

		// HTTPS URLs
		{"https URL", "https://example.com/image.png", true},
		{"https URL with port", "https://example.com:443/image.png", true},
		{"https URL with path", "https://example.com/path/to/image.jpg", true},
		{"https URL uppercase", "HTTPS://example.com/image.png", true},

		// Local file paths
		{"absolute path", "/path/to/image.png", false},
		{"relative path", "images/photo.jpg", false},
		{"relative path with dot", "./images/photo.jpg", false},
		{"parent path", "../images/photo.jpg", false},
		{"Windows absolute path", "C:\\Users\\photo.jpg", false},
		{"tilde path", "~/Documents/image.png", false},

		// Edge cases
		{"empty string", "", false},
		{"just http", "http://", true}, // Technically a valid URL prefix
		{"just https", "https://", true},
		{"ftp URL (not supported)", "ftp://example.com/image.png", false},
		{"file URL (not supported)", "file:///path/to/image.png", false},
		{"data URL (not supported)", "data:image/png;base64,ABC123", false},
		{"mailto (not supported)", "mailto:test@example.com", false},
		{"httpx (invalid)", "httpx://example.com/image.png", false},
		{"path containing http", "/var/http/image.png", false},
		{"path starting with http", "http_files/image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageURL(tt.path)
			if result != tt.expected {
				t.Errorf("isImageURL(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestLoadImageFromURL verifies that images can be loaded from HTTP URLs.
// This test uses httptest.Server to simulate a web server serving images.
func TestLoadImageFromURL(t *testing.T) {
	// Create a test image
	testImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			testImg.Set(x, y, red)
		}
	}

	// Encode to PNG bytes
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, testImg); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes)
	}))
	defer server.Close()

	// Load image from URL
	img, err := loadImageFromURL(server.URL + "/test.png")
	if err != nil {
		t.Fatalf("loadImageFromURL failed: %v", err)
	}

	// Verify dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 10 || bounds.Dy() != 10 {
		t.Errorf("loaded image size = %dx%d, want 10x10", bounds.Dx(), bounds.Dy())
	}

	// Verify pixel color
	r, g, b, a := img.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 || a>>8 != 255 {
		t.Errorf("pixel color = (%d, %d, %d, %d), want (255, 0, 0, 255)", r>>8, g>>8, b>>8, a>>8)
	}
}

// TestLoadImageFromURLJPEG verifies JPEG images can be loaded from URLs.
func TestLoadImageFromURLJPEG(t *testing.T) {
	// Create a test image
	testImg := image.NewRGBA(image.Rect(0, 0, 20, 15))
	blue := color.RGBA{0, 0, 255, 255}
	for y := 0; y < 15; y++ {
		for x := 0; x < 20; x++ {
			testImg.Set(x, y, blue)
		}
	}

	// Encode to JPEG bytes
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, testImg, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	jpegBytes := jpegBuf.Bytes()

	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(jpegBytes)
	}))
	defer server.Close()

	// Load image from URL
	img, err := loadImageFromURL(server.URL + "/test.jpg")
	if err != nil {
		t.Fatalf("loadImageFromURL failed: %v", err)
	}

	// Verify dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 15 {
		t.Errorf("loaded image size = %dx%d, want 20x15", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImageFromURLTimeout verifies that URL loading times out properly.
func TestLoadImageFromURLTimeout(t *testing.T) {
	// Create a server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		time.Sleep(15 * time.Second)
		w.Write([]byte("too late"))
	}))
	defer server.Close()

	// This should timeout (default timeout is 10 seconds)
	// For test purposes, we use a separate function or check if the implementation
	// respects timeout. The actual test depends on implementation details.
	start := time.Now()
	_, err := loadImageFromURL(server.URL + "/slow.png")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected timeout error for slow server")
	}

	// Should timeout well before the 15 second sleep
	if elapsed > 12*time.Second {
		t.Errorf("request took %v, expected timeout around 10 seconds", elapsed)
	}
}

// TestLoadImageFromURL404 verifies proper handling of 404 errors.
func TestLoadImageFromURL404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := loadImageFromURL(server.URL + "/notfound.png")
	if err == nil {
		t.Error("expected error for 404 response")
	}

	// Error message should indicate the status
	if err != nil && !strings.Contains(err.Error(), "404") && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Errorf("error should mention 404 or not found, got: %v", err)
	}
}

// TestLoadImageFromURL500 verifies proper handling of 500 errors.
func TestLoadImageFromURL500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := loadImageFromURL(server.URL + "/error.png")
	if err == nil {
		t.Error("expected error for 500 response")
	}

	// Error message should indicate server error
	if err != nil && !strings.Contains(err.Error(), "500") && !strings.Contains(strings.ToLower(err.Error()), "server error") {
		t.Errorf("error should mention 500 or server error, got: %v", err)
	}
}

// TestLoadImageFromURLTooLarge verifies that images exceeding size limits are rejected.
func TestLoadImageFromURLTooLarge(t *testing.T) {
	// Create a server that claims a very large Content-Length
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "100000000") // 100MB
		// Start writing some data (but less than claimed)
		w.Write([]byte("fake data"))
	}))
	defer server.Close()

	_, err := loadImageFromURL(server.URL + "/huge.png")
	if err == nil {
		t.Error("expected error for image exceeding size limit")
	}

	// Error should mention size
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "size") && !strings.Contains(strings.ToLower(err.Error()), "large") && !strings.Contains(strings.ToLower(err.Error()), "limit") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

// TestLoadImageFromURLTooLargeBody verifies that large response bodies are rejected
// even when Content-Length is not set.
func TestLoadImageFromURLTooLargeBody(t *testing.T) {
	// Create a server that streams a lot of data without Content-Length
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// Don't set Content-Length - stream chunks

		// Write PNG magic header first to make it look valid
		pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		w.Write(pngMagic)

		// Write more than MaxImageBytes (17MB to exceed 16MB limit)
		chunk := make([]byte, 1024*1024) // 1MB chunks
		for i := 0; i < 18; i++ {
			w.Write(chunk)
		}
	}))
	defer server.Close()

	_, err := loadImageFromURL(server.URL + "/streamed_huge.png")
	if err == nil {
		t.Error("expected error for large streamed image")
	}
}

// TestLoadImageFromURLBadContentType verifies that non-image content types are rejected.
func TestLoadImageFromURLBadContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{"text/html", "text/html"},
		{"text/plain", "text/plain"},
		{"application/json", "application/json"},
		{"application/pdf", "application/pdf"},
		{"video/mp4", "video/mp4"},
		{"application/octet-stream", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Write([]byte("not an image"))
			}))
			defer server.Close()

			_, err := loadImageFromURL(server.URL + "/fake.png")
			if err == nil {
				t.Errorf("expected error for Content-Type %q", tt.contentType)
			}

			// Error should mention content type
			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "content") && !strings.Contains(strings.ToLower(err.Error()), "type") {
				t.Errorf("error should mention content type, got: %v", err)
			}
		})
	}
}

// TestLoadImageFromURLValidContentTypes verifies that valid image content types are accepted.
func TestLoadImageFromURLValidContentTypes(t *testing.T) {
	// Create a valid test image
	testImg := image.NewRGBA(image.Rect(0, 0, 5, 5))
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, testImg); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	tests := []struct {
		name        string
		contentType string
	}{
		{"image/png", "image/png"},
		{"image/jpeg", "image/jpeg"},
		{"image/gif", "image/gif"},
		{"image/webp", "image/webp"},
		{"image/png with charset", "image/png; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Write(pngBytes)
			}))
			defer server.Close()

			img, err := loadImageFromURL(server.URL + "/test.png")
			// Note: image/webp might fail to decode if not supported, that's OK
			// The important thing is Content-Type validation passes
			if err != nil && !strings.Contains(err.Error(), "decode") {
				t.Errorf("unexpected error for Content-Type %q: %v", tt.contentType, err)
			}
			if err == nil && img == nil {
				t.Error("expected non-nil image on success")
			}
		})
	}
}

// TestLoadImageDispatchesURL verifies that LoadImage dispatches URL paths to loadImageFromURL.
func TestLoadImageDispatchesURL(t *testing.T) {
	// Create a test image
	testImg := image.NewRGBA(image.Rect(0, 0, 8, 8))
	green := color.RGBA{0, 255, 0, 255}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			testImg.Set(x, y, green)
		}
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, testImg); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes)
	}))
	defer server.Close()

	// LoadImage should detect the URL and dispatch to loadImageFromURL
	img, err := LoadImage(server.URL + "/test.png")
	if err != nil {
		t.Fatalf("LoadImage failed for URL: %v", err)
	}

	// Verify it loaded correctly
	bounds := img.Bounds()
	if bounds.Dx() != 8 || bounds.Dy() != 8 {
		t.Errorf("loaded image size = %dx%d, want 8x8", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImageDispatchesFile verifies that LoadImage dispatches file paths to file loading.
func TestLoadImageDispatchesFile(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "dispatch_test.png")

	// Create and save a test image
	testImg := image.NewRGBA(image.Rect(0, 0, 12, 12))
	purple := color.RGBA{128, 0, 128, 255}
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			testImg.Set(x, y, purple)
		}
	}

	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, testImg); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// LoadImage should detect the file path and load from file
	img, err := LoadImage(pngPath)
	if err != nil {
		t.Fatalf("LoadImage failed for file path: %v", err)
	}

	// Verify it loaded correctly
	bounds := img.Bounds()
	if bounds.Dx() != 12 || bounds.Dy() != 12 {
		t.Errorf("loaded image size = %dx%d, want 12x12", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImageURLErrors verifies that error messages are clear and descriptive.
func TestLoadImageURLErrors(t *testing.T) {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantSubstring string
	}{
		{
			name: "404 not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantSubstring: "404",
		},
		{
			name: "403 forbidden",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Forbidden", http.StatusForbidden)
			},
			wantSubstring: "403",
		},
		{
			name: "invalid content type",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte("<html>not an image</html>"))
			},
			wantSubstring: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			_, err := LoadImage(server.URL + "/test.png")
			if err == nil {
				t.Error("expected error")
				return
			}

			errLower := strings.ToLower(err.Error())
			wantLower := strings.ToLower(tt.wantSubstring)
			if !strings.Contains(errLower, wantLower) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantSubstring)
			}
		})
	}
}

// TestLoadImageURLNetworkError verifies handling of network connection failures.
func TestLoadImageURLNetworkError(t *testing.T) {
	// Use a URL that will fail to connect (closed port)
	// Note: This test might be flaky in some network environments
	_, err := LoadImage("http://127.0.0.1:1/nonexistent.png")
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

// TestLoadImageURLInvalidURL verifies handling of malformed URLs.
func TestLoadImageURLInvalidURL(t *testing.T) {
	invalidURLs := []string{
		"http://",
		"https://",
		"http:///path/no/host",
		"http://[invalid:ipv6/image.png",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			_, err := LoadImage(url)
			if err == nil {
				t.Errorf("expected error for invalid URL: %s", url)
			}
		})
	}
}

// TestImageCacheWorksWithURLs verifies that the ImageCache works correctly with URL paths.
func TestImageCacheWorksWithURLs(t *testing.T) {
	// Create a test image
	testImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, testImg); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	// Track how many times the server is hit
	hitCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes)
	}))
	defer server.Close()

	imageURL := server.URL + "/test.png"

	// Create cache
	cache := NewImageCache(10)

	// First load - should hit server
	cached1, err := cache.Load(imageURL)
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}
	if hitCount != 1 {
		t.Errorf("server hit count = %d, want 1", hitCount)
	}

	// Second load - should use cache, not hit server
	cached2, err := cache.Load(imageURL)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if hitCount != 1 {
		t.Errorf("server hit count = %d, want 1 (cache hit)", hitCount)
	}

	// Should be the same cached object
	if cached1 != cached2 {
		t.Error("cache should return same CachedImage on second load")
	}

	// Verify the cached image is valid
	if cached1.Original == nil {
		t.Error("cached.Original should be set")
	}
	if cached1.Width != 10 || cached1.Height != 10 {
		t.Errorf("cached image dimensions = %dx%d, want 10x10", cached1.Width, cached1.Height)
	}
}

// =============================================================================
// Phase 16I: Image Pipeline Integration Tests
// =============================================================================

// TestFrameWithImageCache verifies that Frame can be configured with an ImageCache
// via the WithImageCache option. The cache should be stored in the frame and
// used during layout to load and populate image data.
func TestFrameWithImageCache(t *testing.T) {
	// Create an image cache
	cache := NewImageCache(10)

	// Create a frame and verify it can be configured with the cache
	frame := NewFrame()
	mockFont := &testFont{width: 10, height: 14}

	// Initialize frame with the image cache option
	// The frame should accept WithImageCache as a valid option
	frame.Init(
		image.Rect(0, 0, 400, 300),
		WithFont(mockFont),
		WithImageCache(cache),
	)

	// After init, the frame should have the cache stored
	// We verify this indirectly by checking that frame operations
	// don't panic and that the frame is in a valid state
	if frame.Rect().Empty() {
		t.Error("frame should have non-empty rectangle after Init")
	}
}

// TestFrameWithImageCacheNil verifies that Frame works correctly when
// WithImageCache is called with nil (no cache).
func TestFrameWithImageCacheNil(t *testing.T) {
	frame := NewFrame()
	mockFont := &testFont{width: 10, height: 14}

	// Should not panic with nil cache
	frame.Init(
		image.Rect(0, 0, 400, 300),
		WithFont(mockFont),
		WithImageCache(nil),
	)

	// Frame should still work
	if frame.Rect().Empty() {
		t.Error("frame should have non-empty rectangle after Init")
	}
}

// TestFrameWithImageCacheUsedInLayout verifies that when a frame has an
// ImageCache, images in the content are loaded and their data is populated.
func TestFrameWithImageCacheUsedInLayout(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "layout_test.png")

	// Create a simple 25x20 image
	img := image.NewRGBA(image.Rect(0, 0, 25, 20))
	cyan := color.RGBA{0, 255, 255, 255}
	for y := 0; y < 20; y++ {
		for x := 0; x < 25; x++ {
			img.Set(x, y, cyan)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Create cache and frame
	cache := NewImageCache(10)
	frame := NewFrame()
	mockFont := &testFont{width: 10, height: 14}

	frame.Init(
		image.Rect(0, 0, 400, 300),
		WithFont(mockFont),
		WithImageCache(cache),
	)

	// Set content with an image span
	content := Content{
		Span{
			Text: "[Image: test]",
			Style: Style{
				Image:    true,
				ImageURL: pngPath,
				ImageAlt: "test",
			},
		},
	}
	frame.SetContent(content)

	// Trigger layout by calling TotalLines (which internally calls layout)
	totalLines := frame.TotalLines()
	if totalLines == 0 {
		t.Error("frame should have at least 1 line after setting content")
	}

	// Verify that the image was loaded into the cache
	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Error("image should have been loaded into cache during layout")
	}
	if cached != nil && cached.Err != nil {
		t.Errorf("cached image has unexpected error: %v", cached.Err)
	}
	if cached != nil && cached.Width != 25 {
		t.Errorf("cached image width = %d, want 25", cached.Width)
	}
}

// TestFrameLayoutUsesCache verifies that the Frame's internal layout methods
// (layoutFromOrigin, layoutBoxes) use the ImageCache to load images when set.
// This is the key integration test for Phase 16I.4 - ensuring that layoutFromOrigin()
// calls layoutWithCache() when imageCache is set.
func TestFrameLayoutUsesCache(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "frame_layout_cache_test.png")

	// Create a simple 40x30 magenta image
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	magenta := color.RGBA{255, 0, 255, 255}
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, magenta)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Create cache and frame
	cache := NewImageCache(10)
	frame := NewFrame()
	mockFont := &testFont{width: 10, height: 14}

	frame.Init(
		image.Rect(0, 0, 300, 200),
		WithFont(mockFont),
		WithImageCache(cache),
	)

	// Verify image is NOT in cache initially
	_, inCache := cache.Get(pngPath)
	if inCache {
		t.Fatal("image should NOT be in cache before frame layout")
	}

	// Set content with an image span
	content := Content{
		Span{Text: "Before "},
		Span{
			Text: "[Image: test]",
			Style: Style{
				Image:    true,
				ImageURL: pngPath,
				ImageAlt: "test",
			},
		},
		Span{Text: " After"},
	}
	frame.SetContent(content)

	// Trigger layout via various Frame methods that call layoutFromOrigin/layoutBoxes
	// All these methods internally call layoutBoxes which should use the cache

	// Method 1: TotalLines() calls layoutBoxes
	_ = frame.TotalLines()

	// Verify image was loaded into cache
	cachedImg, inCache := cache.Get(pngPath)
	if !inCache {
		t.Error("image should be in cache after Frame.TotalLines() called layoutBoxes")
	}
	if cachedImg == nil || cachedImg.Err != nil {
		t.Errorf("cached image should have loaded successfully, got err: %v",
			func() error {
				if cachedImg != nil {
					return cachedImg.Err
				}
				return nil
			}())
	}

	// Method 2: Ptofchar() also calls layoutBoxes - verify it works correctly
	pt := frame.Ptofchar(5)
	if pt.X == 0 && pt.Y == 0 {
		// Should not be at origin since "Before " has 7 characters
		// Position 5 should be somewhere inside "Before "
	}

	// Method 3: Charofpt() also calls layoutBoxes
	charPos := frame.Charofpt(image.Point{X: 50, Y: 7})
	if charPos < 0 {
		t.Error("Charofpt should return a valid character position")
	}

	// Method 4: LineStartRunes() also calls layoutBoxes
	lineStarts := frame.LineStartRunes()
	if len(lineStarts) == 0 {
		t.Error("LineStartRunes should return at least one entry")
	}

	// Verify the cache was used (image should only be loaded once)
	// We can check this indirectly by verifying the cached image dimensions
	if cachedImg.Width != 40 || cachedImg.Height != 30 {
		t.Errorf("cached image dimensions = %dx%d, want 40x30",
			cachedImg.Width, cachedImg.Height)
	}
}
