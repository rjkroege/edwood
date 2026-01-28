# Tables, Lists, and Images Design

## Overview

This document describes the design for:
1. Rendering markdown lists (bulleted and numbered)
2. Rendering markdown tables
3. Rendering markdown images

## Design Philosophy

Following the established patterns from Phase 13 (code blocks), we use:
- **Style flags** on spans to indicate list/table context
- **Special runes** for visual markers (like HRuleRune for horizontal rules)
- **SourceMap integration** for copy operations to map back to raw markdown
- **Progressive implementation**: lists first (most common), then tables, then images

---

## Part 1: Lists

### Markdown List Syntax

**Unordered lists** use `-`, `*`, or `+` followed by a space:
```markdown
- Item one
- Item two
  - Nested item
```

**Ordered lists** use numbers followed by `.` or `)`:
```markdown
1. First item
2. Second item
   1. Nested item
```

### Detection Rules

A line is a list item if:
1. It starts with optional whitespace (for nesting)
2. Followed by a list marker:
   - Unordered: `-`, `*`, or `+` followed by space
   - Ordered: one or more digits followed by `.` or `)` and space
3. Followed by item content

```go
// isUnorderedListItem returns true if line starts with unordered list marker.
// Returns: (isListItem bool, indentLevel int, contentStart int)
func isUnorderedListItem(line string) (bool, int, int) {
    // Count leading whitespace (2 spaces or 1 tab = 1 indent level)
    indent := 0
    i := 0
    for i < len(line) {
        if line[i] == ' ' {
            i++
            if i < len(line) && line[i] == ' ' {
                i++
                indent++
            }
        } else if line[i] == '\t' {
            i++
            indent++
        } else {
            break
        }
    }

    // Check for list marker
    if i < len(line) && (line[i] == '-' || line[i] == '*' || line[i] == '+') {
        if i+1 < len(line) && line[i+1] == ' ' {
            return true, indent, i + 2
        }
    }
    return false, 0, 0
}

// isOrderedListItem returns true if line starts with ordered list marker.
func isOrderedListItem(line string) (bool, int, int, int) {
    // Similar logic, but parse number followed by . or )
    // Returns: (isListItem, indentLevel, contentStart, itemNumber)
}
```

### Rendering Approach

Lists are rendered with:
- **Bullet character** for unordered: `•` (U+2022 BULLET)
- **Number + period** for ordered: `1.`, `2.`, etc.
- **Indentation** based on nesting level

#### Data Model

Add to `rich/style.go`:

```go
type Style struct {
    // ... existing fields ...

    // List formatting
    ListItem    bool  // This span is a list item
    ListBullet  bool  // This span is a list bullet/number marker
    ListIndent  int   // Indentation level (0 = top level)
    ListOrdered bool  // true for ordered lists, false for unordered
    ListNumber  int   // For ordered lists, the item number
}
```

#### Parser Output

For input:
```markdown
- First item
- Second item
```

Parser emits:
```go
[]rich.Span{
    {Text: "•", Style: Style{ListBullet: true, ListIndent: 0}},
    {Text: " ", Style: Style{ListItem: true, ListIndent: 0}},
    {Text: "First item", Style: Style{ListItem: true, ListIndent: 0}},
    {Text: "\n", Style: DefaultStyle()},
    {Text: "•", Style: Style{ListBullet: true, ListIndent: 0}},
    {Text: " ", Style: Style{ListItem: true, ListIndent: 0}},
    {Text: "Second item", Style: Style{ListItem: true, ListIndent: 0}},
    {Text: "\n", Style: DefaultStyle()},
}
```

#### Layout Changes

In `rich/frame.go` layout:
- When starting a line with `ListBullet`, add indentation based on `ListIndent`
- Indentation = `ListIndent * indentWidth` where `indentWidth` ≈ 20px

```go
func (f *frameImpl) indentForListItem(style Style) int {
    if !style.ListBullet && !style.ListItem {
        return 0
    }
    return style.ListIndent * f.listIndentWidth
}
```

### Source Mapping

List items map from rendered bullet + text back to source marker + text:

| Source | Rendered | Notes |
|--------|----------|-------|
| `- Item` | `• Item` | 2 source chars (`- `) → 2 rendered chars (`• `) |
| `1. Item` | `1. Item` | Direct mapping |
| `  - Nested` | `  • Nested` | Preserve indent in mapping |

### Colors and Styling

| Element | Style | Notes |
|---------|-------|-------|
| Bullet `•` | Default text color | Consistent with text |
| Numbers | Default text color | Match body text |
| Item text | Default | Supports inline formatting |

### Testing Strategy

Parser tests (`markdown/parse_test.go`):
- `TestParseUnorderedList`
- `TestParseOrderedList`
- `TestParseNestedList`
- `TestParseListWithInlineFormatting`
- `TestParseMixedListTypes`

Rendering tests (`rich/frame_test.go`):
- `TestDrawListBullet`
- `TestDrawOrderedListNumber`
- `TestDrawNestedListIndent`

---

## Part 2: Tables

### Markdown Table Syntax

```markdown
| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |
```

Optional alignment row with `:`:
```markdown
| Left | Center | Right |
|:-----|:------:|------:|
| L    |   C    |     R |
```

### Detection Rules

A table consists of:
1. **Header row**: Line with `|` delimiters
2. **Separator row**: Line with `|` and `-` (and optional `:` for alignment)
3. **Data rows**: Lines with `|` delimiters

```go
// isTableSeparatorRow detects the alignment/separator row.
func isTableSeparatorRow(line string) bool {
    trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
    if !strings.HasPrefix(trimmed, "|") {
        return false
    }
    // Check for pattern: |---| or |:--| or |--:| or |:-:|
    // with at least one cell
    cells := splitTableCells(trimmed)
    if len(cells) == 0 {
        return false
    }
    for _, cell := range cells {
        cell = strings.TrimSpace(cell)
        if !isTableSeparatorCell(cell) {
            return false
        }
    }
    return true
}

func isTableSeparatorCell(cell string) bool {
    // Must be dashes with optional : on either end
    if len(cell) < 3 {
        return false
    }
    // Remove optional colons
    cell = strings.TrimPrefix(cell, ":")
    cell = strings.TrimSuffix(cell, ":")
    // Rest must be all dashes
    for _, c := range cell {
        if c != '-' {
            return false
        }
    }
    return true
}
```

### Rendering Approach

Tables are rendered as **monospace grid** with fixed column widths:

```
┌──────────┬──────────┬──────────┐
│ Header 1 │ Header 2 │ Header 3 │
├──────────┼──────────┼──────────┤
│ Cell 1   │ Cell 2   │ Cell 3   │
│ Cell 4   │ Cell 5   │ Cell 6   │
└──────────┴──────────┴──────────┘
```

**Alternative (simpler)**: ASCII table with pipes and dashes, using code font:
```
| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
```

We'll start with the simpler approach: render tables in code font with their original ASCII structure, padded for alignment.

#### Data Model

Add to `rich/style.go`:

```go
type Style struct {
    // ... existing fields ...

    // Table formatting
    Table       bool      // This span is part of a table
    TableHeader bool      // This is a header cell
    TableAlign  Alignment // Cell alignment
}

type Alignment int

const (
    AlignLeft Alignment = iota
    AlignCenter
    AlignRight
)
```

#### Parser Strategy

The parser collects table rows and emits them as a unit:

```go
type TableCell struct {
    Content string
    Align   Alignment
}

type TableRow struct {
    Cells    []TableCell
    IsHeader bool
}

// parseTable parses consecutive table lines and returns spans.
func parseTable(lines []string) []rich.Span {
    // 1. Parse header row
    // 2. Parse separator row (get alignments)
    // 3. Parse data rows
    // 4. Calculate column widths
    // 5. Emit padded, aligned cells with table style
}
```

#### Column Width Calculation

```go
func calculateColumnWidths(rows []TableRow) []int {
    if len(rows) == 0 {
        return nil
    }

    numCols := len(rows[0].Cells)
    widths := make([]int, numCols)

    for _, row := range rows {
        for i, cell := range row.Cells {
            if i < numCols {
                w := runeWidth(cell.Content)
                if w > widths[i] {
                    widths[i] = w
                }
            }
        }
    }
    return widths
}
```

#### Rendered Output

Tables render as code-styled text blocks:

```go
spans := []rich.Span{
    {Text: "| Header 1   | Header 2   |\n", Style: tableHeaderStyle},
    {Text: "|------------|------------|\n", Style: tableSepStyle},
    {Text: "| Cell 1     | Cell 2     |\n", Style: tableStyle},
}
```

Where:
- `tableHeaderStyle = Style{Code: true, Bold: true, Block: true}`
- `tableStyle = Style{Code: true, Block: true}`

This gives tables the same visual treatment as code blocks (gray background, monospace font) while preserving structure.

### Source Mapping

Table rendering preserves cell content, so source mapping is straightforward:
- Padding spaces are not in source
- Cell content maps 1:1

### Testing Strategy

Parser tests:
- `TestParseSimpleTable`
- `TestParseTableWithAlignment`
- `TestParseTableMissingCells`
- `TestParseTableInDocument`

Rendering tests:
- `TestDrawTable`
- `TestDrawTableHeader`
- `TestDrawTableAlignment`

---

### Table Remediation: Grid Layout and Visual Polish

The initial table implementation (Phase 15B) established parsing and basic rendering
but left several gaps. This section describes the improvements needed to bring tables
up to proper visual quality.

#### Current State Assessment

What works:
- Detection: `isTableRow()`, `splitTableCells()`, `isTableSeparatorRow()`,
  `parseTableSeparator()`, `parseSeparatorCell()` all function correctly.
- Alignment parsing: `parseTableSeparator()` correctly extracts per-column
  `AlignLeft`/`AlignCenter`/`AlignRight` values.
- Style fields: `Table`, `TableHeader`, `TableAlign` exist on `Style`.
- Layout integration: tables recognized as `BlockTable` for scrollbar/block
  region handling.
- Test coverage: 11+ test functions covering detection, parsing, alignment,
  source mapping, and negative cases.

What is broken or missing:

1. **No column width normalization.** `calculateColumnWidths()` exists in
   `parse.go` but is never called. `parseTableBlock()` emits raw line text
   verbatim, so misaligned source text stays misaligned in the rendered output.

2. **No grid lines.** The design above envisions box-drawing characters
   (`┌──┬──┐`, `│`, `├──┼──┤`, `└──┴──┘`) but the current code just passes
   through the original `|------|` ASCII. There are no proper horizontal rules
   (top border, header separator, bottom border) or vertical rules.

3. **Alignment parsed but not applied.** `parseTableSeparator()` extracts
   per-column `Alignment` values, but `parseTableBlock()` never uses them. It
   does not pad/align cell content. `TableAlign` is effectively always
   `AlignLeft`.

4. **Each row is one monolithic span.** Instead of emitting per-cell spans
   (which would allow per-cell styling, alignment, and layout), each row is
   emitted as a single `rich.Span` with the entire line as `Text`. This makes
   it impossible for the layout engine to do cell-level formatting.

#### Improvement Phase 1: Column Width Normalization + Alignment

**Goal:** Tables render with uniform column widths and correct cell alignment.

**Changes to `parseTableBlock()` in `markdown/parse.go`:**

1. After collecting all table lines, parse every row into `[][]string` (cells).
2. Call `calculateColumnWidths()` to get `[]int` max widths.
3. Parse the separator row to get `[]rich.Alignment` per column.
4. Rebuild each row's text by padding cells to their column width:
   - `AlignLeft`: content + trailing spaces
   - `AlignRight`: leading spaces + content
   - `AlignCenter`: split padding on both sides
5. Rebuild the separator row with dashes padded to column widths.
6. Emit the rebuilt text as spans (still one span per row for now).

```
Before (raw passthrough):
| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 7 |

After (normalized):
| Name  | Age |
|-------|-----|
| Alice |  30 |
| Bob   |   7 |
```

**Source mapping impact:** The source map function
`parseTableBlockWithSourceMap()` must apply the same normalization. Padding
spaces are synthetic (not in source), so source map entries skip them.

**Tests:**
- `TestParseTableNormalizedWidths` — verify cells padded to uniform widths
- `TestParseTableRightAlign` — right-aligned column has leading spaces
- `TestParseTableCenterAlign` — center-aligned column has balanced padding
- `TestParseTableSourceMapWithPadding` — source map skips synthetic padding

#### Improvement Phase 2: Box-Drawing Grid Lines

**Goal:** Replace ASCII pipes and dashes with Unicode box-drawing characters
for clean visual presentation.

**Character mapping:**

| Position | ASCII | Box-drawing |
|----------|-------|-------------|
| Top-left corner | (none) | `┌` U+250C |
| Top-right corner | (none) | `┐` U+2510 |
| Bottom-left corner | (none) | `└` U+2514 |
| Bottom-right corner | (none) | `┘` U+2518 |
| Horizontal rule | `-` | `─` U+2500 |
| Vertical rule | `\|` | `│` U+2502 |
| Top tee | (none) | `┬` U+252C |
| Bottom tee | (none) | `┴` U+2534 |
| Left tee | (none) | `├` U+251C |
| Right tee | (none) | `┤` U+2524 |
| Cross | (none) | `┼` U+253C |

**Rendered output:**

```
┌───────┬─────┬──────┐
│ Name  │ Age │ City │
├───────┼─────┼──────┤
│ Alice │  30 │ NYC  │
│ Bob   │   7 │ LA   │
└───────┴─────┴──────┘
```

**Changes to `parseTableBlock()`:**

1. After normalization (Phase 1), generate three additional synthetic lines:
   - **Top border:** `┌` + `─`×(width+2) joined by `┬` + `┐`
   - **Header separator:** `├` + `─`×(width+2) joined by `┼` + `┤`
     (replaces the raw `|---|` separator)
   - **Bottom border:** `└` + `─`×(width+2) joined by `┴` + `┘`
2. Replace `|` cell delimiters with `│` in header and data rows.
3. Border/separator lines get `Style{Table: true, Code: true, Block: true}`
   (no Bold, no TableHeader).

**Helper function:**

```go
// buildGridLine builds a box-drawing horizontal line.
// left/mid/right are corner/tee characters; fill is the horizontal rule char.
func buildGridLine(widths []int, left, mid, right, fill rune) string {
    var b strings.Builder
    b.WriteRune(left)
    for i, w := range widths {
        for j := 0; j < w+2; j++ { // +2 for padding spaces
            b.WriteRune(fill)
        }
        if i < len(widths)-1 {
            b.WriteRune(mid)
        }
    }
    b.WriteRune(right)
    return b.String()
}
```

**Source mapping impact:** Box-drawing characters are synthetic and do not
appear in source. The source map must map cell content only, skipping border
characters and padding. Grid border lines (top, separator, bottom) have no
source mapping — they map to zero-length source ranges.

**Tests:**
- `TestParseTableBoxDrawing` — verify box-drawing characters in output
- `TestParseTableTopBorder` — top border line present with correct corners
- `TestParseTableBottomBorder` — bottom border line present
- `TestParseTableHeaderSeparator` — header separator uses `├──┼──┤`
- `TestParseTableVerticalRules` — `│` replaces `|` in cell rows
- `TestParseTableSourceMapBoxDrawing` — source map ignores synthetic chars

#### Improvement Phase 3: Per-Cell Spans (Future)

**Goal:** Emit individual spans per cell for true grid layout support.

This is a deeper structural change that would touch the layout engine. It is
documented here for future reference but is **not planned for immediate
implementation**.

Each cell would become its own `rich.Span` with:
- `TableAlign` set to the column's alignment
- `TableHeader` set for header row cells
- Column index and row index metadata for layout

The layout engine would need to:
- Recognize adjacent table cell spans and arrange them in a grid
- Handle cell backgrounds, selection, and overflow per-cell
- Support horizontal scrolling at the table level (existing `BlockTable`
  region handling)

This phase is deferred because Phases 1 and 2 achieve good visual quality
using the existing one-span-per-row model, and the layout engine changes
would be invasive.

---

## Part 3: Images

### Markdown Image Syntax

```markdown
![Alt text](image-url.png)
![Alt text](image-url.png "Optional title")
```

### Rendering Approach

Images present a challenge for text-based rendering. Options:

**Option A: Placeholder text (Recommended for v1)**
- Render as `[Image: Alt text]` in a distinctive style
- Simple, consistent with text-only rendering

**Option B: Load and display image**
- Use draw.Image to load image file
- Requires async loading, error handling, sizing
- Complex but rich experience

**Option C: Link-style indicator**
- Render alt text as blue link
- Clicking opens image in external viewer

We'll implement **Option A** initially for simplicity, with the structure to support Option B later.

#### Data Model

```go
type Style struct {
    // ... existing fields ...

    // Image placeholder
    Image    bool   // This span is an image placeholder
    ImageURL string // URL/path of the image
    ImageAlt string // Alt text
}
```

#### Parser Output

For `![Logo](logo.png)`:

```go
[]rich.Span{
    {Text: "[Image: Logo]", Style: Style{
        Image:    true,
        ImageURL: "logo.png",
        ImageAlt: "Logo",
        Fg:       ImagePlaceholderColor,  // Light gray
        Bg:       ImagePlaceholderBg,     // Subtle background
    }},
}
```

#### Detection

```go
// isImageSyntax detects ![alt](url) pattern.
func parseImage(text string, pos int) (altText, url string, endPos int, ok bool) {
    if pos+1 >= len(text) || text[pos] != '!' || text[pos+1] != '[' {
        return "", "", 0, false
    }

    // Find closing ]
    altEnd := strings.Index(text[pos+2:], "]")
    if altEnd == -1 {
        return "", "", 0, false
    }
    altEnd += pos + 2

    // Must be followed by (
    if altEnd+1 >= len(text) || text[altEnd+1] != '(' {
        return "", "", 0, false
    }

    // Find closing )
    urlEnd := strings.Index(text[altEnd+2:], ")")
    if urlEnd == -1 {
        return "", "", 0, false
    }
    urlEnd += altEnd + 2

    alt := text[pos+2 : altEnd]
    url := text[altEnd+2 : urlEnd]

    // Strip optional title: "title"
    url = strings.TrimSpace(url)
    if idx := strings.Index(url, " "); idx != -1 {
        url = url[:idx]
    }

    return alt, url, urlEnd + 1, true
}
```

#### Visual Style

| Element | Style | Notes |
|---------|-------|-------|
| Placeholder text | Light gray fg, subtle bg | Clearly distinguishable |
| Format | `[Image: <alt>]` | Shows it's an image |

#### Future: Actual Image Rendering

For Option B support later:

```go
// In rich/frame.go
type frameImpl struct {
    // ... existing fields ...
    images map[string]*draw.Image  // Cached loaded images
}

func (f *frameImpl) drawImage(pb positionedBox) {
    if !pb.Box.Style.Image {
        return
    }

    img, ok := f.images[pb.Box.Style.ImageURL]
    if !ok {
        // Load async, show placeholder for now
        go f.loadImage(pb.Box.Style.ImageURL)
        f.drawImagePlaceholder(pb)
        return
    }

    // Scale image to fit within max width (frame width)
    // Draw image at pb.Pt
    // ...
}
```

### Source Mapping

`![Alt](url)` (14+ chars) → `[Image: Alt]` (12+ chars)

The source map needs to handle this contraction.

### Testing Strategy

Parser tests:
- `TestParseImage`
- `TestParseImageWithTitle`
- `TestParseImageInline`
- `TestParseImageNotLink`

Rendering tests:
- `TestDrawImagePlaceholder`

---

## Implementation Order

### Phase 15A: Lists

1. **15A.1** Add list style fields to `rich/style.go`
2. **15A.2** Implement `isUnorderedListItem()` detection
3. **15A.3** Implement `isOrderedListItem()` detection
4. **15A.4** Parser emits list bullets and items
5. **15A.5** Layout applies list indentation
6. **15A.6** Nested list support
7. **15A.7** List source mapping
8. **15A.8** Visual verification

### Phase 15B: Tables

1. **15B.1** Add table style fields to `rich/style.go`
2. **15B.2** Implement table row detection
3. **15B.3** Implement table separator detection
4. **15B.4** Parser collects and formats table
5. **15B.5** Calculate column widths
6. **15B.6** Emit aligned table spans
7. **15B.7** Table source mapping
8. **15B.8** Visual verification

### Phase 15C: Images

1. **15C.1** Add image style fields to `rich/style.go`
2. **15C.2** Implement image syntax detection
3. **15C.3** Parser emits image placeholder
4. **15C.4** Render placeholder text
5. **15C.5** Image source mapping
6. **15C.6** Visual verification

---

## Open Questions

1. **List continuation**: Should we support multi-line list items? Initially: no, single line per item.

2. **Table cell overflow**: If a cell is very wide, should it wrap? Initially: truncate at reasonable width.

3. **Image sizing**: If we later support actual images, what's the max size? Initially: placeholder only.

4. **Nested structures**: Lists inside tables, etc.? Initially: no nesting across element types.

---

## Future Enhancements

- Multi-line list items with proper indentation
- Task lists (`- [ ]` and `- [x]`)
- Definition lists (`term : definition`)
- Table cell spanning
- Actual image rendering
- Image caching and lazy loading
