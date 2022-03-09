package file

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rjkroege/edwood/sam"
	"github.com/rjkroege/edwood/util"
)

// The ObservableEditableBuffer is used by the main program to add,
// remove and check on the current observer(s) for a Text. Text in turn,
// implements BufferObserver for the various required callback functions
// in BufferObserver.
type ObservableEditableBuffer struct {
	currobserver    BufferObserver
	observers       map[BufferObserver]struct{}
	statusobservers map[TagStatusObserver]struct{}

	f *Buffer

	Elog sam.Elog

	// Used to note that the oeb's contents will be replaced with a new disk backing
	// when the Elog is applied and should be marked Clean() at that time.
	EditClean bool

	details *DiskDetails

	// Tracks the editing sequence.
	seq    int // undo sequencing
	putseq int // seq on last put

	// TODO(rjk): Can we get rid of these two booleans?
	isscratch    bool // Used to track if this File should warn on unsaved deletion.
	treatasclean bool // Toggle to override the Dirty check on closing a buffer with unsaved changes.

	filtertagobservers bool // If true, TagStatus updates are filtered.
}

// A ObservableEditableBuffer can have a specific file-backing name that
// permits it to be persisted to disk but typically would not be. These
// two constants are suffixes of disk-file names that have this property.
// TODO(rjk): Consider making this a detail of file.Details?
const (
	slashguide = "/guide"
	plusErrors = "+Errors"
)

// Set is a forwarding function for file_hash.Set
func (e *ObservableEditableBuffer) Set(hash []byte) {
	e.details.Hash.Set(hash)
}

func (e *ObservableEditableBuffer) SetInfo(info os.FileInfo) {
	e.details.Info = info
}

// AddObserver adds e as an observer for edits to this File.
func (e *ObservableEditableBuffer) AddObserver(observer BufferObserver) {
	if e.observers == nil {
		e.observers = make(map[BufferObserver]struct{})
	}
	e.observers[observer] = struct{}{}
	e.currobserver = observer
}

// DelObserver removes e as an observer for edits to this File.
func (e *ObservableEditableBuffer) DelObserver(observer BufferObserver) error {
	if _, exists := e.observers[observer]; exists {
		delete(e.observers, observer)
		if observer == e.currobserver {
			for k := range e.observers {
				e.currobserver = k
				break
			}
		}
		return nil
	}
	// This never happens right?
	return fmt.Errorf("can't find editor in File.DelObserver")
}

// SetCurObserver sets the current observer.
func (e *ObservableEditableBuffer) SetCurObserver(observer BufferObserver) {
	e.currobserver = observer
}

// GetCurObserver gets the current observer and returns it as a interface type.
func (e *ObservableEditableBuffer) GetCurObserver() BufferObserver {
	return e.currobserver
}

// AllObservers preforms tf(all observers...).
func (e *ObservableEditableBuffer) AllObservers(tf func(i interface{})) {
	for t := range e.observers {
		tf(t)
	}
}

// AddTagStatusObserver adds obs as a status observer.
func (e *ObservableEditableBuffer) AddTagStatusObserver(obs TagStatusObserver) {
	if e.statusobservers == nil {
		e.statusobservers = make(map[TagStatusObserver]struct{})
	}
	e.statusobservers[obs] = struct{}{}
}

// DelTagStatusObserver removes e as an observer for edits to this File.
func (e *ObservableEditableBuffer) DelTagStatusObserver(obs TagStatusObserver) {
	if _, exists := e.statusobservers[obs]; exists {
		delete(e.statusobservers, obs)
		return
	}
}

// GetObserverSize will return the size of the observer map.
func (e *ObservableEditableBuffer) GetObserverSize() int {
	return len(e.observers)
}

// HasMultipleObservers returns true if their are multiple observers to the File.
func (e *ObservableEditableBuffer) HasMultipleObservers() bool {
	return len(e.observers) > 1
}

// MakeObservableEditableBuffer is a constructor wrapper for NewFile() to abstract File from the main program.
func MakeObservableEditableBuffer(filename string, b []rune) *ObservableEditableBuffer {
	return _makeObservableEditableBuffer(filename, b, true)
}

func _makeObservableEditableBuffer(filename string, b []rune, newtype bool) *ObservableEditableBuffer {
	oeb := &ObservableEditableBuffer{
		currobserver: nil,
		observers:    nil,
		details:      &DiskDetails{Name: filename, Hash: Hash{}},
		Elog:         sam.MakeElog(),
		EditClean:    true,
	}
	oeb.f = NewTypeBuffer(b, oeb)
	return oeb
}

// Clean marks the ObservableEditableBuffer as being non-dirty: the
// backing is the same as File. In particular, invoked in response to a
// Put operation.
func (e *ObservableEditableBuffer) Clean() {
	before := e.getTagStatus()

	e.treatasclean = false
	op := e.putseq
	e.putseq = e.seq

	e.notifyTagObservers(before)

	if op != e.seq {
		e.filtertagobservers = false
	}
}

// getTagState returns the current tag state. Assumption: this method needs to be cheap to
// call.
func (e *ObservableEditableBuffer) getTagStatus() TagStatus {
	return TagStatus{
		UndoableChanges:  e.HasUndoableChanges(),
		RedoableChanges:  e.HasRedoableChanges(),
		SaveableAndDirty: e.SaveableAndDirty(),
	}
}

// notifyTagObservers will invoke the tag state observers (e.g. the
// Window instances that will want to adjust their tags to reflect
// alterations to this buffer.) Invoke this function at the end of any
// entry point that mutates the state of the ObservableEditableBuffer in a
// way that would mutate the tag contents.
func (e *ObservableEditableBuffer) notifyTagObservers(before TagStatus) {
	after := e.getTagStatus()
	if e.filtertagobservers && before == after {
		return
	}
	e.filtertagobservers = true

	for t := range e.statusobservers {
		t.UpdateTag(after)
	}
}

// Mark is a forwarding function for file.Mark.
// This sets an undo point. NB: call Mark before mutating the file.
// seq must be 1 to enable Undo/Redo on the file.
func (e *ObservableEditableBuffer) Mark(seq int) {
	e.f.Mark()
	e.seq = seq
}

// HasUncommitedChanges is a forwarding function for file.HasUncommitedChanges.
// Should be a nop with file.Buffer
func (e *ObservableEditableBuffer) HasUncommitedChanges() bool {
	return e.f.HasUncommitedChanges()
}

// HasRedoableChanges is a forwarding function for file.HasRedoableChanges.
func (e *ObservableEditableBuffer) HasRedoableChanges() bool {
	return e.f.HasRedoableChanges()
}

// HasUndoableChanges is a forwarding function for file.HasUndoableChanges
func (e ObservableEditableBuffer) HasUndoableChanges() bool {
	if e.seq > 0 {
		return e.f.HasUndoableChanges() || e.f.HasUncommitedChanges()
	}
	return false
}

// IsDir is a forwarding function for DiskDetails.IsDir.
func (e *ObservableEditableBuffer) IsDir() bool {
	return e.details.IsDir()
}

// SetDir is a forwarding function for DiskDetails.SetDir.
func (e *ObservableEditableBuffer) SetDir(flag bool) {
	e.details.SetDir(flag)
}

// Nr is a forwarding function for file.Nr.
func (e *ObservableEditableBuffer) Nr() int {
	return e.f.Nr()
}

// ReadC is a forwarding function for file.ReadC.
func (e *ObservableEditableBuffer) ReadC(q int) rune {
	return e.f.ReadC(q)
}

// SaveableAndDirty returns true if the ObservableEditableBuffer's
// contents differ from the backing diskfile File.name, and the diskfile
// is plausibly writable (not a directory or scratch file).
//
// When this is true, the tag's button should
// be drawn in the modified state if appropriate to the window type
// and Edit commands should treat the file as modified.
//
// TODO(rjk): figure out how this overlaps with hash. (hash would appear
// to be used to determine the "if the contents differ")
//
// Latest thought: there are two separate issues: are we at a point marked
// as clean and is this File writable to a backing. They are combined in
// this method.
func (e *ObservableEditableBuffer) SaveableAndDirty() bool {
	sad := e.Dirty() && !e.IsDirOrScratch()
	return e.details.Name != "" && sad
}

// Load inserts fd's contents into File at location q0. Typically, follow
// this up with a call to f.Clean() to indicate that the file corresponds
// to its disk file backing.
//
// TODO(rjk): hypothesis: we can make this API cleaner: we will only
// compute a hash when the file corresponds to its diskfile right?
//
// TODO(rjk): Consider renaming InsertAtFromFd or something similar.
//
// TODO(rjk): Read and insert in chunks.
//
// TODO(flux): Innefficient to load the file, then copy into the slice,
// but I need the UTF-8 interpretation. I could fix this by using a UTF-8
// -> []rune reader on top of the os.File instead.
//
func (e *ObservableEditableBuffer) Load(q0 int, fd io.Reader, sethash bool) (int, bool, error) {
	d, err := ioutil.ReadAll(fd)
	// TODO(rjk): improve handling of read errors.
	if err != nil {
		err = errors.New("read error in RuneArray.Load")
	}
	if sethash {
		e.SetHash(CalcHash(d))
	}

	runes, _, hasNulls := util.Cvttorunes(d, len(d))
	e.InsertAt(q0, runes)
	return len(runes), hasNulls, err
}

// Dirty returns true when the ObservableEditableBuffer differs from its disk
// backing as tracked by the undo system.
func (e *ObservableEditableBuffer) Dirty() bool {
	return e.seq != e.putseq
}

// InsertAt is a forwarding function for file.InsertAt.
// p0 is position in runes.
func (e *ObservableEditableBuffer) InsertAt(rp0 int, rs []rune) {
	p0 := e.f.RuneTuple(rp0)
	s, nr := RunesToBytes(rs)

	e.Insert(p0, s, nr)
}

// Insert is a forwarding function for file.Insert.
// p0 is position in runes.
func (e *ObservableEditableBuffer) Insert(p0 OffsetTuple, s []byte, nr int) {
	before := e.getTagStatus()
	defer e.notifyTagObservers(before)

	e.f.Insert(p0, s, nr, e.seq)
	if e.seq < 1 {
		e.f.FlattenHistory()
	}
	e.inserted(p0, s, nr)
}

// SetName sets the name of the backing for this file. Some backing names
// are "virtual": the name is displayed in the ObservableEditableBuffer's
// corresponding tag but there is no backing. Setting e's name to its
// existing value will not invoke the observers.
func (e *ObservableEditableBuffer) SetName(name string) {
	if e.Name() == name {
		return
	}

	// SetName always forces an update of the tag.
	// TODO(rjk): This reset of filtertagobservers might now be unnecessary.
	e.filtertagobservers = false
	before := e.getTagStatus()
	defer e.notifyTagObservers(before)

	if e.seq > 0 {
		// TODO(rjk): Pass in the name, make the function name better reflect its purpose.
		e.f.UnsetName(e.Name(), e.seq)
	}
	e.setfilename(name)
}

// Undo is a forwarding function for file.Undo.
func (e *ObservableEditableBuffer) Undo(isundo bool) (q0, q1 int, ok bool) {
	before := e.getTagStatus()
	defer e.notifyTagObservers(before)

	if isundo {
		q0, q1, ok, e.seq = e.f.Undo(e.seq)
	} else {
		q0, q1, ok, e.seq = e.f.Redo(e.seq)
	}
	return q0, q1, ok
}

// DeleteAt is a forwarding function for buffer.DeleteAt.
// rp0, rp1 are in runes.
func (e *ObservableEditableBuffer) DeleteAt(rp0, rp1 int) {
	p0 := e.f.RuneTuple(rp0)
	p1 := e.f.RuneTuple(rp1)

	e.Delete(p0, p1)
}

// Delete is a forwarding function for buffer.Delete.
func (e *ObservableEditableBuffer) Delete(q0, q1 OffsetTuple) {
	before := e.getTagStatus()
	defer e.notifyTagObservers(before)

	e.f.Delete(q0, q1, e.seq)
	if e.seq < 1 {
		e.f.FlattenHistory()
	}
	e.deleted(q0, q1)
}

// TreatAsClean is a forwarding function for file.TreatAsClean.
func (e *ObservableEditableBuffer) TreatAsClean() {
	e.treatasclean = true
}

// Modded marks the File if we know that its backing is different from
// its contents. This is needed to track when Edwood has modified the
// backing without changing the File (e.g. via the Edit w command.)
func (e *ObservableEditableBuffer) Modded() {
	before := e.getTagStatus()
	defer e.notifyTagObservers(before)

	e.putseq = -1
	e.treatasclean = false
}

// Name is a getter for file.DiskDetails.Name.
func (e *ObservableEditableBuffer) Name() string {
	return e.details.Name
}

// Info is a Getter for e.DiskDetails.Info
func (e *ObservableEditableBuffer) Info() os.FileInfo {
	return e.details.Info
}

// UpdateInfo is a forwarding function for DiskDetails.UpdateInfo
func (e *ObservableEditableBuffer) UpdateInfo(filename string, d os.FileInfo) error {
	return e.details.UpdateInfo(filename, d)
}

// Hash is a getter for DiskDetails.Hash
func (e *ObservableEditableBuffer) Hash() Hash {
	return e.details.Hash
}

// SetHash is a setter for DiskDetails.Hash
func (e *ObservableEditableBuffer) SetHash(hash Hash) {
	e.details.Hash = hash
}

// Seq is a getter for file.details.Seq.
func (e *ObservableEditableBuffer) Seq() int {
	return e.seq
}

// RedoSeq finds the seq of the last redo record. Forwards its
// implementation to file.File or file.Buffer.
func (e *ObservableEditableBuffer) RedoSeq() int {
	return e.f.RedoSeq()
}

// inserted is a package-only entry point from the underlying
// buffer (file.Buffer or file.File) to run the registered observers
// on a change in the buffer.
func (e *ObservableEditableBuffer) inserted(q0 OffsetTuple, b []byte, nr int) {
	e.treatasclean = false
	for observer := range e.observers {
		observer.Inserted(q0, b, nr)
	}
}

// deleted is a package-only entry point from the underlying
// buffer (file.Buffer or file.File) to run the registered observers
// on a change in the buffer.
func (e *ObservableEditableBuffer) deleted(q0, q1 OffsetTuple) {
	e.treatasclean = false
	for observer := range e.observers {
		observer.Deleted(q0, q1)
	}
}

// Commit is a forwarding function for file.Commit.
// nop with file.Buffer.
func (e *ObservableEditableBuffer) Commit() {
	e.f.Commit(e.seq)
}

// InsertAtWithoutCommit is a forwarding function for file.InsertAtWithoutCommit.
// forwards to InsertAt for file.Buffer.
func (e *ObservableEditableBuffer) InsertAtWithoutCommit(p0 int, s []rune) {
	e.InsertAt(p0, s)
}

// IsDirOrScratch returns true if the File has a synthetic backing of
// a directory listing or has a name pattern that excludes it from
// being saved under typical circumstances.
func (e *ObservableEditableBuffer) IsDirOrScratch() bool {
	return e.isscratch || e.IsDir()
}

// TreatAsDirty returns true if the File should be considered modified
// for the purpose of warning the user if Del-ing a Dirty() file.
// TODO(rjk): Consider removing this.
func (e *ObservableEditableBuffer) TreatAsDirty() bool {
	return !e.treatasclean && e.Dirty()
}

// Read is a forwarding function for rune_array.Read.
// q0 is in runes
// ReadC can be implemented in terms of Read when using file.Buffer
// because the "cache" concept is not germane.
func (e *ObservableEditableBuffer) Read(q0 int, r []rune) (int, error) {
	return e.f.Read(q0, r)
}

// String is a forwarding function for rune_array.String.
// Returns the entire buffer as a string.
// TODO(rjk): Consider making this aware of the cache. (If test results depend
// on this not
func (e *ObservableEditableBuffer) String() string {
	return e.f.String()
}

// ResetBuffer is a forwarding function for rune_array.Reset. Equivalent
// to re-creating the buffer.
func (e *ObservableEditableBuffer) ResetBuffer() {
	e.filtertagobservers = false
	e.seq = 0
	e.f = NewTypeBuffer([]rune{}, e)
}

// Reader is a forwarding function for rune_array.Reader.
func (e *ObservableEditableBuffer) Reader(q0 int, q1 int) io.Reader {
	return e.f.Reader(q0, q1)
}

// IndexRune is a forwarding function for rune_array.IndexRune.
func (e *ObservableEditableBuffer) IndexRune(r rune) int {
	return e.f.IndexRune(r)
}

// setfilename updates the oeb.details.name and isscratch bit at the same
// time. The underlying buffer (file.Buffer or file.File) needs to invoke
// this when Undo-ing a filename change.
//
// If we get here via invoking Undo (e.g. oeb.Undo, file.Undo,
// oeb.setfilename), we will execute the tag update observers if
// appropriate to update the tag status.
func (e *ObservableEditableBuffer) setfilename(name string) {
	e.treatasclean = false

	e.details.Name = name
	if strings.HasSuffix(name, slashguide) || strings.HasSuffix(name, plusErrors) {
		e.isscratch = true
	} else {
		e.isscratch = false
	}
}

// SetSeq is a setter for file.seq for use in tests.
func (e *ObservableEditableBuffer) SetSeq(seq int) {
	e.seq = seq
}

// SetPutseq is a setter for file.putseq for use in tests.
func (e *ObservableEditableBuffer) SetPutseq(putseq int) {
	e.putseq = putseq
}

// RuneTuple is a forwarding function.
func (e *ObservableEditableBuffer) RuneTuple(q int) OffsetTuple {
	return e.f.RuneTuple(q)
}

// ByteTuple is a forwarding function.
func (e *ObservableEditableBuffer) ByteTuple(q int) OffsetTuple {
	return e.f.ByteTuple(q)
}

// End is a forwarding function.
func (e *ObservableEditableBuffer) End() OffsetTuple {
	return e.f.End()
}

// MakeBufferCursor is a forwarding function.
func (e *ObservableEditableBuffer) MakeBufferCursor(p0, p1 OffsetTuple) *BufferCursor {
	return MakeBufferCursor(e.f, p0, p1)
}
