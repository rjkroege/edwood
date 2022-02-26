package file

import "fmt"

type OffsetTuple struct {
	B int
	R int
}

func (o OffsetTuple) String() string {
	return fmt.Sprintf("offsettuple b: %d r: %d", o.B, o.R)
}

func Ot(b, r int) OffsetTuple {
	return OffsetTuple{
		B: b,
		R: r,
	}
}

func (o OffsetTuple) decrement(p *piece) OffsetTuple {
	return Ot(
		o.B-len(p.data),
		o.R-p.nr,
	)
}
