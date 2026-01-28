# Markdeep Selection and Interaction Design

## Overview

This document describes the mouse interaction model for Edwood's Markdeep preview mode, ensuring parity with Acme's standard mouse-driven interface while handling the complexities of rendered-to-source text mapping.

### Goals

1. Provide the same mouse interaction semantics as standard Acme windows
2. Map selections in rendered preview back to source markdown accurately
3. Support Cut, Paste, and Snarf operations with proper context handling
4. Handle boundary conditions at markdown element boundaries gracefully

### References

- [Acme: A User Interface for Programmers](https://plan9.io/sys/doc/acme/acme.html)
- [Acme man page](https://9fans.github.io/plan9port/man/man1/acme.html)

---

## Acme Mouse Interaction Model

### The Three Buttons

| Button | Name | Action |
|--------|------|--------|
| B1 (Left) | Select | Sweep to select text; double-click selects word/line |
| B2 (Middle) | Execute | Execute the swept/clicked text as a command |
| B3 (Right) | Look | Locate/open file, or search for text |

### Chording Operations

When B1 is held with a selection:

| Chord | Action | Effect |
|-------|--------|--------|
| B1 + B2 | Cut | Delete selection, copy to snarf buffer |
| B1 + B3 | Paste | Replace selection with snarf buffer |
| B1 + B2 + B3 | Snarf | Copy selection to snarf buffer (no delete) |

### Selection Mechanics

- **B1 sweep**: Creates a text selection from start to end position
- **B2 sweep**: Executes the swept text as a command
- **B3 sweep**: Looks up/searches for the swept text
- **Null click**: Expands to word under cursor (B2/B3) or sets insertion point (B1)

---

## Markdeep Preview Model

### Key Challenge: Two Text Buffers

In Markdeep preview mode, we have two distinct text representations:

1. **Source text** (`w.body`): The raw markdown in the standard Text buffer
2. **Rendered text** (`w.richBody`): The styled preview in the RichText frame

All Acme operations (Cut, Paste, Execute, Look) must operate on the **source text**, but the user interacts visually with the **rendered text**.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Markdeep Window                          │
├─────────────────────────────────────────────────────────────┤
│  Tag: /path/to/file.md  Del Snarf | Markdeep Snarf         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Rendered Preview (richBody)             │   │
│  │                                                      │   │
│  │  # Heading → "Heading" (rendered, large, bold)      │   │
│  │  **bold** → "bold" (rendered, bold)                 │   │
│  │  [link](url) → "link" (rendered, blue, underlined)  │   │
│  │                                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ↕ SourceMap                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Source Text (body, hidden)              │   │
│  │                                                      │   │
│  │  # Heading                                          │   │
│  │  **bold**                                           │   │
│  │  [link](url)                                        │   │
│  │                                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Selection Translation

### SourceMap Structure

The `SourceMap` tracks correspondences between rendered and source positions:

```go
type SourceMapEntry struct {
    RenderedStart int // Rune position in rendered content
    RenderedEnd   int
    SourceStart   int // Byte position in source markdown
    SourceEnd     int
    PrefixLen     int // Length of source prefix not in rendered (e.g., "# ")
}
```

### Translation Rules

#### Rule 1: Selection at Element Boundary (Include Markup)

When a selection starts exactly at the beginning of a formatted element, include the opening markup in the source selection.

**Example: Heading**
```
Rendered: |Heading|  (user selects from start)
Source:   # Heading
Result:   |# Heading| (includes "# ")
```

#### Rule 2: Selection Within Element (Exclude Markup)

When a selection starts within a formatted element, map directly without including markup.

**Example: Heading**
```
Rendered: He|ading|  (user selects mid-word)
Source:   # Heading
Result:   # He|ading| (excludes "# ")
```

#### Rule 3: Selection Spanning Elements

When a selection spans multiple elements, map each endpoint independently.

**Example: Across bold and plain**
```
Rendered: Some |bold text| here
Source:   Some **bold text** here
Result:   Some **|bold text|** here  (includes closing **)
```

---

## Boundary Conditions

### Case 1: Heading Selection

| User Action | Rendered Selection | Source Selection |
|-------------|-------------------|------------------|
| Select from start of "Heading" | `|Heading|` | `|# Heading|` |
| Select "eading" only | `H|eading|` | `# H|eading|` |
| Select across heading to next line | `|Heading\nText|` | `|# Heading\nText|` |

### Case 2: Bold/Italic Text

| User Action | Rendered Selection | Source Selection |
|-------------|-------------------|------------------|
| Select entire bold word | `|bold|` | `|**bold**|` |
| Select partial bold | `b|ol|d` | `**b|ol|d**` |
| Select into bold from plain | `plain |bold|` | `plain |**bold**|` |

### Case 3: Code Blocks

**Fenced code block:**
```markdown
```go
func main() {}
```
```

| User Action | Rendered Selection | Source Selection |
|-------------|-------------------|------------------|
| Select from block start | `|func main()...|` | `|```go\nfunc main()...|` |
| Select within block | `func |main|() {}` | `func |main|() {}` |
| Select entire block | Full block | Include fences and language |

**Inline code:**
| User Action | Rendered Selection | Source Selection |
|-------------|-------------------|------------------|
| Select entire inline code | `|code|` | `|\`code\`|` |
| Select partial | `co|de|` | `\`co|de|\`` |

### Case 4: Links

```markdown
[link text](https://example.com)
```

| User Action | Rendered Selection | Source Selection |
|-------------|-------------------|------------------|
| Select link text | `|link text|` | `|[link text](https://example.com)|` |
| Select partial link text | `link |text|` | `[link |text|](https://example.com)` |

### Case 5: Images

```markdown
![alt text](image.png)
```

| User Action | Rendered Selection | Source Selection |
|-------------|-------------------|------------------|
| Click on image | (image placeholder) | `|![alt text](image.png)|` |

---

## Selection Context Tracking

### The Problem

When the user performs Cut or Paste in preview mode, we need to:
1. Know what kind of content was selected (plain text, formatted, code, etc.)
2. Apply appropriate transformations when pasting

### SelectionContext Structure

```go
// SelectionContext tracks metadata about the current selection for
// Cut/Paste operations in preview mode.
type SelectionContext struct {
    // Position in source text (body)
    SourceStart int
    SourceEnd   int

    // Position in rendered text (richBody)
    RenderedStart int
    RenderedEnd   int

    // What kind of content is selected
    ContentType SelectionContentType

    // Style information for the selection
    PrimaryStyle rich.Style

    // For code blocks: the language specifier
    CodeLanguage string

    // Whether selection includes structural markers
    IncludesOpenMarker  bool
    IncludesCloseMarker bool
}

type SelectionContentType int

const (
    ContentPlain SelectionContentType = iota
    ContentHeading
    ContentBold
    ContentItalic
    ContentBoldItalic
    ContentCode        // inline code
    ContentCodeBlock   // fenced code block
    ContentLink
    ContentImage
    ContentMixed       // spans multiple types
)
```

### Context Usage

#### On Selection Change

When the user sweeps a new selection:

```go
func (w *Window) updateSelectionContext() {
    // 1. Get rendered selection from richBody
    rStart, rEnd := w.richBody.Selection()

    // 2. Translate to source positions
    sStart, sEnd := w.previewSourceMap.ToSource(rStart, rEnd)

    // 3. Analyze content type from spans
    contentType := w.analyzeSelectionContent(rStart, rEnd)

    // 4. Store context
    w.selectionContext = &SelectionContext{
        SourceStart:   sStart,
        SourceEnd:     sEnd,
        RenderedStart: rStart,
        RenderedEnd:   rEnd,
        ContentType:   contentType,
        // ... fill in other fields
    }
}
```

#### On Cut (B1 + B2)

```go
func (w *Window) previewCut() {
    ctx := w.selectionContext
    if ctx == nil {
        return
    }

    // 1. Get source text for the selection
    srcText := w.body.file.ReadRange(ctx.SourceStart, ctx.SourceEnd)

    // 2. Copy to snarf buffer with context metadata
    w.snarfWithContext(srcText, ctx)

    // 3. Delete from source
    w.body.Delete(ctx.SourceStart, ctx.SourceEnd)

    // 4. Re-render preview
    w.RefreshPreview()
}
```

#### On Paste (B1 + B3)

```go
func (w *Window) previewPaste() {
    // 1. Get snarf buffer content
    content, snarfCtx := w.getSnarf()

    // 2. Get current selection/insertion point
    ctx := w.selectionContext

    // 3. Determine if transformation is needed
    transformed := w.transformForPaste(content, snarfCtx, ctx)

    // 4. Replace in source
    w.body.Replace(ctx.SourceStart, ctx.SourceEnd, transformed)

    // 5. Re-render preview
    w.RefreshPreview()
}
```

---

## Mouse Event Handling

### Current Flow

```
mousethread()
  → MovedMouse()
    → HandlePreviewMouse(m, mc)
      → B1: Frame.Select() + SetSelection()
      → B3: Look up link/image or search
      → Scroll wheel: ScrollWheel()
```

### Enhanced Flow

```
mousethread()
  → MovedMouse()
    → HandlePreviewMouse(m, mc)
      → B1:
          → Frame.Select() - visual selection in rendered
          → updateSelectionContext() - compute source mapping
          → syncSourceSelection() - update body.q0/q1
      → B1 + B2 (chord):
          → previewCut()
      → B1 + B3 (chord):
          → previewPaste()
      → B2:
          → getRenderedSelection() or expandWord()
          → translateToSource()
          → execute() with source text
      → B3:
          → getRenderedSelection() or expandWord()
          → Look for links first
          → Otherwise search in source
```

### Synchronized Selection

The source text (`body`) selection should mirror the rendered selection:

```go
func (w *Window) syncSourceSelection() {
    ctx := w.selectionContext
    if ctx == nil {
        return
    }

    // Update the underlying body's selection to match
    // This enables standard Acme operations (Get, Put, etc.) to work
    w.body.q0 = ctx.SourceStart
    w.body.q1 = ctx.SourceEnd
}
```

---

## Implementation Phases

### Phase 1: Basic Selection Sync

1. Enhance B1 sweep to update `selectionContext`
2. Implement `syncSourceSelection()` to keep body.q0/q1 in sync
3. Enable Snarf command in tag to copy source text

**Test cases:**
- Select heading text → Snarf → verify source markdown copied
- Select bold text → Snarf → verify `**...**` included when appropriate
- Select partial text → Snarf → verify correct mapping

### Phase 2: Execute (B2)

1. Implement B2 sweep in preview
2. Translate selection to source
3. Execute as standard Acme command

**Test cases:**
- B2 on "ls" → executes ls in window directory
- B2 on "Edit" → opens Edit menu
- B2 sweep on rendered command → correct execution

### Phase 3: Look (B3) Enhancement

1. Already have link/image plumbing
2. Add text search fallback
3. Implement B3 sweep (not just click)

**Test cases:**
- B3 on link → plumbs URL
- B3 on non-link text → searches in body
- B3 sweep → searches for swept text

### Phase 4: Chording

1. Implement B1+B2 (Cut) in preview
2. Implement B1+B3 (Paste) in preview
3. Implement B1+B2+B3 (Snarf) in preview
4. Track selection context for transformations

**Test cases:**
- Cut heading → paste elsewhere → verify markdown structure
- Cut code block → paste in plain text → appropriate handling
- Snarf then paste → original unchanged

### Phase 5: Context-Aware Paste

1. Store content type in snarf metadata
2. Transform pasted content based on destination context
3. Handle edge cases (paste code into heading, etc.)

**Test cases:**
- Copy from code block → paste in regular text
- Copy heading → paste mid-paragraph
- Copy link → paste (preserve or transform?)

---

## Design Decisions

### Paste Transformation Policy

**Rule: Adapt formatting to destination context.**

Transformation rules:

1. **Partial formatted text** (e.g., selecting part of bold text):
   - Re-apply the formatting at destination: wrap pasted text in `**...**`
   - Exception: if destination is already inside bold, just insert the text (inherits context)

2. **Plain text into formatted context**:
   - Just insert the text; it inherits formatting from destination context naturally

3. **Structural elements (headings, code blocks)**:
   - **With trailing newline**: Treat as structural move (preserve `#` prefix, fences, etc.)
   - **Without trailing newline**: Treat as "just text" - strip structural markers

**Examples:**

| Source Selection | Includes Newline? | Paste Result |
|-----------------|-------------------|--------------|
| `|# Heading\n|` | Yes | `# Heading\n` (structural heading) |
| `|# Heading|` | No | `Heading` (just text) |
| `|**bold**|` (full) | N/A | `**bold**` (preserves) |
| `**|bol|d**` (partial) | N/A | `**bol**` at dest (re-wraps) |
| `**|bol|d**` into bold | N/A | `bol` (inherits existing bold) |

### Selection Highlight

**Rule: Use the same colors as source text selection.**

This maintains visual consistency with the rest of Acme.

### Partial Element Selection

**Rule: Do not expand selection. The swept section is exactly what gets cut/copied.**

If user selects `bol` from `**bold**`, they get `bol` (with formatting re-applied at paste destination per above rules).

### Undo Behavior

**Rule: Undo is managed entirely by the source text's undo buffer.**

Since all preview mode edits operate on `body` using the same primitives as normal editing (Delete, Insert, etc.), the undo stack remains correct. No special handling needed.

---

## Appendix: SourceMap Enhancement

The current SourceMap may need enhancement for precise boundary handling:

```go
type SourceMapEntry struct {
    RenderedStart int
    RenderedEnd   int
    SourceStart   int
    SourceEnd     int
    PrefixLen     int      // "# " for headings, "**" for bold, etc.
    SuffixLen     int      // "**" for bold, ")" for links, etc.
    ElementType   ElementType
    Nestable      bool     // Can contain other formatted elements
}

type ElementType int

const (
    ElementPlain ElementType = iota
    ElementHeading1
    ElementHeading2
    ElementHeading3
    ElementBold
    ElementItalic
    ElementCode
    ElementCodeBlock
    ElementLink
    ElementImage
)
```

This allows precise decisions about when to include/exclude markers based on selection boundaries.
