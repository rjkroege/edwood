# Async Image Loading Design

## Problem

Image loading (including HTTP fetches) happens synchronously during layout
in `layoutWithCacheAndBasePath()` (`rich/layout.go:766`). When the layout
engine encounters an image box, it calls `cache.Load(imgPath)` which, on a
cache miss, calls `LoadImage()` → `loadImageFromURL()` with a 10-second
timeout. This blocks the main goroutine, freezing the entire UI until
every image is fetched and decoded.

The design doc (Known Issue #5) calls this out: "Image loading (including
HTTP fetches) appears to happen synchronously during parsing/rendering."

## Current Architecture

### Synchronous loading path

```
UpdatePreview() / Redraw()
  → richBody.Render(body.all)
    → frame.Redraw()
      → drawTextTo()
        → layoutFromOrigin()
          → layoutBoxes()
            → layoutWithCacheAndBasePath()
              → for each image box:
                  cache.Load(imgPath)        ← BLOCKS here on cache miss
                    → LoadImage(path)
                      → loadImageFromURL()   ← up to 10s per image
                  box.ImageData = cached
              → layout(boxes, ...)           ← sizes images from ImageData
```

### Key data structures

- **`ImageCache`** (`rich/image.go:248`): LRU cache keyed by path string.
  Thread-safe via `sync.RWMutex`. Stores `*CachedImage` entries including
  error entries to avoid re-fetching failures.

- **`CachedImage`** (`rich/image.go:237`): Contains `Original image.Image`,
  `Data []byte` (Plan 9 RGBA32), `Width`, `Height`, `Err error`.

- **`Box.ImageData`** (`rich/box.go:18`): `*CachedImage` pointer set during
  layout. Used by `drawImageTo()` and `imageBoxDimensions()` during rendering.

- **`frameImpl.imageCache`** (`rich/frame.go:119`): Pointer to the shared
  cache, set via `WithImageCache()` option.

### Where images are consumed

1. **Layout** (`layout.go:427-430`): `imageBoxDimensions()` reads
   `box.ImageData.Width/Height` to compute line height and box width.

2. **Rendering** (`frame.go:1028-1065`, Phase 5): `drawImageTo()` reads
   `pb.Box.ImageData.Original` and `pb.Box.ImageData.Data` to scale and
   blit the image.

3. **Error placeholder** (`frame.go:2177`): `drawImageErrorPlaceholder()`
   renders blue alt text when `ImageData.Err != nil` or `ImageData` is nil.

## Design

### Overview

Move image loading off the main thread. On cache miss, immediately return a
**placeholder** `CachedImage` that the layout engine can size (text-height
gray box with alt text). Launch a background goroutine to fetch and decode
the image. When loading completes, store the result in the cache and invoke
a callback that triggers a preview re-render from the main goroutine.

Cache hits remain synchronous — no goroutine, no placeholder, no flicker.

### Placeholder rendering

A new `CachedImage` state indicates "loading in progress":

```go
// In CachedImage, add a field:
type CachedImage struct {
    Original  image.Image
    Data      []byte
    Width     int
    Height    int
    Path      string
    LoadTime  time.Time
    Err       error
    Loading   bool    // true while async load is in progress
}
```

When `Loading` is true and `Original` is nil, the layout engine treats the
image as a text-height placeholder. The rendering path
(`drawImageTo` / Phase 5 in `drawTextTo`) already handles the case where
`ImageData` is non-nil but `IsImage()` returns false (no Original/Data) —
it calls `drawImageErrorPlaceholder()`. We extend this to show a
"Loading..." placeholder instead of an error when `Loading` is true.

**Placeholder appearance**: The existing `drawImageErrorPlaceholder()`
renders the box text (`[Image: alt]`) in blue. For the loading state, we
render `[Loading: alt]` in a muted gray color to distinguish it from errors
and clickable links. The placeholder occupies one line of text at the
default font height.

### Async loading API

Add a new method to `ImageCache`:

```go
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
func (c *ImageCache) LoadAsync(path string, onLoaded func(path string)) (*CachedImage, error)
```

**Concurrency limit**: A semaphore channel (`chan struct{}`) limits
concurrent image downloads. Default: 4. This prevents thundering herd on
documents with many images and bounds goroutine/connection count.

```go
type ImageCache struct {
    mu          sync.RWMutex
    images      map[string]*CachedImage
    order       []string
    maxSize     int
    maxParallel int             // max concurrent loads (default 4)
    sem         chan struct{}    // semaphore for concurrent downloads
}
```

The semaphore is lazily initialized on first `LoadAsync` call.

### Integration with layout

`layoutWithCacheAndBasePath()` currently calls `cache.Load()`. Change it
to call `cache.LoadAsync()` with a nil callback (layout doesn't need a
callback — the frame's `onImageLoaded` handles re-render). The returned
`*CachedImage` may be a loading placeholder, which `imageBoxDimensions()`
handles by returning text-height dimensions.

```go
// In layoutWithCacheAndBasePath, change:
//   cached, _ := cache.Load(imgPath)
// to:
//   cached, _ := cache.LoadAsync(imgPath, nil)
```

The `onLoaded` callback is instead registered at the frame level (see
below).

### Re-render callback

When a background image load completes, the preview must re-render to
replace the placeholder with the actual image. This uses the same
goroutine-marshaling pattern as `SchedulePreviewUpdate()`: acquire the
row lock and call `UpdatePreview()`.

Add a callback field to `frameImpl`:

```go
type frameImpl struct {
    // ...existing fields...
    onImageLoaded func(path string) // called when an async image finishes loading
}

func WithOnImageLoaded(fn func(path string)) Option {
    return func(f *frameImpl) {
        f.onImageLoaded = fn
    }
}
```

In `layoutWithCacheAndBasePath()`, when `LoadAsync` is called on a cache
miss, pass a callback that invokes `f.onImageLoaded`:

```go
// When loading images in layoutWithCacheAndBasePath:
callback := func(path string) {
    if onImageLoaded != nil {
        onImageLoaded(path)
    }
}
cached, _ := cache.LoadAsync(imgPath, callback)
```

The `onImageLoaded` function signature needs to be passed into
`layoutWithCacheAndBasePath`. Since the layout functions are called from
`frameImpl.layoutBoxes()`, which already has access to `f`, we add the
callback parameter there.

At the window level (`previewcmd` in `exec.go`), wire the callback:

```go
rtOpts = append(rtOpts, WithOnImageLoaded(func(path string) {
    // Marshal to main goroutine via row lock (same as SchedulePreviewUpdate)
    go func() {
        global.row.lk.Lock()
        defer global.row.lk.Unlock()
        if !w.previewMode || w.richBody == nil {
            return
        }
        // Re-render (not full re-parse — content hasn't changed, only image data)
        w.richBody.Render(w.body.all)
        if w.display != nil {
            w.display.Flush()
        }
    }()
}))
```

Note: This re-render is lightweight — it doesn't re-parse the markdown.
The image is already in the cache, so `layoutWithCacheAndBasePath` will
get a cache hit on the next layout pass, `imageBoxDimensions` will return
the real dimensions, and `drawImageTo` will render the actual image.

### Cancellation

When preview mode is exited (`previewcmd` exit path), `imageCache.Clear()`
is already called. Extend `Clear()` to also cancel pending loads:

```go
type ImageCache struct {
    // ...existing fields...
    cancelFuncs map[string]context.CancelFunc // per-path cancellation
}

func (c *ImageCache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Cancel all pending async loads
    for _, cancel := range c.cancelFuncs {
        cancel()
    }
    c.cancelFuncs = make(map[string]context.CancelFunc)

    c.images = make(map[string]*CachedImage)
    c.order = make([]string, 0)
}
```

Each `LoadAsync` goroutine uses a `context.WithCancel` derived from a
parent context. When cancelled, HTTP requests are aborted via
`req.WithContext(ctx)`, and file reads check `ctx.Err()`. The goroutine
checks the context before updating the cache and before calling the
callback — if cancelled, it silently exits without modifying cache state.

### Error handling

Errors are cached exactly as today — a `CachedImage` with `Err` set and
`Loading=false`. The `drawImageErrorPlaceholder` renders the error. Error
entries remain in the LRU cache and prevent repeated fetch attempts for
the same broken URL.

HTTP-specific transient errors (timeouts, 5xx) could optionally be
retried, but for simplicity the initial implementation treats all errors
as permanent within a cache lifetime. Users can exit and re-enter preview
mode to retry (which calls `Clear()`).

### Thread safety

- **`ImageCache` mutex**: Already exists (`sync.RWMutex`). The `LoadAsync`
  method holds the write lock briefly to check cache / insert placeholder,
  then releases it before starting the goroutine. The goroutine re-acquires
  the write lock to update the entry on completion.

- **`CachedImage` fields**: The `Loading` flag and the `Original`/`Data`/
  `Width`/`Height`/`Err` fields are written exactly once (by the loading
  goroutine under the cache lock). After that, they are read-only. The
  layout and render paths always read under at least `RLock` (via
  `cache.Get()`) or operate on a local copy obtained from `LoadAsync()`.

- **Callback goroutine safety**: The `onLoaded` callback may run
  concurrently with the main goroutine. It must acquire `global.row.lk`
  before touching any window/frame state, matching the existing
  `SchedulePreviewUpdate` pattern.

### Max concurrent downloads

```go
const DefaultMaxParallelLoads = 4
```

The semaphore channel is sized to `maxParallel`:

```go
func (c *ImageCache) LoadAsync(path string, onLoaded func(string)) (*CachedImage, error) {
    c.mu.Lock()

    // Check cache (hit or already-loading)
    if cached, ok := c.images[path]; ok {
        c.mu.Unlock()
        return cached, cached.Err
    }

    // Create placeholder
    placeholder := &CachedImage{
        Path:     path,
        LoadTime: time.Now(),
        Loading:  true,
    }
    c.images[path] = placeholder
    c.order = append(c.order, path)
    c.evictOldest()

    // Create cancellation context
    ctx, cancel := context.WithCancel(context.Background())
    if c.cancelFuncs == nil {
        c.cancelFuncs = make(map[string]context.CancelFunc)
    }
    c.cancelFuncs[path] = cancel

    // Lazily init semaphore
    if c.sem == nil {
        max := c.maxParallel
        if max <= 0 {
            max = DefaultMaxParallelLoads
        }
        c.sem = make(chan struct{}, max)
    }
    sem := c.sem

    c.mu.Unlock()

    // Launch background load
    go func() {
        // Acquire semaphore slot
        select {
        case sem <- struct{}{}:
            defer func() { <-sem }()
        case <-ctx.Done():
            return
        }

        // Load the image (this is the slow part)
        img, err := LoadImageWithContext(ctx, path)

        // Update cache under lock
        c.mu.Lock()
        delete(c.cancelFuncs, path)

        if ctx.Err() != nil {
            // Cancelled — remove the placeholder so a future LoadAsync retries
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

        // Notify caller (outside lock)
        if onLoaded != nil {
            onLoaded(path)
        }
    }()

    return placeholder, nil
}
```

### Context-aware image loading

Add a context-aware variant of `LoadImage`:

```go
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
    // that mid-read cancellation isn't critical)
    if ctx.Err() != nil {
        return nil, ctx.Err()
    }
    return loadImageFromFile(path)
}
```

For URL loading, the context is passed to `http.NewRequestWithContext()`:

```go
func loadImageFromURLWithContext(ctx context.Context, url string) (image.Image, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    client := &http.Client{Timeout: URLImageTimeout}
    resp, err := client.Do(req)
    // ... same validation and decoding as loadImageFromURL ...
}
```

### Changes to existing code

1. **`rich/image.go`**:
   - Add `Loading bool` to `CachedImage`
   - Add `cancelFuncs`, `sem`, `maxParallel` to `ImageCache`
   - Add `LoadAsync()` method
   - Add `LoadImageWithContext()`, `loadImageFromURLWithContext()`
   - Modify `Clear()` to cancel pending loads
   - Add `NewImageCacheWithOptions()` or option parameters for `maxParallel`

2. **`rich/layout.go`**:
   - Change `layoutWithCacheAndBasePath()` to accept an optional
     `onImageLoaded func(string)` callback parameter
   - Change `cache.Load()` to `cache.LoadAsync()` with the callback

3. **`rich/frame.go`**:
   - Add `onImageLoaded func(string)` field to `frameImpl`
   - Add `WithOnImageLoaded()` option
   - Modify `layoutBoxes()` to pass `onImageLoaded` to layout functions
   - In `drawTextTo` Phase 5, distinguish loading placeholder from error

4. **`rich/box.go`**:
   - No changes needed. `IsImage()` already returns false when
     `ImageData` is nil or has no `Original`, so loading placeholders
     naturally fall through to `drawImageErrorPlaceholder`.

5. **`exec.go`**:
   - Wire `WithOnImageLoaded` callback in `previewcmd()` to trigger
     re-render via row lock

6. **`richtext.go`**:
   - Forward `WithOnImageLoaded` option to frame (like other options)

### Interaction with ImageCache

| State | `Loading` | `Original` | `Err` | Behavior |
|-------|-----------|------------|-------|----------|
| Cache hit (success) | `false` | non-nil | `nil` | Sync return, render image |
| Cache hit (error) | `false` | `nil` | non-nil | Sync return, render error placeholder |
| Cache miss → async | `true` | `nil` | `nil` | Return placeholder, start goroutine |
| Async complete (success) | `false` | non-nil | `nil` | Callback triggers re-render |
| Async complete (error) | `false` | `nil` | non-nil | Callback triggers re-render with error |
| Cancelled | Entry removed | — | — | No callback, no cache pollution |

### Test plan

1. **Cache hit returns immediately**: Pre-populate cache, call `LoadAsync`,
   verify no goroutine launched and `Loading=false`.

2. **Cache miss triggers async load**: Call `LoadAsync` for uncached local
   file, verify placeholder returned with `Loading=true`, wait for callback,
   verify cache updated with real image.

3. **Placeholder shown during load**: Create frame with async-loading image,
   verify layout produces text-height line (not image-height).

4. **Callback invoked on completion**: Use a channel to synchronize, verify
   `onLoaded` called with correct path.

5. **Cancellation prevents callback**: Call `LoadAsync`, then `Clear()`,
   verify callback is never invoked.

6. **Error caching**: Use non-existent path, verify `Err` set after async
   completion, verify subsequent `LoadAsync` returns cached error (no
   re-fetch).

7. **Concurrent download limit**: Load 10 images simultaneously, verify
   only `DefaultMaxParallelLoads` goroutines are actively downloading
   (via semaphore occupancy tracking or timing).

8. **Race detector**: All tests run with `go test -race` to catch data
   races in cache updates and callback invocation.

9. **HTTP cancellation**: Use `httptest.Server` with a deliberate delay,
   call `LoadAsync` then `Clear()`, verify the HTTP request is cancelled
   (server sees connection close).
