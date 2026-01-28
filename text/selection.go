// Package text provides the Text type and related components for edwood.
package text

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

// ExpandToWord will expand the selection at the given position to include
// the entire word. This is a stub for Phase 6B.
func (sm *SelectionManager) ExpandToWord(pos int, textReader func(start, end int) []rune) Range {
	// Stub implementation - will be filled in Phase 6B
	return Range{Start: pos, End: pos}
}

// ExpandToLine will expand the selection at the given position to include
// the entire line. This is a stub for Phase 6B.
func (sm *SelectionManager) ExpandToLine(pos int, textReader func(start, end int) []rune) Range {
	// Stub implementation - will be filled in Phase 6B
	return Range{Start: pos, End: pos}
}
