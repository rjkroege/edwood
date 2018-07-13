package main

import (
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
		r, nb, nulls := cvttorunes(tc.p, tc.n)
		if !reflect.DeepEqual(r, tc.r) || nb != tc.nb || nulls != tc.nulls {
			t.Errorf("cvttorunes of (%q, %v) returned %q, %v, %v; expected %q, %v, %v\n",
				tc.p, tc.n, r, nb, nulls, tc.r, tc.nb, tc.nulls)
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
