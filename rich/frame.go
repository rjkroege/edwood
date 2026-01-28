package rich

import (
	"image"
	"image/color"
	"unicode/utf8"

	"9fans.net/go/draw"
	edwooddraw "github.com/rjkroege/edwood/draw"
)

const (
	// frtickw is the tick (cursor) width in unscaled pixels, matching frame/frame.go.
	frtickw = 3
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
	SelectWithChord(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int, chordButtons int)
	SelectWithColor(mc *draw.Mousectl, m *draw.Mouse, col edwooddraw.Image) (p0, p1 int)
	SelectWithChordAndColor(mc *draw.Mousectl, m *draw.Mouse, col edwooddraw.Image) (p0, p1 int, chordButtons int)
	SetSelection(p0, p1 int)
	GetSelection() (p0, p1 int)

	// Scrolling
	SetOrigin(org int)
	GetOrigin() int
	MaxLines() int
	VisibleLines() int
	TotalLines() int           // Total number of layout lines in the content
	LineStartRunes() []int     // Rune offset at the start of each visual line
	LinePixelHeights() []int   // Pixel height of each visual line (accounts for images)

	// Rendering
	Redraw()

	// Content queries
	ImageURLAt(pos int) string // Returns image URL at position, or "" if not an image

	// Font metrics
	DefaultFontHeight() int // Height of the default font

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

	// Scratch image for clipped rendering - all drawing goes here first,
	// then blitted to screen. This ensures text doesn't overflow frame bounds.
	scratchImage edwooddraw.Image
	scratchRect  image.Rectangle // size of current scratch image

	// Image cache for loading images during layout
	imageCache *ImageCache

	// Base path for resolving relative image paths (e.g., the markdown file path)
	basePath string

	// Temporary sweep color override for colored selection during B2/B3 drags.
	// When non-nil, drawSelectionTo uses this instead of selectionColor.
	// Cleared after each SelectWithColor/SelectWithChordAndColor call.
	sweepColor edwooddraw.Image

	// Tick (cursor bar) fields for drawing the insertion cursor
	tickImage  edwooddraw.Image // pre-rendered tick mask (transparent + opaque pattern)
	tickScale  int              // display.ScaleSize(1)
	tickHeight int              // height of current tickImage (re-init when changed)
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

	// Use layoutFromOrigin to get viewport-relative lines and the origin rune offset.
	// p is a content-absolute rune position; we subtract originRune to get a
	// viewport-relative position for searching through the visible lines.
	lines, originRune := f.layoutFromOrigin()
	if len(lines) == 0 {
		return f.rect.Min
	}

	// Adjust p to be relative to the origin
	p -= originRune
	if p <= 0 {
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
	// Use layoutFromOrigin to get viewport-relative lines and the origin rune offset.
	// After scrolling, click coordinates are viewport-relative but layoutBoxes()
	// returns document-absolute Y positions. layoutFromOrigin() adjusts Y to start
	// from 0 at the first visible line.
	lines, originRune := f.layoutFromOrigin()
	if len(lines) == 0 {
		return originRune
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

	// Count runes up to the target line (viewport-relative)
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
				return originRune + runeCount
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
				return originRune + runeCount
			}
			// Point is before this box
			return originRune + runeCount
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
			return originRune + runeCount + f.runeAtX(pb.Box.Text, pb.Box.Style, localX)
		}

		// Point is before this box (shouldn't normally happen
		// since boxes are laid out left to right)
		return originRune + runeCount
	}

	// Point is past all content on this line
	return originRune + runeCount
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

// ImageURLAt returns the ImageURL if the given character position falls within
// an image box. Returns empty string if not an image.
func (f *frameImpl) ImageURLAt(pos int) string {
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return ""
	}

	// Walk through boxes counting runes until we find the one containing pos
	runeCount := 0
	for _, box := range boxes {
		var boxRunes int
		if box.IsNewline() || box.IsTab() {
			boxRunes = 1
		} else {
			boxRunes = box.Nrune
		}

		// Check if pos falls within this box
		if pos >= runeCount && pos < runeCount+boxRunes {
			if box.Style.Image && box.Style.ImageURL != "" {
				return box.Style.ImageURL
			}
			return ""
		}

		runeCount += boxRunes
	}

	return ""
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

		// Flush the display to make selection visible immediately
		if f.display != nil {
			f.display.Flush()
		}

		// Check if button was released
		if me.Buttons == 0 {
			break
		}
	}

	// Return normalized selection (p0 <= p1)
	return f.p0, f.p1
}

// SelectWithChord handles mouse selection with chord detection.
// Like Select, it tracks drag from the initial B1 mouse-down event,
// but also detects when additional buttons (B2, B3) are pressed during
// the drag. Returns the selection range and the button state at chord
// time (0 if no chord was detected, i.e. only B1 was held).
func (f *frameImpl) SelectWithChord(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int, chordButtons int) {
	anchor := f.Charofpt(m.Point)
	current := anchor
	initialButtons := m.Buttons

	for {
		me := <-mc.C
		current = f.Charofpt(me.Point)

		if anchor <= current {
			f.p0 = anchor
			f.p1 = current
		} else {
			f.p0 = current
			f.p1 = anchor
		}

		f.Redraw()

		if f.display != nil {
			f.display.Flush()
		}

		// Detect chord: additional buttons pressed beyond the initial button
		if me.Buttons != 0 && me.Buttons != initialButtons && chordButtons == 0 {
			chordButtons = me.Buttons
		}

		if me.Buttons == 0 {
			break
		}
	}

	return f.p0, f.p1, chordButtons
}

// SelectWithColor performs a mouse drag selection using a custom sweep color
// for the selection highlight during the drag. After the drag completes, the
// sweep color is cleared so subsequent redraws use the normal selectionColor.
// This matches normal Acme's SelectOpt behavior for B2 (red) and B3 (green) sweeps.
func (f *frameImpl) SelectWithColor(mc *draw.Mousectl, m *draw.Mouse, col edwooddraw.Image) (p0, p1 int) {
	f.sweepColor = col
	defer func() { f.sweepColor = nil }()
	return f.Select(mc, m)
}

// SelectWithChordAndColor performs a mouse drag selection with chord detection
// using a custom sweep color for the selection highlight during the drag.
// After the drag completes, the sweep color is cleared.
func (f *frameImpl) SelectWithChordAndColor(mc *draw.Mousectl, m *draw.Mouse, col edwooddraw.Image) (p0, p1 int, chordButtons int) {
	f.sweepColor = col
	defer func() { f.sweepColor = nil }()
	return f.SelectWithChord(mc, m)
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

	// Layout all boxes (using cache if available)
	lines := f.layoutBoxes(boxes, frameWidth, maxtab)
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

	// Layout all boxes (using cache if available)
	lines := f.layoutBoxes(boxes, frameWidth, maxtab)
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

// LinePixelHeights returns the pixel height of each visual line.
// For lines containing images, the height will be larger than the default font height.
func (f *frameImpl) LinePixelHeights() []int {
	if f.font == nil || f.content == nil {
		return nil
	}

	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return nil
	}

	frameWidth := f.rect.Dx()
	maxtab := 8 * f.font.StringWidth("0")
	lines := f.layoutBoxes(boxes, frameWidth, maxtab)

	heights := make([]int, len(lines))
	for i, line := range lines {
		heights[i] = line.Height
	}
	return heights
}

// Redraw redraws the frame.
func (f *frameImpl) Redraw() {
	if f.display == nil || f.background == nil {
		return
	}

	screen := f.display.ScreenImage()

	// Ensure scratch image exists and is the right size.
	// The scratch image is used to clip text rendering - we draw to it first,
	// then blit to the screen. This prevents text from overflowing frame bounds.
	scratch := f.ensureScratchImage()
	if scratch == nil {
		// Fallback: draw directly to screen (no clipping for text)
		scratch = screen
	}

	// Calculate the destination rectangle for drawing.
	// If using scratch image, we draw at origin (0,0) since scratch is frame-sized.
	// If drawing directly to screen, we draw at f.rect.Min.
	var drawRect image.Rectangle
	var drawOffset image.Point // offset to add when calculating screen coordinates
	if scratch != screen {
		// Drawing to scratch: use local coordinates (0,0 origin)
		drawRect = image.Rect(0, 0, f.rect.Dx(), f.rect.Dy())
		drawOffset = image.ZP
	} else {
		// Drawing directly to screen: use frame coordinates
		drawRect = f.rect
		drawOffset = f.rect.Min
	}

	// Fill with background color
	scratch.Draw(drawRect, f.background, f.background, image.ZP)

	// Draw selection highlight (before text so text appears on top)
	if f.content != nil && f.font != nil && f.selectionColor != nil && f.p0 != f.p1 {
		f.drawSelectionTo(scratch, drawOffset)
	}

	// Draw text if we have content, font, and text color
	if f.content != nil && f.font != nil && f.textColor != nil {
		f.drawTextTo(scratch, drawOffset)
	}

	// Draw cursor tick when selection is a point (p0 == p1)
	if f.content != nil && f.font != nil && f.display != nil && f.p0 == f.p1 {
		f.drawTickTo(scratch, drawOffset)
	}

	// If we used a scratch image, blit it to the screen
	if scratch != screen {
		screen.Draw(f.rect, scratch, nil, image.ZP)
	}
}

// ensureScratchImage allocates or resizes the scratch image to match frame dimensions.
// Returns nil if allocation fails.
func (f *frameImpl) ensureScratchImage() edwooddraw.Image {
	frameSize := image.Rect(0, 0, f.rect.Dx(), f.rect.Dy())

	// Check if we already have a correctly-sized scratch image
	if f.scratchImage != nil && f.scratchRect.Eq(frameSize) {
		return f.scratchImage
	}

	// Free old scratch image if it exists
	if f.scratchImage != nil {
		f.scratchImage.Free()
		f.scratchImage = nil
	}

	// Allocate new scratch image
	pix := f.display.ScreenImage().Pix()
	img, err := f.display.AllocImage(frameSize, pix, false, 0)
	if err != nil {
		return nil
	}

	f.scratchImage = img
	f.scratchRect = frameSize
	return f.scratchImage
}

// drawTextTo renders the content boxes onto the target image.
// The offset parameter specifies where the frame's (0,0) maps to in the target.
// When drawing to a scratch image, offset is (0,0). When drawing directly to
// screen, offset is f.rect.Min.
func (f *frameImpl) drawTextTo(target edwooddraw.Image, offset image.Point) {
	// Get layout lines starting from origin
	lines, _ := f.layoutFromOrigin()
	if len(lines) == 0 {
		return
	}

	// frameHeight is used to skip lines that start completely outside the frame
	frameHeight := f.rect.Dy()
	frameWidth := f.rect.Dx()


	// Phase 1: Draw block-level backgrounds (full line width for fenced code blocks)
	// This must happen first so text appears on top
	for _, line := range lines {
		// Skip lines that start at or below the frame bottom
		if line.Y >= frameHeight {
			break
		}
		// Check if any box on this line has Block=true with a background
		for _, pb := range line.Boxes {
			if pb.Box.Style.Block && pb.Box.Style.Bg != nil {
				f.drawBlockBackgroundTo(target, line, offset, frameWidth, frameHeight)
				break // Only draw once per line
			}
		}
	}

	// Phase 2: Draw box backgrounds (for inline code, etc.)
	// This must happen before text rendering so backgrounds appear behind text
	for _, line := range lines {
		// Skip lines that start at or below the frame bottom
		if line.Y >= frameHeight {
			break
		}
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
				f.drawBoxBackgroundTo(target, pb, line, offset, frameWidth, frameHeight)
			}
		}
	}

	// Phase 3: Draw horizontal rules
	for _, line := range lines {
		// Skip lines that start at or below the frame bottom
		if line.Y >= frameHeight {
			break
		}
		for _, pb := range line.Boxes {
			if pb.Box.Style.HRule {
				f.drawHorizontalRuleTo(target, line, offset, frameWidth, frameHeight)
				break // Only one rule per line
			}
		}
	}

	// Phase 4: Render text on top of backgrounds
	// Note: Text is now clipped by the scratch image bounds, so we can render
	// partial lines without worrying about overflow into adjacent windows.
	for _, line := range lines {
		// Skip lines that start at or below the frame bottom
		if line.Y >= frameHeight {
			break
		}
		for _, pb := range line.Boxes {
			// Skip newlines and tabs - they don't render visible text
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				continue
			}
			// Skip images - they are handled in Phase 5
			if pb.Box.Style.Image {
				continue
			}
			if len(pb.Box.Text) == 0 {
				continue
			}
			// Skip horizontal rules - they are drawn as lines, not text
			if pb.Box.Style.HRule {
				continue
			}

			// Calculate position in target image
			pt := image.Point{
				X: offset.X + pb.X,
				Y: offset.Y + line.Y,
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
			target.Bytes(pt, textColorImg, image.ZP, boxFont, pb.Box.Text)
		}
	}

	// Phase 5: Render images
	for _, line := range lines {
		// Skip lines that start at or below the frame bottom
		if line.Y >= frameHeight {
			break
		}
		for _, pb := range line.Boxes {
			// Check if this is an image box
			if !pb.Box.Style.Image {
				continue
			}

			// Calculate position in target image
			pt := image.Point{
				X: offset.X + pb.X,
				Y: offset.Y + line.Y,
			}

			// Check for error placeholder case
			if pb.Box.ImageData != nil && pb.Box.ImageData.Err != nil {
				f.drawImageErrorPlaceholder(target, pt, pb.Box.ImageData.Path, pb.Box.Style.ImageAlt)
				continue
			}

			// Check if we have valid image data to render
			if !pb.Box.IsImage() {
				f.drawImageErrorPlaceholder(target, pt, pb.Box.Style.ImageURL, pb.Box.Style.ImageAlt)
				continue
			}

			// Render the actual image
			f.drawImageTo(target, pb, line, offset, frameWidth, frameHeight)
		}
	}
}

// drawBlockBackgroundTo draws a full-width background for a line.
// This is used for fenced code blocks where the background extends to the frame edge.
func (f *frameImpl) drawBlockBackgroundTo(target edwooddraw.Image, line Line, offset image.Point, frameWidth, frameHeight int) {
	// Find the background color and left indent from a block-styled box on this line
	var bgColor color.Color
	leftIndent := 0
	for _, pb := range line.Boxes {
		if pb.Box.Style.Block && pb.Box.Style.Bg != nil {
			bgColor = pb.Box.Style.Bg
			leftIndent = pb.X // Use the box's X position as the left edge
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

	// Background from indent to right edge (not full-width)
	bgRect := image.Rect(
		offset.X+leftIndent,
		offset.Y+line.Y,
		offset.X+frameWidth,
		offset.Y+line.Y+line.Height,
	)

	// Clip to frame bounds (in target coordinates)
	clipRect := image.Rect(offset.X, offset.Y, offset.X+frameWidth, offset.Y+frameHeight)
	bgRect = bgRect.Intersect(clipRect)
	if bgRect.Empty() {
		return
	}

	target.Draw(bgRect, bgImg, bgImg, image.ZP)
}

// drawBoxBackgroundTo draws the background color for a positioned box.
// This is used for inline code backgrounds and other text-width backgrounds.
func (f *frameImpl) drawBoxBackgroundTo(target edwooddraw.Image, pb PositionedBox, line Line, offset image.Point, frameWidth, frameHeight int) {
	bgImg := f.allocColorImage(pb.Box.Style.Bg)
	if bgImg == nil {
		return
	}

	// Calculate the background rectangle for this box
	// X: from box start to box start + box width
	// Y: from line top to line top + line height
	bgRect := image.Rect(
		offset.X+pb.X,
		offset.Y+line.Y,
		offset.X+pb.X+pb.Box.Wid,
		offset.Y+line.Y+line.Height,
	)

	// Clip to frame bounds (in target coordinates)
	clipRect := image.Rect(offset.X, offset.Y, offset.X+frameWidth, offset.Y+frameHeight)
	bgRect = bgRect.Intersect(clipRect)
	if bgRect.Empty() {
		return
	}

	target.Draw(bgRect, bgImg, bgImg, image.ZP)
}

// HRuleColor is the gray color used for horizontal rule lines.
var HRuleColor = color.RGBA{R: 180, G: 180, B: 180, A: 255}

// drawHorizontalRuleTo draws a horizontal rule line across the full frame width.
// The line is drawn vertically centered within the line height.
func (f *frameImpl) drawHorizontalRuleTo(target edwooddraw.Image, line Line, offset image.Point, frameWidth, frameHeight int) {
	// Use a gray color for the rule
	ruleImg := f.allocColorImage(HRuleColor)
	if ruleImg == nil {
		return
	}

	// Draw a 1px line vertically centered in the line
	// The line spans the full frame width
	lineThickness := 1
	centerY := offset.Y + line.Y + line.Height/2

	ruleRect := image.Rect(
		offset.X,
		centerY,
		offset.X+frameWidth,
		centerY+lineThickness,
	)

	// Clip to frame bounds (in target coordinates)
	clipRect := image.Rect(offset.X, offset.Y, offset.X+frameWidth, offset.Y+frameHeight)
	ruleRect = ruleRect.Intersect(clipRect)
	if ruleRect.Empty() {
		return
	}

	target.Draw(ruleRect, ruleImg, ruleImg, image.ZP)
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

	// If origin is 0, just return the normal layout (using cache if available)
	if f.origin == 0 {
		return f.layoutBoxes(boxes, frameWidth, maxtab), 0
	}

	// Layout all boxes first (using cache if available)
	allLines := f.layoutBoxes(boxes, frameWidth, maxtab)
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

// drawSelectionTo renders the selection highlight rectangles.
// The selection spans from p0 to p1 (rune offsets).
// For multi-line selections, multiple rectangles are drawn.
func (f *frameImpl) drawSelectionTo(target edwooddraw.Image, offset image.Point) {
	// Use layoutFromOrigin to get viewport-relative lines and origin rune offset.
	// Selection positions (f.p0, f.p1) are content-absolute, so we subtract
	// originRune to compare against viewport-relative rune counting.
	lines, originRune := f.layoutFromOrigin()
	if len(lines) == 0 {
		return
	}

	frameWidth := f.rect.Dx()
	frameHeight := f.rect.Dy()

	p0, p1 := f.p0, f.p1
	if p0 > p1 {
		p0, p1 = p1, p0
	}
	// Adjust selection to viewport-relative rune positions
	p0 -= originRune
	p1 -= originRune

	// Walk through lines and boxes, tracking rune position
	runePos := 0
	for _, line := range lines {
		// Skip lines that start at or below the frame bottom
		if line.Y >= frameHeight {
			break
		}

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
				offset.X+selStartX,
				offset.Y+line.Y,
				offset.X+selEndX,
				offset.Y+line.Y+line.Height,
			)
			// Clip to frame bounds (in target coordinates)
			clipRect := image.Rect(offset.X, offset.Y, offset.X+frameWidth, offset.Y+frameHeight)
			selRect = selRect.Intersect(clipRect)
			if !selRect.Empty() {
				color := f.selectionColor
				if f.sweepColor != nil {
					color = f.sweepColor
				}
				target.Draw(selRect, color, nil, image.ZP)
			}
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

// DefaultFontHeight returns the height of the default font.
func (f *frameImpl) DefaultFontHeight() int {
	if f.font != nil {
		return f.font.Height()
	}
	return 0
}

// initTick creates or recreates the tick image when the required height changes.
// The tick image is a transparent mask with an opaque vertical line and serif boxes,
// matching the pattern from frame/tick.go:InitTick().
func (f *frameImpl) initTick(height int) {
	if f.display == nil {
		return
	}
	if f.tickImage != nil && f.tickHeight == height {
		return
	}
	if f.tickImage != nil {
		f.tickImage.Free()
		f.tickImage = nil
	}

	scale := f.display.ScaleSize(1)
	f.tickScale = scale
	w := frtickw * scale

	b := f.display.ScreenImage()
	img, err := f.display.AllocImage(
		image.Rect(0, 0, w, height),
		b.Pix(), false, edwooddraw.Transparent)
	if err != nil {
		return
	}

	// Fill transparent
	img.Draw(img.R(), f.display.Transparent(), nil, image.ZP)
	// Vertical line in center
	img.Draw(image.Rect(scale*(frtickw/2), 0, scale*(frtickw/2+1), height),
		f.display.Opaque(), nil, image.ZP)
	// Top serif box
	img.Draw(image.Rect(0, 0, w, w),
		f.display.Opaque(), nil, image.ZP)
	// Bottom serif box
	img.Draw(image.Rect(0, height-w, w, height),
		f.display.Opaque(), nil, image.ZP)

	f.tickImage = img
	f.tickHeight = height
}

// boxHeight returns the height of a box in pixels.
// For text boxes, this is the font height for the box's style.
// For image boxes, this is the scaled image height (via imageBoxDimensions).
func (f *frameImpl) boxHeight(box Box) int {
	if box.Style.Image && box.IsImage() {
		_, h := imageBoxDimensions(&box, f.rect.Dx())
		if h > 0 {
			return h
		}
	}
	return f.fontForStyle(box.Style).Height()
}

// drawTickTo draws the cursor tick (insertion bar) on the target image when
// the selection is a point (p0 == p1). It walks the layout to find the cursor
// position, determines height from the tallest adjacent box, and draws the tick.
func (f *frameImpl) drawTickTo(target edwooddraw.Image, offset image.Point) {
	if f.display == nil || f.font == nil {
		return
	}

	lines, originRune := f.layoutFromOrigin()
	if len(lines) == 0 {
		return
	}

	cursorPos := f.p0 - originRune
	if cursorPos < 0 {
		return
	}

	// Walk lines and boxes to find the cursor position, its X coordinate,
	// and the heights of adjacent boxes.
	runeCount := 0
	for _, line := range lines {
		for i, pb := range line.Boxes {
			boxRunes := pb.Box.Nrune
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				boxRunes = 1
			}

			// Check if cursor is at the start of this box
			if runeCount == cursorPos {
				x := pb.X

				// Adjacent heights: prev box (if any) and this box
				prevHeight := 0
				if i > 0 {
					prevHeight = f.boxHeight(line.Boxes[i-1].Box)
				}
				nextHeight := f.boxHeight(pb.Box)
				tickH := prevHeight
				if nextHeight > tickH {
					tickH = nextHeight
				}
				if tickH == 0 {
					tickH = f.font.Height()
				}

				f.initTick(tickH)
				if f.tickImage == nil {
					return
				}

				w := frtickw * f.tickScale
				r := image.Rect(
					offset.X+x, offset.Y+line.Y,
					offset.X+x+w, offset.Y+line.Y+tickH,
				)
				target.Draw(r, f.display.Black(), f.tickImage, image.ZP)
				return
			}

			// Check if cursor is within this box
			if runeCount+boxRunes > cursorPos {
				// Cursor is inside this box — compute X offset within the box
				runeOffset := cursorPos - runeCount
				var x int
				if pb.Box.IsNewline() || pb.Box.IsTab() {
					x = pb.X
				} else {
					byteOffset := 0
					text := pb.Box.Text
					for j := 0; j < runeOffset && byteOffset < len(text); j++ {
						_, size := utf8.DecodeRune(text[byteOffset:])
						byteOffset += size
					}
					x = pb.X + f.fontForStyle(pb.Box.Style).BytesWidth(text[:byteOffset])
				}

				// The cursor is within this box, so both adjacent boxes are this box
				tickH := f.boxHeight(pb.Box)
				if tickH == 0 {
					tickH = f.font.Height()
				}

				f.initTick(tickH)
				if f.tickImage == nil {
					return
				}

				w := frtickw * f.tickScale
				r := image.Rect(
					offset.X+x, offset.Y+line.Y,
					offset.X+x+w, offset.Y+line.Y+tickH,
				)
				target.Draw(r, f.display.Black(), f.tickImage, image.ZP)
				return
			}

			runeCount += boxRunes
		}
	}

	// Cursor is at end of content — use last box's height
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		// Compute X at end of last line
		endX := 0
		for _, pb := range lastLine.Boxes {
			if pb.Box.IsNewline() {
				endX = 0 // after newline, cursor is at start of next line
			} else {
				endX = pb.X + pb.Box.Wid
			}
		}

		tickH := f.font.Height()
		if len(lastLine.Boxes) > 0 {
			lastBox := lastLine.Boxes[len(lastLine.Boxes)-1].Box
			h := f.boxHeight(lastBox)
			if h > 0 {
				tickH = h
			}
		}

		f.initTick(tickH)
		if f.tickImage == nil {
			return
		}

		y := lastLine.Y
		// If last box was a newline, cursor goes on next line
		if len(lastLine.Boxes) > 0 && lastLine.Boxes[len(lastLine.Boxes)-1].Box.IsNewline() {
			y = lastLine.Y + lastLine.Height
			endX = 0
		}

		w := frtickw * f.tickScale
		r := image.Rect(
			offset.X+endX, offset.Y+y,
			offset.X+endX+w, offset.Y+y+tickH,
		)
		target.Draw(r, f.display.Black(), f.tickImage, image.ZP)
	}
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

// layoutBoxes runs the layout algorithm on the given boxes.
// If an imageCache is set on the frame, it uses layoutWithCacheAndBasePath to load
// images and populate their ImageData. Otherwise, it uses the regular layout.
func (f *frameImpl) layoutBoxes(boxes []Box, frameWidth, maxtab int) []Line {
	if f.imageCache != nil {
		return layoutWithCacheAndBasePath(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle, f.imageCache, f.basePath)
	}
	return layout(boxes, f.font, frameWidth, maxtab, f.fontHeightForStyle, f.fontForStyle)
}

// drawImageTo renders an image box to the target at the appropriate position.
// The image is clipped to the frame boundaries using Intersect.
func (f *frameImpl) drawImageTo(target edwooddraw.Image, pb PositionedBox, line Line, offset image.Point, frameWidth, frameHeight int) {
	if f.display == nil {
		return
	}

	cached := pb.Box.ImageData
	if cached == nil || cached.Data == nil || cached.Original == nil {
		return
	}

	// Calculate the scaled dimensions for the image
	scaledWidth, scaledHeight := imageBoxDimensions(&pb.Box, frameWidth)
	if scaledWidth == 0 || scaledHeight == 0 {
		return
	}

	// Calculate the destination rectangle
	dstX := offset.X + pb.X
	dstY := offset.Y + line.Y

	// Create destination rectangle for the image
	dstRect := image.Rect(dstX, dstY, dstX+scaledWidth, dstY+scaledHeight)

	// Clip to frame bounds
	clipRect := image.Rect(offset.X, offset.Y, offset.X+frameWidth, offset.Y+frameHeight)
	clippedDst := dstRect.Intersect(clipRect)
	if clippedDst.Empty() {
		return
	}

	// Allocate an image to hold the pixel data
	// Use RGBA32 format to match our ConvertToPlan9 output
	srcRect := image.Rect(0, 0, cached.Width, cached.Height)
	srcImg, err := f.display.AllocImage(srcRect, edwooddraw.RGBA32, false, 0)
	if err != nil {
		// Fall back to error placeholder
		pt := image.Point{X: dstX, Y: dstY}
		f.drawImageErrorPlaceholder(target, pt, cached.Path, pb.Box.Style.ImageAlt)
		return
	}
	defer srcImg.Free()

	// Load the pixel data into the source image
	_, err = srcImg.Load(srcRect, cached.Data)
	if err != nil {
		// Fall back to error placeholder
		pt := image.Point{X: dstX, Y: dstY}
		f.drawImageErrorPlaceholder(target, pt, cached.Path, pb.Box.Style.ImageAlt)
		return
	}

	// Calculate the source point for clipping
	// If the destination was clipped, we need to adjust which part of the source we draw
	srcPt := image.ZP
	if dstRect.Min.X < clippedDst.Min.X {
		// Left edge was clipped, adjust source X
		srcPt.X = (clippedDst.Min.X - dstRect.Min.X) * cached.Width / scaledWidth
	}
	if dstRect.Min.Y < clippedDst.Min.Y {
		// Top edge was clipped, adjust source Y
		srcPt.Y = (clippedDst.Min.Y - dstRect.Min.Y) * cached.Height / scaledHeight
	}

	// Draw the image (using the display's draw operation for scaling)
	// Note: Plan 9's draw doesn't do automatic scaling, so for scaled images
	// we would need to either:
	// 1. Pre-scale the image data before loading
	// 2. Use a draw operation that supports scaling
	// For now, we draw at the original size and clip
	//
	// When image is scaled down (scaledWidth < cached.Width), we draw the original
	// and it will be clipped. For proper scaling, we'd need to scale the pixel data.
	if scaledWidth == cached.Width && scaledHeight == cached.Height {
		// No scaling needed, direct draw
		target.Draw(clippedDst, srcImg, nil, srcPt)
	} else {
		// Image needs scaling - for now, draw at original size (clipped)
		// A more sophisticated implementation would scale the pixel data first
		// This produces correct results for images that fit; larger images are clipped
		actualDst := image.Rect(dstX, dstY, dstX+cached.Width, dstY+cached.Height)
		actualDst = actualDst.Intersect(clipRect)
		if !actualDst.Empty() {
			actualSrcPt := image.ZP
			if dstX < actualDst.Min.X {
				actualSrcPt.X = actualDst.Min.X - dstX
			}
			if dstY < actualDst.Min.Y {
				actualSrcPt.Y = actualDst.Min.Y - dstY
			}
			target.Draw(actualDst, srcImg, nil, actualSrcPt)
		}
	}
}

// drawImageErrorPlaceholder renders an error placeholder for failed image loads.
// It displays "[Image: alt]" in blue (like a link) so it can be clicked to open the image path.
func (f *frameImpl) drawImageErrorPlaceholder(target edwooddraw.Image, pt image.Point, path string, alt string) {
	if f.font == nil || f.textColor == nil {
		return
	}

	// Create placeholder text with alt text
	placeholder := "[Image: " + alt + "]"
	if alt == "" {
		placeholder = "[Image]"
	}

	// Use blue (like links) so users know it's clickable
	blueColor := f.allocColorImage(LinkBlue)
	if blueColor == nil {
		blueColor = f.textColor // Fall back to default text color
	}

	// Render the placeholder text
	target.Bytes(pt, blueColor, image.ZP, f.font, []byte(placeholder))
}
