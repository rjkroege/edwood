package file

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// RunesToBytes converts and returns the []byte representation and number of runes.
// TODO(rjk): Replace this with util.Cvttorunes
func RunesToBytes(inputrunes []rune) ([]byte, int) {
	// TODO(rjk): We can do better.
	buffy := new(bytes.Buffer)
	buffy.Grow(len(inputrunes))
	for _, r := range inputrunes {
		buffy.WriteRune(r)
	}
	return buffy.Bytes(), len(inputrunes)
}

func NewTypeBuffer(inputrunes []rune, oeb *ObservableEditableBuffer) *Buffer {
	nb := NewBuffer(RunesToBytes(inputrunes))
	nb.oeb = oeb
	return nb
}

// Commit writes the in-progress edits to the real buffer instead of
// keeping them in the cache.
func (b *Buffer) Commit(seq int) {
	// NOP
}

// ReadC reads a single rune from the File.
// TODO(rjk): Implement RuneReader
func (b *Buffer) ReadC(q int) rune {
	p0 := b.RuneTuple(q)
	r, _, _ := b.ReadRuneAt(p0)
	return r
}

func (b *Buffer) IndexRune(r rune) int {
	p0 := b.RuneTuple(0)

	sr := io.NewSectionReader(b, int64(p0.B), int64(b.Size()))
	// TODO(rjk): Tune the default size.
	bsr := bufio.NewReader(sr)

	for ro := 0; ; ro++ {
		gr, _, err := bsr.ReadRune()
		if err != nil {
			return -1
		}
		if gr == r {
			return ro
		}
	}
	return -1
}

// Mark sets an Undo point and and discards Redo records. Call this at
// the beginning of a set of edits that ought to be undo-able as a unit.
// This is equivalent to file.Buffer.Commit() NB: current implementation
// permits calling Mark on an empty file to indicate that one can undo to
// the file state at the time of calling Mark.
func (b *Buffer) Mark() {
	b.SetUndoPoint()
}

func (b *Buffer) Read(rq0 int, r []rune) (int, error) {
	p0 := b.RuneTuple(rq0)

	sr := io.NewSectionReader(b, int64(p0.B), int64(b.Size()-p0.B))
	bsr := bufio.NewReader(sr)

	for i := range r {
		ir, _, err := bsr.ReadRune()
		if err != nil {
			return i, err
		}
		r[i] = ir
	}
	return len(r), nil
}

func (b *Buffer) Reader(rq0 int, rq1 int) io.Reader {
	p0 := b.RuneTuple(rq0)
	p1 := b.RuneTuple(rq1)

	return io.NewSectionReader(b, int64(p0.B), int64(p1.B-p0.B))
}

func (b *Buffer) String() string {
	sr := io.NewSectionReader(b, int64(0), int64(b.Size()))

	buffy := new(strings.Builder)

	// TODO(rjk): Add some error checking.
	io.Copy(buffy, sr)
	return buffy.String()
}

// viewedState returns a string representation of a Buffer b good for debugging.
func (b *Buffer) viewedState() string {
	sb := new(strings.Builder)

	fmt.Fprintf(sb, "Buffer (vws: %v, vwl: %v) {\n", b.vws, b.vwl)
	for p := b.begin; p != nil; p = p.next {
		if p == b.viewed {
			sb.WriteString("->")
		}
		fmt.Fprintf(sb, "	id: %d len: %d nr: %d data: %q\n", p.id, p.len(), p.nr, string(p.data))
	}
	sb.WriteString("}")
	return sb.String()
}
