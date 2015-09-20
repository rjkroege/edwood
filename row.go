package main

import (
	"image"
	"sync"
)

type Row struct {
	lk   *sync.Mutex
	r    image.Rectangle
	tag  Text
	col  **Column
	ncol int
}

func NewRow(r image.Rectangle) *Row {
	return nil
}

func (r *Row) Add(c *Column, x int) *Column {
	return nil
}

func (r *Row) Resize(rect image.Rectangle) {

}

func (r *Row) DragCol(c *Column, _0 int) {

}

func (r *Row) Close(c *Column, dofree bool) {

}

func (r *Row) WhichCol(p image.Point) *Column {
	return nil
}

func (r *Row) Which(p image.Point) *Text {
	return nil
}

func (r *Row) Type(n string, p image.Point) *Text {
	return nil
}

func (r *Row) Clean() int {
	return 0
}

func (r *Row) Dump(file string) {

}

func (r *Row) LoadFonts(file string) {

}

func (r *Row) Load(file string, initing bool) int {
	return 0
}

func AllWindows(f func(*Window, interface{}), arg interface{}) {

}
