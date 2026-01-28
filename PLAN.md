# Rich Text Implementation Plan

For completed phases 1-22, see `PLAN_ARCHIVE.md`.

## Status Legend
- `[ ]` = not done
- `[x]` = done

---

## Current Task

**Phase 23**: Cursor bar for rich text frame

## Phase 23: Cursor Bar for Rich Text Frame

The Markdeep preview has no visible insertion cursor. Regular Edwood windows show a black tick (vertical bar with serif boxes) when the selection is a point (`p0 == p1`). The rich frame should match this, with cursor height scaled to the tallest of the two adjacent boxes.

See `docs/cursor-bar-design.md` for full design.

### Design Summary

- **Tick appearance**: Vertical line with serif boxes at top/bottom, matching `frame/tick.go`
- **Height rule**: Tallest of the two adjacent boxes (not line height)
- **No save/restore**: Rich frame fully redraws via scratch image, so tick is drawn each pass
- **All code in `rich/frame.go`**: No new files

---

### Phase 23A: Add Tick Fields and initTick Method

Add `tickImage`, `tickScale`, `tickHeight` fields to `frameImpl`. Port tick image creation from `frame/tick.go:InitTick()` — allocate transparent image, draw vertical line and serif boxes. Re-create when height changes.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestInitTickCreatesImage, TestInitTickReusesForSameHeight in rich/frame_test.go; stub initTick() and tick fields added to frameImpl |
| Code written | [x] | Add fields to `frameImpl`, implement `initTick(height int)` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 24d05a0 |

### Phase 23B: Add boxHeight Helper

Add `boxHeight(box Box) int` method that returns the height of a box: `fontForStyle(box.Style).Height()` for text/special boxes, or the image's scaled height for image boxes.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestBoxHeightBody, TestBoxHeightHeading, TestBoxHeightImage in rich/frame_test.go |
| Code written | [x] | Implement `boxHeight()` on `frameImpl` — returns font height for text, scaled image height for images |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit d6cca86 |

### Phase 23C: Add drawTickTo Method

Implement `drawTickTo(target, offset)`: walk layout lines/boxes to find cursor position, compute X coordinate (same logic as `Ptofchar`), determine height from tallest adjacent box, call `initTick(height)`, draw tick via `target.Draw(rect, display.Black(), tickImage, image.ZP)`.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestDrawTickAtCursor, TestNoTickWithSelection, TestTickHeightScaling, TestTickHeightBodyText in rich/frame_test.go |
| Code written | [x] | Implement `drawTickTo()` on `frameImpl` |
| Tests pass | [x] | go test ./rich/... passes |
| Code committed | [x] | Commit 7e6e16f |

### Phase 23D: Wire into Redraw

Call `drawTickTo()` from `Redraw()` after text drawing, when `p0 == p1`. This draws the cursor on top of text in the scratch image before blit to screen.

| Stage | Status | Notes |
|-------|--------|-------|
| Tests exist | [x] | TestRedrawDrawsTickWhenCursorPoint, TestRedrawNoTickWhenSelection in rich/frame_test.go |
| Code written | [x] | Add conditional call in `Redraw()` after `drawTextTo()` |
| Tests pass | [x] | go test ./... passes |
| Code committed | [x] | Commit 373c8c4 |

---

## Future Enhancements (Post Phase 23)

- **Blockquotes**: `>` syntax with indentation and vertical bar
- **Task lists**: `- [ ]` and `- [x]` checkbox syntax
- **Definition lists**: `term : definition` syntax
- **Syntax highlighting**: Language-aware code block coloring
- **Table cell spanning**: Complex table layouts
- **Multi-line list items**: Proper continuation handling
- **Footnotes**: `[^1]` reference syntax
- **Animated GIF support**: Display animations

---

## Test Summary

| Suite | Count | File Location |
|-------|-------|---------------|
| Style | 2+ | rich/style_test.go |
| Span | 3 | rich/span_test.go |
| Box | 2 | rich/box_test.go |
| Frame Init | 2 | rich/frame_test.go |
| Layout | 4+ | rich/layout_test.go |
| Coordinates | 4 | rich/coords_test.go |
| Selection | 3 | rich/select_test.go |
| Scrolling | 3 | rich/scroll_test.go |
| Markdown | 8+ | markdown/parse_test.go |
| Lists | TBD | markdown/parse_test.go (Phase 15A) |
| Tables | TBD | markdown/parse_test.go (Phase 15B) |
| Images | TBD | markdown/parse_test.go (Phase 15C) |
| Integration | 4 | richtext_test.go |
| **Total** | **~35+** | |

## How to Run Tests

```bash
# All rich text tests
go test ./rich/... ./markdown/...

# With verbose output
go test -v ./rich/...

# Specific package
go test ./rich/
```

## Files

| File | Purpose |
|------|---------|
| docs/richtext-design.md | Design document and architecture |
| docs/codeblock-design.md | Code block shading design (Phase 13) |
| docs/preview-resize-design.md | Preview resize bug analysis and options |
| docs/single-rect-owner.md | Single rectangle owner implementation plan (Phase 14) |
| docs/tables-lists-images-design.md | Tables, lists, and images design (Phase 15) |
| docs/image-rendering-design.md | Image rendering design (Phase 16) |
| docs/chord-undo-remediation.md | Chord undo bypass analysis and fix design (Phase 19) |
| docs/preview-look-remediation.md | Preview Look (B3) display and bleed-through fix design (Phase 20) |
| docs/preview-interaction-fixes.md | Coordinate mapping, colored sweep, cursor warp fixes (Phase 21) |
| docs/cursor-bar-design.md | Cursor bar for rich text frame (Phase 23) |
| PLAN.md | This file - implementation tracking |
| PLAN_ARCHIVE.md | Completed phases 1-22 |
| rich/style.go | Style type definition |
| rich/span.go | Span and Content types |
| rich/box.go | Box type for layout |
| rich/frame.go | RichFrame implementation |
| rich/select.go | Selection handling |
| rich/options.go | Functional options |
| markdown/parse.go | Markdown to Content parser |
| richtext.go | RichText component |
| preview.go | Preview window integration |
