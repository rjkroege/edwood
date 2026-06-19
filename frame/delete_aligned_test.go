package frame

import (
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestDeleteAligned runs delete scenarios in a frame whose width is an exact
// multiple of the character width (39 px = 3 × 13 px), exercising the
// exact-boundary paths in cklinewrap, canfit, and fillNonGlyphAreas.
func TestDeleteAligned(t *testing.T) {
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
			name:     "deleteSingleCharacterAtLineEnd",
			fn:       deleteSingleCharacterAtLineEnd,
			textarea: image.Rect(20, 10, 44, 55),
			want: []string{
				"fill (36,10)-(44,25) [2,0],[1,1]",
			},
		},
		{
			name:     "deleteSingleCharacterInMiddle",
			fn:       deleteSingleCharacterInMiddle,
			textarea: image.Rect(20, 10, 44, 55),
			want: []string{
				"blit (36,10)-(44,25) [2,0],[1,1], to (28,10)-(36,25) [1,0],[1,1]",
				"fill (36,10)-(36,25) [2,0],[0,1]",
				"fill (36,10)-(44,25) [2,0],[1,1]",
			},
		},
		{
			// In the aligned frame the merged line "0ab1cd" soft-wraps at the
			// exact right edge (39 px), so the fill at the end of "1cd" is
			// zero-width ([0,1] in character units).
			name:     "deleteNewlineTocreateWrappedLine",
			fn:       deleteNewlineTocreateWrappedLine,
			textarea: image.Rect(20, 10, 44, 55),
			want: []string{
				"blit (20,25)-(44,40) [0,1],[3,1], to (20,25)-(44,40) [0,1],[3,1]",
				"fill (44,25)-(44,40) [3,1],[0,1]",
			},
		},
		{
			// Each ripple blit copies content that ends exactly at the frame
			// right edge, so the trailing fills are all zero-width.
			name:     "rippleUpDeletedChar",
			fn:       rippleUpDeletedChar,
			textarea: image.Rect(20, 10, 44, 55),
			want: []string{
				"blit (36,10)-(44,25) [2,0],[1,1], to (28,10)-(36,25) [1,0],[1,1]",
				"fill (36,10)-(36,25) [2,0],[0,1]",
				"blit (20,25)-(28,40) [0,1],[1,1], to (36,10)-(44,25) [2,0],[1,1]",
				"fill (44,10)-(44,25) [3,0],[0,1]",
				"blit (28,25)-(44,40) [1,1],[2,1], to (20,25)-(36,40) [0,1],[2,1]",
				"fill (36,25)-(36,40) [2,1],[0,1]",
				"blit (20,40)-(28,55) [0,2],[1,1], to (36,25)-(44,40) [2,1],[1,1]",
				"fill (44,25)-(44,40) [3,1],[0,1]",
				"blit (28,40)-(44,55) [1,2],[2,1], to (20,40)-(36,55) [0,2],[2,1]",
				"fill (36,40)-(36,55) [2,2],[0,1]",
				"fill (36,40)-(44,55) [2,2],[1,1]",
			},
		},
		{
			name:     "rippleUpMultiLine",
			fn:       rippleUpMultiLine,
			textarea: image.Rect(20, 10, 44, 55),
			want: []string{
				"blit (20,40)-(44,55) [0,2],[3,1], to (20,10)-(44,25) [0,0],[3,1]",
				"blit (20,55)-(44,55) [0,3],[3,0], to (20,25)-(44,25) [0,1],[3,0]",
				"fill (20,25)-(44,40) [0,1],[3,1]",
				"fill (20,40)-(44,55) [0,2],[3,1]",
				"fill (20,55)-(20,70) [0,3],[0,1]",
			},
		},
		{
			// The soft-wrap cancellation path with exact alignment: after the
			// delete the first logical line fits in exactly 39 px, leaving a
			// zero-width fill at the frame right edge.
			name:     "deleteEliminatesSoftWrap",
			fn:       deleteEliminatesSoftWrap,
			textarea: image.Rect(20, 10, 44, 55),
			want: []string{
				"fill (20,25)-(44,40) [0,1],[3,1]",
				"blit (20,40)-(44,55) [0,2],[3,1], to (20,25)-(44,40) [0,1],[3,1]",
				"blit (20,55)-(44,55) [0,3],[3,0], to (20,40)-(44,40) [0,2],[3,0]",
				"fill (43,25)-(44,40) [-,1],[-,1]",
				"fill (20,40)-(44,55) [0,2],[3,1]",
			},
		},
		{
			// Aligned tab tests use a frame 9 × 13 px = 117 px wide (right
			// edge at x = 137).
			name:     "deleteTab",
			fn:       deleteTab,
			textarea: image.Rect(20, 10, 92, 55),
			want: []string{
				"blit (84,10)-(92,25) [8,0],[1,1], to (28,10)-(36,25) [1,0],[1,1]",
				"fill (36,10)-(36,25) [2,0],[0,1]",
				"blit (20,25)-(76,40) [0,1],[7,1], to (36,10)-(92,25) [2,0],[7,1]",
				"fill (92,10)-(92,25) [9,0],[0,1]",
				"fill (91,10)-(92,25) [-,0],[-,1]",
				"fill (20,25)-(76,40) [0,1],[7,1]",
				"fill (91,10)-(92,25) [-,0],[-,1]",
				"fill (20,25)-(76,40) [0,1],[7,1]",
			},
		},
		{
			name:     "deleteCharBeforeTab",
			fn:       deleteCharBeforeTab,
			textarea: image.Rect(20, 10, 92, 55),
			want: []string{
				"fill (28,10)-(84,25) [1,0],[7,1]",
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

			tc.fn(t, fr, iv, tc.name)

			got := gdo(t, fr).DrawOps()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

			visualizedoutputtest(t, fr)
			snapAfterPNG(t, fr, tc.name)
		})
	}
}
