package main

import (
	"crypto/sha1"
	"io/ioutil"
	"os"
	"strings"
	"unicode/utf8"
)

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

func (b *Buffer) Load(q0 int, fd *os.File) (n int, h [sha1.Size]byte, hasNulls bool, err error) {
	// TODO(flux): Innefficient to load the file, then copy into the slice,
	// but I need the UTF-8 interpretation.  I could fix this by using a
	// UTF-8 -> []rune reader on top of the os.File instead.

	d, err := ioutil.ReadAll(fd)
	if err != nil {
		warning(nil, "read error in Buffer.Load")
	}
	s := string(d)
	s = strings.Replace(s, "\000", "", -1)
	hasNulls = len(s) != len(d)
	runes := []rune(s)
	(*b).Insert(q0, runes)
	return (len(runes)), sha1.Sum(d), hasNulls, err
}

func (b *Buffer) Read(q0 int, r []rune) (n int, err error) {
	n = len(r)
	if !(q0 <= (len(*b)) && q0+n <= (len(*b))) {
		panic("internal error: Buffer.Read")
	}
	copy(r, (*b)[q0:q0+n])
	return n, nil
}

func (b *Buffer) ReadC(q int) rune { return (*b)[q] }

func (b *Buffer) Close() {
	(*b).Reset()

}

func (b *Buffer) Reset() {
	(*b) = (*b)[0:0]
}

func (b *Buffer) Nc() int {
	return len(*b)
}

func fbufalloc() []rune {
	return make([]rune, BUFSIZE/utf8.UTFMax)
}
