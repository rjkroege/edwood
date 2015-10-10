package frame

import (
	"image"
	"unicode/utf8"
)

func (f *Frame) ptofcharptb(p uint64, pt image.Point, bn int) image.Point {

	var b *Frbox
	var w int
	var r rune
	
	for ; bn < f.nbox; bn++ {
		b = f.box[bn]
		f.cklinewrap(&pt, b)
		l := nrune(b)
		if p < uint64(l) {
			if b.Nrune > 0 {
				for s := 0; s < len(b.Ptr) && p > 0; s += w {
					p--
					r, w = utf8.DecodeRune(b.Ptr[s:])
					pt.X += f.Font.StringWidth(string(b.Ptr[s:s+1]))
					if (r == 0 || pt.X > f.R.Max.X) {
						panic("frptofchar")
					}
				}
			}
			break
		}
		p -= uint64(l)
		f.advance(&pt, b)
	}
	
	return pt
}

func (f *Frame) Ptofchar(p uint64) image.Point {
	return f.ptofcharptb(p, f.R.Min, 0)
}

func (f *Frame) ptofcharnb(p uint64, nb int) image.Point {
	pt := image.Point{}
	nbox := f.nbox
	pt = f.ptofcharptb(p, f.R.Min, 0)
	f.nbox = nbox
	return pt
}

func (f *Frame) grid(p image.Point) image.Point {
	p.Y -= f.R.Min.Y
	p.Y -= p.Y % f.Font.Height
	p.Y += f.R.Min.Y
	if p.X > f.R.Max.X {
		p.X = f.R.Max.X
	}
	return p
}

func (f *Frame) Charofpt(pt image.Point) uint64 {
	
	var w, bn int
	var b *Frbox
	var p uint64
	var r rune
	
	pt = f.grid(pt)
	qt := f.R.Min
	
	for bn = 0; bn < f.nbox && qt.Y < pt.Y; bn++ {
		b = f.box[bn]
		f.cklinewrap(&pt, b)
		if qt.Y >= pt.Y {
			break
		}
		f.advance(&qt, b)
		p += uint64(nrune(b))
	}
	
	for ; bn < f.nbox && qt.X <= pt.X; bn++ {
		f.cklinewrap(&qt, b)
		if qt.Y > pt.Y {
			break
		}
		if qt.X + b.Wid > pt.X {
			if b.Nrune < 0 {
				f.advance(&qt, b)
			} else {
				s := 0
				for ; s < len(b.Ptr); s += w {
					r, w = utf8.DecodeRune(b.Ptr[s:])
					if r == 0 {
						panic("end of string in frcharofpt")
					}
					qt.X += f.Font.StringWidth(string(b.Ptr[s:s+1]))
					if qt.X > pt.X {
						break
					}
					p++
				}
			}
		} else {
			p += uint64(nrune(b))
			f.advance(&qt, b)
		}
	}
	return p
}
