package rich

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

// TestSetOrigin tests that SetOrigin stores the origin offset correctly.
func TestSetOrigin(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set some content
	f.SetContent(Plain("hello world\nline two\nline three"))

	// Set an origin
	f.SetOrigin(10)

	// Verify via GetOrigin
	org := f.GetOrigin()
	if org != 10 {
		t.Errorf("SetOrigin(10) then GetOrigin() = %d, want 10", org)
	}
}

// TestGetOrigin tests that GetOrigin returns the current scroll origin.
func TestGetOrigin(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Before setting content or origin, should return 0
	org := f.GetOrigin()
	if org != 0 {
		t.Errorf("Initial GetOrigin() = %d, want 0", org)
	}

	// Set content and origin
	f.SetContent(Plain("hello world\nline two\nline three"))
	f.SetOrigin(5)

	org = f.GetOrigin()
	if org != 5 {
		t.Errorf("GetOrigin() after SetOrigin(5) = %d, want 5", org)
	}
}

// TestOriginZero tests that setting origin to 0 works correctly.
func TestOriginZero(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello"))

	// Set origin to non-zero
	f.SetOrigin(3)
	if f.GetOrigin() != 3 {
		t.Errorf("GetOrigin() = %d, want 3", f.GetOrigin())
	}

	// Set origin back to 0
	f.SetOrigin(0)
	if f.GetOrigin() != 0 {
		t.Errorf("GetOrigin() after SetOrigin(0) = %d, want 0", f.GetOrigin())
	}
}

// TestOriginClear tests that Clear resets origin to 0.
func TestOriginClear(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello world"))
	f.SetOrigin(8)

	// Verify origin was set
	org := f.GetOrigin()
	if org != 8 {
		t.Errorf("GetOrigin() = %d, want 8", org)
	}

	// Clear the frame
	f.Clear()

	// After Clear, origin should be reset
	org = f.GetOrigin()
	if org != 0 {
		t.Errorf("GetOrigin() after Clear() = %d, want 0", org)
	}
}

// TestOriginUpdateOverwrites tests that setting a new origin overwrites the previous one.
func TestOriginUpdateOverwrites(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello world\nline two\nline three"))

	// Set first origin
	f.SetOrigin(5)
	org := f.GetOrigin()
	if org != 5 {
		t.Errorf("First GetOrigin() = %d, want 5", org)
	}

	// Set second origin
	f.SetOrigin(12)
	org = f.GetOrigin()
	if org != 12 {
		t.Errorf("Second GetOrigin() = %d, want 12", org)
	}
}

// TestDisplayFromOrigin tests that Redraw starts displaying content from the origin offset.
// When origin is non-zero, text before the origin should not be displayed.
func TestDisplayFromOrigin(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content: "hello\nworld" (hello + newline + world)
	// Rune positions: h=0, e=1, l=2, l=3, o=4, \n=5, w=6, o=7, r=8, l=9, d=10
	f.SetContent(Plain("hello\nworld"))

	// Test 1: With origin at 0, both "hello" and "world" should be displayed
	f.SetOrigin(0)
	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
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
		t.Errorf("With origin=0, expected 'hello' to be drawn, ops: %v", ops)
	}
	if !foundWorld {
		t.Errorf("With origin=0, expected 'world' to be drawn, ops: %v", ops)
	}

	// Test 2: With origin at 6 (start of "world"), only "world" should be displayed
	f.SetOrigin(6)
	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops = display.(edwoodtest.GettableDrawOps).DrawOps()
	foundHello = false
	foundWorld = false
	for _, op := range ops {
		if strings.Contains(op, `string "hello"`) {
			foundHello = true
		}
		if strings.Contains(op, `string "world"`) {
			foundWorld = true
		}
	}

	if foundHello {
		t.Errorf("With origin=6, 'hello' should NOT be drawn, ops: %v", ops)
	}
	if !foundWorld {
		t.Errorf("With origin=6, expected 'world' to be drawn, ops: %v", ops)
	}

	// Test 3: With origin at 6, "world" should be drawn at the frame's top (Y=0, not Y=14)
	// Since we're starting from origin, the first visible content should be at rect.Min.Y
	worldAtTop := false
	expectedPos := fmt.Sprintf("(%d,%d)", rect.Min.X, rect.Min.Y)
	for _, op := range ops {
		if strings.Contains(op, `string "world"`) && strings.Contains(op, expectedPos) {
			worldAtTop = true
			break
		}
	}

	if !worldAtTop {
		t.Errorf("With origin=6, 'world' should be drawn at top of frame %s, ops: %v", expectedPos, ops)
	}
}

// TestMaxLines tests that MaxLines returns the maximum number of lines that can fit in the frame.
func TestMaxLines(t *testing.T) {
	// Frame is 300 pixels tall, font is 14 pixels high
	// So 300 / 14 = 21 lines can fit
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	maxLines := f.MaxLines()
	// 300 / 14 = 21.4, so 21 full lines fit
	if maxLines != 21 {
		t.Errorf("MaxLines() = %d, want 21 (frame height 300, font height 14)", maxLines)
	}
}

// TestMaxLinesSmallFrame tests MaxLines with a frame that can only fit a few lines.
func TestMaxLinesSmallFrame(t *testing.T) {
	// Frame is 42 pixels tall, font is 14 pixels high
	// So 42 / 14 = 3 lines can fit
	rect := image.Rect(0, 0, 400, 42)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	maxLines := f.MaxLines()
	if maxLines != 3 {
		t.Errorf("MaxLines() = %d, want 3 (frame height 42, font height 14)", maxLines)
	}
}

// TestMaxLinesNoFont tests MaxLines when no font is set.
func TestMaxLinesNoFont(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage))

	// Without a font, MaxLines should return 0
	maxLines := f.MaxLines()
	if maxLines != 0 {
		t.Errorf("MaxLines() without font = %d, want 0", maxLines)
	}
}

// TestVisibleLines tests that VisibleLines returns the number of lines currently displayed.
func TestVisibleLines(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with 3 lines: "line1\nline2\nline3"
	f.SetContent(Plain("line1\nline2\nline3"))

	visibleLines := f.VisibleLines()
	if visibleLines != 3 {
		t.Errorf("VisibleLines() = %d, want 3 for 3-line content", visibleLines)
	}
}

// TestVisibleLinesEmpty tests VisibleLines with no content.
func TestVisibleLinesEmpty(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// No content set
	visibleLines := f.VisibleLines()
	if visibleLines != 0 {
		t.Errorf("VisibleLines() with no content = %d, want 0", visibleLines)
	}
}

// TestVisibleLinesWithOrigin tests VisibleLines when origin is set.
func TestVisibleLinesWithOrigin(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with 3 lines: "line1\nline2\nline3"
	// Rune positions: l=0, i=1, n=2, e=3, 1=4, \n=5, l=6, ...
	f.SetContent(Plain("line1\nline2\nline3"))

	// With origin at 0, all 3 lines visible
	f.SetOrigin(0)
	if visibleLines := f.VisibleLines(); visibleLines != 3 {
		t.Errorf("VisibleLines() with origin=0 = %d, want 3", visibleLines)
	}

	// With origin at 6 (start of line2), 2 lines visible (line2 and line3)
	f.SetOrigin(6)
	if visibleLines := f.VisibleLines(); visibleLines != 2 {
		t.Errorf("VisibleLines() with origin=6 = %d, want 2", visibleLines)
	}
}

// TestVisibleLinesSingleLine tests VisibleLines with single-line content.
func TestVisibleLinesSingleLine(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Single line content
	f.SetContent(Plain("hello world"))

	visibleLines := f.VisibleLines()
	if visibleLines != 1 {
		t.Errorf("VisibleLines() = %d, want 1 for single-line content", visibleLines)
	}
}

// TestVisibleLinesWrapped tests VisibleLines when text wraps.
func TestVisibleLinesWrapped(t *testing.T) {
	// Frame is 50px wide, font is 10px per char
	// So 5 chars fit per line before wrapping
	rect := image.Rect(0, 0, 50, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "helloworld" = 10 chars, wraps to 2 lines (hello, world)
	f.SetContent(Plain("helloworld"))

	visibleLines := f.VisibleLines()
	if visibleLines != 2 {
		t.Errorf("VisibleLines() = %d, want 2 for wrapped content", visibleLines)
	}
}

// TestFullNotFull tests that Full returns false when content fits in the frame.
func TestFullNotFull(t *testing.T) {
	// Frame is 300 pixels tall, font is 14 pixels high
	// 21 lines can fit
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with only 3 lines - should not be full
	f.SetContent(Plain("line1\nline2\nline3"))

	if f.Full() {
		t.Errorf("Full() = true, want false for content with 3 lines (max 21)")
	}
}

// TestFullIsFull tests that Full returns true when content exceeds frame capacity.
func TestFullIsFull(t *testing.T) {
	// Frame is 42 pixels tall (3 lines at 14px per line)
	rect := image.Rect(0, 0, 400, 42)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with 5 lines - should be full (only 3 fit)
	f.SetContent(Plain("line1\nline2\nline3\nline4\nline5"))

	if !f.Full() {
		t.Errorf("Full() = false, want true for content with 5 lines (max 3)")
	}
}

// TestFullEmpty tests that Full returns false for empty content.
func TestFullEmpty(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// No content
	if f.Full() {
		t.Errorf("Full() = true, want false for empty frame")
	}
}

// TestFullExactFit tests Full when content exactly fills the frame.
func TestFullExactFit(t *testing.T) {
	// Frame is 42 pixels tall (exactly 3 lines at 14px per line)
	rect := image.Rect(0, 0, 400, 42)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with exactly 3 lines - should not be full (exact fit is not overfilled)
	f.SetContent(Plain("line1\nline2\nline3"))

	if f.Full() {
		t.Errorf("Full() = true, want false for content that exactly fills frame (3 lines)")
	}
}

// TestFullWithOrigin tests Full when origin is set (scrolled).
func TestFullWithOrigin(t *testing.T) {
	// Frame is 42 pixels tall (3 lines at 14px per line)
	rect := image.Rect(0, 0, 400, 42)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set content with 5 lines
	f.SetContent(Plain("line1\nline2\nline3\nline4\nline5"))

	// With origin at 0, should be full (5 lines, only 3 fit)
	f.SetOrigin(0)
	if !f.Full() {
		t.Errorf("Full() with origin=0 = false, want true for 5 lines (max 3)")
	}

	// With origin at start of line3 (12), only 3 lines visible (line3, line4, line5)
	// Still full because all 3 remaining lines are shown and exactly fill
	f.SetOrigin(12)
	if f.Full() {
		t.Errorf("Full() with origin=12 = true, want false for remaining 3 lines (max 3)")
	}

	// With origin at start of line4 (18), only 2 lines visible (line4, line5)
	// Not full because only 2 lines are visible
	f.SetOrigin(18)
	if f.Full() {
		t.Errorf("Full() with origin=18 = true, want false for remaining 2 lines (max 3)")
	}
}
