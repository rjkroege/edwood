package main

import (
	"image"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

// PreviewWindow is a window that displays rendered markdown content
// using a RichText component. It provides a read-only preview of
// markdown files with styled text rendering.
type PreviewWindow struct {
	rect     image.Rectangle
	display  draw.Display
	richText *RichText
	content  rich.Content
	source   string // Path or identifier of the source being previewed
}

// NewPreviewWindow creates a new PreviewWindow.
func NewPreviewWindow() *PreviewWindow {
	return &PreviewWindow{}
}

// Init initializes the PreviewWindow with the given rectangle, display, font, and options.
func (pw *PreviewWindow) Init(r image.Rectangle, display draw.Display, font draw.Font, opts ...PreviewOption) {
	pw.rect = r
	pw.display = display

	// Create the underlying RichText component
	pw.richText = NewRichText()

	// Build RichText options from preview options
	var rtOpts []RichTextOption
	for _, opt := range opts {
		if rtOpt := opt.toRichTextOption(); rtOpt != nil {
			rtOpts = append(rtOpts, rtOpt)
		}
	}

	pw.richText.Init(r, display, font, rtOpts...)
}

// RichText returns the underlying RichText component.
func (pw *PreviewWindow) RichText() *RichText {
	return pw.richText
}

// Display returns the display.
func (pw *PreviewWindow) Display() draw.Display {
	return pw.display
}

// Rect returns the preview window's rectangle.
func (pw *PreviewWindow) Rect() image.Rectangle {
	return pw.rect
}

// SetMarkdown sets the content from a markdown string.
// The markdown is parsed and converted to rich.Content for display.
func (pw *PreviewWindow) SetMarkdown(md string) {
	pw.content = markdown.Parse(md)
	if pw.richText != nil {
		pw.richText.SetContent(pw.content)
	}
}

// Content returns the current rich.Content being displayed.
func (pw *PreviewWindow) Content() rich.Content {
	return pw.content
}

// SetSource sets the source identifier (e.g., file path) being previewed.
func (pw *PreviewWindow) SetSource(source string) {
	pw.source = source
}

// Source returns the source identifier being previewed.
func (pw *PreviewWindow) Source() string {
	return pw.source
}

// Redraw redraws the preview window.
func (pw *PreviewWindow) Redraw() {
	if pw.richText != nil {
		pw.richText.Redraw()
	}
}

// SyncToSourcePosition synchronizes the preview scroll position based on a
// position in the source document. This enables the preview to follow along
// as the user scrolls in the source editor.
// sourcePos is the current position in the source document (e.g., cursor or
// top of visible area), and totalSourceRunes is the total length of the source.
func (pw *PreviewWindow) SyncToSourcePosition(sourcePos, totalSourceRunes int) {
	if pw.richText == nil || pw.content == nil {
		return
	}

	totalPreviewRunes := pw.content.Len()
	if totalPreviewRunes == 0 || totalSourceRunes == 0 {
		return
	}

	// Check if content fits on screen - no scrolling needed
	frame := pw.richText.Frame()
	if frame == nil {
		return
	}

	// Count lines in preview content
	lineCount := 1
	lineStarts := []int{0}
	for i, span := range pw.content {
		runeOffset := 0
		if i > 0 {
			for j := 0; j < i; j++ {
				runeOffset += len([]rune(pw.content[j].Text))
			}
		}
		for j, r := range span.Text {
			if r == '\n' {
				lineCount++
				lineStarts = append(lineStarts, runeOffset+j+1)
			}
		}
	}

	maxLines := frame.MaxLines()

	// If all content fits, no scrolling needed
	if lineCount <= maxLines {
		pw.richText.SetOrigin(0)
		return
	}

	// Calculate the proportion through the source document
	proportion := float64(sourcePos) / float64(totalSourceRunes)
	if proportion < 0 {
		proportion = 0
	}
	if proportion > 1 {
		proportion = 1
	}

	// Map proportion to a line in the preview content
	targetLine := int(float64(lineCount-1) * proportion)
	if targetLine < 0 {
		targetLine = 0
	}
	if targetLine >= len(lineStarts) {
		targetLine = len(lineStarts) - 1
	}

	// Set the origin to the start of the target line
	newOrigin := lineStarts[targetLine]
	pw.richText.SetOrigin(newOrigin)
}

// PreviewOption is a functional option for configuring PreviewWindow.
type PreviewOption struct {
	bg        draw.Image
	textColor draw.Image
	scrBg     draw.Image
	scrThumb  draw.Image
	optType   previewOptType
}

type previewOptType int

const (
	optBackground previewOptType = iota
	optTextColor
	optScrollbarColors
)

// toRichTextOption converts this preview option to a RichText option.
func (po PreviewOption) toRichTextOption() RichTextOption {
	switch po.optType {
	case optBackground:
		return WithRichTextBackground(po.bg)
	case optTextColor:
		return WithRichTextColor(po.textColor)
	case optScrollbarColors:
		return WithScrollbarColors(po.scrBg, po.scrThumb)
	}
	return nil
}

// WithPreviewBackground sets the background image for the preview window.
func WithPreviewBackground(bg draw.Image) PreviewOption {
	return PreviewOption{bg: bg, optType: optBackground}
}

// WithPreviewTextColor sets the text color image for the preview window.
func WithPreviewTextColor(c draw.Image) PreviewOption {
	return PreviewOption{textColor: c, optType: optTextColor}
}

// WithPreviewScrollbarColors sets the scrollbar background and thumb colors.
func WithPreviewScrollbarColors(bg, thumb draw.Image) PreviewOption {
	return PreviewOption{scrBg: bg, scrThumb: thumb, optType: optScrollbarColors}
}

// PreviewState holds the state for an active preview window.
// It tracks the preview window, its source, and provides mouse handling.
// PreviewState implements file.BufferObserver to receive live updates
// when the source file is edited.
type PreviewState struct {
	Window *PreviewWindow
	Source string // Source file path being previewed
	buffer *file.ObservableEditableBuffer // Source buffer for live updates
}

// Compile-time check that PreviewState implements file.BufferObserver
var _ file.BufferObserver = (*PreviewState)(nil)

// NewPreviewState creates a preview state for a given source file.
func NewPreviewState(source string, r image.Rectangle, display draw.Display, font draw.Font) *PreviewState {
	pw := NewPreviewWindow()

	// Allocate colors for the preview window
	bgColor := draw.Color(0xFFFFF0FF) // Light ivory background
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		bgImage = nil
	}

	textColor := draw.Black
	textImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, textColor)
	if err != nil {
		textImage = nil
	}

	scrBgColor := draw.Color(0xEEEEEEFF) // Light gray scrollbar background
	scrBgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, scrBgColor)
	if err != nil {
		scrBgImage = nil
	}

	scrThumbColor := draw.Color(0x999999FF) // Darker gray scrollbar thumb
	scrThumbImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, scrThumbColor)
	if err != nil {
		scrThumbImage = nil
	}

	var opts []PreviewOption
	if bgImage != nil {
		opts = append(opts, WithPreviewBackground(bgImage))
	}
	if textImage != nil {
		opts = append(opts, WithPreviewTextColor(textImage))
	}
	if scrBgImage != nil && scrThumbImage != nil {
		opts = append(opts, WithPreviewScrollbarColors(scrBgImage, scrThumbImage))
	}

	pw.Init(r, display, font, opts...)
	pw.SetSource(source)

	return &PreviewState{
		Window: pw,
		Source: source,
	}
}

// Rect returns the preview window's rectangle.
func (ps *PreviewState) Rect() image.Rectangle {
	if ps.Window == nil {
		return image.Rectangle{}
	}
	return ps.Window.Rect()
}

// HandleMouse handles mouse events for the preview window.
// Returns true if the event was handled.
func (ps *PreviewState) HandleMouse(m *draw.Mouse) bool {
	if ps == nil || ps.Window == nil || ps.Window.richText == nil {
		return false
	}

	r := ps.Window.Rect()
	if !m.Point.In(r) {
		return false
	}

	rt := ps.Window.richText

	// Handle scroll wheel (buttons 4 and 5)
	if m.Buttons&8 != 0 { // Button 4 - scroll up
		rt.ScrollWheel(true)
		ps.Window.Redraw()
		ps.Window.display.Flush()
		return true
	}
	if m.Buttons&16 != 0 { // Button 5 - scroll down
		rt.ScrollWheel(false)
		ps.Window.Redraw()
		ps.Window.display.Flush()
		return true
	}

	// Handle scrollbar clicks (buttons 1, 2, 3 in scrollbar area)
	scrRect := rt.ScrollRect()
	if m.Point.In(scrRect) {
		if m.Buttons&1 != 0 { // Button 1
			rt.ScrollClick(1, m.Point)
			ps.Window.Redraw()
			ps.Window.display.Flush()
			return true
		}
		if m.Buttons&2 != 0 { // Button 2
			rt.ScrollClick(2, m.Point)
			ps.Window.Redraw()
			ps.Window.display.Flush()
			return true
		}
		if m.Buttons&4 != 0 { // Button 3
			rt.ScrollClick(3, m.Point)
			ps.Window.Redraw()
			ps.Window.display.Flush()
			return true
		}
	}

	return false
}

// SetBuffer sets the source buffer for live updates.
// The PreviewState will register itself as an observer to receive
// notifications when the buffer changes.
func (ps *PreviewState) SetBuffer(buf *file.ObservableEditableBuffer) {
	if ps == nil {
		return
	}
	// Unregister from previous buffer if any
	if ps.buffer != nil {
		ps.buffer.DelObserver(ps)
	}
	ps.buffer = buf
	if buf != nil {
		buf.AddObserver(ps)
	}
}

// Inserted implements file.BufferObserver.
// Called when text is inserted into the source buffer.
func (ps *PreviewState) Inserted(q0 file.OffsetTuple, b []byte, nr int) {
	ps.updateFromBuffer()
}

// Deleted implements file.BufferObserver.
// Called when text is deleted from the source buffer.
func (ps *PreviewState) Deleted(q0, q1 file.OffsetTuple) {
	ps.updateFromBuffer()
}

// updateFromBuffer reads the current content from the buffer and updates the preview.
func (ps *PreviewState) updateFromBuffer() {
	if ps == nil || ps.Window == nil || ps.buffer == nil {
		return
	}
	content := ps.buffer.String()
	ps.Window.SetMarkdown(content)
	ps.Window.Redraw()
	if ps.Window.display != nil {
		ps.Window.display.Flush()
	}
}
