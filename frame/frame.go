package frame

import (
	"9fans.net/go/draw"
	"image"
)

// TODO(rjk): Make this into a struct of colours?
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
	f.maxtab = m
}

// GetMaxtab returns the current maximum size of a tab in pixels.
func (f *Frame) GetMaxtab() int { return f.maxtab }

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

func (f *Frame) Rect() image.Rectangle {
	return f.rect
}

type Frame struct {
	font       Fontmetrics
	display    *draw.Display           // on which the frame is displayed
	background *draw.Image             // on which the frame appears
	cols       [NumColours]*draw.Image // background and text colours
	rect       image.Rectangle         // in which the text appears

	defaultfontheight int // height of default font

	box []*frbox // the boxes of text in this frame.

	sp0, sp1 int // bounds of a selection
	maxtab   int // max size of a tab (in pixels)
	nchars   int // number of runes in frame
	nlines   int // number of lines with text

	// TODO(rjk): figure out what to do about this for multiple line fonts.
	maxlines     int // total number of lines in frame
	lastlinefull bool
	modified     bool

	tickimage   *draw.Image // typing tick
	tickback    *draw.Image // image under tick
	ticked      bool        // Is the tick on.
	highlighton bool        // True if the highlight is painted.

	// Set this to true to indicate that the Frame should not emit drawing ops.
	// Use this if the Frame is being used "headless" to measure some text.
	noredraw  bool
	tickscale int // tick scaling factor
}

// NewFrame creates a new Frame with Font ft, background image b, colours cols, and
// of the size r
func NewFrame(r image.Rectangle, ft *draw.Font, b *draw.Image, cols [NumColours]*draw.Image) *Frame {
	f := new(Frame)
	f.Init(r, OptColors(cols), OptFont(ft), OptBackground(b), OptMaxTab(8))
	return f
}

// optioncontext is context passed into each option function
// that aggregates knowledge about additional updates needed
// to do to the Frame object that should only be one once per
// call to Init.
type optioncontext struct {
	updatetick  bool // True if the tick needs to initialized
	maxtabchars int  // Number of '0' characters that should be the width of a tab.
}

// TODO(rjk): Relocate the documentation somehow. And maybe the code.
// Option handling per https://commandcenter.blogspot.ca/2014/01/self-referential-functions-and-design.html
//
// Returns true if the option requires resetting the tick.
// TODO(rjk): It is possible to generalize this as needed with a more
// complex state object. One might imagine a set of updater functions?
type option func(*Frame, *optioncontext)

// Option sets the options specified and returns true if
// we need to init the tick.
func (f *Frame) Option(opts ...option) *optioncontext {
	ctx := &optioncontext{
		updatetick:  false,
		maxtabchars: -1,
	}

	for _, opt := range opts {
		opt(f, ctx)
	}
	return ctx
}

// OptColors sets the default colours.
func OptColors(cols [NumColours]*draw.Image) option {
	return func(f *Frame, ctx *optioncontext) {
		f.cols = cols
		// TODO(rjk): I think so. Make sure that this is required.
		ctx.updatetick = true
	}
}

// OptBackground sets the background screen image.
func OptBackground(b *draw.Image) option {
	return func(f *Frame, ctx *optioncontext) {
		f.background = b
		// TODO(rjk): This is safe but is it necessary? I think so.
		ctx.updatetick = true
	}
}

// OptFont sets the default font.
func OptFont(ft *draw.Font) option {
	return func(f *Frame, ctx *optioncontext) {
		f.font = &frfont{ft}
		ctx.updatetick = f.defaultfontheight != f.font.DefaultHeight()
	}
}

// OptFontMetrics sets the default font metrics object.
func OptFontMetrics(ft Fontmetrics) option {
	return func(f *Frame, ctx *optioncontext) {
		f.font = ft
		ctx.updatetick = f.defaultfontheight != f.font.DefaultHeight()
	}
}

// OptMaxTab sets the default tabwidth in `0` characters.
func OptMaxTab(maxtabchars int) option {
	return func(f *Frame, ctx *optioncontext) {
		ctx.maxtabchars = maxtabchars
	}
}

// computemaxtab returns the new ftw value
func (ctx *optioncontext) computemaxtab(maxtab, ftw int) int {
	if ctx.maxtabchars < 0 {
		return maxtab
	}
	return ctx.maxtabchars * ftw
}

func (f *Frame) DefaultFontHeight() int {
	return f.defaultfontheight
}

// Init prepares the Frame f for the display of text in rectangle r.
// Frame f will re-use previously set FontMetrics, colours and
// destination image for drawing unless these are overriden with
// one or more instances of the OptColors, OptBackground
// OptFont or OptFontMetrics option settings.
//
// The background (OptBackground setter) may be null to allow
// calling the other routines to maintain the model in, for example,
// an obscured window.
//
// Changing the background or font will force the tick to be
// recreated.
//
// TODO(rjk): This may do unnecessary work for some option settings.
// At some point, consider the code carefully.
func (f *Frame) Init(r image.Rectangle, opts ...option) {
	f.nchars = 0
	f.nlines = 0
	f.sp0 = 0
	f.sp1 = 0
	f.box = nil
	f.lastlinefull = false

	// Update additional options. The values are optional so that the frame
	// will re-use the existing values if new ones are not provided.
	ctx := f.Option(opts...)

	f.defaultfontheight = f.font.DefaultHeight()
	f.display = f.background.Display
	f.maxtab = ctx.computemaxtab(f.maxtab, f.font.StringWidth("0"))
	f.setrects(r)

	if ctx.updatetick || (f.tickimage == nil && f.cols[ColBack] != nil) {
		f.InitTick()
	}
}

// InitTick sets up the TickImage (e.g. cursor)
func (f *Frame) InitTick() {

	var err error
	if f.cols[ColBack] == nil || f.display == nil {
		return
	}

	f.tickscale = f.display.ScaleSize(1)
	b := f.display.ScreenImage
	ft := f.font

	if f.tickimage != nil {
		f.tickimage.Free()
	}

	height := ft.DefaultHeight()

	f.tickimage, err = f.display.AllocImage(image.Rect(0, 0, f.tickscale*frtickw, height), b.Pix, false, draw.Transparent)
	if err != nil {
		return
	}

	f.tickback, err = f.display.AllocImage(f.tickimage.R, b.Pix, false, draw.White)
	if err != nil {
		f.tickimage.Free()
		f.tickimage = nil
		return
	}
	f.tickback.Draw(f.tickback.R, f.cols[ColBack], nil, image.ZP)

	f.tickimage.Draw(f.tickimage.R, f.display.Transparent, nil, image.Pt(0, 0))
	// vertical line
	f.tickimage.Draw(image.Rect(f.tickscale*(frtickw/2), 0, f.tickscale*(frtickw/2+1), height), f.display.Opaque, nil, image.Pt(0, 0))
	// box on each end
	f.tickimage.Draw(image.Rect(0, 0, f.tickscale*frtickw, f.tickscale*frtickw), f.display.Opaque, nil, image.Pt(0, 0))
	f.tickimage.Draw(image.Rect(0, height-f.tickscale*frtickw, f.tickscale*frtickw, height), f.display.Opaque, nil, image.Pt(0, 0))
}

// setrects initializes the geometry of the frame.
func (f *Frame) setrects(r image.Rectangle) {
	height := f.defaultfontheight
	f.rect = r
	f.rect.Max.Y -= (r.Max.Y - r.Min.Y) % height
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
		f.tickimage.Free()
		f.tickback.Free()
		f.tickimage = nil
		f.tickback = nil
	}
	f.ticked = false
}
