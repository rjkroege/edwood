package file

import "fmt"

type OffSetTuple struct {
	B int
	R int
}

func (o OffSetTuple) String() string {
	return fmt.Sprintf("offsettuple b: %d r: %d", o.B, o.R)
}
