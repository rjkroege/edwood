package frame

import (
	"image"
)

func (f *Frame) canfit(pt image.Point, b *Frbox) int {
	return 0
}

func (f *Frame) cklinewrap(p *image.Point, b *Frbox) {

}

func (f *Frame) cklinewrap0(p *image.Point, b *Frbox) {

}

func (f *Frame) advance(p *image.Point, b *Frbox) {

}

func (f *Frame) newwid(pt image.Point, b *Frbox) int {
	return 0
}

func (f *Frame) newwid0(pt image.Point, b *Frbox) int {
	return 0
}

func (f *Frame) clean(pt image.Point, n0, n1 int) {

}

func nbyte(f *Frbox) uint {
	if f.Nrune < 0 {
		return 1
	} else {
		return uint(f.Nrune)
	}
}

func nrune(f *Frbox) int {
	return len(f.Ptr)
}
