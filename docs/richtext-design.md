# Rich Text Design Document

## Overview

Extend Edwood with rich text rendering capabilities, enabling styled text display for use cases like markdown preview. The initial focus is on **rendering, selection, and scrolling** - not editing.

### Goals

1. Render styled text (colors, bold/italic, font sizes)
2. Support selection and copy operations
3. Support scrolling through large documents
4. Provide live preview of markdown files in Edwood
5. **Do not break existing Edwood functionality**

### Non-Goals (for v1)

- Rich text editing
- Full markdown support (start with minimal subset)
- WYSIWYG editing
- Complex layouts (tables, images, nested lists)

---

## Architecture

### Design Principles

1. **Parallel implementation**: New packages alongside existing `frame/`, not modifications to it
2. **Incremental delivery**: Each phase produces working, testable code
3. **Reuse where possible**: Leverage existing `draw` interfaces and coordinate systems
4. **Clean separation**: Markdown parsing separate from rendering

### Package Structure

```
edwood/
├── frame/              # Existing - unchanged
├── rich/               # New package
│   ├── style.go        # Style definitions
│   ├── span.go         # Attributed text spans (input model)
│   ├── box.go          # Styled boxes (layout model)
│   ├── frame.go        # RichFrame - layout and rendering
│   ├── frame_test.go
│   ├── select.go       # Selection handling
│   ├── scroll.go       # Scrolling support
│   └── options.go      # Functional options pattern
├── markdown/           # New package
│   ├── parse.go        # Markdown → Spans
│   └── parse_test.go
├── richtext.go         # RichText component (like Text but for rich content)
└── preview.go          # Preview window integration
```

---

## Core Types

### Style

```go
// rich/style.go

package rich

import "9fans.net/go/draw"

// Style defines visual attributes for a span of text.
type Style struct {
    // Colors (nil means use default)
    Fg *draw.Color
    Bg *draw.Color

    // Font variations
    Bold   bool
    Italic bool

    // Size multiplier (1.0 = normal body text)
    // Used for headings: H1=2.0, H2=1.5, H3=1.25, etc.
    Scale float64
}

// DefaultStyle returns the default body text style.
func DefaultStyle() Style {
    return Style{Scale: 1.0}
}

// Common styles
var (
    StyleH1     = Style{Bold: true, Scale: 2.0}
    StyleH2     = Style{Bold: true, Scale: 1.5}
    StyleH3     = Style{Bold: true, Scale: 1.25}
    StyleBold   = Style{Bold: true, Scale: 1.0}
    StyleItalic = Style{Italic: true, Scale: 1.0}
    StyleCode   = Style{Scale: 1.0} // Will use monospace font
)
```

### Span (Input Model)

```go
// rich/span.go

package rich

// Span represents a run of text with uniform style.
// This is the input model - what markdown parsing produces.
type Span struct {
    Text  string
    Style Style
}

// Content is a sequence of styled spans representing a document.
type Content []Span

// Plain creates Content from unstyled text.
func Plain(text string) Content {
    return Content{{Text: text, Style: DefaultStyle()}}
}

// Len returns total rune count.
func (c Content) Len() int {
    n := 0
    for _, s := range c {
        n += len([]rune(s.Text))
    }
    return n
}
```

### Box (Layout Model)

```go
// rich/box.go

package rich

// Box represents a positioned, styled fragment of text.
// This is the layout model - produced by laying out Spans.
type Box struct {
    // Content
    Text  []byte // UTF-8 content (empty for newline/tab)
    Nrune int    // Rune count (-1 for special boxes)
    Bc    rune   // Box character: 0 for text, '\n' for newline, '\t' for tab

    // Style
    Style Style

    // Layout (computed)
    Wid int // Width in pixels
}

// IsNewline returns true if this is a newline box.
func (b *Box) IsNewline() bool {
    return b.Nrune < 0 && b.Bc == '\n'
}

// IsTab returns true if this is a tab box.
func (b *Box) IsTab() bool {
    return b.Nrune < 0 && b.Bc == '\t'
}
```

### RichFrame Interface

```go
// rich/frame.go

package rich

import (
    "image"
    "9fans.net/go/draw"
)

// Frame renders styled text content with selection support.
type Frame interface {
    // Initialization
    Init(r image.Rectangle, opts ...Option)
    Clear()

    // Content
    SetContent(c Content)

    // Geometry
    Rect() image.Rectangle
    Ptofchar(p int) image.Point    // Character position → screen point
    Charofpt(pt image.Point) int   // Screen point → character position

    // Selection
    Select(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int)
    SetSelection(p0, p1 int)
    GetSelection() (p0, p1 int)

    // Scrolling
    SetOrigin(org int)
    GetOrigin() int
    MaxLines() int
    VisibleLines() int

    // Rendering
    Redraw()

    // Status
    Full() bool // True if frame is at capacity
}
```

---

## Integration with Edwood

### RichText Component

Similar to `Text` but wraps `rich.Frame` instead of `frame.Frame`:

```go
// richtext.go

type RichText struct {
    display draw.Display
    fr      rich.Frame
    content rich.Content

    // Scrolling
    org int // Origin in content (rune offset)

    // Selection
    q0, q1 int // Selection in content coordinates

    // Geometry
    all     image.Rectangle // Full area including scrollbar
    scrollr image.Rectangle // Scrollbar area

    // Parent
    w *Window
}

func (t *RichText) Init(r image.Rectangle, display draw.Display) {
    // Initialize frame with rectangle minus scrollbar
}

func (t *RichText) SetContent(c rich.Content) {
    t.content = c
    t.fr.SetContent(c)
}

func (t *RichText) Redraw() {
    t.fr.Redraw()
    t.scrDraw() // Draw scrollbar
}

func (t *RichText) Scroll(n int) {
    // Adjust origin, redraw
}
```

### Preview Window

A window variant that shows rich text in its body:

```go
// preview.go

type PreviewWindow struct {
    *Window           // Embed standard window
    rich    *RichText // Rich text body (replaces normal body for display)
    source  *Window   // Source window being previewed
}

// Could be triggered by:
// - Command: "Markdeep" in window tag
// - Automatic for .md files (configurable)
```

### Alternative: Rich Body Mode

Instead of a separate window type, windows could have a "rich mode" toggle:

```go
type Window struct {
    // ... existing fields ...

    richBody *RichText // Non-nil when in rich/preview mode
    richMode bool
}

func (w *Window) SetRichMode(on bool) {
    if on {
        // Parse body content as markdown
        // Display via richBody
    } else {
        // Return to normal editing
    }
}
```

**Decision needed**: Separate preview window vs. toggle mode?

---

## Markdown Parsing

### Phase 1: Minimal Subset

Start with only headings:

```go
// markdown/parse.go

package markdown

import (
    "strings"
    "edwood/rich"
)

// Parse converts markdown text to styled content.
// Initially supports only headings.
func Parse(text string) rich.Content {
    var content rich.Content
    lines := strings.Split(text, "\n")

    for i, line := range lines {
        span := parseLine(line)
        content = append(content, span)

        // Add newline between lines (except last)
        if i < len(lines)-1 {
            content = append(content, rich.Span{
                Text:  "\n",
                Style: rich.DefaultStyle(),
            })
        }
    }

    return content
}

func parseLine(line string) rich.Span {
    // Count leading #
    level := 0
    for _, c := range line {
        if c == '#' {
            level++
        } else {
            break
        }
    }

    // Must have space after # to be a heading
    if level > 0 && level <= 6 && len(line) > level && line[level] == ' ' {
        text := strings.TrimPrefix(line, strings.Repeat("#", level)+" ")
        return rich.Span{
            Text:  text,
            Style: headingStyle(level),
        }
    }

    return rich.Span{Text: line, Style: rich.DefaultStyle()}
}

func headingStyle(level int) rich.Style {
    scales := []float64{2.0, 1.5, 1.25, 1.1, 1.05, 1.0}
    return rich.Style{
        Bold:  true,
        Scale: scales[level-1],
    }
}
```

### Future Phases

Add incrementally:
- `**bold**` and `*italic*`
- `` `code` `` spans
- `> blockquotes`
- `- lists`
- Code blocks (```)
- Links (display only, no clicking initially)

---

## Implementation Phases

### Phase 1: Scaffold and Visual Proof

**Goal**: See something different on screen.

**Deliverables**:
- `rich/` package skeleton
- `rich.Frame` that renders plain text with **distinct background color**
- Integration point in Edwood (even if just a test window)
- Basic tests

**Acceptance**: Can create a rich frame, set plain text content, see it rendered with a different background color than normal frames.

### Phase 2: Styled Text Rendering

**Goal**: Render text with multiple styles.

**Deliverables**:
- Style application in rendering
- Font variants for bold/italic (or color simulation if fonts unavailable)
- Box layout with mixed styles
- Tests for multi-style content

**Acceptance**: Can render content like "Normal **bold** normal" with visible style differences.

### Phase 3: Selection

**Goal**: Select text with mouse.

**Deliverables**:
- `Charofpt` / `Ptofchar` coordinate mapping
- Mouse-driven selection
- Selection highlighting (different background for selected region)
- Copy to snarf buffer

**Acceptance**: Can click and drag to select text, selection is visually highlighted.

### Phase 4: Scrolling

**Goal**: Navigate large documents.

**Deliverables**:
- Origin-based scrolling
- Scrollbar rendering
- Mouse wheel support
- Keyboard scrolling (PgUp/PgDn)

**Acceptance**: Can scroll through document larger than viewport.

### Phase 5: Markdown Integration

**Goal**: Render markdown files.

**Deliverables**:
- `markdown/` package with heading parsing
- Markdeep command or mode in Edwood
- Live update when source changes

**Acceptance**: Open a .md file, trigger Markdeep, see headings rendered larger/bold.

### Phase 6: Expand Markdown Support

**Goal**: Useful markdown rendering.

**Deliverables**:
- Bold, italic, code spans
- Lists (bulleted)
- Code blocks
- Blockquotes

---

## Testing Strategy

### Unit Tests

Each package has `_test.go` files:

```go
// rich/frame_test.go

func TestFrameInit(t *testing.T) {
    // Frame initializes with correct bounds
}

func TestFrameSetContent(t *testing.T) {
    // Content is stored and retrievable
}

func TestFramePtofchar(t *testing.T) {
    // Character positions map to correct screen points
}

func TestFrameCharofpt(t *testing.T) {
    // Screen points map to correct character positions
}

// markdown/parse_test.go

func TestParseHeading(t *testing.T) {
    content := Parse("# Hello")
    // Assert single span with H1 style
}

func TestParseMultipleHeadings(t *testing.T) {
    content := Parse("# H1\n## H2\nplain")
    // Assert three lines with correct styles
}
```

### Visual Tests in Edwood

Create test files that exercise rendering:

```
testdata/
├── headings.md      # All heading levels
├── mixed.md         # Mixed styles
├── long.md          # Tests scrolling
└── selection.md     # Tests selection edge cases
```

### Integration Tests

Test the full pipeline:
1. Parse markdown
2. Convert to Content
3. Render in Frame
4. Verify visual output (screenshot comparison or coordinate checks)

---

## Open Questions

### 1. Package Organization

**Option A**: Single `rich/` package (proposed above)
**Option B**: Separate `richframe/` and `richtext/` packages

Recommendation: Start with single package, split if it grows unwieldy.

### 2. Integration Point

**Option A**: Separate preview window (new window type)
**Option B**: Toggle mode on existing windows
**Option C**: Special body type that windows can use

Recommendation: Option A is simplest to implement without touching existing window code.

### 3. Font Handling

How to handle bold/italic?

**Option A**: Load font variants (e.g., "GoMono-Bold", "GoMono-Italic")
**Option B**: Simulate with color (bold = brighter, italic = different hue)
**Option C**: Use Unicode bold/italic characters (hacky, bad for copy/paste)

Recommendation: Try Option A first, fall back to Option B if fonts unavailable.

### 4. Coordinate Systems

Should rich.Frame use the same coordinate conventions as frame.Frame?

- Frame coordinates: 0-based rune offset within visible frame
- Buffer coordinates: 0-based rune offset in entire content

Recommendation: Yes, match existing conventions for consistency.

### 5. Scrollbar

Reuse existing scrollbar code from `scrl.go` or implement fresh?

Recommendation: Extract scrollbar to shared utility if possible, otherwise copy and adapt.

---

## Appendix: Existing Frame Reference

Key types from `frame/` for reference:

```go
// frame/frame.go - existing box model
type frbox struct {
    Wid    int    // Width in pixels
    Nrune  int    // Rune count (-1 for newline/tab)
    Ptr    []byte // UTF-8 content
    Bc     rune   // '\n' or '\t' for special boxes
    Minwid byte   // Minimum width for tabs
}

// frame/frame.go - color slots
const (
    ColBack   = 0 // Background
    ColHigh   = 1 // Selection highlight
    ColBord   = 2 // Border
    ColText   = 3 // Normal text
    ColHText  = 4 // Highlighted text
)
```

Key methods we'll want equivalents for:
- `Insert` / `Delete` - for rich text, use `SetContent` instead
- `Ptofchar` / `Charofpt` - coordinate mapping (need this)
- `Select` - mouse selection (need this)
- `DrawSel` - selection highlighting (need this)
- `Redraw` - full redraw (need this)

---

---

## Phase 11: Window Integration Design

### Overview

The current `PreviewWindow` implementation is a standalone component. This phase integrates rich text rendering directly into the existing `Window` type, making Markdeep a **toggle mode** rather than a separate window.

### Key Design Decisions

1. **Same Window, Different View**: Markdeep is a mode toggle on the existing window, not a separate linked window
2. **Tag Unchanged**: Window uses its normal tag (filename, standard commands)
3. **Snarf Maps to Source**: Selection in Markdeep maps back to raw markdown for copy operations
4. **Full Participant**: Markdeep windows participate fully in Row/Column layout
5. **Mouse Chords Work**: Look, Exec, and other chords function as in normal windows

### Window Structure Changes

```go
type Window struct {
    // ... existing fields ...

    tag    Text       // Unchanged - normal editable tag
    body   Text       // Normal body (raw markdown when editing)

    // New fields for Markdeep mode
    previewMode bool       // true when showing rendered Markdeep
    richBody    *RichText  // Rich text renderer (created on demand)

    // Source mapping for Snarf
    // Maps preview character positions to source positions
}
```

### Mode Toggle Behavior

**Entering Markdeep Mode** (e.g., "Markdeep" command in tag):
1. Parse `body` content as markdown
2. Create/update `richBody` with rendered content
3. Set `previewMode = true`
4. Redraw shows `richBody` instead of `body`
5. Tag remains editable, body Text still exists (hidden)

**Exiting Markdeep Mode** (toggle "Markdeep" again, or edit tag filename):
1. Set `previewMode = false`
2. Redraw shows normal `body`
3. `richBody` can be retained or cleared

### Tag Behavior

The tag works identically to normal windows:

| Component | Behavior in Markdeep Mode |
|-----------|-------------------------|
| Filename | Displayed normally, editable |
| Del | Closes window |
| Snarf | Copies selected text **from raw markdown source** |
| Put | Saves the raw markdown (useful if filename changed) |
| Undo/Redo | Applies to raw markdown body |
| Get | Reloads from file, re-renders preview |
| Right side (`\|` ...) | User-editable commands as normal |

### Selection and Snarf Mapping

When user selects text in Markdeep mode:

1. **Visual Selection**: Highlight shown in `richBody` at positions `p0..p1`
2. **Source Mapping**: Maintain mapping from preview positions to source positions
3. **Snarf Operation**: Copy raw markdown from `body` at mapped source positions

**Mapping Strategy**:

The markdown parser already tracks source positions. Extend `rich.Span` or create a parallel structure:

```go
// Option A: Extend Span with source info
type Span struct {
    Text       string
    Style      Style
    SourcePos  int  // Byte offset in original markdown
    SourceLen  int  // Length in source (may differ from len(Text))
}

// Option B: Separate mapping structure
type SourceMap struct {
    // Maps rendered rune position to source byte range
    entries []SourceMapEntry
}

type SourceMapEntry struct {
    RenderedStart int  // Rune position in rendered content
    RenderedEnd   int
    SourceStart   int  // Byte position in source markdown
    SourceEnd     int
}
```

**Examples**:

| Source Markdown | Rendered Text | Mapping Notes |
|-----------------|---------------|---------------|
| `**bold**` | `bold` | 4 rendered chars map to 8 source chars |
| `# Heading` | `Heading` | 7 rendered chars map to 9 source chars |
| `normal text` | `normal text` | 1:1 mapping |

### Mouse Chord Behavior

All mouse chords work in preview mode:

| Chord | Behavior |
|-------|----------|
| B1 (click) | Position cursor, start selection |
| B1 (drag) | Extend selection |
| B2 (click) | Execute selected text as command |
| B3 (click) | Look/search for selected text |
| B1+B2 | Cut (no-op in preview - read-only) |
| B1+B3 | Paste (no-op in preview - read-only) |

**Note**: B2 (Exec) and B3 (Look) operate on the **rendered text**, not the source markdown. This is intuitive - if user sees "function_name" and B3-clicks it, they want to search for "function_name".

### Live Updates

When the source markdown changes:

1. `body.file` observer notifies window
2. Window re-parses markdown
3. `richBody` content updated
4. Markdeep redraws

This already works via `PreviewState.BufferChanged()` - needs integration with Window.

### Scrollbar

Reuse existing scrollbar rendering from `scrl.go`:

- `richBody` occupies same rectangle as `body` would
- Scrollbar on left side, same width as normal Text scrollbar
- Same interaction: B1 scroll up, B2 absolute position, B3 scroll down

The current `RichText` component already implements scrollbar. Ensure it matches the visual style and behavior of the standard Text scrollbar.

### Column Participation

Markdeep windows are full citizens:

- **Resize**: Drag tag to resize, works normally
- **Grow**: Grow button works (uses same tag interaction)
- **Move**: Can drag between columns
- **Close**: Del closes window
- **Sort**: Participates in column Sort command

No special handling needed - the window is just a normal window that happens to render its body differently.

### Keyboard Handling

In Markdeep mode:

| Key | Action |
|-----|--------|
| Page Up/Down | Scroll |
| Arrow keys | Scroll (or no-op) |
| Escape | Exit preview mode (optional) |
| Typing | No-op (body is read-only in preview) |

Alternatively, typing could auto-exit preview mode and apply to the body.

### Implementation Notes

**Window.Draw() changes**:
```go
func (w *Window) Draw() {
    // ... draw tag ...

    if w.previewMode && w.richBody != nil {
        w.richBody.Redraw()
    } else {
        w.body.Redraw()
    }
}
```

**Window.Type() changes**:
```go
func (w *Window) Type(r rune) {
    if w.previewMode {
        // Option A: Ignore input
        return
        // Option B: Exit Markdeep mode and type
        // w.SetPreviewMode(false)
        // fall through to normal handling
    }
    // ... existing body typing ...
}
```

**Window.Mouse() changes**:
```go
func (w *Window) Mouse(m *draw.Mouse, mc *draw.Mousectl) {
    // Determine if click is in body area
    if !m.Point.In(w.body.all) {
        // Tag handling unchanged
        return
    }

    if w.previewMode {
        w.richBody.Mouse(m, mc)  // Handles selection, scrollbar
        return
    }
    // ... existing body mouse handling ...
}
```

### Files to Modify

| File | Changes |
|------|---------|
| `wind.go` | Add `previewMode`, `richBody` fields; modify Draw, Type, Mouse |
| `richtext.go` | Ensure compatible with Window integration |
| `preview.go` | Refactor: PreviewWindow becomes helper or is removed |
| `exec.go` | "Markdeep" command toggles mode instead of creating window |
| `text.go` | May need method to get raw content for source mapping |
| `markdown/parse.go` | Return source position mapping |

### Testing Strategy

1. **Unit tests**: Source position mapping accuracy
2. **Integration tests**: Mode toggle, snarf mapping
3. **Visual tests**: Scrollbar matches, selection highlight correct
4. **Interaction tests**: Mouse chords work correctly

---

## Known Issues

### Preview Mode Text Selection Not Working

**Status**: Not implemented
**Severity**: Major functionality gap

#### Problem Description

Text selection (click-and-drag with button 1) does not work in Markdeep preview mode. Users can click to position the cursor, but dragging to select a range of text has no effect. Interestingly, Look-clicking (button 3) on links works correctly.

#### Root Cause Analysis

The issue stems from a mismatch between how mouse events are routed in preview mode versus normal mode.

**Normal text selection flow** (text.go:1165):
```go
sP0, sP1 := t.fr.Select(global.mousectl, global.mouse, ...)
```
The frame's `Select()` method receives the `Mousectl`, which provides a channel to read subsequent mouse events during the drag operation.

**Preview mode flow** (acme.go:457-462):
```go
if w != nil && t.what == Body && w.IsPreviewMode() {
    w.Lock('M')
    w.HandlePreviewMouse(&m)  // Only passes mouse, not mousectl!
    w.Unlock()
    return
}
```

**HandlePreviewMouse** (wind.go:721-732):
```go
if m.Point.In(frameRect) && m.Buttons&1 != 0 {
    charPos := rt.Frame().Charofpt(m.Point)
    rt.SetSelection(charPos, charPos)  // Sets point, not range
    w.Draw()
    return true
}
```

The current implementation:
1. Only handles a single mouse event, not the drag loop
2. Sets selection to a single point `(charPos, charPos)` instead of a range
3. Returns immediately without tracking mouse movement
4. Does not call `Frame.Select(mc, m)` which handles the full drag loop

#### Design Intent (from Phase 11 spec)

The original design specified:
```go
func (w *Window) Mouse(m *draw.Mouse, mc *draw.Mousectl) {
    if w.previewMode {
        w.richBody.Mouse(m, mc)  // Handles selection, scrollbar
        return
    }
}
```

This was intended to pass the `Mousectl` through to enable proper selection.

#### Solution

1. Modify `HandlePreviewMouse` signature to accept `*draw.Mousectl`
2. Update call site in `acme.go` to pass `global.mousectl`
3. When button 1 is pressed in frame area, call `Frame.Select(mc, m)` for drag selection
4. Handle the display flush after selection completes

See PLAN.md Phase 17 for detailed implementation plan.

---

## Notes / Discussion

<!-- Space for iteration - add comments and edits below -->


