# Async Image Loading Design

## Goal

Render alt text placeholders immediately while images load in the background,
then re-layout and render once images arrive. This eliminates the multi-second
blocking stall on first preview entry when remote images (e.g.
`http://9p.io/plan9/img/plan9bunnyblack.jpg`) need to be fetched.

## Current Behavior

- `ImageCache.Load()` blocks synchronously on HTTP fetch (2+ seconds for remote images)
- Layout in `layoutWithCacheAndBasePath` calls `Load()` for every image box
- The entire Render call stalls until all images are loaded
- User sees nothing until everything is ready

## Proposed Behavior

1. Preview renders immediately with alt text placeholders (`[Image: Glenda]`)
   shown in blue where images will appear
2. Images are fetched in background goroutines
3. When each image arrives, a re-render is triggered automatically
4. On re-render the cache hits instantly, layout includes the image dimensions,
   and the image replaces the placeholder

## Design

### 1. Add `LoadAsync` to ImageCache (`rich/image.go`)

Add a callback field and an async load method to the existing `ImageCache`:

```go
type ImageCache struct {
    // ... existing fields (mu, images, order, maxSize) ...
    onLoaded func(path string)  // called when a background load completes
}

func (c *ImageCache) SetOnLoaded(fn func(path string))

// LoadAsync returns the cached entry immediately if available (cache hit).
// If not cached, starts a background goroutine to fetch it and returns nil.
// When the background load completes, calls onLoaded(path).
func (c *ImageCache) LoadAsync(path string) (*CachedImage, error)
```

**`LoadAsync` logic:**

1. Lock, check cache map
2. If found and not loading: return it immediately (cache hit or cached error)
3. If found and loading: return `nil, nil` (fetch already in progress)
4. If not found: insert sentinel `CachedImage{Loading: true}`, launch
   `go c.backgroundLoad(path)`, return `nil, nil`

**`backgroundLoad(path)` logic:**

1. Call `LoadImage(path)` (existing function, handles HTTP URLs and local files)
2. Call `ConvertToPlan9()` to convert to Plan 9 pixel format
3. Lock, store result in cache map (replacing the Loading sentinel)
4. Unlock, then call `c.onLoaded(path)` if set

**New field on `CachedImage`:**

```go
type CachedImage struct {
    // ... existing fields ...
    Loading bool  // true while background fetch is in progress
}
```

This distinguishes three states:
- Not in cache: never requested
- In cache with `Loading: true`: fetch in progress
- In cache with `Loading: false`: ready (with data or with error)

### 2. Use `LoadAsync` in layout (`rich/layout.go`)

In `layoutWithCacheAndBasePath`, replace `cache.Load(imgPath)` with
`cache.LoadAsync(imgPath)`:

```go
// Before:
cached, _ := cache.Load(imgPath)
if cached != nil {
    box.ImageData = cached
}

// After:
cached, _ := cache.LoadAsync(imgPath)
if cached != nil {
    box.ImageData = cached
}
```

When `LoadAsync` returns nil (image still loading), `box.ImageData` remains nil.
The existing rendering code in `frame.go` already handles this case:

- `imageBoxDimensions()` returns (0, 0) when `ImageData` is nil
- `drawImageErrorPlaceholder()` renders `[Image: alttext]` in blue text
- The box occupies zero space in layout (content flows as if no image)

When the image arrives and re-render triggers, `LoadAsync` returns the cached
image instantly, `ImageData` gets populated, layout reserves the correct
dimensions, and `drawImageTo()` renders the actual image.

### 3. Wire callback to re-render (`exec.go`)

In `previewcmd`, after creating/reusing the image cache, set the `onLoaded`
callback to trigger a preview re-render:

```go
w.imageCache.SetOnLoaded(func(path string) {
    // Image loaded in background -- re-render preview
    w.richBody.Render(w.body.all)
    if w.display != nil {
        w.display.Flush()
    }
})
```

On re-render, `LoadAsync` hits the cache immediately, layout includes the
image, and the placeholder is replaced with the actual image.

## Files to Modify

| File | Change |
|------|--------|
| `rich/image.go` | Add `Loading` field to `CachedImage`; add `onLoaded` callback field, `SetOnLoaded` method, `LoadAsync` method, and `backgroundLoad` helper to `ImageCache` |
| `rich/layout.go` | Change `cache.Load()` to `cache.LoadAsync()` in `layoutWithCacheAndBasePath` |
| `exec.go` | Set `onLoaded` callback on the image cache in `previewcmd` |

## Edge Cases

- **Duplicate requests**: The `Loading` sentinel in the cache map prevents
  launching multiple goroutines for the same URL. `LoadAsync` checks for it
  and returns nil without spawning another goroutine.

- **Thread safety**: `backgroundLoad` acquires the mutex before writing to the
  cache map, same locking pattern as the existing `Load` method. The
  `onLoaded` callback is called after releasing the lock to avoid deadlocks.

- **Error images**: Failed loads are cached with their error (same as current
  `Load` behavior). No retry is attempted on subsequent calls.

- **Cache cleared while loading**: If the cache is cleared (e.g. window closed)
  while a background load is in flight, `backgroundLoad` should check that the
  cache entry still exists before storing results. If the entry was evicted,
  store the result anyway (it will just be a fresh entry).

- **Callback from goroutine**: The `onLoaded` callback runs on a background
  goroutine. The `Render` + `Flush` calls must be safe to call from a non-main
  goroutine, which matches the existing `SchedulePreviewUpdate` pattern where
  `time.AfterFunc` also calls `UpdatePreview` from a timer goroutine.

## Open Questions

None -- the design is intentionally minimal, touching only the cache and its
single call site in layout. The existing placeholder rendering and re-render
machinery handle everything else.
