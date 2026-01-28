# Preview Interaction Fixes (Phase 21)

Three bugs in Markdeep preview mode interaction:

1. **No cursor-warp after Look** — after B3 search finds a match, the mouse cursor stays where it was instead of warping to the found text
2. **No colored sweep** — B2 sweep should be red, B3 sweep should be green, matching normal Acme behavior
3. **Broken coordinate mapping after scroll** — after jump-scrolling to show a search result, `Charofpt()`/`Ptofchar()` return wrong positions because they don't account for the scroll origin

## Bug 1: Missing Cursor Warp After Look

### How Normal Acme Does It

In `look.go:165`:
```go
if search(ct, r[:n]) && e.jump {
    global.row.display.MoveTo(
        ct.fr.Ptofchar(getP0(ct.fr)).Add(image.Pt(4, ct.fr.DefaultFontHeight()-4)))
}
```

After `search()` succeeds:
1. Get the start of the selection: `getP0(ct.fr)` returns the frame's `p0`
2. Map to screen coordinates: `ct.fr.Ptofchar(p0)`
3. Add offset: `(+4, +fontHeight-4)` places the cursor just inside the matched text
4. Warp: `display.MoveTo(pt)`

### What Preview Mode Is Missing

The B3 handler in `wind.go` calls `search()`, maps back to rendered coordinates, sets the preview selection, scrolls, and redraws — but never warps the cursor. After `w.Draw()` and flush, the cursor remains at the original click location.

### Fix

After setting the preview selection and scrolling, add:
```go
if w.display != nil {
    warpPt := rt.Frame().Ptofchar(rendStart).Add(
        image.Pt(4, rt.Frame().Font().Height()-4))
    w.display.MoveTo(warpPt)
}
```

This requires `Ptofchar` to be correct (see Bug 3 fix) and the rich frame to expose a font height method. The rich frame currently has no `DefaultFontHeight()` method — we need to add one, or use the font directly.

## Bug 2: No Colored Sweep Selection

### How Normal Acme Does It

The normal frame has `SelectOpt(mc, m, getmorelines, fg, bg)` which temporarily swaps selection colors:

```go
// frame/select.go:25-46
func (f *frameimpl) SelectOpt(..., fg, bg draw.Image) (int, int) {
    oback := f.cols[ColHigh]
    f.cols[ColHigh] = bg       // swap to custom color
    defer func() {
        f.cols[ColHigh] = oback // restore
    }()
    return f.selectimpl(mc, downevent, getmorelines)
}
```

The `Text` layer calls this with button-specific colors:

```go
// text.go — Select2() passes red, Select3() passes green
func (t *Text) Select2() { t.Select23(global.but2col, 4) }   // 0xAA0000FF = red
func (t *Text) Select3() { t.Select23(global.but3col, 1|2) } // 0x006600FF = green
```

### What the Rich Frame Lacks

The rich frame's `Select()` method:
- Takes no color parameter
- Always uses `f.selectionColor` (a fixed field)
- Has no `SelectOpt()` variant
- `drawSelectionTo()` always uses `f.selectionColor`

### Fix

Add a temporary selection color override mechanism to the rich frame:

1. Add `sweepColor edwooddraw.Image` field to `frameImpl` — when non-nil, `drawSelectionTo()` uses it instead of `selectionColor`
2. Add `SelectWithColor(mc, m, color) (p0, p1 int)` method that sets `sweepColor` for the duration of the drag, then clears it
3. Add `SelectWithChordAndColor(mc, m, color) (p0, p1, chordButtons int)` — same but with chord detection
4. In `HandlePreviewMouse`, pass `global.but2col` for B2 sweeps and `global.but3col` for B3 sweeps
5. B1 sweeps continue using the default `selectionColor` (no change needed)

The `drawSelectionTo()` change is minimal:
```go
color := f.selectionColor
if f.sweepColor != nil {
    color = f.sweepColor
}
target.Draw(selRect, color, color, image.ZP)
```

After the sweep completes, `sweepColor` is set back to nil so the final selection renders in the normal highlight color (matching Acme behavior where the colored sweep reverts to normal highlight color on button release).

## Bug 3: Broken Coordinate Mapping After Scroll

### Root Cause

`Charofpt()` and `Ptofchar()` use `f.layoutBoxes()` which lays out the ENTIRE document from position 0. Line Y-coordinates are absolute positions in the full document. But click coordinates are frame-relative (point.Y - frame.Min.Y), starting from 0 at the top of the visible viewport.

Meanwhile, `Redraw()` calls `drawTextTo()` which uses `layoutFromOrigin()` — this correctly adjusts Y coordinates by subtracting `originY`, so the first visible line has Y=0.

After `scrollPreviewToMatch()` sets the origin (e.g., origin=500 → line 20), the text renders correctly at viewport-relative positions. But when the user clicks:

1. Click at frame-relative Y=100 (3rd visible line)
2. `Charofpt()` lays out from position 0
3. Line at Y=100 in the full layout is maybe line 3 of the document
4. But the viewport is showing lines 20+ — the actual line at Y=100 in the viewport is line 22
5. **Result**: click maps to the wrong text, off by ~17 lines

### Fix

Both `Charofpt()` and `Ptofchar()` must use `layoutFromOrigin()` instead of `layoutBoxes()`, and `Charofpt()` must add `f.origin` to its rune count so it returns content-absolute rune positions (not viewport-relative ones).

**Charofpt fix:**
```go
func (f *frameImpl) Charofpt(pt image.Point) int {
    // ...
    lines, originRune := f.layoutFromOrigin()  // was: f.layoutBoxes(...)
    // ... find line, count runes ...
    return originRune + runeCount  // was: just runeCount
}
```

**Ptofchar fix:**
```go
func (f *frameImpl) Ptofchar(p int) image.Point {
    // ...
    lines, originRune := f.layoutFromOrigin()  // was: f.layoutBoxes(...)
    // Adjust p to be relative to origin
    p -= originRune
    // ... rest of logic works with viewport-relative positions
}
```

The key insight: `layoutFromOrigin()` returns lines with Y-coordinates adjusted to start from 0, and also returns the origin rune offset. `Charofpt` needs to add that offset to its result; `Ptofchar` needs to subtract it from its input.

### Why This Wasn't Caught Earlier

Before Phase 20D, the preview never scrolled programmatically — users only scrolled via the scrollbar, which presumably triggers a full re-render. The `scrollPreviewToMatch()` function introduced in Phase 20D was the first code path that changed the origin without going through the normal scroll flow.

## Implementation Phases

### Phase 21A: Fix Charofpt/Ptofchar Origin Handling

This is the highest priority — it's a correctness bug that makes the frame unusable after any scroll.

1. Change `Charofpt()` to use `layoutFromOrigin()` and add `originRune` to result
2. Change `Ptofchar()` to use `layoutFromOrigin()` and subtract `originRune` from input
3. Update all tests that depend on these methods with scrolled content

### Phase 21B: Add Colored Sweep to Rich Frame

1. Add `sweepColor` field to `frameImpl`
2. Add `SelectWithColor()` and `SelectWithChordAndColor()` methods
3. Update `drawSelectionTo()` to check `sweepColor`
4. Update `HandlePreviewMouse` B2 handler to pass `global.but2col`
5. Update `HandlePreviewMouse` B3 handler to pass `global.but3col`
6. Update `HandlePreviewMouse` B1 handler to pass `global.but2col` for chord detection (SelectWithChordAndColor)

### Phase 21C: Add Cursor Warp After Look

1. Add `DefaultFontHeight()` method to rich frame (or expose font height through existing interface)
2. After successful B3 search + ToRendered mapping, add `MoveTo()` call
3. Use `Ptofchar(rendStart).Add(image.Pt(4, fontHeight-4))` matching normal Acme

### Phase 21D: Tests

1. Test `Charofpt`/`Ptofchar` with non-zero origin
2. Test colored sweep color override
3. Test cursor warp coordinates after Look
4. Test round-trip: scroll → click → correct character position
