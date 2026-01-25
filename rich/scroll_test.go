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
