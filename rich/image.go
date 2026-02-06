// Package rich provides image loading and management for Markdeep rendering.
package rich

import (
	"context"
	"crypto/tls"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Image size limits to prevent memory exhaustion.
const (
	MaxImageWidth  = 4096              // Maximum width in pixels
	MaxImageHeight = 4096              // Maximum height in pixels
	MaxImageBytes  = 16 * 1024 * 1024  // 16MB uncompressed (RGBA at 4 bytes/pixel)
)

// URLImageTimeout is the maximum time to wait for URL image downloads.
const URLImageTimeout = 10 * time.Second

// isTLSError returns true if the error is a TLS-related error that might
// be resolved by retrying with InsecureSkipVerify. Checks for "tls:" and
// "certificate" substrings which cover tls: handshake failure, x509:
// certificate errors, etc.
func isTLSError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "tls:") || strings.Contains(msg, "certificate")
}

// isImageURL returns true if path is an HTTP or HTTPS URL.
// Only http:// and https:// schemes are supported for security reasons.
func isImageURL(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// loadImageFromURL fetches an image from an HTTP(S) URL.
// It enforces timeout, size limits, and content-type validation.
// On TLS errors, it retries with InsecureSkipVerify to handle servers
// with legacy or self-signed certificates.
func loadImageFromURL(url string) (image.Image, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: URLImageTimeout,
	}

	// Make the request
	resp, err := client.Get(url)
	if err != nil && isTLSError(err) {
		// Retry with relaxed TLS settings
		insecureClient := &http.Client{
			Timeout: URLImageTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		resp, err = insecureClient.Get(url)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Validate Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		// Extract media type (ignore parameters like charset)
		mediaType := strings.TrimSpace(strings.Split(contentType, ";")[0])
		if !strings.HasPrefix(mediaType, "image/") {
			return nil, fmt.Errorf("invalid content type: expected image/*, got %q", contentType)
		}
	}

	// Check Content-Length if provided
	if resp.ContentLength > int64(MaxImageBytes) {
		return nil, fmt.Errorf("image too large: %d bytes exceeds limit of %d bytes", resp.ContentLength, MaxImageBytes)
	}

	// Use LimitReader to prevent reading more than MaxImageBytes
	limitedReader := io.LimitReader(resp.Body, int64(MaxImageBytes)+1)

	// Decode the image
	img, _, err := image.Decode(limitedReader)
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

	// Check uncompressed size
	uncompressedSize := width * height * 4
	if uncompressedSize > MaxImageBytes {
		return nil, fmt.Errorf("image uncompressed size exceeds limit: %d bytes (max %d bytes)",
			uncompressedSize, MaxImageBytes)
	}

	return img, nil
}

// LoadImage loads an image from a file path or URL.
// Supports PNG, JPEG, and GIF (first frame only for GIF).
// For URLs, only http:// and https:// schemes are supported.
// Returns the decoded image or an error if the file cannot be read,
// the format is not supported, or the image exceeds size limits.
func LoadImage(path string) (image.Image, error) {
	// Check if this is a URL
	if isImageURL(path) {
		return loadImageFromURL(path)
	}

	// Load from local file
	return loadImageFromFile(path)
}

// loadImageFromFile loads an image from a local file path.
func loadImageFromFile(path string) (image.Image, error) {
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
// pixel being 4 bytes in ABGR order (little-endian RGBA32):
// byte[0]=A, byte[1]=B, byte[2]=G, byte[3]=R.
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
				// Plan 9 RGBA32 uses little-endian byte order (ABGR in memory).
				// Note: RGBA() already returns pre-multiplied values for most image types
				// so we just convert from 16-bit to 8-bit here.
				data[i] = a                  // byte 0: Alpha
				data[i+1] = uint8(b32 >> 8)  // byte 1: Blue
				data[i+2] = uint8(g32 >> 8)  // byte 2: Green
				data[i+3] = uint8(r32 >> 8)  // byte 3: Red
			}
			i += 4
		}
	}

	return data, nil
}

// DefaultCacheSize is the default maximum number of images to cache.
const DefaultCacheSize = 50

// CachedImage represents a loaded and converted image.
type CachedImage struct {
	Original image.Image // Go stdlib image
	Data     []byte      // Plan 9 RGBA32 pixel data
	Width    int         // Original width in pixels
	Height   int         // Original height in pixels
	Path     string      // Source path for debugging
	LoadTime time.Time   // When loaded
	Err      error       // Error if load failed
	Loading  bool        // true while async load is in progress
}

// DefaultMaxParallelLoads is the maximum number of concurrent async image downloads.
const DefaultMaxParallelLoads = 4

// ImageCache provides an LRU cache for loaded images.
type ImageCache struct {
	mu          sync.RWMutex
	images      map[string]*CachedImage
	order       []string                    // LRU order (oldest first)
	maxSize     int                         // Maximum number of cached images
	maxParallel int                         // max concurrent loads (default 4)
	sem         chan struct{}               // semaphore for concurrent downloads
	cancelFuncs map[string]context.CancelFunc // per-path cancellation
}

// NewImageCache creates a new image cache with the specified maximum size.
// If maxSize <= 0, DefaultCacheSize is used.
func NewImageCache(maxSize int) *ImageCache {
	if maxSize <= 0 {
		maxSize = DefaultCacheSize
	}
	return &ImageCache{
		images:  make(map[string]*CachedImage),
		order:   make([]string, 0),
		maxSize: maxSize,
	}
}

// Get retrieves a cached image without loading.
// Returns the CachedImage and true if found, nil and false otherwise.
func (c *ImageCache) Get(path string) (*CachedImage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.images[path]
	return cached, ok
}

// Load loads an image from the given path, using the cache if available.
// On error, returns a CachedImage with Err set (and caches the error).
func (c *ImageCache) Load(path string) (*CachedImage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already cached (including cached errors)
	if cached, ok := c.images[path]; ok {
		return cached, cached.Err
	}

	// Create new cache entry
	cached := &CachedImage{
		Path:     path,
		LoadTime: time.Now(),
	}

	// Try to load the image
	img, err := LoadImage(path)
	if err != nil {
		cached.Err = err
		c.images[path] = cached
		c.order = append(c.order, path)
		c.evictOldest()
		return cached, err
	}

	// Store original image and dimensions
	cached.Original = img
	cached.Width = img.Bounds().Dx()
	cached.Height = img.Bounds().Dy()

	// Convert to Plan 9 format
	data, err := ConvertToPlan9(img)
	if err != nil {
		cached.Err = fmt.Errorf("failed to convert image: %w", err)
		c.images[path] = cached
		c.order = append(c.order, path)
		c.evictOldest()
		return cached, cached.Err
	}

	cached.Data = data
	c.images[path] = cached
	c.order = append(c.order, path)
	c.evictOldest()

	return cached, nil
}

// Clear removes all cached images and cancels any pending async loads.
func (c *ImageCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel all pending async loads.
	for _, cancel := range c.cancelFuncs {
		cancel()
	}
	c.cancelFuncs = make(map[string]context.CancelFunc)

	// Note: In a full implementation with Plan 9 display integration,
	// we would call Plan9Image.Free() for each cached image here.
	c.images = make(map[string]*CachedImage)
	c.order = make([]string, 0)
}

// Evict removes a specific image from the cache.
func (c *ImageCache) Evict(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.images[path]; !ok {
		return
	}

	delete(c.images, path)

	// Remove from order list
	for i, p := range c.order {
		if p == path {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

// evictOldest removes the oldest entries if cache exceeds maxSize.
// Must be called with lock held.
func (c *ImageCache) evictOldest() {
	for len(c.order) > c.maxSize {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.images, oldest)
	}
}

// LoadAsync checks the cache for path. On hit, returns the cached entry
// immediately (synchronous). On miss, creates a placeholder entry with
// Loading=true, starts a background goroutine to load the image, and
// returns the placeholder. When loading completes, the cache entry is
// updated in place and onLoaded is called (if non-nil).
//
// The onLoaded callback runs on an unspecified goroutine. Callers that
// need main-goroutine execution must marshal through the row lock or a
// channel (same pattern as SchedulePreviewUpdate).
//
// If path is already loading (Loading=true in cache), returns the
// existing placeholder without starting a second goroutine.
func (c *ImageCache) LoadAsync(path string, onLoaded func(path string)) (*CachedImage, error) {
	c.mu.Lock()

	// Check cache (hit or already-loading).
	if cached, ok := c.images[path]; ok {
		c.mu.Unlock()
		return cached, cached.Err
	}

	// Create placeholder.
	placeholder := &CachedImage{
		Path:     path,
		LoadTime: time.Now(),
		Loading:  true,
	}
	c.images[path] = placeholder
	c.order = append(c.order, path)
	c.evictOldest()

	// Create cancellation context.
	ctx, cancel := context.WithCancel(context.Background())
	if c.cancelFuncs == nil {
		c.cancelFuncs = make(map[string]context.CancelFunc)
	}
	c.cancelFuncs[path] = cancel

	// Lazily init semaphore.
	if c.sem == nil {
		max := c.maxParallel
		if max <= 0 {
			max = DefaultMaxParallelLoads
		}
		c.sem = make(chan struct{}, max)
	}
	sem := c.sem

	c.mu.Unlock()

	// Launch background load.
	go func() {
		// Acquire semaphore slot.
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
		case <-ctx.Done():
			return
		}

		// Load the image (this is the slow part).
		img, err := LoadImageWithContext(ctx, path)

		// Update cache under lock.
		c.mu.Lock()
		delete(c.cancelFuncs, path)

		if ctx.Err() != nil {
			// Cancelled — remove the placeholder so a future LoadAsync retries.
			delete(c.images, path)
			c.mu.Unlock()
			return
		}

		placeholder.Loading = false
		if err != nil {
			placeholder.Err = err
		} else {
			placeholder.Original = img
			placeholder.Width = img.Bounds().Dx()
			placeholder.Height = img.Bounds().Dy()
			data, convErr := ConvertToPlan9(img)
			if convErr != nil {
				placeholder.Err = convErr
			} else {
				placeholder.Data = data
			}
		}
		c.mu.Unlock()

		// Notify caller (outside lock).
		if onLoaded != nil {
			onLoaded(path)
		}
	}()

	return placeholder, nil
}

// LoadImageWithContext is like LoadImage but supports cancellation via context.
// For HTTP URLs, the context is attached to the request.
// For local files, the context is checked before and after the file read.
func LoadImageWithContext(ctx context.Context, path string) (image.Image, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if isImageURL(path) {
		return loadImageFromURLWithContext(ctx, path)
	}
	// For local files, check context then load (local I/O is fast enough
	// that mid-read cancellation isn't critical).
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return loadImageFromFile(path)
}

// loadImageFromURLWithContext fetches an image from an HTTP(S) URL with context support.
func loadImageFromURLWithContext(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: URLImageTimeout}
	resp, err := client.Do(req)
	if err != nil && isTLSError(err) {
		// Retry with relaxed TLS settings.
		insecureClient := &http.Client{
			Timeout: URLImageTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		req2, err2 := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err2 != nil {
			return nil, fmt.Errorf("failed to create request: %w", err2)
		}
		resp, err = insecureClient.Do(req2)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	// Check status code.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Validate Content-Type.
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		mediaType := strings.TrimSpace(strings.Split(contentType, ";")[0])
		if !strings.HasPrefix(mediaType, "image/") {
			return nil, fmt.Errorf("invalid content type: expected image/*, got %q", contentType)
		}
	}

	// Check Content-Length if provided.
	if resp.ContentLength > int64(MaxImageBytes) {
		return nil, fmt.Errorf("image too large: %d bytes exceeds limit of %d bytes", resp.ContentLength, MaxImageBytes)
	}

	// Use LimitReader to prevent reading more than MaxImageBytes.
	limitedReader := io.LimitReader(resp.Body, int64(MaxImageBytes)+1)

	// Decode the image.
	img, _, err := image.Decode(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Validate image dimensions.
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width > MaxImageWidth || height > MaxImageHeight {
		return nil, fmt.Errorf("image too large: %dx%d (max %dx%d)",
			width, height, MaxImageWidth, MaxImageHeight)
	}

	// Check uncompressed size.
	uncompressedSize := width * height * 4
	if uncompressedSize > MaxImageBytes {
		return nil, fmt.Errorf("image uncompressed size exceeds limit: %d bytes (max %d bytes)",
			uncompressedSize, MaxImageBytes)
	}

	return img, nil
}
