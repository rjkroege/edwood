package rich

import (
	"image"
	"image/color"
	"unicode/utf8"

	"9fans.net/go/draw"
	edwooddraw "github.com/rjkroege/edwood/draw"
)

// Option is a functional option for configuring a Frame.
type Option func(*frameImpl)

// Frame renders styled text content with selection support.
type Frame interface {
	// Initialization
	Init(r image.Rectangle, opts ...Option)
	Clear()

	// Content
	SetContent(c Content)

	// Geometry
	Rect() image.Rectangle
	SetRect(r image.Rectangle)   // Update the frame's rectangle
	Ptofchar(p int) image.Point  // Character position → screen point
	Charofpt(pt image.Point) int // Screen point → character position

	// Selection
	Select(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int)
	SetSelection(p0, p1 int)
	GetSelection() (p0, p1 int)

	// Scrolling
	SetOrigin(org int)
	GetOrigin() int
	MaxLines() int
	VisibleLines() int
	TotalLines() int       // Total number of layout lines in the content
	LineStartRunes() []int // Rune offset at the start of each visual line

	// Rendering
	Redraw()

	// Status
	Full() bool // True if frame is at capacity
}

// frameImpl is the concrete implementation of Frame.
type frameImpl struct {
	rect           image.Rectangle
	display        edwooddraw.Display
	background     edwooddraw.Image // background image for filling
	textColor      edwooddraw.Image // text color image for rendering
	selectionColor edwooddraw.Image // selection highlight color
	font           edwooddraw.Font  // font for text rendering
	content        Content
	origin         int
	p0, p1         int // selection

	// Font variants for styled text
	boldFont       edwooddraw.Font
	italicFont     edwooddraw.Font
	boldItalicFont edwooddraw.Font
	codeFont       edwooddraw.Font // monospace font for code spans

	// Scaled fonts for headings (key is scale factor: 2.0 for H1, 1.5 for H2, etc.)
	scaledFonts map[float64]edwooddraw.Font
}

// NewFrame creates a new Frame.
func NewFrame() Frame {
	return &frameImpl{}
}

// Init initializes the frame with the given rectangle and options.
func (f *frameImpl) Init(r image.Rectangle, opts ...Option) {
	f.rect = r
	for _, opt := range opts {
		opt(f)
	}
}

// Clear resets the frame.
func (f *frameImpl) Clear() {
	f.content = nil
	f.origin = 0
	f.p0 = 0
	f.p1 = 0
}

// SetContent sets the content to display.
func (f *frameImpl) SetContent(c Content) {
	f.content = c
}

// Rect returns the frame's rectangle.
func (f *frameImpl) Rect() image.Rectangle {
	return f.rect
}

// SetRect updates the frame's rectangle.
// This is used when the frame needs to be resized without full re-initialization.
// Subsequent calls to layout-dependent methods (TotalLines, Redraw, etc.)
// will use the new rectangle dimensions.
func (f *frameImpl) SetRect(r image.Rectangle) {
	f.rect = r
}

// Ptofchar maps a character position to a screen point.
// The position p is a rune offset into the content.
// Returns the screen point where that character would be drawn.
func (f *frameImpl) Ptofchar(p int) image.Point {
	if p <= 0 {
		return f.rect.Min
	}

	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return f.rect.Min
	}

	// Calculate frame width and tab width for layout
	frameWidth := f.rect.Dx()
	maxtab := 8 * f.font.StringWidth("0")

	// Layout boxes into lines
	lines := layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
	if len(lines) == 0 {
		return f.rect.Min
	}

	// Walk through positioned boxes counting runes
	runeCount := 0
	for _, line := range lines {
		for _, pb := range line.Boxes {
			boxRunes := pb.Box.Nrune
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				// Special characters count as 1 rune
				boxRunes = 1
			}

			// Check if position p is within this box
			if runeCount+boxRunes > p {
				// p is within this box, calculate offset within the box
				runeOffset := p - runeCount

				// For newline/tab, just return the start position
				if pb.Box.IsNewline() || pb.Box.IsTab() {
					return image.Point{
						X: f.rect.Min.X + pb.X,
						Y: f.rect.Min.Y + line.Y,
					}
				}

				// For text, measure the width of the first runeOffset runes
				text := pb.Box.Text
				byteOffset := 0
				for i := 0; i < runeOffset && byteOffset < len(text); i++ {
					_, size := utf8.DecodeRune(text[byteOffset:])
					byteOffset += size
				}
				partialWidth := f.fontForStyle(pb.Box.Style).BytesWidth(text[:byteOffset])

				return image.Point{
					X: f.rect.Min.X + pb.X + partialWidth,
					Y: f.rect.Min.Y + line.Y,
				}
			}

			runeCount += boxRunes
		}
	}

	// Position is past end of content - return position after last character
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		// Calculate X position at end of last line
		endX := 0
		for _, pb := range lastLine.Boxes {
			if pb.Box.IsNewline() {
				// After a newline, position is at start of next line
				return image.Point{
					X: f.rect.Min.X,
					Y: f.rect.Min.Y + lastLine.Y + lastLine.Height,
				}
			}
			endX = pb.X + pb.Box.Wid
		}
		return image.Point{
			X: f.rect.Min.X + endX,
			Y: f.rect.Min.Y + lastLine.Y,
		}
	}

	return f.rect.Min
}

// Charofpt maps a screen point to a character position.
// The point is in screen coordinates. Returns the rune offset
// of the character at that position.
func (f *frameImpl) Charofpt(pt image.Point) int {
	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return 0
	}

	// Calculate frame width and tab width for layout
	frameWidth := f.rect.Dx()
	maxtab := 8 * f.font.StringWidth("0")

	// Layout boxes into lines
	lines := layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
	if len(lines) == 0 {
		return 0
	}

	// Convert point to frame-relative coordinates
	relX := pt.X - f.rect.Min.X
	relY := pt.Y - f.rect.Min.Y

	// Handle points above or to the left of frame
	if relX < 0 {
		relX = 0
	}
	if relY < 0 {
		relY = 0
	}

	// Find which line the point is on
	lineIdx := 0
	for i, line := range lines {
		// Check if point is within this line's Y range
		lineTop := line.Y
		lineBottom := line.Y + line.Height
		if relY >= lineTop && relY < lineBottom {
			lineIdx = i
			break
		}
		// If we're past this line, keep updating lineIdx
		if relY >= lineTop {
			lineIdx = i
		}
	}

	// Count runes up to the target line
	runeCount := 0
	for i := 0; i < lineIdx; i++ {
		for _, pb := range lines[i].Boxes {
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				runeCount++
			} else {
				runeCount += pb.Box.Nrune
			}
		}
	}

	// Now find the position within the target line
	targetLine := lines[lineIdx]
	for _, pb := range targetLine.Boxes {
		boxStart := pb.X
		boxEnd := pb.X + pb.Box.Wid

		// Handle newline boxes (width 0, but still represent a character)
		if pb.Box.IsNewline() {
			// Point at or after the newline position returns the newline's position
			// We return here because we've found the position
			if relX >= boxStart {
				return runeCount
			}
			continue
		}

		// Handle tab boxes
		if pb.Box.IsTab() {
			if relX >= boxEnd {
				// Point is past this tab
				runeCount++
				continue
			}
			if relX >= boxStart {
				// Point is within the tab
				return runeCount
			}
			// Point is before this box
			return runeCount
		}

		// Handle text boxes
		if relX >= boxEnd {
			// Point is past this box
			runeCount += pb.Box.Nrune
			continue
		}

		if relX >= boxStart {
			// Point is within this box - find which character
			localX := relX - boxStart
			return runeCount + f.runeAtX(pb.Box.Text, pb.Box.Style, localX)
		}

		// Point is before this box (shouldn't normally happen
		// since boxes are laid out left to right)
		return runeCount
	}

	// Point is past all content on this line
	return runeCount
}

// runeAtX finds which rune in text corresponds to pixel offset x.
// Returns the rune index (0-based) within the text.
func (f *frameImpl) runeAtX(text []byte, style Style, x int) int {
	font := f.fontForStyle(style)
	cumWidth := 0
	runeIdx := 0

	for i := 0; i < len(text); {
		_, runeLen := utf8.DecodeRune(text[i:])
		runeWidth := font.BytesWidth(text[i : i+runeLen])

		// Check if x falls within this rune
		// We use midpoint - if x is in the first half, return current index
		// if in second half, return next index
		if cumWidth+runeWidth > x {
			// x is within this rune's span
			midpoint := cumWidth + runeWidth/2
			if x < midpoint {
				return runeIdx
			}
			return runeIdx
		}

		cumWidth += runeWidth
		runeIdx++
		i += runeLen
	}

	// x is past all runes
	return runeIdx
}

// Select handles mouse selection.
// It takes the Mousectl for reading subsequent mouse events and the
// initial mouse-down event. It tracks the mouse drag and returns the
// selection range (p0, p1) where p0 <= p1. The frame's internal
// selection state is also updated.
func (f *frameImpl) Select(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int) {
	// Get the initial position from the mouse-down event
	anchor := f.Charofpt(m.Point)
	current := anchor

	// Read mouse events until button is released
	for {
		me := <-mc.C
		current = f.Charofpt(me.Point)

		// Update selection as we drag (for visual feedback)
		if anchor <= current {
			f.p0 = anchor
			f.p1 = current
		} else {
			f.p0 = current
			f.p1 = anchor
		}

		// Redraw to show updated selection during drag
		f.Redraw()

		// Check if button was released
		if me.Buttons == 0 {
			break
		}
	}

	// Return normalized selection (p0 <= p1)
	return f.p0, f.p1
}

// SetSelection sets the selection range.
func (f *frameImpl) SetSelection(p0, p1 int) {
	f.p0 = p0
	f.p1 = p1
}

// GetSelection returns the current selection range.
func (f *frameImpl) GetSelection() (p0, p1 int) {
	return f.p0, f.p1
}

// SetOrigin sets the scroll origin.
func (f *frameImpl) SetOrigin(org int) {
	f.origin = org
}

// GetOrigin returns the current scroll origin.
func (f *frameImpl) GetOrigin() int {
	return f.origin
}

// MaxLines returns the maximum number of lines that can be displayed.
// This is based on the frame height divided by the font height.
func (f *frameImpl) MaxLines() int {
	if f.font == nil {
		return 0
	}
	fontHeight := f.font.Height()
	if fontHeight <= 0 {
		return 0
	}
	return f.rect.Dy() / fontHeight
}

// VisibleLines returns the number of lines currently visible.
// This accounts for the origin offset and line wrapping.
func (f *frameImpl) VisibleLines() int {
	if f.font == nil || f.content == nil {
		return 0
	}
	lines, _ := f.layoutFromOrigin()
	return len(lines)
}

// TotalLines returns the total number of layout lines in the content.
// This includes all lines after word wrapping, not just source newlines.
func (f *frameImpl) TotalLines() int {
	if f.font == nil || f.content == nil {
		return 0
	}

	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return 0
	}

	// Calculate frame width for layout
	frameWidth := f.rect.Dx()

	// Default tab width (8 characters worth)
	maxtab := 8 * f.font.StringWidth("0")

	// Layout all boxes
	lines := layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
	return len(lines)
}

// LineStartRunes returns the rune offset at the start of each visual line.
// This maps visual line indices to rune positions for scrolling.
func (f *frameImpl) LineStartRunes() []int {
	if f.font == nil || f.content == nil {
		return []int{0}
	}

	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return []int{0}
	}

	// Calculate frame width for layout
	frameWidth := f.rect.Dx()

	// Default tab width (8 characters worth)
	maxtab := 8 * f.font.StringWidth("0")

	// Layout all boxes
	lines := layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
	if len(lines) == 0 {
		return []int{0}
	}

	// Walk through lines and calculate rune offset at start of each line
	lineStarts := make([]int, len(lines))
	runeCount := 0
	for i, line := range lines {
		lineStarts[i] = runeCount
		// Count runes in this line
		for _, pb := range line.Boxes {
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				runeCount++
			} else {
				runeCount += pb.Box.Nrune
			}
		}
	}

	return lineStarts
}

// Redraw redraws the frame.
func (f *frameImpl) Redraw() {
	if f.display == nil || f.background == nil {
		return
	}
	// Fill the frame rectangle with the background color
	screen := f.display.ScreenImage()
	screen.Draw(f.rect, f.background, f.background, image.ZP)

	// Draw selection highlight (before text so text appears on top)
	if f.content != nil && f.font != nil && f.selectionColor != nil && f.p0 != f.p1 {
		f.drawSelection(screen)
	}

	// Draw text if we have content, font, and text color
	if f.content != nil && f.font != nil && f.textColor != nil {
		f.drawText(screen)
	}
}

// drawText renders the content boxes onto the screen.
func (f *frameImpl) drawText(screen edwooddraw.Image) {
	// Get layout lines starting from origin
	lines, _ := f.layoutFromOrigin()
	if len(lines) == 0 {
		return
	}

	// Phase 1: Draw block-level backgrounds (full line width for fenced code blocks)
	// This must happen first so text appears on top
	for _, line := range lines {
		// Check if any box on this line has Block=true with a background
		for _, pb := range line.Boxes {
			if pb.Box.Style.Block && pb.Box.Style.Bg != nil {
				f.drawBlockBackground(screen, line)
				break // Only draw once per line
			}
		}
	}

	// Phase 2: Draw box backgrounds (for inline code, etc.)
	// This must happen before text rendering so backgrounds appear behind text
	for _, line := range lines {
		for _, pb := range line.Boxes {
			// Skip newlines and tabs - they don't have backgrounds
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				continue
			}
			if len(pb.Box.Text) == 0 {
				continue
			}

			// Draw background if style has Bg color set, but NOT for block-level styles
			// (those are handled in Phase 1 with full-width backgrounds)
			if pb.Box.Style.Bg != nil && !pb.Box.Style.Block {
				f.drawBoxBackground(screen, pb, line)
			}
		}
	}

	// Phase 3: Draw horizontal rules
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.HRule {
				f.drawHorizontalRule(screen, line)
				break // Only one rule per line
			}
		}
	}

	// Phase 4: Render text on top of backgrounds
	for _, line := range lines {
		for _, pb := range line.Boxes {
			// Skip newlines and tabs - they don't render visible text
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				continue
			}
			if len(pb.Box.Text) == 0 {
				continue
			}
			// Skip horizontal rules - they are drawn as lines, not text
			if pb.Box.Style.HRule {
				continue
			}

			// Calculate screen position
			pt := image.Point{
				X: f.rect.Min.X + pb.X,
				Y: f.rect.Min.Y + line.Y,
			}

			// Determine text color: use box style Fg if set, otherwise default
			textColorImg := f.textColor
			if pb.Box.Style.Fg != nil {
				// Allocate an image for this color
				colorImg := f.allocColorImage(pb.Box.Style.Fg)
				if colorImg != nil {
					textColorImg = colorImg
				}
			}

			// Select the appropriate font for this box's style
			boxFont := f.fontForStyle(pb.Box.Style)

			// Render the text
			screen.Bytes(pt, textColorImg, image.ZP, boxFont, pb.Box.Text)
		}
	}
}

// drawBlockBackground draws a full-width background for a line.
// This is used for fenced code blocks where the background extends to the frame edge.
func (f *frameImpl) drawBlockBackground(screen edwooddraw.Image, line Line) {
	// Find the background color from a block-styled box on this line
	var bgColor color.Color
	for _, pb := range line.Boxes {
		if pb.Box.Style.Block && pb.Box.Style.Bg != nil {
			bgColor = pb.Box.Style.Bg
			break
		}
	}
	if bgColor == nil {
		return
	}

	bgImg := f.allocColorImage(bgColor)
	if bgImg == nil {
		return
	}

	// Full-width background: from frame left edge to frame right edge
	bgRect := image.Rect(
		f.rect.Min.X,
		f.rect.Min.Y+line.Y,
		f.rect.Max.X,
		f.rect.Min.Y+line.Y+line.Height,
	)

	screen.Draw(bgRect, bgImg, bgImg, image.ZP)
}

// drawBoxBackground draws the background color for a positioned box.
// This is used for inline code backgrounds and other text-width backgrounds.
func (f *frameImpl) drawBoxBackground(screen edwooddraw.Image, pb PositionedBox, line Line) {
	bgImg := f.allocColorImage(pb.Box.Style.Bg)
	if bgImg == nil {
		return
	}

	// Calculate the background rectangle for this box
	// X: from box start to box start + box width
	// Y: from line top to line top + line height
	bgRect := image.Rect(
		f.rect.Min.X+pb.X,
		f.rect.Min.Y+line.Y,
		f.rect.Min.X+pb.X+pb.Box.Wid,
		f.rect.Min.Y+line.Y+line.Height,
	)

	screen.Draw(bgRect, bgImg, bgImg, image.ZP)
}

// HRuleColor is the gray color used for horizontal rule lines.
var HRuleColor = color.RGBA{R: 180, G: 180, B: 180, A: 255}

// drawHorizontalRule draws a horizontal rule line across the full frame width.
// The line is drawn vertically centered within the line height.
func (f *frameImpl) drawHorizontalRule(screen edwooddraw.Image, line Line) {
	// Use a gray color for the rule
	ruleImg := f.allocColorImage(HRuleColor)
	if ruleImg == nil {
		return
	}

	// Draw a 1px line vertically centered in the line
	// The line spans the full frame width
	lineThickness := 1
	centerY := f.rect.Min.Y + line.Y + line.Height/2

	ruleRect := image.Rect(
		f.rect.Min.X,
		centerY,
		f.rect.Max.X,
		centerY+lineThickness,
	)

	screen.Draw(ruleRect, ruleImg, ruleImg, image.ZP)
}

// layoutFromOrigin returns the layout lines starting from the origin position.
// It skips content before the origin and adjusts Y coordinates so that the
// first visible content starts at Y=0.
// Returns the lines and the rune offset of the first visible content.
func (f *frameImpl) layoutFromOrigin() ([]Line, int) {
	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return nil, 0
	}

	// Calculate frame width for layout
	frameWidth := f.rect.Dx()

	// Default tab width (8 characters worth)
	maxtab := 8 * f.font.StringWidth("0")

	// If origin is 0, just return the normal layout
	if f.origin == 0 {
		return layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle), 0
	}

	// Layout all boxes first
	allLines := layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
	if len(allLines) == 0 {
		return nil, 0
	}

	// Find which line contains the origin position
	runeCount := 0
	startLineIdx := 0
	originY := 0

	for lineIdx, line := range allLines {
		lineStartRune := runeCount
		for _, pb := range line.Boxes {
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				runeCount++
			} else {
				runeCount += pb.Box.Nrune
			}
		}

		// Check if origin is within or at the start of this line
		if f.origin >= lineStartRune && f.origin < runeCount {
			startLineIdx = lineIdx
			originY = line.Y
			break
		}
		// If we've passed the origin position, the origin was at the end of the previous line
		if f.origin < runeCount {
			startLineIdx = lineIdx
			originY = line.Y
			break
		}
		// Keep track of the last line in case origin is past all content
		startLineIdx = lineIdx
		originY = line.Y
	}

	// Extract lines from the origin line onwards and adjust Y coordinates
	visibleLines := make([]Line, 0, len(allLines)-startLineIdx)
	for i := startLineIdx; i < len(allLines); i++ {
		line := allLines[i]
		// Adjust Y coordinate to start from 0, preserving Height
		adjustedLine := Line{
			Y:      line.Y - originY,
			Height: line.Height,
			Boxes:  line.Boxes,
		}
		visibleLines = append(visibleLines, adjustedLine)
	}

	return visibleLines, f.origin
}

// drawSelection renders the selection highlight rectangles.
// The selection spans from p0 to p1 (rune offsets).
// For multi-line selections, multiple rectangles are drawn.
func (f *frameImpl) drawSelection(screen edwooddraw.Image) {
	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return
	}

	// Calculate frame width and tab width for layout
	frameWidth := f.rect.Dx()
	maxtab := 8 * f.font.StringWidth("0")

	// Layout boxes into lines
	lines := layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
	if len(lines) == 0 {
		return
	}

	p0, p1 := f.p0, f.p1
	if p0 > p1 {
		p0, p1 = p1, p0
	}

	// Walk through lines and boxes, tracking rune position
	runePos := 0
	for _, line := range lines {
		lineStartRune := runePos
		lineEndRune := lineStartRune

		// Calculate the end rune position for this line
		for _, pb := range line.Boxes {
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				lineEndRune++
			} else {
				lineEndRune += pb.Box.Nrune
			}
		}

		// Check if this line overlaps with the selection
		if lineEndRune <= p0 || lineStartRune >= p1 {
			// No overlap with selection, skip this line
			runePos = lineEndRune
			continue
		}

		// This line has selected content - calculate the selection rectangle
		selStartX := -1 // Start of selection on this line (relative to line start)
		selEndX := 0    // End of selection on this line

		boxRunePos := lineStartRune
		for _, pb := range line.Boxes {
			boxRunes := pb.Box.Nrune
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				boxRunes = 1
			}

			boxStartRune := boxRunePos
			boxEndRune := boxStartRune + boxRunes

			// Check if selection starts in or before this box (only set once)
			if selStartX < 0 {
				if p0 <= boxStartRune {
					// Selection starts at or before this box
					selStartX = pb.X
				} else if p0 > boxStartRune && p0 < boxEndRune {
					// Selection starts within this box
					if pb.Box.IsNewline() || pb.Box.IsTab() {
						selStartX = pb.X
					} else {
						// Calculate partial position within the box
						runeOffset := p0 - boxStartRune
						selStartX = pb.X + f.runeWidthInBox(&pb.Box, runeOffset)
					}
				}
			}

			// Check if selection ends in or after this box
			if p1 >= boxEndRune {
				// Selection extends past this box
				selEndX = pb.X + pb.Box.Wid
			} else if p1 > boxStartRune && p1 < boxEndRune {
				// Selection ends within this box
				if pb.Box.IsNewline() || pb.Box.IsTab() {
					selEndX = pb.X + pb.Box.Wid
				} else {
					// Calculate partial position within the box
					runeOffset := p1 - boxStartRune
					selEndX = pb.X + f.runeWidthInBox(&pb.Box, runeOffset)
				}
			}

			boxRunePos = boxEndRune
		}

		// If selStartX wasn't set, default to 0
		if selStartX < 0 {
			selStartX = 0
		}
		if selEndX > frameWidth {
			selEndX = frameWidth
		}

		// Draw the selection rectangle for this line
		if selEndX > selStartX {
			selRect := image.Rect(
				f.rect.Min.X+selStartX,
				f.rect.Min.Y+line.Y,
				f.rect.Min.X+selEndX,
				f.rect.Min.Y+line.Y+line.Height,
			)
			screen.Draw(selRect, f.selectionColor, f.selectionColor, image.ZP)
		}

		runePos = lineEndRune
	}
}

// runeWidthInBox calculates the pixel width of the first n runes in a text box.
func (f *frameImpl) runeWidthInBox(box *Box, n int) int {
	if n <= 0 {
		return 0
	}
	text := box.Text
	byteOffset := 0
	for i := 0; i < n && byteOffset < len(text); i++ {
		_, size := utf8.DecodeRune(text[byteOffset:])
		byteOffset += size
	}
	return f.fontForStyle(box.Style).BytesWidth(text[:byteOffset])
}

// allocColorImage allocates (or retrieves from cache) an image for the given color.
func (f *frameImpl) allocColorImage(c color.Color) edwooddraw.Image {
	if f.display == nil {
		return nil
	}

	// Convert color.Color to draw.Color
	r, g, b, a := c.RGBA()
	// RGBA returns values in 0-65535 range, scale to 0-255
	drawColor := edwooddraw.Color(uint32(r>>8)<<24 | uint32(g>>8)<<16 | uint32(b>>8)<<8 | uint32(a>>8))

	// Allocate a replicated 1x1 image with this color
	img, err := f.display.AllocImage(image.Rect(0, 0, 1, 1), f.display.ScreenImage().Pix(), true, drawColor)
	if err != nil {
		return nil
	}
	return img
}

// Full returns true if the frame is at capacity.
// A frame is full when more content is visible than can fit in the frame.
func (f *frameImpl) Full() bool {
	return f.VisibleLines() > f.MaxLines()
}

// fontHeightForStyle returns the font height for a given style.
// This is used by the layout algorithm to calculate line heights.
func (f *frameImpl) fontHeightForStyle(style Style) int {
	return f.fontForStyle(style).Height()
}

// fontForStyle returns the appropriate font for the given style.
// Falls back to the regular font if the variant is not available.
// When a style has a Scale != 1.0, the scaled font takes precedence
// since it provides the correct metrics for heading layout.
func (f *frameImpl) fontForStyle(style Style) edwooddraw.Font {
	// Check for scaled fonts first (for headings like H1, H2, H3)
	// Scale takes precedence because heading layout requires the correct metrics
	if style.Scale != 1.0 && f.scaledFonts != nil {
		if scaledFont, ok := f.scaledFonts[style.Scale]; ok {
			return scaledFont
		}
	}

	// Check for code font (monospace for inline code and code blocks)
	if style.Code && f.codeFont != nil {
		return f.codeFont
	}

	// Check for bold/italic variants for non-scaled text
	if style.Bold && style.Italic {
		if f.boldItalicFont != nil {
			return f.boldItalicFont
		}
	} else if style.Bold {
		if f.boldFont != nil {
			return f.boldFont
		}
	} else if style.Italic {
		if f.italicFont != nil {
			return f.italicFont
		}
	}
	return f.font
}
