package frame

import (
	"9fans.net/go/draw"
	"image"
)

func (f *Frame) DrawText(pt image.Point, text *draw.Image, back *draw.Image) {

	for nb := 0; nb < f.nbox; nb++ {
		b := f.box[nb]
		f.cklinewrap(&pt, b)
		if !f.noredraw && b.Nrune >= 0 {
			f.Background.String(pt, text, image.ZP, f.Font, string(b.Ptr))
		}
		pt.X += b.Wid
	}
}

func (f *Frame) DrawSel(pt image.Point, p0, p1 uint64, issel bool) {
	var back, text *draw.Image

	if f.ticked {
		f.Tick(f.Ptofchar(f.p0), false)
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

	f.drawsel0(pt, p0, p1, back, text)
}

func (f *Frame) drawsel0(pt image.Point, p0, p1 uint64, back *draw.Image, text *draw.Image) image.Point {
	p := 0
	bi := 0
	b := f.box[bi]
	trim := false
	i := 0
	x := 0
	var w int

	for nb := 0; nb < f.nbox && p < int(p1); nb++ {
		nr := b.Nrune
		if nr < 0 {
			nr = 1
		}
		if p+nr <= int(p0) {
			goto Continue
		}
		if p >= int(p0) {
			qt := pt
			f.cklinewrap(&pt, b)
			if pt.Y > qt.Y {
				f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
			}
		}
		i = 0
		if p < int(p0) {
			i += len(b.Ptr[:int(p0)-p])
			nr -= int(p0) - p
			p = int(p0)
		}
		trim = false
		if p+nr > int(p1) {
			nr -= (p + nr) - int(p1)
			trim = true
		}
		if b.Nrune < 0 || nr == b.Nrune {
			w = b.Wid
		} else {
			w = f.Font.StringWidth(string(b.Ptr[i : i+nr]))
		}
		x = pt.X + w
		if x > f.Rect.Max.X {
			x = f.Rect.Max.X
		}
		f.Background.Draw(image.Rect(pt.X, pt.Y, x, pt.Y+f.Font.Height), back, nil, pt)
		if b.Nrune >= 0 {
			f.Background.String(pt, text, image.ZP, f.Font, string(b.Ptr[i:i+nr]))
		}
		pt.X += w
	Continue:
		bi++
		b = f.box[bi]
		p += nr
	}

	if p1 > p0 && bi > 0 && bi < f.nbox && f.box[bi-1].Nrune > 0 && !trim {
		qt := pt
		f.cklinewrap(&pt, b)
		if pt.Y > qt.Y {
			f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
		}
	}
	return pt
}

func (f *Frame) Redraw() {
	ticked := false
	var pt image.Point

	if f.p0 == f.p1 {
		ticked = f.ticked
		if ticked {
			f.Tick(f.Ptofchar(f.p0), false)
		}
		f.drawsel0(f.Ptofchar(0), 0, uint64(f.nchars), f.Cols[ColBack], f.Cols[ColText])
		if ticked {
			f.Tick(f.Ptofchar(f.p0), true)
		}
	}

	pt = f.Ptofchar(0)
	pt = f.drawsel0(pt, 0, f.p0, f.Cols[ColBack], f.Cols[ColText])
	pt = f.drawsel0(pt, f.p0, f.p1, f.Cols[ColHigh], f.Cols[ColHText])
	pt = f.drawsel0(pt, f.p1, uint64(f.nchars), f.Cols[ColBack], f.Cols[ColText])

}

func (f *Frame) _tick(pt image.Point, ticked bool) {
	if f.ticked == ticked || f.tick == nil || !pt.In(f.Rect) {
		return
	}

	pt.X -= f.tickscale
	r := image.Rect(pt.X, pt.Y, pt.X+frtickw*f.tickscale, pt.Y+f.Font.Height)

	if r.Max.X > f.Rect.Max.X {
		r.Max.X = f.Rect.Max.X
	}
	if ticked {
		f.tickback.Draw(f.tickback.R, f.Background, nil, pt)
		f.Background.Draw(r, f.tick, nil, image.ZP)
	} else {
		f.Background.Draw(r, f.tickback, nil, image.ZP)
	}
	f.ticked = ticked
}

func (f *Frame) Tick(pt image.Point, ticked bool) {
	if f.tickscale != f.Display.ScaleSize(1) {
		if f.ticked {
			f._tick(pt, false)
		}
		f.InitTick()
	}
	f._tick(pt, ticked)
}

func (f *Frame) _draw(pt image.Point) image.Point {
	for nb := 0; nb < f.nbox; nb++ {
		b := f.box[nb]
		f.cklinewrap0(&pt, b)
		if pt.Y == f.Rect.Max.Y {
			f.nchars -= f.strlen(nb)
			f.delbox(nb, f.nbox-1)
			break
		}

		if b.Nrune > 0 {
			n, fits := f.canfit(pt, b)
			if !fits {
				break
			}
			if n != b.Nrune {
				f.splitbox(uint64(nb), uint64(n))
				b = f.box[nb]
			}
			pt.X += b.Wid
		} else {
			if b.Bc == '\n' {
				pt.X = f.Rect.Min.X
				pt.Y += f.Font.Height
			} else {
				pt.X += f.newwid(pt, b)
			}
		}
	}
	return pt
}

func (f *Frame) strlen(nb int) int {
	var n int
	for n = 0; nb < f.nbox; nb++ {
		n += nrune(f.box[nb])
	}
	return n
}
