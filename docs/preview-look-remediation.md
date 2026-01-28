# Preview Look (B3) Remediation Plan

## Problem Summary

When right-clicking (B3/Look) a word in Markdeep preview mode, the search finds the next occurrence but:

1. **Highlights in source body, not preview**: `search()` calls `w.body.Show(q0, q1, true)` which updates `w.body.q0/q1` and calls `SetSelect()`/`DrawSel()` on the source body frame — not the preview.
2. **Source body bleeds through**: The source body frame's selection highlight becomes visible underneath/over the preview rendering.
3. **No scroll to match**: The preview doesn't scroll to show the found match.

## Current Flow (Broken)

```
B3 click in preview (wind.go:1045-1135)
  → Select/expand word in rendered text
  → syncSourceSelection()             # preview→source position sync
  → w.Draw()                          # renders preview (old selection)
  → PreviewLookText()                 # extracts RENDERED text
  → search(&w.body, renderedText)     # searches SOURCE buffer for rendered text
    → w.body.Show(q0, q1, true)       # highlights in SOURCE body frame
      → w.body.SetSelect(q0, q1)      # sets source frame selection
        → w.body.fr.DrawSel(...)      # DRAWS ON SOURCE FRAME (bleeds through!)
      → w.body.ScrDraw(...)           # no-op in preview mode
  → (no preview selection update)
  → (no preview scroll)
```

### Bug Details

**Bug 1 — Wrong search target**: `PreviewLookText()` returns rendered text (markdown formatting stripped). This is then searched in the source buffer which contains raw markdown. For plain words this accidentally works, but for formatted text ("bold" won't match "**bold**") it silently fails.

**Bug 2 — Highlight in wrong frame**: `search()` calls `w.body.Show()` which calls `w.body.SetSelect()` → `w.body.fr.DrawSel()`. This draws the highlight on the source body frame, which is underneath the preview. The source frame is never made invisible — it's just painted over by the preview render. When `Show()` draws on it after the preview render, those pixels become visible.

**Bug 3 — No preview update**: After `search()` updates `w.body.q0/q1`, nothing maps those positions back to rendered coordinates, and nothing updates `w.richBody.SetSelection()` or scrolls the preview.

## Design Decisions

- **Search the source markdown** (the canonical content) — the preview is just a view of the same underlying file
- **Display results in the rendered view** — requires a source→rendered reverse mapping
- **Suppress source body rendering** in preview mode — `Show()` and `SetSelect()` on `w.body` should not draw when preview is active
- **Scroll the preview** to show the match, analogous to how normal Acme scrolls the body on Look

## Required: Source→Rendered Reverse Mapping

The existing `SourceMap.ToSource(renderedStart, renderedEnd)` maps rendered→source. We need the reverse: given source byte positions from `search()`, find the corresponding rendered rune positions.

### ToRendered() Design

```go
// ToRendered maps a range in source markdown (srcStart, srcEnd as rune positions)
// to the corresponding range in the rendered content (as rune positions).
// Returns (-1, -1) if no mapping exists.
func (sm *SourceMap) ToRendered(srcStart, srcEnd int) (renderedStart, renderedEnd int)
```

The SourceMapEntry already stores both sides:
```go
type SourceMapEntry struct {
    RenderedStart int  // Rune position in rendered content
    RenderedEnd   int
    SourceStart   int  // Byte position in source markdown
    SourceEnd     int
    PrefixLen     int  // Length of source prefix not in rendered
}
```

`ToRendered()` finds entries whose source range contains `[srcStart, srcEnd]` and calculates the rendered offset within those entries. This is the mirror of `ToSource()`.

**Complication**: Source positions from `search()` are in runes (via `file.ByteTuple().R`), but `SourceMapEntry.SourceStart/End` are byte positions. We need to handle the rune↔byte conversion. The simplest approach: accept rune positions and convert using the source text, or store rune positions in the entries.

## Fix Plan

### Phase 20A: Add ToRendered() to SourceMap

Add `SourceMap.ToRendered(srcRuneStart, srcRuneEnd int) (renderedStart, renderedEnd int)` method.

This requires either:
- (a) Storing source rune positions alongside byte positions in SourceMapEntry, or
- (b) Passing the source text to ToRendered() so it can convert bytes↔runes

Option (a) is cleaner — add `SourceRuneStart`, `SourceRuneEnd` fields to `SourceMapEntry`, populated during parse.

### Phase 20B: Suppress Source Body Rendering in Preview Mode

When `w.IsPreviewMode()`, `Text.Show()` should not call `SetSelect()`/`DrawSel()` on the source body frame. Instead it should only update `w.body.q0/q1` (the logical selection) without drawing.

Options:
- (a) Guard `Show()` with `IsPreviewMode()` check — skip the draw-related calls
- (b) Create a `previewSearch()` that calls `search()` logic but skips `Show()`

Option (a) is simpler: in `Text.Show()`, when `t.w.IsPreviewMode() && t.what == Body`, update `t.q0/q1` but skip `fr.DrawSel()` and scroll operations.

### Phase 20C: Implement Preview-Aware Look

Rewrite the B3 search path in `HandlePreviewMouse` to:

1. Get the search text from the **source** buffer (using `w.body.q0/q1` after `syncSourceSelection()`)
2. Call `search(&w.body, sourceText)` — this finds the next match in source and sets `w.body.q0/q1`
3. Map result back: `rendStart, rendEnd := w.previewSourceMap.ToRendered(w.body.q0, w.body.q1)`
4. Update preview selection: `w.richBody.SetSelection(rendStart, rendEnd)`
5. Scroll preview to show the match (see Phase 20D)
6. Redraw: `w.Draw()` + flush

```
B3 click in preview (fixed)
  → Select/expand word in rendered text
  → syncSourceSelection()              # preview→source position sync
  → Read source text at w.body.q0..q1  # get SOURCE text to search for
  → search(&w.body, sourceText)        # search source buffer
    → w.body.Show() (suppressed draw)  # updates q0/q1 only, no DrawSel
  → ToRendered(w.body.q0, w.body.q1)   # source→rendered mapping
  → richBody.SetSelection(rStart,rEnd) # highlight in PREVIEW
  → Scroll preview to show match       # Phase 20D
  → w.Draw() + flush                   # redraw preview with highlight
```

### Phase 20D: Scroll Preview to Match

When the found match is outside the currently visible area, scroll the preview so the match is visible.

The preview's `SetOrigin(org)` sets the rune offset of the first visible content. To scroll to a match at rendered position `rendStart`:

1. Check if `rendStart` is within the currently visible range: `origin <= rendStart < origin + visibleRunes`
2. If not visible, set origin to place `rendStart` roughly 1/3 from the top (matching Acme's `Show()` behavior)
3. Redraw after scroll

This may require a helper to estimate visible content length, similar to `VisibleLines()`.

### Phase 20E: Update Tests

- Update `TestPreviewB3Search` to verify the search result appears in the preview selection (not just source `q0/q1`)
- Add `TestPreviewB3SearchScroll` to verify the preview scrolls to the match
- Add `TestPreviewB3SearchNoBleed` to verify the source frame doesn't draw selection highlights in preview mode
- Add `TestSourceMapToRendered` unit tests for the new reverse mapping
- Verify round-trip: `ToSource(ToRendered(src))` ≈ original source positions

## Files Affected

| File | Change |
|------|--------|
| `markdown/sourcemap.go` | Add `SourceRuneStart/End` fields; add `ToRendered()` method |
| `markdown/sourcemap_test.go` | Tests for `ToRendered()` |
| `markdown/parse.go` | Populate `SourceRuneStart/End` during parse |
| `text.go` | Guard `Show()` drawing in preview mode |
| `wind.go` | Rewrite B3 search path in `HandlePreviewMouse` |
| `wind_test.go` | Updated B3/Look tests |

## Relationship to Phase 19

Phase 19 (chord undo remediation) fixed Cut/Paste/Snarf chords to use high-level primitives. This phase fixes Look (B3) to properly display results in the preview. Both address the same root issue: preview mode operations incorrectly interacting with the source body's display layer.
