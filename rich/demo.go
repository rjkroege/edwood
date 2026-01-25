package rich

import (
	"image"

	"github.com/rjkroege/edwood/draw"
)

// DemoFrame creates and draws a rich.Frame for visual testing.
// This is a temporary hook for development - remove when no longer needed.
// It creates a frame showing plain text in the bottom-right corner.
func DemoFrame(display draw.Display, screenR image.Rectangle, font draw.Font) {
	// Create a frame in the bottom-right corner
	// Size: 300x200 pixels (larger to show text)
	frameWidth := 300
	frameHeight := 200
	margin := 20

	r := image.Rect(
		screenR.Max.X-frameWidth-margin,
		screenR.Max.Y-frameHeight-margin,
		screenR.Max.X-margin,
		screenR.Max.Y-margin,
	)

	// Ensure rectangle is valid (in case screen is too small)
	if r.Min.X < screenR.Min.X {
		r.Min.X = screenR.Min.X + margin
		r.Max.X = r.Min.X + frameWidth
	}
	if r.Min.Y < screenR.Min.Y {
		r.Min.Y = screenR.Min.Y + margin
		r.Max.Y = r.Min.Y + frameHeight
	}

	// Allocate a distinct background color (light yellow for readability)
	bgColor := draw.Color(0xFFFFCCFF) // Light yellow: R=255, G=255, B=204, A=255
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		return
	}

	// Allocate text color (black)
	textColor := draw.Black
	textImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, textColor)
	if err != nil {
		return
	}

	// If no font provided, fall back to background-only display
	if font == nil {
		f := NewFrame()
		f.Init(r, withDisplay(display), withBackground(bgImage))
		f.Redraw()
		return
	}

	// Create and initialize the frame with font and text color
	f := NewFrame()
	f.Init(r, withDisplay(display), withBackground(bgImage), withFont(font), withTextColor(textImage))

	// Set demo content - plain text with multiple lines
	demoText := `Rich Text Demo
===============

This is a test of the
rich text frame rendering.

Features:
- Line wrapping
- Multiple lines
- Tab	stops

The quick brown fox
jumps over the lazy dog.`

	f.SetContent(Plain(demoText))

	// Draw the frame
	f.Redraw()
}

// withDisplay is an Option that sets the display for the frame.
func withDisplay(d draw.Display) Option {
	return func(f *frameImpl) {
		f.display = d
	}
}

// withBackground is an Option that sets the background image for the frame.
func withBackground(b draw.Image) Option {
	return func(f *frameImpl) {
		f.background = b
	}
}

// withFont is an Option that sets the font for the frame.
func withFont(fnt draw.Font) Option {
	return func(f *frameImpl) {
		f.font = fnt
	}
}

// withTextColor is an Option that sets the text color for the frame.
func withTextColor(c draw.Image) Option {
	return func(f *frameImpl) {
		f.textColor = c
	}
}
