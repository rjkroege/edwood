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
