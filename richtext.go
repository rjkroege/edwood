package main

import (
	"image"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/rich"
)

// Note: scrollbar dimensions use Scrollwid and Scrollgap from dat.go
// with display.ScaleSize() for proper high-DPI support

// RichText is a component that combines a rich.Frame with a scrollbar.
// It manages the layout of the scrollbar area and the text frame area.
type RichText struct {
	// Cached rectangles from last Render() call, used for hit-testing.
	// The canonical rectangle is body.all - these are derived at render time.
	lastRect       image.Rectangle // Full area including scrollbar (cached)
	lastScrollRect image.Rectangle // Scrollbar area (cached)

	display draw.Display
	frame      rich.Frame
	content    rich.Content

	// Options stored for frame initialization
	background     draw.Image
	textColor      draw.Image
	selectionColor draw.Image

	// Font variants for styled text
	boldFont       draw.Font
	italicFont     draw.Font
	boldItalicFont draw.Font
	codeFont       draw.Font

	// Scaled fonts for headings (key is scale factor)
	scaledFonts map[float64]draw.Font

	// Scrollbar colors
	scrollBg    draw.Image // Scrollbar background color
	scrollThumb draw.Image // Scrollbar thumb color

	// Image cache for loading images in markdown
	imageCache *rich.ImageCache

	// Base path for resolving relative image paths (e.g., the markdown file path)
	basePath string
}

// NewRichText creates a new RichText component.
func NewRichText() *RichText {
	return &RichText{}
}

// Init initializes the RichText component with the given display, font, and options.
// The rectangle is not provided at init time - use Render(rect) to draw into a specific area.
// This allows the rectangle to be provided dynamically (e.g., from body.all).
func (rt *RichText) Init(display draw.Display, font draw.Font, opts ...RichTextOption) {
	rt.display = display

	// Apply options
	for _, opt := range opts {
		opt(rt)
	}

	// Create the frame (but don't init with a rectangle yet)
	rt.frame = rich.NewFrame()

	// Build frame options
	frameOpts := []rich.Option{
		rich.WithDisplay(display),
		rich.WithFont(font),
	}
	if rt.background != nil {
		frameOpts = append(frameOpts, rich.WithBackground(rt.background))
	}
	if rt.textColor != nil {
		frameOpts = append(frameOpts, rich.WithTextColor(rt.textColor))
	}
	if rt.boldFont != nil {
		frameOpts = append(frameOpts, rich.WithBoldFont(rt.boldFont))
	}
	if rt.italicFont != nil {
		frameOpts = append(frameOpts, rich.WithItalicFont(rt.italicFont))
	}
	if rt.boldItalicFont != nil {
		frameOpts = append(frameOpts, rich.WithBoldItalicFont(rt.boldItalicFont))
	}
	if rt.codeFont != nil {
		frameOpts = append(frameOpts, rich.WithCodeFont(rt.codeFont))
	}
	for scale, f := range rt.scaledFonts {
		frameOpts = append(frameOpts, rich.WithScaledFont(scale, f))
	}
	if rt.imageCache != nil {
		frameOpts = append(frameOpts, rich.WithImageCache(rt.imageCache))
	}
	if rt.basePath != "" {
		frameOpts = append(frameOpts, rich.WithBasePath(rt.basePath))
	}
	if rt.selectionColor != nil {
		frameOpts = append(frameOpts, rich.WithSelectionColor(rt.selectionColor))
	}

	// Initialize frame with empty rectangle - will be set on first Render() call
	rt.frame.Init(image.Rectangle{}, frameOpts...)
}

// All returns the full rectangle area of the RichText component.
func (rt *RichText) All() image.Rectangle {
	return rt.lastRect
}

// Frame returns the underlying rich.Frame.
func (rt *RichText) Frame() rich.Frame {
	return rt.frame
}

// Display returns the display.
func (rt *RichText) Display() draw.Display {
	return rt.display
}

// ScrollRect returns the scrollbar rectangle.
func (rt *RichText) ScrollRect() image.Rectangle {
	return rt.lastScrollRect
}

// SetContent sets the content to display.
func (rt *RichText) SetContent(c rich.Content) {
	rt.content = c
	if rt.frame != nil {
		rt.frame.SetContent(c)
	}
}

// Content returns the current content.
func (rt *RichText) Content() rich.Content {
	return rt.content
}

// Selection returns the current selection range.
func (rt *RichText) Selection() (p0, p1 int) {
	if rt.frame == nil {
		return 0, 0
	}
	return rt.frame.GetSelection()
}

// SetSelection sets the selection range.
func (rt *RichText) SetSelection(p0, p1 int) {
	if rt.frame != nil {
		rt.frame.SetSelection(p0, p1)
	}
}

// Origin returns the current scroll origin.
func (rt *RichText) Origin() int {
	if rt.frame == nil {
		return 0
	}
	return rt.frame.GetOrigin()
}

// SetOrigin sets the scroll origin.
func (rt *RichText) SetOrigin(org int) {
	if rt.frame != nil {
		rt.frame.SetOrigin(org)
	}
}

// Redraw redraws the RichText component using the last rendered rectangle.
func (rt *RichText) Redraw() {
	// Draw scrollbar first (behind frame)
	rt.scrDraw()

	// Draw the frame content
	if rt.frame != nil {
		rt.frame.Redraw()
	}
}

// Render draws the rich text component into the given rectangle.
// This computes scrollbar and frame areas from r at render time,
// allowing the rectangle to be provided dynamically (e.g., from body.all).
func (rt *RichText) Render(r image.Rectangle) {
	rt.lastRect = r

	// Compute scrollbar rectangle (left side)
	scrollWid := rt.display.ScaleSize(Scrollwid)
	scrollGap := rt.display.ScaleSize(Scrollgap)

	rt.lastScrollRect = image.Rect(
		r.Min.X,
		r.Min.Y,
		r.Min.X+scrollWid,
		r.Max.Y,
	)

	// Compute gap rectangle (between scrollbar and frame)
	gapRect := image.Rect(
		r.Min.X+scrollWid,
		r.Min.Y,
		r.Min.X+scrollWid+scrollGap,
		r.Max.Y,
	)

	// Compute frame rectangle (right of scrollbar with gap)
	frameRect := image.Rect(
		r.Min.X+scrollWid+scrollGap,
		r.Min.Y,
		r.Max.X,
		r.Max.Y,
	)

	// Update frame geometry if changed
	if rt.frame != nil && rt.frame.Rect() != frameRect {
		rt.frame.SetRect(frameRect)
	}

	// Draw scrollbar
	rt.scrDraw()

	// Fill the gap with the frame background color
	if rt.display != nil && rt.background != nil {
		screen := rt.display.ScreenImage()
		screen.Draw(gapRect, rt.background, rt.background, image.ZP)
	}

	// Draw frame content
	if rt.frame != nil {
		rt.frame.Redraw()
	}
}

// scrDraw renders the scrollbar background and thumb using cached rectangles.
func (rt *RichText) scrDraw() {
	rt.scrDrawAt(rt.lastScrollRect)
}

// scrDrawAt renders the scrollbar at the given rectangle.
func (rt *RichText) scrDrawAt(scrollRect image.Rectangle) {
	if rt.display == nil {
		return
	}

	screen := rt.display.ScreenImage()

	// Draw scrollbar background
	if rt.scrollBg != nil {
		screen.Draw(scrollRect, rt.scrollBg, rt.scrollBg, image.ZP)
	}

	// Draw scrollbar thumb
	if rt.scrollThumb != nil {
		thumbRect := rt.scrThumbRectAt(scrollRect)
		screen.Draw(thumbRect, rt.scrollThumb, rt.scrollThumb, image.ZP)
	}
}

// ScrollClick handles a click on the scrollbar using cached rectangles.
// It takes the button number (1, 2, or 3) and the click point,
// calculates the new origin based on the button behavior, and returns it.
// Button 1 (left): scroll up (backward in content)
// Button 2 (middle): jump to absolute position
// Button 3 (right): scroll down (forward in content)
// The origin is also updated in the RichText component.
func (rt *RichText) ScrollClick(button int, pt image.Point) int {
	return rt.scrollClickAt(button, pt, rt.lastScrollRect)
}

// scrollClickAt handles a click on the scrollbar using a given scroll rectangle.
func (rt *RichText) scrollClickAt(button int, pt image.Point, scrollRect image.Rectangle) int {
	// If no content or frame, return 0
	if rt.content == nil || rt.frame == nil {
		return 0
	}

	totalRunes := rt.content.Len()
	if totalRunes == 0 {
		return 0
	}

	// Get visual line information from the frame
	lineCount := rt.frame.TotalLines()
	lineStarts := rt.frame.LineStartRunes()
	maxLines := rt.frame.MaxLines()

	// If all content fits, no scrolling needed
	if lineCount <= maxLines {
		return 0
	}

	// Calculate click position as a proportion of the scrollbar height
	scrollHeight := scrollRect.Dy()
	if scrollHeight <= 0 {
		return rt.Origin()
	}

	clickY := pt.Y - scrollRect.Min.Y
	if clickY < 0 {
		clickY = 0
	}
	if clickY > scrollHeight {
		clickY = scrollHeight
	}
	clickProportion := float64(clickY) / float64(scrollHeight)

	// Calculate the number of lines that can be scrolled
	// (total lines minus the lines that fit in the visible area)
	scrollableLines := lineCount - maxLines
	if scrollableLines < 0 {
		scrollableLines = 0
	}

	var newOrigin int

	switch button {
	case 1:
		// Button 1 (left): scroll up - move back by a number of lines based on click position
		// Clicking higher in the scrollbar scrolls up more
		linesToMove := int(float64(maxLines) * (1.0 - clickProportion))
		if linesToMove < 1 {
			linesToMove = 1
		}

		// Find current line
		currentOrigin := rt.Origin()
		currentLine := 0
		for i, start := range lineStarts {
			if currentOrigin >= start {
				currentLine = i
			} else {
				break
			}
		}

		// Calculate new line
		newLine := currentLine - linesToMove
		if newLine < 0 {
			newLine = 0
		}

		newOrigin = lineStarts[newLine]

	case 2:
		// Button 2 (middle): jump to absolute position based on click location
		// The click proportion maps to the entire content range (all lines)
		targetLine := int(float64(lineCount-1) * clickProportion)
		if targetLine < 0 {
			targetLine = 0
		}
		if targetLine >= len(lineStarts) {
			targetLine = len(lineStarts) - 1
		}
		newOrigin = lineStarts[targetLine]

	case 3:
		// Button 3 (right): scroll down - move forward by a number of lines based on click position
		// Clicking lower in the scrollbar scrolls down more
		linesToMove := int(float64(maxLines) * clickProportion)
		if linesToMove < 1 {
			linesToMove = 1
		}

		// Find current line
		currentOrigin := rt.Origin()
		currentLine := 0
		for i, start := range lineStarts {
			if currentOrigin >= start {
				currentLine = i
			} else {
				break
			}
		}

		// Calculate new line (can't go past the last scrollable line)
		newLine := currentLine + linesToMove
		maxScrollLine := len(lineStarts) - 1
		if newLine > maxScrollLine {
			newLine = maxScrollLine
		}

		newOrigin = lineStarts[newLine]

	default:
		return rt.Origin()
	}

	// Update the origin
	rt.SetOrigin(newOrigin)
	return newOrigin
}

// scrThumbRect returns the rectangle for the scrollbar thumb using cached rectangles.
func (rt *RichText) scrThumbRect() image.Rectangle {
	return rt.scrThumbRectAt(rt.lastScrollRect)
}

// scrThumbRectAt computes thumb position for a given scrollbar rectangle.
// The thumb position and size reflect the current scroll position and
// the proportion of visible content to total content.
func (rt *RichText) scrThumbRectAt(scrollRect image.Rectangle) image.Rectangle {
	// If no content or frame, fill the whole scrollbar
	if rt.content == nil || rt.frame == nil {
		return scrollRect
	}

	totalRunes := rt.content.Len()
	if totalRunes == 0 {
		// No content - thumb fills the whole scrollbar
		return scrollRect
	}

	// Get scroll metrics from the frame using visual line counts
	origin := rt.frame.GetOrigin()
	maxLines := rt.frame.MaxLines()
	lineCount := rt.frame.TotalLines()
	lineStarts := rt.frame.LineStartRunes()

	scrollHeight := scrollRect.Dy()

	// If all content fits, fill the scrollbar
	if lineCount <= maxLines {
		return scrollRect
	}

	// Calculate thumb height based on visible vs total lines
	visibleProportion := float64(maxLines) / float64(lineCount)
	if visibleProportion > 1.0 {
		visibleProportion = 1.0
	}

	thumbHeight := int(float64(scrollHeight) * visibleProportion)
	if thumbHeight < 10 {
		thumbHeight = 10 // Minimum thumb height for usability
	}

	// Find which line the origin corresponds to
	originLine := 0
	for i, start := range lineStarts {
		if origin >= start {
			originLine = i
		} else {
			break
		}
	}

	// Position proportion based on line position in the document
	// Use (lineCount - 1) as denominator so that last line maps to bottom.
	denominator := lineCount - 1
	if denominator < 1 {
		denominator = 1
	}
	posProportion := float64(originLine) / float64(denominator)
	if posProportion > 1.0 {
		posProportion = 1.0
	}

	// When viewing content near the end of the document (past ~70% of lines),
	// adjust the position to ensure the thumb reaches the bottom.
	// This ensures "near the end" positions map to "near the bottom" of scrollbar.
	endThreshold := float64(lineCount) * 0.70 // 70% threshold
	if float64(originLine) >= endThreshold {
		// Map from [endThreshold, lineCount-1] to [currentProportion, 1.0]
		// Scale faster toward 1.0 for end positions
		linesBeyondThreshold := float64(originLine) - endThreshold
		linesInEndRange := float64(lineCount-1) - endThreshold
		if linesInEndRange > 0 {
			// Linear approach to bottom - more aggressive than ease-out
			normalizedPos := linesBeyondThreshold / linesInEndRange
			// Remap: at normalizedPos 0.5, we want to be 90%+ of the way to the bottom
			adjustment := normalizedPos * (1.0 - posProportion)
			posProportion += adjustment
		}
	}

	// Available space for thumb movement
	availableSpace := scrollHeight - thumbHeight

	// Thumb top position
	thumbTop := scrollRect.Min.Y + int(float64(availableSpace)*posProportion)

	return image.Rect(
		scrollRect.Min.X,
		thumbTop,
		scrollRect.Max.X,
		thumbTop+thumbHeight,
	)
}

// RichTextOption is a functional option for configuring RichText.
type RichTextOption func(*RichText)

// WithRichTextBackground sets the background image for the rich text component.
func WithRichTextBackground(bg draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.background = bg
	}
}

// WithRichTextColor sets the text color image for the rich text component.
func WithRichTextColor(c draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.textColor = c
	}
}

// WithScrollbarColors sets the scrollbar background and thumb colors.
func WithScrollbarColors(bg, thumb draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.scrollBg = bg
		rt.scrollThumb = thumb
	}
}

// WithRichTextBoldFont sets the bold font variant for the RichText frame.
func WithRichTextBoldFont(f draw.Font) RichTextOption {
	return func(rt *RichText) {
		rt.boldFont = f
	}
}

// WithRichTextItalicFont sets the italic font variant for the RichText frame.
func WithRichTextItalicFont(f draw.Font) RichTextOption {
	return func(rt *RichText) {
		rt.italicFont = f
	}
}

// WithRichTextBoldItalicFont sets the bold-italic font variant for the RichText frame.
func WithRichTextBoldItalicFont(f draw.Font) RichTextOption {
	return func(rt *RichText) {
		rt.boldItalicFont = f
	}
}

// WithRichTextCodeFont sets the monospace font for code spans and code blocks.
func WithRichTextCodeFont(f draw.Font) RichTextOption {
	return func(rt *RichText) {
		rt.codeFont = f
	}
}

// WithRichTextScaledFont sets a scaled font for a specific scale factor (e.g., 2.0 for H1).
func WithRichTextScaledFont(scale float64, f draw.Font) RichTextOption {
	return func(rt *RichText) {
		if rt.scaledFonts == nil {
			rt.scaledFonts = make(map[float64]draw.Font)
		}
		rt.scaledFonts[scale] = f
	}
}

// WithRichTextImageCache sets the image cache for loading images in markdown content.
// The cache is passed through to the underlying Frame for use during layout.
func WithRichTextImageCache(cache *rich.ImageCache) RichTextOption {
	return func(rt *RichText) {
		rt.imageCache = cache
	}
}

// WithRichTextBasePath sets the base path for resolving relative image paths.
// This should be the path to the source file (e.g., markdown file) containing image references.
// When combined with WithRichTextImageCache, relative paths will be resolved relative to this path.
func WithRichTextBasePath(path string) RichTextOption {
	return func(rt *RichText) {
		rt.basePath = path
	}
}

// WithRichTextSelectionColor sets the selection highlight color.
// This color is used to highlight selected text in the rich text frame.
func WithRichTextSelectionColor(c draw.Image) RichTextOption {
	return func(rt *RichText) {
		rt.selectionColor = c
	}
}

// scrollWheelLines is the number of lines to scroll per mouse wheel event.
const scrollWheelLines = 3

// ScrollWheel handles mouse scroll wheel events.
// If up is true, scroll up (show earlier content), otherwise scroll down.
// Returns the new origin after scrolling.
func (rt *RichText) ScrollWheel(up bool) int {
	// If no content or frame, return 0
	if rt.content == nil || rt.frame == nil {
		return 0
	}

	totalRunes := rt.content.Len()
	if totalRunes == 0 {
		return 0
	}

	// Get visual line information from the frame
	lineCount := rt.frame.TotalLines()
	lineStarts := rt.frame.LineStartRunes()
	maxLines := rt.frame.MaxLines()

	// If all content fits, no scrolling needed
	if lineCount <= maxLines {
		return 0
	}

	// Find current line
	currentOrigin := rt.Origin()
	currentLine := 0
	for i, start := range lineStarts {
		if currentOrigin >= start {
			currentLine = i
		} else {
			break
		}
	}

	var newLine int
	if up {
		// Scroll up - go back scrollWheelLines lines
		newLine = currentLine - scrollWheelLines
		if newLine < 0 {
			newLine = 0
		}
	} else {
		// Scroll down - go forward scrollWheelLines lines
		newLine = currentLine + scrollWheelLines
		// Don't go past the last line
		maxScrollLine := len(lineStarts) - 1
		if newLine > maxScrollLine {
			newLine = maxScrollLine
		}
	}

	newOrigin := lineStarts[newLine]
	rt.SetOrigin(newOrigin)
	return newOrigin
}
