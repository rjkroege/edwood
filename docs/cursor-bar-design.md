# Cursor Bar for Rich Text Frame (Phase 23)

The Markdeep preview rich frame has no visible cursor (tick). Regular Edwood windows display a thin black bar with serif-like boxes at top and bottom when the selection is a point (`p0 == p1`). The rich frame should match this behavior, with cursor height scaled to the content at the insertion point.

## How the Regular Frame Does It

In `frame/tick.go`, `InitTick()` pre-renders a tick image:

1. Allocate a `frtickw*tickscale` x `fontHeight` image with `Transparent` background
2. Draw a 1-pixel-wide (scaled) vertical line in the center
3. Draw `frtickw x frtickw` boxes at the top and bottom (serifs)

In `frame/draw.go`, `tick(pt, ticked)`:

```go
r := image.Rect(pt.X, pt.Y, pt.X+frtickw*tickscale, pt.Y+defaultfontheight)
background.Draw(r, display.Black(), tickimage, image.ZP)
```

The tick image is used as an alpha mask against black, producing a black cursor with transparent gaps. The regular frame also saves/restores background pixels under the cursor (`tickback`), but this is unnecessary for the rich frame since it fully redraws via a scratch image.

Key constants:
- `frtickw = 3` — tick width in unscaled pixels
- `tickscale = display.ScaleSize(1)` — DPI scale factor
- Height = `font.Height()` (fixed, single font)

## Rich Frame Differences

The rich frame has variable line heights: a heading line (Scale 2.0) is taller than a body text line. The cursor height must adapt.

### Height Rule: Tallest Adjacent Box

The cursor sits between two boxes (or at the start/end of a line). Its height is determined by the **tallest of the two adjacent boxes** — the box immediately before and immediately after the cursor position. This produces natural-looking cursors:

- Between two body-text boxes: cursor is body font height
- Between a heading box and a body box: cursor is heading font height
- At the start of a heading line: cursor is heading font height (only the box after exists)
- At the end of a line: cursor uses the last box's height

The box height is determined by `fontForStyle(box.Style).Height()` for text boxes, or the image's scaled height for image boxes.

### Why Adjacent Boxes, Not Line Height

Line height is the max of all boxes on the line. A line containing one tall image and several short text boxes would give the cursor the image height even when positioned between two short text boxes far from the image. Using adjacent boxes gives a locally correct cursor height.

## Implementation

### Approach

All rendering in `rich/frame.go`. No new files.

The rich frame already fully redraws each frame to a scratch image, so there is no need for the save/restore (`tickback`) mechanism used by the regular frame. The tick is simply drawn as part of each `Redraw()` pass.

### Data Structures

Add to `frameImpl`:

```go
tickImage  edwooddraw.Image // pre-rendered tick mask (transparent + opaque pattern)
tickScale  int              // display.ScaleSize(1)
tickHeight int              // height of current tickImage (re-init when changed)
```

### initTick(height int)

Creates or recreates the tick image when the required height changes. Follows the same pattern as `frame/tick.go:InitTick()`:

```go
func (f *frameImpl) initTick(height int) {
    if f.display == nil { return }
    if f.tickImage != nil && f.tickHeight == height { return }
    if f.tickImage != nil { f.tickImage.Free() }

    scale := f.display.ScaleSize(1)
    f.tickScale = scale
    w := frtickw * scale

    img, err := f.display.AllocImage(
        image.Rect(0, 0, w, height),
        f.display.ScreenImage().Pix(), false, draw.Transparent)
    if err != nil { return }

    // Fill transparent
    img.Draw(img.R(), f.display.Transparent(), nil, image.ZP)
    // Vertical line in center
    img.Draw(image.Rect(scale*(frtickw/2), 0, scale*(frtickw/2+1), height),
        f.display.Opaque(), nil, image.ZP)
    // Top serif
    img.Draw(image.Rect(0, 0, w, w),
        f.display.Opaque(), nil, image.ZP)
    // Bottom serif
    img.Draw(image.Rect(0, height-w, w, height),
        f.display.Opaque(), nil, image.ZP)

    f.tickImage = img
    f.tickHeight = height
}
```

### drawTickTo(target, offset)

Called from `Redraw()` when `p0 == p1`. Locates the cursor position in the layout, determines height from adjacent boxes, and draws the tick.

```go
func (f *frameImpl) drawTickTo(target edwooddraw.Image, offset image.Point) {
    lines, originRune := f.layoutFromOrigin()
    cursorPos := f.p0 - originRune
    if cursorPos < 0 { return }

    // Walk lines and boxes to find cursor X position and adjacent box heights
    runeCount := 0
    for _, line := range lines {
        lineStartRune := runeCount
        for i, pb := range line.Boxes {
            boxRunes := boxRuneCount(pb)
            if runeCount+boxRunes > cursorPos || (runeCount == cursorPos && i == len(line.Boxes)-1) {
                // Found the cursor position
                x := ... // compute X within box (same logic as Ptofchar)

                // Adjacent box heights
                prevHeight, nextHeight := 0, 0
                if i > 0 {
                    prevHeight = f.boxHeight(line.Boxes[i-1].Box)
                }
                nextHeight = f.boxHeight(pb.Box)
                if runeCount == cursorPos && i > 0 {
                    // Cursor is at the boundary: prev is the box before, next is this box
                    nextHeight = f.boxHeight(pb.Box)
                }
                tickH := max(prevHeight, nextHeight)
                if tickH == 0 { tickH = f.font.Height() }

                f.initTick(tickH)
                // Draw tick centered on the line
                // ... draw call ...
                return
            }
            runeCount += boxRunes
        }
    }
}
```

### Redraw Integration

After text drawing, before blit to screen:

```go
// Draw cursor tick when selection is a point (p0 == p1)
if f.content != nil && f.font != nil && f.display != nil && f.p0 == f.p1 {
    f.drawTickTo(scratch, drawOffset)
}
```

## Testing

1. **TestDrawTickAtCursor** — p0 == p1 at a known position; verify Draw called with Black() and tick-sized rectangle
2. **TestNoTickWithSelection** — p0 != p1; verify no tick draw calls
3. **TestTickHeightScaling** — cursor between heading box and body box; verify tick height matches heading font height (the taller adjacent box)
4. **TestTickHeightBodyText** — cursor between two body text boxes; verify tick height is body font height
