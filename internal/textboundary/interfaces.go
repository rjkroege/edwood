// Package textboundary provides interfaces to decouple Text from Window internals.
// These interfaces help fix layering violations noted at text.go:1151 and text.go:1658.
package textboundary

import "image"

// MouseReader abstracts mouse input reading for selection operations.
// This interface helps fix the layering violation at text.go:1151 where
// Text.Select() directly calls global.mousectl.Read() in a loop.
type MouseReader interface {
	// Read blocks until a mouse event is available and updates the mouse state.
	Read()
}

// MouseState abstracts access to current mouse state.
// This interface helps fix the layering violation at text.go:1151 where
// Text.Select() directly accesses global.mouse.Buttons and global.mouse.Point.
type MouseState interface {
	// Buttons returns the current mouse button state.
	Buttons() int

	// Point returns the current mouse position.
	Point() image.Point

	// Msec returns the timestamp of the last mouse event in milliseconds.
	Msec() uint32
}

// MouseWaiter combines MouseReader and MouseState for waiting on mouse events.
// This provides a clean interface for Text to wait for mouse movement/button changes
// without directly accessing global state.
type MouseWaiter interface {
	// WaitForChange blocks until the mouse state changes (buttons or position).
	// Returns true if the wait was interrupted, false otherwise.
	// The threshold parameter specifies the minimum pixel movement to detect.
	// originalButton is the button state to compare against.
	WaitForChange(originalButton int, originalPos image.Point, threshold int) bool

	// CurrentState returns the current mouse state.
	CurrentState() MouseState
}

// MouseWaiterFunc is a function adapter for simple WaitForChange implementations.
type MouseWaiterFunc func(originalButton int, originalPos image.Point, threshold int) bool

func (f MouseWaiterFunc) WaitForChange(originalButton int, originalPos image.Point, threshold int) bool {
	return f(originalButton, originalPos, threshold)
}

// DirectoryResolver abstracts the logic for resolving relative paths to absolute paths.
// This interface helps fix the layering violation at text.go:1658 where
// Text.dirName() reaches into t.w.tag.file and t.w.ParseTag() to get directory context.
type DirectoryResolver interface {
	// ResolveDir returns the directory context for this text.
	// If the text has no associated directory (nil window, empty tag), returns "".
	ResolveDir() string
}

// TagProvider abstracts access to window tag information for directory resolution.
// This is the counterpart to DirectoryResolver for when Text needs tag content.
type TagProvider interface {
	// TagFileName returns the filename parsed from the window tag.
	// Returns empty string if there is no tag or it's empty.
	TagFileName() string

	// HasTag returns true if this text has an associated tag.
	HasTag() bool
}

// DirectoryContext combines TagProvider with working directory information.
// This provides everything Text needs to resolve relative paths.
type DirectoryContext interface {
	TagProvider

	// WorkingDir returns the global working directory for fallback resolution.
	WorkingDir() string
}

// NilDirectoryResolver is a DirectoryResolver that always returns empty string.
// Use this for Text instances that have no window/directory context.
type NilDirectoryResolver struct{}

func (NilDirectoryResolver) ResolveDir() string {
	return ""
}

// StaticDirectoryResolver is a DirectoryResolver that returns a fixed directory.
type StaticDirectoryResolver struct {
	Dir string
}

func (r StaticDirectoryResolver) ResolveDir() string {
	return r.Dir
}

// FuncDirectoryResolver adapts a function to the DirectoryResolver interface.
type FuncDirectoryResolver func() string

func (f FuncDirectoryResolver) ResolveDir() string {
	return f()
}

// MouseSnapshot captures mouse state at a point in time for comparison.
type MouseSnapshot struct {
	ButtonState int
	Position    image.Point
	Timestamp   uint32
}

// Buttons returns the button state from the snapshot.
func (s MouseSnapshot) Buttons() int {
	return s.ButtonState
}

// Point returns the position from the snapshot.
func (s MouseSnapshot) Point() image.Point {
	return s.Position
}

// Msec returns the timestamp from the snapshot.
func (s MouseSnapshot) Msec() uint32 {
	return s.Timestamp
}

// HasMoved returns true if the position has moved at least threshold pixels.
// Movement of exactly (threshold-1) pixels in each axis is considered within bounds.
// This matches the acme behavior at text.go:1154 which uses `util.Abs(delta) < threshold`.
func (s MouseSnapshot) HasMoved(other image.Point, threshold int) bool {
	dx := s.Position.X - other.X
	if dx < 0 {
		dx = -dx
	}
	dy := s.Position.Y - other.Y
	if dy < 0 {
		dy = -dy
	}
	// Use > to match original acme behavior (breaking when delta >= threshold)
	return dx >= threshold || dy >= threshold
}

// ButtonsChanged returns true if the button state differs from the given state.
func (s MouseSnapshot) ButtonsChanged(buttons int) bool {
	return s.ButtonState != buttons
}
