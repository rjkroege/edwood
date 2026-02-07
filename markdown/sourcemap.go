package markdown

import "sort"

// EntryKind discriminates the type of a source map entry so that gap handling
// and marker-length computation can use an explicit switch instead of heuristics.
type EntryKind int

const (
	KindPlainText       EntryKind = iota // 1:1 mapping (default zero value)
	KindSymmetricMarker                  // bold/italic/code: opening = closing = extra/2
	KindPrefix                           // heading/blockquote: PrefixLen runes stripped
	KindTableCell                        // cell content within padded row
	KindSynthetic                        // border lines with no source
)

// startGapSnap reports whether a gap before this entry kind should snap to
// the preceding entry's SourceRuneEnd (true) or the following entry's
// SourceRuneStart (false).
func (k EntryKind) startGapSnap() bool {
	return k == KindTableCell
}

// endGapSnap reports whether a gap after this entry kind should snap to the
// entry's SourceRuneEnd (true) or use a 1:1 offset from it (false).
func (k EntryKind) endGapSnap() bool {
	return k == KindTableCell
}

// SourceMap maps positions in rendered content back to positions in source markdown.
type SourceMap struct {
	entries            []SourceMapEntry
	runePositionsValid bool // true after populateRunePositions; false when byte positions are shifted without rune recalculation
}

// SourceMapEntry maps a range in rendered content to a range in source markdown.
type SourceMapEntry struct {
	RenderedStart   int // Rune position in rendered content
	RenderedEnd     int
	SourceStart     int // Byte position in source markdown
	SourceEnd       int
	SourceRuneStart int // Rune position in source markdown
	SourceRuneEnd   int
	PrefixLen       int       // Length of source prefix not in rendered (e.g., "# " for headings)
	Kind            EntryKind // Discriminant for entry type
	CellBorderPos   int       // For KindTableCell: rendered position of the │ delimiter to the left of this cell
}

// searchRendered returns the index of the entry containing the rendered
// position pos (i.e. RenderedStart <= pos < RenderedEnd), or -1 if pos
// falls in a gap between entries.
func (sm *SourceMap) searchRendered(pos int) int {
	// Find rightmost entry with RenderedStart <= pos.
	i := sort.Search(len(sm.entries), func(i int) bool {
		return sm.entries[i].RenderedStart > pos
	}) - 1
	if i >= 0 && pos < sm.entries[i].RenderedEnd {
		return i
	}
	return -1
}

// precedingRendered returns the index of the last entry with
// RenderedEnd <= pos, or -1 if no such entry exists.
func (sm *SourceMap) precedingRendered(pos int) int {
	// Find rightmost entry with RenderedEnd <= pos.
	i := sort.Search(len(sm.entries), func(i int) bool {
		return sm.entries[i].RenderedEnd > pos
	}) - 1
	return i // -1 if none
}

// followingRendered returns the index of the first entry with
// RenderedStart >= pos, or -1 if no such entry exists.
func (sm *SourceMap) followingRendered(pos int) int {
	i := sort.Search(len(sm.entries), func(i int) bool {
		return sm.entries[i].RenderedStart >= pos
	})
	if i < len(sm.entries) {
		return i
	}
	return -1
}

// searchSource returns the index of the entry containing the source rune
// position pos (i.e. SourceRuneStart <= pos < SourceRuneEnd), or -1 if
// pos falls in a gap between entries.
func (sm *SourceMap) searchSource(pos int) int {
	i := sort.Search(len(sm.entries), func(i int) bool {
		return sm.entries[i].SourceRuneStart > pos
	}) - 1
	if i >= 0 && pos < sm.entries[i].SourceRuneEnd {
		return i
	}
	return -1
}

// precedingSource returns the index of the last entry with
// SourceRuneEnd <= pos, or -1 if no such entry exists.
func (sm *SourceMap) precedingSource(pos int) int {
	i := sort.Search(len(sm.entries), func(i int) bool {
		return sm.entries[i].SourceRuneEnd > pos
	}) - 1
	return i // -1 if none
}

// ToSource maps a range in rendered content (renderedStart, renderedEnd) to
// the corresponding range in the source markdown as RUNE positions.
// This is used by syncSourceSelection to set body.q0/q1 which expect rune positions.
// When the selection spans formatted elements, it expands to include the full
// source markup (e.g., selecting "bold" in "**bold**" returns the rune range of "**bold**").
func (sm *SourceMap) ToSource(renderedStart, renderedEnd int) (srcStart, srcEnd int) {
	if len(sm.entries) == 0 {
		return renderedStart, renderedEnd
	}
	sm.requireRunePositions("ToSource")

	// Find the entry containing renderedStart
	srcStart = -1
	var startEntry *SourceMapEntry
	if idx := sm.searchRendered(renderedStart); idx >= 0 {
		startEntry = &sm.entries[idx]
	}

	if startEntry == nil {
		// Position falls in a gap between entries.
		// Find the preceding entry to determine gap behavior.
		var preceding *SourceMapEntry
		if idx := sm.precedingRendered(renderedStart); idx >= 0 {
			preceding = &sm.entries[idx]
		}
		if preceding != nil && preceding.Kind.startGapSnap() {
			// Default: snap to preceding cell's end.
			srcStart = preceding.SourceRuneEnd
			// If we're past the │ border of the following cell,
			// snap to its content start instead — the click is
			// in the following cell's area.
			if idx := sm.followingRendered(renderedStart); idx >= 0 {
				following := &sm.entries[idx]
				if following.Kind == KindTableCell && renderedStart > following.CellBorderPos {
					srcStart = following.SourceRuneStart
				}
			}
		} else {
			// Find nearest entry after and snap to its start.
			if idx := sm.followingRendered(renderedStart); idx >= 0 {
				srcStart = sm.entries[idx].SourceRuneStart
			}
			if srcStart == -1 {
				srcStart = renderedStart
			}
		}
	} else {
		// Use the unified formula: map rendered position to source content
		// position past the opening marker, then apply boundary expansion.
		offset := renderedStart - startEntry.RenderedStart
		srcStart = startEntry.SourceRuneStart + entryOpeningLen(startEntry) + offset

		// Boundary expansion: if selection starts at entry start (range selection),
		// include opening markup.
		if renderedStart == startEntry.RenderedStart && renderedStart != renderedEnd {
			srcStart = startEntry.SourceRuneStart
		}
	}

	// Find the entry containing renderedEnd-1 (or handle empty/edge cases)
	srcEnd = -1
	var endEntry *SourceMapEntry
	lookupPos := renderedEnd
	if renderedEnd > renderedStart {
		lookupPos = renderedEnd - 1
	}
	if idx := sm.searchRendered(lookupPos); idx >= 0 {
		endEntry = &sm.entries[idx]
	}

	if endEntry == nil {
		// Position falls in a gap — find nearest entry before renderedEnd.
		// Gap behavior depends on the entry's Kind.
		if idx := sm.precedingRendered(renderedEnd); idx >= 0 {
			endEntry = &sm.entries[idx]
		}
		if endEntry != nil {
			if endEntry.Kind.endGapSnap() {
				// Default: snap to this cell's end.
				srcEnd = endEntry.SourceRuneEnd
				// If we're past the │ border of the following cell,
				// snap to its content start instead.
				if idx := sm.followingRendered(renderedEnd); idx >= 0 {
					following := &sm.entries[idx]
					if following.Kind == KindTableCell && renderedEnd > following.CellBorderPos {
						srcEnd = following.SourceRuneStart
					}
				}
			} else {
				srcEnd = endEntry.SourceRuneEnd + (renderedEnd - endEntry.RenderedEnd)
			}
		} else {
			srcEnd = renderedEnd
		}
	} else {
		// Boundary expansion: if selection ends at entry end (range selection),
		// include closing markup.
		if renderedEnd == endEntry.RenderedEnd {
			srcEnd = endEntry.SourceRuneEnd
		} else {
			// Use the unified formula: opening marker length + content offset
			offset := renderedEnd - endEntry.RenderedStart
			srcEnd = endEntry.SourceRuneStart + entryOpeningLen(endEntry) + offset
		}
	}

	// A point selection in rendered content must map to a point in source.
	// With the unified formula, start and end should agree for point selections.
	// Keep as safety normalization.
	if renderedStart == renderedEnd && srcStart != srcEnd {
		srcStart = srcEnd
	}

	return srcStart, srcEnd
}

// entryOpeningLen computes the rune length of the opening marker for a source
// map entry, using the entry's Kind discriminant.
func entryOpeningLen(e *SourceMapEntry) int {
	switch e.Kind {
	case KindPrefix:
		return e.PrefixLen
	case KindSymmetricMarker:
		extra := (e.SourceRuneEnd - e.SourceRuneStart) - (e.RenderedEnd - e.RenderedStart)
		return extra / 2
	default:
		return 0
	}
}

// ToRendered maps a range in source markdown (srcRuneStart, srcRuneEnd as rune positions)
// to the corresponding range in the rendered content (as rune positions).
// Returns (-1, -1) if no mapping exists.
// This is the inverse of ToSource(): given source positions (e.g., from search()),
// find where that content appears in the rendered preview.
func (sm *SourceMap) ToRendered(srcRuneStart, srcRuneEnd int) (renderedStart, renderedEnd int) {
	if len(sm.entries) == 0 {
		return -1, -1
	}
	sm.requireRunePositions("ToRendered")

	// Find the entry containing srcRuneStart
	renderedStart = -1
	if idx := sm.searchSource(srcRuneStart); idx >= 0 {
		renderedStart = sm.sourceRuneToRendered(&sm.entries[idx], srcRuneStart)
	}

	if renderedStart == -1 {
		// Source position falls in a gap between entries (e.g., a newline
		// between paragraph lines that becomes a join space in rendered
		// content). Find the nearest entry before this position and map
		// to its rendered end.
		if idx := sm.precedingSource(srcRuneStart); idx >= 0 {
			renderedStart = sm.entries[idx].RenderedEnd
		}
		if renderedStart == -1 {
			// Before all entries — map to start of first entry.
			if sm.entries[0].SourceRuneStart > srcRuneStart {
				renderedStart = sm.entries[0].RenderedStart
			} else {
				return -1, -1
			}
		}
	}

	// Find the entry containing srcRuneEnd-1 (or handle edge cases)
	renderedEnd = -1
	lookupPos := srcRuneEnd
	if srcRuneEnd > srcRuneStart {
		lookupPos = srcRuneEnd - 1
	}
	if idx := sm.searchSource(lookupPos); idx >= 0 {
		e := &sm.entries[idx]
		// For end position, if srcRuneEnd is at or past the entry end,
		// map to the full rendered end
		if srcRuneEnd >= e.SourceRuneEnd {
			renderedEnd = e.RenderedEnd
		} else {
			renderedEnd = sm.sourceRuneToRendered(e, srcRuneEnd)
		}
	}

	if renderedEnd == -1 {
		// Same gap handling as for start: find nearest entry before.
		if idx := sm.precedingSource(lookupPos); idx >= 0 {
			renderedEnd = sm.entries[idx].RenderedEnd
		}
		if renderedEnd == -1 {
			if len(sm.entries) > 0 && sm.entries[0].SourceRuneStart > lookupPos {
				renderedEnd = sm.entries[0].RenderedStart
			} else {
				return -1, -1
			}
		}
	}

	return renderedStart, renderedEnd
}

// sourceRuneToRendered maps a single source rune position to a rendered position
// within a given entry. For 1:1 entries (plain text), the offset is direct.
// For formatted entries (bold, italic, heading, code), the source contains
// opening and closing markers around the rendered content.
func (sm *SourceMap) sourceRuneToRendered(e *SourceMapEntry, srcRunePos int) int {
	offset := srcRunePos - e.SourceRuneStart
	renderedLen := e.RenderedEnd - e.RenderedStart

	openingLen := entryOpeningLen(e)
	if openingLen == 0 {
		// 1:1 mapping (plain text, code block content, list content, etc.)
		return e.RenderedStart + offset
	}

	if offset <= openingLen {
		// Within or at the opening marker - map to rendered start
		return e.RenderedStart
	}

	contentOffset := offset - openingLen
	if contentOffset >= renderedLen {
		// Within or at the closing marker - map to rendered end
		return e.RenderedEnd
	}

	return e.RenderedStart + contentOffset
}


// requireRunePositions panics if rune positions have not been populated or have
// been invalidated by byte-position shifts. This turns silent wrong-answer bugs
// (stale SourceRuneStart/End values) into loud failures.
func (sm *SourceMap) requireRunePositions(caller string) {
	if !sm.runePositionsValid {
		panic("sourcemap: " + caller + " called before PopulateRunePositions")
	}
}

// PopulateRunePositions fills in SourceRuneStart/SourceRuneEnd for all entries
// by converting byte positions (SourceStart/SourceEnd) to rune positions using
// the provided source text. This should be called after Stitch to ensure rune
// positions are derived from the full document's byte-to-rune mapping.
func (sm *SourceMap) PopulateRunePositions(source string) {
	sm.populateRunePositions(source)
}

// InvalidateRunePositions marks the source map's rune positions as stale.
// Call this after shifting byte positions (e.g., in ParseRegion) to prevent
// mapping functions from using outdated rune positions.
func (sm *SourceMap) InvalidateRunePositions() {
	sm.runePositionsValid = false
}

// populateRunePositions fills in SourceRuneStart/SourceRuneEnd for all entries
// by converting byte positions to rune positions using the source text.
func (sm *SourceMap) populateRunePositions(source string) {
	if len(sm.entries) == 0 {
		sm.runePositionsValid = true
		return
	}

	sourceLen := len(source)

	// Build byte-to-rune position lookup table.
	// b2r[byteOffset] = runeOffset for each valid byte boundary.
	b2r := make([]int, sourceLen+1)
	runePos := 0
	bi := 0
	for _, r := range source {
		b2r[bi] = runePos
		bi += len(string(r))
		runePos++
	}
	b2r[sourceLen] = runePos

	for i := range sm.entries {
		e := &sm.entries[i]
		start := e.SourceStart
		end := e.SourceEnd
		if start < 0 {
			start = 0
		}
		if start > sourceLen {
			start = sourceLen
		}
		if end < 0 {
			end = 0
		}
		if end > sourceLen {
			end = sourceLen
		}
		e.SourceRuneStart = b2r[start]
		e.SourceRuneEnd = b2r[end]
	}
	sm.runePositionsValid = true
}
