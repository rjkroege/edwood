package file

import (
	"fmt"
	"io"
)

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

func (p0 OffsetTuple) Less(p1 OffsetTuple) bool {
	return p0.R < p1.R
}

func (p0 OffsetTuple) Add(b, r int) OffsetTuple {
	return Ot(p0.B+b, p0.R+r)
}

func (p0 OffsetTuple) Sub(b, r int) OffsetTuple {
	return Ot(p0.B-b, p0.R-r)
}

var (
	// Force that BufferCursor is an io.RuneReader.
	_ io.RuneReader = (*BufferCursor)(nil)

	// Implement other interfaces as convenient.
)

type BufferCursor struct {
	p0 OffsetTuple
	p1 OffsetTuple
	b  *Buffer
}

func MakeBufferCursor(b *Buffer, p0, p1 OffsetTuple) *BufferCursor {
	return &BufferCursor{
		b:  b,
		p0: p0,
		p1: p1,
	}
}

func (cursor *BufferCursor) ReadRune() (rune, int, error) {
	if cursor.p0.Less(cursor.p1) {
		r, size, err := cursor.b.ReadRuneAt(cursor.p0)
		cursor.p0 = cursor.b.RuneTuple(cursor.p0.R + 1)
		return r, size, err
	}
	return 0, 0, io.EOF
}
