// Tests specific to the file.File implementation.
package file

import (
	"testing"

	"github.com/rjkroege/edwood/sam"
)

// TestFileHandlesNilEpsilonDelta shows that File.Undo correctly reports selection values across Undo
// actions and does not fail if the File's delta or epsilon arrays are
// nil. This test is derived from TestWindowUndoSelection. That test was
// fragile as it reached into specific details of the file.File
// implementation where file.File should have been an opaque detail of
// the implementation of the file package.
func TestFileHandlesNilEpsilonDelta(t *testing.T) {
	var (
		word = RuneArray("hello")
		p0   = 3
		undo = &Undo{
			T:   sam.Insert,
			Buf: word,
			P0:  p0,
			N:   word.Nc(),
		}
	)
	for _, tc := range []struct {
		name           string
		isundo         bool
		q0, q1         int
		wantQ0, wantQ1 int
		delta, epsilon []*Undo
	}{
		{"undo", true, 14, 17, p0, p0 + word.Nc(), []*Undo{undo}, nil},
		{"redo", false, 14, 17, p0, p0 + word.Nc(), nil, []*Undo{undo}},
		{"undo (nil delta)", true, 14, 17, 14, 17, nil, nil},
		{"redo (nil epsilon)", false, 14, 17, 14, 17, nil, nil},
	} {
		oeb := MakeObservableEditableBuffer("", []rune("This is an example sentence.\n"))
		oeb.f.delta = tc.delta
		oeb.f.epsilon = tc.epsilon
		
		q0, q1 := tc.q0, tc.q1
		if nq0, nq1, hazselection := oeb.Undo(tc.isundo); hazselection {
			q0, q1 = nq0, nq1
		}

		if q0 != tc.wantQ0 || q1 != tc.wantQ1 {
			t.Errorf("%v changed q0, q1 to %v, %v; want %v, %v",
				tc.name, q0, q1, tc.wantQ0, tc.wantQ1)
		}
	}
}
