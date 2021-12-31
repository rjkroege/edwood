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
				pt1 := image.Pt(10, 15)
				pt2, f := f.bxscan([]byte("本"), &pt1)
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
				pt2, f := f.bxscan([]byte("本"), &pt1)
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
				pt2, f := f.bxscan([]byte("本"), &pt1)
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
				pt2, f := f.bxscan([]byte("本a"), &pt1)
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
				pt2, f := f.bxscan([]byte(bigstring), &pt1)
				return pt1, pt2, f
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
				pt1 := image.Pt(10, 15)
				pt2, f := f.bxscan([]byte("\ta\n"), &pt1)
				return pt1, pt2, f
			},
			3,
			[]*frbox{makeBox("\t"), makeBox("a"), makeBox("\n")},
			image.Pt(10, 15),
			image.Pt(10, 15+13),
		},
	})
}

type invariants struct {
	topcorner image.Point
	textarea  image.Rectangle
}

// setupFrame makes a Frame object for testing with a recording Display
// implementation.
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

func insertLongLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("ab\ncd\nef"), 0)
	gdo(t, fr).Clear()

	bigstring := makereplicatedstring(iv.textarea.Dx() / 10)
	s := fr.Insert([]rune(bigstring), 4)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertIntoLongLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("ab\ncd\nef"), 0)
	bigstring := makereplicatedstring(iv.textarea.Dx() / 10)
	s := fr.Insert([]rune(bigstring), 4)
	gdo(t, fr).Clear()

	fr.Insert([]rune("X"), 4)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertTabAndChar(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("ab"), 0)
	gdo(t, fr).Clear()
	s := fr.Insert([]rune("\t"), 1)
	fr.Insert([]rune("X"), 1)

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertPastEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	s := fr.Insert([]rune(makereplicatedstring(6)), 0)

	// I would have expected that this should be true?
	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func splitWrappedLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	rss := []rune(makereplicatedstring(6))

	t.Logf("%q", string(rss))

	fr.Insert(rss, 0)
	s := fr.Insert([]rune{'\n'}, 3)

	// I would have expected that this should be true?
	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertForcesWrap(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n4ij"), 0)

	s := fr.Insert([]rune{'X'}, 2)

	// This is a
	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func appendAtEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n4ij"), 0)

	s := fr.Insert([]rune{'X', 'X'}, len("0ab\n1cd\n2ef\n3gh\n4ij"))

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func appendHangingLongAtEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0\n1\n2\n3\n4\n"), 0)

	s := fr.Insert([]rune("XXX"), len("0\n1\n2\n3\n4"))

	if got, want := s, false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertWrappedThatForcesRipple(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0\n1\n2\n3b\n4\n"), 0)

	s := fr.Insert([]rune("ijXX"), len("0\n1\n2\n3"))

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertPushesBlankLineOffEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n\n"), 0)

	s := fr.Insert([]rune("X"), 1)

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertsRippledNewLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0ab\n1cd\n2ef\n3gh\n"), 0)

	s := fr.Insert([]rune("\n"), len("0ab\n1cd\n2ef\n"))

	if got, want := s, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func insertForcesRippleOfWrapped(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	gdo(t, fr).Clear()
	fr.Insert([]rune("0ab1cd2ef3gh4ij5"), 0)

	s := fr.Insert([]rune("ABC"), 0)

	if got, want := s, false; got != want {
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
func TestInsert(t *testing.T) {
	iv := &invariants{
		topcorner: image.Pt(20, 10),
	}

	*validate = true

	tests := []struct {
		name        string
		fn          func(t *testing.T, fr Frame, iv *invariants)
		want        []string
		textarea    image.Rectangle
		knowntofail bool
	}{
		// TODO(rjk): Test cases
		// 3. add a newline after one already there.
		{
			name: "setupFrame",
			fn:   nop,
			want: []string{"fill (0,0)-(3,10) [-,-1],[-,1]",
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				"fill (1,0)-(2,10) [-,-1],[-,1]",
				"fill (0,0)-(3,3) [-,-1],[-,-]",
				"fill (0,7)-(3,10) [-,-],[-,-]",
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// Inserts ab at the start of the line with no wrapping or ripple.
			name: "simpleInsertShortString",
			fn:   simpleInsertShortString,
			want: []string{
				"fill (20,10)-(46,20) [0,0],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,10) [0,0] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// A short but newline containing string that fits inserted at the start.
			name: "multiInsertShortString",
			fn:   multiInsertShortString,
			want: []string{"fill (20,10)-(400,20) [0,0],[-,1]",
				"fill (20,20)-(46,30) [0,1],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "cd" atpoint: (20,20) [0,1] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// A long line inserted. Requires wrapping the inserted line and rippling
			// the remaining text.
			name: "insertLongLine",
			fn:   insertLongLine,
			want: []string{
				"blit (20,30)-(46,40) [0,2],[2,1], to (20,40)-(46,50) [0,3],[2,1]",
				"fill (254,30)-(400,40) [18,2],[-,1]",
				"blit (33,20)-(46,30) [1,1],[1,1], to (241,30)-(254,40) [17,2],[1,1]",
				"fill (33,20)-(400,30) [1,1],[-,1]",
				"fill (20,30)-(241,40) [0,2],[17,1]",
				`screen-800x600 <- string "a本ポポポポポポポポポポポポポポポポポポポポポポポポポポ" atpoint: (33,20) [1,1] fill: black`,
				`screen-800x600 <- string "ポポポポポポポポポポポポhello" atpoint: (20,30) [0,2] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// Insert into a long line
			name: "insertIntoLongLine",
			fn:   insertIntoLongLine,
			want: []string{
				"blit (20,40)-(46,50) [0,3],[2,1], to (20,40)-(46,50) [0,3],[2,1]",
				"fill (267,30)-(400,40) [19,2],[-,1]",
				"blit (20,30)-(254,40) [0,2],[18,1], to (33,30)-(267,40) [1,2],[18,1]",
				"blit (384,20)-(397,30) [28,1],[1,1], to (20,30)-(33,40) [0,2],[1,1]",
				"blit (33,20)-(384,30) [1,1],[27,1], to (46,20)-(397,30) [2,1],[27,1]",
				"fill (397,20)-(400,30) [29,1],[-,1]",
				"fill (33,20)-(46,30) [1,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,20) [1,1] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// Insert into a line with a tab
			name: "insertTabAndChar",
			fn:   insertTabAndChar,
			want: []string{
				"blit (33,10)-(46,20) [1,0],[1,1], to (124,10)-(137,20) [8,0],[1,1]",
				"fill (33,10)-(124,20) [1,0],[7,1]",
				"fill (46,10)-(124,20) [2,0],[6,1]",
				"fill (33,10)-(46,20) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,10) [1,0] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// Insert text that doesn't fit.
			name: "insertPastEnd",
			fn:   insertPastEnd,
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,40) [0,1],[-,2]",
				"fill (20,40)-(20,50) [0,3],[0,1]",
				`screen-800x600 <- string "a本ポ" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "ポポh" atpoint: (20,30) [0,2] fill: black`},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Split a wrapped line by inserting a newline.
			name:     "splitWrappedLine",
			fn:       splitWrappedLine,
			textarea: image.Rect(20, 10, 60, 60),
			// This inserts an additional blankline for a newline added to the end of
			// a full text row that doesn't belong there. The contents of the screen
			// no longer match what we'd expect based on the box model. e.g.
			// insertForcesWrap below shows that the newline should add a box without
			// actually drawing anything. Subsequent edits then induce confusion.
			knowntofail: true,
		},
		{
			// Insert a single character that forces conversion of non-wrapped to
			// wrapped with wripple to end.
			name:     "insertForcesWrap",
			fn:       insertForcesWrap,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,50) [0,1],[-,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,30)-(60,50) [0,2],[-,2], to (20,40)-(60,60) [0,3],[-,2]",
				"blit (59,20)-(60,30) [3,1],[-,1], to (59,30)-(60,40) [3,2],[-,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(60,30) [1,1],[-,1]",
				"blit (46,10)-(59,20) [2,0],[1,1], to (20,20)-(33,30) [0,1],[1,1]",
				"fill (46,10)-(60,20) [2,0],[-,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "X" atpoint: (46,10) [2,0] fill: black`,
			},
		},
		{
			// Append a pair of characters at the end of the otherwise full text
			// area.
			name:     "appendAtEnd",
			fn:       appendAtEnd,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,50) [0,1],[-,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
			},
		},

		{
			// Append a multibox string that hangs off the end. TODO(rjk): Draws a
			// zero-width fill off the end of text area. This is conceivably wrong.
			// It would (for example) make some drawing stacks unhappy.
			name:     "appendHangingLongAtEnd",
			fn:       appendHangingLongAtEnd,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4" atpoint: (20,50) [0,4] fill: black`,
				"fill (33,50)-(60,60) [1,4],[-,1]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "XX" atpoint: (33,50) [1,4] fill: black`,
			},
		},
		{
			// Insert a multibox string that forces ripple past the end.
			name:     "insertWrappedThatForcesRipple",
			fn:       insertWrappedThatForcesRipple,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3b" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4" atpoint: (20,50) [0,4] fill: black`,
				"fill (59,50)-(60,60) [3,4],[-,1]",
				"blit (33,40)-(46,50) [1,3],[1,1], to (46,50)-(59,60) [2,4],[1,1]",
				"fill (33,40)-(60,50) [1,3],[-,1]",
				"fill (20,50)-(46,60) [0,4],[2,1]",
				`screen-800x600 <- string "ij" atpoint: (33,40) [1,3] fill: black`,
				`screen-800x600 <- string "XX" atpoint: (20,50) [0,4] fill: black`,
			},
		},
		{
			// Insert a string that pushes a blank line off the end.
			name:     "insertPushesBlankLineOffEnd",
			fn:       insertPushesBlankLineOffEnd,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				"blit (20,30)-(60,50) [0,2],[-,2], to (20,40)-(60,60) [0,3],[-,2]",
				"blit (59,20)-(60,30) [3,1],[-,1], to (59,30)-(60,40) [3,2],[-,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(60,30) [1,1],[-,1]",
				"blit (46,10)-(59,20) [2,0],[1,1], to (20,20)-(33,30) [0,1],[1,1]",
				"blit (33,10)-(46,20) [1,0],[1,1], to (46,10)-(59,20) [2,0],[1,1]",
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"fill (33,10)-(46,20) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,10) [1,0] fill: black`,
			},
		},
		{
			// Insert into a long line
			name: "insertIntoLongLine",
			fn:   insertIntoLongLine,
			want: []string{
				"blit (20,40)-(46,50) [0,3],[2,1], to (20,40)-(46,50) [0,3],[2,1]",
				"fill (267,30)-(400,40) [19,2],[-,1]",
				"blit (20,30)-(254,40) [0,2],[18,1], to (33,30)-(267,40) [1,2],[18,1]",
				"blit (384,20)-(397,30) [28,1],[1,1], to (20,30)-(33,40) [0,2],[1,1]",
				"blit (33,20)-(384,30) [1,1],[27,1], to (46,20)-(397,30) [2,1],[27,1]",
				"fill (397,20)-(400,30) [29,1],[-,1]",
				"fill (33,20)-(46,30) [1,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,20) [1,1] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// Insert into a line with a tab
			name: "insertTabAndChar",
			fn:   insertTabAndChar,
			want: []string{
				"blit (33,10)-(46,20) [1,0],[1,1], to (124,10)-(137,20) [8,0],[1,1]",
				"fill (33,10)-(124,20) [1,0],[7,1]",
				"fill (46,10)-(124,20) [2,0],[6,1]",
				"fill (33,10)-(46,20) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,10) [1,0] fill: black`,
			},
			textarea: image.Rect(20, 10, 400, 500),
		},
		{
			// Insert text that doesn't fit.
			name: "insertPastEnd",
			fn:   insertPastEnd,
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,40) [0,1],[-,2]",
				"fill (20,40)-(20,50) [0,3],[0,1]",
				`screen-800x600 <- string "a本ポ" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "ポポh" atpoint: (20,30) [0,2] fill: black`},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Split a wrapped line by inserting a newline.
			name:     "splitWrappedLine",
			fn:       splitWrappedLine,
			textarea: image.Rect(20, 10, 60, 60),
			// This inserts an additional blankline for a newline added to the end of
			// a full text row that doesn't belong there. The contents of the screen
			// no longer match what we'd expect based on the box model. e.g.
			// insertForcesWrap below shows that the newline should add a box without
			// actually drawing anything. Subsequent edits then induce confusion.
			knowntofail: true,
		},
		{
			// Insert a single character that forces conversion of non-wrapped to
			// wrapped with wripple to end.
			name:     "insertForcesWrap",
			fn:       insertForcesWrap,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,50) [0,1],[-,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,30)-(60,50) [0,2],[-,2], to (20,40)-(60,60) [0,3],[-,2]",
				"blit (59,20)-(60,30) [3,1],[-,1], to (59,30)-(60,40) [3,2],[-,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(60,30) [1,1],[-,1]",
				"blit (46,10)-(59,20) [2,0],[1,1], to (20,20)-(33,30) [0,1],[1,1]",
				"fill (46,10)-(60,20) [2,0],[-,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "X" atpoint: (46,10) [2,0] fill: black`,
			},
		},
		{
			// Append a pair of characters at the end of the otherwise full text
			// area.
			name:     "appendAtEnd",
			fn:       appendAtEnd,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,50) [0,1],[-,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
			},
		},

		{
			// Append a multibox string that hangs off the end. TODO(rjk): Draws a
			// zero-width fill off the end of text area. This is conceivably wrong.
			// It would (for example) make some drawing stacks unhappy.
			name:     "appendHangingLongAtEnd",
			fn:       appendHangingLongAtEnd,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4" atpoint: (20,50) [0,4] fill: black`,
				"fill (33,50)-(60,60) [1,4],[-,1]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "XX" atpoint: (33,50) [1,4] fill: black`,
			},
		},
		{
			// Insert a multibox string that forces ripple past the end.
			name:     "insertWrappedThatForcesRipple",
			fn:       insertWrappedThatForcesRipple,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3b" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4" atpoint: (20,50) [0,4] fill: black`,
				"fill (59,50)-(60,60) [3,4],[-,1]",
				"blit (33,40)-(46,50) [1,3],[1,1], to (46,50)-(59,60) [2,4],[1,1]",
				"fill (33,40)-(60,50) [1,3],[-,1]",
				"fill (20,50)-(46,60) [0,4],[2,1]",
				`screen-800x600 <- string "ij" atpoint: (33,40) [1,3] fill: black`,
				`screen-800x600 <- string "XX" atpoint: (20,50) [0,4] fill: black`,
			},
		},
		{
			// Insert a string that pushes a blank line off the end.
			name:     "insertPushesBlankLineOffEnd",
			fn:       insertPushesBlankLineOffEnd,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				"blit (20,30)-(60,50) [0,2],[-,2], to (20,40)-(60,60) [0,3],[-,2]",
				"blit (59,20)-(60,30) [3,1],[-,1], to (59,30)-(60,40) [3,2],[-,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(60,30) [1,1],[-,1]",
				"blit (46,10)-(59,20) [2,0],[1,1], to (20,20)-(33,30) [0,1],[1,1]",
				"blit (33,10)-(46,20) [1,0],[1,1], to (46,10)-(59,20) [2,0],[1,1]",
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"fill (33,10)-(46,20) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,10) [1,0] fill: black`,
			},
		},
		{
			// Insert a new line that pushes another newline down.
			name:     "insertsRippledNewLine",
			fn:       insertsRippledNewLine,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{

				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,50) [0,1],[-,3]",
				"fill (20,50)-(20,60) [0,4],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				"blit (20,40)-(60,50) [0,3],[-,1], to (20,50)-(60,60) [0,4],[-,1]",
				"fill (20,40)-(60,50) [0,3],[-,1]",
				"fill (20,50)-(20,60) [0,4],[0,1]",
			},
		},
		{
			// Rippled down off edge of frame of wrapped text.
			name:     "insertForcesRippleOfWrapped",
			fn:       insertForcesRippleOfWrapped,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{

				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(60,60) [0,1],[-,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,20)-(60,50) [0,1],[-,3], to (20,30)-(60,60) [0,2],[-,3]",
				"blit (59,10)-(60,20) [3,0],[-,1], to (59,20)-(60,30) [3,1],[-,1]",
				"blit (20,10)-(59,20) [0,0],[3,1], to (20,20)-(59,30) [0,1],[3,1]",
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "ABC" atpoint: (20,10) [0,0] fill: black`,
			},

			// TODO(rjk): Wrapping with tabs
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.knowntofail {
				return
			}

			iv.textarea = tc.textarea
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

	// TODO(rjk): I had wanted a "nice" way to describe and validate the
	// tests by automatically generating diagrams. I thought about this. See
	// Thursday-Morning.md in the wiki. I eventually (reluctantly) concluded
	// that the testing and debugging effort to make sure that the
	// automatically generated diagrams were correct was the same effort as
	// validating the ops here.
	//
	// I started drawing the op sequences in OmniGraffle. This is was only a
	// little more work than doing it on paper.
	//
	// TODO(rjk): include the diagrams in the source tree.
}
