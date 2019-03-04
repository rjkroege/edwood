package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/internal/file"
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
	f.Commit()

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

	// Because of how seq managed the number of Undo actions, this
	// corresponds to the case of not incrementing seq and undoes every
	// action in the log.
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
		t.Errorf("%s: HasUncommitedChanges failed. got %v want %v", testname, got, want)
	}
	if got, want := f.HasUndoableChanges(), fss.HasUndoableChanges; got != want {
		t.Errorf("%s: HasUndoableChanges failed. got %v want %v", testname, got, want)
	}
	if got, want := f.HasRedoableChanges(), fss.HasRedoableChanges; got != want {
		t.Errorf("%s: HasUndoableChanges failed. got %v want %v", testname, got, want)
	}
	if got, want := f.HasSaveableChanges(), fss.HasSaveableChanges; got != want {
		t.Errorf("%s: HasSaveableChanges failed. got %v want %v", testname, got, want)
	}
	if got, want := f.SaveableAndDirty(), fss.SaveableAndDirty; got != want {
		t.Errorf("%s: SaveableAndDirty failed. got %v want %v", testname, got, want)
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

func TestFileLoadNoUndo(t *testing.T) {
	f := NewFile("edwood")

	// Insert some pre-existing content.
	f.InsertAt(0, []rune(s1))

	buffy := bytes.NewBuffer([]byte(s2 + s2))

	n, hasNulls, err := f.Load(2, buffy, false)

	if got, want := n, len(s2)+len(s2); got != want {
		t.Errorf("TestFileLoadNoUndo rune count wrong. got %v want %v", got, want)
	}
	if got, want := hasNulls, false; got != want {
		t.Errorf("TestFileLoadNoUndo hasNulls wrong. got %v want %v", got, want)
	}
	if got, want := err, error(nil); got != want {
		t.Errorf("TestFileLoadNoUndo err wrong. got %v want %v", got, want)
	}

	// TODO(rjk): The file has been modified because of the insert. But
	// without undo, SaveableAndDirty and HasSaveableChanges diverge.
	check(t, "TestFileLoadNoUndo after file load", f,
		&fileStateSummary{false, false, false, false, true, s1[0:2] + s2 + s2 + s1[2:]})

}

func TestFileLoadUndoHash(t *testing.T) {
	hashOfS2nS2 :=
		file.Hash{0xf0, 0x21, 0xb5, 0x73, 0x6a, 0xb5, 0x21, 0x6d, 0x29, 0x1b, 0x19, 0xfb, 0xe, 0xa8, 0x53, 0x4a, 0x59, 0x7e, 0xb3, 0xfa}

	f := NewFile("edwood")
	if got, want := f.name, "edwood"; got != want {
		t.Errorf("TestFileLoadUndoHash bad initial name. got %v want %v", got, want)
	}

	buffy := bytes.NewBuffer([]byte(s2 + s2))

	f.Load(0, buffy, true)
	// f.Load marks the file as modified.
	f.Clean()

	if got, want := f.hash, hashOfS2nS2; !got.Eq(want) {
		t.Errorf("TestFileLoadUndoHash bad initial name. got %#v want %#v", got, want)
	}

	// Having loaded the file and then Clean(),
	check(t, "TestFileLoadUndoHash after file load", f,
		&fileStateSummary{false, false, false, false, false, s2 + s2})

	// Enable undo.
	f.Mark(1)

	// Enabling Undo will cause HasSaveableChanges to be true.
	// This is strange and I need to rationalize seq.
	check(t, "TestFileLoadUndoHash after Mark", f,
		&fileStateSummary{false, false, false, true, false, s2 + s2})

	// SaveableAndDirty should return true if the File is plausibly writable
	// to f.name. At this point, we don't know and the value of SaveableAndDirty
	// is not well defined.
	// TODO(rjk): The API should be clear about what SaveableAndDiry actually
	// does.
	f.SetName("plan9")
	check(t, "TestFileLoadUndoHash after SetName", f,
		&fileStateSummary{false, true, false, true, false, s2 + s2})

	if got, want := f.name, "plan9"; got != want {
		t.Errorf("TestFileLoadUndoHash failed to set name. got %v want %v", got, want)
	}

	// Undo renmaing the file.
	f.Undo(true)
	check(t, "TestFileLoadUndoHash after Undo", f,
		&fileStateSummary{false, false, true, false, false, s2 + s2})
	if got, want := f.name, "edwood"; got != want {
		t.Errorf("TestFileLoadUndoHash failed to set name. got %v want %v", got, want)
	}
}

// Multiple interleaved actions do the right thing.
func TestFileInsertDeleteUndo(t *testing.T) {
	f := NewFile("edwood")

	// Empty File is an Undo point.
	f.Mark(1)

	f.InsertAt(0, []rune(s1))
	f.InsertAt(0, []rune(s2))
	// After inserting two strings is an Undo point:  byehi 海老麺
	f.Mark(2)

	f.DeleteAt(0, 1) // yehi 海老
	f.DeleteAt(1, 3) // yi 海老
	// After deleting is an Undo point.
	f.Mark(3)

	f.InsertAt(f.Nr()-1, []rune(s1)) // yi 海老hi 海老麺

	check(t, "TestFileInsertDeleteUndo after setup", f,
		&fileStateSummary{false, true, false, true, true, "yi 海老hi 海老麺麺"})

	f.Undo(true)
	check(t, "TestFileInsertDeleteUndo after 1 Undo", f,
		&fileStateSummary{false, true, true, true, true, "yi 海老麺"})

	f.Undo(true) // 2 deletes should get removed because they have the same sequence.
	check(t, "TestFileInsertDeleteUndo after 2 Undo", f,
		&fileStateSummary{false, true, true, true, true, "byehi 海老麺"})

	f.Undo(false) // 2 deletes should be put back.
	check(t, "TestFileInsertDeleteUndo after 1 Undo", f,
		&fileStateSummary{false, true, true, true, true, "yi 海老麺"})
}

func TestFileRedoSeq(t *testing.T) {
	f := NewFile("edwood")

	// Empty File is an Undo point.
	f.Mark(1)

	f.InsertAt(0, []rune(s1))
	check(t, "TestFileRedoSeq after setup", f,
		&fileStateSummary{false, true, false, true, true, s1})

	if got, want := f.RedoSeq(), 0; got != want {
		t.Errorf("TestFileRedoSeq no redo. got %#v want %#v", got, want)
	}

	f.Undo(true)
	check(t, "TestFileRedoSeq after Undo", f,
		&fileStateSummary{false, false, true, false, false, ""})

	if got, want := f.RedoSeq(), 1; got != want {
		t.Errorf("TestFileRedoSeq no redo. got %#v want %#v", got, want)
	}
}
