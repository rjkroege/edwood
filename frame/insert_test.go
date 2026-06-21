package frame

import (
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
	t.Helper()
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

func makereplicatedstring(c int) string {
	var b strings.Builder
	b.WriteString("a本")
	for i := 0; i < c; i++ {
		b.WriteString("ポ")
	}
	b.WriteString("hello")
	return b.String()
}

func TestBxscan(t *testing.T) {
	bigstring := makereplicatedstring(57 / 10)

	comparecore(t, "TestBxscan", []BoxTester{
		InsertTest{
			"1 rune insertion into empty",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte("本"), 0, 0)
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
				box:               []*frbox{makeBox("abc")},
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte("本"), 4, 1)
			},
			1,
			[]*frbox{makeBox("本")},
			image.Pt(10+3*10, 15),
			image.Pt(10+4*10, 15),
		},
		InsertTest{
			"1 rune insertion wraps at end of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
				box:               []*frbox{makeBox("abcde")},
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte("本"), 5, 1)
			},
			1,
			[]*frbox{makeBox("本")},
			image.Pt(10+5*10, 15),
			image.Pt(10+1*10, 15+13),
		},
		InsertTest{
			"splittable 2 rune insertion at end of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
				box:               []*frbox{makeBox("abcd")},
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte("本a"), 4, 1)
			},
			2,
			[]*frbox{
				makeBox("本"),
				makeBox("a"),
			},
			image.Pt(10+4*10, 15),
			image.Pt(10+1*10, 15+13),
		},
		InsertTest{
			"splittable multi-rune rune insertion at start of line",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte(bigstring), 0, 0)
			},
			3,
			[]*frbox{makeBox("a本ポポポ"), makeBox("ポポhel"), makeBox("lo")},
			image.Pt(10, 15),
			image.Pt(10+2*10, 15+13+13),
		},
		InsertTest{
			"tabs and newlines placed in dedicated boxes",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
				maxtab:            8,
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte("\ta\n"), 0, 0)
			},
			3,
			[]*frbox{makeBox("\t"), makeBox("a"), makeBox("\n")},
			image.Pt(10, 15),
			image.Pt(10, 15+13),
		},
		InsertTest{
			"a newline is inserted at the point point between boxes",
			&frameimpl{
				font:              mockFont(),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
				maxtab:            8,
				box: []*frbox{
					makeBox("abcde"),
					makeBox("123"),
				},
			},
			func(f *frameimpl) (image.Point, image.Point, *frameimpl) {
				return f.bxscan([]byte("\n"), 5, 1)
			},
			1,
			[]*frbox{
				makeBox("\n"),
			},
			image.Pt(10+5*10, 15),
			image.Pt(10, 15+13),
		},
	})
}

type invariants struct {
	topcorner image.Point
	textarea  image.Rectangle
}

// setupFrame makes a Frame object for testing with a pixel-rendering Display.
func setupFrame(t *testing.T, iv *invariants) Frame {
	t.Helper()
	display := edwoodtest.NewDisplayWithDPI(iv.textarea)

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

func simpleInsertShortString(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("ab"), 0)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(0), image.Pt(0, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(1), image.Pt(8, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func multiInsertShortString(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("ab\ncd"), 0)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(0), image.Pt(0, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fr.Ptofchar(1), image.Pt(8, 0).Add(iv.topcorner); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertLongLine(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("ab\ncd\nef"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	bigstring := makereplicatedstring(iv.textarea.Dx() / 8)
	s := fr.Insert([]rune(bigstring), 4)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertIntoLongLine(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("ab\ncd\nef"), 0)
	bigstring := makereplicatedstring(iv.textarea.Dx() / 8)
	s := fr.Insert([]rune(bigstring), 4)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	fr.Insert([]rune("X"), 4)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertTabAndChar(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("ab"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()
	s := fr.Insert([]rune("\t"), 1)
	fr.Insert([]rune("X"), 1)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertPastEnd(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()
	s := fr.Insert([]rune(makereplicatedstring(6)), 0)

	// I would have expected that this should be true?
	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func splitWrappedLine(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	rss := []rune(makereplicatedstring(6))
	fr.Insert(rss, 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune{'\n'}, 3)

	// I would have expected that this should be true?
	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertForcesWrap(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n4ij"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune{'X'}, 2)

	// This is a
	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func appendAtEnd(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n4ij"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune{'X', 'X'}, len("0ab\n1cd\n2ef\n3gh\n4ij"))

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func appendHangingLongAtEnd(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0\n1\n2\n3\n4\n"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("XXX"), len("0\n1\n2\n3\n4"))

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertWrappedThatForcesRipple(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0\n1\n2\n3b\n4\n"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("ijXX"), len("0\n1\n2\n3"))

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertPushesBlankLineOffEnd(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n\n"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("X"), 1)

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertsRippledNewLine(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("\n"), len("0ab\n1cd\n2ef\n"))

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertForcesRippleOfWrapped(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0ab1cd2ef3gh4ij5"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune("ABC"), 0)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// insertAtExactWrapBoundary inserts a character at position 3, which is exactly
// at the end of the first soft-wrapped visual line (3 chars × 13 px = 39 px).
// In the aligned frame the inserted character has 0 px of space remaining and
// must wrap; in the non-aligned frame it has 1 px remaining, which also forces
// a wrap. Either way the fill left on line 1 differs: 1 px vs 0 px.
func insertAtExactWrapBoundary(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n4ij"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune{'X'}, 3)

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// insertExactlyFillsAlignedLine inserts the character that makes the first
// visual line exactly reach the frame's right edge (from 2 chars to 3 chars =
// 39 px in the aligned frame, 39 px + 1 px gap in the non-aligned frame).
// No wrap should be triggered by this insertion.
func insertExactlyFillsAlignedLine(t *testing.T, fr Frame, iv *invariants, name string) {
	t.Helper()

	fr.Insert([]rune("0a\n1cd\n2ef\n3gh\n4ij"), 0)
	snapBeforePNG(t, fr, name)
	gdo(t, fr).Clear()

	s := fr.Insert([]rune{'X'}, 2)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func nop(t *testing.T, fr Frame, _ *invariants, name string) {
	snapBeforePNG(t, fr, name)
}

// TODO(rjk): Conceivably the bxscan test can go away once I have written
// this to my satisfaction?
func TestInsert(t *testing.T) {
	iv := &invariants{
		topcorner: image.Pt(20, 10),
	}

	*validate = true

	tests := []struct {
		name        string
		fn          func(t *testing.T, fr Frame, iv *invariants, name string)
		want        []string
		textarea    image.Rectangle
		knowntofail bool
	}{
		// TODO(rjk): Test cases
		// 3. add a newline after one already there.
		{
			name: "setupFrame",
			fn:   nop,
			want: []string{
				"fill (0,0)-(6,15) [-,-],[-,1]",
				"fill (0,0)-(6,15) [-,-],[-,1]",
				"fill (2,0)-(4,15) [-,-],[-,1]",
				"fill (0,0)-(6,6) [-,-],[-,-]",
				"fill (0,9)-(6,15) [-,-],[-,-]",
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		{
			// Inserts ab at the start of the line with no wrapping or ripple.
			name: "simpleInsertShortString",
			fn:   simpleInsertShortString,
			want: []string{
				"fill (20,10)-(36,25) [0,0],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,10) [0,0] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		{
			// A short but newline containing string that fits inserted at the start.
			name: "multiInsertShortString",
			fn:   multiInsertShortString,
			want: []string{
				"fill (20,10)-(400,25) [0,0],[-,1]",
				"fill (20,25)-(36,40) [0,1],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "cd" atpoint: (20,25) [0,1] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		{
			// A long line inserted. Requires wrapping the inserted line and rippling
			// the remaining text.
			name: "insertLongLine",
			fn:   insertLongLine,
			want: []string{
				"blit (20,40)-(36,55) [0,2],[2,1], to (20,55)-(36,70) [0,3],[2,1]",
				"fill (92,40)-(400,55) [9,2],[-,1]",
				"blit (28,25)-(36,40) [1,1],[1,1], to (84,40)-(92,55) [8,2],[1,1]",
				"fill (28,25)-(400,40) [1,1],[-,1]",
				"fill (20,40)-(84,55) [0,2],[8,1]",
				`screen-800x600 <- string "a本ポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポ" atpoint: (28,25) [1,1] fill: black`,
				`screen-800x600 <- string "ポポポhello" atpoint: (20,40) [0,2] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		// one
		{
			// Insert into a long line
			name: "insertIntoLongLine",
			fn:   insertIntoLongLine,
			want: []string{
				// This first blit is a nop.
				"blit (20,55)-(36,70) [0,3],[2,1], to (20,55)-(36,70) [0,3],[2,1]",
				"fill (100,40)-(400,55) [10,2],[-,1]",
				"blit (20,40)-(92,55) [0,2],[9,1], to (28,40)-(100,55) [1,2],[9,1]",
				"blit (388,25)-(396,40) [46,1],[1,1], to (20,40)-(28,55) [0,2],[1,1]",
				"blit (28,25)-(388,40) [1,1],[45,1], to (36,25)-(396,40) [2,1],[45,1]",
				"fill (396,25)-(400,40) [47,1],[-,1]",
				"fill (28,25)-(36,40) [1,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (28,25) [1,1] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		{
			// Insert into a line with a tab
			name: "insertTabAndChar",
			fn:   insertTabAndChar,
			want: []string{
				"blit (28,10)-(36,25) [1,0],[1,1], to (84,10)-(92,25) [8,0],[1,1]",
				"fill (28,10)-(84,25) [1,0],[7,1]",
				"fill (36,10)-(84,25) [2,0],[6,1]",
				"fill (28,10)-(36,25) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (28,10) [1,0] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		{
			// Split a wrapped line by inserting a newline.
			name:     "splitWrappedLine",
			fn:       splitWrappedLine,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				// The previously failing insertion starts here. We didn't have to do
				// anything in this case. But we still fill blank space at the end of the
				// line over again. This is (hopefully) harmless.
				// TODO(rjk): Elide the 0-width draws.
				"fill (44,10)-(45,25) [3,0],[-,1]",
				"fill (20,25)-(20,40) [0,1],[0,1]",
			},
		},
		{
			// Insert a single character that forces conversion of non-wrapped to
			// wrapped with wripple to end.
			name:     "insertForcesWrap",
			fn:       insertForcesWrap,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"blit (20,40)-(45,70) [0,2],[-,2], to (20,55)-(45,85) [0,3],[-,2]",
				"blit (44,25)-(45,40) [3,1],[-,1], to (44,40)-(45,55) [3,2],[-,1]",
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,40)-(44,55) [0,2],[3,1]",
				"fill (28,25)-(45,40) [1,1],[-,1]",
				"blit (36,10)-(44,25) [2,0],[1,1], to (20,25)-(28,40) [0,1],[1,1]",
				"fill (36,10)-(45,25) [2,0],[-,1]",
				"fill (20,25)-(20,40) [0,1],[0,1]",
				`screen-800x600 <- string "X" atpoint: (36,10) [2,0] fill: black`,
			},
		},
		{
			// Append a pair of characters at the end of the otherwise full text
			// area.
			name:     "appendAtEnd",
			fn:       appendAtEnd,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"fill (44,70)-(45,85) [3,4],[-,1]",
				"fill (20,85)-(20,100) [0,5],[0,1]",
			},
		},

		{
			// Insert a multibox string that forces ripple past the end.
			name:     "insertWrappedThatForcesRipple",
			fn:       insertWrappedThatForcesRipple,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"fill (44,70)-(45,85) [3,4],[-,1]",
				"blit (28,55)-(36,70) [1,3],[1,1], to (36,70)-(44,85) [2,4],[1,1]",
				"fill (28,55)-(45,70) [1,3],[-,1]",
				"fill (20,70)-(36,85) [0,4],[2,1]",
				`screen-800x600 <- string "ij" atpoint: (28,55) [1,3] fill: black`,
				`screen-800x600 <- string "XX" atpoint: (20,70) [0,4] fill: black`,
			},
		},
		{
			// Insert a string that pushes a blank line off the end.
			name:     "insertPushesBlankLineOffEnd",
			fn:       insertPushesBlankLineOffEnd,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"blit (20,40)-(45,70) [0,2],[-,2], to (20,55)-(45,85) [0,3],[-,2]",
				"blit (44,25)-(45,40) [3,1],[-,1], to (44,40)-(45,55) [3,2],[-,1]",
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,40)-(44,55) [0,2],[3,1]",
				"fill (28,25)-(45,40) [1,1],[-,1]",
				"blit (36,10)-(44,25) [2,0],[1,1], to (20,25)-(28,40) [0,1],[1,1]",
				"blit (28,10)-(36,25) [1,0],[1,1], to (36,10)-(44,25) [2,0],[1,1]",
				"fill (44,10)-(45,25) [3,0],[-,1]",
				"fill (28,10)-(36,25) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (28,10) [1,0] fill: black`,
			},
		},
		{
			// Insert text that doesn't fit.
			name: "insertPastEnd",
			fn:   insertPastEnd,
			want: []string{
				"fill (20,10)-(45,25) [0,0],[-,1]",
				"fill (20,25)-(45,55) [0,1],[-,2]",
				"fill (20,55)-(20,70) [0,3],[0,1]",
				`screen-800x600 <- string "a本ポ" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,25) [0,1] fill: black`,
				`screen-800x600 <- string "ポポh" atpoint: (20,40) [0,2] fill: black`},
			textarea: image.Rect(20, 10, 45, 55),
		},
		{
			// Append a multibox string that hangs off the end. TODO(rjk): Draws a
			// zero-width fill off the end of text area. This is conceivably wrong.
			// It would (for example) make some drawing stacks unhappy.
			name:     "appendHangingLongAtEnd",
			fn:       appendHangingLongAtEnd,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"fill (28,70)-(45,85) [1,4],[-,1]",
				"fill (20,85)-(20,100) [0,5],[0,1]",
				`screen-800x600 <- string "XX" atpoint: (28,70) [1,4] fill: black`,
			},
		},
		{
			// Insert a new line that pushes another newline down.
			name:     "insertsRippledNewLine",
			fn:       insertsRippledNewLine,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"blit (20,55)-(45,70) [0,3],[-,1], to (20,70)-(45,85) [0,4],[-,1]",
				"fill (20,55)-(45,70) [0,3],[-,1]",
				"fill (20,70)-(20,85) [0,4],[0,1]",
			},
		},
		{
			// Rippled down off edge of frame of wrapped text.
			name:     "insertForcesRippleOfWrapped",
			fn:       insertForcesRippleOfWrapped,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"blit (20,25)-(45,70) [0,1],[-,3], to (20,40)-(45,85) [0,2],[-,3]",
				"blit (44,10)-(45,25) [3,0],[-,1], to (44,25)-(45,40) [3,1],[-,1]",
				"blit (20,10)-(44,25) [0,0],[3,1], to (20,25)-(44,40) [0,1],[3,1]",
				"fill (20,10)-(45,25) [0,0],[-,1]",
				"fill (20,10)-(45,25) [0,0],[-,1]",
				"fill (20,25)-(20,40) [0,1],[0,1]",
				`screen-800x600 <- string "ABC" atpoint: (20,10) [0,0] fill: black`,
			},

			// TODO(rjk): Wrapping with tabs
		},
		{
			// Insert a character exactly at the wrap boundary (position where the
			// first visual line is already full). The character wraps to line 2.
			name:     "insertAtExactWrapBoundary",
			fn:       insertAtExactWrapBoundary,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"blit (20,40)-(45,70) [0,2],[-,2], to (20,55)-(45,85) [0,3],[-,2]",
				"blit (44,25)-(45,40) [3,1],[-,1], to (44,40)-(45,55) [3,2],[-,1]",
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,40)-(44,55) [0,2],[3,1]",
				"fill (28,25)-(45,40) [1,1],[-,1]",
				"fill (44,10)-(45,25) [3,0],[-,1]",
				"fill (20,25)-(28,40) [0,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (20,25) [0,1] fill: black`,
			},
		},
		{
			// Insert a character that exactly fills the first visual line from 2
			// to 3 characters. No wrap should occur.
			name:     "insertExactlyFillsAlignedLine",
			fn:       insertExactlyFillsAlignedLine,
			textarea: image.Rect(20, 10, 45, 85),
			want: []string{
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,25)-(44,40) [0,1],[3,1]",
				"fill (44,10)-(45,25) [3,0],[-,1]",
				"fill (36,10)-(44,25) [2,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (36,10) [2,0] fill: black`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iv.textarea = tc.textarea
			fr := setupFrame(t, iv)

			if tc.knowntofail {
				tc.fn(t, fr, iv, tc.name)
				generateVisualizedOutput(t, fr)
				snapAfterPNG(t, fr, tc.name)
				t.Log("known failing: bug not yet fixed")
				t.Fail()
				return
			}

			// TODO(rjk): validate here

			tc.fn(t, fr, iv, tc.name)

			// TODO(rjk): validate here

			frimpl := fr.(*frameimpl)
			t.Logf("rect=%v nlines=%d nchars=%d nbox=%d lastlinefull=%v",
				frimpl.rect, frimpl.nlines, frimpl.nchars, len(frimpl.box), frimpl.lastlinefull)

			// Peek inside.
			got := gdo(t, fr).DrawOps()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

			// SVG based output and comparison.
			visualizedoutputtest(t, fr)
			snapAfterPNG(t, fr, tc.name)
		})
	}
}
