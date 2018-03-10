package frame

import (
	"image"
	//	"log"

	"9fans.net/go/draw"
)

func (f *Frame) DrawText(pt image.Point, text *draw.Image, back *draw.Image) {
//	log.Println("DrawText at", pt, "NoRedraw", f.NoRedraw, text)
	for nb := 0; nb < f.nbox; nb++ {
		b := f.box[nb]
		pt = f.cklinewrap(pt, b)
//		log.Printf("box [%d] %#v pt %v NoRedraw %v nrune %d\n",  nb, string(b.Ptr), pt, f.NoRedraw, b.Nrune)

		if !f.NoRedraw && b.Nrune >= 0 {
			f.Background.Bytes(pt, text, image.ZP, f.Font.Impl(), b.Ptr)
		}
		pt.X += b.Wid
	}
}

func (f *Frame) DrawSel(pt image.Point, p0, p1 int, issel bool) {
	//	log.Println("DrawSel")
	var back, text *draw.Image

	if f.Ticked {
		f.Tick(f.Ptofchar(f.P0), false)
	}

	if p0 == p1 {
		f.Tick(pt, issel)
		return
	}

	if issel {
		back = f.Cols[ColHigh]
		text = f.Cols[ColHText]
	} else {
		back = f.Cols[ColBack]
		text = f.Cols[ColText]
	}

	f.DrawSel0(pt, p0, p1, back, text)
}

func (f *Frame) DrawSel0(pt image.Point, p0, p1 int, back *draw.Image, text *draw.Image) image.Point {
	//	log.Println("drawsel0")
	p := 0
	nb := 0
	nr := 0
	var b *frbox
	trim := false
	x := 0
	var w int

	for nb = 0; nb < f.nbox && p < p1; nb++ {
		b = f.box[nb]
		p += nr

		nr = b.Nrune
		if nr < 0 {
			nr = 1
		}
		if p+nr <= p0 {
			continue
		}
		if p >= p0 {
			qt := pt
			pt = f.cklinewrap(pt, b)
			// fill in the end of a wrapped line
			if pt.Y > qt.Y {
				f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
			}
		}
		runes := []rune(string(b.Ptr))
		if p < p0 {
			runes = runes[p0 - p:]
			nr -= p0 - p
			p = p0
		}
		trim = false
		if p+nr > p1 {
			// end of region: trim box
			nr -= (p + nr) - p1
			trim = true
		}

		if b.Nrune < 0 || nr == b.Nrune {
			w = b.Wid
		} else {
			w = f.Font.RunesWidth(runes[:nr])
		}
		x = pt.X + w
		if x > f.Rect.Max.X {
			x = f.Rect.Max.X
		}
		f.Background.Draw(image.Rect(pt.X, pt.Y, x, pt.Y+f.Font.DefaultHeight()),  back, nil, pt)
		if b.Nrune >= 0 {
			f.Background.Runes(pt, text, image.ZP, f.Font.Impl(), runes[:nr])
		}
		pt.X += w
	}

	if p1 > p0 && nb > 0 && nb < f.nbox && f.box[nb].Nrune > 0 && !trim {
		qt := pt
		pt = f.cklinewrap(pt, b)
		if pt.Y > qt.Y {
			f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
		}
	}
	return pt
}

func (f *Frame) Redraw() {
	//	log.Println("Redraw")
	ticked := false
	var pt image.Point

	if f.P0 == f.P1 {
		ticked = f.Ticked
		if ticked {
			f.Tick(f.Ptofchar(f.P0), false)
		}
		f.DrawSel0(f.Ptofchar(0), 0, f.NChars, f.Cols[ColBack], f.Cols[ColText])
		if ticked {
			f.Tick(f.Ptofchar(f.P0), true)
		}
	}

	pt = f.Ptofchar(0)
	pt = f.DrawSel0(pt, 0, f.P0, f.Cols[ColBack], f.Cols[ColText])
	pt = f.DrawSel0(pt, f.P0, f.P1, f.Cols[ColHigh], f.Cols[ColHText])
	pt = f.DrawSel0(pt, f.P1, f.NChars, f.Cols[ColBack], f.Cols[ColText])

}

func (f *Frame) _tick(pt image.Point, ticked bool) {
	//	log.Println("_tick")
	if f.Ticked == ticked || f.TickImage == nil || !pt.In(f.Rect) {
		return
	}

	pt.X -= f.TickScale
	r := image.Rect(pt.X, pt.Y, pt.X+frtickw*f.TickScale, pt.Y+f.Font.DefaultHeight())

	if r.Max.X > f.Rect.Max.X {
		r.Max.X = f.Rect.Max.X
	}
	if ticked {
		f.TickBack.Draw(f.TickBack.R, f.Background, nil, pt)
		f.Background.Draw(r, f.TickImage, nil, image.ZP)
	} else {
		f.Background.Draw(r, f.TickBack, nil, image.ZP)
	}
	f.Ticked = ticked
}

func (f *Frame) Tick(pt image.Point, ticked bool) {
//	log.Println("Tick")
	if f.TickScale != f.Display.ScaleSize(1) {
		if f.Ticked {
			f._tick(pt, false)
		}
		f.InitTick()
	}
	f._tick(pt, ticked)
}

func (f *Frame) _draw(pt image.Point) image.Point {
	//	log.Println("_draw")
	for nb := 0; nb < f.nbox; nb++ {
		b := f.box[nb]
		pt = f.cklinewrap0(pt, b)
		if pt.Y == f.Rect.Max.Y {
			f.NChars -= f.strlen(nb)
			f.delbox(nb, f.nbox-1)
			break
		}

		if b.Nrune > 0 {
			n, fits := f.canfit(pt, b)
			if !fits {
				break
			}
			if n != b.Nrune {
				f.splitbox(nb, n)
				b = f.box[nb]
			}
			pt.X += b.Wid
		} else {
			if b.Bc == '\n' {
				pt.X = f.Rect.Min.X
				pt.Y += f.Font.DefaultHeight()
			} else {
				pt.X += f.newwid(pt, b)
			}
		}
	}
	return pt
}

func (f *Frame) strlen(nb int) int {
	//	log.Println("strlen")
	var n int
	for n = 0; nb < f.nbox; nb++ {
		n += nrune(f.box[nb])
	}
	return n
}
