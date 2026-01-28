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

// =============================================================================
// Tests for EditingManager (Phase 6D)
// =============================================================================

// TestEditingManagerNew tests that NewEditingManager creates a valid manager.
func TestEditingManagerNew(t *testing.T) {
	em := NewEditingManager(nil, nil)
	if em == nil {
		t.Fatal("NewEditingManager(nil, nil) returned nil")
	}

	// Should create its own state
	if em.State() == nil {
		t.Error("EditingManager should have non-nil edit state")
	}
	if em.Selection() == nil {
		t.Error("EditingManager should have non-nil selection state")
	}

	// Test with provided states
	editState := NewEditState()
	editState.SetIQ1(100)
	selState := NewSelectionState()
	selState.SetSelection(10, 20)

	em = NewEditingManager(editState, selState)
	if em.State() != editState {
		t.Error("EditingManager should use provided edit state")
	}
	if em.Selection() != selState {
		t.Error("EditingManager should use provided selection state")
	}
	if em.IQ1() != 100 {
		t.Errorf("EditingManager should reflect provided state's IQ1; got %d", em.IQ1())
	}
}

// TestEditingManagerBasicOps tests basic editing state operations.
func TestEditingManagerBasicOps(t *testing.T) {
	em := NewEditingManager(nil, nil)

	// Initial state
	if em.IQ1() != 0 {
		t.Errorf("initial IQ1 should be 0; got %d", em.IQ1())
	}
	if em.EQ0() != ^0 {
		t.Errorf("initial EQ0 should be ^0; got %d", em.EQ0())
	}
	if em.TypingStarted() {
		t.Error("typing should not have started initially")
	}

	// Set IQ1
	em.SetIQ1(50)
	if em.IQ1() != 50 {
		t.Errorf("IQ1 should be 50; got %d", em.IQ1())
	}

	// Start typing
	em.SetEQ0(0)
	if !em.TypingStarted() {
		t.Error("typing should have started after SetEQ0(0)")
	}

	// Reset typing
	em.ResetTyping()
	if em.TypingStarted() {
		t.Error("typing should not have started after ResetTyping")
	}
}

// TestEditingManagerPrepareInsert tests preparation for text insertion.
func TestEditingManagerPrepareInsert(t *testing.T) {
	tests := []struct {
		name       string
		initialQ0  int
		initialQ1  int
		initialEQ0 int
		wantEQ0    int
	}{
		{"first insert starts typing", 10, 10, ^0, 10},
		{"subsequent insert preserves eq0", 15, 15, 10, 10},
		{"insert with selection", 10, 20, ^0, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.Selection().SetSelection(tc.initialQ0, tc.initialQ1)
			em.State().SetEQ0(tc.initialEQ0)

			em.PrepareInsert()

			if em.EQ0() != tc.wantEQ0 {
				t.Errorf("EQ0 after PrepareInsert = %d; want %d", em.EQ0(), tc.wantEQ0)
			}
		})
	}
}

// TestEditingManagerAdjustForInsert tests selection adjustment after insert.
// This matches the behavior in text.go's Inserted method where:
// - if insertPos < iq1, adjust iq1
// - if insertPos < q1, adjust q1
// - if insertPos < q0, adjust q0
func TestEditingManagerAdjustForInsert(t *testing.T) {
	tests := []struct {
		name      string
		q0        int
		q1        int
		iq1       int
		insertPos int
		insertLen int
		wantQ0    int
		wantQ1    int
		wantIQ1   int
	}{
		{"insert before all", 10, 20, 15, 5, 3, 13, 23, 18},       // 5 < 10, 15, 20 - all adjust
		{"insert at q0", 10, 20, 15, 10, 3, 10, 23, 18},           // 10 < 15, 20 but not < 10 - q1, iq1 adjust
		{"insert in selection", 10, 20, 15, 15, 3, 10, 23, 15},    // 15 < 20 but not < 10, 15 - only q1 adjusts
		{"insert after selection", 10, 20, 15, 25, 3, 10, 20, 15}, // 25 not < any - nothing adjusts
		{"insert at iq1", 10, 20, 15, 15, 3, 10, 23, 15},          // 15 < 20 but not < 15 - only q1 adjusts
		{"insert after iq1", 10, 20, 15, 20, 3, 10, 20, 15},       // 20 not < any - nothing adjusts
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.Selection().SetSelection(tc.q0, tc.q1)
			em.SetIQ1(tc.iq1)

			em.AdjustForInsert(tc.insertPos, tc.insertLen)

			if em.Selection().Q0() != tc.wantQ0 || em.Selection().Q1() != tc.wantQ1 {
				t.Errorf("selection after insert = (%d, %d); want (%d, %d)",
					em.Selection().Q0(), em.Selection().Q1(), tc.wantQ0, tc.wantQ1)
			}
			if em.IQ1() != tc.wantIQ1 {
				t.Errorf("IQ1 after insert = %d; want %d", em.IQ1(), tc.wantIQ1)
			}
		})
	}
}

// TestEditingManagerAdjustForDelete tests selection adjustment after delete.
// This matches the behavior in text.go's Deleted method where:
// - if delQ0 < iq1, adjust iq1 by min(n, iq1-delQ0)
// - if delQ0 < q0, adjust q0 by min(n, q0-delQ0)
// - if delQ0 < q1, adjust q1 by min(n, q1-delQ0)
func TestEditingManagerAdjustForDelete(t *testing.T) {
	tests := []struct {
		name    string
		q0      int
		q1      int
		iq1     int
		delQ0   int
		delQ1   int
		wantQ0  int
		wantQ1  int
		wantIQ1 int
	}{
		{"delete before all", 10, 20, 15, 2, 5, 7, 17, 12},           // n=3, all adjust by 3
		{"delete overlapping start", 10, 20, 15, 5, 15, 5, 10, 5},    // n=10, q0-=min(10,5)=5, q1-=min(10,15)=10, iq1-=min(10,10)=10
		{"delete inside selection", 10, 20, 15, 12, 15, 10, 17, 12},  // n=3, delQ0(12)<q0(10) false, delQ0(12)<q1(20) true, delQ0(12)<iq1(15) true
		{"delete overlapping end", 10, 20, 15, 15, 25, 10, 15, 15},   // n=10, delQ0(15)<q0(10) false, delQ0(15)<q1(20) true->q1-=5, delQ0(15)<iq1(15) false
		{"delete containing selection", 10, 20, 15, 5, 25, 5, 5, 5},  // n=20, all adjust
		{"delete after selection", 10, 20, 15, 25, 30, 10, 20, 15},   // delQ0(25) not < any - nothing adjusts
		{"delete at iq1", 10, 20, 15, 15, 18, 10, 17, 15},            // n=3, delQ0(15)<q0(10) false, delQ0(15)<q1(20) true->q1-=3, delQ0(15)<iq1(15) false
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.Selection().SetSelection(tc.q0, tc.q1)
			em.SetIQ1(tc.iq1)

			em.AdjustForDelete(tc.delQ0, tc.delQ1)

			if em.Selection().Q0() != tc.wantQ0 || em.Selection().Q1() != tc.wantQ1 {
				t.Errorf("selection after delete = (%d, %d); want (%d, %d)",
					em.Selection().Q0(), em.Selection().Q1(), tc.wantQ0, tc.wantQ1)
			}
			if em.IQ1() != tc.wantIQ1 {
				t.Errorf("IQ1 after delete = %d; want %d", em.IQ1(), tc.wantIQ1)
			}
		})
	}
}

// TestEditingManagerBsWidth tests backspace width calculation.
// The word erase behavior:
// 1. Skip non-alphanumeric characters (skipping = true)
// 2. When alphanumeric found, stop skipping (skipping = false)
// 3. Continue erasing alphanumeric until non-alphanumeric found
func TestEditingManagerBsWidth(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		q0      int
		r       rune
		wantLen int
	}{
		// ^H: erase one character
		{"backspace single char", "hello", 5, 0x08, 1},
		{"backspace at start", "hello", 0, 0x08, 0},
		{"backspace mid word", "hello", 3, 0x08, 1},

		// ^W: erase word - skips non-alnum then erases alnum
		{"erase word at end", "hello world", 11, 0x17, 5},       // "world"
		{"erase word mid", "hello world", 5, 0x17, 5},           // "hello"
		{"erase word with spaces", "hello   world", 8, 0x17, 8}, // skips spaces, then erases "hello" = 8 chars
		{"erase word at start", "hello", 0, 0x17, 0},
		{"erase word after spaces", "hello   ", 8, 0x17, 8},     // skip spaces, erase "hello"

		// ^U: erase to beginning of line
		{"erase line", "hello\nworld", 11, 0x15, 5},        // "world"
		{"erase line mid", "hello\nworld", 8, 0x15, 2},     // "wo"
		{"erase line at newline", "hello\nworld", 6, 0x15, 1}, // the newline
		{"erase line at start", "hello", 0, 0x15, 0},
		{"erase entire first line", "hello", 5, 0x15, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.Selection().SetQ0(tc.q0)
			em.Selection().SetQ1(tc.q0)

			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			got := em.BsWidth(tc.r, charReader)
			if got != tc.wantLen {
				t.Errorf("BsWidth(%q, %#x) at pos %d = %d; want %d",
					tc.text, tc.r, tc.q0, got, tc.wantLen)
			}
		})
	}
}

// TestEditingManagerDeleteRange tests calculating delete range for backspace ops.
func TestEditingManagerDeleteRange(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		q0           int
		r            rune
		org          int
		wantDelQ0    int
		wantDelQ1    int
		wantAdjusted bool
	}{
		{"normal backspace", "hello", 5, 0x08, 0, 4, 5, false},
		{"backspace at org boundary", "hello world", 5, 0x08, 5, 5, 5, true}, // adjusted to org
		{"backspace within org", "hello world", 8, 0x08, 5, 7, 8, false},
		{"erase word crosses org", "hello world", 11, 0x17, 8, 8, 11, true}, // adjusted to org
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.Selection().SetQ0(tc.q0)
			em.Selection().SetQ1(tc.q0)

			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			delQ0, delQ1, adjusted := em.CalculateDeleteRange(tc.r, tc.org, charReader)
			if delQ0 != tc.wantDelQ0 || delQ1 != tc.wantDelQ1 {
				t.Errorf("CalculateDeleteRange = (%d, %d); want (%d, %d)",
					delQ0, delQ1, tc.wantDelQ0, tc.wantDelQ1)
			}
			if adjusted != tc.wantAdjusted {
				t.Errorf("adjusted = %v; want %v", adjusted, tc.wantAdjusted)
			}
		})
	}
}

// TestEditingManagerTypeCommit tests type commit behavior.
func TestEditingManagerTypeCommit(t *testing.T) {
	em := NewEditingManager(nil, nil)

	// Set up some typing state
	em.SetEQ0(10)
	em.SetIQ1(20)

	// After commit, eq0 should be reset but iq1 preserved
	// (actual commit behavior depends on window, but we test state management)
	if !em.TypingStarted() {
		// eq0 = 10 means typing has NOT started (typing started when eq0 == 0)
		// Let's set it to indicate typing started
		em.SetEQ0(0)
	}

	if !em.TypingStarted() {
		t.Error("typing should have started")
	}

	// Commit resets typing state
	em.CommitTyping()
	if em.TypingStarted() {
		t.Error("typing should not have started after commit")
	}
}

// TestEditingManagerHasSelection tests selection presence checking.
func TestEditingManagerHasSelection(t *testing.T) {
	em := NewEditingManager(nil, nil)

	if em.HasSelection() {
		t.Error("new EditingManager should not have selection")
	}

	em.Selection().SetSelection(10, 20)
	if !em.HasSelection() {
		t.Error("EditingManager should have selection after SetSelection")
	}

	em.Selection().SetSelection(15, 15)
	if em.HasSelection() {
		t.Error("EditingManager should not have selection when q0 == q1")
	}
}

// TestEditingManagerInSelection tests position-in-selection checking.
func TestEditingManagerInSelection(t *testing.T) {
	tests := []struct {
		name      string
		q0, q1    int
		pos       int
		wantInSel bool
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
			em := NewEditingManager(nil, nil)
			em.Selection().SetSelection(tc.q0, tc.q1)

			result := em.InSelection(tc.pos)
			if result != tc.wantInSel {
				t.Errorf("InSelection(%d) with selection (%d, %d) = %v; want %v",
					tc.pos, tc.q0, tc.q1, result, tc.wantInSel)
			}
		})
	}
}

// TestEditingManagerSelectToInsertionPoint tests selecting back to insertion point.
func TestEditingManagerSelectToInsertionPoint(t *testing.T) {
	tests := []struct {
		name    string
		eq0     int
		q0      int
		wantQ0  int
		wantQ1  int
	}{
		{"eq0 before q0", 5, 10, 5, 10},
		{"eq0 after q0", 15, 10, 10, 15},
		{"eq0 equals q0", 10, 10, 10, 10},
		{"eq0 sentinel (no change)", ^0, 10, 10, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.SetEQ0(tc.eq0)
			em.Selection().SetQ0(tc.q0)
			em.Selection().SetQ1(tc.q0)

			em.SelectToInsertionPoint()

			if em.Selection().Q0() != tc.wantQ0 || em.Selection().Q1() != tc.wantQ1 {
				t.Errorf("SelectToInsertionPoint with eq0=%d, q0=%d = (%d, %d); want (%d, %d)",
					tc.eq0, tc.q0, em.Selection().Q0(), em.Selection().Q1(), tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestEditingManagerAutoIndent tests autoindent calculation.
// It uses BsWidth(^U) to find the beginning of the current line,
// then extracts leading whitespace from that position.
func TestEditingManagerAutoIndent(t *testing.T) {
	// "one\n  two\n    three"
	//  0123 456789 ...
	// Position 4 is ' ', 5 is ' ', 6 is 't'
	// Position 10 is ' ', 11 is ' ', 12 is ' ', 13 is ' ', 14 is 't'
	tests := []struct {
		name       string
		text       string
		q0         int
		wantIndent string
	}{
		{"no indent", "hello\nworld", 6, ""},          // 'w' is start of line, no indent
		{"tab indent", "\thello\n\tworld", 8, "\t"},   // line starts with '\t'
		{"space indent", "    hello\n    world", 15, "    "}, // line starts with "    "
		{"mixed indent", "\t  hello\n\t  world", 12, "\t  "}, // line starts with "\t  "
		{"indent at start of file", "hello", 0, ""},
		{"at end of indented line", "  hello", 7, "  "},      // cursor at end, line starts with "  "
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.Selection().SetQ0(tc.q0)
			em.Selection().SetQ1(tc.q0)

			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			got := em.CalculateAutoIndent(charReader)
			if got != tc.wantIndent {
				t.Errorf("CalculateAutoIndent in %q at pos %d = %q; want %q",
					tc.text, tc.q0, got, tc.wantIndent)
			}
		})
	}
}

// TestEditingManagerTabExpansion tests tab expansion calculation.
func TestEditingManagerTabExpansion(t *testing.T) {
	tests := []struct {
		name    string
		tabstop int
		wantLen int
	}{
		{"default tab stop", 4, 4},
		{"tab stop 8", 8, 8},
		{"tab stop 2", 2, 2},
		{"tab stop 1", 1, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)
			em.SetTabStop(tc.tabstop)

			got := em.ExpandedTabWidth()
			if got != tc.wantLen {
				t.Errorf("ExpandedTabWidth with tabstop=%d = %d; want %d",
					tc.tabstop, got, tc.wantLen)
			}
		})
	}
}

// TestEditingManagerUpdateIQ1 tests IQ1 update after operations.
func TestEditingManagerUpdateIQ1(t *testing.T) {
	em := NewEditingManager(nil, nil)

	em.Selection().SetQ0(10)
	em.Selection().SetQ1(20)

	// After paste-like operation, iq1 should be set to q1
	em.UpdateIQ1AfterPaste()
	if em.IQ1() != 20 {
		t.Errorf("IQ1 after paste should be 20; got %d", em.IQ1())
	}

	// After cut-like operation, iq1 should be set to q0
	em.Selection().SetQ0(5)
	em.Selection().SetQ1(5)
	em.UpdateIQ1AfterCut()
	if em.IQ1() != 5 {
		t.Errorf("IQ1 after cut should be 5; got %d", em.IQ1())
	}

	// After regular typing, iq1 should be set to q0
	em.Selection().SetQ0(15)
	em.Selection().SetQ1(15)
	em.UpdateIQ1AfterType()
	if em.IQ1() != 15 {
		t.Errorf("IQ1 after type should be 15; got %d", em.IQ1())
	}
}

// TestEditingManagerIntegration tests EditingManager in a realistic scenario.
func TestEditingManagerIntegration(t *testing.T) {
	em := NewEditingManager(nil, nil)

	// Simulate typing scenario
	text := []rune("hello world")
	charReader := func(pos int) rune {
		if pos < 0 || pos >= len(text) {
			return 0
		}
		return text[pos]
	}

	// Position cursor at end
	em.Selection().SetQ0(11)
	em.Selection().SetQ1(11)
	em.SetIQ1(11)

	// Prepare for insert
	em.PrepareInsert()
	if em.EQ0() != 11 {
		t.Errorf("EQ0 should be 11 after PrepareInsert; got %d", em.EQ0())
	}

	// Simulate inserting "!" (1 rune)
	em.AdjustForInsert(11, 1)
	if em.Selection().Q0() != 11 || em.Selection().Q1() != 11 {
		t.Errorf("selection should be (11, 11) after insert at end; got (%d, %d)",
			em.Selection().Q0(), em.Selection().Q1())
	}

	// Update text and move cursor
	text = append(text, '!')
	em.Selection().SetQ0(12)
	em.Selection().SetQ1(12)
	em.UpdateIQ1AfterType()

	// Test backspace width
	bsWidth := em.BsWidth(0x08, charReader)
	if bsWidth != 1 {
		t.Errorf("BsWidth should be 1; got %d", bsWidth)
	}

	// Test word erase width
	// "hello world!" at position 12 - word erase skips "!" (non-alnum),
	// then erases "world" (5 alnum chars), total = 6
	wordWidth := em.BsWidth(0x17, charReader)
	if wordWidth != 6 {
		t.Errorf("word erase width should be 6 (skip '!', erase 'world'); got %d", wordWidth)
	}

	// Select some text
	em.Selection().SetSelection(6, 11) // "world"
	if !em.HasSelection() {
		t.Error("should have selection")
	}
	if !em.InSelection(8) {
		t.Error("position 8 should be in selection")
	}
}

// TestEditingManagerFileWidth tests file path width calculation.
func TestEditingManagerFileWidth(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		q0         int
		oneElement bool
		wantWidth  int
	}{
		{"simple word", "hello world", 5, false, 5},
		{"path element", "/usr/local/bin", 14, true, 3},       // "bin"
		{"path full", "/usr/local/bin", 14, false, 14},        // full path
		{"space terminates", "hello world", 11, false, 5},     // "world"
		{"at start", "hello", 0, false, 0},
		{"with slash", "foo/bar/baz", 11, true, 3},            // "baz"
		{"with slash full", "foo/bar/baz", 11, false, 11},     // full
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEditingManager(nil, nil)

			textRunes := []rune(tc.text)
			charReader := func(pos int) rune {
				if pos < 0 || pos >= len(textRunes) {
					return 0
				}
				return textRunes[pos]
			}

			got := em.FileWidth(tc.q0, tc.oneElement, charReader)
			if got != tc.wantWidth {
				t.Errorf("FileWidth(%d, %v) in %q = %d; want %d",
					tc.q0, tc.oneElement, tc.text, got, tc.wantWidth)
			}
		})
	}
}

// =============================================================================
// Tests for Text interface (Phase 6E)
// These tests validate the Text interface and TextBase implementation.
// =============================================================================

// TestTextInterfaceImplementation verifies TextBase implements the Text interface.
func TestTextInterfaceImplementation(t *testing.T) {
	var _ Text = (*TextBase)(nil)
	var _ Text = NewTextBase()
}

// TestTextBaseTextInterface tests that TextBase provides all Text interface methods.
func TestTextBaseTextInterface(t *testing.T) {
	tb := NewTextBase()

	// Use as Text interface
	var txt Text = tb

	// Test selection accessors
	txt.SetQ0(10)
	txt.SetQ1(20)
	if txt.Q0() != 10 {
		t.Errorf("Q0 should be 10; got %d", txt.Q0())
	}
	if txt.Q1() != 20 {
		t.Errorf("Q1 should be 20; got %d", txt.Q1())
	}
	if !txt.HasSelection() {
		t.Error("should have selection when Q0 != Q1")
	}
	txt.ClearSelection()
	if txt.HasSelection() {
		t.Error("should not have selection after ClearSelection")
	}

	// Test display accessors
	txt.SetOrg(100)
	if txt.Org() != 100 {
		t.Errorf("Org should be 100; got %d", txt.Org())
	}
	if !txt.NeedsRedraw() {
		t.Error("should need redraw after SetOrg")
	}
	txt.ClearRedrawFlag()
	if txt.NeedsRedraw() {
		t.Error("should not need redraw after ClearRedrawFlag")
	}
	txt.SetNeedsRedraw(true)
	if !txt.NeedsRedraw() {
		t.Error("should need redraw after SetNeedsRedraw(true)")
	}

	// Test edit state accessors
	txt.SetIQ1(50)
	if txt.IQ1() != 50 {
		t.Errorf("IQ1 should be 50; got %d", txt.IQ1())
	}
	txt.SetEQ0(25)
	if txt.EQ0() != 25 {
		t.Errorf("EQ0 should be 25; got %d", txt.EQ0())
	}

	// Test tab settings
	txt.SetTabStop(8)
	if txt.TabStop() != 8 {
		t.Errorf("TabStop should be 8; got %d", txt.TabStop())
	}
	txt.SetTabExpand(true)
	if !txt.TabExpand() {
		t.Error("TabExpand should be true")
	}

	// Test fill control
	txt.SetNoFill(true)
	if !txt.NoFill() {
		t.Error("NoFill should be true")
	}
}

// TestTextBaseWithManagers tests using TextBase alongside manager types.
func TestTextBaseWithManagers(t *testing.T) {
	tb := NewTextBase()

	// Create managers using TextBase's state components
	sm := NewSelectionManager(tb.Selection)
	dm := NewDisplayManager(tb.Display)
	em := NewEditingManager(tb.Edit, tb.Selection)

	// Changes through TextBase should be visible through managers
	tb.SetQ0(10)
	tb.SetQ1(20)
	sel := sm.Selection()
	if sel.Start != 10 || sel.End != 20 {
		t.Errorf("SelectionManager should see (10, 20); got (%d, %d)", sel.Start, sel.End)
	}

	tb.SetOrg(100)
	if dm.Org() != 100 {
		t.Errorf("DisplayManager should see org=100; got %d", dm.Org())
	}

	tb.Edit.SetIQ1(50)
	if em.IQ1() != 50 {
		t.Errorf("EditingManager should see iq1=50; got %d", em.IQ1())
	}

	// Changes through managers should be visible through TextBase
	sm.SetSelection(30, 40)
	if tb.Q0() != 30 || tb.Q1() != 40 {
		t.Errorf("TextBase should see (30, 40); got (%d, %d)", tb.Q0(), tb.Q1())
	}

	dm.SetOrg(200)
	if tb.Org() != 200 {
		t.Errorf("TextBase should see org=200; got %d", tb.Org())
	}

	em.SetIQ1(75)
	if tb.IQ1() != 75 {
		t.Errorf("TextBase should see iq1=75; got %d", tb.IQ1())
	}
}

// TestTextBaseEditingScenario tests TextBase in an editing scenario.
func TestTextBaseEditingScenario(t *testing.T) {
	tb := NewTextBase()

	// Simulate a text editing session
	text := []rune("hello world")
	charReader := func(pos int) rune {
		if pos < 0 || pos >= len(text) {
			return 0
		}
		return text[pos]
	}

	// Set up initial state
	tb.SetOrg(0)
	tb.SetQ0(11) // end of "world"
	tb.SetQ1(11)
	tb.SetTabStop(4)
	tb.SetTabExpand(false)

	// Create editing manager to use with TextBase's state
	em := NewEditingManager(tb.Edit, tb.Selection)

	// Test backspace width calculation
	bsWidth := em.BsWidth(0x08, charReader)
	if bsWidth != 1 {
		t.Errorf("BsWidth for single char should be 1; got %d", bsWidth)
	}

	// Test word erase width - should erase "world"
	wordWidth := em.BsWidth(0x17, charReader)
	if wordWidth != 5 {
		t.Errorf("BsWidth for word erase should be 5; got %d", wordWidth)
	}

	// Simulate selecting text
	tb.SetQ0(6)
	tb.SetQ1(11)
	if !tb.HasSelection() {
		t.Error("should have selection")
	}

	// Test that selection is visible through manager
	if !em.HasSelection() {
		t.Error("EditingManager should see selection")
	}
	if !em.InSelection(8) {
		t.Error("position 8 should be in selection")
	}
}

// TestTextBaseDisplayScenario tests TextBase in a display scenario.
func TestTextBaseDisplayScenario(t *testing.T) {
	tb := NewTextBase()
	dm := NewDisplayManager(tb.Display)

	// Simulate scrolling
	tb.SetOrg(0)
	tb.ClearRedrawFlag()

	// Check visibility with 50 chars visible
	nchars := 50

	if !dm.IsPositionVisible(25, nchars) {
		t.Error("position 25 should be visible with org=0 and 50 chars")
	}

	// Scroll down
	tb.SetOrg(100)
	if !tb.NeedsRedraw() {
		t.Error("should need redraw after scrolling")
	}

	// Check visibility after scroll
	if dm.IsPositionVisible(50, nchars) {
		t.Error("position 50 should not be visible with org=100")
	}
	if !dm.IsPositionVisible(125, nchars) {
		t.Error("position 125 should be visible with org=100 and 50 chars")
	}

	// Test range visibility
	r := Range{Start: 90, End: 110}
	if !dm.IsRangeVisible(r, nchars) {
		t.Error("range (90, 110) should be partially visible with org=100")
	}
	if dm.IsRangeFullyVisible(r, nchars) {
		t.Error("range (90, 110) should not be fully visible with org=100")
	}
}

// TestTextBaseSelectionScenario tests TextBase in a selection scenario.
func TestTextBaseSelectionScenario(t *testing.T) {
	tb := NewTextBase()
	sm := NewSelectionManager(tb.Selection)

	text := []rune("func main() {\n\tfmt.Println(\"hello\")\n}")
	textReader := func(start, end int) []rune {
		if start < 0 {
			start = 0
		}
		if end > len(text) {
			end = len(text)
		}
		if start >= end {
			return nil
		}
		return text[start:end]
	}
	charReader := func(pos int) rune {
		if pos < 0 || pos >= len(text) {
			return 0
		}
		return text[pos]
	}

	// Double-click to select word "main"
	pos := 7 // position in "main"
	wordRange := sm.ExpandToWord(pos, textReader)
	if wordRange.Start != 5 || wordRange.End != 9 {
		t.Errorf("word selection should be (5, 9); got (%d, %d)", wordRange.Start, wordRange.End)
	}

	// Apply selection to TextBase
	tb.SetQ0(wordRange.Start)
	tb.SetQ1(wordRange.End)
	if !tb.HasSelection() {
		t.Error("should have selection after word expand")
	}

	// Triple-click to select line
	lineRange := sm.ExpandToLine(20, textReader) // position in second line
	if lineRange.Start != 14 {
		t.Errorf("line start should be 14; got %d", lineRange.Start)
	}

	// Test bracket matching - position inside parentheses "Println()"
	// Position 27 is inside "hello", between the quotes
	// ExpandToBrackets at position right after opening quote (pos 28) should match quotes
	bracketRange := sm.ExpandToBrackets(28, len(text), textReader, charReader)
	// Position 27 is the opening quote, position 33 is the closing quote
	// When at position 28 (after opening quote), it should expand to content between quotes
	if bracketRange.Start != 28 || bracketRange.End != 33 {
		t.Errorf("quote match should be (28, 33); got (%d, %d)", bracketRange.Start, bracketRange.End)
	}
}

// TestTextInterfacePolymorphism tests that Text interface can be used polymorphically.
func TestTextInterfacePolymorphism(t *testing.T) {
	// Create a TextBase and use it as Text interface
	tb := NewTextBase()
	tb.SetQ0(10)
	tb.SetQ1(20)
	tb.SetOrg(100)
	tb.SetTabStop(8)

	// Use as Text interface
	var txt Text = tb

	if txt.Q0() != 10 {
		t.Errorf("Text.Q0() should be 10; got %d", txt.Q0())
	}
	if txt.Q1() != 20 {
		t.Errorf("Text.Q1() should be 20; got %d", txt.Q1())
	}
	if txt.Org() != 100 {
		t.Errorf("Text.Org() should be 100; got %d", txt.Org())
	}
	if txt.TabStop() != 8 {
		t.Errorf("Text.TabStop() should be 8; got %d", txt.TabStop())
	}
	if !txt.HasSelection() {
		t.Error("Text.HasSelection() should be true")
	}
}

// TestTextBaseAdjustAfterInsert tests TextBase state adjustment after text insertion.
func TestTextBaseAdjustAfterInsert(t *testing.T) {
	tb := NewTextBase()
	sm := NewSelectionManager(tb.Selection)
	em := NewEditingManager(tb.Edit, tb.Selection)

	// Set up initial selection
	tb.SetQ0(10)
	tb.SetQ1(20)
	tb.Edit.SetIQ1(15)

	// Simulate inserting 5 chars at position 5 (before selection)
	em.AdjustForInsert(5, 5)

	// Selection should be shifted
	if tb.Q0() != 15 || tb.Q1() != 25 {
		t.Errorf("selection should be shifted to (15, 25); got (%d, %d)", tb.Q0(), tb.Q1())
	}
	if tb.IQ1() != 20 {
		t.Errorf("iq1 should be shifted to 20; got %d", tb.IQ1())
	}

	// Verify through SelectionManager
	sel := sm.Selection()
	if sel.Start != 15 || sel.End != 25 {
		t.Errorf("SelectionManager should see (15, 25); got (%d, %d)", sel.Start, sel.End)
	}
}

// TestTextBaseAdjustAfterDelete tests TextBase state adjustment after text deletion.
func TestTextBaseAdjustAfterDelete(t *testing.T) {
	tb := NewTextBase()
	sm := NewSelectionManager(tb.Selection)
	em := NewEditingManager(tb.Edit, tb.Selection)

	// Set up initial selection
	tb.SetQ0(10)
	tb.SetQ1(20)
	tb.Edit.SetIQ1(15)

	// Simulate deleting 3 chars at position 5 (before selection)
	em.AdjustForDelete(5, 8)

	// Selection should be shifted back
	if tb.Q0() != 7 || tb.Q1() != 17 {
		t.Errorf("selection should be shifted to (7, 17); got (%d, %d)", tb.Q0(), tb.Q1())
	}
	if tb.IQ1() != 12 {
		t.Errorf("iq1 should be shifted to 12; got %d", tb.IQ1())
	}

	// Verify through SelectionManager
	sel := sm.Selection()
	if sel.Start != 7 || sel.End != 17 {
		t.Errorf("SelectionManager should see (7, 17); got (%d, %d)", sel.Start, sel.End)
	}
}

// TestTextBaseComposition tests that TextBase properly composes all state types.
func TestTextBaseComposition(t *testing.T) {
	tb := NewTextBase()

	// Simulate a complete text setup
	tb.SetQ0(100)
	tb.SetQ1(200)
	tb.SetOrg(50)
	tb.Edit.SetIQ1(150)
	tb.Edit.SetEQ0(100)
	tb.SetTabStop(8)
	tb.SetTabExpand(true)
	tb.SetNoFill(false)

	// Verify all state is correctly set across components
	if tb.Q0() != 100 || tb.Q1() != 200 {
		t.Errorf("selection should be (100, 200); got (%d, %d)", tb.Q0(), tb.Q1())
	}
	if tb.Org() != 50 {
		t.Errorf("org should be 50; got %d", tb.Org())
	}
	if tb.IQ1() != 150 {
		t.Errorf("iq1 should be 150; got %d", tb.IQ1())
	}
	if tb.EQ0() != 100 {
		t.Errorf("eq0 should be 100; got %d", tb.EQ0())
	}
	if tb.TabStop() != 8 {
		t.Errorf("tabstop should be 8; got %d", tb.TabStop())
	}
	if !tb.TabExpand() {
		t.Error("tabexpand should be true")
	}
	if tb.NoFill() {
		t.Error("nofill should be false")
	}

	// Verify state is synchronized across components
	if tb.Selection.Q0() != 100 || tb.Selection.Q1() != 200 {
		t.Error("SelectionState should match")
	}
	if tb.Display.Org() != 50 {
		t.Error("DisplayState should match")
	}
	if tb.Edit.IQ1() != 150 {
		t.Error("EditState should match")
	}
}
