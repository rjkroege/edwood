package frame

import (
	"image"
)

func (f *Frame) ptofcharptb(p uint64, pt image.Point, bn int) image.Point {
	return pt
}

func (f *Frame) Ptofchar(p uint64) image.Point {
	return image.Point{}
}

func (f *Frame) ptofcharnb(p uint64, nb int) image.Point {
	return image.Point{}
}

func (f *Frame) grid(p uint64, nb int) image.Point {
	return image.Point{}
}

func (f *Frame) Charofpt(pt image.Point) uint64 {
	return 0
}
