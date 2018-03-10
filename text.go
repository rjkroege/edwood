package main

import (
	"crypto/sha1"
	"fmt"
	"image"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/paul-lalonde/acme/frame"
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

type TextKind byte
const (
	Columntag = iota
	Rowtag
	Tag
	Body
)

// Text is a view onto a buffer, managing a frame.
// Files have possible multiple texts corresponding to clones.
type Text struct {
	file *File
	fr 	*frame.Frame
	font *draw.Font
	org     uint		// Origin of the frame withing the buffer
	q0      uint
	q1      uint
	what    TextKind
	tabstop int
	w       *Window
	scrollr image.Rectangle
	lastsr  image.Rectangle
	all     image.Rectangle
	row     *Row
	col     *Column

	iq1         uint
	eq0         int
	cq0         uint
	ncache      int
	ncachealloc int
	cache       []rune
	nofill      bool
	needundo    bool
}

func (t *Text)Init(f *File, r image.Rectangle, rf *draw.Font, cols [frame.NumColours]*draw.Image) *Text {
	if t == nil {
		t = new(Text)
	}
	t.file = f
	t.all = r
	t.scrollr = r
	t.scrollr.Max.X = r.Min.X + display.ScaleSize(Scrollwid)
	t.lastsr = nullrect
	r.Min.X += display.ScaleSize(Scrollwid) + display.ScaleSize(Scrollgap)
	t.eq0 = math.MaxInt64
	t.ncache = 0
	t.font = rf
	t.tabstop = int(maxtab)
	t.fr = frame.NewFrame( r, rf, display.ScreenImage, cols)
	t.Redraw(r, rf, display.ScreenImage, -1)
	return t
}

func (t *Text) Redraw(r image.Rectangle, f *draw.Font, b *draw.Image, odx int) {
	t.fr.Init(r, f, b, t.fr.Cols)
	rr := t.fr.Rect;
	rr.Min.X -= display.ScaleSize(Scrollwid+Scrollgap)	/* back fill to scroll bar */
	if !t.fr.NoRedraw {
		t.fr.Background.Draw(rr, t.fr.Cols[frame.ColBack], nil, image.ZP)
	}
	/* use no wider than 3-space tabs in a directory */
	maxt := int(maxtab)
	if t.what == Body {
		if(t.w.isdir) {
			maxt = min(TABDIR, int(maxtab))
		} else {
			maxt = t.tabstop
		}
	}
	t.fr.MaxTab = maxt*f.StringWidth("0")
	if t.what==Body && t.w.isdir && odx!=t.all.Dx() {
		if t.fr.MaxLines > 0 {
			t.Reset()
			t.Columnate(t.w.dirnames,  t.w.widths)
			t.Show(0, 0, false)
		}
	}else{
		t.Fill()
		t.SetSelect(t.q0, t.q1)
	}
}

func (t *Text) Resize(r image.Rectangle, keepextra bool) int {
	if r.Dy() <= 0 {
		r.Max.Y = r.Min.Y
	} else {
		if !keepextra {
			r.Max.Y -= r.Dy()%t.fr.Font.DefaultHeight()
		}
	}
	odx := t.all.Dx()
	t.all = r;
	t.scrollr = r;
	t.scrollr.Max.X = r.Min.X+Scrollwid
	t.lastsr = image.ZR
	r.Min.X += display.ScaleSize(Scrollwid+Scrollgap)
	t.fr.Clear(false)
	t.Redraw(r, t.fr.Font.Impl(), t.fr.Background, odx)
	if keepextra && t.fr.Rect.Max.Y < t.all.Max.Y /* && !t.fr.NoRedraw */ {
		/* draw background in bottom fringe of window */
		r.Min.X -= display.ScaleSize(Scrollgap)
		r.Min.Y = t.fr.Rect.Max.Y
		r.Max.Y = t.all.Max.Y
		display.ScreenImage.Draw(r, t.fr.Cols[frame.ColBack], nil, image.ZP)
	}
	return t.all.Max.Y
}

func (t *Text) Close() {
Unimpl()
}

func (t *Text) Columnate(names []string, widths []int) {

	var colw, mint, maxt, ncol, nrow int
	q1 := uint(0)
	Lnl := []rune("\n")
	Ltab := []rune("\t")

	if len(t.file.text) > 1 {
		return
	}
	mint = t.fr.Font.StringWidth("0")
	/* go for narrower tabs if set more than 3 wide */
	t.fr.MaxTab = min(int(maxtab), TABDIR)*mint
	maxt = t.fr.MaxTab
	for _, w := range widths {
		if maxt-w%maxt < mint || w%maxt==0 {
			w += mint
		}
		if w % maxt != 0 {
			w += maxt-(w%maxt)
		}
		if w > colw {
			colw = w
		}
	}
	if colw == 0 {
		ncol = 1
	} else {
		ncol = max(1, t.fr.Rect.Dx()/colw)
	}
	nrow = (len(names)+ncol-1)/ncol

	q1 = 0
	for i:=0; i<nrow; i++ {
		for j:=i; j<len(names); j+=nrow {
			dl := names[j]
			t.file.Insert(q1, []rune(dl))
			q1 += uint(len(dl))
			if j+nrow >= len(names) {
				break
			}
			w := widths[j];
			if maxt-w%maxt < mint {
				t.file.Insert(q1, Ltab)
				q1++
				w += mint
			}
			for {
				t.file.Insert(q1, Ltab)
				q1++
				w += maxt-(w%maxt)
				if !(w < colw) {
					break
				}
			}
		}
		t.file.Insert(q1, Lnl)
		q1++
	}
}

func (t *Text) Load(q0 uint, filename string, setqid bool) (nread uint, err error) {
	if t.ncache!=0 || t.file.b.nc() > 0 || t.w==nil || t!=&t.w.body {
		panic("text.load")
	}
	if t.w.isdir && t.file.name==""{
		warning(nil, "empty directory name")
		return 0, fmt.Errorf("empty directory name")
	}
	if ismtpt(filename){
		warning(nil, "will not open self mount point %s\n", filename)
		return 0, fmt.Errorf("will not open self mount point %s\n", filename)
	}
	fd, err := os.Open(filename);
	if err != nil{
		warning(nil, "can't open %s: %v\n", filename, err)
		return 0, fmt.Errorf("can't open %s: %v\n", filename, err)
	}
	defer fd.Close()
	d, err := fd.Stat()
	if err != nil{
		warning(nil, "can't fstat %s: %v\n", filename, err)
		return 0, fmt.Errorf("can't fstat %s: %v\n", filename, err)
	}

	var count uint
	q1 := uint(0)
	hasNulls := false
	var sha1 [sha1.Size]byte
	if d.IsDir() {
		/* this is checked in get() but it's possible the file changed underfoot */
		if len(t.file.text) > 1{
			warning(nil, "%s is a directory; can't read with multiple windows on it\n", filename)
			return 0, fmt.Errorf("%s is a directory; can't read with multiple windows on it\n", filename)
		}
		t.w.isdir = true;
		t.w.filemenu = false;
		// TODO(flux): Find all '/' and replace with filepath.Separator properly
		if len(t.file.name) > 0 && !strings.HasSuffix(t.file.name, "/") {
			t.file.name = t.file.name + "/"
			t.w.SetName(t.file.name);
		}
		dirNames, err := fd.Readdirnames(0)
		if err != nil {
			warning(nil, "failed to Readdirnames: %s\n", filename)
			return 0, fmt.Errorf("failed to Readdirnames: %s\n", filename)
		}
		for i, dn := range dirNames {
			f, err := os.Open(dn)
			if err != nil{
				warning(nil, "can't open %s: %v\n", dn, err)
			}
			s, err  := f.Stat()
			if err != nil{
				warning(nil, "can't fstat %s: %r\n", dn, err)
			} else {
				if s.IsDir() {
					dirNames[i] = dn+"/"
				}
			}
		}
		sort.Strings(dirNames)
		widths := make([]int, len(dirNames))
		for i, s := range dirNames {
			widths[i] = t.fr.Font.StringWidth(s)
		}
		t.Columnate(dirNames, widths)
		t.w.dirnames = dirNames
		t.w.widths = widths
		q1 = t.file.b.nc()
	}else{ 
		t.w.isdir = false
		t.w.filemenu = true
		count, sha1, hasNulls, err = t.file.Load(q0, fd)
		if err != nil {
			warning(nil, "Error reading file %s: %v", filename, err)
			return 0, fmt.Errorf("Error reading file %s: %v", filename, err)
		}
		q1 = q0 + count
	}
	if setqid{
		if q0 == 0 {
			t.file.sha1 = sha1
		}
		//t.file.dev = d.dev;
		t.file.mtime = d.ModTime().UnixNano()
		t.file.qidpath = d.Name() // TODO(flux): Gross hack to use filename as unique ID of file.
	}
	fd.Close()
	n := q1-q0
	if q0 < t.org { // TODO(flux) I don't understand this test, moving origin of frame past the text.
		t.org += n
	} else {
		if q0 <= t.org+uint(t.fr.NChars) { // Text is within the window, put it there.
			t.fr.Insert(t.file.b[q0:q0+n], int(q0-t.org))
		}
	}
	// For each clone, redraw
	for _, u := range t.file.text {
		if u != t { // Skip the one we just redrew
			if u.org > u.file.b.nc() {	/* will be 0 because of reset(), but safety first */
				u.org = 0;
			}
			u.Resize(u.all, true)
			u.Backnl(u.org, 0)	/* go to beginning of line */
		}
		u.SetSelect(q0, q0);
	}
	if hasNulls {
		warning(nil, "%s: NUL bytes elided\n", filename);
	}
	return q1-q0, nil

}

func (t *Text) Backnl(p, n uint) uint {
Unimpl()
	return 0
}

func (t *Text) BsInsert(q0 uint, r []rune, n uint, tofile bool, nrp *int) uint {
Unimpl()
	return 0
}

func (t *Text) Insert(q0 uint, r []rune, tofile bool) {
	if tofile && t.ncache != 0 {
		panic("text.insert")
	}
	if len(r) == 0 {
		return
	}
	if tofile  {
		t.file.Insert(q0, r);
		if t.what == Body  {
			t.w.dirty = true
			t.w.utflastqid = -1
		}
		if len(t.file.text) > 1 {
			for _, u := range t.file.text {
				if u != t  {
					u.w.dirty = true	/* always a body */
					u.Insert(q0, r, false)
					u.SetSelect(u.q0, u.q1)
					u.ScrDraw()
				}
			}
		}		
	}
	n := uint(len(r))
	if q0 < t.iq1 {
		t.iq1 += n
	}
	if q0 < t.q1 {
		t.q1 += n
	}
	if q0 < t.q0 {
		t.q0 += n
	}
	if q0 < t.org {
		t.org += n
	} else {
		 if q0 <= t.org+uint(t.fr.NChars)  {
			t.fr.Insert(r[:n], int(q0-t.org))
		}
	}
	if t.w != nil {
		c := 'i'
		if t.what == Body {
			c = 'I'
		}
		if n <= EVENTSIZE  {
			t.w.Event("%c%d %d 0 %d %.*S\n", c, q0, q0+n, n, n, r)
		} else {
			t.w.Event("%c%d %d 0 0 \n", c, q0, q0+n, n)
		}
	}
}

func (t *Text)TypeCommit(){
	if t.w != nil {
		t.w.Commit(t)
	} else {
		t.Commit(true)
	}
}

func (t *Text) Fill() {
	if t.fr.LastLineFull != 0 || t.nofill {
		return
	}
	if(t.ncache > 0) {
		t.TypeCommit()
	}
	
	nl := t.fr.MaxLines-t.fr.NLines;
	lines := runesplitN(t.file.b[t.org+uint(t.fr.NChars):], []rune("\n"), nl)
	for _, s := range lines {
		t.fr.Insert(s, t.fr.NChars);
		if t.fr.LastLineFull != 0 {
			break
		}
	}
}

func (t *Text) Delete(q0, q1 uint, tofile bool) {
	if tofile && t.ncache != 0 {
		panic("text.delete")
	}
	n := q1-q0
	if n == 0 {
		return
	}
	if tofile {
		t.file.Delete(q0, q1);
		if t.what == Body {
			t.w.dirty = true
			t.w.utflastqid = -1
		}
		if len(t.file.text) > 1 {
			for _, u := range t.file.text {
				if u != t {
					u.w.dirty = true	/* always a body */
					u.Delete(q0, q1, false)
					u.SetSelect(u.q0, u.q1)
					u.ScrDraw()
				}
			}
		}
	}
	if q0 < t.iq1 {
		t.iq1 -= minu(n, t.iq1-q0);
	}
	if q0 < t.q0 {
		t.q0 -= minu(n, t.q0-q0);
	}
	if q0 < t.q1 {
		t.q1 -= minu(n, t.q1-q0);
	}
	if q1 <= t.org {
		t.org -= n;
	} else if q0 < t.org+uint(t.fr.NChars) {
		p1 := q1 - t.org
		p0 := uint(0)
		if p1 > uint(t.fr.NChars)  {
			p1 = uint(t.fr.NChars)
		}
		if q0 < t.org {
			t.org = q0
			p0 = 0
		}else {
			p0 = q0 - t.org
		}
		t.fr.Delete(int(p0), int(p1))
		t.Fill()
	}
	if t.w != nil {
		c := 'd'
		if t.what == Body  {
			c = 'D'
		}
		t.w.Event("%c%d %d 0 0 \n", c, q0, q1)
	}
}

func (t *Text) Constrain(q0, q1 uint, p0, p1 *uint) {
	*p0 = minu(q0, t.file.b.nc())
	*p1 = minu(q1, t.file.b.nc())
}

func (t *Text) ReadRune(q uint) rune {
Unimpl()
	return ' '
}

func (t *Text) BsWidth(c rune) int {
Unimpl()
	return 0
}

func (t *Text) FileWidth(q0 uint, oneelement int) int {
Unimpl()
	return 0
}

func (t *Text) Complete() []rune {
Unimpl()
	return nil
}

func (t *Text) Type(r rune) {
Unimpl()

}

func (t *Text) Commit(tofile bool) {
	if(t.ncache == 0) {
		return
	}
	if tofile {
		t.file.Insert(t.cq0, t.cache)
	}
	if t.what == Body {
		t.w.dirty = true
		t.w.utflastqid = -1
	}
	t.ncache = 0
}

func (t *Text) FrameScroll(dl int) {
Unimpl()

}

func (t *Text) Select() {
Unimpl()

}

func (t *Text) Show(q0, q1 uint, doselect bool) {
Unimpl()

}

func (t *Text) SetSelect(q0, q1 uint) {
Unimpl()

}

func (t *Text) Select23(q0, q1 *uint, high *draw.Image, mask int) int {
Unimpl()
	return 0
}

func (t *Text) Select2()(q0, q1 uint, tp *Text, ret bool) {
Unimpl()
	return 0,0,nil,false
}


func (t *Text) Select3()(q0, q1 uint, r bool) {
Unimpl()
	return 0,0,false
}

func (t *Text) DoubleClick(q0, q1 *uint) {
Unimpl()

}

func (t *Text) ClickMatch(cl, cr, dir int, q *uint) int {
Unimpl()
	return 0
}

func (t *Text) ishtmlstart(q uint, q1 *uint) bool {
Unimpl()
	return false
}

func (t *Text) ishtmlend(q uint, q0 *uint) bool {
Unimpl()
	return false
}

func (t *Text) ClickHTMLMatch(q0, q1 *uint) int {
Unimpl()
	return 0
}

func (t *Text) BackNL(p, n uint) uint {
Unimpl()
	return 0
}

func (t *Text) SetOrigin(org uint, exact int) {
Unimpl()

}

func (t *Text) Reset() {
Unimpl()

}
