package main

import (
	"image"
	"sort"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/util"
)

var (
	Lheader = []rune("New Cut Paste Snarf Sort Zerox Delcol ")
)

type Column struct {
	display draw.Display
	Border  int
	r       image.Rectangle
	tag     Text
	row     *Row
	w       []*Window // These are sorted from top to bottom (increasing Y)
	safe    bool
	fortest bool // True if running in test mode (to elide hard to mock actions.)
}

// nw returns the number of Window pointers in Column c.
// TODO(rjk): Consider that this helper is not particularly useful. len is handy.
func (c *Column) nw() int {
	return len(c.w)
}

// Init initializes a new Column object filling image r and drawn to
// display dis.
// TODO(rjk): Why does this need to handle the case where c is nil?
// TODO(rjk): Do we (re)initialize a Column object? It would seem likely.
func (c *Column) Init(r image.Rectangle, dis draw.Display) *Column {
	if c == nil {
		c = &Column{}
	}
	c.display = dis
	c.w = []*Window{}
	c.Border = c.display.ScaleSize(Border)
	if c.display != nil {
		c.display.ScreenImage().Draw(r, c.display.White(), nil, image.Point{})
		c.Border = c.display.ScaleSize(Border)
	}
	c.r = r
	c.tag.col = c
	r1 := r
	r1.Max.Y = r1.Min.Y + fontget(global.tagfont, c.display).Height()

	// TODO(rjk) better code: making tag should be split out.
	tagfile := file.MakeObservableEditableBuffer("", nil)
	tagfile.AddObserver(&c.tag)
	c.tag.file = tagfile
	c.tag.Init(r1, global.tagfont, global.tagcolors, c.display)
	c.tag.what = Columntag
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += c.display.ScaleSize(Border)
	if c.display != nil {
		c.display.ScreenImage().Draw(r1, c.display.Black(), nil, image.Point{})
	}
	c.tag.Insert(0, Lheader, true)
	c.tag.SetSelect(c.tag.file.Nr(), c.tag.file.Nr())
	if c.display != nil {
		c.display.ScreenImage().Draw(c.tag.scrollr, global.colbutton, nil, global.colbutton.R().Min)
		// As a general practice, Edwood is very over-eager to Flush. Flushes hurt
		// perf.
		c.display.Flush()
	}
	c.safe = true
	return c
}

// findWindowContainingY finds the window containing vertical offset y
// and returns the Window and its index.
// TODO(rjk): It's almost certain that we repeat this code somewhere else.
// possibly multiple times.
// TODO(rjk): Get rid of the index requirement?
func (c *Column) findWindowContainingY(y int) (i int, v *Window) {
	for i, v = range c.w {
		if y < v.r.Max.Y {
			return i, v
		}
	}
	return len(c.w), v
}

// findWindowIndex returns the index of window w in the column,
// or -1 if the window is not found.
func (c *Column) findWindowIndex(w *Window) int {
	for i, win := range c.w {
		if win == w {
			return i
		}
	}
	return -1
}

// Add adds a window to the Column.
// TODO(rjk): what are the args?
func (c *Column) Add(w, clone *Window, y int) *Window {
	// Figure out new window placement
	var v *Window

	r := c.r
	r.Min.Y = c.tag.fr.Rect().Max.Y + c.display.ScaleSize(Border)
	if y < r.Min.Y && c.nw() > 0 { // Steal half the last window
		v = c.w[c.nw()-1]
		y = v.body.fr.Rect().Min.Y + v.body.fr.Rect().Dx()/2
	}

	// Which window will we land on?
	var windex int
	windex, v = c.findWindowContainingY(y)

	// TODO(rjk): be polite. :-)
	buggered := false // historical variable name
	if c.nw() > 0 {
		if windex < c.nw() {
			windex++
		}

		// if landing window (v) is too small, grow it first (landing window
		// will be split to accommodate the newly added window.)
		// minht is the height of the first line of the tag and the border thickness
		// TODO(rjk): Make minht a method of the tag to simplify variable height fonts.
		minht := v.tag.fr.DefaultFontHeight() + c.display.ScaleSize(Border) + 1
		j := 0
		// Code inspection suggests that the frame fill status may have altered
		// after resizing.
		for !c.safe || v.body.fr.GetFrameFillStatus().Maxlines < 3 || v.body.all.Dy() <= minht {
			j++
			if j > 10 {
				buggered = true // Too many windows in column
				break
			}
			c.Grow(v, 1)
		}

		// figure out where to split v to make room for w
		// new window stops where next window begins
		var ymax int
		if windex < c.nw() {
			ymax = c.w[windex].r.Min.Y - c.display.ScaleSize(Border)
		} else {
			ymax = c.r.Max.Y
		}

		// new window must start after v's tag ends
		y = util.Max(y, v.tagtop.Max.Y+c.display.ScaleSize(Border))

		// new window must start early enough to end before ymax
		y = util.Min(y, ymax-minht)

		// if y is too small, too many windows in column
		if y < v.tagtop.Max.Y+c.display.ScaleSize(Border) {
			buggered = true
		}

		// Resize & redraw v
		r = v.r
		r.Max.Y = ymax
		if c.display != nil {
			c.display.ScreenImage().Draw(r, global.textcolors[frame.ColBack], nil, image.Point{})
		}
		r1 := r
		y = util.Min(y, ymax-(v.tag.fr.DefaultFontHeight()*v.taglines+v.body.fr.DefaultFontHeight()+c.display.ScaleSize(Border)+1))
		ffs := v.body.fr.GetFrameFillStatus()
		r1.Max.Y = util.Min(y, v.body.fr.Rect().Min.Y+ffs.Nlines*v.body.fr.DefaultFontHeight())
		r1.Min.Y = v.Resize(r1, false, false)
		r1.Max.Y = r1.Min.Y + c.display.ScaleSize(Border)
		if c.display != nil {
			c.display.ScreenImage().Draw(r1, c.display.Black(), nil, image.Point{})
		}
		//
		// leave r with w's coordinates
		//
		r.Min.Y = r1.Max.Y
	}
	if w == nil {
		w = NewWindow()
		w.col = c
		if c.display != nil {
			c.display.ScreenImage().Draw(r, global.textcolors[frame.ColBack], nil, image.Point{})
		}
		w.Init(clone, r, c.display)
	} else {
		w.col = c
		w.Resize(r, false, true)
	}
	w.tag.col = c
	w.tag.row = c.row
	w.body.col = c
	w.body.row = c.row
	c.w = append(c.w, nil)
	copy(c.w[windex+1:], c.w[windex:])
	c.w[windex] = w
	c.safe = true
	if buggered {
		c.Resize(c.r)
	}
	savemouse(w)
	if c.display != nil {
		c.display.MoveTo(w.tag.scrollr.Max.Add(image.Pt(3, 3)))
	}
	global.barttext = &w.body
	return w
}

// Close called to remove w from Column c. Set dofree to true to actually
// delete window w. Otherwise, w will be moved to another Column.
func (c *Column) Close(w *Window, dofree bool) {
	var (
		r            image.Rectangle
		i            int
		didmouse, up bool
	)
	// w is locked
	if !c.safe && !c.fortest {
		c.Grow(w, 1)
	}
	for i = 0; i < len(c.w); i++ {
		if c.w[i] == w {
			goto Found
		}
	}
	util.AcmeError("can't find window", nil)
Found:
	r = w.r
	// Crash noted in #385 happens when closing windows with the
	// Edit command. Col.Close is invoked to remove the windows by ecmd.go/D1.
	// When we place the Window in the new column, we'll set this. Or we'll
	// delete the Window in the dofree block.
	w.tag.col = nil
	w.body.col = nil
	w.col = nil
	didmouse = restoremouse(w)
	if dofree {
		w.Delete()
		// This Close call will decrement the w's reference count.
		w.Close()
	}
	c.w = append(c.w[:i], c.w[i+1:]...)
	if len(c.w) == 0 {
		if c.display != nil {
			c.display.ScreenImage().Draw(r, c.display.White(), nil, image.Point{})
		}
		return
	}
	up = false
	if i == len(c.w) { // extend last window down
		w = c.w[i-1]
		r.Min.Y = w.r.Min.Y
		r.Max.Y = c.r.Max.Y
	} else { // extend next window up
		up = true
		w = c.w[i]
		r.Max.Y = w.r.Max.Y
	}
	if c.display != nil {
		c.display.ScreenImage().Draw(r, global.textcolors[frame.ColBack], nil, image.Point{})
	}
	if c.safe && !c.fortest {
		if !didmouse && up {
			w.showdel = true
		}
		w.Resize(r, false, true)
		if !didmouse && up {
			w.moveToDel()
		}
	}
}

func (c *Column) CloseAll() {
	if c == global.activecol {
		global.activecol = nil
	}
	c.tag.Close()
	for _, w := range c.w {
		w.Close()
	}
	clearmouse()
}

func (c *Column) MouseBut() {
	if c.display != nil {
		c.display.MoveTo(c.tag.scrollr.Min.Add(c.tag.scrollr.Max).Div(2))
	}
}

func (c *Column) Resize(r image.Rectangle) {
	clearmouse()
	r1 := r
	r1.Max.Y = r1.Min.Y + c.tag.fr.DefaultFontHeight()
	c.tag.Resize(r1, true, false)
	if c.display != nil {
		c.display.ScreenImage().Draw(c.tag.scrollr, global.colbutton, nil, global.colbutton.R().Min)
	}
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += c.display.ScaleSize(Border)
	c.display.ScreenImage().Draw(r1, c.display.Black(), nil, image.Point{})
	r1.Max.Y = r.Max.Y
	for i := 0; i < c.nw(); i++ {
		w := c.w[i]
		w.maxlines = 0
		if i == c.nw()-1 {
			r1.Max.Y = r.Max.Y
		} else {
			r1.Max.Y = r1.Min.Y
			if c.r.Dy() != 0 {
				r1.Max.Y += (w.r.Dy() + c.display.ScaleSize(Border)) * r.Dy() / c.r.Dy()
			}
		}
		r1.Max.Y = util.Max(r1.Max.Y, r1.Min.Y+c.display.ScaleSize(Border)+fontget(global.tagfont, c.display).Height())
		r2 := r1
		r2.Max.Y = r2.Min.Y + c.display.ScaleSize(Border)
		c.display.ScreenImage().Draw(r2, c.display.Black(), nil, image.Point{})
		r1.Min.Y = r2.Max.Y
		r1.Min.Y = w.Resize(r1, false, i == c.nw()-1)
	}
	c.r = r
}

func (c *Column) Sort() {
	sort.Slice(c.w, func(i, j int) bool { return c.w[i].body.file.Name() < c.w[j].body.file.Name() })

	r := c.r
	r.Min.Y = c.tag.fr.Rect().Max.Y
	c.display.ScreenImage().Draw(r, global.textcolors[frame.ColBack], nil, image.Point{})
	y := r.Min.Y
	for i := 0; i < len(c.w); i++ {
		w := c.w[i]
		r.Min.Y = y
		if i == len(c.w)-1 {
			r.Max.Y = c.r.Max.Y
		} else {
			r.Max.Y = r.Min.Y + w.r.Dy() + c.display.ScaleSize(Border)
		}
		r1 := r
		r1.Max.Y = r1.Min.Y + c.display.ScaleSize(Border)
		c.display.ScreenImage().Draw(r1, c.display.Black(), nil, image.Point{})
		r.Min.Y = r1.Max.Y
		y = w.Resize(r, false, i == len(c.w)-1)
	}
}

// Grow Window w with a mode determined by mouse button but.
func (c *Column) Grow(w *Window, but int) {
	var windex int

	for windex = 0; windex < len(c.w); windex++ {
		if c.w[windex] == w {
			break
		}
	}
	if windex == len(c.w) {
		util.AcmeError("can't find window", nil)
	}

	cr := c.r
	if but < 0 { // make sure window fills its own space properly
		r := w.r
		if windex == c.nw()-1 || !c.safe { // Last window in column
			r.Max.Y = cr.Max.Y // Clamp to column bottom.
		} else {
			// Fill space down to the next window.
			r.Max.Y = c.w[windex+1].r.Min.Y - c.display.ScaleSize(Border)
		}
		w.Resize(r, false, true)
		return
	}
	cr.Min.Y = c.w[0].r.Min.Y
	if but == 3 { // Switch to full size window
		if windex != 0 {
			v := c.w[0]
			c.w[0] = w
			c.w[windex] = v
		}
		c.display.ScreenImage().Draw(cr, global.textcolors[frame.ColBack], nil, image.Point{})
		w.Resize(cr, false, true)
		for i := 1; i < c.nw(); i++ {
			ffs := c.w[i].body.fr.GetFrameFillStatus()
			ffs.Maxlines = 0
		}
		c.safe = false
		return
	}

	// Observation: before I can support lines of arbitrary height, I need to change
	// Frame to paint partial lines of text.
	// TODO(rjk): Rewrite this logic for computing heights when font heights vary.
	// store old #lines for each window
	onl := w.body.fr.GetFrameFillStatus().Maxlines
	nl := make([]int, c.nw())
	tot := 0
	for j := 0; j < c.nw(); j++ {
		l := c.w[j].taglines - 1 + c.w[j].body.fr.GetFrameFillStatus().Maxlines // TODO(flux): This taglines subtraction (for scrolling tags) assumes tags take the same number of pixels height as the body lines.  This is clearly false.
		nl[j] = l
		tot += l
	}

	// approximate new #lines for this window
	if but == 2 { // as big as can be
		for i := range nl {
			nl[i] = 0
		}
	} else {
		nnl := util.Min(onl+util.Max(util.Min(5, w.taglines-1+w.maxlines), onl/2), tot) // TODO(flux) more bad taglines use
		if nnl < w.taglines-1+w.maxlines {
			nnl = (w.taglines - 1 + w.maxlines + nnl) / 2
		}
		if nnl == 0 {
			nnl = 2
		}
		dnl := nnl - onl
		// compute new #lines for each window
		for k := 1; k < c.nw(); k++ {
			// prune from later window
			j := windex + k
			if j < c.nw() && nl[j] != 0 {
				l := util.Min(dnl, util.Max(1, nl[j]/2))
				nl[j] -= l
				nl[windex] += l
				dnl -= l
			}
			// prune from earlier window
			j = windex - k
			if j >= 0 && nl[j] != 0 {
				l := util.Min(dnl, util.Max(1, nl[j]/2))
				nl[j] -= l
				nl[windex] += l
				dnl -= l
			}
		}
	}
	c.packColumn(w, windex, cr, nl)
}

// packColumn resizes all windows in the column to accommodate the target window w at windex.
// nl contains the target number of lines for each window.
func (c *Column) packColumn(w *Window, windex int, cr image.Rectangle, nl []int) {
	ny := make([]int, c.nw())
	// pack everyone above
	y1 := cr.Min.Y
	var v *Window

	// Resize windows [0, target window)
	for j := 0; j < windex; j++ {
		v = c.w[j]
		r := v.r
		r.Min.Y = y1
		r.Max.Y = y1 + v.tagtop.Dy()
		if nl[j] != 0 {
			r.Max.Y += 1 + nl[j]*v.body.fr.DefaultFontHeight()
		}
		r.Min.Y = v.Resize(r, false, false)
		r.Max.Y += c.display.ScaleSize(Border)
		c.display.ScreenImage().Draw(r, c.display.Black(), nil, image.Point{})
		y1 = r.Max.Y
	}
	// scan to see new size of everyone below
	y2 := c.r.Max.Y
	for j := c.nw() - 1; j > windex; j-- {
		v = c.w[j]
		r := v.r
		r.Min.Y = y2 - v.tagtop.Dy()
		if nl[j] != 0 {
			r.Min.Y -= 1 + nl[j]*v.body.fr.DefaultFontHeight()
		}
		r.Min.Y -= c.display.ScaleSize(Border)
		ny[j] = r.Min.Y
		y2 = r.Min.Y
	}
	// compute new size of window
	r := w.r
	r.Min.Y = y1
	r.Max.Y = y2
	h := w.body.fr.DefaultFontHeight() // TODO(flux) Is this the right frame font height to use?
	if r.Dy() < w.tagtop.Dy()+1+h+c.display.ScaleSize(Border) {
		r.Max.Y = r.Min.Y + w.tagtop.Dy() + 1 + h + c.display.ScaleSize(Border)
	}
	// draw window
	r.Max.Y = w.Resize(r, false, true)
	if windex < c.nw()-1 {
		r.Min.Y = r.Max.Y
		r.Max.Y += c.display.ScaleSize(Border)
		c.display.ScreenImage().Draw(r, c.display.Black(), nil, image.Point{})
		for j := windex + 1; j < c.nw(); j++ {
			ny[j] -= (y2 - r.Max.Y)
		}
	}
	// pack everyone below
	y1 = r.Max.Y
	for j := windex + 1; j < c.nw(); j++ {
		v = c.w[j]
		r = v.r
		r.Min.Y = y1
		r.Max.Y = y1 + v.tagtop.Dy()
		if nl[j] != 0 {
			r.Max.Y += 1 + nl[j]*v.body.fr.DefaultFontHeight()
		}
		y1 = v.Resize(r, false, j == c.nw()-1)
		if j < c.nw()-1 { // no border on last window
			r.Min.Y = y1
			r.Max.Y += c.display.ScaleSize(Border)
			c.display.ScreenImage().Draw(r, c.display.Black(), nil, image.Point{})
			y1 = r.Max.Y
		}
	}
	c.safe = true
	w.MouseBut()
}

func (c *Column) DragWin(w *Window, but int) {
	var (
		r     image.Rectangle
		b     int
		p, op image.Point
		v     *Window
		nc    *Column
	)
	clearmouse()
	c.display.SetCursor(&boxcursor)
	b = global.mouse.Buttons
	op = global.mouse.Point
	for global.mouse.Buttons == b {
		global.mousectl.Read()
	}
	c.display.SetCursor(nil)
	if global.mouse.Buttons != 0 {
		for global.mouse.Buttons != 0 {
			global.mousectl.Read()
		}
		return
	}

	// Find window w in our column
	i := c.findWindowIndex(w)
	if i < 0 {
		util.AcmeError("can't find window", nil)
		return
	}
	if w.tagexpand { // force recomputation of window tag size
		w.taglines = 1
	}
	p = global.mouse.Point
	if util.Abs(p.X-op.X) < 5 && util.Abs(p.Y-op.Y) < 5 {
		c.Grow(w, but)
		w.MouseBut()
		return
	}
	// is it a flick to the right? Or a jump to the le-e-e-eft?
	if util.Abs(p.Y-op.Y) < 10 && p.X > op.X+30 && c.row.WhichCol(p) == c {
		p.X = op.X + w.r.Dx() // yes: toss to next column
	}
	nc = c.row.WhichCol(p)
	if nc != nil && nc != c {
		c.Close(w, false)
		nc.Add(w, nil, p.Y)
		w.MouseBut()
		return
	}
	if i == 0 && len(c.w) == 1 {
		return // can't do it
	}
	if (i > 0 && p.Y < c.w[i-1].r.Min.Y) || (i < len(c.w)-1 && p.Y > w.r.Max.Y || (i == 0 && p.Y > w.r.Max.Y)) {
		// shuffle
		c.Close(w, false)
		c.Add(w, nil, p.Y)
		w.MouseBut()
		return
	}
	if i == 0 {
		return
	}
	v = c.w[i-1]
	if p.Y < v.tagtop.Max.Y {
		p.Y = v.tagtop.Max.Y
	}
	if p.Y > w.r.Max.Y-w.tagtop.Dy()-c.row.display.ScaleSize(Border) {
		p.Y = w.r.Max.Y - w.tagtop.Dy() - c.row.display.ScaleSize(Border)
	}
	r = v.r
	r.Max.Y = p.Y
	if r.Max.Y > v.body.fr.Rect().Min.Y {
		r.Max.Y -= (r.Max.Y - v.body.fr.Rect().Min.Y) % v.body.fr.DefaultFontHeight()
		if v.body.fr.Rect().Min.Y == v.body.fr.Rect().Max.Y {
			r.Max.Y++
		}
	}
	r.Min.Y = v.Resize(r, c.safe, false)
	r.Max.Y = r.Min.Y + c.row.display.ScaleSize(Border)
	c.display.ScreenImage().Draw(r, c.display.Black(), nil, image.Point{})
	r.Min.Y = r.Max.Y
	if i == len(c.w)-1 {
		r.Max.Y = c.r.Max.Y
	} else {
		r.Max.Y = c.w[i+1].r.Min.Y - c.row.display.ScaleSize(Border)
	}
	w.Resize(r, c.safe, true)
	c.safe = true
	w.MouseBut()
}

func (c *Column) Which(p image.Point) *Text {
	if !p.In(c.r) {
		return nil
	}
	if p.In(c.tag.all) {
		return &c.tag
	}
	for _, w := range c.w {
		if p.In(w.r) {
			if p.In(w.tagtop) || p.In(w.tag.all) {
				return &w.tag
			}
			// exclude partial line at bottom
			if p.X >= w.body.scrollr.Max.X && p.Y >= w.body.fr.Rect().Max.Y {
				return nil
			}
			return &w.body
		}
	}
	return nil
}

func (c *Column) Clean() bool {
	clean := true
	for _, w := range c.w {
		clean = w.Clean(true) && clean
	}
	return clean
}
