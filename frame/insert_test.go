package frame

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

type InsertTestResult struct {
	ppt      image.Point
	resultpt image.Point
	frame    *frameimpl
}

type InsertTest struct {
	name       string
	frame      *frameimpl
	stim       func(*frameimpl) (image.Point, image.Point, *frameimpl)
	nbox       int
	afterboxes []*frbox
	ppt        image.Point
	resultpt   image.Point
}

func (bx InsertTest) Try() interface{} {
	a, b, c := bx.stim(bx.frame)
	return InsertTestResult{a, b, c}
}

func (bx InsertTest) Verify(t *testing.T, prefix string, result interface{}) {
	r := result.(InsertTestResult)

	if got, want := r.ppt, bx.ppt; got != want {
		t.Errorf("%s-%s: running stim ppt got %d but want %d\n", prefix, bx.name, got, want)
	}
	if got, want := r.resultpt, bx.resultpt; got != want {
		t.Errorf("%s-%s: running stim resultpt got %d but want %d\n", prefix, bx.name, got, want)
	}
	// We use the global frame here to make sure that bxscan works as desired.
	// I note in passing that encapsulation here could be improved.
	testcore(t, prefix, bx.name, r.frame, bx.nbox, bx.afterboxes)
}

func mkRu(s string) []rune {
	return bytes.Runes([]byte(s))
}

func TestBxscan(t *testing.T) {
	var b strings.Builder
	b.WriteString("a本")
	for i := 0; i < (57 / 10); i++ {
		b.WriteString("ポ")
	}
	b.WriteString("hello")
	bigstring := b.String()

	comparecore(t, "TestBxscan", []BoxTester{
		InsertTest{
			"1 rune insertion into empty",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				pt1 := image.Pt(10, 15)
				pt2, f := f.bxscan(mkRu("本"), &pt1)
				return pt1, pt2, f
			},
			1,
			[]*frbox{makeBox("本")},
			image.Pt(10, 15),
			image.Pt(20, 15),
		},
		InsertTest{
			"1 rune insertion fits at end of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				pt1 := image.Pt(56, 15)
				pt2, f := f.bxscan(mkRu("本"), &pt1)
				return pt1, pt2, f
			},
			1,
			[]*frbox{makeBox("本")},
			image.Pt(56, 15),
			image.Pt(66, 15),
		},
		InsertTest{
			"1 rune insertion wraps at end of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				pt1 := image.Pt(58, 15)
				pt2, f := f.bxscan(mkRu("本"), &pt1)
				return pt1, pt2, f
			},
			1,
			[]*frbox{makeBox("本")},
			image.Pt(10, 15+13),
			image.Pt(20, 15+13),
		},
		InsertTest{
			"splittable 2 rune insertion at end of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				pt1 := image.Pt(56, 15)
				pt2, f := f.bxscan(mkRu("本a"), &pt1)
				return pt1, pt2, f
			},
			2,
			[]*frbox{makeBox("本"), makeBox("a")},
			image.Pt(56, 15),
			image.Pt(20, 15+13),
		},
		InsertTest{
			"splittable multi-rune rune insertion at start of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				pt1 := image.Pt(10, 15)
				pt2, f := f.bxscan(mkRu(bigstring), &pt1)
				return pt1, pt2, f
			},
			3,
			[]*frbox{makeBox("a本ポポポ"), makeBox("ポポhel"), makeBox("lo")},
			image.Pt(10, 15),
			image.Pt(10+2*10, 15+13+13),
		},
	})
}

type invariants struct {
	topcorner image.Point
	textarea  image.Rectangle
}

func setupFrame(t *testing.T, iv *invariants) Frame {
	t.Helper()
	display := edwoodtest.NewDisplay()

	var textcolors [NumColours]draw.Image

	textcolors[ColBack] = display.AllocImageMix(draw.Paleyellow, draw.White)
	textcolors[ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Darkyellow)
	textcolors[ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Yellowgreen)
	textcolors[ColText] = display.Black()
	textcolors[ColHText] = display.Black()

	// TODO(rjk) Needs to be something with valid metrics for the tests to be be useful.
	name := "helvetica"

	font, err := display.OpenFont(name)
	if err != nil {
		t.Fatalf("can't make mock font %q: %v", name, err)
	}
	fr := NewFrame(iv.textarea, font, display.ScreenImage(), textcolors)

	return fr
}

func simpleInsertShortString(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()

	s := fr.Insert([]rune("ab"), 0)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(0), image.Pt(0, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(1), image.Pt(13, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func multiInsertShortString(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()

	s := fr.Insert([]rune("ab\ncd"), 0)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(0), image.Pt(0, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(1), image.Pt(13, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func gdo(t *testing.T, fr Frame) edwoodtest.GettableDrawOps {
	t.Helper()
	frimpl := fr.(*frameimpl)
	gdo := frimpl.display.(edwoodtest.GettableDrawOps)
	return gdo
}

func nop(t *testing.T, _ Frame, _ *invariants) {
	t.Log("hi from nop")
}

// TODO(rjk): Conceivably the bxscan test can go away once I have written this
// to my satisfaction?
// TODO(rjk): This can be restructured to look like the edit tests: stim
// function, list of ops, post condition.
func TestInsert(t *testing.T) {
	iv := &invariants{
		topcorner: image.Pt(20, 10),
		// Remember that the "screen image" is 800,600.
		textarea: image.Rect(20, 10, 400, 500),
	}

	*validate = true

	tests := []struct {
		name string
		fn   func(t *testing.T, fr Frame, iv *invariants)
		want []string
	}{
		// TODO(rjk): Test cases
		// 1. add a string that doesn't fit on one line
		// 2. add a string that causes a line to spill over
		// 3. add a newline
		// 4. start a line with a tab, insert before the tab
		// 5. split a line
		// multiple boxes, few lines, insert line before, into multi-line,
		// split a multiline
		// join lines into a multiline (delete test)
		// remove a line before a multiline
		// remove enough of a multiline for it to become a single line
		{
			name: "setupFrame",
			fn:   nop,
			want: []string{
				"White <- draw r: (0,0)-(0,10) src: mix(Paleyellow,White) mask mix(Paleyellow,White) p1: (0,0)",
				"Transparent <- draw r: (0,0)-(0,10) src: transparent mask transparent p1: (0,0)",
				"Transparent <- draw r: (0,0)-(0,10) src: opaque mask opaque p1: (0,0)",
				"Transparent <- draw r: (0,0)-(0,0) src: opaque mask opaque p1: (0,0)",
				"Transparent <- draw r: (0,10)-(0,10) src: opaque mask opaque p1: (0,0)",
			},
		},
		{
			// A short string that fits on one line without breaking.
			name: "simpleInsertShortString",
			fn:   simpleInsertShortString,
			want: []string{
				// TODO(rjk): Where do we draw a background for the text area.
				"screen-800x600 <- draw r: (20,10)-(46,20) src: mix(Paleyellow,White) mask mix(Paleyellow,White) p1: (0,0)",
				`screen-800x600 <- draw-chars "ab" atpoint: (20,10) font: /lib/font/edwood.font fill: black sp: (0,0)`,
			},
		},
		{
			// A multi-line string
			name: "multiInsertShortString",
			fn:   multiInsertShortString,
			want: []string{
				// TODO(rjk): Where do we draw a background for the text area.
				"screen-800x600 <- draw r: (20,10)-(400,20) src: mix(Paleyellow,White) mask mix(Paleyellow,White) p1: (0,0)",
				"screen-800x600 <- draw r: (20,20)-(46,30) src: mix(Paleyellow,White) mask mix(Paleyellow,White) p1: (0,0)",
				`screen-800x600 <- draw-chars "ab" atpoint: (20,10) font: /lib/font/edwood.font fill: black sp: (0,0)`,
				`screen-800x600 <- draw-chars "cd" atpoint: (20,20) font: /lib/font/edwood.font fill: black sp: (0,0)`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fr := setupFrame(t, iv)

			// TODO(rjk): validate here

			tc.fn(t, fr, iv)

			// TODO(rjk): validate here

			// Peek inside.
			got := gdo(t, fr).DrawOps()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// TODO(rjk): how to document these things? I'd like little box diagrams
	// how can I make little box diagrams? (HTML? OmniFocus?)
	// HTML three-column: name, before, after (grid layout?)
	// nb: can imagine generating the code from the diagrams? (that would be rsc-ian.)
}
