# Code Blocks and Horizontal Rules Design

## Overview

This document describes the design for:
1. Rendering markdown code blocks with shaded background boxes
2. Rendering horizontal rules as visual dividers

## Current State (Implemented)

- **Inline code spans** (`` `code` ``): Parsed with `Style.Code = true`
  - Uses monospace font when available via `WithCodeFont()`
  - Renders with subtle gray background (text-width)
- **Fenced code blocks** (` ``` `): Fully supported
  - Detected by opening/closing ``` delimiters
  - Renders with `Style.Block = true` for full-width background
  - Uses monospace font
- **Indented code blocks** (4 spaces or 1 tab): Fully supported
  - Consecutive indented lines merged into a single block
  - Renders identically to fenced code blocks
- **Background rendering**: `Style.Bg` renders via `drawBoxBackground()` (inline)
  and `drawBlockBackground()` (full-width for `Style.Block = true`)

## Goals

1. Fenced code blocks render with a visible shaded background
2. Code text uses monospace font (if available)
3. Background extends to full line width (block-level shading)
4. Inline code also gets subtle background shading

## Design

### Block vs Inline Code

| Type | Syntax | Background | Font |
|------|--------|------------|------|
| Inline | `` `code` `` | Subtle gray, text-width only | Monospace |
| Fenced | ` ``` ... ``` ` | Gray, full line width | Monospace |
| Indented | 4 spaces or 1 tab | Gray, full line width | Monospace |

### Background Rendering Approach

Two options for rendering backgrounds:

**Option A: Box-level backgrounds**
- Each Box carries its Style.Bg
- `drawText()` draws a filled rectangle before each box's text
- Simple but creates gaps between adjacent boxes

**Option B: Line-level backgrounds (Recommended)**
- Track "block regions" that span full lines
- Before drawing text, fill entire line backgrounds
- Cleaner visual appearance for code blocks

We'll use **Option B** for fenced code blocks (full-line shading) and **Option A** for inline code (text-width shading).

### Data Model Changes

#### New: BlockRegion in layout

```go
// BlockRegion marks a vertical region with special background
type BlockRegion struct {
    StartLine int         // First line of region
    EndLine   int         // Last line of region (inclusive)
    Bg        color.Color // Background color for region
}
```

The layout phase will track which lines belong to code blocks.

#### Style.Code usage

The existing `Style.Code` field will be used to:
1. Select monospace font in `fontForStyle()`
2. Trigger inline code background rendering

### Parser Changes

#### Fenced Code Block Detection

In `markdown/parse.go`, detect lines starting with ` ``` `:

```go
// Pseudocode
if strings.HasPrefix(line, "```") {
    if inCodeBlock {
        // End code block
        inCodeBlock = false
    } else {
        // Start code block, optionally capture language
        inCodeBlock = true
        language = strings.TrimPrefix(line, "```")
    }
    continue // Don't emit the fence line
}

if inCodeBlock {
    // Emit as code-styled span (preserve whitespace)
    spans = append(spans, Span{Text: line + "\n", Style: codeBlockStyle})
}
```

#### Code Block Style

```go
var codeBlockStyle = Style{
    Code:  true,
    Block: true,
    Bg:    color.RGBA{R: 240, G: 240, B: 240, A: 255}, // Light gray
}
```

#### Indented Code Block Detection

In addition to fenced code blocks, markdown supports indented code blocks where
lines starting with 4 spaces or 1 tab are treated as code:

```go
// isIndentedCodeLine returns true if the line is an indented code line
// (starts with 4 spaces or 1 tab).
func isIndentedCodeLine(line string) bool {
    if len(line) == 0 {
        return false
    }
    // Check for tab indent
    if line[0] == '\t' {
        return true
    }
    // Check for 4-space indent
    if len(line) >= 4 && line[0:4] == "    " {
        return true
    }
    return false
}

// stripIndent removes the leading indent (4 spaces or 1 tab) from a line.
func stripIndent(line string) string {
    if len(line) == 0 {
        return line
    }
    if line[0] == '\t' {
        return line[1:]
    }
    if len(line) >= 4 && line[0:4] == "    " {
        return line[4:]
    }
    return line
}
```

Consecutive indented lines are merged into a single code block. When a
non-indented line is encountered, the indented block is emitted with the
same `Style.Block = true` and `Style.Code = true` as fenced blocks.

### Rendering Changes

#### 1. Font Selection for Code

Update `fontForStyle()` in `rich/frame.go`:

```go
func (f *frameImpl) fontForStyle(s Style) *draw.Font {
    // Existing scale/bold/italic logic...

    // Add code font handling
    if s.Code && f.codeFont != nil {
        return f.codeFont
    }

    return f.font // fallback
}
```

Add `WithCodeFont(font *draw.Font)` option.

#### 2. Background Rendering

In `drawText()`, render backgrounds before text:

```go
func (f *frameImpl) drawText() {
    // Phase 1: Draw block-level backgrounds (full line width)
    for _, region := range f.blockRegions {
        f.drawBlockBackground(region)
    }

    // Phase 2: Draw inline code backgrounds (box width only)
    for _, line := range f.lines {
        for _, pb := range line {
            if pb.Box.Style.Code && pb.Box.Style.Bg != nil {
                f.drawBoxBackground(pb)
            }
        }
    }

    // Phase 3: Draw text (existing code)
    for _, line := range f.lines {
        for _, pb := range line {
            // ... existing text rendering
        }
    }
}
```

#### 3. Block Background Drawing

```go
func (f *frameImpl) drawBlockBackground(region BlockRegion) {
    if region.Bg == nil {
        return
    }

    bgImg := f.allocColorImage(region.Bg)

    // Calculate vertical bounds
    y0 := f.rect.Min.Y + region.StartLine * f.lineHeight
    y1 := f.rect.Min.Y + (region.EndLine + 1) * f.lineHeight

    // Full width of frame
    r := image.Rect(f.rect.Min.X, y0, f.rect.Max.X, y1)
    f.screen.Draw(r, bgImg, nil, image.Point{})
}
```

### Source Mapping Considerations

The `SourceMap` must account for fence lines being excluded from rendered output:

```markdown
```go        <- source line 1 (not rendered)
func main() <- source line 2, rendered line 1
```           <- source line 3 (not rendered)
```

The parser will track source positions correctly by:
- Recording source offset when entering code block
- Mapping each rendered line back to its source line

### Color Choices

| Element | Color | Rationale |
|---------|-------|-----------|
| Fenced block bg | `#F0F0F0` (light gray) | Visible but not distracting |
| Inline code bg | `#E8E8E8` (slightly darker) | Distinguish from surrounding text |
| Code text | Default (black) | Maintain readability |

These match common markdown renderer conventions (GitHub, VS Code).

### Testing Strategy

1. **Unit tests** in `markdown/parse_test.go`:
   - `TestParseFencedCodeBlock`
   - `TestParseFencedCodeBlockWithLanguage`
   - `TestParseFencedCodeBlockPreservesWhitespace`
   - `TestParseFencedCodeBlockSourceMap`

2. **Rendering tests** in `rich/frame_test.go`:
   - `TestDrawCodeBackground`
   - `TestDrawBlockBackground`
   - `TestCodeFontSelection`

3. **Integration tests**:
   - `TestPreviewCodeBlock`
   - `TestPreviewCodeBlockScrolling`

### Implementation Order

1. **Background rendering infrastructure** - Enable `Style.Bg` rendering in `drawText()`
2. **Code font selection** - Use `Style.Code` in `fontForStyle()`
3. **Inline code backgrounds** - Add background to inline `` `code` ``
4. **Fenced block parsing** - Parse ` ``` ` blocks in markdown
5. **Block-level backgrounds** - Full-width shading for fenced blocks
6. **Visual verification** - Manual testing in Edwood

---

## Horizontal Rules

### Syntax

Standard markdown/Markdeep horizontal rules are created with three or more of:
- Hyphens: `---`, `----`, `- - -`
- Asterisks: `***`, `****`, `* * *`
- Underscores: `___`, `____`, `_ _ _`

The line must contain only these characters (and optional spaces), with nothing else.

### Detection Rules

A line is a horizontal rule if:
1. It contains only hyphens, asterisks, or underscores (plus optional spaces)
2. It has at least 3 of the rule character
3. It uses only one type of character (no mixing `---***`)
4. The line has no other content

```go
// Pseudocode
func isHorizontalRule(line string) bool {
    trimmed := strings.TrimSpace(line)
    if len(trimmed) < 3 {
        return false
    }

    // Remove spaces to check the core pattern
    noSpaces := strings.ReplaceAll(trimmed, " ", "")
    if len(noSpaces) < 3 {
        return false
    }

    // Must be all same character: -, *, or _
    first := noSpaces[0]
    if first != '-' && first != '*' && first != '_' {
        return false
    }

    for _, c := range noSpaces {
        if byte(c) != first {
            return false
        }
    }
    return true
}
```

### Rendering Approach

Horizontal rules are rendered as a styled visual element:

**Option A: Line with spacing**
- Render a 1px horizontal line
- Add vertical padding above and below (e.g., half line height each)
- Line color: medium gray (`#CCCCCC`)

**Option B: Full-height divider region (Recommended)**
- Allocate a full line height for the rule
- Draw centered horizontal line within that space
- Provides consistent spacing without special padding logic

### Data Model

#### New: HorizontalRule marker

The horizontal rule needs to be represented in the Content/Span model. Options:

**Option 1: Special Span with marker**
```go
// In rich/span.go or style.go
type Style struct {
    // ... existing fields
    HRule bool // This span represents a horizontal rule
}
```

The span text would be empty or a placeholder, and rendering checks `HRule` flag.

**Option 2: Special character/rune (Recommended)**

Use a designated rune (e.g., Unicode `\u2500` BOX DRAWINGS LIGHT HORIZONTAL or a private-use character) that the renderer recognizes:

```go
const HRuleRune = '\u2500' // ─

// Parser emits:
spans = append(spans, Span{
    Text:  string(HRuleRune) + "\n",
    Style: hruleStyle,
})
```

The renderer detects this rune and draws a line instead of text.

### Rendering Implementation

In `drawText()`:

```go
// When drawing a box
if pb.Box.Text == string(HRuleRune) {
    f.drawHorizontalRule(pb)
    continue
}
```

```go
func (f *frameImpl) drawHorizontalRule(pb positionedBox) {
    // Line positioned vertically centered in the line height
    y := pb.Pt.Y + f.lineHeight/2

    // Full width of frame content area
    x0 := f.rect.Min.X
    x1 := f.rect.Max.X

    // Draw 1px line
    lineColor := color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
    lineImg := f.allocColorImage(lineColor)

    r := image.Rect(x0, y, x1, y+1)
    f.screen.Draw(r, lineImg, nil, image.Point{})
}
```

### Source Mapping

The horizontal rule line maps 1:1 with the source:
- Source: `---\n` at position N
- Rendered: HRule marker at position N

The rendered position has length 1 (the HRuleRune) but maps to the full source line.

### Color and Styling

| Element | Value | Notes |
|---------|-------|-------|
| Line color | `#CCCCCC` | Medium gray, visible but subtle |
| Line thickness | 1px | Clean, minimal appearance |
| Vertical space | 1 line height | Consistent with text line spacing |

### Testing Strategy

1. **Parser tests** in `markdown/parse_test.go`:
   - `TestParseHorizontalRuleHyphens` - `---`
   - `TestParseHorizontalRuleAsterisks` - `***`
   - `TestParseHorizontalRuleUnderscores` - `___`
   - `TestParseHorizontalRuleWithSpaces` - `- - -`
   - `TestParseHorizontalRuleLong` - `----------`
   - `TestParseNotHorizontalRule` - `--` (too short), `--text` (has content)

2. **Rendering tests** in `rich/frame_test.go`:
   - `TestDrawHorizontalRule`
   - `TestHorizontalRuleFullWidth`

3. **Integration tests**:
   - `TestPreviewHorizontalRule`

### Implementation Order

1. **Add HRuleRune constant** - Define the marker rune
2. **Parse horizontal rules** - Detect `---`/`***`/`___` patterns
3. **Render horizontal rules** - Draw line when HRuleRune encountered
4. **Source mapping** - Ensure correct position mapping
5. **Visual verification** - Manual testing

## Open Questions

1. **Language hint**: Should we display the language (e.g., "go") anywhere? Initially: no, just parse and ignore.

2. **Horizontal scrolling**: If code lines are very long, should they scroll horizontally or wrap? Initially: wrap like normal text.

3. **Syntax highlighting**: Out of scope for this phase. Could be a future enhancement using tree-sitter or similar.

## Double-Click to Select Code Block

### Problem

In preview mode, there is no way to quickly select all the text in a
fenced code block. Users need to be able to double-click inside a code
block to select the entire block text, enabling chord-copy and other
operations.

### Acme Pattern

Acme's `Text.Select` (text.go:1136-1192) tracks double-click with two
module-level variables: `clicktext` (which text buffer received the
last click) and `clickmsec` (timestamp). A second B1 click within
500ms at the same position triggers `DoubleClick()`, which expands the
selection (bracket matching or word expansion).

### Design

**Double-click state** on `Window`:

```go
previewClickPos  int        // rune position of last B1 null-click
previewClickMsec uint32     // timestamp of last B1 null-click
previewClickRT   *RichText  // which richtext got the last click
```

**Expansion method** on `Frame` (`rich/frame.go`):

```go
// ExpandAtPos returns the expanded selection range for a rune offset.
// If the position is inside a code block (Block && Code), returns the
// full code block range. Otherwise returns word boundaries.
func (f *frameImpl) ExpandAtPos(pos int) (q0, q1 int)
```

Walks `f.content` spans accumulating rune offsets. Finds the span
containing `pos`. If `Style.Block && Style.Code`, scans backward and
forward for all contiguous spans with the same block-code style and
returns their combined rune range. Otherwise scans for word characters
(alphanumeric + underscore).

**B1 handler** in `HandlePreviewMouse` (`wind.go`):

Before `SelectWithChord`: check double-click conditions (same RT,
within 500ms, same position). If met, call `ExpandAtPos` to expand
selection, then wait for button release with jitter tolerance before
proceeding to chord detection. Otherwise proceed with normal
`SelectWithChord`.

After `SelectWithChord` returns: if null click (p0 == p1), record
click state. If dragged, clear click state.

### Files

| File | Changes |
|------|---------|
| `rich/frame.go` | Add `ExpandAtPos` method; add to `Frame` interface |
| `wind.go` | Add double-click state; modify B1 handler |

## Future Enhancements

- Syntax highlighting per language
- Copy button for code blocks
- Line numbers in code blocks
- Horizontal scroll for long lines
