package main

import (
	"image"
	"math"
	"sync"
)

type Window struct {
	lk   *sync.Mutex
	ref  Ref
	tag  Text
	body Text
	r    image.Rectangle

	isdir     bool
	isscratch bool
	filemenu  bool
	dirty     bool
	autoident bool
	showdel   bool

	id    int
	addr  Range
	limit Range

	nopen     [QMAX]bool
	nomark    bool
	wselrange Range
	rdselfd   int

	col    *Column
	eventx Xfid
	events string

	nevents     int
	owner       int
	maxlines    int
	dlp         **Dirlist
	ndl         int
	putseq      int
	nincl       int
	incl        []string
	reffont     *Reffont
	ctrllock    *sync.Mutex
	ctlfid      uint
	dumpstr     string
	dumpdir     string
	dumpid      int
	utflastqid  int
	utflastboff int
	utflastq    int
	tagsafe     int
	tagexpand   bool
	taglines    int
	tagtop      image.Rectangle
	editoutlk   *sync.Mutex
}

func (w *Window) WinInit(clone *Window, r image.Rectangle) {

	//	var r1, br image.Rectangle
	//	var f *File
	//	var rf *Reffont
	//	var rp []rune
	//	var nc int

	w.tag.w = w
	w.taglines = 1
	w.tagexpand = true
	w.body.w = w

	//	WinId++
	//	w.id = WinId

	w.ref.Inc()
	if globalincref {
		w.ref.Inc()
	}
	w.ctlfid = math.MaxUint64
	w.utflastqid = -1
	//	r1 = r

	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + font.Height
}

func (w *Window) DrawButton() {

}

func (w *Window) RunePos() int {
	return 0
}

func (w *Window) ToDel() {

}

func (w *Window) TagLines(r image.Rectangle) int {
	return 0
}

func (w *Window) Resize(r image.Rectangle, safe, keepextra int) int {
	return 0
}

func (w *Window) Lock1(owner int) {

}

func (w *Window) Lock(owner int) {

}

func (w *Window) Unlock() {

}

func (w *Window) MouseBut() {

}

func (w *Window) DirFree() {

}

func (w *Window) Close() {

}

func (w *Window) Delete() {

}

func (w *Window) Undo(isundo bool) {

}

func (w *Window) SetName(name string, n int) {

}

func (w *Window) Type(t *Text, r rune) {

}

func (w *Window) ClearTag() {

}

func (w *Window) SetTag1() {

}

func (w *Window) SetTag() {

}

func (w *Window) Commit(t *Text) {

}

func (w *Window) AddIncl(r string, n int) {

}

func (w *Window) Clean(conservative bool) int {
	return 0
}

func (w *Window) CtlPrint(buf string, fonts int) string {
	return ""
}

func (w *Window) Event(fmt string, args ...interface{}) {

}
