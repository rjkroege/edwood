package main

import (
	"image"
	"sync"
)

type Row struct {
	lk   sync.Mutex
	r    image.Rectangle
	tag  Text
	col  []*Column
}

func (row *Row)Init(r image.Rectangle) *Row {
	if row == nil {
		row = &Row{}
	}
	display.ScreenImage.Draw(r, display.White, nil, image.ZP)
	row.col = []*Column{}
	row.r = r
	tagfile := NewTagFile()
	r1 := r
	r1.Max.Y = r1.Min.Y + tagfont.Height
	t := &row.tag
	t.Init(tagfile, r1, fontget(0, false, false, ""), tagcolors)
	t.what = Rowtag
	t.row = row
	t.w = nil
	t.col = nil
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += display.ScaleSize(Border)
	display.ScreenImage.Draw(r1, display.Black, nil, image.ZP)
	t.Insert(0, []rune("Newcol Kill Putall Dump Exit"), true)
	t.SetSelect(t.file.b.nc(), t.file.b.nc())
	return row
}

func (r *Row) ncol() uint { return uint(len(r.col)) }

func (row *Row) Add(c *Column, x int) *Column {
	r := row.r
	var d *Column

	// Work out the geometry of the column.
	r.Min.Y = row.tag.fr.Rect.Max.Y + display.ScaleSize(Border)
	if x < r.Min.X && row.ncol() > 0 { // Take 40% of last column unless specified
		d = row.col[row.ncol()-1]
		x = d.r.Min.X + 3 * d.r.Dx()/5
	}
	/* look for column we'll land on */
	var colidx int
	for colidx, _ = range row.col {
		d = row.col[colidx]
		if x < d.r.Max.X {
			break
		}
	}
	if row.ncol() > 0 {
		if uint(colidx) < row.ncol() {
			colidx++ // Place new column after d
		}
		r = d.r
		if r.Dx() < 100 {
			return nil  // Refuse columns too narrow
		}
		display.ScreenImage.Draw(r, display.White, nil, image.ZP)
		r1 := r
		r1.Max.X = min(x-display.ScaleSize(Border), r.Max.X-50)
		if r1.Dx() < 50 {
			r1.Max.X = r1.Min.X + 50
		}
		d.Resize(r1)
		r1.Min.X = r1.Max.X
		r1.Max.X = r1.Min.X + display.ScaleSize(Border)
		display.ScreenImage.Draw(r1, display.Black, nil, image.ZP)
		r.Min.X = r1.Max.X
	}
	if c == nil {
		c = &Column{}
		c.Init(r)
	} else {
		c.Resize(r)
	}
	c.row = row
	c.tag.row = row
	row.col = append(row.col, c)
	//clearmouse() // TODO(flux): Mouse?  weird
	return c
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

func (r *Row) Load(file string, initing bool) error {
	return nil
}

func AllWindows(f func(*Window, interface{}), arg interface{}) {

}
