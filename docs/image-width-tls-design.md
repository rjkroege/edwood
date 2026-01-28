# Image Width Tag and TLS Retry (Phase 24)

## Overview

Two related improvements to image handling in Markdeep preview:

1. **Width tag**: Support `width=Npx` in the image title position to explicitly size images.
2. **TLS retry**: Retry HTTPS image fetches with relaxed TLS settings when strict TLS fails.

## Width Tag Syntax

Standard markdown image syntax: `![alt](url "title")`

We extend the title position to accept a width directive:

```markdown
![Glenda](https://9p.io/plan9/img/plan9bunnyblack.jpg "width=200px")
![Robot](robot.jpg "width=400px")
![Small](logo.png "width=50px")
```

Only `px` units are supported. No `%`, `em`, `auto`, etc. The height is computed automatically to preserve aspect ratio.

If the width tag is absent (or 0), the image renders at its natural size, clamped to the frame width as today.

If the specified width exceeds the frame width, the image is clamped to the frame width (same as natural-size behavior).

## Current State

### Image parsing (`markdown/parse.go`)

The parser detects `![alt](url)` and `![alt](url "title")`. The `parseURLPart()` function (line 1182) extracts the URL and discards the title string entirely. The URL is stored in `Style.ImageURL`, the alt text in `Style.ImageAlt`.

### Image layout (`rich/layout.go`)

`imageBoxDimensions()` (line 167) computes display size:
- If image width <= frame width, use natural dimensions
- If wider, scale down proportionally to fit frame width

There is no mechanism for an explicit target width.

### Image rendering (`rich/frame.go`)

`drawImageTo()` (line 1489) blits the image to the scratch buffer. When the image needs scaling (display size != original size), it currently draws at original size and clips. The comment at line 1553 notes this limitation. Plan 9's `Draw()` does not scale; pixel data must be pre-scaled in software.

### HTTP image loading (`rich/image.go`)

`loadImageFromURL()` (line 37) uses a plain `http.Client` with a 10-second timeout. No custom `Transport`, no TLS configuration. Servers with legacy TLS (like `9p.io`) cause `tls: handshake failure` errors.

## Design

### 1. TLS Retry (`rich/image.go`)

Modify `loadImageFromURL()`:

```go
func loadImageFromURL(url string) (image.Image, error) {
    client := &http.Client{Timeout: URLImageTimeout}

    resp, err := client.Get(url)
    if err != nil && isTLSError(err) {
        // Retry with relaxed TLS settings
        tlsTransport := &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
        insecureClient := &http.Client{
            Timeout:   URLImageTimeout,
            Transport: tlsTransport,
        }
        resp, err = insecureClient.Get(url)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to fetch image: %w", err)
    }
    defer resp.Body.Close()
    // ... rest unchanged
}
```

`isTLSError()` checks if the error string contains `"tls:"` or `"certificate"`. This covers `tls: handshake failure`, `x509: certificate` errors, etc.

### 2. ImageWidth Field (`rich/style.go`)

Add to the `Style` struct:

```go
ImageWidth int  // Explicit width in pixels (0 = use natural size)
```

This field is only meaningful when `Image == true`.

### 3. Width Parsing (`markdown/parse.go`)

Change `parseURLPart()` to return both URL and title:

```go
func parseURLPart(urlPart string) (url, title string)
```

Add a new helper:

```go
func parseImageWidth(title string) int
```

This scans the title string for `width=Npx` using a simple regex or string scan. Returns 0 if not found or invalid.

In the image span creation code (lines ~389 and ~656), extract the width from the title and set `imageStyle.ImageWidth`.

### 4. Layout with Explicit Width (`rich/layout.go`)

Update `imageBoxDimensions()`:

```go
func imageBoxDimensions(box *Box, maxWidth int) (width, height int) {
    if !box.IsImage() || box.ImageData == nil {
        return 0, 0
    }

    imgWidth := box.ImageData.Width
    imgHeight := box.ImageData.Height

    targetWidth := imgWidth
    if box.Style.ImageWidth > 0 {
        targetWidth = box.Style.ImageWidth
    }

    // Clamp to frame width
    if targetWidth > maxWidth {
        targetWidth = maxWidth
    }

    // Scale height proportionally
    if targetWidth == imgWidth {
        return imgWidth, imgHeight
    }
    scale := float64(targetWidth) / float64(imgWidth)
    return targetWidth, int(float64(imgHeight) * scale)
}
```

### 5. Image Pre-Scaling (`rich/frame.go`)

In `drawImageTo()`, when `scaledWidth != cached.Width || scaledHeight != cached.Height`, pre-scale the Go image before converting to Plan 9 format:

```go
import "golang.org/x/image/draw"

// Scale the image in Go-land before converting to Plan 9 format
scaled := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
draw.BiLinear.Scale(scaled, scaled.Bounds(), cached.Original, cached.Original.Bounds(), draw.Src, nil)

// Convert scaled image to Plan 9 pixel data
data, err := ConvertToPlan9(scaled)
// ... allocate Plan 9 image with scaledWidth x scaledHeight, load data, blit
```

This replaces the current "draw at original size, clip" workaround.

**Dependency**: `golang.org/x/image/draw` for `BiLinear.Scale()`. Check if already in `go.mod`; if not, use `image/draw` with `draw.ApproxBiLinear` or a manual nearest-neighbor scaler to avoid adding a dependency.

## Files Modified

| File | Change |
|------|--------|
| `rich/image.go` | TLS retry in `loadImageFromURL()`, add `isTLSError()` |
| `rich/style.go` | Add `ImageWidth int` field |
| `markdown/parse.go` | Change `parseURLPart()` signature, add `parseImageWidth()`, wire into image spans |
| `rich/layout.go` | Update `imageBoxDimensions()` for explicit width |
| `rich/frame.go` | Pre-scale images in `drawImageTo()` |
