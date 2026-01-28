package rich

import (
	"path/filepath"
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
)

// contentToBoxes converts Content (styled spans) into a sequence of Boxes.
// Each Box represents either a run of text, a newline, or a tab.
// Text is split on newlines and tabs, which become their own boxes.
func contentToBoxes(c Content) []Box {
	var boxes []Box

	for _, span := range c {
		if span.Text == "" {
			continue
		}
		boxes = appendSpanBoxes(boxes, span)
	}

	return boxes
}

// appendSpanBoxes appends boxes from a single span to the slice.
// It splits the span text on newlines, tabs, and spaces to enable word wrapping.
// Image spans are kept as a single box without splitting.
func appendSpanBoxes(boxes []Box, span Span) []Box {
	text := span.Text
	style := span.Style

	// Image spans should be kept as a single box without splitting
	if style.Image {
		boxes = append(boxes, Box{
			Text:  []byte(text),
			Nrune: utf8.RuneCountInString(text),
			Bc:    0,
			Style: style,
		})
		return boxes
	}

	for len(text) > 0 {
		// Find the next break character (newline, tab, or space)
		idx := -1
		var special rune
		for i, r := range text {
			if r == '\n' || r == '\t' || r == ' ' {
				idx = i
				special = r
				break
			}
		}

		if idx == -1 {
			// No more break characters, emit the rest as a text box
			boxes = append(boxes, Box{
				Text:  []byte(text),
				Nrune: utf8.RuneCountInString(text),
				Bc:    0,
				Style: style,
			})
			break
		}

		// Emit text before the break character (if any)
		if idx > 0 {
			prefix := text[:idx]
			boxes = append(boxes, Box{
				Text:  []byte(prefix),
				Nrune: utf8.RuneCountInString(prefix),
				Bc:    0,
				Style: style,
			})
		}

		// Emit the break character box
		if special == '\n' || special == '\t' {
			// Newline and tab are special marker boxes
			boxes = append(boxes, Box{
				Text:  nil,
				Nrune: -1,
				Bc:    special,
				Style: style,
			})
		} else {
			// Space is a regular text box (so it has measurable width)
			boxes = append(boxes, Box{
				Text:  []byte{' '},
				Nrune: 1,
				Bc:    0,
				Style: style,
			})
		}

		// Continue with the rest of the text (after the break character)
		text = text[idx+1:]
	}

	return boxes
}

// boxWidth calculates the width of a box in pixels using font metrics.
// For text boxes, it measures the text width using the font.
// For newline and tab boxes, it returns 0 (tabs are handled separately by tabBoxWidth).
// For image boxes with ImageData, returns the image width (not scaled).
func boxWidth(box *Box, font draw.Font) int {
	if box.IsNewline() || box.IsTab() {
		return 0
	}
	// For image boxes with ImageData, return the image width
	if box.IsImage() {
		return box.ImageData.Width
	}
	if len(box.Text) == 0 {
		return 0
	}
	return font.BytesWidth(box.Text)
}

// tabBoxWidth calculates the width of a tab box based on its position.
// Tab stops are aligned to multiples of maxtab pixels, relative to minX.
// xPos is the current X position, minX is the left edge of the frame.
func tabBoxWidth(box *Box, xPos, minX, maxtab int) int {
	if !box.IsTab() {
		return 0
	}
	// Calculate position relative to frame origin
	relPos := xPos - minX
	// Find distance to next tab stop
	return maxtab - (relPos % maxtab)
}

// Line represents a line of positioned boxes in the layout.
// This is the output of the layout algorithm.
type Line struct {
	Boxes  []PositionedBox // Boxes on this line
	Y      int             // Y position of the line (top)
	Height int             // Height of this line (max font height of boxes)
}

// PositionedBox is a Box with its computed screen position.
type PositionedBox struct {
	Box Box
	X   int // X position on screen
}

// FontHeightFunc returns the font height for a given style.
type FontHeightFunc func(style Style) int

// FontForStyleFunc returns the font for a given style.
type FontForStyleFunc func(style Style) draw.Font

// ListIndentWidth is the number of pixels per indent level for list items.
// This is approximately 2 characters wide.
const ListIndentWidth = 20

// CodeBlockIndentChars is the number of 'M' characters to indent code blocks.
const CodeBlockIndentChars = 4

// CodeBlockIndent is the default code block indentation in pixels.
// This assumes a typical M-width of 10 pixels (CodeBlockIndentChars * 10 = 40).
// The actual indentation may vary based on the code font at runtime.
const CodeBlockIndent = 40

// imageBoxDimensions calculates the width and height for an image box,
// scaling down if the image is wider than maxWidth.
// If box.Style.ImageWidth > 0, uses that as the target width (clamped to maxWidth),
// computing height proportionally from the original image dimensions.
// Returns (0, 0) if the box is not an image with ImageData.
func imageBoxDimensions(box *Box, maxWidth int) (width, height int) {
	if !box.IsImage() {
		return 0, 0
	}

	imgWidth := box.ImageData.Width
	imgHeight := box.ImageData.Height

	targetWidth := imgWidth
	if box.Style.ImageWidth > 0 {
		targetWidth = box.Style.ImageWidth
	}

	// Clamp to frame width
	if targetWidth > maxWidth {
		targetWidth = maxWidth
	}

	// Scale height proportionally
	if targetWidth == imgWidth {
		return imgWidth, imgHeight
	}
	scale := float64(targetWidth) / float64(imgWidth)
	return targetWidth, int(float64(imgHeight) * scale)
}

// layout positions boxes into lines, handling wrapping when boxes exceed frameWidth.
// It computes the Wid field for each box and assigns X/Y positions.
// The returned Lines contain positioned boxes ready for rendering.
// If fontHeightFn is nil, the default font height is used for all lines.
// If fontForStyleFn is nil, the default font is used for all width calculations.
func layout(boxes []Box, font draw.Font, frameWidth, maxtab int, fontHeightFn FontHeightFunc, fontForStyleFn FontForStyleFunc) []Line {
	if len(boxes) == 0 {
		return nil
	}

	defaultFontHeight := font.Height()

	// Helper to get font height for a style
	getFontHeight := func(style Style) int {
		if fontHeightFn != nil {
			return fontHeightFn(style)
		}
		return defaultFontHeight
	}

	// Helper to get font for a style (for width calculation)
	getFontForStyle := func(style Style) draw.Font {
		if fontForStyleFn != nil {
			return fontForStyleFn(style)
		}
		return font
	}

	// Calculate code block indent based on code font's M-width
	codeBlockIndent := CodeBlockIndentChars * font.BytesWidth([]byte("M"))
	if fontForStyleFn != nil {
		codeFont := fontForStyleFn(Style{Code: true, Scale: 1.0})
		codeBlockIndent = CodeBlockIndentChars * codeFont.BytesWidth([]byte("M"))
	}

	var lines []Line
	var currentLine Line
	currentLine.Y = 0
	currentLine.Height = defaultFontHeight
	xPos := 0
	pendingParaBreak := false // Track if we just had a paragraph break
	currentListIndent := 0    // Track current list indentation level for wrapped lines

	for i := range boxes {
		box := &boxes[i]

		// Update line height if this box uses a taller font
		boxHeight := getFontHeight(box.Style)

		// For image boxes with ImageData, use the image's scaled height
		if box.IsImage() {
			_, imgHeight := imageBoxDimensions(box, frameWidth)
			if imgHeight > boxHeight {
				boxHeight = imgHeight
			}
		}

		if boxHeight > currentLine.Height {
			currentLine.Height = boxHeight
		}

		// Handle newlines - they end the current line and start a new one
		if box.IsNewline() {
			box.Wid = 0
			currentLine.Boxes = append(currentLine.Boxes, PositionedBox{
				Box: *box,
				X:   xPos,
			})
			lines = append(lines, currentLine)

			// Calculate Y offset (just the line height for now)
			yOffset := currentLine.Height

			// Start new line
			currentLine = Line{
				Y:      currentLine.Y + yOffset,
				Height: defaultFontHeight,
			}
			xPos = 0
			currentListIndent = 0 // Reset list indent on explicit newline

			// Mark that we have a pending paragraph break if this newline is a para break
			if box.Style.ParaBreak {
				pendingParaBreak = true
			}
			continue
		}

		// If we have a pending paragraph break and this is the first content,
		// add space before this paragraph based on the content's font height
		if pendingParaBreak && !box.IsTab() {
			// Add half the height of the upcoming text before this paragraph
			currentLine.Y += boxHeight / 2
			pendingParaBreak = false
		}

		// Calculate indentation for this box
		indentPixels := 0
		if box.Style.ListBullet || box.Style.ListItem {
			indentPixels = box.Style.ListIndent * ListIndentWidth
			currentListIndent = box.Style.ListIndent // Track for wrapped lines
		} else if box.Style.Block && box.Style.Code {
			// Code blocks get indentation based on code font M-width
			indentPixels = codeBlockIndent
		}

		// Apply indentation at the start of a line
		if xPos == 0 && indentPixels > 0 {
			xPos = indentPixels
		}

		// Calculate width for this box using the style-specific font
		var width int
		if box.IsTab() {
			width = tabBoxWidth(box, xPos, 0, maxtab)
		} else if box.IsImage() {
			// For image boxes, use scaled dimensions
			width, _ = imageBoxDimensions(box, frameWidth)
		} else {
			width = boxWidth(box, getFontForStyle(box.Style))
		}

		// Effective frame width accounts for indentation
		effectiveFrameWidth := frameWidth - indentPixels

		// Determine the current indent for wrapped lines
		currentIndent := currentListIndent * ListIndentWidth
		if box.Style.Block && box.Style.Code {
			currentIndent = codeBlockIndent
		}

		// Check if we need to wrap
		if xPos+width > frameWidth && xPos > indentPixels {
			// Need to wrap - but only if we're not at the start of the content area
			// First, check if this box can fit on a new line
			// If the box is wider than effectiveFrameWidth, we'll need to split it

			if width <= effectiveFrameWidth {
				// Box fits on new line, start new line
				lines = append(lines, currentLine)
				currentLine = Line{
					Y:      currentLine.Y + currentLine.Height,
					Height: defaultFontHeight,
				}
				// Maintain indentation on wrapped lines
				xPos = currentIndent

				// Recalculate tab width at new position
				if box.IsTab() {
					width = tabBoxWidth(box, xPos, 0, maxtab)
				}
			} else {
				// Box is wider than frame, need to split it
				lines, currentLine, xPos = splitBoxAcrossLinesWithIndent(lines, currentLine, box, font, frameWidth, currentLine.Height, getFontHeight, getFontForStyle, currentIndent)
				continue
			}
		} else if xPos == indentPixels && width > effectiveFrameWidth {
			// Box is at start of content area but still too wide - split it
			lines, currentLine, xPos = splitBoxAcrossLinesWithIndent(lines, currentLine, box, font, frameWidth, currentLine.Height, getFontHeight, getFontForStyle, indentPixels)
			continue
		}

		// Add box to current line
		box.Wid = width
		currentLine.Boxes = append(currentLine.Boxes, PositionedBox{
			Box: *box,
			X:   xPos,
		})
		xPos += width
	}

	// Don't forget the last line (if it has content)
	if len(currentLine.Boxes) > 0 {
		lines = append(lines, currentLine)
	}

	// A trailing newline creates an empty final line
	// Check if the last box was a newline - if so, add the empty line
	if len(boxes) > 0 && boxes[len(boxes)-1].IsNewline() && len(currentLine.Boxes) == 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// splitBoxAcrossLines splits a text box that's too wide to fit on a single line.
// It creates multiple boxes, each fitting within frameWidth.
func splitBoxAcrossLines(lines []Line, currentLine Line, box *Box, defaultFont draw.Font, frameWidth, defaultFontHeight int, fontHeightFn func(Style) int, fontForStyleFn func(Style) draw.Font) ([]Line, Line, int) {
	return splitBoxAcrossLinesWithIndent(lines, currentLine, box, defaultFont, frameWidth, defaultFontHeight, fontHeightFn, fontForStyleFn, 0)
}

// splitBoxAcrossLinesWithIndent splits a text box that's too wide to fit on a single line,
// maintaining the specified indentation on wrapped lines.
func splitBoxAcrossLinesWithIndent(lines []Line, currentLine Line, box *Box, defaultFont draw.Font, frameWidth, defaultFontHeight int, fontHeightFn func(Style) int, fontForStyleFn func(Style) draw.Font, indent int) ([]Line, Line, int) {
	// Tabs and newlines should never need splitting
	if box.IsTab() || box.IsNewline() {
		box.Wid = 0
		currentLine.Boxes = append(currentLine.Boxes, PositionedBox{
			Box: *box,
			X:   indent,
		})
		return lines, currentLine, indent
	}

	text := box.Text
	style := box.Style
	xPos := indent

	// Get the correct font for this box's style
	font := defaultFont
	if fontForStyleFn != nil {
		font = fontForStyleFn(style)
	}

	// Get font height for this box's style
	boxHeight := defaultFontHeight
	if fontHeightFn != nil {
		boxHeight = fontHeightFn(style)
	}
	if boxHeight > currentLine.Height {
		currentLine.Height = boxHeight
	}

	// Effective width available for content (after indentation)
	effectiveWidth := frameWidth - indent

	for len(text) > 0 {
		// Find how many bytes fit on this line
		bytesOnLine, widthOnLine := fitBytes(text, font, effectiveWidth)

		if bytesOnLine == 0 {
			// At least one rune must fit (even if it exceeds effectiveWidth)
			_, runeLen := utf8.DecodeRune(text)
			bytesOnLine = runeLen
			widthOnLine = font.BytesWidth(text[:runeLen])
		}

		// Create box for this portion
		portionText := text[:bytesOnLine]
		portionBox := Box{
			Text:  portionText,
			Nrune: utf8.RuneCount(portionText),
			Bc:    0,
			Style: style,
			Wid:   widthOnLine,
		}
		currentLine.Boxes = append(currentLine.Boxes, PositionedBox{
			Box: portionBox,
			X:   xPos,
		})
		xPos = indent + widthOnLine

		text = text[bytesOnLine:]

		if len(text) > 0 {
			// More text remaining, start a new line
			lines = append(lines, currentLine)
			currentLine = Line{
				Y:      currentLine.Y + currentLine.Height,
				Height: boxHeight, // New line continues with same style
			}
			xPos = indent // Maintain indentation on wrapped lines
		}
	}

	return lines, currentLine, xPos
}

// fitBytes returns how many bytes of text fit within maxWidth pixels,
// along with the actual width of those bytes.
func fitBytes(text []byte, font draw.Font, maxWidth int) (bytesCount int, width int) {
	totalWidth := 0
	for i := 0; i < len(text); {
		_, runeLen := utf8.DecodeRune(text[i:])
		runeWidth := font.BytesWidth(text[i : i+runeLen])

		if totalWidth+runeWidth > maxWidth && bytesCount > 0 {
			// This rune would exceed maxWidth, stop here
			break
		}

		bytesCount += runeLen
		totalWidth += runeWidth
		i += runeLen
	}
	return bytesCount, totalWidth
}

// layoutWithCache is like layout but accepts an ImageCache for loading images.
// This allows image boxes to be sized based on their actual image dimensions.
// If cache is nil, images will use placeholder sizing based on text width.
func layoutWithCache(boxes []Box, font draw.Font, frameWidth, maxtab int, fontHeightFn FontHeightFunc, fontForStyleFn FontForStyleFunc, cache *ImageCache) []Line {
	// If no cache, fall back to regular layout
	if cache == nil {
		return layout(boxes, font, frameWidth, maxtab, fontHeightFn, fontForStyleFn)
	}

	// Load images into cache and populate ImageData for image boxes
	for i := range boxes {
		box := &boxes[i]
		if box.Style.Image && box.Style.ImageURL != "" {
			// Load image from cache (this will cache it for future use)
			cached, _ := cache.Load(box.Style.ImageURL)
			if cached != nil {
				box.ImageData = cached
			}
		}
	}

	// Now run layout with populated ImageData
	return layout(boxes, font, frameWidth, maxtab, fontHeightFn, fontForStyleFn)
}

// layoutWithCacheAndBasePath is like layoutWithCache but also accepts a basePath
// for resolving relative image paths. The basePath should be the path to the
// markdown file containing the image references.
//
// For example, if basePath is "/home/user/docs/readme.md" and an image has
// ImageURL "images/photo.png", the path will be resolved to
// "/home/user/docs/images/photo.png" before loading from the cache.
//
// Absolute paths (starting with /) are used as-is.
// If basePath is empty, relative paths are used as-is (likely failing to load).
func layoutWithCacheAndBasePath(boxes []Box, font draw.Font, frameWidth, maxtab int, fontHeightFn FontHeightFunc, fontForStyleFn FontForStyleFunc, cache *ImageCache, basePath string) []Line {
	// If no cache, fall back to regular layout
	if cache == nil {
		return layout(boxes, font, frameWidth, maxtab, fontHeightFn, fontForStyleFn)
	}

	// Load images into cache and populate ImageData for image boxes
	for i := range boxes {
		box := &boxes[i]
		if box.Style.Image && box.Style.ImageURL != "" {
			// Resolve relative paths using basePath
			imgPath := resolveImagePath(basePath, box.Style.ImageURL)

			// Load image from cache (this will cache it for future use)
			cached, _ := cache.Load(imgPath)
			if cached != nil {
				box.ImageData = cached
			}
		}
	}

	// Now run layout with populated ImageData
	return layout(boxes, font, frameWidth, maxtab, fontHeightFn, fontForStyleFn)
}

// resolveImagePath resolves an image path relative to the base path's directory.
// If imgPath is absolute (starts with /), it is returned unchanged.
// If basePath is empty, imgPath is returned unchanged.
// Otherwise, imgPath is resolved relative to the directory containing basePath.
func resolveImagePath(basePath, imgPath string) string {
	// URLs are returned as-is (not resolved against basePath)
	if isImageURL(imgPath) {
		return imgPath
	}

	// Absolute paths are returned as-is
	if filepath.IsAbs(imgPath) {
		return imgPath
	}

	// If no basePath, can't resolve relative path
	if basePath == "" {
		return imgPath
	}

	// Get the directory containing the base file (e.g., markdown file)
	baseDir := filepath.Dir(basePath)

	// Join and clean the path
	resolved := filepath.Join(baseDir, imgPath)
	return filepath.Clean(resolved)
}
