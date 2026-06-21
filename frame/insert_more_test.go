package frame

import (
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TODO(rjk): Test having a height that's not a multiple of the font
// height. Particularly relevant for supporting lines of differing
// heights.

// TestInsertAligned is a high-level Insert test that uses a frame where
// the character edge aligns with the width of the text region.
func TestInsertAligned(t *testing.T) {
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
		{
			// Insert text that doesn't fit.
			name: "insertPastEnd",
			fn:   insertPastEnd,
			want: []string{
				"fill (20,10)-(44,25) [0,0],[3,1]",
				"fill (20,25)-(44,55) [0,1],[3,2]",
				"fill (20,55)-(20,70) [0,3],[0,1]",
				`screen-800x600 <- string "a本ポ" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,25) [0,1] fill: black`,
				`screen-800x600 <- string "ポポh" atpoint: (20,40) [0,2] fill: black`},
			textarea: image.Rect(20, 10, 44, 55),
		},
		{
			// Split a wrapped line by inserting a newline.
			name:     "splitWrappedLine",
			fn:       splitWrappedLine,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				// The previously failing insertion starts here. We didn't have to do
				// anything in this case. But we still fill blank space at the end of the
				// line over again. This is (hopefully) harmless.
				// TODO(rjk): Elide the 0-width draws.
				"fill (43,10)-(44,25) [-,0],[-,1]",
				"fill (20,25)-(20,40) [0,1],[0,1]",
			},
			knowntofail: false,
		},
		{
			// Insert a single character that forces conversion of non-wrapped to
			// wrapped with wripple to end.
			name:     "insertForcesWrap",
			fn:       insertForcesWrap,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"blit (20,40)-(44,70) [0,2],[3,2], to (20,55)-(44,85) [0,3],[3,2]",
				"blit (44,25)-(44,40) [3,1],[0,1], to (44,40)-(44,55) [3,2],[0,1]",
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,40)-(44,55) [0,2],[3,1]",
				"fill (28,25)-(44,40) [1,1],[2,1]",
				"blit (36,10)-(44,25) [2,0],[1,1], to (20,25)-(28,40) [0,1],[1,1]",
				"fill (36,10)-(44,25) [2,0],[1,1]",
				"fill (20,25)-(20,40) [0,1],[0,1]",
				`screen-800x600 <- string "X" atpoint: (36,10) [2,0] fill: black`,
			},
		},
		{
			// Append a pair of characters at the end of the otherwise full text
			// area.
			name:     "appendAtEnd",
			fn:       appendAtEnd,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"fill (43,70)-(44,85) [-,4],[-,1]",
				// Doesn't this stick below? it's 0 wide?
				"fill (20,85)-(20,100) [0,5],[0,1]",
			},
		},

		{
			// Append a multibox string that hangs off the end. TODO(rjk): Draws a
			// zero-width fill off the end of text area. This is conceivably wrong.
			// It would (for example) make some drawing stacks unhappy.
			name:     "appendHangingLongAtEnd",
			fn:       appendHangingLongAtEnd,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"fill (28,70)-(44,85) [1,4],[2,1]",
				"fill (20,85)-(20,100) [0,5],[0,1]",
				`screen-800x600 <- string "XX" atpoint: (28,70) [1,4] fill: black`,
			},
		},
		{
			// Insert a multibox string that forces ripple past the end.
			name:     "insertWrappedThatForcesRipple",
			fn:       insertWrappedThatForcesRipple,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"fill (44,70)-(44,85) [3,4],[0,1]",
				"blit (28,55)-(36,70) [1,3],[1,1], to (36,70)-(44,85) [2,4],[1,1]",
				"fill (28,55)-(44,70) [1,3],[2,1]",
				"fill (20,70)-(36,85) [0,4],[2,1]",
				`screen-800x600 <- string "ij" atpoint: (28,55) [1,3] fill: black`,
				`screen-800x600 <- string "XX" atpoint: (20,70) [0,4] fill: black`,
			},
		},
		{
			// Rippled down off edge of frame of wrapped text.
			name:     "insertForcesRippleOfWrapped",
			fn:       insertForcesRippleOfWrapped,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"blit (20,25)-(44,70) [0,1],[3,3], to (20,40)-(44,85) [0,2],[3,3]",
				"blit (44,10)-(44,25) [3,0],[0,1], to (44,25)-(44,40) [3,1],[0,1]",
				"blit (20,10)-(44,25) [0,0],[3,1], to (20,25)-(44,40) [0,1],[3,1]",
				"fill (20,10)-(44,25) [0,0],[3,1]",
				"fill (20,10)-(44,25) [0,0],[3,1]",
				"fill (20,25)-(20,40) [0,1],[0,1]",
				`screen-800x600 <- string "ABC" atpoint: (20,10) [0,0] fill: black`,
			},
		},
		{
			// A long line inserted. Requires wrapping the inserted line and
			// rippling the remaining text.
			name:     "insertLongLine",
			fn:       insertLongLine,
			textarea: image.Rect(20, 10, 376, 100),
			want: []string{
				"blit (20,40)-(36,55) [0,2],[2,1], to (20,55)-(36,70) [0,3],[2,1]",
				"fill (92,40)-(376,55) [9,2],[-,1]",
				"blit (28,25)-(36,40) [1,1],[1,1], to (84,40)-(92,55) [8,2],[1,1]",
				"fill (28,25)-(376,40) [1,1],[-,1]",
				"fill (20,40)-(84,55) [0,2],[8,1]",
				`screen-800x600 <- string "a本ポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポポ" atpoint: (28,25) [1,1] fill: black`,
				`screen-800x600 <- string "ポポポhello" atpoint: (20,40) [0,2] fill: black`,
			},
		},
		{
			// Insert into a long line.
			name:     "insertIntoLongLine",
			fn:       insertIntoLongLine,
			textarea: image.Rect(20, 10, 376, 100),
			want: []string{
				"blit (20,55)-(36,70) [0,3],[2,1], to (20,55)-(36,70) [0,3],[2,1]",
				"fill (100,40)-(376,55) [10,2],[-,1]",
				"blit (20,40)-(92,55) [0,2],[9,1], to (28,40)-(100,55) [1,2],[9,1]",
				"blit (364,25)-(372,40) [43,1],[1,1], to (20,40)-(28,55) [0,2],[1,1]",
				"blit (28,25)-(364,40) [1,1],[42,1], to (36,25)-(372,40) [2,1],[42,1]",
				"fill (372,25)-(376,40) [44,1],[-,1]",
				"fill (28,25)-(36,40) [1,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (28,25) [1,1] fill: black`,
			},
		},
		{
			// Insert a new line that pushes another newline down.
			name:     "insertsRippledNewLine",
			fn:       insertsRippledNewLine,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"blit (20,55)-(44,70) [0,3],[3,1], to (20,70)-(44,85) [0,4],[3,1]",
				"fill (20,55)-(44,70) [0,3],[3,1]",
				"fill (20,70)-(20,85) [0,4],[0,1]",
			},
		},
		{
			// Insert a character exactly at the wrap boundary (position where the
			// first visual line is already full). Character wraps to line 2.
			name:     "insertAtExactWrapBoundary",
			fn:       insertAtExactWrapBoundary,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"blit (20,40)-(44,70) [0,2],[3,2], to (20,55)-(44,85) [0,3],[3,2]",
				"blit (44,25)-(44,40) [3,1],[0,1], to (44,40)-(44,55) [3,2],[0,1]",
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,40)-(44,55) [0,2],[3,1]",
				"fill (28,25)-(44,40) [1,1],[2,1]",
				"fill (43,10)-(44,25) [-,0],[-,1]",
				"fill (20,25)-(28,40) [0,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (20,25) [0,1] fill: black`,
			},
		},
		{
			// Insert a character that exactly fills the first visual line from 2
			// to 3 characters. No wrap should occur.
			name:     "insertExactlyFillsAlignedLine",
			fn:       insertExactlyFillsAlignedLine,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,25)-(44,40) [0,1],[3,1]",
				"fill (44,10)-(44,25) [3,0],[0,1]",
				"fill (36,10)-(44,25) [2,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (36,10) [2,0] fill: black`,
			},
		},
		{
			// Insert a string that pushes a blank line off the end.
			name:     "insertPushesBlankLineOffEnd",
			fn:       insertPushesBlankLineOffEnd,
			textarea: image.Rect(20, 10, 44, 85),
			want: []string{
				"blit (20,40)-(44,70) [0,2],[3,2], to (20,55)-(44,85) [0,3],[3,2]",
				"blit (44,25)-(44,40) [3,1],[0,1], to (44,40)-(44,55) [3,2],[0,1]",
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,40)-(44,55) [0,2],[3,1]",
				"fill (28,25)-(44,40) [1,1],[2,1]",
				"blit (36,10)-(44,25) [2,0],[1,1], to (20,25)-(28,40) [0,1],[1,1]",
				"blit (28,10)-(36,25) [1,0],[1,1], to (36,10)-(44,25) [2,0],[1,1]",
				"fill (44,10)-(44,25) [3,0],[0,1]",
				"fill (28,10)-(36,25) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (28,10) [1,0] fill: black`,
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

			visualizedoutputtest(t, fr)
			snapAfterPNG(t, fr, tc.name)
		})
	}
}

// TestInsertBoxModel examines the frame's box array after insertion to
// determine whether text that falls beyond the visible area is stored.
func TestInsertBoxModel(t *testing.T) {
	iv := &invariants{
		topcorner: image.Pt(20, 10),
	}
	*validate = true

	tests := []struct {
		name         string
		fn           func(t *testing.T, fr Frame, iv *invariants)
		textarea     image.Rectangle
		wantNbox     int
		wantNchars   int
		wantNlines   int
		wantLastFull bool
	}{
		{
			// "0a\nb1\ncd2\nef" fills the 3-line frame exactly: cd2\n advances Y
			// to rect.Max.Y=40, so "ef" arrives at the boundary and _draw chops
			// it. No box for "ef" should exist; nchars should count only the 10
			// visible characters.
			name: "occludedTextNotStored",
			fn: func(t *testing.T, fr Frame, iv *invariants) {
				t.Helper()
				fr.Insert([]rune("0a\nb1\ncd2\nef"), 0)
			},
			textarea:     image.Rect(20, 10, 44, 55),
			wantNbox:     6, // "0a" "\n" "b1" "\n" "cd2" "\n" — no "ef"
			wantNchars:   10,
			wantNlines:   3,
			wantLastFull: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iv.textarea = tc.textarea
			fr := setupFrame(t, iv)

			tc.fn(t, fr, iv)

			frimpl := fr.(*frameimpl)
			for i, b := range frimpl.box {
				t.Logf("box[%d] = %v", i, b)
			}
			t.Logf("nchars=%d nlines=%d lastlinefull=%v", frimpl.nchars, frimpl.nlines, frimpl.lastlinefull)

			if got, want := len(frimpl.box), tc.wantNbox; got != want {
				t.Errorf("len(box): got %d, want %d", got, want)
			}
			if got, want := frimpl.nchars, tc.wantNchars; got != want {
				t.Errorf("nchars: got %d, want %d", got, want)
			}
			if got, want := frimpl.nlines, tc.wantNlines; got != want {
				t.Errorf("nlines: got %d, want %d", got, want)
			}
			if got, want := frimpl.lastlinefull, tc.wantLastFull; got != want {
				t.Errorf("lastlinefull: got %v, want %v", got, want)
			}
		})
	}
}
