# Unified Inline Parser Design

## Problem

The markdown package has **6 parallel functions** implementing the same inline formatting logic. They have already diverged, producing bugs where fixes to one copy are missed in the other.

### Existing Functions

#### parse.go (4 functions)

| Function | Links | Images | List Fields | Signature |
|----------|-------|--------|-------------|-----------|
| `parseInlineFormatting` | yes | yes | no | `(text string, baseStyle rich.Style) []rich.Span` |
| `parseInlineFormattingNoLinks` | no | no | no | `(text string, baseStyle rich.Style) []rich.Span` |
| `parseInlineFormattingWithListStyle` | yes | yes | yes | `(text string, baseStyle rich.Style) []rich.Span` |
| `parseInlineFormattingWithListStyleNoLinks` | no | no | yes | `(text string, baseStyle rich.Style) []rich.Span` |

#### sourcemap.go (2 functions)

| Function | Links | Images | List Fields | Source Map | Link Map | Signature |
|----------|-------|--------|-------------|------------|----------|-----------|
| `parseInlineWithSourceMap` | yes | yes | **no (bug)** | yes | yes | `(text string, baseStyle rich.Style, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry)` |
| `parseInlineWithSourceMapNoLinks` | no | no | **no (bug)** | yes | no | `(text string, baseStyle rich.Style, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry)` |

### Known Divergence: List Style Fields Lost in Source Map Path

The source map functions construct new `rich.Style` values for formatted spans (bold, italic, code) but do **not** propagate `ListItem`, `ListIndent`, `ListOrdered`, or `ListNumber` from `baseStyle`. The `parse.go` "WithListStyle" variants explicitly propagate these fields.

Example — bold span in `parseInlineWithSourceMap` (`sourcemap.go:1067`):
```go
Style: rich.Style{
    Fg:     baseStyle.Fg,
    Bg:     baseStyle.Bg,
    Bold:   true,
    Italic: baseStyle.Italic,
    Scale:  baseStyle.Scale,
    // Missing: ListItem, ListIndent, ListOrdered, ListNumber
}
```

Versus bold span in `parseInlineFormattingWithListStyle` (`parse.go:812`):
```go
Style: rich.Style{
    Fg:          baseStyle.Fg,
    Bg:          baseStyle.Bg,
    Bold:        true,
    Italic:      baseStyle.Italic,
    ListItem:    baseStyle.ListItem,
    ListIndent:  baseStyle.ListIndent,
    ListOrdered: baseStyle.ListOrdered,
    ListNumber:  baseStyle.ListNumber,
    Scale:       baseStyle.Scale,
}
```

This means bold/italic/code text within list items rendered via `ParseWithSourceMap` loses its list styling, potentially causing layout differences between the `Parse` and `ParseWithSourceMap` paths.

### Two Axes of Variation

The 6 functions vary along exactly two independent axes:

1. **Link/image parsing**: on or off (off prevents infinite recursion when parsing text inside `[link text](url)`)
2. **Source map generation**: on or off (off for `Parse`, on for `ParseWithSourceMap`)

The "list style" axis is a false distinction — **all** functions should propagate all fields from `baseStyle`. The current approach of manually listing which fields to copy is the root cause of the divergence bug.

---

## Design

### Unified Function Signature

```go
// parseInline parses inline formatting (bold, italic, code, links, images)
// within a text string and returns styled spans.
//
// Options control link/image parsing and source map generation:
//   - opts.NoLinks: set true to disable link/image recognition (used inside link text)
//   - opts.SourceMap: if non-nil, source map entries are appended
//   - opts.LinkMap: if non-nil, link entries are appended
//   - opts.SourceOffset: byte offset in source (for source map)
//   - opts.RenderedOffset: rune offset in rendered content (for source map)
func parseInline(text string, baseStyle rich.Style, opts InlineOpts) []rich.Span
```

### Options Struct

```go
// InlineOpts controls optional behavior of the unified inline parser.
type InlineOpts struct {
    // NoLinks disables link and image parsing. Set true when parsing
    // text inside a link label to prevent infinite recursion.
    NoLinks bool

    // SourceMap, if non-nil, receives source map entries for each
    // parsed element. Entries are appended (not replaced).
    SourceMap *[]SourceMapEntry

    // LinkMap, if non-nil, receives link entries for each parsed link.
    LinkMap *[]LinkEntry

    // SourceOffset is the byte position in the original source text
    // where `text` begins. Only used when SourceMap is non-nil.
    SourceOffset int

    // RenderedOffset is the rune position in the rendered content
    // where output spans begin. Only used when SourceMap is non-nil.
    RenderedOffset int
}
```

### Style Construction: Copy-All-Then-Override

The root cause of the divergence is manual field-by-field style construction. The fix is to **copy the entire baseStyle and then override specific fields**:

```go
// For bold:
s := baseStyle
s.Bold = true
// s retains ListItem, ListIndent, Link, etc. from baseStyle automatically.

// For inline code:
s := baseStyle
s.Code = true
s.Bg = rich.InlineCodeBg

// For link:
s := baseStyle
s.Link = true
s.Fg = rich.LinkBlue
```

This eliminates the possibility of forgetting to propagate a field and makes it trivial to add new `rich.Style` fields in the future.

### Source Map: Conditional Emission

When `opts.SourceMap` is non-nil, the function tracks `srcPos` and `rendPos` and appends entries. When nil, those variables and the append calls are simply skipped. The cost of the nil check per element is negligible.

```go
if opts.SourceMap != nil {
    *opts.SourceMap = append(*opts.SourceMap, SourceMapEntry{
        RenderedStart: rendPos,
        RenderedEnd:   rendPos + innerLen,
        SourceStart:   srcPos,
        SourceEnd:     srcPos + sourceLen,
    })
}
```

### Link/Image Handling: Single Flag

When `opts.NoLinks` is true, the `[` and `![` checks are skipped entirely (those characters are treated as plain text). This matches the existing behavior of the "NoLinks" variants.

When parsing link text recursively, the function calls itself with `NoLinks: true`:

```go
linkSpans := parseInline(linkText, linkStyle, InlineOpts{
    NoLinks:        true,
    SourceMap:      opts.SourceMap,
    LinkMap:        opts.LinkMap,
    SourceOffset:   srcPos + 1, // past the [
    RenderedOffset: rendPos,
})
```

### Inline Elements (parsing order, unchanged)

1. `![` — Image (if `!opts.NoLinks`)
2. `[` — Link (if `!opts.NoLinks`)
3. `` ` `` — Inline code
4. `***` — Bold+italic
5. `**` — Bold
6. `*` — Italic
7. Regular character (fallthrough)

This order is the same as all existing functions. Code spans are checked before bold/italic so that asterisks inside code are literal.

### Callers After Migration

| Current Call | Replacement |
|---|---|
| `parseInlineFormatting(text, style)` | `parseInline(text, style, InlineOpts{})` |
| `parseInlineFormattingNoLinks(text, style)` | `parseInline(text, style, InlineOpts{NoLinks: true})` |
| `parseInlineFormattingWithListStyle(text, style)` | `parseInline(text, style, InlineOpts{})` — list fields already in baseStyle |
| `parseInlineFormattingWithListStyleNoLinks(text, style)` | `parseInline(text, style, InlineOpts{NoLinks: true})` — list fields already in baseStyle |
| `parseInlineWithSourceMap(text, style, srcOff, rendOff)` | `parseInline(text, style, InlineOpts{SourceMap: &entries, LinkMap: &links, SourceOffset: srcOff, RenderedOffset: rendOff})` |
| `parseInlineWithSourceMapNoLinks(text, style, srcOff, rendOff)` | `parseInline(text, style, InlineOpts{NoLinks: true, SourceMap: &entries, SourceOffset: srcOff, RenderedOffset: rendOff})` |

Note: The "WithListStyle" functions become unnecessary because `parseInline` uses copy-all-then-override, so list fields in `baseStyle` are automatically preserved.

---

## Implementation Plan

### File: `markdown/inline.go`

New file containing:
- `InlineOpts` struct
- `parseInline()` function

### Pseudocode

```go
func parseInline(text string, baseStyle rich.Style, opts InlineOpts) []rich.Span {
    var spans []rich.Span
    var currentText strings.Builder
    i := 0

    // Source map tracking (only used when opts.SourceMap != nil)
    srcPos := opts.SourceOffset
    rendPos := opts.RenderedOffset

    flushPlain := func() {
        if currentText.Len() > 0 {
            spans = append(spans, rich.Span{
                Text:  currentText.String(),
                Style: baseStyle,
            })
            currentText.Reset()
        }
    }

    // addSourceEntry appends a source map entry if tracking is enabled.
    addSourceEntry := func(rendStart, rendEnd, srcStart, srcEnd int) {
        if opts.SourceMap != nil {
            *opts.SourceMap = append(*opts.SourceMap, SourceMapEntry{
                RenderedStart: rendStart,
                RenderedEnd:   rendEnd,
                SourceStart:   srcStart,
                SourceEnd:     srcEnd,
            })
        }
    }

    for i < len(text) {
        // 1. Image: ![alt](url) — skip if NoLinks
        if !opts.NoLinks && text[i] == '!' && i+1 < len(text) && text[i+1] == '[' {
            // ... parse image, flushPlain, append span + source entry
            continue
        }

        // 2. Link: [text](url) — skip if NoLinks
        if !opts.NoLinks && text[i] == '[' {
            // ... parse link, flushPlain
            // Recursive call for link text:
            //   parseInline(linkText, linkStyle, InlineOpts{
            //       NoLinks: true, SourceMap: opts.SourceMap, ...
            //   })
            // Append link entry to opts.LinkMap if non-nil
            continue
        }

        // 3. Inline code: `text`
        if text[i] == '`' {
            // ... parse code span
            // Style: s := baseStyle; s.Code = true; s.Bg = rich.InlineCodeBg
            continue
        }

        // 4. Bold+italic: ***text***
        if i+2 < len(text) && text[i:i+3] == "***" {
            // Style: s := baseStyle; s.Bold = true; s.Italic = true
            continue
        }

        // 5. Bold: **text**
        if i+1 < len(text) && text[i:i+2] == "**" {
            // Style: s := baseStyle; s.Bold = true
            continue
        }

        // 6. Italic: *text*
        if text[i] == '*' {
            // Style: s := baseStyle; s.Italic = true
            continue
        }

        // 7. Regular character
        currentText.WriteByte(text[i])
        addSourceEntry(rendPos, rendPos+1, srcPos, srcPos+1)
        rendPos++; srcPos++; i++
    }

    flushPlain()

    if len(spans) == 0 && text != "" {
        spans = []rich.Span{{Text: text, Style: baseStyle}}
        addSourceEntry(opts.RenderedOffset, opts.RenderedOffset+len([]rune(text)),
                        opts.SourceOffset, opts.SourceOffset+len(text))
    }

    return spans
}
```

### Migration Strategy

The migration happens in two phases (PLAN.md 2.3 and 2.4):

1. **Phase 2.3**: Replace 4 `parseInlineFormatting*` callers in `parse.go` with `parseInline`. All existing `parse_test.go` tests must pass unchanged.

2. **Phase 2.4**: Replace 2 `parseInlineWithSourceMap*` callers in `sourcemap.go` with `parseInline` (with source map opts). All existing `sourcemap_test.go` tests must pass unchanged. Remove the TODO at `sourcemap.go:242`.

### Risks and Mitigations

**Risk**: Copy-all style changes observable behavior for spans that previously didn't carry certain fields.
**Mitigation**: The only change is that list fields are now preserved in the sourcemap path (fixing the known bug). For the parse.go path, `parseInlineFormatting` (without list style) is only called from `parseLine` where `baseStyle` is `rich.DefaultStyle()` (which has zero-valued list fields), so the copy-all produces identical output.

**Risk**: Source map entry generation in the unified function doesn't match existing behavior exactly.
**Mitigation**: The unified function follows the same byte-level tracking logic as `parseInlineWithSourceMap`. The test suite from Phase 1.2 provides regression coverage for round-trip correctness.

**Risk**: Performance regression from nil checks on `opts.SourceMap` for every element.
**Mitigation**: Branch prediction makes nil checks essentially free. The allocation pattern is identical (same number of entries appended). No new allocations on the non-sourcemap path since `opts.SourceMap` is nil.

---

## Test Cases

Test file: `markdown/inline_test.go`

### Category A: Basic Formatting (spans only, no source map)

| Test | Input | Expected Spans |
|------|-------|----------------|
| Plain text | `"hello world"` | 1 span, default style |
| Bold | `"**bold**"` | 1 span, Bold=true |
| Italic | `"*italic*"` | 1 span, Italic=true |
| Bold+italic | `"***both***"` | 1 span, Bold=true, Italic=true |
| Inline code | `` "`code`" `` | 1 span, Code=true, Bg=InlineCodeBg |
| Mixed | `"a **b** c"` | 3 spans: plain, bold, plain |
| Unclosed bold | `"**oops"` | 1 span, text="**oops", default style |
| Unclosed italic | `"*oops"` | 1 span (may vary — existing behavior) |

### Category B: Links and Images

| Test | Input | NoLinks | Expected |
|------|-------|---------|----------|
| Link | `"[text](url)"` | false | 1 span: Link=true, Fg=LinkBlue |
| Link with bold | `"[**bold**](url)"` | false | 1 span: Link=true, Bold=true |
| Image | `"![alt](img.png)"` | false | 1 span: Image=true, placeholder text |
| Link, NoLinks mode | `"[text](url)"` | true | plain text: "[text](url)" |
| Image, NoLinks mode | `"![alt](url)"` | true | starts with "!" as plain text |

### Category C: List Style Preservation

| Test | Input | baseStyle | Expected |
|------|-------|-----------|----------|
| Bold in list | `"**bold**"` | `{ListItem: true, ListIndent: 1}` | Bold=true, ListItem=true, ListIndent=1 |
| Code in list | `` "`code`" `` | `{ListItem: true, ListOrdered: true, ListNumber: 3}` | Code=true, ListItem=true, ListOrdered=true, ListNumber=3 |
| Link in list | `"[text](url)"` | `{ListItem: true}` | Link=true, ListItem=true |

### Category D: Source Map Generation

| Test | Input | Expected Entries |
|------|-------|-----------------|
| Plain text | `"abc"` | 3 entries, 1:1 mapping, 1 rune each |
| Bold | `"**b**"` | 1 entry: rendered [0,1), source [0,5) |
| Code | `` "`x`" `` | 1 entry: rendered [0,1), source [0,3) |
| Mixed | `"a **b** c"` | entries for: "a ", "b", " c" |
| Link | `"[t](u)"` | entry for "t" with source within link brackets |

### Category E: Equivalence with Existing Functions

For each of the 6 existing functions, verify that `parseInline` with the corresponding `InlineOpts` produces identical `[]rich.Span` output. This is the primary regression test.

### Category F: Edge Cases

| Test | Input | Notes |
|------|-------|-------|
| Empty string | `""` | no spans, no entries |
| Only markers | `"****"` | treated as empty bold (existing behavior) |
| Nested markers | `"**a *b* c**"` | bold with nested italic — existing parser doesn't handle true nesting, verify parity |
| Multi-byte runes | `"**über**"` | source map entry byte/rune handling |

---

## Open Question Resolution

PLAN.md Open Question #1 asked about the approach: callback/visitor (A), options struct (B), or always-produce-all (C).

**Decision: (B) Options struct with optional output slices.**

Rationale:
- The nil check is trivial and avoids wasted allocations on the `Parse` path (which doesn't need source map entries).
- An options struct is idiomatic Go and easy to extend for future needs (e.g., adding escaped character tracking).
- The callback/visitor pattern would add complexity without benefit since the outputs are simple slice appends.
- Using pointers to slices (`*[]SourceMapEntry`) makes the "optional" semantics clean: nil means "don't produce", non-nil means "append here".
