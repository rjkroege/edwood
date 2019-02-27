package main

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
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

	f.InsertAtWithoutCommit(0, []rune("hi 海老麺"))

	i := 0
	for _, r := range "hi 海老麺" {
		if got, want := f.ReadC(i), r; got != want {
			t.Errorf("ReadC failed. got %v want % v", got, want)
		}
		i++
	}

	if got, want := f.Nr(), 6; got != want {
		t.Errorf("Nr failed. got %v want % v", got, want)
	}
	if got, want := f.HasSaveableChanges(), true; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}

	if got, want := f.HasUndoableChanges(), true; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasRedoableChanges(), false; got != want {
		t.Errorf("HasRedoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasUncommitedChanges(), true; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}
}

func TestFileCommitNoUndo(t *testing.T) {
	f := NewFile("edwood")

	f.InsertAtWithoutCommit(0, []rune("hi 海老麺"))
	f.Commit()

	i := 0
	for _, r := range "hi 海老麺" {
		if got, want := f.ReadC(i), r; got != want {
			t.Errorf("ReadC failed. got %v want % v", got, want)
		}
		i++
	}

	rr := make([]rune, 6)
	if n, err := f.ReadAtRune(rr, 0); n != 6 || err != nil {
		t.Errorf("ReadAtRune failed, bad length %v or err %v", n, err)
	}

	if !reflect.DeepEqual(rr, []rune("hi 海老麺")) {
		t.Errorf("ReadAtRune failed: got %v want % v", string(rr), "hi 海老麺")
	}

	if got, want := f.HasSaveableChanges(), false; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}

	if got, want := f.HasUndoableChanges(), false; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasRedoableChanges(), false; got != want {
		t.Errorf("HasRedoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasUncommitedChanges(), false; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}

}

func TestFileCommit(t *testing.T) {
	f := NewFile("edwood")

	// Force Undo.
	f.seq = 1

	f.InsertAtWithoutCommit(0, []rune("hi 海老麺"))
	f.Commit()

	i := 0
	for _, r := range "hi 海老麺" {
		if got, want := f.ReadC(i), r; got != want {
			t.Errorf("ReadC failed. got %v want % v", got, want)
		}
		i++
	}

	rr := make([]rune, 6)
	if n, err := f.ReadAtRune(rr, 0); n != 6 || err != nil {
		t.Errorf("ReadAtRune failed, bad length %v or err %v", n, err)
	}

	if !reflect.DeepEqual(rr, []rune("hi 海老麺")) {
		t.Errorf("ReadAtRune failed: got %v want % v", string(rr), "hi 海老麺")
	}

	if got, want := f.HasSaveableChanges(), true; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}

	if got, want := f.HasUndoableChanges(), true; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasRedoableChanges(), false; got != want {
		t.Errorf("HasRedoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasUncommitedChanges(), false; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}

}

const s1 = "hi 海老麺"
const s2 = "bye"

func TestFileInsertAt(t *testing.T) {
	f := NewFile("edwood")

	// Force Undo.
	f.seq = 1

	f.InsertAtWithoutCommit(0, []rune(s1))
	f.Commit()

	if got, want := f.HasSaveableChanges(), true; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}

	// TODO(rjk): Read and write different sized chunks.

	f.InsertAt(f.Nr(), []rune(s2))

	runecount := utf8.RuneCount([]byte(s1 + s2))
	rr := make([]rune, runecount)

	if n, err := f.ReadAtRune(rr, 0); n != runecount || err != nil {
		t.Errorf("ReadAtRune failed, bad length %v or err %v", n, err)
	}

	if !reflect.DeepEqual(rr, []rune(s1+s2)) {
		t.Errorf("ReadAtRune failed: got %v want % v", string(rr), s1+s2)
	}

	if got, want := f.HasSaveableChanges(), true; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}

	if got, want := f.HasUndoableChanges(), true; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasRedoableChanges(), false; got != want {
		t.Errorf("HasRedoableChanges failed. got %v want % v", got, want)
	}

	if got, want := f.HasUncommitedChanges(), false; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}
}

func readwholefile(t *testing.T, f *File) string {
	targetbuffer := make([]rune, f.Nr())

	if _, err := f.ReadAtRune(targetbuffer, 0); err != nil {
		t.Fatalf("readwhole could not read File %v", f)
	}

	var sb strings.Builder
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

	// Validate state: we have s1 + s2 inserted.
	if got, want := f.HasUncommitedChanges(), false; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasUndoableChanges(), true; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasRedoableChanges(), false; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasSaveableChanges(), true; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}
	if got, want := readwholefile(t, f), s1+s2; got != want {
		t.Errorf("File contents not expected. got %v want % v", got, want)
	}

	// Because of how seq managed the number of Undo actions, this corresponds
	// to the case of not incrementing seq and undoes every action in the log.
	f.Undo(true)

	// Validate state: we have s1 inserted.
	if got, want := f.HasUncommitedChanges(), false; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasUndoableChanges(), false; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasRedoableChanges(), true; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasSaveableChanges(), false; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), false; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}
	if got, want := readwholefile(t, f), ""; got != want {
		t.Errorf("File contents not expected. got %v want % v", got, want)
	}

	// Redo
	f.Undo(false)

	// Validate state: we have s1 + s2 inserted.
	if got, want := f.HasUncommitedChanges(), false; got != want {
		t.Errorf("HasUncommitedChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasUndoableChanges(), true; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasRedoableChanges(), false; got != want {
		t.Errorf("HasUndoableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.HasSaveableChanges(), true; got != want {
		t.Errorf("HasSaveableChanges failed. got %v want % v", got, want)
	}
	if got, want := f.SaveableAndDirty(), true; got != want {
		t.Errorf("SaveableAndDirty failed. got %v want % v", got, want)
	}
	if got, want := readwholefile(t, f), s1+s2; got != want {
		t.Errorf("File contents not expected. got %v want % v", got, want)
	}
}
