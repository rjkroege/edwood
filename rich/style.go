package rich

import "image/color"

// Style defines visual attributes for a span of text.
type Style struct {
	// Colors (nil means use default)
	Fg color.Color
	Bg color.Color

	// Font variations
	Bold   bool
	Italic bool
	Code   bool // Monospace font for code spans
	Link   bool // Hyperlink (rendered in blue by default)
	Block  bool // Block-level element (full-width background for fenced code blocks)

	// Size multiplier (1.0 = normal body text)
	// Used for headings: H1=2.0, H2=1.5, H3=1.25, etc.
	Scale float64
}

// DefaultStyle returns the default body text style.
func DefaultStyle() Style {
	return Style{Scale: 1.0}
}

// LinkBlue is the standard blue color for hyperlinks.
var LinkBlue = color.RGBA{R: 0, G: 0, B: 238, A: 255}

// InlineCodeBg is the light gray background for inline code spans.
// Uses RGB values around 230 for a subtle but visible distinction.
var InlineCodeBg = color.RGBA{R: 230, G: 230, B: 230, A: 255}

// Common styles
var (
	StyleH1     = Style{Bold: true, Scale: 2.0}
	StyleH2     = Style{Bold: true, Scale: 1.5}
	StyleH3     = Style{Bold: true, Scale: 1.25}
	StyleBold   = Style{Bold: true, Scale: 1.0}
	StyleItalic = Style{Italic: true, Scale: 1.0}
	StyleCode   = Style{Code: true, Scale: 1.0}            // Monospace font
	StyleLink   = Style{Link: true, Fg: LinkBlue, Scale: 1.0} // Blue hyperlink
)
