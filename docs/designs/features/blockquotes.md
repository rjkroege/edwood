# Blockquote Design

## Requirement

From the design doc (`docs/markdown-design.md`):

> Blockquotes use `>` prefixes, optionally nested:
>
> ```
> > single level
> > > nested (depth 2)
> > > > depth 3
> ```

Blockquotes are currently listed as a known limitation ("Blockquotes are not
supported"). This phase adds full blockquote support: parsing, rendering with
left border bars, inline formatting within blockquotes, and source mapping.

## Syntax

Standard markdown blockquote syntax:

```
> This is a blockquote.
> It can span multiple lines.

> > This is a nested blockquote (depth 2).

> > > Depth 3.
```

### Rules

1. A blockquote line starts with `>` optionally followed by a space.
2. Nesting uses repeated `>` markers: `> > ` is depth 2, `> > > ` is depth 3.
3. Consecutive blockquote lines at the same depth form a single block.
4. A blank line or a line with fewer `>` markers ends the current block.
5. Inner content is re-parsed through the same block/inline pipeline, so
   blockquotes can contain paragraphs, headings, lists, code blocks, and
   further nested blockquotes.
6. Lazy continuation: a line that doesn't start with `>` but follows a
   blockquote line is **not** treated as continuation in this implementation
   (strict mode). Each blockquote line must have the `>` prefix. This matches
   the design doc's description and keeps the parser simple.

### Prefix Stripping

For a line like `> > hello`, the prefix is `> > ` (4 bytes: `>`, ` `, `>`, ` `).
The stripped inner content is `hello`.

For depth calculation:
- Count the number of `>` characters at the start of the line, allowing
  optional spaces between and after them.
- `>` → depth 1, prefix `> ` (2 bytes)
- `> >` → depth 2, prefix `> > ` (4 bytes) or `>> ` (3 bytes)
- `> > >` → depth 3, prefix `> > > ` (6 bytes) or `>>> ` (4 bytes)

The prefix length is the number of bytes consumed up to and including the
optional space after the last `>`.

## Style Fields

Add two fields to `rich.Style`:

```go
Blockquote      bool  // Span is inside a blockquote
BlockquoteDepth int   // Nesting level (1 = `>`, 2 = `> >`, …)
```

These are already documented in the design doc's style table and the design
doc's `rich.Style` definition. They just haven't been added to the actual
`rich/style.go` yet.

## Parsing

### Detection Function

```go
// isBlockquoteLine checks if a line starts with a blockquote marker.
// Returns: (isBlockquote bool, depth int, contentStart int)
// - isBlockquote: true if the line starts with `>`
// - depth: nesting level (number of `>` markers)
// - contentStart: byte index where inner content begins (after all `> ` prefixes)
func isBlockquoteLine(line string) (bool, int, int)
```

Algorithm:
1. Scan from the start of the line.
2. For each `>` found: increment depth, skip optional following space.
3. If no `>` found at position 0, return false.
4. `contentStart` is the byte position after the last `>` and its optional space.

### Integration into `Parse()` and `ParseWithSourceMap()`

Blockquote detection is added to the block-level dispatch in the main parsing
loop, after table detection and before heading/list/hrule detection (matching
the priority order in the design doc: fenced code → indented code → hrule →
table → **blockquote** → heading → list → blank → plain text).

When a blockquote line is detected:
1. Strip the `> ` prefix(es) to get the inner content.
2. Re-parse the inner content through `parseLine()` (for `Parse()`) or
   `parseLineWithSourceMap()` (for `ParseWithSourceMap()`).
3. Set `Blockquote=true` and `BlockquoteDepth=depth` on every span produced
   by the inner parse.
4. For multi-line blockquotes: consecutive blockquote lines at the same depth
   are joined as paragraph continuation (space between lines), just like normal
   paragraphs. A blank line within a blockquote (a line that is just `>` with
   no content) creates a paragraph break within the blockquote.

### Blockquote State Machine

The parser needs to track whether we're currently inside a blockquote to handle
multi-line blockquotes and paragraph breaks correctly. The approach:

- When a blockquote line is encountered and we're not in a blockquote (or the
  depth changes), start a new blockquote context.
- When a blockquote line continues at the same depth, treat it as paragraph
  continuation (same as normal text joining: space between lines).
- When a blank line appears (just `>` or `> ` with no content), emit a
  paragraph break span with `Blockquote=true`.
- When a non-blockquote line appears (or EOF), end the blockquote context.

For nested blockquotes, the inner content after stripping the outermost `>`
prefix is itself passed through blockquote detection. Since we strip all `>`
prefixes at once and record the full depth, there's no recursive re-parsing
needed — we handle all depths in a single pass.

**Important**: The design doc says "The stripped inner content is re-parsed
through the same block/inline pipeline, so blockquotes can contain paragraphs,
headings, lists, code blocks, and further nested blockquotes." For the initial
implementation, we support inline formatting within blockquote lines. Supporting
full block elements (headings, lists, code blocks) inside blockquotes is deferred
to Phase 8 (Nested Block Elements in Lists), which handles nested block parsing
more generally. For now, blockquote inner content is parsed as inline text with
blockquote styling applied.

### Applying Blockquote Style to Spans

After parsing the inner content, each resulting span gets `Blockquote=true` and
`BlockquoteDepth=depth` added to its style. This is analogous to how list item
parsing sets `ListItem=true` and `ListIndent` on content spans.

```go
// applyBlockquoteStyle sets blockquote fields on all spans.
func applyBlockquoteStyle(spans []rich.Span, depth int) {
    for i := range spans {
        spans[i].Style.Blockquote = true
        spans[i].Style.BlockquoteDepth = depth
    }
}
```

## Source Mapping

### PrefixLen for Blockquotes

Source map entries for blockquote content use `PrefixLen` to record the stripped
`> ` prefix, analogous to heading `# ` prefixes. The design doc states:

> Source map entries for blockquote content record the `> ` prefix in
> `PrefixLen`, analogous to heading `# ` prefixes. Nested quotes accumulate:
> a `> > ` prefix at depth 2 has `PrefixLen = 4` (two markers plus spaces).

For a line like `> > hello\n`:
- Source: `> > hello\n` (10 bytes)
- Rendered: `hello\n` (6 runes) with `Blockquote=true, BlockquoteDepth=2`
- Source map entry: `PrefixLen = 4` (for the `> > ` prefix)
- The `PrefixLen` here is the rune count of the prefix (since `> ` is all
  ASCII, bytes = runes).

### Source Map Entry Generation

In `parseLineWithSourceMap`, after the heading check and before the list check,
add blockquote detection:

```go
if isBQ, depth, contentStart := isBlockquoteLine(line); isBQ {
    return parseBlockquoteLineWithSourceMap(line, depth, contentStart, sourceOffset, renderedOffset)
}
```

The `parseBlockquoteLineWithSourceMap` function:
1. Computes `prefixLen = contentStart` (the number of bytes/runes in the `> ` prefix(es)).
2. Extracts the inner content: `content = line[contentStart:]`.
3. Parses inline formatting on the content with source map tracking.
4. Creates a source map entry with `PrefixLen = prefixLen` covering the full
   line (source) to content (rendered).
5. Sets `Blockquote=true` and `BlockquoteDepth=depth` on all produced spans.

For simple blockquote lines (no inline formatting), the source map entry looks
like a heading entry: one entry covering the entire line with `PrefixLen`
indicating the stripped prefix.

For blockquote lines with inline formatting (e.g., `> **bold** text`), the
inline parser generates multiple source map entries. The first entry's source
offset starts after the `> ` prefix. The `PrefixLen` approach doesn't directly
apply when inline formatting creates multiple entries. Instead, the approach is
to pass `sourceOffset + contentStart` to the inline parser, and create a
wrapping entry similar to how list items work: the bullet/prefix gets its own
entry, and the content entries start at `sourceOffset + contentStart`.

**Decision**: Follow the list item pattern rather than the heading pattern.
Headings create a single source map entry covering the whole line with
`PrefixLen`. List items create separate entries for the bullet and for content.
Blockquotes will create a separate entry for the `> ` prefix (mapping rendered
position of first content character back to include the prefix) and then let
the inline parser generate content entries starting at the content offset.

Actually, reconsidering: for simplicity and consistency with the heading model,
we can use the simpler approach for blockquotes that contain only plain text (no
inline formatting). For blockquotes with inline formatting, we follow the list
item model. But this creates two different code paths.

**Final decision**: Use the list item model uniformly. The `> ` prefix doesn't
produce rendered output (unlike the list bullet `•`), so we don't emit a
separate span for it. Instead, we just offset the source positions of the
content entries. This means:
- No explicit `PrefixLen` entry is needed in the source map entries from the
  inline parser.
- The source map entries for inline content use `sourceOffset + contentStart`
  as their source origin.
- For ToSource boundary expansion, when the selection covers the full rendered
  content of a blockquote line, the source range should expand to include the
  `> ` prefix. This is achieved by having the first content entry's source start
  positioned right after the prefix, and using the existing boundary expansion
  logic.

Wait — the existing heading source map uses `PrefixLen` specifically so that
`ToSource` and `ToRendered` can correctly account for the stripped prefix when
mapping positions. Without `PrefixLen`, a click at rendered position 0 within
a blockquote would map to source position `contentStart` instead of position 0
of the line. This is actually correct behavior — the `> ` is not rendered, so
clicking on the first rendered character should map to the first content
character in source.

For **boundary expansion** (selecting all rendered content should include `> `
in the source), the existing mechanism in `ToSource` handles this: when
`renderedStart == entry.RenderedStart` for a range selection, it snaps to
`entry.SourceRuneStart`, which would be the start of `> `. So we need the
source map entry to cover the full line including the prefix, with `PrefixLen`
set.

**Actual final decision**: Use the heading model with `PrefixLen`:
- For a simple blockquote line with no inline formatting, create one source map
  entry covering the full line, with `PrefixLen = contentStart`.
- For a blockquote line with inline formatting, the inline parser creates
  entries starting at `sourceOffset + contentStart`. We wrap these with a
  whole-line entry that has `PrefixLen = contentStart`, OR we adjust the first
  inline entry to have its `SourceStart` at `sourceOffset` with `PrefixLen`.

This is getting complex. Let's simplify:

**Simplest approach**: Parse the inner content with the inline parser, passing
`sourceOffset + contentStart` as the source origin. The inline parser generates
entries that cover only the content. Then post-process: set the first entry's
`SourceStart = sourceOffset` (start of line including `> ` prefix) and
`PrefixLen = contentStart`. This makes the first entry encompass the prefix,
which gives `ToSource` and `ToRendered` the information needed for correct
mapping and boundary expansion.

This matches the heading pattern exactly: the heading creates one entry with
`PrefixLen = level + 1` (for `# `). For blockquotes with inline formatting,
the first content entry gets the `PrefixLen`, and subsequent entries don't
need it.

## Rendering (Layout Engine)

### Indentation

From the design doc:

> Each blockquote depth level adds a left indent (same 20px quantum used for
> list indentation) plus a 2px vertical bar drawn in a muted color.

Each depth level adds `ListIndentWidth` (20px) of left indentation. At depth 2,
total indent is 40px. At depth 3, 60px. This stacks with any list indentation
if a list appears inside a blockquote.

### Changes to `layout()` in `rich/layout.go`

In the indentation calculation section, add blockquote indent handling:

```go
if box.Style.Blockquote {
    indentPixels += box.Style.BlockquoteDepth * ListIndentWidth
}
```

This stacks with list indent if both are present:

```go
indentPixels := 0
if box.Style.Blockquote {
    indentPixels += box.Style.BlockquoteDepth * ListIndentWidth
}
if box.Style.ListBullet || box.Style.ListItem {
    indentPixels += box.Style.ListIndent * ListIndentWidth
} else if (box.Style.Block && box.Style.Code) || box.Style.Table || box.Style.Image {
    indentPixels = gutterIndent // gutter overrides for scrollable blocks
}
```

### Vertical Bar (Border)

The vertical bar for each blockquote depth level is drawn during frame rendering
in `rich/frame.go`. Each level gets a 2px wide vertical bar at the left edge of
its indent zone, in a muted gray color.

#### New Constant

```go
// BlockquoteBorderWidth is the width in pixels of the blockquote vertical bar.
const BlockquoteBorderWidth = 2

// BlockquoteBorderGray is the color of the blockquote vertical border bar.
var BlockquoteBorderGray = color.RGBA{R: 200, G: 200, B: 200, A: 255}
```

#### Border Drawing

Add a `drawBlockquoteBorders` function called during line rendering:

```go
func (f *frameImpl) drawBlockquoteBorders(target draw.Image, line Line, offset image.Point) {
    // Find blockquote depth from boxes on this line
    depth := 0
    for _, pb := range line.Boxes {
        if pb.Box.Style.Blockquote && pb.Box.Style.BlockquoteDepth > depth {
            depth = pb.Box.Style.BlockquoteDepth
        }
    }
    if depth == 0 {
        return
    }

    // Draw a 2px vertical bar for each depth level
    for level := 1; level <= depth; level++ {
        barX := offset.X + (level-1)*ListIndentWidth + 2 // small offset from left edge
        barRect := image.Rect(
            barX,
            offset.Y + line.Y,
            barX + BlockquoteBorderWidth,
            offset.Y + line.Y + line.Height,
        )
        // Draw the border bar
        borderImg := f.allocColorImage(BlockquoteBorderGray)
        if borderImg != nil {
            target.Draw(barRect, borderImg, nil, image.ZP)
        }
    }
}
```

This is called from the line rendering loop in `drawTo`, for each line that
contains blockquote content.

### Text Wrapping

Text inside blockquotes wraps within the reduced width
(`frameWidth - blockquoteIndent`). This happens naturally because the layout
engine's wrapping logic uses `effectiveFrameWidth = frameWidth - indentPixels`.
The existing indentation and wrapping code handles this correctly once
`indentPixels` accounts for `BlockquoteDepth`.

## Interaction with Other Elements

### Blockquote + Inline Formatting

Inline formatting (bold, italic, code, links) works inside blockquotes because
the inner content is parsed through `parseInline()` with a base style that has
`Blockquote=true` and `BlockquoteDepth` set. The inline parser preserves the
base style's blockquote fields.

### Blockquote + Lists (Future: Phase 8)

A list inside a blockquote would have both `Blockquote=true` with
`BlockquoteDepth` and `ListItem=true` with `ListIndent`. The indentation
stacks: blockquote indent + list indent. This is handled by the additive
indentation logic in `layout()`.

### Blockquote + Code Blocks (Future: Phase 8)

A fenced code block inside a blockquote would need the parser to recognize
fence delimiters within blockquote context. This is deferred to Phase 8.

### Blockquote + Paragraph Breaks

A blank blockquote line (`>` or `> ` with no content after the prefix) creates
a paragraph break within the blockquote. The paragraph break span should have
`Blockquote=true` and `BlockquoteDepth` set so the border bars continue through
the break.

## Testing Strategy

### Parse Tests (`markdown/parse_test.go`)

1. **Single-line blockquote**: `> hello` → span with text "hello",
   `Blockquote=true`, `BlockquoteDepth=1`.
2. **Multi-line blockquote**: `> line1\n> line2` → joined text "line1 line2"
   (paragraph continuation).
3. **Nested blockquote**: `> > inner` → `BlockquoteDepth=2`.
4. **Triple nested**: `> > > deep` → `BlockquoteDepth=3`.
5. **Blockquote with inline formatting**: `> **bold** text` → bold span +
   plain span, both with `Blockquote=true`.
6. **Blockquote with paragraph break**: `> para1\n>\n> para2` → two content
   groups separated by ParaBreak span, all with blockquote styling.
7. **Empty blockquote**: `>` or `> ` → no content span, possibly just a
   paragraph break or empty span.
8. **Blockquote followed by other blocks**: `> quote\n\nparagraph` → blockquote
   span followed by normal paragraph span.
9. **Blockquote followed by heading**: `> quote\n# Heading` → blockquote span
   followed by heading span.
10. **Blockquote with link**: `> [click](url)` → link span with blockquote
    styling.

### Source Map Tests (`markdown/sourcemap_test.go`)

1. **Simple blockquote ToSource**: Click in rendered "hello" from `> hello`
   maps to correct source position (past the `> ` prefix).
2. **Blockquote boundary expansion**: Selecting all of "hello" in rendered
   output maps source range to include `> ` prefix.
3. **Nested blockquote**: Position mapping through `> > ` double prefix.
4. **ToRendered**: Source position within `> hello` maps to correct rendered
   position.
5. **Blockquote with formatting**: Source mapping for `> **bold**` correctly
   handles both `> ` prefix and `**` markers.

### Layout Tests (`rich/layout_test.go`)

1. **Blockquote indentation**: Depth 1 → 20px indent, depth 2 → 40px indent.
2. **Text wrapping within blockquote**: Content wraps at
   `frameWidth - depth*ListIndentWidth`.
3. **Blockquote + list stacking**: A list item inside a blockquote has
   combined indent.

### Rendering Tests

Blockquote border rendering is visual and harder to test in unit tests.
Manual verification is appropriate for the border bar drawing.

## Summary of File Changes

### `rich/style.go`
- Add `Blockquote bool` field to `Style` struct
- Add `BlockquoteDepth int` field to `Style` struct

### `rich/layout.go`
- Add `BlockquoteBorderWidth` constant (2)
- Add blockquote indentation logic in `layout()`: `BlockquoteDepth * ListIndentWidth`
- Blockquote indent stacks additively with list indent

### `rich/frame.go`
- Add `BlockquoteBorderGray` color variable
- Add `drawBlockquoteBorders()` function
- Call `drawBlockquoteBorders()` from line rendering loop

### `markdown/parse.go`
- Add `isBlockquoteLine()` detection function
- Add blockquote dispatch in `Parse()` main loop
- Add `applyBlockquoteStyle()` helper

### `markdown/sourcemap.go`
- Add blockquote dispatch in `ParseWithSourceMap()` main loop
- Add `parseBlockquoteLineWithSourceMap()` function
- Source map entries use `PrefixLen` for the `> ` prefix (heading model)

## Risks and Mitigations

- **No lazy continuation**: This implementation requires `>` on every line.
  Some markdown parsers support "lazy" continuation where a line without `>`
  continues the blockquote. We can add this later if needed.
- **No nested block elements**: Headings, lists, and code blocks inside
  blockquotes are deferred to Phase 8. For now, blockquote inner content is
  parsed as inline text only.
- **Style comparison**: Adding `Blockquote` and `BlockquoteDepth` fields to
  `Style` affects style equality comparisons, which are used for span merging.
  Two spans with different `BlockquoteDepth` won't merge, which is correct
  behavior.
