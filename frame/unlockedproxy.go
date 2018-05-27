package frame

import (
	"image"
)

type  SelectScrollUpdater interface {
	GetFrameFillStatus() FrameFillStatus
	Charofpt(pt image.Point) int
	DefaultFontHeight() int
	Delete(int,  int) int 
	Insert( []rune,  int) bool
	IsLastLineFull() bool
}

type selectscrollupdaterimpl  frameimpl

func (up *selectscrollupdaterimpl) GetFrameFillStatus() FrameFillStatus {
	f := (*frameimpl)(up)
	return FrameFillStatus{
			Nchars:   f.nchars,
			Nlines:   f.nlines,
			Maxlines: f.maxlines,
		}
}

func (up *selectscrollupdaterimpl) Charofpt(pt image.Point) int {
	f := (*frameimpl)(up)
	return 	f.charofptimpl(pt)
}

func (up *selectscrollupdaterimpl) DefaultFontHeight() int {
	f := (*frameimpl)(up)
	return f.defaultfontheight
}

func (up *selectscrollupdaterimpl) Delete(p0, p1 int) int {
	f := (*frameimpl)(up)
	return f.deleteimpl(p0, p1)
}


func (up *selectscrollupdaterimpl) Insert(r []rune, p0 int) bool {
	f := (*frameimpl)(up)
	return f.insertimpl(r, p0)
}

func (up *selectscrollupdaterimpl) IsLastLineFull() bool {
	f := (*frameimpl)(up)
	return f.lastlinefull
}


