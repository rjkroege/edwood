# Nested Block Elements in Lists

## Requirement

From the design doc (`docs/markdown-design.md`), Known Issue #4:

> **Limited nested block elements**: Lists cannot contain code blocks or
> tables. Blockquotes support nested inner content (paragraphs, headings,
> lists, code blocks, nested blockquotes) but list->code-block and
> list->table nesting is not yet supported.

This phase adds support for code blocks and blockquotes nested inside list
items. Tables inside list items are deferred (uncommon in practice and the
current table parser's line-oriented design makes nesting complex).

## Current Architecture

### Parser Design

The parser (`markdown/parse.go`) operates in a single pass over lines.
The main loop in `Parse()` and `ParseWithSourceMap()` iterates line by
line with this priority order:

1. Fenced code block open/close (`` ``` ``)
2. Inside fenced block -> accumulate verbatim
3. List item check (early, to prevent indented code misidentification)
4. Indented code block (4 spaces or 1 tab) -- but NOT if it's a list item
5. Emit pending indented block
6. Blank line (paragraph break)
7. Table rows
8. Blockquote (`>` prefix)
9. Block-level dispatch: heading, hrule, list item
10. Plain text (paragraph continuation)

### Current List Handling

Each list item is parsed as a single line:

- `isUnorderedListItem(line)` detects the marker and returns `(bool, indentLevel, contentStart)`
- `isOrderedListItem(line)` similarly returns `(bool, indentLevel, contentStart, itemNumber)`
- The content after the marker is passed to `parseInline()` for inline formatting
- Each list item is exactly one line -- there is no multi-line list item support

### Key Limitation

The parser has **no list context**. When it encounters a line, it doesn't
know whether it's inside a list item's continuation. This means:

- A fenced code block `` ``` `` after a list item is treated as a
  top-level code block, not as part of the list item
- An indented line after a list item is treated as an indented code block
  (or, if it matches the list pattern, as another list item)
- A blockquote `> ` after a list item is treated as a top-level blockquote

### Block Index (change.go)

The `buildBlockIndex()` function mirrors the parser's block detection but
only records block extents. It currently treats each list item as a
single-line `BlockListItem`. Nested blocks within list items will need
the block index to track multi-line list items that contain sub-blocks.

## Syntax Specification

### Code Blocks in List Items

Fenced code blocks inside list items use standard markdown nesting rules:

```
- List item text
  ```
  code here
  ```
- Next item
```

The fenced code block must be indented to the list item's content level
(2 spaces for a `- ` marker, or the marker width + 1 space for ordered
lists like `1. `).

Indented code blocks inside list items follow the same rule: the code
must be indented by 4 additional spaces beyond the list content indent:

```
- List item text
      code here (6 spaces: 2 for list + 4 for code)
- Next item
```

### Blockquotes in List Items

Blockquotes inside list items are indented to the list content level:

```
- List item text
  > Quoted text
  > More quoted text
- Next item
```

### Continuation Lines

A list item can span multiple lines. Continuation lines must be indented
to the content level of the list item:

```
- First line of item
  continuation of first item
- Second item
```

A blank line within a list item (when the next non-blank line is indented
to the content level) creates a paragraph break within the item:

```
- Paragraph one of item

  Paragraph two of item
- Next item
```

### Nesting Depth

Nested lists already work (indent level tracked via `ListIndent`).
Nested blocks within nested lists follow the same rule -- indent to the
content level of the innermost list item:

```
- Outer item
  - Inner item
    ```
    code in inner item
    ```
  - Another inner item
```

## Design

### Approach: List Context Stack

The parser needs to track whether we're currently inside a list item and
what the content indentation level is. Rather than a full recursive
descent parser rewrite, we add a **list context** to the existing
single-pass loop.

```go
// listContext tracks the state of a list item being accumulated.
type listContext struct {
    indentLevel  int    // nesting level (0 = top, 1 = first nested, etc.)
    contentStart int    // byte offset where content begins in the original line
    contentCol   int    // column where content begins (for continuation detection)
    ordered      bool   // true for ordered lists
    itemNumber   int    // item number for ordered lists
}
```

The `contentCol` is the column position where the list item's content
starts. For `- text`, `contentCol` is 2 (after `- `). For `  - text`,
`contentCol` is 4 (2 spaces indent + `- `). Lines indented to at least
`contentCol` are treated as continuation content within the list item.

### Detection of Continuation Lines

When the parser is inside a list item context, each subsequent line is
checked:

1. **Is it another list item at the same or lower indent?** -> End the
   current list item, start a new one.
2. **Is it a blank line?** -> Could be a paragraph break within the list
   item. Peek ahead: if the next non-blank line is indented to `contentCol`,
   it's an intra-item paragraph break. Otherwise, end the list item.
3. **Is it indented to at least `contentCol`?** -> It's continuation
   content. Strip the leading `contentCol` spaces and process the
   de-indented content through the normal block dispatch.
4. **Otherwise** -> End the list item. Process the line normally.

### Processing Nested Blocks

When continuation content is identified, the stripped line is processed
through a subset of the block dispatch:

- **Fenced code block**: `` ``` `` (after stripping indent) starts a code
  block within the list item. Subsequent lines must be indented to
  `contentCol` and the closing `` ``` `` must also be at `contentCol`.
  The code block content is de-indented before accumulation.
- **Blockquote**: `> ` (after stripping indent) starts a blockquote
  within the list item. The blockquote content gets both
  `Blockquote=true` and `ListItem=true` styling.
- **Indented code**: An additional 4 spaces beyond `contentCol` starts
  an indented code block.
- **Paragraph continuation**: Plain text continues the list item's
  paragraph.

### Style Application

Nested blocks within list items need composite styles:

- Code block in list: `Code=true, Block=true, ListItem=true, ListIndent=N`
- Blockquote in list: `Blockquote=true, BlockquoteDepth=D, ListItem=true, ListIndent=N`

The layout engine already handles additive indentation for blockquotes
and lists. Code blocks within lists need the layout engine to use list
indent instead of gutter indent. This requires a change to the
indentation logic in `rich/layout.go`.

### Layout Changes

Currently in `layout.go` (lines 494-505):

```go
indentPixels := 0
if box.Style.Blockquote {
    indentPixels += box.Style.BlockquoteDepth * ListIndentWidth
}
if box.Style.ListBullet || box.Style.ListItem {
    indentPixels += box.Style.ListIndent * ListIndentWidth
} else if (box.Style.Block && box.Style.Code) || box.Style.Table || box.Style.Image {
    indentPixels = gutterIndent
}
```

The `else if` means code blocks always get gutter indent, even inside
lists. For code blocks inside list items, the indentation should be the
list indent plus a small additional offset (rather than the full 8em
gutter). The fix:

```go
indentPixels := 0
if box.Style.Blockquote {
    indentPixels += box.Style.BlockquoteDepth * ListIndentWidth
}
if box.Style.ListBullet || box.Style.ListItem {
    indentPixels += box.Style.ListIndent * ListIndentWidth
    if (box.Style.Block && box.Style.Code) || box.Style.Table {
        // Code block or table inside a list item: add extra indent
        // instead of using the full gutter
        indentPixels += ListIndentWidth
    }
} else if (box.Style.Block && box.Style.Code) || box.Style.Table || box.Style.Image {
    indentPixels = gutterIndent
}
```

### Source Mapping

Source map entries for nested blocks within list items must account for
both the list item indent and the block prefix:

- For a fenced code block at `  ```\n  code\n  ```\n` within a list,
  the source map entry covers the full indented source, but the rendered
  content is the de-indented code text.
- The `PrefixLen` for the code content lines accounts for the stripped
  indent.
- For blockquotes within list items (`  > text`), the source map needs
  to account for both the list indent stripping and the `> ` prefix
  stripping.

The approach for source mapping:

1. The list item bullet and space get their own entries (existing behavior).
2. Continuation lines that are plain text get entries with
   `SourceOffset` pointing to the start of the content (past the indent).
3. Nested code blocks get a single entry covering all code content lines,
   with `SourceStart` at the first content line (past indent and fence)
   and `SourceEnd` at the last content line (before closing fence).
4. Nested blockquotes get entries using the same PrefixLen model as
   top-level blockquotes, with source offsets adjusted for the list indent.

### Block Index Changes

The `buildBlockIndex()` function needs to recognize multi-line list items.
Currently each list item is `BlockListItem` spanning one line. With nested
blocks, a list item can span many lines. The block index should treat the
entire multi-line list item (including its nested blocks) as a single
`BlockListItem` block. This ensures that editing within a list item
triggers re-parse of the entire list item, not just one line.

### Implementation Strategy

The implementation is split into two sub-phases:

**8.2 Nested Code Blocks in Lists** -- Add list context tracking to
detect fenced and indented code blocks within list items. This is the
harder sub-phase because it requires the continuation line detection
and indent stripping infrastructure.

**8.3 Nested Blockquotes in Lists** -- Reuse the continuation line
infrastructure from 8.2 to handle blockquotes within list items.
Blockquotes already have their own parsing and styling; the main work
is recognizing `> ` within the list item context.

## Implementation Plan

### Phase 8.2: Nested Code Blocks in Lists

#### Parser Changes (`markdown/parse.go`)

1. Add `listContext` struct and tracking state to the main parse loop.
2. After parsing a list item line, set the list context with the item's
   `contentCol`.
3. On each subsequent line, check if it continues the list item:
   - Indented to `contentCol` -> strip indent, process as block content.
   - Another list item -> end current context, start new.
   - Blank line -> peek ahead for continuation.
   - Otherwise -> end context.
4. Within the list context, detect fenced code block delimiters (after
   indent stripping). Track `inListCodeBlock` state to accumulate code
   lines.
5. On closing fence, emit the code block span with list item styling
   (`ListItem=true, ListIndent=N`).

#### Source Map Changes (`markdown/sourcemap.go`)

Mirror the parser changes in `ParseWithSourceMap()`:

1. Track list context with source position accounting.
2. Code block content within list items gets a source map entry covering
   the content bytes (after indent and fence stripping), with
   `SourceStart`/`SourceEnd` pointing to the full indented source lines.
3. The closing fence line is not mapped (same as top-level code blocks).

#### Layout Changes (`rich/layout.go`)

1. When a code block has `ListItem=true`, use list indentation + extra
   indent instead of full gutter indent.

#### Block Index Changes (`markdown/change.go`)

1. In `buildBlockIndex()`, track list context the same way as the parser.
2. Multi-line list items (with nested blocks) produce a single
   `BlockListItem` block spanning all continuation lines.

### Phase 8.3: Nested Blockquotes in Lists

#### Parser Changes

1. Within the list context, detect blockquote lines (after indent
   stripping) using the existing `isBlockquoteLine()`.
2. Process the blockquote content with both blockquote and list item
   styling applied.
3. Handle multi-line blockquotes within list items (consecutive indented
   `> ` lines).

#### Source Map Changes

1. Blockquote entries within list items combine the list indent offset
   with the blockquote `PrefixLen`.
2. The first blockquote content entry's `SourceStart` points to the
   start of the indented line (including list indent and `> ` prefix),
   with `PrefixLen` covering both the list indent and `> ` prefix.

## Testing Strategy

### Parse Tests (`markdown/parse_test.go`)

**Code blocks in lists (8.2):**

1. Fenced code block inside unordered list item
2. Fenced code block inside ordered list item
3. Fenced code block in nested list item (indent level 2)
4. Multiple list items where only one has a code block
5. Code block followed by another list item
6. Indented code block inside list item (4 extra spaces)
7. Code block with list item styling verification (`ListItem=true`)
8. Multi-line list item with paragraph + code block

**Blockquotes in lists (8.3):**

1. Single-line blockquote inside list item
2. Multi-line blockquote inside list item
3. Nested blockquote inside list item
4. Blockquote with inline formatting inside list item
5. Blockquote followed by another list item
6. Verify composite style: `Blockquote=true, ListItem=true`

### Source Map Tests (`markdown/sourcemap_test.go`)

1. Click within code block inside list item maps to correct source
2. Click within blockquote inside list item maps to correct source
3. Selecting code block content in list item maps to source including
   indentation
4. Round-trip consistency for nested blocks in lists

### Layout Tests (`rich/layout_test.go`)

1. Code block inside list item gets list-relative indentation (not
   full gutter)
2. Blockquote inside list item gets additive indentation
   (list + blockquote)

## Risks and Mitigations

- **Complexity of continuation detection**: The main risk is correctly
  detecting when a line continues a list item vs starts a new block.
  The "indent to contentCol" heuristic is the standard CommonMark
  approach and should handle most cases. Edge cases around blank lines
  and mixed indent characters (spaces vs tabs) need careful testing.

- **Performance**: Adding list context tracking to the main loop adds
  a small amount of state but no additional passes. The performance
  impact should be negligible.

- **Blockquote + list interaction in layout**: The layout engine already
  handles additive indentation for blockquotes and lists. The main
  change is handling code blocks inside lists, which requires a small
  tweak to the indentation logic.

- **Block index correctness**: Multi-line list items change the
  granularity of `BlockListItem` entries in the block index. The
  incremental update system may need more blocks in the affected range
  to ensure correct re-parsing. This is mitigated by the
  `AffectedRange()` expansion logic that includes neighboring blocks.

- **Tables inside lists**: Deferred. Table parsing is complex
  (multi-line, separator detection, column width calculation) and
  tables inside lists are rare in practice.

## File Changes Summary

### `markdown/parse.go`
- Add `listContext` struct
- Add list context tracking to `Parse()` main loop
- Add continuation line detection logic
- Handle fenced code blocks within list context
- Handle blockquotes within list context

### `markdown/sourcemap.go`
- Mirror list context tracking in `ParseWithSourceMap()`
- Source map entries for nested code blocks account for list indent
- Source map entries for nested blockquotes combine list + blockquote
  prefix lengths

### `markdown/change.go`
- Update `buildBlockIndex()` to track multi-line list items
- Multi-line list items produce single `BlockListItem` blocks

### `rich/layout.go`
- Adjust code block indentation when `ListItem=true` to use list-relative
  indent instead of full gutter
