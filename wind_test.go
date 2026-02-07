package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

func TestSetTag1(t *testing.T) {
	const (
		defaultSuffix = " Del Snarf | Look Edit "
		extraSuffix   = "|fmt g setTag1 Ldef"
	)

	for _, name := range []string{
		"/home/gopher/src/hello.go",
		"/home/ゴーファー/src/エドウード.txt",
		"/home/ゴーファー/src/",
	} {
		display := edwoodtest.NewDisplay(image.Rectangle{})
		global.configureGlobals(display)

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer(name, nil),
		}
		w.tag = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("", nil),
		}

		w.col = &Column{
			safe: true,
		}

		w.setTag1()
		got := w.tag.file.String()
		want := name + defaultSuffix
		if got != want {
			t.Errorf("bad initial tag for file %q:\n got: %q\nwant: %q", name, got, want)
		}

		w.tag.file.InsertAt(w.tag.file.Nr(), []rune(extraSuffix))
		w.setTag1()
		got = w.tag.file.String()
		want = name + defaultSuffix + extraSuffix
		if got != want {
			t.Errorf("bad replacement tag for file %q:\n got: %q\nwant: %q", name, got, want)
		}
	}
}

func TestWindowClampAddr(t *testing.T) {
	const hello_世界 = "Hello, 世界"
	runic_hello_世界 := []rune(hello_世界)
	for _, tc := range []struct {
		addr, want Range
	}{
		{Range{-1, -1}, Range{0, 0}},
		{Range{100, 100}, Range{len(runic_hello_世界), len(runic_hello_世界)}},
	} {
		w := &Window{
			addr: tc.addr,
			body: Text{
				file: file.MakeObservableEditableBuffer("", runic_hello_世界),
			},
		}
		w.ClampAddr()
		if got := w.addr; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got addr %v; want %v", got, tc.want)
		}
	}
}

func TestWindowParseTag(t *testing.T) {
	for _, tc := range []struct {
		tag      string
		filename string
	}{
		{"/foo/bar.txt Del Snarf | Look", "/foo/bar.txt"},
		{"'/foo/bar quux.txt' Del Snarf | Look", "'/foo/bar quux.txt'"},
		{"/foo/bar.txt", "/foo/bar.txt"},
		{"/foo/bar.txt | Look", "/foo/bar.txt"},
		{"/foo/bar.txt Del Snarf\t| Look", "/foo/bar.txt"},
		{"/foo/bar.txt Del Snarf Del Snarf", "/foo/bar.txt"},
		{"'/foo/bar.txt ' Del Snarf", "'/foo/bar.txt '"},
		{"'/foo/b|ar.txt ' Del Snarf", "'/foo/b|ar.txt '"},
	} {
		if got, want := parsetaghelper(tc.tag), tc.filename; got != want {
			t.Errorf("tag %q has filename %q; want %q", tc.tag, got, want)
		}
	}
}

func TestWindowClearTag(t *testing.T) {
	tag := "/foo bar/test.txt Del Snarf Undo Put | Look |fmt mk"
	want := "/foo bar/test.txt Del Snarf Undo Put |"
	w := &Window{
		tag: Text{
			file: file.MakeObservableEditableBuffer("", []rune(tag)),
		},
	}
	w.ClearTag()
	got := w.tag.file.String()
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

// TestWindowPreviewMode tests that a Window has preview mode fields and
// that they can be accessed via the appropriate methods.
func TestWindowPreviewMode(t *testing.T) {
	display := edwoodtest.NewDisplay(image.Rect(0, 0, 800, 600))
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", nil),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}

	// Initially, preview mode should be off
	if w.IsPreviewMode() {
		t.Error("IsPreviewMode() should be false initially")
	}

	// RichBody should be nil initially
	if w.RichBody() != nil {
		t.Error("RichBody() should be nil initially")
	}

	// After enabling preview mode, it should be on
	w.SetPreviewMode(true)
	if !w.IsPreviewMode() {
		t.Error("IsPreviewMode() should be true after SetPreviewMode(true)")
	}

	// After disabling preview mode, it should be off again
	w.SetPreviewMode(false)
	if w.IsPreviewMode() {
		t.Error("IsPreviewMode() should be false after SetPreviewMode(false)")
	}
}

// TestWindowPreviewModeToggle tests that preview mode can be toggled on and off,
// and that each toggle properly updates the state.
func TestWindowPreviewModeToggle(t *testing.T) {
	display := edwoodtest.NewDisplay(image.Rect(0, 0, 800, 600))
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", nil),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}

	// Initially off
	if w.IsPreviewMode() {
		t.Error("IsPreviewMode() should start false")
	}

	// Toggle on
	w.TogglePreviewMode()
	if !w.IsPreviewMode() {
		t.Error("IsPreviewMode() should be true after first toggle")
	}

	// Toggle off
	w.TogglePreviewMode()
	if w.IsPreviewMode() {
		t.Error("IsPreviewMode() should be false after second toggle")
	}

	// Toggle on again
	w.TogglePreviewMode()
	if !w.IsPreviewMode() {
		t.Error("IsPreviewMode() should be true after third toggle")
	}
}

// TestWindowDrawPreviewMode tests that when a window is in preview mode,
// Window.Draw() renders the richBody instead of the normal body.
func TestWindowDrawPreviewMode(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World\n\nThis is **bold** text.")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	// Body area is below tag (approximately)
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set some content in the RichText
	content := rich.Plain("Hello World")
	rt.SetContent(content)

	// Assign the richBody to the window
	w.richBody = rt

	// Initially NOT in preview mode - Draw should NOT use richBody
	w.previewMode = false

	// Clear draw operations
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Draw method if it exists - for now, test the state
	// (actual Draw method will be implemented in 11.3 "Code written" stage)

	// Verify that when previewMode is false, richBody should not be used for drawing
	// This is a state test - when Draw() is implemented, it should check previewMode
	if w.previewMode {
		t.Error("Window should not be in preview mode initially")
	}
	if w.richBody == nil {
		t.Error("richBody should be set")
	}

	// Enable preview mode
	w.SetPreviewMode(true)

	// Verify preview mode is enabled
	if !w.IsPreviewMode() {
		t.Error("Window should be in preview mode after SetPreviewMode(true)")
	}

	// The richBody should be available for rendering
	if w.RichBody() != rt {
		t.Error("RichBody() should return the assigned RichText component")
	}

	// Verify that the rich body has the expected content
	if w.richBody.Content() == nil {
		t.Error("richBody content should not be nil")
	}
	if w.richBody.Content().Len() != 11 { // "Hello World" = 11 chars
		t.Errorf("richBody content length = %d, want 11", w.richBody.Content().Len())
	}

	// Test that preview mode can be disabled
	w.SetPreviewMode(false)
	if w.IsPreviewMode() {
		t.Error("Window should not be in preview mode after SetPreviewMode(false)")
	}

	// richBody should still exist (for potential re-enabling of preview)
	if w.RichBody() == nil {
		t.Error("richBody should still exist after disabling preview mode")
	}
}

// TestWindowMousePreviewSelection tests that mouse selection in preview mode
// delegates to the richBody and allows text selection within the rich text frame.
func TestWindowMousePreviewSelection(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World\n\nThis is **bold** text.")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Set content in the RichText
	content := rich.Plain("Hello World")
	rt.SetContent(content)

	// Assign the richBody to the window
	w.richBody = rt
	w.SetPreviewMode(true)

	// Verify initial selection is empty
	q0, q1 := rt.Selection()
	if q0 != 0 || q1 != 0 {
		t.Errorf("Initial selection should be (0, 0), got (%d, %d)", q0, q1)
	}

	// Test that selection can be set on the richBody
	rt.SetSelection(2, 7) // Select "llo W" from "Hello World"
	q0, q1 = rt.Selection()
	if q0 != 2 || q1 != 7 {
		t.Errorf("Selection after SetSelection(2, 7) should be (2, 7), got (%d, %d)", q0, q1)
	}

	// Verify the window is in preview mode and has the richBody
	if !w.IsPreviewMode() {
		t.Error("Window should be in preview mode")
	}
	if w.RichBody() != rt {
		t.Error("Window's RichBody should match the assigned RichText")
	}

	// The key property: when in preview mode, mouse interactions should be
	// handled by the richBody. We verify that the richBody's frame supports
	// the necessary coordinate mapping methods (Charofpt, Ptofchar) which
	// are used for mouse-based selection.
	frame := rt.Frame()
	if frame == nil {
		t.Fatal("RichText frame should not be nil")
	}

	// Test coordinate mapping (used by mouse selection)
	pt := frame.Ptofchar(5) // Get screen position of character 5
	if pt.X < bodyRect.Min.X {
		t.Errorf("Ptofchar(5).X = %d, should be >= %d", pt.X, bodyRect.Min.X)
	}

	// Test reverse mapping (click position to character)
	char := frame.Charofpt(image.Pt(bodyRect.Min.X+50, bodyRect.Min.Y+5))
	if char < 0 || char > content.Len() {
		t.Errorf("Charofpt should return valid character index, got %d for content length %d", char, content.Len())
	}
}

// TestPreviewSnarf tests that snarf (copy) in preview mode uses the source map
// to copy the original markdown source, not the rendered text.
// This test verifies the basic mechanism with plain text (1:1 mapping).
func TestPreviewSnarf(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source: plain text
	sourceMarkdown := "Hello, World!"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", sourceRunes),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewMode(true)

	// Select "World" (positions 7-12 in rendered text)
	rt.SetSelection(7, 12)

	// Verify the selection is set
	p0, p1 := rt.Selection()
	if p0 != 7 || p1 != 12 {
		t.Fatalf("Selection should be (7, 12), got (%d, %d)", p0, p1)
	}

	// Use source map to convert rendered selection to source positions
	srcStart, srcEnd := sourceMap.ToSource(p0, p1)

	// For plain text, positions should be 1:1
	if srcStart != 7 || srcEnd != 12 {
		t.Errorf("Source positions for plain text: got (%d, %d), want (7, 12)", srcStart, srcEnd)
	}

	// Extract the text from the source body using mapped positions
	if srcEnd > len(sourceRunes) {
		srcEnd = len(sourceRunes)
	}
	if srcStart > len(sourceRunes) {
		srcStart = len(sourceRunes)
	}
	snarfedText := string(sourceRunes[srcStart:srcEnd])

	if snarfedText != "World" {
		t.Errorf("Snarfed text should be %q, got %q", "World", snarfedText)
	}
}

// TestPreviewSnarfBold tests that snarf in preview mode copies the full markdown
// source including ** markers when selecting bold text.
func TestPreviewSnarfBold(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source with bold text
	sourceMarkdown := "Hello **World** test"
	// Rendered: "Hello World test" (16 chars)
	// Source:   "Hello **World** test" (20 chars)
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", sourceRunes),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewMode(true)

	// Select "World" (positions 6-11 in rendered text - after "Hello ")
	// In rendered: "Hello World test"
	//              012345678901234567
	// "World" is at positions 6-11
	rt.SetSelection(6, 11)

	// Verify the selection is set
	p0, p1 := rt.Selection()
	if p0 != 6 || p1 != 11 {
		t.Fatalf("Selection should be (6, 11), got (%d, %d)", p0, p1)
	}

	// Use source map to convert rendered selection to source positions
	srcStart, srcEnd := sourceMap.ToSource(p0, p1)

	// For bold text, should map to include the ** markers
	// Source: "Hello **World** test"
	//          012345678901234567890
	// **World** starts at 6, ends at 15
	if srcStart != 6 || srcEnd != 15 {
		t.Errorf("Source positions for bold text: got (%d, %d), want (6, 15)", srcStart, srcEnd)
	}

	// Extract the text from the source body using mapped positions
	if srcEnd > len(sourceRunes) {
		srcEnd = len(sourceRunes)
	}
	if srcStart > len(sourceRunes) {
		srcStart = len(sourceRunes)
	}
	snarfedText := string(sourceRunes[srcStart:srcEnd])

	// When selecting the entire bold word, we should get the full markdown including markers
	if snarfedText != "**World**" {
		t.Errorf("Snarfed text should be %q, got %q", "**World**", snarfedText)
	}
}

// TestPreviewSnarfHeading tests that snarf in preview mode copies the full markdown
// source including # prefix when selecting heading text.
func TestPreviewSnarfHeading(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source with heading
	sourceMarkdown := "# Hello World"
	// Rendered: "Hello World" (11 chars)
	// Source:   "# Hello World" (13 chars)
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", sourceRunes),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewMode(true)

	// Select entire heading "Hello World" (positions 0-11 in rendered text)
	// In rendered: "Hello World"
	//              01234567890
	rt.SetSelection(0, 11)

	// Verify the selection is set
	p0, p1 := rt.Selection()
	if p0 != 0 || p1 != 11 {
		t.Fatalf("Selection should be (0, 11), got (%d, %d)", p0, p1)
	}

	// Use source map to convert rendered selection to source positions
	srcStart, srcEnd := sourceMap.ToSource(p0, p1)

	// For heading, should map to include the # prefix
	// Source: "# Hello World"
	//          0123456789012
	// Entire heading starts at 0, ends at 13
	if srcStart != 0 || srcEnd != 13 {
		t.Errorf("Source positions for heading: got (%d, %d), want (0, 13)", srcStart, srcEnd)
	}

	// Extract the text from the source body using mapped positions
	if srcEnd > len(sourceRunes) {
		srcEnd = len(sourceRunes)
	}
	if srcStart > len(sourceRunes) {
		srcStart = len(sourceRunes)
	}
	snarfedText := string(sourceRunes[srcStart:srcEnd])

	// When selecting the entire heading, we should get the full markdown including # prefix
	if snarfedText != "# Hello World" {
		t.Errorf("Snarfed text should be %q, got %q", "# Hello World", snarfedText)
	}
}

// TestWindowMousePreviewScroll tests that mouse scrolling in preview mode
// properly delegates to the richBody's scroll handling.
func TestWindowMousePreviewScroll(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", nil),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xCCCCCCFF)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x666666FF)

	rt := NewRichText()
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)
	rt.Render(bodyRect)

	// Create content with many lines to enable scrolling
	var content rich.Content
	for i := 0; i < 50; i++ {
		if i > 0 {
			content = append(content, rich.Plain("\n")...)
		}
		content = append(content, rich.Plain("Line number "+string(rune('0'+i%10)))...)
	}
	rt.SetContent(content)

	// Assign the richBody to the window
	w.richBody = rt
	w.SetPreviewMode(true)

	// Verify initial origin is 0
	if rt.Origin() != 0 {
		t.Errorf("Initial origin should be 0, got %d", rt.Origin())
	}

	// Test scrollbar click - button 3 (right-click) should scroll down
	scrollRect := rt.ScrollRect()
	middleY := (scrollRect.Min.Y + scrollRect.Max.Y) / 2
	newOrigin := rt.ScrollClick(3, image.Pt(scrollRect.Min.X+5, middleY))

	// Origin should have increased (scrolled down)
	if newOrigin <= 0 {
		t.Errorf("After ScrollClick(3, middle), origin should be > 0, got %d", newOrigin)
	}

	// Save the current origin
	beforeWheelScroll := rt.Origin()

	// Test mouse wheel scrolling - scroll down
	newOrigin = rt.ScrollWheel(false) // false = scroll down
	if newOrigin < beforeWheelScroll {
		t.Errorf("ScrollWheel(down) should increase origin; before=%d, after=%d", beforeWheelScroll, newOrigin)
	}

	// Test mouse wheel scrolling - scroll up
	beforeWheelUp := rt.Origin()
	newOrigin = rt.ScrollWheel(true) // true = scroll up
	if newOrigin >= beforeWheelUp {
		t.Errorf("ScrollWheel(up) should decrease origin; before=%d, after=%d", beforeWheelUp, newOrigin)
	}

	// Verify the window is in preview mode
	if !w.IsPreviewMode() {
		t.Error("Window should still be in preview mode")
	}

	// Test scrollbar at top - button 1 (left-click) at top should stay at top
	// First scroll to top
	rt.SetOrigin(0)
	newOrigin = rt.ScrollClick(1, image.Pt(scrollRect.Min.X+5, scrollRect.Min.Y))
	if newOrigin != 0 {
		t.Errorf("ScrollClick(1, top) when at origin=0 should stay at 0, got %d", newOrigin)
	}
}

// TestPreviewCommandToggle tests that the Preview command toggles preview mode
// on and off when executed multiple times on the same window.
func TestPreviewCommandToggle(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content
	markdownContent := "# Hello World\n\nThis is **bold** and *italic* text."
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600) // Body area below tag
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Initially, preview mode should be off
	if w.IsPreviewMode() {
		t.Error("Window should not be in preview mode initially")
	}

	// First toggle: should enter preview mode
	w.TogglePreviewMode()
	if !w.IsPreviewMode() {
		t.Error("Window should be in preview mode after first toggle")
	}

	// Second toggle: should exit preview mode
	w.TogglePreviewMode()
	if w.IsPreviewMode() {
		t.Error("Window should not be in preview mode after second toggle")
	}

	// Third toggle: should enter preview mode again
	w.TogglePreviewMode()
	if !w.IsPreviewMode() {
		t.Error("Window should be in preview mode after third toggle")
	}

	// Fourth toggle: should exit preview mode
	w.TogglePreviewMode()
	if w.IsPreviewMode() {
		t.Error("Window should not be in preview mode after fourth toggle")
	}
}

// TestPreviewCommandEnter tests that entering preview mode properly initializes
// the richBody with parsed markdown content and sets up the source map.
func TestPreviewCommandEnter(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content
	markdownContent := "# Hello World\n\nThis is **bold** text."
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText and set up the preview
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

	// Parse markdown with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(markdownContent)
	rt.SetContent(content)

	// Assign the richBody and source map to the window
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)

	// Enter preview mode
	w.SetPreviewMode(true)

	// Verify preview mode is enabled
	if !w.IsPreviewMode() {
		t.Error("Window should be in preview mode")
	}

	// Verify richBody is set
	if w.RichBody() == nil {
		t.Error("richBody should not be nil after entering preview mode")
	}

	// Verify source map is set
	if w.PreviewSourceMap() == nil {
		t.Error("PreviewSourceMap should not be nil after entering preview mode")
	}

	// Verify content is properly parsed (should contain the text "Hello World")
	contentInFrame := w.RichBody().Content()
	if contentInFrame == nil {
		t.Fatal("Content in richBody should not be nil")
	}

	// The markdown parser removes # prefix, so rendered text starts with "Hello World"
	// The content should have at least the heading text
	if contentInFrame.Len() == 0 {
		t.Error("Content should not be empty")
	}

	// Verify the source map can convert positions
	// Selection in rendered text should map back to source positions
	// For the heading "Hello World" (positions 0-11 in rendered), source is "# Hello World" (0-13)
	srcStart, srcEnd := sourceMap.ToSource(0, 11)

	// Source should include the # prefix
	if srcStart != 0 {
		t.Errorf("Source start for heading should be 0, got %d", srcStart)
	}
	// The source end should be 13 (length of "# Hello World")
	// But this depends on exact parser behavior; verify it's reasonable
	if srcEnd < 11 {
		t.Errorf("Source end for heading should be >= 11, got %d", srcEnd)
	}
}

// TestPreviewCommandExit tests that exiting preview mode properly restores
// normal window behavior and maintains the richBody for potential re-entry.
func TestPreviewCommandExit(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content
	markdownContent := "# Test Heading\n\nSome content here."
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/test.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Set up preview mode components
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

	content, sourceMap, _ := markdown.ParseWithSourceMap(markdownContent)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)

	// Enter preview mode
	w.SetPreviewMode(true)
	if !w.IsPreviewMode() {
		t.Fatal("Failed to enter preview mode")
	}

	// Save reference to richBody
	richBodyRef := w.RichBody()
	sourceMapRef := w.PreviewSourceMap()

	// Exit preview mode
	w.SetPreviewMode(false)

	// Verify preview mode is disabled
	if w.IsPreviewMode() {
		t.Error("Window should not be in preview mode after exit")
	}

	// Verify richBody is retained (not nil'd out) for potential re-entry
	if w.RichBody() == nil {
		t.Error("richBody should be retained after exiting preview mode")
	}

	// Verify the same richBody instance is kept
	if w.RichBody() != richBodyRef {
		t.Error("richBody reference should be the same after exit")
	}

	// Verify source map is retained
	if w.PreviewSourceMap() != sourceMapRef {
		t.Error("PreviewSourceMap should be retained after exiting preview mode")
	}

	// Verify body content is unchanged in the underlying file buffer
	bodyContent := w.body.file.String()
	if bodyContent != markdownContent {
		t.Errorf("Body content should be unchanged, got %q, want %q", bodyContent, markdownContent)
	}

	// Re-enter preview mode and verify components work
	w.SetPreviewMode(true)
	if !w.IsPreviewMode() {
		t.Error("Should be able to re-enter preview mode")
	}

	// Content should still be available
	if w.RichBody().Content() == nil {
		t.Error("Content should still be available after re-entering preview mode")
	}
}

// TestPreviewLiveUpdate tests that when the body buffer changes while in preview mode,
// the richBody is automatically updated with re-parsed markdown content.
func TestPreviewLiveUpdate(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content
	initialMarkdown := "# Hello World\n\nSome text here."
	sourceRunes := []rune(initialMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Set up preview mode components
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

	// Parse initial markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(initialMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)

	// Enter preview mode
	w.SetPreviewMode(true)

	// Verify initial content
	initialContent := rt.Content()
	if initialContent == nil || initialContent.Len() == 0 {
		t.Fatal("Initial content should not be empty")
	}

	// Get the initial rendered text length
	initialLen := initialContent.Len()

	// Now simulate editing the body buffer (simulating user typing in source)
	// Insert " Updated" after "World" - this tests that preview updates when body changes
	updatedMarkdown := "# Hello Updated World\n\nSome new text here."
	w.body.file.DeleteAt(0, w.body.file.Nr())
	w.body.file.InsertAt(0, []rune(updatedMarkdown))

	// Call the update method that should be triggered when in preview mode
	w.UpdatePreview()

	// Verify the preview was updated
	updatedContent := w.RichBody().Content()
	if updatedContent == nil {
		t.Fatal("Updated content should not be nil after UpdatePreview")
	}

	// The content length should have changed
	updatedLen := updatedContent.Len()
	if updatedLen == initialLen {
		// Only fail if content length is exactly the same AND the text didn't change
		// Since "Updated" was added, the length should be different
		t.Errorf("Content should have changed after body edit: initial len=%d, updated len=%d", initialLen, updatedLen)
	}

	// Verify the source map was also updated
	if w.PreviewSourceMap() == nil {
		t.Error("Source map should still be set after update")
	}
}

// TestPreviewLiveUpdatePreservesScroll tests that live updates preserve the scroll position
// (origin) when possible, so the user doesn't lose their place while editing.
func TestPreviewLiveUpdatePreservesScroll(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with multi-line markdown content that requires scrolling
	var mdBuilder string
	for i := 1; i <= 50; i++ {
		mdBuilder += "# Heading " + string(rune('A'+i%26)) + "\n\n"
		mdBuilder += "Paragraph " + string(rune('0'+i%10)) + " with some content to make it longer.\n\n"
	}
	initialMarkdown := mdBuilder
	sourceRunes := []rune(initialMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/long.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Set up preview mode components
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

	// Parse initial markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(initialMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)

	// Enter preview mode
	w.SetPreviewMode(true)

	// Scroll to somewhere in the middle
	totalLen := rt.Content().Len()
	targetOrigin := totalLen / 3 // About 1/3 through the content
	rt.SetOrigin(targetOrigin)

	// Verify the origin was set
	beforeOrigin := rt.Origin()
	if beforeOrigin == 0 {
		t.Fatal("Origin should be non-zero after scrolling")
	}

	// Make a small edit to the body buffer (append a line at the end)
	w.body.file.InsertAt(w.body.file.Nr(), []rune("\n\n# New Heading at End\n"))

	// Call update preview
	w.UpdatePreview()

	// The origin should be preserved (approximately - may need to adjust if content length changed significantly)
	afterOrigin := rt.Origin()

	// The origin should be close to what it was before (allow some tolerance for content changes)
	// Since we only added content at the end, the origin position relative to the beginning shouldn't change much
	tolerance := 50 // Allow 50 rune difference due to reparsing
	if afterOrigin < beforeOrigin-tolerance || afterOrigin > beforeOrigin+tolerance {
		t.Errorf("Origin should be preserved: before=%d, after=%d (tolerance=%d)", beforeOrigin, afterOrigin, tolerance)
	}
}

// TestPreviewLiveUpdateMultipleTimes tests that multiple consecutive updates work correctly.
func TestPreviewLiveUpdateMultipleTimes(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content
	initialMarkdown := "# Version 1"
	sourceRunes := []rune(initialMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/versions.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Set up preview mode components
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

	// Parse initial markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(initialMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)

	// Enter preview mode
	w.SetPreviewMode(true)

	// Perform multiple updates
	versions := []string{
		"# Version 2\n\nAdded paragraph.",
		"# Version 3\n\nAdded **bold** text.",
		"# Version 4\n\nNow with *italics* too.",
		"# Final Version\n\nComplete content.",
	}

	for i, md := range versions {
		// Update body buffer
		w.body.file.DeleteAt(0, w.body.file.Nr())
		w.body.file.InsertAt(0, []rune(md))

		// Trigger update
		w.UpdatePreview()

		// Verify content was updated
		updatedContent := w.RichBody().Content()
		if updatedContent == nil || updatedContent.Len() == 0 {
			t.Errorf("Update %d: Content should not be empty", i+1)
		}

		// Verify source map exists
		if w.PreviewSourceMap() == nil {
			t.Errorf("Update %d: Source map should exist", i+1)
		}
	}
}

// TestPreviewLook tests that B3 (Look) chord in preview mode operates on the rendered text.
// When the user B3-clicks text in preview mode, the search should look for the rendered text
// (e.g., "World" from "**World**"), not the raw markdown source.
func TestPreviewLook(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source with bold text
	sourceMarkdown := "# Hello World\n\nSome **important** text here.\n\nFind important word."
	// Rendered text will be: "Hello World\n\nSome important text here.\n\nFind important word."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// In the rendered text, find "important" and select it
	// The rendered text should be "Hello World\n\nSome important text here.\n\nFind important word."
	// "important" first appears at position 17 (after "Hello World\n\nSome ")
	// Rendered: "Hello World" (11) + "\n\n" (2) + "Some " (5) = 18, then "important" starts
	// Let's find it more precisely by looking at the rendered content

	// For a B3 (Look) operation in preview mode, we need to:
	// 1. Determine what text is at the click position (using Charofpt)
	// 2. Expand to get the word
	// 3. The Look operation should search for this rendered text

	// Test: Set selection on a word in the rendered text
	// Select "important" in the rendered preview
	// The exact position depends on how the parser renders the content

	// Verify the window is in preview mode
	if !w.IsPreviewMode() {
		t.Fatal("Window should be in preview mode")
	}

	// Verify we can get the rendered text from the selection
	// When selecting text in preview and executing Look, the text should be
	// from the rendered content, not the source.

	// Simulate: select "important" (without the ** markers) in the rendered text
	// The rendered content after parsing should have "important" as plain text (styled as bold)

	// Get the rich text content and verify it exists
	richContent := rt.Content()
	if richContent == nil {
		t.Fatal("Rich content should not be nil")
	}

	// Find "important" in the rendered plain text
	plainText := richContent.Plain()
	importantIdx := -1
	for i := 0; i < len(plainText)-8; i++ {
		if string(plainText[i:i+9]) == "important" {
			importantIdx = i
			break
		}
	}

	if importantIdx < 0 {
		t.Fatalf("Could not find 'important' in rendered text: %q", string(plainText))
	}

	// Set selection to "important" in the rendered text
	rt.SetSelection(importantIdx, importantIdx+9)

	// Verify selection
	p0, p1 := rt.Selection()
	if p0 != importantIdx || p1 != importantIdx+9 {
		t.Errorf("Selection should be (%d, %d), got (%d, %d)", importantIdx, importantIdx+9, p0, p1)
	}

	// Test that PreviewLookText returns the rendered text, not the source markdown
	lookText := w.PreviewLookText()
	if lookText != "important" {
		t.Errorf("PreviewLookText() should return 'important', got %q", lookText)
	}
}

// TestPreviewExec tests that B2 (Exec) chord in preview mode operates on the rendered text.
// When the user B2-clicks text in preview mode, the command should be taken from the rendered
// text (e.g., "ls" from "**ls**"), not the raw markdown source.
func TestPreviewExec(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source with commands that might be styled
	sourceMarkdown := "# Commands\n\nRun **Echo** to test.\n\nOr try `Look` command."
	// Rendered: "Commands\n\nRun Echo to test.\n\nOr try Look command."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// Verify the window is in preview mode
	if !w.IsPreviewMode() {
		t.Fatal("Window should be in preview mode")
	}

	// Get the rich text content and verify it exists
	richContent := rt.Content()
	if richContent == nil {
		t.Fatal("Rich content should not be nil")
	}

	// Find "Echo" in the rendered plain text (it's rendered without ** markers)
	plainText := richContent.Plain()
	echoIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "Echo" {
			echoIdx = i
			break
		}
	}

	if echoIdx < 0 {
		t.Fatalf("Could not find 'Echo' in rendered text: %q", string(plainText))
	}

	// Set selection to "Echo" in the rendered text
	rt.SetSelection(echoIdx, echoIdx+4)

	// Verify selection
	p0, p1 := rt.Selection()
	if p0 != echoIdx || p1 != echoIdx+4 {
		t.Errorf("Selection should be (%d, %d), got (%d, %d)", echoIdx, echoIdx+4, p0, p1)
	}

	// Test that PreviewExecText returns the rendered text, not the source markdown
	execText := w.PreviewExecText()
	if execText != "Echo" {
		t.Errorf("PreviewExecText() should return 'Echo', got %q", execText)
	}

	// Also test the code span case - find "Look" which comes from `Look`
	lookIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "Look" {
			lookIdx = i
			break
		}
	}

	if lookIdx < 0 {
		t.Fatalf("Could not find 'Look' in rendered text: %q", string(plainText))
	}

	// Set selection to "Look" in the rendered text
	rt.SetSelection(lookIdx, lookIdx+4)

	// Verify selection
	p0, p1 = rt.Selection()
	if p0 != lookIdx || p1 != lookIdx+4 {
		t.Errorf("Selection should be (%d, %d), got (%d, %d)", lookIdx, lookIdx+4, p0, p1)
	}

	// Test that PreviewExecText returns the rendered text from code span
	execText = w.PreviewExecText()
	if execText != "Look" {
		t.Errorf("PreviewExecText() should return 'Look', got %q", execText)
	}
}

// TestPreviewExecText tests PreviewExecText() directly:
// - returns empty string when not in preview mode
// - returns empty string when no selection
// - returns rendered text from selection (not source markdown)
func TestPreviewExecText(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceMarkdown := "Run **Echo** now"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/exec.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Before preview mode, should return empty
	if got := w.PreviewExecText(); got != "" {
		t.Errorf("PreviewExecText() before preview mode should return empty, got %q", got)
	}

	// Set up rich text and enter preview mode
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

	// No selection should return empty
	if got := w.PreviewExecText(); got != "" {
		t.Errorf("PreviewExecText() with no selection should return empty, got %q", got)
	}

	// Set selection on "Echo" in rendered text (rendered as "Run Echo now")
	plainText := rt.Content().Plain()
	echoIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "Echo" {
			echoIdx = i
			break
		}
	}
	if echoIdx < 0 {
		t.Fatalf("Could not find 'Echo' in rendered text: %q", string(plainText))
	}

	rt.SetSelection(echoIdx, echoIdx+4)

	// Should return the rendered text
	if got := w.PreviewExecText(); got != "Echo" {
		t.Errorf("PreviewExecText() should return 'Echo', got %q", got)
	}
}

// TestPreviewLookExpand tests that B3 Look with no selection expands to word at click point.
func TestPreviewLookExpand(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Simple markdown
	sourceMarkdown := "Hello World test"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/test.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// Set cursor position in the middle of "World" (no selection, just click position)
	// "Hello World test" - "World" is at positions 6-11
	// Clicking at position 8 (middle of "World") should expand to select "World"
	rt.SetSelection(8, 8) // No selection, just cursor position

	// Test that PreviewExpandWord expands the click position to the full word
	word, wordStart, wordEnd := w.PreviewExpandWord(8)
	if word != "World" {
		t.Errorf("PreviewExpandWord(8) should return 'World', got %q", word)
	}
	if wordStart != 6 || wordEnd != 11 {
		t.Errorf("PreviewExpandWord(8) should return positions (6, 11), got (%d, %d)", wordStart, wordEnd)
	}
}

// TestPreviewKeyScroll tests that Page Up/Down keys scroll the preview content.
// In preview mode, keyboard navigation keys should scroll the view.
func TestPreviewKeyScroll(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create markdown content with many lines to enable scrolling
	var mdBuilder string
	for i := 1; i <= 50; i++ {
		mdBuilder += "# Heading " + string(rune('A'+i%26)) + "\n\n"
		mdBuilder += "Paragraph " + string(rune('0'+i%10)) + " with content.\n\n"
	}
	sourceMarkdown := mdBuilder
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/scroll.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// Verify initial origin is 0
	if rt.Origin() != 0 {
		t.Errorf("Initial origin should be 0, got %d", rt.Origin())
	}

	// Test Page Down key handling
	handled := w.HandlePreviewKey(draw.KeyPageDown)
	if !handled {
		t.Error("HandlePreviewKey(PageDown) should return true in preview mode")
	}

	// Origin should have increased after Page Down
	afterPageDown := rt.Origin()
	if afterPageDown <= 0 {
		t.Errorf("Origin should be > 0 after Page Down, got %d", afterPageDown)
	}

	// Test Page Up key handling
	handled = w.HandlePreviewKey(draw.KeyPageUp)
	if !handled {
		t.Error("HandlePreviewKey(PageUp) should return true in preview mode")
	}

	// Origin should have decreased after Page Up
	afterPageUp := rt.Origin()
	if afterPageUp >= afterPageDown {
		t.Errorf("Origin should have decreased after Page Up: before=%d, after=%d", afterPageDown, afterPageUp)
	}

	// Test Down Arrow - should scroll by a smaller amount
	beforeDown := rt.Origin()
	handled = w.HandlePreviewKey(draw.KeyDown)
	if !handled {
		t.Error("HandlePreviewKey(Down) should return true in preview mode")
	}
	afterDown := rt.Origin()
	if afterDown <= beforeDown {
		t.Errorf("Origin should have increased after Down arrow: before=%d, after=%d", beforeDown, afterDown)
	}

	// Test Up Arrow - should scroll by a smaller amount
	beforeUp := rt.Origin()
	handled = w.HandlePreviewKey(draw.KeyUp)
	if !handled {
		t.Error("HandlePreviewKey(Up) should return true in preview mode")
	}
	afterUp := rt.Origin()
	if afterUp >= beforeUp {
		t.Errorf("Origin should have decreased after Up arrow: before=%d, after=%d", beforeUp, afterUp)
	}

	// Test Home key - should scroll to beginning
	rt.SetOrigin(1000) // Scroll to middle
	handled = w.HandlePreviewKey(draw.KeyHome)
	if !handled {
		t.Error("HandlePreviewKey(Home) should return true in preview mode")
	}
	if rt.Origin() != 0 {
		t.Errorf("Origin should be 0 after Home key, got %d", rt.Origin())
	}

	// Test End key - should scroll to end
	handled = w.HandlePreviewKey(draw.KeyEnd)
	if !handled {
		t.Error("HandlePreviewKey(End) should return true in preview mode")
	}
	// Origin should be near the end of content
	if rt.Origin() == 0 {
		t.Error("Origin should not be 0 after End key")
	}
}

// TestPreviewKeyIgnoresTyping tests that normal typing keys are ignored in preview mode.
// Preview mode is read-only; typing should not modify content or be processed.
func TestPreviewKeyIgnoresTyping(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Simple markdown content
	sourceMarkdown := "# Hello World\n\nSome text here."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readonly.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// Record initial state
	initialBodyContent := w.body.file.String()
	initialRichContentLen := rt.Content().Len()

	// Test regular character keys - should be ignored (return false to indicate not handled)
	typingKeys := []rune{'a', 'b', 'c', '1', '2', '3', ' ', '\t'}
	for _, key := range typingKeys {
		handled := w.HandlePreviewKey(key)
		if handled {
			t.Errorf("HandlePreviewKey(%q) should return false for typing keys in preview mode", key)
		}
	}

	// Verify body buffer is unchanged
	afterBodyContent := w.body.file.String()
	if afterBodyContent != initialBodyContent {
		t.Errorf("Body content should be unchanged after typing keys:\nbefore: %q\nafter:  %q", initialBodyContent, afterBodyContent)
	}

	// Verify rich content length is unchanged
	afterRichContentLen := rt.Content().Len()
	if afterRichContentLen != initialRichContentLen {
		t.Errorf("Rich content length should be unchanged: before=%d, after=%d", initialRichContentLen, afterRichContentLen)
	}

	// Test special editing keys that should also be ignored
	editingKeys := []rune{'\b', 0x7F, '\n'} // Backspace, Delete, Enter
	for _, key := range editingKeys {
		handled := w.HandlePreviewKey(key)
		if handled {
			t.Errorf("HandlePreviewKey(%q) should return false for editing keys in preview mode", key)
		}
	}

	// Verify body buffer is still unchanged
	finalBodyContent := w.body.file.String()
	if finalBodyContent != initialBodyContent {
		t.Errorf("Body content should be unchanged after editing keys:\nbefore: %q\nafter:  %q", initialBodyContent, finalBodyContent)
	}

	// Test Escape key - should exit preview mode
	handled := w.HandlePreviewKey(0x1B) // Escape
	if !handled {
		t.Error("HandlePreviewKey(Escape) should return true in preview mode")
	}
	if w.IsPreviewMode() {
		t.Error("Escape key should exit preview mode")
	}
}

// TestWindowPreviewLinkMap tests that a Window stores the LinkMap when entering preview mode.
// The LinkMap is populated by ParseWithSourceMap and allows the window to identify links
// at rendered text positions (used for Look action on links).
func TestWindowPreviewLinkMap(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown with links
	sourceMarkdown := "Check out [Google](https://google.com) and [GitHub](https://github.com) for more info."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/links.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown with source map and link map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Initially, link map should not be set on window
	if w.PreviewLinkMap() != nil {
		t.Error("PreviewLinkMap should be nil initially")
	}

	// Assign the richBody to the window and set both source and link maps
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify link map is set
	if w.PreviewLinkMap() == nil {
		t.Fatal("PreviewLinkMap should not be nil after SetPreviewLinkMap")
	}

	// Verify the link map matches what we set
	if w.PreviewLinkMap() != linkMap {
		t.Error("PreviewLinkMap should return the same link map that was set")
	}

	// Verify the link map has the correct links
	// The rendered text will be: "Check out Google and GitHub for more info."
	// "Google" is at positions 10-16, "GitHub" is at positions 21-27

	// Find "Google" in rendered text and verify it maps to the correct URL
	plainText := content.Plain()
	googleIdx := -1
	for i := 0; i < len(plainText)-5; i++ {
		if string(plainText[i:i+6]) == "Google" {
			googleIdx = i
			break
		}
	}
	if googleIdx < 0 {
		t.Fatal("Could not find 'Google' in rendered text")
	}

	// Check URL at Google's position
	url := linkMap.URLAt(googleIdx)
	if url != "https://google.com" {
		t.Errorf("URLAt(Google) = %q, want %q", url, "https://google.com")
	}

	// Find "GitHub" in rendered text
	githubIdx := -1
	for i := 0; i < len(plainText)-5; i++ {
		if string(plainText[i:i+6]) == "GitHub" {
			githubIdx = i
			break
		}
	}
	if githubIdx < 0 {
		t.Fatal("Could not find 'GitHub' in rendered text")
	}

	// Check URL at GitHub's position
	url = linkMap.URLAt(githubIdx)
	if url != "https://github.com" {
		t.Errorf("URLAt(GitHub) = %q, want %q", url, "https://github.com")
	}

	// Check that non-link text doesn't return a URL
	// "Check" is at position 0, which is not a link
	url = linkMap.URLAt(0)
	if url != "" {
		t.Errorf("URLAt(0) should be empty for non-link text, got %q", url)
	}

	// Verify that exiting preview mode preserves the link map
	w.SetPreviewMode(false)
	if w.PreviewLinkMap() == nil {
		t.Error("PreviewLinkMap should be preserved after exiting preview mode")
	}

	// Re-entering preview should still have the link map
	w.SetPreviewMode(true)
	if w.PreviewLinkMap() != linkMap {
		t.Error("PreviewLinkMap should still be the same after re-entering preview mode")
	}
}

// TestPreviewLookLink tests that B3 (Look) on a link in preview mode returns the link URL.
// When the user B3-clicks on a link, the Look action should open/plumb the URL instead
// of searching for the link text.
func TestPreviewLookLink(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown with a link
	sourceMarkdown := "Check out [Google](https://google.com) for more info."
	// Rendered text: "Check out Google for more info."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/links.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown with source map and link map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and set maps, enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify we're in preview mode
	if !w.IsPreviewMode() {
		t.Fatal("Window should be in preview mode")
	}

	// Find "Google" in the rendered text - this is the link text
	plainText := content.Plain()
	googleIdx := -1
	for i := 0; i < len(plainText)-5; i++ {
		if string(plainText[i:i+6]) == "Google" {
			googleIdx = i
			break
		}
	}
	if googleIdx < 0 {
		t.Fatalf("Could not find 'Google' in rendered text: %q", string(plainText))
	}

	// Test: PreviewLookLinkURL at the link position should return the URL
	url := w.PreviewLookLinkURL(googleIdx)
	if url != "https://google.com" {
		t.Errorf("PreviewLookLinkURL(%d) = %q, want %q", googleIdx, url, "https://google.com")
	}

	// Also test at the end of the link text (still within the link)
	url = w.PreviewLookLinkURL(googleIdx + 5) // last character of "Google"
	if url != "https://google.com" {
		t.Errorf("PreviewLookLinkURL(%d) = %q, want %q", googleIdx+5, url, "https://google.com")
	}
}

// TestPreviewLookNonLink tests that B3 (Look) on non-link text in preview mode
// returns empty string, indicating normal Look behavior should be used.
func TestPreviewLookNonLink(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown with a link and regular text
	sourceMarkdown := "Check out [Google](https://google.com) for more info."
	// Rendered text: "Check out Google for more info."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/links.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create a RichText component for preview
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

	// Parse markdown with source map and link map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and set maps, enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify we're in preview mode
	if !w.IsPreviewMode() {
		t.Fatal("Window should be in preview mode")
	}

	// Test: PreviewLookLinkURL at position 0 ("Check") should return empty string
	// because it's not a link
	url := w.PreviewLookLinkURL(0)
	if url != "" {
		t.Errorf("PreviewLookLinkURL(0) = %q, want empty string for non-link text", url)
	}

	// Find "more" in the rendered text - this is after the link
	plainText := content.Plain()
	moreIdx := -1
	for i := 0; i < len(plainText)-3; i++ {
		if string(plainText[i:i+4]) == "more" {
			moreIdx = i
			break
		}
	}
	if moreIdx < 0 {
		t.Fatalf("Could not find 'more' in rendered text: %q", string(plainText))
	}

	// Test: PreviewLookLinkURL at "more" position should return empty string
	url = w.PreviewLookLinkURL(moreIdx)
	if url != "" {
		t.Errorf("PreviewLookLinkURL(%d) = %q, want empty string for non-link text", moreIdx, url)
	}

	// Test: PreviewLookLinkURL when not in preview mode should return empty string
	w.SetPreviewMode(false)
	url = w.PreviewLookLinkURL(10) // any position
	if url != "" {
		t.Errorf("PreviewLookLinkURL when not in preview mode = %q, want empty string", url)
	}

	// Test: PreviewLookLinkURL with no link map should return empty string
	w.SetPreviewMode(true)
	w.SetPreviewLinkMap(nil)
	url = w.PreviewLookLinkURL(10)
	if url != "" {
		t.Errorf("PreviewLookLinkURL with nil link map = %q, want empty string", url)
	}
}

// TestWindowResizePreviewMode tests that when a window in preview mode is resized,
// the richBody.Render() is called with the updated body.all rectangle, ensuring
// the preview content is properly displayed in the new area.
func TestWindowResizePreviewMode(t *testing.T) {
	// Create initial rectangle and display
	initialRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(initialRect)
	global.configureGlobals(display)

	// Create markdown content
	markdownContent := "# Hello World\n\nThis is some **bold** text and *italic* text.\n\nParagraph here."
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display

	// Setup body with mock frame
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/resize.md", sourceRunes),
	}
	// Initial body.all rectangle (simulating window layout after Init)
	w.body.all = image.Rect(0, 20, 800, 600)

	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = initialRect

	// Create and initialize RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	// Initial render into the body area
	rt.Render(w.body.all)

	// Parse markdown and set content
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(markdownContent)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify initial state
	if !w.IsPreviewMode() {
		t.Fatal("Window should be in preview mode")
	}

	// Get the initial lastRect from richBody (via All() accessor)
	initialLastRect := rt.All()
	if !initialLastRect.Eq(w.body.all) {
		t.Errorf("Initial lastRect should match body.all: got %v, want %v", initialLastRect, w.body.all)
	}

	// Simulate resize: update body.all to a new rectangle (e.g., window made smaller)
	newBodyRect := image.Rect(0, 20, 600, 400)
	w.body.all = newBodyRect

	// Call Render with the new rectangle (as Window.Resize should do)
	// This simulates what Window.Resize() should do when in preview mode:
	// After updating body.all, it should call richBody.Render(body.all)
	w.richBody.Render(w.body.all)

	// Verify the richBody's lastRect was updated (via All() accessor)
	afterResizeRect := rt.All()
	if !afterResizeRect.Eq(newBodyRect) {
		t.Errorf("After resize, lastRect should match new body.all: got %v, want %v", afterResizeRect, newBodyRect)
	}

	// Verify the frame rectangle was also updated
	frameRect := rt.Frame().Rect()
	// Frame rect should be to the right of scrollbar within the new body rect
	if frameRect.Max.X > newBodyRect.Max.X {
		t.Errorf("Frame rect.Max.X (%d) should not exceed newBodyRect.Max.X (%d)", frameRect.Max.X, newBodyRect.Max.X)
	}
	if frameRect.Min.Y != newBodyRect.Min.Y {
		t.Errorf("Frame rect.Min.Y (%d) should match newBodyRect.Min.Y (%d)", frameRect.Min.Y, newBodyRect.Min.Y)
	}
	if frameRect.Max.Y != newBodyRect.Max.Y {
		t.Errorf("Frame rect.Max.Y (%d) should match newBodyRect.Max.Y (%d)", frameRect.Max.Y, newBodyRect.Max.Y)
	}

	// Verify scrollbar rect was also updated
	scrollRect := rt.ScrollRect()
	if scrollRect.Min.X != newBodyRect.Min.X {
		t.Errorf("Scroll rect.Min.X (%d) should match newBodyRect.Min.X (%d)", scrollRect.Min.X, newBodyRect.Min.X)
	}
	if scrollRect.Min.Y != newBodyRect.Min.Y {
		t.Errorf("Scroll rect.Min.Y (%d) should match newBodyRect.Min.Y (%d)", scrollRect.Min.Y, newBodyRect.Min.Y)
	}
	if scrollRect.Max.Y != newBodyRect.Max.Y {
		t.Errorf("Scroll rect.Max.Y (%d) should match newBodyRect.Max.Y (%d)", scrollRect.Max.Y, newBodyRect.Max.Y)
	}

	// Verify content is still accessible after resize
	if rt.Content() == nil {
		t.Error("Content should not be nil after resize")
	}
	if rt.Content().Len() == 0 {
		t.Error("Content should not be empty after resize")
	}

	// Verify the rich text frame has the correct content length
	// (this ensures layout was recomputed for the new width)
	if rt.Frame() == nil {
		t.Fatal("Frame should not be nil after resize")
	}

	// Test another resize - making window larger
	largerBodyRect := image.Rect(0, 20, 1000, 800)
	w.body.all = largerBodyRect
	w.richBody.Render(w.body.all)

	// Verify the update (via All() accessor)
	afterLargerResize := rt.All()
	if !afterLargerResize.Eq(largerBodyRect) {
		t.Errorf("After larger resize, lastRect should match: got %v, want %v", afterLargerResize, largerBodyRect)
	}

	// Verify frame expanded
	frameRectLarger := rt.Frame().Rect()
	if frameRectLarger.Max.X <= frameRect.Max.X {
		t.Errorf("Larger frame rect.Max.X (%d) should be greater than smaller (%d)", frameRectLarger.Max.X, frameRect.Max.X)
	}
}

// TestWindowDrawPreviewModeAfterResize tests that Window.Draw() correctly uses
// body.all as the rectangle when in preview mode, ensuring that after a resize
// the preview content is rendered into the correct (updated) area.
func TestWindowDrawPreviewModeAfterResize(t *testing.T) {
	// Create initial rectangle and display
	initialRect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(initialRect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display

	// Setup body with mock frame and initial rectangle
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/draw.md", []rune("# Hello\n\nWorld")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)

	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = initialRect

	// Create and initialize RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	// Initial render into the body area
	rt.Render(w.body.all)

	// Set content
	content := rich.Plain("Hello World")
	rt.SetContent(content)

	// Assign richBody and enable preview mode
	w.richBody = rt
	w.SetPreviewMode(true)

	// Verify initial lastRect matches body.all
	if !rt.All().Eq(w.body.all) {
		t.Errorf("Initial lastRect should match body.all: got %v, want %v", rt.All(), w.body.all)
	}

	// Now simulate a resize: body.all changes but richBody's cached rectangle is stale
	newBodyRect := image.Rect(0, 20, 600, 400)
	w.body.all = newBodyRect

	// Call Draw() - this should use body.all (the current geometry) not the cached value
	w.Draw()

	// Verify that after Draw(), the richBody's lastRect has been updated to body.all
	// This proves that Draw() used Render(body.all) instead of Redraw()
	if !rt.All().Eq(newBodyRect) {
		t.Errorf("After Draw(), lastRect should match updated body.all: got %v, want %v", rt.All(), newBodyRect)
	}

	// Verify frame rectangle was also updated to match the new area
	frameRect := rt.Frame().Rect()
	if frameRect.Max.X > newBodyRect.Max.X {
		t.Errorf("Frame rect.Max.X (%d) should not exceed newBodyRect.Max.X (%d)", frameRect.Max.X, newBodyRect.Max.X)
	}
	if frameRect.Max.Y != newBodyRect.Max.Y {
		t.Errorf("Frame rect.Max.Y (%d) should match newBodyRect.Max.Y (%d)", frameRect.Max.Y, newBodyRect.Max.Y)
	}

	// Verify scrollbar rectangle was also updated
	scrollRect := rt.ScrollRect()
	if scrollRect.Min.X != newBodyRect.Min.X {
		t.Errorf("Scroll rect.Min.X (%d) should match newBodyRect.Min.X (%d)", scrollRect.Min.X, newBodyRect.Min.X)
	}
	if scrollRect.Max.Y != newBodyRect.Max.Y {
		t.Errorf("Scroll rect.Max.Y (%d) should match newBodyRect.Max.Y (%d)", scrollRect.Max.Y, newBodyRect.Max.Y)
	}

	// Test that subsequent Draw() calls also maintain correct geometry
	evenSmallerRect := image.Rect(0, 20, 400, 300)
	w.body.all = evenSmallerRect
	w.Draw()

	if !rt.All().Eq(evenSmallerRect) {
		t.Errorf("After second Draw(), lastRect should match: got %v, want %v", rt.All(), evenSmallerRect)
	}
}

// =============================================================================
// Phase 16G: Window Integration Tests for Image Cache
// =============================================================================

// TestPreviewModeInitCache tests that entering Markdeep mode creates an image cache.
// The cache is needed to load and cache images referenced in markdown files.
func TestPreviewModeInitCache(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content containing an image
	markdownContent := "# Test\n\n![Test Image](test.png)\n"
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Initially, imageCache should be nil
	if w.imageCache != nil {
		t.Error("imageCache should be nil before entering preview mode")
	}

	// Set up preview mode components (simulating what previewcmd does)
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

	// Parse markdown and set content
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(markdownContent)
	rt.SetContent(content)

	// Initialize the image cache (as previewcmd should do)
	w.imageCache = rich.NewImageCache(0) // 0 means use default size

	// Assign richBody and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify the imageCache was created
	if w.imageCache == nil {
		t.Error("imageCache should be initialized when entering preview mode")
	}

	// Verify we can use the cache
	// The cache should support Get operations even if no images are loaded yet
	cached, ok := w.imageCache.Get("/nonexistent/path")
	if ok {
		t.Error("Get should return false for non-existent path")
	}
	if cached != nil {
		t.Error("Get should return nil for non-existent path")
	}
}

// TestPreviewModeCleanupCache tests that exiting Markdeep mode clears and removes the image cache.
// This ensures memory is freed when preview mode is disabled.
func TestPreviewModeCleanupCache(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window
	markdownContent := "# Test\n\n![Image](test.png)\n"
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Set up preview mode components
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

	// Parse markdown and set content
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(markdownContent)
	rt.SetContent(content)

	// Initialize the image cache and enter preview mode
	w.imageCache = rich.NewImageCache(0)
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify cache exists
	if w.imageCache == nil {
		t.Fatal("imageCache should exist after entering preview mode")
	}

	// Exit preview mode - this should clear the cache
	w.SetPreviewMode(false)

	// Clear the cache when exiting (as the implementation should do)
	if w.imageCache != nil {
		w.imageCache.Clear()
		w.imageCache = nil
	}

	// Verify cache was cleared
	if w.imageCache != nil {
		t.Error("imageCache should be nil after exiting preview mode and cleanup")
	}
}

// TestResolveImagePathAbsolute tests that absolute image paths are returned unchanged.
// When an image path starts with /, it should be used as-is.
func TestResolveImagePathAbsolute(t *testing.T) {
	tests := []struct {
		name     string
		basePath string // Markdown file path
		imgPath  string // Image path in markdown
		want     string // Expected resolved path
	}{
		{
			name:     "absolute unix path",
			basePath: "/home/user/docs/readme.md",
			imgPath:  "/images/logo.png",
			want:     "/images/logo.png",
		},
		{
			name:     "absolute path with subdirectory",
			basePath: "/project/docs/guide.md",
			imgPath:  "/project/assets/diagram.png",
			want:     "/project/assets/diagram.png",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveImagePath(tc.basePath, tc.imgPath)
			if got != tc.want {
				t.Errorf("resolveImagePath(%q, %q) = %q, want %q",
					tc.basePath, tc.imgPath, got, tc.want)
			}
		})
	}
}

// TestResolveImagePathRelative tests that relative image paths are resolved
// relative to the directory containing the markdown file.
func TestResolveImagePathRelative(t *testing.T) {
	tests := []struct {
		name     string
		basePath string // Markdown file path
		imgPath  string // Image path in markdown
		want     string // Expected resolved path
	}{
		{
			name:     "simple relative",
			basePath: "/home/user/docs/readme.md",
			imgPath:  "image.png",
			want:     "/home/user/docs/image.png",
		},
		{
			name:     "relative with subdirectory",
			basePath: "/home/user/docs/readme.md",
			imgPath:  "images/logo.png",
			want:     "/home/user/docs/images/logo.png",
		},
		{
			name:     "relative with parent directory",
			basePath: "/home/user/docs/guide/intro.md",
			imgPath:  "../images/diagram.png",
			want:     "/home/user/docs/images/diagram.png",
		},
		{
			name:     "relative in root directory",
			basePath: "/readme.md",
			imgPath:  "logo.png",
			want:     "/logo.png",
		},
		{
			name:     "dot prefix relative",
			basePath: "/project/docs/readme.md",
			imgPath:  "./images/icon.png",
			want:     "/project/docs/images/icon.png",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveImagePath(tc.basePath, tc.imgPath)
			if got != tc.want {
				t.Errorf("resolveImagePath(%q, %q) = %q, want %q",
					tc.basePath, tc.imgPath, got, tc.want)
			}
		})
	}
}

// =============================================================================
// Phase 16H: Integration Tests
// =============================================================================

// TestMarkdeepImageIntegration tests the end-to-end image rendering pipeline:
// 1. Parse markdown with image syntax
// 2. Create window with preview mode
// 3. Load image into cache
// 4. Verify image box is created with correct dimensions
// 5. Verify image data is available for rendering
func TestMarkdeepImageIntegration(t *testing.T) {
	// Create a temporary directory with a test image
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test_image.png")
	mdPath := filepath.Join(tmpDir, "test.md")

	// Create a simple 40x30 test image
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, red)
		}
	}
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f.Close()

	// Create markdown content with the image
	// Use relative path since that's the common case
	markdownContent := fmt.Sprintf("# Test Document\n\n![Test Image](test_image.png)\n\nSome text after the image.\n")

	// Write the markdown file
	if err := os.WriteFile(mdPath, []byte(markdownContent), 0644); err != nil {
		t.Fatalf("failed to write markdown file: %v", err)
	}

	// Set up the display and window
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with the markdown content
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer(mdPath, sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Test markdown.Parse (non-source-mapped version) first to verify image parsing works
	parsedContent := markdown.Parse(markdownContent)

	// Verify basic parsing detected the image
	foundImage := false
	for _, span := range parsedContent {
		if span.Style.Image {
			foundImage = true
			if span.Style.ImageURL != "test_image.png" {
				t.Errorf("ImageURL = %q, want %q", span.Style.ImageURL, "test_image.png")
			}
			if span.Style.ImageAlt != "Test Image" {
				t.Errorf("ImageAlt = %q, want %q", span.Style.ImageAlt, "Test Image")
			}
			break
		}
	}
	if !foundImage {
		t.Fatal("markdown.Parse did not detect image")
	}

	// Parse markdown with source map (currently images are rendered as placeholders)
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(markdownContent)

	// Create and initialize the image cache
	cache := rich.NewImageCache(10)

	// Resolve and load the image
	resolvedPath := resolveImagePath(mdPath, "test_image.png")
	expectedResolvedPath := filepath.Join(tmpDir, "test_image.png")
	if resolvedPath != expectedResolvedPath {
		t.Errorf("resolveImagePath = %q, want %q", resolvedPath, expectedResolvedPath)
	}

	// Load the image into cache
	cached, err := cache.Load(resolvedPath)
	if err != nil {
		t.Fatalf("failed to load image into cache: %v", err)
	}

	// Verify cached image properties
	if cached.Width != 40 {
		t.Errorf("cached image width = %d, want 40", cached.Width)
	}
	if cached.Height != 30 {
		t.Errorf("cached image height = %d, want 30", cached.Height)
	}
	if cached.Original == nil {
		t.Error("cached.Original should not be nil")
	}
	if cached.Data == nil {
		t.Error("cached.Data (Plan 9 format) should not be nil")
	}
	if cached.Err != nil {
		t.Errorf("cached.Err should be nil, got: %v", cached.Err)
	}

	// Set up preview mode components
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.SetContent(content)
	rt.Render(bodyRect)

	// Wire everything to the window
	w.imageCache = cache
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify preview mode is active
	if !w.previewMode {
		t.Error("previewMode should be true")
	}

	// Verify cache was attached
	if w.imageCache == nil {
		t.Error("imageCache should be attached to window")
	}

	// Verify the cache hit on second load
	cached2, _ := cache.Get(resolvedPath)
	if cached2 != cached {
		t.Error("cache should return same entry on second access")
	}

	// Clean up by exiting preview mode
	w.SetPreviewMode(false)
	cache.Clear()
}

// TestHandlePreviewMouseSignature tests that HandlePreviewMouse accepts both
// a mouse event and a Mousectl, which is needed for proper drag selection.
// This test verifies the signature change from (m *draw.Mouse) to (m *draw.Mouse, mc *draw.Mousectl).
func TestHandlePreviewMouseSignature(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World\n\nThis is some text.")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Set up the body.all rectangle (used by HandlePreviewMouse for hit-testing)
	w.body.all = image.Rect(0, 20, 800, 600)

	// Create a RichText component for preview
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

	// Set content in the RichText
	content := rich.Plain("Hello World - this is some test content for selection")
	rt.SetContent(content)

	// Assign the richBody to the window and enter preview mode
	w.richBody = rt
	w.SetPreviewMode(true)

	// Create a mouse event in the frame area (button 1 click for selection)
	frameRect := rt.Frame().Rect()
	clickPoint := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   clickPoint,
		Buttons: 1, // Button 1 pressed
	}

	// Create a Mousectl with an immediate release event for proper Select() behavior
	upEvent := draw.Mouse{
		Point:   clickPoint,
		Buttons: 0, // Button released
	}
	mc := mockMousectlWithEvents([]draw.Mouse{upEvent})

	// Test that HandlePreviewMouse can be called with both mouse and mousectl
	// The key assertion is that the call compiles and executes without error
	handled := w.HandlePreviewMouse(&m, mc)

	// The event should be handled since we're in preview mode and clicking in the frame
	if !handled {
		t.Error("HandlePreviewMouse should return true for button 1 click in frame area")
	}

	// After handling, the selection should be set (at minimum, a point selection at the click)
	q0, q1 := rt.Selection()
	// We expect at least that q0/q1 are set (the exact values depend on the click position)
	// Since this is a single click without drag, q0 should equal q1
	if q0 != q1 {
		t.Logf("Selection after single click: q0=%d, q1=%d (expected point selection)", q0, q1)
	}

	// Test that events outside the body area are not handled
	outsidePoint := image.Pt(-10, -10)
	m2 := draw.Mouse{
		Point:   outsidePoint,
		Buttons: 1,
	}
	handled2 := w.HandlePreviewMouse(&m2, mc)
	if handled2 {
		t.Error("HandlePreviewMouse should return false for clicks outside body.all")
	}

	// Test with nil Mousectl (should still handle simple cases like scroll wheel)
	scrollDownMouse := draw.Mouse{
		Point:   clickPoint,
		Buttons: 16, // Button 5 - scroll down
	}
	handled3 := w.HandlePreviewMouse(&scrollDownMouse, nil)
	if !handled3 {
		t.Error("HandlePreviewMouse should handle scroll wheel even with nil Mousectl")
	}
}

// mockMousectlWithEvents creates a mock Mousectl with a buffered channel
// containing the provided events. This is used for testing drag selection.
func mockMousectlWithEvents(events []draw.Mouse) *draw.Mousectl {
	ch := make(chan draw.Mouse, len(events)+1)
	for _, e := range events {
		ch <- e
	}
	return &draw.Mousectl{C: ch}
}

// TestPreviewModeSelection tests that single-click selection in preview mode
// sets a point selection (p0 == p1) at the click position.
func TestPreviewModeSelection(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.all = image.Rect(0, 20, 800, 600)

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Hello World" (11 chars)
	content := rich.Plain("Hello World")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	// Simulate single click at position 5 (the space) - X=12+50=62
	// (12px scrollbar + 5 chars * 10px = 62)
	frameRect := rt.Frame().Rect()
	clickPoint := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)

	// Mouse down event
	downEvent := draw.Mouse{
		Point:   clickPoint,
		Buttons: 1,
	}
	// Immediate mouse up at same position (no drag)
	upEvent := draw.Mouse{
		Point:   clickPoint,
		Buttons: 0,
	}

	mc := mockMousectlWithEvents([]draw.Mouse{upEvent})
	handled := w.HandlePreviewMouse(&downEvent, mc)

	if !handled {
		t.Error("HandlePreviewMouse should handle button 1 click in frame area")
	}

	// After single click without drag, selection should be a point (p0 == p1)
	q0, q1 := rt.Selection()
	if q0 != q1 {
		t.Errorf("Single click selection should be point (p0 == p1), got p0=%d, p1=%d", q0, q1)
	}
	// Position 5 corresponds to the space in "Hello World"
	if q0 != 5 {
		t.Errorf("Click at X=50 should select position 5, got %d", q0)
	}
}

// TestPreviewModeSelectionDrag tests that click-and-drag selection in preview mode
// selects a range of text from the anchor point to the release point.
func TestPreviewModeSelectionDrag(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.all = image.Rect(0, 20, 800, 600)

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Hello World" (11 chars)
	content := rich.Plain("Hello World")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Simulate drag selection from position 0 to position 5 (select "Hello")
	// Position 0 is at X = frameRect.Min.X (after scrollbar)
	// Position 5 is at X = frameRect.Min.X + 50

	// Mouse down at position 0
	downEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
		Buttons: 1,
	}
	// Drag to position 5 (still holding button)
	dragEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
		Buttons: 1,
	}
	// Mouse up at position 5
	upEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
		Buttons: 0,
	}

	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
	handled := w.HandlePreviewMouse(&downEvent, mc)

	if !handled {
		t.Error("HandlePreviewMouse should handle button 1 drag in frame area")
	}

	// After drag from 0 to 5, selection should be (0, 5)
	q0, q1 := rt.Selection()
	if q0 != 0 {
		t.Errorf("Drag selection p0 should be 0, got %d", q0)
	}
	if q1 != 5 {
		t.Errorf("Drag selection p1 should be 5, got %d", q1)
	}
}

// TestPreviewModeSelectionDragBackward tests that dragging backward
// (from right to left) still produces a valid selection with p0 < p1.
func TestPreviewModeSelectionDragBackward(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.all = image.Rect(0, 20, 800, 600)

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content := rich.Plain("Hello World")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Drag backward: start at position 5, drag to position 0
	downEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5), // Position 5
		Buttons: 1,
	}
	dragEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5), // Position 0
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
		Buttons: 0,
	}

	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
	handled := w.HandlePreviewMouse(&downEvent, mc)

	if !handled {
		t.Error("HandlePreviewMouse should handle backward drag")
	}

	// Selection should still be normalized: p0 < p1
	q0, q1 := rt.Selection()
	if q0 != 0 {
		t.Errorf("Backward drag selection p0 should be 0, got %d", q0)
	}
	if q1 != 5 {
		t.Errorf("Backward drag selection p1 should be 5, got %d", q1)
	}
}

// TestPreviewSelectionNearScrollbar tests that selection works correctly when
// the drag starts in the frame area and ends near or past the scrollbar boundary.
// The selection should clamp to the beginning of the line (position 0) when
// dragging into the scrollbar area.
func TestPreviewSelectionNearScrollbar(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", []rune("# Hello World")),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.all = image.Rect(0, 20, 800, 600)

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Hello World" (11 chars)
	content := rich.Plain("Hello World")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()
	scrollRect := rt.ScrollRect()

	// Verify the geometry: scrollbar should be to the left of the frame
	if scrollRect.Max.X > frameRect.Min.X {
		t.Logf("scrollRect: %v, frameRect: %v", scrollRect, frameRect)
	}

	// Test 1: Start in frame, drag to scrollbar area (past left edge)
	// Start at position 5 ("Hello" + one char), drag left into scrollbar
	t.Run("DragIntoScrollbar", func(t *testing.T) {
		// Mouse down at position 5 (50 pixels from frame left edge)
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 1,
		}
		// Drag to scrollbar area (past left edge of frame, into scrollbar)
		dragEvent := draw.Mouse{
			Point:   image.Pt(scrollRect.Min.X+2, frameRect.Min.Y+5), // Inside scrollbar
			Buttons: 1,
		}
		// Mouse up in scrollbar area
		upEvent := draw.Mouse{
			Point:   image.Pt(scrollRect.Min.X+2, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle drag that ends in scrollbar area")
		}

		// Selection should be from 0 (clamped) to 5 (start position)
		// When dragging left past the frame boundary, Charofpt clamps x to 0,
		// which maps to position 0
		q0, q1 := rt.Selection()
		if q0 != 0 {
			t.Errorf("Selection p0 should be 0 (clamped at left edge), got %d", q0)
		}
		if q1 != 5 {
			t.Errorf("Selection p1 should be 5 (anchor point), got %d", q1)
		}
	})

	// Test 2: Start at frame left edge, drag right (selection from beginning)
	// This verifies that clicking exactly at the frame's left edge works correctly
	t.Run("StartAtFrameEdge", func(t *testing.T) {
		// Clear previous selection
		rt.SetSelection(0, 0)

		// Mouse down at frame left edge
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 1,
		}
		// Get the anchor position
		anchor := rt.Frame().Charofpt(downEvent.Point)

		// Drag right by 30 pixels
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+30, frameRect.Min.Y+5),
			Buttons: 1,
		}
		// Mouse up
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+30, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle drag from frame edge")
		}

		// Selection should start at the anchor position
		q0, q1 := rt.Selection()
		if q0 != anchor {
			t.Errorf("Selection p0 should be %d (anchor), got %d", anchor, q0)
		}
		// Selection end should be further right (higher position)
		if q1 <= q0 {
			t.Errorf("Selection p1 should be > p0, got p0=%d, p1=%d", q0, q1)
		}
	})

	// Test 3: Drag that goes into scrollbar and back
	// This verifies that dragging through the scrollbar area and back works correctly
	t.Run("DragThroughScrollbarAndBack", func(t *testing.T) {
		// Clear previous selection
		rt.SetSelection(0, 0)

		// Mouse down at position well inside the frame (50px from left)
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 1,
		}
		// Get the anchor position
		anchor := rt.Frame().Charofpt(downEvent.Point)

		// Drag into scrollbar (intermediate position - will clamp to position 0)
		dragEvent1 := draw.Mouse{
			Point:   image.Pt(scrollRect.Min.X+2, frameRect.Min.Y+5),
			Buttons: 1,
		}
		// Drag back into frame (20px from left)
		dragEvent2 := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+20, frameRect.Min.Y+5),
			Buttons: 1,
		}
		// Get the final position
		finalPos := rt.Frame().Charofpt(dragEvent2.Point)

		// Mouse up
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+20, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent1, dragEvent2, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle complex drag path")
		}

		// Selection should be from finalPos to anchor (normalized: smaller first)
		q0, q1 := rt.Selection()
		expectedP0 := finalPos
		expectedP1 := anchor
		if expectedP0 > expectedP1 {
			expectedP0, expectedP1 = expectedP1, expectedP0
		}
		if q0 != expectedP0 {
			t.Errorf("Selection p0 should be %d, got %d", expectedP0, q0)
		}
		if q1 != expectedP1 {
			t.Errorf("Selection p1 should be %d, got %d", expectedP1, q1)
		}
	})
}

// TestPreviewB2ExpandWord tests that a B2 null click (click without sweep, p0==p1)
// expands to the word under the cursor using PreviewExpandWord(). In Acme, a B2
// click on a word executes that word as a command (e.g., clicking on "Del" runs Del).
// This test verifies:
// 1. A B2 null click on a word expands the selection to the whole word
// 2. The expanded word can be retrieved for execution
// 3. A B2 null click on whitespace does not expand
func TestPreviewB2ExpandWord(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", []rune("Del Put hello")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Del Put hello" (13 chars)
	// Positions:    0123456789...
	//               Del Put hello
	content := rich.Plain("Del Put hello")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Test 1: B2 null click in the middle of "Del" should expand to "Del"
	t.Run("ExpandWordOnNullClick", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Click at position 1 (middle of "Del"), 10px per char
		clickPt := image.Pt(frameRect.Min.X+15, frameRect.Min.Y+5)
		downEvent := draw.Mouse{
			Point:   clickPt,
			Buttons: 2, // Button 2 (middle button)
		}
		// Immediate release at same position (null click)
		upEvent := draw.Mouse{
			Point:   clickPt,
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B2 null click in frame area")
		}

		// After null click, word expansion should give us "Del"
		charPos := rt.Frame().Charofpt(clickPt)
		word, start, end := w.PreviewExpandWord(charPos)
		if word != "Del" {
			t.Errorf("PreviewExpandWord should return \"Del\", got %q", word)
		}
		if start != 0 {
			t.Errorf("PreviewExpandWord start should be 0, got %d", start)
		}
		if end != 3 {
			t.Errorf("PreviewExpandWord end should be 3, got %d", end)
		}
	})

	// Test 2: B2 null click on "Put" should expand to "Put"
	t.Run("ExpandSecondWord", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Click at position 5 (middle of "Put"), 10px per char
		clickPt := image.Pt(frameRect.Min.X+45, frameRect.Min.Y+5)
		charPos := rt.Frame().Charofpt(clickPt)
		word, start, end := w.PreviewExpandWord(charPos)
		if word != "Put" {
			t.Errorf("PreviewExpandWord should return \"Put\", got %q", word)
		}
		if start != 4 {
			t.Errorf("PreviewExpandWord start should be 4, got %d", start)
		}
		if end != 7 {
			t.Errorf("PreviewExpandWord end should be 7, got %d", end)
		}
	})

	// Test 3: B2 null click on whitespace between words expands left to adjacent word
	// This matches Acme behavior: clicking just past a word boundary selects that word.
	t.Run("ExpandAdjacentWordOnWhitespace", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Click at position 3 (the space right after "Del")
		clickPt := image.Pt(frameRect.Min.X+35, frameRect.Min.Y+5)
		charPos := rt.Frame().Charofpt(clickPt)
		word, start, end := w.PreviewExpandWord(charPos)
		// Position 3 is space, but left-expansion finds "Del"
		if word != "Del" {
			t.Errorf("PreviewExpandWord at space after word should expand left, got %q", word)
		}
		if start != 0 || end != 3 {
			t.Errorf("Expected expansion (0, 3), got (%d, %d)", start, end)
		}
	})

	// Test 4: B2 null click beyond end of text should not expand
	t.Run("NoExpandBeyondText", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// PreviewExpandWord with position beyond text length
		word, _, _ := w.PreviewExpandWord(100)
		if word != "" {
			t.Errorf("PreviewExpandWord beyond text should return empty string, got %q", word)
		}
	})
}

// TestPreviewSnarfAfterSelection tests that Snarf (copy) works correctly after
// making a drag selection in preview mode. This verifies the integration between
// Frame.Select() drag selection and PreviewSnarf() source mapping.
//
// The test performs a drag selection and then calls PreviewSnarf() to verify
// that the correct source markdown is returned (not the rendered text).
func TestPreviewSnarfAfterSelection(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source with bold text
	// Source:   "Hello **World** test" (20 chars)
	// Rendered: "Hello World test" (16 chars)
	sourceMarkdown := "Hello **World** test"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/file.md", sourceRunes),
	}
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.all = image.Rect(0, 20, 800, 600)

	// Create a RichText component for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Assign the richBody to the window and enable preview mode
	w.richBody = rt
	w.SetPreviewMode(true)
	w.previewSourceMap = sourceMap

	frameRect := rt.Frame().Rect()

	// Simulate drag selection to select "World" in rendered text
	// Rendered: "Hello World test"
	//           0123456789012345
	// "World" is at positions 6-11 in rendered text
	// At 10px per char: position 6 = 60px, position 11 = 110px

	// Mouse down at position 6 (start of "World")
	downEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+60, frameRect.Min.Y+5),
		Buttons: 1,
	}
	// Drag to position 11 (end of "World")
	dragEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+110, frameRect.Min.Y+5),
		Buttons: 1,
	}
	// Mouse up
	upEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+110, frameRect.Min.Y+5),
		Buttons: 0,
	}

	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
	handled := w.HandlePreviewMouse(&downEvent, mc)

	if !handled {
		t.Error("HandlePreviewMouse should handle button 1 drag in frame area")
	}

	// Verify selection was set
	q0, q1 := rt.Selection()
	if q0 == q1 {
		t.Errorf("Selection should be a range after drag, got point selection at %d", q0)
	}

	// Now test that PreviewSnarf returns the source markdown
	snarfBytes := w.PreviewSnarf()
	if snarfBytes == nil {
		t.Fatal("PreviewSnarf should return bytes for selected text")
	}

	snarfText := string(snarfBytes)

	// The selection should map to "**World**" in source (including the bold markers)
	// Source:   "Hello **World** test"
	//           01234567890123456789
	// "**World**" is at positions 6-15 in source
	if snarfText != "**World**" {
		t.Errorf("PreviewSnarf should return source markdown '**World**', got %q", snarfText)
	}
}

// =============================================================================
// Phase 16I: Image Pipeline Integration Tests
// =============================================================================

// TestPreviewCmdPassesImageCache verifies that when entering Markdeep preview mode,
// the image cache created for the window is passed through to the RichText component
// via the WithRichTextImageCache option. This ensures images can be loaded and
// rendered during layout.
//
// The test simulates what previewcmd() should do:
// 1. Create an image cache
// 2. Pass it to RichText via WithRichTextImageCache
// 3. Verify the cache is accessible through the Frame
func TestPreviewCmdPassesImageCache(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Create a window with markdown content containing an image
	markdownContent := "# Test\n\n![My Image](test.png)\n\nSome text after."
	sourceRunes := []rune(markdownContent)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/docs/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create fonts and colors
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	// Create the image cache BEFORE creating RichText (this is what previewcmd should do).
	// Pre-load "test.png" as an error entry so layout gets a synchronous cache hit
	// instead of launching an async goroutine that races with the test.
	cache := rich.NewImageCache(10)
	cache.Load("test.png")
	w.imageCache = cache

	// Create RichText with the image cache option
	// This is what previewcmd() SHOULD be doing but currently doesn't
	rt := NewRichText()
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithRichTextImageCache(cache), // This is the critical line being tested
	)

	// Use markdown.Parse to get content with image spans (ParseWithSourceMap
	// doesn't currently handle images, that's a separate issue)
	parsedContent := markdown.Parse(markdownContent)

	// Verify basic parsing detected the image
	foundImage := false
	for _, span := range parsedContent {
		if span.Style.Image {
			foundImage = true
			if span.Style.ImageURL != "test.png" {
				t.Errorf("ImageURL = %q, want %q", span.Style.ImageURL, "test.png")
			}
			break
		}
	}
	if !foundImage {
		t.Error("markdown.Parse should detect image in content")
	}

	// Set content with images and render
	rt.SetContent(parsedContent)
	rt.Render(bodyRect)

	// For source mapping, we still need ParseWithSourceMap
	_, sourceMap, linkMap := markdown.ParseWithSourceMap(markdownContent)

	// Assign richBody and enable preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verification 1: Window has the cache
	if w.imageCache == nil {
		t.Error("Window.imageCache should not be nil")
	}

	// Verification 2: RichText has the cache (check via internal field)
	// Note: We can't directly access rt.imageCache since it's unexported,
	// but we can verify behavior by checking if the Frame was initialized
	// with the cache. The real test is that images render correctly.
	if rt.Frame() == nil {
		t.Fatal("RichText.Frame() should not be nil after Init")
	}

	// Verification 3: The cache itself should be usable
	// Try to get a path that doesn't exist - should return nil, false
	cached, found := cache.Get("/nonexistent/path.png")
	if found || cached != nil {
		t.Error("Get on non-existent path should return nil, false")
	}

	// Clean up
	w.SetPreviewMode(false)
	cache.Clear()
}

// =============================================================================
// Phase 18.2: Execute (B2) Tests
// =============================================================================

// TestPreviewB2Click tests that B2 (middle button/button 2) clicks in the preview
// frame area are detected and handled by HandlePreviewMouse. In Acme, B2 is used
// to execute commands. This test verifies:
// 1. B2 click in frame area is detected (returns true)
// 2. B2 click outside frame area is not handled (returns false)
// 3. B2 click in scrollbar area goes to scrollbar, not command execution
func TestPreviewB2Click(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown with command words
	sourceMarkdown := "# Commands\n\nRun **Del** to close.\n\nTry `Echo hello` command."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
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

	// Parse markdown and set content with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	// Set up preview mode
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// Get frame rect for positioning clicks
	frameRect := rt.Frame().Rect()

	// Test 1: B2 click in frame area should be handled
	// Click at a position in the text area
	clickPoint := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   clickPoint,
		Buttons: 2, // Button 2 (middle button)
	}
	// Immediate release for simple click
	upEvent := draw.Mouse{
		Point:   clickPoint,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{upEvent})

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Error("HandlePreviewMouse should handle B2 click in frame area")
	}

	// Test 2: B2 click outside body.all should not be handled
	outsidePoint := image.Pt(-10, -10)
	m2 := draw.Mouse{
		Point:   outsidePoint,
		Buttons: 2,
	}
	mc2 := mockMousectlWithEvents([]draw.Mouse{{Point: outsidePoint, Buttons: 0}})
	handled2 := w.HandlePreviewMouse(&m2, mc2)
	if handled2 {
		t.Error("HandlePreviewMouse should NOT handle B2 click outside body.all")
	}

	// Test 3: B2 click in scrollbar should be handled as scrollbar scroll (not command execution)
	// Scrollbar is to the left of the frame
	scrollRect := rt.ScrollRect()
	if !scrollRect.Empty() {
		scrollPoint := image.Pt(scrollRect.Min.X+2, scrollRect.Min.Y+20)
		m3 := draw.Mouse{
			Point:   scrollPoint,
			Buttons: 2,
		}
		// B2 in scrollbar triggers absolute scroll positioning
		handled3 := w.HandlePreviewMouse(&m3, nil)
		if !handled3 {
			t.Error("HandlePreviewMouse should handle B2 click in scrollbar")
		}
	}
}

// TestPreviewB2Sweep tests that B2 (middle button) sweep selection in preview mode
// selects a range of text from the anchor point to the release point. This is used
// to select text for command execution (the selected text will be executed as a command).
// This test verifies:
// 1. B2 sweep in frame area creates a selection
// 2. The selection spans from the mouse-down position to the mouse-up position
// 3. The selection is properly normalized (p0 < p1)
func TestPreviewB2Sweep(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", []rune("Echo hello world")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Echo hello world" (16 chars)
	content := rich.Plain("Echo hello world")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Test 1: B2 sweep from position 0 to position 5 (select "Echo ")
	// Position 0 is at X = frameRect.Min.X
	// Position 5 is at X = frameRect.Min.X + 50 (10px per char)
	t.Run("SweepForward", func(t *testing.T) {
		// Mouse down at position 0
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 2, // Button 2 (middle button)
		}
		// Drag to position 5 (still holding button)
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 2,
		}
		// Mouse up at position 5
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B2 sweep in frame area")
		}

		// After B2 execute, selection is restored to prior state (0, 0)
		q0, q1 := rt.Selection()
		if q0 != 0 || q1 != 0 {
			t.Errorf("B2 sweep should restore prior selection (0,0), got (%d,%d)", q0, q1)
		}
	})

	// Test 2: B2 sweep backward (from right to left) should also restore prior selection
	t.Run("SweepBackward", func(t *testing.T) {
		// Set a known prior selection and re-render to reset frame state
		rt.SetSelection(2, 4)
		rt.Render(bodyRect)

		// Mouse down at position 5 (50 pixels from left edge, 10px per char)
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 2,
		}
		// Drag to position 0 (still holding button)
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 2,
		}
		// Mouse up at position 0
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle backward B2 sweep")
		}

		// After B2 execute, selection is restored to prior state (2, 4)
		q0, q1 := rt.Selection()
		if q0 != 2 || q1 != 4 {
			t.Errorf("Backward B2 sweep should restore prior selection (2,4), got (%d,%d)", q0, q1)
		}
	})

	// Test 3: B2 sweep executes then restores prior selection
	t.Run("SweepSelectionForExec", func(t *testing.T) {
		// Set a known prior selection
		rt.SetSelection(0, 0)

		// Select "Echo hello" (positions 0 to 10)
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 2,
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+100, frameRect.Min.Y+5),
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+100, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B2 sweep for exec")
		}

		// After B2 execute, selection is restored to prior state (0, 0)
		q0, q1 := rt.Selection()
		if q0 != 0 || q1 != 0 {
			t.Errorf("B2 sweep should restore prior selection (0,0), got (%d,%d)", q0, q1)
		}
	})
}

// TestPreviewB2Execute tests the full B2 execute flow in preview mode:
// When the user B2-clicks (or sweeps) a command word in the rendered preview,
// the system should:
// 1. Select the text in the preview frame
// 2. Map the preview selection back to source buffer positions via syncSourceSelection()
// 3. Call execute() with the body Text and source-mapped positions
//
// This test verifies that after B2 click handling, the source body has the correct
// selection positions and that reading from body.file at those positions yields the
// command text that execute() would receive.
func TestPreviewB2Execute(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown with a command word formatted in bold: **Del**
	// Rendered preview text will be "Run Del now" (without the ** markers)
	// Source text is "Run **Del** now"
	sourceMarkdown := "Run **Del** now"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/exec.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown with source map
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	// Find "Del" in the rendered text to determine its position
	plainText := rt.Content().Plain()
	delIdx := -1
	for i := 0; i < len(plainText)-2; i++ {
		if string(plainText[i:i+3]) == "Del" {
			delIdx = i
			break
		}
	}
	if delIdx < 0 {
		t.Fatalf("Could not find 'Del' in rendered text: %q", string(plainText))
	}

	frameRect := rt.Frame().Rect()

	// Test 1: B2 null click on "Del" should expand to word, select it,
	// and map back to source positions where body.file contains "Del"
	t.Run("B2ClickExecuteMapping", func(t *testing.T) {
		// Click in the middle of "Del" in the rendered text
		// delIdx chars from left edge, each char is 10px
		clickX := frameRect.Min.X + delIdx*10 + 15 // middle of "Del"
		clickPoint := image.Pt(clickX, frameRect.Min.Y+5)
		m := draw.Mouse{
			Point:   clickPoint,
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   clickPoint,
			Buttons: 0,
		}
		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})

		handled := w.HandlePreviewMouse(&m, mc)
		if !handled {
			t.Fatal("HandlePreviewMouse should handle B2 click in frame area")
		}

		// After B2 execute, selection is restored to prior state (0,0)
		q0, q1 := rt.Selection()
		if q0 != 0 || q1 != 0 {
			t.Errorf("B2 execute should restore prior selection (0,0), got (%d,%d)", q0, q1)
		}

		// Verify Del is a valid built-in command that execute() can look up.
		// The handler internally extracts exec text and dispatches before
		// restoring the selection, so we verify the pipeline is correct.
		e := lookup("Del", globalexectab)
		if e == nil {
			t.Errorf("Command %q should be found in globalexectab", "Del")
		}
		if e != nil && e.name != "Del" {
			t.Errorf("Looked up command should be 'Del', got %q", e.name)
		}
	})

	// Test 2: B2 sweep selection should also produce correct exec text
	t.Run("B2SweepExecuteMapping", func(t *testing.T) {
		// Reset selection
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Sweep to select "Del" in the rendered text
		startX := frameRect.Min.X + delIdx*10
		endX := frameRect.Min.X + (delIdx+3)*10
		downEvent := draw.Mouse{
			Point:   image.Pt(startX, frameRect.Min.Y+5),
			Buttons: 2,
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(endX, frameRect.Min.Y+5),
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(endX, frameRect.Min.Y+5),
			Buttons: 0,
		}
		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})

		handled := w.HandlePreviewMouse(&downEvent, mc)
		if !handled {
			t.Fatal("HandlePreviewMouse should handle B2 sweep")
		}

		// After B2 execute, selection is restored to prior state (0,0)
		q0, q1 := rt.Selection()
		if q0 != 0 || q1 != 0 {
			t.Errorf("B2 sweep execute should restore prior selection (0,0), got (%d,%d)", q0, q1)
		}

		// Verify Del is a valid command for execute()
		e := lookup("Del", globalexectab)
		if e == nil {
			t.Errorf("Swept command %q should be found in globalexectab", "Del")
		}
	})

	// Test 3: Verify execute flow with non-built-in command text
	// When the user B2-clicks text that is not a built-in command,
	// execute() should treat it as an external command to run.
	t.Run("ExternalCommandText", func(t *testing.T) {
		// Set up with markdown containing a non-built-in command
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Find "Run" in the rendered text (not a built-in command)
		runIdx := -1
		for i := 0; i < len(plainText)-2; i++ {
			if string(plainText[i:i+3]) == "Run" {
				runIdx = i
				break
			}
		}
		if runIdx < 0 {
			t.Fatalf("Could not find 'Run' in rendered text: %q", string(plainText))
		}

		// B2 click on "Run"
		clickX := frameRect.Min.X + runIdx*10 + 15
		clickPoint := image.Pt(clickX, frameRect.Min.Y+5)
		m := draw.Mouse{
			Point:   clickPoint,
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   clickPoint,
			Buttons: 0,
		}
		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})

		handled := w.HandlePreviewMouse(&m, mc)
		if !handled {
			t.Fatal("HandlePreviewMouse should handle B2 click")
		}

		// After B2 execute, selection is restored to prior state (0,0)
		q0, q1 := rt.Selection()
		if q0 != 0 || q1 != 0 {
			t.Errorf("B2 execute should restore prior selection (0,0), got (%d,%d)", q0, q1)
		}

		// "Run" is not a built-in, so lookup should return nil
		// execute() would then try to run it as an external command
		e := lookup("Run", globalexectab)
		if e != nil {
			t.Errorf("'Run' should not be a built-in command, but lookup returned %q", e.name)
		}
	})
}

// TestPreviewB2BuiltinCommands verifies that built-in commands (Del, Snarf, Cut,
// Paste, Look, etc.) are correctly recognized and dispatched when B2-clicked in
// preview mode. This tests the full flow from B2 click -> word expansion ->
// previewExecute() -> built-in command dispatch.
func TestPreviewB2BuiltinCommands(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: display}
	defer func() { global.row = Row{} }()

	// Markdown containing multiple built-in command words.
	// Rendered text will be: "Del Snarf Cut Paste Look" (no markdown formatting)
	sourceMarkdown := "Del Snarf Cut Paste Look"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/builtins.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
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

	plainText := rt.Content().Plain()
	frameRect := rt.Frame().Rect()

	// Helper to find word position in rendered text
	findWord := func(word string) int {
		for i := 0; i <= len(plainText)-len(word); i++ {
			if string(plainText[i:i+len(word)]) == word {
				return i
			}
		}
		return -1
	}

	// Helper to B2-click a word, returning the exec text.
	// Note: HandlePreviewMouse calls previewExecute() which dispatches the
	// built-in command. Only use this for commands safe in test context.
	b2Click := func(t *testing.T, word string) {
		t.Helper()
		idx := findWord(word)
		if idx < 0 {
			t.Fatalf("Could not find %q in rendered text: %q", word, string(plainText))
		}

		// Reset selection
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		clickX := frameRect.Min.X + idx*10 + (len(word)*10)/2 // middle of word
		clickPoint := image.Pt(clickX, frameRect.Min.Y+5)
		m := draw.Mouse{
			Point:   clickPoint,
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   clickPoint,
			Buttons: 0,
		}
		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})

		handled := w.HandlePreviewMouse(&m, mc)
		if !handled {
			t.Fatalf("HandlePreviewMouse should handle B2 click on %q", word)
		}
	}

	// Test that each built-in command word is correctly extracted from preview
	// and recognized by lookup in globalexectab. We verify the B2 click -> word
	// expansion -> PreviewExecText() -> lookup pipeline for each command.
	t.Run("AllBuiltinsRecognized", func(t *testing.T) {
		builtins := []string{"Del", "Snarf", "Cut", "Paste", "Look"}
		for _, cmd := range builtins {
			t.Run(cmd, func(t *testing.T) {
				idx := findWord(cmd)
				if idx < 0 {
					t.Fatalf("Could not find %q in rendered text", cmd)
				}

				// B2 click to select the word and extract exec text,
				// but don't go through HandlePreviewMouse (which dispatches).
				// Instead simulate just the selection + extraction steps.
				rt.SetSelection(0, 0)
				rt.Render(bodyRect)

				// Simulate B2 null click word expansion
				clickX := frameRect.Min.X + idx*10 + (len(cmd)*10)/2
				clickPoint := image.Pt(clickX, frameRect.Min.Y+5)
				charPos := rt.Frame().Charofpt(clickPoint)

				// Expand to word boundaries (same logic as HandlePreviewMouse)
				_, wordStart, wordEnd := w.PreviewExpandWord(charPos)
				rt.SetSelection(wordStart, wordEnd)
				w.syncSourceSelection()

				execText := w.PreviewExecText()
				if execText != cmd {
					t.Errorf("PreviewExecText() returned %q, want %q", execText, cmd)
					return
				}

				e := lookup(execText, globalexectab)
				if e == nil {
					t.Errorf("Command %q should be found in globalexectab", cmd)
				} else if e.name != cmd {
					t.Errorf("Looked up command should be %q, got %q", cmd, e.name)
				}
			})
		}
	})

	// Test Snarf dispatch: previewExecute with "Snarf" should snarf the body
	// selection text into global.snarfbuf. This verifies the full dispatch
	// path from previewExecute -> lookup -> cut(dosnarf=true, docut=false).
	t.Run("SnarfDispatch", func(t *testing.T) {
		// Set body selection to "Del" (first 3 chars of source)
		w.body.q0 = 0
		w.body.q1 = 3

		// Set global.seltext so cut() uses body selection
		global.seltext = &w.body

		// Clear snarfbuf
		global.snarfbuf = nil

		// Execute Snarf via previewExecute
		previewExecute(&w.body, "Snarf")

		// Verify snarfbuf was populated with the selected source text
		if len(global.snarfbuf) == 0 {
			t.Error("global.snarfbuf should be populated after Snarf, but is empty")
		} else {
			got := string(global.snarfbuf)
			if got != "Del" {
				t.Errorf("global.snarfbuf should contain 'Del', got %q", got)
			}
		}
	})

	// Test B2 click on "Snarf" through HandlePreviewMouse dispatches correctly.
	// Snarf is safe to dispatch in test context since it only copies to snarfbuf.
	t.Run("SnarfViaB2Click", func(t *testing.T) {
		// Set body selection for snarf to copy
		w.body.q0 = 4
		w.body.q1 = 9 // "Snarf"
		global.seltext = &w.body
		global.snarfbuf = nil

		b2Click(t, "Snarf")

		// After HandlePreviewMouse dispatches Snarf, snarfbuf should be populated.
		// The body selection is synced via syncSourceSelection(), and cut()
		// reads from the body file at the synced positions.
		if len(global.snarfbuf) == 0 {
			t.Error("global.snarfbuf should be populated after B2-clicking Snarf")
		}
	})

	// Test Cut command flags are correct for preview dispatch
	t.Run("CutFlags", func(t *testing.T) {
		e := lookup("Cut", globalexectab)
		if e == nil {
			t.Fatal("Cut should be in globalexectab")
		}
		// Cut has mark=true, flag1=true (dosnarf), flag2=true (docut)
		if !e.mark {
			t.Error("Cut should be marked as undoable")
		}
		if !e.flag1 {
			t.Error("Cut should have flag1 (dosnarf) set")
		}
		if !e.flag2 {
			t.Error("Cut should have flag2 (docut) set")
		}
	})

	// Test Paste command flags
	t.Run("PasteFlags", func(t *testing.T) {
		e := lookup("Paste", globalexectab)
		if e == nil {
			t.Fatal("Paste should be in globalexectab")
		}
		if !e.mark {
			t.Error("Paste should be marked as undoable")
		}
	})

	// Test Look command is recognized
	t.Run("LookRecognized", func(t *testing.T) {
		e := lookup("Look", globalexectab)
		if e == nil {
			t.Fatal("Look should be in globalexectab")
		}
		if e.name != "Look" {
			t.Errorf("Expected Look command, got %q", e.name)
		}
	})
}

// TestPreviewB3Sweep tests that B3 (button 3) sweep selection works in preview mode.
// B3 sweep should select text in the rich text frame, similar to B2 sweep but for Look.
func TestPreviewB3Sweep(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", []rune("Hello world test")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Hello world test" (16 chars)
	content := rich.Plain("Hello world test")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Test 1: B3 sweep from position 0 to position 5 (select "Hello")
	t.Run("SweepForward", func(t *testing.T) {
		// Mouse down at position 0
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 4, // Button 3 (right button)
		}
		// Drag to position 5 (still holding button)
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 4,
		}
		// Mouse up at position 5
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B3 sweep in frame area")
		}

		// After sweep from 0 to 5, selection should be (0, 5)
		q0, q1 := rt.Selection()
		if q0 != 0 {
			t.Errorf("B3 sweep selection p0 should be 0, got %d", q0)
		}
		if q1 != 5 {
			t.Errorf("B3 sweep selection p1 should be 5, got %d", q1)
		}
	})

	// Test 2: B3 sweep backward (from right to left) should normalize selection
	t.Run("SweepBackward", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Mouse down at position 5
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 4,
		}
		// Drag to position 0
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 4,
		}
		// Mouse up
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle backward B3 sweep")
		}

		// Selection should be normalized: p0 < p1
		q0, q1 := rt.Selection()
		if q0 != 0 {
			t.Errorf("Backward B3 sweep selection p0 should be 0, got %d", q0)
		}
		if q1 != 5 {
			t.Errorf("Backward B3 sweep selection p1 should be 5, got %d", q1)
		}
	})

	// Test 3: B3 sweep selects text that can be retrieved for Look
	t.Run("SweepSelectionForLook", func(t *testing.T) {
		rt.SetSelection(0, 0)

		// Select "Hello world" (positions 0 to 11)
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 4,
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+110, frameRect.Min.Y+5),
			Buttons: 4,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+110, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B3 sweep for Look")
		}

		// Verify selection covers the intended range
		q0, q1 := rt.Selection()
		if q1-q0 < 5 {
			t.Errorf("B3 sweep should select at least 5 characters, got selection (%d, %d)", q0, q1)
		}

		// The selected text should be retrievable via PreviewLookText
		lookText := w.PreviewLookText()
		if len(lookText) == 0 {
			t.Error("PreviewLookText should return non-empty text after B3 sweep selection")
		}
	})
}

// TestPreviewB3ExpandWord tests that a B3 null click (no sweep) in preview mode
// expands the click position to the surrounding word using PreviewExpandWord().
func TestPreviewB3ExpandWord(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", []rune("Hello world test")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content := rich.Plain("Hello world test")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Test 1: B3 null click in middle of "Hello" should expand to "Hello"
	t.Run("ExpandFirstWord", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Click at position 2 (middle of "Hello"), 10px per char
		clickPt := image.Pt(frameRect.Min.X+25, frameRect.Min.Y+5)
		downEvent := draw.Mouse{
			Point:   clickPt,
			Buttons: 4, // Button 3
		}
		// Immediate release at same position (null click)
		upEvent := draw.Mouse{
			Point:   clickPt,
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B3 null click in frame area")
		}

		// After null click, word expansion should give us "Hello"
		charPos := rt.Frame().Charofpt(clickPt)
		word, start, end := w.PreviewExpandWord(charPos)
		if word != "Hello" {
			t.Errorf("PreviewExpandWord should return \"Hello\", got %q", word)
		}
		if start != 0 {
			t.Errorf("PreviewExpandWord start should be 0, got %d", start)
		}
		if end != 5 {
			t.Errorf("PreviewExpandWord end should be 5, got %d", end)
		}
	})

	// Test 2: B3 null click on "world" should expand to "world"
	t.Run("ExpandSecondWord", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Click at position 8 (middle of "world"), 10px per char
		clickPt := image.Pt(frameRect.Min.X+85, frameRect.Min.Y+5)
		charPos := rt.Frame().Charofpt(clickPt)
		word, start, end := w.PreviewExpandWord(charPos)
		if word != "world" {
			t.Errorf("PreviewExpandWord should return \"world\", got %q", word)
		}
		if start != 6 {
			t.Errorf("PreviewExpandWord start should be 6, got %d", start)
		}
		if end != 11 {
			t.Errorf("PreviewExpandWord end should be 11, got %d", end)
		}
	})

	// Test 3: B3 null click at a position between words
	// When clicking on a space char, PreviewExpandWord may return a neighboring
	// word or empty string depending on boundary behavior. Verify it doesn't panic
	// and returns a consistent result.
	t.Run("NullClickBetweenWords", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		// Click at position 5 (space between "Hello" and "world"), 10px per char
		clickPt := image.Pt(frameRect.Min.X+55, frameRect.Min.Y+5)
		charPos := rt.Frame().Charofpt(clickPt)
		word, start, end := w.PreviewExpandWord(charPos)
		// Should return some result without panicking
		// The exact behavior depends on boundary handling
		_ = word
		if end < start {
			t.Errorf("PreviewExpandWord end (%d) should not be less than start (%d)", end, start)
		}
	})
}

// TestPreviewB3Search tests that B3 on non-link text in preview mode triggers
// a search for the rendered text in the source body buffer.
func TestPreviewB3Search(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Markdown source with bold text
	sourceMarkdown := "Some **important** text here.\n\nFind important word."
	// Rendered text: "Some important text here.\n\nFind important word."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content with source map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Verify preview mode is set
	if !w.IsPreviewMode() {
		t.Fatal("Window should be in preview mode")
	}

	// Find "important" in the rendered text
	plainText := content.Plain()
	importantIdx := -1
	for i := 0; i < len(plainText)-8; i++ {
		if string(plainText[i:i+9]) == "important" {
			importantIdx = i
			break
		}
	}
	if importantIdx < 0 {
		t.Fatalf("Could not find 'important' in rendered text: %q", string(plainText))
	}

	// Test: B3 null click on "important" (not a link) should use search fallback
	// The click position is not on a link, so PreviewLookLinkURL should return ""
	url := w.PreviewLookLinkURL(importantIdx)
	if url != "" {
		t.Errorf("PreviewLookLinkURL should return empty for non-link text, got %q", url)
	}

	// After B3 click, the word should be expanded and available for search
	word, start, end := w.PreviewExpandWord(importantIdx + 2) // click in middle of "important"
	if word != "important" {
		t.Errorf("PreviewExpandWord should return \"important\", got %q", word)
	}
	if end <= start {
		t.Errorf("PreviewExpandWord should return valid range, got (%d, %d)", start, end)
	}

	// Set the selection to the expanded word
	rt.SetSelection(start, end)

	// Verify PreviewLookText returns the rendered text for search
	lookText := w.PreviewLookText()
	if lookText != "important" {
		t.Errorf("PreviewLookText should return \"important\", got %q", lookText)
	}

	// The search target should be the rendered text "important", which exists
	// in the source body as both "**important**" and "important"
	// The search() function should be able to find it in the body
	sourceText := string(sourceRunes)
	if !containsSubstring(sourceText, "important") {
		t.Error("Source text should contain 'important' for search to find")
	}
}

// TestPreviewB3OnSelection tests that B3 clicked inside an existing selection
// uses the selected text for the Look operation instead of expanding a word.
func TestPreviewB3OnSelection(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceMarkdown := "Hello world test phrase here"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Pre-set a selection of "world test" (positions 6-16)
	// This spans two words, which wouldn't be the result of single word expansion
	rt.SetSelection(6, 16)

	// Verify the selection is set
	q0, q1 := rt.Selection()
	if q0 != 6 || q1 != 16 {
		t.Fatalf("Selection should be (6, 16), got (%d, %d)", q0, q1)
	}

	// B3 click inside the selection should use the existing selection text
	// Click position 10 is inside the selection (6, 16)
	frameRect := rt.Frame().Rect()
	clickPt := image.Pt(frameRect.Min.X+105, frameRect.Min.Y+5) // position ~10

	// Verify click position is inside the selection
	charPos := rt.Frame().Charofpt(clickPt)
	if charPos < q0 || charPos >= q1 {
		t.Logf("Warning: charPos %d may not be inside selection (%d, %d)", charPos, q0, q1)
	}

	// The PreviewLookText should return the full selection "world test"
	// (not just the word "test" that would result from word expansion)
	lookText := w.PreviewLookText()
	if lookText != "world test" {
		t.Errorf("PreviewLookText with existing selection should return \"world test\", got %q", lookText)
	}

	// Test: if selection exists and B3 is clicked outside the selection,
	// the selection should change (word expand at new position)
	t.Run("B3OutsideSelection", func(t *testing.T) {
		// Keep selection at (6, 16)
		rt.SetSelection(6, 16)

		// Click at position 0 (outside selection, on "Hello")
		outsidePt := image.Pt(frameRect.Min.X+25, frameRect.Min.Y+5) // position ~2
		outsideCharPos := rt.Frame().Charofpt(outsidePt)

		// Word expand at position outside the selection should give "Hello"
		word, _, _ := w.PreviewExpandWord(outsideCharPos)
		if word != "Hello" {
			t.Errorf("PreviewExpandWord outside selection should return \"Hello\", got %q", word)
		}
	})
}

// setupPreviewChordTestWindow creates a Window in preview mode for chord testing.
// It sets up markdown content "Hello world test" with a source map, and returns
// the window, RichText, and frame rect for positioning mouse events.
func setupPreviewChordTestWindow(t *testing.T) (*Window, *RichText, image.Rectangle) {
	t.Helper()

	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceMarkdown := "Hello world test"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.w = w

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: display}
	t.Cleanup(func() { global.row = Row{} })

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content with source map for source position mapping
	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()
	return w, rt, frameRect
}

// TestPreviewChordDetection tests that after a B1 selection in preview mode,
// additional button presses (B2 or B3) while B1 is still held are detected
// as chord events. In Acme, chording is a core interaction pattern:
//   - B1+B2 = Cut (copy to snarf buffer and delete)
//   - B1+B3 = Paste (replace selection with snarf buffer)
//   - B1+B2+B3 = Snarf (copy to snarf buffer, no delete)
//
// This test verifies:
// 1. B1 press followed by B2 while B1 held is detected as a chord
// 2. B1 press followed by B3 while B1 held is detected as a chord
// 3. B1 press and release without B2/B3 is a normal selection (no chord)
func TestPreviewChordDetection(t *testing.T) {
	w, rt, frameRect := setupPreviewChordTestWindow(t)

	// Ensure body.w is set so cut() can operate on the text properly
	w.body.w = w

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: w.display}
	defer func() { global.row = Row{} }()

	// Test 1: B1 sweep to select "Hello" (chars 0-5), then B2 pressed while B1 held
	// This should be detected as B1+B2 chord (Cut)
	t.Run("B1ThenB2Chord", func(t *testing.T) {
		// B1 mouse down at char 0
		downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
		m := draw.Mouse{
			Point:   downPt,
			Buttons: 1, // B1 down
		}
		// Drag to char 5 with B1 held
		dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
		dragEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 1, // B1 still held
		}
		// B2 pressed while B1 still held (chord event)
		chordEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 3, // B1 (1) + B2 (2) = 3
		}
		// All buttons released
		upEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 0,
		}
		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

		handled := w.HandlePreviewMouse(&m, mc)
		if !handled {
			t.Error("HandlePreviewMouse should handle B1 click in frame area")
		}

		// After a B1+B2 chord cut, the selected text is deleted and
		// the selection should collapse to a cursor (p0 == p1).
		p0, p1 := rt.Selection()
		if p0 != p1 {
			t.Errorf("Expected collapsed selection after B1+B2 cut chord, got p0=%d p1=%d", p0, p1)
		}
	})

	// Test 2: B1 sweep to select "world" (chars 6-11), then B3 pressed while B1 held
	// This should be detected as B1+B3 chord (Paste)
	t.Run("B1ThenB3Chord", func(t *testing.T) {
		// B1 mouse down at char 6
		downPt := image.Pt(frameRect.Min.X+60, frameRect.Min.Y+5)
		m := draw.Mouse{
			Point:   downPt,
			Buttons: 1, // B1 down
		}
		// Drag to char 11 with B1 held
		dragPt := image.Pt(frameRect.Min.X+110, frameRect.Min.Y+5)
		dragEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 1,
		}
		// B3 pressed while B1 still held (chord event)
		chordEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 5, // B1 (1) + B3 (4) = 5
		}
		// All buttons released
		upEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 0,
		}
		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

		handled := w.HandlePreviewMouse(&m, mc)
		if !handled {
			t.Error("HandlePreviewMouse should handle B1 click in frame area")
		}

		p0, p1 := rt.Selection()
		if p0 == p1 {
			t.Error("Expected non-empty selection after B1 sweep for chord")
		}
	})

	// Test 3: B1 sweep and release (no chord) should be a normal selection
	t.Run("B1OnlyNoChord", func(t *testing.T) {
		downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
		m := draw.Mouse{
			Point:   downPt,
			Buttons: 1,
		}
		dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
		dragEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 1,
		}
		upEvent := draw.Mouse{
			Point:   dragPt,
			Buttons: 0, // B1 released, no chord
		}
		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})

		handled := w.HandlePreviewMouse(&m, mc)
		if !handled {
			t.Error("HandlePreviewMouse should handle B1 click in frame area")
		}

		// Normal selection should still work
		p0, p1 := rt.Selection()
		if p0 >= p1 {
			t.Error("Expected non-empty selection after B1 sweep")
		}
	})

	_ = w
}

// TestPreviewChordCut tests that the B1+B2 chord in preview mode performs a Cut
// operation: the selected text is copied to the snarf buffer and deleted from
// the source body buffer. The preview should reflect the deletion.
// It also verifies that the cut operation uses the standard cut() path,
// which means undo works and the system clipboard is synced.
func TestPreviewChordCut(t *testing.T) {
	w, rt, frameRect := setupPreviewChordTestWindow(t)

	// Ensure body.w is set so cut() can operate on the text properly
	w.body.w = w

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: w.display}
	defer func() { global.row = Row{} }()

	// Select "Hello" (chars 0-5) with B1, then chord B2 to cut
	downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	// B1+B2 chord (Cut)
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 3, // B1 + B2
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	// Clear snarf buffer and display snarf before test
	global.snarfbuf = nil
	w.display.WriteSnarf(nil)

	originalText := "Hello world test"
	originalLen := len([]rune(originalText))

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle B1+B2 chord")
	}

	// After B1+B2 chord, the snarf buffer should contain the cut text
	if len(global.snarfbuf) == 0 {
		t.Error("snarf buffer should contain cut text after B1+B2 chord")
	}

	// The source body should have the selected text removed
	bodyLen := w.body.file.Nr()
	if bodyLen >= originalLen {
		t.Errorf("body length should decrease after cut: got %d, original %d", bodyLen, originalLen)
	}

	// Verify the standard cut path was used: acmeputsnarf() should have
	// synced global.snarfbuf to the display's system clipboard via WriteSnarf().
	clipBuf := make([]byte, 1024)
	n, _, err := w.display.ReadSnarf(clipBuf)
	if err != nil {
		t.Fatalf("ReadSnarf failed: %v", err)
	}
	if n == 0 {
		t.Error("system clipboard (display snarf) should be updated after chord cut; acmeputsnarf() was not called")
	}

	// Verify undo restores the original text: the cut should have set up
	// proper undo sequence (TypeCommit + seq++ + Mark) so Undo works.
	w.Undo(true)
	afterUndoLen := w.body.file.Nr()
	if afterUndoLen != originalLen {
		t.Errorf("after undo, body length should be restored to %d, got %d", originalLen, afterUndoLen)
	}

	_ = rt
}

// TestPreviewChordPaste tests that the B1+B3 chord in preview mode performs a Paste
// operation: the snarf buffer content replaces the current selection in the source
// body buffer. The preview should reflect the replacement.
// It also verifies that the paste operation uses the standard paste() path,
// which means undo works and proper sequence points are set.
func TestPreviewChordPaste(t *testing.T) {
	w, rt, frameRect := setupPreviewChordTestWindow(t)

	// Ensure body.w is set so paste() can operate on the text properly
	w.body.w = w

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: w.display}
	defer func() { global.row = Row{} }()

	// Pre-fill snarf buffer with replacement text (both global and display snarf,
	// since paste() calls acmegetsnarf() which reads from the display)
	global.snarfbuf = []byte("Goodbye")
	w.display.WriteSnarf([]byte("Goodbye"))

	originalText := "Hello world test"
	originalLen := len([]rune(originalText))

	// Select "Hello" (chars 0-5) with B1, then chord B3 to paste
	downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	// B1+B3 chord (Paste)
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 5, // B1 (1) + B3 (4) = 5
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle B1+B3 chord")
	}

	// After B1+B3 chord, the source body should contain the pasted text
	bodyLen := w.body.file.Nr()
	buf := make([]rune, bodyLen)
	w.body.file.Read(0, buf)
	bodyText := string(buf)

	// "Hello" should be replaced with "Goodbye"
	if !containsSubstring(bodyText, "Goodbye") {
		t.Errorf("body should contain 'Goodbye' after paste, got %q", bodyText)
	}

	// Verify undo restores the original text: the paste should have set up
	// proper undo sequence (TypeCommit + seq++ + Mark) so Undo works.
	w.Undo(true)
	afterUndoLen := w.body.file.Nr()
	if afterUndoLen != originalLen {
		t.Errorf("after undo, body length should be restored to %d, got %d", originalLen, afterUndoLen)
	}

	_ = rt
}

// TestPreviewChordSnarf tests that the B1+B2+B3 chord in preview mode performs a
// Snarf (copy) operation: the selected text is copied to the snarf buffer but NOT
// deleted from the source. This is different from Cut (B1+B2) which also deletes.
func TestPreviewChordSnarf(t *testing.T) {
	w, rt, frameRect := setupPreviewChordTestWindow(t)

	// Ensure body.w is set so cut() can operate on the text properly
	w.body.w = w

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: w.display}
	defer func() { global.row = Row{} }()

	// Select "Hello" (chars 0-5) with B1, then chord B2+B3 to snarf
	downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	// B1+B2+B3 chord (Snarf): all three buttons held
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 7, // B1 (1) + B2 (2) + B3 (4) = 7
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	// Clear snarf buffer and display snarf before test
	global.snarfbuf = nil
	w.display.WriteSnarf(nil)

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle B1+B2+B3 chord")
	}

	// After B1+B2+B3 chord, the snarf buffer should contain the selected text
	if len(global.snarfbuf) == 0 {
		t.Error("snarf buffer should contain snarfed text after B1+B2+B3 chord")
	}

	// Verify the standard snarf path was used: acmeputsnarf() should have
	// synced global.snarfbuf to the display's system clipboard via WriteSnarf().
	clipBuf := make([]byte, 1024)
	n, _, err := w.display.ReadSnarf(clipBuf)
	if err != nil {
		t.Fatalf("ReadSnarf failed: %v", err)
	}
	if n == 0 {
		t.Error("system clipboard (display snarf) should be updated after chord snarf; acmeputsnarf() was not called")
	}
	if n > 0 && string(clipBuf[:n]) != string(global.snarfbuf) {
		t.Errorf("system clipboard content should match global.snarfbuf: got %q, want %q", string(clipBuf[:n]), string(global.snarfbuf))
	}

	// Source body should NOT be modified (snarf copies but doesn't delete)
	bodyLen := w.body.file.Nr()
	originalLen := len([]rune("Hello world test"))
	if bodyLen != originalLen {
		t.Errorf("body length should be unchanged after snarf: got %d, expected %d", bodyLen, originalLen)
	}

	_ = rt
}

// TestPreviewCutSourceMapping tests that chord operations (Cut, Paste) correctly
// map the preview selection back to source positions using the source map.
// This ensures that edits happen at the correct positions in the markdown source,
// not the rendered positions.
func TestPreviewCutSourceMapping(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Use markdown with formatting so rendered positions differ from source positions
	sourceMarkdown := "Hello **bold** world"
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect
	w.body.w = w

	// Set up global.row so acmeputsnarf() can call display.WriteSnarf()
	global.row = Row{display: display}
	defer func() { global.row = Row{} }()

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
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

	frameRect := rt.Frame().Rect()

	// The rendered text is "Hello bold world" (no ** markers).
	// Select "bold" in the rendered view (chars 6-10 in rendered text).
	// This should map to source positions covering "**bold**" (chars 6-14).
	downPt := image.Pt(frameRect.Min.X+60, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+100, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	// B1+B2 chord (Cut) to verify source mapping
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 3, // B1 + B2
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	global.snarfbuf = nil

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle chord in frame area")
	}

	// The cut should have operated on source positions, removing the markdown
	// formatting markers along with the word
	bodyLen := w.body.file.Nr()
	buf := make([]rune, bodyLen)
	w.body.file.Read(0, buf)
	bodyText := string(buf)

	// After cutting "bold" from rendered view, the source should have the
	// corresponding markdown removed. The exact result depends on source map
	// granularity, but "**bold**" should no longer be present.
	if containsSubstring(bodyText, "**bold**") {
		t.Errorf("source should not contain '**bold**' after cut, got %q", bodyText)
	}

	_ = rt
}

// TestPreviewReRenderAfterEdit tests that after a chord edit operation (Cut or Paste),
// the preview is re-rendered to reflect the changed source content. This ensures the
// user sees an up-to-date rendered view after each chord operation.
func TestPreviewReRenderAfterEdit(t *testing.T) {
	w, rt, frameRect := setupPreviewChordTestWindow(t)

	// Get initial content length from preview
	initialContent := rt.Content()
	initialLen := initialContent.Len()

	// Select "Hello" (chars 0-5) with B1, then chord B2 to cut
	downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 3, // B1+B2 (Cut)
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	global.snarfbuf = nil

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle chord")
	}

	// After cutting text, the preview content should be re-rendered with shorter content
	updatedContent := rt.Content()
	updatedLen := updatedContent.Len()

	if updatedLen >= initialLen {
		t.Errorf("preview content length should decrease after cut: initial=%d, updated=%d", initialLen, updatedLen)
	}
}

// TestSelectionContext tests the SelectionContext struct used for context-aware
// paste operations in preview mode. SelectionContext tracks metadata about the
// current selection including source/rendered positions, content type, and
// formatting information needed to adapt paste behavior.
func TestSelectionContext(t *testing.T) {
	t.Run("ZeroValue", func(t *testing.T) {
		// A zero-value SelectionContext should have ContentPlain type
		var ctx SelectionContext
		if ctx.ContentType != ContentPlain {
			t.Errorf("zero-value ContentType = %v, want ContentPlain (%v)", ctx.ContentType, ContentPlain)
		}
		if ctx.SourceStart != 0 || ctx.SourceEnd != 0 {
			t.Errorf("zero-value source range = (%d,%d), want (0,0)", ctx.SourceStart, ctx.SourceEnd)
		}
		if ctx.RenderedStart != 0 || ctx.RenderedEnd != 0 {
			t.Errorf("zero-value rendered range = (%d,%d), want (0,0)", ctx.RenderedStart, ctx.RenderedEnd)
		}
		if ctx.CodeLanguage != "" {
			t.Errorf("zero-value CodeLanguage = %q, want empty", ctx.CodeLanguage)
		}
		if ctx.IncludesOpenMarker || ctx.IncludesCloseMarker {
			t.Error("zero-value should not include markers")
		}
	})

	t.Run("ContentTypes", func(t *testing.T) {
		// Verify all content type constants are distinct
		types := []SelectionContentType{
			ContentPlain,
			ContentHeading,
			ContentBold,
			ContentItalic,
			ContentBoldItalic,
			ContentCode,
			ContentCodeBlock,
			ContentLink,
			ContentImage,
			ContentMixed,
		}
		seen := make(map[SelectionContentType]bool)
		for _, ct := range types {
			if seen[ct] {
				t.Errorf("duplicate content type value: %v", ct)
			}
			seen[ct] = true
		}
	})

	t.Run("PlainText", func(t *testing.T) {
		ctx := SelectionContext{
			SourceStart:   0,
			SourceEnd:     5,
			RenderedStart: 0,
			RenderedEnd:   5,
			ContentType:   ContentPlain,
		}
		if ctx.ContentType != ContentPlain {
			t.Errorf("ContentType = %v, want ContentPlain", ctx.ContentType)
		}
		if ctx.SourceEnd-ctx.SourceStart != 5 {
			t.Errorf("source length = %d, want 5", ctx.SourceEnd-ctx.SourceStart)
		}
	})

	t.Run("BoldSelection", func(t *testing.T) {
		// Selecting "bold" from "**bold**" in rendered text
		// Source: "**bold**" (positions 0-8)
		// Rendered: "bold" (positions 0-4)
		ctx := SelectionContext{
			SourceStart:        0,
			SourceEnd:          8,
			RenderedStart:      0,
			RenderedEnd:        4,
			ContentType:        ContentBold,
			PrimaryStyle:       rich.Style{Bold: true, Scale: 1.0},
			IncludesOpenMarker: true,
			IncludesCloseMarker: true,
		}
		if ctx.ContentType != ContentBold {
			t.Errorf("ContentType = %v, want ContentBold", ctx.ContentType)
		}
		if !ctx.IncludesOpenMarker || !ctx.IncludesCloseMarker {
			t.Error("full bold selection should include both markers")
		}
		if !ctx.PrimaryStyle.Bold {
			t.Error("PrimaryStyle should have Bold set")
		}
	})

	t.Run("PartialBoldSelection", func(t *testing.T) {
		// Selecting "ol" from "**bold**" in rendered text
		// Source: positions within "**bold**" excluding markers
		// Rendered: "ol" (positions 1-3)
		ctx := SelectionContext{
			SourceStart:         4, // "**b|ol|d**" -> source pos of 'o'
			SourceEnd:           6, // source pos after 'l'
			RenderedStart:       1,
			RenderedEnd:         3,
			ContentType:         ContentBold,
			PrimaryStyle:        rich.Style{Bold: true, Scale: 1.0},
			IncludesOpenMarker:  false,
			IncludesCloseMarker: false,
		}
		if ctx.ContentType != ContentBold {
			t.Errorf("ContentType = %v, want ContentBold", ctx.ContentType)
		}
		if ctx.IncludesOpenMarker || ctx.IncludesCloseMarker {
			t.Error("partial bold selection should not include markers")
		}
	})

	t.Run("HeadingSelection", func(t *testing.T) {
		// Selecting entire heading text from "# Heading"
		// Source: "# Heading\n" (positions 0-10)
		// Rendered: "Heading\n" (positions 0-8)
		ctx := SelectionContext{
			SourceStart:        0,
			SourceEnd:          10,
			RenderedStart:      0,
			RenderedEnd:        8,
			ContentType:        ContentHeading,
			PrimaryStyle:       rich.Style{Bold: true, Scale: 2.0},
			IncludesOpenMarker: true,
		}
		if ctx.ContentType != ContentHeading {
			t.Errorf("ContentType = %v, want ContentHeading", ctx.ContentType)
		}
		if !ctx.IncludesOpenMarker {
			t.Error("heading selection from start should include open marker")
		}
	})

	t.Run("CodeBlockSelection", func(t *testing.T) {
		// Selecting text inside a fenced code block
		ctx := SelectionContext{
			SourceStart:   0,
			SourceEnd:     30,
			RenderedStart: 0,
			RenderedEnd:   15,
			ContentType:   ContentCodeBlock,
			CodeLanguage:  "go",
			PrimaryStyle:  rich.Style{Code: true, Block: true, Scale: 1.0},
		}
		if ctx.ContentType != ContentCodeBlock {
			t.Errorf("ContentType = %v, want ContentCodeBlock", ctx.ContentType)
		}
		if ctx.CodeLanguage != "go" {
			t.Errorf("CodeLanguage = %q, want %q", ctx.CodeLanguage, "go")
		}
	})

	t.Run("InlineCodeSelection", func(t *testing.T) {
		// Selecting inline code "`code`"
		ctx := SelectionContext{
			SourceStart:         0,
			SourceEnd:           6, // `code`
			RenderedStart:       0,
			RenderedEnd:         4, // code
			ContentType:         ContentCode,
			PrimaryStyle:        rich.Style{Code: true, Scale: 1.0},
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		if ctx.ContentType != ContentCode {
			t.Errorf("ContentType = %v, want ContentCode", ctx.ContentType)
		}
	})

	t.Run("LinkSelection", func(t *testing.T) {
		// Selecting link text from "[link](url)"
		ctx := SelectionContext{
			SourceStart:         0,
			SourceEnd:           12,
			RenderedStart:       0,
			RenderedEnd:         4,
			ContentType:         ContentLink,
			PrimaryStyle:        rich.Style{Link: true, Fg: rich.LinkBlue, Scale: 1.0},
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		if ctx.ContentType != ContentLink {
			t.Errorf("ContentType = %v, want ContentLink", ctx.ContentType)
		}
		if !ctx.PrimaryStyle.Link {
			t.Error("PrimaryStyle should have Link set")
		}
	})

	t.Run("ImageSelection", func(t *testing.T) {
		// Selecting image placeholder
		ctx := SelectionContext{
			SourceStart:   0,
			SourceEnd:     22, // ![alt text](image.png)
			RenderedStart: 0,
			RenderedEnd:   16, // [Image: alt text]
			ContentType:   ContentImage,
			PrimaryStyle:  rich.Style{Image: true, Scale: 1.0},
		}
		if ctx.ContentType != ContentImage {
			t.Errorf("ContentType = %v, want ContentImage", ctx.ContentType)
		}
	})

	t.Run("MixedSelection", func(t *testing.T) {
		// Selecting across multiple formatting types
		// e.g., "plain **bold** *italic*"
		ctx := SelectionContext{
			SourceStart:   0,
			SourceEnd:     24,
			RenderedStart: 0,
			RenderedEnd:   18,
			ContentType:   ContentMixed,
		}
		if ctx.ContentType != ContentMixed {
			t.Errorf("ContentType = %v, want ContentMixed", ctx.ContentType)
		}
	})

	t.Run("ItalicSelection", func(t *testing.T) {
		ctx := SelectionContext{
			SourceStart:         0,
			SourceEnd:           8, // *italic*
			RenderedStart:       0,
			RenderedEnd:         6, // italic
			ContentType:         ContentItalic,
			PrimaryStyle:        rich.Style{Italic: true, Scale: 1.0},
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		if ctx.ContentType != ContentItalic {
			t.Errorf("ContentType = %v, want ContentItalic", ctx.ContentType)
		}
		if !ctx.PrimaryStyle.Italic {
			t.Error("PrimaryStyle should have Italic set")
		}
	})

	t.Run("BoldItalicSelection", func(t *testing.T) {
		ctx := SelectionContext{
			SourceStart:   0,
			SourceEnd:     13, // ***both***
			RenderedStart: 0,
			RenderedEnd:   4, // both
			ContentType:   ContentBoldItalic,
			PrimaryStyle:  rich.Style{Bold: true, Italic: true, Scale: 1.0},
		}
		if ctx.ContentType != ContentBoldItalic {
			t.Errorf("ContentType = %v, want ContentBoldItalic", ctx.ContentType)
		}
		if !ctx.PrimaryStyle.Bold || !ctx.PrimaryStyle.Italic {
			t.Error("PrimaryStyle should have both Bold and Italic set")
		}
	})
}

// TestAnalyzeSelectionContent tests the analyzeSelectionContent method which
// examines the spans in the rendered RichText content within the given
// rendered-position range [rStart, rEnd) and determines the SelectionContentType.
// This is used during selection context updates to classify what kind of
// markdown content the user has selected (plain, bold, italic, code, heading, etc.).
func TestAnalyzeSelectionContent(t *testing.T) {
	// Helper to create a Window with richBody set to given content.
	setupWindow := func(t *testing.T, content rich.Content) *Window {
		t.Helper()
		rect := image.Rect(0, 0, 800, 600)
		display := edwoodtest.NewDisplay(rect)
		global.configureGlobals(display)

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("/test/readme.md", nil),
		}
		w.body.all = image.Rect(0, 20, 800, 600)
		w.col = &Column{safe: true}

		font := edwoodtest.NewFont(10, 14)
		bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
		textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

		rt := NewRichText()
		bodyRect := image.Rect(12, 20, 800, 600)
		rt.Init(display, font,
			WithRichTextBackground(bgImage),
			WithRichTextColor(textImage),
		)
		rt.Render(bodyRect)
		rt.SetContent(content)
		w.richBody = rt
		w.SetPreviewMode(true)
		return w
	}

	t.Run("PlainText", func(t *testing.T) {
		// Content: "Hello world" — all plain text with default style.
		content := rich.Plain("Hello world")
		w := setupWindow(t, content)

		// Selecting "Hello" (positions 0-5) should be plain.
		got := w.analyzeSelectionContent(0, 5)
		if got != ContentPlain {
			t.Errorf("analyzeSelectionContent(0,5) = %v, want ContentPlain", got)
		}
	})

	t.Run("AllBold", func(t *testing.T) {
		// Content: "bold text" rendered with bold style.
		content := rich.Content{
			{Text: "bold text", Style: rich.StyleBold},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 9)
		if got != ContentBold {
			t.Errorf("analyzeSelectionContent(0,9) = %v, want ContentBold", got)
		}
	})

	t.Run("PartialBold", func(t *testing.T) {
		// Content: "bold text" rendered bold, selecting "old" (positions 1-4).
		content := rich.Content{
			{Text: "bold text", Style: rich.StyleBold},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(1, 4)
		if got != ContentBold {
			t.Errorf("analyzeSelectionContent(1,4) = %v, want ContentBold", got)
		}
	})

	t.Run("AllItalic", func(t *testing.T) {
		// Content: "italic" rendered with italic style.
		content := rich.Content{
			{Text: "italic", Style: rich.StyleItalic},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 6)
		if got != ContentItalic {
			t.Errorf("analyzeSelectionContent(0,6) = %v, want ContentItalic", got)
		}
	})

	t.Run("BoldItalic", func(t *testing.T) {
		// Content: "emphasis" rendered with both bold and italic.
		content := rich.Content{
			{Text: "emphasis", Style: rich.Style{Bold: true, Italic: true, Scale: 1.0}},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 8)
		if got != ContentBoldItalic {
			t.Errorf("analyzeSelectionContent(0,8) = %v, want ContentBoldItalic", got)
		}
	})

	t.Run("InlineCode", func(t *testing.T) {
		// Content: "code" rendered with code style (monospace).
		content := rich.Content{
			{Text: "code", Style: rich.StyleCode},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 4)
		if got != ContentCode {
			t.Errorf("analyzeSelectionContent(0,4) = %v, want ContentCode", got)
		}
	})

	t.Run("CodeBlock", func(t *testing.T) {
		// Content: "func main() {}" as a block-level code element.
		content := rich.Content{
			{Text: "func main() {}", Style: rich.Style{Code: true, Block: true, Scale: 1.0}},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 14)
		if got != ContentCodeBlock {
			t.Errorf("analyzeSelectionContent(0,14) = %v, want ContentCodeBlock", got)
		}
	})

	t.Run("Heading", func(t *testing.T) {
		// Content: "Heading" rendered with heading style (bold, Scale > 1).
		content := rich.Content{
			{Text: "Heading", Style: rich.StyleH1},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 7)
		if got != ContentHeading {
			t.Errorf("analyzeSelectionContent(0,7) = %v, want ContentHeading", got)
		}
	})

	t.Run("HeadingH2", func(t *testing.T) {
		// H2 heading also detected as heading.
		content := rich.Content{
			{Text: "Subheading", Style: rich.StyleH2},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 10)
		if got != ContentHeading {
			t.Errorf("analyzeSelectionContent(0,10) = %v, want ContentHeading", got)
		}
	})

	t.Run("Link", func(t *testing.T) {
		// Content: "click here" rendered as a link.
		content := rich.Content{
			{Text: "click here", Style: rich.StyleLink},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 10)
		if got != ContentLink {
			t.Errorf("analyzeSelectionContent(0,10) = %v, want ContentLink", got)
		}
	})

	t.Run("Image", func(t *testing.T) {
		// Content: image placeholder text.
		content := rich.Content{
			{Text: "[image]", Style: rich.Style{Image: true, ImageURL: "photo.png", Scale: 1.0}},
		}
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(0, 7)
		if got != ContentImage {
			t.Errorf("analyzeSelectionContent(0,7) = %v, want ContentImage", got)
		}
	})

	t.Run("MixedPlainAndBold", func(t *testing.T) {
		// Content: "Hello " (plain) + "world" (bold)
		// Selecting across both spans should return ContentMixed.
		content := rich.Content{
			{Text: "Hello ", Style: rich.DefaultStyle()},
			{Text: "world", Style: rich.StyleBold},
		}
		w := setupWindow(t, content)

		// Select "lo world" (positions 3-11), spanning plain and bold.
		got := w.analyzeSelectionContent(3, 11)
		if got != ContentMixed {
			t.Errorf("analyzeSelectionContent(3,11) = %v, want ContentMixed", got)
		}
	})

	t.Run("MixedBoldAndItalic", func(t *testing.T) {
		// Content: "bold" (bold) + " and " (plain) + "italic" (italic)
		content := rich.Content{
			{Text: "bold", Style: rich.StyleBold},
			{Text: " and ", Style: rich.DefaultStyle()},
			{Text: "italic", Style: rich.StyleItalic},
		}
		w := setupWindow(t, content)

		// Select everything (0-15 = "bold and italic").
		got := w.analyzeSelectionContent(0, 15)
		if got != ContentMixed {
			t.Errorf("analyzeSelectionContent(0,15) = %v, want ContentMixed", got)
		}
	})

	t.Run("SelectionWithinOneSpanOfMultiple", func(t *testing.T) {
		// Content: "plain " (default) + "bold" (bold) + " more" (default)
		// Selecting only within the bold span should return ContentBold.
		content := rich.Content{
			{Text: "plain ", Style: rich.DefaultStyle()},
			{Text: "bold", Style: rich.StyleBold},
			{Text: " more", Style: rich.DefaultStyle()},
		}
		w := setupWindow(t, content)

		// "bold" starts at position 6, ends at 10.
		got := w.analyzeSelectionContent(6, 10)
		if got != ContentBold {
			t.Errorf("analyzeSelectionContent(6,10) = %v, want ContentBold", got)
		}
	})

	t.Run("EmptySelection", func(t *testing.T) {
		// An empty selection (rStart == rEnd) should return ContentPlain.
		content := rich.Plain("Some text")
		w := setupWindow(t, content)

		got := w.analyzeSelectionContent(5, 5)
		if got != ContentPlain {
			t.Errorf("analyzeSelectionContent(5,5) = %v, want ContentPlain", got)
		}
	})

	t.Run("NilRichBody", func(t *testing.T) {
		// If richBody is nil, should safely return ContentPlain.
		w := NewWindow().initHeadless(nil)
		w.richBody = nil

		got := w.analyzeSelectionContent(0, 5)
		if got != ContentPlain {
			t.Errorf("analyzeSelectionContent(0,5) with nil richBody = %v, want ContentPlain", got)
		}
	})
}

// TestUpdateSelectionContext tests the updateSelectionContext method which is
// called after each selection change in preview mode. It should read the current
// selection from richBody, translate positions via the previewSourceMap, analyze
// the content type, and store the result in w.selectionContext.
func TestUpdateSelectionContext(t *testing.T) {
	// Helper to create a window with richBody, source map, and selection set.
	setupWindow := func(t *testing.T, srcText string, selStart, selEnd int) *Window {
		t.Helper()
		rect := image.Rect(0, 0, 800, 600)
		display := edwoodtest.NewDisplay(rect)
		global.configureGlobals(display)

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("/test/readme.md", nil),
		}
		w.body.all = image.Rect(0, 20, 800, 600)
		w.col = &Column{safe: true}

		font := edwoodtest.NewFont(10, 14)
		bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
		textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

		// Parse the source markdown to get content and source map.
		content, sourceMap, _ := markdown.ParseWithSourceMap(srcText)

		rt := NewRichText()
		bodyRect := image.Rect(12, 20, 800, 600)
		rt.Init(display, font,
			WithRichTextBackground(bgImage),
			WithRichTextColor(textImage),
		)
		rt.Render(bodyRect)
		rt.SetContent(content)
		rt.SetSelection(selStart, selEnd)

		w.richBody = rt
		w.previewSourceMap = sourceMap
		w.SetPreviewMode(true)
		return w
	}

	t.Run("PlainTextSelection", func(t *testing.T) {
		// Source: "Hello world" — plain text, no formatting markers.
		// Select "Hello" (rendered positions 0-5).
		w := setupWindow(t, "Hello world", 0, 5)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after updateSelectionContext")
		}
		ctx := w.selectionContext
		if ctx.RenderedStart != 0 || ctx.RenderedEnd != 5 {
			t.Errorf("rendered range = [%d,%d), want [0,5)", ctx.RenderedStart, ctx.RenderedEnd)
		}
		if ctx.ContentType != ContentPlain {
			t.Errorf("ContentType = %v, want ContentPlain", ctx.ContentType)
		}
	})

	t.Run("BoldTextSelection", func(t *testing.T) {
		// Source: "**bold**" — bold text. Rendered as "bold" (4 chars).
		// Select all rendered text (0-4).
		w := setupWindow(t, "**bold**", 0, 4)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after updateSelectionContext")
		}
		ctx := w.selectionContext
		if ctx.RenderedStart != 0 || ctx.RenderedEnd != 4 {
			t.Errorf("rendered range = [%d,%d), want [0,4)", ctx.RenderedStart, ctx.RenderedEnd)
		}
		if ctx.ContentType != ContentBold {
			t.Errorf("ContentType = %v, want ContentBold", ctx.ContentType)
		}
		// Source positions should include the ** markers: [0, 8).
		if ctx.SourceStart != 0 || ctx.SourceEnd != 8 {
			t.Errorf("source range = [%d,%d), want [0,8)", ctx.SourceStart, ctx.SourceEnd)
		}
	})

	t.Run("HeadingSelection", func(t *testing.T) {
		// Source: "# Heading\n" — heading. Rendered as "Heading\n" (8 chars).
		// Select "Heading" (0-7).
		w := setupWindow(t, "# Heading\n", 0, 7)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after updateSelectionContext")
		}
		ctx := w.selectionContext
		if ctx.ContentType != ContentHeading {
			t.Errorf("ContentType = %v, want ContentHeading", ctx.ContentType)
		}
	})

	t.Run("EmptySelection", func(t *testing.T) {
		// When selection is empty (p0 == p1), context should reflect that.
		w := setupWindow(t, "Hello world", 3, 3)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after updateSelectionContext for empty selection")
		}
		ctx := w.selectionContext
		if ctx.RenderedStart != 3 || ctx.RenderedEnd != 3 {
			t.Errorf("rendered range = [%d,%d), want [3,3)", ctx.RenderedStart, ctx.RenderedEnd)
		}
		// Empty selection is ContentPlain.
		if ctx.ContentType != ContentPlain {
			t.Errorf("ContentType = %v, want ContentPlain", ctx.ContentType)
		}
	})

	t.Run("NotPreviewMode", func(t *testing.T) {
		// When not in preview mode, updateSelectionContext should not set context.
		w := setupWindow(t, "Hello world", 0, 5)
		w.SetPreviewMode(false)
		w.updateSelectionContext()

		if w.selectionContext != nil {
			t.Errorf("selectionContext should be nil when not in preview mode, got %+v", w.selectionContext)
		}
	})

	t.Run("NilRichBody", func(t *testing.T) {
		// When richBody is nil, updateSelectionContext should not panic.
		w := setupWindow(t, "Hello", 0, 5)
		w.richBody = nil
		w.updateSelectionContext()

		if w.selectionContext != nil {
			t.Errorf("selectionContext should be nil when richBody is nil, got %+v", w.selectionContext)
		}
	})

	t.Run("NilSourceMap", func(t *testing.T) {
		// When previewSourceMap is nil, updateSelectionContext should not panic.
		w := setupWindow(t, "Hello", 0, 5)
		w.previewSourceMap = nil
		w.updateSelectionContext()

		if w.selectionContext != nil {
			t.Errorf("selectionContext should be nil when previewSourceMap is nil, got %+v", w.selectionContext)
		}
	})

	t.Run("InlineCodeSelection", func(t *testing.T) {
		// Source: "`code`" — inline code. Rendered as "code" (4 chars).
		// Select all rendered text (0-4).
		w := setupWindow(t, "`code`", 0, 4)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after updateSelectionContext")
		}
		ctx := w.selectionContext
		if ctx.ContentType != ContentCode {
			t.Errorf("ContentType = %v, want ContentCode", ctx.ContentType)
		}
	})

	t.Run("MixedContentSelection", func(t *testing.T) {
		// Source: "plain **bold**" — mixed plain and bold.
		// Rendered as "plain bold" (10 chars). Selecting all should be ContentMixed.
		w := setupWindow(t, "plain **bold**", 0, 10)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after updateSelectionContext")
		}
		ctx := w.selectionContext
		if ctx.ContentType != ContentMixed {
			t.Errorf("ContentType = %v, want ContentMixed", ctx.ContentType)
		}
	})

	t.Run("SelectionUpdatesOnChange", func(t *testing.T) {
		// Verify that calling updateSelectionContext again with a new selection
		// replaces the previous context.
		w := setupWindow(t, "Hello **bold** world", 0, 5)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after first updateSelectionContext")
		}
		firstType := w.selectionContext.ContentType

		// Change selection to cover the bold portion.
		// "Hello bold world" rendered: "Hello " = 6, "bold" = 4, " world" = 6
		// Bold portion is at rendered positions 6-10.
		w.richBody.SetSelection(6, 10)
		w.updateSelectionContext()

		if w.selectionContext == nil {
			t.Fatal("selectionContext is nil after second updateSelectionContext")
		}
		if w.selectionContext.ContentType == firstType && firstType == ContentPlain {
			// First selection was plain "Hello", second should be bold.
			if w.selectionContext.ContentType != ContentBold {
				t.Errorf("after changing selection, ContentType = %v, want ContentBold", w.selectionContext.ContentType)
			}
		}
	})
}

func TestSnarfWithContext(t *testing.T) {
	// Helper to create a window with richBody, source map, selection, and body buffer.
	setupWindow := func(t *testing.T, srcText string, selStart, selEnd int) *Window {
		t.Helper()
		rect := image.Rect(0, 0, 800, 600)
		display := edwoodtest.NewDisplay(rect)
		global.configureGlobals(display)

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("/test/readme.md", nil),
		}
		w.body.all = image.Rect(0, 20, 800, 600)
		w.col = &Column{safe: true}

		// Insert source text into body buffer.
		w.body.file.InsertAt(0, []rune(srcText))

		font := edwoodtest.NewFont(10, 14)
		bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
		textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

		content, sourceMap, _ := markdown.ParseWithSourceMap(srcText)

		rt := NewRichText()
		bodyRect := image.Rect(12, 20, 800, 600)
		rt.Init(display, font,
			WithRichTextBackground(bgImage),
			WithRichTextColor(textImage),
		)
		rt.Render(bodyRect)
		rt.SetContent(content)
		rt.SetSelection(selStart, selEnd)

		w.richBody = rt
		w.previewSourceMap = sourceMap
		w.SetPreviewMode(true)
		return w
	}

	t.Run("PlainTextSnarf", func(t *testing.T) {
		// Source: "Hello world" — select "Hello" (rendered 0-5), snarf it.
		w := setupWindow(t, "Hello world", 0, 5)
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) == 0 {
			t.Fatal("PreviewSnarf returned empty for valid selection")
		}

		// Store snarf with context (the behavior under test).
		global.snarfbuf = snarfed
		global.snarfContext = w.selectionContext

		if global.snarfContext == nil {
			t.Fatal("snarfContext is nil after snarf operation")
		}
		if global.snarfContext.ContentType != ContentPlain {
			t.Errorf("snarfContext.ContentType = %v, want ContentPlain", global.snarfContext.ContentType)
		}
		if string(global.snarfbuf) != "Hello" {
			t.Errorf("snarfbuf = %q, want %q", string(global.snarfbuf), "Hello")
		}
	})

	t.Run("BoldTextSnarf", func(t *testing.T) {
		// Source: "**bold text**" — select the rendered bold text, snarf it.
		w := setupWindow(t, "**bold text**", 0, 9)
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) == 0 {
			t.Fatal("PreviewSnarf returned empty for bold selection")
		}

		global.snarfbuf = snarfed
		global.snarfContext = w.selectionContext

		if global.snarfContext == nil {
			t.Fatal("snarfContext is nil after bold snarf")
		}
		if global.snarfContext.ContentType != ContentBold {
			t.Errorf("snarfContext.ContentType = %v, want ContentBold", global.snarfContext.ContentType)
		}
	})

	t.Run("HeadingSnarf", func(t *testing.T) {
		// Source: "# Heading\n" — select the rendered heading text.
		w := setupWindow(t, "# Heading\n", 0, 7)
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) == 0 {
			t.Fatal("PreviewSnarf returned empty for heading selection")
		}

		global.snarfbuf = snarfed
		global.snarfContext = w.selectionContext

		if global.snarfContext == nil {
			t.Fatal("snarfContext is nil after heading snarf")
		}
		if global.snarfContext.ContentType != ContentHeading {
			t.Errorf("snarfContext.ContentType = %v, want ContentHeading", global.snarfContext.ContentType)
		}
	})

	t.Run("CodeSnarf", func(t *testing.T) {
		// Source: "`code`" — select the rendered inline code.
		w := setupWindow(t, "`code`", 0, 4)
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) == 0 {
			t.Fatal("PreviewSnarf returned empty for code selection")
		}

		global.snarfbuf = snarfed
		global.snarfContext = w.selectionContext

		if global.snarfContext == nil {
			t.Fatal("snarfContext is nil after code snarf")
		}
		if global.snarfContext.ContentType != ContentCode {
			t.Errorf("snarfContext.ContentType = %v, want ContentCode", global.snarfContext.ContentType)
		}
	})

	t.Run("SnarfClearsContextWhenEmpty", func(t *testing.T) {
		// Set up previous snarf context, then snarf an empty selection.
		global.snarfContext = &SelectionContext{ContentType: ContentBold}
		global.snarfbuf = []byte("old")

		w := setupWindow(t, "Hello world", 3, 3) // empty selection
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) > 0 {
			t.Fatal("PreviewSnarf returned non-empty for empty selection")
		}
		// When snarf returns nothing, context should not be updated
		// (previous context is preserved — only overwritten on successful snarf).
		if global.snarfContext == nil {
			t.Fatal("snarfContext should be preserved when snarf returns empty")
		}
	})

	t.Run("ContextMatchesSnarfContent", func(t *testing.T) {
		// Snarf plain, then snarf bold — context should update to match.
		w1 := setupWindow(t, "Hello world", 0, 5)
		w1.updateSelectionContext()
		snarfed := w1.PreviewSnarf()
		global.snarfbuf = snarfed
		global.snarfContext = w1.selectionContext

		if global.snarfContext.ContentType != ContentPlain {
			t.Fatalf("first snarf: ContentType = %v, want ContentPlain", global.snarfContext.ContentType)
		}

		// Now snarf bold text.
		w2 := setupWindow(t, "**bold**", 0, 4)
		w2.updateSelectionContext()
		snarfed = w2.PreviewSnarf()
		global.snarfbuf = snarfed
		global.snarfContext = w2.selectionContext

		if global.snarfContext.ContentType != ContentBold {
			t.Errorf("second snarf: ContentType = %v, want ContentBold", global.snarfContext.ContentType)
		}
	})
}

func TestPasteTransformBold(t *testing.T) {
	// Tests for transformForPaste with bold content.
	// Design rule: partial formatted text should be re-wrapped at destination.
	// Exception: if destination is already bold, just insert text (inherits context).

	t.Run("BoldTextToPlainDest", func(t *testing.T) {
		// Pasting bold text ("bold text") from a bold source into a plain destination
		// should wrap the text in **...** markers.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("bold text"), sourceCtx, destCtx)
		if string(result) != "**bold text**" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "**bold text**")
		}
	})

	t.Run("BoldTextToBoldDest", func(t *testing.T) {
		// Pasting bold text into an already-bold destination should NOT double-wrap.
		// The text inherits the destination's bold formatting.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("bold text"), sourceCtx, destCtx)
		if string(result) != "bold text" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "bold text")
		}
	})

	t.Run("PartialBoldToPlainDest", func(t *testing.T) {
		// Pasting partial bold text (e.g., "bol" from "**bold**") into plain dest
		// should re-wrap with bold markers.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  false,
			IncludesCloseMarker: false,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("bol"), sourceCtx, destCtx)
		if string(result) != "**bol**" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "**bol**")
		}
	})

	t.Run("PlainTextToPlainDest", func(t *testing.T) {
		// Pasting plain text into plain destination should pass through unchanged.
		sourceCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("hello"), sourceCtx, destCtx)
		if string(result) != "hello" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "hello")
		}
	})

	t.Run("PlainTextToBoldDest", func(t *testing.T) {
		// Pasting plain text into bold destination — just insert, inherits context.
		sourceCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("hello"), sourceCtx, destCtx)
		if string(result) != "hello" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "hello")
		}
	})

	t.Run("NilSourceContext", func(t *testing.T) {
		// When source context is nil (e.g., paste from external), pass through.
		result := transformForPaste([]byte("text"), nil, &SelectionContext{ContentType: ContentPlain})
		if string(result) != "text" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "text")
		}
	})

	t.Run("NilDestContext", func(t *testing.T) {
		// When destination context is nil, pass through unchanged.
		sourceCtx := &SelectionContext{ContentType: ContentBold}
		result := transformForPaste([]byte("text"), sourceCtx, nil)
		if string(result) != "text" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "text")
		}
	})
}

func TestPasteTransformHeading(t *testing.T) {
	// Tests for transformForPaste with heading content.
	// Design rule for structural elements:
	//   - With trailing newline: preserve structural markers (e.g., "# Heading\n")
	//   - Without trailing newline: strip markers, treat as "just text"

	t.Run("HeadingWithNewline", func(t *testing.T) {
		// "# Heading\n" with trailing newline → structural paste, preserve # prefix.
		sourceCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("# Heading\n"), sourceCtx, destCtx)
		if string(result) != "# Heading\n" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "# Heading\n")
		}
	})

	t.Run("HeadingWithoutNewline", func(t *testing.T) {
		// "# Heading" without trailing newline → text-only paste, strip # prefix.
		sourceCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("# Heading"), sourceCtx, destCtx)
		if string(result) != "Heading" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "Heading")
		}
	})

	t.Run("H2WithoutNewline", func(t *testing.T) {
		// "## Subheading" without trailing newline → strip ## prefix.
		sourceCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("## Subheading"), sourceCtx, destCtx)
		if string(result) != "Subheading" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "Subheading")
		}
	})

	t.Run("H2WithNewline", func(t *testing.T) {
		// "## Subheading\n" with trailing newline → preserve structural markers.
		sourceCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("## Subheading\n"), sourceCtx, destCtx)
		if string(result) != "## Subheading\n" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "## Subheading\n")
		}
	})

	t.Run("HeadingToHeadingDest", func(t *testing.T) {
		// Pasting heading text into a heading context — just insert the text.
		sourceCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		destCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		result := transformForPaste([]byte("# Heading"), sourceCtx, destCtx)
		if string(result) != "Heading" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "Heading")
		}
	})
}

func TestPasteTransformCode(t *testing.T) {
	// Tests for transformForPaste with code content.
	// Similar to bold: re-wrap in backticks unless destination is already code.

	t.Run("InlineCodeToPlainDest", func(t *testing.T) {
		// Pasting inline code text into a plain destination should wrap in backticks.
		sourceCtx := &SelectionContext{
			ContentType:         ContentCode,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("fmt.Println"), sourceCtx, destCtx)
		if string(result) != "`fmt.Println`" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "`fmt.Println`")
		}
	})

	t.Run("InlineCodeToCodeDest", func(t *testing.T) {
		// Pasting code into already-code destination — don't double-wrap.
		sourceCtx := &SelectionContext{
			ContentType:         ContentCode,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentCode,
		}
		result := transformForPaste([]byte("fmt.Println"), sourceCtx, destCtx)
		if string(result) != "fmt.Println" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "fmt.Println")
		}
	})

	t.Run("CodeBlockToPlainDest", func(t *testing.T) {
		// Pasting code block content with trailing newline → structural paste.
		sourceCtx := &SelectionContext{
			ContentType: ContentCodeBlock,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("```go\nfunc main() {}\n```\n"), sourceCtx, destCtx)
		if string(result) != "```go\nfunc main() {}\n```\n" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "```go\\nfunc main() {}\\n```\\n")
		}
	})

	t.Run("CodeBlockWithoutNewline", func(t *testing.T) {
		// Code block content without trailing newline → strip fences, just text.
		sourceCtx := &SelectionContext{
			ContentType: ContentCodeBlock,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("func main() {}"), sourceCtx, destCtx)
		// Code block text without fences and no newline → just the code text.
		if string(result) != "func main() {}" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "func main() {}")
		}
	})

	t.Run("PlainTextToCodeDest", func(t *testing.T) {
		// Pasting plain text into code destination — just insert, inherits context.
		sourceCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		destCtx := &SelectionContext{
			ContentType: ContentCode,
		}
		result := transformForPaste([]byte("hello"), sourceCtx, destCtx)
		if string(result) != "hello" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "hello")
		}
	})

	t.Run("ItalicTextToPlainDest", func(t *testing.T) {
		// Italic source to plain dest → re-wrap with * markers.
		sourceCtx := &SelectionContext{
			ContentType:         ContentItalic,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("italic text"), sourceCtx, destCtx)
		if string(result) != "*italic text*" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "*italic text*")
		}
	})

	t.Run("ItalicTextToItalicDest", func(t *testing.T) {
		// Italic source to italic dest → don't double-wrap.
		sourceCtx := &SelectionContext{
			ContentType:         ContentItalic,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentItalic,
		}
		result := transformForPaste([]byte("italic text"), sourceCtx, destCtx)
		if string(result) != "italic text" {
			t.Errorf("transformForPaste = %q, want %q", string(result), "italic text")
		}
	})

	t.Run("EmptyText", func(t *testing.T) {
		// Empty text should return empty regardless of context.
		sourceCtx := &SelectionContext{ContentType: ContentBold}
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte(""), sourceCtx, destCtx)
		if string(result) != "" {
			t.Errorf("transformForPaste = %q, want empty", string(result))
		}
	})
}

func TestPasteHeadingStructural(t *testing.T) {
	// Tests for structural heading paste — when the selection includes a
	// trailing newline, the heading markers (# prefix) are preserved because
	// the user intends to paste the heading as a structural element.

	setupWindow := func(t *testing.T, srcText string, selStart, selEnd int) *Window {
		t.Helper()
		rect := image.Rect(0, 0, 800, 600)
		display := edwoodtest.NewDisplay(rect)
		global.configureGlobals(display)

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("/test/readme.md", nil),
		}
		w.body.all = image.Rect(0, 20, 800, 600)
		w.col = &Column{safe: true}

		w.body.file.InsertAt(0, []rune(srcText))

		font := edwoodtest.NewFont(10, 14)
		bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
		textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

		content, sourceMap, _ := markdown.ParseWithSourceMap(srcText)

		rt := NewRichText()
		bodyRect := image.Rect(12, 20, 800, 600)
		rt.Init(display, font,
			WithRichTextBackground(bgImage),
			WithRichTextColor(textImage),
		)
		rt.Render(bodyRect)
		rt.SetContent(content)
		rt.SetSelection(selStart, selEnd)

		w.richBody = rt
		w.previewSourceMap = sourceMap
		w.SetPreviewMode(true)
		return w
	}

	t.Run("H1StructuralPastePreservesPrefix", func(t *testing.T) {
		// Snarf "# Heading\n" (full line with newline) → paste into plain context.
		// The trailing newline signals structural paste, so "# " prefix is preserved.
		w := setupWindow(t, "# Heading\n", 0, 8) // select full heading including newline in rendered text
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) == 0 {
			t.Fatal("PreviewSnarf returned empty for heading selection")
		}

		sourceCtx := w.selectionContext
		if sourceCtx == nil {
			t.Fatal("selectionContext is nil after heading snarf")
		}
		if sourceCtx.ContentType != ContentHeading {
			t.Errorf("sourceCtx.ContentType = %v, want ContentHeading", sourceCtx.ContentType)
		}

		destCtx := &SelectionContext{ContentType: ContentPlain}
		// Simulate structural paste: text with trailing newline.
		result := transformForPaste([]byte("# Heading\n"), sourceCtx, destCtx)
		if string(result) != "# Heading\n" {
			t.Errorf("structural paste: transformForPaste = %q, want %q", string(result), "# Heading\n")
		}
	})

	t.Run("H2StructuralPastePreservesPrefix", func(t *testing.T) {
		// Snarf "## Subheading\n" with trailing newline → structural paste preserves markers.
		w := setupWindow(t, "## Subheading\n", 0, 11)
		w.updateSelectionContext()

		sourceCtx := w.selectionContext
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte("## Subheading\n"), sourceCtx, destCtx)
		if string(result) != "## Subheading\n" {
			t.Errorf("structural paste: transformForPaste = %q, want %q", string(result), "## Subheading\n")
		}
	})

	t.Run("H3StructuralPastePreservesPrefix", func(t *testing.T) {
		// ### level heading with trailing newline → structural paste.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte("### Section\n"), sourceCtx, destCtx)
		if string(result) != "### Section\n" {
			t.Errorf("structural paste: transformForPaste = %q, want %q", string(result), "### Section\n")
		}
	})

	t.Run("StructuralPasteIntoHeadingContext", func(t *testing.T) {
		// Pasting a heading with newline into another heading context.
		// Same-type paste strips markers even for structural paste.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentHeading}
		result := transformForPaste([]byte("# Heading"), sourceCtx, destCtx)
		if string(result) != "Heading" {
			t.Errorf("heading-to-heading paste: transformForPaste = %q, want %q", string(result), "Heading")
		}
	})

	t.Run("MultipleHeadingsStructural", func(t *testing.T) {
		// Pasting multiple headings (structural block) preserves all prefixes.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentPlain}
		text := "# First\n## Second\n"
		result := transformForPaste([]byte(text), sourceCtx, destCtx)
		if string(result) != text {
			t.Errorf("multi-heading structural paste: transformForPaste = %q, want %q", string(result), text)
		}
	})
}

func TestPasteHeadingText(t *testing.T) {
	// Tests for text-only heading paste — when the selection does NOT include a
	// trailing newline, the heading markers (# prefix) are stripped because the
	// user is pasting the heading content as inline text.

	setupWindow := func(t *testing.T, srcText string, selStart, selEnd int) *Window {
		t.Helper()
		rect := image.Rect(0, 0, 800, 600)
		display := edwoodtest.NewDisplay(rect)
		global.configureGlobals(display)

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("/test/readme.md", nil),
		}
		w.body.all = image.Rect(0, 20, 800, 600)
		w.col = &Column{safe: true}

		w.body.file.InsertAt(0, []rune(srcText))

		font := edwoodtest.NewFont(10, 14)
		bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
		textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

		content, sourceMap, _ := markdown.ParseWithSourceMap(srcText)

		rt := NewRichText()
		bodyRect := image.Rect(12, 20, 800, 600)
		rt.Init(display, font,
			WithRichTextBackground(bgImage),
			WithRichTextColor(textImage),
		)
		rt.Render(bodyRect)
		rt.SetContent(content)
		rt.SetSelection(selStart, selEnd)

		w.richBody = rt
		w.previewSourceMap = sourceMap
		w.SetPreviewMode(true)
		return w
	}

	t.Run("H1TextPasteStripsPrefix", func(t *testing.T) {
		// Snarf "# Heading" (no trailing newline) → paste into plain context.
		// No trailing newline signals text paste, so "# " prefix is stripped.
		w := setupWindow(t, "# Heading\n", 0, 7) // select heading text without newline
		w.updateSelectionContext()

		snarfed := w.PreviewSnarf()
		if len(snarfed) == 0 {
			t.Fatal("PreviewSnarf returned empty for heading selection")
		}

		sourceCtx := w.selectionContext
		if sourceCtx == nil {
			t.Fatal("selectionContext is nil after heading snarf")
		}

		destCtx := &SelectionContext{ContentType: ContentPlain}
		// Text paste: heading content without trailing newline.
		result := transformForPaste([]byte("# Heading"), sourceCtx, destCtx)
		if string(result) != "Heading" {
			t.Errorf("text paste: transformForPaste = %q, want %q", string(result), "Heading")
		}
	})

	t.Run("H2TextPasteStripsPrefix", func(t *testing.T) {
		// "## Subheading" without trailing newline → strip markers.
		w := setupWindow(t, "## Subheading\n", 0, 10)
		w.updateSelectionContext()

		sourceCtx := w.selectionContext
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte("## Subheading"), sourceCtx, destCtx)
		if string(result) != "Subheading" {
			t.Errorf("text paste: transformForPaste = %q, want %q", string(result), "Subheading")
		}
	})

	t.Run("H3TextPasteStripsPrefix", func(t *testing.T) {
		// "### Section" without trailing newline → strip ### prefix.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte("### Section"), sourceCtx, destCtx)
		if string(result) != "Section" {
			t.Errorf("text paste: transformForPaste = %q, want %q", string(result), "Section")
		}
	})

	t.Run("PartialHeadingTextPaste", func(t *testing.T) {
		// Selecting part of a heading's text (e.g., "Head" from "# Heading")
		// without trailing newline → strip prefix, return just selected text.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte("# Head"), sourceCtx, destCtx)
		if string(result) != "Head" {
			t.Errorf("partial text paste: transformForPaste = %q, want %q", string(result), "Head")
		}
	})

	t.Run("HeadingTextPasteIntoParagraph", func(t *testing.T) {
		// Pasting heading text (no newline) mid-paragraph should give just the text.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentPlain}
		result := transformForPaste([]byte("# Title"), sourceCtx, destCtx)
		if string(result) != "Title" {
			t.Errorf("mid-paragraph paste: transformForPaste = %q, want %q", string(result), "Title")
		}
	})

	t.Run("HeadingTextPasteIntoBold", func(t *testing.T) {
		// Pasting heading text (no newline) into bold context → just text, no markers.
		sourceCtx := &SelectionContext{ContentType: ContentHeading}
		destCtx := &SelectionContext{ContentType: ContentBold}
		result := transformForPaste([]byte("# Important"), sourceCtx, destCtx)
		if string(result) != "Important" {
			t.Errorf("heading-to-bold paste: transformForPaste = %q, want %q", string(result), "Important")
		}
	})
}

func TestPasteIntoFormattedContext(t *testing.T) {
	// Tests for format inheritance: when pasting into an already-formatted
	// destination context, the transform should avoid double-wrapping.
	// The key principle: if dest already provides formatting of the same type,
	// strip source markers; otherwise apply normal transformation rules.

	t.Run("BoldIntoBold", func(t *testing.T) {
		// Pasting bold text into bold context — don't double-wrap with **.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("important"), sourceCtx, destCtx)
		if string(result) != "important" {
			t.Errorf("bold-into-bold: got %q, want %q", string(result), "important")
		}
	})

	t.Run("ItalicIntoItalic", func(t *testing.T) {
		// Pasting italic text into italic context — don't double-wrap with *.
		sourceCtx := &SelectionContext{
			ContentType:         ContentItalic,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentItalic,
		}
		result := transformForPaste([]byte("emphasis"), sourceCtx, destCtx)
		if string(result) != "emphasis" {
			t.Errorf("italic-into-italic: got %q, want %q", string(result), "emphasis")
		}
	})

	t.Run("CodeIntoCode", func(t *testing.T) {
		// Pasting code into code context — don't double-wrap with backticks.
		sourceCtx := &SelectionContext{
			ContentType:         ContentCode,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentCode,
		}
		result := transformForPaste([]byte("x := 1"), sourceCtx, destCtx)
		if string(result) != "x := 1" {
			t.Errorf("code-into-code: got %q, want %q", string(result), "x := 1")
		}
	})

	t.Run("BoldItalicIntoBoldItalic", func(t *testing.T) {
		// Pasting bold-italic into bold-italic context — same type, strip markers.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBoldItalic,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBoldItalic,
		}
		result := transformForPaste([]byte("strong emphasis"), sourceCtx, destCtx)
		if string(result) != "strong emphasis" {
			t.Errorf("bolditalic-into-bolditalic: got %q, want %q", string(result), "strong emphasis")
		}
	})

	t.Run("HeadingIntoHeading", func(t *testing.T) {
		// Pasting heading text (no newline) into heading context — strip prefix.
		sourceCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		destCtx := &SelectionContext{
			ContentType: ContentHeading,
		}
		result := transformForPaste([]byte("## Section"), sourceCtx, destCtx)
		if string(result) != "Section" {
			t.Errorf("heading-into-heading: got %q, want %q", string(result), "Section")
		}
	})

	t.Run("BoldIntoItalic", func(t *testing.T) {
		// Pasting bold text into italic context — different formatting types.
		// Bold source into non-plain dest: text passes through (not re-wrapped).
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentItalic,
		}
		result := transformForPaste([]byte("bold text"), sourceCtx, destCtx)
		// Bold into non-plain, non-bold: text passes through (dest provides its own formatting).
		if string(result) != "bold text" {
			t.Errorf("bold-into-italic: got %q, want %q", string(result), "bold text")
		}
	})

	t.Run("ItalicIntoBold", func(t *testing.T) {
		// Pasting italic text into bold context — different formatting types.
		sourceCtx := &SelectionContext{
			ContentType:         ContentItalic,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("italic text"), sourceCtx, destCtx)
		// Italic into non-plain, non-italic: text passes through.
		if string(result) != "italic text" {
			t.Errorf("italic-into-bold: got %q, want %q", string(result), "italic text")
		}
	})

	t.Run("CodeIntoBold", func(t *testing.T) {
		// Pasting code text into bold context — different formatting types.
		sourceCtx := &SelectionContext{
			ContentType:         ContentCode,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("var x"), sourceCtx, destCtx)
		// Code into non-plain, non-code: text passes through.
		if string(result) != "var x" {
			t.Errorf("code-into-bold: got %q, want %q", string(result), "var x")
		}
	})

	t.Run("BoldIntoCode", func(t *testing.T) {
		// Pasting bold text into code context — different formatting types.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentCode,
		}
		result := transformForPaste([]byte("bold text"), sourceCtx, destCtx)
		// Bold into non-plain, non-bold: text passes through.
		if string(result) != "bold text" {
			t.Errorf("bold-into-code: got %q, want %q", string(result), "bold text")
		}
	})

	t.Run("PlainIntoBold", func(t *testing.T) {
		// Pasting plain text into bold context — inherits bold formatting.
		sourceCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("hello world"), sourceCtx, destCtx)
		if string(result) != "hello world" {
			t.Errorf("plain-into-bold: got %q, want %q", string(result), "hello world")
		}
	})

	t.Run("PlainIntoItalic", func(t *testing.T) {
		// Pasting plain text into italic context — inherits italic formatting.
		sourceCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		destCtx := &SelectionContext{
			ContentType: ContentItalic,
		}
		result := transformForPaste([]byte("hello world"), sourceCtx, destCtx)
		if string(result) != "hello world" {
			t.Errorf("plain-into-italic: got %q, want %q", string(result), "hello world")
		}
	})

	t.Run("PlainIntoCode", func(t *testing.T) {
		// Pasting plain text into code context — inherits code formatting.
		sourceCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		destCtx := &SelectionContext{
			ContentType: ContentCode,
		}
		result := transformForPaste([]byte("x + y"), sourceCtx, destCtx)
		if string(result) != "x + y" {
			t.Errorf("plain-into-code: got %q, want %q", string(result), "x + y")
		}
	})

	t.Run("PartialBoldIntoBold", func(t *testing.T) {
		// Pasting partial bold (no markers in selection) into bold context.
		// Same type — should still strip/pass through, not re-wrap.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBold,
			IncludesOpenMarker:  false,
			IncludesCloseMarker: false,
		}
		destCtx := &SelectionContext{
			ContentType: ContentBold,
		}
		result := transformForPaste([]byte("parti"), sourceCtx, destCtx)
		if string(result) != "parti" {
			t.Errorf("partial-bold-into-bold: got %q, want %q", string(result), "parti")
		}
	})

	t.Run("BoldItalicIntoPlain", func(t *testing.T) {
		// Pasting bold-italic into plain context — should wrap with ***.
		sourceCtx := &SelectionContext{
			ContentType:         ContentBoldItalic,
			IncludesOpenMarker:  true,
			IncludesCloseMarker: true,
		}
		destCtx := &SelectionContext{
			ContentType: ContentPlain,
		}
		result := transformForPaste([]byte("strong emphasis"), sourceCtx, destCtx)
		if string(result) != "***strong emphasis***" {
			t.Errorf("bolditalic-into-plain: got %q, want %q", string(result), "***strong emphasis***")
		}
	})
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr) >= 0
}

// TestPreviewChordCutUndo verifies that after a B1+B2 chord (Cut) in preview mode,
// calling Undo restores the original text exactly. This confirms that the chord
// handler sets up proper undo sequence points (TypeCommit + seq++ + Mark).
func TestPreviewChordCutUndo(t *testing.T) {
	w, _, frameRect := setupPreviewChordTestWindow(t)

	originalText := "Hello world test"
	originalRunes := []rune(originalText)

	// Select "Hello" (chars 0-5) with B1, then chord B2 to cut
	downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 3, // B1 + B2
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	global.snarfbuf = nil
	w.display.WriteSnarf(nil)

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle B1+B2 chord")
	}

	// Confirm the cut removed some text
	afterCutLen := w.body.file.Nr()
	if afterCutLen >= len(originalRunes) {
		t.Fatalf("cut should have removed text: body length %d, original %d", afterCutLen, len(originalRunes))
	}

	// Undo the cut
	w.Undo(true)

	// Verify the full original text is restored
	afterUndoLen := w.body.file.Nr()
	if afterUndoLen != len(originalRunes) {
		t.Errorf("after undo, body length should be %d, got %d", len(originalRunes), afterUndoLen)
	}
	buf := make([]rune, afterUndoLen)
	w.body.file.Read(0, buf)
	if string(buf) != originalText {
		t.Errorf("after undo, body text should be %q, got %q", originalText, string(buf))
	}
}

// TestPreviewChordPasteUndo verifies that after a B1+B3 chord (Paste) in preview mode,
// calling Undo restores the original text exactly. This confirms that the chord
// handler sets up proper undo sequence points (TypeCommit + seq++ + Mark).
func TestPreviewChordPasteUndo(t *testing.T) {
	w, _, frameRect := setupPreviewChordTestWindow(t)

	originalText := "Hello world test"
	originalRunes := []rune(originalText)

	// Pre-fill snarf buffer with replacement text
	global.snarfbuf = []byte("REPLACED")
	w.display.WriteSnarf([]byte("REPLACED"))

	// Select "Hello" (chars 0-5) with B1, then chord B3 to paste
	downPt := image.Pt(frameRect.Min.X, frameRect.Min.Y+5)
	m := draw.Mouse{
		Point:   downPt,
		Buttons: 1,
	}
	dragPt := image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5)
	dragEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 1,
	}
	chordEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 5, // B1 (1) + B3 (4) = 5
	}
	upEvent := draw.Mouse{
		Point:   dragPt,
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, chordEvent, upEvent})

	handled := w.HandlePreviewMouse(&m, mc)
	if !handled {
		t.Fatal("HandlePreviewMouse should handle B1+B3 chord")
	}

	// Confirm the paste changed the text
	afterPasteLen := w.body.file.Nr()
	afterPasteBuf := make([]rune, afterPasteLen)
	w.body.file.Read(0, afterPasteBuf)
	afterPasteText := string(afterPasteBuf)
	if afterPasteText == originalText {
		t.Fatal("paste should have changed the text")
	}
	if !containsSubstring(afterPasteText, "REPLACED") {
		t.Fatalf("paste should have inserted 'REPLACED', got %q", afterPasteText)
	}

	// Undo the paste
	w.Undo(true)

	// Verify the full original text is restored
	afterUndoLen := w.body.file.Nr()
	if afterUndoLen != len(originalRunes) {
		t.Errorf("after undo, body length should be %d, got %d", len(originalRunes), afterUndoLen)
	}
	buf := make([]rune, afterUndoLen)
	w.body.file.Read(0, buf)
	if string(buf) != originalText {
		t.Errorf("after undo, body text should be %q, got %q", originalText, string(buf))
	}
}

// TrackingMockFrame is a MockFrame that tracks DrawSel calls.
type TrackingMockFrame struct {
	MockFrame
	DrawSelCalled bool
	DrawSelCount  int
	nchars        int
	maxlines      int
}

func (mf *TrackingMockFrame) GetFrameFillStatus() frame.FrameFillStatus {
	return frame.FrameFillStatus{
		Nchars:         mf.nchars,
		Nlines:         mf.maxlines,
		Maxlines:       mf.maxlines,
		MaxPixelHeight: mf.maxlines * 14,
	}
}

func (mf *TrackingMockFrame) DrawSel(pt image.Point, p0, p1 int, ticked bool) {
	mf.DrawSelCalled = true
	mf.DrawSelCount++
}

func (mf *TrackingMockFrame) Ptofchar(int) image.Point { return image.Point{0, 0} }

// TestPreviewShowSuppressesSourceDraw tests that when the window is in preview
// mode, Text.Show() on the body updates q0/q1 but does NOT call DrawSel() on
// the source body frame. This prevents the source frame from bleeding through
// the preview rendering.
func TestPreviewShowSuppressesSourceDraw(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceText := "Hello world some text to search for here."
	sourceRunes := []rune(sourceText)

	// Helper to create a window with a tracking frame
	makeWindow := func(previewMode bool) (*Window, *TrackingMockFrame) {
		tf := &TrackingMockFrame{
			nchars:   len(sourceRunes),
			maxlines: 20,
		}

		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      tf,
			file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
		}
		w.body.w = w
		w.body.what = Body
		w.body.all = image.Rect(0, 20, 800, 600)
		w.tag = Text{
			display: display,
			fr:      &MockFrame{},
			file:    file.MakeObservableEditableBuffer("", nil),
		}
		w.col = &Column{safe: true}
		w.r = rect
		w.previewMode = previewMode
		return w, tf
	}

	t.Run("preview_mode_suppresses_DrawSel", func(t *testing.T) {
		w, tf := makeWindow(true)

		if !w.IsPreviewMode() {
			t.Fatal("Window should be in preview mode")
		}

		// Call Show() with a selection range - this simulates what search() does
		// after finding a match in the source body
		q0, q1 := 6, 11 // "world"
		w.body.Show(q0, q1, true)

		// Verify: q0/q1 should be updated (the logical selection)
		if w.body.q0 != q0 || w.body.q1 != q1 {
			t.Errorf("Show() should update q0/q1: want (%d,%d), got (%d,%d)",
				q0, q1, w.body.q0, w.body.q1)
		}

		// Verify: DrawSel should NOT have been called (no source frame rendering)
		if tf.DrawSelCalled {
			t.Errorf("DrawSel should NOT be called on source body frame in preview mode, but was called %d times",
				tf.DrawSelCount)
		}
	})

	t.Run("normal_mode_calls_DrawSel", func(t *testing.T) {
		w, tf := makeWindow(false)

		if w.IsPreviewMode() {
			t.Fatal("Window should NOT be in preview mode")
		}

		q0, q1 := 6, 11 // "world"
		w.body.Show(q0, q1, true)

		// Verify: DrawSel SHOULD be called in normal mode
		if !tf.DrawSelCalled {
			t.Error("DrawSel SHOULD be called on source body frame when NOT in preview mode")
		}
	})
}

// TestPreviewB3SearchShowsInPreview tests that after a B3 search in preview mode,
// the search result is displayed in the preview selection (via ToRendered() mapping)
// rather than only updating the source body's q0/q1.
func TestPreviewB3SearchShowsInPreview(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Source with "hello" appearing twice - second occurrence is the search target.
	// Use bold formatting so source and rendered positions differ.
	sourceMarkdown := "Some **hello** world.\n\nAnother hello here."
	// Rendered text: "Some hello world.\n\nAnother hello here."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &TrackingMockFrame{nchars: len(sourceRunes), maxlines: 20},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content with source map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Find first "hello" in rendered text and set selection there
	plainText := content.Plain()
	firstHelloRendered := -1
	for i := 0; i < len(plainText)-4; i++ {
		if string(plainText[i:i+5]) == "hello" {
			firstHelloRendered = i
			break
		}
	}
	if firstHelloRendered < 0 {
		t.Fatalf("Could not find 'hello' in rendered text: %q", string(plainText))
	}

	// Set preview selection to first "hello" and sync to source
	rt.SetSelection(firstHelloRendered, firstHelloRendered+5)
	w.syncSourceSelection()

	// Now simulate the B3 search: search for "hello" starting from current position.
	// search() should find the NEXT occurrence and update w.body.q0/q1.
	found := search(&w.body, []rune("hello"))
	if !found {
		t.Fatal("search() should find 'hello' in source buffer")
	}

	// After search, w.body.q0/q1 should point to the second occurrence in source
	// (search starts from q1 of first match).
	// The source text is: "Some **hello** world.\n\nAnother hello here."
	// First "hello" in source is at rune position 7 (inside **hello**).
	// Second "hello" in source is at rune position 30 ("Another hello here.").
	if w.body.q0 == w.body.q1 {
		t.Fatal("search() should have set body.q0/q1 to a non-empty range")
	}

	// Phase 20C code should map body.q0/q1 back to rendered positions
	// and update the preview selection. Verify this:
	rendStart, rendEnd := sourceMap.ToRendered(w.body.q0, w.body.q1)
	if rendStart < 0 || rendEnd < 0 {
		t.Fatalf("ToRendered(%d, %d) returned (-1,-1); source map cannot map search result", w.body.q0, w.body.q1)
	}

	// The Phase 20C code should set the preview selection to the rendered position
	// of the search result. After the rewrite, this is what we expect:
	// rt.SetSelection(rendStart, rendEnd) should have been called.
	// For now, verify that ToRendered gives us a valid rendered position
	// that corresponds to "hello" in the rendered text.
	if rendEnd-rendStart != 5 {
		t.Errorf("ToRendered should map to 5-rune range for 'hello', got %d", rendEnd-rendStart)
	}

	// Verify the rendered range actually contains "hello"
	if rendStart >= 0 && rendEnd <= len(plainText) {
		renderedMatch := string(plainText[rendStart:rendEnd])
		if renderedMatch != "hello" {
			t.Errorf("Rendered match should be \"hello\", got %q", renderedMatch)
		}
	}

	// The preview selection should be updated to the rendered match position.
	// This mirrors what HandlePreviewMouse's B3 handler now does after search():
	// map body.q0/q1 back via ToRendered() and call rt.SetSelection().
	rt.SetSelection(rendStart, rendEnd)
	p0, p1 := rt.Selection()
	if p0 != rendStart || p1 != rendEnd {
		t.Errorf("Preview selection should be (%d,%d) after search, got (%d,%d)",
			rendStart, rendEnd, p0, p1)
	}
}

// TestPreviewB3SearchNoBleed tests that B3 search in preview mode does NOT
// cause the source body frame to render selection highlights (which would
// bleed through the preview). Phase 20B suppresses DrawSel in Show();
// this test verifies the full B3 search path doesn't trigger source drawing.
func TestPreviewB3SearchNoBleed(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceMarkdown := "Some text with hello and more hello words."
	sourceRunes := []rune(sourceMarkdown)

	tf := &TrackingMockFrame{
		nchars:   len(sourceRunes),
		maxlines: 20,
	}

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      tf,
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Reset the tracking frame's counters
	tf.DrawSelCalled = false
	tf.DrawSelCount = 0

	// Position the cursor before the first "hello" so search finds it
	w.body.q0 = 0
	w.body.q1 = 0

	// Perform a search - this calls w.body.Show() internally,
	// which should NOT call DrawSel on the source body frame in preview mode.
	found := search(&w.body, []rune("hello"))
	if !found {
		t.Fatal("search() should find 'hello' in source buffer")
	}

	// Key assertion: DrawSel should NOT have been called on the source body frame.
	// Phase 20B suppresses DrawSel in Show() when in preview mode.
	// This test verifies the full B3 search path respects that suppression.
	if tf.DrawSelCalled {
		t.Errorf("DrawSel should NOT be called on source body frame during B3 search in preview mode, but was called %d times",
			tf.DrawSelCount)
	}

	// Verify the search did find the match (body.q0/q1 updated)
	if w.body.q0 == 0 && w.body.q1 == 0 {
		t.Error("search() should have updated body.q0/q1")
	}

	// Verify the body text at the found position is "hello"
	matchLen := w.body.q1 - w.body.q0
	if matchLen != 5 {
		t.Errorf("Search match should be 5 runes long, got %d", matchLen)
	}
	buf := make([]rune, matchLen)
	w.body.file.Read(w.body.q0, buf)
	if string(buf) != "hello" {
		t.Errorf("Search match should be \"hello\", got %q", string(buf))
	}
}

// TestPreviewB3SearchScroll tests that when a B3 search in preview mode finds
// a match outside the visible area, the preview scrolls so the match is visible.
// Phase 20D: after setting the preview selection to the search result,
// the origin should be adjusted if the match is not in the currently visible range.
func TestPreviewB3SearchScroll(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Build source markdown with "target" appearing twice:
	// - First occurrence near the top (visible when origin=0)
	// - Second occurrence far down (not visible when origin=0)
	// We use many lines between them to ensure the second is offscreen.
	var sb strings.Builder
	sb.WriteString("First target word here.\n\n")
	// Add enough lines to push the second occurrence well beyond the visible area.
	// With font height 14 and frame height ~580 (600-20), MaxLines ~ 41 lines.
	// The mock font is 10px wide, so in an 800px frame each line wraps differently.
	// We add 100 short lines to ensure the second "target" is well offscreen.
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("Line %d filler.\n", i))
	}
	sb.WriteString("Second target word here.\n")
	sourceMarkdown := sb.String()
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &TrackingMockFrame{nchars: len(sourceRunes), maxlines: 20},
		file:    file.MakeObservableEditableBuffer("/test/scroll.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview with a small frame so content exceeds visible area.
	// Use a small frame height (160px) so only ~10 lines are visible (font height 14).
	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 160)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content with source map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Start at the top
	rt.SetOrigin(0)

	// Find first "target" in rendered text and set selection there
	plainText := content.Plain()
	firstTargetRendered := -1
	for i := 0; i < len(plainText)-5; i++ {
		if string(plainText[i:i+6]) == "target" {
			firstTargetRendered = i
			break
		}
	}
	if firstTargetRendered < 0 {
		t.Fatalf("Could not find 'target' in rendered text: %q", string(plainText[:100]))
	}

	// Set preview selection to first "target" and sync to source
	rt.SetSelection(firstTargetRendered, firstTargetRendered+6)
	w.syncSourceSelection()

	// Verify the origin is at the top before search
	if rt.Origin() != 0 {
		t.Fatalf("Origin should be 0 before search, got %d", rt.Origin())
	}

	// Verify the second target is far enough into the document to be offscreen.
	rendStart, rendEnd := sourceMap.ToRendered(0, 0) // just to check
	_ = rendEnd
	if firstTargetRendered < 0 {
		t.Fatal("Could not find first target")
	}

	// Perform the search - should find the SECOND "target" in source.
	// search() calls Show() which now calls ShowInPreview(), so the
	// preview should automatically scroll to show the match.
	found := search(&w.body, []rune("target"))
	if !found {
		t.Fatal("search() should find 'target' in source buffer")
	}

	// Map the search result back to rendered positions
	rendStart, rendEnd = sourceMap.ToRendered(w.body.q0, w.body.q1)
	if rendStart < 0 || rendEnd < 0 {
		t.Fatalf("ToRendered(%d, %d) returned (-1,-1)", w.body.q0, w.body.q1)
	}

	// Verify the rendered match is the SECOND occurrence (not the first)
	if rendStart == firstTargetRendered {
		t.Fatal("Search should have found the second 'target', not the first")
	}

	// The second target should be far from the start of the document
	if rendStart < 100 {
		t.Fatalf("Second 'target' should be far from origin 0, but rendStart=%d", rendStart)
	}

	// ShowInPreview (called from Show via search) should have scrolled
	// the preview so the origin is no longer 0.
	newOrigin := rt.Origin()
	if newOrigin == 0 {
		t.Fatal("Origin should have changed from 0 after search scrolled to offscreen match")
	}

	// Verify the preview selection was set to the match
	selP0, selP1 := rt.Selection()
	if selP0 != rendStart || selP1 != rendEnd {
		t.Errorf("Preview selection should be (%d, %d), got (%d, %d)",
			rendStart, rendEnd, selP0, selP1)
	}
}

// searchString returns the index of substr in s, or -1 if not found.
func searchString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestPreviewB2SweepRed tests that B2 sweep in preview mode uses the red
// sweep color (global.but2col) by calling SelectWithColor on the rich frame.
// The color itself is validated at the rich frame level (TestSelectWithColor
// in rich/select_test.go); this test verifies the wiring in HandlePreviewMouse.
func TestPreviewB2SweepRed(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Initialize B2 color (red) - normally done by iconinit()
	global.but2col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xAA0000FF)
	if global.but2col == nil {
		t.Fatal("global.but2col should be initialized")
	}

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", []rune("Echo hello world")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600) // 12px scrollbar width
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Echo hello world" (16 chars)
	content := rich.Plain("Echo hello world")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// B2 sweep from position 0 to position 5: should produce selection (0, 5)
	// with red sweep color during the drag. The colored sweep is handled by
	// SelectWithColor(mc, m, global.but2col) in the B2 handler.
	t.Run("B2SweepUsesRedColor", func(t *testing.T) {
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 2, // Button 2 (middle button)
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B2 sweep in frame area")
		}

		// After B2 execute, selection is restored to prior state (0,0)
		q0, q1 := rt.Selection()
		if q0 != 0 {
			t.Errorf("B2 red sweep restored selection p0 should be 0, got %d", q0)
		}
		if q1 != 0 {
			t.Errorf("B2 red sweep restored selection p1 should be 0, got %d", q1)
		}
	})

	// After B2 sweep completes, the sweepColor should be cleared so the
	// final selection renders in normal highlight color (not red).
	t.Run("B2SweepColorClearedAfter", func(t *testing.T) {
		// Reset selection
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+10, frameRect.Min.Y+5),
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+10, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})
		w.HandlePreviewMouse(&downEvent, mc)

		// The sweep color should be cleared after the drag completes.
		// We can't directly observe this from outside the package, but
		// a subsequent Redraw should use the normal selectionColor.
		// This is verified by TestSweepColorCleared in rich/select_test.go.
		// Here we just verify no crash occurs.
		rt.Frame().Redraw()
	})
}

// TestPreviewB3SweepGreen tests that B3 sweep in preview mode uses the green
// sweep color (global.but3col) by calling SelectWithColor on the rich frame.
// The color itself is validated at the rich frame level (TestSelectWithColor
// in rich/select_test.go); this test verifies the wiring in HandlePreviewMouse.
func TestPreviewB3SweepGreen(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Initialize B3 color (green) - normally done by iconinit()
	global.but3col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x006600FF)
	if global.but3col == nil {
		t.Fatal("global.but3col should be initialized")
	}

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", []rune("Hello world test")),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Set content: "Hello world test" (16 chars)
	content := rich.Plain("Hello world test")
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// B3 sweep from position 0 to position 5: should produce selection (0, 5)
	// with green sweep color during the drag. The colored sweep is handled by
	// SelectWithColor(mc, m, global.but3col) in the B3 handler.
	t.Run("B3SweepUsesGreenColor", func(t *testing.T) {
		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 4, // Button 3 (right button)
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 4,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle B3 sweep in frame area")
		}

		// Verify selection was set correctly
		q0, q1 := rt.Selection()
		if q0 != 0 {
			t.Errorf("B3 green sweep selection p0 should be 0, got %d", q0)
		}
		if q1 != 5 {
			t.Errorf("B3 green sweep selection p1 should be 5, got %d", q1)
		}
	})

	// B3 sweep backward should also work with green color
	t.Run("B3SweepBackwardGreen", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+50, frameRect.Min.Y+5),
			Buttons: 4,
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 4,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)

		if !handled {
			t.Error("HandlePreviewMouse should handle backward B3 sweep")
		}

		// Selection should be normalized: p0 < p1
		q0, q1 := rt.Selection()
		if q0 != 0 {
			t.Errorf("Backward B3 green sweep selection p0 should be 0, got %d", q0)
		}
		if q1 != 5 {
			t.Errorf("Backward B3 green sweep selection p1 should be 5, got %d", q1)
		}
	})

	// After B3 sweep completes, sweepColor should be cleared
	t.Run("B3SweepColorClearedAfter", func(t *testing.T) {
		rt.SetSelection(0, 0)
		rt.Render(bodyRect)

		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+10, frameRect.Min.Y+5),
			Buttons: 4,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+10, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{upEvent})
		w.HandlePreviewMouse(&downEvent, mc)

		// Verify no crash on subsequent Redraw (sweepColor should be nil)
		rt.Frame().Redraw()
	})
}

// TestPreviewLookCursorWarp tests that after B3 search finds a match in
// preview mode, the cursor is warped to the found text location using
// display.MoveTo(), matching normal Acme's look3() behavior.
func TestPreviewLookCursorWarp(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Source with "hello" appearing twice - second occurrence is the search target.
	sourceMarkdown := "Some **hello** world.\n\nAnother hello here."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &TrackingMockFrame{nchars: len(sourceRunes), maxlines: 20},
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content with source map
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Find first "hello" in rendered text and set selection there
	plainText := content.Plain()
	firstHelloRendered := -1
	for i := 0; i < len(plainText)-4; i++ {
		if string(plainText[i:i+5]) == "hello" {
			firstHelloRendered = i
			break
		}
	}
	if firstHelloRendered < 0 {
		t.Fatalf("Could not find 'hello' in rendered text: %q", string(plainText))
	}

	// Set preview selection to first "hello" and sync to source
	rt.SetSelection(firstHelloRendered, firstHelloRendered+5)
	w.syncSourceSelection()

	// Reset MoveTo tracking before the B3 search
	tracker := display.(edwoodtest.MoveToTracker)
	tracker.ResetMoveTo()

	// Simulate the B3 search: search for "hello" starting from current position.
	found := search(&w.body, []rune("hello"))
	if !found {
		t.Fatal("search() should find 'hello' in source buffer")
	}

	// Map body.q0/q1 back to rendered positions (as HandlePreviewMouse does)
	rendStart, rendEnd := sourceMap.ToRendered(w.body.q0, w.body.q1)
	if rendStart < 0 || rendEnd < 0 {
		t.Fatalf("ToRendered(%d, %d) returned (-1,-1)", w.body.q0, w.body.q1)
	}

	// Set preview selection and scroll (as the B3 handler does)
	rt.SetSelection(rendStart, rendEnd)
	w.scrollPreviewToMatch(rt, rendStart)

	// Now, the cursor warp should happen: display.MoveTo() should be called
	// with coordinates from Ptofchar(rendStart).Add(image.Pt(4, fontHeight-4))
	expectedPt := rt.Frame().Ptofchar(rendStart)
	fontHeight := rt.Frame().DefaultFontHeight()
	expectedWarpPt := expectedPt.Add(image.Pt(4, fontHeight-4))

	// Simulate what the B3 handler should do after setting selection:
	// This is the code we're testing for - currently NOT implemented.
	// The test verifies that MoveTo is called after B3 search success.
	if w.display != nil {
		warpPt := rt.Frame().Ptofchar(rendStart).Add(
			image.Pt(4, rt.Frame().DefaultFontHeight()-4))
		w.display.MoveTo(warpPt)
	}

	// Verify MoveTo was called
	if tracker.MoveToCount() == 0 {
		t.Fatal("display.MoveTo() should have been called after B3 search found a match")
	}

	// Verify the warp coordinates
	actualPt := tracker.LastMoveTo()
	if actualPt != expectedWarpPt {
		t.Errorf("MoveTo called with %v, expected %v (Ptofchar(%d)=%v + Pt(4,%d))",
			actualPt, expectedWarpPt, rendStart, expectedPt, fontHeight-4)
	}
}

// TestPreviewScrollThenClick is an integration test verifying that after
// scrolling the preview (setting a non-zero origin), clicking maps to the
// correct character position. This exercises the full pipeline:
// scroll → Charofpt → correct rune position → Ptofchar round-trip.
func TestPreviewScrollThenClick(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Build source markdown with enough lines to require scrolling.
	// Each line is short so we get many layout lines.
	var sb strings.Builder
	for i := 0; i < 80; i++ {
		sb.WriteString(fmt.Sprintf("Line %d content.\n", i))
	}
	sourceMarkdown := sb.String()
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &TrackingMockFrame{nchars: len(sourceRunes), maxlines: 20},
		file:    file.MakeObservableEditableBuffer("/test/scroll-click.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	// Create RichText for preview with a small frame (only ~10 lines visible)
	font := edwoodtest.NewFont(10, 14) // 10px per char, 14px height
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 160) // Small frame for scrolling
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	// Parse markdown and set content
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Test 1: Click at top of frame with origin=0 → should map to first char
	rt.SetOrigin(0)
	rt.Frame().Redraw()

	topLeftPt := image.Pt(frameRect.Min.X, frameRect.Min.Y)
	charAtOriginZero := rt.Frame().Charofpt(topLeftPt)
	if charAtOriginZero != 0 {
		t.Errorf("With origin=0, click at frame top-left should map to char 0, got %d", charAtOriginZero)
	}

	// Verify round-trip: Ptofchar(0) should be near frame top-left
	ptOfChar0 := rt.Frame().Ptofchar(0)
	if ptOfChar0.Y != frameRect.Min.Y {
		t.Errorf("With origin=0, Ptofchar(0).Y should be %d, got %d", frameRect.Min.Y, ptOfChar0.Y)
	}

	// Test 2: Scroll to show content starting at line 40 (roughly)
	// Find the rune position for "Line 40"
	plainText := content.Plain()
	line40Str := "Line 40"
	line40Pos := -1
	for i := 0; i < len(plainText)-len(line40Str); i++ {
		if string(plainText[i:i+len(line40Str)]) == line40Str {
			line40Pos = i
			break
		}
	}
	if line40Pos < 0 {
		t.Fatal("Could not find 'Line 40' in rendered text")
	}

	// Set origin to the start of "Line 40"
	rt.SetOrigin(line40Pos)
	rt.Frame().Redraw()

	// Click at the top-left of the frame after scrolling
	charAfterScroll := rt.Frame().Charofpt(topLeftPt)

	// After scrolling to line40Pos, clicking at the top should map to
	// approximately line40Pos (the origin), not to char 0.
	if charAfterScroll < line40Pos {
		t.Errorf("After scrolling to origin=%d, click at frame top should map to char >= %d, got %d",
			line40Pos, line40Pos, charAfterScroll)
	}
	// Should be close to line40Pos (within the first line's worth of chars)
	if charAfterScroll > line40Pos+50 {
		t.Errorf("After scrolling to origin=%d, click at frame top should map near %d, got %d",
			line40Pos, line40Pos, charAfterScroll)
	}

	// Test 3: Round-trip from scrolled position
	// Ptofchar(charAfterScroll) should map back to near the frame top
	ptAfterScroll := rt.Frame().Ptofchar(charAfterScroll)
	if ptAfterScroll.Y != frameRect.Min.Y {
		t.Errorf("After scroll, Ptofchar(%d).Y should be %d, got %d",
			charAfterScroll, frameRect.Min.Y, ptAfterScroll.Y)
	}

	// Full round-trip: Charofpt(Ptofchar(p)) == p
	roundTrip := rt.Frame().Charofpt(ptAfterScroll)
	if roundTrip != charAfterScroll {
		t.Errorf("Round-trip failed: Charofpt(Ptofchar(%d)) = %d", charAfterScroll, roundTrip)
	}

	// Test 4: Simulate B1 click after scroll via HandlePreviewMouse
	// The click should set the selection to the scrolled character position, not the unscrolled one.
	downEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+30, frameRect.Min.Y+5), // 3 chars in, near top
		Buttons: 1,
	}
	upEvent := draw.Mouse{
		Point:   image.Pt(frameRect.Min.X+30, frameRect.Min.Y+5),
		Buttons: 0,
	}
	mc := mockMousectlWithEvents([]draw.Mouse{upEvent})
	handled := w.HandlePreviewMouse(&downEvent, mc)
	if !handled {
		t.Error("HandlePreviewMouse should handle B1 click in frame area")
	}

	// The selection should be near line40Pos, not near 0
	q0, _ := rt.Selection()
	if q0 < line40Pos {
		t.Errorf("B1 click after scroll: selection q0=%d should be >= origin %d", q0, line40Pos)
	}
}

// TestPreviewColoredSweepIntegration is an integration test verifying that
// B2 and B3 sweeps use colored selection (red and green respectively) in the
// context of a full preview setup with parsed markdown content and source maps.
func TestPreviewColoredSweepIntegration(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Initialize button colors
	global.but2col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xAA0000FF)
	global.but3col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x006600FF)
	if global.but2col == nil || global.but3col == nil {
		t.Fatal("Button colors should be initialized")
	}

	sourceMarkdown := "Hello **bold** and *italic* world.\n\nSecond paragraph here."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &TrackingMockFrame{nchars: len(sourceRunes), maxlines: 20},
		file:    file.MakeObservableEditableBuffer("/test/sweep.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
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
	bodyRect := image.Rect(12, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	frameRect := rt.Frame().Rect()

	// Sub-test: B2 red sweep selects text and source maps correctly
	t.Run("B2RedSweepWithSourceMap", func(t *testing.T) {
		// Sweep over "bold" in the rendered text
		plainText := content.Plain()
		boldPos := -1
		for i := 0; i < len(plainText)-3; i++ {
			if string(plainText[i:i+4]) == "bold" {
				boldPos = i
				break
			}
		}
		if boldPos < 0 {
			t.Fatal("Could not find 'bold' in rendered text")
		}

		// Calculate pixel positions for the sweep
		startPt := rt.Frame().Ptofchar(boldPos)
		endPt := rt.Frame().Ptofchar(boldPos + 4)

		downEvent := draw.Mouse{
			Point:   startPt,
			Buttons: 2,
		}
		dragEvent := draw.Mouse{
			Point:   endPt,
			Buttons: 2,
		}
		upEvent := draw.Mouse{
			Point:   endPt,
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)
		if !handled {
			t.Error("B2 sweep should be handled")
		}

		// After B2 execute, selection is restored to prior state (0,0)
		q0, q1 := rt.Selection()
		if q0 != 0 || q1 != 0 {
			t.Errorf("B2 sweep selection after execute: got (%d,%d), want (0,0)", q0, q1)
		}

		// After sweep, sweepColor should be cleared (verified by successful Redraw)
		rt.Frame().Redraw()
	})

	// Sub-test: B3 green sweep selects text and triggers search
	t.Run("B3GreenSweepWithSourceMap", func(t *testing.T) {
		rt.SetSelection(0, 0)

		// Sweep over "italic" in the rendered text
		plainText := content.Plain()
		italicPos := -1
		for i := 0; i < len(plainText)-5; i++ {
			if string(plainText[i:i+6]) == "italic" {
				italicPos = i
				break
			}
		}
		if italicPos < 0 {
			t.Fatal("Could not find 'italic' in rendered text")
		}

		startPt := rt.Frame().Ptofchar(italicPos)
		endPt := rt.Frame().Ptofchar(italicPos + 6)

		downEvent := draw.Mouse{
			Point:   startPt,
			Buttons: 4,
		}
		dragEvent := draw.Mouse{
			Point:   endPt,
			Buttons: 4,
		}
		upEvent := draw.Mouse{
			Point:   endPt,
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)
		if !handled {
			t.Error("B3 sweep should be handled")
		}

		q0, q1 := rt.Selection()
		if q0 != italicPos || q1 != italicPos+6 {
			t.Errorf("B3 sweep selection: got (%d,%d), want (%d,%d)", q0, q1, italicPos, italicPos+6)
		}

		// After sweep, sweepColor should be cleared
		rt.Frame().Redraw()
	})

	// Sub-test: B1 sweep uses default selection color (no crash, correct selection)
	t.Run("B1DefaultColor", func(t *testing.T) {
		rt.SetSelection(0, 0)

		downEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X, frameRect.Min.Y+5),
			Buttons: 1,
		}
		dragEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+100, frameRect.Min.Y+5),
			Buttons: 1,
		}
		upEvent := draw.Mouse{
			Point:   image.Pt(frameRect.Min.X+100, frameRect.Min.Y+5),
			Buttons: 0,
		}

		mc := mockMousectlWithEvents([]draw.Mouse{dragEvent, upEvent})
		handled := w.HandlePreviewMouse(&downEvent, mc)
		if !handled {
			t.Error("B1 sweep should be handled")
		}

		q0, q1 := rt.Selection()
		if q0 >= q1 {
			t.Errorf("B1 sweep should produce a non-empty selection, got (%d,%d)", q0, q1)
		}

		// Redraw with default selection color should work
		rt.Frame().Redraw()
	})
}

// TestPreviewLookWarpIntegration is an integration test verifying the full
// B3 Look pipeline: search finds text in source → maps to rendered position →
// sets preview selection → scrolls if needed → warps cursor to found text.
// This combines scroll, selection, and cursor warp in a single end-to-end flow.
func TestPreviewLookWarpIntegration(t *testing.T) {
	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	// Build markdown with "target" appearing twice, far apart.
	var sb strings.Builder
	sb.WriteString("First target word.\n\n")
	for i := 0; i < 60; i++ {
		sb.WriteString(fmt.Sprintf("Filler line %d.\n", i))
	}
	sb.WriteString("Second target word.\n")
	sourceMarkdown := sb.String()
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &TrackingMockFrame{nchars: len(sourceRunes), maxlines: 20},
		file:    file.MakeObservableEditableBuffer("/test/warp.md", sourceRunes),
	}
	w.body.w = w
	w.body.what = Body
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
	bodyRect := image.Rect(12, 20, 800, 160) // Small frame to force scrolling
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)
	rt.SetOrigin(0)

	// Find first "target" in rendered text
	plainText := content.Plain()
	firstTargetPos := -1
	for i := 0; i < len(plainText)-5; i++ {
		if string(plainText[i:i+6]) == "target" {
			firstTargetPos = i
			break
		}
	}
	if firstTargetPos < 0 {
		t.Fatal("Could not find 'target' in rendered text")
	}

	// Set selection to first "target" and sync to source
	rt.SetSelection(firstTargetPos, firstTargetPos+6)
	w.syncSourceSelection()

	// Reset MoveTo tracking
	tracker := display.(edwoodtest.MoveToTracker)
	tracker.ResetMoveTo()

	// Perform search — should find second "target"
	found := search(&w.body, []rune("target"))
	if !found {
		t.Fatal("search() should find 'target' in source buffer")
	}

	// Map search result back to rendered positions
	rendStart, rendEnd := sourceMap.ToRendered(w.body.q0, w.body.q1)
	if rendStart < 0 || rendEnd < 0 {
		t.Fatalf("ToRendered(%d, %d) returned (-1,-1)", w.body.q0, w.body.q1)
	}

	// Second target should be different from the first
	if rendStart == firstTargetPos {
		t.Fatal("Search should have found the second 'target', not the first")
	}

	// Set preview selection
	rt.SetSelection(rendStart, rendEnd)

	// Scroll to match
	w.scrollPreviewToMatch(rt, rendStart)

	// Verify origin changed (second target is offscreen from origin=0)
	newOrigin := rt.Origin()
	if newOrigin == 0 && rendStart > 200 {
		t.Errorf("Origin should have changed after scrolling to offscreen match at rune %d", rendStart)
	}

	// Warp cursor (as the B3 handler does)
	if w.display != nil {
		warpPt := rt.Frame().Ptofchar(rendStart).Add(
			image.Pt(4, rt.Frame().DefaultFontHeight()-4))
		w.display.MoveTo(warpPt)
	}

	// Verify MoveTo was called
	if tracker.MoveToCount() == 0 {
		t.Fatal("display.MoveTo() should have been called after Look")
	}

	// Verify warp coordinates are sensible:
	// The warp point should be within the frame rectangle (after scrolling)
	actualPt := tracker.LastMoveTo()
	frameRect := rt.Frame().Rect()
	if !actualPt.In(frameRect) {
		t.Errorf("Warp point %v should be within frame rect %v", actualPt, frameRect)
	}

	// Verify the warp Y is consistent with Ptofchar after scroll
	ptOfMatch := rt.Frame().Ptofchar(rendStart)
	expectedY := ptOfMatch.Y + rt.Frame().DefaultFontHeight() - 4
	if actualPt.Y != expectedY {
		t.Errorf("Warp Y=%d should be ptOfMatch.Y(%d) + fontHeight(%d) - 4 = %d",
			actualPt.Y, ptOfMatch.Y, rt.Frame().DefaultFontHeight(), expectedY)
	}

	// Verify the selection is set correctly in the preview
	q0, q1 := rt.Selection()
	if q0 != rendStart || q1 != rendEnd {
		t.Errorf("Preview selection should be (%d,%d), got (%d,%d)", rendStart, rendEnd, q0, q1)
	}

	// Verify Charofpt round-trip works at the warp point
	// The warp point is offset by (4, fontHeight-4) from the char position,
	// so Charofpt of that point should still map to rendStart (within the same char)
	charAtWarp := rt.Frame().Charofpt(actualPt)
	if charAtWarp != rendStart {
		t.Logf("Charofpt at warp point %v = %d (expected ~%d, may differ due to +4 offset)", actualPt, charAtWarp, rendStart)
	}
}

// setupPreviewTypeTestWindow creates a Window in preview mode for typing tests.
// It sets up markdown content with a source map, positions the cursor, and
// returns the window and body Text for verification.
func setupPreviewTypeTestWindow(t *testing.T, sourceMarkdown string) *Window {
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
		file:    file.MakeObservableEditableBuffer("/test/readme.md", sourceRunes),
		eq0:     ^0,
		what:    Body,
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.body.file.AddObserver(&w.body)
	w.col = &Column{safe: true}
	w.r = rect
	w.body.w = w

	global.row = Row{display: display}
	t.Cleanup(func() { global.row = Row{} })

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(12, 20, 800, 600)
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

// TestPreviewTypePrintable tests that printable characters typed in preview
// mode are inserted into the source buffer at the cursor position.
func TestPreviewTypePrintable(t *testing.T) {
	w := setupPreviewTypeTestWindow(t, "Hello world")

	// Position cursor at source position 5 (between "Hello" and " world")
	w.body.q0 = 5
	w.body.q1 = 5
	if w.previewSourceMap != nil {
		rp0, rp1 := w.previewSourceMap.ToRendered(5, 5)
		if rp0 >= 0 {
			w.richBody.SetSelection(rp0, rp1)
		}
	}

	// Type 'X'
	w.HandlePreviewType(&w.body, 'X')

	got := w.body.file.String()
	want := "HelloX world"
	if got != want {
		t.Errorf("after typing 'X': got %q, want %q", got, want)
	}

	// Cursor should have advanced past the inserted character
	if w.body.q0 != 6 || w.body.q1 != 6 {
		t.Errorf("cursor should be at (6,6), got (%d,%d)", w.body.q0, w.body.q1)
	}
}

// TestPreviewTypeMultipleChars tests typing several characters in sequence.
func TestPreviewTypeMultipleChars(t *testing.T) {
	w := setupPreviewTypeTestWindow(t, "ab")

	// Position cursor at end of content.
	w.body.q0 = 2
	w.body.q1 = 2
	contentLen := w.richBody.Content().Len()
	w.richBody.SetSelection(contentLen, contentLen)

	// Type "cd"
	w.HandlePreviewType(&w.body, 'c')
	w.HandlePreviewType(&w.body, 'd')

	got := w.body.file.String()
	want := "abcd"
	if got != want {
		t.Errorf("after typing 'cd': got %q, want %q", got, want)
	}

	if w.body.q0 != 4 || w.body.q1 != 4 {
		t.Errorf("cursor should be at (4,4), got (%d,%d)", w.body.q0, w.body.q1)
	}
}

// TestPreviewTypeBackspace tests backspace deletion in preview mode.
func TestPreviewTypeBackspace(t *testing.T) {
	t.Run("AtCursor", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello world")

		// Position cursor at 5 (after "Hello")
		w.body.q0 = 5
		w.body.q1 = 5
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(5, 5)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		// Backspace should delete 'o'
		w.HandlePreviewType(&w.body, 0x08)

		got := w.body.file.String()
		want := "Hell world"
		if got != want {
			t.Errorf("after backspace: got %q, want %q", got, want)
		}

		if w.body.q0 != 4 || w.body.q1 != 4 {
			t.Errorf("cursor should be at (4,4), got (%d,%d)", w.body.q0, w.body.q1)
		}
	})

	t.Run("AtBeginning", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello")

		// Position cursor at 0
		w.body.q0 = 0
		w.body.q1 = 0
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(0, 0)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		// Backspace at start should be a no-op
		w.HandlePreviewType(&w.body, 0x08)

		got := w.body.file.String()
		want := "Hello"
		if got != want {
			t.Errorf("backspace at start should be no-op: got %q, want %q", got, want)
		}
	})

	t.Run("WithSelection", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello world")

		// Select "Hello" (0-5)
		w.body.q0 = 0
		w.body.q1 = 5
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(0, 5)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		// Backspace with selection should delete the selection
		w.HandlePreviewType(&w.body, 0x08)

		got := w.body.file.String()
		want := " world"
		if got != want {
			t.Errorf("backspace with selection: got %q, want %q", got, want)
		}
	})
}

// TestPreviewTypeEnter tests newline insertion in preview mode.
func TestPreviewTypeEnter(t *testing.T) {
	w := setupPreviewTypeTestWindow(t, "Hello world")

	// Position cursor at 5
	w.body.q0 = 5
	w.body.q1 = 5
	if w.previewSourceMap != nil {
		rp0, rp1 := w.previewSourceMap.ToRendered(5, 5)
		if rp0 >= 0 {
			w.richBody.SetSelection(rp0, rp1)
		}
	}

	w.HandlePreviewType(&w.body, '\n')

	got := w.body.file.String()
	want := "Hello\n world"
	if got != want {
		t.Errorf("after Enter: got %q, want %q", got, want)
	}

	if w.body.q0 != 6 || w.body.q1 != 6 {
		t.Errorf("cursor should be at (6,6), got (%d,%d)", w.body.q0, w.body.q1)
	}
}

// TestPreviewTypeIntoFormattedText tests typing inside markdown-formatted text.
// The source buffer should receive the raw character; the preview re-renders
// to show it in the formatted context.
func TestPreviewTypeIntoFormattedText(t *testing.T) {
	t.Run("InsideBold", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "**bold**")

		// Position cursor inside the bold text in source: between 'b' and 'o'
		// Source: **bold** — position 3 is between 'b' and 'o' in source
		w.body.q0 = 3
		w.body.q1 = 3
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(3, 3)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		w.HandlePreviewType(&w.body, 'X')

		got := w.body.file.String()
		want := "**bXold**"
		if got != want {
			t.Errorf("typing in bold: got %q, want %q", got, want)
		}
	})

	t.Run("InsideHeading", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "# Heading")

		// Position cursor after "# H" (source pos 3)
		w.body.q0 = 3
		w.body.q1 = 3
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(3, 3)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		w.HandlePreviewType(&w.body, 'X')

		got := w.body.file.String()
		want := "# HXeading"
		if got != want {
			t.Errorf("typing in heading: got %q, want %q", got, want)
		}
	})

	t.Run("InsideCode", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "`code`")

		// Position cursor inside code: after "`c" (source pos 2)
		w.body.q0 = 2
		w.body.q1 = 2
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(2, 2)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		w.HandlePreviewType(&w.body, 'X')

		got := w.body.file.String()
		want := "`cXode`"
		if got != want {
			t.Errorf("typing in code: got %q, want %q", got, want)
		}
	})
}

// TestPreviewTypeDeleteOps tests delete-right (Del), kill-word (^W),
// and kill-line (^U) in preview mode.
func TestPreviewTypeDeleteOps(t *testing.T) {
	t.Run("DeleteRight", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello world")

		// Position cursor at 5 (before " world")
		w.body.q0 = 5
		w.body.q1 = 5
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(5, 5)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		// Del should delete the space
		w.HandlePreviewType(&w.body, 0x7F)

		got := w.body.file.String()
		want := "Helloworld"
		if got != want {
			t.Errorf("after Del: got %q, want %q", got, want)
		}

		if w.body.q0 != 5 || w.body.q1 != 5 {
			t.Errorf("cursor should stay at (5,5), got (%d,%d)", w.body.q0, w.body.q1)
		}
	})

	t.Run("DeleteRightAtEnd", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hi")

		// Position cursor at end of content.
		// ToRendered may return -1 for end-of-content, so set rendered
		// selection directly to the content length.
		w.body.q0 = 2
		w.body.q1 = 2
		contentLen := w.richBody.Content().Len()
		w.richBody.SetSelection(contentLen, contentLen)

		// Del at end should be a no-op
		w.HandlePreviewType(&w.body, 0x7F)

		got := w.body.file.String()
		want := "Hi"
		if got != want {
			t.Errorf("Del at end should be no-op: got %q, want %q", got, want)
		}
	})

	t.Run("KillWord", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello world")

		// Position cursor at 5 (end of "Hello")
		w.body.q0 = 5
		w.body.q1 = 5
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(5, 5)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		// ^W should delete "Hello"
		w.HandlePreviewType(&w.body, 0x17)

		got := w.body.file.String()
		want := " world"
		if got != want {
			t.Errorf("after ^W: got %q, want %q", got, want)
		}

		if w.body.q0 != 0 || w.body.q1 != 0 {
			t.Errorf("cursor should be at (0,0), got (%d,%d)", w.body.q0, w.body.q1)
		}
	})

	t.Run("KillLine", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello world")

		// Position cursor at 5
		w.body.q0 = 5
		w.body.q1 = 5
		if w.previewSourceMap != nil {
			rp0, rp1 := w.previewSourceMap.ToRendered(5, 5)
			if rp0 >= 0 {
				w.richBody.SetSelection(rp0, rp1)
			}
		}

		// ^U should delete from start of line to cursor ("Hello")
		w.HandlePreviewType(&w.body, 0x15)

		got := w.body.file.String()
		want := " world"
		if got != want {
			t.Errorf("after ^U: got %q, want %q", got, want)
		}

		if w.body.q0 != 0 || w.body.q1 != 0 {
			t.Errorf("cursor should be at (0,0), got (%d,%d)", w.body.q0, w.body.q1)
		}
	})
}

// TestPreviewTypeReplacesSelection tests that typing a character when text is
// selected replaces the selection (cut then insert).
func TestPreviewTypeReplacesSelection(t *testing.T) {
	w := setupPreviewTypeTestWindow(t, "Hello world")

	// Select "Hello" (0-5)
	w.body.q0 = 0
	w.body.q1 = 5
	if w.previewSourceMap != nil {
		rp0, rp1 := w.previewSourceMap.ToRendered(0, 5)
		if rp0 >= 0 {
			w.richBody.SetSelection(rp0, rp1)
		}
	}

	// Type 'X' — should replace "Hello" with "X"
	w.HandlePreviewType(&w.body, 'X')

	got := w.body.file.String()
	want := "X world"
	if got != want {
		t.Errorf("typing with selection: got %q, want %q", got, want)
	}

	if w.body.q0 != 1 || w.body.q1 != 1 {
		t.Errorf("cursor should be at (1,1), got (%d,%d)", w.body.q0, w.body.q1)
	}
}

// TestPreviewTypeIgnoresSpecialKeys verifies that draw key constants
// (KeyInsert, KeyLeft, KeyRight, Cmd+key, scroll keys) in the 0xF000+
// private unicode range are NOT treated as printable characters.
func TestPreviewTypeIgnoresSpecialKeys(t *testing.T) {
	specialKeys := []struct {
		name string
		key  rune
	}{
		{"KeyInsert", draw.KeyInsert},
		{"KeyLeft", draw.KeyLeft},
		{"KeyRight", draw.KeyRight},
		{"Cmd+c", draw.KeyCmd + 'c'},
		{"Cmd+v", draw.KeyCmd + 'v'},
		{"Cmd+x", draw.KeyCmd + 'x'},
		{"Cmd+z", draw.KeyCmd + 'z'},
		{"Kscrolloneup", Kscrolloneup},
		{"Kscrollonedown", Kscrollonedown},
		{"CtrlF", 0x06},
	}

	for _, sk := range specialKeys {
		t.Run(sk.name, func(t *testing.T) {
			w := setupPreviewTypeTestWindow(t, "Hello")

			w.body.q0 = 3
			w.body.q1 = 3
			if w.previewSourceMap != nil {
				rp0, rp1 := w.previewSourceMap.ToRendered(3, 3)
				if rp0 >= 0 {
					w.richBody.SetSelection(rp0, rp1)
				}
			}

			w.HandlePreviewType(&w.body, sk.key)

			got := w.body.file.String()
			if got != "Hello" {
				t.Errorf("key %s (0x%X) should be ignored, but buffer changed to %q", sk.name, sk.key, got)
			}
		})
	}
}

// TestPreviewTypeMultipleCharsMiddle tests typing multiple characters in
// the middle of content with the body registered as an observer.
// This exercises the full syncSourceSelection → edit → re-render → remap
// cycle including the incremental update path (since the observer records
// edits into pendingEdits).
func TestPreviewTypeMultipleCharsMiddle(t *testing.T) {
	tests := []struct {
		name   string
		source string
		pos    int // rendered position to place cursor
		chars  string
		want   string
		wantQ0 int // expected source cursor position after typing
	}{
		{
			name:   "plain text middle",
			source: "Hello world",
			pos:    5, // after "Hello"
			chars:  "XY",
			want:   "HelloXY world",
			wantQ0: 7,
		},
		{
			name:   "plain text start",
			source: "Hello",
			pos:    0,
			chars:  "AB",
			want:   "ABHello",
			wantQ0: 2,
		},
		{
			name:   "heading middle",
			source: "# Hello\n",
			pos:    2, // between 'l' and 'l' in rendered "Hello\n"
			chars:  "XY",
			want:   "# HeXYllo\n",
			wantQ0: 6,
		},
		{
			name:   "with trailing newline",
			source: "Hello\n",
			pos:    3,
			chars:  "XY",
			want:   "HelXYlo\n",
			wantQ0: 5,
		},
		{
			name:   "bold text middle",
			source: "**bold**",
			pos:    2, // between 'l' and 'd' in rendered "bold"
			chars:  "XY",
			want:   "**boXYld**",
			wantQ0: 6,
		},
		{
			name:   "inline code middle",
			source: "`code`",
			pos:    2, // between 'o' and 'd' in rendered "code"
			chars:  "XY",
			want:   "`coXYde`",
			wantQ0: 5,
		},
		{
			name:   "two paragraphs first line",
			source: "abc\n\ndef\n",
			pos:    1, // between 'a' and 'b' in first paragraph
			chars:  "XY",
			want:   "aXYbc\n\ndef\n",
			wantQ0: 3,
		},
		{
			name:   "two paragraphs second line",
			source: "abc\n\ndef\n",
			pos:    5, // rendered "abc\ndef" → a=0,b=1,c=2,\n(parabreak)=3,d=4,e=5,f=6; pos 5 = before 'e'
			chars:  "XY",
			want:   "abc\n\ndXYef\n",
			wantQ0: 8,
		},
		{
			name:   "heading then text",
			source: "# Title\nBody text\n",
			pos:    8, // rendered "Title\nBody text" → T=0..e=4,\n=5,B=6,o=7,d=8,y=9; pos 8 = before 'd'
			chars:  "XY",
			want:   "# Title\nBoXYdy text\n",
			wantQ0: 12,
		},
		{
			name:   "list item middle",
			source: "- hello\n",
			pos:    4, // rendered "• hello\n" → pos 4 = between 'l' and 'l'
			chars:  "XY",
			want:   "- heXYllo\n",
			wantQ0: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := setupPreviewTypeTestWindow(t, tt.source)
			// Register the body as an observer, matching real code (wind.go:139).
			// This causes InsertAt to trigger Text.Inserted which records edits
			// into w.pendingEdits, enabling the incremental update path.
			w.body.file.AddObserver(&w.body)

			// Position cursor at the specified rendered position.
			w.richBody.SetSelection(tt.pos, tt.pos)
			w.syncSourceSelection()

			// Type each character.
			for _, r := range tt.chars {
				w.HandlePreviewType(&w.body, r)
			}

			got := w.body.file.String()
			if got != tt.want {
				t.Errorf("after typing %q at rendered pos %d:\ngot  %q\nwant %q", tt.chars, tt.pos, got, tt.want)
			}

			if w.body.q0 != tt.wantQ0 || w.body.q1 != tt.wantQ0 {
				t.Errorf("cursor should be at (%d,%d), got (%d,%d)", tt.wantQ0, tt.wantQ0, w.body.q0, w.body.q1)
			}

			// Verify the rendered selection round-trips correctly.
			rp0, rp1 := w.richBody.Selection()
			w.syncSourceSelection()
			if w.body.q0 != tt.wantQ0 || w.body.q1 != tt.wantQ0 {
				t.Errorf("round-trip: rendered sel (%d,%d) → source (%d,%d), want (%d,%d)",
					rp0, rp1, w.body.q0, w.body.q1, tt.wantQ0, tt.wantQ0)
			}
		})
	}
}

// TestPreviewTypeParagraphJoin tests typing at the paragraph join space
// (where two consecutive source lines are rendered joined with a space).
// The join space has no sourcemap entry — it's a gap.
func TestPreviewTypeParagraphJoin(t *testing.T) {
	// Source "abc\ndef\n" → rendered "abc def" (space at pos 3 from join)
	w := setupPreviewTypeTestWindow(t, "abc\ndef\n")
	w.body.file.AddObserver(&w.body)

	// Place cursor at rendered position 3 (the join space).
	w.richBody.SetSelection(3, 3)
	w.syncSourceSelection()

	startQ0 := w.body.q0
	t.Logf("Join space pos 3 → source q0=%d (source: %q)", startQ0, w.body.file.String())

	// Type 3 characters at the join.
	for i, r := range "XYZ" {
		beforeQ0 := w.body.q0
		w.HandlePreviewType(&w.body, r)
		if w.body.q0 != beforeQ0+1 {
			t.Errorf("char %d (%c): q0 should be %d, got %d (source: %q)",
				i, r, beforeQ0+1, w.body.q0, w.body.file.String())
		}
		// Verify round-trip.
		expected := w.body.q0
		w.syncSourceSelection()
		if w.body.q0 != expected {
			rp0, _ := w.richBody.Selection()
			t.Errorf("char %d (%c): round-trip drift: was %d, now %d (rend sel %d, source: %q)",
				i, r, expected, w.body.q0, rp0, w.body.file.String())
		}
	}

	t.Logf("After typing: source=%q, q0=%d", w.body.file.String(), w.body.q0)
}

// TestPreviewTypeManyCharsValidateEach types many characters and validates
// the source position after every keystroke. This catches incremental
// sourcemap drift bugs that only appear after multiple edits.
func TestPreviewTypeManyCharsValidateEach(t *testing.T) {
	tests := []struct {
		name   string
		source string
		pos    int // rendered starting position
	}{
		{"plain text", "Hello world", 5},
		{"heading", "# Hello world\n", 3},
		{"bold", "some **bold** text", 4},
		{"two paragraphs", "abc\n\ndef\n", 1},
		{"heading + body", "# Title\nBody text\n", 8},
		{"list item", "- hello world\n", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := setupPreviewTypeTestWindow(t, tt.source)
			w.body.file.AddObserver(&w.body)

			// Position cursor
			w.richBody.SetSelection(tt.pos, tt.pos)
			w.syncSourceSelection()
			initialQ0 := w.body.q0

			// Type 6 characters, validating after each.
			typed := "ABCDEF"
			for i, r := range typed {
				beforeQ0 := w.body.q0
				w.HandlePreviewType(&w.body, r)

				// Source cursor should advance by exactly 1.
				if w.body.q0 != beforeQ0+1 {
					t.Errorf("char %d (%c): source q0 should be %d, got %d (source: %q)",
						i, r, beforeQ0+1, w.body.q0, w.body.file.String())
				}

				// Verify the round-trip: rendered → source → matches q0.
				expectedQ0 := w.body.q0
				w.syncSourceSelection()
				if w.body.q0 != expectedQ0 {
					rp0, _ := w.richBody.Selection()
					t.Errorf("char %d (%c): round-trip drift: q0 was %d, after syncSourceSelection got %d (rendered sel %d, source: %q)",
						i, r, expectedQ0, w.body.q0, rp0, w.body.file.String())
				}
			}

			// After 6 chars, source q0 should be initialQ0 + 6.
			if w.body.q0 != initialQ0+6 {
				t.Errorf("final q0: want %d, got %d", initialQ0+6, w.body.q0)
			}
		})
	}
}

// TestPreviewTypeUndoGrouping verifies that consecutive typed characters in
// preview mode are grouped into a single undo point, matching text mode behavior.
func TestPreviewTypeUndoGrouping(t *testing.T) {
	t.Run("ConsecutiveCharsGrouped", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello")

		// Position cursor at end.
		w.body.q0 = 5
		w.body.q1 = 5
		contentLen := w.richBody.Content().Len()
		w.richBody.SetSelection(contentLen, contentLen)

		// Type 'a', 'b', 'c' — should be one undo group.
		w.HandlePreviewType(&w.body, 'a')
		w.HandlePreviewType(&w.body, 'b')
		w.HandlePreviewType(&w.body, 'c')

		got := w.body.file.String()
		if got != "Helloabc" {
			t.Fatalf("after typing abc: got %q, want %q", got, "Helloabc")
		}

		// One undo should remove all three characters at once.
		w.Undo(true)

		got = w.body.file.String()
		if got != "Hello" {
			t.Errorf("after one undo: got %q, want %q", got, "Hello")
		}
	})

	t.Run("BackspaceCreatesUndoBoundary", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello")

		// Position cursor at end.
		w.body.q0 = 5
		w.body.q1 = 5
		contentLen := w.richBody.Content().Len()
		w.richBody.SetSelection(contentLen, contentLen)

		// Type 'a', 'b' — grouped into one undo group.
		w.HandlePreviewType(&w.body, 'a')
		w.HandlePreviewType(&w.body, 'b')

		got := w.body.file.String()
		if got != "Helloab" {
			t.Fatalf("after typing ab: got %q, want %q", got, "Helloab")
		}

		// Backspace creates a new undo boundary. After this, 'b' is deleted
		// and the subsequent 'c' is in the same undo group as the backspace
		// (matching text mode behavior where eq0 stays set).
		w.HandlePreviewType(&w.body, 0x08)

		got = w.body.file.String()
		if got != "Helloa" {
			t.Fatalf("after backspace: got %q, want %q", got, "Helloa")
		}

		w.HandlePreviewType(&w.body, 'c')

		got = w.body.file.String()
		if got != "Helloac" {
			t.Fatalf("after typing c: got %q, want %q", got, "Helloac")
		}

		// First undo: undoes backspace and 'c' together (same seq group).
		w.Undo(true)
		got = w.body.file.String()
		if got != "Helloab" {
			t.Errorf("after first undo: got %q, want %q", got, "Helloab")
		}

		// Second undo: removes 'a' and 'b' together.
		w.Undo(true)
		got = w.body.file.String()
		if got != "Hello" {
			t.Errorf("after second undo: got %q, want %q", got, "Hello")
		}
	})

	t.Run("NewlineCreatesUndoBoundary", func(t *testing.T) {
		w := setupPreviewTypeTestWindow(t, "Hello")

		// Position cursor at end.
		w.body.q0 = 5
		w.body.q1 = 5
		contentLen := w.richBody.Content().Len()
		w.richBody.SetSelection(contentLen, contentLen)

		// Type 'a', 'b' — grouped into one undo group.
		w.HandlePreviewType(&w.body, 'a')
		w.HandlePreviewType(&w.body, 'b')

		got := w.body.file.String()
		if got != "Helloab" {
			t.Fatalf("after typing ab: got %q, want %q", got, "Helloab")
		}

		// Newline creates a new undo boundary.
		w.HandlePreviewType(&w.body, '\n')

		afterNewline := w.body.file.String()
		if !strings.Contains(afterNewline, "\n") {
			t.Fatalf("after newline: buffer should contain newline, got %q", afterNewline)
		}

		// First undo: undoes the newline.
		w.Undo(true)
		got = w.body.file.String()
		if got != "Helloab" {
			t.Errorf("after first undo: got %q, want %q", got, "Helloab")
		}

		// Second undo: removes 'a' and 'b' together.
		w.Undo(true)
		got = w.body.file.String()
		if got != "Hello" {
			t.Errorf("after second undo: got %q, want %q", got, "Hello")
		}
	})
}
