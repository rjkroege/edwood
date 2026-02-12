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
	b   *Buffer
	pos OffsetTuple
	seq int
}

// Write implements io.Writer
func (w *bufferWriter) Write(p []byte) (int, error) {
	nr := utf8.RuneCount(p)
	npos := Ot(w.pos.B+len(p), w.pos.R+nr)
	err := w.b.Insert(w.pos, p, nr, w.seq)
	w.pos = npos
	return len(p), err
}

// The entire I/O op is treated as a single undoable action.
func (b *Buffer) NewWriter(pos OffsetTuple, seq int) io.Writer {
	return &bufferWriter{
		b:   b,
		pos: pos,
		seq: seq,
	}
}
