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

	// Scrollbar colors
	scrollBg    draw.Image // Scrollbar background color
	scrollThumb draw.Image // Scrollbar thumb color
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
	// Draw scrollbar first (behind frame)
	rt.scrDraw()

	// Draw the frame content
	if rt.frame != nil {
		rt.frame.Redraw()
	}
}

// scrDraw renders the scrollbar background and thumb.
func (rt *RichText) scrDraw() {
	if rt.display == nil {
		return
	}

	screen := rt.display.ScreenImage()

	// Draw scrollbar background
	if rt.scrollBg != nil {
		screen.Draw(rt.scrollRect, rt.scrollBg, rt.scrollBg, image.ZP)
	}

	// Draw scrollbar thumb
	if rt.scrollThumb != nil {
		thumbRect := rt.scrThumbRect()
		screen.Draw(thumbRect, rt.scrollThumb, rt.scrollThumb, image.ZP)
	}
}

// scrThumbRect returns the rectangle for the scrollbar thumb.
// The thumb position and size reflect the current scroll position and
// the proportion of visible content to total content.
func (rt *RichText) scrThumbRect() image.Rectangle {
	// If no content or frame, fill the whole scrollbar
	if rt.content == nil || rt.frame == nil {
		return rt.scrollRect
	}

	totalRunes := rt.content.Len()
	if totalRunes == 0 {
		// No content - thumb fills the whole scrollbar
		return rt.scrollRect
	}

	// Get scroll metrics from the frame
	origin := rt.frame.GetOrigin()
	maxLines := rt.frame.MaxLines()

	scrollHeight := rt.scrollRect.Dy()

	// Count lines in content and build a map of line start positions
	lineCount := 1 // At least one line
	lineStarts := []int{0}
	for i, span := range rt.content {
		runeOffset := 0
		if i > 0 {
			// Sum up runes from previous spans
			for j := 0; j < i; j++ {
				runeOffset += len([]rune(rt.content[j].Text))
			}
		}
		for j, r := range span.Text {
			if r == '\n' {
				lineCount++
				lineStarts = append(lineStarts, runeOffset+j+1)
			}
		}
	}

	// If all content fits, fill the scrollbar
	if lineCount <= maxLines {
		return rt.scrollRect
	}

	// Calculate thumb height based on visible vs total lines
	visibleProportion := float64(maxLines) / float64(lineCount)
	if visibleProportion > 1.0 {
		visibleProportion = 1.0
	}

	thumbHeight := int(float64(scrollHeight) * visibleProportion)
	if thumbHeight < 10 {
		thumbHeight = 10 // Minimum thumb height for usability
	}

	// Find which line the origin corresponds to
	originLine := 0
	for i, start := range lineStarts {
		if origin >= start {
			originLine = i
		} else {
			break
		}
	}

	// Position proportion based on line position in the document
	// Use (lineCount - 1) as denominator so that last line maps to bottom.
	denominator := lineCount - 1
	if denominator < 1 {
		denominator = 1
	}
	posProportion := float64(originLine) / float64(denominator)
	if posProportion > 1.0 {
		posProportion = 1.0
	}

	// When viewing content near the end of the document (past ~70% of lines),
	// adjust the position to ensure the thumb reaches the bottom.
	// This ensures "near the end" positions map to "near the bottom" of scrollbar.
	endThreshold := float64(lineCount) * 0.70 // 70% threshold
	if float64(originLine) >= endThreshold {
		// Map from [endThreshold, lineCount-1] to [currentProportion, 1.0]
		// Scale faster toward 1.0 for end positions
		linesBeyondThreshold := float64(originLine) - endThreshold
		linesInEndRange := float64(lineCount-1) - endThreshold
		if linesInEndRange > 0 {
			// Linear approach to bottom - more aggressive than ease-out
			normalizedPos := linesBeyondThreshold / linesInEndRange
			// Remap: at normalizedPos 0.5, we want to be 90%+ of the way to the bottom
			adjustment := normalizedPos * (1.0 - posProportion)
			posProportion += adjustment
		}
	}

	// Available space for thumb movement
	availableSpace := scrollHeight - thumbHeight

	// Thumb top position
	thumbTop := rt.scrollRect.Min.Y + int(float64(availableSpace)*posProportion)

	return image.Rect(
		rt.scrollRect.Min.X,
		thumbTop,
		rt.scrollRect.Max.X,
		thumbTop+thumbHeight,
	)
}
