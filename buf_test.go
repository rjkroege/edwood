package main

import (
	"testing"
)

// Let's make sure our test fixture has the right form.
func TestBufferDelete(t *testing.T) {
	tab := []struct {
		q0, q1   uint
		tb       Buffer
		expected string
	}{
		{0, 5, Buffer{[]rune("0123456789")}, "56789"},
		{0, 0, Buffer{[]rune("0123456789")}, "0123456789"},
		{0, 10, Buffer{[]rune("0123456789")}, ""},
		{1, 5, Buffer{[]rune("0123456789")}, "056789"},
		{8, 10, Buffer{[]rune("0123456789")}, "01234567"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Delete(test.q0, test.q1)
		if string(tb.buf) != test.expected {
			t.Errorf("Delete Failed.  Expected %v, got %v", test.expected, string(tb.buf))
		}
	}
}

func TestBufferInsert(t *testing.T) {
	tab := []struct {
		q0       uint
		tb       Buffer
		insert   string
		expected string
	}{
		{5, Buffer{[]rune("01234")}, "56789", "0123456789"},
		{0, Buffer{[]rune("56789")}, "01234", "0123456789"},
		{1, Buffer{[]rune("06789")}, "12345", "0123456789"},
		{5, Buffer{[]rune("01234")}, "56789", "0123456789"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Insert(test.q0, []rune(test.insert))
		if string(tb.buf) != test.expected {
			t.Errorf("Insert Failed.  Expected %v, got %v", test.expected, string(tb.buf))
		}
	}
}
