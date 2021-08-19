package frame

import (
	"image"

	"github.com/rjkroege/edwood/draw"
)

// InitTick sets up the TickImage (e.g. cursor)
// TODO(rjk): doesn't appear to need to be exposed publically.
func (f *frameimpl) InitTick() {
	if f.cols[ColBack] == nil || f.display == nil {
		return
	}

	f.tickscale = f.display.ScaleSize(1)
	b := f.display.ScreenImage()
	ft := f.font

	if f.tickimage != nil {
		f.tickimage.Free()
	}

	height := ft.Height()

	var err error
	f.tickimage, err = f.display.AllocImage(image.Rect(0, 0, f.tickscale*frtickw, height), b.Pix(), false, draw.Transparent)
	if err != nil {
		return
	}

	f.tickback, err = f.display.AllocImage(f.tickimage.R(), b.Pix(), false, draw.White)
	if err != nil {
		f.tickimage.Free()
		f.tickimage = nil
		return
	}
	f.tickback.Draw(f.tickback.R(), f.cols[ColBack], nil, image.Point{})

	f.tickimage.Draw(f.tickimage.R(), f.display.Transparent(), nil, image.Pt(0, 0))
	// vertical line
	f.tickimage.Draw(image.Rect(f.tickscale*(frtickw/2), 0, f.tickscale*(frtickw/2+1), height), f.display.Opaque(), nil, image.Pt(0, 0))
	// box on each end
	f.tickimage.Draw(image.Rect(0, 0, f.tickscale*frtickw, f.tickscale*frtickw), f.display.Opaque(), nil, image.Pt(0, 0))
	f.tickimage.Draw(image.Rect(0, height-f.tickscale*frtickw, f.tickscale*frtickw, height), f.display.Opaque(), nil, image.Pt(0, 0))
}
