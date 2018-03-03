package main

import (
	"fmt"
	"testing"
)

// TestText implements texter for elog tests.
type TextMock struct {
	q0, q1 uint
	buf    []rune
}

func (t TextMock) Constrain(q0, q1 uint) (p0, p1 uint) {
	p0 = minu(q0, uint(len(t.buf)))
	p1 = minu(q1, uint(len(t.buf)))
	return p0, p1
}

func (t *TextMock) Delete(q0, q1 uint, tofile bool) {
	_ = tofile
	if q0 > uint(len(t.buf)) || q1 > uint(len(t.buf)) {
		panic("Out-of-range Delete")
	}
	copy(t.buf[q0:], t.buf[q1:])
	t.buf = t.buf[:uint(len(t.buf))-(q1-q0)] // Reslice to length
}

// Let's make sure our test fixture has the right form.
func TestDelete(t *testing.T) {
	tab := []struct {
		q0, q1   uint
		tb       TextMock
		expected string
	}{
		{0, 5, TextMock{0, 0, []rune("0123456789")}, "56789"},
		{0, 0, TextMock{0, 0, []rune("0123456789")}, "0123456789"},
		{0, 10, TextMock{0, 0, []rune("0123456789")}, ""},
		{1, 5, TextMock{0, 0, []rune("0123456789")}, "056789"},
		{8, 10, TextMock{0, 0, []rune("0123456789")}, "01234567"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Delete(test.q0, test.q1, true)
		if string(tb.buf) != test.expected {
			t.Errorf("Delete Failed.  Expected %v, got %v", test.expected, string(tb.buf))
		}
	}
}

func (t *TextMock) Insert(q0 uint, r []rune, tofile bool) {
	_ = tofile
	if q0 > uint(len(t.buf)) {
		panic("Out of range insertion")
	}
	t.buf = append(t.buf[:q0], append(r, t.buf[q0:]...)...)
}

func TestInsert(t *testing.T) {
	tab := []struct {
		q0       uint
		tb       TextMock
		insert   string
		expected string
	}{
		{5, TextMock{0, 0, []rune("01234")}, "56789", "0123456789"},
		{0, TextMock{0, 0, []rune("56789")}, "01234", "0123456789"},
		{1, TextMock{0, 0, []rune("06789")}, "12345", "0123456789"},
		{5, TextMock{0, 0, []rune("01234")}, "56789", "0123456789"},
	}
	for _, test := range tab {
		tb := test.tb
		tb.Insert(test.q0, []rune(test.insert), true)
		if string(tb.buf) != test.expected {
			t.Errorf("Insert Failed.  Expected %v, got %v", test.expected, string(tb.buf))
		}
	}
}

func (t *TextMock) Q0() uint      { return t.q0 }
func (t *TextMock) SetQ0(q0 uint) { t.q0 = q0 }
func (t *TextMock) Q1() uint      { return t.q1 }
func (t *TextMock) SetQ1(q1 uint) { t.q1 = q1 }

func TestElogInsertDelete(t *testing.T) {
	t0 := []rune("This")
	t1 := []rune(" is")
	t2 := []rune(" a test")

	e := MakeElog()

	e.Insert(0, t0)
	e.Insert(0, t1)
	e.Insert(0, t2)

	if len(e.log) != 2 {
		t.Errorf("Insertions should have catenated")
	}
	if e.log[0].t != Null {
		t.Errorf("Sentinel displaced")
	}
	if string(e.log[1].r) != "This is a test" {
		t.Errorf("Failed to catenate properly")
	}

	e.Reset()
	e.Insert(0, t0)
	e.Insert(0, t1)
	e.Insert(1, t2)
	if len(e.log) != 3 {
		t.Errorf("Expected 3 elements, have %d", len(e.log))
	}
	if string(e.log[1].r) != "This is" {
		t.Errorf("Failed to catenate properly.  Expected 'This is', got '%s'", string(e.log[1].r))
	}
	if string(e.log[2].r) != " a test" {
		t.Errorf("Failed to catenate properly")
	}
	e.Insert(1, t2)
	if string(e.log[2].r) != " a test a test" {
		t.Errorf("Failed to catenate properly.  Expected ' a test a test', got '%s'", string(e.log[1].r))
	}

	e.Delete(1, 5)
	if len(e.log) != 4 {
		t.Errorf("Expected 4 elements, have %d", len(e.log))
	}
	e.Delete(5, 5)
	if len(e.log) != 4 {
		fmt.Println(e)
		t.Errorf("Expected 4 elements, have %d", len(e.log))
	}
}

func TestApply(t *testing.T) {
	tab := []struct {
		tb       TextMock
		elog     Elog
		expected string
	}{
		{TextMock{0, 0, []rune{}},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Insert, 0, 0, []rune("0123456789")},
			}, false},
			"0123456789"},

		{TextMock{0, 0, []rune("0123456789")},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Delete, 0, 5, []rune{}},
			}, false},
			"56789"},

		{TextMock{0, 0, []rune("XXX56789")},
			Elog{[]ElogOperation{
				{Null, 0, 0, []rune{}},
				{Insert, 0, 0, []rune("01234")},
				{Delete, 0, 3, []rune{}},
			}, false},
			"0123456789"},

		{TextMock{0, 0, []rune("XXX56789")},
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
