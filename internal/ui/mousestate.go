// Package ui provides user interface utilities for edwood.
package ui

import "image"

// MouseMover is an interface for types that can move the mouse cursor.
// This is implemented by display types in the draw package.
type MouseMover interface {
	MoveTo(pt image.Point) error
}

// MouseWindow is an interface for window types that have a display.
// This allows the MouseState to work with any window implementation.
type MouseWindow interface {
	Display() MouseMover
}

// MouseState manages saved mouse position state for cursor restoration.
// It replaces the previous global prevmouse/mousew variables with
// a proper encapsulated type.
type MouseState struct {
	point  image.Point
	window MouseWindow
}

// NewMouseState creates a new MouseState with no saved state.
func NewMouseState() *MouseState {
	return &MouseState{}
}

// Save stores the current mouse position and associated window.
// This overwrites any previously saved state.
func (ms *MouseState) Save(w MouseWindow, pt image.Point) {
	ms.point = pt
	ms.window = w
}

// Clear removes any saved mouse state.
func (ms *MouseState) Clear() {
	ms.window = nil
}

// HasSaved returns true if there is saved mouse state.
func (ms *MouseState) HasSaved() bool {
	return ms.window != nil
}

// Restore moves the mouse cursor to the saved position if the given window
// matches the saved window. Returns true if the cursor was moved.
// The saved state is always cleared after calling Restore, regardless
// of whether the cursor was moved.
func (ms *MouseState) Restore(w MouseWindow) bool {
	defer func() { ms.window = nil }()

	if ms.window != nil && ms.window == w && w != nil {
		w.Display().MoveTo(ms.point)
		return true
	}
	return false
}
