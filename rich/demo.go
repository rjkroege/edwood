package rich

import (
	"image"
	"image/color"

	"9fans.net/go/draw"
	edwooddraw "github.com/rjkroege/edwood/draw"
)

// DemoFrameOptions holds optional font variants for the demo frame.
type DemoFrameOptions struct {
	BoldFont       edwooddraw.Font
	ItalicFont     edwooddraw.Font
	BoldItalicFont edwooddraw.Font
}

// DemoState holds state for the interactive demo frame.
type DemoState struct {
	Frame   Frame
	Rect    image.Rectangle
	Display edwooddraw.Display
}

// DemoFrame creates and draws a rich.Frame for visual testing.
// This is a temporary hook for development - remove when no longer needed.
// It creates a frame showing styled text in the bottom-right corner.
// The optional opts parameter allows passing font variants for styled text.
// Returns a DemoState that can be used to handle mouse events.
func DemoFrame(display edwooddraw.Display, screenR image.Rectangle, font edwooddraw.Font, opts ...DemoFrameOptions) *DemoState {
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
	bgColor := edwooddraw.Color(0xFFFFCCFF) // Light yellow: R=255, G=255, B=204, A=255
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		return nil
	}

	// Allocate text color (black)
	textColor := edwooddraw.Black
	textImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, textColor)
	if err != nil {
		return nil
	}

	// If no font provided, fall back to background-only display
	if font == nil {
		f := NewFrame()
		f.Init(r, withDisplay(display), withBackground(bgImage))
		f.Redraw()
		return &DemoState{Frame: f, Rect: r, Display: display}
	}

	// Allocate selection color (light blue highlight)
	selColor := edwooddraw.Color(0x9EEEEE99) // Light cyan with some transparency
	selImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, selColor)
	if err != nil {
		selImage = nil
	}

	// Build frame options
	frameOpts := []Option{
		withDisplay(display),
		withBackground(bgImage),
		withFont(font),
		withTextColor(textImage),
		WithSelectionColor(selImage),
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

	return &DemoState{Frame: f, Rect: r, Display: display}
}

// createStyledDemoContent creates Content with various styles for demonstration.
// The content is longer than the frame height to demonstrate scrolling.
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
		{Text: "\n\n", Style: DefaultStyle()},

		// Additional content for scrolling demo
		{Text: "Scrolling Demo", Style: StyleH2},
		{Text: "\n", Style: DefaultStyle()},
		{Text: "Use scroll wheel or button 4/5", Style: DefaultStyle()},
		{Text: "\nto scroll this content up/down.", Style: DefaultStyle()},
		{Text: "\n\n", Style: DefaultStyle()},

		// More lines to ensure content is longer than frame
		{Text: "Line 1: ", Style: StyleBold},
		{Text: "Lorem ipsum dolor sit amet.\n", Style: DefaultStyle()},
		{Text: "Line 2: ", Style: StyleBold},
		{Text: "Consectetur adipiscing elit.\n", Style: DefaultStyle()},
		{Text: "Line 3: ", Style: StyleBold},
		{Text: "Sed do eiusmod tempor.\n", Style: DefaultStyle()},
		{Text: "Line 4: ", Style: StyleBold},
		{Text: "Incididunt ut labore et dolore.\n", Style: DefaultStyle()},
		{Text: "Line 5: ", Style: StyleBold},
		{Text: "Magna aliqua ut enim.\n", Style: DefaultStyle()},
		{Text: "Line 6: ", Style: StyleBold},
		{Text: "Ad minim veniam quis.\n", Style: DefaultStyle()},
		{Text: "Line 7: ", Style: StyleBold},
		{Text: "Nostrud exercitation ullamco.\n", Style: DefaultStyle()},
		{Text: "Line 8: ", Style: StyleBold},
		{Text: "Laboris nisi ut aliquip.\n", Style: DefaultStyle()},
		{Text: "Line 9: ", Style: StyleBold},
		{Text: "Ex ea commodo consequat.\n", Style: DefaultStyle()},
		{Text: "Line 10: ", Style: StyleBold},
		{Text: "Duis aute irure dolor.\n", Style: DefaultStyle()},
		{Text: "\n", Style: DefaultStyle()},

		// End marker
		{Text: "--- End of Content ---", Style: Style{Fg: darkRed, Italic: true, Scale: 1.0}},
	}
}

// withDisplay is an Option that sets the display for the frame.
func withDisplay(d edwooddraw.Display) Option {
	return func(f *frameImpl) {
		f.display = d
	}
}

// withBackground is an Option that sets the background image for the frame.
func withBackground(b edwooddraw.Image) Option {
	return func(f *frameImpl) {
		f.background = b
	}
}

// withFont is an Option that sets the font for the frame.
func withFont(fnt edwooddraw.Font) Option {
	return func(f *frameImpl) {
		f.font = fnt
	}
}

// withTextColor is an Option that sets the text color for the frame.
func withTextColor(c edwooddraw.Image) Option {
	return func(f *frameImpl) {
		f.textColor = c
	}
}

// HandleMouse handles mouse events for the demo frame.
// Returns true if the event was handled (mouse was in the demo area).
// - Button 1: starts a selection
// - Button 4 (scroll up): scrolls content up (decreases origin)
// - Button 5 (scroll down): scrolls content down (increases origin)
func (ds *DemoState) HandleMouse(mc *draw.Mousectl, m *draw.Mouse) bool {
	if ds == nil || ds.Frame == nil {
		return false
	}

	// Check if click is in the demo frame rectangle
	if !m.Point.In(ds.Rect) {
		return false
	}

	// Handle scroll wheel (buttons 4 and 5)
	// Button 4 = scroll up (show earlier content)
	// Button 5 = scroll down (show later content)
	if m.Buttons&8 != 0 { // Button 4 - scroll up
		ds.scrollLines(-3)
		return true
	}
	if m.Buttons&16 != 0 { // Button 5 - scroll down
		ds.scrollLines(3)
		return true
	}

	// Handle button 1 (selection)
	if m.Buttons&1 != 0 {
		ds.Frame.Select(mc, m)
		ds.Frame.Redraw()
		ds.Display.ScreenImage().Display().Flush()
		return true
	}

	return false
}

// scrollLines scrolls the demo frame by the given number of lines.
// Positive n scrolls down (shows later content), negative scrolls up.
func (ds *DemoState) scrollLines(n int) {
	if ds == nil || ds.Frame == nil {
		return
	}

	// Get current origin
	origin := ds.Frame.GetOrigin()

	// Calculate new origin
	// We need to find the rune offset of lines relative to current position
	// For simplicity, we'll estimate based on average characters per line
	// A more accurate approach would track line boundaries

	// Get content length for bounds checking
	content := ds.getContentLength()

	// Approximate characters per line (rough estimate)
	// This scrolls roughly 3 lines worth of content
	charsPerLine := 30
	delta := n * charsPerLine

	newOrigin := origin + delta
	if newOrigin < 0 {
		newOrigin = 0
	}
	if newOrigin > content {
		newOrigin = content
	}

	ds.Frame.SetOrigin(newOrigin)
	ds.Frame.Redraw()
	ds.Display.ScreenImage().Display().Flush()
}

// getContentLength returns the total length of the demo content in runes.
func (ds *DemoState) getContentLength() int {
	// Create the demo content and count runes
	content := createStyledDemoContent()
	total := 0
	for _, span := range content {
		total += len([]rune(span.Text))
	}
	return total
}
