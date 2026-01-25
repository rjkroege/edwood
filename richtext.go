package main

import (
	"image"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/rich"
)

// scrollbarWidth is the width of the scrollbar in pixels.
const scrollbarWidth = 12

// scrollbarGap is the gap between the scrollbar and the frame.
const scrollbarGap = 4

// RichText is a component that combines a rich.Frame with a scrollbar.
// It manages the layout of the scrollbar area and the text frame area.
type RichText struct {
	all        image.Rectangle // Full area including scrollbar
	scrollRect image.Rectangle // Scrollbar area
	display    draw.Display
	frame      rich.Frame
	content    rich.Content

	// Options stored for frame initialization
	background draw.Image
	textColor  draw.Image
}

// NewRichText creates a new RichText component.
func NewRichText() *RichText {
	return &RichText{}
}

// Init initializes the RichText component with the given rectangle, display, font, and options.
func (rt *RichText) Init(r image.Rectangle, display draw.Display, font draw.Font, opts ...RichTextOption) {
	rt.all = r
	rt.display = display

	// Apply options
	for _, opt := range opts {
		opt(rt)
	}

	// Calculate scrollbar rectangle (left side)
	rt.scrollRect = image.Rect(
		r.Min.X,
		r.Min.Y,
		r.Min.X+scrollbarWidth,
		r.Max.Y,
	)

	// Calculate frame rectangle (right of scrollbar with gap)
	frameRect := image.Rect(
		r.Min.X+scrollbarWidth+scrollbarGap,
		r.Min.Y,
		r.Max.X,
		r.Max.Y,
	)

	// Create and initialize the frame
	rt.frame = rich.NewFrame()

	// Build frame options
	frameOpts := []rich.Option{
		rich.WithDisplay(display),
		rich.WithFont(font),
	}
	if rt.background != nil {
		frameOpts = append(frameOpts, rich.WithBackground(rt.background))
	}
	if rt.textColor != nil {
		frameOpts = append(frameOpts, rich.WithTextColor(rt.textColor))
	}

	rt.frame.Init(frameRect, frameOpts...)
}

// All returns the full rectangle area of the RichText component.
func (rt *RichText) All() image.Rectangle {
	return rt.all
}

// Frame returns the underlying rich.Frame.
func (rt *RichText) Frame() rich.Frame {
	return rt.frame
}

// Display returns the display.
func (rt *RichText) Display() draw.Display {
	return rt.display
}

// ScrollRect returns the scrollbar rectangle.
func (rt *RichText) ScrollRect() image.Rectangle {
	return rt.scrollRect
}

// SetContent sets the content to display.
func (rt *RichText) SetContent(c rich.Content) {
	rt.content = c
	if rt.frame != nil {
		rt.frame.SetContent(c)
	}
}

// Content returns the current content.
func (rt *RichText) Content() rich.Content {
	return rt.content
}

// Selection returns the current selection range.
func (rt *RichText) Selection() (p0, p1 int) {
	if rt.frame == nil {
		return 0, 0
	}
	return rt.frame.GetSelection()
}

// SetSelection sets the selection range.
func (rt *RichText) SetSelection(p0, p1 int) {
	if rt.frame != nil {
		rt.frame.SetSelection(p0, p1)
	}
}

// Origin returns the current scroll origin.
func (rt *RichText) Origin() int {
	if rt.frame == nil {
		return 0
	}
	return rt.frame.GetOrigin()
}

// SetOrigin sets the scroll origin.
func (rt *RichText) SetOrigin(org int) {
	if rt.frame != nil {
		rt.frame.SetOrigin(org)
	}
}

// Redraw redraws the RichText component.
func (rt *RichText) Redraw() {
	if rt.frame != nil {
		rt.frame.Redraw()
	}
	// TODO: Also redraw scrollbar in future phase
}
