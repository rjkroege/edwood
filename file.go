package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rjkroege/edwood/internal/file"
)

// File is an editable text buffer with undo. Many Text can share one
// File (to implement Zerox). The File is responsible for updating the
// Text instances. File is a model in MVC parlance while Text is a
// View-Controller.
// TODO(rjk): File will be a facade pattern composing an undo.Buffer
// and a wrapping utf8string.String indexing wrapper.
// TODO(rjk): my version of undo.Buffer  will implement Reader, Writer,
// RuneReader, Seeker and I will restructure this code to follow the
// patterns of the Go I/O libraries. I will probably want to provide a cache
// around undo.Buffer.
// Observe: Character motion routines in Text can be written
// in terms of any object that is Seeker and RuneReader.
// Observe: Frame can report addresses in byte and rune offsets.
type File struct {
	b       Buffer
	delta   []*Undo // [private]
	epsilon []*Undo // [private]
	elog    Elog
	name    string
	qidpath string // TODO(flux): Gross hack to use filename instead of qidpath for file uniqueness
	mtime   time.Time
	// dev       int
	// unread bool

	// TODO(rjk): Remove this when I've inserted undo.Buffer.
	// At present, InsertAt and DeleteAt have an implicit Commit operation
	// associated with them. In an undo.Buffer context, these two ops
	// don't have an implicit Commit. We set editclean in the Edit cmd
	// implementation code to let multiple Inserts be grouped together?
	// Figure out how this inter-operates with seq.
	editclean bool

	// Tracks the Edit sequence.
	seq          int
	putseq       int  // seq on last put [private]
	mod          bool // true if the file has been changed. [private]
	treatasclean bool // Window Clean tests should succeed if set. [private]

	// Observer pattern: many Text instances can share a File.
	curtext *Text
	text    []*Text // [private I think]

	dumpid int // Used to track the identifying name of this File for Dump.

	isscratch bool // Used to track if this File should warn on unsaved deletion.
	isdir     bool // Used to track if this File is populated from a directory list.

	hash file.Hash // Used to check if the file has changed on disk since loaded.

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
// Corresponds to undo.Buffer.Dirty()
func (f *File) HasUndoableChanges() bool {
	return len(f.delta) > 0 || len(f.cache) != 0
}

// HasSaveableChanges returns true if there are changes to the File
// that can be saved.
// TODO(rjk): HasUnsavedChanges should be its name
// TODO(rjk): it's conceivable that mod and SeqDiffer track the same
// thing.
func (f *File) HasSaveableChanges() bool {
	return f.name != "" && (len(f.cache) != 0 || f.SeqDiffer())
}

// HasRedoableChanges returns true if there are entries in the Redo
// log that can be redone.
func (f *File) HasRedoableChanges() bool {
	return len(f.epsilon) > 0
}

// Size returns the complete size of the buffer including both commited
// and uncommitted runes.
// NB: naturally forwards to undo.Buffer.Size()
// TODO(rjk): needs to return the size in bytes.
func (f *File) Size() int {
	return int(f.b.nc()) + len(f.cache)
}

// Nr returns the number of valid runes in the Buffer.
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
// File.name's contents on disk and the File contents could be written
// to that disk file.
//
// When this is true, the tag's button should
// be drawn in the modified state if appropriate to the window type
// and Edit commands should treat the file as modified.
//
// TODO(rjk): figure out how this overlaps with hash. (hash would appear
// to be used to determine the "if the contents differ")
//
// TOOD(rjk): HasSaveableChanges and this overlap. They are almost
// the same and could perhaps be unified.
func (f *File) SaveableAndDirty() bool {
	return (f.mod || len(f.cache) > 0) && !f.isdir && !f.isscratch
}

// Commit sets an undo point for the current state of the file.
// The File observers are not run as part of a Commit. Observers
// only run on an InsertAt* operation.
// TODO(rjk): AFAIK. maps to undo.Buffer.Commit() correctly.
func (f *File) Commit() {
	if !f.HasUncommitedChanges() {
		return
	}

	if f.cq0 > f.b.nc() {
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
	t   int
	mod bool
	seq int
	p0  int
	n   int
	buf []rune
}

// Load inserts fd's contents into File at location q0.
// TODO(rjk): Consider renaming InsertAtFromFd or something similar.
// TODO(rjk): Read and insert in chunks.
// TODO(flux): Innefficient to load the file, then copy into the slice,
// but I need the UTF-8 interpretation.  I could fix this by using a
// UTF-8 -> []rune reader on top of the os.File instead.
func (f *File) Load(q0 int, fd *os.File, sethash bool) (n int, hasNulls bool, err error) {
	d, err := ioutil.ReadAll(fd)
	if err != nil {
		warning(nil, "read error in Buffer.Load")
	}
	runes, _, hasNulls := cvttorunes(d, len(d))

	if sethash {
		f.hash = file.CalcHash(d)
	}

	// Would appear to require a commit operation.
	// NB: Runs the observers.
	f.InsertAt(q0, runes)

	return len(runes), hasNulls, err
}

// SnapshotSeq saves the current seq to putseq. Call this on Put actions.
// TODO(rjk): switching to undo.Buffer will require removing use of seq
func (f *File) SnapshotSeq() {
	f.putseq = f.seq
}

// SeqDiffer returns true if the current seq differs from a previously snapshot.
// TODO(rjk): switching to undo.Buffer will require removing use of seq
func (f *File) SeqDiffer() bool {
	return f.seq != f.putseq
}

// AddText adds t as an observer for edits to this File.
// TODO(rjk): The observer should be an interface.
func (f *File) AddText(t *Text) *File {
	f.text = append(f.text, t)
	f.curtext = t
	return f
}

// DelText removes t as an observer for edits to this File.
// TODO(rjk): The observer should be an interface.
// TODO(rjk): Can make this more idiomatic?
func (f *File) DelText(t *Text) error {
	for i, text := range f.text {
		if text == t {
			f.text[i] = f.text[len(f.text)-1]
			f.text = f.text[:len(f.text)-1]
			if len(f.text) == 0 {
				return nil
			}
			if t == f.curtext {
				f.curtext = f.text[0]
			}
			return nil
		}
	}
	return fmt.Errorf("can't find text in File.DelText")
}

func (f *File) AllText(tf func(t *Text)) {
	for _, t := range f.text {
		tf(t)
	}
}

// HasMultipleTexts returns true if this File has multiple texts
// display its contents.
func (f *File) HasMultipleTexts() bool {
	return len(f.text) > 1
}

// InsertAt inserts s runes at rune address p0.
// TODO(rjk): run the observers here to simplify the Text code.
// TODO(rjk): In terms of the undo.Buffer conversion, this correponds
// to undo.Buffer.Insert.
// NB: At suffix is to correspond to utf8string.String.At().
func (f *File) InsertAt(p0 int, s []rune) {
	if p0 > f.b.nc() {
		panic("internal error: fileinsert")
	}
	if f.seq > 0 {
		f.Uninsert(&f.delta, p0, len(s))
	}
	f.b.Insert(p0, s)
	if len(s) != 0 {
		f.Modded()
	}
	for _, text := range f.text {
		text.inserted(p0, s)
	}
}

// InsertAtWithoutCommit inserts s at p0 without creating
// an undo record.
// TODO(rjk): Remove this as a prelude to converting to undo.Buffer
func (f *File) InsertAtWithoutCommit(p0 int, s []rune) {
	if p0 > f.b.nc()+len(f.cache) {
		panic("File.InsertAtWithoutCommit insertion off the end")
	}

	if len(f.cache) == 0 {
		f.cq0 = p0
	} else {
		if p0 != f.cq0+len(f.cache) {
			// TODO(rjk): actually print something useful here
			acmeerror("File.InsertAtWithoutCommit cq0", nil)
		}
	}
	f.cache = append(f.cache, s...)

	// run the observers
	for _, text := range f.text {
		text.inserted(p0, s)
	}
}

// Uninsert generates an action record that deletes runes from the File
// to undo an insertion.
func (f *File) Uninsert(delta *[]*Undo, q0, ns int) {
	var u Undo
	// undo an insertion by deleting
	u.t = Delete

	u.mod = f.mod
	u.seq = f.seq
	u.p0 = q0
	u.n = ns
	(*delta) = append(*delta, &u)
}

// DeleteAt removes the rune range [p0,p1) from File.
// TODO(rjk): Currently, adds an Undo record. It shouldn't
// TODO(rjk): should map onto undo.Buffer.Delete
// TODO(rjk): DeleteAt has an implied Commit operation
// that makes it not match with undo.Buffer.Delete
func (f *File) DeleteAt(p0, p1 int) {
	if !(p0 <= p1 && p0 <= f.b.nc() && p1 <= f.b.nc()) {
		acmeerror("internal error: DeleteAt", nil)
	}
	if len(f.cache) > 0 {
		acmeerror("internal error: DeleteAt", nil)
	}

	if f.seq > 0 {
		f.Undelete(&f.delta, p0, p1)
	}
	f.b.Delete(p0, p1)

	// Validate if this is right.
	if p1 > p0 {
		f.Modded()
	}
	for _, text := range f.text {
		text.deleted(p0, p1)
	}
}

// Undelete generates an action record that inserts runes into the File
// to undo a deletion.
func (f *File) Undelete(delta *[]*Undo, p0, p1 int) {
	// undo a deletion by inserting
	var u Undo
	u.t = Insert
	u.mod = f.mod
	u.seq = f.seq
	u.p0 = p0
	u.n = p1 - p0
	u.buf = make([]rune, u.n)
	f.b.Read(p0, u.buf)
	(*delta) = append(*delta, &u)
}

func (f *File) SetName(name string) {
	if f.seq > 0 {
		f.UnsetName(&f.delta)
	}
	f.name = name
}

func (f *File) UnsetName(delta *[]*Undo) {
	var u Undo
	// undo a file name change by restoring old name
	u.t = Filename
	u.mod = f.mod
	u.seq = f.seq
	u.p0 = 0 // unused
	u.n = len(f.name)
	u.buf = []rune(f.name)
	(*delta) = append(*delta, &u)
}

func NewFile(filename string) *File {
	return &File{
		b:       NewBuffer(),
		delta:   []*Undo{},
		epsilon: []*Undo{},
		elog:    MakeElog(),
		name:    filename,
		//	qidpath   uint64
		//	mtime     uint64
		//	dev       int
		editclean: true,
		//	seq       int
		mod: false,

		curtext: nil,
		text:    []*Text{},
		//	ntext   int
		//	dumpid  int
	}
}

func NewTagFile() *File {

	return &File{
		b:       NewBuffer(),
		delta:   []*Undo{},
		epsilon: []*Undo{},

		elog: MakeElog(),
		name: "",
		//	qidpath   uint64
		//	mtime     uint64
		//	dev       int
		editclean: true,
		//	seq       int
		mod: false,

		//	curtext *Text
		//	text    **Text
		//	ntext   int
		//	dumpid  int
	}
}

func (f *File) RedoSeq() int {
	delta := &f.epsilon
	if len(*delta) == 0 {
		return 0
	}
	u := (*delta)[len(*delta)-1]
	return u.seq
}

// TODO(rjk): Separate Undo and Redo for better alignment with undo.Buffer
// TODO(rjk): This Undo implementation may Undo/Redo multiple changes.
// The number actually processed is controlled by mutations to File.seq.
// This does not align with the semantics of undo.Buffer.
func (f *File) Undo(isundo bool) (q0p, q1p int) {
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
		switch u.t {
		default:
			panic(fmt.Sprintf("undo: 0x%x\n", u.t))
		case Delete:
			f.seq = u.seq
			f.Undelete(epsilon, u.p0, u.p0+u.n)
			f.mod = u.mod
			f.treatasclean = false
			f.b.Delete(u.p0, u.p0+u.n)
			for _, text := range f.text {
				text.deleted(u.p0, u.p0+u.n)
			}
			q0p = u.p0
			q1p = u.p0
		case Insert:
			f.seq = u.seq
			f.Uninsert(epsilon, u.p0, u.n)
			f.mod = u.mod
			f.treatasclean = false
			f.b.Insert(u.p0, u.buf)
			for _, text := range f.text {
				text.inserted(u.p0, u.buf)
			}
			q0p = u.p0
			q1p = u.p0 + u.n
		case Filename:
			// TODO(rjk): If I have a zerox, does undo a filename change update?
			f.seq = u.seq
			f.UnsetName(epsilon)
			f.mod = u.mod
			f.treatasclean = false
			if u.n == 0 {
				f.name = ""
			} else {
				f.name = string(u.buf)
			}
		}
		(*delta) = (*delta)[0 : len(*delta)-1]
	}
	if isundo {
		f.seq = 0
	}
	return q0p, q1p
}

// Reset removes all Undo records for this File.
// TODO(rjk): This concept doesn't particularly exist in undo.Buffer.
// Or is it part of Clean()? I think that undo.Buffer.Clean should
// reset the buffer.
// Why can't I just create a new File?
func (f *File) Reset() {
	f.delta = f.delta[0:0]
	f.epsilon = f.epsilon[0:0]
	f.seq = 0
}

// Mark starts a new set of records that can be undone as
// a unit and discards Redo records. Call this at the beginning
// of a set of edits that ought to be undo-able as a unit. This
// should be implemented in terms of undo.Buffer.Commit()
func (f *File) Mark(seq int) {
	f.epsilon = f.epsilon[0:0]
	f.seq = seq
}

// Dirty returns true if the File should be considered modified.
// TODO(rjk): This method's purpose is unclear.
func (f *File) Dirty() bool {
	return !f.treatasclean && f.mod
}

// TreatAsClean notes that the File should be considered as not Dirty
// until its next modification.
func (f *File) TreatAsClean() {
	f.treatasclean = true
}

// Modded marks the File as having changes that could be written to the
// File's backing disk file if it exists per SaveableAndDirty.
// TODO(rjk): File.mod is unneeded?
// f.mod is true when the File contents do not match the backing file.
func (f *File) Modded() {
	f.mod = true
	f.treatasclean = false
}

// Clean marks the file as not modified. In particular SaveableAndDirty()
// will return false after calling this.
// This may maps to undo.Buffer.Clean.
// TODO(rjkroege): Perhaps I should discard Undo records here?
func (f *File) Clean() {
	f.mod = false
	f.treatasclean = false
	// TODO(rjk): Should I do this? It seems desirable.
	// f.Reset()
}
