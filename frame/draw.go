package frame

import (
	"9fans.net/go/draw"
	"image"
)

func (f *Frame) DrawText(pt image.Point, text *draw.Image, back *draw.Image) {

}

func (f *Frame) DrawSel(pt image.Point, p0, p1 uint64, issel bool) {

}

func (f *Frame) drawsel0(pt image.Point, p0, p1 uint64, back *draw.Image, text *draw.Image) image.Point {
	return pt
}

func (f *Frame) Redraw() {

}

func (f *Frame) _tick(pt image.Point, ticked bool) {

}

func (f *Frame) Tick(pt image.Point, ticked bool) {

}

func (f *Frame) _draw(pt image.Point) image.Point {
	return pt
}

func (f *Frame) strlen(nb int) int {
	return 0
}
