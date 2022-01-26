package file

import "fmt"

type OffSetTuple struct {
	b int
	r int
}

// can work this api to be a little better...
func (o OffSetTuple) add(b, r int) OffSetTuple {
	return OffSetTuple{
		b: o.b + b,
		r: o.r + r,
	}
}

func (o OffSetTuple) String() string {
	return fmt.Sprintf("offsettuple b: %d r: %d", o.b, o.r)
}
