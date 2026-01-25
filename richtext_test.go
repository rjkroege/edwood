package main

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/rich"
)

// TestRichTextInit verifies that RichText can be initialized correctly.
func TestRichTextInit(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	if rt == nil {
		t.Fatal("NewRichText() returned nil")
	}

	rt.Init(rect, display, font)

	// Verify the full area is stored
	if got := rt.All(); got != rect {
		t.Errorf("All() = %v, want %v", got, rect)
	}

	// Verify the frame was created and initialized
	if rt.Frame() == nil {
		t.Error("Frame() returned nil after Init")
	}

	// Verify display is stored
	if rt.Display() != display {
		t.Error("Display() does not match the initialized display")
	}
}

// TestRichTextScrollRect verifies that the scrollbar rectangle is calculated correctly.
func TestRichTextScrollRect(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(rect, display, font)

	// Scrollbar should be on the left side of the full area
	scrollr := rt.ScrollRect()

	// Scrollbar should start at rect.Min.X
	if scrollr.Min.X != rect.Min.X {
		t.Errorf("ScrollRect().Min.X = %d, want %d", scrollr.Min.X, rect.Min.X)
	}

	// Scrollbar should have the same Y bounds as the full area
	if scrollr.Min.Y != rect.Min.Y || scrollr.Max.Y != rect.Max.Y {
		t.Errorf("ScrollRect() Y bounds = (%d, %d), want (%d, %d)",
			scrollr.Min.Y, scrollr.Max.Y, rect.Min.Y, rect.Max.Y)
	}

	// Scrollbar should have some width (implementation detail, just verify it's positive)
	if scrollr.Dx() <= 0 {
		t.Errorf("ScrollRect().Dx() = %d, want > 0", scrollr.Dx())
	}
}

// TestRichTextFrameRect verifies that the frame rectangle excludes the scrollbar.
func TestRichTextFrameRect(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(rect, display, font)

	scrollr := rt.ScrollRect()
	framer := rt.Frame().Rect()

	// Frame should start after the scrollbar (with some gap)
	if framer.Min.X <= scrollr.Max.X {
		t.Errorf("Frame().Rect().Min.X = %d, should be > %d (scrollbar end)",
			framer.Min.X, scrollr.Max.X)
	}

	// Frame should have the same Y bounds as the full area
	if framer.Min.Y != rect.Min.Y || framer.Max.Y != rect.Max.Y {
		t.Errorf("Frame().Rect() Y bounds = (%d, %d), want (%d, %d)",
			framer.Min.Y, framer.Max.Y, rect.Min.Y, rect.Max.Y)
	}
}

// TestRichTextSetContent verifies that content can be set and retrieved.
func TestRichTextSetContent(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(rect, display, font)

	// Set content
	content := rich.Plain("hello world")
	rt.SetContent(content)

	// Content should be stored
	got := rt.Content()
	if len(got) != len(content) {
		t.Errorf("Content() length = %d, want %d", len(got), len(content))
	}
}

// TestRichTextSelection verifies selection state management.
func TestRichTextSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(rect, display, font)

	rt.SetContent(rich.Plain("hello world"))

	// Initial selection should be empty (0, 0)
	q0, q1 := rt.Selection()
	if q0 != 0 || q1 != 0 {
		t.Errorf("Initial Selection() = (%d, %d), want (0, 0)", q0, q1)
	}

	// Set selection
	rt.SetSelection(2, 7)
	q0, q1 = rt.Selection()
	if q0 != 2 || q1 != 7 {
		t.Errorf("Selection() = (%d, %d), want (2, 7)", q0, q1)
	}
}

// TestRichTextOrigin verifies origin (scroll position) management.
func TestRichTextOrigin(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(rect, display, font)

	// Initial origin should be 0
	if got := rt.Origin(); got != 0 {
		t.Errorf("Initial Origin() = %d, want 0", got)
	}

	// Set origin
	rt.SetOrigin(100)
	if got := rt.Origin(); got != 100 {
		t.Errorf("Origin() = %d, want 100", got)
	}
}

// TestRichTextRedraw verifies that Redraw calls through to the frame.
func TestRichTextRedraw(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Create background and text color images
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

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	rt.SetContent(rich.Plain("hello"))

	// Clear any draw ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw
	rt.Redraw()

	// Verify that some draw operations occurred
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Redraw() did not produce any draw operations")
	}
}

// RichTextOption is a functional option for configuring RichText.
type RichTextOption func(*RichText)

// WithRichTextBackground sets the background image for the rich text component.
func WithRichTextBackground(bg draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.background = bg
	}
}

// WithRichTextColor sets the text color image for the rich text component.
func WithRichTextColor(c draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.textColor = c
	}
}
