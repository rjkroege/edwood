package rich

import (
	"image"
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
