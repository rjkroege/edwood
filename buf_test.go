package main

import (
	"testing"
)

// Let's make sure our test fixture has the right form.
func TestBufferDelete(t *testing.T) {
	tab := []struct {
		q0, q1   int
		tb       Buffer
		expected string
	}{
		{0, 5, Buffer([]rune("0123456789")), "56789"},
		{0, 0, Buffer([]rune("0123456789")), "0123456789"},
		{0, 10, Buffer([]rune("0123456789")), ""},
		{1, 5, Buffer([]rune("0123456789")), "056789"},
		{8, 10, Buffer([]rune("0123456789")), "01234567"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Delete(test.q0, test.q1)
		if string(tb) != test.expected {
			t.Errorf("Delete Failed.  Expected %v, got %v", test.expected, string(tb))
		}
	}
}

func TestBufferInsert(t *testing.T) {
	tab := []struct {
		q0       int
		tb       Buffer
		insert   string
		expected string
	}{
		{5, Buffer([]rune("01234")), "56789", "0123456789"},
		{0, Buffer([]rune("56789")), "01234", "0123456789"},
		{1, Buffer([]rune("06789")), "12345", "0123456789"},
		{5, Buffer([]rune("01234")), "56789", "0123456789"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Insert(test.q0, []rune(test.insert))
		if string(tb) != test.expected {
			t.Errorf("Insert Failed.  Expected %v, got %v", test.expected, string(tb))
		}
	}
}

func TestBufferIndexRune(t *testing.T) {
	tt := []struct {
		b Buffer
		r rune
		n int
	}{
		{Buffer(nil), '0', -1},
		{Buffer([]rune("01234")), '0', 0},
		{Buffer([]rune("01234")), '3', 3},
		{Buffer([]rune("αβγ")), 'α', 0},
		{Buffer([]rune("αβγ")), 'γ', 2},
	}
	for _, tc := range tt {
		n := tc.b.IndexRune(tc.r)
		if n != tc.n {
			t.Errorf("IndexRune(%v) for buffer %v returned %v; expected %v",
				tc.r, tc.b, n, tc.n)
		}
	}
}

func TestBufferEqual(t *testing.T) {
	tt := []struct {
		a, b Buffer
		ok   bool
	}{
		{Buffer(nil), Buffer(nil), true},
		{Buffer(nil), Buffer([]rune{}), true},
		{Buffer([]rune{}), Buffer(nil), true},
		{Buffer([]rune("01234")), Buffer([]rune("01234")), true},
		{Buffer([]rune("01234")), Buffer([]rune("01x34")), false},
		{Buffer([]rune("αβγ")), Buffer([]rune("αβγ")), true},
		{Buffer([]rune("αβγ")), Buffer([]rune("αλγ")), false},
	}
	for _, tc := range tt {
		ok := tc.a.Equal(tc.b)
		if ok != tc.ok {
			t.Errorf("Equal(%v) for buffer %v returned %v; expected %v",
				tc.b, tc.a, ok, tc.ok)
		}
	}
}
