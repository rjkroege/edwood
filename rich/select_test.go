package rich

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

// TestSetSelection tests that SetSelection stores p0 and p1 correctly.
func TestSetSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Set some content
	f.SetContent(Plain("hello world"))

	// Set a selection
	f.SetSelection(2, 7)

	// Verify via GetSelection
	p0, p1 := f.GetSelection()
	if p0 != 2 || p1 != 7 {
		t.Errorf("SetSelection(2, 7) then GetSelection() = (%d, %d), want (2, 7)", p0, p1)
	}
}

// TestGetSelection tests that GetSelection returns the current selection bounds.
func TestGetSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// Before setting content or selection, should return (0, 0)
	p0, p1 := f.GetSelection()
	if p0 != 0 || p1 != 0 {
		t.Errorf("Initial GetSelection() = (%d, %d), want (0, 0)", p0, p1)
	}

	// Set content and selection
	f.SetContent(Plain("hello world"))
	f.SetSelection(0, 5)

	p0, p1 = f.GetSelection()
	if p0 != 0 || p1 != 5 {
		t.Errorf("GetSelection() after SetSelection(0, 5) = (%d, %d), want (0, 5)", p0, p1)
	}
}

// TestSelectionEmptyRange tests selection with p0 == p1 (cursor position, no selection).
func TestSelectionEmptyRange(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello"))

	// Set cursor position (no selection)
	f.SetSelection(3, 3)

	p0, p1 := f.GetSelection()
	if p0 != 3 || p1 != 3 {
		t.Errorf("SetSelection(3, 3) then GetSelection() = (%d, %d), want (3, 3)", p0, p1)
	}
}

// TestSelectionClear tests that Clear resets selection to (0, 0).
func TestSelectionClear(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello world"))
	f.SetSelection(2, 8)

	// Verify selection was set
	p0, p1 := f.GetSelection()
	if p0 != 2 || p1 != 8 {
		t.Errorf("GetSelection() = (%d, %d), want (2, 8)", p0, p1)
	}

	// Clear the frame
	f.Clear()

	// After Clear, selection should be reset
	p0, p1 = f.GetSelection()
	if p0 != 0 || p1 != 0 {
		t.Errorf("GetSelection() after Clear() = (%d, %d), want (0, 0)", p0, p1)
	}
}

// TestSelectionFullContent tests selecting the entire content.
func TestSelectionFullContent(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	// "hello" = 5 characters
	f.SetContent(Plain("hello"))
	f.SetSelection(0, 5)

	p0, p1 := f.GetSelection()
	if p0 != 0 || p1 != 5 {
		t.Errorf("Full selection GetSelection() = (%d, %d), want (0, 5)", p0, p1)
	}
}

// TestSelectionUpdateOverwrites tests that setting a new selection overwrites the previous one.
func TestSelectionUpdateOverwrites(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage), WithFont(font), WithTextColor(textImage))

	f.SetContent(Plain("hello world"))

	// Set first selection
	f.SetSelection(0, 5)
	p0, p1 := f.GetSelection()
	if p0 != 0 || p1 != 5 {
		t.Errorf("First GetSelection() = (%d, %d), want (0, 5)", p0, p1)
	}

	// Set second selection
	f.SetSelection(6, 11)
	p0, p1 = f.GetSelection()
	if p0 != 6 || p1 != 11 {
		t.Errorf("Second GetSelection() = (%d, %d), want (6, 11)", p0, p1)
	}
}
