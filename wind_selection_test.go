package main

// Tests for Phase 1.3: Selection Integration Tests
//
// These tests exercise the wind.go selection sync paths:
// - syncSourceSelection(): maps preview selection to source body positions
// - PreviewSnarf(): extracts source markdown for the current preview selection
// - PreviewLookText() / PreviewExecText(): extract rendered text for Look/Exec
// - Bounds validation: positions are clamped to buffer length
//
// Targets Bug 5 from the source map correctness design doc.

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/markdown"
)

// setupSelectionTestWindow creates a Window in preview mode with the given
// markdown source. It returns the window with richBody, sourceMap, and body
// buffer all wired up. The caller can then set selections on w.richBody and
// call syncSourceSelection, PreviewSnarf, etc.
func setupSelectionTestWindow(t *testing.T, sourceMarkdown string) *Window {
	t.Helper()

	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/selection.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	return w
}

// --------------------------------------------------------------------------
// syncSourceSelection tests
// --------------------------------------------------------------------------

// TestSyncSourceSelection_ValidRange tests that syncSourceSelection correctly
// maps a valid rendered selection to source positions in the body buffer.
// Plain text has 1:1 mapping.
func TestSyncSourceSelection_ValidRange(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World!")

	// Select "World" (rendered positions 7-12)
	w.richBody.SetSelection(7, 12)
	w.syncSourceSelection()

	// Plain text: rendered positions == source positions
	if w.body.q0 != 7 || w.body.q1 != 12 {
		t.Errorf("syncSourceSelection plain text: body.q0=%d, body.q1=%d, want (7, 12)", w.body.q0, w.body.q1)
	}
}

// TestSyncSourceSelection_PointSelectionInHeading tests that a point selection
// (click) in a heading maps to a single point in the source, not a range.
// This exercises the heading prefix mapping (Bug 1/Bug 3 from design doc).
func TestSyncSourceSelection_PointSelectionInHeading(t *testing.T) {
	w := setupSelectionTestWindow(t, "# Hello")
	// Rendered: "Hello" (5 chars, prefix "# " stripped)
	// Source:   "# Hello" (7 chars)

	// Point selection at rendered position 0 (start of "Hello")
	w.richBody.SetSelection(0, 0)
	w.syncSourceSelection()

	// Point selection must produce srcStart == srcEnd (Invariant R3)
	if w.body.q0 != w.body.q1 {
		t.Errorf("syncSourceSelection point in heading: body.q0=%d != body.q1=%d, want equal (point selection)", w.body.q0, w.body.q1)
	}

	// Point selection at rendered position 3 (middle of "Hello")
	w.richBody.SetSelection(3, 3)
	w.syncSourceSelection()

	if w.body.q0 != w.body.q1 {
		t.Errorf("syncSourceSelection point in heading middle: body.q0=%d != body.q1=%d, want equal", w.body.q0, w.body.q1)
	}
	// Position 3 in rendered "Hello" is 'l', which is rune 5 in source "# Hello"
	if w.body.q0 != 5 {
		t.Errorf("syncSourceSelection point in heading middle: body.q0=%d, want 5", w.body.q0)
	}
}

// TestSyncSourceSelection_RangeCrossingFormattedText tests selection that starts
// in plain text and crosses into bold-formatted text.
func TestSyncSourceSelection_RangeCrossingFormattedText(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello **bold** world")
	// Rendered: "Hello bold world" (16 chars)
	// Source:   "Hello **bold** world" (20 chars)

	// Select "o bold w" — crosses from plain "Hello" into bold "bold" into plain "world"
	// Rendered: positions 4-12
	// "Hell[o bold w]orld"
	//  0123456789012345
	w.richBody.SetSelection(4, 12)
	w.syncSourceSelection()

	// Source: "Hell[o **bold** w]orld"
	//          01234567890123456789
	// srcStart should be 4 (plain "o" at position 4)
	// srcEnd should be 16 (plain "w" at position 15, end at 16)
	if w.body.q0 != 4 {
		t.Errorf("syncSourceSelection cross-format start: body.q0=%d, want 4", w.body.q0)
	}
	if w.body.q1 != 16 {
		t.Errorf("syncSourceSelection cross-format end: body.q1=%d, want 16", w.body.q1)
	}
}

// TestSyncSourceSelection_OutOfBoundsAfterEdit tests that when the source map
// is stale (document was shortened after parsing), syncSourceSelection clamps
// positions to the buffer length rather than panicking or producing invalid indices.
// This is the primary Bug 5 scenario.
func TestSyncSourceSelection_OutOfBoundsAfterEdit(t *testing.T) {
	// Start with a longer document
	w := setupSelectionTestWindow(t, "Hello, World! This is a long sentence.")

	// Now shorten the body buffer to simulate an edit, but keep the stale source map.
	// Delete everything after "Hello" — body is now 5 runes.
	bodyLen := w.body.file.Nr()
	w.body.file.DeleteAt(5, bodyLen)

	newLen := w.body.file.Nr()
	if newLen != 5 {
		t.Fatalf("after trim, body length should be 5, got %d", newLen)
	}

	// Select text beyond the new buffer length using the stale source map
	// The source map still thinks positions 7-12 map somewhere valid.
	w.richBody.SetSelection(7, 12)
	w.syncSourceSelection()

	// After clamping, positions should be within [0, 5]
	if w.body.q0 < 0 || w.body.q0 > newLen {
		t.Errorf("syncSourceSelection out-of-bounds: body.q0=%d out of range [0, %d]", w.body.q0, newLen)
	}
	if w.body.q1 < 0 || w.body.q1 > newLen {
		t.Errorf("syncSourceSelection out-of-bounds: body.q1=%d out of range [0, %d]", w.body.q1, newLen)
	}
}

// TestSyncSourceSelection_NotPreviewMode tests that syncSourceSelection is a
// no-op when the window is not in preview mode.
func TestSyncSourceSelection_NotPreviewMode(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello")

	// Exit preview mode
	w.SetPreviewMode(false)

	// Set known body positions
	w.body.q0 = 42
	w.body.q1 = 99

	w.syncSourceSelection()

	// Should not have changed
	if w.body.q0 != 42 || w.body.q1 != 99 {
		t.Errorf("syncSourceSelection when not in preview: modified body positions to (%d, %d)", w.body.q0, w.body.q1)
	}
}

// TestSyncSourceSelection_NilSourceMap tests that syncSourceSelection is a
// no-op when the source map is nil.
func TestSyncSourceSelection_NilSourceMap(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello")

	w.SetPreviewSourceMap(nil)

	w.body.q0 = 42
	w.body.q1 = 99

	w.syncSourceSelection()

	if w.body.q0 != 42 || w.body.q1 != 99 {
		t.Errorf("syncSourceSelection with nil source map: modified body positions to (%d, %d)", w.body.q0, w.body.q1)
	}
}

// --------------------------------------------------------------------------
// PreviewSnarf bounds validation tests
// --------------------------------------------------------------------------

// TestPreviewSnarf_MixedFormatting tests PreviewSnarf with a selection spanning
// multiple formatting types (plain + bold + plain).
func TestPreviewSnarf_MixedFormatting(t *testing.T) {
	w := setupSelectionTestWindow(t, "Say **hello** world")
	// Rendered: "Say hello world" (15 chars)
	// Source:   "Say **hello** world" (19 chars)

	// Select "hello world" in rendered (positions 4-15)
	w.richBody.SetSelection(4, 15)

	snarfBytes := w.PreviewSnarf()
	if snarfBytes == nil {
		t.Fatal("PreviewSnarf should not return nil for valid selection")
	}

	snarfText := string(snarfBytes)
	// Should include the ** markers around "hello" plus " world"
	// Source positions 4-19: "**hello** world"
	want := "**hello** world"
	if snarfText != want {
		t.Errorf("PreviewSnarf mixed formatting: got %q, want %q", snarfText, want)
	}
}

// TestPreviewSnarf_OutOfBoundsAfterEdit tests that PreviewSnarf clamps to buffer
// bounds when the source map is stale after a document edit.
func TestPreviewSnarf_OutOfBoundsAfterEdit(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World! Extra text here.")

	// Shorten the buffer
	bodyLen := w.body.file.Nr()
	w.body.file.DeleteAt(5, bodyLen)

	// Select a range in rendered text that the stale source map maps beyond buffer
	w.richBody.SetSelection(7, 12)

	// Should not panic; should return nil or a valid (possibly empty) result
	snarfBytes := w.PreviewSnarf()
	// The result depends on clamping behavior — if both positions clamp to 5,
	// srcStart >= srcEnd and we get nil. That's acceptable.
	if snarfBytes != nil {
		// If we got something, verify it's from within the buffer
		snarfText := string(snarfBytes)
		if len(snarfText) > 5 {
			t.Errorf("PreviewSnarf after edit: result %q is longer than buffer (%d runes)", snarfText, 5)
		}
	}
}

// TestPreviewSnarf_NoSelection tests that PreviewSnarf returns nil when there
// is no selection (point selection p0==p1).
func TestPreviewSnarf_NoSelection(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World!")

	// Point selection (no range)
	w.richBody.SetSelection(3, 3)

	if got := w.PreviewSnarf(); got != nil {
		t.Errorf("PreviewSnarf with no selection: got %q, want nil", string(got))
	}
}

// TestPreviewSnarf_NotPreviewMode tests that PreviewSnarf returns nil when
// not in preview mode.
func TestPreviewSnarf_NotPreviewMode(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World!")

	w.SetPreviewMode(false)
	w.richBody.SetSelection(0, 5)

	if got := w.PreviewSnarf(); got != nil {
		t.Errorf("PreviewSnarf not in preview mode: got %q, want nil", string(got))
	}
}

// TestPreviewSnarf_NilSourceMap tests that PreviewSnarf returns nil when
// the source map is nil.
func TestPreviewSnarf_NilSourceMap(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World!")

	w.SetPreviewSourceMap(nil)
	w.richBody.SetSelection(0, 5)

	if got := w.PreviewSnarf(); got != nil {
		t.Errorf("PreviewSnarf nil source map: got %q, want nil", string(got))
	}
}

// --------------------------------------------------------------------------
// PreviewLookText / PreviewExecText bounds validation tests
// --------------------------------------------------------------------------

// TestPreviewLookText_ValidSelection tests that PreviewLookText returns the
// rendered text (not source markdown) for a valid selection.
func TestPreviewLookText_ValidSelection(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello **bold** world")
	// Rendered: "Hello bold world"

	// Find "bold" in rendered text
	plainText := w.richBody.Content().Plain()
	boldIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "bold" {
			boldIdx = i
			break
		}
	}
	if boldIdx < 0 {
		t.Fatalf("Could not find 'bold' in rendered text: %q", string(plainText))
	}

	w.richBody.SetSelection(boldIdx, boldIdx+4)

	got := w.PreviewLookText()
	if got != "bold" {
		t.Errorf("PreviewLookText: got %q, want %q", got, "bold")
	}
}

// TestPreviewLookText_NoSelection tests that PreviewLookText returns empty
// string when there is no selection.
func TestPreviewLookText_NoSelection(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World!")

	w.richBody.SetSelection(3, 3) // point selection

	if got := w.PreviewLookText(); got != "" {
		t.Errorf("PreviewLookText no selection: got %q, want empty", got)
	}
}

// TestPreviewLookText_NotPreviewMode tests that PreviewLookText returns empty
// string when not in preview mode.
func TestPreviewLookText_NotPreviewMode(t *testing.T) {
	w := setupSelectionTestWindow(t, "Hello, World!")

	w.SetPreviewMode(false)
	w.richBody.SetSelection(0, 5)

	if got := w.PreviewLookText(); got != "" {
		t.Errorf("PreviewLookText not in preview: got %q, want empty", got)
	}
}

// TestPreviewExecText_DelegatesToLookText tests that PreviewExecText returns
// the same result as PreviewLookText (they share implementation).
func TestPreviewExecText_DelegatesToLookText(t *testing.T) {
	w := setupSelectionTestWindow(t, "Run **echo** now")

	// Find "echo" in rendered text
	plainText := w.richBody.Content().Plain()
	echoIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "echo" {
			echoIdx = i
			break
		}
	}
	if echoIdx < 0 {
		t.Fatalf("Could not find 'echo' in rendered text: %q", string(plainText))
	}

	w.richBody.SetSelection(echoIdx, echoIdx+4)

	lookText := w.PreviewLookText()
	execText := w.PreviewExecText()

	if lookText != execText {
		t.Errorf("PreviewExecText=%q differs from PreviewLookText=%q", execText, lookText)
	}
	if execText != "echo" {
		t.Errorf("PreviewExecText: got %q, want %q", execText, "echo")
	}
}

// --------------------------------------------------------------------------
// Bounds validation: PreviewSnarf incomplete clamping (Bug 5)
// --------------------------------------------------------------------------

// TestPreviewSnarf_SrcStartClampedAboveBodyLen tests that PreviewSnarf clamps
// srcStart when it exceeds bodyLen, not just srcEnd. The design doc notes that
// PreviewSnarf clamps srcEnd > bodyLen but not srcStart > bodyLen.
func TestPreviewSnarf_SrcStartClampedAboveBodyLen(t *testing.T) {
	// Create a document, then shrink it so all stale source positions are beyond buffer
	w := setupSelectionTestWindow(t, "ABCDEFGHIJKLMNOP")

	// Delete the entire body content
	bodyLen := w.body.file.Nr()
	w.body.file.DeleteAt(0, bodyLen)

	newLen := w.body.file.Nr()
	if newLen != 0 {
		t.Fatalf("after delete, body length should be 0, got %d", newLen)
	}

	// Select a range in rendered text with stale source map
	w.richBody.SetSelection(2, 8)

	// Should not panic — both srcStart and srcEnd will be beyond buffer
	snarfBytes := w.PreviewSnarf()
	// With correct clamping, both positions clamp to 0, so srcStart >= srcEnd → nil
	if snarfBytes != nil {
		t.Errorf("PreviewSnarf with empty buffer: got %q, want nil", string(snarfBytes))
	}
}

// --------------------------------------------------------------------------
// syncSourceSelection with heading range (includes prefix expansion)
// --------------------------------------------------------------------------

// TestSyncSourceSelection_HeadingFullRange tests that selecting the entire
// rendered heading maps to the full source including the "# " prefix.
func TestSyncSourceSelection_HeadingFullRange(t *testing.T) {
	w := setupSelectionTestWindow(t, "# Title")
	// Rendered: "Title" (5 chars)
	// Source:   "# Title" (7 chars)

	// Select entire rendered heading
	w.richBody.SetSelection(0, 5)
	w.syncSourceSelection()

	// Should map to full source including prefix
	if w.body.q0 != 0 {
		t.Errorf("syncSourceSelection heading full range: body.q0=%d, want 0", w.body.q0)
	}
	if w.body.q1 != 7 {
		t.Errorf("syncSourceSelection heading full range: body.q1=%d, want 7", w.body.q1)
	}
}

// TestSyncSourceSelection_HeadingPartialRange tests that selecting part of a
// rendered heading maps to the corresponding source positions past the prefix.
func TestSyncSourceSelection_HeadingPartialRange(t *testing.T) {
	w := setupSelectionTestWindow(t, "# Title")
	// Rendered: "Title" (5 chars)
	// Source:   "# Title" (7 chars)

	// Select "itl" in rendered (positions 1-4)
	w.richBody.SetSelection(1, 4)
	w.syncSourceSelection()

	// "itl" is at source positions 3-6 (past "# T", up to but not including "e")
	if w.body.q0 != 3 {
		t.Errorf("syncSourceSelection heading partial: body.q0=%d, want 3", w.body.q0)
	}
	if w.body.q1 != 6 {
		t.Errorf("syncSourceSelection heading partial: body.q1=%d, want 6", w.body.q1)
	}
}

// --------------------------------------------------------------------------
// Code block selection
// --------------------------------------------------------------------------

// TestSyncSourceSelection_CodeBlock tests selection within a fenced code block.
func TestSyncSourceSelection_CodeBlock(t *testing.T) {
	source := "```\ncode line\n```"
	w := setupSelectionTestWindow(t, source)

	// The rendered content for a code block should include "code line"
	plainText := w.richBody.Content().Plain()

	// Find "code" in the rendered text
	codeIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "code" {
			codeIdx = i
			break
		}
	}
	if codeIdx < 0 {
		t.Fatalf("Could not find 'code' in rendered text: %q", string(plainText))
	}

	// Select "code" in rendered text
	w.richBody.SetSelection(codeIdx, codeIdx+4)
	w.syncSourceSelection()

	// Verify the selected source text is "code"
	buf := make([]rune, w.body.q1-w.body.q0)
	w.body.file.Read(w.body.q0, buf)
	got := string(buf)
	if got != "code" {
		t.Errorf("syncSourceSelection code block: selected source text %q, want %q", got, "code")
	}
}

// --------------------------------------------------------------------------
// PreviewSnarf with heading
// --------------------------------------------------------------------------

// TestPreviewSnarf_HeadingFullSelection tests that snarfing the entire rendered
// heading includes the "# " prefix in the source text.
func TestPreviewSnarf_HeadingFullSelection(t *testing.T) {
	w := setupSelectionTestWindow(t, "# Hello")
	// Rendered: "Hello" (5 chars)

	// Select entire rendered heading
	w.richBody.SetSelection(0, 5)

	snarfBytes := w.PreviewSnarf()
	if snarfBytes == nil {
		t.Fatal("PreviewSnarf heading: got nil")
	}

	snarfText := string(snarfBytes)
	if snarfText != "# Hello" {
		t.Errorf("PreviewSnarf heading full: got %q, want %q", snarfText, "# Hello")
	}
}

// --------------------------------------------------------------------------
// PreviewLookText with out-of-bounds rendered positions
// --------------------------------------------------------------------------

// TestPreviewLookText_OutOfBoundsRenderedPositions tests that PreviewLookText
// returns empty string when rendered positions exceed the content length
// (e.g., p0 < 0 or p1 > len(plainText)).
func TestPreviewLookText_OutOfBoundsRenderedPositions(t *testing.T) {
	w := setupSelectionTestWindow(t, "Short")

	plainText := w.richBody.Content().Plain()
	contentLen := len(plainText)

	// Selection beyond content end
	w.richBody.SetSelection(0, contentLen+10)

	got := w.PreviewLookText()
	// Should return empty string because p1 > len(plainText)
	if got != "" {
		t.Errorf("PreviewLookText out-of-bounds: got %q, want empty", got)
	}
}
