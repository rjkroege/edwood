package main

import (
	"crypto/sha1"
	"io/ioutil"
	"os"
	"strings"
)

type Buffer []rune

func NewBuffer() Buffer { return []rune{} }

func (b *Buffer) Insert(q0 uint, r []rune) {
	if q0 > uint(len(*b)) {
		panic("internal error: buffer.Insert: Out of range insertion")
	}
	(*b) = append((*b)[:q0], append(r, (*b)[q0:]...)...)
}

func (b *Buffer) Delete(q0, q1 uint) {
	if q0 > uint(len(*b)) || q1 > uint(len(*b)) {
		panic("internal error: buffer.Delete: Out-of-range Delete")
	}
	copy((*b)[q0:], (*b)[q1:])
	(*b) = (*b)[:uint(len(*b))-(q1-q0)] // Reslice to length
}

func (b *Buffer) Load(q0 uint, fd *os.File) (n uint, h [sha1.Size]byte, hasNulls bool, err error) {
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
	return uint(len(runes)), sha1.Sum(d), hasNulls, err
}

func (b *Buffer) Read(q0, n uint) (r []rune) {
	// TODO(flux): Can I just reslice here, or do I need to copy?
	if !(q0 <= uint(len(*b)) && q0+n <= uint(len(*b))) {
		panic("internal error: Buffer.Read")
	}

	return (*b)[q0 : q0+n]
}

func (b *Buffer) Close() {
	(*b).Reset()

}

func (b *Buffer) Reset() {
	(*b) = (*b)[0:0]
}

func (b *Buffer) nc() uint {
	return uint(len(*b))
}
