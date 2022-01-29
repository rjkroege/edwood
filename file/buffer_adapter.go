package file

import (
	"flag"
	"io"
)

// BufferAdapter is a (temporary) interface between
// ObservableEditableBuffer and either the legacy file.File or
// file.Buffer implementations.
type BufferAdapter interface {
	// Mark sets an Undo point and and discards Redo records. Call this at
	// the beginning of a set of edits that ought to be undo-able as a unit.
	// This is equivalent to file.Buffer.Commit() NB: current implementation
	// permits calling Mark on an empty file to indicate that one can undo to
	// the file state at the time of calling Mark.
	Mark()

	// HasUncommitedChanges returns true if there are changes that
	// have been made to the File since the last Commit.
	HasUncommitedChanges() bool

	// HasRedoableChanges returns true if there are entries in the Redo
	// log that can be redone.
	HasRedoableChanges() bool
	// HasUndoableChanges returns true if there are changes to the File
	// that can be undone.
	HasUndoableChanges() bool

	// Nr returns the number of valid runes in the File.
	Nr() int

	// ReadC reads a single rune from the File.
	ReadC(q int) rune

	// InsertAt inserts s runes at rune address p0.
	InsertAt(p0 int, s []rune, seq int)

	UnsetName(seq int)

	// Undo undoes edits if isundo is true or redoes edits if isundo is false.
	// It returns the new selection q0, q1 and a bool indicating if the
	// returned selection is meaningful.
	// TODO(rjk): do we use the returned values?
	Undo(isundo bool, seq int) (int, int, bool, int)

	// DeleteAt removes the rune range [p0,p1) from File.
	DeleteAt(p0, p1, seq int)

	// RedoSeq finds the seq of the last redo record.
	RedoSeq() int

	// Commit writes the in-progress edits to the real buffer instead of
	// keeping them in the cache.
	Commit(seq int)

	Read(q0 int, r []rune) (int, error)
	String() string
	Reader(q0 int, q1 int) io.Reader
	IndexRune(r rune) int

	// InsertAtWithoutCommit inserts s at p0 by only writing to the cache.
	InsertAtWithoutCommit(p0 int, s []rune)
}

// Enforce that *file.File implements BufferAdapter.
var (
	_ BufferAdapter = (*File)(nil)

	// TODO(rjk): Make this compile. :-)
	// _ BufferAdapter = (*Buffer)(nil)

	newTypeBuffer bool
)

func init() {
	flag.BoolVar(&newTypeBuffer, "newtypebuffer", false, "turn on the file.Buffer new Buffer implementation")
}

func NewTypeBuffer(r []rune, oeb *ObservableEditableBuffer) BufferAdapter {
	// TODO(rjk): Write this.
	return BufferAdapter(nil)
}
