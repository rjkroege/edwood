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

	// Size multiplier (1.0 = normal body text)
	// Used for headings: H1=2.0, H2=1.5, H3=1.25, etc.
	Scale float64
}

// DefaultStyle returns the default body text style.
func DefaultStyle() Style {
	return Style{Scale: 1.0}
}

// Common styles
var (
	StyleH1     = Style{Bold: true, Scale: 2.0}
	StyleH2     = Style{Bold: true, Scale: 1.5}
	StyleH3     = Style{Bold: true, Scale: 1.25}
	StyleBold   = Style{Bold: true, Scale: 1.0}
	StyleItalic = Style{Italic: true, Scale: 1.0}
	StyleCode   = Style{Scale: 1.0} // Will use monospace font
)
