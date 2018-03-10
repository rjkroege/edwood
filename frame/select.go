package frame

import (
	"9fans.net/go/draw"
	"image"

"fmt"
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

// called when mouse 1 is down
func (f *Frame) Select(mc *draw.Mousectl) {
	mp := mc.Mouse.Point
	b := mc.Mouse.Buttons

	f.Modified = false
	f.DrawSel(f.Ptofchar(f.P0), f.P0, f.P1, false)
	p0 := f.Charofpt(mp)
	p1 := p0

	f.P0 = p0
	f.P1 = p1

	pt0 := f.Ptofchar(p0)
	pt1 := f.Ptofchar(p1)

	f.DrawSel(pt0, p0, p1, true)
	reg := 0

	var q int
	for {
		scrled := false
		if f.Scroll != nil {
			if mp.Y < f.Rect.Min.Y {
				f.Scroll(f, -(f.Rect.Min.Y-mp.Y)/f.Font.DefaultHeight()-1)
				p0 = f.P1
				p1 = f.P0
				scrled = true
			} else if mp.Y > f.Rect.Max.Y {
				f.Scroll(f, (mp.Y-f.Rect.Max.Y)/f.Font.DefaultHeight()+1)
				p0 = f.P1
				p1 = f.P0
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
fmt.Println("q = ", q)
		if p1 != q {
			if reg != region(q, p0) {
				if reg > 0 {
					f.DrawSel(pt0, p0, p1, false)
fmt.Printf("Clearing selection reg > 0 %v %v %v\n", pt0, p0, p1)
				} else if reg < 0 {
					f.DrawSel(pt1, p1, p0, false)
fmt.Printf("Clearing selection reg < 0 %v %v %v\n", pt1, p1, p0)
				}
				p1 = p0
				pt1 = pt0
				reg = region(q, p0)
				if reg == 0 {
					f.DrawSel(pt0, p0, p1, true)
fmt.Printf("Drawing selection reg = 0 %v %v %v\n", pt0, p0, p1)
				}
			}
			qt := f.Ptofchar(q)
fmt.Println("qt = ", qt)
			if reg > 0 {
				if q > p1 {
					f.DrawSel(pt1, p1, q, true)
fmt.Printf("Drawing selection q > p1 %v %v %v\n", pt1, p1, q)
				} else if q < p1 {
					f.DrawSel(qt, q, p1, false)
fmt.Printf("Clearing selection q < p1 %v %v %v\n", qt, q, p1)
				}
			} else if reg < 0 {
				if q > p1 {
					f.DrawSel(pt1, p1, q, false)
fmt.Printf("Clearing selection q > p1 %v %v %v\n", pt1, p1, q)
				} else {
					f.DrawSel(qt, q, p1, true)
fmt.Printf("Drawing selection q < p1 %v %v %v\n", qt, q, p1)
				}
			}
			p1 = q
			pt1 = qt
		}
		f.Modified = false
		if p0 < p1 {
			f.P0 = p0
			f.P1 = p1
		} else {
			f.P0 = p1
			f.P1 = p0
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
		if mc.Mouse.Buttons != b {
			break
		}
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
