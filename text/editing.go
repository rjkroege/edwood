// Package text provides the Text type and related components for edwood.
// This file contains editing-related types and methods for text.
package text

import "strings"

// EditingManager handles text editing operations.
// This extracts editing logic from the main Text type, including operations like:
// - Insert/Delete adjustments
// - Backspace and word/line erase
// - Type commit state management
// - Autoindent calculation
// - Tab expansion
type EditingManager struct {
	state     *EditState
	selection *SelectionState
	tabstop   int
	tabexpand bool
}

// NewEditingManager creates a new EditingManager with the given states.
func NewEditingManager(editState *EditState, selState *SelectionState) *EditingManager {
	if editState == nil {
		editState = NewEditState()
	}
	if selState == nil {
		selState = NewSelectionState()
	}
	return &EditingManager{
		state:     editState,
		selection: selState,
		tabstop:   4, // default tab stop
	}
}

// State returns the underlying EditState.
func (em *EditingManager) State() *EditState {
	return em.state
}

// Selection returns the underlying SelectionState.
func (em *EditingManager) Selection() *SelectionState {
	return em.selection
}

// IQ1 returns the initial q1 value (insertion point marker).
func (em *EditingManager) IQ1() int {
	return em.state.IQ1()
}

// SetIQ1 sets the initial q1 value.
func (em *EditingManager) SetIQ1(iq1 int) {
	em.state.SetIQ1(iq1)
}

// EQ0 returns the eq0 value (editing start marker).
func (em *EditingManager) EQ0() int {
	return em.state.EQ0()
}

// SetEQ0 sets the eq0 value.
func (em *EditingManager) SetEQ0(eq0 int) {
	em.state.SetEQ0(eq0)
}

// TypingStarted returns true if typing has started (eq0 == 0).
func (em *EditingManager) TypingStarted() bool {
	return em.state.EQ0() == 0
}

// ResetTyping resets the typing state to the sentinel value.
func (em *EditingManager) ResetTyping() {
	em.state.ResetTyping()
}

// TabStop returns the current tab stop value.
func (em *EditingManager) TabStop() int {
	return em.tabstop
}

// SetTabStop sets the tab stop value.
func (em *EditingManager) SetTabStop(tabstop int) {
	if tabstop < 1 {
		tabstop = 1
	}
	em.tabstop = tabstop
}

// TabExpand returns whether tabs should be expanded to spaces.
func (em *EditingManager) TabExpand() bool {
	return em.tabexpand
}

// SetTabExpand sets whether tabs should be expanded to spaces.
func (em *EditingManager) SetTabExpand(expand bool) {
	em.tabexpand = expand
}

// PrepareInsert prepares the editing state for an insert operation.
// Sets eq0 to q0 if this is the first insert (eq0 was sentinel).
func (em *EditingManager) PrepareInsert() {
	if em.state.EQ0() == ^0 {
		em.state.SetEQ0(em.selection.Q0())
	}
}

// AdjustForInsert adjusts the selection and iq1 after text is inserted.
// insertPos is where the insertion occurred, insertLen is the number of runes inserted.
func (em *EditingManager) AdjustForInsert(insertPos, insertLen int) {
	q0 := em.selection.Q0()
	q1 := em.selection.Q1()
	iq1 := em.state.IQ1()

	// Following text.go behavior:
	// if insertPos < q1, adjust q1
	// if insertPos < q0, adjust q0
	// if insertPos < iq1, adjust iq1
	if insertPos < iq1 {
		iq1 += insertLen
	}
	if insertPos < q1 {
		q1 += insertLen
	}
	if insertPos < q0 {
		q0 += insertLen
	}

	em.selection.SetSelection(q0, q1)
	em.state.SetIQ1(iq1)
}

// AdjustForDelete adjusts the selection and iq1 after text is deleted.
// delQ0 and delQ1 define the range that was deleted.
func (em *EditingManager) AdjustForDelete(delQ0, delQ1 int) {
	q0 := em.selection.Q0()
	q1 := em.selection.Q1()
	iq1 := em.state.IQ1()
	n := delQ1 - delQ0

	// min helper
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	// Following text.go behavior:
	// if delQ0 < iq1, adjust iq1 by min(n, iq1-delQ0)
	// if delQ0 < q0, adjust q0 by min(n, q0-delQ0)
	// if delQ0 < q1, adjust q1 by min(n, q1-delQ0)
	if delQ0 < iq1 {
		iq1 -= min(n, iq1-delQ0)
	}
	if delQ0 < q0 {
		q0 -= min(n, q0-delQ0)
	}
	if delQ0 < q1 {
		q1 -= min(n, q1-delQ0)
	}

	em.selection.SetSelection(q0, q1)
	em.state.SetIQ1(iq1)
}

// BsWidth calculates the number of characters to erase for a backspace operation.
// r is the control character:
//   - 0x08 (^H): erase one character
//   - 0x15 (^U): erase to beginning of line
//   - 0x17 (^W): erase word
//
// charReader returns the rune at a specific position (or 0 if out of bounds).
func (em *EditingManager) BsWidth(r rune, charReader func(pos int) rune) int {
	q0 := em.selection.Q0()
	if q0 == 0 {
		return 0
	}

	// ^H: erase one character
	if r == 0x08 {
		return 1
	}

	q := q0
	skipping := true

	for q > 0 {
		c := charReader(q - 1)
		if c == '\n' {
			// eat at most one more character
			if q == q0 {
				// eat the newline
				q--
			}
			break
		}
		if r == 0x17 { // ^W: erase word
			eq := isAlnum(c)
			if eq && skipping {
				// found alphanumeric; stop skipping
				skipping = false
			} else if !eq && !skipping {
				// found non-alphanumeric after word; stop
				break
			}
		}
		q--
	}

	return q0 - q
}

// CalculateDeleteRange calculates the range to delete for a backspace operation.
// Returns (delQ0, delQ1, adjusted) where adjusted is true if the range was
// clamped to the org (visible area boundary).
func (em *EditingManager) CalculateDeleteRange(r rune, org int, charReader func(pos int) rune) (delQ0, delQ1 int, adjusted bool) {
	q0 := em.selection.Q0()
	nnb := em.BsWidth(r, charReader)

	delQ1 = q0
	delQ0 = q0 - nnb

	// if selection is at beginning of window, avoid deleting invisible text
	if delQ0 < org {
		delQ0 = org
		adjusted = true
	}

	return delQ0, delQ1, adjusted
}

// CommitTyping commits the current typing session by resetting eq0.
func (em *EditingManager) CommitTyping() {
	em.state.ResetTyping()
}

// HasSelection returns true if there is a non-empty selection.
func (em *EditingManager) HasSelection() bool {
	return em.selection.HasSelection()
}

// InSelection returns true if pos is within the current selection.
func (em *EditingManager) InSelection(pos int) bool {
	q0 := em.selection.Q0()
	q1 := em.selection.Q1()
	return q1 > q0 && q0 <= pos && pos <= q1
}

// SelectToInsertionPoint selects from eq0 to q0 (or vice versa).
// This is used for the Escape key behavior to select typed text.
func (em *EditingManager) SelectToInsertionPoint() {
	eq0 := em.state.EQ0()
	q0 := em.selection.Q0()

	if eq0 == ^0 {
		// No insertion point set
		return
	}

	if eq0 <= q0 {
		em.selection.SetSelection(eq0, q0)
	} else {
		em.selection.SetSelection(q0, eq0)
	}
}

// CalculateAutoIndent calculates the indentation string for autoindent.
// It finds the beginning of the current line and returns the leading
// whitespace (tabs and spaces).
// charReader returns the rune at a specific position (or 0 if out of bounds).
func (em *EditingManager) CalculateAutoIndent(charReader func(pos int) rune) string {
	q0 := em.selection.Q0()
	if q0 == 0 {
		return ""
	}

	// Find beginning of line using ^U logic
	nnb := em.BsWidth(0x15, charReader)

	// Collect leading whitespace
	var indent strings.Builder
	for i := 0; i < nnb; i++ {
		r := charReader(q0 - nnb + i)
		if r != ' ' && r != '\t' {
			break
		}
		indent.WriteRune(r)
	}

	return indent.String()
}

// ExpandedTabWidth returns the number of spaces for tab expansion.
func (em *EditingManager) ExpandedTabWidth() int {
	return em.tabstop
}

// UpdateIQ1AfterPaste updates iq1 to q1 after a paste operation.
func (em *EditingManager) UpdateIQ1AfterPaste() {
	em.state.SetIQ1(em.selection.Q1())
}

// UpdateIQ1AfterCut updates iq1 to q0 after a cut operation.
func (em *EditingManager) UpdateIQ1AfterCut() {
	em.state.SetIQ1(em.selection.Q0())
}

// UpdateIQ1AfterType updates iq1 to q0 after regular typing.
func (em *EditingManager) UpdateIQ1AfterType() {
	em.state.SetIQ1(em.selection.Q0())
}

// FileWidth calculates the width of a filename/path element ending at q0.
// If oneElement is true, stops at path separators ('/').
// charReader returns the rune at a specific position (or 0 if out of bounds).
func (em *EditingManager) FileWidth(q0 int, oneElement bool, charReader func(pos int) rune) int {
	q := q0
	for q > 0 {
		r := charReader(q - 1)
		if r <= ' ' {
			break
		}
		if oneElement && r == '/' {
			break
		}
		q--
	}
	return q0 - q
}
