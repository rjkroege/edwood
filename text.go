package main

import (
	"image"
	"math"

	"9fans.net/go/draw"
)

const (
	Ldot   = "."
	TABDIR = 3
)

var (
	left1  = []rune{'{', '[', '(', '<', 0xab}
	right1 = []rune{'}', ']', ')', '>', 0xbb}
	left2  = []rune{'\n'}
	left3  = []rune{'\'', '"', '`'}

	left = [][]rune{
		left1,
		left2,
		left3,
	}

	right = [][]rune{
		right1,
		left2,
		left3,
	}
)

type Text struct {
	file *File
	//fr Frame
	reffont *Reffont
	org     uint
	q0      uint
	q1      uint
	what    int
	tabstop int
	w       *Window
	scrollr image.Rectangle
	lastsr  image.Rectangle
	all     image.Rectangle
	row     *Row
	col     *Column

	iq1         uint
	eq0         uint
	cq0         uint
	ncache      int
	ncachealloc int
	cache       []rune
	nofill      int
	needundo    int
}

func NewText(f *File, r image.Rectangle, rf *Reffont, cols []*draw.Image) *Text {
	t := new(Text)
	t.file = f
	t.all = r
	t.scrollr = r
	t.scrollr.Max.X = r.Min.X + display.ScaleSize(12)
	t.lastsr = nullrect
	r.Min.X += display.ScaleSize(12) + display.ScaleSize(4)
	t.eq0 = math.MaxUint64
	t.ncache = 0
	t.reffont = rf
	t.tabstop = int(maxtab)
	//copy(t.fr, cols)
	t.Redraw(r, rf.f, screen, -1)
	return t
}

func (t *Text) Redraw(r image.Rectangle, f *draw.Font, b *draw.Image, odx int) {

}

func (t *Text) Resize(r image.Rectangle, keepextra int) int {
	return 0
}

func (t *Text) Close() {

}

func (t *Text) Columnate(dlp **Dirlist, ndl int) {

}

func (t *Text) Load(q0 uint, file string, setquid bool) int {
	return 0
}

func (t *Text) Backnl(p, n uint) uint {
	return 0
}

func (t *Text) BsInsert(q0 uint, r []rune, n uint, tofile bool, nrp *int) uint {
	return 0
}

func (t *Text) Insert(q0 uint, r []rune, tofile bool) {

}

func (t *Text) TypeCommit() {

}

func (t *Text) Fill() {

}

func (t *Text) Delete(q0, q1 uint, tofile bool) {

}

func (t *Text) Constrain(q0, q1 uint, p0, p1 *uint) {
	*p0 = minu(q0, t.file.b.nc)
	*p1 = minu(q1, t.file.b.nc)
}

func (t *Text) ReadRune(q uint) rune {
	return ' '
}

func (t *Text) BsWidth(c rune) int {
	return 0
}

func (t *Text) FileWidth(q0 uint, oneelement int) int {
	return 0
}

func (t *Text) Complete() []rune {
	return nil
}

func (t *Text) Type(r rune) {

}

func (t *Text) Commit(tofile bool) {

}

func (t *Text) FrameScroll(dl int) {

}

func (t *Text) Select() {

}

func (t *Text) Show(q0, q1 uint, doselect bool) {

}

func (t *Text) SetSelect(q0, q1 uint) {

}

func (t *Text) Select23(q0, q1 *uint, high *draw.Image, mask int) int {
	return 0
}

func (t *Text) Select3(q0, q1 *uint) int {
	return 0
}

func (t *Text) DoubleClick(q0, q1 *uint) {

}

func (t *Text) ClickMatch(cl, cr, dir int, q *uint) int {
	return 0
}

func (t *Text) ishtmlstart(q uint, q1 *uint) bool {
	return false
}

func (t *Text) ishtmlend(q uint, q0 *uint) bool {
	return false
}

func (t *Text) ClickHTMLMatch(q0, q1 *uint) int {
	return 0
}

func (t *Text) BackNL(p, n uint) uint {
	return 0
}

func (t *Text) SetOrigin(org uint, exact int) {

}

func (t *Text) Reset() {

}
