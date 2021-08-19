package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/runes"
	"github.com/rjkroege/edwood/util"
)

type Window struct {
	display draw.Display
	lk      sync.Mutex
	ref     Ref
	tag     Text
	body    Text
	r       image.Rectangle

	//	isdir      bool // true if this Window is showing a directory in its body.
	filemenu   bool
	autoindent bool
	showdel    bool

	id    int
	addr  Range
	limit Range

	nopen      [QMAX]byte // number of open Fid for each file in the file server
	nomark     bool
	wrselrange Range
	rdselfd    *os.File // temporary file for rdsel read requests

	col    *Column
	eventx *Xfid
	events []byte

	owner       int // TODO(fhs): change type to rune
	maxlines    int
	dirnames    []string
	widths      []int
	incl        []string
	ctrllock    sync.Mutex // used for lock/unlock ctl mesage
	ctlfid      uint32     // ctl file Fid which has the ctrllock
	dumpstr     string
	dumpdir     string
	utflastqid  int    // Qid of last read request (QWbody or QWtag)
	utflastboff uint64 // Byte offset of last read of body or tag
	utflastq    int    // Rune offset of last read of body or tag
	tagsafe     bool
	tagexpand   bool
	taglines    int
	tagtop      image.Rectangle
	editoutlk   chan bool
}

func NewWindow() *Window {
	return &Window{}
}

// Initialize the headless parts of the window.
func (w *Window) initHeadless(clone *Window) *Window {
	w.tag.w = w
	w.taglines = 1
	w.tagsafe = false
	w.tagexpand = true
	w.body.w = w
	w.incl = []string{}
	WinID++
	w.id = WinID
	w.ref.Inc()
	if globalincref {
		w.ref.Inc()
	}

	w.ctlfid = MaxFid
	w.utflastqid = -1

	f := file.MakeObservableEditableBufferTag(nil)
	f.AddObserver(&w.tag)
	w.tag.file = f

	// Body setup.
	f = file.MakeObservableEditableBuffer("", nil)
	if clone != nil {
		f = clone.body.file
		w.body.org = clone.body.org
	}
	f.AddObserver(&w.body)
	w.body.file = f
	w.filemenu = true
	w.autoindent = *globalAutoIndent

	if clone != nil {
		w.autoindent = clone.autoindent
	}
	w.editoutlk = make(chan bool, 1)
	return w
}

func (w *Window) Init(clone *Window, r image.Rectangle, dis draw.Display) {
	w.initHeadless(clone)
	w.display = dis
	r1 := r

	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + fontget(tagfont, w.display).Height()
	r1.Max.Y = r1.Min.Y + w.taglines*fontget(tagfont, w.display).Height()

	w.tag.Init(r1, tagfont, tagcolors, w.display)
	w.tag.what = Tag

	// tag is a copy of the contents, not a tracked image
	if clone != nil {
		w.tag.Delete(0, w.tag.Nc(), true)
		// TODO(sn0w): find a nicer way to do this.
		clonebuff := []rune(clone.tag.file.String())
		w.tag.Insert(0, clonebuff, true)
		w.tag.file.Reset()
		w.tag.SetSelect(w.tag.file.Nbyte(), w.tag.file.Nbyte())
	}
	r1 = r
	r1.Min.Y += w.taglines*fontget(tagfont, w.display).Height() + 1
	if r1.Max.Y < r1.Min.Y {
		r1.Max.Y = r1.Min.Y
	}

	var rf string
	if clone != nil {
		rf = clone.body.font
	} else {
		rf = tagfont
	}
	w.body.Init(r1, rf, textcolors, w.display)
	w.body.what = Body
	r1.Min.Y--
	r1.Max.Y = r1.Min.Y + 1
	if w.display != nil {
		w.display.ScreenImage().Draw(r1, tagcolors[frame.ColBord], nil, image.Point{})
	}
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.r = r
	var br image.Rectangle
	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + button.R().Dx()
	br.Max.Y = br.Min.Y + button.R().Dy()
	if w.display != nil {
		w.display.ScreenImage().Draw(br, button, nil, button.R().Min)
	}
	w.maxlines = w.body.fr.GetFrameFillStatus().Maxlines
	if clone != nil {
		w.body.SetSelect(clone.body.q0, clone.body.q1)
		w.SetTag()
	}
}

func (w *Window) DrawButton() {
	b := button
	if w.body.file.SaveableAndDirty() {
		b = modbutton
	}
	var br image.Rectangle
	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + b.R().Dx()
	br.Max.Y = br.Min.Y + b.R().Dy()
	if w.display != nil {
		w.display.ScreenImage().Draw(br, b, nil, b.R().Min)
	}
}

func (w *Window) delRunePos() int {
	i := len([]rune(w.ParseTag())) + 2
	if i >= w.tag.Nc() {
		return -1
	}
	return i
}

func (w *Window) moveToDel() {
	n := w.delRunePos()
	if n < 0 {
		return
	}
	if w.display != nil {
		w.display.MoveTo(w.tag.fr.Ptofchar(n).Add(image.Pt(4, w.tag.fr.DefaultFontHeight()-4)))
	}
}

// TagLines computes the number of lines in the tag that can fit in r.
func (w *Window) TagLines(r image.Rectangle) int {
	if !w.tagexpand && !w.showdel {
		return 1
	}
	w.showdel = false
	w.tag.Resize(r, true, true /* noredraw */)
	w.tagsafe = false

	if !w.tagexpand {
		// use just as many lines as needed to show the Del
		n := w.delRunePos()
		if n < 0 {
			return 1
		}
		p := w.tag.fr.Ptofchar(n).Sub(w.tag.fr.Rect().Min)
		return 1 + p.Y/w.tag.fr.DefaultFontHeight()
	}

	// can't use more than we have
	if w.tag.fr.GetFrameFillStatus().Nlines >= w.tag.fr.GetFrameFillStatus().Maxlines {
		return w.tag.fr.GetFrameFillStatus().Maxlines
	}

	// if tag ends with \n, include empty line at end for typing
	n := w.tag.fr.GetFrameFillStatus().Nlines
	if w.tag.file.Size() > 0 {
		c := w.tag.file.ReadC(w.tag.file.Size() - 1)
		if c == '\n' {
			n++
		}
	}
	if n == 0 {
		n = 1
	}
	return n
}

// Resize the specified Window to rectangle r.
// TODO(rjk): when collapsing the tag, this is called twice. Once would seem
// sufficient.
// TODO(rjk): This function does not appear to update the Window's rect correctly
// in all cases.
func (w *Window) Resize(r image.Rectangle, safe, keepextra bool) int {
	// log.Printf("Window.Resize r=%v safe=%v keepextra=%v\n", r, safe, keepextra)
	// defer log.Println("Window.Resize End\n")

	// TODO(rjk): Do not leak global event state into this function.
	mouseintag := mouse.Point.In(w.tag.all)
	mouseinbody := mouse.Point.In(w.body.all)

	// Tagtop is a rectangle corresponding to one line of tag.
	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + fontget(tagfont, w.display).Height()

	r1 := r
	r1.Max.Y = util.Min(r.Max.Y, r1.Min.Y+w.taglines*fontget(tagfont, w.display).Height())

	// If needed, recompute number of lines in tag.
	if !safe || !w.tagsafe || !w.tag.all.Eq(r1) {
		w.taglines = w.TagLines(r)
		r1.Max.Y = util.Min(r.Max.Y, r1.Min.Y+w.taglines*fontget(tagfont, w.display).Height())
	}

	// Resize/redraw tag TODO(flux)
	y := r1.Max.Y
	if !safe || !w.tagsafe || !w.tag.all.Eq(r1) {
		w.tag.Resize(r1, true, false /* noredraw */)
		y = w.tag.fr.Rect().Max.Y
		w.DrawButton()
		w.tagsafe = true

		// If mouse is in tag, pull up as tag closes.
		if mouseintag && !mouse.Point.In(w.tag.all) {
			p := mouse.Point
			p.Y = w.tag.all.Max.Y - 3
			if w.display != nil {
				w.display.MoveTo(p)
			}
		}
		// If mouse is in body, push down as tag expands.
		if mouseinbody && mouse.Point.In(w.tag.all) {
			p := mouse.Point
			p.Y = w.tag.all.Max.Y + 3
			if w.display != nil {
				w.display.MoveTo(p)
			}
		}
	}
	// Redraw body
	r1 = r
	r1.Min.Y = y
	if !safe || !w.body.all.Eq(r1) {
		oy := y
		if y+1+w.body.fr.DefaultFontHeight() <= r.Max.Y { // room for one line
			r1.Min.Y = y
			r1.Max.Y = y + 1
			if w.display != nil {
				w.display.ScreenImage().Draw(r1, tagcolors[frame.ColBord], nil, image.Point{})
			}
			y++
			r1.Min.Y = util.Min(y, r.Max.Y)
			r1.Max.Y = r.Max.Y
		} else {
			r1.Min.Y = y
			r1.Max.Y = y
		}
		y = w.body.Resize(r1, keepextra, false /* noredraw */)
		w.r = r
		w.r.Max.Y = y
		w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
		w.body.all.Min.Y = oy
	}
	w.maxlines = util.Min(w.body.fr.GetFrameFillStatus().Nlines, util.Max(w.maxlines, w.body.fr.GetFrameFillStatus().Maxlines))
	// TODO(rjk): this value doesn't make sense when we've collapsed
	// the tag if the rectangle update block is not executed.
	return w.r.Max.Y
}

// Lock1 locks just this Window. This is a helper for Lock.
// TODO(rjk): This should be an internal detail of Window.
func (w *Window) lock1(owner int) {
	w.lk.Lock()
	w.ref.Inc()
	w.owner = owner
}

// Lock locks every text/clone of w
func (w *Window) Lock(owner int) {
	w.lk.Lock()
	w.ref.Inc()
	w.owner = owner
	f := w.body.file
	f.AllObservers(func(i interface{}) {
		t := i.(*Text)
		if t.w != w {
			t.w.lock1(owner)
		}
	})
}

// unlock1 unlocks a single window.
func (w *Window) unlock1() {
	w.owner = 0
	w.Close()
	w.lk.Unlock()
}

// Unlock releases the lock on each clone of w
func (w *Window) Unlock() {
	w.body.file.AllObservers(func(i interface{}) {
		t := i.(*Text)
		if t.w != w {
			t.w.unlock1()
		}
	})
	w.unlock1()
}

func (w *Window) MouseBut() {
	if w.display != nil {
		w.display.MoveTo(w.tag.scrollr.Min.Add(
			image.Pt(w.tag.scrollr.Dx(), fontget(tagfont, w.display).Height()).Div(2)))
	}
}

//func (w *Window) DirFree() {
//	// TODO(rjk): This doesn't actually free the memory
//	w.dirnames = w.dirnames[0:0]
//	w.widths = w.widths[0:0]
//}

func (w *Window) Close() {
	if w.ref.Dec() == 0 {
		xfidlog(w, "del")
		//		w.DirFree()
		w.tag.Close()
		w.body.Close()
		if activewin == w {
			activewin = nil
		}
	}
}

func (w *Window) Delete() {
	x := w.eventx
	if x != nil {
		w.events = w.events[0:0]
		w.eventx = nil
		x.c <- nil // wake him up
	}
}

func (w *Window) Undo(isundo bool) {
	w.utflastqid = -1
	body := &w.body
	if q0, q1, ok := body.file.Undo(isundo); ok {
		body.q0, body.q1 = q0, q1
	}

	// TODO(rjk): Is this absolutely essential.
	body.Show(body.q0, body.q1, true)

	w.SetTag()
}

func (w *Window) SetName(name string) {
	t := &w.body
	t.file.SetName(name)

	w.SetTag()
}

func (w *Window) Type(t *Text, r rune) {
	t.Type(r)
	w.SetTag()
}

func (w *Window) ClearTag() {
	// w must be committed
	n := w.tag.Nc()
	r := make([]rune, n)
	w.tag.file.Read(0, r)
	i := len([]rune(w.ParseTag()))
	for ; i < n; i++ {
		if r[i] == '|' {
			break
		}
	}
	if i == n {
		return
	}
	i++
	w.tag.Delete(i, n, true)
	w.tag.file.Clean()
	if w.tag.q0 > i {
		w.tag.q0 = i
	}
	if w.tag.q1 > i {
		w.tag.q1 = i
	}
	w.tag.SetSelect(w.tag.q0, w.tag.q1)
}

// ParseTag returns the filename in the window tag.
func (w *Window) ParseTag() string {
	r := make([]rune, w.tag.Nc())
	w.tag.file.Read(0, r)
	tag := string(r)

	// " |" or "\t|" ends left half of tag
	// If we find " Del Snarf" in the left half of the tag
	// (before the pipe), that ends the file name.
	pipe := strings.Index(tag, " |")
	if i := strings.Index(tag, "\t|"); i >= 0 && (pipe < 0 || i < pipe) {
		pipe = i
	}
	if i := strings.Index(tag, " Del Snarf"); i >= 0 && (pipe < 0 || i < pipe) {
		return tag[:i]
	}
	if i := strings.IndexAny(tag, " \t"); i >= 0 {
		return tag[:i]
	}
	return tag
}

// SetTag updates the tag for this Window and all of its clones.
func (w *Window) SetTag() {
	f := w.body.file
	f.AllObservers(func(i interface{}) {
		u := i.(*Text)
		if u.w.col.safe || u.fr.GetFrameFillStatus().Maxlines > 0 {
			u.w.setTag1()
		}
	})
}

// setTag1 updates the tag contents for a given window w.
func (w *Window) setTag1() {
	const (
		Ldelsnarf = " Del Snarf"
		Lundo     = " Undo"
		Lredo     = " Redo"
		Lget      = " Get"
		Lput      = " Put"
		Llook     = " Look"
		Ledit     = " Edit"
		Lpipe     = " |"
	)

	// (flux) The C implemtation does a lot of work to avoid
	// re-setting the tag text if unchanged.  That's probably not
	// relevant in the modern world.  We can build a new tag trivially
	// and put up with the traffic implied for a tag line.

	var sb strings.Builder
	sb.WriteString(w.body.file.Name())
	sb.WriteString(Ldelsnarf)

	if w.filemenu {
		if w.body.needundo || w.body.file.HasUndoableChanges() {
			sb.WriteString(Lundo)
		}
		if w.body.file.HasRedoableChanges() {
			sb.WriteString(Lredo)
		}
		if w.body.file.SaveableAndDirty() {
			sb.WriteString(Lput)
		}
	}
	if w.body.file.IsDir() {
		sb.WriteString(Lget)
	}
	old := w.tag.file.View(0, w.tag.file.Nbyte()) // TODO(sn0w) find another way to do this without using View.
	oldbarIndex := w.tag.file.IndexRune('|')
	if oldbarIndex >= 0 {
		sb.WriteString(" ")
		sb.WriteString(string(old[oldbarIndex:]))
	} else {
		sb.WriteString(Lpipe)
		sb.WriteString(Llook)
		sb.WriteString(Ledit)
		sb.WriteString(" ")
	}

	new := []rune(sb.String())

	// replace tag if the new one is different
	resize := false
	if !runes.Equal(new, []rune(w.tag.file.String())) {
		resize = true // Might need to resize the tag
		// try to preserve user selection
		newbarIndex := runes.IndexRune(new, '|') // New always has '|'
		q0 := w.tag.q0
		q1 := w.tag.q1

		// These alter the Text's selection values.
		w.tag.Delete(0, w.tag.Nc(), true)
		w.tag.Insert(0, new, true)

		// Rationalize the selection as best as possible
		w.tag.q0 = util.Min(q0, w.tag.Nc())
		w.tag.q1 = util.Min(q1, w.tag.Nc())
		if oldbarIndex != -1 && q0 > oldbarIndex {
			bar := newbarIndex - oldbarIndex
			w.tag.q0 = q0 + bar
			w.tag.q1 = q1 + bar
		}
	}
	w.tag.file.Clean()
	n := w.tag.file.Size()
	if w.tag.q0 > n {
		w.tag.q0 = n
	}
	if w.tag.q1 > n {
		w.tag.q1 = n
	}
	// TODO(rjk): This may redraw the selection unnecessarily
	// if we replaced the tag above.
	w.tag.SetSelect(w.tag.q0, w.tag.q1)
	w.DrawButton()
	if resize {
		w.tagsafe = false
		w.Resize(w.r, true, true)
	}
}

// TODO(rjk): In the future of File's replacement with undo buffer,
// this method could be renamed to something like "UpdateTag"
func (w *Window) Commit(t *Text) {
	t.Commit() // will set the file.mod to true
	if t.what == Body {
		return
	}
	filename := w.ParseTag()
	if filename != w.body.file.Name() {
		seq++
		w.body.file.Mark(seq)
		w.body.file.Modded()
		w.SetName(filename)
		w.SetTag()
	}
}

func isDir(r string) (bool, error) {
	f, err := os.Open(r)
	if err != nil {
		return false, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		return true, nil
	}

	return false, nil
}

func (w *Window) AddIncl(r string) {
	// Tries to open absolute paths, and if fails, tries
	// to use dirname instead.
	d, err := isDir(r)
	if !d {
		if filepath.IsAbs(r) {
			warning(nil, "%s: Not a directory: %v", r, err)
			return
		}
		r = w.body.DirName(r)
		d, err := isDir(r)
		if !d {
			warning(nil, "%s: Not a directory: %v", r, err)
			return
		}
	}
	w.incl = append(w.incl, r)
}

// Clean returns true iff w can be treated as unmodified.
// This will modify the File so that the next call to Clean will return true
// even if this one returned false.
func (w *Window) Clean(conservative bool) bool {
	if w.body.file.IsDirOrScratch() { // don't whine if it's a guide file, error window, etc.
		return true
	}
	if !conservative && w.nopen[QWevent] > 0 {
		return true
	}
	if w.body.file.TreatAsDirty() {
		if len(w.body.file.Name()) != 0 {
			warning(nil, "%v modified\n", w.body.file.Name())
		} else {
			if w.body.Nc() < 100 { // don't whine if it's too small
				return true
			}
			warning(nil, "unnamed file modified\n")
		}
		// This toggle permits checking if we can safely destroy the window.
		w.body.file.TreatAsClean()
		return false
	}
	return true
}

// CtlPrint generates the contents of the fsys's acme/<id>/ctl pseduo-file if fonts is true.
// Otherwise,it emits a portion of the per-window dump file contents.
func (w *Window) CtlPrint(fonts bool) string {
	isdir := 0
	if w.body.file.IsDir() {
		isdir = 1
	}
	dirty := 0
	if w.body.file.Dirty() {
		dirty = 1
	}
	buf := fmt.Sprintf("%11d %11d %11d %11d %11d ", w.id, w.tag.Nc(),
		w.body.Nc(), isdir, dirty)
	if fonts {
		// fsys exposes the actual physical font name.
		buf = fmt.Sprintf("%s%11d %s %11d ", buf, w.body.fr.Rect().Dx(),
			quote(fontget(w.body.font, w.display).Name()), w.body.fr.GetMaxtab())
	}
	return buf
}

func (w *Window) Eventf(format string, args ...interface{}) {
	var (
		x *Xfid
	)
	if w.nopen[QWevent] == 0 {
		return
	}
	if w.owner == 0 {
		util.AcmeError("no window owner", nil)
	}
	b := []byte(fmt.Sprintf(format, args...))
	w.events = append(w.events, byte(w.owner))
	w.events = append(w.events, b...)
	x = w.eventx
	if x != nil {
		w.eventx = nil
		x.c <- nil
	}
}

// ClampAddr clamps address range based on the body buffer.
func (w *Window) ClampAddr() {
	if w.addr.q0 < 0 {
		w.addr.q0 = 0
	}
	if w.addr.q1 < 0 {
		w.addr.q1 = 0
	}
	if w.addr.q0 > w.body.Nc() {
		w.addr.q0 = w.body.Nc()
	}
	if w.addr.q1 > w.body.Nc() {
		w.addr.q1 = w.body.Nc()
	}
}
