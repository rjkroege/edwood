package frame

import (
	"9fans.net/go/draw"
	"image"
)

// SetSelectionExtent sets the rune offsets of the selection maintained
// by the Frame. p0 and p1 must be values that could be returned by Charofpt.
// TODO(rjk): It is conceivable that we don't need this. It seems like an egregious
// abstraction violation that it exists.
func (f *Frame) SetSelectionExtent(p0, p1 int) {
	f.p0, f.p1 = p0, p1
}

// GetSelectionExtent returns the rune offsets of the selection maintained by
// the Frame.
func (f *Frame) GetSelectionExtent() (int, int) {
	return f.p0, f.p1
}

func region(a, b int) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}

// called when mouse 1 is down
func (f *Frame) Select(mc draw.Mousectl) {
	mp := mc.Mouse.Point
	b := mc.Mouse.Buttons

	f.modified = false
	f.DrawSel(f.Ptofchar(f.p0), f.p0, f.p1, false)
	p0 := f.Charofpt(mp)
	p1 := p0

	f.p0 = p0
	f.p1 = p1

	pt0 := f.Ptofchar(p0)
	pt1 := f.Ptofchar(p1)

	f.DrawSel(pt0, p0, p1, true)
	reg := 0

	var q int
	for mc.Mouse.Buttons == b {
		scrled := false
		if f.Scroll != nil {
			if mp.Y < f.Rect.Min.Y {
				f.Scroll(f, -(f.Rect.Min.Y-mp.Y)/f.Font.DefaultHeight()-1)
				p0 = f.p1
				p1 = f.p0
				scrled = true
			} else if mp.Y > f.Rect.Max.Y {
				f.Scroll(f, (mp.Y-f.Rect.Max.Y)/f.Font.DefaultHeight()+1)
				p0 = f.p1
				p1 = f.p0
				scrled = true
			}
			if scrled {
				if reg != region(p1, p0) {
					q = p0
					p0 = p1
					p1 = q
				}
				pt0 = f.Ptofchar(p0)
				pt1 = f.Ptofchar(p1)
				reg = region(p1, p0)
			}
		}
		q = f.Charofpt(mp)
		if p1 != q {
			if reg != region(q, p0) {
				if reg > 0 {
					f.DrawSel(pt0, p0, p1, false)
				} else if reg < 0 {
					f.DrawSel(pt1, p1, p0, false)
				}
				p1 = p0
				pt1 = pt0
				reg = region(q, p0)
				if reg == 0 {
					f.DrawSel(pt0, p0, p1, true)
				}
			}
			qt := f.Ptofchar(q)
			if reg > 0 {
				if q > p1 {
					f.DrawSel(pt1, p1, q, true)
				} else if q < p1 {
					f.DrawSel(qt, q, p1, false)
				}
			} else if reg < 0 {
				if q > p1 {
					f.DrawSel(pt1, p1, q, false)
				} else {
					f.DrawSel(qt, q, p1, true)
				}
			}
			p1 = q
			pt1 = qt
		}
		f.modified = false
		if p0 < p1 {
			f.p0 = p0
			f.p1 = p1
		} else {
			f.p0 = p1
			f.p1 = p0
		}

		if scrled {
			f.Scroll(f, 0)
		}
		if err := f.Display.Flush(); err != nil {
			panic(err)
		}
		if !scrled {
			mc.Read()
		}
		mp = mc.Mouse.Point
	}

}

func (f *Frame) SelectPaint(p0, p1 image.Point, col *draw.Image) {
	q0 := p0
	q1 := p1

	q0.Y += f.Font.DefaultHeight()
	q1.Y += f.Font.DefaultHeight()

	n := (p1.Y - p0.Y) / f.Font.DefaultHeight()
	if f.Background == nil {
		panic("Frame.SelectPaint B == nil")
	}
	if p0.Y == f.Rect.Max.Y {
		return
	}
	if n == 0 {
		f.Background.Draw(Rpt(p0, q1), col, nil, image.ZP)
	} else {
		if p0.X >= f.Rect.Max.X {
			p0.X = f.Rect.Max.X - 1
		}
		f.Background.Draw(image.Rect(p0.X, p0.Y, f.Rect.Max.X, q0.Y), col, nil, image.ZP)
		if n > 1 {
			f.Background.Draw(image.Rect(f.Rect.Min.X, q0.Y, f.Rect.Max.X, p1.Y), col, nil, image.ZP)
		}
		f.Background.Draw(image.Rect(f.Rect.Min.X, p1.Y, q1.X, q1.Y), col, nil, image.ZP)
	}
}
