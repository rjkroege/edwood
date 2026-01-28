// Package command provides command dispatch functionality for edwood.
package command

import (
	"testing"
)

// =============================================================================
// Tests for Edit Command Interfaces and Types
// =============================================================================
//
// These tests verify the interfaces and behaviors needed for edit commands
// (Cut, Paste, Snarf, Undo, Redo) that will be extracted from exec.go.
//
// The actual command implementations depend on main package types (Text, Window,
// File, etc.). These tests verify the command package can support edit operations
// through well-defined interfaces.

// =============================================================================
// Selection State Tests
// =============================================================================

// TestSelectionStateNew tests SelectionState creation.
func TestSelectionStateNew(t *testing.T) {
	s := NewSelectionState(10, 20, true)

	if s.Q0() != 10 {
		t.Errorf("Q0() = %d, want 10", s.Q0())
	}
	if s.Q1() != 20 {
		t.Errorf("Q1() = %d, want 20", s.Q1())
	}
	if !s.HasOwner() {
		t.Error("HasOwner() = false, want true")
	}
}

// TestSelectionStateHasSelection tests the HasSelection method.
func TestSelectionStateHasSelection(t *testing.T) {
	tests := []struct {
		name string
		q0   int
		q1   int
		want bool
	}{
		{"no selection (equal)", 10, 10, false},
		{"no selection (q0 > q1)", 20, 10, false},
		{"has selection", 10, 20, true},
		{"single char selection", 5, 6, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSelectionState(tc.q0, tc.q1, true)
			if got := s.HasSelection(); got != tc.want {
				t.Errorf("HasSelection() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSelectionStateLength tests the Length method.
func TestSelectionStateLength(t *testing.T) {
	tests := []struct {
		name string
		q0   int
		q1   int
		want int
	}{
		{"no selection", 10, 10, 0},
		{"has selection", 10, 20, 10},
		{"single char", 5, 6, 1},
		{"inverted (invalid)", 20, 10, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSelectionState(tc.q0, tc.q1, true)
			if got := s.Length(); got != tc.want {
				t.Errorf("Length() = %d, want %d", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Cut Command Tests
// =============================================================================

// TestCutOperationForCut tests CutOperation for the Cut command.
func TestCutOperationForCut(t *testing.T) {
	op := NewCutOperation()

	if !op.DoSnarf() {
		t.Error("Cut should snarf")
	}
	if !op.DoCut() {
		t.Error("Cut should delete")
	}

	// With selection
	sel := NewSelectionState(10, 20, true)
	if !op.ShouldCopyToBuffer(sel) {
		t.Error("Cut should copy selection to buffer")
	}
	if !op.ShouldDeleteSelection(sel) {
		t.Error("Cut should delete selection")
	}

	// Without selection
	nosel := NewSelectionState(10, 10, true)
	if op.ShouldCopyToBuffer(nosel) {
		t.Error("Cut should not copy when no selection")
	}
	if op.ShouldDeleteSelection(nosel) {
		t.Error("Cut should not delete when no selection")
	}
}

// TestCutOperationForSnarf tests CutOperation for the Snarf command.
func TestCutOperationForSnarf(t *testing.T) {
	op := NewSnarfOperation()

	if !op.DoSnarf() {
		t.Error("Snarf should snarf")
	}
	if op.DoCut() {
		t.Error("Snarf should not delete")
	}

	// With selection
	sel := NewSelectionState(10, 20, true)
	if !op.ShouldCopyToBuffer(sel) {
		t.Error("Snarf should copy selection to buffer")
	}
	if op.ShouldDeleteSelection(sel) {
		t.Error("Snarf should not delete selection")
	}
}

// TestCutWithNoSelection tests that cut operations handle empty selections.
func TestCutWithNoSelection(t *testing.T) {
	tests := []struct {
		name       string
		op         *CutOperation
		sel        *SelectionState
		wantCopy   bool
		wantDelete bool
	}{
		{
			name:       "Cut with selection",
			op:         NewCutOperation(),
			sel:        NewSelectionState(0, 10, true),
			wantCopy:   true,
			wantDelete: true,
		},
		{
			name:       "Cut without selection",
			op:         NewCutOperation(),
			sel:        NewSelectionState(5, 5, true),
			wantCopy:   false,
			wantDelete: false,
		},
		{
			name:       "Snarf with selection",
			op:         NewSnarfOperation(),
			sel:        NewSelectionState(0, 10, true),
			wantCopy:   true,
			wantDelete: false,
		},
		{
			name:       "Snarf without selection",
			op:         NewSnarfOperation(),
			sel:        NewSelectionState(5, 5, true),
			wantCopy:   false,
			wantDelete: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.op.ShouldCopyToBuffer(tc.sel); got != tc.wantCopy {
				t.Errorf("ShouldCopyToBuffer() = %v, want %v", got, tc.wantCopy)
			}
			if got := tc.op.ShouldDeleteSelection(tc.sel); got != tc.wantDelete {
				t.Errorf("ShouldDeleteSelection() = %v, want %v", got, tc.wantDelete)
			}
		})
	}
}

// =============================================================================
// Paste Command Tests
// =============================================================================

// TestPasteOperationNew tests PasteOperation creation.
func TestPasteOperationNew(t *testing.T) {
	op := NewPasteOperation()

	if !op.SelectAll() {
		t.Error("Paste should select all")
	}
	if !op.ToBody() {
		t.Error("Paste should go to body")
	}
}

// TestPasteOperationShouldPaste tests the ShouldPaste method.
func TestPasteOperationShouldPaste(t *testing.T) {
	op := NewPasteOperation()

	tests := []struct {
		name     string
		snarfLen int
		want     bool
	}{
		{"empty snarf buffer", 0, false},
		{"has content", 10, true},
		{"single char", 1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := op.ShouldPaste(tc.snarfLen); got != tc.want {
				t.Errorf("ShouldPaste(%d) = %v, want %v", tc.snarfLen, got, tc.want)
			}
		})
	}
}

// TestPasteOperationCalculateNewSelection tests selection calculation after paste.
func TestPasteOperationCalculateNewSelection(t *testing.T) {
	tests := []struct {
		name      string
		selectAll bool
		insertPos int
		pasteLen  int
		wantQ0    int
		wantQ1    int
	}{
		{
			name:      "select all after paste",
			selectAll: true,
			insertPos: 10,
			pasteLen:  5,
			wantQ0:    10,
			wantQ1:    15,
		},
		{
			name:      "cursor at end after paste",
			selectAll: false,
			insertPos: 10,
			pasteLen:  5,
			wantQ0:    15,
			wantQ1:    15,
		},
		{
			name:      "paste at start with select",
			selectAll: true,
			insertPos: 0,
			pasteLen:  20,
			wantQ0:    0,
			wantQ1:    20,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := &PasteOperation{selectAll: tc.selectAll, toBody: true}
			q0, q1 := op.CalculateNewSelection(tc.insertPos, tc.pasteLen)
			if q0 != tc.wantQ0 || q1 != tc.wantQ1 {
				t.Errorf("CalculateNewSelection(%d, %d) = (%d, %d), want (%d, %d)",
					tc.insertPos, tc.pasteLen, q0, q1, tc.wantQ0, tc.wantQ1)
			}
		})
	}
}

// TestPasteCutsExistingSelection tests that paste deletes existing selection first.
// This matches the behavior in exec.go where cut(t, t, nil, false, true, "") is called.
func TestPasteCutsExistingSelection(t *testing.T) {
	// Paste behavior: if there's a selection, delete it first, then insert
	tests := []struct {
		name            string
		initialQ0       int
		initialQ1       int
		pasteLen        int
		wantDeleteFirst bool
	}{
		{
			name:            "paste replaces selection",
			initialQ0:       5,
			initialQ1:       10,
			pasteLen:        3,
			wantDeleteFirst: true,
		},
		{
			name:            "paste at cursor (no selection)",
			initialQ0:       5,
			initialQ1:       5,
			pasteLen:        3,
			wantDeleteFirst: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sel := NewSelectionState(tc.initialQ0, tc.initialQ1, true)
			shouldDelete := sel.HasSelection()
			if shouldDelete != tc.wantDeleteFirst {
				t.Errorf("HasSelection() = %v, want %v", shouldDelete, tc.wantDeleteFirst)
			}
		})
	}
}

// =============================================================================
// Undo/Redo Command Tests
// =============================================================================

// TestUndoOperationNew tests UndoOperation creation.
func TestUndoOperationNew(t *testing.T) {
	undo := NewUndoOperation()
	if !undo.IsUndo() {
		t.Error("Undo operation should be undo")
	}
	if undo.IsRedo() {
		t.Error("Undo operation should not be redo")
	}

	redo := NewRedoOperation()
	if redo.IsUndo() {
		t.Error("Redo operation should not be undo")
	}
	if !redo.IsRedo() {
		t.Error("Redo operation should be redo")
	}
}

// TestUndoStateCanUndo tests the CanUndo method.
func TestUndoStateCanUndo(t *testing.T) {
	tests := []struct {
		name string
		seq  int
		want bool
	}{
		{"no undo available", 0, false},
		{"undo available", 1, true},
		{"multiple undos available", 10, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := NewUndoState(tc.seq, 0)
			if got := state.CanUndo(); got != tc.want {
				t.Errorf("CanUndo() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestUndoStateCanRedo tests the CanRedo method.
func TestUndoStateCanRedo(t *testing.T) {
	tests := []struct {
		name    string
		redoSeq int
		want    bool
	}{
		{"no redo available", 0, false},
		{"redo available", 1, true},
		{"multiple redos available", 10, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := NewUndoState(0, tc.redoSeq)
			if got := state.CanRedo(); got != tc.want {
				t.Errorf("CanRedo() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestUndoStateSeqOf tests the SeqOf method.
func TestUndoStateSeqOf(t *testing.T) {
	state := NewUndoState(5, 3)

	if got := state.SeqOf(true); got != 5 {
		t.Errorf("SeqOf(true) = %d, want 5", got)
	}
	if got := state.SeqOf(false); got != 3 {
		t.Errorf("SeqOf(false) = %d, want 3", got)
	}
}

// TestUndoNothingToUndo tests undo behavior when nothing to undo.
func TestUndoNothingToUndo(t *testing.T) {
	state := NewUndoState(0, 0)
	op := NewUndoOperation()

	seq := state.SeqOf(op.IsUndo())
	if seq != 0 {
		t.Errorf("expected seq=0 for nothing to undo, got %d", seq)
	}
}

// TestRedoNothingToRedo tests redo behavior when nothing to redo.
func TestRedoNothingToRedo(t *testing.T) {
	state := NewUndoState(5, 0) // Has undo history but no redo
	op := NewRedoOperation()

	seq := state.SeqOf(op.IsUndo())
	if seq != 0 {
		t.Errorf("expected seq=0 for nothing to redo, got %d", seq)
	}
}

// =============================================================================
// Edit Command Entry Tests
// =============================================================================

// TestEditCommandEntries tests that edit command entries have correct properties.
func TestEditCommandEntries(t *testing.T) {
	d := NewDispatcher()
	reg := NewEditCommandRegistry()
	reg.RegisterEditCommands(d)

	// Verify Cut is undoable
	cut := d.LookupCommand("Cut")
	if cut == nil {
		t.Fatal("Cut command not found")
	}
	if !cut.Mark() {
		t.Error("Cut should be undoable (mark=true)")
	}
	if !cut.Flag1() {
		t.Error("Cut should have flag1=true (dosnarf)")
	}
	if !cut.Flag2() {
		t.Error("Cut should have flag2=true (docut)")
	}

	// Verify Paste is undoable
	paste := d.LookupCommand("Paste")
	if paste == nil {
		t.Fatal("Paste command not found")
	}
	if !paste.Mark() {
		t.Error("Paste should be undoable (mark=true)")
	}

	// Verify Snarf is NOT undoable (it doesn't modify the buffer)
	snarf := d.LookupCommand("Snarf")
	if snarf == nil {
		t.Fatal("Snarf command not found")
	}
	if snarf.Mark() {
		t.Error("Snarf should not be undoable (mark=false)")
	}
	if !snarf.Flag1() {
		t.Error("Snarf should have flag1=true (dosnarf)")
	}
	if snarf.Flag2() {
		t.Error("Snarf should have flag2=false (docut)")
	}

	// Verify Undo is NOT undoable (you can't undo an undo that way)
	undo := d.LookupCommand("Undo")
	if undo == nil {
		t.Fatal("Undo command not found")
	}
	if undo.Mark() {
		t.Error("Undo should not be undoable (mark=false)")
	}
	if !undo.Flag1() {
		t.Error("Undo should have flag1=true (is undo)")
	}

	// Verify Redo is NOT undoable
	redo := d.LookupCommand("Redo")
	if redo == nil {
		t.Fatal("Redo command not found")
	}
	if redo.Mark() {
		t.Error("Redo should not be undoable (mark=false)")
	}
	if redo.Flag1() {
		t.Error("Redo should have flag1=false (is redo)")
	}
}

// =============================================================================
// Edit Command Dispatch Tests
// =============================================================================

// TestEditCommandDispatch tests that edit commands can be looked up correctly.
func TestEditCommandDispatch(t *testing.T) {
	d := NewDispatcher()
	reg := NewEditCommandRegistry()
	reg.RegisterEditCommands(d)

	tests := []struct {
		input    string
		wantName string
		found    bool
	}{
		{"Cut", "Cut", true},
		{"Paste", "Paste", true},
		{"Snarf", "Snarf", true},
		{"Undo", "Undo", true},
		{"Redo", "Redo", true},
		{"cut", "", false},   // Case sensitive
		{"PASTE", "", false}, // Case sensitive
		{"Copy", "", false},  // Not registered (acme uses Snarf)
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			cmd := d.LookupCommand(tc.input)
			if tc.found {
				if cmd == nil {
					t.Errorf("expected to find command for %q", tc.input)
					return
				}
				if cmd.Name() != tc.wantName {
					t.Errorf("expected name %q, got %q", tc.wantName, cmd.Name())
				}
			} else {
				if cmd != nil {
					t.Errorf("expected nil for %q, got %v", tc.input, cmd)
				}
			}
		})
	}
}

// TestEditCommandRegistryIntegration tests the full registration flow.
func TestEditCommandRegistryIntegration(t *testing.T) {
	d := NewDispatcher()
	reg := NewEditCommandRegistry()
	reg.RegisterEditCommands(d)

	// Should have exactly 5 edit commands
	cmds := d.Commands()
	if len(cmds) != 5 {
		t.Errorf("expected 5 commands, got %d", len(cmds))
	}

	// Verify all expected commands are present
	expected := map[string]bool{
		"Cut":   true,
		"Paste": true,
		"Snarf": true,
		"Undo":  true,
		"Redo":  true,
	}

	for _, cmd := range cmds {
		if !expected[cmd.Name()] {
			t.Errorf("unexpected command: %s", cmd.Name())
		}
		delete(expected, cmd.Name())
	}

	if len(expected) > 0 {
		t.Errorf("missing commands: %v", expected)
	}
}

// TestCutSnarfDifference tests the key difference between Cut and Snarf.
func TestCutSnarfDifference(t *testing.T) {
	d := NewDispatcher()
	reg := NewEditCommandRegistry()
	reg.RegisterEditCommands(d)

	cut := d.LookupCommand("Cut")
	snarf := d.LookupCommand("Snarf")

	// Both should copy (flag1=true means dosnarf)
	if !cut.Flag1() || !snarf.Flag1() {
		t.Error("Both Cut and Snarf should copy to snarf buffer")
	}

	// Only Cut should delete (flag2 means docut)
	if !cut.Flag2() {
		t.Error("Cut should delete selection (flag2=true)")
	}
	if snarf.Flag2() {
		t.Error("Snarf should NOT delete selection (flag2=false)")
	}

	// Only Cut is undoable (because it modifies the document)
	if !cut.Mark() {
		t.Error("Cut should be undoable (modifies document)")
	}
	if snarf.Mark() {
		t.Error("Snarf should not be undoable (doesn't modify document)")
	}
}

// TestUndoRedoDifference tests the key difference between Undo and Redo.
func TestUndoRedoDifference(t *testing.T) {
	d := NewDispatcher()
	reg := NewEditCommandRegistry()
	reg.RegisterEditCommands(d)

	undo := d.LookupCommand("Undo")
	redo := d.LookupCommand("Redo")

	// flag1 distinguishes Undo (true) from Redo (false)
	if !undo.Flag1() {
		t.Error("Undo should have flag1=true")
	}
	if redo.Flag1() {
		t.Error("Redo should have flag1=false")
	}

	// Neither should be marked as undoable
	if undo.Mark() || redo.Mark() {
		t.Error("Undo/Redo commands should not be undoable themselves")
	}
}

// =============================================================================
// Combined File and Edit Command Tests
// =============================================================================

// TestAllCommandsRegistered tests that file and edit commands can coexist.
func TestAllCommandsRegistered(t *testing.T) {
	d := NewDispatcher()

	fileReg := NewFileCommandRegistry()
	fileReg.RegisterFileCommands(d)

	editReg := NewEditCommandRegistry()
	editReg.RegisterEditCommands(d)

	// Should have 11 total commands (6 file + 5 edit)
	cmds := d.Commands()
	if len(cmds) != 11 {
		t.Errorf("expected 11 commands, got %d", len(cmds))
	}

	// Verify some from each category
	if d.LookupCommand("Del") == nil {
		t.Error("Del (file command) should be registered")
	}
	if d.LookupCommand("Cut") == nil {
		t.Error("Cut (edit command) should be registered")
	}
	if d.LookupCommand("Undo") == nil {
		t.Error("Undo (edit command) should be registered")
	}
}
