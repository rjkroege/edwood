package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rjkroege/edwood/internal/file"
)

// OldEditableBuffer is an editable text buffer with undo. Many Text can share one
// OldEditableBuffer (to implement Zerox). The OldEditableBuffer is responsible for updating the
// Text instances. OldEditableBuffer is a model in MVC parlance while Text is a
// View-Controller.
//
// A OldEditableBuffer tracks several related concepts. First it is a text buffer with
// undo/redo back to an initial state. Mark (undo.RuneArray.Commit) notes
// an undo point.
//
// Next, an OldEditableBuffer might have a backing to a disk OldEditableBuffer.
//
// Lastly the text buffer might be clean/dirty. A clean buffer is possibly
// the same as its disk backing. A specific point in the undo record is
// considered clean.
//
// TODO(rjk): file.EditableBuffer will be a facade pattern composing an undo.RuneArray
// and a wrapping utf8string.String indexing wrapper.
// TODO(rjk): my version of undo.RuneArray  will implement Reader, Writer,
// RuneReader, Seeker and I will restructure this code to follow the
// patterns of the Go I/O libraries. I will probably want to provide a cache
// around undo.RuneArray.
// Observe: Character motion routines in Text can be written
// in terms of any object that is Seeker and RuneReader.
// Observe: Frame can report addresses in byte and rune offsets.
type OldEditableBuffer struct {
	b       RuneArray
	delta   []*Undo // [private]
	epsilon []*Undo // [private]
	elog    Elog
	name    string
	info    os.FileInfo

	// TODO(rjk): Remove this when I've inserted undo.RuneArray.
	// At present, InsertAt and DeleteAt have an implicit Commit operation
	// associated with them. In an undo.RuneArray context, these two ops
	// don't have an implicit Commit. We set editclean in the Edit cmd
	// implementation code to let multiple Inserts be grouped together?
	// Figure out how this inter-operates with seq.
	editclean bool

	// Tracks the Edit sequence.
	seq          int  // undo sequencing [private]
	putseq       int  // seq on last put [private]
	mod          bool // true if the file has been changed. [private]
	treatasclean bool // Window Clean tests should succeed if set. [private]

	// Observer pattern: many Text instances can share a OldEditableBuffer.
	curtext *Text
	text    []*Text // [private I think]

	isscratch bool // Used to track if this OldEditableBuffer should warn on unsaved deletion. [private]
	isdir     bool // Used to track if this OldEditableBuffer is populated from a directory list. [private]

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
// have been made to the OldEditableBuffer after the last Commit.
func (t *OldEditableBuffer) HasUncommitedChanges() bool {
	return len(t.cache) != 0
}

// HasUndoableChanges returns true if there are changes to the OldEditableBuffer
// that can be undone.
// Has no analog in buffer.Undo. It will require modification.
func (f *OldEditableBuffer) HasUndoableChanges() bool {
	return len(f.delta) > 0 || len(f.cache) != 0
}

// HasRedoableChanges returns true if there are entries in the Redo
// log that can be redone.
// Has no analog in buffer.Undo. It will require modification.
func (f *OldEditableBuffer) HasRedoableChanges() bool {
	return len(f.epsilon) > 0
}

// IsDirOrScratch returns true if the OldEditableBuffer has a synthetic backing of
// a directory listing or has a name pattern that excludes it from
// being saved under typical circumstances.
func (f *OldEditableBuffer) IsDirOrScratch() bool {
	return f.isscratch || f.isdir
}

// IsDir returns true if the OldEditableBuffer has a synthetic backing of
// a directory.
// TODO(rjk): OldEditableBuffer is a facade that subsumes the entire Model
// of an Edwood MVC. As such, it should look like a text buffer for
// view/controller code. isdir is true for a specific kind of File innards
// where we automatically alter the contents in various ways.
// Automatically altering the contents should be expressed differently.
// Directory listings should not be special cased throughout.
func (f *OldEditableBuffer) IsDir() bool {
	return f.isdir
}

// SetDir updates the setting of the isdir flag.
func (f *OldEditableBuffer) SetDir(flag bool) {
	f.isdir = flag
}

// Size returns the complete size of the buffer including both committed
// and uncommitted runes.
// NB: naturally forwards to undo.RuneArray.Size()
// TODO(rjk): Switch all callers to Nr() as would be the number of
// bytes when backed by undo.RuneArray.
func (f *OldEditableBuffer) Size() int {
	return int(f.b.nc()) + len(f.cache)
}

// Nr returns the number of valid runes in the RuneArray.
// At the moment, this is the same as Size. But when OldEditableBuffer is backed
// with utf8, this will require adjustment.
// TODO(rjk): utf8 adjustment
func (f *OldEditableBuffer) Nr() int {
	return f.Size()
}

// ReadC reads a single rune from the OldEditableBuffer.
// Can be easily converted to being utf8 backed but
// every caller will require adjustment.
// TODO(rjk): OldEditableBuffer needs to implement RuneReader and code should
// use that interface instead.
// TODO(rjk): Better name to align with utf8string.String.At().
func (f *OldEditableBuffer) ReadC(q int) rune {
	if f.cq0 <= q && q < f.cq0+len(f.cache) {
		return f.cache[q-f.cq0]
	}
	return f.b.ReadC(q)
}

// ReadAtRune reads at most len(r) runes from OldEditableBuffer at rune off.
// It returns the number of  runes read and an error if something goes wrong.
func (f *OldEditableBuffer) ReadAtRune(r []rune, off int) (n int, err error) {
	// TODO(rjk): This should include cache contents but currently
	// callers do not require it to.
	return f.b.Read(off, r)
}

// SaveableAndDirty returns true if the OldEditableBuffer's contents differ from the
// backing diskfile OldEditableBuffer.name, and the diskfile is plausibly writable
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
// as clean and is this OldEditableBuffer writable to a backing. They are combined in this
// this method.
func (f *OldEditableBuffer) SaveableAndDirty() bool {
	return f.name != "" && (f.mod || f.Dirty() || len(f.cache) > 0) && !f.IsDirOrScratch()
}

// Commit writes the in-progress edits to the real buffer instead of
// keeping them in the cache. Does not map to undo.RuneArray.Commit (that
// method is Mark). Remove this method.
func (f *OldEditableBuffer) Commit() {
	f.treatasclean = false
	if !f.HasUncommitedChanges() {
		return
	}

	if f.cq0 > f.b.nc() {
		// TODO(rjk): Generate a better error message.
		panic("internal error: OldEditableBuffer.Commit")
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

// Load inserts fd's contents into OldEditableBuffer at location q0. Load will always
// mark the file as modified so follow this up with a call to f.Clean() to
// indicate that the file corresponds to its disk file backing.
// TODO(rjk): hypothesis: we can make this API cleaner: we will only
// compute a hash when the file corresponds to its diskfile right?
// TODO(rjk): Consider renaming InsertAtFromFd or something similar.
// TODO(rjk): Read and insert in chunks.
// TODO(flux): Innefficient to load the file, then copy into the slice,
// but I need the UTF-8 interpretation.  I could fix this by using a
// UTF-8 -> []rune reader on top of the os.File instead.
func (f *OldEditableBuffer) Load(q0 int, fd io.Reader, sethash bool) (n int, hasNulls bool, err error) {
	d, err := ioutil.ReadAll(fd)
	if err != nil {
		warning(nil, "read error in RuneArray.Load")
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

// UpdateInfo updates OldEditableBuffer's info to d if file hash hasn't changed.
func (f *OldEditableBuffer) UpdateInfo(filename string, d os.FileInfo) error {
	h, err := file.HashFor(filename)
	if err != nil {
		return warnError(nil, "failed to compute hash for %v: %v", filename, err)
	}
	if h.Eq(f.hash) {
		f.info = d
	}
	return nil
}

// SnapshotSeq saves the current seq to putseq. Call this on Put actions.
// TODO(rjk): switching to undo.RuneArray will require removing use of seq
// TODO(rjk): This function maps to undo.RuneArray.Clean()
func (f *OldEditableBuffer) SnapshotSeq() {
	f.putseq = f.seq
}

// Dirty reports whether the current state of the OldEditableBuffer is different from
// the initial state or from the one at the time of calling Clean.
//
// TODO(rjk): switching to undo.RuneArray will require removing external uses
// of seq.
func (f *OldEditableBuffer) Dirty() bool {
	return f.seq != f.putseq
}

// AddText adds t as an observer for edits to this OldEditableBuffer.
// TODO(rjk): The observer should be an interface.
func (f *OldEditableBuffer) AddText(t *Text) *OldEditableBuffer {
	f.text = append(f.text, t)
	f.curtext = t
	return f
}

// DelText removes t as an observer for edits to this OldEditableBuffer.
// TODO(rjk): The observer should be an interface.
// TODO(rjk): Can make this more idiomatic?
func (f *OldEditableBuffer) DelText(t *Text) error {
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
	return fmt.Errorf("can't find text in OldEditableBuffer.DelText")
}

func (f *OldEditableBuffer) AllText(tf func(t *Text)) {
	for _, t := range f.text {
		tf(t)
	}
}

// HasMultipleTexts returns true if this OldEditableBuffer has multiple texts
// display its contents.
func (f *OldEditableBuffer) HasMultipleTexts() bool {
	return len(f.text) > 1
}

// InsertAt inserts s runes at rune address p0.
// TODO(rjk): run the observers here to simplify the Text code.
// TODO(rjk): In terms of the undo.RuneArray conversion, this correponds
// to undo.RuneArray.Insert.
// NB: At suffix is to correspond to utf8string.String.At().
func (f *OldEditableBuffer) InsertAt(p0 int, s []rune) {
	f.treatasclean = false
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
// TODO(rjk): Remove this as a prelude to converting to undo.RuneArray
// But preserve the cache. Every "small" insert should go into the cache.
// It almost certainly greatly improves performance for a series of single
// character insertions.
func (f *OldEditableBuffer) InsertAtWithoutCommit(p0 int, s []rune) {
	f.treatasclean = false
	if p0 > f.b.nc()+len(f.cache) {
		panic("OldEditableBuffer.InsertAtWithoutCommit insertion off the end")
	}

	if len(f.cache) == 0 {
		f.cq0 = p0
	} else {
		if p0 != f.cq0+len(f.cache) {
			// TODO(rjk): actually print something useful here
			acmeerror("OldEditableBuffer.InsertAtWithoutCommit cq0", nil)
		}
	}
	f.cache = append(f.cache, s...)

	// run the observers
	for _, text := range f.text {
		text.inserted(p0, s)
	}
}

// Uninsert generates an action record that deletes runes from the OldEditableBuffer
// to undo an insertion.
func (f *OldEditableBuffer) Uninsert(delta *[]*Undo, q0, ns int) {
	var u Undo
	// undo an insertion by deleting
	u.t = Delete

	u.mod = f.mod
	u.seq = f.seq
	u.p0 = q0
	u.n = ns
	(*delta) = append(*delta, &u)
}

// DeleteAt removes the rune range [p0,p1) from OldEditableBuffer.
// TODO(rjk): Currently, adds an Undo record. It shouldn't
// TODO(rjk): should map onto undo.RuneArray.Delete
// TODO(rjk): DeleteAt has an implied Commit operation
// that makes it not match with undo.RuneArray.Delete
func (f *OldEditableBuffer) DeleteAt(p0, p1 int) {
	f.treatasclean = false
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

// Undelete generates an action record that inserts runes into the OldEditableBuffer
// to undo a deletion.
func (f *OldEditableBuffer) Undelete(delta *[]*Undo, p0, p1 int) {
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

// A OldEditableBuffer can have a spcific name that permit it to be persisted to disk
// but typically would not be. These two constants are suffixes of OldEditableBuffer
// names that have this property.
const (
	slashguide = "/guide"
	plusErrors = "+Errors"
)

// SetName sets the name of the backing for this file.
// Some backings that opt them out of typically being persisted.
// Resetting a file name to a new value does not have any effect.
func (f *OldEditableBuffer) SetName(name string) {
	if f.name == name {
		return
	}

	if f.seq > 0 {
		f.UnsetName(&f.delta)
	}
	f.setnameandisscratch(name)
}

// setnameandisscratch updates the OldEditableBuffer.name and isscratch bit
// at the same time.
func (f *OldEditableBuffer) setnameandisscratch(name string) {
	f.name = name
	if strings.HasSuffix(name, slashguide) || strings.HasSuffix(name, plusErrors) {
		f.isscratch = true
	} else {
		f.isscratch = false
	}
}

func (f *OldEditableBuffer) UnsetName(delta *[]*Undo) {
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

func NewFile(filename string) *OldEditableBuffer {
	return &OldEditableBuffer{
		b:         NewBuffer(),
		delta:     []*Undo{},
		epsilon:   []*Undo{},
		elog:      MakeElog(),
		name:      filename,
		editclean: true,
		//	seq       int
		mod: false,

		curtext: nil,
		text:    []*Text{},
		//	ntext   int
	}
}

func NewTagFile() *OldEditableBuffer {

	return &OldEditableBuffer{
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
	}
}

// RedoSeq finds the seq of the last redo record. TODO(rjk): This has no
// analog in undo.RuneArray. The value of seq is used to track intra and
// inter OldEditableBuffer edit actions so that cross-OldEditableBuffer changes via Edit X can be
// undone with a single action. An implementation of OldEditableBuffer that wraps
// undo.RuneArray will need to to preserve seq tracking.
func (f *OldEditableBuffer) RedoSeq() int {
	delta := &f.epsilon
	if len(*delta) == 0 {
		return 0
	}
	u := (*delta)[len(*delta)-1]
	return u.seq
}

// Seq returns the current value of seq.
func (f *OldEditableBuffer) Seq() int {
	return f.seq
}

// Undo undoes edits if isundo is true or redoes edits if isundo is false.
// It returns the new selection q0, q1 and a bool indicating if the
// returned selection is meaningful.
//
// TODO(rjk): Separate Undo and Redo for better alignment with undo.RuneArray
// TODO(rjk): This Undo implementation may Undo/Redo multiple changes.
// The number actually processed is controlled by mutations to OldEditableBuffer.seq.
// This does not align with the semantics of undo.RuneArray.
// Each "Mark" needs to have a seq value provided.
// TODO(rjk): Consider providing the target seq value as an argument.
func (f *OldEditableBuffer) Undo(isundo bool) (q0, q1 int, ok bool) {
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
			q0 = u.p0
			q1 = u.p0
			ok = true
		case Insert:
			f.seq = u.seq
			f.Uninsert(epsilon, u.p0, u.n)
			f.mod = u.mod
			f.treatasclean = false
			f.b.Insert(u.p0, u.buf)
			for _, text := range f.text {
				text.inserted(u.p0, u.buf)
			}
			q0 = u.p0
			q1 = u.p0 + u.n
			ok = true
		case Filename:
			// TODO(rjk): If I have a zerox, does undo a filename change update?
			f.seq = u.seq
			f.UnsetName(epsilon)
			f.mod = u.mod
			f.treatasclean = false
			newfname := string(u.buf)
			f.setnameandisscratch(newfname)
		}
		(*delta) = (*delta)[0 : len(*delta)-1]
	}
	// TODO(rjk): Why do we do this?
	if isundo {
		f.seq = 0
	}
	return q0, q1, ok
}

// Reset removes all Undo records for this OldEditableBuffer.
// TODO(rjk): This concept doesn't particularly exist in undo.RuneArray.
// Why can't I just create a new OldEditableBuffer?
func (f *OldEditableBuffer) Reset() {
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
func (f *OldEditableBuffer) Mark(seq int) {
	f.epsilon = f.epsilon[0:0]
	f.seq = seq
}

// TreatAsDirty returns true if the OldEditableBuffer should be considered modified
// for the purpose of warning the user if Del-ing a Dirty() file.
func (f *OldEditableBuffer) TreatAsDirty() bool {
	return !f.treatasclean && f.Dirty()
}

// TreatAsClean notes that the OldEditableBuffer should be considered as not Dirty
// until its next modification.
func (f *OldEditableBuffer) TreatAsClean() {
	f.treatasclean = true
}

// Modded marks the OldEditableBuffer if we know that its backing is different from
// its contents. This is needed to track when Edwood has modified the
// backing without changing the OldEditableBuffer (e.g. via the Edit w command.
func (f *OldEditableBuffer) Modded() {
	f.mod = true
	f.treatasclean = false
}

// Clean marks OldEditableBuffer as being non-dirty: the backing is the same as OldEditableBuffer.
func (f *OldEditableBuffer) Clean() {
	f.mod = false
	f.treatasclean = false
	f.SnapshotSeq()
}
