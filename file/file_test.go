package file

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestDelObserver(t *testing.T) {
	f := MakeObservableEditableBuffer("", RuneArray{})

	testData := []*testObserver{
		{t: t},
		{t: t},
		{t: t},
	}

	t.Run("Nonexistent", func(t *testing.T) {
		err := f.DelObserver(&testObserver{t})
		if err == nil {
			t.Errorf("expected panic when deleting nonexistent observers")
		}
	})
	for _, text := range testData {
		f.AddObserver(text)
	}
	println("Size is now: ", f.GetObserverSize())
	for i, text := range testData {
		err := f.DelObserver(text)
		if err != nil {
			t.Errorf("DelObserver of observers at index %d failed: %v", i, err)
			continue
		}
		if got, want := f.GetObserverSize(), len(testData)-(i+1); got != want {
			println("On test #", i)
			t.Fatalf("DelObserver resulted in observers of length %v; expected %v", got, want)
		}
		f.AllObservers(func(i interface{}) {
			inText := i.(*testObserver)
			if inText == text {
				t.Fatalf("DelObserver did not delete correctly at index %v", i)
			}

		})
	}
}

func TestFileInsertAtWithoutCommit(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)

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
		&stateSummary{true, true, false, true, s1})
}

const s1 = "hi 海老麺"
const s2 = "bye"

func TestFileInsertAt(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)

	// Force Undo.
	f.seq = 1

	f.InsertAtWithoutCommit(0, []rune(s1))

	// NB: the read code not include the uncommitted content.
	check(t, "TestFileInsertAt after InsertAtWithoutCommits", f,
		&stateSummary{true, true, false, true, s1})

	f.Commit()

	check(t, "TestFileInsertAt after InsertAtWithoutCommits", f,
		&stateSummary{false, true, false, true, s1})

	f.InsertAt(f.Nr(), []rune(s2))
	f.Commit()

	check(t, "TestFileUndoRedo after InsertAt", f,
		&stateSummary{false, true, false, true, s1 + s2})
}

func TestFileUndoRedo(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)

	// Validate before.
	check(t, "TestFileUndoRedo on an empty buffer", f,
		&stateSummary{false, false, false, false, ""})

	f.Mark(1)
	f.InsertAt(0, []rune(s1))
	f.InsertAt(f.Nr(), []rune(s2))

	check(t, "TestFileUndoRedo after 2 inserts", f,
		&stateSummary{false, true, false, true, s1 + s2})

	// Because of how seq managed the number of Undo actions, this
	// corresponds to the case of not incrementing seq and undoes every
	// action in the log.
	f.Undo(true)

	check(t, "TestFileUndoRedo after 1 undo", f,
		&stateSummary{false, false, true, false, ""})

	// Redo
	f.Undo(false)

	// Validate state: we have s1 + s2 inserted.
	check(t, "TestFileUndoRedo after 1 Redos", f,
		&stateSummary{false, true, false, true, s1 + s2})
}

func TestFileUndoRedoWithMark(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)

	// Force Undo to operate.
	f.Mark(1)
	f.InsertAt(0, []rune(s1))

	f.Mark(2)
	f.InsertAt(f.Nr(), []rune(s2))

	check(t, "TestFileUndoRedoWithMark after 2 inserts", f,
		&stateSummary{false, true, false, true, s1 + s2})

	f.Undo(true)

	check(t, "TestFileUndoRedoWithMark after 1 undo", f,
		&stateSummary{false, true, true, true, s1})

	// Redo
	f.Undo(false)

	check(t, "TestFileUndoRedoWithMark after 1 redo", f,
		&stateSummary{false, true, false, true, s1 + s2})
}

func TestFileLoadNoUndo(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)

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

	check(t, "TestFileLoadNoUndo after file load", f,
		&stateSummary{false, false, false, false, s1[0:2] + s2 + s2 + s1[2:]})
}

func TestFileLoadUndoHash(t *testing.T) {
	hashOfS2nS2 :=
		Hash{0xf0, 0x21, 0xb5, 0x73, 0x6a, 0xb5, 0x21, 0x6d, 0x29, 0x1b, 0x19, 0xfb, 0xe, 0xa8, 0x53, 0x4a, 0x59, 0x7e, 0xb3, 0xfa}

	f := MakeObservableEditableBuffer("edwood", nil)
	if got, want := f.Name(), "edwood"; got != want {
		t.Errorf("TestFileLoadUndoHash bad initial name. got %v want %v", got, want)
	}

	buffy := bytes.NewBuffer([]byte(s2 + s2))

	f.Load(0, buffy, true)
	// f.Load marks the file as modified.
	f.Clean()

	if got, want := f.Hash(), hashOfS2nS2; !got.Eq(want) {
		t.Errorf("TestFileLoadUndoHash bad initial name. got %#v want %#v", got, want)
	}

	// Having loaded the file and then Clean(),
	check(t, "TestFileLoadUndoHash after file load", f,
		&stateSummary{false, false, false, false, s2 + s2})

	// Set an undo point (the initial state)
	f.Mark(1)

	// Enabling Undo will cause HasSaveableChanges to be true.
	// This is strange and I need to rationalize seq.
	check(t, "TestFileLoadUndoHash after Mark", f,
		&stateSummary{false, false, false, true, s2 + s2})

	// SaveableAndDirty should return true if the File is plausibly writable
	// to f.details.Name and has changes that might require writing it out.
	f.SetName("plan9")
	check(t, "TestFileLoadUndoHash after SetName", f,
		&stateSummary{false, true, false, true, s2 + s2})

	if got, want := f.Name(), "plan9"; got != want {
		t.Errorf("TestFileLoadUndoHash failed to set name. got %v want %v", got, want)
	}

	// Undo renmaing the file.
	f.Undo(true)
	check(t, "TestFileLoadUndoHash after Undo", f,
		&stateSummary{false, false, true, false, s2 + s2})
	if got, want := f.Name(), "edwood"; got != want {
		t.Errorf("TestFileLoadUndoHash failed to set name. got %v want %v", got, want)
	}
}

// Multiple interleaved actions do the right thing.
func TestFileInsertDeleteUndo(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)
	f.AddObserver(&testObserver{t})

	// Empty File is an Undo point and considered clean.
	f.Mark(1)
	f.Clean()
	check(t, "TestFileInsertDeleteUndo after init", f,
		&stateSummary{false, false, false, false, ""})

	f.Mark(2)
	f.InsertAt(0, []rune(s1))
	f.InsertAt(0, []rune(s2))
	// After inserting two strings is an Undo point:  byehi 海老麺
	check(t, "TestFileInsertDeleteUndo after second Mark", f,
		&stateSummary{false, true, false, true, "byehi 海老麺"})

	f.Mark(3)
	f.DeleteAt(0, 1) // yehi 海老
	f.DeleteAt(1, 3) // yi 海老
	// After deleting is an Undo point.
	check(t, "TestFileInsertDeleteUndo after third Mark", f,
		&stateSummary{false, true, false, true, "yi 海老麺"})

	f.Mark(4)
	f.InsertAt(f.Nr()-1, []rune(s1)) // yi 海老hi 海老麺
	check(t, "TestFileInsertDeleteUndo after setup", f,
		&stateSummary{false, true, false, true, "yi 海老hi 海老麺麺"})
	t.Logf("after setup seq %d, putseq %d", f.seq, f.putseq)

	f.Undo(true)
	check(t, "TestFileInsertDeleteUndo after 1 Undo", f,
		&stateSummary{false, true, true, true, "yi 海老麺"})
	t.Logf("after 1 Undo seq %d, putseq %d", f.seq, f.putseq)

	f.Undo(true) // 2 deletes should get removed because they have the same sequence.
	check(t, "TestFileInsertDeleteUndo after 2 Undo", f,
		&stateSummary{false, true, true, true, "byehi 海老麺"})
	t.Logf("after 2 Undo seq %d, putseq %d", f.seq, f.putseq)

	f.Undo(false) // 2 deletes should be put back.
	check(t, "TestFileInsertDeleteUndo after 1 Undo", f,
		&stateSummary{false, true, true, true, "yi 海老麺"})
	t.Logf("after 1 Redo seq %d, putseq %d", f.seq, f.putseq)
}

func TestFileRedoSeq(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)
	// Empty File is an Undo point and considered clean

	f.Mark(1)
	f.InsertAt(0, []rune(s1))

	check(t, "TestFileRedoSeq after setup", f,
		&stateSummary{false, true, false, true, s1})

	if got, want := f.RedoSeq(), 0; got != want {
		t.Errorf("TestFileRedoSeq no redo. got %#v want %#v", got, want)
	}

	f.Undo(true)
	check(t, "TestFileRedoSeq after Undo", f,
		&stateSummary{false, false, true, false, ""})

	if got, want := f.RedoSeq(), 1; got != want {
		t.Errorf("TestFileRedoSeq no redo. got %#v want %#v", got, want)
	}
}

func TestFileUpdateInfo(t *testing.T) {
	tmp, err := ioutil.TempFile("", "edwood_test")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	filename := tmp.Name()
	defer os.Remove(filename)

	if _, err := tmp.WriteString("Hello, 世界\n"); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	d, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	f := MakeObservableEditableBuffer(filename, nil).f
	f.oeb.SetHash(EmptyHash)
	f.oeb.SetInfo(nil)

	f.oeb.UpdateInfo(filename, d)
	if f.oeb.Info() != nil {
		t.Errorf("File info is %v; want nil", f.oeb.Info())
	}

	h, err := HashFor(filename)
	if err != nil {
		t.Fatalf("HashFor(%v) failed: %v", filename, err)
	}
	f.oeb.SetHash(h)
	f.oeb.SetInfo(nil)
	f.oeb.UpdateInfo(filename, d)
	if f.oeb.Info() != d {
		t.Errorf("File info is %v; want %v", f.oeb.Info(), d)
	}
}

func TestFileUpdateInfoError(t *testing.T) {
	filename := "/non-existent-file"
	f := MakeObservableEditableBuffer(filename, nil).f
	err := f.oeb.UpdateInfo(filename, nil)
	want := "failed to compute hash for"
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("File.UpdateInfo returned error %q; want prefix %q", err, want)
	}
}

func TestFileNameSettingWithScratch(t *testing.T) {
	f := MakeObservableEditableBuffer("edwood", nil)
	// Empty File is an Undo point and considered clean

	if got, want := f.Name(), "edwood"; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init name. got %v want %v", got, want)
	}
	if got, want := f.isscratch, false; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init isscratch. got %v want %v", got, want)
	}

	f.Mark(1)
	f.SetName("/guide")

	f.Mark(2)
	f.SetName("/hello/+Errors")

	if got, want := f.Name(), "/hello/+Errors"; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init name. got %v want %v", got, want)
	}
	if got, want := f.isscratch, true; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init isscratch. got %v want %v", got, want)
	}

	f.Undo(true)

	if got, want := f.Name(), "/guide"; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init name. got %v want %v", got, want)
	}
	if got, want := f.isscratch, true; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init isscratch. got %v want %v", got, want)
	}

	f.Undo(true)
	if got, want := f.Name(), "edwood"; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init name. got %v want %v", got, want)
	}
	if got, want := f.isscratch, false; got != want {
		t.Errorf("TestFileNameSettingWithScratch failed to init isscratch. got %v want %v", got, want)
	}
}
