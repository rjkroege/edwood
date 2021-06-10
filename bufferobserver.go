package main

// BufferObserver implementations can register themselves
// with an ObservableEditableBuffer so the observers can be
// notified of all buffer mutations made.
type BufferObserver interface {

	// inserted informs the implementer that rune array r was inserted at position q0.
	inserted(q0 int, r []rune)

	// deleted informs the implementer that character range [q0,q1) was deleted.
	deleted(q0, q1 int)
}
