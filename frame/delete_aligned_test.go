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
		fn          func(t *testing.T, fr Frame, iv *invariants)
		want        []string
		textarea    image.Rectangle
		knowntofail bool
	}{
		{
			name:     "deleteSingleCharacterAtLineEnd",
			fn:       deleteSingleCharacterAtLineEnd,
			textarea: image.Rect(20, 10, 59, 40),
			want: []string{
				"fill (46,10)-(59,20)",
			},
		},
		{
			name:     "deleteSingleCharacterInMiddle",
			fn:       deleteSingleCharacterInMiddle,
			textarea: image.Rect(20, 10, 59, 40),
			want: []string{
				"blit (46,10)-(59,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"fill (46,10)-(59,20)",
			},
		},
		{
			// In the aligned frame the merged line "0ab1cd" soft-wraps at the
			// exact right edge (39 px), so the fill at the end of "1cd" is
			// zero-width ([0,1] in character units).
			name:     "deleteNewlineTocreateWrappedLine",
			fn:       deleteNewlineTocreateWrappedLine,
			textarea: image.Rect(20, 10, 59, 40),
			want: []string{
				"blit (20,20)-(59,30), to (20,20)-(59,30)",
				"fill (59,20)-(59,30)",
			},
		},
		{
			// Each ripple blit copies content that ends exactly at the frame
			// right edge, so the trailing fills are all zero-width.
			name:     "rippleUpDeletedChar",
			fn:       rippleUpDeletedChar,
			textarea: image.Rect(20, 10, 59, 40),
			want: []string{
				"blit (46,10)-(59,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"blit (20,20)-(33,30), to (46,10)-(59,20)",
				"fill (59,10)-(59,20)",
				"blit (33,20)-(59,30), to (20,20)-(46,30)",
				"fill (46,20)-(46,30)",
				"blit (20,30)-(33,40), to (46,20)-(59,30)",
				"fill (59,20)-(59,30)",
				"blit (33,30)-(59,40), to (20,30)-(46,40)",
				"fill (46,30)-(46,40)",
				"fill (46,30)-(59,40)",
			},
		},
		{
			name:     "rippleUpMultiLine",
			fn:       rippleUpMultiLine,
			textarea: image.Rect(20, 10, 59, 40),
			want: []string{
				"blit (20,30)-(59,40), to (20,10)-(59,20)",
				"blit (20,40)-(59,40), to (20,20)-(59,20)",
				"fill (20,20)-(59,30)",
				"fill (20,30)-(59,40)",
				"fill (20,40)-(20,50)",
			},
		},
		{
			// The soft-wrap cancellation path with exact alignment: after the
			// delete the first logical line fits in exactly 39 px, leaving a
			// zero-width fill at the frame right edge.
			name:     "deleteEliminatesSoftWrap",
			fn:       deleteEliminatesSoftWrap,
			textarea: image.Rect(20, 10, 59, 40),
			want: []string{
				"fill (20,20)-(59,30)",
				"blit (20,30)-(59,40), to (20,20)-(59,30)",
				"blit (20,40)-(59,40), to (20,30)-(59,30)",
				"fill (58,20)-(59,30)",
				"fill (20,30)-(59,40)",
			},
		},
		{
			// Aligned tab tests use a frame 9 × 13 px = 117 px wide (right
			// edge at x = 137).
			name:     "deleteTab",
			fn:       deleteTab,
			textarea: image.Rect(20, 10, 137, 40),
			want: []string{
				"blit (124,10)-(137,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"blit (20,20)-(111,30), to (46,10)-(137,20)",
				"fill (137,10)-(137,20)",
				"fill (136,10)-(137,20)",
				"fill (20,20)-(111,30)",
				"fill (136,10)-(137,20)",
				"fill (20,20)-(111,30)",
			},
		},
		{
			name:     "deleteCharBeforeTab",
			fn:       deleteCharBeforeTab,
			textarea: image.Rect(20, 10, 137, 40),
			want: []string{
				"fill (33,10)-(124,20)",
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
