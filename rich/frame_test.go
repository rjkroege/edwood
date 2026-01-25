package rich

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

func TestNewFrame(t *testing.T) {
	f := NewFrame()
	if f == nil {
		t.Fatal("NewFrame() returned nil")
	}
}

func TestFrameInit(t *testing.T) {
	// Create a mock display
	rect := image.Rect(10, 20, 200, 300)
	display := edwoodtest.NewDisplay(rect)

	f := NewFrame()
	fi := f.(*frameImpl)

	// Initialize with rect and display
	f.Init(rect, WithDisplay(display))

	// Verify rect is stored
	if got := f.Rect(); got != rect {
		t.Errorf("Rect() = %v, want %v", got, rect)
	}

	// Verify display is stored
	if fi.display != display {
		t.Errorf("display not stored correctly")
	}
}

func TestFrameInitWithOptions(t *testing.T) {
	rect := image.Rect(0, 0, 100, 100)
	display := edwoodtest.NewDisplay(rect)

	f := NewFrame()
	fi := f.(*frameImpl)

	// Test that multiple options can be applied
	f.Init(rect, WithDisplay(display))

	if fi.display == nil {
		t.Error("WithDisplay option not applied")
	}
	if f.Rect() != rect {
		t.Errorf("Rect() = %v, want %v", f.Rect(), rect)
	}
}

// WithDisplay is an Option that sets the display for the frame.
func WithDisplay(d draw.Display) Option {
	return func(f *frameImpl) {
		f.display = d
	}
}

// WithBackground is an Option that sets the background image for the frame.
func WithBackground(b draw.Image) Option {
	return func(f *frameImpl) {
		f.background = b
	}
}

// WithFont is an Option that sets the font for the frame.
func WithFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.font = f
	}
}

// WithTextColor is an Option that sets the text color image for the frame.
func WithTextColor(c draw.Image) Option {
	return func(fi *frameImpl) {
		fi.textColor = c
	}
}

func TestFrameWithFont(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	f := NewFrame()
	fi := f.(*frameImpl)

	// Initialize with display and font
	f.Init(rect, WithDisplay(display), WithFont(font))

	// Verify font is stored
	if fi.font == nil {
		t.Error("font not stored in frame")
	}
	if fi.font != font {
		t.Errorf("font = %v, want %v", fi.font, font)
	}

	// Verify font properties are accessible
	if fi.font.Height() != 14 {
		t.Errorf("font.Height() = %d, want 14", fi.font.Height())
	}
}

func TestFrameRedrawFillsBackground(t *testing.T) {
	rect := image.Rect(10, 20, 200, 300)
	display := edwoodtest.NewDisplay(rect)

	// Allocate a distinct background color image (use Medblue as a visually distinct color)
	bgColor := draw.Medblue
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		t.Fatalf("AllocImage failed: %v", err)
	}

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage))

	// Clear any draw ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw
	f.Redraw()

	// Verify that a fill operation occurred for the frame's rectangle
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Fatal("Redraw() did not produce any draw operations")
	}

	// Look for a fill operation covering the frame rectangle
	found := false
	expectedFill := "fill " + rect.String()
	for _, op := range ops {
		if len(op) >= len(expectedFill) && op[:len(expectedFill)] == expectedFill {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not fill the background rectangle %v\ngot ops: %v", rect, ops)
	}
}

func TestDrawText(t *testing.T) {
	rect := image.Rect(10, 20, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Allocate background and text color images
	bgColor := draw.Medblue
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		t.Fatalf("AllocImage for background failed: %v", err)
	}

	textColor := draw.Black
	textImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, textColor)
	if err != nil {
		t.Fatalf("AllocImage for text color failed: %v", err)
	}

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content
	f.SetContent(Plain("hello"))

	// Clear any draw ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw
	f.Redraw()

	// Verify that text was rendered
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Look for a string draw operation containing "hello"
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "hello"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render text 'hello'\ngot ops: %v", ops)
	}
}

func TestDrawTextMultipleBoxes(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with two lines
	f.SetContent(Plain("hello\nworld"))

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Both "hello" and "world" should be rendered
	foundHello := false
	foundWorld := false
	for _, op := range ops {
		if strings.Contains(op, `string "hello"`) {
			foundHello = true
		}
		if strings.Contains(op, `string "world"`) {
			foundWorld = true
		}
	}

	if !foundHello {
		t.Errorf("Redraw() did not render 'hello'\ngot ops: %v", ops)
	}
	if !foundWorld {
		t.Errorf("Redraw() did not render 'world'\ngot ops: %v", ops)
	}
}

func TestDrawTextAtCorrectPosition(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set simple content
	f.SetContent(Plain("test"))

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Text should be rendered at the frame origin (rect.Min)
	// The mock records: "string \"test\" atpoint: (20,10)"
	foundAtOrigin := false
	expectedPos := fmt.Sprintf("atpoint: %v", rect.Min)
	for _, op := range ops {
		if strings.Contains(op, `string "test"`) && strings.Contains(op, expectedPos) {
			foundAtOrigin = true
			break
		}
	}

	if !foundAtOrigin {
		t.Errorf("Redraw() did not render 'test' at frame origin %v\ngot ops: %v", rect.Min, ops)
	}
}

func TestDrawTextSecondLinePosition(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // height = 14

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with newline
	f.SetContent(Plain("line1\nline2"))

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// First line at Y=10, second line at Y=10+14=24
	firstLineY := rect.Min.Y
	secondLineY := rect.Min.Y + 14

	foundFirstLine := false
	foundSecondLine := false

	for _, op := range ops {
		if strings.Contains(op, `string "line1"`) && strings.Contains(op, fmt.Sprintf("(%d,%d)", rect.Min.X, firstLineY)) {
			foundFirstLine = true
		}
		if strings.Contains(op, `string "line2"`) && strings.Contains(op, fmt.Sprintf("(%d,%d)", rect.Min.X, secondLineY)) {
			foundSecondLine = true
		}
	}

	if !foundFirstLine {
		t.Errorf("Redraw() did not render 'line1' at Y=%d\ngot ops: %v", firstLineY, ops)
	}
	if !foundSecondLine {
		t.Errorf("Redraw() did not render 'line2' at Y=%d\ngot ops: %v", secondLineY, ops)
	}
}

func TestDrawTextWithColor(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Use named images so we can identify them in the draw ops
	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	defaultTextImage := edwoodtest.NewImage(display, "default-text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(defaultTextImage))

	// Create content with a colored span using blue foreground
	// Style.Fg is image/color.Color, so we use color.RGBA
	blueColor := color.RGBA{R: 0, G: 0, B: 153, A: 255}
	blueStyle := Style{
		Fg:    blueColor,
		Scale: 1.0,
	}
	content := Content{
		{Text: "blue text", Style: blueStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify text was drawn with a custom color, not the default text color
	// When a style has Fg set, it should NOT use "default-text-color"
	foundWithCustomColor := false
	for _, op := range ops {
		if strings.Contains(op, `string "blue text"`) {
			// Check that it's NOT using the default text color
			if !strings.Contains(op, "default-text-color") {
				foundWithCustomColor = true
			}
			break
		}
	}

	if !foundWithCustomColor {
		t.Errorf("Redraw() should render 'blue text' with custom color, not default-text-color\ngot ops: %v", ops)
	}
}

func TestDrawTextWithMultipleColors(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Use named images so we can identify them in the draw ops
	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	defaultTextImage := edwoodtest.NewImage(display, "default-text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(defaultTextImage))

	// Create content with multiple colored spans using color.RGBA
	blueColor := color.RGBA{R: 0, G: 0, B: 153, A: 255}
	redColor := color.RGBA{R: 238, G: 0, B: 0, A: 255}
	blueStyle := Style{Fg: blueColor, Scale: 1.0}
	redStyle := Style{Fg: redColor, Scale: 1.0}
	content := Content{
		{Text: "blue ", Style: blueStyle},
		{Text: "red", Style: redStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify each text segment was drawn with custom colors (not default-text-color)
	blueNotDefault := false
	redNotDefault := false
	blueOp := ""
	redOp := ""

	for _, op := range ops {
		if strings.Contains(op, `string "blue "`) {
			blueOp = op
			if !strings.Contains(op, "default-text-color") {
				blueNotDefault = true
			}
		}
		if strings.Contains(op, `string "red"`) {
			redOp = op
			if !strings.Contains(op, "default-text-color") {
				redNotDefault = true
			}
		}
	}

	if !blueNotDefault {
		t.Errorf("Redraw() should render 'blue ' with custom color, not default-text-color\ngot op: %s\nall ops: %v", blueOp, ops)
	}
	if !redNotDefault {
		t.Errorf("Redraw() should render 'red' with custom color, not default-text-color\ngot op: %s\nall ops: %v", redOp, ops)
	}
}

func TestDrawTextWithDefaultColor(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Use named images so we can identify them in the draw ops
	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	defaultTextImage := edwoodtest.NewImage(display, "default-text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(defaultTextImage))

	// Plain text with no Fg color specified should use the default text color
	f.SetContent(Plain("default"))

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify text was drawn with the default text color (not a custom color)
	foundWithDefault := false
	for _, op := range ops {
		if strings.Contains(op, `string "default"`) && strings.Contains(op, "default-text-color") {
			foundWithDefault = true
			break
		}
	}

	if !foundWithDefault {
		t.Errorf("Redraw() should render 'default' with default-text-color\ngot ops: %v", ops)
	}
}

// WithBoldFont is an Option that sets the bold font variant for the frame.
func WithBoldFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.boldFont = f
	}
}

// WithItalicFont is an Option that sets the italic font variant for the frame.
func WithItalicFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.italicFont = f
	}
}

// WithBoldItalicFont is an Option that sets the bold-italic font variant for the frame.
func WithBoldItalicFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.boldItalicFont = f
	}
}

func TestFontVariantsBoldText(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	// Create distinct fonts for each variant with different widths to distinguish them
	regularFont := edwoodtest.NewFont(10, 14)
	boldFont := edwoodtest.NewFont(11, 14) // slightly wider to simulate bold

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithBoldFont(boldFont),
		WithTextColor(textImage),
	)

	// Set content with bold text
	boldStyle := Style{Bold: true, Scale: 1.0}
	content := Content{
		{Text: "bold text", Style: boldStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify bold text was rendered
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "bold text"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'bold text'\ngot ops: %v", ops)
	}

	// Verify the bold font was used by checking that fontForStyle returns boldFont
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(boldStyle)
	if selectedFont != boldFont {
		t.Errorf("fontForStyle(bold) should return boldFont, got %v", selectedFont)
	}
}

func TestFontVariantsItalicText(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	italicFont := edwoodtest.NewFont(10, 14) // same size, different instance

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithItalicFont(italicFont),
		WithTextColor(textImage),
	)

	// Set content with italic text
	italicStyle := Style{Italic: true, Scale: 1.0}
	content := Content{
		{Text: "italic text", Style: italicStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify italic text was rendered
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "italic text"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'italic text'\ngot ops: %v", ops)
	}

	// Verify the italic font was used
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(italicStyle)
	if selectedFont != italicFont {
		t.Errorf("fontForStyle(italic) should return italicFont, got %v", selectedFont)
	}
}

func TestFontVariantsBoldItalicText(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	boldFont := edwoodtest.NewFont(11, 14)
	italicFont := edwoodtest.NewFont(10, 14)
	boldItalicFont := edwoodtest.NewFont(11, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithBoldFont(boldFont),
		WithItalicFont(italicFont),
		WithBoldItalicFont(boldItalicFont),
		WithTextColor(textImage),
	)

	// Set content with bold+italic text
	boldItalicStyle := Style{Bold: true, Italic: true, Scale: 1.0}
	content := Content{
		{Text: "bold italic", Style: boldItalicStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify bold+italic text was rendered
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "bold italic"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'bold italic'\ngot ops: %v", ops)
	}

	// Verify the bold+italic font was used
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(boldItalicStyle)
	if selectedFont != boldItalicFont {
		t.Errorf("fontForStyle(bold+italic) should return boldItalicFont, got %v", selectedFont)
	}
}

func TestFontVariantsFallbackToRegular(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	// No bold, italic, or bold-italic fonts set

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithTextColor(textImage),
	)

	fi := f.(*frameImpl)

	// When no variant font is set, fontForStyle should fall back to regular font
	boldStyle := Style{Bold: true, Scale: 1.0}
	if got := fi.fontForStyle(boldStyle); got != regularFont {
		t.Errorf("fontForStyle(bold) without boldFont should return regularFont, got %v", got)
	}

	italicStyle := Style{Italic: true, Scale: 1.0}
	if got := fi.fontForStyle(italicStyle); got != regularFont {
		t.Errorf("fontForStyle(italic) without italicFont should return regularFont, got %v", got)
	}

	boldItalicStyle := Style{Bold: true, Italic: true, Scale: 1.0}
	if got := fi.fontForStyle(boldItalicStyle); got != regularFont {
		t.Errorf("fontForStyle(bold+italic) without boldItalicFont should return regularFont, got %v", got)
	}
}

func TestFontVariantsMixedContent(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	boldFont := edwoodtest.NewFont(11, 14)
	italicFont := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithBoldFont(boldFont),
		WithItalicFont(italicFont),
		WithTextColor(textImage),
	)

	// Set content with mixed styles
	content := Content{
		{Text: "normal ", Style: DefaultStyle()},
		{Text: "bold ", Style: Style{Bold: true, Scale: 1.0}},
		{Text: "italic", Style: Style{Italic: true, Scale: 1.0}},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// All three text segments should be rendered
	foundNormal := false
	foundBold := false
	foundItalic := false
	for _, op := range ops {
		if strings.Contains(op, `string "normal "`) {
			foundNormal = true
		}
		if strings.Contains(op, `string "bold "`) {
			foundBold = true
		}
		if strings.Contains(op, `string "italic"`) {
			foundItalic = true
		}
	}

	if !foundNormal {
		t.Errorf("Redraw() did not render 'normal '\ngot ops: %v", ops)
	}
	if !foundBold {
		t.Errorf("Redraw() did not render 'bold '\ngot ops: %v", ops)
	}
	if !foundItalic {
		t.Errorf("Redraw() did not render 'italic'\ngot ops: %v", ops)
	}
}
