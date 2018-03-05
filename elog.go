package main

import (
	"fmt"
)

type ElogType byte

const (
	DeleteType ElogType = iota
	InsertType
	FilenameType
	Wsequence = "warning: changes out of sequence\n"
)

/*
 * Elog is a log of changes made by editing commands.  Three reasons for this:
 * 1) We want addresses in commands to apply to old file, not file-in-change.
 * 2) It's difficult to track changes correctly as things move, e.g. ,x m$
 * 3) This gives an opportunity to optimize by merging adjacent changes.
 * It's a little bit like the Undo/Redo log in Files, but Point 3) argues for a
 * separate implementation.  To do this well, we use Replace as well as
 * Insert and Delete
 *
 * There is a significant assumption that the log has increasing q0s.
 * The log is then played back backwards to apply the changes to the text.
 * Out-of-order edits are warned about.
 */
type Elog struct {
	log    []ElogOperation
	warned bool
}

type ElogOperation struct {
	t  ElogType // Delete, Insert, Filename
	q0 uint     // location of change (unused in f)
	nd uint     // number of deleted characters
	r  []rune
}

func MakeElog() Elog {
	return Elog{[]ElogOperation{
		ElogOperation{Null, 0, 0, []rune{}}, // Sentinel
	}, false,
	}
}

func (e *Elog) Reset() {
	// TODO(flux): If working on large documents we may want to actually trim the
	// array here, as it will hog memory after a fine-grained edit.  But don't worry about
	// that until there's a memory issue.
	(*e).log = (*e).log[0:1] // Just the sentinel
	(*e).log[0].t = Null
}

func (e *Elog) Term() {
	(*e).log = (*e).log[0:0]
	(*e).warned = false
}

func (e *ElogOperation) reset() {
	e.t = Null
	e.nd = 0
	e.r = e.r[0:0]
}

func elogclose(f *File) {}

// Make sure buffer is large enough.  This could be simplified, but at the
// cost of significant allocation churn.
func (e *Elog) extend() {
	// Slightly too clever code:  Double the slice if we're out,
	// adding the reservation for the new eo
	if cap((*e).log) == len((*e).log) {
		t := make([]ElogOperation, len((*e).log), (cap((*e).log)+1)*2)
		copy(t, (*e).log)
		(*e).log = t
	}
	(*e).log = (*e).log[:len((*e).log)+1]
}

func (e *Elog) last() *ElogOperation {
	return &((*e).log)[len((*e).log)-1]
}

func (e *Elog) secondlast() *ElogOperation {
	return &((*e).log)[len((*e).log)-2]
}

func (eo *ElogOperation) setr(r []rune) {
	if eo.r == nil || cap(eo.r) < len(r) {
		eo.r = make([]rune, len(r), len(r))
	} else {
		eo.r = eo.r[0:len(r)]
	}
	copy(eo.r, r)
}

func (e *Elog) Replace(q0, q1 uint, r []rune) {
	if q0 == q1 && len(r) == 0 {
		return
	}

	eo := e.last()

	// Check for out-of-order
	if q0 < eo.q0 && !e.warned {
		e.warned = true
		warning(nil, Wsequence)
	}

	// TODO(flux): try to merge with previous

	eo.t = Replace
	eo.q0 = q0
	eo.nd = q1 - q0
	eo.setr(r)
	if eo.q0 < e.secondlast().q0 {
		panic("Changes not in order")
	}
}

func (e *Elog) Insert(q0 uint, r []rune) {
	if len(r) == 0 {
		return
	}

	// This merge only works on the last item; I assume
	// this is because the insertions at the same point tend
	// to come together [logic vaguely lifted from the C implementation]
	eo := e.last()

	// Check for out-of-order
	if (q0 < eo.q0) && !e.warned {
		e.warned = true
		warning(nil, Wsequence)
	}

	if eo.t == Insert && q0 == eo.q0 {
		eo.r = append(eo.r, r...)
		return
	}

	e.extend()

	eo = e.last()
	eo.t = Insert
	eo.q0 = q0
	eo.nd = 0
	eo.setr(r)

	if eo.q0 < e.secondlast().q0 {
		panic("Changes not in order")
	}
}

func (e *Elog) Delete(q0, q1 uint) {
	if q0 == q1 {
		return
	}

	// Try to merge deletes
	eo := e.last()

	// Check for out-of-order
	if (q0 < eo.q0+eo.nd) && !e.warned {
		e.warned = true
		warning(nil, Wsequence)
	}

	if eo.t == Delete && (eo.q0+eo.nd == q0) {
		eo.nd += q1 - q0
		return
	}

	e.extend()

	eo = e.last()
	eo.t = Delete
	eo.q0 = q0
	eo.nd = q1 - q0
	if eo.q0 < e.secondlast().q0 {
		panic("Changes not in order")
	}
}

const tracelog = true

// Apply plays back the log, from back to front onto the given text.
// Unlike the C version, this does not mark the file - that should happen at a higher
// level.
func (e *Elog) Apply(t Texter) {
	/*
		if len((*e).log) > 1 {
			f.Mark() // TODO(flux): I think this is equivalent to checking inside
					// the individual cases (as in the C code), since there's only modifications in the
					// elog.
		} else {
			panic("Really?  Let's try not applying empty logs")
		}
	*/

	// Will this always make a copy, or will the compiler turn
	// the read-only accesses into an in-place read?

	// The log is applied back-to-front - this avoids disturbing the text ahead of the
	// current application point.
	for i := len((*e).log) - 1; i >= 1; i-- {
		eo := (*e).log[i]
		switch eo.t {
		case Replace:
			if tracelog {
				fmt.Printf("elog replace %d %d (%d %d)\n",
					eo.q0, eo.q0+eo.nd, t.Q0(), t.Q1())
			}
			tq0, tq1 := t.Constrain(eo.q0, eo.q0+eo.nd)
			t.Delete(tq0, tq1, true)
			t.Insert(tq0, eo.r, true)
			// Mark selection
			if t.Q0() == eo.q0 && t.Q1() == eo.q0 {
				t.SetQ1(t.Q1() + uint(len(eo.r)))
			}
		case Insert:
			if tracelog {
				fmt.Printf("elog insert %d %d (%d %d)\n",
					eo.q0, eo.q0+uint(len(eo.r)), t.Q0(), t.Q1())
			}
			tq0, _ := t.Constrain(eo.q0, eo.q0)
			t.Insert(tq0, eo.r, true)
			if t.Q0() == eo.q0 && t.Q1() == eo.q0 {
				t.SetQ1(t.Q1() + uint(len(eo.r)))
			}
		case Delete:
			if tracelog {
				fmt.Printf("elog delete %d %d (%d %d)\n",
					eo.q0, eo.q0+uint(len(eo.r)), t.Q0(), t.Q1())
			}
			tq0, tq1 := t.Constrain(eo.q0, eo.q0+eo.nd)
			t.Delete(tq0, tq1, true)
		}
	}
	(*e).Term()
}
