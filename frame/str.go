package frame

import ()

const (
	CHUNK = 16
)

func roundup(n int) int {
	return ((n+CHUNK) &^ (CHUNK-1))
}

func (f *Frame) AllocStr(n uint) []byte {
	return make([]byte, n)
}

func (f *Frame) Insure(bn int, n uint) {
	b := f.box[bn]
	if b.Nrune < 0 {
		panic("Frame.Insure")
	}
	if roundup(int(b.Nrune)) > int(n) {
		return
	}
	p := f.AllocStr(n)
	copy(p, b.Ptr[:nbyte(b)+1])
	b.Ptr = p
}
