package frame

import (
	"log"
	"unicode/utf8"
)

const slop  = 25

// Brief commentary: the allocation conventions of the box datastructure are inconsistent and weird.

// addbox adds  n boxes after bn and shifts the rest up: * box[bn+n]==box[bn]
func (f *Frame) addbox(bn, n int) {
	log.Println("addbox", bn, n)
	if bn > f.nbox {
		panic("Frame.addbox")
	}

	if f.nbox+n > f.nalloc {
		f.growbox(n + slop)
	}

	for i := f.nbox - 1; i >= bn; i-- {
		log.Println("addbox: f.nbox", f.nbox, "i", i, "n", n, "i+n", i+n)
		if f.box[i+n] == nil {
			f.box[i+n] = new(frbox)
		}
		*f.box[i+n] = *f.box[i]
	}
	f.nbox += int(n)
}

func (f *Frame) closebox(n0, n1 int) {
	if n0 >= f.nbox || n1 >= f.nbox || n1 < n0 {
		panic("Frame.closebox")
	}
	n1++
	for i := n1; i < f.nbox; i++ {
		f.box[i-(n1-n0)] = f.box[i]
	}
	f.nbox -= n1 - n0
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
		if f.box[i].Nrune >= 0 {
			f.box[i].Ptr = nil
		}
	}
}

// growbox adds delta new frbox pointers to f.box
func (f *Frame) growbox(delta int) {
	f.nalloc += delta
	f.box = append(f.box, make([]*frbox, delta)...)
}


func (f *Frame) dupbox(bn int) {
	log.Println("dupbox", bn)

	if f.box[bn].Nrune < 0 {
		panic("dupbox invalid Nrune")
	}

	cp := new(frbox)
	*cp = *f.box[bn]

	f.addbox(bn, 1)

	f.box[bn+1] = cp

	log.Printf("dupbox bn[%d] = %#v, bn+1[%d] = %#v\n", bn, string(f.box[bn].Ptr), bn+1, string(f.box[bn+1].Ptr))

//	if f.box[bn].Nrune >= 0 {
//		p := make([]byte, nbyte(f.box[bn])+1)
//		copy(p, f.box[bn].Ptr)
//		f.box[bn+1].Ptr = p
//	}
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

// fontmetrics lets tests mock the calls into draw for measuring the
// width of UTF8 slices.
type fontmetrics interface {
	BytesWidth([]byte) int
}

// truncatebox drops the  last n characters without allocation.
func (b *frbox) truncatebox(n int, m fontmetrics) {
	if b.Nrune < 0 || b.Nrune < int(n) {
		panic("truncatebox")
	}
	b.Nrune -= n
	b.Ptr = b.Ptr[0:runeindex(b.Ptr, b.Nrune)]
	b.Wid = m.BytesWidth(b.Ptr)
}

// truncatebox drops the  last n characters without allocation.
func (f *Frame) truncatebox(b *frbox, n int) {
	b.truncatebox(n, f.Font)
}

// chopbox removes the first n chars without allocation.
func (b *frbox) chopbox(n int, m fontmetrics) {
	if b.Nrune < 0 || b.Nrune < n {
		panic("chopbox")
	}
	i := runeindex(b.Ptr, n)
	b.Ptr = b.Ptr[i:]
	b.Nrune -= n
	b.Wid = m.BytesWidth(b.Ptr)
}

// chopbox removes the first n chars without allocation.
func (f *Frame) chopbox(b *frbox, n int) {
	b.chopbox(n, f.Font)
}

func (f *Frame) splitbox(bn, n int) {
	log.Println("splitbox", bn, n)
	f.dupbox(bn)
	f.truncatebox(f.box[bn], f.box[bn].Nrune - n)
	f.chopbox(f.box[bn+1], n)
	log.Printf("splitbox end bn[%d] = %#v, bn+1[%d] = %#v\n", bn, string(f.box[bn].Ptr), bn+1, string(f.box[bn+1].Ptr))
}

func (f *Frame) mergebox(bn int) {
	f.Insure(bn, nbyte(f.box[bn])+nbyte(f.box[bn+1])+1)
	i := runeindex(f.box[bn].Ptr, f.box[bn].Nrune)
	copy(f.box[bn].Ptr[i:], f.box[bn+1].Ptr)
	f.box[bn].Wid += f.box[bn+1].Wid
	f.box[bn].Nrune += f.box[bn+1].Nrune
	f.delbox(bn+1, bn+1)
}

// findbox finds the box containing q and puts q on a box boundary.
func (f *Frame) findbox(bn, p, q int) int {
	log.Println("findbox", bn, p, q)

	for i := 0; bn < f.nbox && p+nrune(f.box[i]) <= q; i++ {
		p += nrune(f.box[i])
		bn++
	}

	if p != q {
		f.splitbox(bn, q-p)
		bn++
	}
	return bn
}
