package frame

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"9fans.net/go/draw"
)

const fixedwidth = 10

// makeBox creates somewhat realistic test boxes in 10pt fixed width font.
func makeBox(s string) *frbox {
	r, _ := utf8.DecodeRuneInString(s)

	switch s {
	case "\t":
		return &frbox{
			Wid:    5000,
			Nrune:  -1,
			Ptr:    []byte(s),
			Bc:     r,
			Minwid: 10,
		}

	case "\n":
		return &frbox{
			Wid:    5000,
			Nrune:  -1,
			Ptr:    []byte(s),
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

// Fakemetrics mocks Fontmetrics as a fixed width 13px high 10px wide font.
type Fakemetrics int

func (w Fakemetrics) BytesWidth(s []byte) int {
	return int(w) * (strings.Count(string(s), "") - 1)
}

func (w Fakemetrics) DefaultHeight() int { return 13 }

func (w Fakemetrics) Impl() *draw.Font { return nil }

func (w Fakemetrics) StringWidth(s string) int {
	return int(w) * (strings.Count(s, "") - 1)
}

func (w Fakemetrics) RunesWidth(r []rune) int {
	return len(r) * int(w)
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
func expectedboxesequal(t *testing.T, prefix, name string, i int, frame *Frame, afterboxes []*frbox) {
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
			t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%s) want %#v (%s)", prefix, name, i, got, string(got.Ptr), want, string(want.Ptr))
		}
	}
}

// testcore checks if the frame's box model matches the provided afterboxes, nbox, nalloc. Use this to implement Verify methods.
func testcore(t *testing.T, prefix, name string, frame *Frame, nbox, nalloc int, afterboxes []*frbox) {
	if got, want := frame.nbox, nbox; got != want {
		t.Errorf("%s-%s: nbox got %d but want %d\n", prefix, name, got, want)
	}
	if got, want := frame.nalloc, nalloc; got != want {
		t.Errorf("%s-%s: nalloc got %d but want %d\n", prefix, name, got, want)
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
