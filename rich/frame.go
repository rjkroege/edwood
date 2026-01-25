package rich

import (
	"image"

	"9fans.net/go/draw"
	edwooddraw "github.com/rjkroege/edwood/draw"
)

// Option is a functional option for configuring a Frame.
type Option func(*frameImpl)

// Frame renders styled text content with selection support.
type Frame interface {
	// Initialization
	Init(r image.Rectangle, opts ...Option)
	Clear()

	// Content
	SetContent(c Content)

	// Geometry
	Rect() image.Rectangle
	Ptofchar(p int) image.Point  // Character position → screen point
	Charofpt(pt image.Point) int // Screen point → character position

	// Selection
	Select(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int)
	SetSelection(p0, p1 int)
	GetSelection() (p0, p1 int)

	// Scrolling
	SetOrigin(org int)
	GetOrigin() int
	MaxLines() int
	VisibleLines() int

	// Rendering
	Redraw()

	// Status
	Full() bool // True if frame is at capacity
}

// frameImpl is the concrete implementation of Frame.
type frameImpl struct {
	rect       image.Rectangle
	display    edwooddraw.Display
	background edwooddraw.Image // background image for filling
	font       edwooddraw.Font  // font for text rendering
	content    Content
	origin     int
	p0, p1     int // selection
}

// NewFrame creates a new Frame.
func NewFrame() Frame {
	return &frameImpl{}
}

// Init initializes the frame with the given rectangle and options.
func (f *frameImpl) Init(r image.Rectangle, opts ...Option) {
	f.rect = r
	for _, opt := range opts {
		opt(f)
	}
}

// Clear resets the frame.
func (f *frameImpl) Clear() {
	f.content = nil
	f.origin = 0
	f.p0 = 0
	f.p1 = 0
}

// SetContent sets the content to display.
func (f *frameImpl) SetContent(c Content) {
	f.content = c
}

// Rect returns the frame's rectangle.
func (f *frameImpl) Rect() image.Rectangle {
	return f.rect
}

// Ptofchar maps a character position to a screen point.
func (f *frameImpl) Ptofchar(p int) image.Point {
	// TODO: Implement
	return f.rect.Min
}

// Charofpt maps a screen point to a character position.
func (f *frameImpl) Charofpt(pt image.Point) int {
	// TODO: Implement
	return 0
}

// Select handles mouse selection.
func (f *frameImpl) Select(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int) {
	// TODO: Implement
	return f.p0, f.p1
}

// SetSelection sets the selection range.
func (f *frameImpl) SetSelection(p0, p1 int) {
	f.p0 = p0
	f.p1 = p1
}

// GetSelection returns the current selection range.
func (f *frameImpl) GetSelection() (p0, p1 int) {
	return f.p0, f.p1
}

// SetOrigin sets the scroll origin.
func (f *frameImpl) SetOrigin(org int) {
	f.origin = org
}

// GetOrigin returns the current scroll origin.
func (f *frameImpl) GetOrigin() int {
	return f.origin
}

// MaxLines returns the maximum number of lines that can be displayed.
func (f *frameImpl) MaxLines() int {
	// TODO: Implement
	return 0
}

// VisibleLines returns the number of lines currently visible.
func (f *frameImpl) VisibleLines() int {
	// TODO: Implement
	return 0
}

// Redraw redraws the frame.
func (f *frameImpl) Redraw() {
	if f.display == nil || f.background == nil {
		return
	}
	// Fill the frame rectangle with the background color
	screen := f.display.ScreenImage()
	screen.Draw(f.rect, f.background, f.background, image.ZP)
}

// Full returns true if the frame is at capacity.
func (f *frameImpl) Full() bool {
	// TODO: Implement
	return false
}
