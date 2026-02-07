# Incremental Preview Update Design

## Problem

`UpdatePreview()` re-parses the entire markdown source and re-renders the
full content on every change. For a large document this is expensive:

1. **Full re-parse**: `ParseWithSourceMap()` scans every line, builds every
   span, and constructs the complete source map and link map from scratch.
2. **Full content replacement**: `richBody.SetContent(content)` replaces the
   entire `Content` slice.
3. **Full layout + render**: `richBody.Render(body.all)` triggers
   `layoutBoxes()` which converts all spans to boxes, wraps them, and
   positions them — O(n) in document size.

The debounce timer (3 seconds) hides the cost during rapid typing, but the
user still experiences a visible pause when the timer fires.

## Goals

- Reduce the work done per update to O(changed) instead of O(document).
- Maintain exact equivalence with full re-parse (correctness first).
- Fall back to full re-parse when the incremental path cannot determine the
  affected region (safety net).
- Keep the design simple — target block-level granularity, not character-level.

## Non-Goals

- Incremental layout within `rich.Frame` (Phase 5 only addresses the parse
  and content-stitching layer; the frame still re-layouts on SetContent).
- Sub-block incremental updates (e.g., updating a single word within a
  paragraph without re-parsing the paragraph).
- Reducing the debounce delay (that can happen after incremental updates
  prove correct and fast).

---

## Architecture Overview

```
  Edit event (Insert or Delete at position q0, length n)
        │
        ▼
  ChangeDetector.AffectedRange(lines, q0, n)
        │
        ├──▶ blockStart  (source line index)
        └──▶ blockEnd    (source line index, exclusive)
                │
                ▼
  ParseWithSourceMap(blockText)  — parse only the affected block(s)
                │
                ├──▶ newContent   (rich.Content for the changed region)
                ├──▶ newSourceMap (entries for the changed region)
                └──▶ newLinkMap   (entries for the changed region)
                        │
                        ▼
  StitchContent(oldContent, newContent, blockStart, blockEnd)
                │
                ├──▶ merged Content
                ├──▶ merged SourceMap  (shifted entries)
                └──▶ merged LinkMap    (shifted entries)
                        │
                        ▼
  richBody.SetContent(merged)
  richBody.Render(body.all)
```

---

## Design Decisions

### Granularity: Block-Level

The parser is line-oriented and processes blocks (paragraphs, code blocks,
lists, tables, headings, horizontal rules) as units. Block boundaries are
well-defined:

- **Paragraph**: consecutive non-blank, non-block lines, terminated by a
  blank line or a block element.
- **Fenced code block**: `` ``` `` to `` ``` `` (inclusive).
- **Indented code block**: consecutive 4-space/tab-indented lines.
- **Table**: header row + separator row + data rows (consecutive pipe rows).
- **Heading**: single `# ` line.
- **Horizontal rule**: single `---`/`***`/`___` line.
- **List item**: `- `/`* `/`+ `/`1. ` line (may span continuation lines in
  future phases).
- **Blank line**: paragraph separator — not a block itself but a boundary
  marker.

Block-level is the right granularity because:
1. It matches the parser's natural unit of work.
2. Most edits affect a single block (typing in a paragraph, editing a code
   block).
3. Block boundaries are easy to detect from line content alone.
4. Cross-block edits (deleting a blank line to merge paragraphs) are handled
   by expanding the affected range to include both adjacent blocks.

### Edit Information Available

The `Text.Inserted()` and `Text.Deleted()` callbacks already receive:
- `q0`: the rune position of the edit in the source buffer.
- `nr` (insert) or `q1` (delete): the extent of the edit.

These are currently used only for event logging. The incremental system will
use them to identify the affected source region.

### Fallback Strategy

If the affected range cannot be determined (e.g., the edit touches a fenced
code block delimiter, which can change the parsing of everything after it),
fall back to full re-parse. The full re-parse path is already correct and
tested. The incremental path is an optimization that must produce identical
results.

---

## Component Design

### 1. EditRecord

A lightweight struct capturing what changed:

```go
// EditRecord describes a single edit operation on the source buffer.
type EditRecord struct {
    Pos    int  // rune position in source where edit occurred
    OldLen int  // runes removed (0 for pure insert)
    NewLen int  // runes inserted (0 for pure delete)
}
```

`Window` accumulates `EditRecord`s between preview updates. When the
debounce timer fires, it passes the accumulated edits to the incremental
update path. Multiple edits between timer firings are coalesced into a
single affected range.

### 2. BlockIndex

A structure built during parsing that records where each block starts and
ends in the source, enabling efficient lookup of which block(s) an edit
falls within.

```go
// BlockInfo records the source extent of a parsed block.
type BlockInfo struct {
    SourceLineStart int  // first line index (0-based) in splitLines output
    SourceLineEnd   int  // last line index (exclusive)
    SourceRuneStart int  // first rune position in source
    SourceRuneEnd   int  // last rune position in source (exclusive)
    ContentStart    int  // index into Content slice where this block's spans begin
    ContentEnd      int  // index into Content slice where this block's spans end (exclusive)
    SMStart         int  // index into SourceMap.entries for this block
    SMEnd           int  // index into SourceMap.entries for this block (exclusive)
    LMStart         int  // index into LinkMap.entries for this block
    LMEnd           int  // index into LinkMap.entries for this block (exclusive)
    Type            BlockType
}

type BlockType int
const (
    BlockParagraph BlockType = iota
    BlockFencedCode
    BlockIndentedCode
    BlockHeading
    BlockHRule
    BlockTable
    BlockListItem
    BlockBlankLine
)

// BlockIndex maps source positions to block extents.
type BlockIndex struct {
    Blocks []BlockInfo
}
```

`ParseWithSourceMap` will be extended to also return a `*BlockIndex`. This
is built naturally during the existing line-by-line parse loop by recording
the start/end of each block as the parser transitions between block types.

### 3. AffectedRange

Given accumulated `EditRecord`s and the previous `BlockIndex`, determine
which blocks need re-parsing:

```go
// AffectedRange returns the range of blocks that must be re-parsed
// given the edit. Returns (startBlock, endBlock) indices into
// BlockIndex.Blocks, or (-1, -1) if a full re-parse is needed.
func (bi *BlockIndex) AffectedRange(edits []EditRecord) (int, int)
```

Algorithm:
1. Coalesce edits into a single source rune range `[editStart, editEnd)`.
2. Binary search `BlockIndex.Blocks` to find blocks overlapping `[editStart,
   editEnd)`.
3. Expand by one block in each direction to handle boundary effects (e.g.,
   deleting the blank line between two paragraphs merges them).
4. **Fence check**: if any affected block is a fenced code block, check
   whether the edit could have added or removed a fence delimiter (`` ``` ``).
   If so, return `(-1, -1)` to trigger full re-parse — a fence change can
   alter the parsing of the entire remainder of the document.
5. **Table check**: similarly, if a table separator row was added or removed,
   the table block boundaries may have changed. Expand to include the full
   table.
6. Return `(startBlock, endBlock)` — the range of blocks to re-parse.

### 4. Partial Re-parse

Extract the source lines for the affected block range, parse them, and
produce a new `Content` slice, `SourceMap`, and `LinkMap` for just that
region:

```go
// ParseRegion parses a contiguous range of source lines and returns
// the content, source map, and link map for that region.
// sourceOffset is the rune position of the first line in the full source.
func ParseRegion(lines []string, sourceOffset int) (rich.Content, *SourceMap, *LinkMap)
```

This reuses the existing `ParseWithSourceMap` logic but operates on a
subset of lines. The `sourceOffset` parameter shifts source map entries so
they reference positions in the full source document, not the substring.

Implementation note: `ParseRegion` is essentially `ParseWithSourceMap` with
an initial `sourcePos = sourceOffset` and `renderedPos` parameter. Rather
than duplicating the parser, we can refactor `ParseWithSourceMap` to accept
optional start offsets (defaulting to 0 for full-document parse).

### 5. Content Stitching

Replace the affected region in the old content with the newly parsed
content, and shift all positions in the unaffected suffix:

```go
// StitchResult holds the merged output of an incremental update.
type StitchResult struct {
    Content  rich.Content
    SM       *SourceMap
    LM       *LinkMap
    BlockIdx *BlockIndex
}

// Stitch merges newly parsed content for blocks [startBlock, endBlock)
// into the existing parse result, adjusting positions in the suffix.
func Stitch(
    old       StitchResult,
    newRegion StitchResult,  // parsed from affected lines
    startBlock, endBlock int,
    sourceDelta int,         // change in source rune count (newLen - oldLen)
    renderedDelta int,       // change in rendered rune count
) StitchResult
```

Algorithm:
1. **Prefix**: copy `old.Content[:prefixContentEnd]`, `old.SM.entries[:prefixSMEnd]`,
   `old.LM.entries[:prefixLMEnd]`, `old.BlockIdx.Blocks[:startBlock]` unchanged.
2. **Middle**: append `newRegion.Content`, `newRegion.SM.entries`,
   `newRegion.LM.entries`, `newRegion.BlockIdx.Blocks`.
3. **Suffix**: copy `old.Content[suffixContentStart:]` etc., shifting all
   source rune positions by `sourceDelta` and all rendered positions by
   `renderedDelta`.
4. Update `BlockIndex` entries in the suffix: shift `SourceRuneStart/End` by
   `sourceDelta`, `ContentStart/End` by content-index delta,
   `RenderedStart/End` implicitly via source map shifts.

### 6. Updated UpdatePreview

```go
func (w *Window) UpdatePreview() {
    if !w.previewMode || w.richBody == nil {
        return
    }

    currentOrigin := w.richBody.Origin()
    bodyContent := w.body.file.String()

    var content rich.Content
    var sourceMap *SourceMap
    var linkMap *LinkMap
    var blockIdx *BlockIndex

    // Try incremental path
    if w.prevBlockIndex != nil && len(w.pendingEdits) > 0 {
        startBlock, endBlock := w.prevBlockIndex.AffectedRange(w.pendingEdits)
        if startBlock >= 0 {
            // Incremental: re-parse only affected blocks
            lines := splitLines(bodyContent)
            // ... extract affected lines, call ParseRegion, Stitch ...
        }
    }

    if content == nil {
        // Full re-parse fallback
        content, sourceMap, linkMap, blockIdx = ParseWithSourceMapAndIndex(bodyContent)
    }

    w.pendingEdits = w.pendingEdits[:0]  // clear edit log

    w.richBody.SetContent(content)
    w.previewSourceMap = sourceMap
    w.previewLinkMap = linkMap
    w.prevBlockIndex = blockIdx

    // Restore scroll position
    newLen := content.Len()
    if currentOrigin > newLen {
        currentOrigin = newLen
    }
    w.richBody.SetOrigin(currentOrigin)
    w.richBody.Render(w.body.all)
    if w.display != nil {
        w.display.Flush()
    }
}
```

### 7. Edit Accumulation

In `Text.Inserted()` and `Text.Deleted()`, in addition to calling
`SchedulePreviewUpdate()`, record the edit:

```go
// In Inserted():
w.pendingEdits = append(w.pendingEdits, EditRecord{
    Pos:    q0,
    OldLen: 0,
    NewLen: nr,
})

// In Deleted():
w.pendingEdits = append(w.pendingEdits, EditRecord{
    Pos:    q0,
    OldLen: q1 - q0,
    NewLen: 0,
})
```

---

## Fence Delimiter Safety

The most dangerous case for incremental parsing is when an edit adds or
removes a fenced code block delimiter (`` ``` ``). A single `` ``` `` can
change the interpretation of everything after it (code becomes text, text
becomes code). The design handles this by:

1. During `AffectedRange`, if any affected block is `BlockFencedCode`,
   scan the edit text for `` ``` ``. If found (or if the edit straddles a
   fence line), return `(-1, -1)` → full re-parse.
2. Similarly, if the edit is in a non-code block but the inserted/deleted
   text contains `` ``` ``, return `(-1, -1)`.
3. This is conservative but safe. Fence edits are relatively rare (users
   aren't constantly adding/removing code fences), so the full re-parse
   fallback is acceptable for these cases.

---

## Correctness Verification

The incremental path must produce identical output to full re-parse. The
test strategy (Phase 5.2–5.4) will verify this by:

1. For each test case, perform the edit incrementally AND do a full re-parse.
2. Compare `Content`, `SourceMap`, and `LinkMap` field-by-field.
3. Any mismatch is a bug in the incremental path.

This is a property-based test: for any document and any edit, `Stitch(old,
ParseRegion(affected)) == ParseWithSourceMap(newDocument)`.

---

## Minimum Viable Slice

For the initial implementation (Phase 5.2–5.3), focus on:

1. **BlockIndex generation** during `ParseWithSourceMap` — record block
   boundaries as a side effect of the existing parse loop.
2. **AffectedRange** — binary search + one-block expansion + fence check.
3. **ParseRegion** — `ParseWithSourceMap` with offset parameters.
4. **Stitch** — splice Content, SourceMap, LinkMap, BlockIndex with position
   shifts.
5. **Edit accumulation** — record edits in `Window.pendingEdits`.

The frame/layout layer (`rich.Frame.SetContent`, `Render`) is untouched.
The win is eliminating the full parse — even though we still do a full
layout, the parse is often the more expensive step for large documents with
tables, source maps, and link maps.

---

## Window State Additions

```go
// New fields on Window:
prevBlockIndex *markdown.BlockIndex  // block boundaries from last parse
pendingEdits   []markdown.EditRecord // edits since last UpdatePreview
```

---

## API Changes Summary

### New types in `markdown/`

| Type | Purpose |
|------|---------|
| `EditRecord` | Describes a single insert or delete |
| `BlockType` | Enum for block kinds |
| `BlockInfo` | Source/content/sourcemap extent of one block |
| `BlockIndex` | Ordered slice of `BlockInfo` with lookup methods |

### New functions in `markdown/`

| Function | Purpose |
|----------|---------|
| `ParseWithSourceMapAndIndex(text) → (Content, *SourceMap, *LinkMap, *BlockIndex)` | Extended parse returning block index |
| `(*BlockIndex).AffectedRange(edits) → (int, int)` | Find blocks needing re-parse |
| `ParseRegion(lines, sourceOffset, renderedOffset) → (Content, *SourceMap, *LinkMap, *BlockIndex)` | Parse a contiguous subset of lines |
| `Stitch(old, new, start, end, srcDelta, renDelta) → StitchResult` | Merge incremental parse into full result |

### Modified functions

| Function | Change |
|----------|--------|
| `ParseWithSourceMap` | Refactored to call internal `parseLines()` with offset params; `ParseWithSourceMap` is now a thin wrapper passing offset=0 |
| `UpdatePreview` | Try incremental path, fall back to full re-parse |
| `Text.Inserted` | Record `EditRecord` on window |
| `Text.Deleted` | Record `EditRecord` on window |

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Incremental result differs from full parse | Property-based equivalence tests (Phase 5.2–5.4) |
| Fence delimiter edit breaks everything after | Conservative full-re-parse fallback for fence edits |
| Edit coalescing loses information | Coalesce into range, not individual positions; expand affected range by ±1 block |
| BlockIndex becomes stale after multiple incremental updates | BlockIndex is rebuilt on each update (incremental Stitch produces a new BlockIndex) |
| Paragraph continuation across blocks | Expand affected range to include adjacent paragraph blocks |
| Source map position drift after many incremental updates | Each Stitch shifts suffix positions by exact delta; verified by equivalence tests |
