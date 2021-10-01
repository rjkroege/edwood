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

	mod          bool // true if the file has been changed. [private]
//	treatasclean bool // Window Clean tests should succeed if set. [private]

	// cache holds  that are not yet part of an undo record.
	cache []rune // [private]

	// cq0 tracks the insertion point for the cache.
	cq0 int // [private]
}

// Remember that the high-level goal is to slowly coerce this into looking like
// a scrawny wrapper around the Undo implementation. As a result, we should
// expect to see the following entry points:
//
// func (b *Buffer) Clean()
//func (b *Buffer) Commit()
//func (b *Buffer) Delete(off, length int64) error
//func (b *Buffer) Dirty() bool
//func (b *Buffer) Insert(off int64, data []byte) error
//func (b *Buffer) ReadAt(data []byte, off int64) (n int, err error)
//func (b *Buffer) Redo() (off, n int64)
//func (b *Buffer) Size() int64
//func (b *Buffer) Undo() (off, n int64)
//
// NB how the cache is folded into Buffer.
//TODO(rjk): make undo.Buffer implement Reader and Writer.

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
// This is currently in runes. Note that undo.Buffer.Size() is in bytes.
func (f *File) Size() int {
	return int(f.b.Nc()) + len(f.cache)
}

// Nr returns the number of valid runes in the File.
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

func (f *File) saveableAndDirtyImpl() bool {
	return f.mod || len(f.cache) > 0
}

// Commit writes the in-progress edits to the real buffer instead of
// keeping them in the cache. Does not map to undo.RuneArray.Commit (that
// method is Mark). Remove this method.
func (f *File) Commit(seq int) {
//	f.treatasclean = false
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
func (f *File) Load(q0 int, d []byte, seq int) (n int, hasNulls bool) {

	runes, _, hasNulls := util.Cvttorunes(d, len(d))

	// Would appear to require a commit operation.
	// NB: Runs the observers.
	f.InsertAt(q0, runes, seq)

	return len(runes), hasNulls
}

// InsertAt inserts s runes at rune address p0.
// TODO(rjk): run the observers here to simplify the Text code.
// TODO(rjk): In terms of the file.Buffer conversion, this corresponds
// to file.Buffer.Insert.
// NB: At suffix is to correspond to utf8string.String.At().
func (f *File) InsertAt(p0 int, s []rune, seq int) {
//	f.treatasclean = false
	if p0 > f.b.Nc() {
		panic("internal error: fileinsert")
	}
	if seq > 0 {
		f.Uninsert(&f.delta, p0, len(s), seq)
	}
	f.b.Insert(p0, s)
	if len(s) != 0 {
		f.Modded()
	}
	f.oeb.inserted(p0, s)
}

// InsertAtWithoutCommit inserts s at p0 without creating
// an undo record.
// TODO(rjk): Remove this as a prelude to converting to file.Buffer
// undo.Buffer 
func (f *File) InsertAtWithoutCommit(p0 int, s []rune) {
//	f.treatasclean = false
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
func (f *File) Uninsert(delta *[]*Undo, q0, ns, seq int) {
	var u Undo
	// undo an insertion by deleting
	u.T = sam.Delete

	u.mod = f.mod
	u.seq = seq
	u.P0 = q0
	u.N = ns
	*delta = append(*delta, &u)
}

// DeleteAt removes the rune range [p0,p1) from File.
// TODO(rjk): Currently, adds an Undo record. It shouldn't
// TODO(rjk): should map onto file.Buffer.Delete
// TODO(rjk): DeleteAt has an implied Commit operation
// that makes it not match with file.Buffer.Delete
func (f *File) DeleteAt(p0, p1, seq int) {
//	f.treatasclean = false
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

	// Validate if this is right.
	if p1 > p0 {
		f.Modded()
	}
	f.oeb.deleted(p0, p1)
}

// Undelete generates an action record that inserts runes into the File
// to undo a deletion.
func (f *File) Undelete(delta *[]*Undo, p0, p1, seq int) {
	// undo a deletion by inserting
	var u Undo
	u.T = sam.Insert
	u.mod = f.mod
	u.seq = seq
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

func (f *File) UnsetName(delta *[]*Undo, seq int) {
	var u Undo
	// undo a file name change by restoring old name
	u.T = sam.Filename
	u.mod = f.mod
	u.seq = seq
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
		mod: false,

		//	curtext *Text
		//	text    **Text
		//	ntext   int
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

// Undo undoes edits if isundo is true or redoes edits if isundo is false.
// It returns the new selection q0, q1 and a bool indicating if the
// returned selection is meaningful.
//
// TODO(rjk): Separate Undo and Redo for better alignment with undo.RuneArray
// TODO(rjk): This Undo implementation may Undo/Redo multiple changes.
// The number actually processed is controlled by mutations to File.seq.
// This does not align with the semantics of undo.RuneArray.
// Each "Mark" needs to have a seq value provided.
// Returns new q0, q1, ok, new seq
func (f *File) Undo(isundo bool, seq int) (int, int,  bool,  int)  {
	var (
		stop           int
		delta, epsilon *[]*Undo
		q0 int
		q1 int
		ok bool
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
			f.mod = u.mod
			// f.treatasclean = false
			f.b.Delete(u.P0, u.P0+u.N)
			f.oeb.deleted(u.P0, u.P0+u.N)
			q0 = u.P0
			q1 = u.P0
			ok = true
		case sam.Insert:
			seq = u.seq
			f.Uninsert(epsilon, u.P0, u.N, seq)
			f.mod = u.mod
			// f.treatasclean = false
			f.b.Insert(u.P0, u.Buf)
			f.oeb.inserted(u.P0, u.Buf)
			q0 = u.P0
			q1 = u.P0 + u.N
			ok = true
		case sam.Filename:
			// TODO(rjk): Fix Undo on Filename once the code has matured, removing broken code in the meantime.
			// TODO(rjk): If I have a zerox, does undo a filename change update?
			seq = u.seq
			f.UnsetName(epsilon, seq)
			f.mod = u.mod
			// f.treatasclean = false
			newfname := string(u.Buf)
			f.oeb.Setnameandisscratch(newfname)
		}
		*delta = (*delta)[0 : len(*delta)-1]
	}
	// TODO(rjk): Why do we do this?
	if isundo {
		seq = 0
	}
	return q0, q1, ok, seq
}

// Reset removes all Undo records for this File.
// TODO(rjk): This concept doesn't particularly exist in file.Buffer.
// Why can't I just create a new File?
func (f *File) Reset() {
	f.delta = f.delta[0:0]
	f.epsilon = f.epsilon[0:0]
//	f.seq = 0
}

// Mark sets an Undo point and
// and discards Redo records. Call this at the beginning
// of a set of edits that ought to be undo-able as a unit. This
// is equivalent to file.Buffer.Commit()
// NB: current implementation permits calling Mark on an empty
// file to indicate that one can undo to the file state at the time of
// calling Mark.
// TODO(rjk): Consider renaming to SetUndoPoint
func (f *File) Mark() {
	f.epsilon = f.epsilon[0:0]
//	f.seq = seq
}

// Modded marks the File if we know that its backing is different from
// its contents. This is needed to track when Edwood has modified the
// backing without changing the File (e.g. via the Edit w command.
func (f *File) Modded() {
	f.mod = true
}

// Clean marks File as being non-dirty: the backing is the same as File.
func (f *File) Clean() {
	f.mod = false
}
