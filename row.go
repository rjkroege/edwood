package main

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"9fans.net/go/draw"
)

type Row struct {
	display *draw.Display
	lk      sync.Mutex
	r       image.Rectangle
	tag     Text
	col     []*Column
}

func (row *Row) Init(r image.Rectangle, dis *draw.Display) *Row {
	if row == nil {
		row = &Row{}
	}
	row.display = dis
	row.display.ScreenImage.Draw(r, row.display.White, nil, image.ZP)
	row.col = []*Column{}
	row.r = r
	r1 := r
	r1.Max.Y = r1.Min.Y + fontget(tagfont, row.display).Height
	t := &row.tag
	f := new(File)
	t.file = f.AddText(t)
	t.Init(r1, tagfont, tagcolors, row.display)
	t.what = Rowtag
	t.row = row
	t.w = nil
	t.col = nil
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += row.display.ScaleSize(Border)
	row.display.ScreenImage.Draw(r1, row.display.Black, nil, image.ZP)
	t.Insert(0, []rune("Newcol Kill Putall Dump Exit"), true)
	t.SetSelect(t.file.b.Nc(), t.file.b.Nc())
	return row
}

func (row *Row) Add(c *Column, x int) *Column {
	r := row.r
	var d *Column

	// Work out the geometry of the column.
	r.Min.Y = row.tag.fr.Rect().Max.Y + row.display.ScaleSize(Border)
	if x < r.Min.X && len(row.col) > 0 { // Take 40% of last column unless specified
		d = row.col[len(row.col)-1]
		x = d.r.Min.X + 3*d.r.Dx()/5
	}
	/* look for column we'll land on */
	var colidx int
	for colidx, d = range row.col {
		if x < d.r.Max.X {
			break
		}
	}
	if len(row.col) > 0 {
		if colidx < len(row.col) {
			colidx++ // Place new column after d
		}
		r = d.r
		if r.Dx() < 100 {
			return nil // Refuse columns too narrow
		}
		row.display.ScreenImage.Draw(r, row.display.White, nil, image.ZP)
		r1 := r
		r1.Max.X = min(x-row.display.ScaleSize(Border), r.Max.X-row.display.ScaleSize(50))
		if r1.Dx() < row.display.ScaleSize(50) {
			r1.Max.X = r1.Min.X + row.display.ScaleSize(50)
		}
		d.Resize(r1)
		r1.Min.X = r1.Max.X
		r1.Max.X = r1.Min.X + row.display.ScaleSize(Border)
		row.display.ScreenImage.Draw(r1, row.display.Black, nil, image.ZP)
		r.Min.X = r1.Max.X
	}
	if c == nil {
		c = &Column{}
		c.Init(r, row.display)
	} else {
		c.Resize(r)
	}
	c.row = row
	c.tag.row = row
	row.col = append(row.col, nil)
	copy(row.col[colidx+1:], row.col[colidx:])
	row.col[colidx] = c
	clearmouse()
	return c
}

func (r *Row) Resize(rect image.Rectangle) {
	or := row.r
	deltax := rect.Min.X - or.Min.X
	row.r = rect
	r1 := rect
	r1.Max.Y = r1.Min.Y + fontget(tagfont, r.display).Height
	row.tag.Resize(r1, true, false)
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += row.display.ScaleSize(Border)
	row.display.ScreenImage.Draw(r1, row.display.Black, nil, image.ZP)
	rect.Min.Y = r1.Max.Y
	r1 = rect
	r1.Max.X = r1.Min.X
	for i := 0; i < len(row.col); i++ {
		c := row.col[i]
		r1.Min.X = r1.Max.X
		/* the test should not be necessary, but guarantee we don't lose a pixel */
		if i == len(row.col)-1 {
			r1.Max.X = rect.Max.X
		} else {
			r1.Max.X = (c.r.Max.X-or.Min.X)*rect.Dx()/or.Dx() + deltax
		}
		if i > 0 {
			r2 := r1
			r2.Max.X = r2.Min.X + row.display.ScaleSize(Border)
			row.display.ScreenImage.Draw(r2, row.display.Black, nil, image.ZP)
			r1.Min.X = r2.Max.X
		}
		c.Resize(r1)
	}
}

func (row *Row) DragCol(c *Column, _ int) {
	var (
		r       image.Rectangle
		i, b, x int
		p, op   image.Point
		d       *Column
	)
	clearmouse()
	row.display.SetCursor(&boxcursor)
	b = mouse.Buttons
	op = mouse.Point
	for mouse.Buttons == b {
		mousectl.Read()
	}
	row.display.SetCursor(nil)
	if mouse.Buttons != 0 {
		for mouse.Buttons != 0 {
			mousectl.Read()
		}
		return
	}

	for i = 0; i < len(row.col); i++ {
		if row.col[i] == c {
			goto Found
		}
	}
	acmeerror("can't find column", nil)

Found:
	p = mouse.Point
	if abs(p.X-op.X) < 5 && abs(p.Y-op.Y) < 5 {
		return
	}
	if (i > 0 && p.X < row.col[i-1].r.Min.X) || (i < len(row.col)-1 && p.X > c.r.Max.X) {
		/* shuffle */
		x = c.r.Min.X
		row.Close(c, false)
		if (row.Add(c, p.X) == nil) && /* whoops! */
			(row.Add(c, x) == nil) && /* WHOOPS! */
			(row.Add(c, -1) == nil) { /* shit! */
			row.Close(c, true)
			return
		}
		c.MouseBut()
		return
	}
	if i == 0 {
		return
	}
	d = row.col[i-1]
	if p.X < d.r.Min.X+row.display.ScaleSize(80+Scrollwid) {
		p.X = d.r.Min.X + row.display.ScaleSize(80+Scrollwid)
	}
	if p.X > c.r.Max.X-row.display.ScaleSize(80-Scrollwid) {
		p.X = c.r.Max.X - row.display.ScaleSize(80-Scrollwid)
	}
	r = d.r
	r.Max.X = c.r.Max.X
	row.display.ScreenImage.Draw(r, row.display.White, nil, image.ZP)
	r.Max.X = p.X
	d.Resize(r)
	r = c.r
	r.Min.X = p.X
	r.Max.X = r.Min.X
	r.Max.X += row.display.ScaleSize(Border)
	row.display.ScreenImage.Draw(r, row.display.Black, nil, image.ZP)
	r.Min.X = r.Max.X
	r.Max.X = c.r.Max.X
	c.Resize(r)
	c.MouseBut()
}

func (row *Row) Close(c *Column, dofree bool) {
	var (
		r image.Rectangle
		i int
	)

	for i = 0; i < len(row.col); i++ {
		if row.col[i] == c {
			goto Found
		}
	}
	acmeerror("can't find column", nil)
Found:
	r = c.r
	if dofree {
		c.CloseAll()
	}
	row.col = append(row.col[:i], row.col[i+1:]...)
	if len(row.col) == 0 {
		row.display.ScreenImage.Draw(r, row.display.White, nil, image.ZP)
		return
	}
	if i == len(row.col) { /* extend last column right */
		c = row.col[i-1]
		r.Min.X = c.r.Min.X
		r.Max.X = row.r.Max.X
	} else { /* extend next window left */
		c = row.col[i]
		r.Max.X = c.r.Max.X
	}
	row.display.ScreenImage.Draw(r, row.display.White, nil, image.ZP)
	c.Resize(r)
}

func (r *Row) WhichCol(p image.Point) *Column {
	for i := 0; i < len(row.col); i++ {
		c := row.col[i]
		if p.In(c.r) {
			return c
		}
	}
	return nil
}

func (r *Row) Which(p image.Point) *Text {
	if p.In(row.tag.all) {
		return &row.tag
	}
	c := row.WhichCol(p)
	if c != nil {
		return c.Which(p)
	}
	return nil
}

func (row *Row) Type(r rune, p image.Point) *Text {
	var (
		w *Window
		t *Text
	)

	if r == 0 {
		r = utf8.RuneError
	}

	clearmouse()
	row.lk.Lock()
	if bartflag {
		t = barttext
	} else {
		t = row.Which(p)
	}
	if t != nil && !(t.what == Tag && p.In(t.scrollr)) {
		w = t.w
		if w == nil {
			t.Type(r)
		} else {
			w.Lock('K')
			w.Type(t, r)
			/* Expand tag if necessary */
			if t.what == Tag {
				t.w.tagsafe = false
				if r == '\n' {
					t.w.tagexpand = true
				}
				w.Resize(w.r, true, true)
			}
			w.Unlock()
		}
	}
	row.lk.Unlock()
	return t
}

func (row *Row) Clean() bool {

	clean := true
	for _, col := range row.col {
		clean = clean && col.Clean()
	}
	return clean
}

// firstbufline returns the first line of a buffer.
// TODO(rjk): Why don't we save more than the first line of a tag. I want the whole tag saved.
func firstbufline(b *Buffer) string {
	ru := make([]rune, RBUFSIZE)
	n, _ := b.Read(0, ru)

	su := string(ru[0:n])
	// TODO(rjk): I presume that we'll eventually use string everywhere.
	if o := strings.IndexRune(su, '\n'); o > -1 {
		su = su[0:o]
	}
	return su
}

func (r *Row) Dump(file string) {
	dumped := false

	if len(r.col) == 0 {
		return
	}

	if file == "" {
		if home == "" {
			warning(nil, "can't find file for dump: $home not defined\n")
			return
		}

		// Lower risk of simultaneous use of edwood and acme.
		file = filepath.Join(home, "edwood.dump")
	}

	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		warning(nil, "can't open %s: %v\n", err)
		return
	}
	defer fd.Close()

	b := bufio.NewWriter(fd)

	fmt.Fprintf(b, "%s\n", wdir)
	fmt.Fprintf(b, "%s\n", *varfontflag)
	fmt.Fprintf(b, "%s\n", *fixedfontflag)

	for i, c := range r.col {
		fmt.Fprintf(b, "%11.7f", 100.0*float64(c.r.Min.X-row.r.Min.X)/float64(r.r.Dx()))
		if i == len(r.col)-1 {
			b.WriteRune('\n')
		} else {
			b.WriteRune(' ')
		}
	}

	for _, c := range r.col {
		for _, w := range c.w {
			w.body.file.dumpid = 0
		}
	}

	fmt.Fprintf(b, "w %s\n", firstbufline(&r.tag.file.b))
	for i, c := range r.col {
		fmt.Fprintf(b, "c%11d %s\n", i, firstbufline(&c.tag.file.b))
	}

	for i, c := range r.col {
	NextWindow:
		for j, w := range c.w {
			// Do we need to Commit on the other tags?
			w.Commit(&w.tag)
			t := &w.body

			/* windows owned by others get special treatment */
			if w.nopen[QWevent] > 0 {
				if w.dumpstr == "" {
					continue
				}
			}

			/* zeroxes of external windows are tossed */
			if len(t.file.text) > 1 {
				for _, t1 := range t.file.text {
					if w == t1.w {
						continue
					}

					if t1.w.nopen[QWevent] != 0 {
						continue NextWindow
					}
				}
			}

			// We always include the font name.
			fontname := t.font

			if t.file.dumpid > 0 {
				dumped = false
				fmt.Fprintf(b, "x%11d %11d %11d %11d %11.7f %s\n", i, t.file.dumpid,
					w.body.q0, w.body.q1,
					100.0*float64(w.r.Min.Y-c.r.Min.Y)/float64(c.r.Dy()),
					fontname)
			} else if w.dumpstr != "" {
				dumped = false
				fmt.Fprintf(b, "e%11d %11d %11d %11d %11.7f %s\n", i, t.file.dumpid,
					0, 0,
					100.0*float64(w.r.Min.Y-c.r.Min.Y)/float64(c.r.Dy()),
					fontname)
			} else if w.dirty == false && access(t.file.name) || w.isdir {
				dumped = false
				t.file.dumpid = w.id
				fmt.Fprintf(b, "f%11d %11d %11d %11d %11.7f %s\n", i, w.id,
					w.body.q0, w.body.q1,
					100.0*float64(w.r.Min.Y-c.r.Min.Y)/float64(c.r.Dy()),
					fontname)
			} else {
				dumped = true
				t.file.dumpid = w.id
				fmt.Fprintf(b, "F%11d %11d %11d %11d %11.7f %11d %s\n", i, j,
					w.body.q0, w.body.q1,
					100.0*float64(w.r.Min.Y-c.r.Min.Y)/float64(c.r.Dy()),
					w.body.file.b.Nc(), fontname)
			}
			b.WriteString(w.CtlPrint(false))
			fmt.Fprintf(b, "%s\n", firstbufline(&w.tag.file.b))
			if dumped {
				for q0, q1 := 0, t.file.b.Nc(); q0 < q1; {
					ru := make([]rune, RBUFSIZE)
					n, _ := t.file.b.Read(q0, ru)
					su := string(ru[0:n])
					fmt.Fprintf(b, "%s", su)
					q0 += n
				}
			}
			if w.dumpstr != "" {
				if w.dumpdir != "" {
					fmt.Fprintf(b, "%s\n%s\n", w.dumpdir, w.dumpstr)
				} else {
					fmt.Fprintf(b, "\n%s\n", w.dumpstr)
				}
			}
		}
	}

	b.Flush()
}

// LoadFonts gets the font names from the load file so we don't load 
// fonts that we won't use.
func LoadFonts(file string) []string {
	// C: rowloadfonts
	f, err  := os.Open(file)
	if err != nil {
		return []string{}
	}
	defer f.Close()
	b := bufio.NewReader(f)

	// Read first line of dump file (the current directory) and discard.
	if _, err := b.ReadString('\n'); err != nil {
		return []string{}
	}

	// Read names of global fonts
	fontnames := make([]string, 0, 2)

	for i := 0; i < 2; i++ {
		fn, err := b.ReadString('\n')
		if err != nil {
			return []string{}
		}
		
		fontnames = append(fontnames, strings.TrimRight(fn, "\n"))
	}
	return fontnames
}

func (r *Row) Load(file string, initing bool) error {
	Unimpl()
	// C: rowload
	return nil
}

func (r *Row) AllWindows(f func(*Window)) {
	for _, c := range r.col {
		for _, w := range c.w {
			f(w)
		}
	}
}

func (r *Row) LookupWin(id int, dump bool) *Window {
	for _, c := range r.col {
		for _, w := range c.w {
			if dump && w.dumpid == id {
				return w
			}
			if !dump && w.id == id {
				return w
			}
		}
	}
	return nil
}
