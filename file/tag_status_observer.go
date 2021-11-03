package file


type TagStatus {
	UndoableChanges bool
	RedoableChanges bool
	SaveableAndDirty bool
}


// TagStatusObserver implementations can register themselves with an
// ObservableEditableBuffer so the observers can be notified of all
// changes to the ObservableEditableBuffer that would prompt changing the
// contents of an Edwood tag.
type TagStatusObserver interface {

	// UndoMemoize is called inside of an Undo when a previously memoized
	// action is undone. This is used to propagate notice of finding a memoized
	// point in the Undo history to the owning observer (typically a Window.)
	UndoMemoize(undo bool)

	// Moar.
	// What are the state changes to accomodate when we actually change the tag.
	// TODO(rjk): Write some comments here.
	UpdateTag(newtagstatus TagStatus)
	
}
