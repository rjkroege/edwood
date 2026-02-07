# Markdown Preview Improvement Plan

Improve the Edwood markdown preview system: fix correctness bugs, eliminate code duplication, improve performance, and add missing features.

For Rich Text Implementation (Phases 1-24), see `PLAN_ARCHIVE.md`.
For Codebase Refactoring (Phases 1-10), see `PLAN_ARCHIVE.md` (to be merged).

**Base design doc**: `docs/markdown-design.md`

## Status Legend
- `[ ]` = not done
- `[x]` = done

---

## Current Task

**Phase 9**: Async Image Loading

---

## Phase 1: Source Mapping Correctness

Selection in preview mode frequently produces misaligned source positions. An audit of `sourcemap.go` and the `wind.go` selection integration reveals several concrete bugs and a near-total absence of tests for the cases most likely to fail. This phase adds comprehensive tests first, then fixes the identified bugs.

**Known bugs (from code audit):**

1. **Byte/rune confusion** (`sourcemap.go:86`): `PrefixLen` is a byte length but is added to `SourceRuneStart` (a rune position). Breaks heading selections with non-ASCII content (e.g., `# Über`).
2. **ToSource/ToRendered asymmetry** (`sourcemap.go:107-151`): `ToRendered()` snaps to entry boundaries on certain edge cases but `ToSource()` does not. Round-trip `rendered→source→rendered` does not produce the original positions.
3. **Point selection normalization masking** (`sourcemap.go:90-97`): When `renderedStart == renderedEnd`, forces `srcStart = srcEnd`. This masks the byte/rune bug for cursor clicks but not for range selections.
4. **Entry boundary lookup** (`sourcemap.go:64-74`): When `renderedEnd` falls exactly on an entry boundary, `lookupPos = renderedEnd - 1` finds the previous entry. Produces wrong results if entries are non-contiguous or at paragraph/block transitions.
5. **Missing bounds validation** (`wind.go`): `syncSourceSelection()` doesn't validate source positions against buffer length before use. Silent failures after editing.

**Testing gaps (from test audit):**

The existing `sourcemap_test.go` (1,060 lines, 15 test functions) covers individual element types but has zero coverage for:
- Selections spanning multiple formatting spans (e.g., bold → plain)
- Selections crossing block boundaries (heading → paragraph)
- Selections at exact entry boundary positions
- Point selections (clicks) in headings with non-ASCII content
- Round-trip consistency: `ToSource(ToRendered(x)) ≈ x` and `ToRendered(ToSource(x)) ≈ x`
- Selections after source edits that shift positions
- Empty/degenerate selections at document start/end

### 1.1 Source Mapping Audit Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Audit source mapping and design fixes for all identified bugs | `docs/markdown-design.md`, `markdown/sourcemap.go`, `wind.go` (syncSourceSelection, PreviewSnarf, PreviewLookText, PreviewExecText) | Output: `docs/designs/features/sourcemap-correctness.md`. Must document: each bug with line numbers and reproduction case, the fix for each, the full set of test categories needed, how `PrefixLen` should work (bytes vs runes — pick one and be consistent), the round-trip invariant that ToSource/ToRendered must satisfy, how boundary cases should behave (beginning/end of entry, beginning/end of document, non-contiguous entries). |

### 1.2 Source Map Round-Trip and Boundary Tests

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write comprehensive source map correctness tests | `docs/designs/features/sourcemap-correctness.md` | Test file: `markdown/sourcemap_correctness_test.go`. Categories: **(A) Round-trip**: for a variety of documents, verify `ToRendered(ToSource(r0,r1))` produces positions that cover the same text, and vice versa. **(B) Cross-boundary**: select from bold into plain text, from heading into paragraph, from list item into next list item, from code block into paragraph. **(C) Exact boundary**: renderedEnd exactly at entry.RenderedEnd, renderedStart exactly at entry.RenderedStart. **(D) Point selections**: single-rune clicks in headings (ASCII and non-ASCII), in bold markers, at paragraph breaks. **(E) Non-ASCII**: headings, bold, links containing multi-byte runes. **(F) Edge positions**: position 0, position at document end, empty document. Document each test's assumption and which bug it targets. |
| [x] Iterate | Red/green/review — expect many tests to fail initially | `docs/designs/features/sourcemap-correctness.md` | Fix bugs in `markdown/sourcemap.go`: fix PrefixLen byte/rune confusion, fix ToRendered boundary snapping to be symmetric with ToSource, fix entry boundary lookup for exact-match positions. Do NOT change `parse.go` or `sourcemap.go` entry generation yet — that's Phase 2. |
| [x] Commit | Commit source map correctness tests and fixes | — | Message: `Fix source map byte/rune confusion, boundary lookup, and round-trip asymmetry` |

### 1.3 Selection Integration Tests

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for wind.go selection sync paths | `docs/designs/features/sourcemap-correctness.md` | Test file: extend `wind_test.go` or new `wind_selection_test.go`. Test `syncSourceSelection()` with: valid range, out-of-bounds range (after edit), point selection in heading, range crossing formatted text, selection of entire code block via double-click. Test `PreviewSnarf()` extracts correct source markdown for: plain text, bold text, heading, mixed formatting. Test that bounds are validated before buffer read. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/sourcemap-correctness.md` | Add bounds validation in `syncSourceSelection()`, `PreviewSnarf()`, `PreviewLookText()`, `PreviewExecText()`. Clamp source positions to `[0, body.file.Nr()]`. |
| [x] Commit | Commit selection integration fixes | — | Message: `Add bounds validation to preview selection sync and snarf` |

### 1.4 Markup-Boundary Selection Heuristics

The design doc specifies: "If q0 is at the start of a markup operation (ie, first text after bold, italic, code block, image, table, etc) the source q0 should include the markup; likewise if on the last character, the trailing markup should be included."

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for markup-boundary selection expansion | `docs/designs/features/sourcemap-correctness.md` | Test file: extend `markdown/sourcemap_correctness_test.go`. Test: selecting all of "bold" in `**bold**` should map to source range including the `**` delimiters. Selecting partial "bol" should NOT include delimiters. Same for `*italic*`, `` `code` ``, `[link](url)`, `# heading`. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/sourcemap-correctness.md` | Modify `ToSource()` to expand to include markup delimiters when selection start/end align with entry boundaries. This is the "heuristics for choosing where the source map points when q0 != q1" described in the design doc. |
| [x] Commit | Commit markup-boundary selection heuristics | — | Message: `Expand source selection to include markup delimiters at boundaries` |

---

## Phase 2: Inline Parser Unification

The markdown parser has the most critical code quality issue in the preview system: duplicated inline parsing logic across `parse.go` and `sourcemap.go` that has already diverged, causing bugs where fixes to one copy are missed in the other.

There are **6 parallel functions** that implement the same inline formatting logic:

In `parse.go`:
- `parseInlineFormatting()` — standard inline parsing
- `parseInlineFormattingNoLinks()` — inline parsing without link recognition
- `parseInlineFormattingWithListStyle()` — preserves list style fields
- `parseInlineFormattingWithListStyleNoLinks()` — list style, no links

In `sourcemap.go`:
- `parseInlineWithSourceMap()` — inline parsing + source mapping + link mapping
- `parseInlineWithSourceMapNoLinks()` — source-mapped inline parsing without links

The existing TODO at `sourcemap.go:199` explicitly calls this out. This phase unifies them into a single implementation that optionally produces source map entries.

### 2.1 Unified Inline Parser Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Distill unified inline parser design from base doc and existing code | `docs/markdown-design.md`, `markdown/parse.go`, `markdown/sourcemap.go` | Output: `docs/designs/features/unified-inline-parser.md`. Must cover: single function signature with optional source map callback, handling of bold/italic/code/links/images, list-style preservation via base style, link vs no-link mode via flag, span merging. Document all 6 existing functions, their differences, and the unified approach. Include test cases. |

### 2.2 Unified Inline Parser Tests

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for unified `parseInline()` function | `docs/designs/features/unified-inline-parser.md` | Test file: `markdown/inline_test.go`. Cover: bold, italic, bold+italic, inline code, links, images, nested formatting, escaped characters, list-style preservation, with and without source map, with and without link parsing. Reuse existing test cases from `parse_test.go` and `sourcemap_test.go` that exercise inline parsing. Document assumptions. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/unified-inline-parser.md` | New file: `markdown/inline.go`. Single `parseInline()` function with options struct or functional options for source map and link tracking. Flag any O(n²) or worse. |
| [x] Commit | Commit unified inline parser | — | Message: `Add unified inline parser with optional source mapping` |

### 2.3 Migrate parse.go Callers

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Verify existing `parse_test.go` tests still pass after migration | `docs/designs/features/unified-inline-parser.md` | No new test file — existing 4,375-line test suite is the verification. Run `go test ./markdown/...` |
| [x] Iterate | Replace 4 `parseInlineFormatting*` functions in `parse.go` with calls to unified `parseInline()` | `docs/designs/features/unified-inline-parser.md` | Delete: `parseInlineFormatting`, `parseInlineFormattingNoLinks`, `parseInlineFormattingWithListStyle`, `parseInlineFormattingWithListStyleNoLinks`. All existing tests must pass unchanged. |
| [x] Commit | Commit parse.go migration | — | Message: `Migrate parse.go to unified inline parser` |

### 2.4 Migrate sourcemap.go Callers

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Verify existing `sourcemap_test.go` tests still pass after migration | `docs/designs/features/unified-inline-parser.md` | No new test file — existing 1,060-line test suite is the verification. |
| [x] Iterate | Replace 2 `parseInlineWithSourceMap*` functions in `sourcemap.go` with calls to unified `parseInline()` | `docs/designs/features/unified-inline-parser.md` | Delete: `parseInlineWithSourceMap`, `parseInlineWithSourceMapNoLinks`. All existing tests must pass unchanged. Remove the TODO at line 199. |
| [x] Commit | Commit sourcemap.go migration | — | Message: `Migrate sourcemap.go to unified inline parser, remove duplication TODO` |

---

## Phase 3: Debounce Timer Thread Safety

The design doc (Known Issue #2) identifies that `SchedulePreviewUpdate()` uses `time.AfterFunc` which fires on a separate goroutine, but `UpdatePreview()` must be called from the main goroutine. The current implementation calls it directly from the timer callback, creating a data race.

### 3.1 Thread-Safe Preview Update Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Design thread-safe debounce mechanism | `docs/markdown-design.md`, `wind.go:1520-1542` | Output: `docs/designs/features/safe-preview-debounce.md`. Must cover: how to marshal the UpdatePreview call to the main goroutine (channel-based dispatch or display.QueueFunc or similar), interaction with existing event loop, timer cancellation semantics, what happens if window is closed while timer is pending. |

### 3.2 Thread-Safe Debounce Implementation

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write test for debounce race condition | `docs/designs/features/safe-preview-debounce.md` | Test file: `wind_preview_test.go` or extend `wind_test.go`. Test with `-race` flag. Document assumptions about goroutine scheduling. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/safe-preview-debounce.md` | Modify `SchedulePreviewUpdate()` in `wind.go`. Must pass `go test -race ./...` |
| [x] Commit | Commit thread-safe debounce | — | Message: `Fix preview update debounce to run on main goroutine` |

---

## Phase 4: wind/preview.go Type Completion

The `wind/preview.go` PreviewState struct uses `interface{}` stubs for `sourceMap`, `linkMap`, and `imageCache`. These need proper types. This is blocked by circular dependency concerns (wind/ importing markdown/ and rich/).

### 4.1 PreviewState Typing Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Design proper typing for PreviewState | `docs/markdown-design.md`, `wind/preview.go`, `markdown/sourcemap.go`, `markdown/linkmap.go`, `rich/image.go` | Output: `docs/designs/features/preview-state-typing.md`. Options: (A) wind/ imports markdown/ and rich/ directly, (B) define interfaces in wind/ that markdown/rich types satisfy, (C) use a shared types package. Evaluate circular dependency risk for each. The wind/ package currently has no dependency on markdown/ or rich/. |

### 4.2 PreviewState Type Implementation

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Update `wind/preview_test.go` to use typed fields | `docs/designs/features/preview-state-typing.md` | Existing tests use `interface{}` — convert to proper types. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/preview-state-typing.md` | Update `wind/preview.go`. Replace `interface{}` with proper types. |
| [x] Commit | Commit PreviewState typing | — | Message: `Replace interface{} stubs in PreviewState with proper types` |

---

## Phase 5: Incremental Preview Updates

The design doc explicitly states: "We will replace the re-render version we have now with an incremental, much more efficient version in the near future." Currently `UpdatePreview()` re-parses the entire markdown source on every change. This is the primary performance bottleneck for the live editing experience.

### 5.1 Incremental Update Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Design incremental preview update system | `docs/markdown-design.md`, `wind.go` (UpdatePreview), `markdown/parse.go`, `markdown/sourcemap.go` | Output: `docs/designs/features/incremental-preview.md`. Key questions: What granularity of change tracking? (line-level, paragraph-level, block-level). How to detect which blocks changed? (diff, dirty flags, edit position). How to incrementally update source map entries? Can rich.Frame accept partial content updates? What's the minimum viable slice: re-parse only the changed block and stitch into existing Content? |

### 5.2 Change Detection

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for change detection in markdown source | `docs/designs/features/incremental-preview.md` | Test file: `markdown/change_test.go` or `markdown/incremental_test.go`. Test: single-line edit within a paragraph, adding/removing a line in a code block, adding a heading, deleting a list item. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/incremental-preview.md` | New file: `markdown/change.go` or similar. Must identify affected block range given an edit position and length. |
| [x] Commit | Commit change detection | — | Message: `Add markdown change detection for incremental preview` |

### 5.3 Partial Re-parse

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for partial re-parse and content stitching | `docs/designs/features/incremental-preview.md` | Test file: extend `markdown/change_test.go`. Verify that partially re-parsed content matches full re-parse output. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/incremental-preview.md` | Implement block-level re-parse that produces a Content slice replaceable in the existing Content array. Update source map entries for shifted positions. |
| [x] Commit | Commit partial re-parse | — | Message: `Add partial markdown re-parse for changed blocks` |

### 5.4 Integrate Incremental Updates

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write integration tests for incremental UpdatePreview | `docs/designs/features/incremental-preview.md` | Extend `wind_test.go`. Verify: edit in paragraph, edit in code block, delete heading, add list item — all produce same rendered output as full re-parse. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/incremental-preview.md` | Modify `UpdatePreview()` in `wind.go` to use incremental path when edit position is known, falling back to full re-parse otherwise. |
| [x] Commit | Commit incremental preview integration | — | Message: `Integrate incremental preview updates into UpdatePreview` |

---

## Phase 6: Scrollable Block Gutter

The design doc specifies: "All scrollable objects must be indented by at least 8ems of space to leave a clean, non-scrolling gutter on the left of the page." This ensures code blocks, tables, and images don't overlap the vertical scrollbar gutter area. Verify and fix if not fully implemented.

### 6.1 Block Gutter Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Audit current gutter implementation and design fixes | `docs/markdown-design.md`, `rich/layout.go`, `rich/frame.go` | Output: `docs/designs/features/block-gutter.md`. Check if 8em left indent exists for code blocks, tables, images. If partially present, document what's missing. Define "8ems" in pixels relative to base font. |

### 6.2 Block Gutter Implementation

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write/extend tests for block gutter spacing | `docs/designs/features/block-gutter.md` | Test file: extend `rich/layout_test.go`. Verify code blocks, tables, and images have left gutter >= 8em. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/block-gutter.md` | Modify `rich/layout.go` as needed. |
| [x] Commit | Commit block gutter | — | Message: `Ensure 8em left gutter for scrollable block regions` |

---

## Phase 7: Blockquote Support

The design doc notes "Blockquotes are not supported" as a known limitation. This adds basic blockquote rendering (`> ` prefix).

### 7.1 Blockquote Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Design blockquote parsing and rendering | `docs/markdown-design.md`, `markdown/parse.go`, `rich/style.go`, `rich/layout.go` | Output: `docs/designs/features/blockquotes.md`. Cover: syntax (`> `, `>> ` nesting), style fields needed (Blockquote bool, BlockquoteLevel int), rendering (left border bar, indentation, optional background tint), interaction with inline formatting, source map behavior. |

### 7.2 Blockquote Style and Parsing

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for blockquote parsing | `docs/designs/features/blockquotes.md` | Test file: extend `markdown/parse_test.go`. Cover: single-line, multi-line, nested, with inline formatting, with paragraph breaks, empty blockquote, blockquote followed by other blocks. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/blockquotes.md` | Add `Blockquote` and `BlockquoteLevel` to `rich.Style`. Add blockquote detection to `parse.go` block-level dispatch. Use unified inline parser from Phase 1. |
| [x] Commit | Commit blockquote parsing | — | Message: `Add blockquote parsing support` |

### 7.3 Blockquote Rendering

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for blockquote layout and rendering | `docs/designs/features/blockquotes.md` | Test file: extend `rich/layout_test.go`. Verify indentation per nesting level, left border placement, content wrapping within blockquote width. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/blockquotes.md` | Modify `rich/layout.go` for blockquote indentation and `rich/frame.go` for border rendering. |
| [x] Commit | Commit blockquote rendering | — | Message: `Add blockquote rendering with left border and indentation` |

### 7.4 Blockquote Source Mapping

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for blockquote source mapping | `docs/designs/features/blockquotes.md` | Test file: extend `markdown/sourcemap_test.go`. Verify ToSource/ToRendered correctly handle `> ` prefix stripping. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/blockquotes.md` | Update `sourcemap.go` to handle blockquote prefix similar to heading prefix handling. |
| [x] Commit | Commit blockquote source mapping | — | Message: `Add source mapping for blockquotes` |

---

## Phase 8: Nested Block Elements in Lists

The design doc notes "Lists cannot contain code blocks or tables" as a known limitation. This adds support for code blocks and blockquotes nested inside list items.

### 8.1 Nested Blocks Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Design nested block parsing within list items | `docs/markdown-design.md`, `markdown/parse.go` | Output: `docs/designs/features/nested-list-blocks.md`. Key challenge: the current parser is single-pass line-oriented. Nested blocks require tracking list context across multiple lines. Cover: code blocks in list items (indented by list indent + 4 spaces or fenced within list), blockquotes in list items, continuation semantics (when does a list item end?), how nesting interacts with source mapping. Consider: is a multi-pass approach needed, or can context stack handle this? |

### 8.2 Nested Code Blocks in Lists

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for code blocks nested in list items | `docs/designs/features/nested-list-blocks.md` | Test file: extend `markdown/parse_test.go`. Cover: fenced code block inside list item, indented code inside list item, code block ending list item vs continuing, nested list with code block. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/nested-list-blocks.md` | Modify block-level dispatch in `parse.go` to track list context. |
| [x] Commit | Commit nested code blocks in lists | — | Message: `Support code blocks nested within list items` |

### 8.3 Nested Blockquotes in Lists

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for blockquotes nested in list items | `docs/designs/features/nested-list-blocks.md` | Test file: extend `markdown/parse_test.go`. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/nested-list-blocks.md` | Depends on Phase 7 (blockquote support). |
| [x] Commit | Commit nested blockquotes in lists | — | Message: `Support blockquotes nested within list items` |

---

## Phase 9: Async Image Loading

The design doc notes "Image loading (including HTTP fetches) appears to happen synchronously during parsing/rendering" as a known issue. This moves image loading off the main thread.

### 9.1 Async Image Loading Design

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Design | Design async image loading with placeholder rendering | `docs/markdown-design.md`, `rich/image.go`, `rich/frame.go` | Output: `docs/designs/features/async-image-loading.md`. Cover: placeholder rendering during load (gray box with alt text), goroutine-per-image with result channel, callback to trigger re-render when image arrives, cancellation on preview exit, error display, interaction with ImageCache (cache hit = sync, cache miss = async). Max concurrent downloads. |

### 9.2 Image Loading Pipeline

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write tests for async image loading | `docs/designs/features/async-image-loading.md` | Test file: `rich/image_async_test.go` or extend `rich/image_test.go`. Test: cache hit returns immediately, cache miss triggers async load, placeholder shown during load, callback invoked on completion, cancellation prevents callback, error caching. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/async-image-loading.md` | Modify `rich/image.go` to add `LoadAsync()` method. Add placeholder image support to frame rendering. |
| [x] Commit | Commit async image loading | — | Message: `Add async image loading with placeholder rendering` |

### 9.3 Integrate Async Loading into Preview

| Stage | Description | Read | Notes |
|-------|-------------|------|-------|
| [x] Tests | Write integration tests for preview with async images | `docs/designs/features/async-image-loading.md` | Extend `wind_test.go`. Verify preview renders immediately with placeholders, then updates when images load. |
| [x] Iterate | Red/green/review until tests pass | `docs/designs/features/async-image-loading.md` | Wire async loading into `exec.go` preview initialization and `UpdatePreview()`. |
| [x] Commit | Commit preview async image integration | — | Message: `Integrate async image loading into preview mode` |

---

## Future Phases (Unscheduled)

These are documented in the design doc as known limitations. They can be planned in detail when prioritized.

### Syntax Highlighting
Code blocks are rendered in monospace with gray background but no language-specific coloring. Would require a tokenizer/lexer for common languages and color mapping to `rich.Style.Fg`.

### Proportional Table Layout
Tables currently use box-drawing characters and monospace column padding. A proportional layout engine would measure cell content with the actual font and distribute column widths accordingly.

---

## Open Questions

1. **Parser unification approach**: The unified inline parser needs to handle both "fire-and-forget" span generation (parse.go) and "span + source map entry + link entry" generation (sourcemap.go). Should this use:
   - (A) A callback/visitor pattern where the caller provides handlers?
   - (B) An options struct with optional output slices?
   - (C) Always produce all outputs, let callers ignore what they don't need?

   Tradeoff: (A) is most flexible but complex; (B) is clean but requires nil-checking; (C) is simplest but wastes allocations on the non-source-map path.

2. **Incremental update granularity**: The design doc says "Make sure all the update operations are separate from other semantics." What's the right block boundary for incremental re-parse? Options:
   - (A) Line-level: re-parse from edit line to next blank line. Simplest but may miss block boundaries.
   - (B) Block-level: identify the block (paragraph, code block, list, table) containing the edit and re-parse it entirely. Most natural for the parser.
   - (C) Section-level: re-parse from previous heading to next heading. Broadest but guaranteed correct.

3. **wind/ package circular dependency**: How should `wind/preview.go` reference markdown and rich types? The wind/ package currently has zero dependencies on markdown/ or rich/. Adding them may create coupling issues if the wind/ package is intended to be imported by markdown/ or rich/ in the future. Should we use interfaces in wind/ to avoid the import?

4. **Blockquote priority**: Blockquotes (Phase 7) are listed as a known unsupported element. How important is this relative to the incremental update performance work (Phase 5)? Should Phase 5 be done first since the design doc specifically calls it out as "near future"?

---

## Test Summary

| Suite | Target Coverage | Notes |
|-------|-----------------|-------|
| Source mapping | Round-trip, boundary, cross-span | Correctness regression suite |
| Selection integration | Sync, snarf, bounds | wind.go paths with source map |
| Inline parser | 100% formatting paths | Replaces 6 duplicated functions |
| Debounce | Race-free | `go test -race ./...` |
| Change detection | All block types | Paragraph, code, list, table, heading |
| Incremental parse | Equivalence with full parse | Property-based comparison |
| Blockquote | Block + inline + nesting | New element type |
| Async images | Cache hit/miss/error/cancel | Concurrency testing |

## How to Run Tests

```bash
# All tests with race detector
go test -race ./...

# Specific packages
go test ./markdown/...
go test ./rich/...
go test ./wind/...

# Verbose with race detector
go test -v -race ./...
```
