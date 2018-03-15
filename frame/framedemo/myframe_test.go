package main

import (
	"reflect"
	"testing"
)

func TestMyframeInsert(t *testing.T) {

	mf := Myframe{}
	mf.buffer = make([]rune, 0)

	mf.Insert('a')

	if mf.cursor != 1 {
		t.Errorf("didn't update cursor. want 1, got %d", mf.cursor)
	}
	if mf.buffer[0] != 'a' {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}

	mf.Insert('m')
	if mf.cursor != 2 {
		t.Errorf("didn't update cursor. want 2, got %d", mf.cursor)
	}
	if mf.buffer[0] != 'a' {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}
	if mf.buffer[1] != 'm' {
		t.Errorf("didn't update buffer, want m but got %#v", mf.buffer)
	}

	mf.cursor = 1
	mf.Insert('c')
	if mf.cursor != 2 {
		t.Errorf("didn't update cursor. want 2, got %d", mf.cursor)
	}
	if mf.buffer[0] != 'a' {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}
	if mf.buffer[1] != 'c' {
		t.Errorf("didn't update buffer, want c but got %#v", mf.buffer)
	}
	if mf.buffer[2] != 'm' {
		t.Errorf("didn't update buffer, want m but got %#v", mf.buffer)
	}
}

func TestMyframeDelete(t *testing.T) {

	mf := Myframe{}
	mf.buffer = []rune{'x', 'a', 'c', 'x', 'm', 'e', 'x'}

	mf.Delete()
	if mf.cursor != 0 {
		t.Errorf("didn't update cursor. want 0, got %d", mf.cursor)
	}
	if !reflect.DeepEqual(mf.buffer, []rune{'x', 'a', 'c', 'x', 'm', 'e', 'x'}) {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}

	mf.Right()
	mf.Delete()
	if mf.cursor != 0 {
		t.Errorf("didn't update cursor. want 0, got %d", mf.cursor)
	}
	if !reflect.DeepEqual(mf.buffer, []rune{'a', 'c', 'x', 'm', 'e', 'x'}) {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}

	mf.Right()
	mf.Right()
	mf.Right()
	mf.Delete()
	if mf.cursor != 2 {
		t.Errorf("didn't update cursor. want 2, got %d", mf.cursor)
	}
	if !reflect.DeepEqual(mf.buffer, []rune{'a', 'c', 'm', 'e', 'x'}) {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}

	mf.Right()
	mf.Right()
	mf.Right()
	mf.Delete()
	if mf.cursor != 4 {
		t.Errorf("didn't update cursor. want 4, got %d", mf.cursor)
	}
	if !reflect.DeepEqual(mf.buffer, []rune{'a', 'c', 'm', 'e'}) {
		t.Errorf("didn't update buffer, want a but got %#v", mf.buffer)
	}
}
