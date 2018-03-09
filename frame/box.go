package frame

import (
	"unicode/utf8"
)

const slop = 25

// addbox adds  n boxes after bn and shifts the rest up: * box[bn+n]==box[bn]
func (f *Frame) addbox(bn, n int) {
	if bn > f.nbox {
		panic("Frame.addbox")
	}

	if f.nbox+n > f.nalloc {
		f.growbox(n + slop)
	}

	// TODO(rjk): This does some extra work becasue nbox != len(f.box)
	// Use a slice and remove nbox and nalloc.
	copy(f.box[bn+n:], f.box[bn:])
	f.nbox += int(n)
}

func (f *Frame) closebox(n0, n1 int) {
	if n0 >= f.nbox || n1 >= f.nbox || n1 < n0 {
		panic("Frame.closebox")
	}
	// TODO(rjk): Use copy.
	n1++

	for i := n1; i < f.nbox; i++ {
		f.box[i-(n1-n0)] = f.box[i]
	}
	f.nbox -= n1 - n0
	for i := f.nbox; i < f.nbox+(n1-n0); i++ {
		f.box[i] = nil
	}
}

func (f *Frame) delbox(n0, n1 int) {
	if n0 >= f.nbox || n1 >= f.nbox || n1 < n0 {
		panic("Frame.delbox")
	}
	f.freebox(n0, n1)
	f.closebox(n0, n1)
}

func (f *Frame) freebox(n0, n1 int) {
	if n1 < n0 {
		return
	}
	if n0 >= f.nbox || n1 >= f.nbox {
		panic("Frame.freebox")
	}
	n1++
	for i := n0; i < n1; i++ {
		f.box[i] = nil
	}
}

// growbox adds delta new frbox pointers to f.box
func (f *Frame) growbox(delta int) {
	f.nalloc += delta
	f.box = append(f.box, make([]*frbox, delta)...)
}

func (f *Frame) dupbox(bn int) {
	if f.box[bn].Nrune < 0 {
		panic("dupbox invalid Nrune")
	}

	cp := new(frbox)
	*cp = *f.box[bn]

	f.addbox(bn, 1)
	f.box[bn+1] = cp

	if f.box[bn].Nrune >= 0 {
		p := make([]byte, len(f.box[bn].Ptr))
		copy(p, f.box[bn].Ptr)
		f.box[bn+1].Ptr = p
	}
}

func runeindex(p []byte, n int) int {
	offs := 0
	for i := 0; i < n; i++ {
		if p[offs] < 0x80 {
			offs += 1
		} else {
			_, size := utf8.DecodeRune(p[offs:])
			offs += size
		}
	}
	return offs
}

// truncatebox drops the  last n characters from box b without allocation.
func (f *Frame) truncatebox(b *frbox, n int) {
	if b.Nrune < 0 || b.Nrune < int(n) {
		panic("truncatebox")
	}
	b.Nrune -= n
	b.Ptr = b.Ptr[0:runeindex(b.Ptr, b.Nrune)]
	b.Wid = f.Font.BytesWidth(b.Ptr)
}

// chopbox removes the first n chars from box b without allocation.
func (f *Frame) chopbox(b *frbox, n int) {
	if b.Nrune < 0 || b.Nrune < n {
		panic("chopbox")
	}
	i := runeindex(b.Ptr, n)
	b.Ptr = b.Ptr[i:]
	b.Nrune -= n
	b.Wid = f.Font.BytesWidth(b.Ptr)
}

// splitbox duplicates box [bn] and divides it at rune n into prefix and suffix boxes.
func (f *Frame) splitbox(bn, n int) {
	f.dupbox(bn)
	f.truncatebox(f.box[bn], f.box[bn].Nrune-n)
	f.chopbox(f.box[bn+1], n)
}

// mergebox combines boxes bn and bn+1
func (f *Frame) mergebox(bn int) {
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
// rune p in box bn.
func (f *Frame) findbox(bn, p, q int) int {
	for i := bn; bn < f.nbox && p+nrune(f.box[i]) <= q; i++ {
		p += nrune(f.box[i])
		bn++
	}

	if p != q {
		f.splitbox(bn, q-p)
		bn++
	}
	return bn
}
