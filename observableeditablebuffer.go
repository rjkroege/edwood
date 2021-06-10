package main

import "fmt"

// ObservableEditableBuffer has a file that and is a
// type through which the main program will add, remove and check
// on the current observer(s) for a Text
type ObservableEditableBuffer struct {
	currobserver BufferObserver
	observers    map[BufferObserver]struct{} // [private I think]
	f            *File
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

// SetCurObserver sets the current observer
func (e *ObservableEditableBuffer) SetCurObserver(observer BufferObserver) {
	e.currobserver = observer
}

// GetCurObserver gets the current observer and returns a interface{}
func (e *ObservableEditableBuffer) GetCurObserver() interface{} {
	return e.currobserver
}

// AllObservers preforms tf(all observers...)
func (e *ObservableEditableBuffer) AllObservers(tf func(i interface{})) {
	for t := range e.observers {
		tf(t)
	}
}

// GetObserverSize will return the size of the observer map
func (e *ObservableEditableBuffer) GetObserverSize() int {
	return len(e.observers)
}

// HasMultipleObservers returns true if their are multiple observers to the File
func (e *ObservableEditableBuffer) HasMultipleObservers() bool {
	return len(e.observers) > 1
}

// insertOnAll inserts at q0 for all observers in the observer map
func (e *ObservableEditableBuffer) insertOnAll(q0 int, r []rune) {
	e.AllObservers(func(i interface{}) {
		i.(BufferObserver).inserted(q0, r)
	})
}

// deleteOnAll deletes q0 to q1 on all of the observer in the observer map
func (e *ObservableEditableBuffer) deleteOnAll(q0 int, q1 int) {
	e.AllObservers(func(i interface{}) {
		i.(BufferObserver).deleted(q0, q1)
	})
}

// MakeObservableEditableBuffer is a constructor wrapper for NewFile() to abstract File from the main program
func MakeObservableEditableBuffer(filename string, b RuneArray) *ObservableEditableBuffer {
	f := NewFile(filename)
	f.b = b
	return &ObservableEditableBuffer{
		currobserver: nil,
		observers:    nil,
		f:            f,
	}
}

// MakeObservableEditableBufferTag is a constructor wrapper for NewTagFile() to abstract File from the main program
func MakeObservableEditableBufferTag(b RuneArray) *ObservableEditableBuffer {
	f := NewTagFile()
	f.b = b
	return &ObservableEditableBuffer{
		currobserver: nil,
		observers:    nil,
		f:            f,
	}
}
