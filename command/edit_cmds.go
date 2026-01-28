// Package command provides command dispatch functionality for edwood.
// This file contains types and helpers for edit commands (Cut, Paste, Snarf, Undo, Redo).
package command

// =============================================================================
// Selection State Types
// =============================================================================

// SelectionState represents the state of a text selection for edit commands.
// Edit commands like Cut/Paste/Snarf operate on selections.
type SelectionState struct {
	q0       int  // Selection start
	q1       int  // Selection end
	hasOwner bool // Whether this selection belongs to a window
}

// NewSelectionState creates a new SelectionState.
func NewSelectionState(q0, q1 int, hasOwner bool) *SelectionState {
	return &SelectionState{
		q0:       q0,
		q1:       q1,
		hasOwner: hasOwner,
	}
}

// Q0 returns the selection start.
func (s *SelectionState) Q0() int { return s.q0 }

// Q1 returns the selection end.
func (s *SelectionState) Q1() int { return s.q1 }

// HasOwner returns true if this selection belongs to a window.
func (s *SelectionState) HasOwner() bool { return s.hasOwner }

// HasSelection returns true if there is a non-empty selection.
func (s *SelectionState) HasSelection() bool {
	return s.q1 > s.q0
}

// Length returns the length of the selection.
func (s *SelectionState) Length() int {
	if s.q1 > s.q0 {
		return s.q1 - s.q0
	}
	return 0
}

// =============================================================================
// Cut Command Types
// =============================================================================

// CutOperation represents the parameters for a cut operation.
// In exec.go, cut() is called with dosnarf and docut flags:
// - Cut command: dosnarf=true, docut=true
// - Snarf command: dosnarf=true, docut=false
type CutOperation struct {
	doSnarf bool // Copy selection to snarf buffer
	doCut   bool // Delete selection after copying
}

// NewCutOperation creates a CutOperation for Cut command.
func NewCutOperation() *CutOperation {
	return &CutOperation{doSnarf: true, doCut: true}
}

// NewSnarfOperation creates a CutOperation for Snarf command.
func NewSnarfOperation() *CutOperation {
	return &CutOperation{doSnarf: true, doCut: false}
}

// DoSnarf returns true if the operation should copy to snarf buffer.
func (c *CutOperation) DoSnarf() bool { return c.doSnarf }

// DoCut returns true if the operation should delete the selection.
func (c *CutOperation) DoCut() bool { return c.doCut }

// ShouldCopyToBuffer returns true if selection should be copied.
func (c *CutOperation) ShouldCopyToBuffer(sel *SelectionState) bool {
	return c.doSnarf && sel.HasSelection()
}

// ShouldDeleteSelection returns true if selection should be deleted.
func (c *CutOperation) ShouldDeleteSelection(sel *SelectionState) bool {
	return c.doCut && sel.HasSelection()
}

// =============================================================================
// Paste Command Types
// =============================================================================

// PasteOperation represents the parameters for a paste operation.
// In exec.go, paste() is called with selectall and tobody flags:
// - Paste command: selectall=true, tobody=true
type PasteOperation struct {
	selectAll bool // Select the pasted text after insert
	toBody    bool // Paste into window body (not tag)
}

// NewPasteOperation creates a PasteOperation for Paste command.
func NewPasteOperation() *PasteOperation {
	return &PasteOperation{selectAll: true, toBody: true}
}

// SelectAll returns true if pasted text should be selected.
func (p *PasteOperation) SelectAll() bool { return p.selectAll }

// ToBody returns true if paste should go to window body.
func (p *PasteOperation) ToBody() bool { return p.toBody }

// ShouldPaste returns true if there is content to paste.
func (p *PasteOperation) ShouldPaste(snarfLen int) bool {
	return snarfLen > 0
}

// CalculateNewSelection returns the selection range after paste.
// If selectAll is true, selects all pasted text; otherwise cursor at end.
func (p *PasteOperation) CalculateNewSelection(insertPos, pasteLen int) (q0, q1 int) {
	if p.selectAll {
		return insertPos, insertPos + pasteLen
	}
	return insertPos + pasteLen, insertPos + pasteLen
}

// =============================================================================
// Undo/Redo Command Types
// =============================================================================

// UndoOperation represents the parameters for an undo/redo operation.
// In exec.go, undo() is called with flag1:
// - Undo command: flag1=true
// - Redo command: flag1=false
type UndoOperation struct {
	isUndo bool // true for Undo, false for Redo
}

// NewUndoOperation creates an UndoOperation for Undo command.
func NewUndoOperation() *UndoOperation {
	return &UndoOperation{isUndo: true}
}

// NewRedoOperation creates an UndoOperation for Redo command.
func NewRedoOperation() *UndoOperation {
	return &UndoOperation{isUndo: false}
}

// IsUndo returns true if this is an undo operation.
func (u *UndoOperation) IsUndo() bool { return u.isUndo }

// IsRedo returns true if this is a redo operation.
func (u *UndoOperation) IsRedo() bool { return !u.isUndo }

// UndoState represents the undo/redo state for a file.
// This mirrors the sequence-based undo system in edwood.
type UndoState struct {
	seq     int // Current sequence number
	redoSeq int // Sequence for redo
}

// NewUndoState creates a new UndoState.
func NewUndoState(seq, redoSeq int) *UndoState {
	return &UndoState{seq: seq, redoSeq: redoSeq}
}

// CanUndo returns true if undo is possible.
func (u *UndoState) CanUndo() bool {
	return u.seq > 0
}

// CanRedo returns true if redo is possible.
func (u *UndoState) CanRedo() bool {
	return u.redoSeq > 0
}

// SeqOf returns the appropriate sequence based on operation type.
// This matches the seqof() function in exec.go.
func (u *UndoState) SeqOf(isUndo bool) int {
	if isUndo {
		return u.seq
	}
	return u.redoSeq
}

// =============================================================================
// Edit Command Registry
// =============================================================================

// EditCommandRegistry provides standard edit command entries for registration.
type EditCommandRegistry struct{}

// NewEditCommandRegistry creates a new EditCommandRegistry.
func NewEditCommandRegistry() *EditCommandRegistry {
	return &EditCommandRegistry{}
}

// RegisterEditCommands registers all edit commands with the dispatcher.
// The commands registered are: Cut, Paste, Snarf, Undo, Redo
func (r *EditCommandRegistry) RegisterEditCommands(d *Dispatcher) {
	// These match the entries in globalexectab for edit commands
	// Format: name, mark (undoable), flag1, flag2
	d.RegisterCommand(NewCommandEntry("Cut", true, true, true))     // mark=true, dosnarf=true, docut=true
	d.RegisterCommand(NewCommandEntry("Paste", true, true, true))   // mark=true, selectall=true, tobody=true
	d.RegisterCommand(NewCommandEntry("Snarf", false, true, false)) // mark=false, dosnarf=true, docut=false
	d.RegisterCommand(NewCommandEntry("Undo", false, true, true))   // mark=false, flag1=true (is undo)
	d.RegisterCommand(NewCommandEntry("Redo", false, false, true))  // mark=false, flag1=false (is redo)
}
