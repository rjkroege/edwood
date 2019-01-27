package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"time"
	// remove
	//	"log"
)

// File is an editable text buffer with undo. Many Text can share one
// File (to implement Zerox). The File is responsible for updating the
// Text instances. File is a model in MVC parlance while Text is a
// View-Controller.
type File struct {
	b       Buffer
	delta   []*Undo
	epsilon []*Undo
	elog    Elog
	name    string
	qidpath string // TODO(flux): Gross hack to use filename instead of qidpath for file uniqueness
	mtime   time.Time
	// dev       int
	unread bool

	// TODO(rjk): Could possibly be removed.
	editclean    bool
	seq          int
	putseq       int // seq on last put
	mod          bool
	treatasclean bool // Window Clean tests should succeed if set.

	// Observer pattern: many Text instances can share a File.
	curtext *Text
	text    []*Text
	dumpid  int

	hash FileHash // Used to check if the file has changed on disk since loaded

	// cache holds  that are not yet part of an undo record.
	cache []rune

	// TODO(rjk): may need to insert cq0 here?
	cq0 int
}

// Remember that the high-level goal is to slowly coerce this into looking like
// a scrawny wrapper around the Undo implementation. As a result, we should
// expect to see the following entry points:

// func (b *Buffer) Clean()
//func (b *Buffer) Commit()
//func (b *Buffer) Delete(off, length int64) error
//func (b *Buffer) Dirty() bool
//func (b *Buffer) Insert(off int64, data []byte) error
//func (b *Buffer) ReadAt(data []byte, off int64) (n int, err error)
//func (b *Buffer) Redo() (off, n int64)
//func (b *Buffer) Size() int64
//func (b *Buffer) Undo() (off, n int64)

// NB how the cache is folded into Buffer.
//TODO(rjk): make undo.Buffer implement Reader and Writer.

// HasUnCommittedChanges returns true if there are changes that
// have been made to the File after the last Commit.
func (t *File) HasUncommitedChanges() bool {
	return len(t.cache) != 0
}

// HasUndoableChanges returns true if there are changes to the File
// that can be undone.
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

func (f *File) HasRedoableChanges() bool {
	return len(f.epsilon) > 0
}

func (u *File) UpdateCq0(q0 int) {
	if len(u.cache) == 0 {
		u.cq0 = q0
	} else {
		if q0 != u.cq0+len(u.cache) {
			acmeerror("File.UpdateCq0 cq1", nil)
		}
	}

}

// Size returns the complete size of the buffer including both commited
// and uncommitted runes.
// NB: converts naturally to use of Undo.
// Buffers should be sized in int
func (u *File) Size() int {
	return int(u.b.Nc()) + len(u.cache)
}

// ReadC reads a single rune from the File.
// Can be trivially converted to Undo.
func (t *File) ReadC(q int) rune {
	if t.cq0 <= q && q < t.cq0+len(t.cache) {
		return t.cache[q-t.cq0]
	}
	return t.b.ReadC(q)
}

// DiffersFromDisk returns true if the File's contents differ from the
// File.name's contents. When this is true, the tag's button should
// be drawn in the modified state if appropriate to the window type.
// TODO(rjk): figure out what mod really means anyway.
// For files that aren't saved like tag Texts, it's not clear if this is
// a very good name.
func (f *File) DiffersFromDisk() bool {
	return f.mod || len(f.cache) > 0
}

// Commit pushes the edits that are not undoable to something with
// an undo record.
func (t *File) Commit(tofile bool) {
	if !t.HasUncommitedChanges() {
		return
	}
	// Do we ever call this with false?
	if tofile {
		t.Insert(t.cq0, t.cache)
	}
	t.cache = t.cache[:0]
}

// AppendCache adds to the un-committed inserts. This
func (b *File) AppendCache(rp []rune) {
	b.cache = append(b.cache, rp...)
}

// DeleteAtMostNbChars removes nb characters from the cache and
// updates the nb value.
func (t *File) DeleteAtMostNbChars(nb, q1 int, u *Text) int {
	n := len(t.cache)
	if n > 0 {
		if q1 != t.cq0+n {
			acmeerror("text.type backspace", nil)
		}
		if n > nb {
			n = nb
		}
		t.cache = t.cache[:len(t.cache)-n]
		u.Delete(q1-n, q1, false)
		nb -= n
	}

	return nb
}

// TODO(rjk): I could meld the Text.TypeCommit with HasUncommitedChanges

type Undo struct {
	t   int
	mod bool
	seq int
	p0  int
	n   int
	buf []rune
}

type FileHash [sha1.Size]byte

func (f *File) Load(q0 int, fd *os.File, sethash bool) (n int, hasNulls bool, err error) {
	var h FileHash
	n, h, hasNulls, err = f.b.Load(q0, fd)
	if sethash {
		f.hash = h
	}
	return n, hasNulls, err
}

// SnapshotSeq saves the current seq to putseq. Call this on Put actions.
func (f *File) SnapshotSeq() {
	f.putseq = f.seq
}

// SeqDiffer returns true if the current seq differs from a previously snapshot.
func (f *File) SeqDiffer() bool {
	return f.seq != f.putseq
}

func HashFile(filename string) (h FileHash, err error) {
	fd, err := os.Open(filename)
	if err != nil {
		return h, err
	}
	defer fd.Close()

	hh := sha1.New()
	if _, err := io.Copy(hh, fd); err != nil {
		return h, err
	}
	h.Set(hh.Sum(nil))
	return
}

func (h *FileHash) Set(b []byte) {
	if len(b) != len(h) {
		panic("internal error: wrong hash size")
	}
	copy(h[:], b)
}

func (h FileHash) Eq(h1 FileHash) bool {
	return bytes.Equal(h[:], h1[:])
}

func calcFileHash(b []byte) FileHash {
	return sha1.Sum(b)
}

func (f *File) AddText(t *Text) *File {
	f.text = append(f.text, t)
	f.curtext = t
	return f
}

func (f *File) DelText(t *Text) error {
	for i, text := range f.text {
		if text == t {
			f.text[i] = f.text[len(f.text)-1]
			f.text = f.text[:len(f.text)-1]
			if len(f.text) == 0 {
				f.Close()
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

func (f *File) Insert(p0 int, s []rune) {
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
}

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

func (f *File) Delete(p0, p1 int) {
	if !(p0 <= p1 && p0 <= f.b.Nc() && p1 <= f.b.Nc()) {
		acmeerror("internal error: filedelete", nil)
	}
	if f.seq > 0 {
		f.Undelete(&f.delta, p0, p1)
	}
	f.b.Delete(p0, p1)
	if p1 > p0 {
		f.Modded()
	}
}

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
	f.unread = true
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
		unread:    true,
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
		unread:    true,
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
				text.Delete(u.p0, u.p0+u.n, false)
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
				text.Insert(u.p0, u.buf, false)
			}
			q0p = u.p0
			q1p = u.p0 + u.n

		case Filename:
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

func (f *File) Reset() {
	f.delta = f.delta[0:0]
	f.epsilon = f.epsilon[0:0]
	f.seq = 0
}

func (f *File) Close() {
	f.b.Close()
	elogclose(f)
}

func (f *File) Mark() {
	f.epsilon = f.epsilon[0:0]
	f.seq = seq
}

// Dirty returns true if the File should be considered modified.
func (f *File) Dirty() bool {
	return !f.treatasclean && f.mod
}

// TreatAsClean notes that the File should be considered as not Dirty
// until its next modification.
func (f *File) TreatAsClean() {
	f.treatasclean = true
}

func (f *File) Modded() {
	f.mod = true
	f.treatasclean = false
}

func (f *File) Unmodded() {
	f.mod = false
	f.treatasclean = false
}
