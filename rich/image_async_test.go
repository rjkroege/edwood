package rich

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// createTestPNG creates a temporary PNG file with the given dimensions and returns its path.
func createTestPNG(t *testing.T, dir string, name string, width, height int) string {
	t.Helper()
	pngPath := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	green := color.RGBA{0, 255, 0, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, green)
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

// TestLoadAsyncCacheHitReturnsImmediately verifies that when an image is already
// in the cache, LoadAsync returns it immediately with Loading=false and does not
// launch a goroutine.
func TestLoadAsyncCacheHitReturnsImmediately(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := createTestPNG(t, tmpDir, "cached.png", 20, 15)

	cache := NewImageCache(10)

	// Pre-populate cache with a synchronous load.
	preloaded, err := cache.Load(pngPath)
	if err != nil {
		t.Fatalf("pre-load failed: %v", err)
	}
	if preloaded.Width != 20 || preloaded.Height != 15 {
		t.Fatalf("pre-loaded dimensions = %dx%d, want 20x15", preloaded.Width, preloaded.Height)
	}

	// LoadAsync should return the cached entry immediately.
	callbackCalled := make(chan string, 1)
	cached, err := cache.LoadAsync(pngPath, func(path string) {
		callbackCalled <- path
	})
	if err != nil {
		t.Fatalf("LoadAsync returned error for cached image: %v", err)
	}
	if cached == nil {
		t.Fatal("LoadAsync returned nil for cached image")
	}
	if cached.Loading {
		t.Error("cached image should have Loading=false on cache hit")
	}
	if cached.Original == nil {
		t.Error("cached image should have Original set on cache hit")
	}
	if cached.Width != 20 || cached.Height != 15 {
		t.Errorf("cached dimensions = %dx%d, want 20x15", cached.Width, cached.Height)
	}

	// Callback should NOT be called for cache hits.
	select {
	case path := <-callbackCalled:
		t.Errorf("callback should not be called on cache hit, got path=%q", path)
	case <-time.After(100 * time.Millisecond):
		// Good — no callback.
	}
}

// TestLoadAsyncCacheMissTriggersAsyncLoad verifies that LoadAsync returns a
// placeholder with Loading=true on a cache miss, then asynchronously loads
// the image and updates the cache.
func TestLoadAsyncCacheMissTriggersAsyncLoad(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := createTestPNG(t, tmpDir, "async.png", 30, 25)

	// Use a delayed server so we can observe the loading state before completion.
	img := image.NewRGBA(image.Rect(0, 0, 30, 25))
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ready
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}))
	defer server.Close()
	url := server.URL + "/async.png"

	cache := NewImageCache(10)

	callbackDone := make(chan string, 1)
	placeholder, err := cache.LoadAsync(url, func(path string) {
		callbackDone <- path
	})
	if err != nil {
		t.Fatalf("LoadAsync returned error: %v", err)
	}
	if placeholder == nil {
		t.Fatal("LoadAsync returned nil placeholder")
	}

	// While server is blocked, read via cache.Get() (uses RLock) to safely check Loading.
	cached, ok := cache.Get(url)
	if !ok {
		t.Fatal("placeholder should be in cache immediately after LoadAsync")
	}
	if !cached.Loading {
		t.Error("placeholder should have Loading=true while server is blocked")
	}
	if cached.Original != nil {
		t.Error("placeholder should have nil Original while loading")
	}

	// Also test with a local file path to verify the full pipeline.
	localDone := make(chan string, 1)
	_, err = cache.LoadAsync(pngPath, func(path string) {
		localDone <- path
	})
	if err != nil {
		t.Fatalf("LoadAsync for local file returned error: %v", err)
	}

	// Wait for local file load.
	select {
	case path := <-localDone:
		if path != pngPath {
			t.Errorf("local callback path = %q, want %q", path, pngPath)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for local async load callback")
	}

	// Verify local file loaded correctly via cache.Get().
	localCached, ok := cache.Get(pngPath)
	if !ok {
		t.Fatal("local image should be in cache after async load")
	}
	if localCached.Loading {
		t.Error("local cached entry should have Loading=false after load completes")
	}
	if localCached.Original == nil {
		t.Error("local cached entry should have Original set after load completes")
	}
	if localCached.Width != 30 || localCached.Height != 25 {
		t.Errorf("local cached dimensions = %dx%d, want 30x25", localCached.Width, localCached.Height)
	}
	if localCached.Data == nil {
		t.Error("local cached entry should have Plan 9 Data after load completes")
	}
	if localCached.Err != nil {
		t.Errorf("local cached entry should have no error, got: %v", localCached.Err)
	}

	// Release server and wait for URL load.
	close(ready)
	select {
	case path := <-callbackDone:
		if path != url {
			t.Errorf("callback path = %q, want %q", path, url)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for URL async load callback")
	}

	// Verify URL loaded via cache.Get().
	urlCached, ok := cache.Get(url)
	if !ok {
		t.Fatal("URL image should be in cache after async load")
	}
	if urlCached.Loading {
		t.Error("URL cached entry should have Loading=false after load completes")
	}
	if urlCached.Original == nil {
		t.Error("URL cached entry should have Original set after load completes")
	}
	if urlCached.Err != nil {
		t.Errorf("URL cached entry should have no error, got: %v", urlCached.Err)
	}
}

// TestLoadAsyncPlaceholderDuringLoad verifies that while an image is loading,
// the placeholder entry has Loading=true, nil Original, and no error.
func TestLoadAsyncPlaceholderDuringLoad(t *testing.T) {
	// Use an HTTP server with a delay to ensure we can observe the loading state.
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait until test says to proceed — this keeps the image "loading".
		<-ready
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}))
	defer server.Close()

	cache := NewImageCache(10)

	callbackDone := make(chan struct{}, 1)
	placeholder, err := cache.LoadAsync(server.URL+"/slow.png", func(path string) {
		callbackDone <- struct{}{}
	})
	if err != nil {
		t.Fatalf("LoadAsync returned error: %v", err)
	}

	// While server is blocked, the placeholder should be in loading state.
	if !placeholder.Loading {
		t.Error("placeholder should have Loading=true during load")
	}
	if placeholder.Original != nil {
		t.Error("placeholder should have nil Original during load")
	}
	if placeholder.Err != nil {
		t.Errorf("placeholder should have nil Err during load, got: %v", placeholder.Err)
	}

	// Release the server and wait for completion.
	close(ready)
	select {
	case <-callbackDone:
		// Good.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for async load")
	}

	// After completion, loading should be false.
	if placeholder.Loading {
		t.Error("placeholder should have Loading=false after completion")
	}
}

// TestLoadAsyncCallbackInvokedOnCompletion verifies that the onLoaded callback
// is called with the correct path when an async load finishes.
func TestLoadAsyncCallbackInvokedOnCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := createTestPNG(t, tmpDir, "callback.png", 10, 10)

	cache := NewImageCache(10)

	var callbackPath string
	done := make(chan struct{})
	cache.LoadAsync(pngPath, func(path string) {
		callbackPath = path
		close(done)
	})

	select {
	case <-done:
		if callbackPath != pngPath {
			t.Errorf("callback path = %q, want %q", callbackPath, pngPath)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}

// TestLoadAsyncCallbackNilIsAllowed verifies that passing a nil callback
// to LoadAsync does not panic.
func TestLoadAsyncCallbackNilIsAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := createTestPNG(t, tmpDir, "nil_cb.png", 5, 5)

	cache := NewImageCache(10)

	placeholder, err := cache.LoadAsync(pngPath, nil)
	if err != nil {
		t.Fatalf("LoadAsync returned error: %v", err)
	}
	if placeholder == nil {
		t.Fatal("LoadAsync returned nil")
	}

	// Wait a bit for async load to finish without panicking.
	time.Sleep(500 * time.Millisecond)

	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Fatal("image should be in cache after async load")
	}
	if cached.Loading {
		t.Error("should be done loading")
	}
	if cached.Original == nil {
		t.Error("should have loaded the image")
	}
}

// TestLoadAsyncCancellationPreventsCallback verifies that calling Clear()
// cancels in-progress loads and the callback is never invoked.
func TestLoadAsyncCancellationPreventsCallback(t *testing.T) {
	// Use a slow server so we can cancel before it completes.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block for a long time — we'll cancel before this finishes.
		select {
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			// Should not reach here.
		}
	}))
	defer server.Close()

	cache := NewImageCache(10)

	callbackCalled := int32(0)
	cache.LoadAsync(server.URL+"/cancel.png", func(path string) {
		atomic.StoreInt32(&callbackCalled, 1)
	})

	// Give the goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	// Cancel by clearing the cache.
	cache.Clear()

	// Wait and verify callback was never called.
	time.Sleep(500 * time.Millisecond)
	if atomic.LoadInt32(&callbackCalled) != 0 {
		t.Error("callback should not be called after cancellation")
	}

	// Verify the entry was removed from cache.
	_, ok := cache.Get(server.URL + "/cancel.png")
	if ok {
		t.Error("cancelled entry should not remain in cache")
	}
}

// TestLoadAsyncErrorCaching verifies that when an async load fails (e.g.,
// non-existent file), the error is cached and subsequent LoadAsync calls
// return the cached error without re-fetching.
func TestLoadAsyncErrorCaching(t *testing.T) {
	cache := NewImageCache(10)

	badPath := "/nonexistent/path/to/missing.png"

	done := make(chan struct{})
	_, err := cache.LoadAsync(badPath, func(path string) {
		close(done)
	})
	if err != nil {
		t.Fatalf("LoadAsync should not return error for initial call, got: %v", err)
	}

	// Wait for async load to complete (with error).
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for error callback")
	}

	// After callback, read through cache.Get() (uses RLock) to avoid races.
	cached, ok := cache.Get(badPath)
	if !ok {
		t.Fatal("error entry should be in cache")
	}
	if cached.Loading {
		t.Error("loading should be false after error")
	}
	if cached.Err == nil {
		t.Error("cached entry should have Err set for failed load")
	}
	if cached.Original != nil {
		t.Error("cached entry should have nil Original for failed load")
	}

	// Subsequent LoadAsync should return the cached error immediately.
	callbackCalled := make(chan struct{}, 1)
	cached2, err := cache.LoadAsync(badPath, func(path string) {
		callbackCalled <- struct{}{}
	})
	if err == nil {
		t.Error("subsequent LoadAsync should return cached error")
	}
	if cached2 == nil {
		t.Fatal("subsequent LoadAsync should return cached entry")
	}
	if cached2.Loading {
		t.Error("cached error entry should not be loading")
	}
	if cached2.Err == nil {
		t.Error("cached error entry should have Err set")
	}

	// Callback should NOT be called for a cached error hit.
	select {
	case <-callbackCalled:
		t.Error("callback should not be called for cached error")
	case <-time.After(100 * time.Millisecond):
		// Good.
	}
}

// TestLoadAsyncDuplicateRequestDeduplication verifies that if LoadAsync is
// called for a path that is already loading, it returns the existing
// placeholder without starting a second goroutine.
func TestLoadAsyncDuplicateRequestDeduplication(t *testing.T) {
	// Use a slow server.
	ready := make(chan struct{})
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ready
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}))
	defer server.Close()

	cache := NewImageCache(10)
	url := server.URL + "/dedup.png"

	var callbackCount int32
	cb := func(path string) {
		atomic.AddInt32(&callbackCount, 1)
	}

	// First call starts the load.
	p1, err := cache.LoadAsync(url, cb)
	if err != nil {
		t.Fatalf("first LoadAsync error: %v", err)
	}
	if !p1.Loading {
		t.Error("first call should return loading placeholder")
	}

	// Second call for same path should return the same placeholder.
	p2, err := cache.LoadAsync(url, cb)
	if err != nil {
		t.Fatalf("second LoadAsync error: %v", err)
	}
	if p1 != p2 {
		t.Error("second LoadAsync should return the same placeholder pointer")
	}

	// Release server and wait.
	close(ready)
	time.Sleep(500 * time.Millisecond)

	// Only one callback should have been invoked (from the first goroutine).
	count := atomic.LoadInt32(&callbackCount)
	if count != 1 {
		t.Errorf("callback called %d times, want 1 (deduplication)", count)
	}
}

// TestLoadAsyncConcurrentDownloadLimit verifies that the semaphore limits
// the number of concurrent downloads to DefaultMaxParallelLoads.
func TestLoadAsyncConcurrentDownloadLimit(t *testing.T) {
	var activeCount int32
	var maxActive int32
	var mu sync.Mutex

	// Server tracks how many concurrent requests are active.
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&activeCount, 1)
		defer atomic.AddInt32(&activeCount, -1)

		mu.Lock()
		if current > maxActive {
			maxActive = current
		}
		mu.Unlock()

		// Hold the connection briefly so concurrent requests overlap.
		time.Sleep(100 * time.Millisecond)

		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}))
	defer server.Close()

	cache := NewImageCache(50)

	// Start 10 image loads (more than DefaultMaxParallelLoads=4).
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		path := server.URL + "/img" + string(rune('a'+i)) + ".png"
		cache.LoadAsync(path, func(p string) {
			wg.Done()
		})
	}

	// Wait for all loads to complete.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for all async loads to complete")
	}

	mu.Lock()
	observed := maxActive
	mu.Unlock()

	if observed > int32(DefaultMaxParallelLoads) {
		t.Errorf("max concurrent downloads = %d, want <= %d (DefaultMaxParallelLoads)",
			observed, DefaultMaxParallelLoads)
	}
	if observed == 0 {
		t.Error("max concurrent downloads should be > 0")
	}
}

// TestLoadAsyncHTTPCancellation verifies that when LoadAsync is cancelled via
// Clear(), the HTTP request is actually cancelled (server sees connection close).
func TestLoadAsyncHTTPCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCtxDone := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		// Wait for context cancellation or timeout.
		select {
		case <-r.Context().Done():
			close(requestCtxDone)
			return
		case <-time.After(10 * time.Second):
			// Should not reach.
		}
	}))
	defer server.Close()

	cache := NewImageCache(10)
	cache.LoadAsync(server.URL+"/http_cancel.png", nil)

	// Wait for the request to reach the server.
	select {
	case <-requestStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for request to start")
	}

	// Cancel by clearing the cache.
	cache.Clear()

	// Server should see the cancellation.
	select {
	case <-requestCtxDone:
		// Good — HTTP request was cancelled.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for HTTP request cancellation")
	}
}

// TestLoadAsyncRaceDetector exercises LoadAsync with concurrent calls to
// exercise the race detector. This test is primarily useful with `go test -race`.
func TestLoadAsyncRaceDetector(t *testing.T) {
	tmpDir := t.TempDir()

	// Create several test images.
	paths := make([]string, 5)
	for i := range paths {
		paths[i] = createTestPNG(t, tmpDir, "race"+string(rune('0'+i))+".png", 10, 10)
	}

	cache := NewImageCache(10)
	var wg sync.WaitGroup

	// Concurrently load all images and also concurrently read from cache.
	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			cache.LoadAsync(path, func(p string) {})
		}(p)

		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			cache.Get(path)
		}(p)
	}

	wg.Wait()

	// Give async loads time to finish.
	time.Sleep(1 * time.Second)

	// Verify all images loaded.
	for _, p := range paths {
		cached, ok := cache.Get(p)
		if !ok {
			t.Errorf("image %s not in cache after concurrent load", filepath.Base(p))
			continue
		}
		if cached.Loading {
			t.Errorf("image %s still loading after wait", filepath.Base(p))
		}
	}
}

// TestLoadImageWithContextCancellation verifies that LoadImageWithContext
// respects context cancellation before starting the load.
func TestLoadImageWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := LoadImageWithContext(ctx, "/some/file.png")
	if err == nil {
		t.Error("LoadImageWithContext should return error for cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestLoadImageWithContextLocalFile verifies that LoadImageWithContext
// successfully loads a local file when the context is not cancelled.
func TestLoadImageWithContextLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := createTestPNG(t, tmpDir, "ctx_local.png", 15, 12)

	ctx := context.Background()
	img, err := LoadImageWithContext(ctx, pngPath)
	if err != nil {
		t.Fatalf("LoadImageWithContext failed: %v", err)
	}
	if img == nil {
		t.Fatal("LoadImageWithContext returned nil image")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 15 || bounds.Dy() != 12 {
		t.Errorf("image dimensions = %dx%d, want 15x12", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImageWithContextHTTPURL verifies that LoadImageWithContext
// loads an image from an HTTP URL with context support.
func TestLoadImageWithContextHTTPURL(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}))
	defer server.Close()

	ctx := context.Background()
	loaded, err := LoadImageWithContext(ctx, server.URL+"/test.png")
	if err != nil {
		t.Fatalf("LoadImageWithContext failed for URL: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadImageWithContext returned nil for URL")
	}
	bounds := loaded.Bounds()
	if bounds.Dx() != 8 || bounds.Dy() != 8 {
		t.Errorf("URL image dimensions = %dx%d, want 8x8", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImageWithContextHTTPCancellation verifies that LoadImageWithContext
// aborts an HTTP request when the context is cancelled mid-download.
func TestLoadImageWithContextHTTPCancellation(t *testing.T) {
	requestArrived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestArrived)
		// Block until context is done.
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := LoadImageWithContext(ctx, server.URL+"/slow.png")
		errCh <- err
	}()

	// Wait for the request to arrive at the server.
	select {
	case <-requestArrived:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for HTTP request")
	}

	// Cancel the context.
	cancel()

	// The load should fail with a context error.
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error from cancelled context")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for LoadImageWithContext to return")
	}
}

// TestImageCacheClearCancelsPendingLoads verifies that Clear() cancels
// pending async loads and reinitializes the cancelFuncs map.
func TestImageCacheClearCancelsPendingLoads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block indefinitely.
		<-r.Context().Done()
	}))
	defer server.Close()

	cache := NewImageCache(10)

	// Start several async loads.
	for i := 0; i < 3; i++ {
		path := server.URL + "/clear_test_" + string(rune('a'+i)) + ".png"
		cache.LoadAsync(path, nil)
	}

	// Give goroutines time to start.
	time.Sleep(50 * time.Millisecond)

	// Clear should cancel all pending loads.
	cache.Clear()

	// After clear, cache should be empty.
	for i := 0; i < 3; i++ {
		path := server.URL + "/clear_test_" + string(rune('a'+i)) + ".png"
		_, ok := cache.Get(path)
		if ok {
			t.Errorf("cache should be empty after Clear, but found %s", path)
		}
	}

	// Should be able to start new loads after Clear.
	tmpDir := t.TempDir()
	pngPath := createTestPNG(t, tmpDir, "after_clear.png", 5, 5)

	done := make(chan struct{})
	cache.LoadAsync(pngPath, func(path string) {
		close(done)
	})

	select {
	case <-done:
		// Good — new load works after Clear.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for load after Clear")
	}

	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Error("new load after Clear should populate cache")
	}
	if cached != nil && cached.Loading {
		t.Error("new load should complete after Clear")
	}
}

// TestDefaultMaxParallelLoads verifies the constant value.
func TestDefaultMaxParallelLoads(t *testing.T) {
	if DefaultMaxParallelLoads <= 0 {
		t.Errorf("DefaultMaxParallelLoads = %d, want > 0", DefaultMaxParallelLoads)
	}
	if DefaultMaxParallelLoads != 4 {
		t.Errorf("DefaultMaxParallelLoads = %d, want 4", DefaultMaxParallelLoads)
	}
}
