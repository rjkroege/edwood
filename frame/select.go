package frame

import (
	"9fans.net/go/draw"
	"image"
)

func region(a, b int) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}

func (f *Frame) Select(mc draw.Mousectl) {

}

func (f *Frame) SelectPaint(p0, p1 image.Point, col *draw.Image) {

}
