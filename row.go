package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/util"
)

const RowTag = "Newcol Kill Putall Dump Exit"

type Row struct {
	display draw.Display
	lk      sync.Mutex
	r       image.Rectangle
	tag     Text
	col     []*Column
}

func (row *Row) Init(r image.Rectangle, dis draw.Display) *Row {
	if row == nil {
		row = &Row{}
	}
	row.display = dis
	row.display.ScreenImage().Draw(r, row.display.White(), nil, image.Point{})
	row.col = []*Column{}
	row.r = r
	r1 := r
	r1.Max.Y = r1.Min.Y + fontget(tagfont, row.display).Height()
	t := &row.tag
	f := file.MakeObservableEditableBuffer("", nil)
	f.AddObserver(t)
	t.file = f
	t.Init(r1, tagfont, tagcolors, row.display)
	t.what = Rowtag
	t.row = row
	t.w = nil
	t.col = nil
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += row.display.ScaleSize(Border)
	row.display.ScreenImage().Draw(r1, row.display.Black(), nil, image.Point{})
	t.Insert(0, []rune(RowTag+" "), true)
	t.SetSelect(t.file.Size(), t.file.Size())
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
	// look for column we'll land on
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
		row.display.ScreenImage().Draw(r, row.display.White(), nil, image.Point{})
		r1 := r
		r1.Max.X = util.Min(x-row.display.ScaleSize(Border), r.Max.X-row.display.ScaleSize(50))
		if r1.Dx() < row.display.ScaleSize(50) {
			r1.Max.X = r1.Min.X + row.display.ScaleSize(50)
		}
		d.Resize(r1)
		r1.Min.X = r1.Max.X
		r1.Max.X = r1.Min.X + row.display.ScaleSize(Border)
		row.display.ScreenImage().Draw(r1, row.display.Black(), nil, image.Point{})
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
	row.r = rect
	r1 := rect
	r1.Max.Y = r1.Min.Y + fontget(tagfont, r.display).Height()
	row.tag.Resize(r1, true, false)
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += row.display.ScaleSize(Border)
	row.display.ScreenImage().Draw(r1, row.display.Black(), nil, image.Point{})
	rect.Min.Y = r1.Max.Y
	r1 = rect
	r1.Max.X = r1.Min.X
	for i := 0; i < len(row.col); i++ {
		c := row.col[i]
		r1.Min.X = r1.Max.X
		// the test should not be necessary, but guarantee we don't lose a pixel
		if i == len(row.col)-1 {
			r1.Max.X = rect.Max.X
		} else {
			r1.Max.X = rect.Min.X + (c.r.Max.X-or.Min.X)*rect.Dx()/or.Dx()
		}
		if i > 0 {
			r2 := r1
			r2.Max.X = r2.Min.X + row.display.ScaleSize(Border)
			row.display.ScreenImage().Draw(r2, row.display.Black(), nil, image.Point{})
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
	util.AcmeError("can't find column", nil)

Found:
	p = mouse.Point
	if util.Abs(p.X-op.X) < 5 && util.Abs(p.Y-op.Y) < 5 {
		return
	}
	if (i > 0 && p.X < row.col[i-1].r.Min.X) || (i < len(row.col)-1 && p.X > c.r.Max.X) {
		// shuffle
		x = c.r.Min.X
		row.Close(c, false)
		if (row.Add(c, p.X) == nil) && // whoops!
			(row.Add(c, x) == nil) && // WHOOPS!
			(row.Add(c, -1) == nil) { // shit!
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
	row.display.ScreenImage().Draw(r, row.display.White(), nil, image.Point{})
	r.Max.X = p.X
	d.Resize(r)
	r = c.r
	r.Min.X = p.X
	r.Max.X = r.Min.X
	r.Max.X += row.display.ScaleSize(Border)
	row.display.ScreenImage().Draw(r, row.display.Black(), nil, image.Point{})
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
	util.AcmeError("can't find column", nil)
Found:
	r = c.r
	if dofree {
		c.CloseAll()
	}
	row.col = append(row.col[:i], row.col[i+1:]...)
	if len(row.col) == 0 {
		row.display.ScreenImage().Draw(r, row.display.White(), nil, image.Point{})
		return
	}
	if i == len(row.col) { // extend last column right
		c = row.col[i-1]
		r.Min.X = c.r.Min.X
		r.Max.X = row.r.Max.X
	} else { // extend next window left
		c = row.col[i]
		r.Max.X = c.r.Max.X
	}
	row.display.ScreenImage().Draw(r, row.display.White(), nil, image.Point{})
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
	if *barflag {
		t = barttext
	} else {
		t = row.Which(p)
	}
	if t != nil && !(t.what == Tag && p.In(t.scrollr)) {
		w = t.w
		if w == nil {
			// Texts in column tags or the very top.
			t.Type(r)
		} else {
			w.Lock('K')
			w.Type(t, r)
			// Expand tag if necessary
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

func (r *Row) Dump(file string) error {
	if len(r.col) == 0 {
		return nil
	}

	if file == "" {
		f, err := defaultDumpFile()
		if err != nil {
			return warnError(nil, "can't find file for dump: %v", err)
		}
		file = f
	}
	dump, err := r.dump()
	if err != nil {
		return err
	}

	if err := dump.Save(file); err != nil {
		return warnError(nil, "dumping to %v failed: %v", file, err)
	}
	return nil
}

func (r *Row) dump() (*dumpfile.Content, error) {
	rowTag := r.tag.file.String()
	// Remove commands at the beginning of row tag.
	if i := strings.Index(rowTag, RowTag); i > 1 {
		rowTag = rowTag[i:]
	}
	dump := &dumpfile.Content{
		CurrentDir: wdir,
		VarFont:    *varfontflag,
		FixedFont:  *fixedfontflag,
		RowTag: dumpfile.Text{
			Buffer: rowTag,
			Q0:     r.tag.q0,
			Q1:     r.tag.q1,
		},
		Columns: make([]dumpfile.Column, len(r.col)),
		Windows: nil,
	}

	dumpid := make(map[*file.ObservableEditableBuffer]int)

	for i, c := range r.col {
		dump.Columns[i] = dumpfile.Column{
			Position: 100.0 * float64(c.r.Min.X-row.r.Min.X) / float64(r.r.Dx()),
			Tag: dumpfile.Text{
				Buffer: c.tag.file.String(),
				Q0:     c.tag.q0,
				Q1:     c.tag.q1,
			},
		}
		for _, w := range c.w {
			if w.nopen[QWevent] != 0 {
				// Mark zeroxes of external windows specially.
				dumpid[w.body.file] = -1
			}
		}
	}

	for i, c := range r.col {
		for _, w := range c.w {
			// Do we need to Commit on the other tags?
			w.Commit(&w.tag)
			t := &w.body

			// External windows can't be recreated so skip them.
			if w.nopen[QWevent] > 0 {
				if w.dumpstr == "" {
					continue
				}
			}

			// zeroxes of external windows are tossed
			if dumpid[t.file] < 0 && w.nopen[QWevent] == 0 {
				continue
			}

			// We always include the font name.
			fontname := t.font

			dump.Windows = append(dump.Windows, &dumpfile.Window{
				Column: i,
				Body: dumpfile.Text{
					Buffer: "", // filled in later if Unsaved
					Q0:     w.body.q0,
					Q1:     w.body.q1,
				},
				Position: 100.0 * float64(w.r.Min.Y-c.r.Min.Y) / float64(c.r.Dy()),
				Font:     fontname,
			})
			dw := dump.Windows[len(dump.Windows)-1]

			switch {
			case dumpid[t.file] > 0:
				dw.Type = dumpfile.Zerox

			case w.dumpstr != "":
				dw.Type = dumpfile.Exec
				dw.ExecDir = w.dumpdir
				dw.ExecCommand = w.dumpstr

			case !w.body.file.Dirty() && access(t.file.Name()) || w.body.file.IsDir():
				dumpid[t.file] = w.id
				dw.Type = dumpfile.Saved

			default:
				dumpid[t.file] = w.id
				// TODO(rjk): Conceivably this is a bit of a layering violation?
				dw.Type = dumpfile.Unsaved
				dw.Body.Buffer = t.file.String()
			}
			dw.Tag = dumpfile.Text{
				Buffer: w.tag.file.String(),
				Q0:     w.tag.q0,
				Q1:     w.tag.q1,
			}
		}
	}
	return dump, nil
}

// loadhelper breaks out common load file parsing functionality for selected row
// types.
func (row *Row) loadhelper(win *dumpfile.Window) error {
	// Column for this window.
	i := win.Column

	if i > len(row.col) { // Didn't we already make sure that we have a column?
		i = len(row.col)
	}
	c := row.col[i]
	y := c.r.Min.Y + int((win.Position*float64(c.r.Dy()))/100.+0.5)
	if y < c.r.Min.Y || y >= c.r.Max.Y {
		y = -1
	}

	subl := strings.SplitN(win.Tag.Buffer, " ", 2)
	if len(subl) != 2 {
		return fmt.Errorf("bad window tag in dump file %q", win.Tag)
	}

	var w *Window
	if win.Type != dumpfile.Zerox {
		w = c.Add(nil, nil, y)
	} else {
		w = c.Add(nil, lookfile(subl[0]), y)
	}
	if w == nil {
		// Why is this not an error?
		return nil
	}

	if win.Type != dumpfile.Zerox {
		w.SetName(subl[0])
	}

	// TODO(rjk): I feel that the code for managing tags could be extracted and unified.
	// Maybe later. Window.setTag1 would seem fixable.
	afterbar := strings.SplitN(subl[1], "|", 2)
	if len(afterbar) != 2 {
		return fmt.Errorf("bad window tag in dump file %q", win.Tag)
	}
	w.ClearTag()

	w.tag.Insert(w.tag.file.Size(), []rune(afterbar[1]), true)
	w.tag.Show(win.Tag.Q0, win.Tag.Q1, true)

	if win.Type == dumpfile.Unsaved {
		w.body.LoadReader(0, subl[0], strings.NewReader(win.Body.Buffer), true)
		w.body.file.Modded()

		// This shows an example where an observer would be useful?
		w.SetTag()
	} else if win.Type != dumpfile.Zerox && len(subl[0]) > 0 && subl[0][0] != '+' && subl[0][0] != '-' {
		// Implementation of the Get command: open the file.
		get(&w.body, nil, nil, false, false, "")
	}

	if win.Font != "" {
		fontx(&w.body, nil, nil, false, false, win.Font)
	}

	q0 := win.Body.Q0
	q1 := win.Body.Q1
	if q0 > w.body.file.Size() || q1 > w.body.file.Size() || q0 > q1 {
		q0 = 0
		q1 = 0
	}
	// Update the selection on the Text.
	w.body.Show(q0, q1, true)
	ffs := w.body.fr.GetFrameFillStatus()
	w.maxlines = util.Min(ffs.Nlines, util.Max(w.maxlines, ffs.Nlines))

	// TODO(rjk): Conceivably this should be a zerox xfidlog when reconstituting a zerox?
	xfidlog(w, "new")
	return nil
}

// Load restores Edwood's state stored in dump. If dump is nil, it is parsed from file.
// If initing is true, Row will be initialized.
func (row *Row) Load(dump *dumpfile.Content, file string, initing bool) error {
	if dump == nil {
		if file == "" {
			f, err := defaultDumpFile()
			if err != nil {
				return warnError(nil, "can't find file for load: %v", err)
			}
			file = f
		}
		d, err := dumpfile.Load(file)
		if err != nil {
			return warnError(nil, "can't load dump file: %v", err)
		}
		dump = d
	}
	err := row.loadimpl(dump, initing)
	if err != nil {
		return warnError(nil, "can't load row: %v", err)
	}
	return err
}

// TODO(rjk): split this apart into smaller functions and files.
func (row *Row) loadimpl(dump *dumpfile.Content, initing bool) error {
	// log.Println("Load start", file, initing)
	// defer log.Println("Load ended")

	// Current directory.
	if err := os.Chdir(dump.CurrentDir); err != nil {
		return err
	}
	wdir = dump.CurrentDir

	// variable width font
	*varfontflag = dump.VarFont

	// fixed width font
	*fixedfontflag = dump.FixedFont

	if initing && len(row.col) == 0 {
		row.Init(row.display.ScreenImage().R(), row.display)
	}

	// Column widths
	if len(dump.Columns) > 10 {
		return fmt.Errorf("Load: bad number of columns %d", len(dump.Columns))
	}

	// TODO(rjk): put column width parsing in a separate function.
	for i, col := range dump.Columns {
		percent := col.Position
		if percent < 0 || percent >= 100 {
			return fmt.Errorf("Load: column width %f is invalid", percent)
		}

		x := int(float64(row.r.Min.X) + percent*float64(row.r.Dx())/100.0 + 0.5)

		// TODO(rjk): Sigh. A more explicit MVC would simplify thinking about this code.
		if i < len(row.col) {
			if i == 0 {
				continue
			}
			c1 := row.col[i-1]
			c2 := row.col[i]
			r1 := c1.r
			r2 := c2.r
			if x < Border {
				x = Border
			}
			r1.Max.X = x - Border
			r2.Max.X = x
			if r1.Dx() < 50 || r2.Dx() < 50 {
				continue
			}
			row.display.ScreenImage().Draw(image.Rectangle{r1.Min, r2.Max}, row.display.White(), nil, image.Point{})
			c1.Resize(r1)
			c2.Resize(r2)
			r2.Min.X = x - Border
			r2.Max.X = x
			row.display.ScreenImage().Draw(r2, row.display.Black(), nil, image.Point{})
		}
		if i >= len(row.col) {
			row.Add(nil, x)
		}
	}

	// Set row tag
	row.tag.Delete(0, row.tag.file.Size(), true)
	row.tag.Insert(0, []rune(dump.RowTag.Buffer), true)
	row.tag.Show(dump.RowTag.Q0, dump.RowTag.Q1, true)

	// Set column tags
	for i, col := range dump.Columns {
		// Acme's handling of column headers is perplexing. It is conceivable
		// that this code does not do the right thing even if it replicates Acme
		// correctly.
		row.col[i].tag.Delete(0, row.col[i].tag.file.Size(), true)
		row.col[i].tag.Insert(0, []rune(col.Tag.Buffer), true)
		row.col[i].tag.Show(col.Tag.Q0, col.Tag.Q1, true)
	}

	// Load the windows.
	for _, win := range dump.Windows {
		switch win.Type {
		case dumpfile.Exec: // command block
			dirline := win.ExecDir
			if dirline == "" {
				dirline = home
			}
			// log.Println("cmdline", cmdline, "dirline", dirline)
			run(nil, win.ExecCommand, dirline, true, "", "", false)

		case dumpfile.Saved, dumpfile.Unsaved, dumpfile.Zerox:
			if err := row.loadhelper(win); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown dump file window type %v", win.Type)
		}
	}
	return nil
}

func (r *Row) AllWindows(f func(*Window)) {
	for _, c := range r.col {
		for _, w := range c.w {
			f(w)
		}
	}
}

func (r *Row) LookupWin(id int) *Window {
	for _, c := range r.col {
		for _, w := range c.w {
			if w.id == id {
				return w
			}
		}
	}
	return nil
}

func defaultDumpFile() (string, error) {
	if home == "" {
		return "", fmt.Errorf("can't find home directory")
	}
	// Lower risk of simultaneous use of edwood and acme.
	return filepath.Join(home, "edwood.dump"), nil
}
