package file

// BufferObserver implementations can register themselves
// with an ObservableEditableBuffer so the observers can be
// notified of all buffer mutations made.
type BufferObserver interface {

	// inserted informs the implementer that byte array b was inserted at position q0.
	Inserted(q0 OffsetTuple, b []byte, nr int)

	// deleted informs the implementer that character range [q0,q1) was deleted.
	Deleted(q0, q1 OffsetTuple)
}
