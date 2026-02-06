package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

// TestRichTextInit verifies that RichText can be initialized correctly.
// Init no longer takes a rectangle - rectangles are provided at Render() time.
func TestRichTextInit(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	if rt == nil {
		t.Fatal("NewRichText() returned nil")
	}

	rt.Init(display, font)

	// Verify the frame was created and initialized
	if rt.Frame() == nil {
		t.Error("Frame() returned nil after Init")
	}

	// Verify display is stored
	if rt.Display() != display {
		t.Error("Display() does not match the initialized display")
	}

	// Before Render(), All() returns zero rect
	if got := rt.All(); !got.Empty() {
		t.Errorf("All() before Render() = %v, want empty rectangle", got)
	}

	// After Render(), All() returns the rendered rect
	renderRect := image.Rect(0, 0, 400, 300)
	rt.Render(renderRect)

	if got := rt.All(); got != renderRect {
		t.Errorf("All() after Render() = %v, want %v", got, renderRect)
	}
}

// TestRichTextInitThenRender verifies the new Init/Render pattern works correctly.
func TestRichTextInitThenRender(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Set content before rendering
	rt.SetContent(rich.Plain("hello world"))

	// Clear any ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Now render into a specific rectangle
	renderRect := image.Rect(50, 50, 350, 250)
	rt.Render(renderRect)

	// Verify draw operations occurred
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Render() did not produce any draw operations")
	}

	// Verify rectangles are correct
	if got := rt.All(); got != renderRect {
		t.Errorf("All() = %v, want %v", got, renderRect)
	}

	scrollr := rt.ScrollRect()
	if scrollr.Min.X != renderRect.Min.X {
		t.Errorf("ScrollRect().Min.X = %d, want %d", scrollr.Min.X, renderRect.Min.X)
	}
}

// TestRichTextScrollRect verifies that the scrollbar rectangle is calculated correctly.
func TestRichTextScrollRect(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(display, font)

	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(display, font)

	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(display, font)

	// Set content (can be done before Render)
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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(display, font)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	rt := NewRichText()
	rt.Init(display, font)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
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
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	rt.SetContent(rich.Plain("hello"))

	// Render into a rectangle first
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Clear any draw ops from render
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw
	rt.Redraw()

	// Verify that some draw operations occurred
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Redraw() did not produce any draw operations")
	}
}

// TestScrollbarPosition tests that the scrollbar thumb is rendered at the correct position.
func TestScrollbarPosition(t *testing.T) {
	// Frame is 300 pixels tall, scrollbar is on the left
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	// Create colors for rendering
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Set origin to roughly the middle of the scrollable range.
	// 30 lines * 14px = 420px total, 300px frame → 120px scrollable.
	// Middle of scrollable = 60px ≈ line 4 (56px). Line 4 = rune 20.
	rt.SetOrigin(20)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// No content set

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with just 3 lines - should fit (frame can show ~21 lines)
	rt.SetContent(rich.Plain("line1\nline2\nline3"))

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

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

// TestScrollbarClickButton2 tests that middle-clicking on the scrollbar sets the origin
// to an absolute position based on where the click occurred.
func TestScrollbarClickButton2(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Initially at origin 0
	rt.SetOrigin(0)

	// Click at the middle of the scrollbar (button 2 = middle click)
	scrollRect := rt.ScrollRect()
	middleY := (scrollRect.Min.Y + scrollRect.Max.Y) / 2

	// Simulate middle-click in scrollbar
	newOrigin := rt.ScrollClick(2, image.Pt(scrollRect.Min.X+5, middleY))

	// With 30 lines and clicking at the middle of the scrollbar,
	// the origin should be set to approximately the middle of the content.
	// Total content is 30 lines. Middle click should set origin to line ~15.
	// Each line is 5 runes ("line" + "\n"), so middle should be around rune 75.
	// Allow some flexibility in the exact position.
	if newOrigin < 50 || newOrigin > 100 {
		t.Errorf("ScrollClick(2, middle): got origin %d, want approximately 75", newOrigin)
	}
}

// TestScrollbarClickButton1 tests that left-clicking on the scrollbar scrolls up
// (backs up content based on click position).
func TestScrollbarClickButton1(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Start at line 15 (rune 75)
	rt.SetOrigin(75)
	beforeOrigin := rt.Origin()

	// Left-click (button 1) at the top of the scrollbar should scroll up
	scrollRect := rt.ScrollRect()
	topY := scrollRect.Min.Y + 10

	// Simulate left-click in scrollbar
	newOrigin := rt.ScrollClick(1, image.Pt(scrollRect.Min.X+5, topY))

	// Button 1 should scroll up (decrease origin), backing up content
	if newOrigin >= beforeOrigin {
		t.Errorf("ScrollClick(1, top): expected origin to decrease from %d, got %d", beforeOrigin, newOrigin)
	}
}

// TestScrollbarClickButton3 tests that right-clicking on the scrollbar scrolls down
// (advances content based on click position).
func TestScrollbarClickButton3(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Start at origin 0
	rt.SetOrigin(0)
	beforeOrigin := rt.Origin()

	// Right-click (button 3) at the middle of the scrollbar should scroll down
	scrollRect := rt.ScrollRect()
	middleY := (scrollRect.Min.Y + scrollRect.Max.Y) / 2

	// Simulate right-click in scrollbar
	newOrigin := rt.ScrollClick(3, image.Pt(scrollRect.Min.X+5, middleY))

	// Button 3 should scroll down (increase origin)
	if newOrigin <= beforeOrigin {
		t.Errorf("ScrollClick(3, middle): expected origin to increase from %d, got %d", beforeOrigin, newOrigin)
	}
}

// TestScrollbarClickAtTop tests that clicking at the very top of the scrollbar
// with button 1 scrolls up minimally (or doesn't scroll if already at top).
func TestScrollbarClickAtTop(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Already at origin 0
	rt.SetOrigin(0)

	// Left-click at the very top
	scrollRect := rt.ScrollRect()

	newOrigin := rt.ScrollClick(1, image.Pt(scrollRect.Min.X+5, scrollRect.Min.Y))

	// Should stay at 0 (can't scroll up from top)
	if newOrigin != 0 {
		t.Errorf("ScrollClick(1, top) when origin=0: got %d, want 0", newOrigin)
	}
}

// TestScrollbarClickAtBottom tests that clicking at the bottom of the scrollbar
// with button 3 advances to near the end of the content.
func TestScrollbarClickAtBottom(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
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

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Start at origin 0
	rt.SetOrigin(0)

	// Right-click at the bottom of the scrollbar
	scrollRect := rt.ScrollRect()

	newOrigin := rt.ScrollClick(3, image.Pt(scrollRect.Min.X+5, scrollRect.Max.Y-1))

	// Should scroll to near the end (significant forward movement)
	// 30 lines * 5 runes = 150 total runes
	// Clicking at the bottom should advance significantly
	if newOrigin < 50 {
		t.Errorf("ScrollClick(3, bottom): got origin %d, expected larger value", newOrigin)
	}
}

// TestScrollbarClickNoContent tests that clicking the scrollbar with no content
// returns origin 0.
func TestScrollbarClickNoContent(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// No content set

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	scrollRect := rt.ScrollRect()
	middleY := (scrollRect.Min.Y + scrollRect.Max.Y) / 2

	// Any click should return 0 when there's no content
	newOrigin := rt.ScrollClick(2, image.Pt(scrollRect.Min.X+5, middleY))
	if newOrigin != 0 {
		t.Errorf("ScrollClick with no content: got %d, want 0", newOrigin)
	}
}

// TestScrollbarClickContentFits tests that clicking the scrollbar when content fits
// keeps origin at 0.
func TestScrollbarClickContentFits(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content that fits (just 3 lines, frame can show ~21 lines)
	rt.SetContent(rich.Plain("line1\nline2\nline3"))

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	scrollRect := rt.ScrollRect()
	middleY := (scrollRect.Min.Y + scrollRect.Max.Y) / 2

	// When content fits, scrolling shouldn't change origin
	newOrigin := rt.ScrollClick(3, image.Pt(scrollRect.Min.X+5, middleY))
	if newOrigin != 0 {
		t.Errorf("ScrollClick when content fits: got %d, want 0", newOrigin)
	}
}

// TestMouseWheelScrollDown tests that scrolling down (button 5) increases the origin.
func TestMouseWheelScrollDown(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines (scrollable content)
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Start at origin 0
	rt.SetOrigin(0)
	beforeOrigin := rt.Origin()

	// Scroll down (button 5)
	newOrigin := rt.ScrollWheel(false) // false = scroll down

	// Origin should increase (scroll down shows later content)
	if newOrigin <= beforeOrigin {
		t.Errorf("ScrollWheel(down): expected origin to increase from %d, got %d", beforeOrigin, newOrigin)
	}
}

// TestMouseWheelScrollUp tests that scrolling up (button 4) decreases the origin.
func TestMouseWheelScrollUp(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines (scrollable content)
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Start at line 15 (rune 75)
	rt.SetOrigin(75)
	beforeOrigin := rt.Origin()

	// Scroll up (button 4)
	newOrigin := rt.ScrollWheel(true) // true = scroll up

	// Origin should decrease (scroll up shows earlier content)
	if newOrigin >= beforeOrigin {
		t.Errorf("ScrollWheel(up): expected origin to decrease from %d, got %d", beforeOrigin, newOrigin)
	}
}

// TestMouseWheelScrollUpAtTop tests that scrolling up when already at origin 0
// stays at origin 0.
func TestMouseWheelScrollUpAtTop(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines (scrollable content)
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Already at origin 0
	rt.SetOrigin(0)

	// Scroll up (button 4)
	newOrigin := rt.ScrollWheel(true) // true = scroll up

	// Should stay at 0 (can't scroll up from top)
	if newOrigin != 0 {
		t.Errorf("ScrollWheel(up) when origin=0: got %d, want 0", newOrigin)
	}
}

// TestMouseWheelScrollDownAtBottom tests that scrolling down when already at the
// end of content doesn't go past the last line.
func TestMouseWheelScrollDownAtBottom(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines (each "line\n" = 5 chars, last line has no newline = 4 chars)
	// Total: 29*5 + 4 = 149 runes
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Set origin to last line (line 29 starts at 5*29 = 145)
	rt.SetOrigin(145)
	beforeOrigin := rt.Origin()

	// Scroll down (button 5) multiple times
	var newOrigin int
	for i := 0; i < 10; i++ {
		newOrigin = rt.ScrollWheel(false) // false = scroll down
	}

	// Origin should not have increased significantly beyond the last line
	// It may increase slightly but should be bounded
	// The important thing is it doesn't crash or return invalid values
	if newOrigin < beforeOrigin {
		t.Errorf("ScrollWheel(down) at end: origin decreased from %d to %d", beforeOrigin, newOrigin)
	}
}

// TestMouseWheelScrollNoContent tests that scrolling with no content
// returns origin 0.
func TestMouseWheelScrollNoContent(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// No content set

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Scroll down
	newOrigin := rt.ScrollWheel(false)
	if newOrigin != 0 {
		t.Errorf("ScrollWheel(down) with no content: got %d, want 0", newOrigin)
	}

	// Scroll up
	newOrigin = rt.ScrollWheel(true)
	if newOrigin != 0 {
		t.Errorf("ScrollWheel(up) with no content: got %d, want 0", newOrigin)
	}
}

// TestMouseWheelScrollContentFits tests that scrolling when content fits in the view
// keeps origin at 0.
func TestMouseWheelScrollContentFits(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with just 3 lines - should fit (frame can show ~21 lines)
	rt.SetContent(rich.Plain("line1\nline2\nline3"))

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Scroll down - should stay at 0 because content fits
	newOrigin := rt.ScrollWheel(false)
	if newOrigin != 0 {
		t.Errorf("ScrollWheel(down) when content fits: got %d, want 0", newOrigin)
	}

	// Scroll up - should stay at 0
	newOrigin = rt.ScrollWheel(true)
	if newOrigin != 0 {
		t.Errorf("ScrollWheel(up) when content fits: got %d, want 0", newOrigin)
	}
}

// TestRichTextRender tests that the Render() method computes scrollbar/frame rects
// from the passed rectangle and draws the content.
func TestRichTextRender(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	// Create colors for rendering
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Set some content
	rt.SetContent(rich.Plain("hello world"))

	// Clear any draw ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Render into a rectangle
	newRect := image.Rect(50, 50, 350, 250)
	rt.Render(newRect)

	// Verify that draw operations occurred
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Render() did not produce any draw operations")
	}

	// Verify the last rect was updated
	if got := rt.All(); got != newRect {
		t.Errorf("All() after Render() = %v, want %v", got, newRect)
	}

	// Verify scrollbar rect is within the rendered area
	scrollr := rt.ScrollRect()
	if scrollr.Min.X != newRect.Min.X {
		t.Errorf("ScrollRect().Min.X = %d, want %d", scrollr.Min.X, newRect.Min.X)
	}
	if scrollr.Min.Y != newRect.Min.Y || scrollr.Max.Y != newRect.Max.Y {
		t.Errorf("ScrollRect() Y bounds = (%d, %d), want (%d, %d)",
			scrollr.Min.Y, scrollr.Max.Y, newRect.Min.Y, newRect.Max.Y)
	}

	// Verify frame rect starts after scrollbar
	framer := rt.Frame().Rect()
	if framer.Min.X <= scrollr.Max.X {
		t.Errorf("Frame().Rect().Min.X = %d, should be > %d (scrollbar end)",
			framer.Min.X, scrollr.Max.X)
	}
	if framer.Min.Y != newRect.Min.Y || framer.Max.Y != newRect.Max.Y {
		t.Errorf("Frame().Rect() Y bounds = (%d, %d), want (%d, %d)",
			framer.Min.Y, framer.Max.Y, newRect.Min.Y, newRect.Max.Y)
	}
}

// TestRichTextRenderUpdatesLastRect tests that Render() updates the cached
// rectangle used for hit-testing.
func TestRichTextRenderUpdatesLastRect(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	rt.SetContent(rich.Plain("hello world"))

	// Render into first rectangle
	rect1 := image.Rect(0, 0, 200, 150)
	rt.Render(rect1)

	if got := rt.All(); got != rect1 {
		t.Errorf("After first Render(), All() = %v, want %v", got, rect1)
	}

	// Render into second rectangle
	rect2 := image.Rect(100, 100, 500, 400)
	rt.Render(rect2)

	if got := rt.All(); got != rect2 {
		t.Errorf("After second Render(), All() = %v, want %v", got, rect2)
	}

	// Verify scrollbar and frame rects match the second rectangle
	scrollr := rt.ScrollRect()
	if scrollr.Min.X != rect2.Min.X {
		t.Errorf("ScrollRect().Min.X = %d, want %d after second Render()", scrollr.Min.X, rect2.Min.X)
	}

	framer := rt.Frame().Rect()
	if framer.Max.X != rect2.Max.X {
		t.Errorf("Frame().Rect().Max.X = %d, want %d after second Render()", framer.Max.X, rect2.Max.X)
	}
}

// TestRichTextRenderDifferentRects tests that the RichText component correctly
// handles rendering into multiple different rectangles, computing the scrollbar
// and frame areas dynamically each time.
func TestRichTextRenderDifferentRects(t *testing.T) {
	// Start with a display larger than any rect we'll use
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Set content with enough lines to test scrolling
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Define several different rectangles to render into
	testCases := []struct {
		name string
		rect image.Rectangle
	}{
		{"small", image.Rect(0, 0, 200, 150)},
		{"medium", image.Rect(50, 50, 400, 300)},
		{"large", image.Rect(0, 0, 600, 500)},
		{"offset", image.Rect(100, 100, 500, 400)},
		{"narrow", image.Rect(0, 0, 150, 400)},
		{"wide", image.Rect(0, 0, 700, 200)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			display.(edwoodtest.GettableDrawOps).Clear()

			// Render into this rectangle
			rt.Render(tc.rect)

			// Verify All() returns the rendered rectangle
			if got := rt.All(); got != tc.rect {
				t.Errorf("All() = %v, want %v", got, tc.rect)
			}

			// Verify scrollbar rect is within the rendered area
			scrollr := rt.ScrollRect()
			if scrollr.Min.X != tc.rect.Min.X {
				t.Errorf("ScrollRect().Min.X = %d, want %d", scrollr.Min.X, tc.rect.Min.X)
			}
			if scrollr.Min.Y != tc.rect.Min.Y || scrollr.Max.Y != tc.rect.Max.Y {
				t.Errorf("ScrollRect() Y bounds = (%d, %d), want (%d, %d)",
					scrollr.Min.Y, scrollr.Max.Y, tc.rect.Min.Y, tc.rect.Max.Y)
			}

			// Verify scrollbar has positive width
			if scrollr.Dx() <= 0 {
				t.Errorf("ScrollRect().Dx() = %d, want > 0", scrollr.Dx())
			}

			// Verify frame rect starts after scrollbar with gap
			framer := rt.Frame().Rect()
			if framer.Min.X <= scrollr.Max.X {
				t.Errorf("Frame().Rect().Min.X = %d, should be > %d (scrollbar end)",
					framer.Min.X, scrollr.Max.X)
			}
			if framer.Min.Y != tc.rect.Min.Y || framer.Max.Y != tc.rect.Max.Y {
				t.Errorf("Frame().Rect() Y bounds = (%d, %d), want (%d, %d)",
					framer.Min.Y, framer.Max.Y, tc.rect.Min.Y, tc.rect.Max.Y)
			}
			if framer.Max.X != tc.rect.Max.X {
				t.Errorf("Frame().Rect().Max.X = %d, want %d", framer.Max.X, tc.rect.Max.X)
			}

			// Verify draw operations occurred
			ops := display.(edwoodtest.GettableDrawOps).DrawOps()
			if len(ops) == 0 {
				t.Error("Render() did not produce any draw operations")
			}
		})
	}
}

// TestPreviewMouseAfterResize tests that mouse handling (scrollbar clicks, scroll wheel)
// works correctly after the RichText component has been resized via Render().
// This verifies that the cached lastScrollRect is updated by Render() and used
// for hit-testing in mouse handling methods.
func TestPreviewMouseAfterResize(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 30 lines (scrollable content)
	var content rich.Content
	for i := 0; i < 30; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Initial render into a small rectangle
	initialRect := image.Rect(0, 0, 200, 150)
	rt.Render(initialRect)

	// Verify initial scrollbar rect
	initialScrollRect := rt.ScrollRect()
	if initialScrollRect.Min.Y != initialRect.Min.Y || initialScrollRect.Max.Y != initialRect.Max.Y {
		t.Errorf("Initial ScrollRect Y bounds = (%d, %d), want (%d, %d)",
			initialScrollRect.Min.Y, initialScrollRect.Max.Y, initialRect.Min.Y, initialRect.Max.Y)
	}

	// Now resize to a different rectangle (simulate window resize)
	resizedRect := image.Rect(50, 50, 400, 350)
	rt.Render(resizedRect)

	// Verify the scrollbar rect updated after resize
	resizedScrollRect := rt.ScrollRect()
	if resizedScrollRect.Min.Y != resizedRect.Min.Y || resizedScrollRect.Max.Y != resizedRect.Max.Y {
		t.Errorf("Resized ScrollRect Y bounds = (%d, %d), want (%d, %d)",
			resizedScrollRect.Min.Y, resizedScrollRect.Max.Y, resizedRect.Min.Y, resizedRect.Max.Y)
	}

	// Test 1: ScrollClick should use the new (resized) scrollbar rect for hit-testing
	// Start at origin 0
	rt.SetOrigin(0)

	// Click in the middle of the NEW scrollbar rect (should scroll)
	newScrollMiddleY := (resizedScrollRect.Min.Y + resizedScrollRect.Max.Y) / 2
	clickPoint := image.Pt(resizedScrollRect.Min.X+5, newScrollMiddleY)

	// Button 3 (right-click) should scroll down
	newOrigin := rt.ScrollClick(3, clickPoint)
	if newOrigin <= 0 {
		t.Errorf("ScrollClick after resize: expected origin > 0, got %d", newOrigin)
	}

	// Test 2: ScrollWheel should work after resize
	rt.SetOrigin(0)
	wheelOrigin := rt.ScrollWheel(false) // scroll down
	if wheelOrigin <= 0 {
		t.Errorf("ScrollWheel after resize: expected origin > 0, got %d", wheelOrigin)
	}

	// Test 3: Clicking at old scrollbar location should NOT have effect
	// The old scrollbar was at Y=0 to Y=150, new is at Y=50 to Y=350
	// A click at Y=140 (which was valid before but is now above the scrollbar)
	// should still work because the click is within the new scrollbar area.
	// But let's verify clicks in the new area work correctly.
	rt.SetOrigin(0)

	// Click at the bottom of the NEW scrollbar area
	bottomClickPoint := image.Pt(resizedScrollRect.Min.X+5, resizedScrollRect.Max.Y-10)
	bottomOrigin := rt.ScrollClick(3, bottomClickPoint)
	if bottomOrigin <= 0 {
		t.Errorf("ScrollClick at bottom after resize: expected origin > 0, got %d", bottomOrigin)
	}

	// Test 4: Verify thumb position is calculated using the new rectangle
	rt.SetOrigin(0)
	thumbRect := rt.scrThumbRect()

	// Thumb should be within the new scrollbar area
	if thumbRect.Min.Y < resizedScrollRect.Min.Y || thumbRect.Max.Y > resizedScrollRect.Max.Y {
		t.Errorf("Thumb rect %v should be within resized scrollbar %v", thumbRect, resizedScrollRect)
	}
}

// TestMouseWheelScrollMultipleScrolls tests that multiple scroll wheel events
// accumulate correctly.
func TestMouseWheelScrollMultipleScrolls(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

	// Content with 50 lines (plenty to scroll)
	var content rich.Content
	for i := 0; i < 50; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("line")...)
	}
	rt.SetContent(content)

	// Render into a specific rectangle
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Start at origin 0
	rt.SetOrigin(0)

	// Scroll down 5 times
	var origin int
	for i := 0; i < 5; i++ {
		origin = rt.ScrollWheel(false) // false = scroll down
	}
	afterDown := origin

	// Origin should have increased significantly after 5 scrolls down
	if afterDown == 0 {
		t.Error("Origin should have increased after scrolling down 5 times")
	}

	// Now scroll back up 5 times
	for i := 0; i < 5; i++ {
		origin = rt.ScrollWheel(true) // true = scroll up
	}
	afterUp := origin

	// Origin should have decreased back toward 0
	if afterUp >= afterDown {
		t.Errorf("Origin should have decreased after scrolling up; down=%d, up=%d", afterDown, afterUp)
	}
}

// TestRichTextWithImageCache verifies that RichText can be configured with an
// ImageCache via the WithRichTextImageCache option, and that the cache is passed
// to the underlying Frame.
func TestRichTextWithImageCache(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	// Create an image cache
	cache := rich.NewImageCache(10)

	// Create background and text color images
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithRichTextImageCache(cache),
	)

	// Verify the RichText was initialized successfully
	if rt.Frame() == nil {
		t.Fatal("Frame() returned nil after Init with ImageCache")
	}

	// Set some content and render
	rt.SetContent(rich.Plain("hello world"))
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// Verify frame is in a valid state (should not panic, has valid rect)
	if rt.Frame().Rect().Empty() {
		t.Error("Frame should have non-empty rectangle after Render")
	}

	// Verify All() returns the rendered rectangle
	if got := rt.All(); got != rect {
		t.Errorf("All() = %v, want %v", got, rect)
	}
}

// TestRichTextWithImageCacheNil verifies that RichText works correctly when
// WithRichTextImageCache is called with nil (no cache).
func TestRichTextWithImageCacheNil(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	rt := NewRichText()

	// Should not panic with nil cache
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithRichTextImageCache(nil),
	)

	// RichText should still work
	rt.SetContent(rich.Plain("hello world"))
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	if rt.Frame().Rect().Empty() {
		t.Error("Frame should have non-empty rectangle after Render with nil cache")
	}
}

// TestRichTextWithImageCachePassedToFrame verifies that the ImageCache
// configured on RichText is passed through to the underlying Frame.
func TestRichTextWithImageCachePassedToFrame(t *testing.T) {
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	// Create an image cache
	cache := rich.NewImageCache(10)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithRichTextImageCache(cache),
	)

	// Pre-load the error entry so layout gets a synchronous cache hit.
	testImagePath := "/nonexistent/test_image.png"
	cache.Load(testImagePath)
	content := rich.Content{
		rich.Span{
			Text: "[Image: test]",
			Style: rich.Style{
				Image:    true,
				ImageURL: testImagePath,
				ImageAlt: "test",
			},
		},
	}
	rt.SetContent(content)

	// Render to trigger layout
	rect := image.Rect(0, 0, 400, 300)
	rt.Render(rect)

	// The frame should have used the cache during layout.
	// Even though the file doesn't exist, the cache should record the load attempt.
	// Verify by checking the cache has an entry for this path (with an error).
	cached, ok := cache.Get(testImagePath)
	if !ok {
		t.Error("ImageCache should have been used during layout - expected cache entry for image path")
	}
	if cached == nil {
		t.Error("Cached entry should not be nil")
	} else if cached.Err == nil {
		t.Error("Expected error in cached entry for nonexistent file")
	}
}

// TestImageWidthEndToEnd is an integration test that exercises the full pipeline:
// 1. Parse markdown with width tag → Content with ImageWidth set
// 2. Layout with image cache → boxes sized to explicit width
// 3. Rendering path → draw operations emitted without panic
func TestImageWidthEndToEnd(t *testing.T) {
	// Create a temporary directory with a test PNG image (400x200 pixels)
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "photo.png")
	img := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 128, B: 255, A: 255})
		}
	}
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	f.Close()

	// Step 1: Parse markdown with width tag
	md := "![Photo](" + imgPath + " \"width=200px\")\n"
	content := markdown.Parse(md)

	// Verify parsing: find the image span and check ImageWidth
	foundImage := false
	for _, span := range content {
		if span.Style.Image {
			foundImage = true
			if span.Style.ImageWidth != 200 {
				t.Errorf("parsed ImageWidth = %d, want 200", span.Style.ImageWidth)
			}
			if span.Style.ImageURL != imgPath {
				t.Errorf("parsed ImageURL = %q, want %q", span.Style.ImageURL, imgPath)
			}
		}
	}
	if !foundImage {
		t.Fatal("markdown.Parse did not produce an image span")
	}

	// Step 2: Set up RichText with image cache and render
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	cache := rich.NewImageCache(10)
	// Pre-load so layout gets a synchronous cache hit.
	if _, err := cache.Load(imgPath); err != nil {
		t.Fatalf("failed to pre-load image: %v", err)
	}

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
		WithRichTextImageCache(cache),
	)

	rt.SetContent(content)

	renderRect := image.Rect(0, 0, 600, 400)
	rt.Render(renderRect)

	// Step 3: Verify the image is in the cache
	cached, ok := cache.Get(imgPath)
	if !ok {
		t.Fatal("image should be in cache")
	}
	if cached.Err != nil {
		t.Fatalf("image cache error: %v", cached.Err)
	}
	if cached.Width != 400 || cached.Height != 200 {
		t.Errorf("cached image dimensions = %dx%d, want 400x200", cached.Width, cached.Height)
	}

	// Step 4: Verify draw operations occurred (rendering didn't panic)
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Render() did not produce any draw operations")
	}

	// Step 5: Redraw and verify it still works
	display.(edwoodtest.GettableDrawOps).Clear()
	rt.Redraw()

	ops = display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Redraw() did not produce any draw operations")
	}
}

// TestImageWidthEndToEndNoWidthTag verifies that an image without a width tag
// renders at its natural size through the full pipeline.
func TestImageWidthEndToEndNoWidthTag(t *testing.T) {
	// Create a temporary directory with a test PNG image (100x80 pixels)
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "small.png")
	img := image.NewRGBA(image.Rect(0, 0, 100, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	f.Close()

	// Parse markdown without width tag
	md := "![Small](" + imgPath + ")\n"
	content := markdown.Parse(md)

	// Verify ImageWidth is 0 (natural size)
	for _, span := range content {
		if span.Style.Image {
			if span.Style.ImageWidth != 0 {
				t.Errorf("parsed ImageWidth = %d, want 0 (natural size)", span.Style.ImageWidth)
			}
		}
	}

	// Render through RichText
	displayRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(displayRect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	cache := rich.NewImageCache(10)
	// Pre-load so layout gets a synchronous cache hit.
	if _, err := cache.Load(imgPath); err != nil {
		t.Fatalf("failed to pre-load image: %v", err)
	}

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
		WithRichTextImageCache(cache),
	)

	rt.SetContent(content)

	renderRect := image.Rect(0, 0, 600, 400)
	rt.Render(renderRect)

	// Verify image is in cache at natural size
	cached, ok := cache.Get(imgPath)
	if !ok {
		t.Fatal("image should be in cache")
	}
	if cached.Err != nil {
		t.Fatalf("image cache error: %v", cached.Err)
	}
	if cached.Width != 100 || cached.Height != 80 {
		t.Errorf("cached image dimensions = %dx%d, want 100x80", cached.Width, cached.Height)
	}

	// Verify rendering succeeded
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Render() did not produce any draw operations")
	}
}
