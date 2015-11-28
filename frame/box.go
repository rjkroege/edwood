package frame

import (
	"unicode/utf8"
)

const slop uint64 = 25

func (f *Frame) addbox(bn, n uint64) {
	if bn > uint64(f.nbox) {
		panic("Frame.addbox")
	}
	if uint64(f.nbox)+n > uint64(f.nalloc) {
		f.growbox(n + slop)
	}
	for i := uint64(f.nbox); i >= bn; i-- {
		f.box[i+n] = f.box[i]
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

func (f *Frame) growbox(delta uint64) {
	f.nalloc += int(delta)
	f.box = append(f.box, make([]*frbox, delta)...)
}

func (f *Frame) dupbox(bn uint64) {
	if f.box[bn].Nrune < 0 {
		panic("dupbox")
	}
	f.addbox(bn, 1)
	if f.box[bn].Nrune >= 0 {
		p := make([]byte, nbyte(f.box[bn])+1)
		copy(p, f.box[bn].Ptr)
		f.box[bn+1].Ptr = p
	}
}

func runeindex(p []byte, n uint64) int {
	offs := 0
	for i := uint64(0); i < n; i++ {
		if p[offs] < 0x80 {
			offs += 1
		} else {
			_, size := utf8.DecodeRune(p[offs:])
			offs += size
		}
	}
	return offs
}

func (f *Frame) truncatebox(b *frbox, n uint64) {
	if b.Nrune < 0 || b.Nrune < int(n) {
		panic("truncatebox")
	}
	b.Nrune -= int(n)
	b.Ptr[runeindex(b.Ptr, uint64(len(b.Ptr)))] = 0
	b.Wid = f.Font.StringWidth(string(b.Ptr))
}

func (f *Frame) chopbox(b *frbox, n uint64) {
	if b.Nrune < 0 || b.Nrune < int(n) {
		panic("chopbox")
	}
	i := runeindex(b.Ptr, n)
	copy(b.Ptr, b.Ptr[i:])
	b.Nrune -= int(n)
	b.Wid = f.Font.StringWidth(string(b.Ptr))
}

func (f *Frame) splitbox(bn, n uint64) {
	f.dupbox(bn)
	f.truncatebox(f.box[bn], uint64(f.box[bn].Nrune-int(n)))
	f.chopbox(f.box[bn+1], n)
}

func (f *Frame) mergebox(bn int) {
	f.Insure(bn, nbyte(f.box[bn])+nbyte(f.box[bn+1])+1)
	i := runeindex(f.box[bn].Ptr, uint64(f.box[bn].Nrune))
	copy(f.box[bn].Ptr[i:], f.box[bn+1].Ptr)
	f.box[bn].Wid += f.box[bn+1].Wid
	f.box[bn].Nrune += f.box[bn+1].Nrune
	f.delbox(bn+1, bn+1)
}

func (f *Frame) findbox(bn, p, q uint64) int {
	for i := 0; bn < uint64(f.nbox) && p+uint64(nrune(f.box[i])) <= q; i++ {
		p += uint64(nrune(f.box[i]))
		bn++
	}
	if p != q {
		f.splitbox(bn, uint64(q-p))
		bn++
	}
	return int(bn)
}
