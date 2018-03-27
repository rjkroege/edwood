package main

import (
	"testing"
)

func TestAddr(t *testing.T) {

	testtab := []struct {
		dot  Range
		addr string
		r    Range
		qp   bool
		q    int
	}{
		{Range{0, 0}, "2", Range{10, 21}, true, 1},
		{Range{0, 0}, "1,2", Range{0, 21}, true, 3},
		{Range{0, 0}, "2,3", Range{10, 39}, true, 3},
		{Range{0, 0}, "2,2", Range{10, 21}, true, 3},
		{Range{0, 0}, "1+", Range{21, 39}, true, 2},
		{Range{0, 0}, "1+-", Range{10, 21}, true, 3},
		{Range{12, 12}, "+-", Range{10, 21}, true, 2},

		{Range{0, 0}, "/addressing", Range{28, 38}, true, 11},
		{Range{0, 0}, "/addressing/+-", Range{21, 39}, true, 14},
		{Range{0, 0}, "/d+", Range{29, 31}, true, 3},
		{Range{0, 0}, "/d+/,/ss/", Range{29, 35}, true, 9},
		{Range{0, 0}, "/d+/,/s+/", Range{29, 4}, true, 9},
	}

	text := &TextBuffer{0, 0, []rune("This is a\nshort text\nto try addressing\n")}

	for i, test := range testtab {
		r, ep, q := address(false, text, Range{-1, -1}, test.dot, 0, len(test.addr),
			func(q int) rune { return []rune(test.addr)[q] }, true)
		if test.r != r || test.qp != ep || test.q != q {
			t.Errorf("test %d: address %v: %v, %v, %v", i, test.addr, r, ep, q)
		}
	}
}
