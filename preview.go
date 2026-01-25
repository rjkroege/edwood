package main

import (
	"image"

	"github.com/rjkroege/edwood/draw"
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
