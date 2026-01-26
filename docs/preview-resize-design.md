# Preview Mode Resize Bug - Design Options

## Problem Statement

When a window in preview mode is resized, the text body gets resized but the rich text preview does not. The preview visually disappears (replaced by the normal text body), but input focus remains on the preview component.

### Root Cause

`Window.Resize()` (wind.go:263-337) always calls `w.body.Resize()` regardless of preview mode. It never checks `previewMode` and never updates `richBody`.

The current architecture has **two separate rectangle owners**:
- `body Text` with `body.all` rectangle
- `richBody *RichText` with `richBody.all` rectangle

These get out of sync on resize.

### Affected Code Paths

```
Window.Resize()
    └── body.Resize(r1, keepextra, false)  // Always called
    └── body.ScrDraw(...)                   // Redraws body
    // richBody is never touched
```

---

## Option A: Patch Resize (Minimal Change)

### Concept

Add a `Resize()` method to `RichText` and check `previewMode` in `Window.Resize()`.

### Changes Required

**1. Add RichText.Resize() in richtext.go:**

```go
func (rt *RichText) Resize(r image.Rectangle) {
    rt.all = r

    // Recalculate scrollbar rectangle
    scrollWid := rt.display.ScaleSize(Scrollwid)
    scrollGap := rt.display.ScaleSize(Scrollgap)

    rt.scrollRect = image.Rect(
        r.Min.X,
        r.Min.Y,
        r.Min.X+scrollWid,
        r.Max.Y,
    )

    // Recalculate frame rectangle
    frameRect := image.Rect(
        r.Min.X+scrollWid+scrollGap,
        r.Min.Y,
        r.Max.X,
        r.Max.Y,
    )

    // Re-initialize frame with new rectangle
    // (Need to add SetRect to rich.Frame, or re-Init)
    rt.frame.SetRect(frameRect)
}
```

**2. Add Frame.SetRect() in rich/frame.go:**

```go
func (f *frameImpl) SetRect(r image.Rectangle) {
    f.rect = r
    f.relayout()  // Re-run layout with new width
}
```

**3. Modify Window.Resize() in wind.go (~line 312):**

```go
// Redraw body
r1 = r
r1.Min.Y = y
if !safe || !w.body.all.Eq(r1) {
    // ... existing rectangle setup code ...

    if w.previewMode && w.richBody != nil {
        // Preview mode: resize richBody instead
        w.richBody.Resize(r1)
        w.richBody.Redraw()
    } else {
        // Normal mode: resize body as before
        y = w.body.Resize(r1, keepextra, false)
        w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
    }
    // ... rest of method
}
```

### Pros
- Minimal code change
- Easy to understand and test
- Low risk of breaking existing functionality

### Cons
- Every body-related method needs similar `if previewMode` checks
- Two objects holding parallel state (body.all and richBody.all)
- Easy to miss a code path and introduce similar bugs

### Files Changed
- `richtext.go` - add Resize()
- `rich/frame.go` - add SetRect() or similar
- `wind.go` - modify Resize()

---

## Option B: Body Interface Abstraction

### Concept

Create a common interface that both `Text` and `RichText` implement. Window stores `body Body` where Body can be either type.

### Interface Definition

```go
// body.go (new file)

package main

import "image"

// Body represents a text display area that can be either
// editable Text or read-only RichText.
type Body interface {
    // Geometry
    All() image.Rectangle
    Resize(r image.Rectangle, keepextra, noredraw bool) int

    // Rendering
    Redraw()
    ScrDraw(nchars int)

    // Mouse handling
    HandleMouse(mouse *draw.Mouse) bool
    Select(mousectl *draw.Mousectl) (p0, p1 int)

    // Selection
    GetSelection() (p0, p1 int)
    SetSelection(p0, p1 int)

    // Scrolling
    Origin() int
    SetOrigin(org int)

    // Content (for snarf, etc.)
    SelectedText() string
}
```

### Changes Required

**1. Create body.go with interface definition**

**2. Make Text implement Body:**
- Most methods already exist
- Add wrapper methods where signatures differ
- Possibly add `All()` accessor if not present

**3. Make RichText implement Body:**
- Add missing methods (Resize, ScrDraw, etc.)
- Ensure method signatures match interface

**4. Change Window struct:**
```go
type Window struct {
    // ...
    tag    Text   // Tag stays as Text (always editable)
    body   Body   // Can be *Text or *RichText

    // For preview mode, keep reference to underlying Text
    textBody *Text  // Always the Text, even when body is RichText
}
```

**5. Update all Window methods:**
- Replace `w.body.` calls with interface methods
- For Text-specific operations (editing), use type assertion or textBody

### Pros
- Clean abstraction - single code path for body operations
- Enforces consistency between Text and RichText
- Makes the "render body differently" design explicit

### Cons
- Large refactor touching many files
- Text is also used for tags (need to ensure tag stays as Text)
- Interface may become bloated as edge cases emerge
- Type assertions needed for edit operations

### Files Changed
- New `body.go`
- `text.go` - ensure interface compliance
- `richtext.go` - add interface methods
- `wind.go` - change body type, update all usages
- Potentially many other files that access w.body

---

## Option C: Single Rectangle Owner

### Concept

The body `Text` always owns the rectangle. Preview mode changes rendering, not geometry. `RichText` renders into whatever rectangle it's given at render time.

This aligns with the original design intent: "window is just a normal window that happens to render its body differently."

### Architecture

```
Window
├── body Text        <- Always owns geometry (body.all)
├── previewMode bool
└── richRenderer     <- Stateless renderer, no rectangle storage

Resize:
    body.Resize()    <- Always updates body.all

Draw:
    if previewMode:
        richRenderer.Render(body.all, parsedContent)
    else:
        body.Redraw()
```

### Changes Required

**1. Restructure RichText to not own rectangle:**

```go
// RichText becomes more like a renderer than a component
type RichText struct {
    display    draw.Display
    frame      rich.Frame
    content    rich.Content
    // NO all or scrollRect fields - computed from passed rect

    // Options for styling
    fonts      FontSet
    colors     ColorSet
}

// Render draws into the given rectangle
func (rt *RichText) Render(r image.Rectangle) {
    scrollWid := rt.display.ScaleSize(Scrollwid)
    scrollGap := rt.display.ScaleSize(Scrollgap)

    scrollRect := image.Rect(r.Min.X, r.Min.Y, r.Min.X+scrollWid, r.Max.Y)
    frameRect := image.Rect(r.Min.X+scrollWid+scrollGap, r.Min.Y, r.Max.X, r.Max.Y)

    // Ensure frame has correct rect
    if rt.frame.Rect() != frameRect {
        rt.frame.SetRect(frameRect)
    }

    rt.drawScrollbar(scrollRect)
    rt.frame.Redraw()
}
```

**2. Modify Window to use body.all as source of truth:**

```go
func (w *Window) Resize(r image.Rectangle, safe, keepextra bool) int {
    // ... tag resize unchanged ...

    // Body resize - always update body.all
    r1 = r
    r1.Min.Y = y
    if !safe || !w.body.all.Eq(r1) {
        // Always resize body Text to maintain canonical rectangle
        y = w.body.Resize(r1, keepextra, true /* noredraw - we'll draw ourselves */)
        w.r = r
        w.r.Max.Y = y
        w.body.all.Min.Y = oy

        // Now draw the appropriate view
        if w.previewMode && w.richBody != nil {
            w.richBody.Render(w.body.all)
        } else {
            w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
        }
    }
    return w.r.Max.Y
}
```

**3. Update Window.Draw():**

```go
func (w *Window) Draw() {
    if w.previewMode && w.richBody != nil {
        w.richBody.Render(w.body.all)  // Use body's rectangle
    } else {
        // Normal body rendering
        w.body.fr.Redraw(w.body.fr.Rect())
    }
}
```

### Pros
- Single source of truth for geometry
- Aligns with design doc: "window renders body differently"
- Resize code doesn't need preview mode checks
- Clear separation: Text owns geometry, RichText handles rendering

### Cons
- Requires restructuring RichText significantly
- rich.Frame needs to support rectangle changes
- Scrollbar state (thumb position) needs to be managed differently
- Mouse hit-testing needs to use computed rectangles

### Files Changed
- `richtext.go` - major restructure
- `rich/frame.go` - add SetRect() or similar
- `wind.go` - update Draw(), Resize()
- Tests may need updates

---

## Option D: RichText Re-initialization

### Concept

Add a `SetRect()` or `Resize()` method to RichText that allows updating geometry after initial creation. This is a pragmatic middle ground - keep two rectangles but ensure they stay in sync.

### Changes Required

**1. Add RichText.SetRect() in richtext.go:**

```go
// SetRect updates the component's rectangle and reinitializes geometry.
// Call this when the window is resized.
func (rt *RichText) SetRect(r image.Rectangle) {
    if rt.all.Eq(r) {
        return  // No change
    }

    rt.all = r

    // Recalculate scrollbar
    scrollWid := rt.display.ScaleSize(Scrollwid)
    scrollGap := rt.display.ScaleSize(Scrollgap)

    rt.scrollRect = image.Rect(
        r.Min.X,
        r.Min.Y,
        r.Min.X+scrollWid,
        r.Max.Y,
    )

    // Recalculate frame rect
    frameRect := image.Rect(
        r.Min.X+scrollWid+scrollGap,
        r.Min.Y,
        r.Max.X,
        r.Max.Y,
    )

    // Update frame geometry
    if rt.frame != nil {
        rt.frame.SetRect(frameRect)
    }
}
```

**2. Add Frame.SetRect() in rich/frame.go:**

```go
// SetRect updates the frame's rectangle and triggers relayout.
func (f *frameImpl) SetRect(r image.Rectangle) {
    if f.rect.Eq(r) {
        return
    }
    f.rect = r
    // Relayout content for new width
    if f.content != nil {
        f.layout()
    }
}
```

**3. Modify Window.Resize() in wind.go:**

```go
func (w *Window) Resize(r image.Rectangle, safe, keepextra bool) int {
    // ... tag handling unchanged ...

    // Body resize
    r1 = r
    r1.Min.Y = y
    if !safe || !w.body.all.Eq(r1) {
        oy := y
        if y+1+w.body.fr.DefaultFontHeight() <= r.Max.Y {
            // ... border drawing unchanged ...
        }

        // Always resize body to keep its rectangle current
        y = w.body.Resize(r1, keepextra, w.previewMode /* noredraw if preview */)
        w.r = r
        w.r.Max.Y = y
        w.body.all.Min.Y = oy

        // If in preview mode, also resize richBody
        if w.previewMode && w.richBody != nil {
            w.richBody.SetRect(r1)
            w.richBody.Redraw()
        } else {
            w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
        }
    }

    // ... maxlines calculation ...
    return w.r.Max.Y
}
```

### Pros
- Straightforward implementation
- Maintains existing architecture
- Easy to understand - "when rectangle changes, update both"
- Low risk of breaking other code paths

### Cons
- Still two rectangles to keep in sync
- Need to audit other code paths that might resize
- Slightly duplicated logic (body.all and richBody.all should always match)

### Files Changed
- `richtext.go` - add SetRect()
- `rich/frame.go` - add SetRect()
- `wind.go` - modify Resize()

---

## Comparison Matrix

| Aspect | Option A | Option B | Option C | Option D |
|--------|----------|----------|----------|----------|
| Code change size | Small | Large | Medium | Small |
| Risk of regression | Low | Medium | Medium | Low |
| Architectural cleanliness | Poor | Good | Best | Fair |
| Duplicate state | Yes | No | No | Yes |
| Future maintenance | Harder | Easier | Easier | Medium |
| Implementation time | ~1 hour | ~1 day | ~3 hours | ~1 hour |

## Recommendation

**For quick fix**: Option D - pragmatic, low risk, addresses the immediate bug.

**For long-term**: Option C - aligns with design intent, single source of truth.

Option B is overkill for this problem and touches too much code. Option A is essentially Option D but with more scattered changes.

---

## Testing Plan

Regardless of approach chosen:

1. **Unit tests** for new SetRect/Resize methods
2. **Integration test**: Toggle preview, resize window, verify preview displays correctly
3. **Manual verification**:
   - Resize preview window by dragging border
   - Resize preview window via column operations
   - Grow/shrink via tag button
   - Verify scrollbar still works after resize
   - Verify selection still works after resize
   - Verify mouse wheel scrolling after resize
