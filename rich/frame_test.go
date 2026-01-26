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

	// Text should be rendered at origin (0,0) in scratch image coordinates.
	// The scratch image is then blitted to the frame origin on screen.
	// When using scratch-based clipping, text is drawn at local coords.
	foundText := false
	for _, op := range ops {
		if strings.Contains(op, `string "test"`) {
			foundText = true
			break
		}
	}

	if !foundText {
		t.Errorf("Redraw() did not render 'test'\ngot ops: %v", ops)
	}

	// Verify the final blit to screen places content at frame origin
	foundBlit := false
	expectedRect := fmt.Sprintf("fill %v", rect)
	for _, op := range ops {
		if strings.Contains(op, expectedRect) {
			foundBlit = true
			break
		}
	}
	if !foundBlit {
		t.Errorf("Redraw() did not blit to frame rect %v\ngot ops: %v", rect, ops)
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

	// When using scratch-based clipping, text is drawn at local coordinates.
	// First line at Y=0, second line at Y=14 (font height)
	// The scratch image is then blitted to screen at frame origin.
	foundFirstLine := false
	foundSecondLine := false

	for _, op := range ops {
		// Check for line1 at local Y=0
		if strings.Contains(op, `string "line1"`) && strings.Contains(op, "(0,0)") {
			foundFirstLine = true
		}
		// Check for line2 at local Y=14 (one line height below)
		if strings.Contains(op, `string "line2"`) && strings.Contains(op, "(0,14)") {
			foundSecondLine = true
		}
	}

	if !foundFirstLine {
		t.Errorf("Redraw() did not render 'line1' at local Y=0\ngot ops: %v", ops)
	}
	if !foundSecondLine {
		t.Errorf("Redraw() did not render 'line2' at local Y=14\ngot ops: %v", ops)
	}

	// Verify the final blit places content at correct screen position
	foundBlit := false
	expectedRect := fmt.Sprintf("fill %v", rect)
	for _, op := range ops {
		if strings.Contains(op, expectedRect) {
			foundBlit = true
			break
		}
	}
	if !foundBlit {
		t.Errorf("Redraw() did not blit to frame rect %v\ngot ops: %v", rect, ops)
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
	// Note: "blue text" is now split into separate words during layout
	foundWithCustomColor := false
	for _, op := range ops {
		if strings.Contains(op, `string "blue"`) {
			// Check that it's NOT using the default text color
			if !strings.Contains(op, "default-text-color") {
				foundWithCustomColor = true
			}
			break
		}
	}

	if !foundWithCustomColor {
		t.Errorf("Redraw() should render 'blue' with custom color, not default-text-color\ngot ops: %v", ops)
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

	// Note: "blue " is now split into "blue" and " " during layout
	for _, op := range ops {
		if strings.Contains(op, `string "blue"`) && !strings.Contains(op, `string "blue "`) {
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
		t.Errorf("Redraw() should render 'blue' with custom color, not default-text-color\ngot op: %s\nall ops: %v", blueOp, ops)
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

	// Verify bold text was rendered (now split into words)
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "bold"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'bold'\ngot ops: %v", ops)
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
		if strings.Contains(op, `string "italic"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'italic'\ngot ops: %v", ops)
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

	// Verify bold+italic text was rendered (now split into words)
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "bold"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'bold'\ngot ops: %v", ops)
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
	// Note: text is now split into words, so "normal " becomes "normal" and " "
	for _, op := range ops {
		if strings.Contains(op, `string "normal"`) {
			foundNormal = true
		}
		if strings.Contains(op, `string "bold"`) && !strings.Contains(op, `string "bold "`) {
			foundBold = true
		}
		if strings.Contains(op, `string "italic"`) {
			foundItalic = true
		}
	}

	if !foundNormal {
		t.Errorf("Redraw() did not render 'normal'\ngot ops: %v", ops)
	}
	if !foundBold {
		t.Errorf("Redraw() did not render 'bold'\ngot ops: %v", ops)
	}
	if !foundItalic {
		t.Errorf("Redraw() did not render 'italic'\ngot ops: %v", ops)
	}
}

// ScaledFont wraps a font and applies a scale factor to its metrics.
// This is used for testing scaled fonts for headings.
type ScaledFont struct {
	base  draw.Font
	scale float64
}

func (sf *ScaledFont) Name() string { return sf.base.Name() }
func (sf *ScaledFont) Height() int {
	return int(float64(sf.base.Height()) * sf.scale)
}
func (sf *ScaledFont) BytesWidth(b []byte) int {
	return int(float64(sf.base.BytesWidth(b)) * sf.scale)
}
func (sf *ScaledFont) RunesWidth(r []rune) int {
	return int(float64(sf.base.RunesWidth(r)) * sf.scale)
}
func (sf *ScaledFont) StringWidth(s string) int {
	return int(float64(sf.base.StringWidth(s)) * sf.scale)
}

// NewScaledFont creates a font with scaled metrics for testing.
func NewScaledFont(base draw.Font, scale float64) draw.Font {
	return &ScaledFont{base: base, scale: scale}
}

func TestFontScaleH1Text(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	// Regular font: 10px wide, 14px tall
	regularFont := edwoodtest.NewFont(10, 14)
	// H1 font should be 2x scale: 20px wide, 28px tall
	h1Font := NewScaledFont(regularFont, 2.0)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithScaledFont(2.0, h1Font),
		WithTextColor(textImage),
	)

	// Set content with H1 heading style (Scale: 2.0)
	content := Content{
		{Text: "Big Heading", Style: StyleH1},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify H1 text was rendered
	// Note: "Big Heading" is now split into words
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "Big"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'Big'\ngot ops: %v", ops)
	}

	// Verify the H1 scaled font is returned for StyleH1
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(StyleH1)
	if selectedFont != h1Font {
		t.Errorf("fontForStyle(StyleH1) should return h1Font, got %v", selectedFont)
	}

	// Verify the scaled font has correct metrics
	if h1Font.Height() != 28 {
		t.Errorf("H1 font height = %d, want 28", h1Font.Height())
	}
	// "test" is 4 chars, at 20px per char = 80px
	if h1Font.StringWidth("test") != 80 {
		t.Errorf("H1 font StringWidth(\"test\") = %d, want 80", h1Font.StringWidth("test"))
	}
}

func TestFontScaleH2Text(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	// H2 font should be 1.5x scale: 15px wide, 21px tall
	h2Font := NewScaledFont(regularFont, 1.5)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithScaledFont(1.5, h2Font),
		WithTextColor(textImage),
	)

	// Set content with H2 heading style (Scale: 1.5)
	content := Content{
		{Text: "Medium Heading", Style: StyleH2},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify H2 text was rendered (now split into words)
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "Medium"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'Medium'\ngot ops: %v", ops)
	}

	// Verify the H2 scaled font is returned for StyleH2
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(StyleH2)
	if selectedFont != h2Font {
		t.Errorf("fontForStyle(StyleH2) should return h2Font, got %v", selectedFont)
	}

	// Verify the scaled font has correct metrics
	if h2Font.Height() != 21 {
		t.Errorf("H2 font height = %d, want 21", h2Font.Height())
	}
}

func TestFontScaleH3Text(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	// H3 font should be 1.25x scale: 12px wide (truncated), 17px tall (truncated)
	h3Font := NewScaledFont(regularFont, 1.25)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithScaledFont(1.25, h3Font),
		WithTextColor(textImage),
	)

	// Set content with H3 heading style (Scale: 1.25)
	content := Content{
		{Text: "Small Heading", Style: StyleH3},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify H3 text was rendered (now split into words)
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "Small"`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'Small'\ngot ops: %v", ops)
	}

	// Verify the H3 scaled font is returned for StyleH3
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(StyleH3)
	if selectedFont != h3Font {
		t.Errorf("fontForStyle(StyleH3) should return h3Font, got %v", selectedFont)
	}

	// Verify the scaled font has correct metrics (int truncation)
	// 14 * 1.25 = 17.5, truncated to 17
	if h3Font.Height() != 17 {
		t.Errorf("H3 font height = %d, want 17", h3Font.Height())
	}
}

func TestFontScaleFallbackToRegular(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	// No scaled fonts configured

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

	// When no scaled font is available, fontForStyle should fall back to regular font
	if got := fi.fontForStyle(StyleH1); got != regularFont {
		t.Errorf("fontForStyle(StyleH1) without scaled font should return regularFont, got %v", got)
	}

	if got := fi.fontForStyle(StyleH2); got != regularFont {
		t.Errorf("fontForStyle(StyleH2) without scaled font should return regularFont, got %v", got)
	}

	if got := fi.fontForStyle(StyleH3); got != regularFont {
		t.Errorf("fontForStyle(StyleH3) without scaled font should return regularFont, got %v", got)
	}
}

func TestFontScaleMixedContent(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	h1Font := NewScaledFont(regularFont, 2.0)
	h2Font := NewScaledFont(regularFont, 1.5)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithScaledFont(2.0, h1Font),
		WithScaledFont(1.5, h2Font),
		WithTextColor(textImage),
	)

	// Content with multiple heading levels and body text
	content := Content{
		{Text: "Title\n", Style: StyleH1},
		{Text: "Subtitle\n", Style: StyleH2},
		{Text: "Body text", Style: DefaultStyle()},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// All text segments should be rendered
	foundTitle := false
	foundSubtitle := false
	foundBody := false
	for _, op := range ops {
		if strings.Contains(op, `string "Title"`) {
			foundTitle = true
		}
		if strings.Contains(op, `string "Subtitle"`) {
			foundSubtitle = true
		}
		// Note: "Body text" is now split into words
		if strings.Contains(op, `string "Body"`) && !strings.Contains(op, `string "Body "`) {
			foundBody = true
		}
	}

	if !foundTitle {
		t.Errorf("Redraw() did not render 'Title'\ngot ops: %v", ops)
	}
	if !foundSubtitle {
		t.Errorf("Redraw() did not render 'Subtitle'\ngot ops: %v", ops)
	}
	if !foundBody {
		t.Errorf("Redraw() did not render 'Body'\ngot ops: %v", ops)
	}

	// Verify correct fonts are selected for each style
	fi := f.(*frameImpl)
	if got := fi.fontForStyle(StyleH1); got != h1Font {
		t.Errorf("fontForStyle(StyleH1) = %v, want h1Font", got)
	}
	if got := fi.fontForStyle(StyleH2); got != h2Font {
		t.Errorf("fontForStyle(StyleH2) = %v, want h2Font", got)
	}
	if got := fi.fontForStyle(DefaultStyle()); got != regularFont {
		t.Errorf("fontForStyle(DefaultStyle()) = %v, want regularFont", got)
	}
}

func TestFontScaleWithBoldCombination(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	boldFont := edwoodtest.NewFont(11, 14)
	// H1 is Bold:true, Scale:2.0 - we need a bold scaled font
	h1BoldFont := NewScaledFont(boldFont, 2.0)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithBoldFont(boldFont),
		WithScaledFont(2.0, h1BoldFont), // Scaled bold for H1
		WithTextColor(textImage),
	)

	// StyleH1 has both Bold:true and Scale:2.0
	content := Content{
		{Text: "Bold Heading", Style: StyleH1},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify text was rendered (now split into words)
	found := false
	for _, op := range ops {
		if strings.Contains(op, `string "Bold"`) && !strings.Contains(op, `string "Bold "`) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not render 'Bold'\ngot ops: %v", ops)
	}

	// For StyleH1 (Bold:true, Scale:2.0), the scaled font should take precedence
	// since it provides the scaled metrics needed for heading layout
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(StyleH1)
	if selectedFont != h1BoldFont {
		t.Errorf("fontForStyle(StyleH1) should return h1BoldFont for bold+scaled style, got %v", selectedFont)
	}
}

// TestPtofcharStart tests that position 0 returns the frame origin.
func TestPtofcharStart(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content
	f.SetContent(Plain("hello world"))

	// Position 0 should return the frame origin (rect.Min)
	pt := f.Ptofchar(0)
	if pt != rect.Min {
		t.Errorf("Ptofchar(0) = %v, want %v", pt, rect.Min)
	}
}

// TestPtofcharMiddle tests character positions within a single line.
func TestPtofcharMiddle(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello" = 5 chars
	f.SetContent(Plain("hello"))

	// Position 3 should be at X = rect.Min.X + 3*10 = 20 + 30 = 50
	pt := f.Ptofchar(3)
	want := image.Point{X: rect.Min.X + 30, Y: rect.Min.Y}
	if pt != want {
		t.Errorf("Ptofchar(3) = %v, want %v", pt, want)
	}
}

// TestPtofcharEnd tests position at the end of content.
func TestPtofcharEnd(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello" = 5 chars
	f.SetContent(Plain("hello"))

	// Position 5 (one past last char) should be at X = rect.Min.X + 5*10 = 20 + 50 = 70
	pt := f.Ptofchar(5)
	want := image.Point{X: rect.Min.X + 50, Y: rect.Min.Y}
	if pt != want {
		t.Errorf("Ptofchar(5) = %v, want %v", pt, want)
	}
}

// TestPtofcharMultiLine tests positions on different lines.
func TestPtofcharMultiLine(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello\nworld" = "hello" (5 chars) + newline (1 char) + "world" (5 chars)
	f.SetContent(Plain("hello\nworld"))

	// Position 6 is the 'w' of "world", first char on second line
	// X should be at rect.Min.X, Y should be rect.Min.Y + fontHeight
	pt := f.Ptofchar(6)
	want := image.Point{X: rect.Min.X, Y: rect.Min.Y + 14}
	if pt != want {
		t.Errorf("Ptofchar(6) = %v, want %v", pt, want)
	}

	// Position 8 is the 'r' of "world" (3rd char on second line)
	// X should be at rect.Min.X + 2*10
	pt = f.Ptofchar(8)
	want = image.Point{X: rect.Min.X + 20, Y: rect.Min.Y + 14}
	if pt != want {
		t.Errorf("Ptofchar(8) = %v, want %v", pt, want)
	}
}

// TestPtofcharWrappedLine tests positions when text wraps to next line.
func TestPtofcharWrappedLine(t *testing.T) {
	// Frame is 50px wide (rect from 20 to 70), font is 10px per char
	// So 5 chars fit per line before wrapping
	rect := image.Rect(20, 10, 70, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "helloworld" = 10 chars
	// Should wrap: "hello" on line 1, "world" on line 2
	f.SetContent(Plain("helloworld"))

	// Position 5 should be 'w', first char on second line (wrapped)
	pt := f.Ptofchar(5)
	want := image.Point{X: rect.Min.X, Y: rect.Min.Y + 14}
	if pt != want {
		t.Errorf("Ptofchar(5) = %v, want %v", pt, want)
	}

	// Position 7 should be 'r', 3rd char on second line
	pt = f.Ptofchar(7)
	want = image.Point{X: rect.Min.X + 20, Y: rect.Min.Y + 14}
	if pt != want {
		t.Errorf("Ptofchar(7) = %v, want %v", pt, want)
	}
}

// TestPtofcharEmptyContent tests Ptofchar with no content.
func TestPtofcharEmptyContent(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// No content set
	f.SetContent(Plain(""))

	// Position 0 in empty frame should still return rect.Min
	pt := f.Ptofchar(0)
	if pt != rect.Min {
		t.Errorf("Ptofchar(0) on empty = %v, want %v", pt, rect.Min)
	}
}

// TestPtofcharWithTab tests positions with tab characters.
func TestPtofcharWithTab(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height, tab = 8*10 = 80px

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "a\tb" = 'a' (1 char) + tab (1 char) + 'b' (1 char)
	f.SetContent(Plain("a\tb"))

	// Position 0 = 'a', at origin
	pt := f.Ptofchar(0)
	if pt != rect.Min {
		t.Errorf("Ptofchar(0) = %v, want %v", pt, rect.Min)
	}

	// Position 1 = tab, at X = 10 (after 'a')
	pt = f.Ptofchar(1)
	want := image.Point{X: 10, Y: 0}
	if pt != want {
		t.Errorf("Ptofchar(1) = %v, want %v", pt, want)
	}

	// Position 2 = 'b', should be at next tab stop after 'a'
	// Tab stop at 80, so 'b' is at X = 80
	pt = f.Ptofchar(2)
	want = image.Point{X: 80, Y: 0}
	if pt != want {
		t.Errorf("Ptofchar(2) = %v, want %v", pt, want)
	}
}

// TestCharofptStart tests that a point at the frame origin returns position 0.
func TestCharofptStart(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content
	f.SetContent(Plain("hello world"))

	// Point at frame origin should return position 0
	pos := f.Charofpt(rect.Min)
	if pos != 0 {
		t.Errorf("Charofpt(%v) = %d, want 0", rect.Min, pos)
	}
}

// TestCharofptMiddle tests character positions within a single line.
func TestCharofptMiddle(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello" = 5 chars, each 10px wide
	f.SetContent(Plain("hello"))

	// Point at X = rect.Min.X + 35 (middle of 4th char 'l') should return position 3
	pt := image.Point{X: rect.Min.X + 35, Y: rect.Min.Y}
	pos := f.Charofpt(pt)
	if pos != 3 {
		t.Errorf("Charofpt(%v) = %d, want 3", pt, pos)
	}

	// Point at X = rect.Min.X + 5 (middle of 1st char 'h') should return position 0
	pt = image.Point{X: rect.Min.X + 5, Y: rect.Min.Y}
	pos = f.Charofpt(pt)
	if pos != 0 {
		t.Errorf("Charofpt(%v) = %d, want 0", pt, pos)
	}

	// Point at X = rect.Min.X + 15 (middle of 2nd char 'e') should return position 1
	pt = image.Point{X: rect.Min.X + 15, Y: rect.Min.Y}
	pos = f.Charofpt(pt)
	if pos != 1 {
		t.Errorf("Charofpt(%v) = %d, want 1", pt, pos)
	}
}

// TestCharofptEnd tests position at and beyond the end of content.
func TestCharofptEnd(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello" = 5 chars, total width 50px
	f.SetContent(Plain("hello"))

	// Point at X = rect.Min.X + 50 (end of last char) should return position 5
	pt := image.Point{X: rect.Min.X + 50, Y: rect.Min.Y}
	pos := f.Charofpt(pt)
	if pos != 5 {
		t.Errorf("Charofpt(%v) = %d, want 5", pt, pos)
	}

	// Point beyond end of content should return last position
	pt = image.Point{X: rect.Min.X + 200, Y: rect.Min.Y}
	pos = f.Charofpt(pt)
	if pos != 5 {
		t.Errorf("Charofpt(%v) beyond content = %d, want 5", pt, pos)
	}
}

// TestCharofptMultiLine tests positions on different lines.
func TestCharofptMultiLine(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello\nworld" = "hello" (5 chars) + newline (1 char) + "world" (5 chars)
	f.SetContent(Plain("hello\nworld"))

	// Point on second line at X = rect.Min.X should return position 6 ('w' of "world")
	pt := image.Point{X: rect.Min.X, Y: rect.Min.Y + 14}
	pos := f.Charofpt(pt)
	if pos != 6 {
		t.Errorf("Charofpt(%v) = %d, want 6", pt, pos)
	}

	// Point on second line at X = rect.Min.X + 25 (middle of 'r') should return position 8
	pt = image.Point{X: rect.Min.X + 25, Y: rect.Min.Y + 14}
	pos = f.Charofpt(pt)
	if pos != 8 {
		t.Errorf("Charofpt(%v) = %d, want 8", pt, pos)
	}
}

// TestCharofptWrappedLine tests positions when text wraps to next line.
func TestCharofptWrappedLine(t *testing.T) {
	// Frame is 50px wide (rect from 20 to 70), font is 10px per char
	// So 5 chars fit per line before wrapping
	rect := image.Rect(20, 10, 70, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "helloworld" = 10 chars
	// Should wrap: "hello" on line 1, "world" on line 2
	f.SetContent(Plain("helloworld"))

	// Point on second line at X = rect.Min.X should return position 5 ('w')
	pt := image.Point{X: rect.Min.X, Y: rect.Min.Y + 14}
	pos := f.Charofpt(pt)
	if pos != 5 {
		t.Errorf("Charofpt(%v) = %d, want 5", pt, pos)
	}

	// Point on second line at X = rect.Min.X + 25 (middle of 'r') should return position 7
	pt = image.Point{X: rect.Min.X + 25, Y: rect.Min.Y + 14}
	pos = f.Charofpt(pt)
	if pos != 7 {
		t.Errorf("Charofpt(%v) = %d, want 7", pt, pos)
	}
}

// TestCharofptEmptyContent tests Charofpt with no content.
func TestCharofptEmptyContent(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// No content set
	f.SetContent(Plain(""))

	// Any point in empty frame should return 0
	pos := f.Charofpt(rect.Min)
	if pos != 0 {
		t.Errorf("Charofpt(%v) on empty = %d, want 0", rect.Min, pos)
	}
}

// TestCharofptWithTab tests positions with tab characters.
func TestCharofptWithTab(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height, tab = 8*10 = 80px

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "a\tb" = 'a' (1 char) + tab (1 char) + 'b' (1 char)
	// Layout: 'a' at 0-10, tab from 10-80, 'b' at 80-90
	f.SetContent(Plain("a\tb"))

	// Point at X = 5 (middle of 'a') should return position 0
	pt := image.Point{X: 5, Y: 0}
	pos := f.Charofpt(pt)
	if pos != 0 {
		t.Errorf("Charofpt(%v) = %d, want 0", pt, pos)
	}

	// Point at X = 40 (middle of tab) should return position 1
	pt = image.Point{X: 40, Y: 0}
	pos = f.Charofpt(pt)
	if pos != 1 {
		t.Errorf("Charofpt(%v) = %d, want 1", pt, pos)
	}

	// Point at X = 85 (middle of 'b') should return position 2
	pt = image.Point{X: 85, Y: 0}
	pos = f.Charofpt(pt)
	if pos != 2 {
		t.Errorf("Charofpt(%v) = %d, want 2", pt, pos)
	}
}

// TestCharofptOutsideFrame tests points outside the frame rectangle.
func TestCharofptOutsideFrame(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello"))

	// Point to the left of frame should return 0
	pt := image.Point{X: 0, Y: rect.Min.Y}
	pos := f.Charofpt(pt)
	if pos != 0 {
		t.Errorf("Charofpt(%v) left of frame = %d, want 0", pt, pos)
	}

	// Point above frame should return 0
	pt = image.Point{X: rect.Min.X, Y: 0}
	pos = f.Charofpt(pt)
	if pos != 0 {
		t.Errorf("Charofpt(%v) above frame = %d, want 0", pt, pos)
	}
}

// TestCoordinateRoundTrip verifies that Charofpt(Ptofchar(n)) == n for all valid positions.
// This is a critical property for correct cursor positioning and selection behavior.
func TestCoordinateRoundTrip(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Test with simple text
	f.SetContent(Plain("hello"))

	// Test every position from 0 to len(content)+1
	for i := 0; i <= 5; i++ {
		pt := f.Ptofchar(i)
		got := f.Charofpt(pt)
		if got != i {
			t.Errorf("Charofpt(Ptofchar(%d)) = %d, want %d (pt=%v)", i, got, i, pt)
		}
	}
}

// TestCoordinateRoundTripMultiLine tests round-trip with multi-line content.
func TestCoordinateRoundTripMultiLine(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "hello\nworld" = 11 characters (5 + 1 + 5)
	f.SetContent(Plain("hello\nworld"))

	// Test every position from 0 to len(content)+1
	for i := 0; i <= 11; i++ {
		pt := f.Ptofchar(i)
		got := f.Charofpt(pt)
		if got != i {
			t.Errorf("Charofpt(Ptofchar(%d)) = %d, want %d (pt=%v)", i, got, i, pt)
		}
	}
}

// TestCoordinateRoundTripWrapped tests round-trip with wrapped lines.
func TestCoordinateRoundTripWrapped(t *testing.T) {
	// Frame is 50px wide, font is 10px per char
	// So 5 chars fit per line before wrapping
	rect := image.Rect(20, 10, 70, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "helloworld" = 10 chars, wraps after 5
	f.SetContent(Plain("helloworld"))

	// Test every position from 0 to len(content)+1
	for i := 0; i <= 10; i++ {
		pt := f.Ptofchar(i)
		got := f.Charofpt(pt)
		if got != i {
			t.Errorf("Charofpt(Ptofchar(%d)) = %d, want %d (pt=%v)", i, got, i, pt)
		}
	}
}

// TestCoordinateRoundTripWithTabs tests round-trip with tab characters.
func TestCoordinateRoundTripWithTabs(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "a\tb" = 3 characters
	f.SetContent(Plain("a\tb"))

	// Test every position from 0 to len(content)+1
	for i := 0; i <= 3; i++ {
		pt := f.Ptofchar(i)
		got := f.Charofpt(pt)
		if got != i {
			t.Errorf("Charofpt(Ptofchar(%d)) = %d, want %d (pt=%v)", i, got, i, pt)
		}
	}
}

// TestCoordinateRoundTripEmpty tests round-trip with empty content.
func TestCoordinateRoundTripEmpty(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Empty content
	f.SetContent(Plain(""))

	// Position 0 should round-trip
	pt := f.Ptofchar(0)
	got := f.Charofpt(pt)
	if got != 0 {
		t.Errorf("Charofpt(Ptofchar(0)) on empty = %d, want 0 (pt=%v)", got, pt)
	}
}

// TestDrawBoxBackground tests that Style.Bg causes a background fill before text.
func TestDrawBoxBackground(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // "code" is 4 chars = 40px wide

	bgImage := edwoodtest.NewImage(display, "frame-background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create content with a background color (like inline code)
	grayBg := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	codeStyle := Style{Bg: grayBg, Scale: 1.0}
	content := Content{
		{Text: "code", Style: codeStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// There should be a fill operation for the box background before the text
	// The fill should happen BEFORE the text rendering
	// The box background fill should be roughly the size of the text (4 chars * 10px = 40px wide)
	foundBoxFill := false
	foundText := false
	fillBeforeText := false
	frameBackgroundRect := "(0,0)-(400,300)"

	for _, op := range ops {
		// Look for fill operation that's NOT the frame background fill
		// Fill ops start with "fill " - this distinguishes from "fill:" in string ops
		if strings.HasPrefix(op, "fill ") {
			if strings.Contains(op, frameBackgroundRect) {
				continue // Skip the frame background
			}
			// This must be a box background fill (smaller than full frame)
			foundBoxFill = true
			if !foundText {
				fillBeforeText = true
			}
		}
		// Look for text rendering
		if strings.Contains(op, `string "code"`) {
			foundText = true
		}
	}

	if !foundBoxFill {
		t.Errorf("Redraw() did not render box background fill for styled text with Bg\ngot ops: %v", ops)
	}
	if !foundText {
		t.Errorf("Redraw() did not render 'code' text\ngot ops: %v", ops)
	}
	if foundBoxFill && !fillBeforeText {
		t.Errorf("Box background fill should occur before text rendering\ngot ops: %v", ops)
	}
}

// TestDrawBoxBackgroundMultiple tests multiple boxes with backgrounds.
func TestDrawBoxBackgroundMultiple(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "frame-background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create content with multiple background-styled spans and regular text
	grayBg := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	codeStyle := Style{Bg: grayBg, Scale: 1.0}
	content := Content{
		{Text: "normal ", Style: DefaultStyle()},
		{Text: "code1", Style: codeStyle},
		{Text: " more ", Style: DefaultStyle()},
		{Text: "code2", Style: codeStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	frameBackgroundRect := "(0,0)-(400,300)"

	// Count fill operations (excluding the initial frame background fill)
	fillCount := 0
	for _, op := range ops {
		// Fill ops start with "fill " - this distinguishes from "fill:" in string ops
		if strings.HasPrefix(op, "fill ") && !strings.Contains(op, frameBackgroundRect) {
			fillCount++
		}
	}

	// Should have 2 fills for the two code spans
	if fillCount != 2 {
		t.Errorf("Expected 2 box background fills, got %d\ngot ops: %v", fillCount, ops)
	}

	// Verify all text was rendered (words are now split)
	texts := []string{"normal", "code1", "more", "code2"}
	for _, text := range texts {
		found := false
		for _, op := range ops {
			if strings.Contains(op, fmt.Sprintf(`string "%s"`, text)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Redraw() did not render '%s'\ngot ops: %v", text, ops)
		}
	}
}

// TestCodeFontSelection tests that Style.Code causes the code font to be used.
func TestCodeFontSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	// Create distinct fonts: regular (10px per char) and code (12px per char, monospace)
	regularFont := edwoodtest.NewFont(10, 14)
	codeFont := edwoodtest.NewFont(12, 14) // Different width to simulate monospace

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		WithCodeFont(codeFont),
		WithTextColor(textImage),
	)

	// Verify the code font is returned for StyleCode
	fi := f.(*frameImpl)
	selectedFont := fi.fontForStyle(StyleCode)
	if selectedFont != codeFont {
		t.Errorf("fontForStyle(StyleCode) should return codeFont, got %v", selectedFont)
	}

	// Also test with explicitly constructed code style
	codeStyle := Style{Code: true, Scale: 1.0}
	selectedFont = fi.fontForStyle(codeStyle)
	if selectedFont != codeFont {
		t.Errorf("fontForStyle(Code:true) should return codeFont, got %v", selectedFont)
	}

	// Regular style should still return regular font
	selectedFont = fi.fontForStyle(DefaultStyle())
	if selectedFont != regularFont {
		t.Errorf("fontForStyle(DefaultStyle()) should return regularFont, got %v", selectedFont)
	}
}

// TestCodeFontFallback tests that Style.Code falls back to regular font when no code font is set.
func TestCodeFontFallback(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	regularFont := edwoodtest.NewFont(10, 14)
	// No code font set

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(regularFont),
		// Note: WithCodeFont is NOT called
		WithTextColor(textImage),
	)

	fi := f.(*frameImpl)

	// When no code font is set, fontForStyle should fall back to regular font
	if got := fi.fontForStyle(StyleCode); got != regularFont {
		t.Errorf("fontForStyle(StyleCode) without codeFont should return regularFont, got %v", got)
	}

	codeStyle := Style{Code: true, Scale: 1.0}
	if got := fi.fontForStyle(codeStyle); got != regularFont {
		t.Errorf("fontForStyle(Code:true) without codeFont should return regularFont, got %v", got)
	}
}

// TestDrawBlockBackground tests that BlockRegions cause indented background fills.
// This is used for fenced code blocks where the background extends from the indent
// to the frame edge, not from the left edge.
func TestDrawBlockBackground(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300) // Frame is 400px wide
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // "code" is 4 chars = 40px wide

	bgImage := edwoodtest.NewImage(display, "frame-background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create content with block-level background (like fenced code)
	// The Block flag indicates this should have indented background
	grayBg := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	codeBlockStyle := Style{Code: true, Bg: grayBg, Block: true, Scale: 1.0}
	content := Content{
		{Text: "code\n", Style: codeBlockStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Code blocks are indented by CodeBlockIndentChars * M-width = 4 * 10 = 40 pixels
	// Background should start at x=40 and extend to frame width (400px)
	foundBlockFill := false
	frameBackgroundRect := "(0,0)-(400,300)"
	expectedIndent := CodeBlockIndentChars * font.BytesWidth([]byte("M")) // 40 pixels

	for _, op := range ops {
		// Look for fill operations that are NOT the frame background
		// but extend from indent to right edge
		if strings.HasPrefix(op, "fill ") {
			if strings.Contains(op, frameBackgroundRect) {
				continue // Skip the frame background
			}
			// Check if this fill starts at the indent and extends to full frame width
			// Format: "fill (40,0)-(400,14)" for first line with 40px indent
			expectedPrefix := fmt.Sprintf("(%d,", expectedIndent)
			if strings.Contains(op, expectedPrefix) && strings.Contains(op, "-(400,") {
				foundBlockFill = true
			}
		}
	}

	if !foundBlockFill {
		t.Errorf("Redraw() did not render indented block background for code block\nExpected fill from x=%d to x=400, got ops: %v", expectedIndent, ops)
	}

	// Also verify text was rendered at the indented position
	foundText := false
	for _, op := range ops {
		if strings.Contains(op, `string "code"`) {
			foundText = true
			break
		}
	}
	if !foundText {
		t.Errorf("Redraw() did not render 'code' text\ngot ops: %v", ops)
	}
}

// TestDrawBlockBackgroundMultiLine tests indented backgrounds spanning multiple lines.
func TestDrawBlockBackgroundMultiLine(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300) // Frame is 400px wide
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 14px line height

	bgImage := edwoodtest.NewImage(display, "frame-background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create multi-line content with block-level background
	grayBg := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	codeBlockStyle := Style{Code: true, Bg: grayBg, Block: true, Scale: 1.0}
	content := Content{
		{Text: "line1\nline2\nline3\n", Style: codeBlockStyle},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	frameBackgroundRect := "(0,0)-(400,300)"

	// Code blocks are indented by CodeBlockIndentChars * M-width = 4 * 10 = 40 pixels
	expectedIndent := CodeBlockIndentChars * font.BytesWidth([]byte("M")) // 40 pixels

	// Count indented fill operations (excluding frame background)
	// Each line should have its own background fill starting at the indent
	blockFillCount := 0
	expectedPrefix := fmt.Sprintf("(%d,", expectedIndent)
	for _, op := range ops {
		if strings.HasPrefix(op, "fill ") {
			if strings.Contains(op, frameBackgroundRect) {
				continue // Skip the frame background
			}
			// Check if this fill starts at indent and extends to right edge
			if strings.Contains(op, "-(400,") && strings.Contains(op, expectedPrefix) {
				blockFillCount++
			}
		}
	}

	// Should have 3 indented fills for 3 lines of code
	// (newlines create separate lines, each with their own fill)
	if blockFillCount < 3 {
		t.Errorf("Expected at least 3 indented block background fills for 3-line code block, got %d\ngot ops: %v", blockFillCount, ops)
	}

	// Verify all text lines were rendered
	texts := []string{"line1", "line2", "line3"}
	for _, text := range texts {
		found := false
		for _, op := range ops {
			if strings.Contains(op, fmt.Sprintf(`string "%s"`, text)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Redraw() did not render '%s'\ngot ops: %v", text, ops)
		}
	}
}

// TestDrawHorizontalRule tests that HRuleRune causes a horizontal line to be drawn instead of text.
// When a box contains HRuleRune with StyleHRule, the renderer should draw a line
// instead of rendering the rune as text.
func TestDrawHorizontalRule(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300) // Frame is 400px wide
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 14px line height

	bgImage := edwoodtest.NewImage(display, "frame-background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create content with a horizontal rule marker followed by text
	// The HRuleRune should be rendered as a line, not as text
	content := Content{
		{Text: "above\n", Style: DefaultStyle()},
		{Text: string(HRuleRune) + "\n", Style: StyleHRule},
		{Text: "below", Style: DefaultStyle()},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify "above" and "below" are rendered as text
	foundAbove := false
	foundBelow := false
	for _, op := range ops {
		if strings.Contains(op, `string "above"`) {
			foundAbove = true
		}
		if strings.Contains(op, `string "below"`) {
			foundBelow = true
		}
	}

	if !foundAbove {
		t.Errorf("Redraw() did not render 'above' text\ngot ops: %v", ops)
	}
	if !foundBelow {
		t.Errorf("Redraw() did not render 'below' text\ngot ops: %v", ops)
	}

	// Verify that HRuleRune is NOT rendered as text (it should be drawn as a line instead)
	hruleAsText := false
	for _, op := range ops {
		// The HRuleRune character should NOT appear in any string rendering operation
		if strings.Contains(op, `string "`) && strings.Contains(op, string(HRuleRune)) {
			hruleAsText = true
			break
		}
	}

	if hruleAsText {
		t.Errorf("HRuleRune should not be rendered as text (should be drawn as a line)\ngot ops: %v", ops)
	}

	// Verify that a horizontal line (fill operation) was drawn for the hrule
	// The line should span the full width of the frame and be thin (1px or similar)
	frameBackgroundRect := "(0,0)-(400,300)"
	foundHRuleLine := false
	for _, op := range ops {
		if strings.HasPrefix(op, "fill ") {
			if strings.Contains(op, frameBackgroundRect) {
				continue // Skip the frame background fill
			}
			// Look for a thin fill that spans full width (x from 0 to 400)
			// The horizontal rule line should be on line 2 (Y around 14-28 area)
			// and be 1-2px tall
			if strings.Contains(op, "(0,") && strings.Contains(op, "-(400,") {
				foundHRuleLine = true
			}
		}
	}

	if !foundHRuleLine {
		t.Errorf("Redraw() did not render horizontal rule line\nExpected a full-width fill for the hrule, got ops: %v", ops)
	}
}

// TestHorizontalRuleFullWidth tests that the horizontal rule line spans the full frame width.
func TestHorizontalRuleFullWidth(t *testing.T) {
	rect := image.Rect(0, 0, 500, 300) // Frame is 500px wide
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 14px line height

	bgImage := edwoodtest.NewImage(display, "frame-background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create content with just a horizontal rule
	content := Content{
		{Text: string(HRuleRune) + "\n", Style: StyleHRule},
	}
	f.SetContent(content)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// The horizontal rule should span from X=0 to X=500 (full frame width)
	// It should be a thin line (1-2px tall)
	frameBackgroundRect := "(0,0)-(500,300)"
	foundFullWidthLine := false

	for _, op := range ops {
		if strings.HasPrefix(op, "fill ") {
			if strings.Contains(op, frameBackgroundRect) {
				continue // Skip the frame background fill
			}
			// Look for a fill from X=0 to X=500 (full width)
			// The exact Y position depends on line height and vertical centering
			if strings.Contains(op, "(0,") && strings.Contains(op, "-(500,") {
				foundFullWidthLine = true
			}
		}
	}

	if !foundFullWidthLine {
		t.Errorf("Horizontal rule line should span full frame width (500px)\ngot ops: %v", ops)
	}
}

// TestFrameSetRect verifies that SetRect() updates the frame's rectangle.
func TestFrameSetRect(t *testing.T) {
	// Create a frame with an initial rectangle
	initialRect := image.Rect(10, 20, 200, 300)
	display := edwoodtest.NewDisplay(initialRect)

	f := NewFrame()
	f.Init(initialRect, WithDisplay(display))

	// Verify initial rect
	if got := f.Rect(); got != initialRect {
		t.Errorf("Initial Rect() = %v, want %v", got, initialRect)
	}

	// Set a new rectangle
	newRect := image.Rect(0, 0, 400, 500)
	f.SetRect(newRect)

	// Verify rect was updated
	if got := f.Rect(); got != newRect {
		t.Errorf("After SetRect(), Rect() = %v, want %v", got, newRect)
	}
}

// TestFrameSetRectNoChange verifies that SetRect() with same rectangle is a no-op.
func TestFrameSetRectNoChange(t *testing.T) {
	rect := image.Rect(10, 20, 200, 300)
	display := edwoodtest.NewDisplay(rect)

	f := NewFrame()
	f.Init(rect, WithDisplay(display))

	// Set the same rectangle
	f.SetRect(rect)

	// Verify rect is unchanged
	if got := f.Rect(); got != rect {
		t.Errorf("Rect() = %v, want %v", got, rect)
	}
}

// TestFrameSetRectRelayout verifies that layout uses the new width after SetRect().
// When the rectangle width changes, text wrapping should adapt accordingly.
func TestFrameSetRectRelayout(t *testing.T) {
	// Start with a narrow frame where "hello world" will wrap
	narrowRect := image.Rect(0, 0, 60, 200) // Only ~6 chars wide with mock font
	display := edwoodtest.NewDisplay(narrowRect)
	font := edwoodtest.NewFont(10, 14) // 10px per character

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(narrowRect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content that won't fit on one line at narrow width
	f.SetContent(Plain("hello world"))

	// Count visible lines at narrow width
	narrowLines := f.TotalLines()
	if narrowLines < 2 {
		t.Logf("Expected multiple lines at narrow width, got %d", narrowLines)
	}

	// Now widen the frame so everything fits on one line
	wideRect := image.Rect(0, 0, 200, 200) // Wide enough for "hello world"
	f.SetRect(wideRect)

	// After SetRect, TotalLines() should use the new width
	wideLines := f.TotalLines()

	// At 200px wide with 10px per char, "hello world" (11 chars = 110px) should fit
	if wideLines != 1 {
		t.Errorf("After SetRect to wider frame, TotalLines() = %d, want 1 (text should fit on one line)", wideLines)
	}

	// Verify Rect() returns the new rectangle
	if got := f.Rect(); got != wideRect {
		t.Errorf("Rect() = %v, want %v", got, wideRect)
	}
}

// TestFrameSetRectRedraw verifies that Redraw() uses the new rectangle after SetRect().
func TestFrameSetRectRedraw(t *testing.T) {
	initialRect := image.Rect(0, 0, 100, 100)
	display := edwoodtest.NewDisplay(initialRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(initialRect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))
	f.SetContent(Plain("test"))

	// Change to a new rectangle
	newRect := image.Rect(50, 50, 300, 400)
	f.SetRect(newRect)

	// Clear draw ops and redraw
	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	// Verify the background fill uses the new rectangle
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	expectedFill := fmt.Sprintf("fill %s", newRect)
	foundNewRect := false
	for _, op := range ops {
		if strings.HasPrefix(op, expectedFill) {
			foundNewRect = true
			break
		}
	}

	if !foundNewRect {
		t.Errorf("Redraw() should fill new rectangle %v\ngot ops: %v", newRect, ops)
	}
}

// TestDrawTextClipsToFrame verifies that drawText doesn't draw lines beyond
// the frame's rectangle boundary. This is a regression test for the bug where
// Markdeep preview would overwrite the window below when content exceeded the frame.
func TestDrawTextClipsToFrame(t *testing.T) {
	// Create a small frame that can only fit 2 lines (28 pixels at 14px per line)
	rect := image.Rect(0, 0, 200, 28)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with 5 lines - only 2 should fit in the frame
	f.SetContent(Plain("line1\nline2\nline3\nline4\nline5"))

	// Clear draw ops and redraw
	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	// Check that no draw operations were made with Y coordinates at or below the frame bottom
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	frameBottom := rect.Max.Y

	for _, op := range ops {
		// Look for "bytes at" operations which indicate text rendering
		// Format: "bytes at (X,Y) ..."
		if strings.Contains(op, "bytes at") {
			// Parse the Y coordinate from the operation
			var x, y int
			if n, err := fmt.Sscanf(op, "bytes at (%d,%d)", &x, &y); n == 2 && err == nil {
				if y >= frameBottom {
					t.Errorf("draw operation at Y=%d exceeds frame bottom %d: %s", y, frameBottom, op)
				}
			}
		}
	}
}
