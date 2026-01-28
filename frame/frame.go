package frame

import (
	"image"
	"sync"

	"github.com/rjkroege/edwood/draw"
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

// SelectScrollUpdater are those frame.Frame methods offered to
// frame.Select callbacks.
type SelectScrollUpdater interface {
	// GetFrameFillStatus returns a snapshot of the capacity of the frame.
	GetFrameFillStatus() FrameFillStatus

	// Charofpt returns the index of the closest rune whose image's upper
	// left corner is up and to the left of pt.
	Charofpt(pt image.Point) int

	// DefaultFontHeight returns the height of the Frame's default font.
	// TODO(rjk): Reconsider this for Frames containing many styles.
	DefaultFontHeight() int

	// Delete deletes from the Frame the text between p0 and p1; p1 points at
	// the first rune beyond the deletion. Returns the number of whole lines
	// removed.
	//
	// Delete will clear a selection or tick if present but not put it back.
	Delete(int, int) int

	// Insert inserts r into Frame f starting at rune index p0.
	// If a NUL (0) character is inserted, chaos will ensue. Tabs
	// and newlines are handled by the library, but all other characters,
	// including control characters, are just displayed. For example,
	// backspaces are printed; to erase a character, use Delete.
	//
	// Insert will remove the selection or tick  if present but update selection offsets.
	Insert([]rune, int) bool
	InsertByte([]byte, int) bool

	IsLastLineFull() bool
	Rect() image.Rectangle

	// TextOccupiedHeight returns the height of the region in the frame
	// occupied by boxes (which in the future could be of varying height)
	// that is closest to the height of rectangle r such that only unclipped
	// boxes fit in the returned height. If r.Dy() exeeds the total height of
	// the current boxes, then returns the height of current set of boxes.
	TextOccupiedHeight(r image.Rectangle) int
}

// Frame is the public interface to a frame of text. Unlike the C implementation,
// new Frame instances should be created with NewFrame.
type Frame interface {
	SelectScrollUpdater

	// Maxtab sets the maximum size of a tab in pixels.
	Maxtab(m int)

	// GetMaxtab returns the current maximum size of a tab in pixels.
	GetMaxtab() int

	// Init prepares the Frame for the display of text in rectangle r.
	// Frame f will reuse previously set font, colours, tab width and
	// destination image for drawing unless these are overridden with
	// one or more instances of the OptColors, OptBackground
	// OptFont or OptMaxTab option settings.
	//
	// The background (OptBackground setter) may be null to allow
	// calling the other routines to maintain the model in, for example,
	// an obscured window.
	//
	// Changing the background or font will force the tick to be
	// recreated.
	Init(image.Rectangle, ...OptionClosure)

	// Clear frees the internal structures associated with f, permitting
	// another Init or SetRects on the Frame. It does not clear the
	// associated display. If f is to be deallocated, the associated Font and
	// Image must be freed separately. The resize argument should be non-zero
	// if the frame is to be redrawn with a different font; otherwise the
	// frame will maintain some data structures associated with the font.
	//
	// To resize a Frame, use Clear and Init and then Insert to recreate the
	// display. If a Frame is being moved but not resized, that is, if the
	// shape of its containing rectangle is unchanged, it is sufficient to
	// use Draw to copy the containing rectangle from the old to the new
	// location and then call SetRects to establish the new geometry.
	Clear(bool)

	// Ptofchar returns the location of the upper left corner of the p'th
	// rune, starting from 0, in the receiver Frame. If the Frame holds
	// fewer than p runes, Ptofchar returns the location of the upper right
	// corner of the last character in the Frame
	Ptofchar(int) image.Point

	// Redraw redraws the background of the Frame where the Frame is inside
	// enclosing. Frame is responsible for drawing all of the pixels inside
	// enclosing though may fill less than enclosing with text. (In particular,
	// a margin may be added and the rectangle occupied by text is always
	// a multiple of the fixed line height.)
	// TODO(rjk): Modify this function to redraw the text as well and stop having
	// the drawing of text strings be a side-effect of Insert, Delete, etc.
	// TODO(rjk): Draw text to the bottom of enclosing as opposed to filling the
	// bottom partial text row with blank.
	//
	// Note: this function is not part of the documented libframe entrypoints and
	// was not invoked from Edwood code. Consequently, I am repurposing the name.
	// Future changes will have this function able to clear the Frame and draw the
	// entire box model.
	Redraw(enclosing image.Rectangle)

	// GetSelectionExtent returns the rune offsets of the selection maintained by
	// the Frame.
	GetSelectionExtent() (int, int)

	// Select takes ownership of the mouse channel to update the selection
	// so long as a button is down in downevent. Selection stops when the
	// staring point buttondown is altered. getmorelines is a callback provided
	// by the caller to provide n additional lines on demand to the specified frame.
	// The implementation of the callback must use the Frame instance provided
	// in place of the one that Select is invoked on.
	//
	// Select returns the selection range in the Frame.
	Select(*draw.Mousectl, *draw.Mouse, func(SelectScrollUpdater, int)) (int, int)

	// SelectOpt makes a selection in the same fashion as Select but does it in a
	// temporary way with the specified text colours fg, bg.
	SelectOpt(*draw.Mousectl, *draw.Mouse, func(SelectScrollUpdater, int), draw.Image, draw.Image) (int, int)

	// DrawSel repaints a section of the frame, delimited by rune
	// positions p0 and p1, either with plain background or entirely
	// highlighted, according to the flag highlighted, managing the tick
	// appropriately. The point pt0 is the geometrical location of p0 on the
	// screen; like all of the selection-helper routines' Point arguments, it
	// must be a value generated by Ptofchar.
	//
	// Clarification of semantics: the point of this routine is to redraw the
	// state of the Frame with selection p0, p1. In particular, this requires
	// updating f.p0 and f.p1 so that other entry points (e.g. Insert) can (transparently) remove
	// a pre-existing selection.
	//
	// Note that the original C code does not remove the pre-existing selection where
	// this code does draw the selection to the p0, p1. I (rjk) believe that this is a better
	// API.
	//
	// DrawSel does the minimum work needed to clear a highlight and (in particular)
	// multiple calls to DrawSel with highlighted false will be cheap.
	// TODO(rjk): DrawSel does more drawing work than necessary.
	DrawSel(image.Point, int, int, bool)
}

// TODO(rjk): Consider calling this SetMaxtab?
func (f *frameimpl) Maxtab(m int) {
	f.lk.Lock()
	defer f.lk.Unlock()

	f.maxtab = m
}

func (f *frameimpl) GetMaxtab() int { return f.maxtab }

// FrameFillStatus is a snapshot of the capacity of the Frame.
type FrameFillStatus struct {
	Nchars         int
	Nlines         int
	Maxlines       int
	MaxPixelHeight int
}

func (f *frameimpl) GetFrameFillStatus() FrameFillStatus {
	f.lk.Lock()
	defer f.lk.Unlock()
	return FrameFillStatus{
		Nchars:         f.nchars,
		Nlines:         f.nlines,
		Maxlines:       f.maxlines,
		MaxPixelHeight: f.maxlines * f.defaultfontheight,
	}
}

func (f *frameimpl) TextOccupiedHeight(r image.Rectangle) int {
	f.lk.Lock()
	defer f.lk.Unlock()

	return f.textoccupiedheightimpl(r)
}

func (f *frameimpl) textoccupiedheightimpl(r image.Rectangle) int {
	f.lk.Lock()
	defer f.lk.Unlock()

	// TODO(rjk): To support multiple different fonts at once in a Frame,
	// this will have to be extended to be the sum of the height of the boxes
	// less than r.Dy
	if r.Dy() > f.nlines*f.defaultfontheight {
		return f.nlines * f.defaultfontheight
	}
	return (r.Dy() / f.defaultfontheight) * f.defaultfontheight
}

func (f *frameimpl) IsLastLineFull() bool {
	f.lk.Lock()
	defer f.lk.Unlock()
	return f.lastlinefull
}

func (f *frameimpl) Rect() image.Rectangle {
	f.lk.Lock()
	defer f.lk.Unlock()
	return f.rect
}

// TODO(rjk): no need for this to have public fields.
// TODO(rjk): Could fold Minwid && Bc into Nrune.
type frbox struct {
	Wid    int    // In pixels. Fixed large size for layout box.
	Nrune  int    // Number of runes in Ptr or -1 for special layout boxes (tab, newline)
	Ptr    []byte // UTF-8 string in this box.
	Bc     rune   // The kind of special layout box: '\n' or '\t'
	Minwid byte
}

// TODO(rjk): It might make sense to group frameimpl into context (e.g.
// fonts, etc.) and the actual boxes. At any rate, it's worth thinking
// carefully about the data structures and how they should really be put
// together.
type frameimpl struct {
	lk sync.Mutex

	font       draw.Font
	display    draw.Display           // on which the frame is displayed
	background draw.Image             // on which the frame appears
	cols       [NumColours]draw.Image // background and text colours
	rect       image.Rectangle        // in which the text appears

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

	tickimage   draw.Image // typing tick
	tickback    draw.Image // image under tick
	ticked      bool       // Is the tick on.
	highlighton bool       // True if the highlight is painted.

	// Set this to true to indicate that the Frame should not emit drawing ops.
	// Use this if the Frame is being used "headless" to measure some text.
	noredraw  bool
	tickscale int // tick scaling factor
}

// NewFrame creates a new Frame with Font ft, background image b, colours cols, and
// of the size r
func NewFrame(r image.Rectangle, ft draw.Font, b draw.Image, cols [NumColours]draw.Image) Frame {
	f := new(frameimpl)
	f.Init(r, OptColors(cols), OptFont(ft), OptBackground(b), OptMaxTab(8))
	return f
}

func (f *frameimpl) DefaultFontHeight() int {
	f.lk.Lock()
	defer f.lk.Unlock()
	return f.defaultfontheight
}

// TODO(rjk): This may do unnecessary work for some option settings.
// At some point, consider the code carefully.
func (f *frameimpl) Init(r image.Rectangle, opts ...OptionClosure) {
	f.lk.Lock()
	defer f.lk.Unlock()
	f.nchars = 0
	f.nlines = 0
	f.sp0 = 0
	f.sp1 = 0
	f.box = nil
	f.lastlinefull = false

	// Update additional options. The values are optional so that the frame
	// will re-use the existing values if new ones are not provided.
	ctx := f.Option(opts...)

	f.defaultfontheight = f.font.Height()
	f.display = f.background.Display()
	f.maxtab = ctx.computemaxtab(f.maxtab, f.font.StringWidth("0"))
	f.setrects(r)

	if ctx.updatetick || (f.tickimage == nil && f.cols[ColBack] != nil) {
		f.InitTick()
	}
}

// setrects initializes the geometry of the frame.
func (f *frameimpl) setrects(r image.Rectangle) {
	height := f.defaultfontheight
	f.rect = r
	f.rect.Max.Y -= (r.Max.Y - r.Min.Y) % height
	f.maxlines = (r.Max.Y - r.Min.Y) / height
}

func (f *frameimpl) Clear(freeall bool) {
	f.lk.Lock()
	defer f.lk.Unlock()
	f.box = make([]*frbox, 0, 25)
	if freeall {
		f.tickimage.Free()
		f.tickback.Free()
		f.tickimage = nil
		f.tickback = nil
	}
	f.ticked = false
}
