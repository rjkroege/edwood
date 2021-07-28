package main

import (
	"fmt"
	"github.com/rjkroege/edwood/internal/file"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// The ObservableEditableBuffer is used by the main program
// to add, remove and check on the current observer(s) for a Text.
// Text in turn, implements BufferObserver for the various required callback functions in BufferObserver.
type ObservableEditableBuffer struct {
	currobserver BufferObserver
	observers    map[BufferObserver]struct{} // [private I think]
	f            *File
	details      *file.DiskDetails
}

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
	return fmt.Errorf("can't find editor in File.DelObserver")
}

// SetCurObserver sets the current observer.
func (e *ObservableEditableBuffer) SetCurObserver(observer BufferObserver) {
	e.currobserver = observer
}

// GetCurObserver gets the current observer and returns it as a interface type.
func (e *ObservableEditableBuffer) GetCurObserver() interface{} {
	return e.currobserver
}

// AllObservers preforms tf(all observers...).
func (e *ObservableEditableBuffer) AllObservers(tf func(i interface{})) {
	for t := range e.observers {
		tf(t)
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
func MakeObservableEditableBuffer(filename string, b RuneArray) *ObservableEditableBuffer {
	f := NewFile()
	f.b = b
	oeb := &ObservableEditableBuffer{
		currobserver: nil,
		observers:    nil,
		f:            f,
		details:      &file.DiskDetails{Name: filename, Hash: file.Hash{}},
	}
	oeb.f.oeb = oeb
	return oeb
}

// MakeObservableEditableBufferTag is a constructor wrapper for NewTagFile() to abstract File from the main program.
func MakeObservableEditableBufferTag(b RuneArray) *ObservableEditableBuffer {
	f := NewTagFile()
	f.b = b
	oeb := &ObservableEditableBuffer{
		currobserver: nil,
		observers:    nil,
		f:            f,
		details:      &file.DiskDetails{Hash: file.Hash{}},
	}
	oeb.f.oeb = oeb
	return oeb
}

// Clean is a forwarding function for file.Clean.
func (e *ObservableEditableBuffer) Clean() {
	e.f.Clean()
}

// Size is a forwarding function for file.Size.
func (e *ObservableEditableBuffer) Size() int {
	return e.f.Size()
}

// Mark is a forwarding function for file.Mark.
func (e *ObservableEditableBuffer) Mark(seq int) {
	e.f.Mark(seq)
}

// Reset is a forwarding function for file.Reset.
func (e *ObservableEditableBuffer) Reset() {
	e.f.Reset()
}

// HasUncommitedChanges is a forwarding function for file.HasUncommitedChanges.
func (e *ObservableEditableBuffer) HasUncommitedChanges() bool {
	return e.f.HasUncommitedChanges()
}

// HasRedoableChanges is a forwarding function for file.HasRedoableChanges.
func (e *ObservableEditableBuffer) HasRedoableChanges() bool {
	return e.f.HasRedoableChanges()
}

// HasUndoableChanges is a forwarding function for file.HasUndoableChanges
func (e ObservableEditableBuffer) HasUndoableChanges() bool {
	return e.f.HasUndoableChanges()
}

// IsDir is a forwarding function for file.IsDir.
func (e *ObservableEditableBuffer) IsDir() bool {
	return e.f.IsDir()
}

// SetDir is a forwarding function for file.SetDir.
func (e *ObservableEditableBuffer) SetDir(flag bool) {
	e.f.SetDir(flag)
}

// Nr is a forwarding function for file.Nr.
func (e *ObservableEditableBuffer) Nr() int {
	return e.f.Nr()
}

// ReadC is a forwarding function for file.ReadC.
func (e *ObservableEditableBuffer) ReadC(q int) rune {
	return e.f.ReadC(q)
}

// SaveableAndDirty is a forwarding function for file.SaveableAndDirty.
func (e *ObservableEditableBuffer) SaveableAndDirty() bool {
	return e.details.Name != "" && e.f.SaveableAndDirty()
}

// Load is a forwarding function for file.Load.
func (e *ObservableEditableBuffer) Load(q0 int, fd io.Reader, sethash bool) (n int, hasNulls bool, err error) {
	d, err := ioutil.ReadAll(fd)
	if err != nil {
		warning(nil, "read error in RuneArray.Load")
	}
	if sethash {
		e.SetHash(file.CalcHash(d))
	}

	return e.f.Load(q0, d)
}

// Dirty is a forwarding function for file.Dirty.
func (e *ObservableEditableBuffer) Dirty() bool {
	return e.f.Dirty()
}

// InsertAt is a forwarding function for file.InsertAt.
func (e *ObservableEditableBuffer) InsertAt(p0 int, s []rune) {
	e.f.InsertAt(p0, s)
}

// SetName is a forwarding function for file.SetName.
func (e *ObservableEditableBuffer) SetName(name string) {
	e.f.SetName(name)
}

// Undo is a forwarding function for file.Undo.
func (e *ObservableEditableBuffer) Undo(isundo bool) (q0, q1 int, ok bool) {
	return e.f.Undo(isundo)
}

// DeleteAt is a forwarding function for file.DeleteAt.
func (e *ObservableEditableBuffer) DeleteAt(q0, q1 int) {
	e.f.DeleteAt(q0, q1)
}

// TreatAsClean is a forwarding function for file.TreatAsClean.
func (e *ObservableEditableBuffer) TreatAsClean() {
	e.f.TreatAsClean()
}

// Modded is a forwarding function for file.Modded.
func (e *ObservableEditableBuffer) Modded() {
	e.f.Modded()
}

// Name is a getter for file.details.Name.
func (e *ObservableEditableBuffer) Name() string {
	return e.details.Name
}

// Info is a Getter for e.details.Info
func (e *ObservableEditableBuffer) Info() os.FileInfo {
	return e.details.Info
}

// UpdateInfo is a forwarding function for file.UpdateInfo
func (e *ObservableEditableBuffer) UpdateInfo(filename string, d os.FileInfo) error {
	return e.details.UpdateInfo(filename, d)
}

// Hash is a getter for DiskDetails.Hash
func (e *ObservableEditableBuffer) Hash() file.Hash {
	return e.details.Hash
}

// SetHash is a setter for DiskDetails.Hash
func (e *ObservableEditableBuffer) SetHash(hash file.Hash) {
	e.details.Hash = hash
}

// Seq is a getter for file.details.Seq.
func (e *ObservableEditableBuffer) Seq() int {
	return e.f.seq
}

// RedoSeq is a getter for file.details.RedoSeq.
func (e *ObservableEditableBuffer) RedoSeq() int {
	return e.f.RedoSeq()
}

// inserted is a forwarding function for text.inserted.
func (e *ObservableEditableBuffer) inserted(q0 int, r []rune) {
	for observer := range e.observers {
		observer.inserted(q0, r)
	}
}

// deleted is a forwarding function for text.deleted.
func (e *ObservableEditableBuffer) deleted(q0 int, q1 int) {
	for observer := range e.observers {
		observer.deleted(q0, q1)
	}
}

// Commit is a forwarding function for file.Commit.
func (e *ObservableEditableBuffer) Commit() {
	e.f.Commit()
}

// InsertAtWithoutCommit is a forwarding function for file.InsertAtWithoutCommit.
func (e *ObservableEditableBuffer) InsertAtWithoutCommit(p0 int, s []rune) {
	e.f.InsertAtWithoutCommit(p0, s)
}

// IsDirOrScratch is a forwarding function for file.IsDirOrScratch.
func (e *ObservableEditableBuffer) IsDirOrScratch() bool {
	return e.f.IsDirOrScratch()
}

// TreatAsDirty is a forwarding function for file.TreatAsDirty.
func (e *ObservableEditableBuffer) TreatAsDirty() bool {
	return e.f.TreatAsDirty()
}

// Read is a forwarding function for rune_array.Read.
func (e *ObservableEditableBuffer) Read(q0 int, r []rune) (int, error) {
	return e.f.b.Read(q0, r)
}

// View is a forwarding function for rune_array.View.
func (e *ObservableEditableBuffer) View(q0 int, q1 int) []rune {
	return e.f.b.View(q0, q1)
}

// String is a forwarding function for rune_array.String.
func (e *ObservableEditableBuffer) String() string {
	return e.f.b.String()
}

// ResetBuffer is a forwarding function for rune_array.Reset.
func (e *ObservableEditableBuffer) ResetBuffer() {
	e.f.b.Reset()
}

// Reader is a forwarding function for rune_array.Reader.
func (e *ObservableEditableBuffer) Reader(q0 int, q1 int) io.Reader {
	return e.f.b.Reader(q0, q1)
}

// IndexRune is a forwarding function for rune_array.IndexRune.
func (e *ObservableEditableBuffer) IndexRune(r rune) int {
	return e.f.b.IndexRune(r)
}

// Equal is a forwarding function for rune_array.Equal.
func (e *ObservableEditableBuffer) Equal(s []rune) bool {
	return e.f.b.Equal(s)
}

// Nbyte is a forwarding function for rune_array.Nbyte.
func (e *ObservableEditableBuffer) Nbyte() int {
	return e.f.b.Nbyte()
}

// Setnameandisscratch updates the oeb.details.name and isscratch bit
// at the same time.
func (e *ObservableEditableBuffer) Setnameandisscratch(name string) {
	e.details.Name = name
	if strings.HasSuffix(name, slashguide) || strings.HasSuffix(name, plusErrors) {
		e.f.isscratch = true
	} else {
		e.f.isscratch = false
	}
}
