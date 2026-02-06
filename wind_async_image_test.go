package main

// Tests for Phase 9.3: Integrate Async Loading into Preview
//
// These tests verify that preview mode correctly handles async image loading:
// - Preview renders immediately with placeholders when images are not yet cached
// - After images load asynchronously, the preview updates to show real images
// - Exiting preview mode cancels pending async image loads
// - Cache hits (pre-loaded images) render synchronously without placeholders
//
// The tests use a test HTTP server with configurable delays to simulate
// slow image downloads and control the timing of async completion.
//
// Run with: go test -race -run TestPreviewAsyncImage ./...

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

// createTestPNGFile creates a temporary PNG file with the given dimensions.
// Returns the path to the created file.
func createTestPNGFile(t *testing.T, dir string, name string, width, height int) string {
	t.Helper()
	pngPath := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	c := color.RGBA{0, 128, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("failed to create test PNG %s: %v", name, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG %s: %v", name, err)
	}
	f.Close()
	return pngPath
}

// setupPreviewWindowWithImageCache creates a Window in preview mode with an image
// cache and base path configured. The markdown content should contain image references.
// Returns the window, the RichText, and the image cache.
func setupPreviewWindowWithImageCache(t *testing.T, markdownContent, basePath string, cache *rich.ImageCache) (*Window, *RichText) {
	t.Helper()

	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer(basePath, sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithRichTextImageCache(cache),
		WithRichTextBasePath(basePath),
	)

	content, sourceMap, linkMap := markdown.ParseWithSourceMap(markdownContent)
	rt.SetContent(content)
	rt.Render(image.Rect(0, 20, 800, 600))

	w.imageCache = cache
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	return w, rt
}

// TestPreviewAsyncImagePlaceholderOnCacheMiss verifies that when a preview is
// rendered with an uncached image, the layout uses a placeholder (Loading=true
// entry in the cache) and the preview still renders immediately without blocking.
//
// This test exercises the integration between UpdatePreview() and LoadAsync():
// - The image is not pre-loaded, so LoadAsync returns a placeholder
// - The layout engine should handle the placeholder gracefully
// - The preview should render without waiting for the image to download
func TestPreviewAsyncImagePlaceholderOnCacheMiss(t *testing.T) {
	// Set up a slow HTTP server that blocks until we release it.
	testImg := image.NewRGBA(image.Rect(0, 0, 50, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 50; x++ {
			testImg.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ready // Block until test releases
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, testImg)
	}))
	defer server.Close()
	defer close(ready)

	imgURL := server.URL + "/slow_image.png"
	markdownContent := fmt.Sprintf("# Test\n\n![Slow Image](%s)\n\nText after image.\n", imgURL)

	cache := rich.NewImageCache(10)

	// Pre-trigger async load so the placeholder is in the cache
	placeholder, err := cache.LoadAsync(imgURL, nil)
	if err != nil {
		t.Fatalf("LoadAsync returned error: %v", err)
	}
	if !placeholder.Loading {
		t.Fatal("placeholder should have Loading=true")
	}

	// Now set up the preview window — the cache already has a loading placeholder.
	// The layout should use the placeholder without blocking.
	w, rt := setupPreviewWindowWithImageCache(t, markdownContent, "/test/readme.md", cache)

	// Verify preview mode is active and rendered without blocking
	if !w.previewMode {
		t.Error("previewMode should be true")
	}
	if rt.Content() == nil {
		t.Error("content should not be nil after render")
	}

	// Verify the placeholder is still loading (server is still blocked)
	cached, ok := cache.Get(imgURL)
	if !ok {
		t.Fatal("image should be in cache as placeholder")
	}
	if !cached.Loading {
		t.Error("image should still be loading (server is blocked)")
	}
	if cached.Original != nil {
		t.Error("placeholder should have nil Original while loading")
	}
}

// TestPreviewAsyncImageCacheHitRendersImmediately verifies that when all images
// are already cached (cache hit), the preview renders immediately with actual
// image data — no placeholders, no async callbacks needed.
func TestPreviewAsyncImageCacheHitRendersImmediately(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real PNG file
	pngPath := createTestPNGFile(t, tmpDir, "cached_img.png", 40, 30)

	// Create markdown referencing the image with an absolute path
	markdownContent := fmt.Sprintf("# Test\n\n![Cached Image](%s)\n\nText after.\n", pngPath)

	cache := rich.NewImageCache(10)

	// Pre-load the image synchronously so it's a cache hit
	preloaded, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("pre-load failed: %v", err)
	}
	if preloaded.Loading {
		t.Fatal("pre-loaded image should not be in loading state")
	}
	if preloaded.Original == nil {
		t.Fatal("pre-loaded image should have Original set")
	}

	mdPath := filepath.Join(tmpDir, "test.md")

	w, rt := setupPreviewWindowWithImageCache(t, markdownContent, mdPath, cache)

	// Preview should be active
	if !w.previewMode {
		t.Error("previewMode should be true")
	}

	// Content should be rendered
	if rt.Content() == nil {
		t.Fatal("content should not be nil")
	}

	// The cached entry should still be fully loaded (not reverted to placeholder)
	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Fatal("image should be in cache")
	}
	if cached.Loading {
		t.Error("cached image should have Loading=false (cache hit)")
	}
	if cached.Original == nil {
		t.Error("cached image should have Original set (cache hit)")
	}
	if cached.Width != 40 || cached.Height != 30 {
		t.Errorf("cached image dimensions = %dx%d, want 40x30", cached.Width, cached.Height)
	}
}

// TestPreviewAsyncImageCallbackUpdatesPreview verifies the end-to-end flow:
// 1. Preview renders with a placeholder for an uncached image
// 2. The image loads asynchronously in the background
// 3. The onLoaded callback fires
// 4. The cache is updated with the real image data
//
// This tests the integration at the cache level. The actual re-render
// wiring (WithOnImageLoaded callback triggering UpdatePreview) is tested
// by verifying the cache transitions from Loading=true to Loading=false
// with valid image data.
func TestPreviewAsyncImageCallbackUpdatesPreview(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real PNG file for async loading
	pngPath := createTestPNGFile(t, tmpDir, "async_load.png", 60, 45)

	markdownContent := fmt.Sprintf("# Async Test\n\n![Async Image](%s)\n\nMore text.\n", pngPath)

	cache := rich.NewImageCache(10)
	mdPath := filepath.Join(tmpDir, "test.md")

	// Set up the callback to track when the async load completes
	callbackDone := make(chan string, 1)
	_, err := cache.LoadAsync(pngPath, func(path string) {
		callbackDone <- path
	})
	if err != nil {
		t.Fatalf("LoadAsync returned error: %v", err)
	}

	// Wait for the async load to complete (local file should be fast)
	select {
	case path := <-callbackDone:
		if path != pngPath {
			t.Errorf("callback path = %q, want %q", path, pngPath)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for async load callback")
	}

	// Now the cache should have the real image
	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Fatal("image should be in cache after async load")
	}
	if cached.Loading {
		t.Error("image should not be loading after callback")
	}
	if cached.Original == nil {
		t.Error("image should have Original set after load")
	}
	if cached.Width != 60 || cached.Height != 45 {
		t.Errorf("image dimensions = %dx%d, want 60x45", cached.Width, cached.Height)
	}

	// Set up preview with the now-populated cache — should be a cache hit
	w, rt := setupPreviewWindowWithImageCache(t, markdownContent, mdPath, cache)

	if !w.previewMode {
		t.Error("previewMode should be true")
	}
	if rt.Content() == nil {
		t.Error("content should not be nil")
	}

	// Now call UpdatePreview to simulate re-render after image loads
	w.UpdatePreview()

	// Verify preview is still functional after update
	if !w.previewMode {
		t.Error("previewMode should still be true after UpdatePreview")
	}
}

// TestPreviewAsyncImageExitCancelsPendingLoads verifies that when preview mode
// is exited, calling Clear() on the image cache cancels any pending async loads.
// This prevents callbacks from firing after the preview is gone, which would
// cause crashes or data races.
func TestPreviewAsyncImageExitCancelsPendingLoads(t *testing.T) {
	// Use a slow HTTP server that blocks indefinitely until cancelled
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request is cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	imgURL := server.URL + "/never_loads.png"
	markdownContent := fmt.Sprintf("# Test\n\n![Never Loads](%s)\n", imgURL)

	cache := rich.NewImageCache(10)

	// Start an async load that will block
	var callbackCalled int32
	cache.LoadAsync(imgURL, func(path string) {
		atomic.StoreInt32(&callbackCalled, 1)
	})

	// Give the goroutine time to start the HTTP request
	time.Sleep(50 * time.Millisecond)

	// Verify it's in loading state
	cached, ok := cache.Get(imgURL)
	if !ok {
		t.Fatal("placeholder should be in cache")
	}
	if !cached.Loading {
		t.Fatal("should be loading")
	}

	// Set up preview with this cache
	w, _ := setupPreviewWindowWithImageCache(t, markdownContent, "/test/readme.md", cache)
	if !w.previewMode {
		t.Fatal("previewMode should be true")
	}

	// Exit preview mode — this should clear the cache and cancel pending loads
	w.SetPreviewMode(false)
	cache.Clear()
	w.imageCache = nil

	// Wait and verify the callback was never called
	time.Sleep(500 * time.Millisecond)
	if atomic.LoadInt32(&callbackCalled) != 0 {
		t.Error("callback should not be called after cache.Clear() cancels pending loads")
	}

	// Verify the cache is empty
	_, ok = cache.Get(imgURL)
	if ok {
		t.Error("cache should be empty after Clear()")
	}
}

// TestPreviewAsyncImageUpdatePreviewWithMixedCacheState verifies that
// UpdatePreview() works correctly when the cache contains a mix of:
// - Fully loaded images (cache hit)
// - Loading placeholders (async in progress)
// - Error entries (failed loads)
//
// This simulates a realistic scenario where a markdown file references
// multiple images in different states.
func TestPreviewAsyncImageUpdatePreviewWithMixedCacheState(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two real PNG files
	goodPath := createTestPNGFile(t, tmpDir, "good.png", 30, 20)

	// Set up a slow server for the "loading" image
	testImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ready
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, testImg)
	}))
	defer server.Close()
	defer close(ready)

	loadingURL := server.URL + "/loading.png"
	badPath := "/nonexistent/path/to/bad.png"

	markdownContent := fmt.Sprintf(
		"# Mixed Images\n\n![Good](%s)\n\n![Loading](%s)\n\n![Bad](%s)\n\nEnd.\n",
		goodPath, loadingURL, badPath,
	)

	cache := rich.NewImageCache(10)

	// Pre-load the good image (cache hit)
	_, err := cache.Load(goodPath)
	if err != nil {
		t.Fatalf("failed to load good image: %v", err)
	}

	// Start async load for the loading image (will block)
	cache.LoadAsync(loadingURL, nil)

	// Start async load for the bad image (will fail quickly)
	badDone := make(chan struct{})
	cache.LoadAsync(badPath, func(path string) {
		close(badDone)
	})

	// Wait for the bad image to fail
	select {
	case <-badDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for bad image error")
	}

	// Now we have: good (loaded), loading (in progress), bad (error)
	mdPath := filepath.Join(tmpDir, "test.md")
	w, rt := setupPreviewWindowWithImageCache(t, markdownContent, mdPath, cache)

	if !w.previewMode {
		t.Error("previewMode should be true")
	}
	if rt.Content() == nil {
		t.Error("content should not be nil")
	}

	// Verify cache state for each image
	goodCached, ok := cache.Get(goodPath)
	if !ok || goodCached.Loading || goodCached.Original == nil {
		t.Error("good image should be fully loaded in cache")
	}

	loadingCached, ok := cache.Get(loadingURL)
	if !ok {
		t.Error("loading image should be in cache as placeholder")
	}
	if !loadingCached.Loading {
		t.Error("loading image should still be loading")
	}

	badCached, ok := cache.Get(badPath)
	if !ok {
		t.Error("bad image should be in cache as error")
	}
	if badCached.Loading {
		t.Error("bad image should not be loading")
	}
	if badCached.Err == nil {
		t.Error("bad image should have error set")
	}

	// Calling UpdatePreview should work without panic
	w.UpdatePreview()

	// Preview should still be functional
	if !w.previewMode {
		t.Error("previewMode should still be true")
	}
}

// TestPreviewAsyncImageLoadThenUpdatePreview verifies the full async lifecycle:
// 1. Start preview with an uncached HTTP image (placeholder in cache)
// 2. Release the HTTP server so the image loads
// 3. After the callback fires, call UpdatePreview
// 4. Verify the cache now has real image data
func TestPreviewAsyncImageLoadThenUpdatePreview(t *testing.T) {
	// Set up an HTTP server that serves a PNG after a signal
	testImg := image.NewRGBA(image.Rect(0, 0, 25, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 25; x++ {
			testImg.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ready
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, testImg)
	}))
	defer server.Close()

	imgURL := server.URL + "/delayed.png"
	markdownContent := fmt.Sprintf("# Delayed\n\n![Delayed Image](%s)\n", imgURL)

	cache := rich.NewImageCache(10)

	// Start async load with a callback
	loadDone := make(chan string, 1)
	cache.LoadAsync(imgURL, func(path string) {
		loadDone <- path
	})

	// Verify placeholder is in loading state
	cached, ok := cache.Get(imgURL)
	if !ok {
		t.Fatal("placeholder should be in cache")
	}
	if !cached.Loading {
		t.Fatal("should be loading before server responds")
	}

	// Set up preview with the loading placeholder
	w, _ := setupPreviewWindowWithImageCache(t, markdownContent, "/test/delayed.md", cache)
	if !w.previewMode {
		t.Fatal("previewMode should be true")
	}

	// Release the server to let the image load
	close(ready)

	// Wait for the async load callback
	select {
	case path := <-loadDone:
		if path != imgURL {
			t.Errorf("callback path = %q, want %q", path, imgURL)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for async load")
	}

	// Verify the cache has been updated with real image data
	cached, ok = cache.Get(imgURL)
	if !ok {
		t.Fatal("image should still be in cache after load")
	}
	if cached.Loading {
		t.Error("image should not be loading after completion")
	}
	if cached.Original == nil {
		t.Error("image should have Original set after load")
	}
	if cached.Width != 25 || cached.Height != 20 {
		t.Errorf("image dimensions = %dx%d, want 25x20", cached.Width, cached.Height)
	}

	// Simulate what the onImageLoaded callback would do: re-render the preview
	w.UpdatePreview()

	// Preview should still be functional with the now-loaded image
	if !w.previewMode {
		t.Error("previewMode should still be true after re-render")
	}
}

// TestPreviewAsyncImageLocalFileLoadsQuickly verifies that local file images
// load quickly via async and the cache is populated correctly. Local files
// should complete almost immediately since there's no network delay.
func TestPreviewAsyncImageLocalFileLoadsQuickly(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := createTestPNGFile(t, tmpDir, "local.png", 35, 28)

	markdownContent := fmt.Sprintf("# Local\n\n![Local Image](%s)\n", pngPath)

	cache := rich.NewImageCache(10)

	// Start async load
	loadDone := make(chan string, 1)
	cache.LoadAsync(pngPath, func(path string) {
		loadDone <- path
	})

	// Local files should load very quickly
	select {
	case path := <-loadDone:
		if path != pngPath {
			t.Errorf("callback path = %q, want %q", path, pngPath)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for local file async load")
	}

	// Verify cache state
	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Fatal("image should be in cache")
	}
	if cached.Loading {
		t.Error("should be done loading")
	}
	if cached.Original == nil {
		t.Error("should have Original set")
	}
	if cached.Width != 35 || cached.Height != 28 {
		t.Errorf("dimensions = %dx%d, want 35x28", cached.Width, cached.Height)
	}

	// Now set up preview — should be a cache hit
	mdPath := filepath.Join(tmpDir, "test.md")
	w, rt := setupPreviewWindowWithImageCache(t, markdownContent, mdPath, cache)

	if !w.previewMode {
		t.Error("previewMode should be true")
	}
	if rt.Content() == nil {
		t.Error("content should not be nil")
	}

	// UpdatePreview should work cleanly with cached images
	w.UpdatePreview()
	if !w.previewMode {
		t.Error("previewMode should still be true")
	}
}

// TestPreviewAsyncImageRaceDetector exercises the async image loading in a
// preview context to verify there are no data races. This test is primarily
// useful when run with `go test -race`.
//
// It creates a preview with multiple images loading concurrently, then
// calls UpdatePreview while images are still loading.
func TestPreviewAsyncImageRaceDetector(t *testing.T) {
	tmpDir := t.TempDir()

	// Create several test images
	paths := make([]string, 5)
	for i := range paths {
		paths[i] = createTestPNGFile(t, tmpDir, fmt.Sprintf("race%d.png", i), 10, 10)
	}

	// Build markdown with all images
	var md string
	md = "# Race Test\n\n"
	for i, p := range paths {
		md += fmt.Sprintf("![Image %d](%s)\n\n", i, p)
	}

	cache := rich.NewImageCache(10)

	// Start all async loads
	for _, p := range paths {
		cache.LoadAsync(p, nil)
	}

	// Set up preview while images may still be loading
	mdPath := filepath.Join(tmpDir, "race_test.md")
	w, _ := setupPreviewWindowWithImageCache(t, md, mdPath, cache)

	// Call UpdatePreview a few times while async loads complete
	for i := 0; i < 3; i++ {
		w.UpdatePreview()
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all images to finish loading
	time.Sleep(2 * time.Second)

	// All images should be loaded
	for _, p := range paths {
		cached, ok := cache.Get(p)
		if !ok {
			t.Errorf("image %s not in cache", filepath.Base(p))
			continue
		}
		if cached.Loading {
			t.Errorf("image %s still loading", filepath.Base(p))
		}
		if cached.Original == nil {
			t.Errorf("image %s has nil Original", filepath.Base(p))
		}
	}

	// Final UpdatePreview with all images loaded
	w.UpdatePreview()
	if !w.previewMode {
		t.Error("previewMode should still be true")
	}
}
