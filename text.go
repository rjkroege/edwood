package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"9fans.net/go/draw"
	"github.com/rjkroege/edwood/frame"
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
	display *draw.Display
	file    *File
	fr      frame.Frame
	font    string

	org     int // Origin of the frame within the buffer
	q0      int
	q1      int
	what    TextKind
	tabstop int
	w       *Window
	scrollr image.Rectangle
	lastsr  image.Rectangle
	all     image.Rectangle
	row     *Row
	col     *Column

	iq1         int
	eq0         int
	cq0         int
	ncache      int
	ncachealloc int
	cache       []rune
	nofill      bool
	needundo    bool
}

// getfont is a convenience accessor that gets the draw.Font from the font
// used in this text.
func (t *Text) getfont() *draw.Font {
	return fontget(t.font, t.display)
}

func (t *Text) Init(r image.Rectangle, rf string, cols [frame.NumColours]*draw.Image, dis *draw.Display) *Text {
	// log.Println("Text.Init start")
	// defer log.Println("Text.Init end")
	if t == nil {
		t = new(Text)
	}
	t.display = dis
	t.all = r
	t.scrollr = r
	t.scrollr.Max.X = r.Min.X + t.display.ScaleSize(Scrollwid)
	t.lastsr = nullrect
	r.Min.X += t.display.ScaleSize(Scrollwid) + t.display.ScaleSize(Scrollgap)
	t.eq0 = ^0
	t.ncache = 0
	t.font = rf
	t.tabstop = int(maxtab)
	t.fr = frame.NewFrame(r, fontget(rf, t.display), t.display.ScreenImage, cols)
	t.Redraw(r,  -1, false /* noredraw */)
	return t
}

func (t *Text) Nc() int {
	return t.file.b.Nc()
}

// whatstring provides an easy-reading version of the Text usage.
func (t *Text) whatstring() string {
	switch t.what {
	case Body:
		return "Body"
	case Rowtag:
		return "Rowtag"
	case Tag:
		return "Tag"
	}
	return "Columntag"
}

func (t *Text) Redraw(r image.Rectangle, odx int, noredraw bool) {
	// log.Println("--- Text Redraw start", r, odx, "tag type:" ,  t.whatstring())
	// defer log.Println("--- Text Redraw end")
	/* use no wider than 3-space tabs in a directory */
	maxt := int(maxtab)
	if t.what == Body {
		if t.w.isdir {
			maxt = min(TABDIR, int(maxtab))
		} else {
			maxt = t.tabstop
		}
	}

	t.fr.Init(r, frame.OptMaxTab(maxt))
	if !noredraw {
		enclosing := r
		enclosing.Min.X -= t.display.ScaleSize(Scrollwid + Scrollgap)
		t.fr.Redraw(enclosing)
	}

	if t.what == Body && t.w.isdir && odx != t.all.Dx() {
		if t.fr.GetFrameFillStatus().Maxlines > 0 {
			t.Reset()
			t.Columnate(t.w.dirnames, t.w.widths)
			t.Show(0, 0, false)
		}
	} else {
		t.fill(t.fr)
		t.SetSelect(t.q0, t.q1)
	}
}

func (t *Text) Resize(r image.Rectangle, keepextra, noredraw bool) int {
	// log.Println("--- Text Resize start", r, keepextra, t.whatstring())
	// defer log.Println("--- Text Resize end")
	if r.Dy() <= 0 {
		// TODO(rjk): Speculative change to draw better. Original:
		// r.Max.Y = r.Min.Y
		// log.Println("r.Dy() <= 0 case")
		r = r.Canon()
	} else {
		if !keepextra {
			r.Max.Y -= r.Dy() % t.fr.DefaultFontHeight()
		}
	}
	odx := t.all.Dx()
	t.all = r
	t.scrollr = r
	t.scrollr.Max.X = r.Min.X + t.display.ScaleSize(Scrollwid)
	t.lastsr = image.ZR
	r.Min.X += t.display.ScaleSize(Scrollwid + Scrollgap)
	t.fr.Clear(false)
	// TODO(rjk): Remove this Font accessor.
	t.Redraw(r,odx, noredraw)
	return t.all.Max.Y
}

func (t *Text) Close() {
	t.fr.Clear(true)
	t.file.DelText(t)
	t.file = nil
	if argtext == t {
		argtext = nil
	}
	if typetext == t {
		typetext = nil
	}
	if seltext == t {
		seltext = nil
	}
	if mousetext == t {
		mousetext = nil
	}
	if barttext == t {
		barttext = nil
	}
}

func (t *Text) Columnate(names []string, widths []int) {
	var colw, mint, maxt, ncol, nrow int
	q1 := (0)
	Lnl := []rune("\n")
	Ltab := []rune("\t")

	if len(t.file.text) > 1 {
		return
	}
	mint = t.getfont().StringWidth("0")
	/* go for narrower tabs if set more than 3 wide */
	t.fr.Maxtab(min(int(maxtab), TABDIR) * mint)
	maxt = t.fr.GetMaxtab()
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
		ncol = max(1, t.fr.Rect().Dx()/colw)
	}
	nrow = (len(names) + ncol - 1) / ncol

	q1 = 0
	for i := 0; i < nrow; i++ {
		for j := i; j < len(names); j += nrow {
			dl := names[j]
			t.file.Insert(q1, []rune(dl))
			q1 += (len(dl))
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

func (t *Text) Load(q0 int, filename string, setqid bool) (nread int, err error) {
	if t.ncache != 0 || t.file.b.Nc() > 0 || t.w == nil || t != &t.w.body {
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

	var count int
	q1 := (0)
	hasNulls := false
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
			s, err := os.Stat(filepath.Join(fd.Name(), dn))
			if err != nil {
				warning(nil, "can't stat %s: %v\n", dn, err)
			} else {
				if s.IsDir() {
					dirNames[i] = dn + "/"
				}
			}
		}
		sort.Strings(dirNames)
		widths := make([]int, len(dirNames))
		dft := t.getfont()
		for i, s := range dirNames {
			widths[i] = dft.StringWidth(s)
		}
		t.Columnate(dirNames, widths)
		t.w.dirnames = dirNames
		t.w.widths = widths
		q1 = t.file.b.Nc()
	} else {
		t.w.isdir = false
		t.w.filemenu = true
		count, hasNulls, err = t.file.Load(q0, fd, setqid && q0 == 0)
		if err != nil {
			warning(nil, "Error reading file %s: %v", filename, err)
			return 0, fmt.Errorf("Error reading file %s: %v", filename, err)
		}
		q1 = q0 + count
	}
	if setqid {
		//t.file.dev = d.dev;
		t.file.mtime = d.ModTime()
		t.file.qidpath = d.Name() // TODO(flux): Gross hack to use filename as unique ID of file.
	}
	fd.Close()
	n := q1 - q0
	if q0 < t.org {
		t.org += n
	} else {
		if q0 <= t.org+(t.fr.GetFrameFillStatus().Nchars) { // Text is within the window, put it there.
			t.fr.Insert(t.file.b[q0:q0+n], int(q0-t.org))
		}
	}
	// For each clone, redraw
	for _, u := range t.file.text {
		if u != t { // Skip the one we just redrew
			if u.org > u.file.b.Nc() { /* will be 0 because of reset(), but safety first */
				u.org = 0
			}
			u.Resize(u.all, true, false /* noredraw */)
			u.Backnl(u.org, 0) /* go to beginning of line */
		}
		u.SetSelect(q0, q0)
	}
	if hasNulls {
		warning(nil, "%s: NUL bytes elided\n", filename)
	}
	return q1 - q0, nil

}

func (t *Text) Backnl(p int, n int) int {
	/* look for start of this line if n==0 */
	if n == 0 && p > 0 && t.ReadRune(p-1) != '\n' {
		n = 1
	}
	i := n
	for i > 0 && p > 0 {
		i--
		p-- /* it's at a newline now; back over it */
		if p == 0 {
			break
		}
		/* at 128 chars, call it a line anyway */
		for j := 128; j > 0 && p > 0; p-- {
			j--
			if t.ReadRune(p-1) == '\n' {
				break
			}
		}
	}
	return p
}

func (t *Text) BsInsert(q0 int, r []rune, tofile bool) (q, nrp int) {
	var (
		tp                 []rune
		bp, up, i, initial int
	)
	n := len(r)
	if t.what == Tag { // can't happen but safety first: mustn't backspace over file name
		t.Insert(q0, r, tofile)
		nrp = n
		return q0, nrp
	}
	bp = 0 // bp indexes r
	for i = 0; i < n; i++ {
		if r[bp] == '\b' {
			initial = 0
			tp = make([]rune, n)
			copy(tp, r[:i])
			up = i // up indexes tp, starting at i
			for ; i < n; i++ {
				tp[up] = r[bp]
				bp++
				if tp[up] == '\b' {
					if up == 0 {
						initial++
					} else {
						up--
					}
				} else {
					up++
				}
			}
			if initial != 0 {
				if initial > q0 {
					initial = q0
				}
				q0 -= initial
				t.Delete(q0, q0+initial, tofile)
			}
			n = up
			t.Insert(q0, tp[:n], tofile)
			nrp = n
			return q0, nrp
		} else {
			bp++
		}
	}
	t.Insert(q0, r, tofile)
	nrp = n
	return q0, nrp
}

func (t *Text) Insert(q0 int, r []rune, tofile bool) {
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
					u.ScrDraw(u.fr.GetFrameFillStatus().Nchars)
				}
			}
		}
	}
	n := (len(r))
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
		if t.fr != nil && q0 <= t.org+(t.fr.GetFrameFillStatus().Nchars) {
			t.fr.Insert(r[:n], int(q0-t.org))
		}
	}
	if t.w != nil {
		c := 'i'
		if t.what == Body {
			c = 'I'
		}
		if n <= EVENTSIZE {
			t.w.Eventf("%c%d %d 0 %d %v\n", c, q0, q0+n, n, string(r))
		} else {
			t.w.Eventf("%c%d %d 0 0 \n", c, q0, q0+n)
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

func (t *Text) inSelection(q0 int) bool {
	return t.q1 > t.q0 && t.q0 <= q0 && q0 <= t.q1
}



// Fill inserts additional text from t into the Frame object until the Frame object is full.
func (t *Text) fill(fr frame.SelectScrollUpdater) {
	// log.Println("Text.Fill Start", t.whatstring())
	// defer log.Println("Text.Fill End")

	// Conceivably, LastLineFull should be true or would it only be true if there are no more
	// characters possible?
	if fr.IsLastLineFull() || t.nofill {
		return
	}
	if t.ncache > 0 {
		t.TypeCommit()
	} 
	for {
		n := t.file.b.Nc() - (t.org + fr.GetFrameFillStatus().Nchars)
		if n == 0 {
			break
		}
		if n > 2000 { // educated guess at reasonable amount
			n = 2000
		}
		rp := make([]rune, n)
		t.file.b.Read(t.org+fr.GetFrameFillStatus().Nchars, rp)
		//
		// it's expensive to frinsert more than we need, so
		// count newlines.
		//
		nl := fr.GetFrameFillStatus().Maxlines - fr.GetFrameFillStatus().Nlines //+1

		m := 0
		var i int
		for i = 0; i < n; {
			i++
			if rp[i-1] == '\n' {
				m++
				if m >= nl {
					break
				}
			}
		}

		if lastlinefull := fr.Insert(rp[:i], fr.GetFrameFillStatus().Nchars); nl == 0 || lastlinefull {
			break
		}
	}
}

func (t *Text) Delete(q0, q1 int, tofile bool) {
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
					u.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
				}
			}
		}
	}
	if q0 < t.iq1 {
		t.iq1 -= min(n, t.iq1-q0)
	}
	if q0 < t.q0 {
		t.q0 -= min(n, t.q0-q0)
	}
	if q0 < t.q1 {
		t.q1 -= min(n, t.q1-q0)
	}
	if q1 <= t.org {
		t.org -= n
	} else if t.fr != nil && q0 < t.org+(t.fr.GetFrameFillStatus().Nchars) {
		p1 := q1 - t.org
		if p1 > (t.fr.GetFrameFillStatus().Nchars) {
			p1 = (t.fr.GetFrameFillStatus().Nchars)
		}
		p0 := 0
		if q0 < t.org {
			t.org = q0
			p0 = 0
		} else {
			p0 = q0 - t.org
		}
		t.fr.Delete((p0), (p1))
		t.fill(t.fr)
	}
	if t.w != nil {
		c := 'd'
		if t.what == Body {
			c = 'D'
		}
		t.w.Eventf("%c%d %d 0 0 \n", c, q0, q1)
	}
}

func (t *Text) View(q0, q1 int) []byte                   { return t.file.b.View(q0, q1) }
func (t *Text) ReadB(q int, r []rune) (n int, err error) { n, err = t.file.b.Read(q, r); return }
func (t *Text) nc() int                                  { return t.file.b.Nc() }
func (t *Text) Q0() int                                  { return t.q0 }
func (t *Text) Q1() int                                  { return t.q1 }
func (t *Text) SetQ0(q0 int)                             { t.q0 = q0 }
func (t *Text) SetQ1(q1 int)                             { t.q1 = q1 }
func (t *Text) Constrain(q0, q1 int) (p0, p1 int) {
	p0 = min(q0, t.file.b.Nc())
	p1 = min(q1, t.file.b.Nc())
	return p0, p1
}

func (t *Text) ReadRune(q int) rune {
	if t.cq0 <= q && q < t.cq0+(t.ncache) {
		return t.cache[q-t.cq0]
	} else {
		return t.file.b.ReadC(q)
	}
}

func (t *Text) BsWidth(c rune) int {
	/* there is known to be at least one character to erase */
	if c == 0x08 { /* ^H: erase character */
		return 1
	}
	q := t.q0
	skipping := true
	for q > 0 {
		r := t.ReadC(q - 1)
		if r == '\n' { /* eat at most one more character */
			if q == t.q0 { /* eat the newline */
				q--
			}
			break
		}
		if c == 0x17 {
			eq := isalnum(r)
			if eq && skipping { /* found one; stop skipping */
				skipping = false
			} else {
				if !eq && !skipping {
					break
				}
			}
		}
		q--
	}
	return t.q0 - q
}

func (t *Text) FileWidth(q0 int, oneelement bool) int {
	q := q0
	for q > 0 {
		r := t.ReadC(q - 1)
		if r <= ' ' {
			break
		}
		if oneelement && r == '/' {
			break
		}
		q--
	}
	return q0 - q
}

func (t *Text) Complete() []rune {
	Unimpl()
	return nil
}

func (t *Text) Type(r rune) {
	var (
		q0, q1        int
		nnb, nb, n, i int
		nr            int
	)
	// Avoid growing column and row tags.
	if t.what != Body && t.what != Tag && r == '\n' {
		return
	}
	if t.what == Tag {
		t.w.tagsafe = false
	}
	nr = 1
	rp := []rune{r}

	Tagdown := func() {
		/* expand tag to show all text */
		if !t.w.tagexpand {
			t.w.tagexpand = true
			t.w.Resize(t.w.r, false, true)
		}
		return
	}

	Tagup := func() {
		/* shrink tag to single line */
		if t.w.tagexpand {
			t.w.tagexpand = false
			t.w.taglines = 1
			t.w.Resize(t.w.r, false, true)
		}
		return
	}

	case_Down := func() {
		q0 = t.org + t.fr.Charofpt(image.Pt(t.fr.Rect().Min.X, t.fr.Rect().Min.Y+n*t.fr.DefaultFontHeight()))
		t.SetOrigin(q0, true)
		return
	}
	case_Up := func() {
		q0 = t.Backnl(t.org, n)
		t.SetOrigin(q0, true)
		return
	}

	switch r {
	case draw.KeyLeft:
		t.TypeCommit()
		if t.q0 > 0 {
			if t.q0 != t.q1 {
				t.Show(t.q0, t.q0, true)
			} else {
				t.Show(t.q0-1, t.q0-1, true)
			}
		}
		return
	case draw.KeyRight:
		t.TypeCommit()
		if t.q1 < t.file.b.Nc() {
			// This is a departure from the plan9/plan9port acme
			// Instead of always going right one char from q1, it
			// collapses multi-character selections first, behaving
			// like every other selection on modern systems. -flux
			if t.q0 != t.q1 {
				t.Show(t.q1, t.q1, true)
			} else {
				t.Show(t.q1+1, t.q1+1, true)
			}
		}
		return
	case draw.KeyDown:
		if t.what == Tag {
			Tagdown()
			return
		}
		n = t.fr.GetFrameFillStatus().Maxlines / 3
		case_Down()
		return
	case Kscrollonedown:
		if t.what == Tag {
			Tagdown()
			return
		}
		n = mousescrollsize(t.fr.GetFrameFillStatus().Maxlines)
		if n <= 0 {
			n = 1
		}
		case_Down()
		return
	case draw.KeyPageDown:
		n = 2 * t.fr.GetFrameFillStatus().Maxlines / 3
		case_Down()
		return
	case draw.KeyUp:
		if t.what == Tag {
			Tagup()
			return
		}
		n = t.fr.GetFrameFillStatus().Maxlines / 3
		case_Up()
		return
	case Kscrolloneup:
		if t.what == Tag {
			Tagup()
			return
		}
		n = mousescrollsize(t.fr.GetFrameFillStatus().Maxlines)
		case_Up()
		return
	case draw.KeyPageUp:
		n = 2 * t.fr.GetFrameFillStatus().Maxlines / 3
		case_Up()
		return
	case draw.KeyHome:
		t.TypeCommit()
		if t.org > t.iq1 {
			q0 = t.Backnl(t.iq1, 1)
			t.SetOrigin(q0, true)
		} else {
			t.Show(0, 0, false)
		}
		return
	case draw.KeyEnd:
		t.TypeCommit()
		if t.iq1 > t.org+t.fr.GetFrameFillStatus().Nchars {
			if t.iq1 > t.file.b.Nc() {
				// should not happen, but does. and it will crash textbacknl.
				t.iq1 = t.file.b.Nc()
			}
			q0 = t.Backnl(t.iq1, 1)
			t.SetOrigin(q0, true)
		} else {
			t.Show(t.file.b.Nc(), t.file.b.Nc(), false)
		}
		return
	case 0x01: /* ^A: beginning of line */
		t.TypeCommit()
		/* go to where ^U would erase, if not already at BOL */
		nnb = 0
		if t.q0 > 0 && t.ReadC(t.q0-1) != '\n' {
			nnb = t.BsWidth(0x15)
		}
		t.Show(t.q0-nnb, t.q0-nnb, true)
		return
	case 0x05: /* ^E: end of line */
		t.TypeCommit()
		q0 = t.q0
		for q0 < t.file.b.Nc() && t.ReadC(q0) != '\n' {
			q0++
		}
		t.Show(q0, q0, true)
		return
	case draw.KeyCmd + 'c': /* %C: copy */
		t.TypeCommit()
		cut(t, t, nil, true, false, "")
		return
	case draw.KeyCmd + 'z': /* %Z: undo */
		t.TypeCommit()
		undo(t, nil, nil, true, false, "")
		return
	case draw.KeyCmd + 'Z': /* %-shift-Z: redo */
		t.TypeCommit()
		undo(t, nil, nil, false, false, "")
		return

	}
	if t.what == Body {
		seq++
		t.file.Mark()
	}
	/* cut/paste must be done after the seq++/filemark */
	switch r {
	case draw.KeyCmd + 'x': /* %X: cut */
		t.TypeCommit()
		if t.what == Body {
			seq++
			t.file.Mark()
		}
		cut(t, t, nil, true, true, "")
		t.Show(t.q0, t.q0, true)
		t.iq1 = t.q0
		return
	case draw.KeyCmd + 'v': /* %V: paste */
		t.TypeCommit()
		if t.what == Body {
			seq++
			t.file.Mark()
		}
		paste(t, t, nil, true, false, "")
		t.Show(t.q0, t.q1, true)
		t.iq1 = t.q1
		return
	}
	wasrange := t.q0 != t.q1
	if t.q1 > t.q0 {
		if t.ncache != 0 {
			acmeerror("text.type", nil)
		}
		cut(t, t, nil, true, true, "")
		t.eq0 = ^0
	}
	t.Show(t.q0, t.q0, true)
	switch r {
	case 0x06:
		fallthrough /* ^F: complete */
	case draw.KeyInsert:
		t.TypeCommit()
		rp = t.Complete()
		if rp == nil {
			return
		}
		nr = len(rp) // runestrlen(rp);
		break        /* fall through to normal insertion case */
	case 0x1B:
		if t.eq0 != ^0 {
			if t.eq0 <= t.q0 {
				t.SetSelect(t.eq0, t.q0)
			} else {
				t.SetSelect(t.q0, t.eq0)
			}
		}
		if t.ncache > 0 {
			t.TypeCommit()
		}
		t.iq1 = t.q0
		return
	case 0x7F: /* Del: erase character right */
		if t.q1 >= t.Nc()-1 {
			return // End of file
		}
		t.TypeCommit() // Avoid messing with the cache?
		if !wasrange {
			t.q1++
			cut(t, t, nil, false, true, "")
		}
		return
	case 0x08:
		fallthrough /* ^H: erase character */
	case 0x15:
		fallthrough /* ^U: erase line */
	case 0x17: /* ^W: erase word */
		if t.q0 == 0 { /* nothing to erase */
			return
		}
		nnb = t.BsWidth(r)
		q1 = t.q0
		q0 = q1 - nnb
		/* if selection is at beginning of window, avoid deleting invisible text */
		if q0 < t.org {
			q0 = t.org
			nnb = q1 - q0
		}
		if nnb <= 0 {
			return
		}
		for _, u := range t.file.text { // u is *Text
			u.nofill = true
			nb = nnb
			n = u.ncache
			if n > 0 {
				if q1 != u.cq0+n {
					acmeerror("text.type backspace", nil)
				}
				if n > nb {
					n = nb
				}
				u.ncache -= n
				u.Delete(q1-n, q1, false)
				nb -= n
			}
			if u.eq0 == q1 || u.eq0 == ^0 {
				u.eq0 = q0
			}
			if nb != 0 && u == t {
				u.Delete(q0, q0+nb, true)
			}
			if u != t {
				u.SetSelect(u.q0, u.q1)
			} else {
				t.SetSelect(q0, q0)
			}
			u.nofill = false
		}
		for _, t := range t.file.text {
			t.fill(t.fr)
		}
		t.iq1 = t.q0
		return
	case '\n':
		if t.w.autoindent {
			/* find beginning of previous line using backspace code */
			nnb = t.BsWidth(0x15)    /* ^U case */
			rp = make([]rune, nnb+1) //runemalloc(nnb + 1);
			nr = 0
			rp[nr] = r
			nr++
			for i = 0; i < nnb; i++ {
				r = t.ReadC(t.q0 - nnb + i)
				if r != ' ' && r != '\t' {
					break
				}
				rp[nr] = r
				nr++
			}
			rp = rp[:nr]
		}
	}
	/* otherwise ordinary character; just insert, typically in caches of all texts */
	for _, u := range t.file.text { // u is *Text
		if u.eq0 == ^0 {
			u.eq0 = t.q0
		}
		if u.ncache == 0 {
			u.cq0 = t.q0
		} else {
			if t.q0 != u.cq0+u.ncache {
				acmeerror("text.type cq1", nil)
			}
		}
		/*
		 * Change the tag before we add to ncache,
		 * so that if the window body is resized the
		 * commit will not find anything in ncache.
		 */
		if u.what == Body && u.ncache == 0 {
			u.needundo = true
			t.w.SetTag()
			u.needundo = false
		}
		u.Insert(t.q0, rp, false)
		if u != t {
			u.SetSelect(u.q0, u.q1)
		}
		if u.ncache+nr > u.ncachealloc {
			u.ncachealloc += 10 + nr
			u.cache = append(u.cache, make([]rune, 10+nr)...) //runerealloc(u.cache, u.ncachealloc);
		}
		//runemove(u.cache+u.ncache, rp, nr);
		copy(u.cache[u.ncache:], rp[:nr])
		u.ncache += nr
		if t.what == Tag { // TODO(flux): This is hideous work-around for
			// what looks like a subtle bug near here.
			t.w.Commit(t)
		}
	}
	t.SetSelect(t.q0+nr, t.q0+nr)
	if r == '\n' && t.w != nil {
		t.w.Commit(t)
	}
	t.iq1 = t.q0

}

func (t *Text) Commit(tofile bool) {
	if t.ncache == 0 {
		return
	}
	if tofile {
		t.file.Insert(t.cq0, t.cache[:t.ncache])
	}
	if t.what == Body {
		t.w.dirty = true
		t.w.utflastqid = -1
	}
	t.ncache = 0
}

// TODO(rjk): Conceivably, this can be removed.
func getP0(fr frame.Frame) int {
	p0, _ := fr.GetSelectionExtent()
	return p0
}
func getP1(fr frame.Frame) int {
	_, p1 := fr.GetSelectionExtent()
	return p1
}

func (t *Text) FrameScroll(fr frame.SelectScrollUpdater, dl int) {
	var q0 int
	if dl == 0 {
		// TODO(rjk): Make this mechanism better? It seems unfortunate.
		ScrSleep(100)
		return
	}
	if dl < 0 {
		q0 = t.Backnl(t.org, (-dl))
	} else {
		if t.org+(fr.GetFrameFillStatus().Nchars) == t.file.b.Nc() {
			return
		}
		q0 = t.org + (fr.Charofpt(image.Pt(fr.Rect().Min.X, fr.Rect().Min.Y+dl* fr.DefaultFontHeight())))
	}
	// Insert text into the frame.
	t.setorigin(fr, q0, true, true)
}

var (
	clicktext *Text
	clickmsec uint32
	// TODO(rjk): Replace with closure.
	selecttext *Text
	selectq    int
)

func (t *Text) Select() {
	// log.Println("Text.Select Begin")
	// defer log.Println("Text.Select End")

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
	b := mouse.Buttons
	q0 := t.q0
	q1 := t.q1
	selectq = t.org + (t.fr.Charofpt(mouse.Point))
	//	fmt.Printf("Text.Select: mouse.Msec %v, clickmsec %v\n", mouse.Msec, clickmsec)
	//	fmt.Printf("clicktext==t %v, (q0==q1 && selectq==q0): %v", clicktext == t, q0 == q1 && selectq == q0)
	if (clicktext == t && mouse.Msec-uint32(clickmsec) < 500) && (q0 == q1 && selectq == q0) {
		q0, q1 = t.DoubleClick(q0)
		fmt.Printf("Text.Select: DoubleClick returned %d, %d\n", q0, q1)
		t.SetSelect(q0, q1)
		t.display.Flush()
		x := mouse.Point.X
		y := mouse.Point.Y
		/* stay here until something interesting happens */
		// TODO(rjk): Ack. This is horrible? Layering violation?
		for {
			mousectl.Read()
			if !(mouse.Buttons == b && abs(mouse.Point.X-x) < 3 && abs(mouse.Point.Y-y) < 3) {
				break
			}
		}
		mouse.Point.X = x /* in case we're calling frselect */
		mouse.Point.Y = y
		q0 = t.q0 /* may have changed */
		q1 = t.q1
		selectq = q0
	}
	if mouse.Buttons == b {
		sP0, sP1 := t.fr.Select(mousectl, mouse, func(fr frame.SelectScrollUpdater, dl int) { t.FrameScroll(fr, dl) })

		/* horrible botch: while asleep, may have lost selection altogether */
		if selectq > t.file.b.Nc() {
			selectq = t.org + sP0
		}
		if selectq < t.org {
			q0 = selectq
		} else {
			q0 = t.org + sP0
		}
		if selectq > t.org+(t.fr.GetFrameFillStatus().Nchars) {
			q1 = selectq
		} else {
			q1 = t.org + sP1
		}
	}
	if q0 == q1 {
		if q0 == t.q0 && clicktext == t && mouse.Msec-uint32(clickmsec) < 500 {
			q0, q1 = t.DoubleClick(q0)
			clicktext = nil
		} else {
			clicktext = t
			clickmsec = mouse.Msec
		}
	} else {
		clicktext = nil
	}
	t.SetSelect(q0, q1)
	t.display.Flush()
	state := None /* what we've done; undo when possible */
	for mouse.Buttons != 0 {
		mouse.Msec = 0
		b := mouse.Buttons
		if (b&1) != 0 && (b&6) != 0 {
			if state == None && t.what == Body {
				seq++
				t.w.body.file.Mark()
			}
			if b&2 != 0 {
				if state == Paste && t.what == Body {
					t.w.Undo(true)
					t.SetSelect(q0, t.q1)
					state = None
				} else {
					if state != Cut {
						cut(t, t, nil, true, true, "")
						state = Cut
					}
				}
			} else {
				if state == Cut && t.what == Body {
					t.w.Undo(true)
					t.SetSelect(q0, t.q1)
					state = None
				} else {
					if state != Paste {
						paste(t, t, nil, true, false, "")
						state = Paste
					}
				}
			}
			t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
			clearmouse()
		}
		t.display.Flush()
		for mouse.Buttons == b {
			mousectl.Read()
		}
		clicktext = nil
	}
}

func (t *Text) Show(q0, q1 int, doselect bool) {
	var (
		qe  int
		nl  int
		tsd bool
		nc  int
		q   int
	)
	if t.what != Body {
		if doselect {
			t.SetSelect(q0, q1)
		}
		return
	}
	if t.w != nil && t.fr.GetFrameFillStatus().Maxlines == 0 {
		t.col.Grow(t.w, 1)
	}
	if doselect {
		t.SetSelect(q0, q1)
	}
	qe = t.org + t.fr.GetFrameFillStatus().Nchars
	tsd = false /* do we call textscrdraw? */
	nc = t.file.b.Nc() + t.ncache
	if t.org <= q0 {
		if nc == 0 || q0 < qe {
			tsd = true
		} else {
			if q0 == qe && qe == nc {
				if t.ReadC(nc-1) == '\n' {
					if t.fr.GetFrameFillStatus().Nlines < t.fr.GetFrameFillStatus().Maxlines {
						tsd = true
					}
				} else {
					tsd = true
				}
			}
		}
	}
	if tsd {
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
	} else {
		if t.w.nopen[QWevent] > 0 {
			nl = 3 * t.fr.GetFrameFillStatus().Maxlines / 4
		} else {
			nl = t.fr.GetFrameFillStatus().Maxlines / 4
		}
		q = t.Backnl(q0, nl)
		/* avoid going backwards if trying to go forwards - long lines! */
		if !(q0 > t.org && q < t.org) {
			t.SetOrigin(q, true)
		}
		for q0 > t.org+t.fr.GetFrameFillStatus().Nchars {
			t.SetOrigin(t.org+1, false)
		}
	}
}

func (t *Text) ReadC(q int) (r rune) {
	if t.cq0 <= q && q < t.cq0+(t.ncache) {
		r = t.cache[q-t.cq0]
	} else {
		r = t.file.b.ReadC(q)
	}
	return r

}

func (t *Text) SetSelect(q0, q1 int) {
	//  log.Println("Text SetSelect Start", q0, q1)
	// adefer log.Println("Text SetSelect End", q0, q1)

	t.q0 = q0
	t.q1 = q1
	/* compute desired p0,p1 from q0,q1 */
	p0 := q0 - t.org
	p1 := q1 - t.org
	ticked := true
	if p0 < 0 {
		p0 = 0
	}
	if p1 < 0 {
		ticked = false
		p1 = 0
	}
	if t.fr == nil {
		return
	}
	if p0 > (t.fr.GetFrameFillStatus().Nchars) {
		ticked = false
		p0 = (t.fr.GetFrameFillStatus().Nchars)
	}
	if p1 > (t.fr.GetFrameFillStatus().Nchars) {
		p1 = (t.fr.GetFrameFillStatus().Nchars)
	}
	if p0 > p1 {
		panic(fmt.Sprintf("acme: textsetselect p0=%d p1=%d q0=%v q1=%v t.org=%d nchars=%d", p0, p1, q0, q1, t.org, t.fr.GetFrameFillStatus().Nchars))
	}

	t.fr.DrawSel(t.fr.Ptofchar(p0), p0, p1, ticked)
}

// TODO(rjk): The implicit initialization of q0, q1 doesn't seem like very nice
// style? Maybe it is idiomatic?
func (t *Text) Select23(high *draw.Image, mask uint) (q0, q1 int, buts uint) {
	p0, p1 := t.fr.SelectOpt(mousectl, mouse, func(frame.SelectScrollUpdater, int) {}, t.display.White, high)

	buts = uint(mousectl.Mouse.Buttons)
	if (buts & mask) == 0 {
		q0 = p0 + t.org
		q1 = p1 + t.org
	}
	for mousectl.Mouse.Buttons != 0 {
		mousectl.Read()
	}
	return q0, q1, buts
}

func (t *Text) Select2() (q0, q1 int, tp *Text, ret bool) {
	q0, q1, buts := t.Select23(but2col, 4)
	if (buts & 4) != 0 {
		return q0, q1, nil, false
	}
	if (buts & 1) != 0 { /* pick up argument */
		return q0, q1, argtext, true
	}
	return q0, q1, nil, true
}

func (t *Text) Select3() (q0, q1 int, r bool) {
	q0, q1, buts := t.Select23(but3col, 1|2)
	return q0, q1, buts == 0
}

func (t *Text) DoubleClick(inq0 int) (q0, q1 int) {
	q0 = inq0
	if q0, q1, ok := t.ClickHTMLMatch(inq0); ok {
		return q0, q1
	}
	var c rune
	for i, l := range left {
		q := inq0
		r := right[i]
		/* try matching character to left, looking right */
		if q == 0 {
			c = '\n'
		} else {
			c = t.ReadC(q - 1)
		}
		p := runestrchr(l, c)
		if p != -1 {
			if q, ok := t.ClickMatch(c, r[p], 1, q); ok {
				q1 = q
				if c != '\n' {
					q1--
				}
			}
			return
		}
		/* try matching character to right, looking left */
		if q == t.file.b.Nc() {
			c = '\n'
		} else {
			c = t.ReadC(q)
		}
		p = runestrchr(r, c)
		if p != -1 {
			if q, ok := t.ClickMatch(c, l[p], -1, q); ok {
				q1 = inq0
				if q0 < t.file.b.Nc() && c == '\n' {
					q1++
				}
				q0 = q
				if c != '\n' || q != 0 || t.ReadC(0) == '\n' {
					q0++
				}
			}
			return
		}
	}
	/* try filling out word to right */
	q1 = inq0
	for q1 < t.file.b.Nc() && isalnum(t.ReadC(q1)) {
		q1++
	}
	/* try filling out word to left */
	for q0 > 0 && isalnum(t.ReadC(q0-1)) {
		q0--
	}

	return q0, q1
}

func (t *Text) ClickMatch(cl, cr rune, dir int, inq int) (q int, r bool) {
	nest := 1
	var c rune
	for {
		if dir > 0 {
			if inq == t.file.b.Nc() {
				break
			}
			c = t.ReadC(inq)
			(inq)++
		} else {
			if inq == 0 {
				break
			}
			(inq)--
			c = t.ReadC(inq)
		}
		if c == cr {
			nest--
			if nest == 0 {
				return inq, true
			}
		} else {
			if c == cl {
				nest++
			}
		}
	}
	return inq, cl == '\n' && nest == 1
}

func (t *Text) ishtmlstart(q int) (q1 int, stat int) {
	Untested()
	if q+2 > t.file.b.Nc() {
		return 0, 0
	}
	if t.ReadC(q) != '<' {
		return 0, 0
	}
	q++
	c := t.ReadC(q)
	q++
	c1 := c
	c2 := c
	for c != '>' {
		if q >= t.file.b.Nc() {
			return 0, 0
		}
		c2 = c
		c = t.ReadC(q)
		q++
	}
	if c1 == '/' {
		return q, -1
	}
	if c2 == '/' || c2 == '!' {
		return 0, 0
	}
	return q, 1
}

func (t *Text) ishtmlend(q int) (q1 int, stat int) {
	Untested()
	if q < 2 {
		return 0, 0
	}
	q--
	if t.ReadC(q) != '>' {
		return 0, 0
	}
	q--
	c := t.ReadC(q)
	c1 := c
	c2 := c
	for c != '<' {
		if q == 0 {
			return 0, 0
		}
		c1 = c
		q--
		c = t.ReadC(q)
	}
	if c1 == '/' {
		return q, -1
	}
	if c2 == '/' || c2 == '!' {
		return 0, 0
	}
	return q, 1
}

func (t *Text) ClickHTMLMatch(inq0 int) (q0, q1 int, r bool) {
	depth := 0
	q := inq0
	q0 = inq0

	// after opening tag?  scan forward for closing tag
	_, stat := t.ishtmlend(inq0)
	if stat == 1 {
		depth = 1
		for q < t.file.b.Nc() {
			nq, n := t.ishtmlstart(q)
			if n != 0 {
				depth += n
				if depth == 0 {
					return q0, q, true
				}
				q = nq
				continue
			}
			q++
		}
	}

	// before closing tag?  scan backward for opening tag
	_, stat = t.ishtmlstart(q)
	if stat == -1 {
		depth = -1
		for q > 0 {
			nq, n := t.ishtmlend(q)
			if n != 0 {
				depth += n
				if depth == 0 {
					return q, q1, true
				}
				q = nq
				continue
			}
			q--
		}
	}

	return 0, 0, false
}

func (t *Text) BackNL(p, n int) int {
	var i int

	/* look for start of this line if n==0 */
	if n == 0 && p > 0 && t.ReadC(p-1) != '\n' {
		n = 1
	}
	i = n
	for i > 0 && p > 0 {
		i--
		p-- /* it's at a newline now; back over it */
		if p == 0 {
			break
		}
		/* at 128 chars, call it a line anyway */
		for j := 128; j > 0 && p > 0; p-- {
			if t.ReadC(p-1) == '\n' {
				break
			}
			j--
		}
	}
	return p
}

func (t *Text) SetOrigin(org int, exact bool) {
	t.setorigin(t.fr, org, exact, false)
}

func (t *Text) setorigin(fr frame.SelectScrollUpdater, org int, exact bool, calledfromscroll bool) {
	// log.Printf("Text.SetOrigin start: t.org = %v, org = %v, exact = %v\n", t.org, org, exact)
	// defer log.Println("Text.SetOrigin end")
	// log.Printf("\tfr.GetFrameFillStatus().Nchars = %#v\n", fr.GetFrameFillStatus().Nchars)

	var (
		i, a int
		r    []rune
		n    int
	)

	// rjk: I'm not sure what this is for exactly.
	if org > 0 && !exact && t.ReadC(org-1) != '\n' {
		// org is an estimate of the char posn; find a newline
		// don't try harder than 256 chars
		for i = 0; i < 256 && org < t.file.b.Nc(); i++ {
			if t.ReadC(org) == '\n' {
				org++
				break
			}
			org++
		}
	}
	a = org - t.org
	if a >= 0 && a < fr.GetFrameFillStatus().Nchars {
		fr.Delete(0, a)
	} else {
		if a < 0 && -a <fr.GetFrameFillStatus().Nchars {
			n = t.org - org
			r = make([]rune, n)
			t.file.b.Read(org, r)
			fr.Insert(r, 0)
		} else {
			fr.Delete(0, fr.GetFrameFillStatus().Nchars)
		}
	}
	t.org = org
	t.fill(fr)
	t.ScrDraw(fr.GetFrameFillStatus().Nchars)

	if !calledfromscroll {
		t.SetSelect(t.q0, t.q1)
	}
}

func (t *Text) Reset() {
	t.file.seq = 0
	t.eq0 = ^0
	/* do t.delete(0, t.nc, true) without building backup stuff */
	t.SetSelect(t.org, t.org)
	t.fr.Delete(0, t.fr.GetFrameFillStatus().Nchars)
	t.org = 0
	t.q0 = 0
	t.q1 = 0
	t.file.Reset()
	t.file.b.Reset()
}

func (t *Text) DirName(name string) string {
	if t == nil || t.w == nil {
		return string(cleanrname([]rune(name)))
	}
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	b := make([]rune, t.w.tag.file.b.Nc())
	t.w.tag.file.b.Read(0, b)
	spl := strings.SplitN(string(b), " ", 2)[0]
	if !strings.HasSuffix(spl, string(filepath.Separator)) {
		spl = filepath.Dir(spl)
	}
	spl = filepath.Clean(spl + string(filepath.Separator) + name)
	return spl

}
