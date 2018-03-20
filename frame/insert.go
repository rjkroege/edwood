package frame

import (
	"image"
	"log"
	"unicode/utf8"

	"9fans.net/go/draw"
)

var (
	DELTA   = 25
	TMPSIZE = 256
	frame   Frame
)

func (f *Frame) bxscan(r []rune, ppt *image.Point) image.Point {
	//	log.Println("bxscan starting")

	var c rune

	frame.Rect = f.Rect
	frame.Background = f.Background
	frame.Font = f.Font
	frame.maxtab = f.maxtab
	frame.nbox = 0
	frame.nalloc = 0
	frame.nchars = 0
	frame.box = []*frbox{}

	copy(frame.Cols[:], f.Cols[:])
	delta := DELTA
	nl := 0

	// TODO(rjk): There are no boxes allocated?
	// log.Println("boxes are allocated?", "nalloc", f.nalloc, "box len", len(frame.box))

	offs := 0
	for nb := 0; offs < len(r) && nl <= f.maxlines; nb++ {
		if nb >= len(frame.box) {
			// We have no boxes on start. So add on demand.
			// TODO(rjk): consider removing delta, DELTA, nalloc if possible
			// This is not idiomatic.
			frame.growbox(delta)
			if delta < 10000 {
				delta *= 2
			}
		}
		b := frame.box[nb]
		if b == nil {
			b = new(frbox)
			frame.box[nb] = b
		}
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
				rw := utf8.EncodeRune(tmp[s:], c)
				if s+rw >= TMPSIZE {
					break
				}
				w += frame.Font.RunesWidth(r[offs : offs+1])
				offs++
				s += rw
				nr++
			}
			// not idiomatic.
			//			tmp[s] = 0
			//			s++
			p := make([]byte, s)

			log.Println(nb, len(frame.box), frame.box[0])

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
	return frame._draw(*ppt)
}

func (f *Frame) chop(pt image.Point, p, bn int) {
	for b := f.box[bn]; ; bn++ {
		if bn >= f.nbox {
			panic("endofframe")
		}
		f.cklinewrap(&pt, b)
		if pt.Y >= f.Rect.Max.Y {
			break
		}
		p += nrune(b)
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

// Insert inserts r into Frame f starting at index p0.
// If a NUL (0) character is inserted, chaos will ensue. Tabs
// and newlines are handled by the library, but all other characters,
// including control characters, are just displayed. For example,
// backspaces are printed; to erase a character, use Delete.
//
// Insert manages the tick and selection.
func (f *Frame) Insert(r []rune, p0 int) {
	log.Printf("\n\n-----\nframe.Insert: %s", string(r))
	//	f.Logboxes("at very start of insert")

	if p0 > f.nchars || len(r) == 0 || f.Background == nil {
		return
	}

	//	log.Println("frame.Insert, doing some work")

	var rect image.Rectangle
	var col, tcol *draw.Image

	pts := make([]points, 0, 5)

	n0 := f.findbox(0, 0, p0)

	//	f.Logboxes("at end of findbox")

	cn0 := p0
	nn0 := n0
	pt0 := f.ptofcharnb(p0, n0)
	ppt0 := pt0
	opt0 := pt0
	pt1 := f.bxscan(r, &ppt0)
	ppt1 := pt1

	// I expect n0 to be 0. But... the array is empty.
	//	log.Println("len of box", len(f.box), "n0", n0)
	//	f.Logboxes("f after bxscan")
	//	log.Println("----")
	//	frame.Logboxes("frame after bxscan")

	if n0 < f.nbox {
		f.cklinewrap(&pt0, f.box[n0])
		f.cklinewrap0(&ppt1, f.box[n0])
	}
	f.modified = true
	/*
	 * ppt0 and ppt1 are start and end of insertion as they will appear when
	 * insertion is complete. pt0 is current location of insertion position
	 * (p0); pt1 is terminal point (without line wrap) of insertion.
	 */
	// TODO(rjk): Insert should remove the selection. Host should use
	// Drawsel to put it back later?
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
	for ; pt1.X != pt0.X && pt1.Y != f.Rect.Max.Y && n0 < f.nbox; npts++ {
		b := f.box[n0]
		f.cklinewrap(&pt0, b)
		f.cklinewrap0(&pt1, b)

		if b.Nrune > 0 {
			n, fits := f.canfit(pt1, b)
			if !fits {
				panic("frame.canfit == 0")
			}
			if n != b.Nrune {
				f.splitbox(n0, n)
				b = f.box[n0]
			}
		}
		//		if npts == nalloc {
		//			pts = append(pts, make([]points, npts+DELTA)...)
		//			nalloc += DELTA
		//			b = f.box[n0]
		//		}

		pts = append(pts, points{pt0, pt1})
		if pt1.Y == f.Rect.Max.Y {
			break
		}
		f.advance(&pt0, b)
		pt1.X += f.newwid(pt1, b)
		cn0 += nrune(b)
		n0++
	}

	if pt1.Y > f.Rect.Max.Y {
		panic("frame.Insert pt1 too far")
	}
	if pt1.Y == f.Rect.Max.Y && n0 < f.nbox {
		f.nchars -= f.strlen(n0)
		f.delbox(n0, f.nbox-1)
	}
	if n0 == f.nbox {
		div := f.Font.DefaultHeight()
		if pt1.X > f.Rect.Min.X {
			div++
		}
		f.nlines = (pt1.Y - f.Rect.Min.Y) / div
	} else if pt1.Y != pt0.Y {
		y := f.Rect.Max.Y
		q0 := pt0.Y + f.Font.DefaultHeight()
		q1 := pt1.Y + f.Font.DefaultHeight()
		f.nlines += (q1 - q0) / f.Font.DefaultHeight()
		if f.nlines > f.maxlines {
			f.chop(ppt1, p0, nn0)
		}
		if pt1.Y < y {
			rect = f.Rect
			rect.Min.Y = q1
			rect.Max.Y = y
			if q1 < y {
				f.Background.Draw(rect, f.Background, nil, image.Pt(f.Rect.Min.X, q0))
			}
			rect.Min = pt1
			rect.Max.X = pt1.X + (f.Rect.Max.X - pt0.X)
			rect.Max.Y += q1
			f.Background.Draw(rect, f.Background, nil, pt0)
		}
	}

	/*
	 * Move the old stuff down to make room.  The loop will move the stuff
	 * between the insertion and the point where the x's lined up.
	 * The draw()s above moved everything down after the point they lined up.
	 */
	y := 0
	if pt1.Y == f.Rect.Max.Y {
		y = pt1.Y
	}
	npts--
	log.Println("npts", npts, "y", y)
	for n0 = n0 - 1; npts >= 0; n0-- {
		b := f.box[n0]
		pt := pts[npts].pt1

		if b.Nrune > 0 {
			rect.Min = pt
			rect.Max = rect.Min
			rect.Max.X += b.Wid
			rect.Max.Y += f.Font.DefaultHeight()

			f.Background.Draw(rect, f.Background, nil, pts[npts].pt0)
			/* clear bit hanging off right */
			if npts == 0 && pt.Y > pt0.Y {
				rect.Min = opt0
				rect.Max = opt0
				rect.Max.X = f.Rect.Max.X
				rect.Max.Y += f.Font.DefaultHeight()

				if f.p0 <= cn0 && cn0 < f.p1 { /* b+1 is inside selection */
					col = f.Cols[ColHigh]
				} else {
					col = f.Cols[ColBack]
				}
				f.Background.Draw(rect, col, nil, rect.Min)
			} else if pt.Y < y {
				rect.Min = pt
				rect.Max = pt
				rect.Min.X += b.Wid
				rect.Max.X = f.Rect.Max.X
				rect.Max.Y += f.Font.DefaultHeight()

				if f.p0 <= cn0 && cn0 < f.p1 {
					col = f.Cols[ColHigh]
				} else {
					col = f.Cols[ColBack]
				}
				f.Background.Draw(rect, col, nil, rect.Min)
			}
			y = pt.Y
			cn0 -= b.Nrune
		} else {
			rect.Min = pt
			rect.Max = pt
			rect.Max.X += b.Wid
			rect.Max.Y += f.Font.DefaultHeight()
			if rect.Max.X >= f.Rect.Max.X {
				rect.Max.X = f.Rect.Max.X
			}
			cn0--
			if f.p0 <= cn0 && cn0 < f.p1 {
				col = f.Cols[ColHigh]
				tcol = f.Cols[ColHText]
			} else {
				col = f.Cols[ColBack]
				tcol = f.Cols[ColText]
			}
			f.Background.Draw(rect, col, nil, rect.Min)
			y = 0
			if pt.X == f.Rect.Min.X {
				y = pt.Y
			}
		}
		npts--
	}

	if f.p0 < p0 && p0 <= f.p1 {
		col = f.Cols[ColHigh]
		tcol = f.Cols[ColHText]
	} else {
		col = f.Cols[ColBack]
		tcol = f.Cols[ColText]
	}

	f.SelectPaint(ppt0, ppt1, col)
	frame.drawtext(ppt0, tcol, col)

	// Actually add boxes.
	f.addbox(nn0, frame.nbox)
	for n := 0; n < frame.nbox; n++ {
		f.box[nn0+n] = frame.box[n]
	}

	//	f.Logboxes("after adding")

	if nn0 > 0 && f.box[nn0-1].Nrune >= 0 && ppt0.X-f.box[nn0-1].Wid >= f.Rect.Min.X {
		nn0--
		ppt0.X -= f.box[nn0].Wid
	}

	n0 += frame.nbox
	if n0 < f.nbox-1 {
		n0++
	}
	f.clean(ppt0, nn0, n0+1)
	//	f.Logboxes("after clean")
	f.nchars += frame.nchars
	if f.p0 >= p0 {
		f.p0 += frame.nchars
	}
	if f.p0 >= f.nchars {
		f.p0 = f.nchars
	}
	if f.p1 >= p0 {
		f.p1 += frame.nchars
	}
	if f.p1 >= f.nchars {
		f.p1 += f.nchars
	}
	if f.p0 == f.p1 {
		f.Tick(f.Ptofchar(f.p0), true)
	}

	//	log.Printf("first box %#v, %s\n",  *f.box[0], string(f.box[0].Ptr))

}
