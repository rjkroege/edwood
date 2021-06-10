package main

// BufferObserver separates observer functionality out of file.
// A BufferObserver is something that can be kept track
// of through the ObservableEditableBuffer.

type BufferObserver interface {
	// inserted is a callback function which updates the observer's texts.
	inserted(q0 int, r []rune)
	// deleted is a callback function which deletes the observer's texts.
	deleted(q0, q1 int)
}
