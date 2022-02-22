package main

import (
	"image"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/frame"
)

// MockFrame is a mock implementation of a frame.Frame that does nothing.
type MockFrame struct{}

func (mf *MockFrame) GetFrameFillStatus() frame.FrameFillStatus {
	return frame.FrameFillStatus{
		Nchars:         0,
		Nlines:         0,
		Maxlines:       0,
		MaxPixelHeight: 0,
	}
}
func (mf *MockFrame) Charofpt(pt image.Point) int                  { return 0 }
func (mf *MockFrame) DefaultFontHeight() int                       { return 10 }
func (mf *MockFrame) Delete(int, int) int                          { return 0 }
func (mf *MockFrame) Insert([]rune, int) bool                      { return false }
func (mf *MockFrame) InsertByte([]byte, int) bool                  { return false }
func (mf *MockFrame) IsLastLineFull() bool                         { return false }
func (mf *MockFrame) Rect() image.Rectangle                        { return image.Rect(0, 0, 0, 0) }
func (mf *MockFrame) TextOccupiedHeight(r image.Rectangle) int     { return 0 }
func (mf *MockFrame) Maxtab(_ int)                                 {}
func (mf *MockFrame) GetMaxtab() int                               { return 0 }
func (mf *MockFrame) Init(image.Rectangle, ...frame.OptionClosure) {}
func (mf *MockFrame) Clear(bool)                                   {}
func (mf *MockFrame) Ptofchar(int) image.Point                     { return image.Point{0, 0} }
func (mf *MockFrame) Redraw(enclosing image.Rectangle)             {}
func (mf *MockFrame) GetSelectionExtent() (int, int)               { return 0, 0 }
func (mf *MockFrame) Select(*draw.Mousectl, *draw.Mouse, func(frame.SelectScrollUpdater, int)) (int, int) {
	return 0, 0
}
func (mf *MockFrame) SelectOpt(*draw.Mousectl, *draw.Mouse, func(frame.SelectScrollUpdater, int), draw.Image, draw.Image) (int, int) {
	return 0, 0
}
func (mf *MockFrame) DrawSel(image.Point, int, int, bool) {}
