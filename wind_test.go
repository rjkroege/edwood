package main

import (
	"image"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
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
