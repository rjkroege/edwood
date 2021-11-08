package sam

import (
	"fmt"
	"testing"
)

// Let's make sure our test fixture has the right form.
func TestDelete(t *testing.T) {
	tab := []struct {
		q0, q1   int
		tb       TextBuffer
		expected string
	}{
		{0, 5, TextBuffer{0, 0, []rune("0123456789")}, "56789"},
		{0, 0, TextBuffer{0, 0, []rune("0123456789")}, "0123456789"},
		{0, 10, TextBuffer{0, 0, []rune("0123456789")}, ""},
		{1, 5, TextBuffer{0, 0, []rune("0123456789")}, "056789"},
		{8, 10, TextBuffer{0, 0, []rune("0123456789")}, "01234567"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Delete(test.q0, test.q1, true)
		if string(tb.buf) != test.expected {
			t.Errorf("Delete Failed.  Expected %v, got %v", test.expected, string(tb.buf))
		}
	}
}

func TestInsert(t *testing.T) {
	tab := []struct {
		q0       int
		tb       TextBuffer
		insert   string
		expected string
	}{
		{5, TextBuffer{0, 0, []rune("01234")}, "56789", "0123456789"},
		{0, TextBuffer{0, 0, []rune("56789")}, "01234", "0123456789"},
		{1, TextBuffer{0, 0, []rune("06789")}, "12345", "0123456789"},
		{5, TextBuffer{0, 0, []rune("01234")}, "56789", "0123456789"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Insert(test.q0, []rune(test.insert), true)
		if string(tb.buf) != test.expected {
			t.Errorf("Insert Failed.  Expected %v, got %v", test.expected, string(tb.buf))
		}
	}
}

func TestElogInsertDelete(t *testing.T) {
	t0 := []rune("This")
	t1 := []rune(" is")
	t2 := []rune(" a test")

	e := MakeElog()

	e.Insert(0, t0)
	e.Insert(0, t1)
	e.Insert(0, t2)

	if len(e.Log) != 2 {
		t.Errorf("Insertions should have catenated")
	}
	if e.Log[0].T != Null {
		t.Errorf("Sentinel displaced")
	}
	if string(e.Log[1].r) != "This is a test" {
		t.Errorf("Failed to catenate properly")
	}

	e.Reset()
	e.Insert(0, t0)
	e.Insert(0, t1)
	e.Insert(1, t2)
	if len(e.Log) != 3 {
		t.Errorf("Expected 3 elements, have %d", len(e.Log))
	}
	if string(e.Log[1].r) != "This is" {
		t.Errorf("Failed to catenate properly.  Expected 'This is', got '%s'", string(e.Log[1].r))
	}
	if string(e.Log[2].r) != " a test" {
		t.Errorf("Failed to catenate properly")
	}
	e.Insert(1, t2)
	if string(e.Log[2].r) != " a test a test" {
		t.Errorf("Failed to catenate properly.  Expected ' a test a test', got '%s'", string(e.Log[1].r))
	}

	e.Delete(1, 5)
	if len(e.Log) != 4 {
		t.Errorf("Expected 4 elements, have %d", len(e.Log))
	}
	e.Delete(5, 5)
	if len(e.Log) != 4 {
		fmt.Println(e)
		t.Errorf("Expected 4 elements, have %d", len(e.Log))
	}
}

func TestApply(t *testing.T) {
	tab := []struct {
		tb       TextBuffer
		elog     Elog
		expected string
	}{
		{TextBuffer{0, 0, []rune{}},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Insert, 0, 0, []rune("0123456789")},
			}, false},
			"0123456789"},

		{TextBuffer{0, 0, []rune("0123456789")},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Delete, 0, 5, []rune{}},
			}, false},
			"56789"},

		{TextBuffer{0, 0, []rune("XXX56789")},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Insert, 0, 0, []rune("01234")},
				{Delete, 0, 3, []rune{}},
			}, false},
			"0123456789"},

		{TextBuffer{0, 0, []rune("XXX56789")},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Replace, 0, 3, []rune("01234")},
			}, false},
			"0123456789"},
	}

	for i, test := range tab {
		tb := test.tb
		test.elog.Apply(&tb)
		if string(tb.buf) != test.expected {
			t.Errorf("Apply Failed case %d: Expected %v, got %v", i, test.expected, string(tb.buf))
		}
	}
}
