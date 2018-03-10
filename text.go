package main

import (
	"crypto/sha1"
	"fmt"
	"image"
	"math"
	"os"
	"sort"
	"strings"

	"9fans.net/go/draw"
	"github.com/paul-lalonde/acme/frame"
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
	file    *File
	fr      *frame.Frame
	font    *draw.Font
	org     uint // Origin of the frame withing the buffer
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

func (t *Text) Init(f *File, r image.Rectangle, rf *draw.Font, cols [frame.NumColours]*draw.Image) *Text {
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
	t.fr = frame.NewFrame(r, rf, display.ScreenImage, cols)
	t.Redraw(r, rf, display.ScreenImage, -1)
	return t
}

func (t *Text) Redraw(r image.Rectangle, f *draw.Font, b *draw.Image, odx int) {
	t.fr.Init(r, f, b, t.fr.Cols)
	rr := t.fr.Rect
	rr.Min.X -= display.ScaleSize(Scrollwid + Scrollgap) /* back fill to scroll bar */
	if !t.fr.NoRedraw {
		t.fr.Background.Draw(rr, t.fr.Cols[frame.ColBack], nil, image.ZP)
	}
	/* use no wider than 3-space tabs in a directory */
	maxt := int(maxtab)
	if t.what == Body {
		if t.w.isdir {
			maxt = min(TABDIR, int(maxtab))
		} else {
			maxt = t.tabstop
		}
	}
	t.fr.MaxTab = maxt * f.StringWidth("0")
	if t.what == Body && t.w.isdir && odx != t.all.Dx() {
		if t.fr.MaxLines > 0 {
			t.Reset()
			t.Columnate(t.w.dirnames, t.w.widths)
			t.Show(0, 0, false)
		}
	} else {
		t.Fill()
		t.SetSelect(t.q0, t.q1)
	}
}

func (t *Text) Resize(r image.Rectangle, keepextra bool) int {
	if r.Dy() <= 0 {
		r.Max.Y = r.Min.Y
	} else {
		if !keepextra {
			r.Max.Y -= r.Dy() % t.fr.Font.DefaultHeight()
		}
	}
	odx := t.all.Dx()
	t.all = r
	t.scrollr = r
	t.scrollr.Max.X = r.Min.X + Scrollwid
	t.lastsr = image.ZR
	r.Min.X += display.ScaleSize(Scrollwid + Scrollgap)
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
	t.fr.MaxTab = min(int(maxtab), TABDIR) * mint
	maxt = t.fr.MaxTab
	for _, w := range widths {
		if maxt-w%maxt < mint || w%maxt == 0 {
			w += mint
		}
		if w%maxt != 0 {
			w += maxt - (w % maxt)
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
	nrow = (len(names) + ncol - 1) / ncol

	q1 = 0
	for i := 0; i < nrow; i++ {
		for j := i; j < len(names); j += nrow {
			dl := names[j]
			t.file.Insert(q1, []rune(dl))
			q1 += uint(len(dl))
			if j+nrow >= len(names) {
				break
			}
			w := widths[j]
			if maxt-w%maxt < mint {
				t.file.Insert(q1, Ltab)
				q1++
				w += mint
			}
			for {
				t.file.Insert(q1, Ltab)
				q1++
				w += maxt - (w % maxt)
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
	if t.ncache != 0 || t.file.b.nc() > 0 || t.w == nil || t != &t.w.body {
		panic("text.load")
	}
	if t.w.isdir && t.file.name == "" {
		warning(nil, "empty directory name")
		return 0, fmt.Errorf("empty directory name")
	}
	if ismtpt(filename) {
		warning(nil, "will not open self mount point %s\n", filename)
		return 0, fmt.Errorf("will not open self mount point %s\n", filename)
	}
	fd, err := os.Open(filename)
	if err != nil {
		warning(nil, "can't open %s: %v\n", filename, err)
		return 0, fmt.Errorf("can't open %s: %v\n", filename, err)
	}
	defer fd.Close()
	d, err := fd.Stat()
	if err != nil {
		warning(nil, "can't fstat %s: %v\n", filename, err)
		return 0, fmt.Errorf("can't fstat %s: %v\n", filename, err)
	}

	var count uint
	q1 := uint(0)
	hasNulls := false
	var sha1 [sha1.Size]byte
	if d.IsDir() {
		/* this is checked in get() but it's possible the file changed underfoot */
		if len(t.file.text) > 1 {
			warning(nil, "%s is a directory; can't read with multiple windows on it\n", filename)
			return 0, fmt.Errorf("%s is a directory; can't read with multiple windows on it\n", filename)
		}
		t.w.isdir = true
		t.w.filemenu = false
		// TODO(flux): Find all '/' and replace with filepath.Separator properly
		if len(t.file.name) > 0 && !strings.HasSuffix(t.file.name, "/") {
			t.file.name = t.file.name + "/"
			t.w.SetName(t.file.name)
		}
		dirNames, err := fd.Readdirnames(0)
		if err != nil {
			warning(nil, "failed to Readdirnames: %s\n", filename)
			return 0, fmt.Errorf("failed to Readdirnames: %s\n", filename)
		}
		for i, dn := range dirNames {
			f, err := os.Open(dn)
			if err != nil {
				warning(nil, "can't open %s: %v\n", dn, err)
			}
			s, err := f.Stat()
			if err != nil {
				warning(nil, "can't fstat %s: %r\n", dn, err)
			} else {
				if s.IsDir() {
					dirNames[i] = dn + "/"
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
	} else {
		t.w.isdir = false
		t.w.filemenu = true
		count, sha1, hasNulls, err = t.file.Load(q0, fd)
		if err != nil {
			warning(nil, "Error reading file %s: %v", filename, err)
			return 0, fmt.Errorf("Error reading file %s: %v", filename, err)
		}
		q1 = q0 + count
	}
	if setqid {
		if q0 == 0 {
			t.file.sha1 = sha1
		}
		//t.file.dev = d.dev;
		t.file.mtime = d.ModTime().UnixNano()
		t.file.qidpath = d.Name() // TODO(flux): Gross hack to use filename as unique ID of file.
	}
	fd.Close()
	n := q1 - q0
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
			if u.org > u.file.b.nc() { /* will be 0 because of reset(), but safety first */
				u.org = 0
			}
			u.Resize(u.all, true)
			u.Backnl(u.org, 0) /* go to beginning of line */
		}
		u.SetSelect(q0, q0)
	}
	if hasNulls {
		warning(nil, "%s: NUL bytes elided\n", filename)
	}
	return q1 - q0, nil

}

func (t *Text) Backnl(p uint, n uint) uint {
	/* look for start of this line if n==0 */
	if n==0 && p>0 && t.ReadRune(p-1)!='\n' {
		n = 1;
	}
	i := n;
	for  (i>0 && p>0){
		i--
		p--;	/* it's at a newline now; back over it */
		if p == 0 {
			break;
		}
		/* at 128 chars, call it a line anyway */
		for j:=128; j>0 && p>0; p-- {
			j--
			if t.ReadRune(p-1)=='\n' {
				break;
			}
		}
	}
	return p;
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
	if tofile {
		t.file.Insert(q0, r)
		if t.what == Body {
			t.w.dirty = true
			t.w.utflastqid = -1
		}
		if len(t.file.text) > 1 {
			for _, u := range t.file.text {
				if u != t {
					u.w.dirty = true /* always a body */
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
		if q0 <= t.org+uint(t.fr.NChars) {
			t.fr.Insert(r[:n], int(q0-t.org))
		}
	}
	if t.w != nil {
		c := 'i'
		if t.what == Body {
			c = 'I'
		}
		if n <= EVENTSIZE {
			t.w.Event("%c%d %d 0 %d %.*S\n", c, q0, q0+n, n, n, r)
		} else {
			t.w.Event("%c%d %d 0 0 \n", c, q0, q0+n, n)
		}
	}
}

func (t *Text) TypeCommit() {
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
	if t.ncache > 0 {
		t.TypeCommit()
	}

	nl := t.fr.MaxLines - t.fr.NLines
	lines := runesplitN(t.file.b[t.org+uint(t.fr.NChars):], []rune("\n"), nl)
	for _, s := range lines {
		t.fr.Insert(s, t.fr.NChars)
		if t.fr.LastLineFull != 0 {
			break
		}
	}
}

func (t *Text) Delete(q0, q1 uint, tofile bool) {
	if tofile && t.ncache != 0 {
		panic("text.delete")
	}
	n := q1 - q0
	if n == 0 {
		return
	}
	if tofile {
		t.file.Delete(q0, q1)
		if t.what == Body {
			t.w.dirty = true
			t.w.utflastqid = -1
		}
		if len(t.file.text) > 1 {
			for _, u := range t.file.text {
				if u != t {
					u.w.dirty = true /* always a body */
					u.Delete(q0, q1, false)
					u.SetSelect(u.q0, u.q1)
					u.ScrDraw()
				}
			}
		}
	}
	if q0 < t.iq1 {
		t.iq1 -= minu(n, t.iq1-q0)
	}
	if q0 < t.q0 {
		t.q0 -= minu(n, t.q0-q0)
	}
	if q0 < t.q1 {
		t.q1 -= minu(n, t.q1-q0)
	}
	if q1 <= t.org {
		t.org -= n
	} else if q0 < t.org+uint(t.fr.NChars) {
		p1 := q1 - t.org
		p0 := uint(0)
		if p1 > uint(t.fr.NChars) {
			p1 = uint(t.fr.NChars)
		}
		if q0 < t.org {
			t.org = q0
			p0 = 0
		} else {
			p0 = q0 - t.org
		}
		t.fr.Delete(int(p0), int(p1))
		t.Fill()
	}
	if t.w != nil {
		c := 'd'
		if t.what == Body {
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
	if t.cq0<=q && q<t.cq0+uint(t.ncache)  {
		return t.cache[q-t.cq0];
	} else {
		return t.file.b.Read(q, 1)[0];
	}
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
	if t.ncache == 0 {
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
	var q0 uint
	if dl == 0 {
		ScrSleep(100);
		return;
	}
	if dl < 0 {
		q0 = t.Backnl(t.org, uint(-dl));
		if selectq > t.org+uint(t.fr.P0) {
			t.SetSelect(t.org+uint(t.fr.P0), selectq);
		} else {
			t.SetSelect(selectq, t.org+uint(t.fr.P0));
		}
	}else{
		if t.org+uint(t.fr.NChars) == t.file.b.nc() {
			return;
		}
		q0 = t.org+uint(t.fr.Charofpt(image.Pt(t.fr.Rect.Min.X, t.fr.Rect.Min.Y+dl*t.fr.Font.Impl().Height)));
		if selectq > t.org+uint(t.fr.P1) {
			t.SetSelect(t.org+uint(t.fr.P1), selectq);
		} else {
			t.SetSelect(selectq, t.org+uint(t.fr.P1));
		}
	}
	t.SetOrigin(q0, true);
}

var (
	clicktext *Text
	clickmsec uint32
	selecttext *Text
	selectq uint
)

/*
 * called from frame library
 */
func framescroll(f *frame.Frame, dl int) {
	if f != selecttext.fr {
		panic("frameselect not right frame");
	}
	selecttext.FrameScroll(dl);
}

func (t *Text) Select() {
	const (
		None = iota
		Cut
		Paste
	)

	selecttext = t
	/*
	 * To have double-clicking and chording, we double-click
	 * immediately if it might make sense.
	 */
	b := mouse.Buttons;
	q0 := t.q0
	q1 := t.q1
	selectq = t.org+uint(t.fr.Charofpt(mouse.Point))
fmt.Printf("Text.Select: mouse.Msec %v, clickmsec %v\n", mouse.Msec, clickmsec)
fmt.Printf("clicktext==t %v, (q0==q1 && selectq==q0): %v", clicktext==t,q0==q1 && selectq==q0)
	if (clicktext==t && mouse.Msec-uint32(clickmsec)<500) && (q0==q1 && selectq==q0) {
		q0, q1 = t.DoubleClick(q0);
fmt.Printf("Text.Select: DoubleClick returned %d, %d\n", q0,q1)
		t.SetSelect(q0, q1)
		display.Flush()
		x := mouse.Point.X
		y := mouse.Point.Y
		/* stay here until something interesting happens */
		for {
			mousectl.Read()
		 	if !(mouse.Buttons==b && abs(mouse.Point.X-x)<3 && abs(mouse.Point.Y-y)<3) {
				break
			}
		}
		mouse.Point.X = x;	/* in case we're calling frselect */
		mouse.Point.Y = y;
		q0 = t.q0;	/* may have changed */
		q1 = t.q1;
		selectq = q0;
	}
	if mouse.Buttons == b {
		t.fr.Scroll = framescroll;
		t.fr.Select(mousectl);
		/* horrible botch: while asleep, may have lost selection altogether */
		if selectq > t.file.b.nc() {
			selectq = t.org + uint(t.fr.P0);
		}
		t.fr.Scroll = nil;
		if selectq < t.org {
			q0 = selectq;
		} else {
			q0 = t.org + uint(t.fr.P0);
		}
		if selectq > t.org+uint(t.fr.NChars) {
			q1 = selectq;
		} else {
			q1 = t.org+uint(t.fr.P1);
		}
	}
	if q0 == q1 {
		if q0==t.q0 && clicktext==t && mouse.Msec-uint32(clickmsec)<500 {
			q0, q1 = t.DoubleClick(q0);
			clicktext = nil;
		}else{
			clicktext = t;
			clickmsec = mouse.Msec;
		}
	}else {
		clicktext = nil;
	}
	t.SetSelect(q0, q1);
	display.Flush()
	state := None;	/* what we've done; undo when possible */
	for (mouse.Buttons != 0){
		mouse.Msec = 0;
		b := mouse.Buttons;
		if (b&1)!=0 && (b&6)!=0 {
			if state==None && t.what==Body {
				seq++;
				t.w.body.file.Mark();
			}
			if b & 2 != 0{
				if state==Paste && t.what==Body {
					t.w.Undo(true);
					t.SetSelect(q0, t.q1);
					state = None;
				} else {
					if state != Cut {
						cut(t, t, nil, true, true, nil, 0);
						state = Cut;
					}
				}
			}else{
				if state==Cut && t.what==Body {
					t.w.Undo(true);
					t.SetSelect(q0, t.q1);
					state = None;
				} else {
					if state != Paste {
						paste(t, t, nil, true, false, nil, 0);
						state = Paste;
					}
				}
			}
			t.ScrDraw();
			clearmouse();
		}
		display.Flush()
		for (mouse.Buttons == b) {
			mousectl.Read()
		}
		clicktext = nil;
	}
}

func (t *Text) Show(q0, q1 uint, doselect bool) {
	Unimpl()

}

func (t* Text) ReadC(q uint)(r rune) {
	if t.cq0<=q && q<t.cq0+uint(t.ncache) {
		r = t.cache[q-t.cq0]
	} else {
		r = t.file.b.Read(q, 1)[0]
	}
	return r;

}

func (t *Text) SetSelect(q0, q1 uint) {
	/* uint(t.fr.P0) and uint(t.fr.P1) are always right; t.q0 and t.q1 may be off */
	t.q0 = q0;
	t.q1 = q1;
	/* compute desired p0,p1 from q0,q1 */
	p0 := q0-t.org;
	p1 := q1-t.org;
	ticked := true;
	if p0 < 0 {
		ticked = false;
		p0 = 0;
	}
	if p1 < 0 {
		p1 = 0;
	}
	if p0 > uint(t.fr.NChars) {
		p0 = uint(t.fr.NChars);
	}
	if p1 > uint(t.fr.NChars) {
		ticked = false;
		p1 = uint(t.fr.NChars);
	}
	if p0==uint(t.fr.P0) && p1==uint(t.fr.P1) {
		if p0 == p1 && ticked != t.fr.Ticked {
			t.fr.Tick(t.fr.Ptofchar(int(p0)), ticked);
		}
		return;
	}
	if p0 > p1 {
		panic(fmt.Sprintf("acme: textsetselect p0=%d p1=%d q0=%ud q1=%ud t.org=%d nchars=%d", p0, p1, q0, q1, t.org, t.fr.NChars));
	}
	/* screen disagrees with desired selection */
	if uint(t.fr.P1)<=p0 || p1<=uint(t.fr.P0) || p0==p1 || uint(t.fr.P1)==uint(t.fr.P0) {
		/* no overlap or too easy to bother trying */
		t.fr.DrawSel(t.fr.Ptofchar(t.fr.P0), t.fr.P0, t.fr.P1, false);
		if p0 != p1 || ticked {
			t.fr.DrawSel(t.fr.Ptofchar(int(p0)), int(p0), int(p1), true);
		}
		goto Return;
	}
	/* overlap; avoid unnecessary painting */
	if p0 < uint(t.fr.P0) {
		/* extend selection backwards */
		t.fr.DrawSel(t.fr.Ptofchar(int(p0)), int(p0), t.fr.P0, true);
	}else {
		if p0 > uint(t.fr.P0) {
			/* trim first part of selection */
			t.fr.DrawSel(t.fr.Ptofchar(t.fr.P0), t.fr.P0, int(p0), false);
		}
	}
	if p1 > uint(t.fr.P1) {
		/* extend selection forwards */
		t.fr.DrawSel(t.fr.Ptofchar(t.fr.P1), t.fr.P1, int(p1), true);
	}else if p1 < uint(t.fr.P1) {
		/* trim last part of selection */
		t.fr.DrawSel(t.fr.Ptofchar(int(p1)), int(p1), t.fr.P1, false);
	}

    Return:
	t.fr.P0 = int(p0);
	t.fr.P1 = int(p1);

}

func selrestore(f *frame.Frame, pt0 image.Point, p0, p1 uint) {

	if p1<=uint(f.P0) || p0>=uint(f.P1) {
		/* no overlap */
		f.DrawSel0(pt0, int(p0), int(p1), f.Cols[frame.ColBack], f.Cols[frame.ColText]);
		return;
	}
	if p0>=uint(f.P0) && p1<=uint(f.P1) {
		/* entirely inside */
		f.DrawSel0(pt0, int(p0), int(p1), f.Cols[frame.ColHigh], f.Cols[frame.ColHText]);
		return;
	}

	/* they now are known to overlap */

	/* before selection */
	if p0 < uint(f.P0) {
		f.DrawSel0(pt0, int(p0), f.P0, f.Cols[frame.ColBack], f.Cols[frame.ColText]);
		p0 = uint(f.P0);
		pt0 = f.Ptofchar(int(p0));
	}
	/* after selection */
	if p1 > uint(f.P1) {
		f.DrawSel0(f.Ptofchar(f.P1), f.P1, int(p1), f.Cols[frame.ColBack], f.Cols[frame.ColText]);
		p1 = uint(f.P1);
	}
	/* inside selection */
	f.DrawSel0(pt0, int(p0), int(p1), f.Cols[frame.ColHigh], f.Cols[frame.ColHText]);
}

const (
	DELAY = 2
	MINMOVE = 4
)

// When called, button is down.
func xselect(f *frame.Frame, mc *draw.Mousectl, col *draw.Image)(p0p, p1p uint) {
	mp := mc.Mouse.Point
	b := mc.Mouse.Buttons
	msec := mc.Mouse.Msec

	/* remove tick */
	if f.P0 == f.P1 {
		f.Tick(f.Ptofchar(f.P0), false)
	}
	p0 :=  uint(f.Charofpt(mp))
	p1 := uint(p0)
	pt0 := f.Ptofchar(int(p0))
	pt1 := f.Ptofchar(int(p1))
	reg := 0
	f.Tick(pt0, true)
	for {
		q := uint(f.Charofpt(mc.Mouse.Point))
		if p1 != q {
			if p0 == p1 {
				f.Tick(pt0, false)
			}
			if reg != region(q, p0) {	/* crossed starting point; reset */
				if reg > 0 {
					selrestore(f, pt0, p0, p1);
				} else { 
					if reg < 0 {
						selrestore(f, pt1, p1, p0);
					}
				}
				p1 = p0;
				pt1 = pt0;
				reg = region(q, p0);
				if reg == 0 {
					f.DrawSel0(pt0, int(p0), int(p1), col, display.White)
				}
			}
			qt := f.Ptofchar(int(q));
			if reg > 0 {
				if q > p1 {
					f.DrawSel0(pt1, int(p1), int(q), col, display.White);
				} else {
					if q < p1 {
						selrestore(f, qt, q, p1)
					}
				}
			} else { 
				if reg < 0 {
					if q > p1 {
						selrestore(f, pt1, p1, q);
					} else {
						f.DrawSel0(qt, int(q), int(p1), col, display.White);
					}
				}
			}
			p1 = q;
			pt1 = qt;
		}
		if p0 == p1 {
			f.Tick(pt0, true)
		}
		display.Flush()
		mc.Read()
		if (mc.Mouse.Buttons != b) { break }
	}
	if mc.Mouse.Msec-msec < DELAY && p0!=p1&& abs(mp.X-mc.Mouse.Point.X)<MINMOVE && abs(mp.Y-mc.Mouse.Point.Y)<MINMOVE {
		if reg > 0 {
			selrestore(f, pt0, p0, p1);
		} else {
			if reg < 0 {
				selrestore(f, pt1, p1, p0);
			}
		}
		p1 = p0;
	}
	if p1 < p0 {
		p0, p1 = p1, p0
	}
	pt0 = f.Ptofchar(int(p0));
	if p0 == p1 {
		f.Tick(pt0, false)
	}
	selrestore(f, pt0, p0, p1);
	/* restore tick */
	if f.P0 == f.P1 {
		f.Tick(f.Ptofchar(f.P0), true);
	}
	display.Flush();
	return p0, p1
}

func (t *Text) Select23(high *draw.Image, mask uint)(q0, q1 uint, buts uint) {
	p0, p1 := xselect(t.fr, mousectl, high)
	buts = uint(mousectl.Mouse.Buttons)
	if (buts & mask) == 0 {
		q0 = p0+t.org
		q1 = p1+t.org
	}

	for (mousectl.Mouse.Buttons!=0) {
		mousectl.Read()
	}
	return q0, q1, buts
}

func (t *Text) Select2() (q0, q1 uint, tp *Text, ret bool) {
	q0, q1, buts := t.Select23(but2col, 4)
	if(buts & 4) == 0{
		return q0, q1, nil, false
	}
	if(buts & 1) != 0 {	/* pick up argument */
		return q0, q1, argtext, true
	}
	return q0, q1, nil, true
}

func (t *Text) Select3() (q0, q1 uint, r bool) {
	q0, q1, buts := t.Select23(but3col, 1|2)
	return q0, q1, buts == 0
}

func (t *Text) DoubleClick(inq0 uint) (q0, q1 uint) {
	q0 = inq0
	if q0, q1, ok := t.ClickHTMLMatch(inq0); ok {
		return q0, q1
	}
	var c rune
	for i, l := range left {
		q := inq0;
		r := right[i];
		/* try matching character to left, looking right */
		if q == 0 {
			c = '\n';
		} else {
			c = t.ReadC(q-1)
		}
		p := runestrchr(l, c);
		if p != -1 {
			if q, ok := t.ClickMatch(c, r[p], 1, q); ok  {
				q1 = q
				if c!='\n' {
					q1--
				}
			}
			return;
		}
		/* try matching character to right, looking left */
		if q == t.file.b.nc() {
			c = '\n';
		} else {
			c = t.ReadC(q)
		}
		p = runestrchr(r, c);
		if p != -1 {
			if q, ok := t.ClickMatch(c, l[p], -1, q); ok {
				q1 = inq0
				if (q0<t.file.b.nc() && c=='\n') { q1++ }
				q0 = q;
				if c!='\n' || q!=0 || t.ReadC(0)=='\n' {
					q0++;
				}
			}
			return;
		}
	}
	
	/* try filling out word to right */
	for (q1<t.file.b.nc() && isalnum(t.ReadC(q1))) {
		q1++;
	}
	/* try filling out word to left */
	for (q0>0 && isalnum(t.ReadC(q0-1))) {
		q0--;
	}

	return q0, q1
}

func (t *Text) ClickMatch(cl, cr rune, dir int, inq uint) (q uint, r bool) {
	nest := 1;
	var c rune
	for {
		if dir > 0 {
			if inq == t.file.b.nc() {
				break;
			}
			c = t.ReadC(inq)
			(inq)++;
		}else{
			if inq == 0 {
				break;
			}
			(inq)--;
			c = t.ReadC(inq);
		}
		if c == cr {
			nest--
			if nest==0 {
				return inq, true;
			}
		}else {
			if c == cl {
				nest++;
			}
		}
	}
	return inq, cl=='\n' && nest==1;
}

func (t *Text) ishtmlstart(q uint, q1 *uint) bool {
	Unimpl()
	return false
}

func (t *Text) ishtmlend(q uint, q0 *uint) bool {
	Unimpl()
	return false
}

func (t *Text) ClickHTMLMatch(inq0 uint)(q0, q1 uint, r bool) {
	Unimpl()
	return 0, 0, false
}

func (t *Text) BackNL(p, n uint) uint {
	Unimpl()
	return 0
}

func (t *Text) SetOrigin(org uint, exact bool) {
	Unimpl()

}

func (t *Text) Reset() {
	Unimpl()

}
