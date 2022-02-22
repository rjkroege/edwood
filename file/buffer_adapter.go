package file

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

func NewTypeBuffer(inputrunes []rune, oeb *ObservableEditableBuffer) *Buffer {
	// TODO(rjk): Figure out how to plumb in the oeb object to setup Undo
	// observer callbacks.

	// TODO(rjk): We can do better.
	buffy := new(bytes.Buffer)
	buffy.Grow(len(inputrunes))
	for _, r := range inputrunes {
		buffy.WriteRune(r)
	}

	nb := NewBuffer(buffy.Bytes(), len(inputrunes))
	nb.oeb = oeb
	return nb
}

// Commit writes the in-progress edits to the real buffer instead of
// keeping them in the cache.
func (b *Buffer) Commit(seq int) {
	// NOP
}

// DeleteAt removes the rune range [p0,p1) from File.
func (b *Buffer) DeleteAt(rp0, rp1, seq int) {
	p0 := b.RuneTuple(rp0)
	p1 := b.RuneTuple(rp1)

	b.Delete(p0, p1, seq)

	if seq < 1 {
		b.FlattenHistory()
	}
}

// InsertAt inserts s runes at rune address p0.
func (b *Buffer) InsertAt(rp0 int, rs []rune, seq int) {
	p0 := b.RuneTuple(rp0)

	buffy := new(bytes.Buffer)
	for _, r := range rs {
		// TODO(rjk): Some error handling might be needed here?
		buffy.WriteRune(r)
	}
	s := buffy.Bytes()

	b.Insert(p0, s, len(rs), seq)

	if seq < 1 {
		b.FlattenHistory()
	}
}

// ReadC reads a single rune from the File.
// TODO(rjk): Implement RuneReader
func (b *Buffer) ReadC(q int) rune {
	p0 := b.RuneTuple(q)

	sr := io.NewSectionReader(b, int64(p0.B), 8)
	bsr := bufio.NewReaderSize(sr, 8)

	// TODO(rjk): Add some error checking?
	r, _, _ := bsr.ReadRune()
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

func (b *Buffer) InsertAtWithoutCommit(p0 int, s []rune, seq int) {
	b.InsertAt(p0, s, seq)
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
