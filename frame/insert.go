package frame

import (
	"fmt"
	"image"
	"log"
	"unicode/utf8"
)

func (frame *frameimpl) addifnonempty(box *frbox, inby []byte) *frbox {
	if box == nil {
		return &frbox{
			Ptr: inby,
		}
	}

	if len(box.Ptr) > 0 {
		box.Wid = frame.font.BytesWidth(box.Ptr)
		frame.box = append(frame.box, box)
		return &frbox{
			Ptr: inby,
		}
	}
	return nil
}

// bxscan divides inby into single-line, nl and tab boxes. bxscan assumes that
// it has ownership of inby
func (f *frameimpl) bxscan(inby []byte, ppt *image.Point) (image.Point, *frameimpl) {
	frame := &frameimpl{
		rect:              f.rect,
		display:           f.display,
		background:        f.background,
		font:              f.font,
		defaultfontheight: f.defaultfontheight,
		maxtab:            f.maxtab,
		nchars:            0,
		box:               []*frbox{},
	}

	// TODO(rjk): This is (conceivably) pointless works?
	copy(frame.cols[:], f.cols[:])

	nl := 0

	// TODO(rjk): There are no boxes allocated?
	// log.Println("boxes are allocated?", "nalloc", f.nalloc, "box len", len(frame.box))

	var wipbox *frbox

	for i := 0; i < len(inby); frame.nchars++ {
		if nl > f.maxlines {
			break
		}

		switch inby[i] {
		case '\t':
			wipbox = frame.addifnonempty(wipbox, inby[i+1:i+1])

			frame.box = append(frame.box, &frbox{
				Bc:     '\t',
				Wid:    10000,
				Minwid: byte(frame.font.StringWidth(" ")),
				Nrune:  -1,
			})

			i++
		case '\n':
			wipbox = frame.addifnonempty(wipbox, inby[i+1:i+1])

			frame.box = append(frame.box, &frbox{
				Bc:     '\n',
				Wid:    10000,
				Minwid: 0,
				Nrune:  -1,
			})

			i++
			nl++
		default:
			_, n := utf8.DecodeRune(inby[i:])
			if wipbox == nil {
				wipbox = &frbox{
					Ptr: inby[i : i+n],
				}
			} else {
				wipbox.Ptr = wipbox.Ptr[:len(wipbox.Ptr)+n]
			}
			wipbox.Nrune++
			i += n
		}
	}
	frame.addifnonempty(wipbox, []byte{})

	*ppt = f.cklinewrap0(*ppt, frame.box[0])
	return frame._draw(*ppt), frame
}

func (f *frameimpl) chop(pt image.Point, p, bn int) {
	if bn >= len(f.box) {
		f.Logboxes(" -- chop, invalid bn=%d --\n", bn)
		panic("chop bn too large")
	}

	//  better version
	for i, bx := range f.box[bn:] {
		pt = f.cklinewrap(pt, bx)
		if pt.Y >= f.rect.Max.Y {
			f.nchars = p
			f.nlines = f.maxlines
			f.box = f.box[0 : bn+i]
			return
		}

		p += nrune(bx)
		pt = f.advance(pt, bx)
	}

	f.nchars = p
	f.nlines = f.maxlines
}

type points struct {
	pt0, pt1 image.Point
}

func (f *frameimpl) Insert(r []rune, p0 int) bool {
	f.lk.Lock()
	defer f.lk.Unlock()
	return f.insertimpl(r, p0)
}

func (f *frameimpl) InsertByte(b []byte, p0 int) bool {
	f.lk.Lock()
	defer f.lk.Unlock()
	return f.insertbyteimpl(b, p0)
}

func (f *frameimpl) insertimpl(r []rune, p0 int) bool {
	// TODO(rjk): Ick. But we'll get rid of this soon.
	inby := []byte(string(r))
	return f.insertbyteimpl(inby, p0)
}

func (f *frameimpl) insertbyteimpl(inby []byte, p0 int) bool {
	// log.Printf("frame.Insert. Start: %q", string(inby))
	// defer log.Println("frame.Insert end")
	//	f.Logboxes("at very start of insert")
	f.validateboxmodel("Frame.Insert Start p0=%d, «%s»", p0, string(inby))
	defer f.validateboxmodel("Frame.Insert End p0=%d, «%s»", p0, string(inby))
	f.validateinputs(inby, "Frame.Insert Start")

	if p0 > f.nchars || len(inby) == 0 || f.background == nil {
		return f.lastlinefull
	}

	col := f.cols[ColBack]
	tcol := f.cols[ColText]

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

	pt1, nframe := f.bxscan(inby, &ppt0)
	ppt1 := pt1

	if n0 < len(f.box) {
		pt0 = f.cklinewrap(pt0, f.box[n0])
		ppt1 = f.cklinewrap0(ppt1, f.box[n0])
	}
	f.modified = true
	/*
	 * ppt0 and ppt1 are start and end of insertion as they will appear when
	 * insertion is complete. pt0 is current location of insertion position
	 * (p0); pt1 is terminal point (without line wrap) of insertion.
	 */

	// Remove the selection or tick. This will redraw all selected text characters.
	// TODO(rjk): Do not remove the selection if it's unnecessary to do so.
	// Scrolling is the only time that we need to be particularly careful
	// with this. Small edits won't have a selection? Given however the cost
	// of selection drawing, we should do the right thing once?
	f.drawselimpl(f.ptofcharptb(f.sp0, f.rect.Min, 0), f.sp0, f.sp1, false)

	/*
	 * Find point where old and new x's line up
	 * Invariants:
	 *	pt0 is where the next box (b, n0) is now
	 *	pt1 is where it will be after the insertion
	 * If pt1 goes off the rectangle, we can toss everything from there on
	 */
	npts := 0
	for ; pt1.X != pt0.X && pt1.Y != f.rect.Max.Y && n0 < len(f.box); npts++ {
		b := f.box[n0]
		pt0 = f.cklinewrap(pt0, b)
		pt1 = f.cklinewrap0(pt1, b)
		if pt1.Y > f.rect.Max.Y {
			f.Logboxes("-- pt1 violated invariant at box --")
			panic(fmt.Sprint("frame.Insert pt1 too far", " pt1=", pt1, " box=", b))
		}

		if b.Nrune > 0 {
			n, fits := f.canfit(pt1, b)
			if !fits {
				f.Logboxes("-- frame.canfit false  box[%d]=%v %v, %v--", n0, b.String(), pt1, f.rect)
				panic("frame.canfit false")
			}
			if n != b.Nrune {
				f.splitbox(n0, n)
				b = f.box[n0]
			}
		}

		pts = append(pts, points{pt0, pt1})
		if pt1.Y == f.rect.Max.Y {
			break
		}
		pt0 = f.advance(pt0, b)
		pt1.X += f.newwid(pt1, b)
		cn0 += nrune(b)
		n0++
	}

	if pt1.Y > f.rect.Max.Y {
		nframe.validateboxmodel("frame.Insert pt1 too far, nframe validation, %v", pt1)
		panic("frame.Insert pt1 too far")
	}
	if pt1.Y == f.rect.Max.Y && n0 < len(f.box) {
		f.nchars -= f.strlen(n0)
		f.delbox(n0, len(f.box)-1)
	}
	var rect image.Rectangle
	if n0 == len(f.box) {
		div := f.defaultfontheight
		f.nlines = (pt1.Y - f.rect.Min.Y) / div
		if pt1.X > f.rect.Min.X {
			f.nlines++
		}
	} else if pt1.Y != pt0.Y {
		y := f.rect.Max.Y
		q0 := pt0.Y + f.defaultfontheight
		q1 := pt1.Y + f.defaultfontheight
		f.nlines += (q1 - q0) / f.defaultfontheight
		if f.nlines > f.maxlines {
			// log.Println("f.chop", ppt1, p0, nn0, len(f.box), f.nbox)
			f.chop(ppt1, p0, nn0)
		}
		if pt1.Y < y {
			// log.Println("suspect case in frame", "pt1",  pt1, "pt0", pt0, "f.Rect", f.Rect, "q1", q1,  "y", y)
			// log.Println(" f.Font.DefaultHeight()",  f.Font.DefaultHeight())
			rect = f.rect
			rect.Min.Y = q1
			rect.Max.Y = y
			// TODO(rjk): This bitblit may be harmful. Investigate further.
			if q1 < y {
				// log.Println("first blit op on ", rect, "from", image.Pt(f.Rect.Min.X, q0))
				f.background.Draw(rect, f.background, nil, image.Pt(f.rect.Min.X, q0))
			}
			rect.Min = pt1
			rect.Max.X = pt1.X + (f.rect.Max.X - pt0.X)
			rect.Max.Y = q1
			// log.Println("second blit op on ", rect, "from", pt0)
			f.background.Draw(rect, f.background, nil, pt0)
		}
	}

	/*
	 * Move the old stuff down to make room.  The loop will move the stuff
	 * between the insertion and the point where the x's lined up.
	 * The draw()s above moved everything down after the point they lined up.
	 */
	y := 0
	if pt1.Y == f.rect.Max.Y {
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
			rect.Max.Y += f.defaultfontheight

			f.background.Draw(rect, f.background, nil, pts[npts].pt0)
			// clear bit hanging off right
			if npts == 0 && pt.Y > pt0.Y {
				rect.Min = opt0
				rect.Max = opt0
				rect.Max.X = f.rect.Max.X
				rect.Max.Y += f.defaultfontheight

				f.background.Draw(rect, col, nil, rect.Min)
			} else if pt.Y < y {
				rect.Min = pt
				rect.Max = pt
				rect.Min.X += b.Wid
				rect.Max.X = f.rect.Max.X
				rect.Max.Y += f.defaultfontheight

				f.background.Draw(rect, col, nil, rect.Min)
			}
			y = pt.Y
			cn0 -= b.Nrune
		} else {
			// This box (b) is a tab or a newline.
			rect.Min = pt
			rect.Max = pt
			rect.Max.X += b.Wid
			rect.Max.Y += f.defaultfontheight
			if rect.Max.X >= f.rect.Max.X {
				rect.Max.X = f.rect.Max.X
			}
			cn0--
			f.background.Draw(rect, col, nil, rect.Min)
			y = 0
			if pt.X == f.rect.Min.X {
				y = pt.Y
			}
		}
		npts--
	}

	f.SelectPaint(ppt0, ppt1, col)
	nframe.drawtext(ppt0, tcol, col)

	// Actually add boxes.
	f.addbox(nn0, len(nframe.box))
	copy(f.box[nn0:], nframe.box)

	// f.Logboxes("after adding")

	if nn0 > 0 && f.box[nn0-1].Nrune >= 0 && ppt0.X-f.box[nn0-1].Wid >= f.rect.Min.X {
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
	if f.sp0 >= p0 {
		f.sp0 += nframe.nchars
	}
	if f.sp0 >= f.nchars {
		f.sp0 = f.nchars
	}
	if f.sp1 >= p0 {
		f.sp1 += nframe.nchars
	}
	if f.sp1 >= f.nchars {
		f.sp1 += f.nchars
	}

	return f.lastlinefull
}

// validateinputs ensures that the given rune string is valid for
// insertion.
func (f *frameimpl) validateinputs(inby []byte, format string, args ...interface{}) {
	if !*validate {
		return
	}

	for i, r := range inby {
		if r == 0x00 { // Nulls in input string are forbidden.
			log.Printf(format, args...)
			log.Printf("r[%d] null", i)
			panic("-- invalid input to Frame.Insert --")
		}
	}
}
