package main

import (
	"image"
	"math"
	"strings"
	"sync"

	"9fans.net/go/draw"
	"github.com/paul-lalonde/acme/frame"
)

type Window struct {
	lk   sync.Mutex
	ref  Ref
	tag  Text
	body Text
	r    image.Rectangle

	isdir      bool
	isscratch  bool
	filemenu   bool
	dirty      bool
	autoindent bool
	showdel    bool

	id    int
	addr  Range
	limit Range

	nopen     [QMAX]bool
	nomark    bool
	wselrange Range
	rdselfd   int

	col    *Column
	eventx Xfid
	events string

	nevents     int
	owner       int
	maxlines    int
	dirnames    []string
	widths      []int
	putseq      int
	nincl       int
	incl        []string
	reffont     *draw.Font
	ctrllock    *sync.Mutex
	ctlfid      uint
	dumpstr     string
	dumpdir     string
	dumpid      int
	utflastqid  int
	utflastboff int
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

func (w *Window) Init(clone *Window, r image.Rectangle) {

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

	WinId++
	w.id = WinId

	w.ctlfid = math.MaxUint64
	w.utflastqid = -1
	r1 := r

	w.tagtop = r
	w.tagtop.Max.Y = r.Min.Y + tagfont.Height
	r1.Max.Y = r1.Min.Y + w.taglines*tagfont.Height

	f := &File{}
	f.AddText(&w.tag)
	w.tag.Init(f, r1, tagfont, tagcolors)
	w.tag.what = Tag

	/* tag is a copy of the contents, not a tracked image */
	/* TODO(flux): Unimplemented Clone
	if(clone){
		w.tag.Delete(&, 0, w.tag.file.b.nc, true);
		w.tag.Insert(0, clone.tag.file.b, true);
		w.tag.file.Reset();
		w.tag.Setselect(len(w.tag.file.b), len(w.tag.file.b))
	}
	*/
	r1 = r
	r1.Min.Y += w.taglines*tagfont.Height + 1
	if r1.Max.Y < r1.Min.Y {
		r1.Max.Y = r1.Min.Y
	}

	// Body setup.
	f = nil
	/* TODO(flux): Unimplemented Clone
	if clone {
		f = clone.body.file;
		w.body.org = clone.body.org;
		w.isscratch = clone.isscratch;
		rf = rfget(false, false, false, clone.body.reffont.f.name);
	} else {
	*/
	f = &File{}
	rf = fontget(0, false, false, "")
	//	}
	f = f.AddText(&w.body)
	w.body.Init(f, r1, rf, textcolors)
	w.body.what = Body
	r1.Min.Y -= 1
	r1.Max.Y = r1.Min.Y + 1
	display.ScreenImage.Draw(r1, tagcolors[frame.ColBord], nil, image.ZP)
	// TODO(flux) w.body.Scrdraw()
	w.r = r
	var br image.Rectangle
	br.Min = w.tag.scrollr.Min
	br.Max.X = br.Min.X + button.R.Dx()
	br.Max.Y = br.Min.Y + button.R.Dy()
	display.ScreenImage.Draw(br, button, nil, button.R.Min)
	w.filemenu = true
	w.maxlines = w.body.fr.MaxLines
	w.autoindent = globalautoindent
	/* TODO(flux): Unimplemented Clone
	if clone != nil{
		w.dirty = clone.dirty
		w.autoindent = clone.autoindent
		w.body.Setselect(clone.body.q0, clone.body.q1)
		w.Settag(w)
	}
	*/
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
	display.ScreenImage.Draw(br, b, nil, b.R.Min)

}

func (w *Window) RunePos() int {
	return 0
}

func (w *Window) ToDel() {

}

func (w *Window) TagLines(r image.Rectangle) int {
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
			display.MoveTo(p)
		}
		// If mouse is in body, push down as tag expands.
		if mouseinbody && mouse.Point.In(w.tag.all) {
			p := mouse.Point
			p.Y = w.tag.all.Max.Y + 3
			display.MoveTo(p)
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
			display.ScreenImage.Draw(r1, tagcolors[frame.ColBord], nil, image.ZP)
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
		// w.body.Scrdraw()  // TODO(flux) scrollbars
		w.body.all.Min.Y = oy
	}
	w.maxlines = min(w.body.fr.NLines, max(w.maxlines, w.body.fr.MaxLines))
	return w.r.Max.Y
}

func (w *Window) Lock1(owner int) {

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
	display.MoveTo(w.tag.scrollr.Min.Add(
		image.Pt(w.tag.scrollr.Dx(), tagfont.Height).Div(2)))

}

func (w *Window) DirFree() {

}

func (w *Window) Close() {

}

func (w *Window) Delete() {

}

func (w *Window) Undo(isundo bool) {

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

}

func (w *Window) ClearTag() {

}

func (w *Window) SetTag1() {

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
		if w.body.needundo || w.body.file.delta.nc() > 0 || w.body.ncache != 0 {
			sb.WriteString(Lundo)
		}
		if w.body.file.epsilon.nc() > 0 {
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
			if q0 > uint(oldbarIndex) {
				bar := uint(newbarIndex - oldbarIndex)
				w.tag.q0 = q0 + bar
				w.tag.q1 = q1 + bar
			}
		}
	}
	w.tag.file.mod = false
	n := w.tag.file.b.nc() + uint(w.tag.ncache)
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

func (w *Window) AddIncl(r string, n int) {

}

func (w *Window) Clean(conservative bool) int {
	return 0
}

func (w *Window) CtlPrint(buf string, fonts int) string {
	return ""
}

func (w *Window) Event(fmt string, args ...interface{}) {

}
