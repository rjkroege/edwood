# Horizontal Scrollbar Height Remediation

## Problem

When the window is scrolled down, horizontal scrollbars disappear and
click/selection positions are offset upward by the combined heights of
scrollbars above. The root cause is that scrollbar heights are applied
inconsistently across rendering and hit-testing code paths.

## Root Cause

`layoutFromOrigin()` returns lines with Y values that do NOT include
scrollbar height adjustments. `drawTextTo()` applies
`adjustLayoutForScrollbars()` in-place to shift Y for rendering, but all
other callers use unadjusted Y:

| Caller              | Line | Purpose                          | Uses adjusted Y? |
|---------------------|------|----------------------------------|------------------|
| `drawTextTo`        | 761  | Text/image/scrollbar rendering   | Yes (applies it) |
| `Ptofchar`          | 188  | Cursor position -> screen point  | **No**           |
| `Charofpt`          | 273  | Screen point -> character pos    | **No**           |
| `drawSelectionTo`   | 1151 | Selection highlight rendering    | **No**           |
| `drawTickTo`        | 1380 | Cursor bar rendering             | **No**           |
| `VisibleLines`      | 577  | Line count (Y not used)          | N/A              |

Additionally, when scrolled (`origin > 0`), `originY` is computed from
unadjusted `allLines` Y values in `layoutFromOrigin()`. This means the
viewport-relative Y values don't account for scrollbar heights of blocks
above the viewport, causing scrollbars of partially-visible blocks to be
positioned incorrectly.

### Concrete Example

Given:
- Font height: 20px, scrollbar height: 12px
- Lines 0-4: code block (each 20px), with overflowing content
- Line 5: normal text

Without scroll (origin=0):
- `drawTextTo` adjusts: lines 0-4 at Y=0,20,40,60,80; scrollbar at Y=100;
  line 5 at Y=112
- `Charofpt` sees: line 5 at Y=100 (unadjusted)
- Click at screen Y=112 -> Charofpt maps to wrong position (12px off)

With scroll to line 3:
- `layoutFromOrigin` computes `originY = allLines[3].Y = 60` (unadjusted)
- Visible lines get Y = 0, 20, 40, 52 (after drawTextTo adjustment)
- But Ptofchar/Charofpt see Y = 0, 20, 40 (unadjusted)
- Selections and cursors appear 12px above where text is rendered

## Design

### Principle

Apply scrollbar height adjustments exactly once, inside
`layoutFromOrigin()`, so all callers receive lines with correct Y values.
Separate the Y-modification logic from the metadata-computation logic so
that `drawTextTo` can get scrollbar rendering info without re-modifying Y.

### Step 1: New function `computeScrollbarMetadata()` in `layout.go`

Create a read-only function that computes `[]AdjustedBlockRegion` from
**already-adjusted** lines without modifying their Y values:

```go
// computeScrollbarMetadata returns AdjustedBlockRegion metadata for
// lines whose Y values already include scrollbar height adjustments.
// Unlike adjustLayoutForScrollbars, this does NOT modify line Y values.
func computeScrollbarMetadata(lines []Line, regions []BlockRegion, frameWidth, scrollbarHeight int) []AdjustedBlockRegion {
    adjusted := make([]AdjustedBlockRegion, len(regions))
    for i, r := range regions {
        adjusted[i] = AdjustedBlockRegion{BlockRegion: r}
        if r.MaxContentWidth > frameWidth {
            adjusted[i].HasScrollbar = true
        }
        // ScrollbarY is the bottom of the last line of the region
        if r.EndLine > 0 && r.EndLine <= len(lines) {
            lastLine := lines[r.EndLine-1]
            adjusted[i].ScrollbarY = lastLine.Y + lastLine.Height
        }
        // RegionTopY is the first line's Y
        if r.StartLine < len(lines) {
            adjusted[i].RegionTopY = lines[r.StartLine].Y
        }
    }
    return adjusted
}
```

### Step 2: Apply adjustments in `layoutFromOrigin()` (`frame.go`)

#### When `origin == 0`:

Currently returns unadjusted lines. Change to:

```go
if f.origin == 0 {
    lines := f.layoutBoxes(boxes, frameWidth, maxtab)
    regions := findBlockRegions(lines)
    f.syncHScrollState(len(regions))
    // Apply scrollbar height adjustments so all callers get correct Y
    adjustLayoutForScrollbars(lines, regions, frameWidth, 12)
    return lines, 0
}
```

#### When `origin > 0`:

Apply adjustments to `allLines` BEFORE computing `originY`:

```go
allLines := f.layoutBoxes(boxes, frameWidth, maxtab)
regions := findBlockRegions(allLines)
f.syncHScrollState(len(regions))
// Apply scrollbar heights to ALL lines before extracting visible subset
adjustLayoutForScrollbars(allLines, regions, frameWidth, 12)

// Now compute originY from ADJUSTED lines
// ... find startLineIdx as before ...
originY := allLines[startLineIdx].Y  // includes scrollbar heights above

// Extract visible lines with viewport-relative Y
for i := startLineIdx; i < len(allLines); i++ {
    adjustedLine := Line{
        Y:      allLines[i].Y - originY,
        Height: allLines[i].Height,
        Boxes:  allLines[i].Boxes,
    }
    visibleLines = append(visibleLines, adjustedLine)
}
```

### Step 3: Update `drawTextTo()` (`frame.go`)

Replace the call to `adjustLayoutForScrollbars` with the read-only
`computeScrollbarMetadata`:

```go
// Lines from layoutFromOrigin already have correct Y values.
// Compute scrollbar metadata without modifying Y again.
regions := findBlockRegions(lines)
scrollbarHeight := 12
adjustedRegions := computeScrollbarMetadata(lines, regions, frameWidth, scrollbarHeight)
f.hscrollRegions = adjustedRegions
```

### Step 4: Update tests

- Update `TestAdjustLayoutForScrollbarsShiftsSubsequentLines` and similar
  tests that verify Y shifting behavior.
- Add a test that creates a frame with a scrollbar-bearing code block,
  scrolls past it, and verifies that `Charofpt` and `Ptofchar` return
  positions consistent with where `drawTextTo` renders text.

## Files to Modify

| File | Changes |
|------|---------|
| `rich/layout.go` | Add `computeScrollbarMetadata()` |
| `rich/frame.go` | Update `layoutFromOrigin()` to apply adjustments; update `drawTextTo()` to use `computeScrollbarMetadata` |
| `rich/frame_test.go` | Update/add tests for adjusted Y correctness |
| `rich/layout_test.go` | Update tests that depend on `adjustLayoutForScrollbars` behavior |

## Verification

After the fix:
1. Scroll down past a code block with a horizontal scrollbar -- the
   scrollbar of any partially-visible block should render correctly
2. Click on text below a scrollbar -- cursor should appear at the
   clicked position, not above it
3. Selection highlighting should align with rendered text
4. Horizontal scrollbar click/wheel handling should still work
