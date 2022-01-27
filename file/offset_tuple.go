package file

import "fmt"

type OffSetTuple struct {
	b int
	r int
}

func (o OffSetTuple) String() string {
	return fmt.Sprintf("offsettuple b: %d r: %d", o.b, o.r)
}
