// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package utf8Bytes provides an efficient way to index bytes by rune rather than by byte.
// utf8Bytes is a modified version of utf8string found at "https://cs.opensource.google/go/x/exp/+/master:utf8string/"
package file // import "golang.org/x/exp/utf8Bytes"

import (
	"errors"
	"unicode/utf8"
)

// Bytes wraps a regular bytes with a small structure that provides more
// efficient indexing by code point index, as opposed to byte index.
// Scanning incrementally forwards or backwards is O(1) per index operation
// (although not as fast a range clause going forwards).  Random access is
// O(N) in the length of the string, but the overhead is less than always
// scanning from the beginning.
// If the string is ASCII, random access is O(1).
type Bytes struct {
	b        []byte
	numRunes int
	// If width > 0, the rune at runePos starts at bytePos and has the specified width.
	width    int
	bytePos  int
	runePos  int
	nonASCII int // byte index of the first non-ASCII rune.
}

// NewBytes returns a new UTF-8 Bytes with the provided contents.
func NewBytes(contents []byte) *Bytes {
	return new(Bytes).Init(contents)
}

// Init initializes an existing Bytes to hold the provided contents.
// It returns a pointer to the initialized Bytes.
func (bytes *Bytes) Init(contents []byte) *Bytes {
	bytes.b = contents
	bytes.bytePos = 0
	bytes.runePos = 0
	for i := 0; i < len(contents); i++ {
		if contents[i] >= utf8.RuneSelf {
			// Not ASCII.
			bytes.numRunes = utf8.RuneCount(contents)
			_, bytes.width = utf8.DecodeRune(contents)
			bytes.nonASCII = i
			return bytes
		}
	}
	// ASCII is simple.  Also, the empty string is ASCII.
	bytes.numRunes = len(contents)
	bytes.width = 0
	bytes.nonASCII = len(contents)
	return bytes
}

// Bytes returns the contents of the Bytes.  This method also means the
// Bytes is directly printable by fmt.Print.
func (bytes *Bytes) Byte() []byte {
	return bytes.b
}

// RuneCount returns the number of runes (Unicode code points) in the Bytes.
func (bytes *Bytes) RuneCount() int {
	return bytes.numRunes
}

// IsASCII returns a boolean indicating whether the Bytes contains only ASCII bytes.
func (bytes *Bytes) IsASCII() bool {
	return bytes.width == 0
}

// Slice returns the string sliced at rune positions [i:j].
func (bytes *Bytes) Slice(i, j int) []byte {
	// ASCII is easy.  Let the compiler catch the indexing error if there is one.
	if j < bytes.nonASCII {
		return bytes.b[i:j]
	}
	if i < 0 || j > bytes.numRunes || i > j {
		panic(errSliceOutOfRange)
	}
	if i == j {
		return []byte("")
	}
	// For non-ASCII, after At(i), bytePos is always the position of the indexed character.
	var low, high int
	switch {
	case i < bytes.nonASCII:
		low = i
	case i == bytes.numRunes:
		low = len(bytes.b)
	default:
		bytes.At(i)
		low = bytes.bytePos
	}
	switch {
	case j == bytes.numRunes:
		high = len(bytes.b)
	default:
		bytes.At(j)
		high = bytes.bytePos
	}
	return bytes.b[low:high]
}

// At returns the rune with index i in the Bytes.  The sequence of runes is the same
// as iterating over the contents with a "for range" clause.
func (bytes *Bytes) At(i int) rune {
	// ASCII is easy.  Let the compiler catch the indexing error if there is one.
	if i < bytes.nonASCII {
		return rune(bytes.b[i])
	}

	// Now we do need to know the index is valid.
	if i < 0 || i >= bytes.numRunes {
		panic(errOutOfRange)
	}

	var r rune

	// Five easy common cases: within 1 spot of bytePos/runePos, or the beginning, or the end.
	// With these cases, all scans from beginning or end work in O(1) time per rune.
	switch {

	case i == bytes.runePos-1: // backing up one rune
		r, bytes.width = utf8.DecodeLastRune(bytes.b[0:bytes.bytePos])
		bytes.runePos = i
		bytes.bytePos -= bytes.width
		return r
	case i == bytes.runePos+1: // moving ahead one rune
		bytes.runePos = i
		bytes.bytePos += bytes.width
		fallthrough
	case i == bytes.runePos:
		r, bytes.width = utf8.DecodeRune(bytes.b[bytes.bytePos:])
		return r
	case i == 0: // start of string
		r, bytes.width = utf8.DecodeRune(bytes.b)
		bytes.runePos = 0
		bytes.bytePos = 0
		return r

	case i == bytes.numRunes-1: // last rune in string
		r, bytes.width = utf8.DecodeLastRune(bytes.b)
		bytes.runePos = i
		bytes.bytePos = len(bytes.b) - bytes.width
		return r
	}

	// We need to do a linear scan.  There are three places to start from:
	// 1) The beginning
	// 2) bytePos/runePos.
	// 3) The end
	// Choose the closest in rune count, scanning backwards if necessary.
	forward := true
	if i < bytes.runePos {
		// Between beginning and pos.  Which is closer?
		// Since both i and runePos are guaranteed >= nonASCII, that'bytes the
		// lowest location we need to start from.
		if i < (bytes.runePos-bytes.nonASCII)/2 {
			// Scan forward from beginning
			bytes.bytePos, bytes.runePos = bytes.nonASCII, bytes.nonASCII
		} else {
			// Scan backwards from where we are
			forward = false
		}
	} else {
		// Between pos and end.  Which is closer?
		if i-bytes.runePos < (bytes.numRunes-bytes.runePos)/2 {
			// Scan forward from pos
		} else {
			// Scan backwards from end
			bytes.bytePos, bytes.runePos = len(bytes.b), bytes.numRunes
			forward = false
		}
	}
	if forward {
		// TODO: Is it much faster to use a range loop for this scan?
		for {
			r, bytes.width = utf8.DecodeRune(bytes.b[bytes.bytePos:])
			if bytes.runePos == i {
				break
			}
			bytes.runePos++
			bytes.bytePos += bytes.width
		}
	} else {
		for {
			r, bytes.width = utf8.DecodeLastRune(bytes.b[0:bytes.bytePos])
			bytes.runePos--
			bytes.bytePos -= bytes.width
			if bytes.runePos == i {
				break
			}
		}
	}
	return r
}

// HasNull returns true if Bytes contains a null rune.
func (b *Bytes) HasNull() bool {
	for i := 0; i < b.numRunes; i++ {
		if b.At(i) == 0 {
			return true
		}
	}
	return false
}

// Read implements the io.Reader interface.
func (b *Bytes) Read(buf []byte) (n int, err error) {
	n = copy(buf, b.Byte())
	return n, nil
}

var errOutOfRange = errors.New("utf8Bytes: index out of range")
var errSliceOutOfRange = errors.New("utf8Bytes: slice index out of range")
