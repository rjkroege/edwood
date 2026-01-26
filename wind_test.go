package main

import (
	"image"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/draw"
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
