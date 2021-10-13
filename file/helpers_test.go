package file

import (
	"strings"
	"testing"
)

type checkable interface {
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
	if f.HasUncommitedChanges() {
		for i := 0; i < f.Nr(); i++ {
			sb.WriteRune(f.ReadC(i))
		}
		return sb.String()
	}

	targetbuffer := make([]rune, f.Nr())
	if _, err := f.ReadAtRune(targetbuffer, 0); err != nil {
		t.Fatalf("readwhole could not read File %v", f)
	}

	for _, r := range targetbuffer {
		if _, err := sb.WriteRune(r); err != nil {
			t.Fatalf("readwhole could not write rune %v to strings.Builder %s", r, sb.String())
		}
	}

	return sb.String()
}

func (b *Buffer) commitisgermane() bool { return false }

// TODO(camsn0w): write this.
func (b *Buffer) readwholefile(*testing.T) string { return "" }

func check(t *testing.T, testname string, oeb *ObservableEditableBuffer, fss *stateSummary) {
	t.Helper()

	if oeb.f != nil && oeb.b != nil {
		t.Fatalf("only one oeb.f or oeb.b should be in use")
	}

	// Lets the test infrastructure call against file.Buffer or file.File.
	f := checkable(oeb.f)
	if oeb.b != nil {
		f = checkable(oeb.b)
	}

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

// TODO(rjk): These should enforce observer callback contents in a flexible way.
// TODO(rjk): testText and testObserver
type testObserver struct {
	t *testing.T
}

func (to *testObserver) Inserted(q0 int, r []rune) {
	to.t.Logf("Inserted at %d: %q", q0, string(r))
}

func (to *testObserver) Deleted(q0, q1 int) {
	to.t.Logf("Deleted range [%d, %d)", q0, q1)
}

type testText struct {
	file *ObservableEditableBuffer
	b    RuneArray
}

// Inserted is implemented to satisfy the BufferObserver interface
func (t testText) Inserted(q0 int, r []rune) {}

// Deleted is implemented to satisfy the BufferObserver interface
func (t testText) Deleted(q0, q1 int) {}
