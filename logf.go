package main

import (
	"fmt"
	"sync"

	"9fans.net/go/plan9"
)

var eventlog Log

// State for global log file.
type Log struct {
	lk sync.Mutex
	r  sync.Cond

	start int // msg[0] corresponds to 'start' in the global sequence of eventsevents

	// queued events (nev=entries in ev, mev=capacity of p)
	ev  []string
	mev int // cap(ev) //TODO(flux) used by the compaction logic

	// open acme/put files that need to read events
	f []*Fid

	// active (blocked) reads waiting for events
	read []*Xfid
}

func xfidlogopen(x *Xfid) {
	eventlog.lk.Lock()
	defer eventlog.lk.Unlock()
	eventlog.f = append(eventlog.f, x.f)
	x.f.logoff = eventlog.start + len(eventlog.ev)
}

func xfidlogclose(x *Xfid) {
	eventlog.lk.Lock()
	defer eventlog.lk.Unlock()
	for i := 0; i < len(eventlog.f); i++ {
		if eventlog.f[i] == x.f {
			eventlog.f[i] = eventlog.f[len(eventlog.f)-1]
			eventlog.f = eventlog.f[:len(eventlog.f)-1]
			return
		}
	}

}

func xfidlogread(x *Xfid) {
	eventlog.lk.Lock()
	defer eventlog.lk.Unlock()

	eventlog.read = append(eventlog.read, x)

	if eventlog.r.L == nil {
		eventlog.r.L = &eventlog.lk
	}
	x.flushed = false
	for x.f.logoff >= eventlog.start+len(eventlog.ev) && !x.flushed {
		eventlog.r.Wait() // TODO(flux) Did I get the Rendez right?
	}

	for i := 0; i < len(eventlog.read); i++ {
		if eventlog.read[i] == x {
			eventlog.read[i] = eventlog.read[len(eventlog.read)-1]
			eventlog.read = eventlog.read[:len(eventlog.read)-1]
			break
		}
	}

	if x.flushed {
		return
	}

	i := x.f.logoff - eventlog.start
	p := eventlog.ev[i]
	x.f.logoff++

	fc := plan9.Fcall{}
	fc.Data = []byte(p)
	fc.Count = uint32(len(p))
	x.respond(&fc, nil)
}

func xfidlogflush(x *Xfid) {
	eventlog.lk.Lock()
	defer eventlog.lk.Unlock()
	for i := 0; i < len(eventlog.read); i++ {
		rx := eventlog.read[i]
		if rx.fcall.Tag == x.fcall.Oldtag {
			rx.flushed = true
			eventlog.r.Broadcast()
		}
	}
}

// add a log entry for op on w.
// expected calls:
//
// op == "new" for each new window
// - caller of coladd or makenewwindow responsible for calling
// 	xfidlog after setting window name
// - exception: zerox
//
// op == "zerox" for new window created via zerox
// - called from zeroxx
//
// op == "get" for Get executed on window
// - called from get
//
// op == "put" for Put executed on window
// - called from put
//
// op == "del" for deleted window
// - called from winclose
func xfidlog(w *Window, op string) {
	eventlog.lk.Lock()
	defer eventlog.lk.Unlock()
	if len(eventlog.ev) >= cap(eventlog.ev) {
		// Remove and free any entries that all readers have read.
		min := eventlog.start + len(eventlog.ev)
		for i := 0; i < len(eventlog.f); i++ {
			if min > eventlog.f[i].logoff {
				min = eventlog.f[i].logoff
			}
		}
		if min > eventlog.start {
			n := min - eventlog.start
			eventlog.start += n
			copy(eventlog.ev, eventlog.ev[n:])
			eventlog.ev = eventlog.ev[:len(eventlog.ev)-n] // TODO(flux) fussy, might have messed this up
		}
	}
	name := w.body.file.Name()
	eventlog.ev = append(eventlog.ev, fmt.Sprintf("%d %s %s\n", w.id, op, name))
	if eventlog.r.L == nil {
		eventlog.r.L = &eventlog.lk
	}
	eventlog.r.Broadcast()
}
