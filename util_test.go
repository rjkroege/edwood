package main

import (
	"github.com/rjkroege/edwood/util"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCvttorunes(t *testing.T) {
	testCases := []struct {
		p     []byte
		n     int
		r     []rune
		nb    int
		nulls bool
	}{
		{[]byte("Hello world"), 11, []rune("Hello world"), 11, false},
		{[]byte("Hello \x00\x00world"), 13, []rune("Hello world"), 13, true},
		{[]byte("Hello 世界"), 6 + 3 + 3, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 1, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 2, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 1, []rune("Hello 世\uFFFD"), 6 + 3 + 1, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 2, []rune("Hello 世\uFFFD\uFFFD"), 6 + 3 + 2, false},
		{[]byte("\xe4\xb8\x96界 hello"), 3 + 3 + 6, []rune("世界 hello"), 3 + 3 + 6, false},
		{[]byte("\xb8\x96界 hello"), 2 + 3 + 6, []rune("\uFFFD\uFFFD界 hello"), 2 + 3 + 6, false},
		{[]byte("\x96界 hello"), 1 + 3 + 6, []rune("\uFFFD界 hello"), 1 + 3 + 6, false},
	}
	for _, tc := range testCases {
		r, nb, nulls := util.Cvttorunes(tc.p, tc.n)
		if !reflect.DeepEqual(r, tc.r) || nb != tc.nb || nulls != tc.nulls {
			t.Errorf("util.Cvttorunes of (%q, %v) returned %q, %v, %v; expected %q, %v, %v\n",
				tc.p, tc.n, r, nb, nulls, tc.r, tc.nb, tc.nulls)
		}
	}
}

func TestErrorwin1Name(t *testing.T) {
	tt := []struct {
		dir, name string
	}{
		{"", "+Errors"},
		{".", "+Errors"},
		{"/", "/+Errors"},
		{"/home/gopher", "/home/gopher/+Errors"},
		{"/home/gopher/", "/home/gopher/+Errors"},
		{"C:/Users/gopher", "C:/Users/gopher/+Errors"},
		{"C:/Users/gopher/", "C:/Users/gopher/+Errors"},
		{"C:/", "C:/+Errors"},
	}
	for _, tc := range tt {
		name := filepath.ToSlash(errorwin1Name(filepath.FromSlash(tc.dir)))
		if name != tc.name {
			t.Errorf("errorwin1Name(%q) is %q; expected %q", tc.dir, name, tc.name)
		}
	}
}

func TestQuote(t *testing.T) {
	var testCases = []struct {
		s, q string
	}{
		{"", "''"},
		{"Edwood", "Edwood"},
		{"Plan 9", "'Plan 9'"},
		{"Don't", "'Don''t'"},
		{"Don't worry!", "'Don''t worry!'"},
	}
	for _, tc := range testCases {
		q := quote(tc.s)
		if q != tc.q {
			t.Errorf("%q quoted is %q; expected %q\n", tc.s, q, tc.q)
		}
	}
}

func TestSkipbl(t *testing.T) {
	tt := []struct {
		s []rune
		q []rune
	}{
		{nil, nil},
		{[]rune(" \t\n"), nil},
		{[]rune(" \t\nabc"), []rune("abc")},
		{[]rune(" \t\n \t\nabc"), []rune("abc")},
		{[]rune(" \t\nabc \t\nabc"), []rune("abc \t\nabc")},
		{[]rune(" \t\nαβγ \t\nαβγ"), []rune("αβγ \t\nαβγ")},
	}
	for _, tc := range tt {
		q := skipbl(tc.s)
		if !reflect.DeepEqual(q, tc.q) {
			t.Errorf("skipbl(%v) returned %v; expected %v", tc.s, q, tc.q)
		}
	}
}
