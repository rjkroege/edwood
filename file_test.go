package main

import (
	"strings"
	"testing"
)

func TestDelText(t *testing.T) {
	f := &File{
		text: []*Text{{}, {}, {}, {}, {}},
	}
	t.Run("Nonexistent", func(t *testing.T) {
		err := f.DelText(&Text{})
		if err == nil {
			t.Errorf("expected panic when deleting nonexistent text")
		}
	})
	for i := len(f.text) - 1; i >= 0; i-- {
		text := f.text[i]
		err := f.DelText(text)
		if err != nil {
			t.Errorf("DelText of text at index %d failed: %v", i, err)
			continue
		}
		if got, want := len(f.text), i; got != want {
			t.Fatalf("DelText resulted in text of length %v; expected %v", got, want)
		}
		for i, t1 := range f.text {
			if t1 == text {
				t.Fatalf("DelText did not delete correctly at index %v", i)
			}
		}
	}
}

func TestFileInsertAtWithoutCommit(t *testing.T) {
	f := NewFile("edwood")

	f.InsertAtWithoutCommit(0, []rune(s1))

	i := 0
	for _, r := range s1 {
		if got, want := f.ReadC(i), r; got != want {
			t.Errorf("ReadC failed. got %v want % v", got, want)
		}
		i++
	}

	if got, want := f.Nr(), 6; got != want {
		t.Errorf("Nr failed. got %v want % v", got, want)
	}

	check(t, "TestFileInsertAt after TestFileInsertAtWithoutCommit", f,
		&fileStateSummary{true, true, false, true, true, s1})
}

const s1 = "hi 海老麺"
const s2 = "bye"

func TestFileInsertAt(t *testing.T) {
	f := NewFile("edwood")

	// Force Undo.
	f.seq = 1

	f.InsertAtWithoutCommit(0, []rune(s1))

	// NB: the read code not include the uncommited content.
	check(t, "TestFileInsertAt after InsertAtWithoutCommits", f,
		&fileStateSummary{true, true, false, true, true, s1})

	f.Commit()

	check(t, "TestFileInsertAt after InsertAtWithoutCommits", f,
		&fileStateSummary{false, true, false, true, true, s1})

	f.InsertAt(f.Nr(), []rune(s2))

	check(t, "TestFileUndoRedo after InsertAt", f,
		&fileStateSummary{false, true, false, true, true, s1 + s2})
}

func readwholefile(t *testing.T, f *File) string {
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

func TestFileUndoRedo(t *testing.T) {
	f := NewFile("edwood")

	// Force Undo to operate.
	f.seq = 1

	f.InsertAt(0, []rune(s1))
	f.InsertAt(f.Nr(), []rune(s2))

	check(t, "TestFileUndoRedo after 2 inserts", f,
		&fileStateSummary{false, true, false, true, true, s1 + s2})

	// Because of how seq managed the number of Undo actions, this corresponds
	// to the case of not incrementing seq and undoes every action in the log.
	f.Undo(true)

	check(t, "TestFileUndoRedo after 1 undo", f,
		&fileStateSummary{false, false, true, false, false, ""})

	// Redo
	f.Undo(false)

	// Validate state: we have s1 + s2 inserted.
	check(t, "TestFileUndoRedo after 1 Redos", f,
		&fileStateSummary{false, true, false, true, true, s1 + s2})
}

type fileStateSummary struct {
	HasUncommitedChanges bool
	HasUndoableChanges   bool
	HasRedoableChanges   bool
	HasSaveableChanges   bool
	SaveableAndDirty     bool
	filecontents         string
}

func check(t *testing.T, testname string, f *File, fss *fileStateSummary) {
	if got, want := f.HasUncommitedChanges(), fss.HasUncommitedChanges; got != want {
		t.Errorf("%s: HasUncommitedChanges failed. got %v want % v", testname, got, want)
	}
	if got, want := f.HasUndoableChanges(), fss.HasUndoableChanges; got != want {
		t.Errorf("%s: HasUndoableChanges failed. got %v want % v", testname, got, want)
	}
	if got, want := f.HasRedoableChanges(), fss.HasRedoableChanges; got != want {
		t.Errorf("%s: HasUndoableChanges failed. got %v want % v", testname, got, want)
	}
	if got, want := f.HasSaveableChanges(), fss.HasSaveableChanges; got != want {
		t.Errorf("%s: HasSaveableChanges failed. got %v want % v", testname, got, want)
	}
	if got, want := f.SaveableAndDirty(), fss.SaveableAndDirty; got != want {
		t.Errorf("%s: SaveableAndDirty failed. got %v want % v", testname, got, want)
	}
	if got, want := readwholefile(t, f), fss.filecontents; got != want {
		t.Errorf("%s: File contents not expected. got «%#v» want «%#v»", testname, got, want)
	}
}

func TestFileUndoRedoWithMark(t *testing.T) {
	f := NewFile("edwood")

	// Force Undo to operate.
	f.Mark(1)
	f.InsertAt(0, []rune(s1))

	f.Mark(2)
	f.InsertAt(f.Nr(), []rune(s2))

	check(t, "TestFileUndoRedoWithMark after 2 inserts", f,
		&fileStateSummary{false, true, false, true, true, s1 + s2})

	f.Undo(true)

	check(t, "TestFileUndoRedoWithMark after 1 undo", f,
		&fileStateSummary{false, true, true, true, true, s1})

	// Redo
	f.Undo(false)

	check(t, "TestFileUndoRedoWithMark after 1 redo", f,
		&fileStateSummary{false, true, false, true, true, s1 + s2})

}
