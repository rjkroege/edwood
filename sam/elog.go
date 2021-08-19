package sam

import (
	"errors"
	"fmt"
)

type ElogType byte

const (
	DeleteType ElogType = iota
	InsertType
	FilenameType
	Wsequence     = "warning: changes out of sequence"
	WsequenceDire = "warning: changes out of sequence, edit result probably wrong"
	Delete        = 'd'
	Insert        = 'i'
	Filename      = 'f'
	Null          = '-'
	Replace       = 'r'
)

// Elog is a log of changes made by editing commands.  Three reasons for this:
// 1) We want addresses in commands to apply to old file, not file-in-change.
// 2) It's difficult to track changes correctly as things move, e.g. ,x m$
// 3) This gives an opportunity to optimize by merging adjacent changes.
// It's a little bit like the Undo/Redo log in Files, but Point 3) argues for a
// separate implementation.  To do this well, we use Replace as well as
// Insert and Delete
//
// There is a significant assumption that the log has increasing q0s.
// The log is then played back backwards to apply the changes to the text.
// Out-of-order edits are warned about.
type Elog struct {
	Log    []ElogOperation
	warned bool
}

type ElogOperation struct {
	T  ElogType // Delete, Insert, Filename
	q0 int      // location of change (unused in f)
	nd int      // number of deleted characters
	r  []rune
}

func MakeElog() Elog {
	return Elog{[]ElogOperation{
		{Null, 0, 0, []rune{}}, // Sentinel
	}, false,
	}
}

func (e *Elog) Reset() {
	// TODO(flux): If working on large documents we may want to actually trim the
	// array here, as it will hog memory after a fine-grained edit.  But don't worry about
	// that until there's a memory issue.
	(*e).Log = (*e).Log[0:1] // Just the sentinel
	(*e).Log[0].T = Null
}

func (e *Elog) Term() {
	(*e).Reset()
	(*e).warned = false
}

func (eo *ElogOperation) reset() {
	eo.T = Null
	eo.nd = 0
	eo.r = eo.r[0:0]
}

// Make sure buffer is large enough.  This could be simplified, but at the
// cost of significant allocation churn.
func (e *Elog) extend() {
	// Slightly too clever code:  Double the slice if we're out,
	// adding the reservation for the new eo
	if cap((*e).Log) == len((*e).Log) {
		t := make([]ElogOperation, len((*e).Log), (cap((*e).Log)+1)*2)
		copy(t, (*e).Log)
		(*e).Log = t
	}
	(*e).Log = (*e).Log[:len((*e).Log)+1]
}

func (e *Elog) last() *ElogOperation {
	return &((*e).Log)[len((*e).Log)-1]
}

func (e *Elog) secondlast() *ElogOperation {
	return &((*e).Log)[len((*e).Log)-2]
}

func (eo *ElogOperation) setr(r []rune) {
	if eo.r == nil || cap(eo.r) < len(r) {
		eo.r = make([]rune, len(r))
	} else {
		eo.r = eo.r[0:len(r)]
	}
	copy(eo.r, r)
}

func (e *Elog) Replace(q0, q1 int, r []rune) error {
	var err error = nil
	if q0 == q1 && len(r) == 0 {
		return err
	}

	eo := e.last()

	// Check for out-of-order
	if q0 < eo.q0 && !e.warned {
		e.warned = true
		err = errors.New(Wsequence)
	}

	// TODO(flux): try to merge with previous

	e.extend()
	eo = e.last()
	eo.T = Replace
	eo.q0 = q0
	eo.nd = q1 - q0
	eo.setr(r)
	if eo.q0 < e.secondlast().q0 {
		e.warned = true
		if err != nil {
			err = errors.New(err.Error() + "\n" + Wsequence)
		} else {
			err = errors.New(Wsequence)
		}
	}
	return err
}

func (e *Elog) Insert(q0 int, r []rune) error {
	var err error = nil
	if len(r) == 0 {
		return err
	}

	// This merge only works on the last item; I assume
	// this is because the insertions at the same point tend
	// to come together [logic vaguely lifted from the C implementation]
	eo := e.last()

	// Check for out-of-order
	if (q0 < eo.q0) && !e.warned {
		e.warned = true
		err = errors.New(Wsequence)
	}

	if eo.T == Insert && q0 == eo.q0 {
		eo.r = append(eo.r, r...)
		return err
	}

	e.extend()

	eo = e.last()
	eo.T = Insert
	eo.q0 = q0
	eo.nd = 0
	eo.setr(r)

	if eo.q0 < e.secondlast().q0 {
		e.warned = true
		if err != nil {
			err = errors.New(err.Error() + "\n" + WsequenceDire)
		} else {
			err = errors.New(WsequenceDire)
		}
	}
	return err
}

func (e *Elog) Delete(q0, q1 int) error {
	var err error = nil
	if q0 == q1 {
		return err
	}

	// Try to merge deletes
	eo := e.last()

	// Check for out-of-order
	if (q0 < eo.q0+eo.nd) && !e.warned {
		e.warned = true
		err = errors.New(Wsequence)
	}

	if eo.T == Delete && (eo.q0+eo.nd == q0) {
		eo.nd += q1 - q0
		return err
	}

	e.extend()

	eo = e.last()
	eo.T = Delete
	eo.q0 = q0
	eo.nd = q1 - q0
	if eo.q0 < e.secondlast().q0 {
		e.warned = true
		if err != nil {
			err = errors.New(err.Error() + "\n" + WsequenceDire)
		} else {
			err = errors.New(WsequenceDire)
		}
	}
	return err
}

const tracelog = false

func (e *Elog) Empty() bool {
	return len(e.Log) == 1
}

// Apply plays back the log, from back to front onto the given text.
// Unlike the C version, this does not mark the file - that should happen at a higher
// level.
func (e *Elog) Apply(t Texter) {
	// The log is applied back-to-front - this avoids disturbing the text ahead of the
	// current application point.
	for i := len((*e).Log) - 1; i >= 1; i-- {
		eo := (*e).Log[i]
		switch eo.T {
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
				t.SetQ1(t.Q1() + len(eo.r))
			}
		case Insert:
			if tracelog {
				fmt.Printf("elog insert %d %d (%d %d)\n",
					eo.q0, eo.q0+len(eo.r), t.Q0(), t.Q1())
			}
			tq0, _ := t.Constrain(eo.q0, eo.q0)
			t.Insert(tq0, eo.r, true)
			if t.Q0() == eo.q0 && t.Q1() == eo.q0 {
				t.SetQ1(t.Q1() + len(eo.r))
			}
		case Delete:
			if tracelog {
				fmt.Printf("elog delete %d %d (%d %d)\n",
					eo.q0, eo.q0+len(eo.r), t.Q0(), t.Q1())
			}
			tq0, tq1 := t.Constrain(eo.q0, eo.q0+eo.nd)
			t.Delete(tq0, tq1, true)
		}
	}
	(*e).Term()
}
