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

// TestSelectionManagerExpandToWord tests word expansion behavior.
func TestSelectionManagerExpandToWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      int
		wantQ0   int
		wantQ1   int
	}{
		{"middle of word", "hello world", 2, 0, 5},       // pos in "hello" -> select "hello"
		{"start of word", "hello world", 0, 0, 5},        // pos at 'h' -> select "hello"
		{"end of word", "hello world", 5, 5, 5},          // pos at space -> no word
		{"second word", "hello world", 8, 6, 11},         // pos in "world" -> select "world"
		{"single char word", "a b c", 2, 2, 3},           // pos at 'b' -> select "b"
		{"numbers", "abc123def", 5, 0, 9},                // pos in "123" -> select "abc123def" (alnum)
		{"underscore", "foo_bar", 4, 0, 7},               // underscore is not alnum, splits word
		{"empty at start", "hello", 0, 0, 5},             // at start of only word
		{"empty string", "", 0, 0, 0},                    // empty string
		{"only spaces", "   ", 1, 1, 1},                  // only spaces, no word
		{"word at end", "hello", 4, 0, 5},                // last char of word
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			textRunes := []rune(tc.text)
			reader := func(start, end int) []rune {
				if start < 0 {
					start = 0
				}
				if end > len(textRunes) {
					end = len(textRunes)
				}
				if start >= end {
					return nil
				}
				return textRunes[start:end]
			}
			result := sm.ExpandToWord(tc.pos, reader)
			if result.Start != tc.wantQ0 || result.End != tc.wantQ1 {
				t.Errorf("ExpandToWord(%d) in %q = (%d, %d); want (%d, %d)",
					tc.pos, tc.text, result.Start, result.End, tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestSelectionManagerExpandToLine tests line expansion behavior.
func TestSelectionManagerExpandToLine(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      int
		wantQ0   int
		wantQ1   int
	}{
		{"middle of line", "hello\nworld\n", 2, 0, 5},     // pos in "hello" -> select "hello" (not newline)
		{"second line", "hello\nworld\n", 8, 6, 11},       // pos in "world" -> select "world"
		{"at newline", "hello\nworld\n", 5, 0, 5},         // pos at '\n' -> select line before
		{"start of line", "hello\nworld", 6, 6, 11},       // pos at 'w' -> select "world"
		{"single line", "hello", 2, 0, 5},                 // no newlines
		{"empty string", "", 0, 0, 0},                     // empty string
		{"only newline", "\n", 0, 0, 0},                   // just a newline
		{"empty lines", "\n\n\n", 1, 1, 1},                // between newlines
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			textRunes := []rune(tc.text)
			reader := func(start, end int) []rune {
				if start < 0 {
					start = 0
				}
				if end > len(textRunes) {
					end = len(textRunes)
				}
				if start >= end {
					return nil
				}
				return textRunes[start:end]
			}
			result := sm.ExpandToLine(tc.pos, reader)
			if result.Start != tc.wantQ0 || result.End != tc.wantQ1 {
				t.Errorf("ExpandToLine(%d) in %q = (%d, %d); want (%d, %d)",
					tc.pos, tc.text, result.Start, result.End, tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestSelectionManagerExpandBrackets tests bracket matching expansion.
func TestSelectionManagerExpandBrackets(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      int
		wantQ0   int
		wantQ1   int
	}{
		{"parens", "(hello)", 1, 1, 6},           // inside parens -> content
		{"nested parens", "((a))", 1, 1, 4},      // outer to inner closing
		{"braces", "{hello}", 1, 1, 6},           // inside braces
		{"brackets", "[hello]", 1, 1, 6},         // inside brackets
		{"angle brackets", "<hello>", 1, 1, 6},   // inside angle brackets
		{"quotes", "'hello'", 1, 1, 6},           // inside single quotes
		{"double quotes", "\"hello\"", 1, 1, 6},  // inside double quotes
		{"backticks", "`hello`", 1, 1, 6},        // inside backticks
		{"guillemets", "«hello»", 1, 1, 6}, // inside « »
		{"no match", "hello", 2, 0, 5},           // no brackets, fall back to word
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			textRunes := []rune(tc.text)
			reader := func(start, end int) []rune {
				if start < 0 {
					start = 0
				}
				if end > len(textRunes) {
					end = len(textRunes)
				}
				if start >= end {
					return nil
				}
				return textRunes[start:end]
			}
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}
			result := sm.ExpandToBrackets(tc.pos, len(textRunes), reader, charReader)
			if result.Start != tc.wantQ0 || result.End != tc.wantQ1 {
				t.Errorf("ExpandToBrackets(%d) in %q = (%d, %d); want (%d, %d)",
					tc.pos, tc.text, result.Start, result.End, tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestSelectionManagerInSelection tests the InSelection method.
func TestSelectionManagerInSelection(t *testing.T) {
	tests := []struct {
		name       string
		q0, q1     int
		pos        int
		wantInSel  bool
	}{
		{"empty selection", 10, 10, 10, false},
		{"before selection", 10, 20, 5, false},
		{"at start", 10, 20, 10, true},
		{"inside", 10, 20, 15, true},
		{"at end", 10, 20, 20, true},
		{"after selection", 10, 20, 25, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			sm.SetSelection(tc.q0, tc.q1)
			result := sm.InSelection(tc.pos)
			if result != tc.wantInSel {
				t.Errorf("InSelection(%d) with selection (%d, %d) = %v; want %v",
					tc.pos, tc.q0, tc.q1, result, tc.wantInSel)
			}
		})
	}
}

// TestSelectionManagerConstrain tests the Constrain method.
func TestSelectionManagerConstrain(t *testing.T) {
	tests := []struct {
		name       string
		q0, q1     int
		maxLen     int
		wantQ0     int
		wantQ1     int
	}{
		{"within bounds", 10, 20, 100, 10, 20},
		{"q0 beyond max", 150, 200, 100, 100, 100},
		{"q1 beyond max", 50, 200, 100, 50, 100},
		{"both beyond max", 150, 200, 100, 100, 100},
		{"at boundary", 100, 100, 100, 100, 100},
		{"zero max", 10, 20, 0, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			sm.SetSelection(tc.q0, tc.q1)
			p0, p1 := sm.Constrain(tc.maxLen)
			if p0 != tc.wantQ0 || p1 != tc.wantQ1 {
				t.Errorf("Constrain(%d) with selection (%d, %d) = (%d, %d); want (%d, %d)",
					tc.maxLen, tc.q0, tc.q1, p0, p1, tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestSelectionManagerAdjustForInsert tests selection adjustment after text insertion.
// This matches the behavior in text.go's Inserted method:
// - if insertPos < q1, adjust q1
// - if insertPos < q0, adjust q0
func TestSelectionManagerAdjustForInsert(t *testing.T) {
	tests := []struct {
		name       string
		q0, q1     int
		insertPos  int
		insertLen  int
		wantQ0     int
		wantQ1     int
	}{
		{"insert before selection", 10, 20, 5, 3, 13, 23},   // 5 < 10 and 5 < 20, both adjust
		{"insert at selection start", 10, 20, 10, 3, 10, 23}, // 10 < 20 (q1 adjusts), 10 < 10 false (q0 stays)
		{"insert inside selection", 10, 20, 15, 3, 10, 23},   // 15 < 20 (q1 adjusts), 15 < 10 false (q0 stays)
		{"insert at selection end", 10, 20, 20, 3, 10, 20},   // 20 < 20 false (q1 stays), 20 < 10 false (q0 stays)
		{"insert after selection", 10, 20, 25, 3, 10, 20},    // nothing adjusts
		{"insert at cursor (no selection)", 10, 10, 10, 3, 10, 10}, // nothing adjusts
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			sm.SetSelection(tc.q0, tc.q1)
			sm.AdjustForInsert(tc.insertPos, tc.insertLen)
			sel := sm.Selection()
			if sel.Start != tc.wantQ0 || sel.End != tc.wantQ1 {
				t.Errorf("AdjustForInsert(%d, %d) with selection (%d, %d) = (%d, %d); want (%d, %d)",
					tc.insertPos, tc.insertLen, tc.q0, tc.q1,
					sel.Start, sel.End, tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestSelectionManagerAdjustForDelete tests selection adjustment after text deletion.
// This matches the behavior in text.go's Deleted method:
// - if delQ0 < q0, adjust q0 by min(n, q0-delQ0)
// - if delQ0 < q1, adjust q1 by min(n, q1-delQ0)
func TestSelectionManagerAdjustForDelete(t *testing.T) {
	tests := []struct {
		name       string
		q0, q1     int
		delQ0      int
		delQ1      int
		wantQ0     int
		wantQ1     int
	}{
		{"delete before selection", 10, 20, 2, 5, 7, 17},           // n=3, q0-=3, q1-=3
		{"delete overlapping start", 10, 20, 5, 15, 5, 10},         // n=10, q0-=min(10,5)=5, q1-=min(10,15)=10
		{"delete inside selection", 10, 20, 12, 15, 10, 17},        // n=3, delQ0(12)<q0(10) false, q1-=min(3,8)=3
		{"delete overlapping end", 10, 20, 15, 25, 10, 15},         // n=10, delQ0(15)<q0(10) false, q1-=min(10,5)=5
		{"delete containing selection", 10, 20, 5, 25, 5, 5},       // n=20, q0-=min(20,5)=5, q1-=min(20,15)=15
		{"delete after selection", 10, 20, 25, 30, 10, 20},         // delQ0(25)<10 false, delQ0<20 false
		{"delete at cursor (no selection)", 10, 10, 5, 8, 7, 7},    // n=3, q0-=min(3,5)=3, q1-=min(3,5)=3
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewSelectionManager(nil)
			sm.SetSelection(tc.q0, tc.q1)
			sm.AdjustForDelete(tc.delQ0, tc.delQ1)
			sel := sm.Selection()
			if sel.Start != tc.wantQ0 || sel.End != tc.wantQ1 {
				t.Errorf("AdjustForDelete(%d, %d) with selection (%d, %d) = (%d, %d); want (%d, %d)",
					tc.delQ0, tc.delQ1, tc.q0, tc.q1,
					sel.Start, sel.End, tc.wantQ0, tc.wantQ1)
			}
		})
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

// =============================================================================
// Tests for DisplayManager (Phase 6C)
// =============================================================================

// TestDisplayManagerNew tests that NewDisplayManager creates a valid manager.
func TestDisplayManagerNew(t *testing.T) {
	dm := NewDisplayManager(nil)
	if dm == nil {
		t.Fatal("NewDisplayManager(nil) returned nil")
	}

	// Should create its own state
	if dm.State() == nil {
		t.Error("DisplayManager should have non-nil state")
	}

	// Test with provided state
	state := NewDisplayState()
	state.SetOrg(100)
	dm = NewDisplayManager(state)
	if dm.State() != state {
		t.Error("DisplayManager should use provided state")
	}
	if dm.Org() != 100 {
		t.Error("DisplayManager should reflect provided state's org")
	}
}

// TestDisplayManagerBasicOps tests basic display operations.
func TestDisplayManagerBasicOps(t *testing.T) {
	dm := NewDisplayManager(nil)

	// Initial state
	if dm.Org() != 0 {
		t.Errorf("initial org should be 0; got %d", dm.Org())
	}
	if dm.NeedsRedraw() {
		t.Error("initial state should not need redraw")
	}

	// Set org
	dm.SetOrg(100)
	if dm.Org() != 100 {
		t.Errorf("org should be 100; got %d", dm.Org())
	}
	if !dm.NeedsRedraw() {
		t.Error("setting org should trigger redraw")
	}

	// Clear redraw
	dm.ClearRedrawFlag()
	if dm.NeedsRedraw() {
		t.Error("redraw flag should be cleared")
	}

	// Set needs redraw explicitly
	dm.SetNeedsRedraw(true)
	if !dm.NeedsRedraw() {
		t.Error("should need redraw after SetNeedsRedraw(true)")
	}
}

// TestDisplayManagerOrgClamping tests that negative org values are clamped.
func TestDisplayManagerOrgClamping(t *testing.T) {
	dm := NewDisplayManager(nil)

	dm.SetOrg(-10)
	if dm.Org() != 0 {
		t.Errorf("negative org should be clamped to 0; got %d", dm.Org())
	}
}

// TestDisplayManagerCalculateVisibleRange tests visible range calculation.
func TestDisplayManagerCalculateVisibleRange(t *testing.T) {
	tests := []struct {
		name   string
		org    int
		nchars int
		want   Range
	}{
		{"zero org", 0, 100, Range{0, 100}},
		{"non-zero org", 50, 100, Range{50, 150}},
		{"zero nchars", 100, 0, Range{100, 100}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			got := dm.CalculateVisibleRange(tc.nchars)
			if got.Start != tc.want.Start || got.End != tc.want.End {
				t.Errorf("CalculateVisibleRange(%d) = (%d, %d); want (%d, %d)",
					tc.nchars, got.Start, got.End, tc.want.Start, tc.want.End)
			}
		})
	}
}

// TestDisplayManagerIsPositionVisible tests position visibility checking.
func TestDisplayManagerIsPositionVisible(t *testing.T) {
	tests := []struct {
		name    string
		org     int
		nchars  int
		pos     int
		visible bool
	}{
		{"at start", 100, 50, 100, true},
		{"in middle", 100, 50, 125, true},
		{"at end (exclusive)", 100, 50, 150, false},
		{"before start", 100, 50, 99, false},
		{"after end", 100, 50, 151, false},
		{"zero org visible", 0, 100, 50, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			got := dm.IsPositionVisible(tc.pos, tc.nchars)
			if got != tc.visible {
				t.Errorf("IsPositionVisible(%d, %d) with org=%d = %v; want %v",
					tc.pos, tc.nchars, tc.org, got, tc.visible)
			}
		})
	}
}

// TestDisplayManagerIsRangeVisible tests range visibility checking.
func TestDisplayManagerIsRangeVisible(t *testing.T) {
	tests := []struct {
		name    string
		org     int
		nchars  int
		r       Range
		visible bool
	}{
		{"fully visible", 100, 50, Range{110, 140}, true},
		{"overlaps start", 100, 50, Range{80, 120}, true},
		{"overlaps end", 100, 50, Range{140, 160}, true},
		{"contains visible", 100, 50, Range{80, 180}, true},
		{"before visible", 100, 50, Range{50, 90}, false},
		{"after visible", 100, 50, Range{160, 180}, false},
		{"at end boundary", 100, 50, Range{150, 160}, false},
		{"at start boundary", 100, 50, Range{90, 100}, false},
		{"empty range at start", 100, 50, Range{100, 100}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			got := dm.IsRangeVisible(tc.r, tc.nchars)
			if got != tc.visible {
				t.Errorf("IsRangeVisible((%d, %d), %d) with org=%d = %v; want %v",
					tc.r.Start, tc.r.End, tc.nchars, tc.org, got, tc.visible)
			}
		})
	}
}

// TestDisplayManagerIsRangeFullyVisible tests full range visibility checking.
func TestDisplayManagerIsRangeFullyVisible(t *testing.T) {
	tests := []struct {
		name    string
		org     int
		nchars  int
		r       Range
		visible bool
	}{
		{"fully visible", 100, 50, Range{110, 140}, true},
		{"at boundaries", 100, 50, Range{100, 150}, true},
		{"overlaps start", 100, 50, Range{80, 120}, false},
		{"overlaps end", 100, 50, Range{140, 160}, false},
		{"extends both ends", 100, 50, Range{80, 180}, false},
		{"empty at start", 100, 50, Range{100, 100}, true},
		{"empty in middle", 100, 50, Range{125, 125}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			got := dm.IsRangeFullyVisible(tc.r, tc.nchars)
			if got != tc.visible {
				t.Errorf("IsRangeFullyVisible((%d, %d), %d) with org=%d = %v; want %v",
					tc.r.Start, tc.r.End, tc.nchars, tc.org, got, tc.visible)
			}
		})
	}
}

// TestDisplayManagerPositionToFrameOffset tests position to offset conversion.
func TestDisplayManagerPositionToFrameOffset(t *testing.T) {
	tests := []struct {
		name   string
		org    int
		pos    int
		offset int
	}{
		{"at origin", 100, 100, 0},
		{"after origin", 100, 150, 50},
		{"before origin", 100, 50, -1},
		{"zero org", 0, 50, 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			got := dm.PositionToFrameOffset(tc.pos)
			if got != tc.offset {
				t.Errorf("PositionToFrameOffset(%d) with org=%d = %d; want %d",
					tc.pos, tc.org, got, tc.offset)
			}
		})
	}
}

// TestDisplayManagerFrameOffsetToPosition tests offset to position conversion.
func TestDisplayManagerFrameOffsetToPosition(t *testing.T) {
	tests := []struct {
		name   string
		org    int
		offset int
		pos    int
	}{
		{"zero offset", 100, 0, 100},
		{"positive offset", 100, 50, 150},
		{"zero org", 0, 50, 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			got := dm.FrameOffsetToPosition(tc.offset)
			if got != tc.pos {
				t.Errorf("FrameOffsetToPosition(%d) with org=%d = %d; want %d",
					tc.offset, tc.org, got, tc.pos)
			}
		})
	}
}

// TestDisplayManagerBackNL tests backing up by newlines.
func TestDisplayManagerBackNL(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		startPos   int
		nLines     int
		wantPos    int
		maxLineLen int
	}{
		{"back 0 lines from line start", "hello\nworld\n", 6, 0, 6, 128},
		{"back 0 lines from mid line", "hello\nworld\n", 8, 0, 6, 128},
		{"back 1 line", "hello\nworld\n", 12, 1, 6, 128},
		{"back 2 lines", "hello\nworld\nfoo\n", 16, 2, 6, 128},
		{"back from first line", "hello\nworld\n", 3, 1, 0, 128},
		{"back beyond start", "hello\nworld\n", 3, 5, 0, 128},
		{"empty text", "", 0, 1, 0, 128},
		{"single char", "a", 1, 0, 0, 128},
		{"long line truncated", "a" + string(make([]rune, 200)) + "\nb", 202, 1, 73, 128},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			got := dm.BackNL(tc.startPos, tc.nLines, charReader, tc.maxLineLen)
			if got != tc.wantPos {
				t.Errorf("BackNL(%d, %d) in %q = %d; want %d",
					tc.startPos, tc.nLines, tc.text, got, tc.wantPos)
			}
		})
	}
}

// TestDisplayManagerAdjustOriginForExact tests origin adjustment for line boundaries.
func TestDisplayManagerAdjustOriginForExact(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		org     int
		exact   bool
		wantOrg int
	}{
		{"exact mode no change", "hello\nworld\n", 3, true, 3},
		{"at line start", "hello\nworld\n", 6, false, 6},
		{"mid line adjusts forward", "hello\nworld\n", 3, false, 6},
		{"zero org unchanged", "hello\nworld\n", 0, false, 0},
		{"negative org unchanged", "hello\nworld\n", -1, false, -1},
		{"at newline", "hello\nworld\n", 5, false, 6},
		{"no newline within 256", "a" + string(make([]rune, 300)), 1, false, 257},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			got := dm.AdjustOriginForExact(tc.org, tc.exact, charReader, len(textRunes))
			if got != tc.wantOrg {
				t.Errorf("AdjustOriginForExact(%d, %v) in %q = %d; want %d",
					tc.org, tc.exact, tc.text, got, tc.wantOrg)
			}
		})
	}
}

// TestDisplayManagerCalculateNewOrigin tests new origin calculation for scrolling.
func TestDisplayManagerCalculateNewOrigin(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		org           int
		targetPos     int
		nchars        int
		maxlines      int
		quarterScroll bool
		wantOrg       int
	}{
		// Target already visible - no change
		{"target visible", "hello\nworld\nfoo\nbar\n", 0, 5, 20, 4, false, 0},
		// Target at boundary but end of file
		{"target at end of file", "hello", 0, 5, 5, 4, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)
			dm.SetOrg(tc.org)
			dm.ClearRedrawFlag()

			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			got := dm.CalculateNewOrigin(tc.targetPos, tc.nchars, tc.maxlines, tc.quarterScroll, charReader, len(textRunes))
			if got != tc.wantOrg {
				t.Errorf("CalculateNewOrigin(%d, %d, %d, %v) with org=%d in %q = %d; want %d",
					tc.targetPos, tc.nchars, tc.maxlines, tc.quarterScroll, tc.org, tc.text, got, tc.wantOrg)
			}
		})
	}
}

// TestDisplayManagerScrollDelta tests scroll delta calculation.
func TestDisplayManagerScrollDelta(t *testing.T) {
	tests := []struct {
		name    string
		org     int
		delta   int
		nchars  int
		textLen int
		wantOrg int
	}{
		{"zero delta", 100, 0, 50, 200, 100},
		{"at end of file", 150, 1, 50, 200, 150},
		{"scroll down needs calc", 100, 1, 50, 200, -1},
		{"scroll up needs BackNL", 100, -1, 50, 200, -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := NewDisplayManager(nil)

			got := dm.ScrollDelta(tc.org, tc.delta, tc.nchars, tc.textLen)
			if got != tc.wantOrg {
				t.Errorf("ScrollDelta(%d, %d, %d, %d) = %d; want %d",
					tc.org, tc.delta, tc.nchars, tc.textLen, got, tc.wantOrg)
			}
		})
	}
}

// TestDisplayManagerRoundTrip tests that frame offset conversions are inverses.
func TestDisplayManagerRoundTrip(t *testing.T) {
	dm := NewDisplayManager(nil)
	dm.SetOrg(100)
	dm.ClearRedrawFlag()

	// Position -> Offset -> Position
	origPos := 150
	offset := dm.PositionToFrameOffset(origPos)
	if offset < 0 {
		t.Fatalf("position %d should have non-negative offset", origPos)
	}
	roundTrip := dm.FrameOffsetToPosition(offset)
	if roundTrip != origPos {
		t.Errorf("round trip: %d -> %d -> %d (expected %d)",
			origPos, offset, roundTrip, origPos)
	}
}

// TestDisplayManagerIntegration tests DisplayManager in a realistic scenario.
func TestDisplayManagerIntegration(t *testing.T) {
	dm := NewDisplayManager(nil)

	// Simulate initial setup
	dm.SetOrg(0)
	if !dm.NeedsRedraw() {
		t.Error("initial SetOrg should trigger redraw")
	}
	dm.ClearRedrawFlag()

	// Simulate scrolling down
	dm.SetOrg(100)
	if !dm.NeedsRedraw() {
		t.Error("scrolling should trigger redraw")
	}
	dm.ClearRedrawFlag()

	// Check visibility with 50 chars visible
	nchars := 50
	if !dm.IsPositionVisible(100, nchars) {
		t.Error("position at org should be visible")
	}
	if !dm.IsPositionVisible(149, nchars) {
		t.Error("position just before end should be visible")
	}
	if dm.IsPositionVisible(150, nchars) {
		t.Error("position at end should not be visible")
	}

	// Test range visibility
	r := Range{Start: 90, End: 110}
	if !dm.IsRangeVisible(r, nchars) {
		t.Error("overlapping range should be visible")
	}
	if dm.IsRangeFullyVisible(r, nchars) {
		t.Error("partially overlapping range should not be fully visible")
	}

	// Test frame offset conversion
	offset := dm.PositionToFrameOffset(125)
	if offset != 25 {
		t.Errorf("offset should be 25; got %d", offset)
	}
}
