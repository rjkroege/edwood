package rich

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
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

	// Verify first pixel (red) - should be R=255, G=0, B=0, A=255
	if len(data) >= 4 {
		r, g, b, a := data[0], data[1], data[2], data[3]
		if r != 255 || g != 0 || b != 0 || a != 255 {
			t.Errorf("first pixel (red) = (%d, %d, %d, %d), want (255, 0, 0, 255)", r, g, b, a)
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

	// Verify first pixel - should be cyan with full alpha
	if len(data) >= 4 {
		r, g, b, a := data[0], data[1], data[2], data[3]
		if r != 0 || g != 255 || b != 255 || a != 255 {
			t.Errorf("first pixel (cyan) = (%d, %d, %d, %d), want (0, 255, 255, 255)", r, g, b, a)
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

	// Verify black pixel (0,0) - should be R=0, G=0, B=0, A=255
	if len(data) >= 4 {
		r, g, b, a := data[0], data[1], data[2], data[3]
		if r != 0 || g != 0 || b != 0 || a != 255 {
			t.Errorf("black pixel = (%d, %d, %d, %d), want (0, 0, 0, 255)", r, g, b, a)
		}
	}

	// Verify white pixel (1,1) - should be R=255, G=255, B=255, A=255
	// Position in data: pixel at (1,1) is at index (1*2 + 1) * 4 = 12
	if len(data) >= 16 {
		r, g, b, a := data[12], data[13], data[14], data[15]
		if r != 255 || g != 255 || b != 255 || a != 255 {
			t.Errorf("white pixel = (%d, %d, %d, %d), want (255, 255, 255, 255)", r, g, b, a)
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

	// Verify fully opaque blue pixel (1,1) is correct
	// Position: (1 + 1*2) * 4 = 12
	if len(data) >= 16 {
		r, g, b, a := data[12], data[13], data[14], data[15]
		if r != 0 || g != 0 || b != 255 || a != 255 {
			t.Errorf("opaque blue pixel = (%d, %d, %d, %d), want (0, 0, 255, 255)", r, g, b, a)
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

	if len(data) >= 4 {
		r, g, b, a := data[0], data[1], data[2], data[3]
		if r != 100 || g != 150 || b != 200 || a != 255 {
			t.Errorf("pixel = (%d, %d, %d, %d), want (100, 150, 200, 255)", r, g, b, a)
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

	// Verify red pixel (0,0)
	if len(data) >= 4 {
		r, g, b, a := data[0], data[1], data[2], data[3]
		if r != 255 || g != 0 || b != 0 || a != 255 {
			t.Errorf("red pixel = (%d, %d, %d, %d), want (255, 0, 0, 255)", r, g, b, a)
		}
	}

	// Verify yellow pixel (1,1)
	if len(data) >= 16 {
		r, g, b, a := data[12], data[13], data[14], data[15]
		if r != 255 || g != 255 || b != 0 || a != 255 {
			t.Errorf("yellow pixel = (%d, %d, %d, %d), want (255, 255, 0, 255)", r, g, b, a)
		}
	}
}
