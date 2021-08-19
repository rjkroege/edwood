package sam

import (
	"github.com/rjkroege/edwood/util"
)

// Texter abstracts the buffering side of Text, allowing testing of Elog Apply
// TODO(flux): This is probably lame and will get re-done when I understand
// how Text stores its text.
type Texter interface {
	Constrain(q0, q1 int) (p0, p1 int)
	Delete(q0, q1 int, tofile bool)
	Insert(q0 int, r []rune, tofile bool)
	Q0() int // Selection start
	SetQ0(int)
	Q1() int // End of selelection
	SetQ1(int)
	Nc() int
	ReadB(q int, r []rune) (n int, err error)
	ReadC(q int) rune
	View(q0, q1 int) []rune // Return a "read only" slice
}

// TextBuffer implements Texter around a buffer.
type TextBuffer struct {
	q0, q1 int
	buf    []rune
}

// NewTextBuffer is a constructor for texter.TextBuffer.
func NewTextBuffer(q0 int, q1 int, buf []rune) *TextBuffer {
	return &TextBuffer{q0: q0, q1: q1, buf: buf}
}

func (t TextBuffer) Constrain(q0, q1 int) (p0, p1 int) {
	p0 = util.Min(q0, len(t.buf))
	p1 = util.Min(q1, len(t.buf))
	return p0, p1
}

func (t *TextBuffer) View(q0, q1 int) []rune {
	if q1 > len(t.buf) {
		q1 = len(t.buf)
	}
	return t.buf[q0:q1]
}

func (t *TextBuffer) Delete(q0, q1 int, tofile bool) {
	_ = tofile
	if q0 > len(t.buf) || q1 > len(t.buf) {
		panic("Out-of-range Delete")
	}
	copy(t.buf[q0:], t.buf[q1:])
	t.buf = t.buf[:len(t.buf)-(q1-q0)] // Reslice to length
}

func (t *TextBuffer) Insert(q0 int, r []rune, tofile bool) {
	_ = tofile
	if q0 > len(t.buf) {
		panic("Out of range insertion")
	}
	t.buf = append(t.buf[:q0], append(r, t.buf[q0:]...)...)
}

func (t *TextBuffer) ReadB(q int, r []rune) (n int, err error) {
	n = len(r)
	err = nil
	copy(r, t.buf[q:q+n])
	return
}
func (t *TextBuffer) ReadC(q int) rune { return t.buf[q] }
func (t *TextBuffer) Q0() int          { return t.q0 }
func (t *TextBuffer) SetQ0(q0 int)     { t.q0 = q0 }
func (t *TextBuffer) Q1() int          { return t.q1 }
func (t *TextBuffer) SetQ1(q1 int)     { t.q1 = q1 }
func (t *TextBuffer) Nc() int          { return len(t.buf) }
