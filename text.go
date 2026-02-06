package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/rjkroege/edwood/complete"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/draw/drawutil"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/runes"
	"github.com/rjkroege/edwood/util"
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

	_ file.BufferObserver = (*Text)(nil) // Enforce at compile time that Text implements BufferObserver
)

type TextKind byte

const (
	Columntag TextKind = iota
	Rowtag
	Tag
	Body
)

// Text is a view onto a buffer, managing a frame.
// Files have possible multiple texts corresponding to clones.
type Text struct {
	display draw.Display
	file    *file.ObservableEditableBuffer
	fr      frame.Frame
	font    string

	org       int // Origin of the frame within the buffer
	q0        int
	q1        int
	what      TextKind
	tabstop   int
	tabexpand bool
	w         *Window
	scrollr   image.Rectangle
	lastsr    image.Rectangle
	all       image.Rectangle
	row       *Row
	col       *Column

	iq1 int
	eq0 int // When 0, typing has started

	nofill bool // When true, updates to the Text shouldn't update the frame.

	lk sync.Mutex
}

// getfont is a convenience accessor that gets the draw.Font from the font
// used in this text.
func (t *Text) getfont() draw.Font {
	return fontget(t.font, t.display)
}

func (t *Text) Init(r image.Rectangle, rf string, cols [frame.NumColours]draw.Image, dis draw.Display) *Text {
	// log.Println("Text.Init start")
	// defer log.Println("Text.Init end")
	if t == nil {
		t = new(Text)
	}
	t.display = dis
	t.all = r
	t.scrollr = r
	t.scrollr.Max.X = r.Min.X + t.display.ScaleSize(Scrollwid)
	t.lastsr = image.Rectangle{}
	r.Min.X += t.display.ScaleSize(Scrollwid) + t.display.ScaleSize(Scrollgap)
	t.eq0 = ^0
	t.font = rf
	t.tabstop = int(global.maxtab)
	t.tabexpand = global.tabexpand
	t.fr = frame.NewFrame(r, fontget(rf, t.display), t.display.ScreenImage(), cols)
	t.Redraw(r, -1, false /* noredraw */)
	return t
}

func (t *Text) Nc() int {
	return t.file.Nr()
}

// String returns a string representation of the TextKind.
func (tk TextKind) String() string {
	switch tk {
	case Body:
		return "Body"
	case Columntag:
		return "Columntag"
	case Rowtag:
		return "Rowtag"
	case Tag:
		return "Tag"
	}
	return fmt.Sprintf("TextKind(%v)", int(tk))
}

func (t *Text) Redraw(r image.Rectangle, odx int, noredraw bool) {
	// log.Println("--- Text Redraw start", r, odx, "tag type:" ,  t.what)
	// defer log.Println("--- Text Redraw end")
	// use no wider than 3-space tabs in a directory
	maxt := int(global.maxtab)
	if t.what == Body {
		if t.file.IsDir() {
			maxt = util.Min(TABDIR, int(global.maxtab))
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

	if t.what == Body && t.file.IsDir() && odx != t.all.Dx() {
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
	// log.Println("--- Text Resize start", r, keepextra, t.what)
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
	t.lastsr = image.Rectangle{}
	r.Min.X += t.display.ScaleSize(Scrollwid + Scrollgap)
	t.fr.Clear(false)
	// TODO(rjk): Remove this Font accessor.
	t.Redraw(r, odx, noredraw)
	return t.all.Max.Y
}

func (t *Text) Close() {
	t.fr.Clear(true)
	if err := t.file.DelObserver(t); err != nil {
		util.AcmeError(err.Error(), nil)
	}
	t.file = nil
	if global.argtext == t {
		global.argtext = nil
	}
	if global.typetext == t {
		global.typetext = nil
	}
	if global.seltext == t {
		global.seltext = nil
	}
	if global.mousetext == t {
		global.mousetext = nil
	}
	if global.barttext == t {
		global.barttext = nil
	}
}

func (t *Text) Columnate(names []string, widths []int) {
	var colw, mint, maxt, ncol, nrow int
	q1 := 0
	Lnl := []rune("\n")
	Ltab := []rune("\t")

	if t.file.HasMultipleObservers() {
		panic("Text.Columnate is only for directories that can't have zerox")
	}

	mint = t.getfont().StringWidth("0")
	// go for narrower tabs if set more than 3 wide
	t.fr.Maxtab(util.Min(int(global.maxtab), TABDIR) * mint)
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
		ncol = util.Max(1, t.fr.Rect().Dx()/colw)
	}
	nrow = (len(names) + ncol - 1) / ncol

	q1 = 0
	for i := 0; i < nrow; i++ {
		for j := i; j < len(names); j += nrow {
			dl := bytetorune([]byte(names[j]))
			t.file.InsertAt(q1, dl)
			q1 += len(dl)
			if j+nrow >= len(names) {
				break
			}
			w := widths[j]
			if maxt-w%maxt < mint {
				t.file.InsertAt(q1, Ltab)
				q1++
				w += mint
			}
			for {
				t.file.InsertAt(q1, Ltab)
				q1++
				w += maxt - (w % maxt)
				if !(w < colw) {
					break
				}
			}
		}
		t.file.InsertAt(q1, Lnl)
		q1++
	}
}

func (t *Text) checkSafeToLoad(filename string) error {
	if t.file.Nr() > 0 || t.w == nil || t != &t.w.body {
		panic("text.load")
	}

	if t.file.IsDir() && t.file.Name() == "" {
		return warnError(nil, "empty directory name")
	}
	if ismtpt(filename) {
		return warnError(nil, "will not open self mount point %s", filename)
	}
	return nil
}

func (t *Text) loadReader(q0 int, filename string, rd io.Reader, sethash bool) (nread int, err error) {
	t.file.SetDir(false)
	t.w.filemenu = true
	count, hasNulls, err := t.file.Load(q0, rd, sethash)
	if err != nil {
		return 0, warnError(nil, "error reading file %s: %v", filename, err)
	}
	if hasNulls {
		warning(nil, "%s: NUL bytes elided\n", filename)
	}
	return count, nil
}

// LoadReader loads an io.Reader into the Text.file. Text must be of type body.
// Filename is only used for error reporting, not for access to the on-disk file.
func (t *Text) LoadReader(q0 int, filename string, rd io.Reader, sethash bool) (nread int, err error) {
	if err := t.checkSafeToLoad(filename); err != nil {
		return 0, err
	}
	return t.loadReader(q0, filename, rd, sethash)
}

// Load loads filename into the Text.file. Text must be of type body.
func (t *Text) Load(q0 int, filename string, setqid bool) (nread int, err error) {
	if err := t.checkSafeToLoad(filename); err != nil {
		return 0, err
	}
	fd, err := os.Open(filename)
	if err != nil {
		return 0, warnError(nil, "can't open %s: %v", filename, err)
	}
	defer fd.Close()
	d, err := fd.Stat()
	if err != nil {
		return 0, warnError(nil, "can't fstat %s: %v", filename, err)
	}
	if setqid {
		t.file.SetInfo(d)
	}

	if d.IsDir() {
		// this is checked in get() but it's possible the file changed underfoot
		if t.file.HasMultipleObservers() {
			return 0, warnError(nil, "%s is a directory; can't read with multiple windows on it", filename)
		}
		t.file.SetDir(true)
		t.w.filemenu = false
		if len(t.file.Name()) > 0 && !strings.HasSuffix(t.file.Name(), string(filepath.Separator)) {
			t.file.SetName(t.file.Name() + string(filepath.Separator))
			t.w.SetName(t.file.Name())
		}
		dirNames, err := getDirNames(fd)
		if err != nil {
			return 0, warnError(nil, "getDirNames failed: %v", err)
		}
		widths := make([]int, len(dirNames))
		dft := t.getfont()
		for i, s := range dirNames {
			widths[i] = dft.StringWidth(s)
		}
		t.Columnate(dirNames, widths)
		t.w.dirnames = dirNames
		t.w.widths = widths
		q1 := t.file.Nr()
		return q1 - q0, nil
	}
	return t.loadReader(q0, filename, fd, setqid && q0 == 0)
}

func getDirNames(f *os.File) ([]string, error) {
	entries, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, fi := range entries {
		if fi.IsDir() {
			names[i] = fi.Name() + string(filepath.Separator)
		} else {
			names[i] = fi.Name()
		}
	}
	sort.Strings(names)
	for i := range names {
		names[i] = QuoteFilename(names[i])
	}
	return names, nil
}

// BsInsert inserts runes r at text position q0. If r contains backspaces ('\b'),
// they are interpreted, removing the runes preceding them.
// The final text position where r is inserted and the number of runes inserted
// after interpreting backspaces is returned.
func (t *Text) BsInsert(q0 int, r []rune, tofile bool) (q, nr int) {
	n := len(r)
	if t.what == Tag { // can't happen but safety first: mustn't backspace over file name
		t.Insert(q0, r, tofile)
		return q0, n
	}
	bp := 0 // bp indexes r
	for i := 0; i < n; i++ {
		if r[bp] == '\b' {
			initial := 0
			tp := make([]rune, n)
			copy(tp, r[:i])
			up := i // up indexes tp, starting at i
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
			return q0, n
		}
		bp++
	}
	t.Insert(q0, r, tofile)
	return q0, n
}

// inserted is a callback invoked by File on Insert* to update each Text
// that is using a given File.
// TODO(rjk): Carefully scrub this for opportunities to not do work if the
// changes are not in the viewport. Also: minimize scrollbar redraws.
func (t *Text) Inserted(oq0 file.OffsetTuple, b []byte, nr int) {
	q0 := oq0.R
	if t.eq0 == -1 {
		t.eq0 = q0
	}
	if t.what == Body {
		t.w.utflastqid = -1
	}

	if q0 < t.iq1 {
		t.iq1 += nr
	}
	if q0 < t.q1 {
		t.q1 += nr
	}
	if q0 < t.q0 {
		t.q0 += nr
	}

	// In Markdeep mode, don't update the text frame directly.
	// Instead, schedule a debounced re-render of the Markdeep view.
	if t.what == Body && t.w != nil && t.w.IsPreviewMode() {
		t.logInsert(oq0, b, nr)
		t.w.recordEdit(markdown.EditRecord{Pos: q0, OldLen: 0, NewLen: nr})
		t.w.SchedulePreviewUpdate()
		return
	}

	if q0 < t.org {
		t.org += nr
	} else {
		if t.fr != nil && q0 <= t.org+(t.fr.GetFrameFillStatus().Nchars) {
			t.fr.InsertByte(b, q0-t.org)
		}
	}

	t.logInsert(oq0, b, nr)
	// TODO(rjk): The below should only be invoked once (at the end) of a
	// sequence of modifications to the file.Buffer, not here per action.
	t.SetSelect(t.q0, t.q1)
	if t.fr != nil && t.display != nil {
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
	}
}

// writeEventLog emits an event log for an insertion.
// TODO(rjk): Refactor this with the other event log insertions.
// TODO(rjk): can be more stateless.
// TODO(rjk): Can express this more precisely with an interface
// that makes its state dependency obvious
func (t *Text) logInsert(oq0 file.OffsetTuple, b []byte, nr int) {
	q0 := oq0.R
	if t.w != nil {
		c := 'i'
		if t.what == Body {
			c = 'I'
		}
		if nr <= EVENTSIZE {
			// TODO(rjk): Does unnecessary work making a string from r if there's no
			// event reader.
			t.w.Eventf("%c%d %d 0 %d %s\n", c, q0, q0+nr, nr, b)
		} else {
			t.w.Eventf("%c%d %d 0 0 \n", c, q0, q0+nr)
		}
	}
}

// Insert inserts rune buffer r at q0. The selection values will be
// updated appropriately.
func (t *Text) Insert(q0 int, r []rune, tofile bool) {
	if !tofile {
		panic("text.insert")
	}
	if len(r) == 0 {
		return
	}
	t.file.InsertAt(q0, r)
}

func (t *Text) TypeCommit() {
	if t.w != nil {
		t.w.Commit(t)
	} else {
		t.Commit()
	}
}

func (t *Text) inSelection(q0 int) bool {
	return t.q1 > t.q0 && t.q0 <= q0 && q0 <= t.q1
}

// Fill inserts additional text from t into the Frame object until the Frame object is full.
func (t *Text) fill(fr frame.SelectScrollUpdater) error {
	// log.Println("Text.Fill Start", t.what)
	// defer log.Println("Text.Fill End")

	// Conceivably, LastLineFull should be true or would it only be true if there are no more
	// characters possible?
	if fr.IsLastLineFull() || t.nofill {
		return nil
	}
	for {
		n := t.file.Nr() - (t.org + fr.GetFrameFillStatus().Nchars)
		if n < 0 {
			log.Printf("Text.fill: negative slice length %v (file size %v, t.org %v, frame nchars %v)\n",
				n, t.file.Nr(), t.org, fr.GetFrameFillStatus().Nchars)
			return fmt.Errorf("fill: negative slice length %v", n)
		}
		if n == 0 {
			break
		}
		if n > 2000 { // educated guess at reasonable amount
			n = 2000
		}
		rp := make([]rune, n)
		t.file.Read(t.org+fr.GetFrameFillStatus().Nchars, rp)
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
	return nil
}

// Delete removes runes [q0, q1). The selection values will be
// updated appropriately.
func (t *Text) Delete(q0, q1 int, _ bool) {
	n := q1 - q0
	if n == 0 {
		return
	}
	t.file.DeleteAt(q0, q1)
}

// deleted implements the single-text deletion observer for this Text's
// backing File. It updates the Text (i.e. the view) for the removal of
// runes [q0, q1).
func (t *Text) Deleted(oq0, oq1 file.OffsetTuple) {
	q0 := oq0.R
	q1 := oq1.R

	n := q1 - q0
	if t.what == Body {
		t.w.utflastqid = -1
	}
	if q0 < t.iq1 {
		t.iq1 -= util.Min(n, t.iq1-q0)
	}
	if q0 < t.q0 {
		t.q0 -= util.Min(n, t.q0-q0)
	}
	if q0 < t.q1 {
		t.q1 -= util.Min(n, t.q1-q0)
	}

	// In Markdeep mode, don't update the text frame directly.
	// Instead, schedule a debounced re-render of the Markdeep view.
	if t.what == Body && t.w != nil && t.w.IsPreviewMode() {
		t.logInsertDelete(q0, q1)
		t.w.recordEdit(markdown.EditRecord{Pos: q0, OldLen: q1 - q0, NewLen: 0})
		t.w.SchedulePreviewUpdate()
		return
	}

	if q1 <= t.org {
		t.org -= n
	} else if t.fr != nil && q0 < t.org+(t.fr.GetFrameFillStatus().Nchars) {
		p1 := q1 - t.org
		if p1 > (t.fr.GetFrameFillStatus().Nchars) {
			p1 = t.fr.GetFrameFillStatus().Nchars
		}
		p0 := 0
		if q0 < t.org {
			t.org = q0
			p0 = 0
		} else {
			p0 = q0 - t.org
		}
		t.fr.Delete(p0, p1)
		t.fill(t.fr)
	}

	t.logInsertDelete(q0, q1)

	t.SetSelect(t.q0, t.q1)
	if t.fr != nil && t.display != nil {
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
	}
}

// TODO(rjk): Fold this into logInsert is a nice way.
func (t *Text) logInsertDelete(q0, q1 int) {
	if t.w != nil {
		c := 'd'
		if t.what == Body {
			c = 'D'
		}
		t.w.Eventf("%c%d %d 0 0 \n", c, q0, q1)
	}
}

func (t *Text) ReadB(q int, r []rune) (n int, err error) { n, err = t.file.Read(q, r); return }
func (t *Text) nc() int                                  { return t.file.Nr() }
func (t *Text) Q0() int                                  { return t.q0 }
func (t *Text) Q1() int                                  { return t.q1 }
func (t *Text) SetQ0(q0 int)                             { t.q0 = q0 }
func (t *Text) SetQ1(q1 int)                             { t.q1 = q1 }
func (t *Text) Constrain(q0, q1 int) (p0, p1 int) {
	p0 = util.Min(q0, t.file.Nr())
	p1 = util.Min(q1, t.file.Nr())
	return p0, p1
}

func (t *Text) BsWidth(c rune) int {
	// there is known to be at least one character to erase
	if c == 0x08 { // ^H: erase character
		return 1
	}
	q := t.q0
	skipping := true
	for q > 0 {
		r := t.file.ReadC(q - 1)
		if r == '\n' { // eat at most one more character
			if q == t.q0 { // eat the newline
				q--
			}
			break
		}
		if c == 0x17 {
			eq := isalnum(r)
			if eq && skipping { // found one; stop skipping
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
		r := t.file.ReadC(q - 1)
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
	if t.q0 < t.Nc() && t.file.ReadC(t.q0) > ' ' { // must be at end of word
		return nil
	}
	str := make([]rune, t.FileWidth(t.q0, true))
	q := t.q0 - len(str)
	for i := range str {
		str[i] = t.file.ReadC(q)
		q++
	}
	path := make([]rune, t.FileWidth(t.q0-len(str), false))
	q = t.q0 - len(str) - len(path)
	for i := range path {
		path[i] = t.file.ReadC(q)
		q++
	}

	// is path rooted? if not, we need to make it relative to window path
	dir := string(path)
	if !filepath.IsAbs(dir) {
		dir = t.DirName("")
		if len(dir) == 0 {
			dir = Ldot
		}
		dir = filepath.Clean(filepath.Join(dir, string(path)))
	}

	c, err := complete.Complete(dir, string(str))
	if err != nil {
		warning(nil, "error attempting completion: %v\n", err)
		return nil
	}
	if c.Advance {
		return []rune(c.String)
	}
	var b bytes.Buffer
	b.WriteString(dir)
	if len(dir) > 0 && dir[len(dir)-1] != filepath.Separator {
		b.WriteRune(filepath.Separator)
	}
	b.WriteString(string(str) + "*")
	if c.NMatch == 0 {
		b.WriteString(": no matches in:")
	}
	warning(nil, "%s\n", b.String())
	for _, fn := range c.Filename {
		warning(nil, " %s\n", fn)
	}
	return nil
}

func (t *Text) Type(r rune) {
	var (
		q0, q1    int
		nnb, n, i int
		nr        int
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
		// expand tag to show all text
		if !t.w.tagexpand {
			t.w.tagexpand = true
			t.w.Resize(t.w.r, false, true)
		}
	}

	Tagup := func() {
		// shrink tag to single line
		if t.w.tagexpand {
			t.w.tagexpand = false
			t.w.taglines = 1
			t.w.Resize(t.w.r, false, true)
		}
	}

	caseDown := func() {
		q0 = t.org + t.fr.Charofpt(image.Pt(t.fr.Rect().Min.X, t.fr.Rect().Min.Y+n*t.fr.DefaultFontHeight()))
		t.SetOrigin(q0, true)
	}
	caseUp := func() {
		q0 = t.BackNL(t.org, n)
		t.SetOrigin(q0, true)
	}

	setUndoPoint := func() {
		if t.what == Body {
			global.seq++
			t.file.Mark(global.seq)
		}
	}

	// This switch block contains all actions that don't mutate the buffer
	// and hence there is no need to create an Undo record.
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
		if t.q1 < t.file.Nr() {
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
	case draw.KeyDown, 0xF800:
		if t.what == Tag {
			Tagdown()
			return
		}
		n = t.fr.GetFrameFillStatus().Maxlines / 3
		caseDown()
		return
	case Kscrollonedown:
		if t.what == Tag {
			Tagdown()
			return
		}
		n = drawutil.MouseScrollSize(t.fr.GetFrameFillStatus().Maxlines)
		if n <= 0 {
			n = 1
		}
		caseDown()
		return
	case draw.KeyPageDown:
		n = 2 * t.fr.GetFrameFillStatus().Maxlines / 3
		caseDown()
		return
	case draw.KeyUp:
		if t.what == Tag {
			Tagup()
			return
		}
		n = t.fr.GetFrameFillStatus().Maxlines / 3
		caseUp()
		return
	case Kscrolloneup:
		if t.what == Tag {
			Tagup()
			return
		}
		n = drawutil.MouseScrollSize(t.fr.GetFrameFillStatus().Maxlines)
		caseUp()
		return
	case draw.KeyPageUp:
		n = 2 * t.fr.GetFrameFillStatus().Maxlines / 3
		caseUp()
		return
	case draw.KeyHome:
		t.TypeCommit()
		if t.org > t.iq1 {
			q0 = t.BackNL(t.iq1, 1)
			t.SetOrigin(q0, true)
		} else {
			t.Show(0, 0, false)
		}
		return
	case draw.KeyEnd:
		t.TypeCommit()
		if t.iq1 > t.org+t.fr.GetFrameFillStatus().Nchars {
			if t.iq1 > t.file.Nr() {
				// should not happen, but does. and it will crash textbacknl.
				t.iq1 = t.file.Nr()
			}
			q0 = t.BackNL(t.iq1, 1)
			t.SetOrigin(q0, true)
		} else {
			t.Show(t.file.Nr(), t.file.Nr(), false)
		}
		return
	case '\t': // ^I (TAB)
		if t.tabexpand {
			for i := 0; i < t.tabstop; i++ {
				t.Type(' ')
			}
			return
		}
	case 0x01: // ^A: beginning of line
		t.TypeCommit()
		// go to where ^U would erase, if not already at BOL
		nnb = 0
		if t.q0 > 0 && t.file.ReadC(t.q0-1) != '\n' {
			nnb = t.BsWidth(0x15)
		}
		t.Show(t.q0-nnb, t.q0-nnb, true)
		return
	case 0x05: // ^E: end of line
		t.TypeCommit()
		q0 = t.q0
		for q0 < t.file.Nr() && t.file.ReadC(q0) != '\n' {
			q0++
		}
		t.Show(q0, q0, true)
		return
	case 0x3, draw.KeyCmd + 'c': // %C: copy
		t.TypeCommit()
		cut(t, t, nil, true, false, "")
		return
	case 0x1a, draw.KeyCmd + 'z': // %Z: undo
		t.TypeCommit()
		undo(t, nil, nil, true, false, "")
		return
	case draw.KeyCmd + 'Z': // %-shift-Z: redo
		t.TypeCommit()
		undo(t, nil, nil, false, false, "")
		return

	}

	// Note the use of eq0 to always force an undo point at the start typing.
	if t.what == Body && t.eq0 == -1 {
		setUndoPoint()
	}

	// These following blocks contain mutating actions.
	// cut/paste must be done after the seq++/filemark
	switch r {
	case 0x18, draw.KeyCmd + 'x': // %X: cut
		setUndoPoint()
		t.TypeCommit()
		if t.what == Body {
			global.seq++
			t.file.Mark(global.seq)
		}
		cut(t, t, nil, true, true, "")
		t.Show(t.q0, t.q0, true)
		t.iq1 = t.q0
		return
	case 0x16, draw.KeyCmd + 'v': // %V: paste
		setUndoPoint()
		t.TypeCommit()
		if t.what == Body {
			global.seq++
			t.file.Mark(global.seq)
		}
		paste(t, t, nil, true, false, "")
		t.Show(t.q0, t.q1, true)
		t.iq1 = t.q1
		return
	}
	wasrange := t.q0 != t.q1
	removedstuff := false
	if t.q1 > t.q0 {
		setUndoPoint()
		cut(t, t, nil, true, true, "")
		t.eq0 = ^0
		removedstuff = true
	}
	t.Show(t.q0, t.q0, true)
	switch r {
	case 0x06:
		fallthrough // ^F: complete
	case draw.KeyInsert:
		t.TypeCommit()
		rp = t.Complete()
		if rp == nil {
			return
		}
		setUndoPoint()
		nr = len(rp) // runestrlen(rp);
		// break into normal insertion case
	case 0x1B:
		if t.eq0 != ^0 {
			if t.eq0 <= t.q0 {
				t.SetSelect(t.eq0, t.q0)
			} else {
				t.SetSelect(t.q0, t.eq0)
			}
		}
		t.iq1 = t.q0
		return
	case 0x7F: // Del: erase character right
		if t.q1 >= t.Nc()-1 {
			return // End of file
		}
		setUndoPoint()
		t.TypeCommit() // Avoid messing with the cache?
		if !wasrange {
			t.q1++
			cut(t, t, nil, false, true, "")
		}
		return
	case 0x08:
		fallthrough // ^H: erase character
	case 0x15:
		fallthrough // ^U: erase line
	case 0x17: // ^W: erase word
		if removedstuff {
			// No further action needed.
			return
		}

		if t.q0 == 0 { // nothing to erase
			return
		}

		nnb = t.BsWidth(r)
		q1 = t.q0
		q0 = q1 - nnb
		// if selection is at beginning of window, avoid deleting invisible text
		if q0 < t.org {
			q0 = t.org
			nnb = q1 - q0
		}
		if nnb <= 0 {
			return
		}

		setUndoPoint()
		t.Delete(q0, q0+nnb, true)

		// Run through the code that will update the t.w.body.file.details.Name.
		// TODO(rjk): I'm not consistent in when I call this. Perhaps I should figure that out.
		t.TypeCommit()

		t.iq1 = t.q0
		return
	case '\n':
		setUndoPoint()
		if t.w.autoindent {
			// find beginning of previous line using backspace code
			nnb = t.BsWidth(0x15)    // ^U case
			rp = make([]rune, nnb+1) //runemalloc(nnb + 1);
			nr = 0
			rp[nr] = r
			nr++
			for i = 0; i < nnb; i++ {
				r = t.file.ReadC(t.q0 - nnb + i)
				if r != ' ' && r != '\t' {
					break
				}
				rp[nr] = r
				nr++
			}
			rp = rp[:nr]
		}
	}
	// Otherwise ordinary character; just insert it.
	t.file.InsertAt(t.q0, rp[:nr])
	t.SetSelect(t.q0+nr, t.q0+nr)

	// Always commit if the typing is into a tag. The reason to do this is to
	// be sure to invoke the special logic in Window.Commit() that creates an
	// undo point for a file name change and updates the file details.
	//
	// This doesn't seem ideal. We have subtle logic that spans layers. Can
	// we clean this up in some fashion so that it's easier to have Text
	// instances that are editable but have partial auto-generated semantics
	// (e.g. directories, tags)
	//
	// NB: Window.Commit is about updating the tag logic.
	if t.w != nil && (r == '\n' && t.what == Body || t.what != Body) {
		t.w.Commit(t)
	}
	t.iq1 = t.q0
}

func (t *Text) Commit() {
	if t.what == Body {
		t.w.utflastqid = -1
	}
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
	if dl == 0 {
		// TODO(rjk): Make this mechanism better? It seems unfortunate.
		ScrSleep(100)
		return
	}
	var q0 int
	if dl < 0 {
		q0 = t.BackNL(t.org, -dl)
	} else {
		if t.org+(fr.GetFrameFillStatus().Nchars) == t.file.Nr() {
			return
		}
		q0 = t.org + fr.Charofpt(image.Pt(fr.Rect().Min.X, fr.Rect().Min.Y+dl*fr.DefaultFontHeight()))
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

	// To have double-clicking and chording, we double-click
	// immediately if it might make sense.
	b := global.mouse.Buttons
	q0 := t.q0
	q1 := t.q1
	selectq = t.org + t.fr.Charofpt(global.mouse.Point)
	//	fmt.Printf("Text.Select: mouse.Msec %v, clickmsec %v\n", mouse.Msec, clickmsec)
	//	fmt.Printf("clicktext==t %v, (q0==q1 && selectq==q0): %v", clicktext == t, q0 == q1 && selectq == q0)
	if (clicktext == t && global.mouse.Msec-clickmsec < 500) && (q0 == q1 && selectq == q0) {
		q0, q1 = t.DoubleClick(q0, q1)
		t.SetSelect(q0, q1)
		t.display.Flush()
		x := global.mouse.Point.X
		y := global.mouse.Point.Y
		// stay here until something interesting happens
		// TODO(rjk): Ack. This is horrible? Layering violation?
		for {
			global.mousectl.Read()
			if !(global.mouse.Buttons == b && util.Abs(global.mouse.Point.X-x) < 3 && util.Abs(global.mouse.Point.Y-y) < 3) {
				break
			}
		}
		global.mouse.Point.X = x // in case we're calling frselect
		global.mouse.Point.Y = y
		q0 = t.q0 // may have changed
		q1 = t.q1
		selectq = q0
	}
	if global.mouse.Buttons == b {
		sP0, sP1 := t.fr.Select(global.mousectl, global.mouse, func(fr frame.SelectScrollUpdater, dl int) { t.FrameScroll(fr, dl) })

		// horrible botch: while asleep, may have lost selection altogether
		if selectq > t.file.Nr() {
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
		if q0 == t.q0 && clicktext == t && global.mouse.Msec-clickmsec < 500 {
			q0, q1 = t.DoubleClick(q0, q1)
			clicktext = nil
		} else {
			clicktext = t
			clickmsec = global.mouse.Msec
		}
	} else {
		clicktext = nil
	}
	t.SetSelect(q0, q1)
	t.display.Flush()
	state := None // what we've done; undo when possible
	for global.mouse.Buttons != 0 {
		global.mouse.Msec = 0
		b := global.mouse.Buttons
		if (b&1) != 0 && (b&6) != 0 {
			if state == None && t.what == Body {
				global.seq++
				t.w.body.file.Mark(global.seq)
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
		for global.mouse.Buttons == b {
			global.mousectl.Read()
		}
		clicktext = nil
	}
}

func (t *Text) Show(q0, q1 int, doselect bool) {
	t.lk.Lock()
	defer t.lk.Unlock()
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
	// In preview mode, update the logical selection (q0/q1) but suppress
	// DrawSel() and scroll operations on the source body frame. This prevents
	// the source frame from bleeding through the preview rendering.
	if t.w != nil && t.w.IsPreviewMode() {
		if doselect {
			t.q0 = q0
			t.q1 = q1
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
	tsd = false // do we call ScrDraw?
	nc = t.file.Nr()
	if t.org <= q0 {
		if nc == 0 || q0 < qe {
			tsd = true
		} else {
			if q0 == qe && qe == nc {
				if t.file.ReadC(nc-1) == '\n' {
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
		q = t.BackNL(q0, nl)
		// avoid going backwards if trying to go forwards - long lines!
		if !(q0 > t.org && q < t.org) {
			t.SetOrigin(q, true)
		}
		for q0 > t.org+t.fr.GetFrameFillStatus().Nchars {
			t.SetOrigin(t.org+1, false)
		}
	}
}

// TODO(rjk): remove me in a subsequent CL.
func (t *Text) ReadC(q int) rune {
	return t.file.ReadC(q)
}

func (t *Text) SetSelect(q0, q1 int) {
	// log.Println("Text SetSelect Start", q0, q1)
	// defer log.Println("Text SetSelect End", q0, q1)

	t.q0 = q0
	t.q1 = q1
	// compute desired p0,p1 from q0,q1
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
		p0 = t.fr.GetFrameFillStatus().Nchars
	}
	if p1 > (t.fr.GetFrameFillStatus().Nchars) {
		p1 = t.fr.GetFrameFillStatus().Nchars
	}
	if p0 > p1 {
		panic(fmt.Sprintf("acme: textsetselect p0=%d p1=%d q0=%v q1=%v t.org=%d nchars=%d", p0, p1, q0, q1, t.org, t.fr.GetFrameFillStatus().Nchars))
	}

	t.fr.DrawSel(t.fr.Ptofchar(p0), p0, p1, ticked)
}

// TODO(rjk): The implicit initialization of q0, q1 doesn't seem like very nice
// style? Maybe it is idiomatic?
func (t *Text) Select23(high draw.Image, mask uint) (q0, q1 int, buts uint) {
	p0, p1 := t.fr.SelectOpt(global.mousectl, global.mouse, func(frame.SelectScrollUpdater, int) {}, t.display.White(), high)

	buts = uint(global.mousectl.Mouse.Buttons)
	if (buts & mask) == 0 {
		q0 = p0 + t.org
		q1 = p1 + t.org
	}
	for global.mousectl.Mouse.Buttons != 0 {
		global.mousectl.Read()
	}
	return q0, q1, buts
}

func (t *Text) Select2() (q0, q1 int, tp *Text, ret bool) {
	q0, q1, buts := t.Select23(global.but2col, 4)
	if (buts & 4) != 0 {
		return q0, q1, nil, false
	}
	if (buts & 1) != 0 { // pick up argument
		return q0, q1, global.argtext, true
	}
	return q0, q1, nil, true
}

func (t *Text) Select3() (q0, q1 int, r bool) {
	q0, q1, buts := t.Select23(global.but3col, 1|2)
	return q0, q1, buts == 0
}

func (t *Text) DoubleClick(inq0, inq1 int) (q0, q1 int) {
	q0 = inq0
	q1 = inq1
	if q0, q1, ok := t.ClickHTMLMatch(inq0); ok {
		return q0, q1
	}
	var c rune
	for i, l := range left {
		q := inq0
		r := right[i]
		// try matching character to left, looking right
		if q == 0 {
			c = '\n'
		} else {
			c = t.file.ReadC(q - 1)
		}
		p := runes.IndexRune(l, c)
		if p != -1 {
			if q, ok := t.ClickMatch(c, r[p], 1, q); ok {
				q1 = q
				if c != '\n' {
					q1--
				}
			}
			return
		}
		// try matching character to right, looking left
		if q == t.file.Nr() {
			c = '\n'
		} else {
			c = t.file.ReadC(q)
		}
		p = runes.IndexRune(r, c)
		if p != -1 {
			if q, ok := t.ClickMatch(c, l[p], -1, q); ok {
				q1 = inq0
				if q0 < t.file.Nr() && c == '\n' {
					q1++
				}
				q0 = q
				if c != '\n' || q != 0 || t.file.ReadC(0) == '\n' {
					q0++
				}
			}
			return
		}
	}
	// try filling out word to right
	q1 = inq0
	for q1 < t.file.Nr() && isalnum(t.file.ReadC(q1)) {
		q1++
	}
	// try filling out word to left
	for q0 > 0 && isalnum(t.file.ReadC(q0-1)) {
		q0--
	}

	return q0, q1
}

func (t *Text) ClickMatch(cl, cr rune, dir int, inq int) (q int, r bool) {
	nest := 1
	var c rune
	for {
		if dir > 0 {
			if inq == t.file.Nr() {
				break
			}
			c = t.file.ReadC(inq)
			inq++
		} else {
			if inq == 0 {
				break
			}
			inq--
			c = t.file.ReadC(inq)
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

// ishtmlstart checks whether the text starting at location q an html tag.
// Returned stat is 1 for <a>, -1 for </a>, 0 for no tag or <a />.
// Returned q1 is the location after the tag.
func (t *Text) ishtmlstart(q int) (q1 int, stat int) {
	if q+2 > t.file.Nr() {
		return 0, 0
	}
	if t.file.ReadC(q) != '<' {
		return 0, 0
	}
	q++
	c := t.file.ReadC(q)
	q++
	c1 := c
	c2 := c
	for c != '>' {
		if q >= t.file.Nr() {
			return 0, 0
		}
		c2 = c
		c = t.file.ReadC(q)
		q++
	}
	if c1 == '/' { // closing tag
		return q, -1
	}
	if c2 == '/' || c2 == '!' { // open + close tag or comment
		return 0, 0
	}
	return q, 1
}

// ishtmlend checks whether the text ending at location q an html tag.
// Returned stat is 1 for <a>, -1 for </a>, 0 for no tag or <a />.
// Returned q0 is the start of the tag.
func (t *Text) ishtmlend(q int) (q1 int, stat int) {
	if q < 2 {
		return 0, 0
	}
	q--
	if t.file.ReadC(q) != '>' {
		return 0, 0
	}
	q--
	c := t.file.ReadC(q)
	c1 := c
	c2 := c
	for c != '<' {
		if q == 0 {
			return 0, 0
		}
		c1 = c
		q--
		c = t.file.ReadC(q)
	}
	if c1 == '/' { // closing tag
		return q, -1
	}
	if c2 == '/' || c2 == '!' { // open + close tag or comment
		return 0, 0
	}
	return q, 1
}

func (t *Text) ClickHTMLMatch(inq0 int) (q0, q1 int, r bool) {
	q0 = inq0
	q1 = inq0

	// after opening tag?  scan forward for closing tag
	if _, stat := t.ishtmlend(q0); stat == 1 {
		depth := 1
		q := q1
		for q < t.file.Nr() {
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
	if _, stat := t.ishtmlstart(q1); stat == -1 {
		depth := -1
		q := q0
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

// BackNL returns the position at the beginning of the line
// after backing up n lines starting from position p.
func (t *Text) BackNL(p, n int) int {
	// look for start of this line if n==0
	if n == 0 && p > 0 && t.file.ReadC(p-1) != '\n' {
		n = 1
	}
	for n > 0 && p > 0 {
		n--
		p-- // it's at a newline now; back over it
		if p == 0 {
			break
		}
		// at 128 chars, call it a line anyway
		for j := 128; j > 0 && p > 0; p-- {
			if t.file.ReadC(p-1) == '\n' {
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
	if org > 0 && !exact && t.file.ReadC(org-1) != '\n' {
		// org is an estimate of the char posn; find a newline
		// don't try harder than 256 chars
		for i = 0; i < 256 && org < t.file.Nr(); i++ {
			if t.file.ReadC(org) == '\n' {
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
		if a < 0 && -a < fr.GetFrameFillStatus().Nchars {
			n = t.org - org
			r = make([]rune, n)
			t.file.Read(org, r)
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
	t.eq0 = ^0
	t.fr.Delete(0, t.fr.GetFrameFillStatus().Nchars)
	t.org = 0
	t.q0 = 0
	t.q1 = 0
	t.file.ResetBuffer()
}

// TODO(rjk): Is this method on the right object. It reaches into Window
// for nearly every reference to t. Assess how DirName is used and adjust
// appropriately.
func (t *Text) dirName(name string) string {
	if t == nil || t.w == nil || filepath.IsAbs(name) {
		return name
	}
	nt := t.w.tag.file.Nr()
	if nt == 0 {
		return name
	}
	spl := t.w.ParseTag()

	if !strings.HasSuffix(spl, string(filepath.Separator)) {
		spl = filepath.Dir(spl)
	}
	return filepath.Join(spl, name)
}

// DirName returns the directory name of the path in the tag file of t.
// The filename name is appended to the result.
// The returned path is guaranteed to be cleaned, as specified by filepath.Clean.
func (t *Text) DirName(name string) string {
	return filepath.Clean(t.dirName(name))
}

// AbsDirName is the same as DirName but always returns an absolute path.
func (t *Text) AbsDirName(name string) string {
	d := t.dirName(name)
	if !filepath.IsAbs(d) {
		return filepath.Join(global.wdir, d)
	}
	return filepath.Clean(d)
}

// DebugString provides a Text representation convenient for logging for
// debugging.
func (t *Text) DebugString() string {
	return fmt.Sprintf("t.what (kind): %s contents: %q", t.what, t.file.String())
}

// TODO(PAL): This probably wants to check for ' in the filename
// and escape it - that would mean understanding \ escapes in Look.
func QuoteFilename(name string) string {
	if strings.ContainsAny(name, " \t") {
		return "'" + name + "'"
	}
	return name
}

func UnquoteFilename(s string) string {
	if len(s) > 0 && s[0] == '\'' {
		if s[len(s)-1] == '\'' {
			return s[1 : len(s)-1]
		}
	}
	return s
}
