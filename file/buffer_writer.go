package file

import (
	"io"
	"unicode/utf8"
)

// Enforce interface implementation.
var _ io.Writer = (*bufferWriter)(nil)

// bufferWriter wraps a Buffer to provide an io.Writer interface
// TODO(rjk): Could put pos inside Buffer?
type bufferWriter struct {
	b       *Buffer
	pos     OffsetTuple
	seq     int
	nr      int
	hasnull bool
}

func removeNulls(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for _, b := range data {
		if b != 0 {
			result = append(result, b)
		}
	}
	return result
}

// Write implements io.Writer
func (w *bufferWriter) Write(mnp []byte) (int, error) {
	p := removeNulls(mnp)
	if len(p) != len(mnp) {
		w.hasnull = true
	}

	nr := utf8.RuneCount(p)
	npos := Ot(w.pos.B+len(p), w.pos.R+nr)
	err := w.b.Insert(w.pos, p, nr, w.seq)
	w.pos = npos
	w.nr += nr

	//  can call the inserted here?
	// TODO(rjk): is the order sensible?
	/// w.b.oeb.inserted()

	return len(p), err
}

func (w *bufferWriter) Nr() int {
	return w.nr
}

func (w *bufferWriter) HadNull() bool {
	return w.hasnull
}

// The entire I/O op is treated as a single undoable action.
func (b *Buffer) NewWriter(pos OffsetTuple, seq int) *bufferWriter {
	return &bufferWriter{
		b:       b,
		pos:     pos,
		seq:     seq,
		nr:      0,
		hasnull: false,
	}
}
