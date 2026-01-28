// Package text provides the Text type and related components for edwood.
// This file contains display-related types and methods for text.
package text

// DisplayManager will handle text display operations.
// This extracts display logic from the main Text type, including operations like:
// - Origin management (org) - the first visible character position
// - Redraw state tracking
// - Scroll position calculations
// - Visibility checks
type DisplayManager struct {
	state *DisplayState
}

// NewDisplayManager creates a new DisplayManager with the given state.
func NewDisplayManager(state *DisplayState) *DisplayManager {
	if state == nil {
		state = NewDisplayState()
	}
	return &DisplayManager{
		state: state,
	}
}

// State returns the underlying DisplayState.
func (dm *DisplayManager) State() *DisplayState {
	return dm.state
}

// Org returns the origin (first visible character position).
func (dm *DisplayManager) Org() int {
	return dm.state.Org()
}

// SetOrg sets the origin.
func (dm *DisplayManager) SetOrg(org int) {
	dm.state.SetOrg(org)
}

// NeedsRedraw returns true if the text needs to be redrawn.
func (dm *DisplayManager) NeedsRedraw() bool {
	return dm.state.NeedsRedraw()
}

// SetNeedsRedraw marks the text as needing redraw.
func (dm *DisplayManager) SetNeedsRedraw(needs bool) {
	dm.state.SetNeedsRedraw(needs)
}

// ClearRedrawFlag clears the redraw flag after drawing.
func (dm *DisplayManager) ClearRedrawFlag() {
	dm.state.ClearRedrawFlag()
}

// CalculateVisibleRange calculates the range of text positions that would be
// visible given the current origin and a frame size (number of visible characters).
// Returns (start, end) where start is the origin and end is start + nchars.
func (dm *DisplayManager) CalculateVisibleRange(nchars int) Range {
	org := dm.state.Org()
	return Range{Start: org, End: org + nchars}
}

// IsPositionVisible returns true if the given position is within the visible range.
// nchars is the number of characters currently displayed in the frame.
func (dm *DisplayManager) IsPositionVisible(pos, nchars int) bool {
	org := dm.state.Org()
	return pos >= org && pos < org+nchars
}

// IsRangeVisible returns true if any part of the given range is visible.
// nchars is the number of characters currently displayed in the frame.
func (dm *DisplayManager) IsRangeVisible(r Range, nchars int) bool {
	org := dm.state.Org()
	visEnd := org + nchars
	// A range is visible if it overlaps with [org, org+nchars)
	return r.Start < visEnd && r.End > org
}

// IsRangeFullyVisible returns true if the entire range is visible.
// nchars is the number of characters currently displayed in the frame.
func (dm *DisplayManager) IsRangeFullyVisible(r Range, nchars int) bool {
	org := dm.state.Org()
	visEnd := org + nchars
	return r.Start >= org && r.End <= visEnd
}

// PositionToFrameOffset converts a buffer position to a frame offset.
// Returns the offset from the origin, or -1 if the position is before the origin.
func (dm *DisplayManager) PositionToFrameOffset(pos int) int {
	org := dm.state.Org()
	if pos < org {
		return -1
	}
	return pos - org
}

// FrameOffsetToPosition converts a frame offset to a buffer position.
func (dm *DisplayManager) FrameOffsetToPosition(offset int) int {
	return dm.state.Org() + offset
}

// BackNL returns the position at the beginning of the line after backing up n lines
// starting from position p. This is a helper for scroll operations.
// textReader returns the rune at a specific position (or 0 if out of bounds).
// maxLineLen limits how far back we search for a newline on a single line.
func (dm *DisplayManager) BackNL(p, n int, charReader func(pos int) rune, maxLineLen int) int {
	if maxLineLen <= 0 {
		maxLineLen = 128 // default from text.go
	}
	// look for start of this line if n==0
	if n == 0 && p > 0 && charReader(p-1) != '\n' {
		n = 1
	}
	for n > 0 && p > 0 {
		n--
		p-- // it's at a newline now; back over it
		if p == 0 {
			break
		}
		// at maxLineLen chars, call it a line anyway
		for j := maxLineLen; j > 0 && p > 0; p-- {
			if charReader(p-1) == '\n' {
				break
			}
			j--
		}
	}
	return p
}

// CalculateNewOrigin calculates a new origin for scrolling to make a target
// position visible. It returns the new origin position.
//
// Parameters:
// - targetPos: the position we want to make visible
// - nchars: number of characters currently displayed
// - maxlines: maximum number of lines in the frame
// - quarterScroll: if true, scroll by 1/4 of visible lines; if false, scroll by 3/4
// - charReader: function to read characters from the buffer
// - textLen: total length of the text buffer
//
// Returns the new origin position.
func (dm *DisplayManager) CalculateNewOrigin(targetPos, nchars, maxlines int, quarterScroll bool, charReader func(pos int) rune, textLen int) int {
	org := dm.state.Org()
	qe := org + nchars

	// If target is already visible, no change needed
	if org <= targetPos && targetPos < qe {
		return org
	}

	// If we're at the end and it's visible (edge case from Show)
	if targetPos == qe && qe == textLen {
		// Check if we can show more
		if textLen > 0 && charReader(textLen-1) == '\n' {
			// might be able to fit it
			return org
		}
		return org
	}

	// Need to scroll - calculate how many lines to back up
	var nl int
	if quarterScroll {
		nl = maxlines / 4
	} else {
		nl = 3 * maxlines / 4
	}

	// Back up n lines from target position
	return dm.BackNL(targetPos, nl, charReader, 128)
}

// AdjustOriginForExact adjusts the origin to land on a line boundary if exact is false.
// When exact is false, this searches forward from org until it finds a newline
// (up to 256 characters), making the origin start at the beginning of a line.
// charReader returns the rune at a specific position.
// textLen is the total length of the text buffer.
func (dm *DisplayManager) AdjustOriginForExact(org int, exact bool, charReader func(pos int) rune, textLen int) int {
	if org <= 0 || exact {
		return org
	}
	// Check if we're already at a line start
	if charReader(org-1) == '\n' {
		return org
	}
	// org is an estimate of the char posn; find a newline
	// don't try harder than 256 chars
	for i := 0; i < 256 && org < textLen; i++ {
		if charReader(org) == '\n' {
			org++
			break
		}
		org++
	}
	return org
}

// ScrollDelta calculates how much the origin should change for a scroll operation.
// A positive delta means scroll down (advance origin), negative means scroll up.
// lineHeight is used to calculate character positions from pixel offsets.
func (dm *DisplayManager) ScrollDelta(currentOrg, delta, nchars, textLen int) int {
	if delta == 0 {
		return currentOrg
	}
	if delta < 0 {
		// Scroll up - this needs BackNL which requires charReader
		// Return -1 to indicate caller should use BackNL
		return -1
	}
	// Scroll down
	if currentOrg+nchars >= textLen {
		// Already at end, can't scroll further
		return currentOrg
	}
	// Caller needs to calculate exact position using frame's Charofpt
	return -1
}
