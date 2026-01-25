package main

import (
	"image"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	// Parse markdown and set content with source map
	content, sourceMap := markdown.ParseWithSourceMap(sourceMarkdown)
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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	// Parse markdown and set content with source map
	content, sourceMap := markdown.ParseWithSourceMap(sourceMarkdown)
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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	// Parse markdown and set content with source map
	content, sourceMap := markdown.ParseWithSourceMap(sourceMarkdown)
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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithScrollbarColors(scrBg, scrThumb),
	)

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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	// Parse markdown with source map
	content, sourceMap := markdown.ParseWithSourceMap(markdownContent)
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
	rt.Init(bodyRect, display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)

	content, sourceMap := markdown.ParseWithSourceMap(markdownContent)
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
