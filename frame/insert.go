package frame

import (
	"fmt"
	"image"
	"unicode/utf8"

	"9fans.net/go/draw"
)

var (
	DELTA   = 25
	TMPSIZE = 256
)

func (f *Frame) bxscan(r []rune, ppt *image.Point) (image.Point, *Frame) {
	var c rune

	frame := &Frame{
		Rect:       f.Rect,
		Display:    f.Display,
		Background: f.Background,
		Font:       f.Font,
		MaxTab:     f.MaxTab,
		nchars:     0,
		box:        []*frbox{},
	}

	copy(frame.Cols[:], f.Cols[:])
	nl := 0

	// TODO(rjk): There are no boxes allocated?
	// log.Println("boxes are allocated?", "nalloc", f.nalloc, "box len", len(frame.box))

	offs := 0
	for nb := 0; offs < len(r) && nl <= f.MaxLines; nb++ {
		switch c = r[offs]; c {
		case '\t':
			frame.box = append(frame.box, &frbox{
				Bc:     c,
				Wid:    10000,
				Minwid: byte(frame.Font.StringWidth(" ")),
				Nrune:  -1,
			})

			frame.nchars++
			offs++
		case '\n':
			frame.box = append(frame.box, &frbox{
				Bc:     c,
				Wid:    10000,
				Minwid: 0,
				Nrune:  -1,
			})

			frame.nchars++
			offs++
			nl++
		default:
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
			p := make([]byte, s)
			copy(p, tmp[:s])

			frame.box = append(frame.box, &frbox{
				Ptr:   p,
				Wid:   w,
				Nrune: nr,
			})
			frame.nchars += nr
		}
	}

	*ppt = f.cklinewrap0(*ppt, frame.box[0])
	return frame._draw(*ppt), frame
}

func (f *Frame) chop(pt image.Point, p, bn int) {
	if bn >= len(f.box) {
		f.Logboxes(" -- chop, invalid bn=%d --\n", bn)
		panic("chop bn too large")
	}
	for {
		b := f.box[bn]
		pt = f.cklinewrap(pt, b)
		if bn >= len(f.box) || pt.Y >= f.Rect.Max.Y {
			break
		}
		p += nrune(b)
		pt = f.advance(pt, b)
		bn++
	}
	f.nchars = p
	f.nlines = f.MaxLines
	if bn < len(f.box) { // BUG
		f.delbox(bn, len(f.box)-1)
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
// Insert will remove the selection or tick  if present but update selection offsets.
func (f *Frame) Insert(r []rune, p0 int) bool {
	// log.Printf("frame.Insert. Start: %s", string(r))
	// defer log.Println("frame.Insert end")
	//	f.Logboxes("at very start of insert")
	f.validateboxmodel("Frame.Insert Start p0=%d, «%s»", p0, string(r))
	defer f.validateboxmodel("Frame.Insert End p0=%d, «%s»", p0, string(r))

	if p0 > f.nchars || len(r) == 0 || f.Background == nil {
		return f.lastlinefull
	}

	var rect image.Rectangle
	var col, tcol *draw.Image

	pts := make([]points, 0, 5)

	n0 := f.findbox(0, 0, p0)
	if n0 > len(f.box) {
		f.Logboxes("-- boxes after findbox when findbox has failed to return a valid box index --")
		panic(fmt.Sprint("findbox is sads", "n0:", n0))
	}

	//	f.Logboxes("at end of findbox")

	cn0 := p0
	nn0 := n0
	pt0 := f.ptofcharnb(p0, n0)
	ppt0 := pt0
	opt0 := pt0
	pt1, nframe := f.bxscan(r, &ppt0)
	ppt1 := pt1

	if n0 < len(f.box) {
		pt0 = f.cklinewrap(pt0, f.box[n0])
		ppt1 = f.cklinewrap0(ppt1, f.box[n0])
	}
	f.Modified = true
	/*
	 * ppt0 and ppt1 are start and end of insertion as they will appear when
	 * insertion is complete. pt0 is current location of insertion position
	 * (p0); pt1 is terminal point (without line wrap) of insertion.
	 */

	// Remove the selection or tick.
	f.DrawSel(f.Ptofchar(f.P0), f.P0, f.P1, false)

	/*
	 * Find point where old and new x's line up
	 * Invariants:
	 *	pt0 is where the next box (b, n0) is now
	 *	pt1 is where it will be after the insertion
	 * If pt1 goes off the rectangle, we can toss everything from there on
	 */
	npts := 0
	for ; pt1.X != pt0.X && pt1.Y != f.Rect.Max.Y && n0 < len(f.box); npts++ {
		b := f.box[n0]
		pt0 = f.cklinewrap(pt0, b)
		pt1 = f.cklinewrap0(pt1, b)
		if pt1.Y > f.Rect.Max.Y {
			f.Logboxes("-- pt1 violated invariant at box --")
			panic(fmt.Sprint("frame.Insert pt1 too far", " pt1=", pt1, " box=", b))
		}

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

		pts = append(pts, points{pt0, pt1})
		if pt1.Y == f.Rect.Max.Y {
			break
		}
		pt0 = f.advance(pt0, b)
		pt1.X += f.newwid(pt1, b)
		cn0 += nrune(b)
		n0++
	}

	if pt1.Y > f.Rect.Max.Y {
		nframe.validateboxmodel("frame.Insert pt1 too far, nframe validation, %v", pt1)
		panic("frame.Insert pt1 too far")
	}
	if pt1.Y == f.Rect.Max.Y && n0 < len(f.box) {
		f.nchars -= f.strlen(n0)
		f.delbox(n0, len(f.box)-1)
	}
	if n0 == len(f.box) {
		div := f.Font.DefaultHeight()
		f.nlines = (pt1.Y - f.Rect.Min.Y) / div
		if pt1.X > f.Rect.Min.X {
			f.nlines++
		}
	} else if pt1.Y != pt0.Y {
		y := f.Rect.Max.Y
		q0 := pt0.Y + f.Font.DefaultHeight()
		q1 := pt1.Y + f.Font.DefaultHeight()
		f.nlines += (q1 - q0) / f.Font.DefaultHeight()
		if f.nlines > f.MaxLines {
			// log.Println("f.chop", ppt1, p0, nn0, len(f.box), f.nbox)
			f.chop(ppt1, p0, nn0)
		}
		if pt1.Y < y {
			// log.Println("suspect case in frame", "pt1",  pt1, "pt0", pt0, "f.Rect", f.Rect, "q1", q1,  "y", y)
			// log.Println(" f.Font.DefaultHeight()",  f.Font.DefaultHeight())
			rect = f.Rect
			rect.Min.Y = q1
			rect.Max.Y = y
			// TODO(rjk): This bitblit may be harmful. Investigate further.
			if q1 < y {
				// log.Println("first blit op on ", rect, "from", image.Pt(f.Rect.Min.X, q0))
				f.Background.Draw(rect, f.Background, nil, image.Pt(f.Rect.Min.X, q0))
			}
			rect.Min = pt1
			rect.Max.X = pt1.X + (f.Rect.Max.X - pt0.X)
			rect.Max.Y = q1
			// log.Println("second blit op on ", rect, "from", pt0)
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
	// log.Println("npts", npts, "y", y)
	for n0 = n0 - 1; npts >= 0; n0-- {
		// log.Println("looping over  boxes..", n0)
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

				if f.P0 <= cn0 && cn0 < f.P1 { /* b+1 is inside selection */
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

				if f.P0 <= cn0 && cn0 < f.P1 {
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
			if f.P0 <= cn0 && cn0 < f.P1 {
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

	if f.P0 < p0 && p0 <= f.P1 {
		col = f.Cols[ColHigh]
		tcol = f.Cols[ColHText]
	} else {
		col = f.Cols[ColBack]
		tcol = f.Cols[ColText]
	}

	f.SelectPaint(ppt0, ppt1, col)
	nframe.drawtext(ppt0, tcol, col)

	// Actually add boxes.
	f.addbox(nn0, len(nframe.box))
	copy(f.box[nn0:], nframe.box)

	// f.Logboxes("after adding")

	if nn0 > 0 && f.box[nn0-1].Nrune >= 0 && ppt0.X-f.box[nn0-1].Wid >= f.Rect.Min.X {
		nn0--
		ppt0.X -= f.box[nn0].Wid
	}

	n0 += len(nframe.box)
	if n0 < len(f.box)-1 {
		n0++
	}
	f.clean(ppt0, nn0, n0+1)
	//	f.Logboxes("after clean")
	f.nchars += nframe.nchars
	if f.P0 >= p0 {
		f.P0 += nframe.nchars
	}
	if f.P0 >= f.nchars {
		f.P0 = f.nchars
	}
	if f.P1 >= p0 {
		f.P1 += nframe.nchars
	}
	if f.P1 >= f.nchars {
		f.P1 += f.nchars
	}

	return f.lastlinefull
}
