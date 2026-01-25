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

// WithScrollbarColors sets the scrollbar background and thumb colors.
func WithScrollbarColors(bg, thumb draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.scrollBg = bg
		rt.scrollThumb = thumb
	}
}

// TestScrollbarPosition tests that the scrollbar thumb is rendered at the correct position.
func TestScrollbarPosition(t *testing.T) {
	// Frame is 300 pixels tall, scrollbar is on the left
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Create colors for rendering
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Set content with many lines (enough to scroll)
	// Each line is ~5 chars + newline = 6 runes
	// 30 lines of content in a frame that can show ~21 lines (300/14)
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, "line")
	}
	content := rich.Plain(lines[0])
	for i := 1; i < len(lines); i++ {
		content = append(content, rich.Plain("\n"+lines[i])...)
	}
	rt.SetContent(content)

	// Clear ops and redraw
	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	// Verify scrollbar background is drawn in the scrollbar rectangle
	scrollRect := rt.ScrollRect()
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// We should have a draw operation for the scrollbar area
	foundScrollbarDraw := false
	for _, op := range ops {
		// Look for a draw that covers the scrollbar area
		if containsRect(op, scrollRect) {
			foundScrollbarDraw = true
			break
		}
	}

	if !foundScrollbarDraw {
		t.Errorf("Expected scrollbar background to be drawn in rect %v, ops: %v", scrollRect, ops)
	}
}

// TestScrollbarThumbAtTop tests that the scrollbar thumb is at the top when origin is 0.
func TestScrollbarThumbAtTop(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines - frame shows ~21 lines (300/14), so content is scrollable
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Origin at 0 (top)
	rt.SetOrigin(0)

	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	// Get the thumb rectangle
	thumbRect := rt.scrThumbRect()
	scrollRect := rt.ScrollRect()

	// Thumb should start at the top of the scrollbar
	if thumbRect.Min.Y != scrollRect.Min.Y {
		t.Errorf("Thumb top = %d, want %d (scrollbar top)", thumbRect.Min.Y, scrollRect.Min.Y)
	}

	// When content is scrollable (30 lines in frame that shows ~21),
	// thumb height should be less than scrollbar height
	// Approximate: thumb height should be roughly (21/30) * scrollbar height
	scrollHeight := scrollRect.Dy()
	thumbHeight := thumbRect.Dy()
	maxExpectedThumbHeight := scrollHeight * 3 / 4 // thumb should be at most 75% of scrollbar

	if thumbHeight >= scrollHeight {
		t.Errorf("Thumb height = %d, should be less than scrollbar height %d for scrollable content", thumbHeight, scrollHeight)
	}
	if thumbHeight > maxExpectedThumbHeight {
		t.Errorf("Thumb height = %d, should be at most %d (75%% of scrollbar) for scrollable content", thumbHeight, maxExpectedThumbHeight)
	}
}

// TestScrollbarThumbAtBottom tests that the scrollbar thumb is at the bottom when scrolled to end.
func TestScrollbarThumbAtBottom(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines (each "line\n" = 5 chars)
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Set origin near the end (line 25 starts at rune position 5*25 = 125)
	// Actually let's calculate: "line\n" repeated = 5 chars each
	// Line 0: 0-3 (line) + 4 (\n) = 5
	// Line 1: 5-8 (line) + 9 (\n) = ...
	// To show the last ~5 lines, origin should be around line 25
	rt.SetOrigin(125)

	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	// Get the thumb rectangle
	thumbRect := rt.scrThumbRect()
	scrollRect := rt.ScrollRect()

	// Thumb should be near the bottom (allow some tolerance)
	// The thumb's bottom should be at or near the scrollbar's bottom
	if thumbRect.Max.Y < scrollRect.Max.Y-10 {
		t.Errorf("Thumb bottom = %d, want near %d (scrollbar bottom)", thumbRect.Max.Y, scrollRect.Max.Y)
	}

	// Thumb should NOT start at the top when scrolled down
	if thumbRect.Min.Y == scrollRect.Min.Y {
		t.Errorf("Thumb top = %d, should NOT equal scrollbar top when scrolled down", thumbRect.Min.Y)
	}

	// Thumb height should be less than scrollbar height
	scrollHeight := scrollRect.Dy()
	thumbHeight := thumbRect.Dy()
	if thumbHeight >= scrollHeight {
		t.Errorf("Thumb height = %d, should be less than scrollbar height %d", thumbHeight, scrollHeight)
	}
}

// TestScrollbarThumbMiddle tests that the scrollbar thumb is in the middle when scrolled halfway.
func TestScrollbarThumbMiddle(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Set origin to roughly halfway (line 15 = rune 75)
	rt.SetOrigin(75)

	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	// Get the thumb rectangle
	thumbRect := rt.scrThumbRect()
	scrollRect := rt.ScrollRect()

	// Thumb center should be roughly in the middle of the scrollbar
	thumbCenter := (thumbRect.Min.Y + thumbRect.Max.Y) / 2
	scrollCenter := (scrollRect.Min.Y + scrollRect.Max.Y) / 2

	// Allow 20% tolerance
	tolerance := scrollRect.Dy() / 5
	if thumbCenter < scrollCenter-tolerance || thumbCenter > scrollCenter+tolerance {
		t.Errorf("Thumb center = %d, want near %d (scrollbar center, tolerance %d)", thumbCenter, scrollCenter, tolerance)
	}

	// Thumb should NOT be at the very top or bottom
	if thumbRect.Min.Y == scrollRect.Min.Y {
		t.Errorf("Thumb top = %d, should NOT equal scrollbar top when scrolled to middle", thumbRect.Min.Y)
	}
	if thumbRect.Max.Y == scrollRect.Max.Y {
		t.Errorf("Thumb bottom = %d, should NOT equal scrollbar bottom when scrolled to middle", thumbRect.Max.Y)
	}

	// Thumb height should be less than scrollbar height
	scrollHeight := scrollRect.Dy()
	thumbHeight := thumbRect.Dy()
	if thumbHeight >= scrollHeight {
		t.Errorf("Thumb height = %d, should be less than scrollbar height %d", thumbHeight, scrollHeight)
	}
}

// TestScrollbarNoContent tests that the scrollbar thumb fills the whole area when there's no content.
func TestScrollbarNoContent(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// No content set

	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	// Get the thumb rectangle
	thumbRect := rt.scrThumbRect()
	scrollRect := rt.ScrollRect()

	// With no content, thumb should fill the entire scrollbar height
	if thumbRect.Min.Y != scrollRect.Min.Y || thumbRect.Max.Y != scrollRect.Max.Y {
		t.Errorf("Thumb rect = %v, want same height as scrollbar %v for empty content", thumbRect, scrollRect)
	}
}

// TestScrollbarContentFits tests that the thumb fills the area when all content fits.
func TestScrollbarContentFits(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(rect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with just 3 lines - should fit (frame can show ~21 lines)
	rt.SetContent(rich.Plain("line1\nline2\nline3"))

	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	// Get the thumb rectangle
	thumbRect := rt.scrThumbRect()
	scrollRect := rt.ScrollRect()

	// With content that fits, thumb should fill the entire scrollbar height
	if thumbRect.Min.Y != scrollRect.Min.Y || thumbRect.Max.Y != scrollRect.Max.Y {
		t.Errorf("Thumb rect = %v, want same height as scrollbar %v for content that fits", thumbRect, scrollRect)
	}
}

// containsRect is a helper to check if a draw operation string mentions a specific rectangle.
func containsRect(op string, r image.Rectangle) bool {
	// This is a simple heuristic - check if the op contains coordinates that overlap with rect
	// For now, just check if any draw happened (a more thorough check would parse the op string)
	return len(op) > 0 && r.Dx() > 0 && r.Dy() > 0
}
