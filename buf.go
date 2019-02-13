package main

import (
	"unicode/utf8"

	"github.com/rjkroege/edwood/internal/runes"
)

// Buffer is a mutable array of runes.
type Buffer []rune

func NewBuffer() Buffer { return []rune{} }

func (b *Buffer) Insert(q0 int, r []rune) {
	if q0 > (len(*b)) {
		panic("internal error: buffer.Insert: Out of range insertion")
	}
	(*b) = append((*b)[:q0], append(r, (*b)[q0:]...)...)
}

func (b *Buffer) Delete(q0, q1 int) {
	if q0 > (len(*b)) || q1 > (len(*b)) {
		panic("internal error: buffer.Delete: Out-of-range Delete")
	}
	copy((*b)[q0:], (*b)[q1:])
	(*b) = (*b)[:(len(*b))-(q1-q0)] // Reslice to length
}

func (b *Buffer) Read(q0 int, r []rune) (int, error) {
	n := copy(r, (*b)[q0:])
	return n, nil
}

func (b *Buffer) ReadC(q int) rune { return (*b)[q] }

func (b *Buffer) Close() {
	(*b).Reset()

}

func (b *Buffer) Reset() {
	(*b) = (*b)[0:0]
}

// nc returns the number of characters in the Buffer.
func (b *Buffer) nc() int {
	return len(*b)
}

// Nbyte returns the number of bytes needed to store the contents
// of the buffer in UTF-8.
func (b *Buffer) Nbyte() int {
	bc := 0
	for _, r := range *b {
		bc += utf8.RuneLen(r)
	}
	return bc
}

// TODO(flux): This is another design constraint of Buffer - we want to efficiently
// present contiguous segments of bytes, possibly by merging/flattening our tree
// when a view is requested. This should be a rare operation...
func (b *Buffer) View(q0, q1 int) []rune {
	if q1 > len(*b) {
		q1 = len(*b)
	}
	return (*b)[q0:q1]
}

func (b Buffer) IndexRune(r rune) int {
	return runes.IndexRune(b, r)
}

func (r Buffer) Equal(s Buffer) bool {
	return runes.Equal(r, s)
}
