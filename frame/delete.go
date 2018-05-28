package frame

import (
	"image"
)

func (f *frameimpl) Delete(p0, p1 int) int {
	f.lk.Lock()
	defer f.lk.Unlock()
	return f.deleteimpl(p0, p1)
}

// TODO(rjk): This code will redraw too much.
func (f *frameimpl) deleteimpl(p0, p1 int) int {
	f.validateboxmodel("Frame.Delete Start p0=%d p1=%d", p0, p1)
	defer f.validateboxmodel("Frame.Delete Start p0=%d p1=%d", p0, p1)
	var r image.Rectangle

	if p1 > f.nchars {
		p1 = f.nchars - 1
	}
	if p0 >= f.nchars || p0 == p1 || f.background == nil {
		return 0
	}

	//log.Println("Delete is doing something")

	n0 := f.findbox(0, 0, p0)
	if n0 == len(f.box) {
		panic("off end in Frame.Delete")
	}

	n1 := f.findbox(n0, p0, p1)
	pt0 := f.ptofcharnb(p0, n0)
	pt1 := f.ptofcharptb(p1, f.rect.Min, 0)

	// Remove the selection or tick.
	f.drawselimpl(f.ptofcharptb(f.sp0, f.rect.Min, 0), f.sp0, f.sp1, false)

	nn0 := n0
	ppt0 := pt0

	// If the previous code was safe, this is harmless.
	// f.freebox(n0, n1-1)
	f.modified = true

	/*
	 * Invariants:
	 *  - pt0 points to beginning, pt1 points to end
	 *  - n0 is box containing beginning of stuff being deleted
	 *  - n1, b are box containing beginning of stuff to be kept after deletion
	 *  - cn1 is char position of n1
	 *  - f->p0 and f->p1 are not adjusted until after all deletion is done
	 */
	off := n1
	cn1 := p1

	for pt1.X != pt0.X && n1 < len(f.box) {
		b := f.box[n1]
		pt0 = f.cklinewrap0(pt0, b)
		pt1 = f.cklinewrap(pt1, b)
		n, fits := f.canfit(pt0, b)

		if !fits {
			panic("Frame.delete, canfit fits is false")
		}

		r.Min = pt0
		r.Max = pt0
		r.Max.Y += f.defaultfontheight

		if b.Nrune > 0 {
			w0 := b.Wid
			if n != b.Nrune {
				f.splitbox(n1, n)
				b = f.box[n1]
			}
			r.Max.X += int(b.Wid)
			f.background.Draw(r, f.background, nil, pt1)
			cn1 += b.Nrune

			r.Min.X = r.Max.X
			r.Max.X += int(w0 - b.Wid)
			if r.Max.X > f.rect.Max.X {
				r.Max.X = f.rect.Max.X
			}
			f.background.Draw(r, f.cols[ColBack], nil, r.Min)
		} else {
			r.Max.X += f.newwid(pt0, b)
			if r.Max.X > f.rect.Max.X {
				r.Max.X = f.rect.Max.X
			}
			col := f.cols[ColBack]
			if f.sp0 <= cn1 && cn1 < f.sp1 {
				col = f.cols[ColHigh]
			}
			f.background.Draw(r, col, nil, pt0)
			cn1++
		}
		pt1 = f.advance(pt1, b)
		pt0.X += f.newwid(pt0, b)
		f.box[n0] = f.box[n1]
		n0++
		n1++
		off++
	}

	if n1 == len(f.box) && pt0.X != pt1.X {
		f.SelectPaint(pt0, pt1, f.cols[ColBack])
	}
	if pt1.Y != pt0.Y {
		// What is going on here?
		pt2 := f.ptofcharptb(32767, pt1, n1)
		if pt2.Y > f.rect.Max.Y {
			panic("Frame.ptofchar in Frame.delete")
		}

		if n1 < len(f.box) {
			height := f.defaultfontheight
			q0 := pt0.Y + height
			q1 := pt1.Y + height
			q2 := pt2.Y + height

			if q2 > f.rect.Max.Y {
				q2 = f.rect.Max.Y
			}

			f.background.Draw(image.Rect(pt0.X, pt0.Y, pt0.X+(f.rect.Max.X-pt1.X), q0), f.background, nil, pt1)
			f.background.Draw(image.Rect(f.rect.Min.X, q0, f.rect.Max.X, q0+(q2-q1)), f.background, nil, image.Pt(f.rect.Min.X, q1))
			f.SelectPaint(image.Pt(pt2.X, pt2.Y-(pt1.Y-pt0.Y)), pt2, f.cols[ColBack])
		} else {
			f.SelectPaint(pt0, pt2, f.cols[ColBack])
		}
	}
	// We crash here.
	f.closebox(n0, n1-1)
	if nn0 > 0 && f.box[nn0-1].Nrune >= 0 && ppt0.X-int(f.box[nn0-1].Wid) >= int(f.rect.Min.X) {
		nn0--
		ppt0.X -= int(f.box[nn0].Wid)
	}

	if n0 < len(f.box)-1 {
		f.clean(ppt0, nn0, n0+1)
	} else {
		f.clean(ppt0, nn0, n0)
	}

	if f.sp1 > p1 {
		f.sp1 -= p1 - p0
	} else if f.sp1 > p0 {
		f.sp1 = p0
	}

	if f.sp0 > p1 {
		f.sp0 -= p1 - p0
	} else if f.sp0 > p0 {
		f.sp0 = p0
	}

	f.nchars -= int(p1 - p0)
	if f.sp0 == f.sp1 {
		f.Tick(f.ptofcharptb(f.sp0, f.rect.Min, 0), true)
	}
	pt0 = f.ptofcharptb(f.nchars, f.rect.Min, 0)
	n := f.nlines
	f.nlines = (pt0.Y - f.rect.Min.Y) / f.defaultfontheight
	if pt0.X > f.rect.Min.X {
		f.nlines++
	}
	return n - f.nlines
}
