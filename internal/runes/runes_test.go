package runes

import "testing"

func TestIndex(t *testing.T) {
	tt := []struct {
		s, sep string
		n      int
	}{
		{"foobar", "", 0},
		{"", "abc", -1},
		{"abc", "abcd", -1},
		{"x", "x", 0},
		{"fooabcbar", "foo", 0},
		{"fooabcbar", "abc", 3},
		{"fooabcbar", "xyz", -1},
		{"fooabcbar", "bar", 6},
		{"fooabcbar", "r", 8},
		{"abcfooabc", "abc", 0},
		{"私はガラスを食べる", "私は", 0},
		{"私はガラスを食べる", "ガラス", 2},
		{"私はガラスを食べる", "る", 8},
		{"私はガラスを食べる", "ケーキ", -1},
		{"私は私", "私", 0},
	}
	for _, tc := range tt {
		n := Index([]rune(tc.s), []rune(tc.sep))
		if n != tc.n {
			t.Errorf("Index(%q, %q) is %v; expected %v", tc.s, tc.sep, n, tc.n)
		}
	}
}

func TestHasPrefix(t *testing.T) {
	tt := []struct {
		s, prefix string
		ok        bool
	}{
		{"", "", true},
		{"", "foo", false},
		{"foobar", "foo", true},
		{"abc", "abcd", false},
		{"fooabc", "abc", false},
		{"私はガラス", "私はガラスを食べる", false},
		{"私はガラスを食べる", "私は", true},
		{"私はガラスを食べる", "ガラス", false},
	}
	for _, tc := range tt {
		ok := HasPrefix([]rune(tc.s), []rune(tc.prefix))
		if ok != tc.ok {
			t.Errorf("HasPrefix(%q, %q) returned %v; expected %v",
				tc.s, tc.prefix, ok, tc.ok)
		}
	}
}
