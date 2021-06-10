package main

import (
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/internal/edwoodtest"
)

// TestWindowUndoSelection checks observers selection change after undo/redo.
// It tests that selection doesn't change when undoing/redoing
// using nil delta/epsilon, which fixes https://github.com/rjkroege/edwood/issues/230.
func TestWindowUndoSelection(t *testing.T) {
	var (
		word = RuneArray("hello")
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
				q0:  tc.q0,
				q1:  tc.q1,
				oeb: MakeObservableEditableBufferTag(RuneArray("This is an example sentence.\n")),
			},
		}
		w.body.oeb.f.delta = tc.delta
		w.body.oeb.f.epsilon = tc.epsilon
		w.Undo(tc.isundo)
		if w.body.q0 != tc.wantQ0 || w.body.q1 != tc.wantQ1 {
			t.Errorf("%v changed q0, q1 to %v, %v; want %v, %v",
				tc.name, w.body.q0, w.body.q1, tc.wantQ0, tc.wantQ1)
		}
	}
}

func TestSetTag1(t *testing.T) {
	const (
		defaultSuffix = " Del Snarf | Look Edit "
		extraSuffix   = "|fmt g setTag1 Ldef"
	)

	for _, name := range []string{
		"/home/gopher/src/hello.go",
		"/home/ゴーファー/src/エドウード.txt",
		"/home/ゴーファー/src/",
	} {
		configureGlobals()

		display := edwoodtest.NewDisplay()
		w := NewWindow().initHeadless(nil)
		w.display = display
		w.body = Text{
			display: display,
			fr:      &MockFrame{},
			oeb:     MakeObservableEditableBuffer(name, nil),
		}
		w.tag = Text{
			display: display,
			fr:      &MockFrame{},
			oeb:     MakeObservableEditableBuffer("", nil),
		}

		w.setTag1()
		got := string(w.tag.oeb.f.b)
		want := name + defaultSuffix
		if got != want {
			t.Errorf("bad initial tag for file %q:\n got: %q\nwant: %q", name, got, want)
		}

		w.tag.oeb.f.InsertAt(w.tag.oeb.f.Nr(), []rune(extraSuffix))
		w.setTag1()
		got = string(w.tag.oeb.f.b)
		want = name + defaultSuffix + extraSuffix
		if got != want {
			t.Errorf("bad replacement tag for file %q:\n got: %q\nwant: %q", name, got, want)
		}
	}
}

func TestWindowClampAddr(t *testing.T) {
	buf := RuneArray("Hello, 世界")

	for _, tc := range []struct {
		addr, want Range
	}{
		{Range{-1, -1}, Range{0, 0}},
		{Range{100, 100}, Range{buf.nc(), buf.nc()}},
	} {
		w := &Window{
			addr: tc.addr,
			body: Text{
				oeb: MakeObservableEditableBufferTag(buf),
			},
		}
		w.ClampAddr()
		if got := w.addr; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got addr %v; want %v", got, tc.want)
		}
	}
}

func TestWindowParseTag(t *testing.T) {
	for _, tc := range []struct {
		tag      string
		filename string
	}{
		{"/foo/bar.txt Del Snarf | Look", "/foo/bar.txt"},
		{"/foo/bar quux.txt Del Snarf | Look", "/foo/bar quux.txt"},
		{"/foo/bar.txt", "/foo/bar.txt"},
		{"/foo/bar.txt | Look", "/foo/bar.txt"},
		{"/foo/bar.txt Del Snarf\t| Look", "/foo/bar.txt"},
	} {
		w := &Window{
			tag: Text{
				oeb: MakeObservableEditableBufferTag(RuneArray(tc.tag)),
			},
		}
		if got, want := w.ParseTag(), tc.filename; got != want {
			t.Errorf("tag %q has filename %q; want %q", tc.tag, got, want)
		}
	}
}

func TestWindowClearTag(t *testing.T) {
	tag := "/foo bar/test.txt Del Snarf Undo Put | Look |fmt mk"
	want := "/foo bar/test.txt Del Snarf Undo Put |"
	w := &Window{
		tag: Text{
			oeb: MakeObservableEditableBufferTag(RuneArray(tag)),
		},
	}
	w.ClearTag()
	got := w.tag.oeb.f.b.String()
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}
