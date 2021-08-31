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

// Package undo provides methods for undoable/redoable text manipulation.
// Modifications are made by two operations: insert or delete.
//
// The package is based on the text manipulation in the vis editor. (Some parts are
// pure ports.) For further information please visit
// 	https://github.com/martanne/vis.
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
package undo

import (
	"errors"
	"io"
	"time"
)

var ErrWrongOffset = errors.New("offset is greater than buffer size")

// A Buffer is a structure capable of two operations: inserting or deleting.
// All operations could be unlimitedly undone or redone.
type Buffer struct {
	piecesCnt   int    // number of pieces allocated
	begin, end  *piece // sentinel nodes which always exists but don't hold any data
	cachedPiece *piece // most recently modified piece

	actions       []*action // stack holding all actions performed to the file
	head          int       // index for the next action to add
	currentAction *action   // action for the current change group
	savedAction   *action
}

// NewBuffer initializes a new buffer with the given content as a starting point.
// To start with an empty buffer pass nil as a content.
func NewBuffer(content []byte) *Buffer {
	// give the actions stack some default capacity
	t := &Buffer{actions: make([]*action, 0, 100)}

	t.begin = t.newEmptyPiece()
	t.end = t.newPiece(nil, t.begin, nil)
	t.begin.next = t.end

	if content != nil {
		p := t.newPiece(content, t.begin, t.end)
		t.begin.next = p
		t.end.prev = p
	}
	return t
}

// Insert inserts the data at the given offset in the buffer. An error is return when the
// given offset is invalid.
func (b *Buffer) Insert(off int64, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	p, offset := b.findPiece(off)
	if p == nil {
		return ErrWrongOffset
	} else if p == b.cachedPiece {
		// just update the last inserted piece
		p.insert(offset, data)
		return nil
	}

	c := b.newChange(off)
	var pnew *piece
	if offset == p.len() {
		// Insert between two existing pieces, hence there is nothing to
		// remove, just add a new piece holding the extra text.
		pnew = b.newPiece(data, p, p.next)
		c.new = newSpan(pnew, pnew)
		c.old = newSpan(nil, nil)
	} else {
		// Insert into middle of an existing piece, therefore split the old
		// piece. That is we have 3 new pieces one containing the content
		// before the insertion point then one holding the newly inserted
		// text and one holding the content after the insertion point.
		before := b.newPiece(p.data[:offset], p.prev, nil)
		pnew = b.newPiece(data, before, nil)
		after := b.newPiece(p.data[offset:], pnew, p.next)
		before.next = pnew
		pnew.next = after
		c.new = newSpan(before, after)
		c.old = newSpan(p, p)
	}

	b.cachedPiece = pnew
	swapSpans(c.old, c.new)
	return nil
}

// Delete deletes the portion of the length at the given offset. An error is returned
// if the portion isn't in the range of the buffer size. If the length exceeds the
// size of the buffer, the portions from off to the end of the buffer will be
// deleted.
func (b *Buffer) Delete(off, length int64) error {
	if length <= 0 {
		return nil
	}

	p, offset := b.findPiece(off)
	if p == nil {
		return ErrWrongOffset
	} else if p == b.cachedPiece && p.delete(offset, length) {
		// try to update the last inserted piece if the length doesn't exceed
		return nil
	}
	b.cachedPiece = nil

	var cur int64 // how much has already been deleted
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
		cur = int64(p.len() - offset)
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
		if p == nil {
		}
		cur += int64(p.len())
	}

	if cur == length {
		// deletion stops at a piece boundary
		end = p
		after = p.next
	} else {
		// deletion stops midway through a piece
		midwayEnd = true
		end = p

		beg := p.len() + int(length-cur)
		newBuf := make([]byte, len(p.data[beg:]))
		copy(newBuf, p.data[beg:])
		after = b.newPiece(newBuf, before, p.next)
	}

	var newStart, newEnd *piece
	if midwayStart {
		// we finally know which piece follows our newly allocated before piece
		newBuf := make([]byte, len(start.data[:offset]))
		copy(newBuf, start.data[:offset])
		before.data = newBuf
		before.prev, before.next = start.prev, after

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
	c := b.newChange(off)
	c.new = newSpan(newStart, newEnd)
	c.old = newSpan(start, end)
	swapSpans(c.old, c.new)

	return nil
}

// newAction creates a new action and throws away all undone actions.
func (b *Buffer) newAction() *action {
	a := &action{time: time.Now()}
	b.actions = append(b.actions[:b.head], a)
	b.head++
	return a
}

// newChange is associated with the current action or a newly allocated one if
// none exists.
func (b *Buffer) newChange(off int64) *change {
	a := b.currentAction
	if a == nil {
		a = b.newAction()
		b.cachedPiece = nil
		b.currentAction = a
	}
	c := &change{off: off}
	a.changes = append(a.changes, c)
	return c
}

func (b *Buffer) newPiece(data []byte, prev, next *piece) *piece {
	b.piecesCnt++
	return &piece{
		id:   b.piecesCnt,
		prev: prev,
		next: next,
		data: data,
	}
}

func (b *Buffer) newEmptyPiece() *piece {
	return b.newPiece(nil, nil, nil)
}

// findPiece returns the piece holding the text at the byte offset. If off happens
// to be at a piece boundary (i.e. the first byte of a piece) then the previous piece
// to the left is returned with an offset of the piece's length.
//
// If off is zero, the beginning sentinel piece is returned.
func (b *Buffer) findPiece(off int64) (p *piece, offset int) {
	var cur int64
	for p = b.begin; p.next != nil; p = p.next {
		if cur <= off && off <= cur+int64(p.len()) {
			return p, int(off - cur)
		}
		cur += int64(p.len())
	}
	return nil, 0
}

// Undo reverts the last performed action. It returns the offset in bytes
// at which the first change of the action occured and the number of bytes
// the change added at off. If there is no action to undo, Undo returns -1
// as the offset.
func (b *Buffer) Undo() (off, n int64) {
	b.Commit()
	a := b.unshiftAction()
	if a == nil {
		return -1, 0
	}

	for i := len(a.changes) - 1; i >= 0; i-- {
		c := a.changes[i]
		swapSpans(c.new, c.old)
		off = c.off
		n = c.old.len - c.new.len
	}
	if n < 0 {
		n = 0
	}
	return
}

func (b *Buffer) unshiftAction() *action {
	if b.head == 0 {
		return nil
	}
	b.head--
	return b.actions[b.head]
}

// Redo repeats the last undone action. It returns the offset in bytes
// at which the last change of the action occured and the number of bytes
// the change added at off. If there is no action to redo, Redo returns -1
// as the offset.
func (b *Buffer) Redo() (off, n int64) {
	b.Commit()
	a := b.shiftAction()
	if a == nil {
		return -1, 0
	}

	for _, c := range a.changes {
		swapSpans(c.old, c.new)
		off = c.off
		n = c.new.len - c.old.len
	}
	if n < 0 {
		n = 0
	}
	return
}

func (b *Buffer) shiftAction() *action {
	if b.head > len(b.actions)-1 {
		return nil
	}
	b.head++
	return b.actions[b.head-1]
}

// Commit commits the currently performed changes and creates an undo/redo point.
func (b *Buffer) Commit() {
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

// Size returns the size of the buffer in the current state. Size is the
// number of bytes available for reading via ReadAt. Operations like Insert,
// Delete, Undo and Redo modify the size.
func (b *Buffer) Size() int64 {
	var size int64
	for p := b.begin; p != nil; p = p.next {
		size += int64(p.len())
	}
	return size
}

// action is a list of changes which are used to undo/redo all modifications.
type action struct {
	changes []*change
	time    time.Time // when the first change of this action was performed
}

// change keeps all needed information to redo/undo an insertion/deletion.
type change struct {
	old span  // all pieces which are being modified/swapped out by the change
	new span  // all pieces which are introduced/swapped int by the change
	off int64 // absolute offset at which the change occured
}

// span holds a certain range of pieces. Changes to the document are allways
// performed by swapping out an existing span with a new one.
type span struct {
	start, end *piece // start/end of the span
	len        int64  // the sum of the lengths of the pieces which form this span
}

func newSpan(start, end *piece) span {
	s := span{start: start, end: end}
	for p := start; p != nil; p = p.next {
		s.len += int64(p.len())
		if p == end {
			break
		}
	}
	return s
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
}

func (p *piece) len() int {
	return len(p.data)
}

func (p *piece) insert(off int, data []byte) {
	p.data = append(p.data[:off], append(data, p.data[off:]...)...)
}

func (p *piece) delete(off int, length int64) bool {
	if int64(off)+length > int64(len(p.data)) {
		return false
	}
	p.data = append(p.data[:off], p.data[off+int(length):]...)
	return true
}
