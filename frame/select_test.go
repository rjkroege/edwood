package frame

import (
	"image"
	"testing"

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
		name     string
		fn       func(*testing.T, Frame, *invariants)
		textarea image.Rectangle
	}{
		{
			name:     "selectSingleCharacterAtLineEnd",
			fn:       selectSingleCharacterAtLineEnd,
			textarea: image.Rect(20, 10, 60, 40),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iv.textarea = tc.textarea
			fr := setupFrame(t, iv)
			tc.fn(t, fr, iv)
			visualizedoutputtest(t, fr)
		})
	}
}
