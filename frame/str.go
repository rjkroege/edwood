package frame

const (
	chunk = 16
)

func roundup(n int) int {
	return ((n + chunk) &^ (chunk - 1))
}

func (f *frameimpl) Insure(bn int, n uint) {
	b := f.box[bn]
	if b.Nrune < 0 {
		panic("Frame.Insure")
	}
	if roundup(int(b.Nrune)) > int(n) {
		return
	}
	p := make([]byte, n)
	copy(p, b.Ptr[:nbyte(b)+1])
	b.Ptr = p
}
