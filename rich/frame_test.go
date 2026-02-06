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

// TestCharofptWithOrigin tests that Charofpt returns the correct content-absolute
// rune position after scrolling (non-zero origin). After SetOrigin(6) on
// "hello\nworld\nfoo", clicking at the top of the frame should return position 6
// (the 'w' of "world"), not position 0.
func TestCharofptWithOrigin(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Content: "hello\nworld\nfoo" = "hello"(5) + \n(1) + "world"(5) + \n(1) + "foo"(3) = 15 runes
	// Line 0: "hello\n" (runes 0-5)
	// Line 1: "world\n" (runes 6-11)
	// Line 2: "foo" (runes 12-14)
	f.SetContent(Plain("hello\nworld\nfoo"))

	// Scroll so "world\n" is the first visible line
	f.SetOrigin(6)

	// Click at the top-left of the frame should return rune 6 ('w' of "world")
	pt := image.Point{X: rect.Min.X, Y: rect.Min.Y}
	got := f.Charofpt(pt)
	if got != 6 {
		t.Errorf("Charofpt(%v) with origin=6: got %d, want 6", pt, got)
	}

	// Click at X offset for 3rd char on first visible line should return rune 8 ('r' of "world")
	pt = image.Point{X: rect.Min.X + 20, Y: rect.Min.Y}
	got = f.Charofpt(pt)
	if got != 8 {
		t.Errorf("Charofpt(%v) with origin=6: got %d, want 8", pt, got)
	}

	// Click on second visible line (Y offset = fontHeight) should return rune 12 ('f' of "foo")
	pt = image.Point{X: rect.Min.X, Y: rect.Min.Y + 14}
	got = f.Charofpt(pt)
	if got != 12 {
		t.Errorf("Charofpt(%v) with origin=6: got %d, want 12", pt, got)
	}
}

// TestPtofcharWithOrigin tests that Ptofchar returns the correct screen point
// for content-absolute rune positions after scrolling. After SetOrigin(6),
// Ptofchar(6) should return the frame's top-left (since rune 6 is the first
// visible character), not a position far down the frame.
func TestPtofcharWithOrigin(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Content: "hello\nworld\nfoo"
	f.SetContent(Plain("hello\nworld\nfoo"))

	// Scroll so "world\n" is the first visible line
	f.SetOrigin(6)

	// Rune 6 ('w') is now the first visible char - should be at frame origin
	pt := f.Ptofchar(6)
	want := rect.Min
	if pt != want {
		t.Errorf("Ptofchar(6) with origin=6: got %v, want %v", pt, want)
	}

	// Rune 8 ('r') is 2 chars into the first visible line
	pt = f.Ptofchar(8)
	want = image.Point{X: rect.Min.X + 20, Y: rect.Min.Y}
	if pt != want {
		t.Errorf("Ptofchar(8) with origin=6: got %v, want %v", pt, want)
	}

	// Rune 12 ('f' of "foo") is at the start of the second visible line
	pt = f.Ptofchar(12)
	want = image.Point{X: rect.Min.X, Y: rect.Min.Y + 14}
	if pt != want {
		t.Errorf("Ptofchar(12) with origin=6: got %v, want %v", pt, want)
	}
}

// TestCharofptPtofcharRoundTripWithOrigin tests that Charofpt(Ptofchar(p)) == p
// for all visible positions after scrolling with a non-zero origin.
func TestCharofptPtofcharRoundTripWithOrigin(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Content: "hello\nworld\nfoo\nbar\nbaz"
	// Line 0: "hello\n" (runes 0-5)
	// Line 1: "world\n" (runes 6-11)
	// Line 2: "foo\n" (runes 12-15)
	// Line 3: "bar\n" (runes 16-19)
	// Line 4: "baz" (runes 20-22)
	f.SetContent(Plain("hello\nworld\nfoo\nbar\nbaz"))

	// Scroll to show from "world" onwards
	f.SetOrigin(6)

	// Test round-trip for all visible rune positions (6 through 22)
	for i := 6; i <= 22; i++ {
		pt := f.Ptofchar(i)
		got := f.Charofpt(pt)
		if got != i {
			t.Errorf("origin=6: Charofpt(Ptofchar(%d)) = %d, want %d (pt=%v)", i, got, i, pt)
		}
	}

	// Also test with a different origin
	f.SetOrigin(12) // Start from "foo"

	for i := 12; i <= 22; i++ {
		pt := f.Ptofchar(i)
		got := f.Charofpt(pt)
		if got != i {
			t.Errorf("origin=12: Charofpt(Ptofchar(%d)) = %d, want %d (pt=%v)", i, got, i, pt)
		}
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
	// Set a range selection to avoid cursor tick drawing (this test is about box backgrounds)
	f.SetSelection(0, 1)

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

	// Code blocks are indented by CodeBlockIndentChars * M-width = 8 * 10 = 80 pixels
	// Background should start at x=80 and extend to frame width (400px)
	foundBlockFill := false
	frameBackgroundRect := "(0,0)-(400,300)"
	expectedIndent := CodeBlockIndentChars * font.BytesWidth([]byte("M")) // 80 pixels

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

	// Code blocks are indented by CodeBlockIndentChars * M-width = 8 * 10 = 80 pixels
	expectedIndent := CodeBlockIndentChars * font.BytesWidth([]byte("M")) // 80 pixels

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

// TestInitTickCreatesImage verifies that initTick creates a tick image with
// the correct dimensions: width = frtickw * ScaleSize(1), height = requested height.
// The tick image should have a transparent background with an opaque vertical line
// and serif boxes at top and bottom.
func TestInitTickCreatesImage(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithFont(font))

	// Before initTick, there should be no tick image
	if fi.tickImage != nil {
		t.Fatal("tickImage should be nil before initTick")
	}

	// Call initTick with a specific height
	fi.initTick(20)

	// tickImage should now be allocated
	if fi.tickImage == nil {
		t.Fatal("tickImage should not be nil after initTick")
	}

	// Verify tick dimensions: width = frtickw * scale, height = requested
	// ScaleSize(1) returns 1 in mock display, so width = 3 * 1 = 3
	tickRect := fi.tickImage.R()
	expectedWidth := frtickw * display.ScaleSize(1) // 3 * 1 = 3
	if tickRect.Dx() != expectedWidth {
		t.Errorf("tick width = %d, want %d", tickRect.Dx(), expectedWidth)
	}
	if tickRect.Dy() != 20 {
		t.Errorf("tick height = %d, want 20", tickRect.Dy())
	}

	// Verify tickHeight and tickScale fields are set
	if fi.tickHeight != 20 {
		t.Errorf("tickHeight = %d, want 20", fi.tickHeight)
	}
	if fi.tickScale != display.ScaleSize(1) {
		t.Errorf("tickScale = %d, want %d", fi.tickScale, display.ScaleSize(1))
	}

	// Calling initTick with a different height should create a new image
	fi.initTick(30)
	if fi.tickImage == nil {
		t.Fatal("tickImage should not be nil after initTick with new height")
	}
	tickRect = fi.tickImage.R()
	if tickRect.Dy() != 30 {
		t.Errorf("tick height after resize = %d, want 30", tickRect.Dy())
	}
	if fi.tickHeight != 30 {
		t.Errorf("tickHeight after resize = %d, want 30", fi.tickHeight)
	}
}

// TestInitTickReusesForSameHeight verifies that calling initTick with the same
// height does not reallocate the image - it reuses the existing one.
func TestInitTickReusesForSameHeight(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithFont(font))

	// Create tick with height 20
	fi.initTick(20)
	if fi.tickImage == nil {
		t.Fatal("tickImage should not be nil after initTick")
	}

	// Record the image pointer
	firstImage := fi.tickImage

	// Clear draw ops to track what happens next
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call initTick again with the same height
	fi.initTick(20)

	// The image should be reused (same pointer)
	if fi.tickImage != firstImage {
		t.Error("initTick should reuse existing image when height hasn't changed")
	}

	// No new AllocImage calls should have been made
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	for _, op := range ops {
		if strings.Contains(op, "AllocImage") {
			t.Errorf("unexpected AllocImage call when height unchanged: %s", op)
		}
	}
}

// TestBoxHeightBody verifies that boxHeight returns the body font height
// for a regular text box with no special styling.
func TestBoxHeightBody(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithFont(font))

	box := Box{
		Text:  []byte("hello"),
		Nrune: 5,
		Style: Style{},
	}

	h := fi.boxHeight(box)
	if h != 14 {
		t.Errorf("boxHeight for body text = %d, want 14", h)
	}
}

// TestBoxHeightHeading verifies that boxHeight returns the heading font height
// for a box styled with Scale 2.0 (H1 heading).
func TestBoxHeightHeading(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)
	h1Font := edwoodtest.NewFont(20, 28) // H1 is larger

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithFont(font), WithScaledFont(2.0, h1Font))

	box := Box{
		Text:  []byte("Heading"),
		Nrune: 7,
		Style: Style{Scale: 2.0, Bold: true},
	}

	h := fi.boxHeight(box)
	if h != 28 {
		t.Errorf("boxHeight for H1 heading = %d, want 28", h)
	}
}

// TestBoxHeightImage verifies that boxHeight returns the scaled image height
// for an image box, using imageBoxDimensions with the frame width.
func TestBoxHeightImage(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithFont(font))

	box := Box{
		Style: Style{
			Image:    true,
			ImageURL: "test.png",
		},
		ImageData: &CachedImage{
			Width:    200,
			Height:   100,
			Data:     []byte{0},
			Original: image.NewRGBA(image.Rect(0, 0, 200, 100)),
		},
	}

	h := fi.boxHeight(box)
	// Image is 200px wide, frame is 400px, so no scaling. Height = 100.
	if h != 100 {
		t.Errorf("boxHeight for image = %d, want 100", h)
	}
}

// TestDrawTickAtCursor verifies that drawTickTo draws a tick (cursor bar)
// when the selection is a point (p0 == p1). The tick should be drawn using
// display.Black() as the source and the tick image as mask, at the correct
// cursor position.
func TestDrawTickAtCursor(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with a simple word and cursor at position 3 (between "hel" and "lo")
	f.SetContent(Plain("hello"))
	fi.p0 = 3
	fi.p1 = 3

	// Allocate scratch image to draw to (same as Redraw does)
	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}

	// Clear draw ops
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call drawTickTo
	fi.drawTickTo(scratch, image.ZP)

	// Verify that a Draw call was made at the expected tick position.
	// Cursor at position 3 in "hello" with font width 10 → X=30.
	// Tick width = frtickw * scale = 3 * 1 = 3. So tick rect is (30,0)-(33,14).
	// Note: the mock's Draw records ops as "fill" format which doesn't include
	// the source name, so we verify by checking the tick rectangle coordinates.
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	foundTick := false
	for _, op := range ops {
		if strings.Contains(op, "(30,0)-(33,14)") {
			foundTick = true
			break
		}
	}
	if !foundTick {
		t.Errorf("drawTickTo() did not draw tick at expected position (30,0)-(33,14)\ngot ops: %v", ops)
	}

	// Verify the tick image was created
	if fi.tickImage == nil {
		t.Error("drawTickTo() should have created tickImage via initTick")
	}
}

// TestNoTickWithSelection verifies that drawTickTo does NOT draw a tick
// when there is a range selection (p0 != p1). The caller (Redraw) should
// only call drawTickTo when p0 == p1, but we verify the method itself
// also does nothing useful when called with a selection.
func TestNoTickWithSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	selImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// Set content with a range selection (p0 != p1)
	f.SetContent(Plain("hello world"))
	f.SetSelection(2, 5)

	// Clear draw ops
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw - it should NOT call drawTickTo because p0 != p1
	f.Redraw()

	// Since p0 != p1, Redraw should not call drawTickTo, so no tick image
	// should be created.
	fi := f.(*frameImpl)
	if fi.tickImage != nil {
		t.Error("Redraw() with range selection should not create tickImage")
	}
}

// TestTickHeightScaling verifies that the tick height is determined by the
// tallest adjacent box. When the cursor is between a heading box and a body
// text box, the tick should use the heading height (the taller of the two).
func TestTickHeightScaling(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)
	h1Font := edwoodtest.NewFont(20, 28) // H1 heading font is taller

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithScaledFont(2.0, h1Font),
	)

	// Content: "Hi" as H1 heading, newline, then "body" as body text.
	// Cursor at position 2 = end of "Hi" on the heading line.
	// Adjacent boxes: the heading text "Hi" (height 28) and the newline.
	// The tick should be at least as tall as the heading font (28).
	content := Content{
		{Text: "Hi", Style: Style{Scale: 2.0, Bold: true}},
		{Text: "\n", Style: DefaultStyle()},
		{Text: "body", Style: DefaultStyle()},
	}
	f.SetContent(content)
	fi.p0 = 2 // End of "Hi"
	fi.p1 = 2

	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}

	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawTickTo(scratch, image.ZP)

	// The tick should have been created with the heading height (28)
	if fi.tickImage == nil {
		t.Fatal("drawTickTo should have created tickImage")
	}
	if fi.tickHeight != 28 {
		t.Errorf("tick height = %d, want 28 (heading font height)", fi.tickHeight)
	}
}

// TestTickHeightBodyText verifies that when the cursor is between two
// body text boxes, the tick height equals the body font height.
func TestTickHeightBodyText(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
	)

	// Plain text: "hello world" — two words separated by space.
	// Cursor at position 5 (after "hello", before space/next word).
	f.SetContent(Plain("hello world"))
	fi.p0 = 5
	fi.p1 = 5

	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}

	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawTickTo(scratch, image.ZP)

	// The tick should be body font height (14)
	if fi.tickImage == nil {
		t.Fatal("drawTickTo should have created tickImage")
	}
	if fi.tickHeight != 14 {
		t.Errorf("tick height = %d, want 14 (body font height)", fi.tickHeight)
	}
}

// TestRedrawDrawsTickWhenCursorPoint verifies that Redraw() draws the cursor
// tick when the selection is a point (p0 == p1). After drawing text and
// selection, Redraw should call drawTickTo to render the insertion cursor.
func TestRedrawDrawsTickWhenCursorPoint(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
	)

	// Set content with cursor at position 3 (point selection)
	f.SetContent(Plain("hello"))
	fi.p0 = 3
	fi.p1 = 3

	// Clear draw ops before Redraw
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw — it should draw the tick since p0 == p1
	f.Redraw()

	// Verify that a tick was drawn. The cursor at position 3 in "hello"
	// with font width 10 → X=30 in scratch coords.
	// Tick rect is (30,0)-(33,14) in scratch image coords.
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	foundTick := false
	for _, op := range ops {
		if strings.Contains(op, "(30,0)-(33,14)") {
			foundTick = true
			break
		}
	}
	if !foundTick {
		t.Errorf("Redraw() with p0==p1 should draw tick at (30,0)-(33,14)\ngot ops: %v", ops)
	}

	// Verify tick image was created
	if fi.tickImage == nil {
		t.Error("Redraw() with point selection should create tickImage")
	}
}

// TestRedrawNoTickWhenSelection verifies that Redraw() does NOT draw the
// cursor tick when there is a range selection (p0 != p1). The tick should
// only appear when the selection is a point.
func TestRedrawNoTickWhenSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	selImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// Set content with a range selection (p0 != p1)
	f.SetContent(Plain("hello world"))
	f.SetSelection(2, 5)

	// Clear draw ops before Redraw
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw — it should NOT draw the tick since p0 != p1
	f.Redraw()

	// Verify no tick image was created
	if fi.tickImage != nil {
		t.Error("Redraw() with range selection (p0 != p1) should not create tickImage")
	}
}

// TestDrawImageScaled verifies that drawImageTo pre-scales images when the
// display size (from imageBoxDimensions) differs from the original image size.
// When Style.ImageWidth is set, the image should be rendered at the scaled
// dimensions rather than drawn at original size and clipped.
func TestDrawImageScaled(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create a 200x100 original image
	origImg := image.NewRGBA(image.Rect(0, 0, 200, 100))
	// Fill with a solid color so pixel data is non-trivial
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			origImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	plan9Data, err := ConvertToPlan9(origImg)
	if err != nil {
		t.Fatalf("ConvertToPlan9 failed: %v", err)
	}

	// Image box with explicit width=100px → should scale to 100x50
	pb := PositionedBox{
		X: 0,
		Box: Box{
			Style: Style{
				Image:      true,
				ImageURL:   "test.png",
				ImageWidth: 100, // Explicit width causes scaling
			},
			Wid:   100,
			Nrune: 1,
			ImageData: &CachedImage{
				Width:    200,
				Height:   100,
				Data:     plan9Data,
				Original: origImg,
			},
		},
	}

	line := Line{Y: 0, Height: 50}

	// Verify imageBoxDimensions returns the scaled size
	scaledW, scaledH := imageBoxDimensions(&pb.Box, rect.Dx())
	if scaledW != 100 || scaledH != 50 {
		t.Fatalf("imageBoxDimensions = (%d, %d), want (100, 50)", scaledW, scaledH)
	}

	// Allocate scratch image
	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}

	// Clear draw ops before drawImageTo
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call drawImageTo — this should pre-scale the image
	fi.drawImageTo(scratch, pb, line, image.ZP, rect.Dx(), rect.Dy())

	// Verify draw operations were emitted
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Fatal("drawImageTo() produced no draw operations")
	}

	// The key assertion: the source image allocated for rendering should be
	// at the scaled dimensions (100x50), not the original (200x100).
	// When pre-scaling is implemented, AllocImage will be called with
	// (0,0)-(100,50) for the source, and the Draw call will target (0,0)-(100,50).
	// Look for a draw op that references the scaled destination rect.
	foundScaledDraw := false
	for _, op := range ops {
		// The draw op should reference the scaled destination (0,0)-(100,50)
		if strings.Contains(op, "(0,0)-(100,50)") {
			foundScaledDraw = true
			break
		}
	}
	if !foundScaledDraw {
		t.Errorf("drawImageTo() should draw at scaled dimensions (0,0)-(100,50), got ops:\n%s",
			strings.Join(ops, "\n"))
	}

	// Negative check: the draw should NOT use the original 200x100 dimensions
	for _, op := range ops {
		if strings.Contains(op, "(0,0)-(200,100)") {
			t.Errorf("drawImageTo() should NOT draw at original dimensions (0,0)-(200,100), got op: %s", op)
		}
	}
}

// TestHScrollOriginsPreservedOnStableCount verifies that when the block region
// count does not change across syncHScrollState calls, existing scroll offsets
// are preserved.
func TestHScrollOriginsPreservedOnStableCount(t *testing.T) {
	f := NewFrame()
	fi := f.(*frameImpl)

	// Simulate initial layout with 2 block regions
	fi.syncHScrollState(2)
	if len(fi.hscrollOrigins) != 2 {
		t.Fatalf("hscrollOrigins length = %d, want 2", len(fi.hscrollOrigins))
	}
	if fi.hscrollBlockCount != 2 {
		t.Fatalf("hscrollBlockCount = %d, want 2", fi.hscrollBlockCount)
	}

	// Set some scroll offsets
	fi.SetHScrollOrigin(0, 50)
	fi.SetHScrollOrigin(1, 120)

	// Simulate re-layout with the same block count
	fi.syncHScrollState(2)

	// Offsets should be preserved
	if got := fi.GetHScrollOrigin(0); got != 50 {
		t.Errorf("after stable re-layout, GetHScrollOrigin(0) = %d, want 50", got)
	}
	if got := fi.GetHScrollOrigin(1); got != 120 {
		t.Errorf("after stable re-layout, GetHScrollOrigin(1) = %d, want 120", got)
	}
}

// TestHScrollOriginsResetOnCountChange verifies that when the block region
// count changes, all scroll offsets are reset to zero.
func TestHScrollOriginsResetOnCountChange(t *testing.T) {
	f := NewFrame()
	fi := f.(*frameImpl)

	// Start with 2 block regions and set offsets
	fi.syncHScrollState(2)
	fi.SetHScrollOrigin(0, 50)
	fi.SetHScrollOrigin(1, 120)

	// Re-layout produces 3 block regions (count changed)
	fi.syncHScrollState(3)

	if len(fi.hscrollOrigins) != 3 {
		t.Fatalf("hscrollOrigins length = %d, want 3", len(fi.hscrollOrigins))
	}
	if fi.hscrollBlockCount != 3 {
		t.Fatalf("hscrollBlockCount = %d, want 3", fi.hscrollBlockCount)
	}

	// All offsets should be zero
	for i := 0; i < 3; i++ {
		if got := fi.GetHScrollOrigin(i); got != 0 {
			t.Errorf("after count change, GetHScrollOrigin(%d) = %d, want 0", i, got)
		}
	}

	// Also verify decreasing count resets
	fi.syncHScrollState(3) // same count
	fi.SetHScrollOrigin(0, 42)
	fi.syncHScrollState(1) // count decreased

	if len(fi.hscrollOrigins) != 1 {
		t.Fatalf("hscrollOrigins length = %d, want 1", len(fi.hscrollOrigins))
	}
	if got := fi.GetHScrollOrigin(0); got != 0 {
		t.Errorf("after decrease, GetHScrollOrigin(0) = %d, want 0", got)
	}
}

// TestSetGetHScrollOrigin verifies the getter/setter methods for horizontal
// scroll origins, including out-of-range handling.
func TestSetGetHScrollOrigin(t *testing.T) {
	f := NewFrame()
	fi := f.(*frameImpl)

	// Before any sync, get should return 0 for any index
	if got := fi.GetHScrollOrigin(0); got != 0 {
		t.Errorf("before sync, GetHScrollOrigin(0) = %d, want 0", got)
	}
	if got := fi.GetHScrollOrigin(-1); got != 0 {
		t.Errorf("GetHScrollOrigin(-1) = %d, want 0", got)
	}

	// Set up 3 regions
	fi.syncHScrollState(3)

	// Set and get each
	fi.SetHScrollOrigin(0, 10)
	fi.SetHScrollOrigin(1, 200)
	fi.SetHScrollOrigin(2, 0)

	if got := fi.GetHScrollOrigin(0); got != 10 {
		t.Errorf("GetHScrollOrigin(0) = %d, want 10", got)
	}
	if got := fi.GetHScrollOrigin(1); got != 200 {
		t.Errorf("GetHScrollOrigin(1) = %d, want 200", got)
	}
	if got := fi.GetHScrollOrigin(2); got != 0 {
		t.Errorf("GetHScrollOrigin(2) = %d, want 0", got)
	}

	// Out-of-range set should be ignored (no panic)
	fi.SetHScrollOrigin(3, 999)
	fi.SetHScrollOrigin(-1, 999)

	// Out-of-range get returns 0
	if got := fi.GetHScrollOrigin(3); got != 0 {
		t.Errorf("GetHScrollOrigin(3) = %d, want 0", got)
	}
	if got := fi.GetHScrollOrigin(-1); got != 0 {
		t.Errorf("GetHScrollOrigin(-1) = %d, want 0", got)
	}

	// Overwrite an existing value
	fi.SetHScrollOrigin(1, 300)
	if got := fi.GetHScrollOrigin(1); got != 300 {
		t.Errorf("after overwrite, GetHScrollOrigin(1) = %d, want 300", got)
	}
}

// TestRenderWithHorizontalOffset verifies that when an hscrollOrigin is set
// for a block region, text within that region is drawn at an X position
// shifted left by the scroll offset. Block backgrounds (phase 1) should
// remain full-width and unshifted, while text (phase 4) and box backgrounds
// (phase 2) for block code are shifted by -hOffset.
func TestRenderWithHorizontalOffset(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	// Code block indent = 8 * 10 = 80px (CodeBlockIndentChars * font width).
	// "a_very_long_code_line_xxxxx" = 27 chars * 10px = 270px content.
	// At indent 80, total extent = 350px, which exceeds 200px frame width.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_xxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
		{Text: "short", Style: codeStyle},
		{Text: "\n", Style: Style{Scale: 1.0}},
		{Text: "normal", Style: Style{Scale: 1.0}},
	}
	f.SetContent(content)

	// First render without scroll offset to get the baseline X positions
	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}

	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawTextTo(scratch, image.ZP)

	baseOps := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Find the X position of "a_very_long_code_line_xxxxx" text draw
	baseCodeX := -1
	for _, op := range baseOps {
		if strings.Contains(op, `"a_very_long_code_line_xxxxx"`) && strings.Contains(op, "string") {
			// Extract X from "atpoint: (X,Y)"
			baseCodeX = extractXFromOp(op)
			break
		}
	}
	if baseCodeX < 0 {
		t.Fatalf("could not find code text draw op in base render; ops: %v", baseOps)
	}

	// Now set a horizontal scroll offset and re-render
	hOffset := 30
	fi.SetHScrollOrigin(0, hOffset)

	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawTextTo(scratch, image.ZP)

	scrolledOps := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Find the X position of the code text after scrolling
	scrolledCodeX := -1
	for _, op := range scrolledOps {
		if strings.Contains(op, `"a_very_long_code_line_xxxxx"`) && strings.Contains(op, "string") {
			scrolledCodeX = extractXFromOp(op)
			break
		}
	}
	if scrolledCodeX < 0 {
		t.Fatalf("could not find code text draw op in scrolled render; ops: %v", scrolledOps)
	}

	// The scrolled X should be shifted left by hOffset
	expectedX := baseCodeX - hOffset
	if scrolledCodeX != expectedX {
		t.Errorf("scrolled code text X = %d, want %d (base %d shifted by -%d)",
			scrolledCodeX, expectedX, baseCodeX, hOffset)
	}

	// Verify "normal" text (not in a block region) is NOT shifted
	baseNormalX := -1
	scrolledNormalX := -1
	for _, op := range baseOps {
		if strings.Contains(op, `"normal"`) && strings.Contains(op, "string") {
			baseNormalX = extractXFromOp(op)
			break
		}
	}
	for _, op := range scrolledOps {
		if strings.Contains(op, `"normal"`) && strings.Contains(op, "string") {
			scrolledNormalX = extractXFromOp(op)
			break
		}
	}
	if baseNormalX >= 0 && scrolledNormalX >= 0 && baseNormalX != scrolledNormalX {
		t.Errorf("normal text X changed from %d to %d after hscroll; should not be affected",
			baseNormalX, scrolledNormalX)
	}
}

// TestRenderClipsAboveScrollbar verifies that content within a block region
// that has a horizontal scrollbar does not draw into the scrollbar area.
// The scrollbar occupies the bottom Scrollwid pixels of the block region.
// Text on lines within such a block should be clipped so it doesn't overlap
// the scrollbar.
func TestRenderClipsAboveScrollbar(t *testing.T) {
	// Use a small frame where block code overflows, creating a scrollbar.
	// scrollbarHeight = 12 (Scrollwid). Frame height = 60 to keep things manageable.
	rect := image.Rect(0, 0, 100, 60)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Block code content that overflows 100px frame width.
	// Code indent = 4 * 10 = 40px. "long_code_xxx" = 13 chars * 10 = 130px.
	// Total extent = 170px > 100px, so this block gets a scrollbar.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "long_code_xxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
		{Text: "line2_code_xx", Style: codeStyle},
		{Text: "\n", Style: Style{Scale: 1.0}},
	}
	f.SetContent(content)

	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}

	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawTextTo(scratch, image.ZP)

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Verify that text draw ops for code lines have Y positions that don't
	// extend into the scrollbar area. The scrollbar is at the bottom of the
	// block region. With two code lines at height 14 each, the block spans
	// Y=0..28. The scrollbar would be at Y=28..40 (12px high).
	// Text on line 2 starts at Y=14 and has height 14, so it ends at Y=28.
	// This text should NOT be drawn if its Y + height would overlap
	// the scrollbar region.
	//
	// We check that no text draw operation for code content has a Y coordinate
	// that would place it at or beyond the scrollbar Y position.
	scrollbarHeight := 12 // Matches Scrollwid

	for _, op := range ops {
		if !strings.Contains(op, "string") {
			continue
		}
		// Extract Y from the operation
		y := extractYFromOp(op)
		if y < 0 {
			continue
		}

		// For code text, check it doesn't overlap the scrollbar area.
		// The block's content area ends at (blockBottomY - scrollbarHeight).
		// With two 14px lines: blockBottomY = 28, content should end at 28 - 12 = 16.
		// Line 1 at Y=0 (renders at 0..14) is fine.
		// Line 2 at Y=14 (renders at 14..28) should be clipped at Y=16.
		//
		// Note: The exact clipping behavior depends on implementation.
		// At minimum, text at Y positions that would fully overlap the scrollbar
		// (Y >= blockBottomY - scrollbarHeight + lineHeight) should not be drawn.
		if strings.Contains(op, "long_code_xxx") || strings.Contains(op, "line2_code_xx") {
			// Text Y + font height should not exceed the block's usable area
			textBottom := y + 14 // font height
			blockBottomY := 28   // 2 lines * 14px
			maxContentY := blockBottomY + scrollbarHeight // after two-pass adjustment
			if textBottom > maxContentY {
				t.Errorf("code text drawn at Y=%d extends to %d, past block content limit %d (scrollbar starts there); op: %s",
					y, textBottom, maxContentY, op)
			}
		}
	}
}

// extractXFromOp extracts the X coordinate from a draw op string like
// "... atpoint: (X,Y) ...". Returns -1 if not found.
func extractXFromOp(op string) int {
	// Look for "atpoint: (X,Y)"
	idx := strings.Index(op, "atpoint: (")
	if idx < 0 {
		return -1
	}
	rest := op[idx+len("atpoint: ("):]
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return -1
	}
	var x int
	_, err := fmt.Sscanf(rest[:commaIdx], "%d", &x)
	if err != nil {
		return -1
	}
	return x
}

// extractYFromOp extracts the Y coordinate from a draw op string like
// "... atpoint: (X,Y) ...". Returns -1 if not found.
func extractYFromOp(op string) int {
	// Look for "atpoint: (X,Y)"
	idx := strings.Index(op, "atpoint: (")
	if idx < 0 {
		return -1
	}
	rest := op[idx+len("atpoint: ("):]
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return -1
	}
	rest = rest[commaIdx+1:]
	parenIdx := strings.Index(rest, ")")
	if parenIdx < 0 {
		return -1
	}
	var y int
	_, err := fmt.Sscanf(rest[:parenIdx], "%d", &y)
	if err != nil {
		return -1
	}
	return y
}

// extractDrawRect extracts the rectangle from a draw operation string.
// Supports both "draw r: (x0,y0)-(x1,y1)" and "fill (x0,y0)-(x1,y1)" formats.
// Returns the rectangle and true if found, or zero rect and false if not.
func extractDrawRect(op string) (image.Rectangle, bool) {
	// Try "draw r: " format first
	idx := strings.Index(op, "draw r: ")
	if idx >= 0 {
		rest := op[idx+len("draw r: "):]
		var x0, y0, x1, y1 int
		n, err := fmt.Sscanf(rest, "(%d,%d)-(%d,%d)", &x0, &y0, &x1, &y1)
		if err == nil && n == 4 {
			return image.Rect(x0, y0, x1, y1), true
		}
	}
	// Try "fill " format
	idx = strings.Index(op, "fill ")
	if idx >= 0 {
		rest := op[idx+len("fill "):]
		var x0, y0, x1, y1 int
		n, err := fmt.Sscanf(rest, "(%d,%d)-(%d,%d)", &x0, &y0, &x1, &y1)
		if err == nil && n == 4 {
			return image.Rect(x0, y0, x1, y1), true
		}
	}
	return image.Rectangle{}, false
}

// TestDrawHScrollbar verifies that an overflowing block region gets a horizontal
// scrollbar drawn at the bottom of the block. The scrollbar consists of a background
// fill and a thumb fill in the scrollbar area.
func TestDrawHScrollbar(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	// Code indent = 4 * 10 = 40px. "a_very_long_code_line_xxxxx" = 27 chars * 10 = 270px.
	// Total extent = 310px > 200px, so this block overflows.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_xxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	// Run layout to populate lines and regions
	lines, _ := fi.layoutFromOrigin()
	regions := findBlockRegions(lines)
	scrollbarHeight := 12
	adjustedRegions := adjustLayoutForScrollbars(lines, regions, rect.Dx(), scrollbarHeight)

	// Verify we have an overflowing region
	if len(adjustedRegions) == 0 {
		t.Fatal("expected at least one block region")
	}
	if !adjustedRegions[0].HasScrollbar {
		t.Fatal("expected block region to have scrollbar (content overflows frame width)")
	}

	// Draw scrollbars
	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}
	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawHScrollbarsTo(scratch, image.ZP, lines, adjustedRegions, rect.Dx())

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// There should be at least two draw operations in the scrollbar area:
	// one for the scrollbar background, one for the thumb.
	// The scrollbar Y is at adjustedRegions[0].ScrollbarY and has height scrollbarHeight.
	scrollbarY := adjustedRegions[0].ScrollbarY
	scrollbarBottom := scrollbarY + scrollbarHeight

	drawOpsInScrollbarArea := 0
	for _, op := range ops {
		r, ok := extractDrawRect(op)
		if !ok {
			continue
		}
		// Check if this draw operation overlaps the scrollbar Y range
		if r.Min.Y >= scrollbarY && r.Max.Y <= scrollbarBottom {
			drawOpsInScrollbarArea++
		}
	}

	if drawOpsInScrollbarArea < 2 {
		t.Errorf("expected at least 2 draw ops in scrollbar area (Y=%d..%d), got %d; ops: %v",
			scrollbarY, scrollbarBottom, drawOpsInScrollbarArea, ops)
	}
}

// TestHScrollbarThumbPosition verifies that the scrollbar thumb position
// reflects the current horizontal scroll offset. When scrolled to offset 0,
// the thumb should be at the left edge. When scrolled partway, the thumb should
// be proportionally positioned.
func TestHScrollbarThumbPosition(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Content: 40px indent + 430px text = 470px total, frame is 200px wide.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	longText := "abcdefghijklmnopqrstuvwxyz_abcdefghijklmno" // 43 chars * 10 = 430px
	content := Content{
		{Text: longText, Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	lines, _ := fi.layoutFromOrigin()
	regions := findBlockRegions(lines)
	scrollbarHeight := 12
	frameWidth := rect.Dx()
	adjustedRegions := adjustLayoutForScrollbars(lines, regions, frameWidth, scrollbarHeight)

	if len(adjustedRegions) == 0 || !adjustedRegions[0].HasScrollbar {
		t.Fatal("expected an overflowing block region with scrollbar")
	}

	scrollbarY := adjustedRegions[0].ScrollbarY

	// Draw with hscroll offset = 0 (thumb at left)
	fi.syncHScrollState(len(adjustedRegions))
	fi.SetHScrollOrigin(0, 0)

	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}
	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawHScrollbarsTo(scratch, image.ZP, lines, adjustedRegions, frameWidth)
	opsAtZero := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Find the thumb rect (the narrowest draw in the scrollbar area).
	// The background spans the full scrollbar width; the thumb is narrower.
	thumbXAtZero := -1
	thumbWidthAtZero := frameWidth + 1
	for _, op := range opsAtZero {
		r, ok := extractDrawRect(op)
		if !ok {
			continue
		}
		if r.Min.Y >= scrollbarY && r.Max.Y <= scrollbarY+scrollbarHeight && r.Dx() < thumbWidthAtZero {
			thumbXAtZero = r.Min.X
			thumbWidthAtZero = r.Dx()
		}
	}

	// Now draw with a non-zero offset (thumb should shift right)
	maxContentWidth := adjustedRegions[0].MaxContentWidth
	maxScrollable := maxContentWidth - frameWidth
	halfOffset := maxScrollable / 2
	fi.SetHScrollOrigin(0, halfOffset)

	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawHScrollbarsTo(scratch, image.ZP, lines, adjustedRegions, frameWidth)
	opsAtHalf := display.(edwoodtest.GettableDrawOps).DrawOps()

	thumbXAtHalf := -1
	thumbWidthAtHalf := frameWidth + 1
	for _, op := range opsAtHalf {
		r, ok := extractDrawRect(op)
		if !ok {
			continue
		}
		if r.Min.Y >= scrollbarY && r.Max.Y <= scrollbarY+scrollbarHeight && r.Dx() < thumbWidthAtHalf {
			thumbXAtHalf = r.Min.X
			thumbWidthAtHalf = r.Dx()
		}
	}

	if thumbXAtZero < 0 || thumbXAtHalf < 0 {
		t.Fatalf("could not find thumb draw ops; atZero=%d, atHalf=%d; opsAtZero=%v, opsAtHalf=%v",
			thumbXAtZero, thumbXAtHalf, opsAtZero, opsAtHalf)
	}

	// When scrolled halfway, the thumb should be further right than at offset 0
	if thumbXAtHalf <= thumbXAtZero {
		t.Errorf("thumb X at half scroll (%d) should be greater than at zero scroll (%d)",
			thumbXAtHalf, thumbXAtZero)
	}
}

// TestNoHScrollbarWhenFits verifies that a block region whose content fits
// within the frame width does NOT get a horizontal scrollbar drawn.
func TestNoHScrollbarWhenFits(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Short code block: 40px indent + 50px text = 90px, well within 400px frame.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "short", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	lines, _ := fi.layoutFromOrigin()
	regions := findBlockRegions(lines)
	scrollbarHeight := 12
	frameWidth := rect.Dx()
	adjustedRegions := adjustLayoutForScrollbars(lines, regions, frameWidth, scrollbarHeight)

	// The region should exist but should NOT have a scrollbar
	if len(adjustedRegions) == 0 {
		t.Fatal("expected at least one block region")
	}
	if adjustedRegions[0].HasScrollbar {
		t.Fatal("block region should NOT have scrollbar (content fits in frame)")
	}

	// Drawing scrollbars should produce no draw operations for non-overflowing regions
	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}
	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawHScrollbarsTo(scratch, image.ZP, lines, adjustedRegions, frameWidth)

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) > 0 {
		t.Errorf("expected no draw ops for non-overflowing block region, got %d: %v", len(ops), ops)
	}
}

// TestHScrollbarMinThumbWidth verifies that even when content is very wide
// (making the visible fraction tiny), the scrollbar thumb is at least 10 pixels wide.
func TestHScrollbarMinThumbWidth(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Very wide content: 40px indent + 5000px text. Frame is 200px.
	// thumbWidth = (200/5040) * 200 ≈ 7.9, which is below the 10px minimum.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	// 500 chars * 10px = 5000px
	veryLongText := strings.Repeat("x", 500)
	content := Content{
		{Text: veryLongText, Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	lines, _ := fi.layoutFromOrigin()
	regions := findBlockRegions(lines)
	scrollbarHeight := 12
	frameWidth := rect.Dx()
	adjustedRegions := adjustLayoutForScrollbars(lines, regions, frameWidth, scrollbarHeight)

	if len(adjustedRegions) == 0 || !adjustedRegions[0].HasScrollbar {
		t.Fatal("expected an overflowing block region with scrollbar")
	}

	scrollbarY := adjustedRegions[0].ScrollbarY
	fi.syncHScrollState(len(adjustedRegions))
	fi.SetHScrollOrigin(0, 0)

	scratch := fi.ensureScratchImage()
	if scratch == nil {
		t.Fatal("could not allocate scratch image")
	}
	display.(edwoodtest.GettableDrawOps).Clear()
	fi.drawHScrollbarsTo(scratch, image.ZP, lines, adjustedRegions, frameWidth)

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Find the thumb draw op (narrower than full width in the scrollbar area)
	minThumbWidth := 10
	thumbFound := false
	for _, op := range ops {
		r, ok := extractDrawRect(op)
		if !ok {
			continue
		}
		if r.Min.Y >= scrollbarY && r.Max.Y <= scrollbarY+scrollbarHeight && r.Dx() < frameWidth {
			thumbFound = true
			if r.Dx() < minThumbWidth {
				t.Errorf("thumb width %d is below minimum %d", r.Dx(), minThumbWidth)
			}
			break
		}
	}
	if !thumbFound {
		t.Errorf("could not find thumb draw op in scrollbar area; ops: %v", ops)
	}
}

// TestHScrollBarAtHit verifies that a point inside a horizontal scrollbar
// rectangle returns the correct region index and ok=true.
func TestHScrollBarAtHit(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_xxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	// Trigger layout and rendering so hscrollRegions is populated.
	f.Redraw()

	// The scrollbar should exist. Look up the cached region to find the scrollbar Y.
	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}

	scrollbarHeight := 12
	// A point inside the scrollbar area (midpoint of the scrollbar).
	hitPt := image.Point{
		X: rect.Min.X + 100,
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}

	regionIndex, ok := f.HScrollBarAt(hitPt)
	if !ok {
		t.Errorf("HScrollBarAt(%v) returned ok=false, expected hit on scrollbar at Y=%d..%d",
			hitPt, ar.ScrollbarY, ar.ScrollbarY+scrollbarHeight)
	}
	if regionIndex != 0 {
		t.Errorf("HScrollBarAt(%v) returned regionIndex=%d, expected 0", hitPt, regionIndex)
	}
}

// TestHScrollBarAtMiss verifies that a point outside any horizontal scrollbar
// rectangle returns ok=false.
func TestHScrollBarAtMiss(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_xxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	// Trigger layout and rendering so hscrollRegions is populated.
	f.Redraw()

	// A point well above the scrollbar (in the text area).
	missPt := image.Point{X: rect.Min.X + 100, Y: rect.Min.Y + 5}

	_, ok := f.HScrollBarAt(missPt)
	if ok {
		t.Errorf("HScrollBarAt(%v) returned ok=true, expected miss (point is in text area, not scrollbar)", missPt)
	}
}

// TestHScrollBarAtNoOverflow verifies that when no block region overflows,
// HScrollBarAt always returns ok=false.
func TestHScrollBarAtNoOverflow(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Short code block that fits within the 400px frame.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "short", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	// Trigger layout and rendering.
	f.Redraw()

	// Try a point that would be in the scrollbar area if one existed.
	// With no overflow, there's no scrollbar, so this should miss.
	testPt := image.Point{X: rect.Min.X + 100, Y: rect.Min.Y + 20}
	_, ok := f.HScrollBarAt(testPt)
	if ok {
		t.Errorf("HScrollBarAt(%v) returned ok=true, expected false (no scrollbar for non-overflowing block)", testPt)
	}
}

// TestHScrollClickB1ScrollsLeft verifies that clicking B1 on a horizontal
// scrollbar scrolls the block region to the left (decreases the scroll offset).
// The amount scrolled is proportional to the click X position within the
// scrollbar: clicking near the left edge scrolls more, near the right edge less.
func TestHScrollClickB1ScrollsLeft(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)
	f.Redraw()

	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}

	// Set an initial scroll offset (scrolled to the right).
	maxScrollable := ar.MaxContentWidth - rect.Dx()
	initialOffset := maxScrollable / 2
	fi.SetHScrollOrigin(0, initialOffset)

	// Click B1 near the middle of the scrollbar (should scroll left by a moderate amount).
	scrollbarHeight := 12
	clickPt := image.Point{
		X: rect.Min.X + rect.Dx()/2, // middle of scrollbar
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}
	f.HScrollClick(1, clickPt, 0)

	newOffset := fi.GetHScrollOrigin(0)
	if newOffset >= initialOffset {
		t.Errorf("B1 click should scroll left (decrease offset): got %d, initial was %d", newOffset, initialOffset)
	}
	if newOffset < 0 {
		t.Errorf("B1 click should not produce negative offset: got %d", newOffset)
	}
}

// TestHScrollClickB2JumpsToPosition verifies that clicking B2 on a horizontal
// scrollbar jumps to an absolute position proportional to the click X location.
func TestHScrollClickB2JumpsToPosition(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)
	f.Redraw()

	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}

	maxScrollable := ar.MaxContentWidth - rect.Dx()

	// Click B2 at the midpoint of the scrollbar — should jump to ~50% of maxScrollable.
	// The scrollbar starts at LeftIndent, so midpoint is at LeftIndent + (frameWidth - LeftIndent)/2.
	scrollbarHeight := 12
	scrollbarMidX := rect.Min.X + ar.LeftIndent + (rect.Dx()-ar.LeftIndent)/2
	clickPt := image.Point{
		X: scrollbarMidX,
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}
	f.HScrollClick(2, clickPt, 0)

	newOffset := fi.GetHScrollOrigin(0)
	expectedApprox := maxScrollable / 2
	tolerance := maxScrollable / 5 // 20% tolerance
	if newOffset < expectedApprox-tolerance || newOffset > expectedApprox+tolerance {
		t.Errorf("B2 click at midpoint should jump to ~%d, got %d (tolerance ±%d)", expectedApprox, newOffset, tolerance)
	}

	// Click B2 at the left edge of the scrollbar — should jump to ~0.
	clickPtLeft := image.Point{
		X: rect.Min.X + ar.LeftIndent,
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}
	f.HScrollClick(2, clickPtLeft, 0)

	newOffsetLeft := fi.GetHScrollOrigin(0)
	if newOffsetLeft > tolerance {
		t.Errorf("B2 click at left edge should jump to ~0, got %d", newOffsetLeft)
	}

	// Click B2 at the right edge — should jump to ~maxScrollable.
	clickPtRight := image.Point{
		X: rect.Min.X + rect.Dx() - 1,
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}
	f.HScrollClick(2, clickPtRight, 0)

	newOffsetRight := fi.GetHScrollOrigin(0)
	if newOffsetRight < maxScrollable-tolerance {
		t.Errorf("B2 click at right edge should jump to ~%d, got %d", maxScrollable, newOffsetRight)
	}
}

// TestHScrollClickB3ScrollsRight verifies that clicking B3 on a horizontal
// scrollbar scrolls the block region to the right (increases the scroll offset).
// The amount scrolled is proportional to the click X position within the scrollbar.
func TestHScrollClickB3ScrollsRight(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)
	f.Redraw()

	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}

	// Start with offset at 0 (scrolled all the way left).
	fi.SetHScrollOrigin(0, 0)

	// Click B3 near the middle of the scrollbar (should scroll right).
	scrollbarHeight := 12
	clickPt := image.Point{
		X: rect.Min.X + rect.Dx()/2,
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}
	f.HScrollClick(3, clickPt, 0)

	newOffset := fi.GetHScrollOrigin(0)
	if newOffset <= 0 {
		t.Errorf("B3 click should scroll right (increase offset): got %d", newOffset)
	}

	maxScrollable := ar.MaxContentWidth - rect.Dx()
	if newOffset > maxScrollable {
		t.Errorf("B3 click should not exceed maxScrollable (%d): got %d", maxScrollable, newOffset)
	}
}

// TestHScrollClickClampsToRange verifies that HScrollClick clamps the resulting
// offset to [0, maxScrollable] regardless of button or click position.
func TestHScrollClickClampsToRange(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)
	f.Redraw()

	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}
	maxScrollable := ar.MaxContentWidth - rect.Dx()

	scrollbarHeight := 12

	// Test clamping to 0: B1 click when already at offset 0 should stay at 0.
	fi.SetHScrollOrigin(0, 0)
	clickPt := image.Point{
		X: rect.Min.X + rect.Dx()/2,
		Y: rect.Min.Y + ar.ScrollbarY + scrollbarHeight/2,
	}
	f.HScrollClick(1, clickPt, 0)
	offset := fi.GetHScrollOrigin(0)
	if offset < 0 {
		t.Errorf("B1 click at offset=0 should clamp to >= 0: got %d", offset)
	}

	// Test clamping to maxScrollable: B3 click when already at maxScrollable.
	fi.SetHScrollOrigin(0, maxScrollable)
	f.HScrollClick(3, clickPt, 0)
	offset = fi.GetHScrollOrigin(0)
	if offset > maxScrollable {
		t.Errorf("B3 click at maxScrollable should clamp to <= %d: got %d", maxScrollable, offset)
	}
	if offset < 0 {
		t.Errorf("offset should not be negative: got %d", offset)
	}
}

// TestHScrollWheelAdjustsOffset verifies that HScrollWheel adjusts the
// horizontal scroll offset by the given delta. A positive delta scrolls right
// (increases offset), and a negative delta scrolls left (decreases offset).
func TestHScrollWheelAdjustsOffset(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)
	f.Redraw()

	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}

	// Start at offset 0.
	fi.SetHScrollOrigin(0, 0)

	// Scroll right by 50 pixels.
	f.HScrollWheel(50, 0)
	offset := fi.GetHScrollOrigin(0)
	if offset != 50 {
		t.Errorf("HScrollWheel(50) should set offset to 50, got %d", offset)
	}

	// Scroll left by 20 pixels (delta = -20).
	f.HScrollWheel(-20, 0)
	offset = fi.GetHScrollOrigin(0)
	if offset != 30 {
		t.Errorf("HScrollWheel(-20) from 50 should set offset to 30, got %d", offset)
	}
}

// TestHScrollWheelClampsToRange verifies that HScrollWheel clamps the resulting
// offset to [0, maxScrollable] regardless of the delta value.
func TestHScrollWheelClampsToRange(t *testing.T) {
	rect := image.Rect(0, 0, 200, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create block code content that overflows frameWidth (200px).
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)
	f.Redraw()

	if len(fi.hscrollRegions) == 0 {
		t.Fatal("expected at least one hscroll region after Redraw")
	}
	ar := fi.hscrollRegions[0]
	if !ar.HasScrollbar {
		t.Fatal("expected region to have scrollbar (content overflows)")
	}
	maxScrollable := ar.MaxContentWidth - rect.Dx()

	// Test clamping to 0: large negative delta from offset 0.
	fi.SetHScrollOrigin(0, 0)
	f.HScrollWheel(-1000, 0)
	offset := fi.GetHScrollOrigin(0)
	if offset != 0 {
		t.Errorf("HScrollWheel(-1000) from 0 should clamp to 0, got %d", offset)
	}

	// Test clamping to maxScrollable: large positive delta from maxScrollable.
	fi.SetHScrollOrigin(0, maxScrollable)
	f.HScrollWheel(1000, 0)
	offset = fi.GetHScrollOrigin(0)
	if offset != maxScrollable {
		t.Errorf("HScrollWheel(1000) from maxScrollable should clamp to %d, got %d", maxScrollable, offset)
	}

	// Test clamping to 0 from a small offset with large negative delta.
	fi.SetHScrollOrigin(0, 10)
	f.HScrollWheel(-100, 0)
	offset = fi.GetHScrollOrigin(0)
	if offset != 0 {
		t.Errorf("HScrollWheel(-100) from 10 should clamp to 0, got %d", offset)
	}

	// Test clamping to maxScrollable from near-max offset with positive delta.
	fi.SetHScrollOrigin(0, maxScrollable-5)
	f.HScrollWheel(100, 0)
	offset = fi.GetHScrollOrigin(0)
	if offset != maxScrollable {
		t.Errorf("HScrollWheel(100) from %d should clamp to %d, got %d", maxScrollable-5, maxScrollable, offset)
	}
}

// TestScrollbarHeightInHitTesting verifies that Ptofchar and Charofpt account
// for scrollbar height when mapping between screen coordinates and rune
// positions. Without the fix, the cursor/selection would appear above the text
// by the combined height of scrollbars above.
func TestScrollbarHeightInHitTesting(t *testing.T) {
	rect := image.Rect(0, 0, 200, 600)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Content: overflowing code block followed by normal text.
	// The code block will get a horizontal scrollbar (12px), which should
	// push the "normal" text down.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	content := Content{
		// Code block line: 70 chars * 10px = 700px > 200px frame → scrollbar
		{Text: "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
		// Normal text after the code block
		{Text: "normal text here", Style: Style{Scale: 1.0}},
	}
	f.SetContent(content)
	f.Redraw()

	// The code block occupies 1 line (14px height) + scrollbar (12px) = 26px.
	// The newline after the code block is another line (14px). So "normal text"
	// should start at Y = 14 (code line) + 12 (scrollbar) + 14 (newline line) = 40.
	// Actually, the exact Y depends on layout details. The key assertion is that
	// Ptofchar returns the same Y as where the text would render.

	// Find rune position of "normal text here"
	normalRunePos := len([]rune("a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx")) + 1 // +1 for \n

	// Get screen point for the start of "normal text here"
	pt := f.Ptofchar(normalRunePos)
	ptY := pt.Y - rect.Min.Y // frame-relative Y

	// Now check the reverse: clicking at that screen Y should map back
	// to the same rune position (or at least the same line).
	clickPt := image.Point{X: rect.Min.X + 5, Y: pt.Y}
	gotRune := f.Charofpt(clickPt)

	// The mapped rune should be on the "normal text here" line.
	if gotRune < normalRunePos {
		t.Errorf("Charofpt at Ptofchar Y mapped to rune %d, want >= %d (normal text start)",
			gotRune, normalRunePos)
	}

	// Verify the Y is past the scrollbar. With the code block line (14px),
	// scrollbar (12px), and the newline line, the normal text should be at
	// Y >= 26. Without the fix it would be at Y = 14 (no scrollbar space).
	if ptY < 26 {
		t.Errorf("Ptofchar Y for normal text = %d, want >= 26 (code line 14 + scrollbar 12)",
			ptY)
	}
}

// TestScrollbarHeightAfterVerticalScroll verifies that when scrolled past a
// scrollbar-bearing code block, cursor positions are still correct. This
// specifically tests that originY computation accounts for scrollbar heights.
func TestScrollbarHeightAfterVerticalScroll(t *testing.T) {
	rect := image.Rect(0, 0, 200, 600)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Content: overflowing code block, then two normal text lines.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	codeText := "a_very_long_code_line_that_is_wider_than_two_hundred_pixels_xxxxxxxxxx"
	content := Content{
		{Text: codeText, Style: codeStyle},
		{Text: "\n", Style: codeStyle},
		{Text: "line two", Style: Style{Scale: 1.0}},
		{Text: "\n", Style: Style{Scale: 1.0}},
		{Text: "line three", Style: Style{Scale: 1.0}},
	}
	f.SetContent(content)
	f.Redraw()

	// Get position of "line two" without scroll
	lineTwoRune := len([]rune(codeText)) + 1 // +1 for \n
	ptNoScroll := f.Ptofchar(lineTwoRune)
	yNoScroll := ptNoScroll.Y - rect.Min.Y

	// Now scroll so that "line two" is the first visible line.
	// Set origin to the rune offset of "line two".
	f.SetOrigin(lineTwoRune)
	f.Redraw()

	// "line two" should now be at Y=0 in the viewport.
	ptScrolled := f.Ptofchar(lineTwoRune)
	yScrolled := ptScrolled.Y - rect.Min.Y

	if yScrolled != 0 {
		t.Errorf("after scrolling to line two, Ptofchar Y = %d, want 0", yScrolled)
	}

	// Verify that clicking at Y=0 maps to "line two" rune position.
	clickPt := image.Point{X: rect.Min.X + 5, Y: rect.Min.Y}
	gotRune := f.Charofpt(clickPt)
	if gotRune < lineTwoRune {
		t.Errorf("Charofpt at Y=0 after scroll mapped to rune %d, want >= %d",
			gotRune, lineTwoRune)
	}

	// Sanity: without scroll, Y should be larger (below the code block + scrollbar)
	if yNoScroll <= 14 { // Must be at least past the code line + scrollbar
		t.Errorf("without scroll, line two Y = %d, want > 14 (past code block + scrollbar)",
			yNoScroll)
	}
}

// TestHScrollRegionOffsetWithOrigin verifies that when the viewport is scrolled
// past some code blocks, the horizontal scroll positions of visible blocks are
// correctly mapped to the global hscrollOrigins slice. Without the offset fix,
// visible block 0 would incorrectly read the scroll position of the first
// (off-screen) block instead of its own.
func TestHScrollRegionOffsetWithOrigin(t *testing.T) {
	rect := image.Rect(0, 0, 200, 600)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	f := NewFrame()
	fi := f.(*frameImpl)
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Create content with 3 code blocks, each overflowing the 200px frame width.
	// Each block is separated by a normal text line.
	codeStyle := Style{Block: true, Code: true, Scale: 1.0, Bg: color.RGBA{R: 240, G: 240, B: 240, A: 255}}
	normalStyle := Style{Scale: 1.0}
	content := Content{
		// Block A
		{Text: "AAAA_long_code_line_overflow_xx", Style: codeStyle}, // 30 chars * 10px = 300px > 200px
		{Text: "\n", Style: codeStyle},
		// Separator
		{Text: "sep1", Style: normalStyle},
		{Text: "\n", Style: normalStyle},
		// Block B
		{Text: "BBBB_long_code_line_overflow_xx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
		// Separator
		{Text: "sep2", Style: normalStyle},
		{Text: "\n", Style: normalStyle},
		// Block C
		{Text: "CCCC_long_code_line_overflow_xx", Style: codeStyle},
		{Text: "\n", Style: codeStyle},
	}
	f.SetContent(content)

	// Step 1: Layout with origin=0 to initialize all 3 block regions.
	fi.origin = 0
	lines0, _ := fi.layoutFromOrigin()
	if len(lines0) == 0 {
		t.Fatal("layoutFromOrigin with origin=0 produced no lines")
	}

	// We should have 3 block regions.
	if fi.hscrollBlockCount != 3 {
		t.Fatalf("expected 3 block regions, got %d", fi.hscrollBlockCount)
	}

	// Set distinct scroll positions: A=100, B=200, C=300.
	// With origin=0, hscrollRegionOffset should be 0 so these go directly
	// into the global slots.
	if fi.hscrollRegionOffset != 0 {
		t.Fatalf("with origin=0, hscrollRegionOffset = %d, want 0", fi.hscrollRegionOffset)
	}
	fi.hscrollOrigins[0] = 100 // Block A (direct access to avoid offset)
	fi.hscrollOrigins[1] = 200 // Block B
	fi.hscrollOrigins[2] = 300 // Block C

	// Step 2: Find the rune offset past block A (after "AAAA...\nsep1\n").
	// Block A line: "AAAA_long_code_line_overflow_xx\n" = 31 runes
	// sep1 line: "sep1\n" = 5 runes
	// Total: 36 runes. Setting origin to 36 should start at block B.
	fi.origin = 36

	// Step 3: Layout from the new origin.
	lines, _ := fi.layoutFromOrigin()
	if len(lines) == 0 {
		t.Fatal("layoutFromOrigin with origin past block A produced no lines")
	}

	// hscrollRegionOffset should be 1 (block A is above the viewport).
	if fi.hscrollRegionOffset != 1 {
		t.Errorf("hscrollRegionOffset = %d, want 1", fi.hscrollRegionOffset)
	}

	// Step 4: Verify GetHScrollOrigin(0) returns block B's position (200),
	// not block A's position (100).
	got := fi.GetHScrollOrigin(0)
	if got != 200 {
		t.Errorf("GetHScrollOrigin(0) with origin past block A = %d, want 200 (block B)", got)
	}

	// GetHScrollOrigin(1) should return block C's position (300).
	got = fi.GetHScrollOrigin(1)
	if got != 300 {
		t.Errorf("GetHScrollOrigin(1) with origin past block A = %d, want 300 (block C)", got)
	}

	// Step 5: Verify SetHScrollOrigin(0, newVal) updates block B, not block A.
	fi.SetHScrollOrigin(0, 999)
	if fi.hscrollOrigins[1] != 999 {
		t.Errorf("SetHScrollOrigin(0, 999) should update global index 1 (block B), got hscrollOrigins[1] = %d", fi.hscrollOrigins[1])
	}
	if fi.hscrollOrigins[0] != 100 {
		t.Errorf("SetHScrollOrigin(0, 999) should not affect global index 0 (block A), got hscrollOrigins[0] = %d", fi.hscrollOrigins[0])
	}
}
