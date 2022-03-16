// Based on the undo/redo functionality in the vis editor by Marc André Tanner,
// licensed under ISC license which can be found bellow. For further information
// please visit http://repo.or.cz/w/vis.git or https://github.com/martanne/vis.
//
// Copyright (c) 2014 Marc André Tanner <mat at brain-dump.org>
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
//
// And under a license that can be found in the undo.LICENSE file.

// Package undo provides methods for undoable/redoable text manipulation.
// Modifications are made by two operations: insert or delete.
//
// The package is based on the text manipulation in the vis editor. (Some parts are
// pure ports.) For further information please visit
// 	https://github.com/martanne/vis.
// This package was taken from https://github.com/mibk/syd/tree/master/undo for use in edwood.
//
//
// Insertion
//
// When inserting new data there are 2 cases to consider:
//
// 1. the insertion point falls into the middle of an existing piece which
// is replaced by three new pieces:
//
//	/-+ --> +---------------+ --> +-\
//	| |     | existing text |     | |
//	\-+ <-- +---------------+ <-- +-/
//	                   ^
//	                   insertion point for "demo "
//
//	/-+ --> +---------+ --> +-----+ --> +-----+ --> +-\
//	| |     | existing|     |demo |     |text |     | |
//	\-+ <-- +---------+ <-- +-----+ <-- +-----+ <-- +-/
//
// 2. it falls at a piece boundary:
//
//	/-+ --> +---------------+ --> +-\
//	| |     | existing text |     | |
//	\-+ <-- +---------------+ <-- +-/
//	      ^
//	      insertion point for "short"
//
//	/-+ --> +-----+ --> +---------------+ --> +-\
//	| |     |short|     | existing text |     | |
//	\-+ <-- +-----+ <-- +---------------+ <-- +-/
//
//
// Deletion
//
// The delete operation can either start/stop midway through a piece or at
// a boundary. In the former case a new piece is created to represent the
// remaining text before/after the modification point.
//
//	/-+ --> +---------+ --> +-----+ --> +-----+ --> +-\
//	| |     | existing|     |demo |     |text |     | |
//	\-+ <-- +---------+ <-- +-----+ <-- +-----+ <-- +-/
//	             ^                         ^
//	             |------ delete range -----|
//
//	/-+ --> +----+ --> +--+ --> +-\
//	| |     | exi|     |t |     | |
//	\-+ <-- +----+ <-- +--+ <-- +-/
//
//
// Changes
//
// Undoing and redoing works with actions (action is a group of changes: insertions
// and deletions). An action is represented by any operations between two calls of
// Commit method. Anything that happens between these two calls is a part of that
// particular action.
package file

// TODO(rjk): Considerations of the efficiency of file.Buffer must make
// an assumption about the relative sizes of the various pieces.
// Pathological piece size distributions exist. I should validate that
// the code can merge/split pieces over time to keep the piece sizes
// roughly equal.

import (
	"errors"
	"io"
	"log"
	"unicode/utf8"

	"github.com/rjkroege/edwood/sam"
)

var _ io.ReaderAt = (*Buffer)(nil)

// expensiveCheckedExecution turns on a number of expensive validations
// of the internal consistency of the file.Buffer implementation. This is
// true for now while development of file.Buffer continues.
const expensiveCheckedExecution = false

var ErrWrongOffset = errors.New("offset is greater than buffer size")

// A Buffer is a structure capable of two operations: inserting or deleting.
// All operations could be ultimately undone or redone.
type Buffer struct {
	piecesCnt   int    // number of pieces allocated
	begin, end  *piece // sentinel nodes which always exists but don't hold any data
	cachedPiece *piece // most recently modified piece

	actions       []*action // stack holding all actions performed to the file
	head          int       // index for the next action to add
	currentAction *action   // action for the current change group
	savedAction   *action

	oeb *ObservableEditableBuffer

	viewed *piece      // piece most recently accessed by ReadRuneAt
	vws    OffsetTuple // OffsetTuple for start of viewed
	vwl    OffsetTuple // Last determined OffsetTuple
	pend   OffsetTuple // Cached end of the buffer.
}

// NewBuffer initializes a new buffer with the given content as a starting point.
// To start with an empty buffer pass nil as a content.
// TODO(rjk): Should we chunk very large content arrays?
func NewBuffer(content []byte, nr int) *Buffer {
	// give the actions stack some default capacity
	t := &Buffer{actions: make([]*action, 0, 100)}

	t.begin = t.newEmptyPiece()
	t.end = t.newPiece(nil, t.begin, nil, 0)
	t.begin.next = t.end

	if content != nil {
		p := t.newPiece(content, t.begin, t.end, nr)
		t.begin.next = p
		t.end.prev = p
		t.pend = Ot(len(content), nr)
	}
	return t
}

func (b *Buffer) FlattenHistory() {
	b.actions = make([]*action, 0, 100)
	b.head = 0
	b.currentAction = nil
	b.savedAction = nil
}

// Insert inserts the data at the given offset in the buffer. An error is return when the
// given offset is invalid.
func (b *Buffer) Insert(start OffsetTuple, data []byte, nr, seq int) error {
	//	log.Printf("Insert start %v %q %d", start, string(data), nr)
	//	defer log.Println("Insert end")
	off := start.B
	if len(data) == 0 {
		return nil
	}

	if expensiveCheckedExecution {
		if c := utf8.RuneCount(data); c != nr {
			log.Fatalf("newPiece runecount mismatch counted %d, provided %d", c, nr)
		}
	}
	b.validateInvariant()

	//log.Println("before", b.viewedState())

	b.pend = b.pend.Add(len(data), nr)
	p, offset, roffset := b.findPiece(start)
	if p == nil {
		b.validateInvariant()
		return ErrWrongOffset
	} else if p == b.cachedPiece {
		// just update the last inserted piece
		p.insert(offset, data, nr)
		b.validateInvariant()
		return nil
	}

	//log.Println("not in cached state")

	c := b.newChange(off, start.R, seq)
	var pnew *piece

	//TODO(rjk): Increase cases where b.v
	b.viewed = nil
	if offset == p.len() {
		// Insert between two existing pieces, hence there is nothing to
		// remove, just add a new piece holding the extra text.
		pnew = b.newPiece(data, p, p.next, nr)
		c.new = newSpan(pnew, pnew)
		c.old = newSpan(nil, nil)
	} else {
		// Insert into middle of an existing piece, therefore split the old
		// piece. That is we have 3 new pieces one containing the content
		// before the insertion point then one holding the newly inserted
		// text and one holding the content after the insertion point.
		before := b.newPiece(p.data[:offset], p.prev, nil, roffset)
		pnew = b.newPiece(data, before, nil, nr)
		after := b.newPiece(p.data[offset:], pnew, p.next, p.nr-roffset)
		before.next = pnew
		pnew.next = after
		c.new = newSpan(before, after)
		c.old = newSpan(p, p)
	}

	b.cachedPiece = pnew
	swapSpans(c.old, c.new)
	b.validateInvariant()
	//log.Println("after", b.viewedState())
	return nil
}

// Delete deletes the portion of the length at the given offset. An error is returned
// if the portion isn't in the range of the buffer size. If the length exceeds the
// size of the buffer, the portions from off to the end of the buffer will be
// deleted.
func (b *Buffer) Delete(startOff, endOff OffsetTuple, seq int) error {
	b.validateInvariant()
	off := startOff.B
	length := endOff.B - startOff.B
	rlength := endOff.R - startOff.R
	if length <= 0 {
		b.validateInvariant()
		return nil
	}

	b.pend = b.pend.Sub(length, rlength)
	p, offset, roffset := b.findPiece(startOff)
	if p == nil {
		b.validateInvariant()
		return ErrWrongOffset
	} else if p == b.cachedPiece && p.delete(offset, length, int(endOff.R-startOff.R)) {
		// try to update the last inserted piece if the length doesn't exceed
		b.validateInvariant()
		return nil
	}
	b.cachedPiece = nil
	// TODO(rjk): Expand caching opportunities.
	b.viewed = nil

	var cur, rcur int // how much has already been deleted
	midwayStart, midwayEnd := false, false

	var before, after *piece // unmodified pieces before/after deletion point
	var start, end *piece    // span which is removed

	if offset == p.len() {
		// deletion starts at a piece boundary
		before = p
		start = p.next
	} else {
		// deletion starts midway through a piece
		midwayStart = true
		cur = p.len() - offset
		rcur = p.nr - roffset
		start = p
		before = b.newEmptyPiece()
	}

	// skip all pieces which fall into deletion range
	for cur < length {
		if p.next == b.end {
			// delete all
			length = cur
			break
		}
		p = p.next
		cur += p.len()
		rcur += p.nr
	}

	if cur == length {
		// deletion stops at a piece boundary
		end = p
		after = p.next
	} else {
		// deletion stops midway through a piece
		midwayEnd = true
		end = p

		beg := p.len() + length - cur
		newBuf := make([]byte, len(p.data[beg:]))
		copy(newBuf, p.data[beg:])
		after = b.newPiece(newBuf, before, p.next, rcur-rlength)
	}

	var newStart, newEnd *piece
	if midwayStart {
		// we finally know which piece follows our newly allocated before piece
		newBuf := make([]byte, len(start.data[:offset]))
		copy(newBuf, start.data[:offset])
		before.data = newBuf
		before.prev, before.next = start.prev, after
		before.nr = utf8.RuneCount(newBuf)

		newStart = before
		if !midwayEnd {
			newEnd = before
		}
	}
	if midwayEnd {
		newEnd = after
		if !midwayStart {
			newStart = after
		}
	}

	b.cachedPiece = newStart
	c := b.newChange(off, startOff.R, seq)
	c.new = newSpan(newStart, newEnd)
	c.old = newSpan(start, end)
	swapSpans(c.old, c.new)

	b.validateInvariant()
	return nil
}

// newAction creates a new action and throws away all undone actions.
func (b *Buffer) newAction(seq int) *action {
	a := &action{seq: seq}
	b.actions = append(b.actions[:b.head], a)
	b.head++
	return a
}

// newChange is associated with the current action or a newly allocated one if
// none exists.
func (b *Buffer) newChange(off, roff, seq int) *change {
	a := b.currentAction
	if a == nil {
		a = b.newAction(seq)
		b.cachedPiece = nil
		b.currentAction = a
	}
	c := &change{
		off:  off,
		roff: roff,
	}
	a.changes = append(a.changes, c)
	return c
}

// newPiece creates a new piece structure. nr is the number of runes in
// data.
func (b *Buffer) newPiece(data []byte, prev, next *piece, nr int) *piece {
	if expensiveCheckedExecution {
		if c := utf8.RuneCount(data); c != nr {
			log.Fatalf("newPiece runecount mismatch counted %d, provided %d", c, nr)
		}
	}

	b.piecesCnt++
	return &piece{
		id:   b.piecesCnt,
		prev: prev,
		next: next,
		data: data,
		nr:   nr,
	}
}

func (b *Buffer) newEmptyPiece() *piece {
	return b.newPiece(nil, nil, nil, 0)
}

// findPiece returns the piece holding the text at the byte offset, the
// byte offset into piece and similarly for the rune offset. If off
// happens to be at a piece boundary (i.e. the first byte of a piece)
// then the previous piece to the left is returned with an offset of the
// piece's length.
//
// If off is zero, the beginning sentinel piece is returned.
func (b *Buffer) findPiece(off OffsetTuple) (*piece, int, int) {
	p := b.viewed
	if p == nil || iabs(off.B-b.vwl.B) > off.B {
		p = b.begin
		b.vwl = Ot(0, 0)
		b.vws = Ot(0, 0)
	}

	if off.B < b.vwl.B { // Go backwards.
		for ; p != nil; p = p.prev {
			if b.vws.B < off.B && off.B <= b.vws.B+p.len() {
				b.vwl = off
				b.viewed = p
				return p, off.B - b.vws.B, off.R - b.vws.R
			}

			b.vws = Ot(b.vws.B-p.prev.len(), b.vws.R-p.prev.nr)
		}
	} else { // Go forwards.
		for ; p.next != nil; p = p.next {
			if b.vws.B <= off.B && off.B <= b.vws.B+p.len() {
				b.vwl = off
				b.viewed = p
				return p, off.B - b.vws.B, off.R - b.vws.R
			}

			b.vws = Ot(b.vws.B+p.len(), b.vws.R+p.nr)
		}
	}
	b.vwl = Ot(0, 0)
	b.vws = Ot(0, 0)
	return nil, 0, 0
}

// Undo reverts the last performed action. It returns the new selection
// q0, q1 and a bool indicating if the returned selection is meaningful..
// If there is no action to undo, Undo returns -1 as the offset.
// TODO(rjk): nil the cached piece here and in Redo
func (b *Buffer) Undo(_ int) (int, int, bool, int) {
	// log.Println("Undo start")
	// defer log.Println("Undo end")
	b.validateInvariant()
	b.SetUndoPoint()
	a := b.unshiftAction()
	if a == nil {
		return -1, 0, false, 0
	}

	// TODO(rjk): This is wrong if a filename change and edits are part of
	// the same action?
	if a.kind == sam.Filename {
		return b.filenameChangeAction(a)
	}

	var roff, nr int

	for i := len(a.changes) - 1; i >= 0; i-- {
		c := a.changes[i]
		swapSpans(c.new, c.old)
		roff = c.roff

		// Every time we call swapSpans, we've altered which pieces comprise the
		// linked list of pieces. p.viewed may now not be a piece actually in the
		// current piece list. Do this for each swapSpans because the undone
		// callback may invoke code that resets the b.viewed piece.
		b.viewed = nil

		// Must happen after the swapSpans.
		nr = b.undone(c, true)
	}

	if b.head == 0 {
		return roff, roff - nr, true, 0
	}
	b.validateInvariant()
	// TODO(rjk): Conceivably, I need better tests for the return values.
	return roff, roff - nr, true, b.actions[b.head-1].seq
}

// undone is an Undo helper to implement Edwood specific callback
// semantics. It returns the number of affected runes. There are 4 cases:
// undo insertion, undo deletion, redo insertion, redo deletion:
//
// 	- undo-insertion: dispatch a deletion operation, return 0
//	- undo-deletion: dispatch an insert, return size of inserted text
// 	- redo-insertion: dispatch an insert, return size of inserted
// 	- redo-deletion: dispatch a deletion, return 0
//
// TODO(rjk): I want a zero-copy API all the way into frame.
func (b *Buffer) undone(c *change, undo bool) int {
	var size, rsize int
	newnb, newnr := c.new.nbr()
	oldnb, oldnr := c.old.nbr()

	if undo {
		size = newnb - oldnb
		rsize = newnr - oldnr
	} else {
		// Redo case.
		size = oldnb - newnb
		rsize = oldnr - newnr
	}

	// Fix-up the cached end.
	b.pend = Ot(b.pend.B-size, b.pend.R-rsize)

	//	log.Println("undone", undo, size, rsize)
	if b.oeb == nil {
		return rsize
	}

	off := c.off // in bytes. The original location where a change started.
	if size > 0 {
		b.oeb.deleted(Ot(c.off, c.roff), Ot(c.off+size, c.roff+rsize))
		rsize = 0
	} else {
		// size is smaller. So we're undoing a deletion
		buffy := make([]byte, -size)

		// Regarding my comment about speeding up ReadAt with the cached view,
		// the b.viewed is nil at this point. But: might be able to speed this up
		// because I think that the read material is always complete pieces.
		if _, err := b.ReadAt(buffy, int64(off)); err != nil {
			log.Fatalf("fatal error in Buffer.undone reading inserted contents: %v", err)
		}
		b.oeb.inserted(Ot(c.off, c.roff), buffy, -rsize)
	}
	return rsize
}

func (b *Buffer) unshiftAction() *action {
	if b.head == 0 {
		return nil
	}
	b.head--
	return b.actions[b.head]
}

func (b *Buffer) filenameChangeAction(a *action) (int, int, bool, int) {
	b.oeb.setfilename(a.fname)
	if b.head > 0 {
		return -1, 0, false, b.actions[b.head-1].seq
	}
	return -1, 0, false, 0
}

// Redo repeats the last undone action. It returns new selection q0, q1
// and a bool indicating if the returned selection is meaningful.. If
// there is no action to redo, Redo returns -1 as the offset.
func (b *Buffer) Redo(_ int) (int, int, bool, int) {
	//	log.Println("Redo start")
	//	defer log.Println("Redo end")
	b.validateInvariant()
	b.SetUndoPoint()
	a := b.shiftAction()
	if a == nil {
		return -1, 0, false, 0
	}

	if a.kind == sam.Filename {
		return b.filenameChangeAction(a)
	}

	var roff, nr int
	for _, c := range a.changes {
		swapSpans(c.old, c.new)
		roff = c.roff

		// Reset the cached piece.
		b.viewed = nil

		// Must happen after swapSpans
		nr = b.undone(c, false)
	}

	//	log.Println("redo", roff, roff+nr, true, len(a.changes))
	b.validateInvariant()
	return roff, roff - nr, true, b.actions[b.head-1].seq
}

// RedoSeq finds the seq of the last redo record.The value of seq is used
// to track intra and inter File edit actions so that cross-File changes
// via Edit X can be undone with a single action.
func (b *Buffer) RedoSeq() int {
	if len(b.actions) > 0 && b.head < len(b.actions) {
		return b.actions[b.head].seq
	}
	return 0
}

func (b *Buffer) shiftAction() *action {
	if b.head > len(b.actions)-1 {
		return nil
	}
	b.head++
	return b.actions[b.head-1]
}

// SetUndoPoint commits the currently performed changes and creates an undo/redo point.
func (b *Buffer) SetUndoPoint() {
	b.currentAction = nil
	b.cachedPiece = nil
}

// Clean marks the buffer as non-dirty.
func (b *Buffer) Clean() {
	if b.head > 0 {
		b.savedAction = b.actions[b.head-1]
	} else {
		b.savedAction = nil
	}
}

// Dirty reports whether the current state of the buffer is different from the
// initial state or from the one in the time of calling Clean.
func (b *Buffer) Dirty() bool {
	return b.head == 0 && b.savedAction != nil ||
		b.head > 0 && b.savedAction != b.actions[b.head-1]
}

// TODO(rjk): It's possible to speed this up with the cached view.
func (b *Buffer) ReadAt(data []byte, off int64) (n int, err error) {
	p := b.begin
	for ; p != nil; p = p.next {
		if off < int64(p.len()) {
			break
		}
		off -= int64(p.len())
	}
	if p == nil {
		if off == 0 {
			return 0, io.EOF
		}
		return 0, ErrWrongOffset
	}

	for n < len(data) && p != nil {
		n += copy(data[n:], p.data[off:])
		p = p.next
		off = 0
	}
	if n < len(data) {
		return n, io.EOF
	}
	return n, nil
}

// End returns an OffsetTuple for the end of the buffer.
func (b *Buffer) End() OffsetTuple {
	if b.pend.B >= 0 && b.pend.R >= 0 {
		return b.pend
	}

	tb, tr := 0, 0
	for p := b.begin; p != b.end; p = p.next {
		tb += p.len()
		tr += p.nr
	}

	b.pend = Ot(tb, tr)
	return b.pend
}

// Size returns the size of the buffer in the current state. Size is the
// number of bytes available for reading via ReadAt. Operations like Insert,
// Delete, Undo and Redo modify the size.
func (b *Buffer) Size() int {
	return b.End().B
}

// Nr returns the sum of the Nr for each piece in the buffer.
func (b *Buffer) Nr() int {
	return b.End().R
}

// UnsetName records a filename change at seq to fname.
func (b *Buffer) UnsetName(fname string, seq int) {
	a := &action{
		seq:   seq,
		kind:  sam.Filename,
		fname: fname,
	}
	b.actions = append(b.actions[:b.head], a)
	b.head++

	b.cachedPiece = nil
	b.currentAction = nil
}

// validateInvariant tests that every piece has a correct rune count
// given its byte length.
func (b *Buffer) validateInvariant() {
	if expensiveCheckedExecution {
		for p := b.begin; p != b.end; p = p.next {
			if p.nr != utf8.RuneCount(p.data) {
				log.Printf("invariant violated in piece %#v", *p)
				panic("file.Buffer piece invariant violated")
			}
		}
	}
}

// action is a list of changes which are used to undo/redo all modifications.
type action struct {
	changes []*change
	seq     int

	kind  int
	fname string
}

// change keeps all needed information to redo/undo an insertion/deletion.
type change struct {
	old  span // all pieces which are being modified/swapped out by the change
	new  span // all pieces which are introduced/swapped in by the change
	off  int  // absolute offset at which the change occurred
	roff int  // absolute offset in runes at which the change occurred.

	// TODO(rjk): These don't get updated.
	// It's length of new - length of old?
	//	nb   int  // number of bytes in this change, negative on deletion.
	//	nr   int  // number of runes in this change, negative on deletion.
}

// span holds a certain range of pieces. Changes to the document are
// always performed by swapping out an existing span with a new one. len
// is required to control swapSpans operation. len might not be updated
// when a cached piece is modified.
type span struct {
	start, end *piece // start/end of the span
	len        int    // the sum of the lengths of the pieces which form this span.
	// TODO(rjk): Tracking the number of runes in the span permits preseving
	// p.viewed across swapSpans operations.
}

func newSpan(start, end *piece) span {
	s := span{start: start, end: end}
	for p := start; p != nil; p = p.next {
		s.len += p.len()
		if p == end {
			break
		}
	}
	return s
}

// nbr returns the number of bytes and runes in a span. span.len is not
// updated for cached piece modification. nbr returns up to date values.
func (s span) nbr() (int, int) {
	nr, nb := 0, 0
	for p := s.start; p != nil; p = p.next {
		nb += p.len()
		nr += p.nr
		if p == s.end {
			break
		}
	}
	return nb, nr
}

// swapSpans swaps out an old span and replace it with a new one.
//  - If old is an empty span do not remove anything, just insert the new one.
//  - If new is an empty span do not insert anything, just remove the old one.
func swapSpans(old, new span) {
	if old.len == 0 && new.len == 0 {
		return
	} else if old.len == 0 {
		// insert new span
		new.start.prev.next = new.start
		new.end.next.prev = new.end
	} else if new.len == 0 {
		// delete old span
		old.start.prev.next = old.end.next
		old.end.next.prev = old.start.prev
	} else {
		// replace old with new
		old.start.prev.next = new.start
		old.end.next.prev = new.end
	}
}

// piece represents a piece of the text. All active pieces chained together form
// the whole content of the text.
type piece struct {
	id         int
	prev, next *piece
	data       []byte
	nr         int
}

func (p *piece) len() int {
	return len(p.data)
}

func (p *piece) insert(off int, data []byte, nr int) {
	p.data = append(p.data[:off], append(data, p.data[off:]...)...)
	p.nr += nr
}

func (p *piece) delete(off int, length int, nr int) bool {
	if off+length > len(p.data) {
		return false
	}
	p.data = append(p.data[:off], p.data[off+int(length):]...)
	p.nr -= nr
	return true
}

// Bytes returns the byte representation of the internal buffer.
func (b *Buffer) Bytes() []byte {
	byteBuf := make([]byte, 0, b.Size())
	for p := b.begin; p != b.end; p = p.next {
		byteBuf = append(byteBuf, p.data...)
	}
	return byteBuf
}

// HasUncommitedChanges returns true if there are changes that
// have been made to the File since the last Commit.
// TODO(rjk): This concept is not needed in a file.Buffer world. Improve
// this.
func (b *Buffer) HasUncommitedChanges() bool {
	return false
}

// HasUndoableChanges returns true if there are changes to the File
// that can be undone.
func (b *Buffer) HasUndoableChanges() bool {
	return b.head != 0
}

// HasRedoableChanges returns true if there are entries in the Redo
// log that can be redone.
func (b *Buffer) HasRedoableChanges() bool {
	return b.head <= len(b.actions)-1
}

func iabs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// ReadRuneAt returns the rune at off.
func (b *Buffer) ReadRuneAt(off OffsetTuple) (rune, int, error) {
	p, tb, _ := b.findPiece(off)

	if p == nil {
		if off.B == 0 {
			return 0, 0, io.EOF
		}
		return 0, 0, ErrWrongOffset
	}

	// Remember that findPiece is finding p that is where next text at off
	// would be appended so must bump forward.
	if tb == p.len() {
		p = p.next
		tb = 0
	}

	r, sz := utf8.DecodeRune(p.data[tb:])
	return r, sz, nil
}

// RuneTuple creates a byte, rune offset pair (i.e. OffsetTuple) for a
// given offset in runes.
//
// In setting b.viewed, RuneTuple follows the semantic that aligns with
// findPiece: if off is on a piece boundary, b.viewed will be set to piece
// before the insertion point.
func (b *Buffer) RuneTuple(off int) OffsetTuple {
	p := b.viewed

	// Start at the beginning if that's cheaper or fallback to regular
	// RuneTuple operation.
	if p == nil || iabs(off-b.vwl.R) > off {
		p = b.begin
		b.vwl = Ot(0, 0)
		b.vws = Ot(0, 0)
	}

	if off < b.vwl.R { // Go backwards.
		for p != b.begin && off <= b.vws.R {
			p = p.prev
			b.vwl = Ot(b.vws.B, b.vws.R)
			b.vws = Ot(b.vws.B-p.len(), b.vws.R-p.nr)
		}

		// Find the byte offset in piece p
		for i := b.vwl.B - b.vws.B; i >= 0 && b.vwl.R > off; {
			_, sz := utf8.DecodeLastRune(p.data[0:i])
			b.vwl = Ot(b.vwl.B-sz, b.vwl.R-1)
			i -= sz
		}
	} else { // Go forwards.
		for ; p != b.end && off > b.vws.R+p.nr; p = p.next {
			b.vws = Ot(b.vws.B+len(p.data), b.vws.R+p.nr)
			b.vwl = b.vws
		}

		// Find the byte offset in piece p
		for i := b.vwl.B - b.vws.B; b.vwl.R < off; {
			_, sz := utf8.DecodeRune(p.data[i:])
			b.vwl = Ot(b.vwl.B+sz, b.vwl.R+1)
			i += sz
		}
	}
	if p == nil {
		panic("not expected!")
	}
	b.viewed = p
	return b.vwl
}

// ByteTuple creates a byte, rune offset pair (i.e. OffsetTuple) for a
// given offset in bytes.
//
// In setting b.viewed, ByteTuple follows the semantic that aligns with
// findPiece: if off is on a piece boundary, b.viewed will be set to piece
// before the insertion point.
//
// TODO(rjk): This code is very similar to RuneTuple. But different.
func (b *Buffer) ByteTuple(off int) OffsetTuple {
	p := b.viewed

	// Start at the beginning if that's cheaper or fallback to regular
	// RuneTuple operation.
	if p == nil || iabs(off-b.vwl.B) > off {
		p = b.begin
		b.vwl = Ot(0, 0)
		b.vws = Ot(0, 0)
	}

	if off < b.vwl.B { // Go backwards.
		for p != b.begin && off <= b.vws.B {
			p = p.prev
			b.vwl = b.vws
			b.vws = Ot(b.vws.B-p.len(), b.vws.R-p.nr)
		}

		// Find the byte offset in piece p
		for i := b.vwl.B - b.vws.B; i >= 0 && b.vwl.B > off; {
			_, sz := utf8.DecodeLastRune(p.data[0:i])
			b.vwl = Ot(b.vwl.B-sz, b.vwl.R-1)
			i -= sz
		}
	} else { // Go forwards.
		for ; p != b.end && off > b.vws.B+p.len(); p = p.next {
			b.vws = Ot(b.vws.B+p.len(), b.vws.R+p.nr)
			b.vwl = b.vws
		}

		// Find the byte offset in piece p
		for i := b.vwl.B - b.vws.B; b.vwl.B < off; {
			_, sz := utf8.DecodeRune(p.data[i:])
			b.vwl = Ot(b.vwl.B+sz, b.vwl.R+1)
			i += sz
		}
	}
	if p == nil {
		panic("not expected!")
	}
	b.viewed = p
	return b.vwl
}
