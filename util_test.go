package main

import (
	"os"
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

func TestMousescrollsize(t *testing.T) {
	const key = "mousescrollsize"
	mss, ok := os.LookupEnv(key)
	if ok {
		defer os.Setenv(key, mss)
	} else {
		defer os.Unsetenv(key)
	}

	tt := []struct {
		s        string
		maxlines int
		n        int
	}{
		{"", 200, 1},
		{"0", 200, 1},
		{"-1", 200, 1},
		{"-42", 200, 1},
		{"two", 200, 1},
		{"1", 200, 1},
		{"42", 200, 42},
		{"123", 200, 123},
		{"%", 200, 1},
		{"0%", 200, 1},
		{"-1%", 200, 1},
		{"-42%", 200, 1},
		{"five%", 200, 1},
		{"123%", 200, 200},
		{"10%", 200, 20},
		{"100%", 200, 200},
	}
	for _, tc := range tt {
		os.Setenv(key, tc.s)
		scrollLines = 0
		scrollPercent = 0
		n := _mousescrollsize(always{}, tc.maxlines)
		if n != tc.n {
			t.Errorf("mousescrollsize of %v for %v lines is %v; expected %v",
				tc.s, tc.maxlines, n, tc.n)
		}
	}
}

type always struct{}

func (a always) Do(f func()) { f() }

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
