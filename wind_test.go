package main

import (
	"reflect"
	"testing"
)

// TestWindowUndoSelection checks text selection change after undo/redo.
// It tests that selection doesn't change when undoing/redoing
// using nil delta/epsilon, which fixes https://github.com/rjkroege/edwood/issues/230.
func TestWindowUndoSelection(t *testing.T) {
	var (
		word = Buffer("hello")
		p0   = 3
		undo = &Undo{
			t:   Insert,
			buf: word,
			p0:  p0,
			n:   word.nc(),
		}
	)
	for _, tc := range []struct {
		name           string
		isundo         bool
		q0, q1         int
		wantQ0, wantQ1 int
		delta, epsilon []*Undo
	}{
		{"undo", true, 14, 17, p0, p0 + word.nc(), []*Undo{undo}, nil},
		{"redo", false, 14, 17, p0, p0 + word.nc(), nil, []*Undo{undo}},
		{"undo (nil delta)", true, 14, 17, 14, 17, nil, nil},
		{"redo (nil epsilon)", false, 14, 17, 14, 17, nil, nil},
	} {
		w := &Window{
			body: Text{
				q0: tc.q0,
				q1: tc.q1,
				file: &File{
					b:       Buffer("This is an example sentence.\n"),
					delta:   tc.delta,
					epsilon: tc.epsilon,
				},
			},
		}
		w.Undo(tc.isundo)
		if w.body.q0 != tc.wantQ0 || w.body.q1 != tc.wantQ1 {
			t.Errorf("%v changed q0, q1 to %v, %v; want %v, %v",
				tc.name, w.body.q0, w.body.q1, tc.wantQ0, tc.wantQ1)
		}
	}
}

func TestWindowClampAddr(t *testing.T) {
	buf := Buffer("Hello, 世界")

	for _, tc := range []struct {
		addr, want Range
	}{
		{Range{-1, -1}, Range{0, 0}},
		{Range{100, 100}, Range{buf.nc(), buf.nc()}},
	} {
		w := &Window{
			addr: tc.addr,
			body: Text{
				file: &File{
					b: buf,
				},
			},
		}
		w.ClampAddr()
		if got := w.addr; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got addr %v; want %v", got, tc.want)
		}
	}
}
