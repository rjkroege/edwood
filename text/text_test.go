// Package text provides the Text type and related components for edwood.
// This package contains text selection management, display methods,
// and editing operations.
package text

import (
	"testing"
)

// =============================================================================
// Tests for Range type
// =============================================================================

// TestRangeZeroValue tests that the zero value of Range is valid.
func TestRangeZeroValue(t *testing.T) {
	var r Range
	if r.Start != 0 || r.End != 0 {
		t.Errorf("zero Range should be (0, 0); got (%d, %d)", r.Start, r.End)
	}
}

// TestRangeIsEmpty tests the IsEmpty method.
func TestRangeIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		r     Range
		empty bool
	}{
		{"zero range", Range{0, 0}, true},
		{"empty at position", Range{10, 10}, true},
		{"non-empty forward", Range{10, 20}, false},
		{"non-empty backward", Range{20, 10}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.IsEmpty(); got != tc.empty {
				t.Errorf("IsEmpty() = %v; want %v", got, tc.empty)
			}
		})
	}
}

// TestRangeLen tests the Len method.
func TestRangeLen(t *testing.T) {
	tests := []struct {
		name string
		r    Range
		len  int
	}{
		{"zero range", Range{0, 0}, 0},
		{"forward range", Range{10, 20}, 10},
		{"backward range", Range{20, 10}, -10},
		{"single position", Range{5, 5}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.Len(); got != tc.len {
				t.Errorf("Len() = %d; want %d", got, tc.len)
			}
		})
	}
}

// TestRangeContains tests the Contains method.
func TestRangeContains(t *testing.T) {
	tests := []struct {
		name     string
		r        Range
		pos      int
		contains bool
	}{
		{"within forward range", Range{10, 20}, 15, true},
		{"at start of forward range", Range{10, 20}, 10, true},
		{"at end of forward range", Range{10, 20}, 20, false}, // end is exclusive
		{"before forward range", Range{10, 20}, 5, false},
		{"after forward range", Range{10, 20}, 25, false},
		{"within backward range", Range{20, 10}, 15, true},
		{"empty range", Range{10, 10}, 10, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.Contains(tc.pos); got != tc.contains {
				t.Errorf("Contains(%d) = %v; want %v", tc.pos, got, tc.contains)
			}
		})
	}
}

// TestRangeNormalize tests the Normalize method.
func TestRangeNormalize(t *testing.T) {
	tests := []struct {
		name   string
		input  Range
		output Range
	}{
		{"already normalized", Range{10, 20}, Range{10, 20}},
		{"needs normalization", Range{20, 10}, Range{10, 20}},
		{"empty range", Range{10, 10}, Range{10, 10}},
		{"zero range", Range{0, 0}, Range{0, 0}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.Normalize()
			if got.Start != tc.output.Start || got.End != tc.output.End {
				t.Errorf("Normalize() = (%d, %d); want (%d, %d)",
					got.Start, got.End, tc.output.Start, tc.output.End)
			}
		})
	}
}

// =============================================================================
// Tests for SelectionState
// =============================================================================

// TestSelectionStateNew tests that NewSelectionState creates a valid state.
func TestSelectionStateNew(t *testing.T) {
	s := NewSelectionState()
	if s == nil {
		t.Fatal("NewSelectionState returned nil")
	}
	if s.Q0() != 0 || s.Q1() != 0 {
		t.Errorf("new SelectionState should have (0, 0); got (%d, %d)", s.Q0(), s.Q1())
	}
}

// TestSelectionStateSetSelection tests setting the selection.
func TestSelectionStateSetSelection(t *testing.T) {
	s := NewSelectionState()
	s.SetSelection(10, 20)

	if s.Q0() != 10 || s.Q1() != 20 {
		t.Errorf("selection should be (10, 20); got (%d, %d)", s.Q0(), s.Q1())
	}

	sel := s.Selection()
	if sel.Start != 10 || sel.End != 20 {
		t.Errorf("Selection() should return (10, 20); got (%d, %d)", sel.Start, sel.End)
	}
}

// TestSelectionStateHasSelection tests the HasSelection method.
func TestSelectionStateHasSelection(t *testing.T) {
	s := NewSelectionState()

	if s.HasSelection() {
		t.Error("new SelectionState should not have selection")
	}

	s.SetSelection(10, 20)
	if !s.HasSelection() {
		t.Error("SelectionState should have selection after SetSelection(10, 20)")
	}

	s.SetSelection(15, 15)
	if s.HasSelection() {
		t.Error("SelectionState should not have selection when q0 == q1")
	}
}

// TestSelectionStateClearSelection tests clearing the selection.
func TestSelectionStateClearSelection(t *testing.T) {
	s := NewSelectionState()
	s.SetSelection(10, 20)
	s.ClearSelection()

	if s.Q0() != 10 {
		t.Errorf("after ClearSelection, Q0 should be 10; got %d", s.Q0())
	}
	if s.Q1() != 10 {
		t.Errorf("after ClearSelection, Q1 should equal Q0 (10); got %d", s.Q1())
	}
	if s.HasSelection() {
		t.Error("after ClearSelection, HasSelection should be false")
	}
}

// TestSelectionStateQ0Q1 tests the individual Q0/Q1 getters and setters.
func TestSelectionStateQ0Q1(t *testing.T) {
	s := NewSelectionState()

	s.SetQ0(5)
	if s.Q0() != 5 {
		t.Errorf("Q0 should be 5; got %d", s.Q0())
	}

	s.SetQ1(15)
	if s.Q1() != 15 {
		t.Errorf("Q1 should be 15; got %d", s.Q1())
	}

	// Selection should reflect individual changes
	sel := s.Selection()
	if sel.Start != 5 || sel.End != 15 {
		t.Errorf("Selection should be (5, 15); got (%d, %d)", sel.Start, sel.End)
	}
}

// =============================================================================
// Tests for DisplayState
// =============================================================================

// TestDisplayStateNew tests that NewDisplayState creates a valid state.
func TestDisplayStateNew(t *testing.T) {
	d := NewDisplayState()
	if d == nil {
		t.Fatal("NewDisplayState returned nil")
	}
	if d.Org() != 0 {
		t.Errorf("new DisplayState should have org=0; got %d", d.Org())
	}
	if d.NeedsRedraw() {
		t.Error("new DisplayState should not need redraw")
	}
}

// TestDisplayStateSetOrg tests setting the origin.
func TestDisplayStateSetOrg(t *testing.T) {
	d := NewDisplayState()

	d.SetOrg(100)
	if d.Org() != 100 {
		t.Errorf("org should be 100; got %d", d.Org())
	}

	// Setting org should trigger redraw
	if !d.NeedsRedraw() {
		t.Error("setting org should trigger redraw")
	}

	// Negative values should be clamped to 0
	d.ClearRedrawFlag()
	d.SetOrg(-10)
	if d.Org() != 0 {
		t.Errorf("negative org should be clamped to 0; got %d", d.Org())
	}
}

// TestDisplayStateRedrawFlag tests the redraw flag management.
func TestDisplayStateRedrawFlag(t *testing.T) {
	d := NewDisplayState()

	d.SetNeedsRedraw(true)
	if !d.NeedsRedraw() {
		t.Error("NeedsRedraw should be true after SetNeedsRedraw(true)")
	}

	d.ClearRedrawFlag()
	if d.NeedsRedraw() {
		t.Error("NeedsRedraw should be false after ClearRedrawFlag")
	}

	d.SetNeedsRedraw(false)
	if d.NeedsRedraw() {
		t.Error("NeedsRedraw should be false after SetNeedsRedraw(false)")
	}
}

// =============================================================================
// Tests for EditState
// =============================================================================

// TestEditStateNew tests that NewEditState creates a valid state.
func TestEditStateNew(t *testing.T) {
	e := NewEditState()
	if e == nil {
		t.Fatal("NewEditState returned nil")
	}

	// eq0 should be initialized to sentinel value (^0)
	if e.EQ0() != ^0 {
		t.Errorf("new EditState should have eq0=^0; got %d", e.EQ0())
	}

	// iq1 should be 0
	if e.IQ1() != 0 {
		t.Errorf("new EditState should have iq1=0; got %d", e.IQ1())
	}
}

// TestEditStateIQ1 tests the IQ1 getter and setter.
func TestEditStateIQ1(t *testing.T) {
	e := NewEditState()

	e.SetIQ1(50)
	if e.IQ1() != 50 {
		t.Errorf("IQ1 should be 50; got %d", e.IQ1())
	}
}

// TestEditStateTyping tests typing state management.
func TestEditStateTyping(t *testing.T) {
	e := NewEditState()

	// Initially, typing should not have started
	if e.TypingStarted() {
		t.Error("new EditState should not have typing started")
	}

	// Start typing by setting eq0 to 0
	e.SetEQ0(0)
	if !e.TypingStarted() {
		t.Error("TypingStarted should be true after SetEQ0(0)")
	}

	// Reset typing
	e.ResetTyping()
	if e.TypingStarted() {
		t.Error("TypingStarted should be false after ResetTyping")
	}
	if e.EQ0() != ^0 {
		t.Errorf("EQ0 should be ^0 after ResetTyping; got %d", e.EQ0())
	}
}

// =============================================================================
// Tests for TextBase
// =============================================================================

// TestTextBaseNew tests that NewTextBase creates a valid TextBase.
func TestTextBaseNew(t *testing.T) {
	tb := NewTextBase()
	if tb == nil {
		t.Fatal("NewTextBase returned nil")
	}

	// All state components should be initialized
	if tb.Selection == nil {
		t.Error("TextBase.Selection should not be nil")
	}
	if tb.Display == nil {
		t.Error("TextBase.Display should not be nil")
	}
	if tb.Edit == nil {
		t.Error("TextBase.Edit should not be nil")
	}

	// Default tabstop should be 4
	if tb.TabStop() != 4 {
		t.Errorf("default TabStop should be 4; got %d", tb.TabStop())
	}
}

// TestTextBaseTabStop tests tab stop management.
func TestTextBaseTabStop(t *testing.T) {
	tb := NewTextBase()

	tb.SetTabStop(8)
	if tb.TabStop() != 8 {
		t.Errorf("TabStop should be 8; got %d", tb.TabStop())
	}

	// Tab stop should be clamped to minimum of 1
	tb.SetTabStop(0)
	if tb.TabStop() != 1 {
		t.Errorf("TabStop should be clamped to 1; got %d", tb.TabStop())
	}

	tb.SetTabStop(-5)
	if tb.TabStop() != 1 {
		t.Errorf("negative TabStop should be clamped to 1; got %d", tb.TabStop())
	}
}

// TestTextBaseTabExpand tests tab expansion flag.
func TestTextBaseTabExpand(t *testing.T) {
	tb := NewTextBase()

	// Default should be false
	if tb.TabExpand() {
		t.Error("default TabExpand should be false")
	}

	tb.SetTabExpand(true)
	if !tb.TabExpand() {
		t.Error("TabExpand should be true after SetTabExpand(true)")
	}

	tb.SetTabExpand(false)
	if tb.TabExpand() {
		t.Error("TabExpand should be false after SetTabExpand(false)")
	}
}

// TestTextBaseNoFill tests nofill flag.
func TestTextBaseNoFill(t *testing.T) {
	tb := NewTextBase()

	// Default should be false
	if tb.NoFill() {
		t.Error("default NoFill should be false")
	}

	tb.SetNoFill(true)
	if !tb.NoFill() {
		t.Error("NoFill should be true after SetNoFill(true)")
	}
}

// TestTextBaseSelectionDelegation tests that selection methods delegate to SelectionState.
func TestTextBaseSelectionDelegation(t *testing.T) {
	tb := NewTextBase()

	tb.SetQ0(10)
	tb.SetQ1(20)

	if tb.Q0() != 10 {
		t.Errorf("Q0 should be 10; got %d", tb.Q0())
	}
	if tb.Q1() != 20 {
		t.Errorf("Q1 should be 20; got %d", tb.Q1())
	}

	// Verify delegation to SelectionState
	if tb.Selection.Q0() != 10 || tb.Selection.Q1() != 20 {
		t.Error("SelectionState should be updated via TextBase methods")
	}
}

// TestTextBaseOrgDelegation tests that org methods delegate to DisplayState.
func TestTextBaseOrgDelegation(t *testing.T) {
	tb := NewTextBase()

	tb.SetOrg(100)

	if tb.Org() != 100 {
		t.Errorf("Org should be 100; got %d", tb.Org())
	}

	// Verify delegation to DisplayState
	if tb.Display.Org() != 100 {
		t.Error("DisplayState should be updated via TextBase methods")
	}
}

// TestTextBaseIntegration tests TextBase in a realistic scenario.
func TestTextBaseIntegration(t *testing.T) {
	tb := NewTextBase()

	// Simulate setting up a text view
	tb.SetTabStop(4)
	tb.SetTabExpand(true)
	tb.SetOrg(0)
	tb.SetQ0(0)
	tb.SetQ1(0)

	// Simulate selecting some text
	tb.SetQ0(10)
	tb.SetQ1(50)

	if tb.Selection.Q0() != 10 || tb.Selection.Q1() != 50 {
		t.Errorf("selection should be (10, 50); got (%d, %d)",
			tb.Selection.Q0(), tb.Selection.Q1())
	}

	// Simulate scrolling
	tb.SetOrg(100)
	if tb.Org() != 100 {
		t.Errorf("org should be 100; got %d", tb.Org())
	}
	if !tb.Display.NeedsRedraw() {
		t.Error("changing org should trigger redraw")
	}

	// Simulate clearing selection
	tb.Selection.ClearSelection()
	if tb.Selection.HasSelection() {
		t.Error("selection should be cleared")
	}
}

// =============================================================================
// Tests for SelectionManager (Phase 6B stubs)
// =============================================================================

// TestSelectionManagerNew tests that NewSelectionManager creates a valid manager.
func TestSelectionManagerNew(t *testing.T) {
	sm := NewSelectionManager(nil)
	if sm == nil {
		t.Fatal("NewSelectionManager(nil) returned nil")
	}

	// Should create its own state
	if sm.State() == nil {
		t.Error("SelectionManager should have non-nil state")
	}

	// Test with provided state
	state := NewSelectionState()
	state.SetSelection(10, 20)
	sm = NewSelectionManager(state)
	if sm.State() != state {
		t.Error("SelectionManager should use provided state")
	}
	if sm.Selection().Start != 10 || sm.Selection().End != 20 {
		t.Error("SelectionManager should reflect provided state")
	}
}

// TestSelectionManagerBasicOps tests basic selection operations.
func TestSelectionManagerBasicOps(t *testing.T) {
	sm := NewSelectionManager(nil)

	// Set selection
	sm.SetSelection(10, 20)
	sel := sm.Selection()
	if sel.Start != 10 || sel.End != 20 {
		t.Errorf("selection should be (10, 20); got (%d, %d)", sel.Start, sel.End)
	}

	// Has selection
	if !sm.HasSelection() {
		t.Error("should have selection")
	}

	// Clear selection
	sm.ClearSelection()
	if sm.HasSelection() {
		t.Error("should not have selection after clear")
	}
}

// TestSelectionManagerClampSelection tests clamping selection to bounds.
func TestSelectionManagerClampSelection(t *testing.T) {
	tests := []struct {
		name    string
		initial Range
		maxLen  int
		clamped Range
	}{
		{"within bounds", Range{10, 20}, 100, Range{10, 20}},
		{"q1 beyond max", Range{10, 150}, 100, Range{10, 100}},
		{"both beyond max", Range{150, 200}, 100, Range{100, 100}},
		{"negative q0", Range{-5, 20}, 100, Range{0, 20}},
		{"negative q1", Range{10, -5}, 100, Range{10, 0}},
		{"both negative", Range{-10, -5}, 100, Range{0, 0}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			sm.SetSelection(tc.initial.Start, tc.initial.End)
			result := sm.ClampSelection(tc.maxLen)

			if result.Start != tc.clamped.Start || result.End != tc.clamped.End {
				t.Errorf("ClampSelection(%d) = (%d, %d); want (%d, %d)",
					tc.maxLen, result.Start, result.End,
					tc.clamped.Start, tc.clamped.End)
			}
		})
	}
}

// TestSelectionManagerExpandStubs tests that expand methods don't panic.
// These are stubs for Phase 6B.
func TestSelectionManagerExpandStubs(t *testing.T) {
	sm := NewSelectionManager(nil)

	// These should not panic and return reasonable values
	wordRange := sm.ExpandToWord(10, nil)
	if wordRange.Start != 10 || wordRange.End != 10 {
		t.Errorf("ExpandToWord stub should return (pos, pos); got (%d, %d)",
			wordRange.Start, wordRange.End)
	}

	lineRange := sm.ExpandToLine(10, nil)
	if lineRange.Start != 10 || lineRange.End != 10 {
		t.Errorf("ExpandToLine stub should return (pos, pos); got (%d, %d)",
			lineRange.Start, lineRange.End)
	}
}

// =============================================================================
// Tests for all types to be independent
// =============================================================================

// TestAllTypesIndependent verifies that all state types can be used independently.
func TestAllTypesIndependent(t *testing.T) {
	// Create all types independently
	ss := NewSelectionState()
	ds := NewDisplayState()
	es := NewEditState()
	tb := NewTextBase()
	sm := NewSelectionManager(nil)

	// Each should be non-nil
	if ss == nil || ds == nil || es == nil || tb == nil || sm == nil {
		t.Fatal("all types should be created successfully")
	}

	// Modify each independently
	ss.SetSelection(1, 2)
	ds.SetOrg(10)
	es.SetIQ1(20)
	tb.SetTabStop(8)
	sm.SetSelection(30, 40)

	// Verify independence
	if ss.Q0() != 1 || ss.Q1() != 2 {
		t.Error("SelectionState should be modified independently")
	}
	if ds.Org() != 10 {
		t.Error("DisplayState should be modified independently")
	}
	if es.IQ1() != 20 {
		t.Error("EditState should be modified independently")
	}
	if tb.TabStop() != 8 {
		t.Error("TextBase should be modified independently")
	}
	sel := sm.Selection()
	if sel.Start != 30 || sel.End != 40 {
		t.Error("SelectionManager should be modified independently")
	}
}
