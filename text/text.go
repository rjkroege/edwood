// Package text provides the Text type and related components for edwood.
// This package contains text selection management, display methods,
// and editing operations.
package text

// Range represents a range of positions within text, typically used
// for selections, addresses, and limits.
type Range struct {
	Start int
	End   int
}

// IsEmpty returns true if the range represents an empty selection (start == end).
func (r Range) IsEmpty() bool {
	return r.Start == r.End
}

// Len returns the length of the range. If Start > End, returns a negative value.
func (r Range) Len() int {
	return r.End - r.Start
}

// Contains returns true if the given position is within the range.
func (r Range) Contains(pos int) bool {
	if r.Start <= r.End {
		return pos >= r.Start && pos < r.End
	}
	// Handle backwards selection
	return pos >= r.End && pos < r.Start
}

// Normalize returns a Range with Start <= End.
func (r Range) Normalize() Range {
	if r.Start <= r.End {
		return r
	}
	return Range{Start: r.End, End: r.Start}
}

// SelectionState holds the current text selection state.
// This will be used by SelectionManager in Phase 6B.
type SelectionState struct {
	// q0 is the start of the selection (cursor position if no selection)
	q0 int
	// q1 is the end of the selection
	q1 int
}

// NewSelectionState creates a new SelectionState with default values.
func NewSelectionState() *SelectionState {
	return &SelectionState{}
}

// Selection returns the current selection range.
func (s *SelectionState) Selection() Range {
	return Range{Start: s.q0, End: s.q1}
}

// SetSelection sets the selection range.
func (s *SelectionState) SetSelection(q0, q1 int) {
	s.q0 = q0
	s.q1 = q1
}

// Q0 returns the start of the selection.
func (s *SelectionState) Q0() int {
	return s.q0
}

// Q1 returns the end of the selection.
func (s *SelectionState) Q1() int {
	return s.q1
}

// SetQ0 sets the start of the selection.
func (s *SelectionState) SetQ0(q0 int) {
	s.q0 = q0
}

// SetQ1 sets the end of the selection.
func (s *SelectionState) SetQ1(q1 int) {
	s.q1 = q1
}

// HasSelection returns true if there is a non-empty selection.
func (s *SelectionState) HasSelection() bool {
	return s.q0 != s.q1
}

// ClearSelection collapses the selection to the cursor position (q0).
func (s *SelectionState) ClearSelection() {
	s.q1 = s.q0
}

// DisplayState holds state related to text display and redrawing.
// This will be used for display methods in Phase 6C.
type DisplayState struct {
	// org is the origin of the visible frame within the buffer
	org int
	// needsRedraw indicates whether the text needs to be redrawn
	needsRedraw bool
}

// NewDisplayState creates a new DisplayState with default values.
func NewDisplayState() *DisplayState {
	return &DisplayState{}
}

// Org returns the origin (first visible character position).
func (d *DisplayState) Org() int {
	return d.org
}

// SetOrg sets the origin.
func (d *DisplayState) SetOrg(org int) {
	if org < 0 {
		org = 0
	}
	d.org = org
	d.needsRedraw = true
}

// NeedsRedraw returns true if the text needs to be redrawn.
func (d *DisplayState) NeedsRedraw() bool {
	return d.needsRedraw
}

// SetNeedsRedraw marks the text as needing redraw.
func (d *DisplayState) SetNeedsRedraw(needs bool) {
	d.needsRedraw = needs
}

// ClearRedrawFlag clears the redraw flag.
func (d *DisplayState) ClearRedrawFlag() {
	d.needsRedraw = false
}

// EditState holds state related to text editing operations.
// This will be used for editing methods in Phase 6D.
type EditState struct {
	// iq1 is the initial q1 value when an editing operation started
	iq1 int
	// eq0 is used to track typing state; when 0, typing has started
	eq0 int
}

// NewEditState creates a new EditState with default values.
func NewEditState() *EditState {
	return &EditState{
		eq0: ^0, // Initialize to sentinel value (all bits set)
	}
}

// IQ1 returns the initial q1 value.
func (e *EditState) IQ1() int {
	return e.iq1
}

// SetIQ1 sets the initial q1 value.
func (e *EditState) SetIQ1(iq1 int) {
	e.iq1 = iq1
}

// EQ0 returns the eq0 value.
func (e *EditState) EQ0() int {
	return e.eq0
}

// SetEQ0 sets the eq0 value.
func (e *EditState) SetEQ0(eq0 int) {
	e.eq0 = eq0
}

// TypingStarted returns true if typing has started (eq0 == 0).
func (e *EditState) TypingStarted() bool {
	return e.eq0 == 0
}

// ResetTyping resets the typing state to the sentinel value.
func (e *EditState) ResetTyping() {
	e.eq0 = ^0
}

// TextBase provides portable state composition for Text implementations.
// This struct can be embedded in the main Text type to provide state
// management without circular dependencies.
type TextBase struct {
	Selection *SelectionState
	Display   *DisplayState
	Edit      *EditState

	// tabstop is the number of spaces for a tab
	tabstop int
	// tabexpand controls whether tabs are expanded to spaces
	tabexpand bool
	// nofill when true, updates shouldn't update the frame
	nofill bool
}

// NewTextBase creates a new TextBase with all state components initialized.
func NewTextBase() *TextBase {
	return &TextBase{
		Selection: NewSelectionState(),
		Display:   NewDisplayState(),
		Edit:      NewEditState(),
		tabstop:   4, // Default tab stop
	}
}

// TabStop returns the current tab stop value.
func (tb *TextBase) TabStop() int {
	return tb.tabstop
}

// SetTabStop sets the tab stop value.
func (tb *TextBase) SetTabStop(tabstop int) {
	if tabstop < 1 {
		tabstop = 1
	}
	tb.tabstop = tabstop
}

// TabExpand returns whether tabs should be expanded to spaces.
func (tb *TextBase) TabExpand() bool {
	return tb.tabexpand
}

// SetTabExpand sets whether tabs should be expanded to spaces.
func (tb *TextBase) SetTabExpand(expand bool) {
	tb.tabexpand = expand
}

// NoFill returns whether frame updates should be skipped.
func (tb *TextBase) NoFill() bool {
	return tb.nofill
}

// SetNoFill sets whether frame updates should be skipped.
func (tb *TextBase) SetNoFill(nofill bool) {
	tb.nofill = nofill
}

// Q0 returns the start of the selection.
func (tb *TextBase) Q0() int {
	return tb.Selection.Q0()
}

// Q1 returns the end of the selection.
func (tb *TextBase) Q1() int {
	return tb.Selection.Q1()
}

// SetQ0 sets the start of the selection.
func (tb *TextBase) SetQ0(q0 int) {
	tb.Selection.SetQ0(q0)
}

// SetQ1 sets the end of the selection.
func (tb *TextBase) SetQ1(q1 int) {
	tb.Selection.SetQ1(q1)
}

// Org returns the origin.
func (tb *TextBase) Org() int {
	return tb.Display.Org()
}

// SetOrg sets the origin.
func (tb *TextBase) SetOrg(org int) {
	tb.Display.SetOrg(org)
}

// IQ1 returns the initial q1 value (insertion point marker).
func (tb *TextBase) IQ1() int {
	return tb.Edit.IQ1()
}

// SetIQ1 sets the initial q1 value.
func (tb *TextBase) SetIQ1(iq1 int) {
	tb.Edit.SetIQ1(iq1)
}

// EQ0 returns the eq0 value (editing start marker).
func (tb *TextBase) EQ0() int {
	return tb.Edit.EQ0()
}

// SetEQ0 sets the eq0 value.
func (tb *TextBase) SetEQ0(eq0 int) {
	tb.Edit.SetEQ0(eq0)
}

// NeedsRedraw returns true if the text needs to be redrawn.
func (tb *TextBase) NeedsRedraw() bool {
	return tb.Display.NeedsRedraw()
}

// SetNeedsRedraw marks the text as needing redraw.
func (tb *TextBase) SetNeedsRedraw(needs bool) {
	tb.Display.SetNeedsRedraw(needs)
}

// ClearRedrawFlag clears the redraw flag after drawing.
func (tb *TextBase) ClearRedrawFlag() {
	tb.Display.ClearRedrawFlag()
}

// HasSelection returns true if there is a non-empty selection.
func (tb *TextBase) HasSelection() bool {
	return tb.Selection.HasSelection()
}

// ClearSelection collapses the selection to the cursor position.
func (tb *TextBase) ClearSelection() {
	tb.Selection.ClearSelection()
}

// Text is the interface that defines the core text view operations.
// This interface will be implemented by the main Text type once it is
// moved to this package.
type Text interface {
	// Selection accessors
	Q0() int
	Q1() int
	SetQ0(q0 int)
	SetQ1(q1 int)
	HasSelection() bool
	ClearSelection()

	// Display accessors
	Org() int
	SetOrg(org int)
	NeedsRedraw() bool
	SetNeedsRedraw(needs bool)
	ClearRedrawFlag()

	// Edit state accessors
	IQ1() int
	SetIQ1(iq1 int)
	EQ0() int
	SetEQ0(eq0 int)

	// Tab settings
	TabStop() int
	SetTabStop(tabstop int)
	TabExpand() bool
	SetTabExpand(expand bool)

	// Fill control
	NoFill() bool
	SetNoFill(nofill bool)
}

// Compile-time check that TextBase implements the Text interface
var _ Text = (*TextBase)(nil)
