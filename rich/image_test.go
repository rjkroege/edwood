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
