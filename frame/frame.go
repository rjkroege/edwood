package frame

import (
	"9fans.net/go/draw"
	"image"

	"log"
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
	draw.Font
}

func (ff *frfont) DefaultHeight() int {
	return ff.Font.Height
}

func (ff *frfont) Impl() *draw.Font {
	return &ff.Font
}

// Maxtab sets the maximum size of a tab in pixels.
func (f *Frame) Maxtab(m int) {
	f.maxtab = m
}

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
	// TODO(rjk): This shouldn't exist. box is a slice and can manage this itself.
	// private
	nbox, nalloc int

	p0, p1 int // bounds of the selection
	maxtab int // max size of a tab (in pixels)
	nchars int // number of runes in frame
	nlines int // number of lines with text
	// TODO(rjk): figure out what to do about this for multiple line fonts.
	maxlines int // total number of lines in frame

	// TODO(rjk): make a bool
	// ro. Doesn't need a getter. Used only with frinsert and frdelete. Return from there.
	lastlinefull int

	modified  bool
	tickimage *draw.Image // typing tick
	tickback  *draw.Image // image under tick

	// TODO(rjk): Expose. public ro
	ticked bool

	// TODO(rjk): Expose public rw.
	noredraw  bool
	tickscale int // tick scaling factor
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
	f.Font = &frfont{*ft}
	f.Display = b.Display
	f.maxtab = 8 * ft.StringWidth("0")
	f.nbox = 0
	f.nalloc = 0
	f.nchars = 0
	f.nlines = 0
	f.p0 = 0
	f.p1 = 0
	f.box = nil
	f.lastlinefull = 0
	f.Cols = cols
	f.SetRects(r, b)

	if f.tickimage == nil && f.Cols[ColBack] != nil {
		f.InitTick()
	}
}

// InitTick sets up the tick (e.g. cursor)
func (f *Frame) InitTick() {
	log.Println("InitTick called")

	var err error
	if f.Cols[ColBack] == nil || f.Display == nil {
		return
	}

	f.tickscale = f.Display.ScaleSize(1)
	b := f.Display.ScreenImage
	ft := f.Font

	if f.tickimage != nil {
		f.tickimage.Free()
	}

	height := ft.DefaultHeight()

	f.tickimage, err = f.Display.AllocImage(image.Rect(0, 0, f.tickscale*frtickw, height), b.Pix, false, draw.Transparent)
	if err != nil {
		return
	}

	f.tickback, err = f.Display.AllocImage(f.tickimage.R, b.Pix, false, draw.White)
	if err != nil {
		f.tickimage.Free()
		f.tickimage = nil
		return
	}
	f.tickback.Draw(f.tickback.R, f.Cols[ColBack], nil, image.ZP)

	f.tickimage.Draw(f.tickimage.R, f.Display.Transparent, nil, image.Pt(0, 0))
	// vertical line
	f.tickimage.Draw(image.Rect(f.tickscale*(frtickw/2), 0, f.tickscale*(frtickw/2+1), height), f.Display.Opaque, nil, image.Pt(0, 0))
	// box on each end
	f.tickimage.Draw(image.Rect(0, 0, f.tickscale*frtickw, f.tickscale*frtickw), f.Display.Opaque, nil, image.Pt(0, 0))
	f.tickimage.Draw(image.Rect(0, height-f.tickscale*frtickw, f.tickscale*frtickw, height), f.Display.Opaque, nil, image.Pt(0, 0))
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
	if f.nbox != 0 {
		f.delbox(0, f.nbox-1)
	}
	if f.box != nil {
		f.box = nil
		f.nbox = 0
		f.nalloc = 0
	}
	if freeall {
		f.tickimage.Free()
		f.tickback.Free()
		f.tickimage = nil
		f.tickback = nil
	}
	f.ticked = false
}

