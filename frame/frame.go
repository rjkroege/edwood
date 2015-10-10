package frame

import (
	"9fans.net/go/draw"
	"image"
)

const (
	BACK = iota
	HIGH
	BORD
	TEXT
	HTEXT
	NCOL

	FRTICKW = 3
)

type Frbox struct {
	Wid    int
	Nrune  int
	Ptr    []byte
	Bc     rune
	Minwid byte
}

type Frame struct {
	Font         *draw.Font
	Display      *draw.Display
	B            *draw.Image
	Cols         [NCOL]*draw.Image
	R            image.Rectangle
	Entire       image.Rectangle
	Scroll       func(*Frame, int)
	box          []*Frbox
	p0, p1       uint64
	nbox, nalloc int
	maxtab       int
	nchars       int
	nlines       int
	maxlines     int
	lastlinefull int
	modified     bool
	tick         *draw.Image
	tickback     *draw.Image
	ticked       bool
	noredraw     bool
	tickscale    int
}

func NewFrame(r image.Rectangle, ft *draw.Font, b *draw.Image, cols [NCOL]*draw.Image) *Frame {
	f := new(Frame)
	f.Font = ft
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
	if f.tick == nil && f.Cols[BACK] != nil {
		f.InitTick()
	}
	return f
}

func (f *Frame) InitTick() {
	var err error
	if f.Cols[BACK] == nil || f.Display == nil {
		return
	}

	f.tickscale = f.Display.ScaleSize(1)
	b := f.Display.ScreenImage
	ft := f.Font

	if f.tick != nil {
		f.tick.Free()
	}

	f.tick, err = f.Display.AllocImage(image.Rect(0, 0, f.tickscale*FRTICKW, ft.Height), b.Pix, false, draw.White)
	if err != nil {
		return
	}

	f.tickback, err = f.Display.AllocImage(f.tick.R, b.Pix, false, draw.White)
	if err != nil {
		f.tick.Free()
		f.tick = nil
		return
	}

	// background colour
	f.tick.Draw(f.tick.R, f.Cols[BACK], nil, image.Pt(0, 0))
	// vertical line
	f.tick.Draw(image.Rect(f.tickscale*(FRTICKW/2), 0, f.tickscale*(FRTICKW/2+1), ft.Height), f.Display.Black, nil, image.Pt(0, 0))
	// box on each end
	f.tick.Draw(image.Rect(0, 0, f.tickscale*FRTICKW, f.tickscale*FRTICKW), f.Cols[TEXT], nil, image.Pt(0, 0))
	f.tick.Draw(image.Rect(0, ft.Height-f.tickscale*FRTICKW, f.tickscale*FRTICKW, ft.Height), f.Cols[TEXT], nil, image.Pt(0, 0))
}

func (f *Frame) SetRects(r image.Rectangle, b *draw.Image) {
	f.B = b
	f.Entire = r
	f.R = r
	f.R.Max.Y -= (r.Max.Y - r.Min.Y) % f.Font.Height
	f.maxlines = (r.Max.Y - r.Min.Y) / f.Font.Height
}

func (f *Frame) Clear(freeall bool) {
	if f.nbox != 0 {
		f.delbox(0, f.nbox-1)
	}
	if f.box != nil {
		f.box = nil
	}
	if freeall {
		f.tick.Free()
		f.tickback.Free()
		f.tick = nil
		f.tickback = nil
	}
	f.box = nil
	f.ticked = false
}
