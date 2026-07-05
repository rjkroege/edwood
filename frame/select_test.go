package frame

import (
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/draw"
)

func selectSingleCharacterAtLineEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab"), 0)
	gdo(t, fr).Clear()

	ch := make(chan draw.Mouse, 1)
	mc := &draw.Mousectl{C: ch}
	downevent := draw.Mouse{Point: image.Pt(46, 10), Buttons: 1}

	// Pre-load a single event: release at right edge of 'b' → selects chars [2, 3)
	ch <- draw.Mouse{Point: image.Pt(59, 10), Buttons: 0}

	p0, p1 := fr.Select(mc, &downevent, func(SelectScrollUpdater, int) {})

	if got, want := p0, 2; got != want {
		t.Errorf("p0: got %v, want %v", got, want)
	}
	if got, want := p1, 3; got != want {
		t.Errorf("p1: got %v, want %v", got, want)
	}
}

// selectDefAndInsertAtStart selects "def" in a 3-line frame then inserts
// "012" at position 0 to verify that Insert correctly handles an existing
// selection.
//
// Frame content: "abc\ndef\nghi\n" (3 lines).
// 'd' = char 4 at (20,20); after 'f' = char 7 at (59,20).
func selectDefAndInsertAtStart(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\ndef\nghi\n"), 0)
	gdo(t, fr).Clear()

	ch := make(chan draw.Mouse, 1)
	mc := &draw.Mousectl{C: ch}
	// click at 'd', release after 'f' → selects "def" = chars [4, 7)
	downevent := draw.Mouse{Point: image.Pt(20, 20), Buttons: 1}
	ch <- draw.Mouse{Point: image.Pt(59, 20), Buttons: 0}

	p0, p1 := fr.Select(mc, &downevent, func(SelectScrollUpdater, int) {})
	if got, want := p0, 4; got != want {
		t.Errorf("select p0: got %v, want %v", got, want)
	}
	if got, want := p1, 7; got != want {
		t.Errorf("select p1: got %v, want %v", got, want)
	}

	fr.Insert([]rune("012"), 0)
}

// selectScrollDown selects "ghi\n0" using a drag that goes below the frame
// bottom, causing getmorelines to scroll the view down (delete "abc\n" from
// the top, append "012\n" at the bottom).
//
// Initial frame: "abc\ndef\nghi\n" (3 lines, full).
// After scroll: "def\nghi\n012\n".
// Selection: [4, 9) = "ghi\n0" in the new frame.
func selectScrollDown(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\ndef\nghi\n"), 0)
	gdo(t, fr).Clear()

	ch := make(chan draw.Mouse, 2)
	mc := &draw.Mousectl{C: ch}
	// click at 'g' (char 8 at (20,30))
	downevent := draw.Mouse{Point: image.Pt(20, 30), Buttons: 1}
	// drag below frame (y=45 > frame max y=40) — triggers getmorelines
	ch <- draw.Mouse{Point: image.Pt(20, 45), Buttons: 1}
	// release at '1' (char 9 in new frame at (33,30)) — selection [4,9) = "ghi\n0"
	ch <- draw.Mouse{Point: image.Pt(33, 30), Buttons: 0}

	scrolled := false
	p0, p1 := fr.Select(mc, &downevent, func(f SelectScrollUpdater, n int) {
		if n == 0 {
			return // touchup call after scroll
		}
		if n > 0 && !scrolled {
			scrolled = true
			// Remove "abc\n" from the top and append "012\n" at the bottom.
			f.Delete(0, 4)
			f.Insert([]rune("012\n"), 8)
		}
	})

	if got, want := p0, 4; got != want {
		t.Errorf("p0: got %v, want %v", got, want)
	}
	if got, want := p1, 9; got != want {
		t.Errorf("p1: got %v, want %v", got, want)
	}
}

// selectScrollUp selects "2\nab" using a drag that goes above the frame top,
// causing getmorelines to scroll the view up (prepend "012\n"; "ghi\n" falls
// off the bottom naturally since the frame is already full).
//
// Initial frame: "abc\ndef\nghi\n" (3 lines, full).
// After scroll: "012\nabc\ndef\n" ("ghi\n" is off screen).
// Selection: [2, 6) = "2\nab" in the new frame.
// (Visually: 'a','b' highlighted on the "abc" line; '2' highlighted on "012" line above.)
func selectScrollUp(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\ndef\nghi\n"), 0)
	gdo(t, fr).Clear()

	ch := make(chan draw.Mouse, 2)
	mc := &draw.Mousectl{C: ch}
	// click at 'c' (char 2 at (46,10)); after scroll 'c' shifts to char 6
	downevent := draw.Mouse{Point: image.Pt(46, 10), Buttons: 1}
	// drag above frame (y=5 < frame min y=10) — triggers getmorelines
	ch <- draw.Mouse{Point: image.Pt(33, 5), Buttons: 1}
	// release at '2' (char 2 in new frame at (46,10)) — selection [2,6) = "2\nab"
	ch <- draw.Mouse{Point: image.Pt(46, 10), Buttons: 0}

	scrolled := false
	p0, p1 := fr.Select(mc, &downevent, func(f SelectScrollUpdater, n int) {
		if n == 0 {
			return // touchup call after scroll
		}
		if n < 0 && !scrolled {
			scrolled = true
			// Prepend "012\n"; the frame truncates "ghi\n" off the bottom.
			f.Insert([]rune("012\n"), 0)
		}
	})

	if got, want := p0, 2; got != want {
		t.Errorf("p0: got %v, want %v", got, want)
	}
	if got, want := p1, 6; got != want {
		t.Errorf("p1: got %v, want %v", got, want)
	}
}

// selectCrossLine selects "c\nd" which spans the boundary between line 0
// ("abc\n") and line 1 ("def\n").
//
// 'c' = char 2 at (46,10); start-of-'e' = char 5 at (33,20).
// Selection: [2, 5) = "c\nd".
func selectCrossLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\ndef\nghi\n"), 0)
	gdo(t, fr).Clear()

	ch := make(chan draw.Mouse, 1)
	mc := &draw.Mousectl{C: ch}
	// click at 'c', release at start of 'e' → selects [2, 5) = "c\nd"
	downevent := draw.Mouse{Point: image.Pt(46, 10), Buttons: 1}
	ch <- draw.Mouse{Point: image.Pt(33, 20), Buttons: 0}

	p0, p1 := fr.Select(mc, &downevent, func(SelectScrollUpdater, int) {})
	if got, want := p0, 2; got != want {
		t.Errorf("p0: got %v, want %v", got, want)
	}
	if got, want := p1, 5; got != want {
		t.Errorf("p1: got %v, want %v", got, want)
	}
}

func TestSelect(t *testing.T) {
	iv := &invariants{topcorner: image.Pt(20, 10)}
	*validate = true

	tests := []struct {
		name        string
		fn          func(*testing.T, Frame, *invariants)
		want        []string
		textarea    image.Rectangle
		knowntofail bool
	}{
		{
			// Select the last character in a single-line frame.
			name:     "selectSingleCharacterAtLineEnd",
			fn:       selectSingleCharacterAtLineEnd,
			textarea: image.Rect(20, 10, 60, 40),
			want: []string{
				// tick saves background at cursor position
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				// tick draws cursor (nil ColTick → no visible pixel change)
				"screen-800x600 <- draw r: (45,10)-(48,20) src: nil mask nil p1: (0,0)",
				// untick restores background
				"fill (45,10)-(48,20) [-,0],[-,1]",
				// selection highlight fill for 'b' with ColHigh
				"fill (46,10)-(59,20) [2,0],[1,1]",
				// selection redraws 'b' text over the highlight
				`screen-800x600 <- string "b" atpoint: (46,10) [2,0] fill: black`,
			},
		},
		{
			// Select "def" in a 3-line frame, then insert "012" at position 0.
			name:     "selectDefAndInsertAtStart",
			fn:       selectDefAndInsertAtStart,
			textarea: image.Rect(20, 10, 60, 40),
			want: []string{
				// tick saves bg at 'd'; cursor drawn (nil ColTick)
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				"screen-800x600 <- draw r: (19,20)-(22,30) src: nil mask nil p1: (0,0)",
				// untick restores bg
				"fill (19,20)-(22,30) [-,1],[-,1]",
				// selection highlight for "def" — drawn twice (once for initial tick,
				// once at release when p0==p1 edge triggers a redraw)
				"fill (20,20)-(59,30) [0,1],[3,1]",
				`screen-800x600 <- string "def" atpoint: (20,20) [0,1] fill: black`,
				"fill (20,20)-(59,30) [0,1],[3,1]",
				`screen-800x600 <- string "def" atpoint: (20,20) [0,1] fill: black`,
				// Insert("012", 0): shift existing lines down then draw "012" on line 0
				"blit (20,20)-(60,30) [0,1],[-,1], to (20,30)-(60,40) [0,2],[-,1]",
				"blit (59,10)-(60,20) [3,0],[-,1], to (59,20)-(60,30) [3,1],[-,1]",
				"blit (20,10)-(59,20) [0,0],[3,1], to (20,20)-(59,30) [0,1],[3,1]",
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "012" atpoint: (20,10) [0,0] fill: black`,
			},
		},
		{
			// Select spanning the bottom of the frame; getmorelines scrolls down.
			// getmorelines deletes "abc\n" and appends "012\n".
			// New frame: "def\nghi\n012\n". Selection [4,9) = "ghi\n0".
			name:     "selectScrollDown",
			fn:       selectScrollDown,
			textarea: image.Rect(20, 10, 60, 40),
			want: []string{
				// tick saves bg at 'g'; cursor drawn (nil ColTick); untick
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				"screen-800x600 <- draw r: (19,30)-(22,40) src: nil mask nil p1: (0,0)",
				"fill (19,30)-(22,40) [-,2],[-,1]",
				// Delete("abc\n"): scroll content up two lines
				"blit (20,20)-(60,30) [0,1],[-,1], to (20,10)-(60,20) [0,0],[-,1]",
				"blit (20,30)-(60,40) [0,2],[-,1], to (20,20)-(60,30) [0,1],[-,1]",
				"fill (20,30)-(60,40) [0,2],[-,1]",
				"fill (20,40)-(20,50) [0,3],[0,1]",
				// Insert("012\n"): tick at new cursor position, then draw "012"
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				"screen-800x600 <- draw r: (19,20)-(22,30) src: nil mask nil p1: (0,0)",
				"fill (19,20)-(22,30) [-,1],[-,1]",
				"fill (20,30)-(60,40) [0,2],[-,1]",
				"fill (20,40)-(20,50) [0,3],[0,1]",
				`screen-800x600 <- string "012" atpoint: (20,30) [0,2] fill: black`,
				// drawselimpl after getmorelines: selection [4,12) drawn (full tail)
				"fill (20,20)-(59,30) [0,1],[3,1]",
				`screen-800x600 <- string "ghi" atpoint: (20,20) [0,1] fill: black`,
				"fill (59,20)-(60,30) [3,1],[-,1]",
				"fill (60,20)-(60,30) [-,1],[0,1]",
				"fill (20,30)-(59,40) [0,2],[3,1]",
				`screen-800x600 <- string "012" atpoint: (20,30) [0,2] fill: black`,
				"fill (59,30)-(60,40) [3,2],[-,1]",
				// drawselimpl on release event (Buttons=0): selection [4,12) re-drawn
				"fill (20,20)-(59,30) [0,1],[3,1]",
				`screen-800x600 <- string "ghi" atpoint: (20,20) [0,1] fill: black`,
				"fill (59,20)-(60,30) [3,1],[-,1]",
				"fill (60,20)-(60,30) [-,1],[0,1]",
				"fill (20,30)-(59,40) [0,2],[3,1]",
				`screen-800x600 <- string "012" atpoint: (20,30) [0,2] fill: black`,
				"fill (59,30)-(60,40) [3,2],[-,1]",
				// drawselimpl final: selection refined to [4,9) = "ghi\n0"
				"fill (20,20)-(59,30) [0,1],[3,1]",
				`screen-800x600 <- string "ghi" atpoint: (20,20) [0,1] fill: black`,
				"fill (59,20)-(60,30) [3,1],[-,1]",
				"fill (60,20)-(60,30) [-,1],[0,1]",
				"fill (20,30)-(33,40) [0,2],[1,1]",
				`screen-800x600 <- string "0" atpoint: (20,30) [0,2] fill: black`,
			},
		},
		{
			// Select spanning the top of the frame; getmorelines scrolls up.
			// getmorelines prepends "012\n"; "ghi\n" falls off the bottom.
			// New frame: "012\nabc\ndef\n". Selection [2,6) = "2\nab".
			name:     "selectScrollUp",
			fn:       selectScrollUp,
			textarea: image.Rect(20, 10, 60, 40),
			want: []string{
				// tick saves bg at 'c'; cursor drawn (nil ColTick); untick
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				"screen-800x600 <- draw r: (45,10)-(48,20) src: nil mask nil p1: (0,0)",
				"fill (45,10)-(48,20) [-,0],[-,1]",
				// Insert("012\n", 0): shift existing lines down
				"blit (20,20)-(60,30) [0,1],[-,1], to (20,30)-(60,40) [0,2],[-,1]",
				"blit (20,10)-(60,20) [0,0],[-,1], to (20,20)-(60,30) [0,1],[-,1]",
				"fill (20,10)-(60,20) [0,0],[-,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "012" atpoint: (20,10) [0,0] fill: black`,
				// drawselimpl: broad selection [0,6) = "012\nab" after above-frame drag
				"fill (33,10)-(59,20) [1,0],[2,1]",
				`screen-800x600 <- string "12" atpoint: (33,10) [1,0] fill: black`,
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"fill (60,10)-(60,20) [-,0],[0,1]",
				"fill (20,20)-(46,30) [0,1],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,20) [0,1] fill: black`,
				// drawselimpl: same selection re-drawn on release event (Buttons=0)
				"fill (33,10)-(59,20) [1,0],[2,1]",
				`screen-800x600 <- string "12" atpoint: (33,10) [1,0] fill: black`,
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"fill (60,10)-(60,20) [-,0],[0,1]",
				"fill (20,20)-(46,30) [0,1],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,20) [0,1] fill: black`,
				// drawselimpl final: selection refined to [2,6) = "2\nab"
				"fill (46,10)-(59,20) [2,0],[1,1]",
				`screen-800x600 <- string "2" atpoint: (46,10) [2,0] fill: black`,
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"fill (60,10)-(60,20) [-,0],[0,1]",
				"fill (20,20)-(46,30) [0,1],[2,1]",
				`screen-800x600 <- string "ab" atpoint: (20,20) [0,1] fill: black`,
			},
		},
		{
			// Select "c\nd" crossing the line 0/line 1 boundary.
			name:     "selectCrossLine",
			fn:       selectCrossLine,
			textarea: image.Rect(20, 10, 60, 40),
			want: []string{
				// tick/untick at 'c'
				"fill (0,0)-(3,10) [-,-1],[-,1]",
				"screen-800x600 <- draw r: (45,10)-(48,20) src: nil mask nil p1: (0,0)",
				"fill (45,10)-(48,20) [-,0],[-,1]",
				// selection fill: 'c' on line 0, partial fill to end of line
				"fill (46,10)-(59,20) [2,0],[1,1]",
				`screen-800x600 <- string "c" atpoint: (46,10) [2,0] fill: black`,
				// fill remainder of line 0 beyond 'c' (the '\n' slot)
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"fill (60,10)-(60,20) [-,0],[0,1]",
				// 'd' on line 1
				"fill (20,20)-(33,30) [0,1],[1,1]",
				`screen-800x600 <- string "d" atpoint: (20,20) [0,1] fill: black`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iv.textarea = tc.textarea
			fr := setupFrame(t, iv)

			if tc.knowntofail {
				tc.fn(t, fr, iv)
				generateVisualizedOutput(t, fr)
				t.Log("known failing: bug not yet fixed")
				return
			}

			tc.fn(t, fr, iv)

			got := gdo(t, fr).DrawOps()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

			visualizedoutputtest(t, fr)
		})
	}
}
