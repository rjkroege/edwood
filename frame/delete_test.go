package frame

import (
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func deleteSingleCharacterAtLineEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(2, 3)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteSingleCharacterInMiddle(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteNewlineTocreateWrappedLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(len("0ab"), len("0ab\n"))

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func rippleUpDeletedChar(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0ab1cd2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // a

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteTab(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	t.Log(fr.GetMaxtab())

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0	ab1cd2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // a

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteCharBeforeTab(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	t.Log(fr.GetMaxtab())

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0a	b1cd2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // a

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func rippleUpMultiLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0a\nb1\ncd2\nef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(0, 6) // 0a\nb1\n

	if got, want := s, 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDelete is a high-level Dete test
func TestDelete(t *testing.T) {
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
			// Delete a single character at line end as we'd see with a backspace
			// key press.
			name: "deleteSingleCharacterAtLineEnd",
			fn:   deleteSingleCharacterAtLineEnd,
			want: []string{
				"fill (46,10)-(59,20) [2,0],[1,1]",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete a single character in the middle of a terminal line.
			name: "deleteSingleCharacterInMiddle",
			fn:   deleteSingleCharacterInMiddle,
			want: []string{
				"blit (46,10)-(59,20) [2,0],[1,1], to (33,10)-(46,20) [1,0],[1,1]",
				"fill (46,10)-(46,20) [2,0],[0,1]",
				"fill (46,10)-(59,20) [2,0],[1,1]",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete a newline to create a wrapped line. TODO(rjk): This op blits a
			// line to itself. This is visually fine but is wasted work. None of the
			// drawops generated here are necessary for a correct screen update.
			name: "deleteNewlineTocreateWrappedLine",
			fn:   deleteNewlineTocreateWrappedLine,
			want: []string{
				"blit (20,20)-(59,30) [0,1],[3,1], to (20,20)-(59,30) [0,1],[3,1]",
				"fill (59,20)-(59,30) [3,1],[0,1]",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},

		{
			// Ripple up a single deleted character.
			name: "rippleUpDeletedChar",
			fn:   rippleUpDeletedChar,
			want: []string{
				"blit (46,10)-(59,20) [2,0],[1,1], to (33,10)-(46,20) [1,0],[1,1]",
				"fill (46,10)-(46,20) [2,0],[0,1]",
				"blit (20,20)-(33,30) [0,1],[1,1], to (46,10)-(59,20) [2,0],[1,1]",
				"fill (59,10)-(60,20) [3,0],[-,1]",
				"blit (33,20)-(59,30) [1,1],[2,1], to (20,20)-(46,30) [0,1],[2,1]",
				"fill (46,20)-(46,30) [2,1],[0,1]",
				"blit (20,30)-(33,40) [0,2],[1,1], to (46,20)-(59,30) [2,1],[1,1]",
				"fill (59,20)-(60,30) [3,1],[-,1]",
				"blit (33,30)-(59,40) [1,2],[2,1], to (20,30)-(46,40) [0,2],[2,1]",
				"fill (46,30)-(46,40) [2,2],[0,1]",
				"fill (46,30)-(59,40) [2,2],[1,1]",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// character followed by tab where character after tab shouldn't move, delete the tab
			// have to make this wide enough for tabs to work.
			name: "deleteTab",
			fn:   deleteTab,
			want: []string{
				"blit (124,10)-(137,20) [8,0],[1,1], to (33,10)-(46,20) [1,0],[1,1]",
				"fill (46,10)-(46,20) [2,0],[0,1]",
				"blit (20,20)-(111,30) [0,1],[7,1], to (46,10)-(137,20) [2,0],[7,1]",
				"fill (137,10)-(137,20) [9,0],[0,1]",
				"fill (137,10)-(140,20) [9,0],[-,1]",
				"fill (20,20)-(111,30) [0,1],[7,1]",
				"fill (137,10)-(140,20) [9,0],[-,1]",
				"fill (20,20)-(111,30) [0,1],[7,1]",
			},
			// Has to be wide enough to accommodate a tab. Tab is 8 * 13 charwidths = 104.
			textarea: image.Rect(20, 10, 140, 40),
		},
		{
			// Character followed by tab where character after tab shouldn't move,
			// delete the character, tab should stretch.
			name: "deleteCharBeforeTab",
			fn:   deleteCharBeforeTab,
			want: []string{
				"fill (33,10)-(124,20) [1,0],[7,1]",
			},
			// Has to be wide enough to accommodate a tab. Tab is 8 * 13 charwidths = 104.
			textarea: image.Rect(20, 10, 140, 40),
		},
		{
			// Ripple up a multiline deletion, text off the bottom.
			name: "rippleUpMultiLine",
			fn:   rippleUpMultiLine,
			want: []string{
				"blit (20,30)-(60,40) [0,2],[-,1], to (20,10)-(60,20) [0,0],[-,1]",
				"blit (20,40)-(60,40) [0,3],[-,0], to (20,20)-(60,20) [0,1],[-,0]",
				"fill (20,20)-(60,30) [0,1],[-,1]",
				"fill (20,30)-(60,40) [0,2],[-,1]",
				"fill (20,40)-(20,50) [0,3],[0,1]",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		// Rippling tabs
		// Tabs in narrow columns (what are they even suppose to do?)
		// Need Tab insertion tests too (At beginning of document, into a narrow Window, forcing ripple)
		// character followed by tab where character after tab shouldn't move, delete the character
		// chunk, range, chunk (what does that mean?)
		// blank line rippling
		// delete whole line
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
}
