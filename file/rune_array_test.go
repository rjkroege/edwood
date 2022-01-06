package file

import (
	"testing"
)

// Let's make sure our test fixture has the right form.
func TestBufferDelete(t *testing.T) {
	tab := []struct {
		q0, q1   int
		tb       RuneArray
		expected string
	}{
		{0, 5, RuneArray([]rune("0123456789")), "56789"},
		{0, 0, RuneArray([]rune("0123456789")), "0123456789"},
		{0, 10, RuneArray([]rune("0123456789")), ""},
		{1, 5, RuneArray([]rune("0123456789")), "056789"},
		{8, 10, RuneArray([]rune("0123456789")), "01234567"},
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
		tb       RuneArray
		insert   string
		expected string
	}{
		{5, RuneArray([]rune("01234")), "56789", "0123456789"},
		{0, RuneArray([]rune("56789")), "01234", "0123456789"},
		{1, RuneArray([]rune("06789")), "12345", "0123456789"},
		{5, RuneArray([]rune("01234")), "56789", "0123456789"},
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
		b RuneArray
		r rune
		n int
	}{
		{RuneArray(nil), '0', -1},
		{RuneArray([]rune("01234")), '0', 0},
		{RuneArray([]rune("01234")), '3', 3},
		{RuneArray([]rune("αβγ")), 'α', 0},
		{RuneArray([]rune("αβγ")), 'γ', 2},
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
		a, b RuneArray
		ok   bool
	}{
		{RuneArray(nil), RuneArray(nil), true},
		{RuneArray(nil), RuneArray([]rune{}), true},
		{RuneArray([]rune{}), RuneArray(nil), true},
		{RuneArray([]rune("01234")), RuneArray([]rune("01234")), true},
		{RuneArray([]rune("01234")), RuneArray([]rune("01x34")), false},
		{RuneArray([]rune("αβγ")), RuneArray([]rune("αβγ")), true},
		{RuneArray([]rune("αβγ")), RuneArray([]rune("αλγ")), false},
	}
	for _, tc := range tt {
		ok := tc.a.Equal(tc.b)
		if ok != tc.ok {
			t.Errorf("Equal(%v) for buffer %v returned %v; expected %v",
				tc.b, tc.a, ok, tc.ok)
		}
	}
}
