package file

import (
	"fmt"

	"github.com/rjkroege/edwood/sam"
	"github.com/rjkroege/edwood/util"
)

// File is an editable text buffer with undo. Many Text can share one
// File (to implement Zerox). The File is responsible for updating the
// Text instances. File is a model in MVC parlance while Text is a
// View-Controller.
//
// A File tracks several related concepts. First it is a text buffer with
// undo/redo back to an initial state. Mark (undo.RuneArray.Commit) notes
// an undo point.
//
// Next, a File might have a backing to a disk file.
//
// Lastly the text buffer might be clean/dirty. A clean buffer is possibly
// the same as its disk backing. A specific point in the undo record is
// considered clean.
//
// TODO(rjk): File will be a facade pattern composing an undo.RuneArray
// and a wrapping utf8string.String indexing wrapper.
// TODO(rjk): my version of undo.RuneArray  will implement Reader, Writer,
// RuneReader, Seeker and I will restructure this code to follow the
// patterns of the Go I/O libraries. I will probably want to provide a cache
// around undo.RuneArray.
// Observe: Character motion routines in Text can be written
// in terms of any object that is Seeker and RuneReader.
// Observe: Frame can report addresses in byte and rune offsets.
type File struct {
	b       RuneArray
	delta   []*Undo // [private]
	epsilon []*Undo // [private]

	oeb *ObservableEditableBuffer

	// Tracks the Edit sequence.
	seq          int  // undo sequencing [private]
	putseq       int  // seq on last put [private]
	mod          bool // true if the file has been changed. [private]
	treatasclean bool // Window Clean tests should succeed if set. [private]

	// cache holds  that are not yet part of an undo record.
	cache []rune // [private]

	// cq0 tracks the insertion point for the cache.
	cq0 int // [private]
}

// Remember that the high-level goal is to slowly coerce this into looking like
// a scrawny wrapper around the Undo implementation. As a result, we should
// expect to see the following entry points:
//
// func (b *RuneArray) Clean()
//func (b *RuneArray) Commit()
//func (b *RuneArray) Delete(off, length int64) error
//func (b *RuneArray) Dirty() bool
//func (b *RuneArray) Insert(off int64, data []byte) error
//func (b *RuneArray) ReadAt(data []byte, off int64) (n int, err error)
//func (b *RuneArray) Redo() (off, n int64)
//func (b *RuneArray) Size() int64
//func (b *RuneArray) Undo() (off, n int64)
//
// NB how the cache is folded into RuneArray.
//TODO(rjk): make undo.RuneArray implement Reader and Writer.

// HasUncommitedChanges returns true if there are changes that
// have been made to the File after the last Commit.
func (t *File) HasUncommitedChanges() bool {
	return len(t.cache) != 0
}

// HasUndoableChanges returns true if there are changes to the File
// that can be undone.
// Has no analog in buffer.Undo. It will require modification.
func (f *File) HasUndoableChanges() bool {
	return len(f.delta) > 0 || len(f.cache) != 0
}

// HasRedoableChanges returns true if there are entries in the Redo
// log that can be redone.
// Has no analog in buffer.Undo. It will require modification.
func (f *File) HasRedoableChanges() bool {
	return len(f.epsilon) > 0
}

// Size returns the complete size of the buffer including both committed
// and uncommitted runes.
// NB: naturally forwards to undo.RuneArray.Size()
// TODO(rjk): Switch all callers to Nr() as would be the number of
// bytes when backed by undo.RuneArray.
func (f *File) Size() int {
	return int(f.b.Nc()) + len(f.cache)
}

// Nr returns the number of valid runes in the RuneArray.
// At the moment, this is the same as Size. But when File is backed
// with utf8, this will require adjustment.
// TODO(rjk): utf8 adjustment
func (f *File) Nr() int {
	return f.Size()
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

// ReadAtRune reads at most len(r) runes from File at rune off.
// It returns the number of  runes read and an error if something goes wrong.
func (f *File) ReadAtRune(r []rune, off int) (n int, err error) {
	// TODO(rjk): This should include cache contents but currently
	// callers do not require it to.
	return f.b.Read(off, r)
}

// SaveableAndDirty returns true if the File's contents differ from the
// backing diskfile File.name, and the diskfile is plausibly writable
// (not a directory or scratch file).
//
// When this is true, the tag's button should
// be drawn in the modified state if appropriate to the window type
// and Edit commands should treat the file as modified.
//
// TODO(rjk): figure out how this overlaps with hash. (hash would appear
// to be used to determine the "if the contents differ")
//
// Latest thought: there are two separate issues: are we at a point marked
// as clean and is this File writable to a backing. They are combined in this
// this method.
func (f *File) SaveableAndDirty() bool {
	return (f.mod || f.Dirty() || len(f.cache) > 0) && !f.oeb.IsDirOrScratch()
}

// Commit writes the in-progress edits to the real buffer instead of
// keeping them in the cache. Does not map to undo.RuneArray.Commit (that
// method is Mark). Remove this method.
func (f *File) Commit() {
	f.treatasclean = false
	if !f.HasUncommitedChanges() {
		return
	}

	if f.cq0 > f.b.Nc() {
		// TODO(rjk): Generate a better error message.
		panic("internal error: File.Commit")
	}
	if f.seq > 0 {
		f.Uninsert(&f.delta, f.cq0, len(f.cache))
	}
	f.b.Insert(f.cq0, f.cache)
	if len(f.cache) != 0 {
		f.Modded()
	}
	f.cache = f.cache[:0]
}

type Undo struct {
	T   int
	mod bool
	seq int
	P0  int
	N   int
	Buf []rune
}

// Load inserts fd's contents into File at location q0. Load will always
// mark the file as modified so follow this up with a call to f.Clean() to
// indicate that the file corresponds to its disk file backing.
// TODO(rjk): hypothesis: we can make this API cleaner: we will only
// compute a hash when the file corresponds to its diskfile right?
// TODO(rjk): Consider renaming InsertAtFromFd or something similar.
// TODO(rjk): Read and insert in chunks.
// TODO(flux): Innefficient to load the file, then copy into the slice,
// but I need the UTF-8 interpretation.  I could fix this by using a
// UTF-8 -> []rune reader on top of the os.File instead.
func (f *File) Load(q0 int, d []byte) (n int, hasNulls bool) {

	runes, _, hasNulls := util.Cvttorunes(d, len(d))

	// Would appear to require a commit operation.
	// NB: Runs the observers.
	f.InsertAt(q0, runes)

	return len(runes), hasNulls
}

// SnapshotSeq saves the current seq to putseq. Call this on Put actions.
// TODO(rjk): switching to undo.RuneArray will require removing use of seq
// TODO(rjk): This function maps to undo.RuneArray.Clean()
func (f *File) SnapshotSeq() {
	f.putseq = f.seq
}

// Dirty reports whether the current state of the File is different from
// the initial state or from the one at the time of calling Clean.
//
// TODO(rjk): switching to undo.RuneArray will require removing external uses
// of seq.
func (f *File) Dirty() bool {
	return f.seq != f.putseq
}

// InsertAt inserts s runes at rune address p0.
// TODO(rjk): run the observers here to simplify the Text code.
// TODO(rjk): In terms of the undo.RuneArray conversion, this correponds
// to undo.RuneArray.Insert.
// NB: At suffix is to correspond to utf8string.String.At().
func (f *File) InsertAt(p0 int, s []rune) {
	f.treatasclean = false
	if p0 > f.b.Nc() {
		panic("internal error: fileinsert")
	}
	if f.seq > 0 {
		f.Uninsert(&f.delta, p0, len(s))
	}
	f.b.Insert(p0, s)
	if len(s) != 0 {
		f.Modded()
	}
	f.oeb.inserted(p0, s)
}

// InsertAtWithoutCommit inserts s at p0 without creating
// an undo record.
// TODO(rjk): Remove this as a prelude to converting to undo.RuneArray
// But preserve the cache. Every "small" insert should go into the cache.
// It almost certainly greatly improves performance for a series of single
// character insertions.
func (f *File) InsertAtWithoutCommit(p0 int, s []rune) {
	f.treatasclean = false
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
	f.oeb.inserted(p0, s)
}

// Uninsert generates an action record that deletes runes from the File
// to undo an insertion.
func (f *File) Uninsert(delta *[]*Undo, q0, ns int) {
	var u Undo
	// undo an insertion by deleting
	u.T = sam.Delete

	u.mod = f.mod
	u.seq = f.seq
	u.P0 = q0
	u.N = ns
	*delta = append(*delta, &u)
}

// DeleteAt removes the rune range [p0,p1) from File.
// TODO(rjk): Currently, adds an Undo record. It shouldn't
// TODO(rjk): should map onto undo.RuneArray.Delete
// TODO(rjk): DeleteAt has an implied Commit operation
// that makes it not match with undo.RuneArray.Delete
func (f *File) DeleteAt(p0, p1 int) {
	f.treatasclean = false
	if !(p0 <= p1 && p0 <= f.b.Nc() && p1 <= f.b.Nc()) {
		util.AcmeError("internal error: DeleteAt", nil)
	}
	if len(f.cache) > 0 {
		util.AcmeError("internal error: DeleteAt", nil)
	}

	if f.seq > 0 {
		f.Undelete(&f.delta, p0, p1)
	}
	f.b.Delete(p0, p1)

	// Validate if this is right.
	if p1 > p0 {
		f.Modded()
	}
	f.oeb.deleted(p0, p1)
}

// Undelete generates an action record that inserts runes into the File
// to undo a deletion.
func (f *File) Undelete(delta *[]*Undo, p0, p1 int) {
	// undo a deletion by inserting
	var u Undo
	u.T = sam.Insert
	u.mod = f.mod
	u.seq = f.seq
	u.P0 = p0
	u.N = p1 - p0
	u.Buf = make([]rune, u.N)
	f.b.Read(p0, u.Buf)
	*delta = append(*delta, &u)
}

// A File can have a spcific name that permit it to be persisted to disk
// but typically would not be. These two constants are suffixes of File
// names that have this property.
const (
	slashguide = "/guide"
	plusErrors = "+Errors"
)

func (f *File) UnsetName(delta *[]*Undo) {
	var u Undo
	// undo a file name change by restoring old name
	u.T = sam.Filename
	u.mod = f.mod
	u.seq = f.seq
	u.P0 = 0 // unused
	u.N = len(f.oeb.Name())
	u.Buf = []rune(f.oeb.Name())
	*delta = append(*delta, &u)
}

func NewFile() *File {
	return &File{
		b:       NewRuneArray(),
		delta:   []*Undo{},
		epsilon: []*Undo{},
		//	seq       int
		mod: false,
		//	ntext   int
	}
}

func NewTagFile() *File {

	return &File{
		b:       NewRuneArray(),
		delta:   []*Undo{},
		epsilon: []*Undo{},
		//	qidpath   uint64
		//	mtime     uint64
		//	dev       int
		//	seq       int
		mod: false,

		//	curtext *Text
		//	text    **Text
		//	ntext   int
	}
}

// RedoSeq finds the seq of the last redo record. TODO(rjk): This has no
// analog in undo.RuneArray. The value of seq is used to track intra and
// inter File edit actions so that cross-File changes via Edit X can be
// undone with a single action. An implementation of File that wraps
// undo.RuneArray will need to to preserve seq tracking.
func (f *File) RedoSeq() int {
	delta := &f.epsilon
	if len(*delta) == 0 {
		return 0
	}
	u := (*delta)[len(*delta)-1]
	return u.seq
}

// Seq returns the current value of seq.
func (f *File) Seq() int {
	return f.seq
}

// Undo undoes edits if isundo is true or redoes edits if isundo is false.
// It returns the new selection q0, q1 and a bool indicating if the
// returned selection is meaningful.
//
// TODO(rjk): Separate Undo and Redo for better alignment with undo.RuneArray
// TODO(rjk): This Undo implementation may Undo/Redo multiple changes.
// The number actually processed is controlled by mutations to File.seq.
// This does not align with the semantics of undo.RuneArray.
// Each "Mark" needs to have a seq value provided.
// TODO(rjk): Consider providing the target seq value as an argument.
func (f *File) Undo(isundo bool) (q0, q1 int, ok bool) {
	var (
		stop           int
		delta, epsilon *[]*Undo
	)
	if isundo {
		// undo; reverse delta onto epsilon, seq decreases
		delta = &f.delta
		epsilon = &f.epsilon
		stop = f.seq
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
				f.seq = u.seq
				return
			}
		} else {
			if stop == 0 {
				stop = u.seq
			}
			if u.seq > stop {
				return
			}
		}
		switch u.T {
		default:
			panic(fmt.Sprintf("undo: 0x%x\n", u.T))
		case sam.Delete:
			f.seq = u.seq
			f.Undelete(epsilon, u.P0, u.P0+u.N)
			f.mod = u.mod
			f.treatasclean = false
			f.b.Delete(u.P0, u.P0+u.N)
			f.oeb.deleted(u.P0, u.P0+u.N)
			q0 = u.P0
			q1 = u.P0
			ok = true
		case sam.Insert:
			f.seq = u.seq
			f.Uninsert(epsilon, u.P0, u.N)
			f.mod = u.mod
			f.treatasclean = false
			f.b.Insert(u.P0, u.Buf)
			f.oeb.inserted(u.P0, u.Buf)
			q0 = u.P0
			q1 = u.P0 + u.N
			ok = true
		case sam.Filename:
			// TODO(rjk): Fix Undo on Filename once the code has matured, removing broken code in the meantime.
			// TODO(rjk): If I have a zerox, does undo a filename change update?
			f.seq = u.seq
			f.UnsetName(epsilon)
			f.mod = u.mod
			f.treatasclean = false
			newfname := string(u.Buf)
			f.oeb.Setnameandisscratch(newfname)
		}
		*delta = (*delta)[0 : len(*delta)-1]
	}
	// TODO(rjk): Why do we do this?
	if isundo {
		f.seq = 0
	}
	return q0, q1, ok
}

// Reset removes all Undo records for this File.
// TODO(rjk): This concept doesn't particularly exist in undo.RuneArray.
// Why can't I just create a new File?
func (f *File) Reset() {
	f.delta = f.delta[0:0]
	f.epsilon = f.epsilon[0:0]
	f.seq = 0
}

// Mark sets an Undo point and
// and discards Redo records. Call this at the beginning
// of a set of edits that ought to be undo-able as a unit. This
// is equivalent to undo.RuneArray.Commit()
// NB: current implementation permits calling Mark on an empty
// file to indicate that one can undo to the file state at the time of
// calling Mark.
// TODO(rjk): Consider renaming to SetUndoPoint
// TODO(rjk): Don't pass in seq. (Remove seq entirely?)
func (f *File) Mark(seq int) {
	f.epsilon = f.epsilon[0:0]
	f.seq = seq
}

// TreatAsDirty returns true if the File should be considered modified
// for the purpose of warning the user if Del-ing a Dirty() file.
func (f *File) TreatAsDirty() bool {
	return !f.treatasclean && f.Dirty()
}

// TreatAsClean notes that the File should be considered as not Dirty
// until its next modification.
func (f *File) TreatAsClean() {
	f.treatasclean = true
}

// Modded marks the File if we know that its backing is different from
// its contents. This is needed to track when Edwood has modified the
// backing without changing the File (e.g. via the Edit w command.
func (f *File) Modded() {
	f.mod = true
	f.treatasclean = false
}

// Clean marks File as being non-dirty: the backing is the same as File.
func (f *File) Clean() {
	f.mod = false
	f.treatasclean = false
	f.SnapshotSeq()
}
