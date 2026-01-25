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

	// Rendering
	Redraw()

	// Status
	Full() bool // True if frame is at capacity
}

// frameImpl is the concrete implementation of Frame.
type frameImpl struct {
	rect       image.Rectangle
	display    edwooddraw.Display
	background edwooddraw.Image // background image for filling
	textColor  edwooddraw.Image // text color image for rendering
	font       edwooddraw.Font  // font for text rendering
	content    Content
	origin     int
	p0, p1     int // selection

	// Font variants for styled text
	boldFont       edwooddraw.Font
	italicFont     edwooddraw.Font
	boldItalicFont edwooddraw.Font

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
	lines := layout(boxes, f.font, frameWidth, maxtab)
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
					Y: f.rect.Min.Y + lastLine.Y + f.font.Height(),
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
func (f *frameImpl) Charofpt(pt image.Point) int {
	// TODO: Implement
	return 0
}

// Select handles mouse selection.
func (f *frameImpl) Select(mc *draw.Mousectl, m *draw.Mouse) (p0, p1 int) {
	// TODO: Implement
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
func (f *frameImpl) MaxLines() int {
	// TODO: Implement
	return 0
}

// VisibleLines returns the number of lines currently visible.
func (f *frameImpl) VisibleLines() int {
	// TODO: Implement
	return 0
}

// Redraw redraws the frame.
func (f *frameImpl) Redraw() {
	if f.display == nil || f.background == nil {
		return
	}
	// Fill the frame rectangle with the background color
	screen := f.display.ScreenImage()
	screen.Draw(f.rect, f.background, f.background, image.ZP)

	// Draw text if we have content, font, and text color
	if f.content != nil && f.font != nil && f.textColor != nil {
		f.drawText(screen)
	}
}

// drawText renders the content boxes onto the screen.
func (f *frameImpl) drawText(screen edwooddraw.Image) {
	// Convert content to boxes
	boxes := contentToBoxes(f.content)
	if len(boxes) == 0 {
		return
	}

	// Calculate frame width for layout
	frameWidth := f.rect.Dx()

	// Default tab width (8 characters worth)
	maxtab := 8 * f.font.StringWidth("0")

	// Layout boxes into lines
	lines := layout(boxes, f.font, frameWidth, maxtab)

	// Render each line
	for _, line := range lines {
		for _, pb := range line.Boxes {
			// Skip newlines and tabs - they don't render visible text
			if pb.Box.IsNewline() || pb.Box.IsTab() {
				continue
			}
			if len(pb.Box.Text) == 0 {
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
func (f *frameImpl) Full() bool {
	// TODO: Implement
	return false
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
