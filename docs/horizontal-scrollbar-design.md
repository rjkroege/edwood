# Horizontal Scroll Bars for Non-Foldable Elements

## Problem

Non-foldable layout elements (code blocks, tables, images) currently have
their content force-wrapped to fit within the frame width. This is wrong for
code blocks (where line breaks are semantically meaningful) and undesirable
for wide tables and images. We need horizontal scrolling with an
acme-style scrollbar along the bottom of each overflowing element.

## Scope

Elements that get horizontal scrollbars:
- Fenced code blocks (`Block && Code`)
- Tables (`Table`)
- Images wider than the frame (already scaled down; this would let them
  display at native width with scroll)

Elements that do NOT get horizontal scrollbars:
- Normal prose paragraphs (continue to wrap)
- Headings, lists, inline code (continue to wrap)
- Horizontal rules

## Current Architecture Summary

- **Layout** (`layout.go`): Converts `Content -> []Box -> []Line`. Forces
  all content to fit `frameWidth` via word wrap and `splitBoxAcrossLines`.
- **Rendering** (`frame.go`): Five-phase `drawTextTo` draws onto a
  frame-sized scratch image, giving implicit vertical+horizontal clipping.
- **Vertical scroll** (`richtext.go`): Acme-style scrollbar on the left.
  Origin is a rune offset. Thumb computed from pixel-height proportions.
- **Data model** (`span.go`, `style.go`): `Content = []Span`, each `Span`
  has a `Style`. Block code is `Block=true, Code=true`. No per-element
  scroll state.

## Design

### 1. Layout Changes: No-Wrap Mode for Block Elements

The `layout()` function currently wraps everything. For block elements
that should scroll horizontally, we need it to lay out their content on
single (potentially very wide) lines instead.

**Approach**: In `layout()`, when processing boxes with `Block && Code`
(or `Table`, etc.), skip the wrap check and splitting. Let the line extend
to whatever width the content requires. Record the actual content width
on each `Line`.

Add a field to `Line`:

```go
type Line struct {
    Boxes      []PositionedBox
    Y          int
    Height     int
    ContentWidth int  // Actual pixel width of content (may exceed frameWidth)
}
```

In the wrapping section of `layout()`, the key change is:

```go
// For block-level code, don't wrap - allow horizontal overflow
if box.Style.Block && box.Style.Code {
    // Skip wrap check; just place the box at xPos
    box.Wid = width
    currentLine.Boxes = append(currentLine.Boxes, PositionedBox{Box: *box, X: xPos})
    xPos += width
    continue
}
```

After each line is finalized, set `ContentWidth` to the maximum X extent
of its boxes.

### 2. Block Regions

A horizontal scrollbar spans an entire block (e.g., the full code fence),
not individual lines. We need to identify contiguous runs of lines that
belong to the same block element and treat them as a scrollable region.

**Approach**: After layout, scan the `[]Line` to identify "block regions" --
contiguous runs of lines sharing the same block type. Each region has:

```go
type BlockRegion struct {
    StartLine    int  // Index into []Line
    EndLine      int  // Exclusive
    MaxContentWidth int  // Widest line in this region
    Kind         BlockKind  // Code, Table, Image
}

type BlockKind int
const (
    BlockCode BlockKind = iota
    BlockTable
    BlockImage
)
```

A `BlockRegion` only needs a horizontal scrollbar when
`MaxContentWidth > frameWidth`.

### 3. Horizontal Scroll State in the Data Model

We need to store horizontal scroll position per block element. The key
challenge is choosing a stable identifier for each block that survives
re-layout.

**Rejected: Rune offset key** -- Any insertion or deletion in the
document shifts all subsequent rune offsets, causing scroll state to get
mismatched to the wrong block.

**Chosen: Ordinal index of the non-wrapping element.**

After layout and block region identification, each `BlockRegion` has a
natural ordinal: the 0th non-wrapping block, the 1st, the 2nd, etc.
This ordinal is stable across re-layouts as long as the set of
non-wrapping blocks doesn't change. The scroll state is a simple slice
indexed by this ordinal:

```go
// In frameImpl
type frameImpl struct {
    // ... existing fields ...

    // Horizontal scroll state per non-wrapping block element.
    // Index is the ordinal of the block region (0th code block, 1st, etc.).
    // Value is the pixel offset from the left edge.
    hscrollOrigins []int

    // Number of non-wrapping blocks seen on the last layout pass.
    // Used to detect when blocks are added or removed.
    hscrollBlockCount int
}
```

**Invalidation**: After each layout pass, compare the new block region
count to `hscrollBlockCount`. If the count changed (a non-wrapping
element was added or removed), reset the entire `hscrollOrigins` slice
to zero. This is a simple and correct policy for the short term. In the
future we could try to be smarter about matching old state to new blocks
(e.g., by diffing block content), but resetting is fine for now.

When the count is unchanged, the existing scroll positions carry over
directly -- ordinal 0 still maps to the 0th block, etc.

### 4. Rendering with Horizontal Offset

During `drawTextTo`, for lines that belong to a scrollable block region,
apply a horizontal pixel offset:

```go
// For each positioned box in a scrollable block:
hOffset := f.hscrollOrigins[regionIndex]  // pixel offset, 0 = no scroll

pt := image.Point{
    X: offset.X + pb.X - hOffset,
    Y: offset.Y + line.Y,
}
```

This shifts all content left by `hOffset` pixels. The scratch image
provides clipping on both left and right edges automatically.

The same offset applies to:
- Phase 1: Block backgrounds (already full-width, no change needed)
- Phase 2: Box backgrounds (apply hOffset to X)
- Phase 4: Text rendering (apply hOffset to X)
- Phase 5: Image rendering (apply hOffset to X)

### 5. Horizontal Scrollbar Drawing

Each block region that overflows draws a horizontal scrollbar along its
bottom edge. The scrollbar is drawn in the same style as the vertical
acme scrollbar, but rotated 90 degrees:

```
+-----------------------------------------------+
|  code line 1 that is very long and extends ... |
|  code line 2                                   |
|  code line 3 also very long content that ex... |
|[====thumb===]                                  |  <- horizontal scrollbar
+-----------------------------------------------+
```

**Geometry**:
- Height: `Scrollwid` (12 scaled pixels), same as vertical scrollbar width
- Width: full frame width (from block left indent to right edge)
- Position: bottom of the block region
- The scrollbar occupies space within the block's visual area; the last
  `Scrollwid` pixels of the block's visible height are the scrollbar

**Thumb calculation** (mirrors vertical logic):
- `thumbWidth = (frameWidth / maxContentWidth) * scrollbarWidth`
- `thumbLeft = (hOffset / maxScrollable) * (scrollbarWidth - thumbWidth)`
- where `maxScrollable = maxContentWidth - frameWidth`

**Drawing**: In `drawTextTo` (or a new phase 6), after rendering block
content, draw the scrollbar background and thumb onto the scratch image
at the bottom of each overflowing block region.

### 6. Mouse Handling

The horizontal scrollbar needs mouse input handling. When a click lands
within a horizontal scrollbar's rectangle, it should be handled with the
same three-button acme semantics, but in the horizontal direction:

- **Button 1 (left)**: Scroll left. Amount scaled by click X position.
- **Button 2 (middle)**: Jump to absolute horizontal position.
- **Button 3 (right)**: Scroll right. Amount scaled by click X position.

**Hit testing**: In the frame's `Charofpt` or a new method, check if the
click point falls within any horizontal scrollbar rectangle. If so,
dispatch to horizontal scroll handling instead of character selection.

This requires the frame to expose block regions and their scrollbar
rectangles:

```go
// New method on Frame interface
HScrollBarAt(pt image.Point) (regionIndex int, ok bool)
```

The `RichText` layer (or the mouse handler in `body.go` / wherever mouse
events are dispatched) checks `HScrollBarAt` before falling through to
normal click handling.

**Scroll wheel**: When the cursor is over a horizontally-scrollable block
region (but not on the scrollbar itself), horizontal scroll wheel events
(shift+scroll or trackpad horizontal swipe) should adjust `hscrollOrigins`.

### 7. Vertical Scrollbar Interaction

The block region's horizontal scrollbar takes up vertical space at the
bottom of the block. This means the block is visually taller by
`Scrollwid` pixels when it overflows horizontally.

**Two-pass layout**: Only show the scrollbar when the block actually
overflows. Pass 1 lays out content without scrollbar space and
determines content widths. Pass 2 identifies overflowing block regions
and inserts `Scrollwid` pixels of additional height at the bottom of
each, shifting all subsequent lines down. The two-pass cost is
negligible since layout is already O(n) in content size.

### 8. Clipping

The scratch image already clips to frame bounds. For horizontal scrolling,
the scratch image continues to work correctly: content shifted left by
`hOffset` that falls outside `[0, frameWidth)` is clipped by the image
boundaries.

For the vertical dimension, we need to ensure that content within a
scrollable block doesn't extend below the scrollbar area. When drawing
text in a block with an h-scrollbar, clip the Y extent to
`blockBottomY - scrollbarHeight`.

## Implementation Order

1. **Add `ContentWidth` to `Line`**: Compute during layout. No behavioral
   change yet.

2. **No-wrap mode for block code**: Modify `layout()` to skip wrapping
   for `Block && Code` boxes. Code blocks now overflow but are clipped
   by the scratch image. Visually the content is just truncated on the
   right -- functional but ugly.

3. **Block region identification**: Add `BlockRegion` computation after
   layout. This is pure data, no rendering yet.

4. **`hscrollOrigins` slice on frameImpl**: Add the slice and block count
   tracking. After layout, compare block count and reset to zero if
   changed. No UI yet.

5. **Apply horizontal offset in rendering**: Modify the five drawing
   phases to offset X by `hscrollOrigins[blockStartRune]`. Content can
   now be scrolled programmatically (e.g., via tests).

6. **Draw horizontal scrollbar**: Add phase 6 to `drawTextTo` (or a
   separate method called from `Redraw`). Draws scrollbar background
   and thumb for overflowing block regions.

7. **Mouse handling**: Add `HScrollBarAt` hit-test method. Wire up
   three-button click handling. Wire up scroll wheel.

8. **Two-pass layout for scrollbar height**: After identifying overflowing
   regions, adjust line Y positions to account for scrollbar height.

9. **Tables**: Extend no-wrap mode to table rows. Table cells may need
   special handling (min-width per column, etc.).

10. **Images**: Optionally allow images to display at native width with
    horizontal scroll instead of always scaling down.

## Scrollbar Latching (Remediation)

### Problem

Clicking a preview scrollbar (vertical or horizontal) fires a single
scroll action and returns. Acme-style scrollbars latch: once a button is
pressed in a scrollbar, the scroll action tracks the mouse until the
button is released, even if the cursor leaves the scrollbar area.

### Acme Pattern (from `scrl.go:101-166`)

```
1. x = center of scrollbar
2. Initial scroll action
3. Loop {
     flush display
     my = clamp(mouse.Y, scrollbar top, scrollbar bottom)
     if mouse != (x, my):
         display.MoveTo(x, my)   // WARP cursor back into scrollbar
         mc.Read()               // absorb synthetic move event
     recompute scroll from (x, my)
     redraw
     debounce: 200ms on first iteration, then ScrSleep(80ms)
     if button released → break
   }
4. Drain remaining mouse events until all buttons released
```

Key details:
- The cursor is **physically warped** back into the scrollbar via
  `display.MoveTo()` on every iteration. The user cannot move the
  cursor out of the scrollbar while the button is held.
- For the vertical scrollbar, X is locked to the center of the
  scrollbar column and Y is clamped to the scrollbar's Y range.
- The `MoveTo` call generates a synthetic mouse event which must be
  absorbed by reading from the mouse channel.
- B2 (middle) tracks continuously per-event (live thumb drag).
  B1/B3 use timer-based debounce for auto-repeat scrolling.

### Implementation

**Vertical scrollbar** (`wind.go`, `HandlePreviewMouse`):

Replace the three single-shot `ScrollClick` calls with a call to a new
`previewVScrollLatch(rt, mc, button, scrRect)` method that:
- Computes `centerX = (scrRect.Min.X + scrRect.Max.X) / 2`
- Performs initial `ScrollClick(button, mouse.Point)` + draw + flush
- Loops reading from `mc.C`
- Clamps mouse Y to `scrRect` bounds
- Warps cursor to `(centerX, clampedY)` via `display.MoveTo` if it
  has moved away; absorbs the resulting synthetic mouse event
- Calls `ScrollClick(button, warpedPt)` each iteration
- B2: reads per-event (live thumb drag)
- B1/B3: debounce with 200ms initial, then 80ms repeat (matching acme)
- Breaks when `buttons & (1 << (button-1)) == 0`
- Drains remaining events until `buttons == 0`

**Horizontal scrollbar** (`wind.go`, `HandlePreviewMouse`):

Same structure via `previewHScrollLatch(rt, mc, button, regionIndex)`:
- Computes `centerY` of the scrollbar band (from `hscrollRegions`)
- Clamps mouse X to frame bounds
- Warps cursor to `(clampedX, centerY)` via `display.MoveTo`
- Calls `HScrollClick(button, warpedPt, regionIndex)` each iteration
- Same debounce timing and break/drain logic

**Debounce helper** (`previewScrSleep`): matches `ScrSleep` from
`scrl.go` but reads from the passed-in `mc` rather than `global.mousectl`.

## Open Questions

1. **Should the scrollbar be inside or outside the block background?**
   If inside, it overlaps the last line of code slightly. If outside,
   it adds height but is visually cleaner. Leaning toward inside (overlay
   on the block background) since code blocks typically have some bottom
   padding.
   Answer: Let's start with inside.

2. **Should horizontal scroll state persist across content refreshes?**
   Currently proposed to reset all scroll positions to zero when the
   number of non-wrapping blocks changes. When the count is stable,
   positions carry over by ordinal index. Could get smarter in the
   future (e.g., diff block content to remap indices), but reset-on-
   change is fine for now.
   Answer: Preserve if count is the same

3. **Keyboard scrolling?** Should arrow keys or Home/End affect
   horizontal scroll when the cursor is in a code block? This is a nice
   future enhancement but not needed initially.
   Answer: Not needed

4. **Minimum scrollbar thumb width?** The vertical scrollbar uses 10px
   minimum thumb height. Should use the same (10px minimum thumb width)
   for horizontal.
   Answer: Sure, that's a good start

## Scrollbar Left Indent (Gutter)

### Problem

The horizontal scrollbar spans the full frame width, starting at X=0.
This means the left gutter area (to the left of the code block content)
is part of the scrollbar, so vertical swipe gestures in that area are
captured by horizontal scrollbar hit-testing instead of scrolling
vertically.

### Design

Limit the horizontal scrollbar to the block content area. Code blocks
are indented by `codeBlockIndent` pixels (4 × code font M-width,
typically ~40px). The scrollbar should start at that indent, leaving a
gutter on the left where vertical scroll gestures pass through.

### Changes

1. **Add `LeftIndent` to `AdjustedBlockRegion`** (`rich/layout.go`):
   New `int` field recording the X pixel offset where block content
   starts. Populated in both `adjustLayoutForScrollbars` and
   `computeScrollbarMetadata` by scanning `lines[StartLine].Boxes` for
   the first block-styled box's X position (same logic as
   `drawBlockBackgroundTo`).

2. **Update `drawHScrollbarsTo`** (`rich/frame.go`): Scrollbar
   background and thumb start at `offset.X + ar.LeftIndent`. Scrollbar
   width becomes `frameWidth - ar.LeftIndent`. Thumb calculations use
   this narrower width.

3. **Update `HScrollBarAt`** (`rich/frame.go`): Hit-test X range
   changes from `[0, frameWidth)` to `[ar.LeftIndent, frameWidth)`.

4. **Update `HScrollBarRect`** (`rich/frame.go`): Returned rectangle's
   `Min.X` changes from `f.rect.Min.X` to
   `f.rect.Min.X + ar.LeftIndent`.

5. **Update `HScrollClick`** (`rich/frame.go`): Click proportion uses
   `relX` relative to `ar.LeftIndent` and divides by scrollbar width
   (`frameWidth - ar.LeftIndent`) instead of `frameWidth`.

6. **Update `previewHScrollLatch`** (`wind.go`): X clamping uses
   `barRect.Min.X` (which now reflects indent) instead of
   `frameRect.Min.X`.
