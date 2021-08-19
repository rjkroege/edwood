package file

import (
	"io"
	"strings"
	"unicode/utf8"

	"github.com/rjkroege/edwood/runes"
)

// RuneArray is a mutable array of runes.
type RuneArray []rune

func NewRuneArray() RuneArray { return []rune{} }

func (b *RuneArray) Insert(q0 int, r []rune) {
	if q0 > len(*b) {
		panic("internal error: buffer.Insert: Out of range insertion")
	}
	*b = append((*b)[:q0], append(r, (*b)[q0:]...)...)
}

func (b *RuneArray) Delete(q0, q1 int) {
	if q0 > len(*b) || q1 > len(*b) {
		panic("internal error: buffer.Delete: Out-of-range Delete")
	}
	copy((*b)[q0:], (*b)[q1:])
	*b = (*b)[:len(*b)-(q1-q0)] // Reslice to length
}

func (b *RuneArray) Read(q0 int, r []rune) (int, error) {
	n := copy(r, (*b)[q0:])
	return n, nil
}

// Reader returns reader for text at [q0, q1).
//
// TODO(fhs): Once RuneArray implements io.ReaderAt,
// we can use io.SectionReader instead of this function.
func (b *RuneArray) Reader(q0, q1 int) io.Reader {
	return strings.NewReader(string((*b)[q0:q1]))
}

func (b *RuneArray) ReadC(q int) rune { return (*b)[q] }

// String returns a string representation of buffer. See fmt.Stringer interface.
func (b *RuneArray) String() string { return string(*b) }

func (b *RuneArray) Reset() {
	*b = (*b)[0:0]
}

// nc returns the number of characters in the RuneArray.
func (b *RuneArray) Nc() int {
	return len(*b)
}

// Nbyte returns the number of bytes needed to store the contents
// of the buffer in UTF-8.
func (b *RuneArray) Nbyte() int {
	bc := 0
	for _, r := range *b {
		bc += utf8.RuneLen(r)
	}
	return bc
}

// TODO(flux): This is another design constraint of RuneArray - we want to efficiently
// present contiguous segments of bytes, possibly by merging/flattening our tree
// when a view is requested. This should be a rare operation...
func (b *RuneArray) View(q0, q1 int) []rune {
	if q1 > len(*b) {
		q1 = len(*b)
	}
	return (*b)[q0:q1]
}

func (b RuneArray) IndexRune(r rune) int {
	return runes.IndexRune(b, r)
}

func (r RuneArray) Equal(s RuneArray) bool {
	return runes.Equal(r, s)
}
