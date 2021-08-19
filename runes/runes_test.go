package runes

import (
	"reflect"
	"testing"
)

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

var indexRuneTests = []struct {
	s string
	r rune
	n int
}{
	{"", 'x', -1},
	{"x", 'x', 0},
	{"abcdef", 'a', 0},
	{"abcdef", 'd', 3},
	{"abcdef", 'f', 5},
	{"abcdef", 'x', -1},
	{"私はガラスを食べる", '私', 0},
	{"私はガラスを食べる", 'を', 5},
	{"私はガラスを食べる", 'る', 8},
	{"私はガラスを食べる", 'α', -1},
}

func TestIndexRune(t *testing.T) {
	for _, tc := range indexRuneTests {
		n := IndexRune([]rune(tc.s), tc.r)
		if n != tc.n {
			t.Errorf("IndexRune(%q, %q) is %v; expected %v", tc.s, tc.r, n, tc.n)
		}
	}
}

func TestContainsRune(t *testing.T) {
	for _, tc := range indexRuneTests {
		ok := ContainsRune([]rune(tc.s), tc.r)
		if want := tc.n >= 0; ok != want {
			t.Errorf("ContainsRune(%q, %q) is %v; expected %v", tc.s, tc.r, ok, want)
		}
	}
}

func TestEqual(t *testing.T) {
	tt := []struct {
		a, b string
		ok   bool
	}{
		{"", "", true},
		{"a", "", false},
		{"", "a", false},
		{"a", "a", true},
		{"ab", "ab", true},
		{"abc", "abc", true},
		{"abc", "axc", false},
		{"axc", "abc", false},
		{"私はガラスを食べる", "私はガラスを食べる", true},
		{"私はガラスを食べる", "私はケーキを食べる", false},
		{"私はケーキを食べる", "私はガラスを食べる", false},
		{"私はガラスを食べる", "私はpieを食べる", false},
		{"私はpieを食べる", "私はガラスを食べる", false},
		{"私はpieを食べる", "私はpieを食べる", true},
	}
	for _, tc := range tt {
		ok := Equal([]rune(tc.a), []rune(tc.b))
		if ok != tc.ok {
			t.Errorf("HasPrefix(%q, %q) returned %v; expected %v", tc.a, tc.b, ok, tc.ok)
		}
	}
}

func TestTrimLeft(t *testing.T) {
	tt := []struct {
		s      []rune
		cutset string
		q      []rune
	}{
		{nil, "", nil},
		{[]rune(""), "", nil},
		{[]rune("abc"), "", []rune("abc")},
		{[]rune("abc"), "a", []rune("bc")},
		{[]rune("abc"), "b", []rune("abc")},
		{[]rune("abc"), "c", []rune("abc")},
		{[]rune("aaabc"), "a", []rune("bc")},
		{[]rune("aaabbbccc"), "", []rune("aaabbbccc")},
		{[]rune("aaabbbccc"), "a", []rune("bbbccc")},
		{[]rune("aaabbbccc"), "b", []rune("aaabbbccc")},
		{[]rune("aaabbbccc"), "c", []rune("aaabbbccc")},
		{[]rune("aaabbbccc"), "ac", []rune("bbbccc")},
		{[]rune("aaabbbccc"), "ab", []rune("ccc")},
		{[]rune("aaabbbccc"), "cb", []rune("aaabbbccc")},
		{[]rune("aaabbbccc"), "cba", nil},
		{[]rune("テテテススストトト"), "", []rune("テテテススストトト")},
		{[]rune("テテテススストトト"), "テ", []rune("ススストトト")},
		{[]rune("テテテススストトト"), "ス", []rune("テテテススストトト")},
		{[]rune("テテテススストトト"), "ト", []rune("テテテススストトト")},
		{[]rune("テテテススストトト"), "テト", []rune("ススストトト")},
		{[]rune("テテテススストトト"), "テス", []rune("トトト")},
		{[]rune("テテテススストトト"), "トス", []rune("テテテススストトト")},
		{[]rune("テテテススストトト"), "トステ", nil},
	}
	for _, tc := range tt {
		q := TrimLeft(tc.s, tc.cutset)
		if !reflect.DeepEqual(q, tc.q) {
			t.Errorf("TrimLeft(%q, %q) is %q; expected %q", tc.s, tc.cutset, q, tc.q)
		}
	}
}
