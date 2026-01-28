// Package ui provides user interface utilities for edwood.
package ui

// LayoutMetrics tracks tag and body font heights separately to support
// proper height calculations when tag and body use different fonts.
// This addresses the issue where taglines calculations incorrectly assumed
// tags take the same number of pixels height as body lines.
type LayoutMetrics struct {
	tagFontHeight  int
	bodyFontHeight int
}

// NewLayoutMetrics creates a new LayoutMetrics with the given font heights.
func NewLayoutMetrics(tagFontHeight, bodyFontHeight int) *LayoutMetrics {
	return &LayoutMetrics{
		tagFontHeight:  tagFontHeight,
		bodyFontHeight: bodyFontHeight,
	}
}

// TagFontHeight returns the height of the tag font.
func (lm *LayoutMetrics) TagFontHeight() int {
	return lm.tagFontHeight
}

// BodyFontHeight returns the height of the body font.
func (lm *LayoutMetrics) BodyFontHeight() int {
	return lm.bodyFontHeight
}

// SetTagFontHeight updates the tag font height.
func (lm *LayoutMetrics) SetTagFontHeight(h int) {
	lm.tagFontHeight = h
}

// SetBodyFontHeight updates the body font height.
func (lm *LayoutMetrics) SetBodyFontHeight(h int) {
	lm.bodyFontHeight = h
}

// TagLinesHeight returns the pixel height for the given number of tag lines.
func (lm *LayoutMetrics) TagLinesHeight(tagLines int) int {
	return tagLines * lm.tagFontHeight
}

// BodyLinesHeight returns the pixel height for the given number of body lines.
func (lm *LayoutMetrics) BodyLinesHeight(bodyLines int) int {
	return bodyLines * lm.bodyFontHeight
}

// WindowHeight returns the total pixel height for a window with the given
// number of tag and body lines, plus the border and separator.
func (lm *LayoutMetrics) WindowHeight(tagLines, bodyLines, border int) int {
	return lm.TagLinesHeight(tagLines) + border + lm.BodyLinesHeight(bodyLines) + 1
}

// BodyLinesForHeight returns the number of complete body lines that fit
// in the given pixel height.
func (lm *LayoutMetrics) BodyLinesForHeight(height int) int {
	if lm.bodyFontHeight == 0 {
		return 0
	}
	return height / lm.bodyFontHeight
}

// TagLinesForHeight returns the number of complete tag lines that fit
// in the given pixel height.
func (lm *LayoutMetrics) TagLinesForHeight(height int) int {
	if lm.tagFontHeight == 0 {
		return 0
	}
	return height / lm.tagFontHeight
}

// MinWindowHeight returns the minimum pixel height for a window,
// which includes 1 tag line, the border, 1 body line, and the separator.
func (lm *LayoutMetrics) MinWindowHeight(border int) int {
	return lm.tagFontHeight + border + lm.bodyFontHeight + 1
}

// TotalLinesEquivalent converts a tag+body line count to an equivalent
// number of body lines, accounting for different font heights.
// This is useful when distributing space in a column where windows
// may have different tag and body font heights.
func (lm *LayoutMetrics) TotalLinesEquivalent(tagLines, bodyLines int) int {
	if lm.bodyFontHeight == 0 {
		return 0
	}
	// Convert tag lines to body-line equivalents
	tagEquiv := (tagLines * lm.tagFontHeight) / lm.bodyFontHeight
	return tagEquiv + bodyLines
}

// Equal returns true if both LayoutMetrics have the same values.
func (lm *LayoutMetrics) Equal(other *LayoutMetrics) bool {
	if other == nil {
		return false
	}
	return lm.tagFontHeight == other.tagFontHeight &&
		lm.bodyFontHeight == other.bodyFontHeight
}

// PixelHeightFromLines returns the total pixel height for the given
// number of tag and body lines, accounting for different font heights.
// This is the raw content height without borders or separators.
func (lm *LayoutMetrics) PixelHeightFromLines(tagLines, bodyLines int) int {
	return tagLines*lm.tagFontHeight + bodyLines*lm.bodyFontHeight
}

// BodyLinesFromPixelHeight returns the number of complete body lines
// that fit in the given pixel height after accounting for tag lines.
func (lm *LayoutMetrics) BodyLinesFromPixelHeight(tagLines, pixelHeight int) int {
	if lm.bodyFontHeight == 0 {
		return 0
	}
	tagPixels := tagLines * lm.tagFontHeight
	remaining := pixelHeight - tagPixels
	if remaining < 0 {
		return 0
	}
	return remaining / lm.bodyFontHeight
}

// TotalPixelHeight returns the total pixel height for a window including
// tag lines, body lines, border, and separator.
func (lm *LayoutMetrics) TotalPixelHeight(tagLines, bodyLines, border, separator int) int {
	return lm.PixelHeightFromLines(tagLines, bodyLines) + border + separator
}

// BodyLinesFromTotalPixels returns the number of complete body lines
// that fit in the given total pixel height after accounting for
// tag lines, border, and separator.
func (lm *LayoutMetrics) BodyLinesFromTotalPixels(tagLines, totalPixels, border, separator int) int {
	if lm.bodyFontHeight == 0 {
		return 0
	}
	contentPixels := totalPixels - border - separator
	return lm.BodyLinesFromPixelHeight(tagLines, contentPixels)
}
