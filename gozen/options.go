package gozen

import (
	"time"

	"9fans.net/go/acme"
)

// Pike-style options: [command center: Self-referential functions and the design of options](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)
// I have written the simplest possible code that I need _now_. It is conceivable
// that I want something more sophisticated like https://golang.design/research/generic-option/#fn:1

type option func(*acme.Win, bool) error

// Addtotag returns an option for Editinacme that adds the provided string
// to the Acme/Edwood tag.
func Addtotag(v string) option {
	return func(w *acme.Win, wasnew bool) error {
		// capture v in a closure.
		if wasnew {
			return w.Fprintf("tag", v)
		}
		return nil
	}
}

// Blinktag returns an option for Editinacme that blinks the window tag.
// TODO(rjk): Could conceivably configure the blink time?
func Blinktag(_ string) option {
	return func(w *acme.Win, wasnew bool) error {
		stopper := w.Blink()
		waiter := time.NewTimer(5 * time.Second)
		<-waiter.C
		stopper()
		return nil
	}
}
