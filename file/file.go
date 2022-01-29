package file

import (
	"fmt"
	"io"

	"github.com/rjkroege/edwood/sam"
	"github.com/rjkroege/edwood/util"
)

// File is an editable text buffer with undo. Many Text can share one
// File (to implement Zerox). The File is responsible for updating the
// Text instances. File is a model in MVC parlance while Text is a
// View-Controller.
//
// A File tracks several related concepts. First it is a text buffer with
// undo/redo back to an initial state. Mark (file.Buffer.Commit) notes
// an undo point.
//
// Lastly the text buffer might be clean/dirty. A clean buffer is possibly
// the same as its disk backing. A specific point in the undo record is
// considered clean.
//
// TODO(rjk): ObservableEditableBuffer will be a facade pattern wrapping
// a file.Buffer. This file.go is the legacy implementation and will be
// removed.
//
// TODO(rjk): The Edwood version of file.Buffer will implement Reader,
// Writer, RuneReader, Seeker. Observe: Character motion routines in Text
// can be written in terms of any object that is Seeker and RuneReader.
// Observe: Frame can report addresses in byte and rune offsets.
type File struct {
	b       RuneArray
	delta   []*Undo
	epsilon []*Undo

	oeb *ObservableEditableBuffer

	// cache holds edits that have not yet been Commit-ed to the backing
	// RuneArray. It's presence should be semantically invisible.
	cache []rune

	// cq0 tracks the insertion point for the cache.
	cq0 int
}

// HasUncommitedChanges returns true if there are changes that
// have been made to the File since the last Commit.
func (f *File) HasUncommitedChanges() bool {
	return len(f.cache) > 0
}

// HasUndoableChanges returns true if there are changes to the File
// that can be undone.
func (f *File) HasUndoableChanges() bool {
	// TODO(rjk): This is wrong. The Commit would change this to false.
	return len(f.delta) > 0
}

// HasRedoableChanges returns true if there are entries in the Redo
// log that can be redone.
func (f *File) HasRedoableChanges() bool {
	return len(f.epsilon) > 0
}

// Nr returns the number of valid runes in the File.
func (f *File) Nr() int {
	return int(f.b.Nc()) + len(f.cache)
}

// ReadC reads a single rune from the File.
// Can be easily converted to being utf8 backed but
// every caller will require adjustment.
// TODO(rjk): File needs to implement RuneReader and code should
// use that interface instead.
// TODO(rjk): Better name to align with utf8string.String.At().
func (f *File) ReadC(q int) rune {
	if f.cq0 <= q && q < f.cq0+len(f.cache) {
		return f.cache[q-f.cq0]
	}
	return f.b.ReadC(q)
}

// Commit writes the in-progress edits to the real buffer instead of
// keeping them in the cache. Does not map to file.Buffer.Commit (that
// method is Mark). Remove this method.
func (f *File) Commit(seq int) {
	if !f.HasUncommitedChanges() {
		return
	}

	if f.cq0 > f.b.Nc() {
		// TODO(rjk): Generate a better error message.
		panic("internal error: File.Commit")
	}
	if seq > 0 {
		f.Uninsert(&f.delta, f.cq0, len(f.cache), seq)
	}
	f.b.Insert(f.cq0, f.cache)
	f.cache = f.cache[:0]
}

type Undo struct {
	T   int
	seq int
	P0  int
	N   int
	Buf []rune
}

// InsertAt inserts s runes at rune address p0.
// TODO(rjk): run the observers here to simplify the Text code.
// TODO(rjk): In terms of the file.Buffer conversion, this corresponds
// to file.Buffer.Insert.
// NB: At suffix is to correspond to utf8string.String.At().
func (f *File) InsertAt(p0 int, s []rune, seq int) {
	if p0 > f.b.Nc() {
		panic("internal error: fileinsert")
	}
	if seq > 0 {
		f.Uninsert(&f.delta, p0, len(s), seq)
	}
	f.b.Insert(p0, s)
}

// InsertAtWithoutCommit inserts s at p0 by only writing to the cache.
func (f *File) InsertAtWithoutCommit(p0 int, s []rune, _ int) {
	if p0 > f.b.Nc()+len(f.cache) {
		panic("File.InsertAtWithoutCommit insertion off the end")
	}

	if len(f.cache) == 0 {
		f.cq0 = p0
	} else {
		if p0 != f.cq0+len(f.cache) {
			// TODO(rjk): actually print something useful here
			util.AcmeError("File.InsertAtWithoutCommit cq0", nil)
		}
	}
	f.cache = append(f.cache, s...)
}

// Uninsert generates an action record that deletes runes from the File
// to undo an insertion.
func (f *File) Uninsert(delta *[]*Undo, q0, ns, seq int) {
	var u Undo
	// undo an insertion by deleting
	u.T = sam.Delete
	u.seq = seq
	u.P0 = q0
	u.N = ns
	*delta = append(*delta, &u)
}

// DeleteAt removes the rune range [p0,p1) from File.
// TODO(rjk): should map onto file.Buffer.Delete
// TODO(rjk): DeleteAt requires a Commit operation
// that makes it not match with file.Buffer.Delete
func (f *File) DeleteAt(p0, p1, seq int) {
	if !(p0 <= p1 && p0 <= f.b.Nc() && p1 <= f.b.Nc()) {
		util.AcmeError("internal error: DeleteAt", nil)
	}
	if len(f.cache) > 0 {
		util.AcmeError("internal error: DeleteAt", nil)
	}

	if seq > 0 {
		f.Undelete(&f.delta, p0, p1, seq)
	}
	f.b.Delete(p0, p1)
}

// Undelete generates an action record that inserts runes into the File
// to undo a deletion.
func (f *File) Undelete(delta *[]*Undo, p0, p1, seq int) {
	// undo a deletion by inserting
	var u Undo
	u.T = sam.Insert
	u.seq = seq
	u.P0 = p0
	u.N = p1 - p0
	u.Buf = make([]rune, u.N)
	f.b.Read(p0, u.Buf)
	*delta = append(*delta, &u)
}

func (f *File) UnsetName(fname string, seq int) {
	f._unsetName(&f.delta, fname, seq)
}

func (f *File) _unsetName(delta *[]*Undo, fname string, seq int) {
	var u Undo
	// undo a file name change by restoring old name
	u.T = sam.Filename
	u.seq = seq
	u.P0 = 0 // unused
	u.N = len(fname)
	u.Buf = []rune(fname)
	*delta = append(*delta, &u)
}

func NewLegacyFile(b []rune, oeb *ObservableEditableBuffer) *File {
	return &File{
		b:       b,
		delta:   []*Undo{},
		epsilon: []*Undo{},
		oeb:     oeb,
	}
}

// RedoSeq finds the seq of the last redo record. TODO(rjk): This has no
// analog in file.Buffer. The value of seq is used to track intra and
// inter File edit actions so that cross-File changes via Edit X can be
// undone with a single action. An implementation of
// ObservableEditableBuffer that wraps file.Buffer will need to to
// preserve seq tracking.
func (f *File) RedoSeq() int {
	delta := &f.epsilon
	if len(*delta) == 0 {
		return 0
	}
	u := (*delta)[len(*delta)-1]
	return u.seq
}

func (f *File) Undo(seq int) (int, int, bool, int) {
	return f._undo(true, seq)
}

func (f *File) Redo(seq int) (int, int, bool, int) {
	return f._undo(false, seq)
}

// Undo undoes edits if isundo is true or redoes edits if isundo is false.
// It returns the new selection q0, q1 and a bool indicating if the
// returned selection is meaningful.
//
// TODO(rjk): This Undo implementation may Undo/Redo multiple changes.
// The number actually processed is controlled by mutations to File.seq.
// This does not align with the semantics of file.Buffer.
// Each "Mark" needs to have a seq value provided.
// Returns new q0, q1, ok, new seq
func (f *File) _undo(isundo bool, seq int) (int, int, bool, int) {
	var (
		stop           int
		delta, epsilon *[]*Undo
		q0             int
		q1             int
		ok             bool
	)
	if isundo {
		// undo; reverse delta onto epsilon, seq decreases
		delta = &f.delta
		epsilon = &f.epsilon
		stop = seq
	} else {
		// redo; reverse epsilon onto delta, seq increases
		delta = &f.epsilon
		epsilon = &f.delta
		stop = 0 // don't know yet
	}

	for len(*delta) > 0 {
		u := (*delta)[len(*delta)-1]
		if isundo {
			if u.seq < stop {
				// f.seq = u.seq
				return q0, q1, ok, u.seq
			}
		} else {
			if stop == 0 {
				stop = u.seq
			}
			if u.seq > stop {
				return q0, q1, ok, u.seq
			}
		}
		switch u.T {
		default:
			panic(fmt.Sprintf("undo: 0x%x\n", u.T))
		case sam.Delete:
			seq = u.seq
			f.Undelete(epsilon, u.P0, u.P0+u.N, seq)
			f.b.Delete(u.P0, u.P0+u.N)
			f.oeb.deleted(u.P0, u.P0+u.N)
			q0 = u.P0
			q1 = u.P0
			ok = true
		case sam.Insert:
			seq = u.seq
			f.Uninsert(epsilon, u.P0, u.N, seq)
			f.b.Insert(u.P0, u.Buf)
			f.oeb.inserted(u.P0, u.Buf)
			q0 = u.P0
			q1 = u.P0 + u.N
			ok = true
		case sam.Filename:
			// If I have a zerox, Undo works via Undo calling
			// TagStatusObserver.UpdateTag on the appropriate observers.
			seq = u.seq
			f._unsetName(epsilon, f.oeb.Name(), seq)
			newfname := string(u.Buf)
			f.oeb.setfilename(newfname)
		}
		*delta = (*delta)[0 : len(*delta)-1]
	}
	// TODO(rjk): Why do we do this?
	if isundo {
		seq = 0
	}
	return q0, q1, ok, seq
}

// Mark sets an Undo point and and discards Redo records. Call this at
// the beginning of a set of edits that ought to be undo-able as a unit.
// This is equivalent to file.Buffer.SetUndoPoint() NB: current implementation
// permits calling Mark on an empty file to indicate that one can undo to
// the file state at the time of calling Mark.
//
// TODO(rjk): Consider renaming to SetUndoPoint
// Might want the seq here?
func (f *File) Mark() {
	f.epsilon = f.epsilon[0:0]
}

// Finish implementing  BufferAdapter

func (f *File) IndexRune(r rune) int {
	return f.b.IndexRune(r)
}

func (f *File) Read(q0 int, r []rune) (int, error) {
	return f.b.Read(q0, r)
}

func (f *File) Reader(q0 int, q1 int) io.Reader {
	return f.b.Reader(q0, q1)
}

func (f *File) String() string {
	return f.b.String()
}
