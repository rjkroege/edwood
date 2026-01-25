package rich

import (
	"image"

	"github.com/rjkroege/edwood/draw"
)

// DemoFrame creates and draws a rich.Frame for visual testing.
// This is a temporary hook for development - remove when no longer needed.
// It creates a 200x150 pixel magenta rectangle in the specified display area.
func DemoFrame(display draw.Display, screenR image.Rectangle) {
	// Create a frame in the bottom-right corner
	// Size: 200x150 pixels
	frameWidth := 200
	frameHeight := 150
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

	// Allocate a distinct background color (magenta for high visibility)
	bgColor := draw.Color(0xFF00FFFF) // Magenta: R=255, G=0, B=255, A=255
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		// Silently fail if we can't allocate the image
		return
	}

	// Create and initialize the frame
	f := NewFrame()
	f.Init(r, withDisplay(display), withBackground(bgImage))

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
