# Block Gutter Design

## Requirement

From the design doc (`docs/markdown-design.md`):

> All scrollable objects must be indented by at least 8ems of space to leave a
> clean, non-scrolling gutter on the left of the page.

The gutter serves two purposes:
1. **Vertical scrollbar pass-through**: Mouse events in the gutter area are not
   captured by horizontal scroll regions, allowing vertical scroll gestures to
   pass through unimpeded.
2. **Visual separation**: A clean left margin keeps scrollable content visually
   separated from the window edge and scrollbar.

## Current State

### Code Blocks (Block && Code)

Code blocks **do** have indentation, but it's based on `CodeBlockIndentChars`
(4) times the M-width of the code font:

- `layout.go:162-168`: `CodeBlockIndentChars = 4`, `CodeBlockIndent = 40` (fallback).
- `layout.go:403-407`: Actual indent computed as `4 * codeFont.BytesWidth("M")`.
- `layout.go:497-499`: Applied at layout time for `Block && Code` boxes.
- `frame.go:1059-1082`: Phase 5b repaints the gutter column `[0, LeftIndent)`
  with the frame background when a block is horizontally scrolled, preventing
  text overflow into the gutter.
- `frame.go:1090-1137`: `drawBlockBackgroundTo` draws the code block background
  starting from `leftIndent`, not from x=0.

**Problem**: The current indent is 4em (4 M-widths), not the required 8em.

### Tables (Table)

Tables have **no indentation**. They start at x=0:

- `layout.go:492-500`: The indentation logic only checks for `ListBullet`,
  `ListItem`, and `Block && Code`. Table-styled boxes fall through with
  `indentPixels = 0`.
- `layout.go:357-372`: `lineBlockKind` correctly identifies table lines as
  `BlockTable`, so they get horizontal scrollbar support and block region
  detection.
- `layout.go:225-234`: `blockLeftIndent` returns x=0 for table lines because
  the first table box is at position 0.

**Problem**: No gutter indent at all.

### Images (Image)

Images have **no indentation**. They start at x=0:

- `layout.go:511-513`: Image width is computed, but no indent is applied.
- `layout.go:519-523`: Images skip wrapping and are positioned at current
  `xPos`, which is 0 at line start.
- `layout.go:366-369`: `lineBlockKind` correctly identifies image lines as
  `BlockImage`.

**Problem**: No gutter indent at all.

## Design

### Defining "8em" in Pixels

An "em" is the width of the capital letter "M" in the current font. For
scrollable blocks, the reference font is:

- **Code blocks**: The code font (`codeFont`), since that's the font used
  within the block.
- **Tables**: The base font, since table content uses the base font.
- **Images**: The base font, since the gutter is measured against the
  surrounding text context.

The indent in pixels is: `8 * font.BytesWidth([]byte("M"))`.

With a typical M-width of 10px, this gives 80px. With the current code font,
M-width is often ~8-10px, giving 64-80px.

### Changes to `layout.go`

#### 1. New Constant: `GutterIndentChars`

Replace the current `CodeBlockIndentChars = 4` with a unified constant:

```go
// GutterIndentChars is the number of 'M' characters to indent all scrollable
// block elements (code blocks, tables, images). This provides a non-scrolling
// gutter on the left for vertical scroll pass-through.
const GutterIndentChars = 8
```

The existing `CodeBlockIndentChars` and `CodeBlockIndent` constants become
aliases or are replaced:

```go
const CodeBlockIndentChars = GutterIndentChars
const CodeBlockIndent = 80 // GutterIndentChars * 10 (typical M-width fallback)
```

#### 2. Compute Gutter Indent for All Block Types

In the `layout()` function, the indentation logic currently only applies to
`Block && Code`. Extend it to also cover `Table` and `Image` boxes:

```go
indentPixels := 0
if box.Style.ListBullet || box.Style.ListItem {
    indentPixels = box.Style.ListIndent * ListIndentWidth
} else if box.Style.Block && box.Style.Code {
    indentPixels = codeBlockIndent  // already 8 * M-width after constant change
} else if box.Style.Table {
    indentPixels = codeBlockIndent  // same gutter width
} else if box.Style.Image {
    indentPixels = codeBlockIndent  // same gutter width
}
```

The variable name `codeBlockIndent` may be renamed to `gutterIndent` for
clarity, since it now applies to all scrollable block types. The computation
stays the same: `GutterIndentChars * font.BytesWidth([]byte("M"))` (using
codeFont for code blocks, base font otherwise — but since both should use the
same gutter width for visual consistency, we compute one value using the base
font, or the code font if available).

**Decision**: Use a single `gutterIndent` value computed from the **base font**
for all block types. This ensures all scrollable blocks have a visually
consistent left margin, regardless of their internal font. The code block
indent was previously computed from the code font, but switching to the base
font for consistency is acceptable since the difference is typically small and
the requirement is a minimum of 8em.

#### 3. Table and Image Non-Wrapping

Tables already don't wrap (they're handled by the box layout without
word-wrapping). However, the layout code's non-wrapping fast path
(`layout.go:518-524`) currently only checks `(Block && Code) || IsImage()`.
Tables need the same treatment. Currently table boxes go through the normal
wrapping path, which works because table content is pre-formatted with
box-drawing characters.

Since tables are pre-formatted and their ContentWidth is already tracked
correctly (the ContentWidth computation at `layout.go:594-610` checks for
`Block && Code` and `IsImage()` but not `Table`), we need to also track
ContentWidth for table lines so that horizontal scrollbar detection works
properly for tables with indent.

**Changes needed**:
- Add `Table` to the non-wrapping fast path: `(Block && Code) || IsImage() || Table`
- Add `Table` to the ContentWidth computation: the `isNonWrap` check needs
  `pb.Box.Style.Table`

Wait — looking more carefully at the existing code: tables are already tracked
as block regions by `lineBlockKind` (returns `BlockTable`), and
`findBlockRegions` already picks them up. The `ContentWidth` computation at
lines 594-610 currently only marks `isNonWrap` for `Block && Code` and image
boxes. If tables also need horizontal scrolling with the new indent, their
`ContentWidth` needs to be computed too. Let's add `Table` to the `isNonWrap`
check.

Actually, re-examining: table boxes do go through the normal wrapping logic
today, which is fine because the parser pre-formats table lines to fit. But
once we add an indent, the effective width is reduced, and long table rows may
exceed it. The tables should NOT wrap — they should overflow horizontally like
code blocks and get a horizontal scrollbar.

So tables need to be added to the non-wrapping fast path alongside code blocks
and images.

#### 4. Gutter Repaint for Tables and Images

The existing Phase 5b gutter repaint in `frame.go:1059-1082` already works for
any block region with `LeftIndent > 0` and a horizontal scroll offset. Once
tables and images have `LeftIndent > 0` (because `blockLeftIndent()` will find
their first box at the new indent position), the repaint will automatically
apply to them.

No changes needed in the gutter repaint logic.

#### 5. Block Background for Tables

Currently `drawBlockBackgroundTo` only fires for boxes with `Block && Bg` set.
Tables don't have `Block=true` or `Bg` set, so they don't get a full-width
background. This is fine — tables use box-drawing characters for their borders
and don't need a background fill. No change needed here.

### Changes to `frame.go`

#### `drawBlockBackgroundTo`

The code block background drawing already uses `leftIndent` from the first
block-styled box's X position. With the new indent, code block backgrounds
will correctly start at the 8em position. No changes needed.

#### `computeCodeBlockIndent`

Rename to `computeGutterIndent` (or keep the name — it's internal). Update the
multiplier from `CodeBlockIndentChars` (which will now be 8) so it stays
consistent.

### Summary of File Changes

**`rich/layout.go`:**
1. Change `CodeBlockIndentChars` from 4 to 8 (or introduce `GutterIndentChars = 8`)
2. Update `CodeBlockIndent` from 40 to 80
3. In `layout()`, rename `codeBlockIndent` to `gutterIndent`, compute as
   `GutterIndentChars * baseFont.BytesWidth("M")`
4. Extend indentation logic to apply `gutterIndent` to `Table` and `Image` boxes
5. Add `Table` to the non-wrapping fast path (`layout.go:518-524`)
6. Add `Table` to the `ContentWidth` computation (`layout.go:594-610`)

**`rich/frame.go`:**
1. Rename `computeCodeBlockIndent` to reflect broader scope (optional, cosmetic)
2. No functional changes needed — gutter repaint already handles any block
   region with `LeftIndent > 0`

### Testing Strategy

Extend `rich/layout_test.go`:

1. **Code block indent**: Update existing `TestLayoutCodeBlockIndent` to expect
   8em (8 * M-width) instead of 4em.
2. **Table indent**: New test verifying table boxes are positioned at 8em indent.
3. **Image indent**: New test verifying image boxes are positioned at 8em indent.
4. **Table ContentWidth**: Verify `ContentWidth` is computed for table lines.
5. **Table non-wrapping**: Verify table content overflows horizontally rather
   than wrapping when it exceeds `frameWidth - gutterIndent`.
6. **Block region detection**: Verify `findBlockRegions` still correctly
   identifies code, table, and image regions after the changes.

### Risks and Mitigations

- **Existing tests**: The `CodeBlockIndentChars` change from 4 to 8 will break
  existing tests that hard-code `expectedIndent = 4 * M-width`. These tests
  need updating to use the new constant value.
- **Visual regression**: Doubling the code block indent from 4em to 8em is a
  significant visual change. The design doc explicitly requires 8em, so this is
  intentional.
- **Table wrapping change**: Moving tables to the non-wrapping path changes
  behavior for tables wider than the frame. Previously they would wrap (or more
  likely break badly); now they overflow and get a scrollbar. This is the
  desired behavior per the design doc.
