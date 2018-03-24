package main

import (
	"fmt"
	"image"
	"os"
	"strings"
	"sync"

	"9fans.net/go/draw"
	"github.com/rjkroege/edwood/frame"
)

type Window struct {
	display *draw.Display
	lk      sync.Mutex
	ref     Ref
	tag     Text
	body    Text
	r       image.Rectangle

	isdir      bool
	isscratch  bool
	filemenu   bool
	dirty      bool
	autoindent bool
	showdel    bool

	id    int
	addr  Range
	limit Range

	nopen      [QMAX]byte
	nomark     bool
	wrselrange Range
	rdselfd    *os.File

	col    *Column
	eventx *Xfid
	events []byte

	nevents     int
	owner       int
	maxlines    int
	dirnames    []string
	widths      []int
	putseq      int
	incl        []string
	reffont     *draw.Font
	ctrllock    *sync.Mutex
	ctlfid      uint32
	dumpstr     string
	dumpdir     string
	dumpid      int
	utflastqid  int
	utflastboff uint64
	utflastq    int
	tagsafe     bool
	tagexpand   bool
	taglines    int
	tagtop      image.Rectangle
	editoutlk   *sync.Mutex
}

func NewWindow() *Window {
	return &Window{}
}

func (w *Window) Init(clone *Window, r image.Rectangle, dis *draw.Display) {

	//	var r1, br image.Rectangle
	//	var f *File
	var rf *draw.Font
	//	var rp []rune
	//	var nc int

	w.tag.w = w
	w.taglines = 1
	w.tagsafe = true
	w.tagexpand = true
	w.body.w = w
	w.incl = []string{}
	WinId++
	w.id = WinId
	w.display = dis

	w.ctlfid = MaxFid
	w.utflastqid = -1
	r1 := r

	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + tagfont.Height
	r1.Max.Y = r1.Min.Y + w.taglines*tagfont.Height

	f := NewTagFile()
	f.AddText(&w.tag)
	w.tag.Init(f, r1, tagfont, tagcolors, w.display)
	w.tag.what = Tag

	/* tag is a copy of the contents, not a tracked image */
	if clone != nil {
		w.tag.Delete(0, w.tag.file.b.nc(), true)
		w.tag.Insert(0, clone.tag.file.b, true)
		w.tag.file.Reset()
		w.tag.SetSelect(len(w.tag.file.b), len(w.tag.file.b))
	}
	r1 = r
	r1.Min.Y += w.taglines*tagfont.Height + 1
	if r1.Max.Y < r1.Min.Y {
		r1.Max.Y = r1.Min.Y
	}

	// Body setup.
	f = NewFile("")
	if clone != nil {
		f = clone.body.file
		w.body.org = clone.body.org
		w.isscratch = clone.isscratch
		rf = fontget(0, false, false, clone.body.font.Name, dis)
	} else {
		rf = fontget(0, false, false, "", dis)
	}
	f = f.AddText(&w.body)
	w.body.Init(f, r1, rf, textcolors, w.display)
	w.body.what = Body
	r1.Min.Y -= 1
	r1.Max.Y = r1.Min.Y + 1
	if w.display != nil {
		w.display.ScreenImage.Draw(r1, tagcolors[frame.ColBord], nil, image.ZP)
	}
	w.body.ScrDraw()
	w.r = r
	var br image.Rectangle
	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + button.R.Dx()
	br.Max.Y = br.Min.Y + button.R.Dy()
	if w.display != nil {
		w.display.ScreenImage.Draw(br, button, nil, button.R.Min)
	}
	w.filemenu = true
	w.maxlines = w.body.fr.GetFrameFillStatus().Maxlines
	w.autoindent = globalautoindent
	if clone != nil {
		w.dirty = clone.dirty
		w.autoindent = clone.autoindent
		w.body.SetSelect(clone.body.q0, clone.body.q1)
		w.SetTag()
	}
}

func (w *Window) DrawButton() {

	b := button
	if !w.isdir && !w.isscratch && w.body.file.mod { // TODO(flux) validate text cache stuff goes away
		b = modbutton
	}
	var br image.Rectangle
	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + b.R.Dx()
	br.Max.Y = br.Min.Y + b.R.Dy()
	if w.display != nil {
		w.display.ScreenImage.Draw(br, b, nil, b.R.Min)
	}
}

func (w *Window) delRunePos() int {
	var n int
	for n = 0; n < w.tag.file.b.nc(); n++ {
		r := w.tag.file.b.Read(n, 1)
		if r[0] == ' ' {
			break
		}
	}
	n += 2
	if n >= w.tag.file.b.nc() {
		return -1
	}
	return n
}

func (w *Window) moveToDel() {
	n := w.delRunePos()
	if n < 0 {
		return
	}
	if w.display != nil {
		w.display.MoveTo(w.tag.fr.Ptofchar(n).Add(image.Pt(4, w.tag.fr.Font.DefaultHeight()-4)))
	}
}

func (w *Window) TagLines(r image.Rectangle) int {
	// Unimpl()
	return 1
}

func (w *Window) Resize(r image.Rectangle, safe, keepextra bool) int {
	mouseintag := mouse.Point.In(w.tag.all)
	mouseinbody := mouse.Point.In(w.body.all)

	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + tagfont.Height

	r1 := r
	r1.Max.Y = min(r.Max.Y, r1.Min.Y+w.taglines*tagfont.Height)
	if !safe || !w.tagsafe || w.tag.all.Eq(r1) {
		w.taglines = w.TagLines(r)
		r1.Max.Y = min(r.Max.Y, r1.Min.Y+w.taglines*tagfont.Height)
	}

	y := r1.Max.Y

	// Resize/redraw tag TODO(flux)
	if !safe || !w.tagsafe || !w.tag.all.Eq(r1) {
		w.tag.Resize(r1, true)
		y = w.tag.fr.Rect.Max.Y
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
		if y+1+w.body.fr.Font.DefaultHeight() <= r.Max.Y { /* room for one line */
			r1.Min.Y = y
			r1.Max.Y = y + 1
			if w.display != nil {
				w.display.ScreenImage.Draw(r1, tagcolors[frame.ColBord], nil, image.ZP)
			}
			y++
			r1.Min.Y = min(y, r.Max.Y)
			r1.Max.Y = r.Max.Y
		} else {
			r1.Min.Y = y
			r1.Max.Y = y
		}
		y = w.body.Resize(r1, keepextra)
		w.r = r
		w.r.Max.Y = y
		w.body.ScrDraw()
		w.body.all.Min.Y = oy
	}
	w.maxlines = min(w.body.fr.GetFrameFillStatus().Nlines, max(w.maxlines, w.body.fr.GetFrameFillStatus().Maxlines))
	return w.r.Max.Y
}

func (w *Window) Lock1(owner int) {
	w.ref.Inc()
	w.lk.Lock()
	w.owner = owner
}

func (w *Window) Lock(owner int) {
	w.owner = owner
	w.lk.Lock()
}

func (w *Window) Unlock() {
	w.owner = 0
	w.lk.Unlock()
}

func (w *Window) MouseBut() {
	if w.display != nil {
		w.display.MoveTo(w.tag.scrollr.Min.Add(
			image.Pt(w.tag.scrollr.Dx(), tagfont.Height).Div(2)))
	}
}

func (w *Window) DirFree() {
	w.dirnames = w.dirnames[0:0]
	w.widths = w.widths[0:0]
}

func (w *Window) Close() {
	if w.ref.Dec() == 0 {
		xfidlog(w, "del")
		w.DirFree()
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
		x.c <- nil /* wake him up */
	}
}

func (w *Window) Undo(isundo bool) {
	w.utflastqid = -1
	body := &w.body
	body.q0, body.q1 = body.file.Undo(isundo)
	body.Show(body.q0, body.q1, true)
	f := body.file
	for _, text := range f.text {
		v := text.w
		v.dirty = (f.seq != v.putseq)
		if v != w {
			v.body.q0 = (getP0(v.body.fr)) + v.body.org
			v.body.q1 = (getP1(v.body.fr)) + v.body.org
		}
	}
	w.SetTag()
}

func (w *Window) SetName(name string) {
	Lslashguide := "/guide"
	LplusErrors := "+Errors"

	t := &w.body
	if t.file.name == name {
		return
	}
	w.isscratch = false
	if strings.HasSuffix(name, Lslashguide) || strings.HasSuffix(name, LplusErrors) {
		w.isscratch = true
	}
	t.file.SetName(name)

	for _, te := range t.file.text {
		te.w.SetTag()
		te.w.isscratch = w.isscratch
	}
}

func (w *Window) Type(t *Text, r rune) {
	t.Type(r)
	if t.what == Body {
		for _, text := range t.file.text {
			text.ScrDraw()
		}
	}
	w.SetTag()
}

func (w *Window) ClearTag() {
	/* w must be committed */
	n := w.tag.file.b.nc()
	r := w.tag.file.b.Read(0, n)
	var i int
	for i = 0; i < n; i++ {
		if r[i] == ' ' || r[i] == '\t' {
			break
		}
	}
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
	w.tag.file.mod = false
	if w.tag.q0 > i {
		w.tag.q0 = i
	}
	if w.tag.q1 > i {
		w.tag.q1 = i
	}
	w.tag.SetSelect(w.tag.q0, w.tag.q1)
}

func (w *Window) SetTag() {
	Ldelsnarf := (" Del Snarf")
	Lundo := (" Undo")
	Lredo := (" Redo")
	Lget := (" Get")
	Lput := (" Put")
	Llook := (" Look")
	Ledit := (" Edit")
	//	Lpipe := (" |")

	/* there are races that get us here with stuff in the tag cache, so we take extra care to sync it */
	if w.tag.ncache != 0 || w.tag.file.mod {
		w.Commit(&w.tag) /* check file name; also guarantees we can modify tag contents */
	}

	// (flux) The C implemtation does a lot of work to avoid
	// re-setting the tag text if unchanged.  That's probably not
	// relevant in the modern world.  We can build a new tag trivially
	// and put up with the traffic implied for a tag line.

	sb := strings.Builder{}
	sb.WriteString(w.body.file.name)
	sb.WriteString(Ldelsnarf)
	if w.filemenu {
		if w.body.needundo || len(w.body.file.delta) > 0 || w.body.ncache != 0 {
			sb.WriteString(Lundo)
		}
		if len(w.body.file.epsilon) > 0 {
			sb.WriteString(Lredo)
		}
		dirty := w.body.file.name != "" && (w.body.ncache != 0 || w.body.file.seq != w.putseq)
		if !w.isdir && dirty {
			sb.WriteString(Lput)
		}
	}
	if w.isdir {
		sb.WriteString(Lget)
	}
	olds := string(w.tag.file.b)
	oldbarIndex := w.tag.file.b.Index([]rune("|"))
	if oldbarIndex >= 0 {
		sb.WriteString(" ")
		sb.WriteString(olds[oldbarIndex:])
	} else {
		sb.WriteString(" |")
		sb.WriteString(Llook)
		sb.WriteString(Ledit)
	}

	new := Buffer([]rune(sb.String()))

	/* replace tag if the new one is different */
	resize := false
	if !new.Eq(w.tag.file.b) {
		resize = true // Might need to resize the tag
		w.tag.Delete(0, w.tag.file.b.nc(), true)
		w.tag.Insert(0, new, true)
		/* try to preserve user selection */
		newbarIndex := new.Index([]rune("|")) // New always has "|"
		q0 := w.tag.q0
		q1 := w.tag.q1
		if oldbarIndex != -1 {
			if q0 > (oldbarIndex) {
				bar := (newbarIndex - oldbarIndex)
				w.tag.q0 = q0 + bar
				w.tag.q1 = q1 + bar
			}
		}
	}
	w.tag.file.mod = false
	n := w.tag.file.b.nc() + (w.tag.ncache)
	if w.tag.q0 > n {
		w.tag.q0 = n
	}
	if w.tag.q1 > n {
		w.tag.q1 = n
	}
	w.tag.SetSelect(w.tag.q0, w.tag.q1)
	w.DrawButton()
	if resize {
		w.tagsafe = false
		w.Resize(w.r, true, true)
	}
}

func (w *Window) Commit(t *Text) {
	t.Commit(true)
	f := t.file
	if len(f.text) > 1 {
		for _, te := range f.text {
			te.Commit(false) /* no-op for t */
		}
	}
	if t.what == Body {
		return
	}
	r := w.tag.file.b.Read(0, w.tag.file.b.nc())
	filename := string(runesplitN(r, []rune(" \t"), 1)[0])
	if filename != w.body.file.name {
		seq++
		w.body.file.Mark()
		w.body.file.mod = true
		w.dirty = true
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
	if d == false {
		if r[0] == '/' {
			warning(nil, "%s: Not a directory: %v", r, err)
			return
		}
		r = string(dirname(&w.body, []rune(r)))
		d, err := isDir(r)
		if d == false {
			warning(nil, "%s: Not a directory: %v", r, err)
			return
		}
	}
	w.incl = append(w.incl, r)
	return

}

func (w *Window) Clean(conservative bool) bool {
	if w.isscratch || w.isdir { /* don't whine if it's a guide file, error window, etc. */
		return true
	}
	if !conservative && w.nopen[QWevent] > 0 {
		return true
	}
	if w.dirty {
		if len(w.body.file.name) != 0 {
			warning(nil, "%v modified\n", w.body.file.name)
		} else {
			if w.body.file.b.nc() < 100 { /* don't whine if it's too small */
				return true
			}
			warning(nil, "unnamed file modified\n")
		}
		w.dirty = false
		return false
	}
	return true
}

func (w *Window) CtlPrint(fonts bool) string {
	isdir := 0
	if w.isdir {
		isdir = 1
	}
	dirty := 0
	if w.dirty {
		dirty = 1
	}
	buf := fmt.Sprintf("%11d %11d %11d %11d %11d ", w.id, w.tag.file.b.nc(),
		w.body.file.b.nc(), isdir, dirty)
	if fonts {
		return fmt.Sprintf("%s%11d %q %11d ", buf, w.body.fr.Rect.Dx(),
			w.body.font.Name, w.body.fr.GetMaxtab())
	}
	return buf
}

func (w *Window) Event(format string, args ...interface{}) {
	var (
		x *Xfid
	)
	if w.nopen[QWevent] == 0 {
		return
	}
	if w.owner == 0 {
		acmeerror("no window owner", nil)
	}
	b := []byte(fmt.Sprintf(format, args...))

	w.events = append(w.events, byte(w.owner))
	w.events = append(w.events, b...)
	w.nevents = len(w.events)
	x = w.eventx
	if x != nil {
		w.eventx = nil
		x.c <- nil
	}
}
