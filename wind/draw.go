// Package wind provides the Window type and related components for edwood.
// This file contains drawing-related types and methods for windows.
package wind

import (
	"image"
)

// DrawContext represents the drawing context needed by a window.
// This interface abstracts the drawing operations used when rendering a window.
type DrawContext interface {
	// Rect returns the current drawing rectangle.
	Rect() image.Rectangle
	// IsPreviewMode returns true if in preview mode.
	IsPreviewMode() bool
}

// DrawState tracks the state needed for window drawing operations.
// This encapsulates drawing-related state that was previously part of Window.
type DrawState struct {
	dirty          bool            // whether the window has unsaved changes
	rect           image.Rectangle // window rectangle
	tagRect        image.Rectangle // tag area rectangle
	bodyRect       image.Rectangle // body area rectangle
	buttonRect     image.Rectangle // dirty indicator button rectangle
	maxLines       int             // maximum visible lines in body
	previewMode    bool            // true when showing rendered preview
	needsRedraw    bool            // true when a redraw is pending
	tagLines       int             // number of lines in the tag
	tagExpand      bool            // whether tag can expand beyond one line
}

// NewDrawState creates a new DrawState with default values.
func NewDrawState() *DrawState {
	return &DrawState{
		tagLines:  1,
		tagExpand: true,
	}
}

// IsDirty returns true if the window has unsaved changes.
func (ds *DrawState) IsDirty() bool {
	return ds.dirty
}

// SetDirty sets the dirty flag.
func (ds *DrawState) SetDirty(dirty bool) {
	ds.needsRedraw = ds.dirty != dirty
	ds.dirty = dirty
}

// Rect returns the window rectangle.
func (ds *DrawState) Rect() image.Rectangle {
	return ds.rect
}

// SetRect sets the window rectangle.
func (ds *DrawState) SetRect(r image.Rectangle) {
	if !ds.rect.Eq(r) {
		ds.needsRedraw = true
	}
	ds.rect = r
}

// TagRect returns the tag area rectangle.
func (ds *DrawState) TagRect() image.Rectangle {
	return ds.tagRect
}

// SetTagRect sets the tag area rectangle.
func (ds *DrawState) SetTagRect(r image.Rectangle) {
	ds.tagRect = r
}

// BodyRect returns the body area rectangle.
func (ds *DrawState) BodyRect() image.Rectangle {
	return ds.bodyRect
}

// SetBodyRect sets the body area rectangle.
func (ds *DrawState) SetBodyRect(r image.Rectangle) {
	ds.bodyRect = r
}

// ButtonRect returns the dirty indicator button rectangle.
func (ds *DrawState) ButtonRect() image.Rectangle {
	return ds.buttonRect
}

// SetButtonRect sets the dirty indicator button rectangle.
func (ds *DrawState) SetButtonRect(r image.Rectangle) {
	ds.buttonRect = r
}

// MaxLines returns the maximum visible lines in the body.
func (ds *DrawState) MaxLines() int {
	return ds.maxLines
}

// SetMaxLines sets the maximum visible lines.
func (ds *DrawState) SetMaxLines(n int) {
	ds.maxLines = n
}

// IsPreviewMode returns true if in preview mode.
func (ds *DrawState) IsPreviewMode() bool {
	return ds.previewMode
}

// SetPreviewMode sets the preview mode state.
func (ds *DrawState) SetPreviewMode(mode bool) {
	if ds.previewMode != mode {
		ds.needsRedraw = true
	}
	ds.previewMode = mode
}

// NeedsRedraw returns true if the window needs to be redrawn.
func (ds *DrawState) NeedsRedraw() bool {
	return ds.needsRedraw
}

// ClearRedrawFlag clears the redraw flag after drawing.
func (ds *DrawState) ClearRedrawFlag() {
	ds.needsRedraw = false
}

// TagLines returns the number of lines in the tag.
func (ds *DrawState) TagLines() int {
	return ds.tagLines
}

// SetTagLines sets the number of tag lines.
func (ds *DrawState) SetTagLines(n int) {
	if n < 1 {
		n = 1
	}
	ds.tagLines = n
}

// TagExpand returns whether the tag can expand beyond one line.
func (ds *DrawState) TagExpand() bool {
	return ds.tagExpand
}

// SetTagExpand sets whether the tag can expand.
func (ds *DrawState) SetTagExpand(expand bool) {
	ds.tagExpand = expand
}
