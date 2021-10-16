package file

// This is passed by value. I'm assuming that it's small.
// TODO(rjk): decide if I want to pack these together.
type TagStatus struct {
	UndoableChanges  bool
	RedoableChanges  bool
	SaveableAndDirty bool
}

// TagStatusObserver implementations can register themselves with an
// ObservableEditableBuffer so the observers can be notified of all
// changes to the ObservableEditableBuffer that would prompt changing the
// contents of an Edwood tag.
type TagStatusObserver interface {

	// MemoizedUndone is called inside of an Undo when a previously memoized
	// action is undone. This is used to propagate notice of finding a memoized
	// point in the Undo history to the owning observer (typically a Window.)
	MemoizedUndone(undo bool)

	// UpdateTag is invoked on the implementation by the
	// ObservableEditableBuffer when oeb state has changed in a way that
	// requires altering the pre-bar tag contents.
	UpdateTag(newtagstatus TagStatus)
}
