# Source Map Correctness Design

## Overview

The source map (`markdown/sourcemap.go`) provides bidirectional mapping between
rendered rune positions and source rune positions. It is the foundation of
preview selection, snarf, Look, and Execute operations. This document audits
all known bugs, specifies fixes, defines invariants, and enumerates the test
categories needed.

---

## Bug Inventory

### Bug 1: PrefixLen byte/rune confusion

**Location**: `sourcemap.go:86` (`ToSource`, end-position calculation)

**Code**:
```go
// Line 85-86 in ToSource():
offset := renderedEnd - endEntry.RenderedStart
srcEnd = endEntry.SourceRuneStart + endEntry.PrefixLen + offset
```

**Problem**: `PrefixLen` is set during entry creation in `parseLineWithSourceMap`
(line 598-612) as a byte count from the source line:

```go
prefixLen := level            // number of '#' characters (bytes)
if len(content) > 0 && content[0] == ' ' {
    prefixLen++               // include the space
}
```

For ASCII headings like `# Hello`, bytes == runes so this works. But for
headings with multi-byte characters in the prefix region (rare but possible if
the parser evolves) or when `PrefixLen` is used in arithmetic with rune
positions, the semantics are confused.

More critically, the current code adds `PrefixLen` (a byte count) to
`SourceRuneStart` (a rune position) at line 86. This produces incorrect results
when the source text preceding the entry contains multi-byte characters, because
`SourceRuneStart` is a rune offset and `PrefixLen` is a byte offset.

**Reproduction**: `# Über` (source: `# Über`, 8 bytes, 7 runes).
- Entry: `SourceRuneStart=0`, `SourceRuneEnd=7`, `PrefixLen=2` (bytes for `# `).
- Select rendered position 2 (the `e` in `Über`): offset=2.
- Current code: `srcEnd = 0 + 2 + 2 = 4` (rune 4 = `r`). Correct because `# `
  is ASCII so bytes==runes. But consider a scenario where source before the entry
  contains non-ASCII, shifting `SourceRuneStart` while `PrefixLen` remains in bytes.

**Actually**: In the current codebase, `PrefixLen` is always ASCII (`# `, `## `,
etc.) so the byte==rune equivalence holds. However, the semantic confusion is
real and will break if extended (e.g., for blockquote `> ` prefixes with
non-ASCII content following). The real issue is that *the code is mixing units*.

**Fix**: Make `PrefixLen` explicitly a **rune count**. Since all current prefixes
(`# `, `## `, etc.) are ASCII, the values don't change, but:
1. Rename `PrefixLen` to `PrefixRuneLen` in the struct and all usages.
2. Add a comment documenting that it is a rune count.
3. In `parseLineWithSourceMap`, compute `prefixLen` as `len([]rune(prefix))`
   instead of byte length. For current ASCII prefixes this is a no-op.

### Bug 2: ToSource/ToRendered round-trip asymmetry

**Location**: `sourcemap.go:31-100` (ToSource) vs `sourcemap.go:107-151` (ToRendered)

**Problem**: `ToRendered()` uses `sourceRuneToRendered()` which has distinct
logic for "within opening marker" → snap to `RenderedStart`, "within closing
marker" → snap to `RenderedEnd`. But `ToSource()` has a different mapping for
start vs end positions:

- `ToSource` start: if `renderedStart == RenderedStart`, use `SourceRuneStart` (includes full markup).
- `ToSource` end: adds `PrefixLen + offset` from `RenderedStart`.

These formulas are not inverses of each other. Specifically:

**Round-trip `rendered→source→rendered` asymmetry**: Select the full bold text
"bold" (rendered 0-4) in `**bold**`. `ToSource(0,4) = (0,8)`. Then
`ToRendered(0,8) = (0,4)`. This works. But for partial selections within
formatted entries, the end-position formula in `ToSource` adds `PrefixLen`
even for symmetric markers (bold, italic, code) where `PrefixLen` is 0.

**Where this actually breaks**: The asymmetry manifests when the start and end
of a selection land in different entries. `ToSource` uses different formulas for
the start entry and end entry: the start uses `SourceRuneStart + offset` while
the end uses `SourceRuneStart + PrefixLen + offset`. For a heading entry with
`PrefixLen=2`, if `renderedEnd` is in the middle of the entry, the end position
accounts for the prefix but the start position does not (unless at entry
boundary). This means a selection starting at rendered position 1 in `# Hello`
maps to source 3 (rune 1 + prefix 2) for the end but source 1 (rune 1 + 0)
for the start of a different calculation.

**Fix**: Unify the position calculation. Both `ToSource` start and end should
use the same formula for mapping a rendered position within an entry to a source
position:

```
srcPos = entry.SourceRuneStart + entry.PrefixRuneLen + (renderedPos - entry.RenderedStart)
```

This maps any rendered position to the corresponding source position past the
prefix. For the entry-boundary expansion heuristic (including markup delimiters),
apply it separately after the base position calculation:
- If `renderedStart == entry.RenderedStart`, set `srcStart = entry.SourceRuneStart`.
- If `renderedEnd == entry.RenderedEnd`, set `srcEnd = entry.SourceRuneEnd`.

### Bug 3: Point selection normalization masking

**Location**: `sourcemap.go:90-97`

**Code**:
```go
if renderedStart == renderedEnd && srcStart != srcEnd {
    srcStart = srcEnd
}
```

**Problem**: This normalization ensures a point selection (click) produces a
point in source. Without it, clicking at position 0 in rendered `# Hello` would
produce `srcStart=0, srcEnd=2` (the prefix length mismatch). The normalization
masks Bug 2 for point selections but not for range selections.

The normalization forces `srcStart = srcEnd`, choosing the end-position formula.
This is the correct behavior for point selections but it silently hides the
underlying formula asymmetry.

**Fix**: After fixing Bug 2 with a unified formula, the start and end formulas
will agree for point selections (same position → same result). The normalization
becomes a no-op and can be removed. Keep it as a safety assertion during testing:

```go
if renderedStart == renderedEnd && srcStart != srcEnd {
    // This should not happen with correct formulas. Log or panic in tests.
    srcStart = srcEnd
}
```

### Bug 4: Entry boundary lookup off-by-one

**Location**: `sourcemap.go:64-74`

**Code**:
```go
lookupPos := renderedEnd
if renderedEnd > renderedStart {
    lookupPos = renderedEnd - 1
}
for i := range sm.entries {
    e := &sm.entries[i]
    if lookupPos >= e.RenderedStart && lookupPos < e.RenderedEnd {
```

**Problem**: When `renderedEnd` falls exactly on an entry boundary (i.e.,
`renderedEnd == someEntry.RenderedEnd`), `lookupPos = renderedEnd - 1` finds
the entry containing the *last selected rune*. This is usually correct — it
means "the end of selection is just past this rune, so find the entry containing
the rune before the end."

However, this breaks when entries are **non-contiguous**. If there is a gap
between entry A (`RenderedEnd=10`) and entry B (`RenderedStart=12`), and
`renderedEnd=12`, then `lookupPos=11` falls in the gap. No entry contains
position 11, so `endEntry` is nil and `srcEnd` falls back to `renderedEnd`
(which is wrong — it's a rendered position used as a source position).

Non-contiguous entries occur at block boundaries. For example, the paragraph
break span (`\n` with `ParaBreak=true`) between two paragraphs may not have a
source map entry if it's synthetic. Table top/bottom borders have zero-length
source ranges (`SourceStart == SourceEnd`).

**Fix**: When `endEntry` is nil after the lookup, search for the nearest entry
before `lookupPos`:

```go
if endEntry == nil {
    // Find the last entry whose RenderedEnd <= renderedEnd
    for i := len(sm.entries) - 1; i >= 0; i-- {
        if sm.entries[i].RenderedEnd <= renderedEnd {
            endEntry = &sm.entries[i]
            break
        }
    }
}
```

If still nil (position is before all entries), fall back to `srcEnd = 0`.

The same issue exists for the start-position lookup (lines 39-45), though it's
less likely to trigger because start positions typically land within entries.
Apply the same nearest-entry fallback there: find the first entry whose
`RenderedStart >= renderedStart`.

### Bug 5: Missing bounds validation in wind.go

**Location**: `wind.go:1665-1694` (`syncSourceSelection`), `wind.go:1548-1579`
(`PreviewSnarf`), `wind.go:1584-1607` (`PreviewLookText`),
`wind.go:1612-1614` (`PreviewExecText`)

**Current state**: `syncSourceSelection()` already has bounds clamping:
```go
bodyLen := w.body.file.Nr()
if srcStart < 0 { srcStart = 0 }
if srcEnd < 0 { srcEnd = 0 }
if srcStart > bodyLen { srcStart = bodyLen }
if srcEnd > bodyLen { srcEnd = bodyLen }
```

`PreviewSnarf()` also has clamping:
```go
bodyLen := w.body.file.Nr()
if srcStart < 0 { srcStart = 0 }
if srcEnd > bodyLen { srcEnd = bodyLen }
```

**Remaining problems**:
1. `PreviewSnarf` clamps `srcStart < 0` but not `srcStart > bodyLen`, and
   clamps `srcEnd > bodyLen` but not `srcEnd < 0`. Inconsistent with
   `syncSourceSelection`.
2. `PreviewLookText` and `PreviewExecText` extract from `content.Plain()`
   using `p0`/`p1` (rendered positions) and check `p0 < 0 || p1 > len(plainText)`,
   but do not check `p0 > p1` or `p1 < 0`.
3. After source edits (typing in preview mode), the source map becomes stale
   until `UpdatePreview()` re-parses. During this window, `ToSource()` can
   return positions beyond the new buffer length. The clamping handles this but
   silently produces incorrect selections (selecting past the edit point).

**Fix**:
1. Unify bounds validation into a helper function:
   ```go
   func clampToBuffer(start, end, bufLen int) (int, int) {
       if start < 0 { start = 0 }
       if end < 0 { end = 0 }
       if start > bufLen { start = bufLen }
       if end > bufLen { end = bufLen }
       if start > end { start = end }
       return start, end
   }
   ```
2. Use this helper in `syncSourceSelection`, `PreviewSnarf`, and anywhere
   source positions are used with the body buffer.
3. Add `start > end` check — if clamping causes inversion, normalize to a
   point at `end`.

---

## PrefixLen Specification: Bytes vs Runes

**Decision**: `PrefixLen` (renamed to `PrefixRuneLen`) is a **rune count**.

**Rationale**:
- All arithmetic in `ToSource()` and `sourceRuneToRendered()` operates on rune
  positions (`SourceRuneStart`, `RenderedStart`, etc.).
- Mixing byte counts into rune arithmetic is a category error.
- All current prefixes are ASCII (`# `, `## `, `### `, `#### `, `##### `,
  `###### `), so the numeric values are unchanged.
- Future prefixes (blockquote `> `, list markers) are also ASCII.

**Migration**: Rename the field and update all usages. Add a comment:
```go
PrefixRuneLen int // Rune length of source prefix not rendered (e.g., "# " = 2)
```

---

## Round-Trip Invariant

The source map must satisfy the following invariant:

### Invariant R1: Rendered→Source→Rendered containment

For any valid rendered range `[r0, r1)`:
```
srcStart, srcEnd := sm.ToSource(r0, r1)
r0', r1' := sm.ToRendered(srcStart, srcEnd)
assert r0' <= r0 && r1' >= r1
```

The round-trip may **expand** the selection (e.g., selecting partial bold text
returns the full bold source, which maps back to the full rendered bold text)
but must never **shrink** it.

### Invariant R2: Source→Rendered→Source containment

For any valid source rune range `[s0, s1)` that falls within source map entries:
```
r0, r1 := sm.ToRendered(s0, s1)
s0', s1' := sm.ToSource(r0, r1)
assert s0' <= s0 && s1' >= s1
```

### Invariant R3: Point selection identity

For any valid rendered position `p`:
```
srcStart, srcEnd := sm.ToSource(p, p)
assert srcStart == srcEnd
```

A click must map to a click, never a range.

### Invariant R4: Monotonicity

For rendered positions `a < b`:
```
sa, _ := sm.ToSource(a, a)
sb, _ := sm.ToSource(b, b)
assert sa <= sb
```

Monotonicity ensures that moving right in the rendered view never moves
backward in the source.

---

## Boundary Case Specifications

### Beginning of entry

When `renderedStart == entry.RenderedStart` (selection starts at the first
rendered character of a formatted element):
- For a **range selection** (`renderedStart != renderedEnd`): `srcStart` should
  be `entry.SourceRuneStart` (include opening markup).
- For a **point selection** (`renderedStart == renderedEnd`): `srcStart` should
  be `entry.SourceRuneStart + entry.PrefixRuneLen` (point past the markup, in
  the content).

### End of entry

When `renderedEnd == entry.RenderedEnd` (selection ends at the last rendered
character of a formatted element):
- For a **range selection**: `srcEnd` should be `entry.SourceRuneEnd` (include
  closing markup).
- For a **point selection**: handled by R3 (same as start).

### Beginning of document

`renderedStart = 0`: Should find the first entry. If no entry starts at 0
(e.g., document starts with a synthetic element), use nearest-entry fallback.

### End of document

`renderedEnd = totalRenderedRunes`: Should find the last entry. If beyond all
entries, clamp to the last entry's source end.

### Non-contiguous entries

When the rendered position falls in a gap between entries (e.g., between a
paragraph break span and the next paragraph), use the nearest-entry fallback:
- For start: find first entry at or after the position.
- For end: find last entry at or before the position.

### Synthetic entries (zero-length source)

Table borders have `SourceStart == SourceEnd`. Selections landing in these
entries should map to the nearest real source position. The current code
produces `srcEnd = srcStart` for such entries, which is acceptable (selecting
a table border maps to a zero-length source range at the border position).

---

## Test Categories

All tests should go in `markdown/sourcemap_correctness_test.go`.

### Category A: Round-trip consistency

For a variety of documents, verify invariants R1 and R2:
1. Plain text: `"Hello, World!"` — trivial 1:1 round-trip.
2. Bold: `"**bold**"` — rendered→source→rendered produces [0,4].
3. Italic: `"*italic*"` — same pattern.
4. Heading: `"# Title"` — rendered [0,5] → source [0,7] → rendered [0,5].
5. Mixed: `"# Title\nSome **bold** text\n"` — test each span separately.
6. Code block: `"```\ncode\n```"` — code content round-trip.
7. List item: `"- item\n"` — bullet and content round-trip.
8. Multiple paragraphs: `"Para one.\n\nPara two."` — round-trip across break.

Targets: Bug 2 (asymmetry), Bug 3 (normalization masking).

### Category B: Cross-boundary selections

Selections spanning multiple entries of different types:
1. Bold → plain: select from `"bol"` into `" text"` in `"**bold** text"`.
2. Heading → paragraph: select from `"Titl"` into `"Body"` in `"# Title\nBody"`.
3. List item → list item: select across `"- one\n- two\n"`.
4. Code block → paragraph: select from inside code to text after `"```\ncode\n```\nAfter"`.
5. Plain → bold: select from `"some "` into `"bol"` in `"some **bold** text"`.

Targets: Bug 4 (boundary lookup off-by-one).

### Category C: Exact boundary positions

Selections where start/end land exactly on entry boundaries:
1. `renderedEnd == entry.RenderedEnd` for each element type.
2. `renderedStart == entry.RenderedStart` for each element type.
3. Selection spanning exactly one entry (start=RenderedStart, end=RenderedEnd).
4. Selection spanning two adjacent entries exactly.
5. Selection at a non-contiguous boundary (table border gap).

Targets: Bug 4 (boundary lookup), Bug 2 (expansion heuristic).

### Category D: Point selections

Single-rune clicks (renderedStart == renderedEnd):
1. Click at start of heading: `# Hello` at rendered position 0.
2. Click in middle of heading: position 3.
3. Click at start of bold: `**bold** text` at rendered position 0.
4. Click in plain text between formatted elements.
5. Click at paragraph break position.
6. Click at document start (position 0).
7. Click at document end.

Targets: Bug 3 (normalization), Bug 1 (PrefixLen), Invariant R3.

### Category E: Non-ASCII content

Headings, bold, links containing multi-byte runes:
1. `"# Über"` — heading with 2-byte rune.
2. `"**café**"` — bold with multi-byte.
3. `"# 日本語"` — heading with 3-byte runes.
4. `"Some **日本語** text"` — bold multi-byte in mixed content.
5. `"# Ém\nSome text"` — heading with multi-byte followed by paragraph.

Targets: Bug 1 (PrefixLen byte/rune confusion).

### Category F: Edge positions

1. Position 0 in empty document.
2. Position 0 in non-empty document.
3. Position at exact document end.
4. Position beyond document end (out of range).
5. Negative position (should not crash).
6. Empty document (no entries) — `ToSource(0,0)` and `ToRendered(0,0)`.
7. Single-character document.

Targets: Bug 5 (bounds validation), robustness.

### Category G: Stale source map (integration tests in wind_selection_test.go)

1. Selection after source edit that shortens the document.
2. Selection after source edit that lengthens the document.
3. Point selection in heading after editing the heading text.

Targets: Bug 5 (bounds validation in wind.go).

---

## Implementation Order

1. **Rename `PrefixLen` to `PrefixRuneLen`** — purely mechanical rename across
   `sourcemap.go`, `sourcemap_test.go`. No behavioral change.

2. **Unify `ToSource` position calculation** — apply `SourceRuneStart +
   PrefixRuneLen + offset` formula for both start and end, then apply
   boundary expansion separately. This fixes Bug 2 and makes Bug 3's
   normalization unnecessary.

3. **Fix entry boundary lookup** — add nearest-entry fallback when position
   falls in a gap. This fixes Bug 4.

4. **Remove or convert point selection normalization** — after Bug 2 fix,
   verify the normalization is unnecessary. Convert to assertion. Fixes Bug 3.

5. **Add bounds validation helper to wind.go** — extract `clampToBuffer`,
   apply to all source-position consumers. Fixes Bug 5.

---

## Files Modified

| File | Changes |
|------|---------|
| `markdown/sourcemap.go` | Rename PrefixLen, fix ToSource formula, fix boundary lookup, fix normalization |
| `markdown/sourcemap_test.go` | Update PrefixLen references if struct is used in tests |
| `markdown/sourcemap_correctness_test.go` | New file: comprehensive test suite (Categories A-F) |
| `wind.go` | Add clampToBuffer helper, fix PreviewSnarf/PreviewLookText bounds |
| `wind_test.go` or `wind_selection_test.go` | Category G integration tests |

---

## Non-Goals

- **Do not change `parse.go` or entry generation** — that's Phase 2.
- **Do not change `sourceRuneToRendered()`** — it is already correct for its
  purpose (ToRendered direction). Only ToSource needs fixing.
- **Do not add markup-boundary expansion heuristics** — that's Phase 1.4.
  The current boundary expansion (`srcStart = SourceRuneStart` when at entry
  start) stays as-is; Phase 1.4 will refine it.
