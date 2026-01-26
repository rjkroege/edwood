# Single Rectangle Owner - Implementation Plan

## Overview

This document details the implementation of Option C from `preview-resize-design.md`: making `body Text` the single owner of geometry, with `RichText` becoming a stateless renderer that draws into whatever rectangle it's given.

## Design Principles

1. **Single source of truth**: `body.all` is the canonical rectangle for the body area
2. **Separation of concerns**: `Text` owns geometry, `RichText` handles rendering
3. **No duplicate state**: `RichText` computes scrollbar/frame rectangles at render time
4. **Minimal resize changes**: `Window.Resize()` always updates `body.all`, no special preview checks needed

## Current Architecture (Problem)

```
Window
├── body Text         <- owns body.all rectangle
├── previewMode bool
└── richBody *RichText <- owns separate all rectangle (gets stale on resize!)

Resize flow:
    Window.Resize()
        └── body.Resize(r1)   <- Updates body.all
        └── body.ScrDraw()    <- Redraws body
        // richBody.all is NEVER updated!
```

When `previewMode=true`, the window renders `richBody` but resize only updates `body.all`. Result: preview disappears, input focus is confused.

## Target Architecture (Solution)

```
Window
├── body Text         <- ALWAYS owns geometry (body.all)
├── previewMode bool
└── richRenderer *RichText <- stateless renderer, no rectangle storage

Resize flow:
    Window.Resize()
        └── body.Resize(r1, ..., noredraw=previewMode)  <- Always updates body.all
        └── if previewMode:
                richRenderer.Render(body.all)
            else:
                body.ScrDraw()

Draw flow:
    Window.Draw()
        └── if previewMode:
                richRenderer.Render(body.all)  <- Uses body's rectangle
            else:
                body.Redraw()
```

## Implementation Steps

### Step 1: Add SetRect() to rich.Frame Interface

Add ability to update frame geometry after initialization.

**File: rich/frame.go**

```go
// In Frame interface, add:
SetRect(r image.Rectangle)

// Implementation in frameImpl:
func (f *frameImpl) SetRect(r image.Rectangle) {
    if f.rect.Eq(r) {
        return // No change
    }
    f.rect = r
    // Layout will be recomputed on next Redraw() since it uses f.rect.Dx()
}
```

### Step 2: Make RichText Rectangle-Agnostic

Restructure `RichText` to compute rectangles from a passed parameter rather than storing them.

**File: richtext.go**

Current fields to change:
```go
type RichText struct {
    all        image.Rectangle  // REMOVE - will be passed at render time
    scrollRect image.Rectangle  // REMOVE - computed at render time
    display    draw.Display
    frame      rich.Frame
    content    rich.Content
    // ... rest stays
}
```

New approach:
```go
type RichText struct {
    display    draw.Display
    frame      rich.Frame
    content    rich.Content

    // Cached for hit-testing between renders
    lastRect   image.Rectangle  // Last rectangle we rendered into
    lastScrollRect image.Rectangle

    // ... styling fields unchanged
}
```

### Step 3: Add Render() Method to RichText

New primary entry point that accepts a rectangle.

**File: richtext.go**

```go
// Render draws the rich text component into the given rectangle.
// This computes scrollbar and frame areas from r at render time.
func (rt *RichText) Render(r image.Rectangle) {
    rt.lastRect = r

    // Compute scrollbar rectangle (left side)
    scrollWid := rt.display.ScaleSize(Scrollwid)
    scrollGap := rt.display.ScaleSize(Scrollgap)

    rt.lastScrollRect = image.Rect(
        r.Min.X,
        r.Min.Y,
        r.Min.X+scrollWid,
        r.Max.Y,
    )

    // Compute frame rectangle (right of scrollbar)
    frameRect := image.Rect(
        r.Min.X+scrollWid+scrollGap,
        r.Min.Y,
        r.Max.X,
        r.Max.Y,
    )

    // Update frame geometry if changed
    if rt.frame.Rect() != frameRect {
        rt.frame.SetRect(frameRect)
    }

    // Draw scrollbar
    rt.scrDrawAt(rt.lastScrollRect)

    // Draw frame content
    rt.frame.Redraw()
}
```

### Step 4: Update Scrollbar Methods

Scrollbar methods need to use cached or passed rectangles.

**File: richtext.go**

```go
// scrDrawAt renders the scrollbar at the given rectangle.
func (rt *RichText) scrDrawAt(scrollRect image.Rectangle) {
    if rt.display == nil {
        return
    }

    screen := rt.display.ScreenImage()

    // Draw scrollbar background
    if rt.scrollBg != nil {
        screen.Draw(scrollRect, rt.scrollBg, rt.scrollBg, image.ZP)
    }

    // Draw scrollbar thumb
    if rt.scrollThumb != nil {
        thumbRect := rt.scrThumbRectAt(scrollRect)
        screen.Draw(thumbRect, rt.scrollThumb, rt.scrollThumb, image.ZP)
    }
}

// scrThumbRectAt computes thumb position for a given scrollbar rectangle.
func (rt *RichText) scrThumbRectAt(scrollRect image.Rectangle) image.Rectangle {
    // ... same logic as scrThumbRect() but uses scrollRect parameter
}
```

### Step 5: Update Hit-Testing Methods

Methods that check if a point is in the scrollbar or frame area.

**File: richtext.go**

```go
// ScrollRect returns the last scrollbar rectangle (for hit-testing).
func (rt *RichText) ScrollRect() image.Rectangle {
    return rt.lastScrollRect
}

// All returns the last full rectangle (for hit-testing).
func (rt *RichText) All() image.Rectangle {
    return rt.lastRect
}

// ScrollClick handles scrollbar clicks using the cached rectangle.
func (rt *RichText) ScrollClick(button int, pt image.Point) int {
    return rt.scrollClickAt(button, pt, rt.lastScrollRect)
}

func (rt *RichText) scrollClickAt(button int, pt image.Point, scrollRect image.Rectangle) int {
    // ... same logic but uses scrollRect parameter
}
```

### Step 6: Update Init() to Not Require Rectangle

Make Init() simpler since geometry comes at render time.

**File: richtext.go**

```go
// Init initializes the RichText component.
// The rectangle is not needed here - it will be provided at Render() time.
func (rt *RichText) Init(display draw.Display, font draw.Font, opts ...RichTextOption) {
    rt.display = display

    // Apply options
    for _, opt := range opts {
        opt(rt)
    }

    // Create frame (will get rectangle at render time)
    rt.frame = rich.NewFrame()

    // Initialize frame with minimal rectangle (will be updated on first Render)
    frameOpts := []rich.Option{
        rich.WithDisplay(display),
        rich.WithFont(font),
    }
    // ... add other options

    rt.frame.Init(image.Rectangle{}, frameOpts...)
}
```

### Step 7: Update Window.Resize()

**File: wind.go**

```go
func (w *Window) Resize(r image.Rectangle, safe, keepextra bool) int {
    // ... tag resize unchanged ...

    // Body resize - ALWAYS update body.all
    r1 = r
    r1.Min.Y = y
    if !safe || !w.body.all.Eq(r1) {
        oy := y
        if y+1+w.body.fr.DefaultFontHeight() <= r.Max.Y {
            // ... border drawing unchanged ...
        }

        // Always resize body Text to maintain canonical rectangle
        // Pass noredraw=true if in preview mode (we'll render ourselves)
        y = w.body.Resize(r1, keepextra, w.previewMode)
        w.r = r
        w.r.Max.Y = y
        w.body.all.Min.Y = oy

        // Render the appropriate view
        if w.previewMode && w.richBody != nil {
            w.richBody.Render(w.body.all)
        } else {
            w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
        }
    }

    // ... maxlines calculation ...
    return w.r.Max.Y
}
```

### Step 8: Update Window Drawing Methods

**File: wind.go**

Update all places that draw the preview to use `body.all`:

```go
func (w *Window) Redraw(...) {
    // ...
    if w.previewMode && w.richBody != nil {
        w.richBody.Render(w.body.all)
    } else {
        // normal body drawing
    }
}
```

### Step 9: Update Window Mouse Handling

Mouse hit-testing uses cached rectangles from last render.

**File: wind.go**

```go
func (w *Window) HandlePreviewMouse(...) {
    // Use cached rectangles from RichText
    if pt.In(w.richBody.ScrollRect()) {
        // scrollbar handling
    }
    if pt.In(w.richBody.Frame().Rect()) {
        // frame handling
    }
}
```

### Step 10: Update Preview Command

**File: previewcmd.go** (or equivalent)

```go
func (w *Window) enterPreviewMode() {
    if w.richBody == nil {
        w.richBody = NewRichText()
        w.richBody.Init(w.display, w.body.fr.Font(), /* options */)
    }

    // Parse and set content
    content, sourceMap, linkMap := markdown.ParseWithSourceMap(w.body.file.String())
    w.richBody.SetContent(content)
    w.previewSourceMap = sourceMap
    w.previewLinkMap = linkMap

    w.previewMode = true

    // Render into body's rectangle
    w.richBody.Render(w.body.all)
}
```

### Step 11: Remove Deprecated Methods

Clean up methods that assumed stored rectangles:

- Remove `RichText.All()` field access (keep method returning `lastRect`)
- Remove `RichText.Resize()` if it existed
- Update callers of old `Init(rect, ...)` signature

### Step 12: Update Tests

Update tests to use new initialization pattern:

```go
// Old:
rt.Init(rect, display, font, opts...)

// New:
rt.Init(display, font, opts...)
rt.Render(rect)
```

## Migration Strategy

1. **Add new methods first**: Add `SetRect()`, `Render()`, `scrDrawAt()` without removing old code
2. **Update Window to use new pattern**: Change resize/draw to use `Render(body.all)`
3. **Deprecate old methods**: Mark `Init(rect, ...)` as deprecated
4. **Update tests incrementally**: Fix tests one file at a time
5. **Remove old code**: Once all callers updated, remove deprecated code

## Testing Plan

### Unit Tests

1. `TestRichTextRenderUpdatesLastRect` - Verify `lastRect` is set after `Render()`
2. `TestRichTextRenderDifferentRects` - Verify rendering works with different rectangles
3. `TestFrameSetRect` - Verify `SetRect()` updates frame geometry
4. `TestFrameSetRectRelayout` - Verify layout uses new width after `SetRect()`

### Integration Tests

1. `TestWindowResizePreviewMode` - Resize window in preview, verify content visible
2. `TestWindowResizePreviewModeThenDraw` - Resize, then explicit draw, verify correct
3. `TestWindowResizeTogglePreviewMode` - Resize normal, toggle preview, verify rectangles match

### Manual Verification

1. Open .md file, toggle Preview
2. Drag window border to resize - preview should resize with window
3. Resize via column operations - preview should follow
4. Resize via Grow button - preview should follow
5. Verify scrollbar position correct after resize
6. Verify selection works after resize
7. Verify mouse wheel scrolling after resize

## Files Changed Summary

| File | Changes |
|------|---------|
| rich/frame.go | Add `SetRect()` to interface and implementation |
| richtext.go | Remove stored rectangles, add `Render()`, update scrollbar methods |
| wind.go | Update `Resize()` and draw methods to use `body.all` |
| previewcmd.go | Update preview initialization |
| richtext_test.go | Update test initialization pattern |
| wind_test.go | Add resize-in-preview tests |

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking existing tests | Add new methods first, migrate incrementally |
| Mouse hit-testing fails | Cache rectangles in `lastRect`/`lastScrollRect` |
| Scroll position lost on resize | Origin is rune-based, unaffected by geometry |
| Performance impact | Layout only recomputes when width changes |

## Success Criteria

1. All existing tests pass
2. Window resize in preview mode shows content correctly
3. No regression in normal (non-preview) window behavior
4. Scrollbar thumb position correct after resize
5. Selection and mouse interactions work after resize
