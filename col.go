package main

import (
	"image"

	"github.com/paul-lalonde/acme/frame"
)

var (
	Lheader = []rune("New Cut Paste Snarf Sort Zerox Delcol")
)

type Column struct {
	r    image.Rectangle
	tag  Text
	row  *Row
	w    []*Window
	safe bool
}

func (c *Column) nw() int {
	return len(c.w)
}

func (c *Column) Init(r image.Rectangle) *Column {
	if c == nil {
		c = &Column{}
	}
	c.w = []*Window{}
	display.ScreenImage.Draw(r, display.White, nil, image.ZP)
	c.r = r
	c.tag.col = c
	tagfile := NewFile("")
	r1 := r
	r1.Max.Y = r1.Min.Y + tagfont.Height
	c.tag.Init(tagfile.AddText(&c.tag), r1, tagfont, tagcolors)
	c.tag.what = Columntag
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += display.ScaleSize(Border)
	display.ScreenImage.Draw(r1, display.Black, nil, image.ZP)
	c.tag.Insert(0, Lheader, true)
	c.tag.SetSelect(c.tag.file.b.nc(), c.tag.file.b.nc())
	display.ScreenImage.Draw(c.tag.scrollr, colbutton, nil, colbutton.R.Min)
	display.Flush()
	c.safe = true
	return c
}

/*
func (c *Column) AddFile(f *File) *Window {
	w := NewWindow(f)
	c.Add(w, nil, 0)
}
*/

func (c *Column) Add(w, clone *Window, y int) *Window {
	// Figure out new window placement
	var v *Window
	var ymax int

	r := c.r
	r.Min.Y = c.tag.fr.Rect.Max.Y + display.ScaleSize(Border)
	if y < r.Min.Y && c.nw() > 0 { // Steal half the last window
		v = c.w[c.nw()-1]
		y = v.body.fr.Rect.Min.Y + v.body.fr.Rect.Dx()/2
	}
	// Which window will we land on?
	var windex int
	for windex = 0; windex < len( c.w ); windex++ {
		v = c.w[windex]
		if y < v.r.Max.Y {
			break
		}
	}
	buggered := false // historical variable name
	if c.nw() > 0 {
		if windex < c.nw() {
			windex++
		}
		/*
		 * if landing window (v) is too small, grow it first.
		 */
		minht := v.tag.fr.Font.DefaultHeight() + display.ScaleSize(Border) + 1
		j := 0
		for !c.safe || v.body.fr.MaxLines < 3 || v.body.all.Dy() <= minht {
			j++
			if j > 10 {
				buggered = true // Too many windows in column
				break
			}
			c.Grow(v, 1)
		}

		/*
		 * figure out where to split v to make room for w
		 */

		// new window stops where next window begins 
		if windex < c.nw() {
			ymax = c.w[windex].r.Min.Y - display.ScaleSize(Border)
		} else {
			ymax = c.r.Max.Y
		}

		// new window must start after v's tag ends 
		y = max(y, v.tagtop.Max.Y+display.ScaleSize(Border))

		// new window must start early enough to end before ymax 
		y = min(y, ymax-minht)

		// if y is too small, too many windows in column 
		if y < v.tagtop.Max.Y+display.ScaleSize(Border) {
			buggered = true
		}

		// Resize & redraw v
		r = v.r
		r.Max.Y = ymax
		display.ScreenImage.Draw(r, textcolors[frame.ColBack], nil, image.ZP)
		r1 := r
		y = min(y, ymax-(v.tag.fr.Font.DefaultHeight()*v.taglines+v.body.fr.Font.DefaultHeight()+display.ScaleSize(Border)+1))
		r1.Max.Y = min(y, v.body.fr.Rect.Min.Y+v.body.fr.NLines*v.body.fr.Font.DefaultHeight())
		r1.Min.Y = v.Resize(r1, false, false)
		r1.Max.Y = r1.Min.Y + display.ScaleSize(Border)
		display.ScreenImage.Draw(r1, display.Black, nil, image.ZP)

		/*
		 * leave r with w's coordinates
		 */
		r.Min.Y = r1.Max.Y
	}
	if w == nil {
		w = NewWindow()
		w.col = c
		display.ScreenImage.Draw(r, textcolors[frame.ColBack], nil, image.ZP)
		w.Init(clone, r)
	} else {
		w.col = c
		w.Resize(r, false, true)
	}
	w.tag.col = c
	w.tag.row = c.row
	w.body.col = c
	w.body.row = c.row
	c.w = append(c.w, w)
	c.safe = true
	if buggered {
		c.Resize(c.r)
	}
	savemouse(w)
	display.MoveTo(w.tag.scrollr.Max.Add(image.Pt(3, 3)))
	barttext = &w.body
	return w
}

func (c *Column) Close(w *Window, dofree bool) {
var (
	r image.Rectangle
	i int
	didmouse, up bool
)
	/* w is locked */
	if !c.safe {
		c.Grow(w, 1);
	}
	for i=0; i<len(c.w); i++ {
		if c.w[i] == w {
			goto Found;
		}
	}
	acmeerror("can't find window", nil);
  Found:
	r = w.r;
	w.tag.col = nil;
	w.body.col = nil;
	w.col = nil;
	didmouse = restoremouse(w);
	if dofree {
		w.Delete();
		w.Close();
	}
	c.w = append(c.w[:i], c.w[i+1:]...)
	if len(c.w) == 0 {
		display.ScreenImage.Draw( r, display.White, nil, image.ZP);
		return;
	}
	up = false;
	if i == len(c.w) {		/* extend last window down */
		w = c.w[i-1];
		r.Min.Y = w.r.Min.Y;
		r.Max.Y = c.r.Max.Y;
	}else{			/* extend next window up */
		up = true;
		w = c.w[i];
		r.Max.Y = w.r.Max.Y;
	}
	display.ScreenImage.Draw(r, textcolors[frame.ColBack], nil, image.ZP);
	if(c.safe) {
		if(!didmouse && up) {
			w.showdel = true;
		}
		w.Resize(r, false, true);
		if(!didmouse && up){
			w.moveToDel();
		}
	}
}

func (c *Column) CloseAll() {
	Unimpl()

}

func (c *Column) MouseBut() {
	Unimpl()

}

func (c *Column) Resize(r image.Rectangle) {
	clearmouse()
	r1 := r
	r1.Max.Y = r1.Min.Y + c.tag.fr.Font.Impl().Height
	c.tag.Resize(r1, true)
	display.ScreenImage.Draw(c.tag.scrollr, colbutton, nil, colbutton.R.Min)
	r1.Min.Y = r1.Max.Y
	r1.Max.Y += display.ScaleSize(Border)
	display.ScreenImage.Draw(r1, display.Black, nil, image.ZP)
	r1.Max.Y = r.Max.Y
	for i := 0; i < c.nw(); i++ {
		w := c.w[i]
		w.maxlines = 0
		if i == c.nw()-1 {
			r1.Max.Y = r.Max.Y
		} else {
			r1.Max.Y = r1.Min.Y + (w.r.Dy()+display.ScaleSize(Border))*r.Dy()/c.r.Dy()
		}
		r1.Max.Y = max(r1.Max.Y, r1.Min.Y+display.ScaleSize(Border)+tagfont.Height)
		r2 := r1
		r2.Max.Y = r2.Min.Y + display.ScaleSize(Border)
		display.ScreenImage.Draw(r2, display.Black, nil, image.ZP)
		r1.Min.Y = r2.Max.Y
		r1.Min.Y = w.Resize(r1, false, i == c.nw()-1)
	}
	c.r = r
}

func cmp(a, b interface{}) int {
	Unimpl()
	return 0
}

func (c *Column) Sort() {
	Unimpl()

}

func (c *Column) Grow(w *Window, but int) {
	//var nl, ny *int
	var v *Window

	var windex int

	for windex = 0; windex < len(c.w); windex++ {
		if c.w[windex] == w {
			break
		}
	}
	if windex == c.nw() {
		panic("can't find window") // TODO(flux): implement counterpart to the C version's error()
	}

	cr := c.r
	if but < 0 { // make sure window fills its own space properly 
		r := w.r
		if windex == int(c.nw()-1) || !c.safe {
			r.Max.Y = cr.Max.Y
		} else {
			r.Max.Y = c.w[windex+1].r.Min.Y - display.ScaleSize(Border)
		}
		w.Resize(r, false, true)
		return
	}
	cr.Min.Y = c.w[0].r.Min.Y
	if but == 3 { // Switch to full size window 
		if windex != 0 {
			v = c.w[0]
			c.w[0] = w
			c.w[windex] = v
		}
		display.ScreenImage.Draw(cr, textcolors[frame.ColBack], nil, image.ZP)
		w.Resize(cr, false, true)
		for i := 1; i < c.nw(); i++ {
			c.w[i].body.fr.MaxLines = 0
		}
		c.safe = false
		return
	}
	// store old #lines for each window 
	onl := w.body.fr.MaxLines
	nl := make([]int, c.nw())
	ny := make([]int, c.nw())
	tot := 0
	for j := 0; j < c.nw(); j++ {
		l := c.w[j].taglines - 1 + c.w[j].body.fr.MaxLines // TODO(flux): This taglines subtraction (for scrolling tags) assumes tags take the same number of pixels height as the body lines.  This is clearly false.
		nl[j] = l
		tot += l
	}
	// approximate new #lines for this window 
	if but == 2 { // as big as can be 
		for i := range nl {
			nl[i] = 0
		}
		goto Pack
	}
	{ // Scope for nnl & dln
		nnl := min(onl+max(min(5, w.taglines-1+w.maxlines), onl/2), tot) // TODO(flux) more bad taglines use
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
				l := min(dnl, max(1, nl[j]/2))
				nl[j] -= l
				nl[windex] += l
				dnl -= l
			}
			// prune from earlier window 
			j = windex - k
			if j >= 0 && nl[j] != 0 {
				l := min(dnl, max(1, nl[j]/2))
				nl[j] -= l
				nl[windex] += l
				dnl -= l
			}
		}
	}
Pack:
	// pack everyone above 
	y1 := cr.Min.Y
	for j := 0; j < windex; j++ {
		v = c.w[j]
		r := v.r
		r.Min.Y = y1
		r.Max.Y = y1 + v.tagtop.Dy()
		if nl[j] != 0 {
			r.Max.Y += 1 + nl[j]*v.body.fr.Font.DefaultHeight()
		}
		r.Min.Y = v.Resize(r, c.safe, false)
		r.Max.Y += display.ScaleSize(Border)
		display.ScreenImage.Draw(r, display.Black, nil, image.ZP)
		y1 = r.Max.Y
	}
	// scan to see new size of everyone below 
	y2 := c.r.Max.Y
	for j := c.nw() - 1; j > windex; j-- {
		v = c.w[j]
		r := v.r
		r.Min.Y = y2 - v.tagtop.Dy()
		if nl[j] != 0 {
			r.Min.Y -= 1 + nl[j]*v.body.fr.Font.DefaultHeight()
		}
		r.Min.Y -= display.ScaleSize(Border)
		ny[j] = r.Min.Y
		y2 = r.Min.Y
	}
	// compute new size of window 
	r := w.r
	r.Min.Y = y1
	r.Max.Y = y2
	h := w.body.fr.Font.DefaultHeight() // TODO(flux) Is this the right frame font height to use?
	if r.Dy() < w.tagtop.Dy()+1+h+display.ScaleSize(Border) {
		r.Max.Y = r.Min.Y + w.tagtop.Dy() + 1 + h + display.ScaleSize(Border)
	}
	// draw window 
	r.Max.Y = w.Resize(r, c.safe, true)
	if windex < c.nw()-1 {
		r.Min.Y = r.Max.Y
		r.Max.Y += display.ScaleSize(Border)
		display.ScreenImage.Draw(r, display.Black, nil, image.ZP)
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
			r.Max.Y += 1 + nl[j]*v.body.fr.Font.DefaultHeight()
		}
		y1 = v.Resize(r, c.safe, j == c.nw()-1)
		if j < c.nw()-1 { // no border on last window 
			r.Min.Y = y1
			r.Max.Y += display.ScaleSize(Border)
			display.ScreenImage.Draw(r, display.Black, nil, image.ZP)
			y1 = r.Max.Y
		}
	}
	c.safe = true
	w.MouseBut()
}

func (c *Column) DragWin(w *Window, but int) {

var (
	r image.Rectangle
	i, b int
	p, op image.Point
	v *Window
	nc *Column
)
	clearmouse();
	// setcursor(mousectl, &boxcursor); // TODO(flux) - No SetCursor in draw!
	b = mouse.Buttons;
	op = mouse.Point;
	for(mouse.Buttons == b) {
		mousectl.Read()
	}
	// setcursor(mousectl, nil);// TODO(flux) - No SetCursor in draw!
	if mouse.Buttons != 0{
		for(mouse.Buttons!=0){
			mousectl.Read()
		}
		return;
	}

	for _, win := range c.w {
		if win == w {
			goto Found;
		}
	}
	acmeerror("can't find window", nil);

  Found:
	if w.tagexpand 	{ // force recomputation of window tag size 
		w.taglines = 1;
	}
	p = mouse.Point;
	if abs(p.X-op.X)<5 && abs(p.Y-op.Y)<5 {
		c.Grow(w, but);
		w.MouseBut();
		return;
	}
	// is it a flick to the right? 
	if abs(p.Y-op.Y)<10 && p.X>op.X+30 && c.row.WhichCol(p)==c {
		p.X = op.X+w.r.Dx();	// yes: toss to next column 
	}
	nc = c.row.WhichCol(p);
	if nc!=nil && nc!=c {
		c.Close(w, false);
		nc.Add(w, nil, p.Y);
		w.MouseBut();
		return;
	}
	if i==0 && len(c.w)==1 {
		return;			// can't do it 
	}
	if (i>0 && p.Y<c.w[i-1].r.Min.Y) || (i<len(c.w)-1 && p.Y>w.r.Max.Y || (i==0 && p.Y>w.r.Max.Y)){
		// shuffle 
		c.Close(w, false);
		c.Add(w, nil, p.Y);
		w.MouseBut();
		return;
	}
	if i == 0 {
		return;
	}
	v = c.w[i-1];
	if p.Y < v.tagtop.Max.Y {
		p.Y = v.tagtop.Max.Y;
	}
	if(p.Y > w.r.Max.Y-w.tagtop.Dy()-display.ScaleSize(Border)) {
		p.Y = w.r.Max.Y-w.tagtop.Dy()-display.ScaleSize(Border);
	}
	if r.Max.Y > v.body.fr.Rect.Min.Y {
		r.Max.Y -= (r.Max.Y-v.body.fr.Rect.Min.Y)%v.body.fr.Font.DefaultHeight();
		if v.body.fr.Rect.Min.Y == v.body.fr.Rect.Max.Y {
			r.Max.Y++
		}
	}
	r.Min.Y = v.Resize(r, c.safe, false);
	r.Max.Y= r.Min.Y+display.ScaleSize(Border);
	display.ScreenImage.Draw(r, display.Black, nil, image.ZP);
	r.Min.Y = r.Max.Y;
	if(i == len(c.w)-1) {
		r.Max.Y = c.r.Max.Y;
	} else {
		r.Max.Y = c.w[i+1].r.Min.Y-display.ScaleSize(Border);
	}
	w.Resize(r, c.safe, true);
	c.safe = true;
	w.MouseBut();
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
			if p.X >= w.body.scrollr.Max.X && p.Y >= w.body.fr.Rect.Max.Y {
				return nil
			}
			return &w.body
		}
	}
	return nil
}

func (c *Column) Clean() bool {
	clean := true;
	for _, w := range c.w {
		clean= clean && w.Clean(true);
	}
	return clean;
}
