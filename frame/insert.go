package frame

import (
	"9fans.net/go/draw"
	"image"
	"unicode/utf8"
)

var (
	DELTA   = 25
	TMPSIZE = 256
	frame   Frame
)

func (f *Frame) bxscan(r []rune, ppt *image.Point) image.Point {

	var c rune
	frame.R = f.R
	frame.B = f.B
	frame.Font = f.Font
	frame.maxtab = f.maxtab
	frame.nbox = 0
	frame.nchars = 0

	copy(frame.Cols[:], f.Cols[:])
	delta := DELTA
	nl := 0

	offs := 0
	for nb := 0; offs < len(r) && nl <= f.maxlines; nb++ {
		if nb == frame.nalloc {
			frame.growbox(uint64(delta))
			if delta < 10000 {
				delta *= 2
			}
		}
		b := frame.box[nb]
		c = r[offs]
		if c == '\t' || c == '\n' {
			b.Bc = c
			b.Wid = 5000
			if c == '\n' {
				b.Minwid = 0
				nl++
			} else {
				b.Minwid = byte(frame.Font.StringWidth(" "))
			}
			b.Nrune = -1
			frame.nchars++
			offs++
		} else {
			s := 0
			nr := 0
			w := 0
			tmp := make([]byte, TMPSIZE+3)

			for offs < len(r) {
				c := r[offs]
				if c == '\t' || c == '\n' {
					break
				}
				nr, rw := utf8.DecodeRune(tmp[s:])
				r[offs] = nr
				if s+rw >= TMPSIZE {
					break
				}
				w += frame.Font.RunesWidth(r[offs : offs+1])
				offs++
				s += w
				nr++
			}
			tmp[s] = 0
			s++
			p := f.AllocStr(uint(s))
			b = frame.box[nb]
			b.Ptr = p
			copy(b.Ptr, tmp[:s])
			b.Wid = w
			b.Nrune = nr
			frame.nchars += nr
		}
		frame.nbox++
	}
	f.cklinewrap0(ppt, frame.box[0])
	return f._draw(*ppt)
}

func (f *Frame) chop(pt image.Point, p uint64, bn int) {
	for b := f.box[bn]; ; bn++ {
		if bn >= f.nbox {
			panic("endofframe")
		}
		f.cklinewrap(&pt, b)
		if pt.Y >= f.R.Max.Y {
			break
		}
		p += uint64(nrune(b))
		f.advance(&pt, b)
	}
	f.nchars = int(p)
	f.nlines = f.maxlines
	if bn < f.nbox { // BUG
		f.delbox(bn, f.nbox-1)
	}
}

type points struct {
	pt0, pt1 image.Point
}
var nalloc = 0

func (f *Frame) Insert(r []rune, p0 uint64) {
	if p0 > uint64(f.nchars) || len(r) == 0 || f.B == nil {
		return
	}

	var rect image.Rectangle
	var col, tcol *draw.Image

	pts := make([]points, 0)

	n0 := f.findbox(0, 0, p0)
	cn0 := p0
	nn0 := n0
	pt0 := f.ptofcharnb(p0, n0)
	ppt0 := pt0
	opt0 := pt0
	pt1 := f.bxscan(r, &ppt0)
	ppt1 := pt1
	b := f.box[n0]

	if n0 < f.nbox {
		f.cklinewrap(&pt0, b)
		f.cklinewrap0(&ppt1, b)
	}
	f.modified = true
	/*
	 * ppt0 and ppt1 are start and end of insertion as they will appear when
	 * insertion is complete. pt0 is current location of insertion position
	 * (p0); pt1 is terminal point (without line wrap) of insertion.
	 */
	if f.p0 == f.p1 {
		f.Tick(f.Ptofchar(f.p0), false)
	}

	/*
	 * Find point where old and new x's line up
	 * Invariants:
	 *	pt0 is where the next box (b, n0) is now
	 *	pt1 is where it will be after the insertion
	 * If pt1 goes off the rectangle, we can toss everything from there on
	 */
	npts := 0 
	for ; pt1.X != pt0.X && pt1.Y != f.R.Max.Y && n0 < f.nbox; npts++ {
		b := f.box[n0]
		f.cklinewrap(&pt0, b)
		f.cklinewrap0(&pt1, b)

		if b.Nrune > 0 {
			n := f.canfit(pt1, b)
			if n == 0 {
				panic("frame.canfit == 0")
			}
			if n != b.Nrune {
				f.splitbox(uint64(n0), uint64(n))
				b = f.box[n0]
			}
		}
		if npts == nalloc {
			pts = append(pts, make([]points, npts+DELTA)...)
			nalloc += DELTA
			b = f.box[n0]
		}
		pts[npts].pt0 = pt0
		pts[npts].pt1 = pt1
		if pt1.Y == f.R.Max.Y {
			break
		}
		f.advance(&pt0, b)
		pt1.X += f.newwid(pt1, b)
		cn0 += uint64(nrune(b))
		n0++
	}

	if pt1.Y > f.R.Max.Y {
		panic("frame.Insert pt1 too far")
	}
	if pt1.Y == f.R.Max.Y && n0 < f.nbox {
		f.nchars -= f.strlen(n0)
		f.delbox(n0, f.nbox-1)
	}
	if n0 == f.nbox {
		div := f.Font.Height
		if pt1.X > f.R.Min.X {
			div++
		}
		f.nlines = (pt1.Y-f.R.Min.Y)/div
	} else if pt1.Y != pt0.Y {
		y := f.R.Max.Y
		q0 := pt0.Y + f.Font.Height
		q1 := pt1.Y + f.Font.Height
		f.nlines += (q1 - q0) / f.Font.Height
		if f.nlines > f.maxlines {
			f.chop(ppt1, p0, nn0)
		}
		if pt1.Y < y {
			rect = f.R
			rect.Min.Y = q1
			rect.Max.Y = y
			if q1 < y {
				f.B.Draw(rect, f.B, nil, image.Pt(f.R.Min.X, q0))
			}
			rect.Min = pt1
			rect.Max.X = pt1.X + (f.R.Max.X - pt0.X)
			rect.Max.Y += q1
			f.B.Draw(rect, f.B, nil, pt0)
		}
	}

	/*
	 * Move the old stuff down to make room.  The loop will move the stuff
	 * between the insertion and the point where the x's lined up.
	 * The draw()s above moved everything down after the point they lined up.
	 */
	y := 0
	if pt1.Y == f.R.Max.Y {
		y = pt1.Y
	}
	for n0 = n0 - 1; npts >= 0; n0-- {
		b := f.box[n0]
		pt := pts[npts].pt1

		if b.Nrune > 0 {
			rect.Min = pt
			rect.Max = rect.Min
			rect.Max.X += b.Wid
			rect.Max.Y += f.Font.Height

			f.B.Draw(rect, f.B, nil, pts[npts].pt0)
			/* clear bit hanging off right */
			if npts == 0 && pt.Y > pt0.Y {
				rect.Min = opt0
				rect.Max = opt0
				rect.Max.X = f.R.Max.X
				rect.Max.Y += f.Font.Height

				if f.p0 <= cn0 && cn0 < f.p1 { /* b+1 is inside selection */
					col = f.Cols[HIGH]
				} else {
					col = f.Cols[BACK]
				}
				f.B.Draw(rect, col, nil, rect.Min)
			} else if pt.Y < y {
				rect.Min = pt
				rect.Max = pt
				rect.Min.X += b.Wid
				rect.Max.X = f.R.Max.X
				rect.Max.Y += f.Font.Height

				if f.p0 <= cn0 && cn0 < f.p1 {
					col = f.Cols[HIGH]
				} else {
					col = f.Cols[BACK]
				}
				f.B.Draw(rect, col, nil, rect.Min)
			}
			y = pt.Y
			cn0 -= uint64(b.Nrune)
		} else {
			rect.Min = pt
			rect.Max = pt
			rect.Max.X += b.Wid
			rect.Max.Y += f.Font.Height
			if rect.Max.X >= f.R.Max.X {
				rect.Max.X = f.R.Max.X
			}
			cn0--
			if f.p0 <= cn0 && cn0 < f.p1 {
				col = f.Cols[HIGH]
				tcol = f.Cols[HTEXT]
			} else {
				col = f.Cols[BACK]
				tcol = f.Cols[TEXT]
			}
			f.B.Draw(rect, col, nil, rect.Min)
			y = 0
			if pt.X == f.R.Min.X {
				y = pt.Y
			}
		}
	}

	if f.p0 < p0 && p0 <= f.p1 {
		col = f.Cols[HIGH]
		tcol = f.Cols[HTEXT]
	} else {
		col = f.Cols[BACK]
		tcol = f.Cols[TEXT]
	}

	f.SelectPaint(ppt0, ppt1, col)
	f.DrawText(ppt0, tcol, col)
	f.addbox(uint64(nn0), uint64(frame.nbox))

	for n := 0; n < frame.nbox; n++ {
		f.box[nn0+n] = frame.box[n]
	}

	if nn0 > 0 && f.box[nn0-1].Nrune >= 0 && ppt0.X-f.box[nn0-1].Wid >= f.R.Min.X {
		nn0--
		ppt0.X -= f.box[nn0].Wid
	}

	n0 += frame.nbox
	if n0 < f.nbox-1 {
		n0++
	}
	f.clean(ppt0, nn0, n0)
	f.nchars += frame.nchars
	if f.p0 >= p0 {
		f.p0 += uint64(frame.nchars)
	}
	if f.p0 >= uint64(f.nchars) {
		f.p0 = uint64(f.nchars)
	}
	if f.p1 >= p0 {
		f.p1 += uint64(frame.nchars)
	}
	if f.p1 >= uint64(f.nchars) {
		f.p1 += uint64(f.nchars)
	}
	if f.p0 == f.p1 {
		f.Tick(f.Ptofchar(f.p0), true)
	}
}
