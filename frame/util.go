package frame

import (
	"fmt"
	"image"
	"log"
	"unicode/utf8"
)

// canfit measures the b's string contents and determines if it fits
// in the region of the screen between pt and the right edge of the
// text-containing region. Returned values have several cases.
//
// If b has width, returns the index of the first rune known
// to not fit and true if more than 0 runes fit.
// If b has no width, use minwidth instead of width.
func (f *frameimpl) canfit(pt image.Point, b *frbox) (int, bool) {
	left := f.rect.Max.X - pt.X
	if b.Nrune < 0 {
		if int(b.Minwid) <= left {
			return 1, true
		}
		return 0, false
	}

	if left >= b.Wid {
		return b.Nrune, (b.Nrune != 0)
	}

	w := 0
	o := 0
	for nr := 0; nr < b.Nrune; nr++ {
		_, w = utf8.DecodeRune(b.Ptr[o:])
		left -= f.font.StringWidth(string(b.Ptr[o : o+w]))
		if left < 0 {
			return nr, nr != 0
		}
		o += w
	}
	return 0, false
}

// cklinewrap returns a new for where the given the box b should be
// placed. NB: this code is not going to do the right thing with a newline box.
func (f *frameimpl) cklinewrap(p image.Point, b *frbox) (ret image.Point) {
	ret = p
	if b.Nrune < 0 {
		if int(b.Minwid) > f.rect.Max.X-p.X {
			ret.X = f.rect.Min.X
			ret.Y = p.Y + f.defaultfontheight
		}
	} else {
		if b.Wid > f.rect.Max.X-p.X {
			ret.X = f.rect.Min.X
			ret.Y = p.Y + f.defaultfontheight
		}
	}
	if ret.Y > f.rect.Max.Y {
		ret.Y = f.rect.Max.Y
	}
	return ret
}

func (f *frameimpl) cklinewrap0(p image.Point, b *frbox) (ret image.Point) {
	ret = p
	if _, ok := f.canfit(p, b); !ok {
		ret.X = f.rect.Min.X
		ret.Y = p.Y + f.defaultfontheight
		if ret.Y > f.rect.Max.Y {
			ret.Y = f.rect.Max.Y
		}
	}
	return ret
}

func (f *frameimpl) advance(p image.Point, b *frbox) image.Point {
	if b.Nrune < 0 && b.Bc == '\n' {
		p.X = f.rect.Min.X
		p.Y += f.defaultfontheight
		if p.Y > f.rect.Max.Y {
			p.Y = f.rect.Max.Y
		}
	} else {
		p.X += b.Wid
	}
	return p
}

// newwid returns the width of a given box and mutates the
// appropriately.
// TODO(rjk): I'm perturbed. This is horrible.
func (f *frameimpl) newwid(pt image.Point, b *frbox) int {
	b.Wid = f.newwid0(pt, b)
	return b.Wid
}

// TODO(rjk): Expand the test cases involving \t characters.
// TODO(rjk): Wouldn't it be nicer if this was a method on frbox?
func (f *frameimpl) newwid0(pt image.Point, b *frbox) int {
	c := f.rect.Max.X
	x := pt.X
	if b.Nrune >= 0 || b.Bc != '\t' {
		return b.Wid
	}
	if x+int(b.Minwid) > c {
		// A tab character doesn't fit at the end of the line.
		pt.X = f.rect.Min.X
		x = pt.X
	}
	x += f.maxtab
	x -= (x - f.rect.Min.X) % f.maxtab
	if x-pt.X < int(b.Minwid) || x > c {
		x = pt.X + int(b.Minwid)
	}
	return x - pt.X
}

// TODO(rjk): Possibly does not work correctly.
// clean merges boxes where possible over boxes [n0, n1)
func (f *frameimpl) clean(pt image.Point, n0, n1 int) {
	//log.Println("clean", pt, n0, n1, f.Rect.Max.X)
	//	f.Logboxes("--- clean: starting ---")
	c := f.rect.Max.X
	nb := 0
	for nb = n0; nb < n1-1; nb++ {
		b := f.box[nb]
		pt = f.cklinewrap(pt, b)
		for f.box[nb].Nrune >= 0 &&
			nb < n1-1 &&
			f.box[nb+1].Nrune >= 0 &&
			pt.X+f.box[nb].Wid+f.box[nb+1].Wid < c {
			f.mergebox(nb)
			n1--
		}
		pt = f.advance(pt, f.box[nb])
	}

	for _, b := range f.box[nb:] {
		pt = f.cklinewrap(pt, b)
		pt = f.advance(pt, b)
	}
	f.lastlinefull = false
	if pt.Y >= f.rect.Max.Y {
		f.lastlinefull = true
	}
	//	f.Logboxes("--- clean: end")
}

func nbyte(f *frbox) int {
	return len(f.Ptr)
}

func nrune(b *frbox) int {
	if b.Nrune < 0 {
		return 1
	}
	return b.Nrune
}

func Rpt(min, max image.Point) image.Rectangle {
	return image.Rectangle{Min: min, Max: max}
}

// Logboxes shows the box model to the log for debugging convenience.
func (f *frameimpl) Logboxes(message string, args ...interface{}) {
	log.Printf(message, args...)
	for i, b := range f.box {
		if b != nil {
			log.Printf("	box[%d] -> %v\n", i, b)
		} else {
			log.Printf("	box[%d] is WRONGLY nil\n", i)
		}
	}
	log.Printf("end: "+message, args...)
}

func (b *frbox) String() string {
	if b.Nrune == -1 && b.Bc == '\n' {
		return "newline"
	} else if b.Nrune == -1 && b.Bc == '\t' {
		return fmt.Sprintf("tab width=%d,%d", b.Wid, b.Minwid)
	}
	return fmt.Sprintf("%#v width=%d nrune=%d", string(b.Ptr), b.Wid, b.Nrune)
}
