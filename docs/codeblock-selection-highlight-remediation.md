# Code Block Selection Highlight Remediation

## Problem

Two related rendering bugs affect the Markdeep preview's rich text frame:

1. **B1 sweep selection is invisible in code blocks.** Dragging to select
   text inside a fenced code block produces no visible highlight.

2. **B2 and B3 sweep colors are washed out** compared to the regular
   (non-rich) text editor. The red (B2) and green (B3) sweep indicators
   appear faded / desaturated in the rich preview.

Both symptoms trace to the same area of `rich/frame.go`.

## Root Cause

### Drawing order (B1 invisible in code blocks)

`Redraw()` renders in this order:

```
1. Background fill                        (line 788)
2. Selection highlight  (drawSelectionTo)  (line 792)
3. Text content         (drawTextTo)       (line 797)
     Phase 1 — block-level backgrounds (fenced code blocks)
     Phase 2 — box backgrounds (inline code spans)
     Phase 3 — horizontal rules
     Phase 4 — text glyphs
     Phase 5 — images
     ...
```

The selection highlight is painted at step 2, then code-block backgrounds
(Phase 1) and inline-code backgrounds (Phase 2) paint **over** the
selection at step 3, completely hiding it.

### Pre-multiplied alpha double-application (B2/B3 washed out)

Every opaque color fill in the file uses the Plan 9 `Draw` call with the
**same image as both source and mask**:

```go
target.Draw(rect, bgImg, bgImg, image.ZP)   // src == mask
```

In Plan 9's compositing model:

```
dst = src × mask_alpha + dst × (1 − mask_alpha)
```

When `src == mask`, the source's pre-multiplied RGB values are multiplied
by the alpha channel **a second time** (double pre-multiplication).  For
fully opaque images (`A = 255`) the arithmetic is `×1.0` so there is no
visible error.  However, the color images are allocated with the
**screen's pixel format** (`display.ScreenImage().Pix()`), which may
lack a dedicated alpha channel.  In that case the draw compositor uses
a channel value (e.g. the highest or a designated channel) as the mask,
producing a mask value of `≈ 245/255 ≈ 0.96` for InlineCodeBg instead
of `1.0`.  The resulting `4 %` bleed-through makes underlying colors
appear washed out rather than fully replaced.

The correct pattern for an opaque overwrite is a **nil mask**:

```go
target.Draw(rect, bgImg, nil, image.ZP)      // straight copy
```

This is already used correctly in two places:

- Selection highlight (`drawSelectionTo`, line 1400)
- Image blitting (`drawImageTo`, line 2085)

## Affected Call Sites

| Line | Current call | Purpose |
|------|-------------|---------|
| 788 | `scratch.Draw(drawRect, f.background, f.background, …)` | Frame background fill |
| 1075 | `target.Draw(gutterRect, f.background, f.background, …)` | Gutter repaint |
| 1120 | `target.Draw(bgRect, bgImg, bgImg, …)` | Fenced code block background |
| 1148 | `target.Draw(bgRect, bgImg, bgImg, …)` | Inline code background |
| 1182 | `target.Draw(ruleRect, ruleImg, ruleImg, …)` | Horizontal rule |
| 1901 | `target.Draw(bgRect, bgImg, bgImg, …)` | Scrollbar background |
| 1937 | `target.Draw(thumbRect, thumbImg, thumbImg, …)` | Scrollbar thumb |

## Fix

### 1. Replace `src, src` with `src, nil` in all opaque Draw calls

Change every call site listed above from:

```go
target.Draw(rect, img, img, image.ZP)
```

to:

```go
target.Draw(rect, img, nil, image.ZP)
```

This eliminates the double pre-multiplication and ensures a true opaque
overwrite regardless of the image's pixel format.

### 2. Move selection drawing after background phases

Move the selection highlight rendering from `Redraw()` (between
background fill and `drawTextTo`) into `drawTextTo()` itself, placed
between Phase 2 (box backgrounds) and Phase 3 (horizontal rules).

This ensures:

- Code block and inline code backgrounds are drawn **first**
- Selection highlight paints **on top** of those backgrounds
- Text glyphs still render **on top** of the highlight

New `drawTextTo` phase order:

```
Phase 1  — block-level backgrounds
Phase 2  — box backgrounds (inline code)
Phase 2b — selection highlight  ← NEW LOCATION
Phase 3  — horizontal rules
Phase 4  — text glyphs
Phase 5  — images
Phase 5b — gutter repaint
Phase 6  — horizontal scrollbars
```

`drawTextTo()` needs access to the selection state (`f.p0`, `f.p1`,
`f.selectionColor`, `f.sweepColor`) which are already fields on
`frameImpl`, so no interface changes are required.

## Testing

- Existing `rich/frame_test.go` and `rich/select_test.go` tests
  continue to pass.
- Manual verification: B1 sweep in a fenced code block shows the
  Darkyellow highlight; B2/B3 sweep colors appear saturated and
  match the regular editor.
