# Image Rendering Design

## Overview

This document describes the design for rendering inline images in Markdeep mode. The system loads images from local files (and later URLs), converts them to Plan 9 bitmap format, and blits them to the frame with proper clipping.

## Current State

Images in markdown are parsed and rendered as placeholder text:
- `![alt text](path/to/image.png)` renders as `[Image: alt text]` in blue
- Style fields exist: `Image bool`, `ImageURL string`, `ImageAlt string`
- No actual image loading or rendering

## Architecture

### New File: `rich/image.go`

All image handling code lives in a single file for clarity and isolation.

```
rich/image.go
├── ImageCache        - LRU cache for loaded images
├── ImageLoader       - Loads and converts images
├── ImageBox          - Represents a positioned image in layout
└── Image rendering   - Blits images to frame with clipping
```

### Dependencies

```
rich/image.go
├── image            (Go standard library - PNG, JPEG, GIF decode)
├── image/png
├── image/jpeg
├── image/gif
├── net/http         (URL image fetching)
├── edwood/draw      (Plan 9 draw interface)
└── os               (file I/O)
```

## Data Flow

```
1. Markdown source
   │
   ▼
2. Parser detects ![alt](path) → Span with Image=true, ImageURL=path
   │
   ▼
3. contentToBoxes() creates ImageBox (new box type)
   │
   ▼
4. layout() positions ImageBox, determines display size
   │
   ▼
5. Frame.Redraw() → for each ImageBox:
   │  a. ImageLoader.Load(path) → returns cached or loads fresh
   │  b. Convert Go image.Image to Plan 9 Image via Load()
   │  c. Blit to screen with clipping
   │
   ▼
6. Display shows image
```

## Detailed Design

### 1. ImageCache

Caches loaded images to avoid repeated disk I/O and conversion.

```go
// ImageCache provides an LRU cache for loaded images.
type ImageCache struct {
    mu       sync.RWMutex
    images   map[string]*CachedImage
    order    []string      // LRU order (oldest first)
    maxSize  int           // Maximum number of cached images
    display  draw.Display  // For allocating Plan 9 images
}

// CachedImage represents a loaded and converted image.
type CachedImage struct {
    Original   image.Image    // Go stdlib image
    Plan9Image draw.Image     // Plan 9 draw.Image (allocated on display)
    Width      int            // Original width in pixels
    Height     int            // Original height in pixels
    Path       string         // Source path for debugging
    LoadTime   time.Time      // When loaded
    Err        error          // Error if load failed
}

func NewImageCache(display draw.Display, maxSize int) *ImageCache
func (c *ImageCache) Get(path string) (*CachedImage, bool)
func (c *ImageCache) Load(path string) (*CachedImage, error)
func (c *ImageCache) Clear()
func (c *ImageCache) Evict(path string)
```

### 2. ImageLoader

Handles loading images from files or URLs and converting to Plan 9 format.

```go
// LoadImage loads an image from a file path or URL.
// Supports PNG, JPEG, GIF (first frame only).
// URLs must use http:// or https:// scheme.
func LoadImage(path string) (image.Image, error) {
    if isImageURL(path) {
        return loadImageFromURL(path)
    }
    return loadImageFromFile(path)
}

// isImageURL returns true if the path is an HTTP/HTTPS URL.
func isImageURL(path string) bool {
    return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// loadImageFromFile loads an image from a local file path.
func loadImageFromFile(path string) (image.Image, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    // image.Decode auto-detects format
    img, _, err := image.Decode(f)
    return img, err
}

// loadImageFromURL fetches an image from an HTTP/HTTPS URL.
// Enforces timeout (10s), size limit (16MB), and content-type validation.
func loadImageFromURL(url string) (image.Image, error) {
    client := &http.Client{
        Timeout: 10 * time.Second,
    }

    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("network error: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("server returned %d", resp.StatusCode)
    }

    // Validate content-type
    contentType := resp.Header.Get("Content-Type")
    if !strings.HasPrefix(contentType, "image/") {
        return nil, fmt.Errorf("invalid content-type: %s", contentType)
    }

    // Enforce size limit (16MB)
    const maxSize = 16 * 1024 * 1024
    if resp.ContentLength > maxSize {
        return nil, fmt.Errorf("image too large: %d bytes", resp.ContentLength)
    }
    limitedReader := io.LimitReader(resp.Body, maxSize)

    img, _, err := image.Decode(limitedReader)
    return img, err
}

// ConvertToPlan9 converts a Go image.Image to a Plan 9 draw.Image.
// The resulting image is allocated on the provided display.
func ConvertToPlan9(display draw.Display, img image.Image) (draw.Image, error) {
    bounds := img.Bounds()
    width := bounds.Dx()
    height := bounds.Dy()

    // Allocate Plan 9 image with RGBA32 pixel format
    p9img, err := display.AllocImage(
        image.Rect(0, 0, width, height),
        draw.RGBA32,
        false,  // not replicated
        draw.Transparent,
    )
    if err != nil {
        return nil, err
    }

    // Convert pixels to RGBA32 format (R8G8B8A8, pre-multiplied alpha)
    data := make([]byte, width*height*4)
    i := 0
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            r, g, b, a := img.At(x, y).RGBA()
            // RGBA() returns 16-bit values, convert to 8-bit
            // Pre-multiply alpha
            alpha := uint8(a >> 8)
            if alpha == 0 {
                data[i], data[i+1], data[i+2], data[i+3] = 0, 0, 0, 0
            } else {
                data[i] = uint8(r >> 8)
                data[i+1] = uint8(g >> 8)
                data[i+2] = uint8(b >> 8)
                data[i+3] = alpha
            }
            i += 4
        }
    }

    // Load pixel data into Plan 9 image
    _, err = p9img.Load(image.Rect(0, 0, width, height), data)
    if err != nil {
        p9img.Free()
        return nil, err
    }

    return p9img, nil
}
```

### 3. Draw Interface Extensions

The current `draw.Image` interface needs `Load` method exposed:

```go
// In draw/interface.go, add to Image interface:
type Image interface {
    // ... existing methods ...
    Load(r image.Rectangle, data []byte) (int, error)
}

// Add implementation in imageImpl:
func (dst *imageImpl) Load(r image.Rectangle, data []byte) (int, error) {
    return dst.drawImage.Load(r, data)
}
```

Also need Pix constants exposed:

```go
// In draw/interface.go or draw/pix.go:
const (
    RGBA32 = draw.RGBA32
    RGB24  = draw.RGB24
    // ... etc
)
```

### 4. Image Box Type

Add a new box type for images:

```go
// In rich/layout.go, modify Box struct:
type Box struct {
    Text  []byte
    Nrune int
    Bc    rune
    Style Style
    Wid   int

    // Image-specific fields (only used when Style.Image is true)
    ImageData *CachedImage  // Loaded image data
}

// IsImage returns true if this box represents an image.
func (b *Box) IsImage() bool {
    return b.Style.Image && b.ImageData != nil
}
```

### 5. Layout Changes

Modify `contentToBoxes` to create image boxes:

```go
// In appendSpanBoxes, handle image spans:
if span.Style.Image {
    // Create an image box
    boxes = append(boxes, Box{
        Text:  nil,  // No text
        Nrune: 0,
        Bc:    0,
        Style: span.Style,
        // ImageData populated during layout when cache is available
    })
    return boxes
}
```

Modify `layout` to size images:

```go
// In layout(), handle image boxes:
if box.Style.Image {
    // Try to load image from cache
    if imageCache != nil {
        cached, err := imageCache.Load(box.Style.ImageURL)
        if err == nil {
            box.ImageData = cached
            // Scale image to fit frame width if needed
            imgWidth := cached.Width
            imgHeight := cached.Height
            if imgWidth > effectiveFrameWidth {
                scale := float64(effectiveFrameWidth) / float64(imgWidth)
                imgWidth = effectiveFrameWidth
                imgHeight = int(float64(imgHeight) * scale)
            }
            box.Wid = imgWidth
            // Image height affects line height
            lineHeight = max(lineHeight, imgHeight)
        } else {
            // Show placeholder on error
            box.Wid = fontWidth("[Image: " + box.Style.ImageAlt + "]")
        }
    }
}
```

### 6. Rendering

Add image rendering to `drawText`:

```go
// In frameImpl.drawText(), add image rendering phase:

// Phase 5: Images
for _, line := range f.lines {
    if line.Y >= frameHeight {
        break
    }
    if line.Y+line.Height > frameHeight {
        continue  // Skip partially visible lines (for now)
    }

    for _, pb := range line.Boxes {
        if !pb.Box.IsImage() {
            continue
        }

        img := pb.Box.ImageData
        if img == nil || img.Plan9Image == nil {
            continue
        }

        // Calculate destination rectangle
        dstRect := image.Rect(
            f.rect.Min.X + pb.X,
            f.rect.Min.Y + line.Y,
            f.rect.Min.X + pb.X + pb.Box.Wid,
            f.rect.Min.Y + line.Y + line.Height,
        )

        // Clip to frame bounds
        dstRect = dstRect.Intersect(f.rect)
        if dstRect.Empty() {
            continue
        }

        // Calculate source point (for partial images)
        srcPt := image.ZP
        if dstRect.Min.X > f.rect.Min.X + pb.X {
            srcPt.X = dstRect.Min.X - (f.rect.Min.X + pb.X)
        }
        if dstRect.Min.Y > f.rect.Min.Y + line.Y {
            srcPt.Y = dstRect.Min.Y - (f.rect.Min.Y + line.Y)
        }

        // Blit image
        screen.Draw(dstRect, img.Plan9Image, nil, srcPt)
    }
}
```

### 7. Image Scaling

For images larger than the frame width:

```go
// ScaleImage creates a scaled version of an image.
func ScaleImage(display draw.Display, src draw.Image, newWidth, newHeight int) (draw.Image, error) {
    // Allocate destination
    dst, err := display.AllocImage(
        image.Rect(0, 0, newWidth, newHeight),
        src.Pix(),
        false,
        draw.Transparent,
    )
    if err != nil {
        return nil, err
    }

    // Use draw with scaling
    // Note: Plan 9's draw doesn't natively support scaling,
    // so we need to do this in software or use nearest-neighbor
    // by drawing the source image stretched to the destination.

    // For now, use simple nearest-neighbor scaling via repeated draws
    // or implement proper bilinear scaling in software.

    // TODO: Implement proper scaling
    return dst, nil
}
```

**Alternative**: Scale in Go using `golang.org/x/image/draw` before converting to Plan 9:

```go
import "golang.org/x/image/draw"

func ScaleGoImage(src image.Image, newWidth, newHeight int) image.Image {
    dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
    draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
    return dst
}
```

## Image Clipping (Frame Boundaries)

The current implementation skips lines that extend past the frame bottom. For images, we need proper clipping:

### Option A: Skip Partial Images (Current Approach Extended)
- If image would extend past frame bottom, don't render it
- Simple but may hide important content

### Option B: Software Clipping
- When blitting, only copy pixels within the clip rectangle
- Requires modifying the source rectangle calculation
- Plan 9's `Draw` already clips to the destination rectangle

### Option C: Render to Scratch Buffer
- Render entire frame to off-screen buffer
- Composite only the visible portion to screen
- More memory, cleaner clipping

**Recommendation**: Plan 9's `Draw` operation naturally clips to the destination rectangle. By computing the proper `dstRect.Intersect(f.rect)` and adjusting `srcPt`, we get correct clipping without extra buffers.

## Error Handling

```go
// When image fails to load:
// 1. Cache the error to avoid repeated load attempts
// 2. Render placeholder text: "[Image: alt text]" or "[Image: load failed]"
// 3. Log error for debugging

type CachedImage struct {
    // ...
    Err error  // Non-nil if load failed
}

func (c *ImageCache) Load(path string) (*CachedImage, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if cached, ok := c.images[path]; ok {
        return cached, cached.Err
    }

    cached := &CachedImage{Path: path, LoadTime: time.Now()}

    img, err := LoadImage(path)
    if err != nil {
        cached.Err = fmt.Errorf("failed to load image %s: %w", path, err)
        c.images[path] = cached
        return cached, cached.Err
    }

    cached.Original = img
    cached.Width = img.Bounds().Dx()
    cached.Height = img.Bounds().Dy()

    p9img, err := ConvertToPlan9(c.display, img)
    if err != nil {
        cached.Err = fmt.Errorf("failed to convert image %s: %w", path, err)
        c.images[path] = cached
        return cached, cached.Err
    }

    cached.Plan9Image = p9img
    c.images[path] = cached
    c.evictOldest()

    return cached, nil
}
```

## Path Resolution

Image paths in markdown can be:
1. **Absolute paths**: `/home/user/images/photo.png`
2. **Relative paths**: `./images/photo.png`, `../assets/logo.png`
3. **URLs** (future): `https://example.com/image.png`

Path resolution:

```go
// ResolvePath resolves an image path relative to the markdown file.
func ResolvePath(imagePath string, markdownDir string) string {
    if filepath.IsAbs(imagePath) {
        return imagePath
    }
    return filepath.Join(markdownDir, imagePath)
}
```

The markdown file's directory is available from the window's file path.

## Memory Management

### Image Size Limits

```go
const (
    MaxImageWidth  = 4096  // Maximum width in pixels
    MaxImageHeight = 4096  // Maximum height in pixels
    MaxImageBytes  = 16 * 1024 * 1024  // 16MB uncompressed
)

func LoadImage(path string) (image.Image, error) {
    // ... load image ...

    bounds := img.Bounds()
    if bounds.Dx() > MaxImageWidth || bounds.Dy() > MaxImageHeight {
        return nil, fmt.Errorf("image too large: %dx%d (max %dx%d)",
            bounds.Dx(), bounds.Dy(), MaxImageWidth, MaxImageHeight)
    }

    if bounds.Dx() * bounds.Dy() * 4 > MaxImageBytes {
        return nil, fmt.Errorf("image uncompressed size exceeds limit")
    }

    return img, nil
}
```

### Cache Eviction

```go
const DefaultCacheSize = 50  // Maximum cached images

func (c *ImageCache) evictOldest() {
    if len(c.order) <= c.maxSize {
        return
    }

    // Evict oldest entries
    for len(c.order) > c.maxSize {
        oldest := c.order[0]
        c.order = c.order[1:]

        if cached, ok := c.images[oldest]; ok {
            if cached.Plan9Image != nil {
                cached.Plan9Image.Free()
            }
            delete(c.images, oldest)
        }
    }
}
```

### Cleanup on Mode Exit

When exiting Markdeep mode:

```go
func (w *Window) exitPreviewMode() {
    if w.imageCache != nil {
        w.imageCache.Clear()
        w.imageCache = nil
    }
    // ... rest of cleanup
}
```

## Testing Strategy

### Unit Tests (rich/image_test.go)

1. **LoadImage tests**
   - Load valid PNG, JPEG, GIF from file
   - Handle missing file
   - Handle corrupt file
   - Handle non-image file
   - Respect size limits
   - Load image from HTTP URL
   - Load image from HTTPS URL
   - Handle URL timeout
   - Handle URL 404
   - Handle URL with invalid content-type
   - Handle URL with oversized response
   - Reject non-http(s) URL schemes

2. **ConvertToPlan9 tests**
   - Convert RGBA image
   - Convert RGB image (no alpha)
   - Convert grayscale image
   - Handle edge cases (1x1, very wide, very tall)

3. **ImageCache tests**
   - Cache hit returns same object
   - Cache miss triggers load
   - LRU eviction works
   - Error caching works
   - Clear frees all images

4. **Path resolution tests**
   - Absolute paths unchanged
   - Relative paths resolved
   - Handle .. traversal

### Integration Tests

1. **Layout tests**
   - Image box sized correctly
   - Image wider than frame scaled
   - Line height includes image height

2. **Render tests**
   - Image appears at correct position
   - Image clipped at frame boundary
   - Placeholder shown on load failure

### Manual Tests

1. Open README.md in Markdeep mode
2. Verify badges render (may fail if URLs, but test local images)
3. Create test file with local images
4. Resize window, verify clipping
5. Scroll, verify images appear/disappear correctly

## Implementation Phases

### Phase 16A: Draw Interface Extensions
- Add `Load` method to `draw.Image` interface
- Add `Pix` constants (RGBA32, etc.)

### Phase 16B: Image Loading
- Implement `LoadImage` function
- Support PNG, JPEG, GIF
- Add size limits

### Phase 16C: Plan 9 Conversion
- Implement `ConvertToPlan9` function
- Handle various source formats
- Handle alpha correctly

### Phase 16D: Image Cache
- Implement `ImageCache` struct
- LRU eviction
- Error caching

### Phase 16E: Layout Integration
- Modify `Box` to hold `ImageData`
- Modify `contentToBoxes` for images
- Modify `layout` to size images

### Phase 16F: Frame Rendering
- Add image rendering phase to `drawText`
- Implement clipping
- Handle placeholder on error

### Phase 16G: Window Integration
- Add `ImageCache` to Window
- Initialize cache on Markdeep entry
- Clear cache on Markdeep exit
- Path resolution

### Phase 16H: Testing
- Unit tests for all components
- Integration tests
- Manual verification

### Phase 16J: URL Image Support
- URL detection (http://, https://)
- HTTP fetching with timeout (10s)
- Size limit enforcement (16MB)
- Content-Type validation
- Integration with LoadImage
- Error handling for network failures

## Security Considerations

### URL Image Loading

When loading images from URLs, the following security measures are implemented:

1. **Scheme restriction**: Only `http://` and `https://` URLs are allowed. Other schemes like `file://`, `data:`, `javascript:`, etc. are rejected.

2. **Timeout**: HTTP requests timeout after 10 seconds to prevent hanging on slow/unresponsive servers.

3. **Size limit**: Responses larger than 16MB are rejected to prevent memory exhaustion attacks.

4. **Content-Type validation**: Only responses with `image/*` Content-Type headers are accepted.

5. **No redirects to file://**: The HTTP client does not follow redirects to non-http(s) schemes.

6. **No authentication**: URLs requiring authentication will fail (no credential handling).

### Caching Considerations

- URL images are cached by their full URL string
- Cache does not persist across sessions
- No disk caching of remote images
- Cache is cleared when exiting Markdeep mode

## Future Enhancements

1. ~~**URL support**: Fetch images from HTTP/HTTPS URLs~~ (Implemented in Phase 16J)
2. **Animated GIFs**: Support animation (complex)
3. **Better scaling**: Bilinear/bicubic scaling
4. **Lazy loading**: Load images as they scroll into view
5. **Progressive loading**: Show placeholder while loading
6. **Image formats**: WebP, SVG (rasterized)

## Files Changed

| File | Changes |
|------|---------|
| `rich/image.go` | New file - all image handling |
| `rich/image_test.go` | New file - tests |
| `draw/interface.go` | Add Load method, Pix constants |
| `rich/layout.go` | Box.ImageData field, layout changes |
| `rich/frame.go` | Image rendering phase |
| `wind.go` | ImageCache field, init/cleanup |
| `exec.go` | Pass image cache to frame |
