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
		fn          func(t *testing.T, fr Frame, iv *invariants)
		want        []string
		textarea    image.Rectangle
		knowntofail bool
	}{
		{
			// Insert text that doesn't fit.
			name: "insertPastEnd",
			fn:   insertPastEnd,
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,40) [0,1],[3,2]",
				"fill (20,40)-(20,50) [0,3],[0,1]",
				`screen-800x600 <- string "a本ポ" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "ポポh" atpoint: (20,30) [0,2] fill: black`},
			textarea: image.Rect(20, 10, 59, 40),
		},
		{
			// Split a wrapped line by inserting a newline.
			name:     "splitWrappedLine",
			fn:       splitWrappedLine,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,50) [0,1],[3,3]",
				"fill (20,50)-(33,60) [0,4],[1,1]",
				`screen-800x600 <- string "a本ポ" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "ポポh" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "ell" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "o" atpoint: (20,50) [0,4] fill: black`,
				// The previously failing insertion starts here. We didn't have to do
				// anything in this case. But we still fill blank space at the end of the
				// line over again. This is (hopefully) harmless.
				// TODO(rjk): Elide the 0-width draws.
				"fill (58,10)-(59,20) [-,0],[-,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
			},
			knowntofail: false,
		},
		{
			// Insert a single character that forces conversion of non-wrapped to
			// wrapped with wripple to end.
			name:     "insertForcesWrap",
			fn:       insertForcesWrap,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,50) [0,1],[3,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,30)-(59,50) [0,2],[3,2], to (20,40)-(59,60) [0,3],[3,2]",
				"blit (59,20)-(59,30) [3,1],[0,1], to (59,30)-(59,40) [3,2],[0,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(59,30) [1,1],[2,1]",
				"blit (46,10)-(59,20) [2,0],[1,1], to (20,20)-(33,30) [0,1],[1,1]",
				"fill (46,10)-(59,20) [2,0],[1,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "X" atpoint: (46,10) [2,0] fill: black`,
			},
		},
		{
			// Append a pair of characters at the end of the otherwise full text
			// area.
			name:     "appendAtEnd",
			fn:       appendAtEnd,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,50) [0,1],[3,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"fill (58,50)-(59,60) [-,4],[-,1]",
				// Doesn't this stick below? it's 0 wide?
				"fill (20,60)-(20,70) [0,5],[0,1]",
			},
		},

		{
			// Append a multibox string that hangs off the end. TODO(rjk): Draws a
			// zero-width fill off the end of text area. This is conceivably wrong.
			// It would (for example) make some drawing stacks unhappy.
			name:     "appendHangingLongAtEnd",
			fn:       appendHangingLongAtEnd,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,60) [0,1],[3,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4" atpoint: (20,50) [0,4] fill: black`,
				"fill (33,50)-(59,60) [1,4],[2,1]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "XX" atpoint: (33,50) [1,4] fill: black`,
			},
		},
		{
			// Insert a multibox string that forces ripple past the end.
			name:     "insertWrappedThatForcesRipple",
			fn:       insertWrappedThatForcesRipple,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,60) [0,1],[3,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3b" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4" atpoint: (20,50) [0,4] fill: black`,
				"fill (59,50)-(59,60) [3,4],[0,1]",
				"blit (33,40)-(46,50) [1,3],[1,1], to (46,50)-(59,60) [2,4],[1,1]",
				"fill (33,40)-(59,50) [1,3],[2,1]",
				"fill (20,50)-(46,60) [0,4],[2,1]",
				`screen-800x600 <- string "ij" atpoint: (33,40) [1,3] fill: black`,
				`screen-800x600 <- string "XX" atpoint: (20,50) [0,4] fill: black`,
			},
		},
		{
			// Rippled down off edge of frame of wrapped text.
			name:     "insertForcesRippleOfWrapped",
			fn:       insertForcesRippleOfWrapped,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,60) [0,1],[3,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,20)-(59,50) [0,1],[3,3], to (20,30)-(59,60) [0,2],[3,3]",
				"blit (59,10)-(59,20) [3,0],[0,1], to (59,20)-(59,30) [3,1],[0,1]",
				"blit (20,10)-(59,20) [0,0],[3,1], to (20,20)-(59,30) [0,1],[3,1]",
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(20,30) [0,1],[0,1]",
				`screen-800x600 <- string "ABC" atpoint: (20,10) [0,0] fill: black`,
			},
		},
		{
			// A long line inserted. Requires wrapping the inserted line and
			// rippling the remaining text.
			name:     "insertLongLine",
			fn:       insertLongLine,
			textarea: image.Rect(20, 10, 59, 100),
			want: []string{
				"blit (20,30)-(46,40) [0,2],[2,1], to (20,60)-(46,70) [0,5],[2,1]",
				"fill (59,50)-(59,60) [3,4],[0,1]",
				"blit (33,20)-(46,30) [1,1],[1,1], to (46,50)-(59,60) [2,4],[1,1]",
				"fill (33,20)-(59,30) [1,1],[2,1]",
				"fill (33,20)-(59,30) [1,1],[2,1]",
				"fill (20,30)-(59,50) [0,2],[3,2]",
				"fill (20,50)-(46,60) [0,4],[2,1]",
				`screen-800x600 <- string "a本" atpoint: (33,20) [1,1] fill: black`,
				`screen-800x600 <- string "ポポポ" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "hel" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "lo" atpoint: (20,50) [0,4] fill: black`,
			},
		},
		{
			// Insert into a long line.
			name:     "insertIntoLongLine",
			fn:       insertIntoLongLine,
			textarea: image.Rect(20, 10, 59, 100),
			want: []string{
				"blit (20,60)-(46,70) [0,5],[2,1], to (20,70)-(46,80) [0,6],[2,1]",
				"fill (33,60)-(59,70) [1,5],[2,1]",
				"blit (46,50)-(59,60) [2,4],[1,1], to (20,60)-(33,70) [0,5],[1,1]",
				"blit (20,50)-(46,60) [0,4],[2,1], to (33,50)-(59,60) [1,4],[2,1]",
				"fill (59,50)-(59,60) [3,4],[0,1]",
				"blit (46,40)-(59,50) [2,3],[1,1], to (20,50)-(33,60) [0,4],[1,1]",
				"blit (20,40)-(46,50) [0,3],[2,1], to (33,40)-(59,50) [1,3],[2,1]",
				"fill (59,40)-(59,50) [3,3],[0,1]",
				"blit (46,30)-(59,40) [2,2],[1,1], to (20,40)-(33,50) [0,3],[1,1]",
				"blit (20,30)-(46,40) [0,2],[2,1], to (33,30)-(59,40) [1,2],[2,1]",
				"fill (59,30)-(59,40) [3,2],[0,1]",
				"blit (46,20)-(59,30) [2,1],[1,1], to (20,30)-(33,40) [0,2],[1,1]",
				"blit (33,20)-(46,30) [1,1],[1,1], to (46,20)-(59,30) [2,1],[1,1]",
				"fill (59,20)-(59,30) [3,1],[0,1]",
				"fill (33,20)-(46,30) [1,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,20) [1,1] fill: black`,
			},
		},
		{
			// Insert a new line that pushes another newline down.
			name:     "insertsRippledNewLine",
			fn:       insertsRippledNewLine,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,50) [0,1],[3,3]",
				"fill (20,50)-(20,60) [0,4],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				"blit (20,40)-(59,50) [0,3],[3,1], to (20,50)-(59,60) [0,4],[3,1]",
				"fill (20,40)-(59,50) [0,3],[3,1]",
				"fill (20,50)-(20,60) [0,4],[0,1]",
			},
		},
		{
			// Insert a character exactly at the wrap boundary (position where the
			// first visual line is already full). Character wraps to line 2.
			name:     "insertAtExactWrapBoundary",
			fn:       insertAtExactWrapBoundary,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,50) [0,1],[3,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,30)-(59,50) [0,2],[3,2], to (20,40)-(59,60) [0,3],[3,2]",
				"blit (59,20)-(59,30) [3,1],[0,1], to (59,30)-(59,40) [3,2],[0,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(59,30) [1,1],[2,1]",
				"fill (58,10)-(59,20) [-,0],[-,1]",
				"fill (20,20)-(33,30) [0,1],[1,1]",
				`screen-800x600 <- string "X" atpoint: (20,20) [0,1] fill: black`,
			},
		},
		{
			// Insert a character that exactly fills the first visual line from 2
			// to 3 characters. No wrap should occur.
			name:     "insertExactlyFillsAlignedLine",
			fn:       insertExactlyFillsAlignedLine,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,50) [0,1],[3,3]",
				"fill (20,50)-(59,60) [0,4],[3,1]",
				`screen-800x600 <- string "0a" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				`screen-800x600 <- string "4ij" atpoint: (20,50) [0,4] fill: black`,
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,20)-(59,30) [0,1],[3,1]",
				"fill (59,10)-(59,20) [3,0],[0,1]",
				"fill (46,10)-(59,20) [2,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (46,10) [2,0] fill: black`,
			},
		},
		{
			// Insert a string that pushes a blank line off the end.
			name:     "insertPushesBlankLineOffEnd",
			fn:       insertPushesBlankLineOffEnd,
			textarea: image.Rect(20, 10, 59, 60),
			want: []string{
				"fill (20,10)-(59,20) [0,0],[3,1]",
				"fill (20,20)-(59,60) [0,1],[3,4]",
				"fill (20,60)-(20,70) [0,5],[0,1]",
				`screen-800x600 <- string "0ab" atpoint: (20,10) [0,0] fill: black`,
				`screen-800x600 <- string "1cd" atpoint: (20,20) [0,1] fill: black`,
				`screen-800x600 <- string "2ef" atpoint: (20,30) [0,2] fill: black`,
				`screen-800x600 <- string "3gh" atpoint: (20,40) [0,3] fill: black`,
				"blit (20,30)-(59,50) [0,2],[3,2], to (20,40)-(59,60) [0,3],[3,2]",
				"blit (59,20)-(59,30) [3,1],[0,1], to (59,30)-(59,40) [3,2],[0,1]",
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,30)-(59,40) [0,2],[3,1]",
				"fill (33,20)-(59,30) [1,1],[2,1]",
				"blit (46,10)-(59,20) [2,0],[1,1], to (20,20)-(33,30) [0,1],[1,1]",
				"blit (33,10)-(46,20) [1,0],[1,1], to (46,10)-(59,20) [2,0],[1,1]",
				"fill (59,10)-(59,20) [3,0],[0,1]",
				"fill (33,10)-(46,20) [1,0],[1,1]",
				`screen-800x600 <- string "X" atpoint: (33,10) [1,0] fill: black`,
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
				t.Fail()
				return
			}

			// TODO(rjk): validate here

			tc.fn(t, fr, iv)

			// TODO(rjk): validate here

			// Peek inside.
			got := gdo(t, fr).DrawOps()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

			visualizedoutputtest(t, fr)
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
			textarea:     image.Rect(20, 10, 59, 40),
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
