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
	HRule  bool // Horizontal rule marker (draw line instead of text)

	// Layout hints
	ParaBreak bool // Paragraph break - adds extra vertical spacing

	// List formatting
	ListItem    bool // This span is a list item
	ListBullet  bool // This span is a list bullet/number marker
	ListIndent  int  // Indentation level (0 = top level)
	ListOrdered bool // true for ordered lists, false for unordered
	ListNumber  int  // For ordered lists, the item number

	// Table formatting
	Table       bool      // This span is part of a table
	TableHeader bool      // This is a header cell
	TableAlign  Alignment // Cell alignment (left, center, right)

	// Blockquote formatting
	Blockquote      bool // This span is inside a blockquote
	BlockquoteDepth int  // Nesting level (1 = `>`, 2 = `> >`, …)

	// Image placeholder
	Image      bool   // This span is an image placeholder
	ImageURL   string // URL/path of the image
	ImageAlt   string // Alt text
	ImageWidth int    // Explicit width in pixels (0 = use natural size)

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

// Alignment represents text alignment within a table cell.
type Alignment int

const (
	AlignLeft   Alignment = iota // Default left alignment
	AlignCenter                  // Center alignment
	AlignRight                   // Right alignment
)

// HRuleRune is the marker rune used to represent a horizontal rule.
// When the renderer encounters this rune, it draws a horizontal line instead of text.
const HRuleRune = '\u2500' // ─ (BOX DRAWINGS LIGHT HORIZONTAL)

// InlineCodeBg is the light gray background for inline code spans and
// fenced code blocks. Light enough that the Darkyellow (0xEEEE9E)
// selection highlight is clearly visible against it.
var InlineCodeBg = color.RGBA{R: 245, G: 245, B: 245, A: 255}

// Common styles
var (
	StyleH1     = Style{Bold: true, Scale: 2.0}
	StyleH2     = Style{Bold: true, Scale: 1.5}
	StyleH3     = Style{Bold: true, Scale: 1.25}
	StyleBold   = Style{Bold: true, Scale: 1.0}
	StyleItalic = Style{Italic: true, Scale: 1.0}
	StyleCode   = Style{Code: true, Scale: 1.0}               // Monospace font
	StyleLink   = Style{Link: true, Fg: LinkBlue, Scale: 1.0} // Blue hyperlink
	StyleHRule  = Style{HRule: true, Scale: 1.0}              // Horizontal rule
)
