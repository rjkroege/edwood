package frame

import (
	"9fans.net/go/draw"
	"image"
)

const (
	ColBack = iota
	ColHigh
	ColBord
	ColText
	ColHText
	NumColours

	frtickw = 3
)

// TODO(rjk): no need for this to have public fields
type frbox struct {
	Wid    int    // In pixels. Fixed large size for layout box.
	Nrune  int    // Number of runes in Ptr or -1 for special layout boxes (tab, newline)
	Ptr    []byte // UTF-8 string in this box.
	Bc     rune   // The kind of special layout box: '\n' or '\t'
	Minwid byte
}

// Fontmetrics lets tests mock the calls into draw for measuring the
// width of UTF8 slices.
type Fontmetrics interface {
	BytesWidth([]byte) int
	DefaultHeight() int
	Impl() *draw.Font
	StringWidth(string) int
	RunesWidth(r []rune) int
}

type frfont struct {
	*draw.Font
}

func (ff *frfont) DefaultHeight() int {
	return ff.Font.Height
}

func (ff *frfont) Impl() *draw.Font {
	return ff.Font
}

// Maxtab sets the maximum size of a tab in pixels.
func (f *Frame) Maxtab(m int) {
	f.MaxTab = m
}

// GetMaxtab returns the current maximum size of a tab in pixels.
func (f *Frame) GetMaxtab() int { return f.MaxTab }

// FrameFillStatus is a snapshot of the capacity of the Frame.
type FrameFillStatus struct {
	Nchars   int
	Nlines   int
	Maxlines int
}

// GetFrameFillStatus returns a snapshot of the capacity of the frame.
func (f *Frame) GetFrameFillStatus() FrameFillStatus {
	return FrameFillStatus{
		Nchars:   f.nchars,
		Nlines:   f.nlines,
		Maxlines: f.maxlines,
	}
}

func (f *Frame) IsLastLineFull() bool {
	return f.lastlinefull
}

type Frame struct {
	// TODO(rjk): Remove public access if possible.
	Font       Fontmetrics
	Display    *draw.Display           // on which the frame is displayed
	Background *draw.Image             // on which the frame appears
	Cols       [NumColours]*draw.Image // background and text colours
	Rect       image.Rectangle         // in which the text appears
	Entire     image.Rectangle         // size of full frame

	// TODO(rjk): Figure out what.
	Scroll func(*Frame, int) // function provided by application

	box []*frbox // the boxes of text in this frame.

	P0, P1 int // bounds of a selection
	MaxTab int // max size of a tab (in pixels)
	nchars int // number of runes in frame
	nlines int // number of lines with text

	// TODO(rjk): figure out what to do about this for multiple line fonts.
	maxlines int // total number of lines in frame

	lastlinefull bool

	Modified  bool
	TickImage *draw.Image // typing tick
	TickBack  *draw.Image // image under tick

	// TODO(rjk): Expose. public ro
	Ticked bool

	// TODO(rjk): Expose public rw.
	// Set this to true to indicate that the Frame should not emit drawing ops.
	// Use this if the Frame is being used "headless" to measure some text.
	NoRedraw  bool
	TickScale int // tick scaling factor

	highlighton bool // True if the highlight is painted.
}

// NewFrame creates a new Frame with Font ft, background image b, colours cols, and
// of the size r
func NewFrame(r image.Rectangle, ft *draw.Font, b *draw.Image, cols [NumColours]*draw.Image) *Frame {
	f := new(Frame)
	f.Init(r, ft, b, cols)
	return f
}

// Init prepares the Frame f so characters drawn in it will appear in the
// single Font ft. It then calls SetRects and InitTick to initialize the
// geometry for the Frame. The Image b is where the Frame is to be drawn;
// Rectangle r defines the limit of the portion of the Image the text
// will occupy. The Image pointer may be null, allowing the other
// routines to be called to maintain the associated data structure in,
// for example, an obscured window.
func (f *Frame) Init(r image.Rectangle, ft *draw.Font, b *draw.Image, cols [NumColours]*draw.Image) {
	f.Font = &frfont{ft}
	f.Display = b.Display
	f.MaxTab = 8 * ft.StringWidth("0")
	f.nchars = 0
	f.nlines = 0
	f.P0 = 0
	f.P1 = 0
	f.box = nil
	f.lastlinefull = false
	f.Cols = cols
	f.SetRects(r, b)

	if f.TickImage == nil && f.Cols[ColBack] != nil {
		f.InitTick()
	}
}

// InitTick sets up the TickImage (e.g. cursor)
func (f *Frame) InitTick() {

	var err error
	if f.Cols[ColBack] == nil || f.Display == nil {
		return
	}

	f.TickScale = f.Display.ScaleSize(1)
	b := f.Display.ScreenImage
	ft := f.Font

	if f.TickImage != nil {
		f.TickImage.Free()
	}

	height := ft.DefaultHeight()

	f.TickImage, err = f.Display.AllocImage(image.Rect(0, 0, f.TickScale*frtickw, height), b.Pix, false, draw.Transparent)
	if err != nil {
		return
	}

	f.TickBack, err = f.Display.AllocImage(f.TickImage.R, b.Pix, false, draw.White)
	if err != nil {
		f.TickImage.Free()
		f.TickImage = nil
		return
	}
	f.TickBack.Draw(f.TickBack.R, f.Cols[ColBack], nil, image.ZP)

	f.TickImage.Draw(f.TickImage.R, f.Display.Transparent, nil, image.Pt(0, 0))
	// vertical line
	f.TickImage.Draw(image.Rect(f.TickScale*(frtickw/2), 0, f.TickScale*(frtickw/2+1), height), f.Display.Opaque, nil, image.Pt(0, 0))
	// box on each end
	f.TickImage.Draw(image.Rect(0, 0, f.TickScale*frtickw, f.TickScale*frtickw), f.Display.Opaque, nil, image.Pt(0, 0))
	f.TickImage.Draw(image.Rect(0, height-f.TickScale*frtickw, f.TickScale*frtickw, height), f.Display.Opaque, nil, image.Pt(0, 0))
}

// SetRects initializes the geometry of the frame.
func (f *Frame) SetRects(r image.Rectangle, b *draw.Image) {
	height := f.Font.DefaultHeight()
	f.Background = b
	f.Entire = r
	f.Rect = r
	f.Rect.Max.Y -= (r.Max.Y - r.Min.Y) % height
	f.maxlines = (r.Max.Y - r.Min.Y) / height
}

// Clear frees the internal structures associated with f, permitting
// another Init or SetRects on the Frame. It does not clear the
// associated display. If f is to be deallocated, the associated Font and
// Image must be freed separately. The resize argument should be non-zero
// if the frame is to be redrawn with a different font; otherwise the
// frame will maintain some data structures associated with the font.
//
// /To resize a Frame, use Clear and Init and then Insert to recreate the
// /display. If a Frame is being moved but not resized, that is, if the
// /shape of its containing rectangle is unchanged, it is sufficient to
// /use Draw to copy the containing rectangle from the old to the new
// /location and then call SetRects to establish the new geometry. (It is
// /unnecessary to call InitTick unless the font size has changed.) No
// /redrawing is necessary.
func (f *Frame) Clear(freeall bool) {
	f.box = make([]*frbox, 0, 25)
	if freeall {
		f.TickImage.Free()
		f.TickBack.Free()
		f.TickImage = nil
		f.TickBack = nil
	}
	f.Ticked = false
}
