package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/sam"
)

func TestAddr(t *testing.T) {

	testtab := []struct {
		dot  Range
		addr string
		r    Range
		ep   bool
		q    int
	}{
		{Range{0, 0}, "2", Range{10, 21}, true, 1},
		{Range{0, 0}, "1,2", Range{0, 21}, true, 3},
		{Range{0, 0}, "2,3", Range{10, 39}, true, 3},
		{Range{0, 0}, "2,2", Range{10, 21}, true, 3},
		{Range{0, 0}, "1;2", Range{0, 21}, true, 3},
		{Range{0, 0}, "2;3", Range{10, 39}, true, 3},
		{Range{0, 0}, "2;2", Range{10, 21}, true, 3},
		{Range{0, 0}, "1+", Range{21, 39}, true, 2},
		{Range{0, 0}, "1+-", Range{10, 21}, true, 3},
		{Range{12, 12}, "+-", Range{10, 21}, true, 2},
		{Range{0, 0}, ".", Range{0, 0}, true, 1},
		{Range{0, 0}, "$", Range{39, 39}, true, 1},
		{Range{0, 0}, "#10", Range{10, 10}, true, 3},
		{Range{10, 10}, ",2", Range{0, 21}, true, 2},
		{Range{0, 0}, "2,", Range{10, 39}, true, 2},
		{Range{0, 0}, "$-1", Range{21, 39}, true, 3},

		{Range{0, 0}, "/addressing", Range{28, 38}, true, 11},
		{Range{0, 0}, "/addressing\n", Range{28, 38}, true, 11},
		{Range{0, 0}, "/text\\nto", Range{16, 23}, true, 9},
		{Range{0, 0}, "/addressing/+-", Range{21, 39}, true, 14},
		{Range{0, 0}, "/d+", Range{29, 31}, true, 3},
		{Range{0, 0}, "/d+/,/ss/", Range{29, 35}, true, 9},
		{Range{0, 0}, "/d+/,/s+/", Range{29, 4}, true, 9},
		{Range{0, 0}, "2,/i/", Range{10, 3}, true, 5},
		{Range{0, 0}, "2;/i/", Range{10, 36}, true, 5},
		{Range{39, 39}, "?s", Range{34, 35}, true, 2},

		{Range{0, 0}, "line2", Range{0, 0}, true, 0},
		{Range{0, 0}, "2$", Range{10, 21}, true, 1},
		{Range{0, 0}, "#", Range{0, 0}, true, 0},
		{Range{0, 0}, "#X", Range{0, 0}, true, 1},
	}

	text := sam.NewTextBuffer(0, 0, []rune("This is a\nshort text\nto try addressing\n"))

	for i, test := range testtab {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			r, ep, q := address(false, text, Range{-1, -1}, test.dot, 0, len(test.addr),
				func(q int) rune { return []rune(test.addr)[q] }, true)
			if test.r != r || test.ep != ep || test.q != q {
				t.Errorf("address %q: r=%v, ep=%v, q=%v; expected r=%v, ep=%v, q=%v",
					test.addr, r, ep, q, test.r, test.ep, test.q)
			}
		})
	}
}

func TestAcmeregexp(t *testing.T) {
	warnings = nil
	pattern = nil
	defer func() {
		warnings = nil
		pattern = nil
	}()

	tt := []struct {
		name    string
		pat     string
		dir     int
		r       Range
		found   bool
		warning string
	}{
		{"FirstEmptyPat", "", Fore, Range{0, 0}, false, "no previous regular expression\n"},
		{"pat=abcd", "abcd", Fore, Range{0, 4}, true, ""},
		{"SecondEmptyPat", "", Fore, Range{0, 4}, true, ""},
		{"pat=αβξδ", "αβξδ", Fore, Range{5, 9}, true, ""},
		{"NoMatch", "xyz", Fore, Range{-1, -1}, false, "no match for regexp\n"},
		{"InvalidPat", "(abcd", Fore, Range{0, 0}, false, ""},
		{"Backwards", "abcd", Back, Range{0, 4}, true, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			warnings = nil
			text := &Text{
				file: file.MakeObservableEditableBuffer("", []rune("abcd αβξδ\n")),
			}
			lim := Range{
				0,
				text.file.Nr(),
			}
			start := Range{0, 0}
			if tc.dir == Back {
				start = Range{text.file.Nr(), text.file.Nr()}
			}
			r, found := acmeregexp(true, text, lim, start, tc.pat, tc.dir)
			if found != tc.found {
				t.Errorf("found=%v; want %v", found, tc.found)
			}
			if !reflect.DeepEqual(r, tc.r) {
				t.Errorf("range=%v; want %v", r, tc.r)
			}
			if tc.warning != "" {
				want := tc.warning
				if len(warnings) == 0 {
					t.Fatalf("no warning generated; want %q", want)
				}
				got := warnings[0].buf.String()
				if got != want {
					t.Errorf("warning is %q; want %q", got, want)
				}
			}
		})
	}
}
