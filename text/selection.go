// Package text provides the Text type and related components for edwood.
package text

import "strings"

// Bracket pairs for matching. Left brackets are opening, right are closing.
// These match the left/right arrays in text.go.
var (
	// left1/right1: standard brackets and guillemets
	leftBrackets  = []rune{'{', '[', '(', '<', '\xab'} // « is 0xab
	rightBrackets = []rune{'}', ']', ')', '>', '\xbb'} // » is 0xbb
	// Quotes are matched with themselves
	quotes = []rune{'\'', '"', '`'}
)

// isAlnum returns true if c is an alphanumeric character.
// This matches the isalnum function in util.go.
func isAlnum(c rune) bool {
	// Hard to get absolutely right.  Use what we know about ASCII
	// and assume anything above the Latin control characters is
	// potentially an alphanumeric.
	if c <= ' ' {
		return false
	}
	if 0x7F <= c && c <= 0xA0 {
		return false
	}
	if strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^`{|}~", c) {
		return false
	}
	return true
}

// indexRune returns the index of r in the slice, or -1 if not found.
func indexRune(slice []rune, r rune) int {
	for i, c := range slice {
		if c == r {
			return i
		}
	}
	return -1
}

// SelectionManager will handle text selection operations.
// This is a stub for Phase 6B implementation.
//
// The SelectionManager will extract selection logic from the main Text type,
// including operations like:
// - Expanding selections (word, line, quoted strings)
// - Double-click selection behavior
// - Selection clamping and validation
// - Selection coordinate transformation (frame <-> buffer positions)
type SelectionManager struct {
	state *SelectionState
}

// NewSelectionManager creates a new SelectionManager with the given state.
func NewSelectionManager(state *SelectionState) *SelectionManager {
	if state == nil {
		state = NewSelectionState()
	}
	return &SelectionManager{
		state: state,
	}
}

// State returns the underlying SelectionState.
func (sm *SelectionManager) State() *SelectionState {
	return sm.state
}

// Selection returns the current selection as a Range.
func (sm *SelectionManager) Selection() Range {
	return sm.state.Selection()
}

// SetSelection sets the selection range.
func (sm *SelectionManager) SetSelection(q0, q1 int) {
	sm.state.SetSelection(q0, q1)
}

// HasSelection returns true if there is a non-empty selection.
func (sm *SelectionManager) HasSelection() bool {
	return sm.state.HasSelection()
}

// ClearSelection collapses the selection to the cursor position.
func (sm *SelectionManager) ClearSelection() {
	sm.state.ClearSelection()
}

// ClampSelection ensures the selection is within valid bounds.
// Returns the clamped range.
func (sm *SelectionManager) ClampSelection(maxLen int) Range {
	q0 := sm.state.Q0()
	q1 := sm.state.Q1()

	if q0 < 0 {
		q0 = 0
	}
	if q1 < 0 {
		q1 = 0
	}
	if q0 > maxLen {
		q0 = maxLen
	}
	if q1 > maxLen {
		q1 = maxLen
	}

	sm.state.SetSelection(q0, q1)
	return Range{Start: q0, End: q1}
}

// ExpandToWord expands the selection at the given position to include
// the entire word. A word is defined as a sequence of alphanumeric characters.
// textReader should return the runes in the range [start, end).
// If pos is on a non-alphanumeric character, returns an empty range at pos.
func (sm *SelectionManager) ExpandToWord(pos int, textReader func(start, end int) []rune) Range {
	q0 := pos
	q1 := pos

	// Check if we're starting on an alphanumeric character
	// If not, we're not inside a word - return empty range
	startRunes := textReader(pos, pos+1)
	if len(startRunes) == 0 || !isAlnum(startRunes[0]) {
		// Not inside a word - check if we're adjacent to a word on the left
		// (This handles the case where cursor is just after a word)
		if pos > 0 {
			leftRunes := textReader(pos-1, pos)
			if len(leftRunes) == 0 || !isAlnum(leftRunes[0]) {
				// Not adjacent to a word either
				return Range{Start: pos, End: pos}
			}
			// We're just after a word - don't select it
			// (clicking on space shouldn't select adjacent word)
			return Range{Start: pos, End: pos}
		}
		return Range{Start: pos, End: pos}
	}

	// Try filling out word to the right
	for {
		runes := textReader(q1, q1+1)
		if len(runes) == 0 {
			break
		}
		if !isAlnum(runes[0]) {
			break
		}
		q1++
	}

	// Try filling out word to the left
	for q0 > 0 {
		runes := textReader(q0-1, q0)
		if len(runes) == 0 {
			break
		}
		if !isAlnum(runes[0]) {
			break
		}
		q0--
	}

	return Range{Start: q0, End: q1}
}

// ExpandToLine expands the selection at the given position to include
// the entire line (excluding the newline character itself).
// textReader should return the runes in the range [start, end).
// If pos is at a newline, selects the line before that newline.
// If on an empty line (between newlines), returns empty range.
func (sm *SelectionManager) ExpandToLine(pos int, textReader func(start, end int) []rune) Range {
	q0 := pos
	q1 := pos

	// Check what character is at pos
	runes := textReader(pos, pos+1)
	atNewline := len(runes) > 0 && runes[0] == '\n'
	atEnd := len(runes) == 0

	// If we're at a newline, we want to select the line BEFORE it
	if atNewline {
		// Check if there's content before this newline
		if pos == 0 {
			// At the very start, nothing to select
			return Range{Start: 0, End: 0}
		}
		// Check if previous char is also a newline (empty line case)
		prevRunes := textReader(pos-1, pos)
		if len(prevRunes) > 0 && prevRunes[0] == '\n' {
			// Empty line - return empty selection
			return Range{Start: pos, End: pos}
		}
		// Select the line before this newline
		q1 = pos
		q0 = pos
	}

	// Expand left to find start of line (stop at newline or beginning)
	for q0 > 0 {
		runes := textReader(q0-1, q0)
		if len(runes) == 0 {
			break
		}
		if runes[0] == '\n' {
			break
		}
		q0--
	}

	// If we started at a newline, don't expand right (we're selecting the line before)
	if !atNewline && !atEnd {
		// Expand right to find end of line (stop at newline or end)
		for {
			runes := textReader(q1, q1+1)
			if len(runes) == 0 {
				break
			}
			if runes[0] == '\n' {
				break
			}
			q1++
		}
	}

	return Range{Start: q0, End: q1}
}

// ExpandToBrackets expands the selection to match bracket/quote pairs.
// This handles parentheses, braces, brackets, quotes, and guillemets.
// If no bracket is found, it falls back to word expansion.
// charReader returns the rune at a specific position (or 0 if out of bounds).
// textReader is used for the word expansion fallback.
func (sm *SelectionManager) ExpandToBrackets(pos, textLen int, textReader func(start, end int) []rune, charReader func(pos int) rune) Range {
	// Get character to the left of pos (for checking opening brackets)
	var leftChar rune
	if pos > 0 {
		leftChar = charReader(pos - 1)
	}

	// Get character at pos (for checking closing brackets)
	var rightChar rune
	if pos < textLen {
		rightChar = charReader(pos)
	}

	// Try matching left bracket (looking right for closing)
	if idx := indexRune(leftBrackets, leftChar); idx != -1 {
		matchChar := rightBrackets[idx]
		if matchPos, ok := sm.matchBracket(leftChar, matchChar, 1, pos, textLen, charReader); ok {
			return Range{Start: pos, End: matchPos}
		}
	}

	// Try matching right bracket (looking left for opening)
	if idx := indexRune(rightBrackets, rightChar); idx != -1 {
		matchChar := leftBrackets[idx]
		if matchPos, ok := sm.matchBracket(rightChar, matchChar, -1, pos, textLen, charReader); ok {
			return Range{Start: matchPos + 1, End: pos}
		}
	}

	// Try matching quote to the left (looking right)
	if idx := indexRune(quotes, leftChar); idx != -1 {
		// For quotes, the opening and closing character are the same
		if matchPos, ok := sm.matchQuote(leftChar, 1, pos, textLen, charReader); ok {
			return Range{Start: pos, End: matchPos}
		}
	}

	// Try matching quote to the right (looking left)
	if idx := indexRune(quotes, rightChar); idx != -1 {
		if matchPos, ok := sm.matchQuote(rightChar, -1, pos, textLen, charReader); ok {
			return Range{Start: matchPos + 1, End: pos}
		}
	}

	// No bracket match found, fall back to word expansion
	return sm.ExpandToWord(pos, textReader)
}

// matchBracket searches for a matching bracket starting from pos.
// openChar is the opening bracket, closeChar is the closing bracket.
// dir is the direction: 1 for forward, -1 for backward.
// Returns the position of the matching bracket and true if found.
func (sm *SelectionManager) matchBracket(openChar, closeChar rune, dir int, pos, textLen int, charReader func(pos int) rune) (int, bool) {
	nest := 1
	q := pos
	for {
		if dir > 0 {
			if q >= textLen {
				break
			}
			c := charReader(q)
			if c == closeChar {
				nest--
				if nest == 0 {
					return q, true
				}
			} else if c == openChar {
				nest++
			}
			q++
		} else {
			if q <= 0 {
				break
			}
			q--
			c := charReader(q)
			if c == closeChar {
				nest--
				if nest == 0 {
					return q, true
				}
			} else if c == openChar {
				nest++
			}
		}
	}
	return q, false
}

// matchQuote searches for a matching quote starting from pos.
// quoteChar is the quote character (same for open and close).
// dir is the direction: 1 for forward, -1 for backward.
// Returns the position of the matching quote and true if found.
func (sm *SelectionManager) matchQuote(quoteChar rune, dir int, pos, textLen int, charReader func(pos int) rune) (int, bool) {
	q := pos
	for {
		if dir > 0 {
			if q >= textLen {
				break
			}
			c := charReader(q)
			if c == quoteChar {
				return q, true
			}
			q++
		} else {
			if q <= 0 {
				break
			}
			q--
			c := charReader(q)
			if c == quoteChar {
				return q, true
			}
		}
	}
	return q, false
}

// InSelection returns true if pos is within the current selection.
// For an empty selection (q0 == q1), returns false.
func (sm *SelectionManager) InSelection(pos int) bool {
	q0 := sm.state.Q0()
	q1 := sm.state.Q1()
	return q1 > q0 && q0 <= pos && pos <= q1
}

// Constrain returns the selection positions clamped to [0, maxLen].
// Unlike ClampSelection, this does not modify the underlying state.
func (sm *SelectionManager) Constrain(maxLen int) (p0, p1 int) {
	p0 = sm.state.Q0()
	p1 = sm.state.Q1()

	if p0 > maxLen {
		p0 = maxLen
	}
	if p1 > maxLen {
		p1 = maxLen
	}
	return p0, p1
}

// AdjustForInsert adjusts the selection after text is inserted.
// insertPos is where the insertion occurred, insertLen is the number of runes inserted.
// This matches the behavior in text.go's Inserted method.
func (sm *SelectionManager) AdjustForInsert(insertPos, insertLen int) {
	q0 := sm.state.Q0()
	q1 := sm.state.Q1()

	// Following text.go behavior:
	// if insertPos < q1, adjust q1
	// if insertPos < q0, adjust q0
	if insertPos < q1 {
		q1 += insertLen
	}
	if insertPos < q0 {
		q0 += insertLen
	}

	sm.state.SetSelection(q0, q1)
}

// AdjustForDelete adjusts the selection after text is deleted.
// delQ0 and delQ1 define the range that was deleted.
// This matches the behavior in text.go's Deleted method.
func (sm *SelectionManager) AdjustForDelete(delQ0, delQ1 int) {
	q0 := sm.state.Q0()
	q1 := sm.state.Q1()
	n := delQ1 - delQ0

	// Following text.go behavior:
	// if delQ0 < q0, adjust q0 by min(n, q0-delQ0)
	// if delQ0 < q1, adjust q1 by min(n, q1-delQ0)
	if delQ0 < q0 {
		delta := n
		if q0-delQ0 < delta {
			delta = q0 - delQ0
		}
		q0 -= delta
	}
	if delQ0 < q1 {
		delta := n
		if q1-delQ0 < delta {
			delta = q1 - delQ0
		}
		q1 -= delta
	}

	sm.state.SetSelection(q0, q1)
}
