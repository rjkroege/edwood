// Package wind provides the Window type and related components for edwood.
package wind

// Range represents a start/end position pair used for addresses and limits.
type Range struct {
	Start int
	End   int
}

// WindowState holds file descriptor tracking, addresses, and dirty flags
// for a window. This encapsulates state that was previously spread across
// multiple fields in the Window struct.
type WindowState struct {
	dirty  bool
	addr   Range
	limit  Range
	nomark bool
}

// NewWindowState creates a new WindowState with default values.
func NewWindowState() *WindowState {
	return &WindowState{}
}

// IsDirty returns true if the window has unsaved changes.
func (ws *WindowState) IsDirty() bool {
	return ws.dirty
}

// SetDirty sets the dirty flag for the window.
func (ws *WindowState) SetDirty(dirty bool) {
	ws.dirty = dirty
}

// Addr returns the current address range.
func (ws *WindowState) Addr() Range {
	return ws.addr
}

// SetAddr sets the address range.
func (ws *WindowState) SetAddr(addr Range) {
	ws.addr = addr
}

// Limit returns the current limit range.
func (ws *WindowState) Limit() Range {
	return ws.limit
}

// SetLimit sets the limit range.
func (ws *WindowState) SetLimit(limit Range) {
	ws.limit = limit
}

// Nomark returns true if marking is disabled.
func (ws *WindowState) Nomark() bool {
	return ws.nomark
}

// SetNomark sets the nomark flag.
func (ws *WindowState) SetNomark(nomark bool) {
	ws.nomark = nomark
}
