package main

import (
	"image"
)

var (
	Lheader = []string{
		"New ",
		"Cut ",
		"Paste ",
		"Snarf ",
		"Sort ",
		"Zerox ",
		"Delcol ",
	}
)

type Column struct {
	r    image.Rectangle
	tag  Text
	row  *Row
	w    **Window
	nw   int
	safe int
}

func NewColumn(r image.Rectangle) *Column {
	return nil
}

func (c *Column) Add(w, clone *Window, y int) *Window {
	return nil
}

func (c *Column) Close(w *Window, dofree bool) {

}

func (c *Column) CloseAll() {

}

func (c *Column) MouseBut() {

}

func (c *Column) Resize(r image.Rectangle) {

}

func cmp(a, b interface{}) int {
	return 0
}

func (c *Column) Sort() {

}

func (c *Column) Grow(w *Window, but int) {

}

func (c *Column) DragWin(w *Window, but int) {

}

func (c *Column) Which(p image.Point) *Text {
	return nil
}

func (c *Column) Clean() int {
	return 0
}
