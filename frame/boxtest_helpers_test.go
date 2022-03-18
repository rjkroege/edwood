package frame

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

const fixedwidth = 10

// makeBox creates somewhat realistic test boxes in 10pt fixed width font.
func makeBox(s string) *frbox {
	r, _ := utf8.DecodeRuneInString(s)

	switch s {
	case "\t":
		return &frbox{
			Wid:    10,
			Nrune:  -1,
			Bc:     r,
			Minwid: 10,
		}

	case "\n":
		return &frbox{
			Wid:    10000,
			Nrune:  -1,
			Bc:     r,
			Minwid: 0,
		}
	default:
		nrune := strings.Count(s, "") - 1
		return &frbox{
			Wid:   fixedwidth * nrune,
			Nrune: nrune,
			Ptr:   []byte(s),
			// Remaining fields not used.
		}
	}
}

// mockFont returns a mock of a fixed width 13px high 10px wide font.
func mockFont() draw.Font {
	return edwoodtest.NewFont(fixedwidth, 13)
}

// BoxTester specifies an abstract interface to each specific test.
type BoxTester interface {
	// Try runs the test.
	Try() interface{}

	// See if the Try() did the correct thing.
	Verify(*testing.T, string, interface{})
}

// comparecore runs the Try and Verify over an array of BoxTester implementations.
func comparecore(t *testing.T, prefix string, testvector []BoxTester) {
	for _, tv := range testvector {
		result := tv.Try()
		tv.Verify(t, prefix, result)
	}
}

// expectedboxesequal tests that the expected box slice afterboxes equals the
// computed box found in frame. prefix and name describe the test and i is the
// box index.
func expectedboxesequal(t *testing.T, prefix, name string, i int, frame *frameimpl, afterboxes []*frbox) {
	if got, want := frame.box[i], afterboxes[i]; !reflect.DeepEqual(got, want) {
		switch {
		case got == nil && want != nil:
			t.Errorf("%s-%s: result box [%d] mismatch: got nil want %#v (%s)", prefix, name, i, want, string(want.Ptr))
		case got != nil && want == nil:
			t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%s) want nil", prefix, name, i, got, string(got.Ptr))
		case got.Ptr == nil && want.Ptr == nil:
			t.Errorf("%s-%s: result box [%d] mismatch: got %#v (nil) want %#v (nil)", prefix, name, i, got, want)
		case got.Ptr == nil && want.Ptr != nil:
			t.Errorf("%s-%s: result box [%d] mismatch: got %#v (nil) want %#v (%s)", prefix, name, i, got, want, string(want.Ptr))
		case want.Ptr == nil && got.Ptr != nil:
			t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%s) want %#v (nil)", prefix, name, i, got, string(got.Ptr), want)
		case want.Ptr != nil && got.Ptr != nil:
			t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%q) want %#v (%q)", prefix, name, i, got, string(got.Ptr), want, string(want.Ptr))
		}
	}
}

// testcore checks if the frame's box model matches the provided afterboxes, nbox, Use this to implement Verify methods.
func testcore(t *testing.T, prefix, name string, frame *frameimpl, nbox int, afterboxes []*frbox) {
	if got, want := len(frame.box), nbox; got != want {
		t.Errorf("%s-%s: len(frame.box) got %d but want %d\n", prefix, name, got, want)
	}
	if frame.box == nil {
		t.Errorf("%s-%s: ran add but did not succeed in creating boxex", prefix, name)
	}

	// First part of box array must match the provided afterboxes slice.
	for i := range afterboxes {
		expectedboxesequal(t, prefix, name, i, frame, afterboxes)
	}

	// Remaining part of box array must merely exist.
	for i, b := range frame.box[len(afterboxes):] {
		if b != nil {
			t.Errorf("%s-%s: result box [%d] should be nil", prefix, name, i+len(afterboxes))
		}
	}
}
