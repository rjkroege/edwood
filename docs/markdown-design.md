# Edwood Markdown Preview: Design Document

## Overview

Edwood's markdown preview renders styled markdown directly within editor
windows.  When a `.md` file is opened (or the user runs the **Markdeep** tag
command), the window body switches from plain-text editing to a live-rendered
view.  The system is built on three pillars:

1. **A custom markdown parser** (`markdown/`) that converts markdown source to
   styled spans (`rich.Content`).
2. **A rich-text rendering engine** (`rich/`, `richtext.go`) that draws styled
   spans—headings, bold, italic, code, images, tables, links, horizontal
   rules—into an Acme-style frame with scrollbar.
3. **Window integration** (`wind.go`, `exec.go`, `look.go`) that wires the
   parser and renderer into the editor's existing Window/Text/Frame
   architecture, handling preview toggle, mouse chords, selection, snarf,
   resize, and live updates.

The preview is a live editing environment, though there is limited control for changing formatting.
All Edwood/acme mouse button interactions work exactly as in regular acme.  Instructions for 
carrying the markdown through copy/paste operations appear below.

---

## Architecture

```
┌───────────────────────────────────────────────────────────┐
│ Window                                                    │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ Tag (includes "Markdeep" command for .md files)     │  │
│  ├─────────────────────────────────────────────────────┤  │
│  │ Body (Text)                                         │  │
│  │  ┌──────────────────────────────────────────────┐   │  │
│  │  │ Normal mode: plain Text + Frame              │   │  │
│  │  │ Preview mode: RichText → rich.Frame          │   │  │
│  │  └──────────────────────────────────────────────┘   │  │
│  │                                                     │  │
│  │  previewMode          bool                          │  │
│  │  richBody             *RichText                     │  │
│  │  previewSourceMap     *markdown.SourceMap            │  │
│  │  previewLinkMap       *markdown.LinkMap              │  │
│  │  imageCache           *rich.ImageCache              │  │
│  │  previewUpdateTimer   *time.Timer                   │  │
│  │  selectionContext     *SelectionContext              │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

### Data Flow

```
  source text (body.file)
        │
        ▼
  markdown.ParseWithSourceMap()
        │
        ├──▶ rich.Content   (styled spans)
        ├──▶ SourceMap       (rendered ↔ source position mapping)
        └──▶ LinkMap         (rendered position → URL)
               │
               ▼
         RichText.SetContent()
               │
               ▼
         rich.Frame.SetContent()  →  layout  →  render
               │
               ▼
         draw to body.all rectangle
```

---

## Packages

### `markdown/`

Custom line-oriented markdown parser.  No external dependencies (no goldmark,
blackfriday, etc.).

| File | Purpose |
|------|---------|
| `parse.go` | `Parse(text) → rich.Content`.  Line-at-a-time parser handling all block and inline elements. |
| `sourcemap.go` | `ParseWithSourceMap(text) → (Content, *SourceMap, *LinkMap)`.  Parallel parsing path that tracks byte/rune positions for bidirectional mapping. |
| `linkmap.go` | `LinkMap` — maps rendered rune positions to URLs or in-document strings findable with Look actions. |

#### Supported Markdown Elements

**Block-level:**

| Element | Syntax | Style Fields |
|---------|--------|-------------|
| Heading H1–H6 | `# ` – `###### ` | `Bold=true`, `Scale` 2.0 / 1.5 / 1.25 / 1.125 / 1.0 / 0.875 |
| Fenced code block | ` ``` ` … ` ``` ` | `Code=true`, `Block=true`, `Bg=InlineCodeBg` |
| Indented code block | 4 spaces or 1 tab | Same as fenced |
| Horizontal rule | `---`, `***`, `___` (3+ chars) | `HRule=true`; rendered as 1px gray line using `HRuleRune` (U+2500) |
| Paragraph | blank line between text | `ParaBreak=true` on the break; single newlines are joined with space |
| Unordered list | `- `, `* `, `+ ` | `ListItem`, `ListBullet` (• marker), `ListIndent` (nesting) |
| Ordered list | `1. `, `2) ` etc. | `ListOrdered=true`, `ListNumber` |
| Table | pipe-delimited `\|` rows | `Table=true`, `TableHeader`, `TableAlign`; box-drawing borders (┌─┬─┐ etc.) |
| Blockquote | `> text` | `Blockquote=true`, `BlockquoteDepth` (nesting level) |
| Image | `![alt](url "width=Npx")` | `Image=true`, `ImageURL`, `ImageAlt`, `ImageWidth` |

**Inline (within paragraphs, headings, list items):**

| Element | Syntax | Style Fields |
|---------|--------|-------------|
| Bold | `**text**` | `Bold=true` |
| Italic | `*text*` | `Italic=true` |
| Bold+Italic | `***text***` | `Bold=true`, `Italic=true` |
| Inline code | `` `text` `` | `Code=true` |
| Link | `[text](url)` | `Link=true`, `Fg=LinkBlue` |

#### Parser Design

The parser operates in a single pass over lines:

1. **Line splitting**: `splitLines()` preserves trailing `\n` on each line.
2. **Block-level dispatch** (in order of priority):
   - Fenced code block open/close (` ``` `)
   - Inside fenced block → accumulate verbatim
   - Indented code block (4 spaces / tab)
   - Horizontal rule (`---`, `***`, `___`)
   - Table rows (pipe-delimited)
   - Blockquote (`>` prefix — may nest; inner content re-parsed)
   - Heading (`#` prefix)
   - Unordered/ordered list item
   - Blank line (paragraph break)
   - Plain text (paragraph continuation)
3. **Inline parsing**: `parseInlineFormatting()` scans for `**`, `*`,
   `` ` ``, `[…](…)`, `![…](…)` with proper nesting and escaping.
4. **Span merging**: Consecutive spans with identical styles are merged to
   reduce span count.

#### Source Mapping

Parsing tracks rune position correspondences in both documents.  Each `SourceMapEntry` records:

```go
type SourceMapEntry struct {
    RenderedStart, RenderedEnd     int  // rune positions in rendered content
    SourceStart, SourceEnd         int  // byte positions in source markdown
    SourceRuneStart, SourceRuneEnd int  // rune positions in source
    PrefixLen                      int  // stripped prefix length (e.g., "# ")
}
```

Two key operations:

- **`ToSource(renderedStart, renderedEnd)`** → source rune range.
- **`ToRendered(srcRuneStart, srcRuneEnd)`** → rendered rune range.

#### Blockquotes

Blockquotes use `>` prefixes, optionally nested:

```
> single level
> > nested (depth 2)
> > > depth 3
```

**Parsing:**

1. During line-level dispatch, lines starting with `>` (followed by an
   optional space) are recognized as blockquote lines.  The prefix `> ` is
   stripped and the `BlockquoteDepth` is incremented for each leading `>`.
2. Consecutive blockquote lines at the same depth are grouped into a single
   blockquote block.  A blank line or a line with fewer `>` markers ends the
   current block.
3. The stripped inner content is re-parsed through the same block/inline
   pipeline, so blockquotes can contain paragraphs, headings, lists, code
   blocks, and further nested blockquotes.
4. Source map entries for blockquote content record the `> ` prefix in
   `PrefixLen`, analogous to heading `# ` prefixes.  Nested quotes
   accumulate: a `> > ` prefix at depth 2 has `PrefixLen = 4` (two markers
   plus spaces).

**Rendering (layout engine):**

- Each blockquote depth level adds a left indent (same 20px quantum used for
  list indentation) plus a 2px vertical bar drawn in a muted color
  (`BlockquoteBorderGray`).
- The vertical bar runs the full height of the blockquote block, positioned
  at the left edge of the indent for that depth level.
- Text inside the blockquote wraps within the reduced width.
- Nested blockquotes produce multiple vertical bars, one per depth level.

**Style fields:**

| Field | Type | Meaning |
|-------|------|---------|
| `Blockquote` | `bool` | Span is inside a blockquote |
| `BlockquoteDepth` | `int` | Nesting level (1 = `>`, 2 = `> >`, …) |

---

### `rich/`

The rich text rendering engine, independent of markdown.

| File | Purpose |
|------|---------|
| `style.go` | `Style` struct (all visual attributes), predefined styles (`StyleH1`, `StyleLink`, etc.), constants (`LinkBlue`, `InlineCodeBg`, `HRuleRune`). |
| `span.go` | `Span{Text, Style}` — the fundamental unit.  `Content` is `[]Span`. |
| `frame.go` | `Frame` interface and `frameImpl` — layout, rendering, selection, scrolling, image display, horizontal scrollbar. |
| `layout.go` | Layout engine: word wrap, list indentation, table column sizing, image sizing, block region identification, scrollbar height reservation. |
| `image.go` | `ImageCache` (LRU, thread-safe), PNG/JPEG/GIF loading, Plan 9 image format conversion, RGBA32 with alpha pre-multiplication. |
| `box.go` | Box model helpers. |

#### Frame Interface (key methods)

| Category | Methods |
|----------|---------|
| Lifecycle | `Init(rect, opts...)`, `SetContent(Content)`, `SetRect(rect)`, `Redraw()` |
| Geometry | `Rect()`, `Ptofchar(pos) → Point`, `Charofpt(Point) → pos` |
| Selection | `Select()`, `SetSelection(p0, p1)`, `GetSelection()`, `SelectWithChord(mc, m)`, `SelectWithColor(mc, m, color)`, `ExpandAtPos(pos) → (q0, q1)` |
| Scrolling | `SetOrigin(org)`, `GetOrigin()`, `MaxLines()`, `TotalLines()`, `LineStartRunes()`, `LinePixelHeights()` |
| H-Scroll | `HScrollBarAt(pt)`, `HScrollBarRect(region)`, `HScrollClick(btn, pt, region)`, `HScrollWheel(delta, region)`, `PointInBlockRegion(pt)` |
| Images | `ImageURLAt(pos)` |

#### Font System

The frame supports multiple font variants, loaded at preview initialization:

| Font | Purpose |
|------|---------|
| Base font | Body text |
| Bold | `**text**` |
| Italic | `*text*` |
| Bold+Italic | `***text***` |
| Code (monospace) | Inline code, code blocks |
| Scaled fonts | H1 (2.0×), H2 (1.5×), H3 (1.25×) |

Fonts are loaded via `tryLoadFontVariant()`, `tryLoadCodeFont()`, and
`tryLoadScaledFont()` in `exec.go`.  If a variant is unavailable, the base
font is used as fallback.

#### Layout Engine

The layout engine performs word-wrapping and positioning:

- **Word wrapping**: Breaks at word boundaries; respects frame width.
- **List indentation**: 20px per indent level for `ListItem`/`ListBullet`
  spans, with proper wrapping of continuation lines.
- **Blockquote indentation**: 20px per depth level, with a 2px vertical bar
  on the left edge of each level.  Continuation lines wrap within the
  reduced width.  Blockquote indent stacks with list indent when a list
  appears inside a blockquote.
- **Table columns**: Calculates column widths, pads cells to uniform width,
  applies alignment (left/center/right), renders box-drawing grid lines.  Tables can be horizontally scrolled independently.
- **Image sizing**: Scales images to fit frame width; honors explicit
  `ImageWidth` from markdown `width=Npx` syntax.  Uses bilinear interpolation
  for pre-scaling.  Can be horizontally scrolled independently.
- **Block regions**: Identifies code blocks and tables as contiguous regions
  that can be horizontally scrolled independently.
- **Paragraph spacing**: Extra vertical space before paragraphs proportional to
  text height.

#### Horizontal Scrollbar

Block regions (code blocks, tables, images) with content wider than the frame get an
independent horizontal scrollbar:

- Drawn at the bottom of the block region.
- Thumb width proportional to visible fraction.
- Acme-style semantics: B1=scroll left, B2=jump, B3=scroll right.
- Scroll wheel redirected to horizontal when cursor is over a block region.
- Latching: once pressed, tracks mouse until release.

All scrollable objects must be idented by at least 8ems of space to leave a clean, non-scrolling gutter on the left of the page.

#### Image Support

| Feature | Detail |
|---------|--------|
| Formats | PNG, JPEG, GIF (local files) |
| URL loading | HTTP/HTTPS with TLS retry and timeouts |
| Caching | LRU `ImageCache`, thread-safe, error caching to avoid repeated failures |
| Plan 9 conversion | RGBA32 byte order, alpha pre-multiplication |
| Sizing | Natural size clamped to frame width; explicit `width=Npx` via image title |
| Overflow | Scratch image approach for partial-line clipping |
| Base path | Relative paths resolved against markdown file directory |

---

### `richtext.go` — RichText Component

High-level wrapper combining `rich.Frame` with a vertical scrollbar:

```go
type RichText struct {
    frame          rich.Frame
    content        rich.Content
    display        draw.Display
    // Font variants: boldFont, italicFont, boldItalicFont, codeFont
    // scaledFonts map[float64]draw.Font   (headings)
    // imageCache  *rich.ImageCache
    // basePath    string
    // scrollBg, scrollThumb  draw.Image
    // lastRect, lastScrollRect  image.Rectangle  (cached for hit-testing)
}
```

Key design principle: **single rectangle owner**.  The rectangle is *not*
provided at `Init()` time.  Instead, `Render(rect)` accepts the rectangle
dynamically (from `body.all`), computes scrollbar and frame sub-rectangles
at render time, and caches them for hit-testing.  This ensures correct
behavior across window resizes.

| Method | Purpose |
|--------|---------|
| `Init(display, font, opts...)` | Initialize without rectangle |
| `Render(rect)` | Draw into the given rectangle (from `body.all`) |
| `SetContent(c)` | Set/update content, notify frame |
| `Selection()` / `SetSelection(p0, p1)` | Selection management |
| `GetOrigin()` / `SetOrigin(org)` | Scroll position |
| `ScrollClick(button, pt)` | Handle vertical scrollbar clicks |
| `ScrollWheel(up)` | Handle scroll wheel |
| `ScrollRect()` | Return cached scrollbar rectangle |
| `Frame()` | Access underlying `rich.Frame` |

---

## Window Integration

### Preview Mode Toggle

The **Markdeep** tag command (`previewcmd()` in `exec.go`) toggles preview
mode:

**Entering preview mode:**
1. Validate: must be a `.md` file with a window.
2. Load font variants (bold, italic, bold+italic, code, H1/H2/H3 scaled).
3. Allocate draw colors (white background, black text).
4. Create `RichText` with options (fonts, colors, scrollbar colors, selection
   color, image cache, base path).
5. Parse markdown: `markdown.ParseWithSourceMap()` → `Content`, `SourceMap`,
   `LinkMap`.
6. `rt.SetContent(content)`; store source map, link map on window.
7. Set `w.previewMode = true`.
8. Call `w.Draw()` → `richBody.Render(body.all)`.

**Exiting preview mode:**
1. Clear image cache.
2. Set `w.previewMode = false`.
3. Redraw body text via `body.ScrDraw()` and `body.SetSelect()`.
4. The current selection must be made visible (Text.Show(q0, q1 int, doselect bool) at text.go:1238. )

### Auto-Enable on Open

When `openfile()` in `look.go` creates a new window for a `.md` file, it
automatically calls `previewcmd()` to enter preview mode.

### Tag Integration

The "Markdeep" command is automatically added to the tag line for `.md` files,
alongside the standard Acme commands.

---

## Mouse Handling

It is of primary importance that mouse handling be the same as in a regular Edwood window.
All Acme mouse semantics are preserved:

### Scroll Wheel (Buttons 4/5)

- Over a block region with horizontal scrollbar: horizontal scroll.
- Elsewhere: vertical scroll.

### Vertical Scrollbar (B1/B2/B3 in scrollbar area)

- **B1**: Scroll up (line at cursor → top of view).
- **B2**: Jump (proportional to click position in scrollbar).
- **B3**: Scroll down (top of view → line at cursor).
- Latching: tracks mouse  with cursor warping until button release.

### Horizontal Scrollbar (B1/B2/B3 on block region scrollbar)

Same semantics as vertical, applied to horizontal scroll of code
blocks/tables.

### Typing

Text insertion happens at the source body.q0; currently the refresh is too slow to make this good,
but we'll imrove that in our next iteration.

### B1 (Left Click) — Selection

Should use the same heuristics as acme; this is best done by reversing the q0 location
of the click into the text buffer using the source map, expanding dot there, then displaying the result
in the markdown view using the inverse source map.

1. Click in frame: `Charofpt()` to get rune position.
2. **Double-click detection**: Same `RichText`, same position, within 500ms.
   - Double-click in a code block: `ExpandAtPos()` selects entire block.
   - Double-click elsewhere: `ExpandAtPos()` selects word.
3. **Drag selection**: inline drag loop tracks mouse until a chord is
   detected or all buttons are released.
4. After selection: `syncSourceSelection()` maps rendered selection to source
   `body.q0`/`body.q1` via `SourceMap.ToSource()`.

### B1 Chords

Chords are processed in a loop while B1 is held (matching the `text.go`
pattern), so sequential chords work correctly.  The typical snarf-via-chord
sequence:

| Action | Effect |
|--------|--------|
| B1 down | move pointer, start selection |
| optional B1 down in 200ms | extend selection (double-click) |
| optional sweep | sweep selection |
| B2 down | Cut text (snarf + delete) |
| B2 up | nothing |
| B3 down | Undo cut (paste text back at original location) |
| B3 up | nothing |
| B1 up | end chord sequence |

B2 and B3 toggle: pressing B3 after a cut undoes the cut; pressing B2
after a paste undoes the paste.  This allows the user to change their
mind while B1 is still held.

| Chord | Action |
|-------|--------|
| B1+B2+B3 (simultaneous) | Snarf only (copy to clipboard, no delete) |
| B1+B2 | Cut (snarf + delete) — triggers `UpdatePreview()` |
| B1+B3 | Paste (replace selection) — triggers `UpdatePreview()` |
| B1+B2 then B1+B3 | Cut, then undo cut |
| B1+B3 then B1+B2 | Paste, then undo paste |

All chord operations go through the standard `cut()`/`paste()` functions
for proper undo/clipboard integration.

### B2 (Middle Click) — Execute

1. `SelectWithColor(mc, m, but2col)` for red sweep.
2. Null click: `ExpandAtPos()` for word/block expansion.
3. `syncSourceSelection()` to map selection.
4. Extract text via `PreviewExecText()`.
5. Execute the command.
6. Restore prior selection.

### B3 (Right Click) — Look

1. `SelectWithColor(mc, m, but3col)` for green sweep.
2. Null click: `ExpandAtPos()` for word expansion.
3. **Link handling**: `previewLinkMap.URLAt(pos)` — if the click is on a link,
   plumb the URL to the system browser.
4. **Text search**: Search for the selected/expanded text in the source buffer.
   If found, use `SourceMap.ToRendered()` to highlight the match in the
   preview, then `scrollPreviewToMatch()` to scroll it to ~1/3 from top (like
   Acme's `Show()`).
5. Restore prior selection.

---

## Selection and Snarf

### Selection Sync

When the user selects text in the preview, `syncSourceSelection()` maps the
rendered range back to source positions via `SourceMap.ToSource()` and sets
`body.q0`/`body.q1` accordingly.  Source body `DrawSel` is suppressed in
preview mode.

The selection itself (`body.q0`/`body.q1`) always reflects the literal
mapped positions — it does not include surrounding markup.  Markup
expansion is deferred to operation time (see below).

### Markup-Expanded Selection (q0prime / q1prime)

When an operation (snarf, cut, look, send) is invoked on a non-empty
selection (`q0 != q1`), the source range may need to be enlarged to
include surrounding markup so that the copied/cut/sent text is
self-contained markdown.  The enlarged range is called
`q0prime`/`q1prime`:

- If `q0` sits at the first content rune after an opening marker (e.g.
  the `t` in `**text**`), `q0prime` backs up to include the marker.
- If `q1` sits at the last content rune before a closing marker,
  `q1prime` advances to include the marker.
- The heuristic applies to bold, italic, code spans, headings, list
  markers, blockquote prefixes, image syntax, and table delimiters.

`expandSelectionForOperation(q0, q1) → (q0prime, q1prime)` computes this
from the source map entries surrounding the selection.  Each chord and
command that acts on the selection calls this function rather than using
`body.q0`/`body.q1` directly.

The visual highlight in the preview remains at the original `q0`/`q1` so
the user sees exactly what they selected; only the buffer range passed
to the operation is widened.

### Snarf (Copy)

`PreviewSnarf()` uses `expandSelectionForOperation()` to obtain
`q0prime`/`q1prime`, then extracts the *source markdown* (with `**`,
`#`, etc.) from that range.  This preserves formatting when pasting.

### Context-Aware Paste

A `SelectionContext` tracks metadata about the current selection:

```go
type SelectionContext struct {
    SourceStart, SourceEnd     int
    RenderedStart, RenderedEnd int
    ContentType                SelectionContentType  // heading, bold, code, etc.
    PrimaryStyle               rich.Style
}
```

`analyzeSelectionContent()` classifies what was selected.  `transformForPaste()`
adapts pasted text to the target context (e.g., wrapping in `**` when pasting
into a bold region, stripping `#` when pasting a heading into plain text).

---

## Live Updates

Make sure all the update operations are separate from other semantics. We will 
replace the re-render version we have now with an incremental, much more efficient
versionin the near future.

`UpdatePreview()` re-parses the markdown from `body.file` and refreshes the
rendered view:

1. Read `body.file.String()`.
2. `markdown.ParseWithSourceMap()` → new content, source map, link map.
3. `richBody.SetContent(content)`.
4. Preserve scroll position (clamp to new content length).
5. Re-render via `richBody.Render(body.all)`.

`SchedulePreviewUpdate()` debounces updates with a 3-second timer to avoid
excessive re-rendering during rapid editing.

---

## Synchronized Scrolling

The preview and the source text are two views of the same content.  Their
scroll positions must stay visually in sync: scrolling in one view adjusts
the other so that the topmost visible line corresponds to the same logical
position in both.

### Source → Preview

When the user scrolls or navigates in the source text view (e.g. via the
scrollbar, keyboard, or `Show()`), the source origin (`body.org`, a rune
offset into the source buffer) changes.  The preview reacts:

1. Map `body.org` through `SourceMap.ToRendered()` to get the corresponding
   rendered rune position.
2. Call `richBody.SetOrigin()` with that position.
3. Re-render the preview at the new origin.

### Preview → Source

When the user scrolls in the preview (vertical scrollbar, scroll wheel),
the preview origin (a rendered rune offset) changes.  The source view
reacts:

1. Map the preview origin through `SourceMap.ToSource()` to get the
   corresponding source rune position.
2. Set `body.org` to that position.
3. Redraw the source scrollbar to reflect the new position.  (The source
   body text itself is not drawn in preview mode, but the scrollbar
   thumb must update so that toggling out of preview mode shows the
   right location.)

### Edge Cases

- **Content length mismatch**: The rendered content is generally shorter
  than the source (stripped markup).  Clamping is needed at both ends.
- **No source map entry**: Positions in gaps between entries (e.g.
  paragraph-break newlines) use the nearest entry's boundary.
- **Scroll granularity**: The source view scrolls by lines; the preview
  scrolls by pixel-height layout lines.  Exact pixel-perfect alignment
  is not required — the goal is that the same *content* is at the top
  of both views, not the same pixel offset.

---

## Resize Handling

The **single rectangle owner** pattern ensures correct resize behavior:

1. `body.all` is the canonical geometry.
2. `Window.Resize()` passes `noredraw=true` to `body.Resize()` in preview mode.
3. Calls `richBody.Render(body.all)` with the new geometry.
4. `Render()` recomputes scrollbar and frame sub-rectangles from the new
   `body.all`.
5. Cached `lastRect`/`lastScrollRect` are updated for hit-testing.

The `rich.Frame` supports `SetRect()` for updating its rectangle without full
re-initialization.

---

## Cursor Tick

A cursor tick (insertion point indicator) is drawn when the selection is a
point (p0 == p1):

- **Appearance**: Transparent mask with a vertical line + serifs (matching Acme
  style).
- **Positioning**: `drawTickTo()` walks the layout to find the screen position
  for a rune offset.
- **Integration**: Rendered during `Redraw()` when `p0 == p1`.

---

## Package Dependencies

```
exec.go, wind.go, look.go  (main package)
    │
    ├── markdown/           (parser, source map, link map)
    │      └── rich/        (Style, Span, Content types)
    │
    ├── richtext.go         (RichText wrapper)
    │      └── rich/        (Frame, layout, image cache)
    │
    └── draw/               (display, images, fonts — Plan 9 graphics)
```

The `markdown/` and `rich/` packages have no dependency on the main package or
on each other beyond the `rich.Style`/`rich.Span`/`rich.Content` types.

---

## Key Data Types

### `rich.Style` (style.go)

```go
type Style struct {
    Fg, Bg      color.Color    // foreground/background (nil = default)
    Bold        bool
    Italic      bool
    Code        bool           // monospace font
    Link        bool           // blue hyperlink
    Block       bool           // full-width background (code blocks)
    HRule       bool           // horizontal rule marker
    ParaBreak   bool           // extra vertical spacing
    ListItem    bool           // list item content
    ListBullet  bool           // bullet/number marker
    ListIndent  int            // nesting level
    ListOrdered bool           // ordered vs unordered
    ListNumber  int            // item number for ordered lists
    Table       bool           // part of a table
    TableHeader bool           // header cell
    TableAlign  Alignment      // left/center/right
    Blockquote      bool       // inside a blockquote
    BlockquoteDepth int        // nesting level (1 = `>`, 2 = `>>`, …)
    Image       bool           // image placeholder
    ImageURL    string         // image source URL/path
    ImageAlt    string         // alt text
    ImageWidth  int            // explicit width in pixels (0 = natural)
    Scale       float64        // size multiplier (1.0 = normal)
}
```

### `rich.Content`

```go
type Content []Span
type Span struct {
    Text  string
    Style Style
}
```

### `markdown.SourceMap`

Bidirectional mapping between rendered rune positions and source rune
positions.  Entries are built during parsing and account for stripped
prefixes (heading `# `, list markers `- `, formatting `**`) and
non-rendered syntax (fence delimiters, table grid characters).

### `markdown.LinkMap`

Simple list of `{Start, End int; URL string}` entries.
`URLAt(pos)` returns the URL containing a given rune position.

---

## Known Issues and Improvement Areas

2. **Debounce timer thread safety**: `SchedulePreviewUpdate()` uses
   `time.AfterFunc` which fires on a separate goroutine.  The comment notes
   that `UpdatePreview` should be called from the main goroutine, but the
   current implementation calls it directly from the timer callback.

3. **No syntax highlighting**: Code blocks are rendered in monospace with a
   gray background but no language-specific coloring.

4. **Limited nested block elements**: Lists cannot contain code blocks or
   tables.  Blockquotes support nested inner content (paragraphs, headings,
   lists, code blocks, nested blockquotes) but list→code-block and
   list→table nesting is not yet supported.

5. **Image loading on main thread**: Image loading (including HTTP fetches)
   appears to happen synchronously during parsing/rendering.

6. **Table layout is character-based**: Tables use box-drawing characters
   and monospace column padding rather than true proportional column layout.
