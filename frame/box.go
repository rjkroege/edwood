package frame

import (
	"flag"
	"fmt"
	"log"
	"unicode/utf8"
)

// addbox adds  n boxes after bn and shifts the rest up: * box[bn+n]==box[bn]
func (f *frameimpl) addbox(bn, n int) {
	if bn > len(f.box) {
		panic(fmt.Sprint("Frame.addbox", " bn=", bn, " len(f.box)", len(f.box)))
	}
	f.box = append(f.box, make([]*frbox, n)...)
	copy(f.box[bn+n:], f.box[bn:])
}

func (f *frameimpl) closebox(n0, n1 int) {
	if n0 >= len(f.box) || n1 >= len(f.box) || n1 < n0 {
		panic(fmt.Sprint("Frame.closebox bounds bad", " n0=", n0, " n1=", n1, " len(box)", len(f.box)))
	}

	n1++
	copy(f.box[n0:], f.box[n1:])
	f.box = f.box[0 : len(f.box)-(n1-n0)]
}

func (f *frameimpl) delbox(n0, n1 int) {
	// TODO(rjk): One of delbox and closebox don't belong.
	f.closebox(n0, n1)
}

func (b *frbox) clone() *frbox {
	// Shallow copy.
	cp := new(frbox)
	*cp = *b

	// Now deep copy the byte array
	// TODO(rjk): Adjust when we use strings.
	cp.Ptr = make([]byte, len(b.Ptr))
	copy(cp.Ptr, b.Ptr)
	return cp
}

// dupbox duplicates box i. box i must exist.
func (f *frameimpl) dupbox(i int) {
	if i >= len(f.box) {
		f.Logboxes("-- dupbox sadness -- ")
		panic(fmt.Sprint("dupbox i is out of bounds", " i=", i))
	}
	if f.box[i].Nrune < 0 {
		panic("dupbox invalid Nrune")
	}

	nb := f.box[i].clone()
	f.box = append(f.box, nil)
	copy(f.box[i+1:], f.box[i:])
	f.box[i] = nb

}

// TODO(rjk): Nicer way when we have a string for box contents.
func runeindex(p []byte, n int) int {
	offs := 0
	for i := 0; i < n; i++ {
		if p[offs] < 0x80 {
			offs++
		} else {
			_, size := utf8.DecodeRune(p[offs:])
			offs += size
		}
	}
	return offs
}

// truncatebox drops the  last n characters from box b without allocation.
// TODO(rjk): make a method on a frbox
// TODO(rjk): measure height.
func (f *frameimpl) truncatebox(b *frbox, n int) {
	if b.Nrune < 0 || b.Nrune < int(n) {
		f.Logboxes("-- truncatebox panic -- ")
		panic(fmt.Sprint("Frame.truncatebox", " Nrune=", b.Nrune, " n=", n))
	}
	b.Nrune -= n
	b.Ptr = b.Ptr[0:runeindex(b.Ptr, b.Nrune)]
	b.Wid = f.font.BytesWidth(b.Ptr)
}

// chopbox removes the first n chars from box b without allocation.
// TODO(rjk): measure height
func (f *frameimpl) chopbox(b *frbox, n int) {
	if b.Nrune < 0 || b.Nrune < n {
		f.Logboxes("-- panic in chopbox --")
		panic(fmt.Sprint("chopbox", " b.Nrune=", b.Nrune, " n=", n))
	}
	i := runeindex(b.Ptr, n)
	b.Ptr = b.Ptr[i:]
	b.Nrune -= n
	b.Wid = f.font.BytesWidth(b.Ptr)
}

// splitbox duplicates box [bn] and divides it at rune n into prefix and suffix boxes.
// It is an error to try to split a non-existent box?
// TODO(rjk): Figure out if you want this to be so.
func (f *frameimpl) splitbox(bn, n int) {
	if bn > len(f.box) {
		panic(fmt.Sprint("splitbox", "bn=", bn, "n=", n))
	}
	f.dupbox(bn)
	f.truncatebox(f.box[bn], f.box[bn].Nrune-n)
	f.chopbox(f.box[bn+1], n)
}

// mergebox combines boxes bn and bn+1
func (f *frameimpl) mergebox(bn int) {
	b1n := len(f.box[bn].Ptr)
	b2n := len(f.box[bn+1].Ptr)

	b := make([]byte, 0, b1n+b2n)
	b = append(b, f.box[bn].Ptr[0:b1n]...)
	b = append(b, f.box[bn+1].Ptr[0:b2n]...)
	f.box[bn].Ptr = b
	f.box[bn].Nrune += f.box[bn+1].Nrune
	f.box[bn].Wid += f.box[bn+1].Wid

	f.delbox(bn+1, bn+1)
}

// findbox finds the box containing q and puts q on a box boundary starting from
// rune p in box bn. NB: p must be the first rune in box[bn].
func (f *frameimpl) findbox(bn, p, q int) int {
	for _, b := range f.box[bn:] {
		if p+nrune(b) > q {
			break
		}
		p += nrune(b)
		bn++
	}
	if p != q {
		f.splitbox(bn, q-p)
		bn++
	}
	return bn
}

// TODO(rjk): Consider moving this code to a new file.
var validate = flag.Bool("validateboxes", false, "Check that box model is valid")

// validateboxmodel returns true if f's box model is valid.
func (f *frameimpl) validateboxmodel(format string, args ...interface{}) {
	if !*validate {
		return
	}

	// Test 0. No holes in the array of boxes.
	for _, b := range f.box {
		if b == nil {
			log.Printf(format, args...)
			f.Logboxes("-- holes in nbox portion of box array --")
			panic("-- holes in nbox portion of box array --")
		}
	}

	// Test 1. NChars is valid
	total := 0
	for _, b := range f.box {
		if b.Nrune < 0 {
			total++
		} else {
			total += b.Nrune
		}
	}
	if total != f.nchars {
		log.Printf(format, args...)
		f.Logboxes("-- runes in boxes != NChars --")
		panic("-- runes in boxes != NChars --")
	}

	// TODO(rjk): Every box is sane.
	for _, b := range f.box {
		// Nrune is right for this box.
		if b.Nrune >= 0 {
			s := string(b.Ptr)
			c := 0
			for range s {
				c++
			}
			if c != b.Nrune {
				log.Printf(format, args...)
				f.Logboxes("-- box with contents has invalid rune count --")
				panic("-- box with contents has invalid rune count --")
			}
		}

		// The width is right.
		if b.Nrune >= 0 {
			s := string(b.Ptr)
			if b.Wid != f.font.StringWidth(s) {
				log.Printf(format, args...)
				f.Logboxes("-- box with contents has invalid width --")
				panic("-- box with contents has invalid width --")
			}
		}

		// TODO(rjk): newline and tab boxes are rational.
	}

	// TODO(rjk): Every box fits in Rect.

}
