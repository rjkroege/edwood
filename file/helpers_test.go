package file

import (
	"fmt"
	"strings"
	"testing"
)

type checkable interface {
	BufferAdapter

	// Return the entire backing as a string.
	readwholefile(*testing.T) string

	// Return true to enable tests of UncommittedChanges. This concept does not
	// exist with file.Buffer.
	commitisgermane() bool
}

type stateSummary struct {
	HasUncommitedChanges bool
	HasUndoableChanges   bool
	HasRedoableChanges   bool
	SaveableAndDirty     bool
	filecontents         string
}

func (f *File) commitisgermane() bool { return true }

func (f *File) readwholefile(t *testing.T) string {
	t.Helper()
	var sb strings.Builder

	// Currently ReadAtRune does not return runes in the cache.
	for i := 0; i < f.Nr(); i++ {
		sb.WriteRune(f.ReadC(i))
	}
	return sb.String()
}

func (b *Buffer) commitisgermane() bool { return false }

func (b *Buffer) readwholefile(*testing.T) string {
	return b.String()
}

func check(t *testing.T, testname string, oeb *ObservableEditableBuffer, fss *stateSummary) {
	t.Helper()

	// Lets the test infrastructure call against file.Buffer or file.File.
	f := oeb.f.(checkable)

	if f.commitisgermane() {
		if got, want := oeb.HasUncommitedChanges(), fss.HasUncommitedChanges; got != want {
			t.Errorf("%s: HasUncommitedChanges failed. got %v want %v", testname, got, want)
		}
	}
	if got, want := oeb.HasUndoableChanges(), fss.HasUndoableChanges; got != want {
		t.Errorf("%s: HasUndoableChanges failed. got %v want %v", testname, got, want)
	}
	if got, want := oeb.HasRedoableChanges(), fss.HasRedoableChanges; got != want {
		t.Errorf("%s: HasUndoableChanges failed. got %v want %v", testname, got, want)
	}
	if got, want := oeb.SaveableAndDirty(), fss.SaveableAndDirty; got != want {
		t.Errorf("%s: SaveableAndDirty failed. got %v want %v", testname, got, want)
	}
	if got, want := f.readwholefile(t), fss.filecontents; got != want {
		t.Errorf("%s: File contents not expected. got «%#v» want «%#v»", testname, got, want)
	}
}

type observation struct {
	callback string
	q0       int
	q1       int
	payload  string
}

func (o *observation) String() string {
	if o.callback == "Inserted" {
		return fmt.Sprintf("%s %q at %d", o.callback, o.payload, o.q0)
	}
	return fmt.Sprintf("%s [%d, %d)", o.callback, o.q0, o.q1)
}

type testObserver struct {
	t    *testing.T
	tape []*observation
}

func MakeTestObserver(t *testing.T) *testObserver {
	return &testObserver{
		t: t,
	}
}

func (to *testObserver) Inserted(q0 int, r []rune) {
	to.t.Helper()
	o := &observation{
		callback: "Inserted",
		q0:       q0,
		payload:  string(r),
	}
	to.t.Log(o)
	to.tape = append(to.tape, o)
}

func (to *testObserver) Deleted(q0, q1 int) {
	to.t.Helper()
	o := &observation{
		callback: "Deleted",
		q0:       q0,
		q1:       q1,
	}
	to.t.Log(o)
	to.tape = append(to.tape, o)
}

func (to *testObserver) Check(expected []*observation) {
	to.t.Helper()
	defer func() { to.tape = nil }()

	if got, want := len(to.tape), len(expected); got != want {
		to.t.Errorf("testObserver: tape length: got %d, want %d", got, want)
		return
	}

	for i, o := range to.tape {
		if got, want := o, expected[i]; *got != *want {
			to.t.Errorf("observation [%d] got: %v, want %v", i, got, want)
		}
	}
}

// String is a convenience function to dump span contents. Helpful for
// debugging logs.
func (s *span) String() string {
	buffy := new(strings.Builder)

	for p := s.start; p != s.end; p = p.next {
		buffy.Write(p.data)
		buffy.WriteString(" -> ")
	}

	return buffy.String()

}

type undoexpectation struct {
	q0 int
	q1 int
	ok bool
}

func (e *ObservableEditableBuffer) checkedUndo(isundo bool, t *testing.T, u undoexpectation) {
	t.Helper()

	q0, q1, ok := e.Undo(isundo)

	if got, want := ok, u.ok; got != want {
		t.Errorf("Undo wrong ok: got %v, want %v", got, want)
	}

	if !ok {
		// Values of q0, q1 don't matter if ok is false
		return
	}

	if got, want := q0, u.q0; got != want {
		t.Errorf("Undo wrong q0: got %d, want %d", got, want)
	}
	if got, want := q1, u.q1; got != want {
		t.Errorf("Undo wrong q1: got %d, want %d", got, want)
	}
}
