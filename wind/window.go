// Package wind provides the Window type and related components for edwood.
//
// This package provides:
//   - Window: an interface defining window operations
//   - WindowBase: a composable struct containing portable window state
//   - WindowState: file descriptor tracking, addresses, dirty flags
//   - PreviewState: preview mode fields for rich text rendering
//   - DrawState: drawing state for window rendering
//   - EventState: event handling state
//
// The Window interface allows the main package's Window type to be used
// polymorphically, while WindowBase provides shared state that can be
// embedded in concrete implementations.
package wind

import (
	"image"
)

// Window defines the interface for window operations.
// This interface is implemented by the main package's Window type
// and allows polymorphic use of windows across packages.
type Window interface {
	// ID returns the window's unique identifier.
	ID() int

	// IsPreviewMode returns true if the window is showing a rendered preview.
	IsPreviewMode() bool

	// SetPreviewMode enables or disables preview mode.
	SetPreviewMode(enabled bool)

	// Rect returns the window's bounding rectangle.
	Rect() image.Rectangle

	// IsDirty returns true if the window has unsaved changes.
	IsDirty() bool
}

// WindowBase contains the portable state for a window that can be
// composed into concrete Window implementations. This struct uses
// the state types defined in this package.
//
// The main package's Window type can embed WindowBase to share
// common state management while still having access to main-package
// types (Text, Column, etc.) that cannot be moved to this package
// due to circular dependency constraints.
type WindowBase struct {
	// State contains file descriptor tracking, addresses, and dirty flags.
	State *WindowState

	// Draw contains drawing-related state.
	Draw *DrawState

	// Events contains event handling state.
	Events *EventState

	// Preview contains preview mode state for rich text rendering.
	Preview *PreviewState

	// id is the window's unique identifier.
	id int

	// rect is the window's bounding rectangle.
	rect image.Rectangle
}

// NewWindowBase creates a new WindowBase with initialized state.
func NewWindowBase() *WindowBase {
	return &WindowBase{
		State:   NewWindowState(),
		Draw:    NewDrawState(),
		Events:  NewEventState(),
		Preview: NewPreviewState(),
	}
}

// ID returns the window's unique identifier.
func (wb *WindowBase) ID() int {
	return wb.id
}

// SetID sets the window's unique identifier.
func (wb *WindowBase) SetID(id int) {
	wb.id = id
}

// Rect returns the window's bounding rectangle.
func (wb *WindowBase) Rect() image.Rectangle {
	return wb.rect
}

// SetRect sets the window's bounding rectangle.
func (wb *WindowBase) SetRect(r image.Rectangle) {
	wb.rect = r
	wb.Draw.SetRect(r)
}

// IsPreviewMode returns true if the window is in preview mode.
func (wb *WindowBase) IsPreviewMode() bool {
	return wb.Preview.IsPreviewMode()
}

// SetPreviewMode enables or disables preview mode.
func (wb *WindowBase) SetPreviewMode(enabled bool) {
	wb.Preview.SetPreviewMode(enabled)
	wb.Draw.SetPreviewMode(enabled)
}

// IsDirty returns true if the window has unsaved changes.
// This delegates to the WindowState.
func (wb *WindowBase) IsDirty() bool {
	return wb.State.IsDirty()
}

// SetDirty sets the dirty flag for the window.
// This updates both State and Draw to keep them in sync.
func (wb *WindowBase) SetDirty(dirty bool) {
	wb.State.SetDirty(dirty)
	wb.Draw.SetDirty(dirty)
}

// Addr returns the current address range.
func (wb *WindowBase) Addr() Range {
	return wb.State.Addr()
}

// SetAddr sets the address range.
func (wb *WindowBase) SetAddr(addr Range) {
	wb.State.SetAddr(addr)
}

// Limit returns the current limit range.
func (wb *WindowBase) Limit() Range {
	return wb.State.Limit()
}

// SetLimit sets the limit range.
func (wb *WindowBase) SetLimit(limit Range) {
	wb.State.SetLimit(limit)
}

// Nomark returns true if marking is disabled.
func (wb *WindowBase) Nomark() bool {
	return wb.State.Nomark()
}

// SetNomark sets the nomark flag.
func (wb *WindowBase) SetNomark(nomark bool) {
	wb.State.SetNomark(nomark)
}

// NeedsRedraw returns true if the window needs to be redrawn.
func (wb *WindowBase) NeedsRedraw() bool {
	return wb.Draw.NeedsRedraw()
}

// ClearRedrawFlag clears the redraw flag after drawing.
func (wb *WindowBase) ClearRedrawFlag() {
	wb.Draw.ClearRedrawFlag()
}

// TagLines returns the number of lines in the tag.
func (wb *WindowBase) TagLines() int {
	return wb.Draw.TagLines()
}

// SetTagLines sets the number of tag lines.
func (wb *WindowBase) SetTagLines(n int) {
	wb.Draw.SetTagLines(n)
}

// TagExpand returns whether the tag can expand beyond one line.
func (wb *WindowBase) TagExpand() bool {
	return wb.Draw.TagExpand()
}

// SetTagExpand sets whether the tag can expand.
func (wb *WindowBase) SetTagExpand(expand bool) {
	wb.Draw.SetTagExpand(expand)
}

// MaxLines returns the maximum visible lines in the body.
func (wb *WindowBase) MaxLines() int {
	return wb.Draw.MaxLines()
}

// SetMaxLines sets the maximum visible lines.
func (wb *WindowBase) SetMaxLines(n int) {
	wb.Draw.SetMaxLines(n)
}

// BodyRect returns the body area rectangle.
func (wb *WindowBase) BodyRect() image.Rectangle {
	return wb.Draw.BodyRect()
}

// SetBodyRect sets the body area rectangle.
func (wb *WindowBase) SetBodyRect(r image.Rectangle) {
	wb.Draw.SetBodyRect(r)
}

// TagRect returns the tag area rectangle.
func (wb *WindowBase) TagRect() image.Rectangle {
	return wb.Draw.TagRect()
}

// SetTagRect sets the tag area rectangle.
func (wb *WindowBase) SetTagRect(r image.Rectangle) {
	wb.Draw.SetTagRect(r)
}

// ButtonRect returns the dirty indicator button rectangle.
func (wb *WindowBase) ButtonRect() image.Rectangle {
	return wb.Draw.ButtonRect()
}

// SetButtonRect sets the dirty indicator button rectangle.
func (wb *WindowBase) SetButtonRect(r image.Rectangle) {
	wb.Draw.SetButtonRect(r)
}

// ClickState returns the last click position and timestamp for double-click detection.
func (wb *WindowBase) ClickState() (pos int, msec uint32) {
	return wb.Preview.ClickState()
}

// SetClickState updates the click state for double-click detection.
func (wb *WindowBase) SetClickState(pos int, msec uint32) {
	wb.Preview.SetClickState(pos, msec)
}

// ClearPreviewCache clears cached preview data.
func (wb *WindowBase) ClearPreviewCache() {
	wb.Preview.ClearCache()
}

// Reset resets all event state to default values.
func (wb *WindowBase) ResetEventState() {
	wb.Events.Reset()
}

// UpdateMouseRegion updates the mouse region flags based on position.
func (wb *WindowBase) UpdateMouseRegion(pos image.Point, tagRect, bodyRect, scrollRect image.Rectangle) {
	wb.Events.UpdateMouseRegion(pos, tagRect, bodyRect, scrollRect)
}

// IsMouseInTag returns true if the mouse is in the tag area.
func (wb *WindowBase) IsMouseInTag() bool {
	return wb.Events.IsMouseInTag()
}

// IsMouseInBody returns true if the mouse is in the body area.
func (wb *WindowBase) IsMouseInBody() bool {
	return wb.Events.IsMouseInBody()
}

// IsMouseInScrollbar returns true if the mouse is in the scrollbar area.
func (wb *WindowBase) IsMouseInScrollbar() bool {
	return wb.Events.IsMouseInScrollbar()
}

// Verify that WindowBase implements Window interface.
var _ Window = (*WindowBase)(nil)
