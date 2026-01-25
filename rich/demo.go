package rich

import (
	"image"
	"image/color"

	"github.com/rjkroege/edwood/draw"
)

// DemoFrameOptions holds optional font variants for the demo frame.
type DemoFrameOptions struct {
	BoldFont       draw.Font
	ItalicFont     draw.Font
	BoldItalicFont draw.Font
}

// DemoFrame creates and draws a rich.Frame for visual testing.
// This is a temporary hook for development - remove when no longer needed.
// It creates a frame showing styled text in the bottom-right corner.
// The optional opts parameter allows passing font variants for styled text.
func DemoFrame(display draw.Display, screenR image.Rectangle, font draw.Font, opts ...DemoFrameOptions) {
	// Create a frame in the bottom-right corner
	// Size: 350x250 pixels (larger to show styled text)
	frameWidth := 350
	frameHeight := 250
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

	// Build frame options
	frameOpts := []Option{
		withDisplay(display),
		withBackground(bgImage),
		withFont(font),
		withTextColor(textImage),
	}

	// Add font variants if provided
	if len(opts) > 0 {
		o := opts[0]
		if o.BoldFont != nil {
			frameOpts = append(frameOpts, WithBoldFont(o.BoldFont))
		}
		if o.ItalicFont != nil {
			frameOpts = append(frameOpts, WithItalicFont(o.ItalicFont))
		}
		if o.BoldItalicFont != nil {
			frameOpts = append(frameOpts, WithBoldItalicFont(o.BoldItalicFont))
		}
	}

	// Create and initialize the frame with font and text color
	f := NewFrame()
	f.Init(r, frameOpts...)

	// Set demo content - styled text with multiple styles
	f.SetContent(createStyledDemoContent())

	// Draw the frame
	f.Redraw()
}

// createStyledDemoContent creates Content with various styles for demonstration.
func createStyledDemoContent() Content {
	// Define some colors
	darkBlue := color.RGBA{R: 0, G: 0, B: 139, A: 255}
	darkGreen := color.RGBA{R: 0, G: 100, B: 0, A: 255}
	darkRed := color.RGBA{R: 139, G: 0, B: 0, A: 255}

	return Content{
		// H1 heading
		{Text: "Styled Text Demo", Style: StyleH1},
		{Text: "\n\n", Style: DefaultStyle()},

		// Regular paragraph
		{Text: "This is ", Style: DefaultStyle()},
		{Text: "bold text", Style: StyleBold},
		{Text: " and ", Style: DefaultStyle()},
		{Text: "italic text", Style: StyleItalic},
		{Text: ".\n\n", Style: DefaultStyle()},

		// H2 heading
		{Text: "Colors", Style: StyleH2},
		{Text: "\n", Style: DefaultStyle()},

		// Colored text
		{Text: "Blue text", Style: Style{Fg: darkBlue, Scale: 1.0}},
		{Text: ", ", Style: DefaultStyle()},
		{Text: "green text", Style: Style{Fg: darkGreen, Scale: 1.0}},
		{Text: ", ", Style: DefaultStyle()},
		{Text: "red text", Style: Style{Fg: darkRed, Scale: 1.0}},
		{Text: ".\n\n", Style: DefaultStyle()},

		// H3 heading
		{Text: "Combined Styles", Style: StyleH3},
		{Text: "\n", Style: DefaultStyle()},

		// Colored + bold
		{Text: "Bold blue", Style: Style{Fg: darkBlue, Bold: true, Scale: 1.0}},
		{Text: " and ", Style: DefaultStyle()},
		{Text: "italic green", Style: Style{Fg: darkGreen, Italic: true, Scale: 1.0}},
		{Text: ".\n\n", Style: DefaultStyle()},

		// Plain text
		{Text: "The quick brown fox\njumps over the lazy dog.", Style: DefaultStyle()},
	}
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
