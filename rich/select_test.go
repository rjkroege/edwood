package rich

import (
	"fmt"
	"image"
	"strings"
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

// TestDrawSelectionHighlightsRegion tests that Redraw draws a highlight for the selection.
func TestDrawSelectionHighlightsRegion(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// Set content: "hello world" = 11 chars
	f.SetContent(Plain("hello world"))

	// Select "ello" (positions 1-5)
	f.SetSelection(1, 5)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Look for a fill operation at the selection rectangle
	// The selection should span from X=10 (position 1) to X=50 (position 5)
	// at Y=0 with height=14
	expectedRect := image.Rect(10, 0, 50, 14)
	foundSelection := false
	for _, op := range ops {
		if strings.Contains(op, "fill") && strings.Contains(op, expectedRect.String()) {
			foundSelection = true
			break
		}
	}

	if !foundSelection {
		t.Errorf("Redraw() did not draw selection highlight at %v\ngot ops: %v", expectedRect, ops)
	}
}

// TestDrawSelectionNoHighlightWhenEmpty tests that no highlight is drawn when p0 == p1.
func TestDrawSelectionNoHighlightWhenEmpty(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	f.SetContent(Plain("hello"))

	// Set cursor position (no selection, p0 == p1)
	f.SetSelection(2, 2)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Check that no selection-color fill is drawn when p0 == p1
	// With scratch-based clipping, there will be multiple fills (background + blit),
	// but none should use the selection color.
	selectionDrawn := false
	for _, op := range ops {
		if strings.Contains(op, "selection-color") {
			selectionDrawn = true
			break
		}
	}

	if selectionDrawn {
		t.Errorf("Redraw() should not draw selection when p0 == p1\nops: %v", ops)
	}
}

// TestDrawSelectionMultiLine tests that selection spanning multiple lines is drawn correctly.
func TestDrawSelectionMultiLine(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello\nworld" = "hello" (5) + newline (1) + "world" (5)
	f.SetContent(Plain("hello\nworld"))

	// Select from "llo" on first line through "wor" on second line (positions 2-9)
	f.SetSelection(2, 9)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Count fill operations excluding background
	// Background is the full rect (0,0)-(400,300)
	bgRect := rect.String()
	selectionFills := 0
	for _, op := range ops {
		if strings.Contains(op, "fill") && !strings.Contains(op, bgRect) {
			selectionFills++
		}
	}

	// Multi-line selection should produce multiple fill rectangles (at least 2)
	if selectionFills < 2 {
		t.Errorf("Redraw() should draw multiple selection rectangles for multi-line selection, got %d fills\nops: %v", selectionFills, ops)
	}
}

// TestDrawSelectionWrappedLine tests selection on wrapped lines.
func TestDrawSelectionWrappedLine(t *testing.T) {
	// Frame is 50px wide, font is 10px per char
	// So 5 chars fit per line before wrapping
	rect := image.Rect(0, 0, 50, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "helloworld" wraps: "hello" on line 1, "world" on line 2
	f.SetContent(Plain("helloworld"))

	// Select "oworl" - spans across the wrap point (positions 4-9)
	f.SetSelection(4, 9)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Count fill operations excluding background
	bgRect := rect.String()
	selectionFills := 0
	for _, op := range ops {
		if strings.Contains(op, "fill") && !strings.Contains(op, bgRect) {
			selectionFills++
		}
	}

	// Wrapped selection should produce multiple fill rectangles
	if selectionFills < 2 {
		t.Errorf("Redraw() should draw multiple selection rectangles for wrapped selection, got %d fills\nops: %v", selectionFills, ops)
	}
}

// TestDrawSelectionCorrectPosition tests that the selection highlight is at the correct pixel position.
func TestDrawSelectionCorrectPosition(t *testing.T) {
	rect := image.Rect(20, 10, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	f.SetContent(Plain("hello"))

	// Select "ell" (positions 1-4)
	// In scratch image coords: X = 10 (1 char) to X = 40 (4 chars), Y = 0 to 14
	// The scratch is then blitted to screen at frame origin.
	f.SetSelection(1, 4)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Look for a fill operation at the correct scratch-local coordinates
	// With scratch-based clipping, selection is drawn at local coords first.
	// Expected: fill (10,0)-(40,14) in scratch image
	expectedLocalRect := image.Rect(10, 0, 40, 14)
	foundCorrectRect := false
	for _, op := range ops {
		if strings.Contains(op, "fill") && strings.Contains(op, expectedLocalRect.String()) {
			foundCorrectRect = true
			break
		}
	}

	if !foundCorrectRect {
		t.Errorf("Redraw() did not draw selection at correct local position %v\ngot ops: %v", expectedLocalRect, ops)
	}

	// Also verify the blit places the scratch at the correct screen position
	foundBlit := false
	expectedScreenRect := fmt.Sprintf("fill %v", rect)
	for _, op := range ops {
		if strings.Contains(op, expectedScreenRect) {
			foundBlit = true
			break
		}
	}
	if !foundBlit {
		t.Errorf("Redraw() did not blit scratch to frame rect %v\ngot ops: %v", rect, ops)
	}
}

// TestDrawSelectionFullLine tests selecting an entire line.
func TestDrawSelectionFullLine(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello\nworld"
	f.SetContent(Plain("hello\nworld"))

	// Select entire first line including newline (positions 0-6)
	f.SetSelection(0, 6)

	display.(edwoodtest.GettableDrawOps).Clear()
	f.Redraw()

	ops := display.(edwoodtest.GettableDrawOps).DrawOps()

	// Should have a selection fill (not the background fill)
	bgRect := rect.String()
	foundSelection := false
	for _, op := range ops {
		if strings.Contains(op, "fill") && !strings.Contains(op, bgRect) {
			foundSelection = true
			break
		}
	}

	if !foundSelection {
		t.Errorf("Redraw() did not draw selection for full line\ngot ops: %v", ops)
	}
}

// mockMousectl creates a mock Mousectl for testing mouse selection.
// It creates a Mousectl with a buffered channel containing the provided events.
// The bidirectional channel is converted to receive-only when assigned to C.
func mockMousectl(events []draw.Mouse) *draw.Mousectl {
	// Create a bidirectional channel, which Go can convert to receive-only
	ch := make(chan draw.Mouse, len(events)+1)
	for _, e := range events {
		ch <- e
	}

	mc := &draw.Mousectl{
		C: ch, // Bidirectional chan can be assigned to <-chan
	}
	return mc
}

// TestSelectWithMouseSimpleClick tests that clicking without dragging
// sets p0 == p1 at the click position.
func TestSelectWithMouseSimpleClick(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello world" with 10px per char
	// Position 5 is at X=50 (the space)
	f.SetContent(Plain("hello world"))

	// Simulate click at position 5 (X=50, Y=7 in the middle of first line)
	// Button 1 down, then immediately button 1 up
	downEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1, // Left button down
	}
	// Mouse up event (buttons = 0)
	upEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 0, // Button released
	}

	mc := mockMousectl([]draw.Mouse{upEvent})
	p0, p1 := f.Select(mc, &downEvent)

	// Should select nothing (p0 == p1) at position 5
	if p0 != p1 {
		t.Errorf("Select() click without drag: got p0=%d, p1=%d, want p0 == p1", p0, p1)
	}
	if p0 != 5 {
		t.Errorf("Select() click position: got p0=%d, want 5", p0)
	}
}

// TestSelectWithMouseDragForward tests dragging from left to right
// to select text.
func TestSelectWithMouseDragForward(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello world" - selecting "ello" (positions 1-5)
	f.SetContent(Plain("hello world"))

	// Mouse down at position 1 (X=10)
	downEvent := draw.Mouse{
		Point:   image.Pt(10, 7),
		Buttons: 1,
	}
	// Drag to position 5 (X=50), then release
	dragEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1, // Still held
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 0, // Released
	}

	mc := mockMousectl([]draw.Mouse{dragEvent, upEvent})
	p0, p1 := f.Select(mc, &downEvent)

	// Should select positions 1-5 ("ello")
	if p0 != 1 {
		t.Errorf("Select() drag forward: got p0=%d, want 1", p0)
	}
	if p1 != 5 {
		t.Errorf("Select() drag forward: got p1=%d, want 5", p1)
	}
}

// TestSelectWithMouseDragBackward tests dragging from right to left
// to select text.
func TestSelectWithMouseDragBackward(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello world"
	f.SetContent(Plain("hello world"))

	// Mouse down at position 5 (X=50)
	downEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1,
	}
	// Drag backward to position 1 (X=10), then release
	dragEvent := draw.Mouse{
		Point:   image.Pt(10, 7),
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(10, 7),
		Buttons: 0,
	}

	mc := mockMousectl([]draw.Mouse{dragEvent, upEvent})
	p0, p1 := f.Select(mc, &downEvent)

	// Should select positions 1-5 (p0 should be <= p1)
	if p0 != 1 {
		t.Errorf("Select() drag backward: got p0=%d, want 1", p0)
	}
	if p1 != 5 {
		t.Errorf("Select() drag backward: got p1=%d, want 5", p1)
	}
}

// TestSelectWithMouseDragMultiLine tests dragging across multiple lines.
func TestSelectWithMouseDragMultiLine(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello\nworld" = 6 + 5 = 11 chars total
	// Position 2 is "l" on first line
	// Position 8 is "r" on second line
	f.SetContent(Plain("hello\nworld"))

	// Mouse down at position 2 (X=20, Y=7 on line 1)
	downEvent := draw.Mouse{
		Point:   image.Pt(20, 7),
		Buttons: 1,
	}
	// Drag to position 8 (X=20, Y=21 on line 2), then release
	// "world" starts at position 6, so position 8 is at X=20 on line 2
	dragEvent := draw.Mouse{
		Point:   image.Pt(20, 21), // Line 2 (Y=14-28)
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(20, 21),
		Buttons: 0,
	}

	mc := mockMousectl([]draw.Mouse{dragEvent, upEvent})
	p0, p1 := f.Select(mc, &downEvent)

	// Should select from position 2 to 8
	if p0 != 2 {
		t.Errorf("Select() multi-line drag: got p0=%d, want 2", p0)
	}
	if p1 != 8 {
		t.Errorf("Select() multi-line drag: got p1=%d, want 8", p1)
	}
}

// TestSelectWithMouseSetsFrameSelection tests that Select() updates
// the frame's internal selection state.
func TestSelectWithMouseSetsFrameSelection(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	f.SetContent(Plain("hello world"))

	// Mouse down at position 0, drag to position 5
	downEvent := draw.Mouse{
		Point:   image.Pt(0, 7),
		Buttons: 1,
	}
	dragEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 0,
	}

	mc := mockMousectl([]draw.Mouse{dragEvent, upEvent})
	f.Select(mc, &downEvent)

	// Verify that GetSelection returns the same values
	p0, p1 := f.GetSelection()
	if p0 != 0 || p1 != 5 {
		t.Errorf("GetSelection() after Select(): got (%d, %d), want (0, 5)", p0, p1)
	}
}

// TestSelectWithMouseAtFrameEdge tests selection behavior at frame boundaries.
func TestSelectWithMouseAtFrameEdge(t *testing.T) {
	rect := image.Rect(10, 10, 100, 50)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	f.SetContent(Plain("hello"))

	// Click at position 2 (accounting for frame offset: X=10+20=30)
	downEvent := draw.Mouse{
		Point:   image.Pt(30, 17), // Y=10+7=17
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(30, 17),
		Buttons: 0,
	}

	mc := mockMousectl([]draw.Mouse{upEvent})
	p0, p1 := f.Select(mc, &downEvent)

	if p0 != 2 || p1 != 2 {
		t.Errorf("Select() at frame offset: got (%d, %d), want (2, 2)", p0, p1)
	}
}

// TestSelectWithColor tests that SelectWithColor uses a custom sweep color
// during the drag and returns the correct selection range.
func TestSelectWithColor(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))
	sweepImage := edwoodtest.NewImage(display, "sweep-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	// "hello world" with 10px per char
	f.SetContent(Plain("hello world"))

	fi := f.(*frameImpl)

	// Verify sweepColor starts nil
	if fi.sweepColor != nil {
		t.Fatal("sweepColor should be nil before SelectWithColor")
	}

	// Mouse down at position 1 (X=10), drag to position 5 (X=50), release
	downEvent := draw.Mouse{
		Point:   image.Pt(10, 7),
		Buttons: 1,
	}
	dragEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 0,
	}

	mc := mockMousectl([]draw.Mouse{dragEvent, upEvent})
	p0, p1 := fi.SelectWithColor(mc, &downEvent, sweepImage)

	// Should select positions 1-5 ("ello ")
	if p0 != 1 {
		t.Errorf("SelectWithColor() p0: got %d, want 1", p0)
	}
	if p1 != 5 {
		t.Errorf("SelectWithColor() p1: got %d, want 5", p1)
	}

	// sweepColor must be cleared after selection completes
	if fi.sweepColor != nil {
		t.Error("sweepColor should be nil after SelectWithColor completes")
	}
}

// TestSelectWithChordAndColor tests that SelectWithChordAndColor uses a custom
// sweep color during the drag and detects chord buttons.
func TestSelectWithChordAndColor(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))
	sweepImage := edwoodtest.NewImage(display, "sweep-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	f.SetContent(Plain("hello world"))

	fi := f.(*frameImpl)

	// Mouse down B1 at position 1 (X=10), drag, chord B2, release
	downEvent := draw.Mouse{
		Point:   image.Pt(10, 7),
		Buttons: 1,
	}
	dragEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1,
	}
	// Chord: B1+B2 pressed simultaneously
	chordEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 1 | 2, // B1+B2
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(50, 7),
		Buttons: 0,
	}

	mc := mockMousectl([]draw.Mouse{dragEvent, chordEvent, upEvent})
	p0, p1, chord := fi.SelectWithChordAndColor(mc, &downEvent, sweepImage)

	// Should select positions 1-5
	if p0 != 1 {
		t.Errorf("SelectWithChordAndColor() p0: got %d, want 1", p0)
	}
	if p1 != 5 {
		t.Errorf("SelectWithChordAndColor() p1: got %d, want 5", p1)
	}

	// Should detect B1+B2 chord
	if chord != (1 | 2) {
		t.Errorf("SelectWithChordAndColor() chord: got %d, want %d", chord, 1|2)
	}

	// sweepColor must be cleared after selection completes
	if fi.sweepColor != nil {
		t.Error("sweepColor should be nil after SelectWithChordAndColor completes")
	}
}

// TestSweepColorCleared tests that sweepColor is always cleared after
// any colored select call completes, even for a simple click (no drag).
func TestSweepColorCleared(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage := edwoodtest.NewImage(display, "background", image.Rect(0, 0, 1, 1))
	textImage := edwoodtest.NewImage(display, "text-color", image.Rect(0, 0, 1, 1))
	selImage := edwoodtest.NewImage(display, "selection-color", image.Rect(0, 0, 1, 1))
	sweepImage := edwoodtest.NewImage(display, "sweep-color", image.Rect(0, 0, 1, 1))

	f := NewFrame()
	f.Init(rect,
		WithDisplay(display),
		WithBackground(bgImage),
		WithFont(font),
		WithTextColor(textImage),
		WithSelectionColor(selImage),
	)

	f.SetContent(Plain("hello world"))

	fi := f.(*frameImpl)

	// Simple click (no drag) - button down then immediately up
	downEvent := draw.Mouse{
		Point:   image.Pt(30, 7),
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(30, 7),
		Buttons: 0,
	}

	// Test SelectWithColor clears sweepColor after click
	mc := mockMousectl([]draw.Mouse{upEvent})
	fi.SelectWithColor(mc, &downEvent, sweepImage)
	if fi.sweepColor != nil {
		t.Error("sweepColor should be nil after SelectWithColor click (no drag)")
	}

	// Test SelectWithChordAndColor clears sweepColor after click
	mc = mockMousectl([]draw.Mouse{upEvent})
	fi.SelectWithChordAndColor(mc, &downEvent, sweepImage)
	if fi.sweepColor != nil {
		t.Error("sweepColor should be nil after SelectWithChordAndColor click (no drag)")
	}
}
