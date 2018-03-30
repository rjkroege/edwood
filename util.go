package main

import (
	"fmt"
	"image"
	"os"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func minu(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func region(a, b int) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}

func acmeerror(s string, err error) {
	fmt.Fprintf(os.Stderr, "acme: %s: %v\n", s, err)
	// panic(fmt.Sprintf(os.Stderr, "acme: %s: %v\n", s, err))
}

var (
	prevmouse image.Point
	mousew    *Window
)

func clearmouse() {
	mousew = nil
}

func savemouse(w *Window) {
	prevmouse = mouse.Point
	mousew = w
}

func restoremouse(w *Window) bool {
	defer func() { mousew = nil }()
	if mousew != nil && mousew == w {
		w.display.MoveTo(prevmouse)
		return true
	}
	return false
}

// TODO(flux) The "correct" answer here is return unicode.IsNumber(c) || unicode.IsLetter(c)
func isalnum(c rune) bool {
	/*
	 * Hard to get absolutely right.  Use what we know about ASCII
	 * and assume anything above the Latin control characters is
	 * potentially an alphanumeric.
	 */
	if c <= ' ' {
		return false
	}
	if 0x7F <= c && c <= 0xA0 {
		return false
	}
	if utfrune([]rune("!\"#$%&'()*+,-./:;<=>?@[\\]^`{|}~"), c) != -1 {
		return false
	}
	return true
}

func runeeq(s1, s2 []rune) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}
func runestrchr(s []rune, r rune) int {
	for ret, sr := range s {
		if sr == r {
			return ret
		}
	}
	return -1
}

func utfrune(s []rune, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

func errorwin1(dir string, incl []string) *Window {
	var Lpluserrors = "+Errors"

	r := dir + string("/") + Lpluserrors
	w := lookfile(r)
	if w == nil {
		if len(row.col) == 0 {
			if row.Add(nil, -1) == nil {
				acmeerror("can't create column to make error window", nil)
			}
		}
		w = row.col[len(row.col)-1].Add(nil, nil, -1)
		w.filemenu = false
		w.SetName(r)
		xfidlog(w, "new")
	}
	for _, in := range incl {
		w.AddIncl(in)
	}
	w.autoindent = globalautoindent
	return w
}

/* make new window, if necessary; return with it locked */
func errorwin(md *MntDir, owner int) *Window {
	var w *Window

	for {
		if md == nil {
			w = errorwin1("", nil)
		} else {
			w = errorwin1(md.dir, md.incl)
		}
		w.Lock(owner)
		if w.col != nil {
			break
		}
		/* window was deleted too fast */
		w.Unlock()
	}
	return w
}

/*
 * Incoming window should be locked.
 * It will be unlocked and returned window
 * will be locked in its place.
 */
func errorwinforwin(w *Window) *Window {
	var (
		owner int
		incl  []string
		dir   string
		t     *Text
	)

	t = &w.body
	dir = t.DirName("")
	if dir == "." { /* sigh */
		dir = ""
	}
	incl = []string{}
	for _, in := range w.incl {
		incl = append(incl, in)
	}
	owner = w.owner
	w.Unlock()
	for {
		w = errorwin1(dir, incl)
		w.Lock(owner)
		if w.col != nil {
			break
		}
		/* window deleted too fast */
		w.Unlock()
	}
	return w
}

/*
 * Heuristic city.
 */
func makenewwindow(t *Text) *Window {
	var (
		c               *Column
		w, bigw, emptyw *Window
		emptyb          *Text
		i, y, el        int
	)
	switch {
	case activecol != nil:
		c = activecol
	case seltext != nil && seltext.col != nil:
		c = seltext.col
	case t != nil && t.col != nil:
		c = t.col
	default:
		if len(row.col) == 0 && row.Add(nil, -1) == nil {
			acmeerror("can't make column", nil)
		}
		c = row.col[len(row.col)-1]
	}
	activecol = c
	if t == nil || t.w == nil || len(c.w) == 0 {
		return c.Add(nil, nil, -1)
	}

	/* find biggest window and biggest blank spot */
	emptyw = c.w[0]
	bigw = emptyw
	for i = 1; i < len(c.w); i++ {
		w = c.w[i]
		/* use >= to choose one near bottom of screen */
		if w.body.fr.GetFrameFillStatus().Maxlines >= bigw.body.fr.GetFrameFillStatus().Maxlines {
			bigw = w
		}
		if w.body.fr.GetFrameFillStatus().Maxlines-w.body.fr.GetFrameFillStatus().Nlines >= emptyw.body.fr.GetFrameFillStatus().Maxlines-emptyw.body.fr.GetFrameFillStatus().Nlines {
			emptyw = w
		}
	}
	emptyb = &emptyw.body
	el = emptyb.fr.GetFrameFillStatus().Maxlines - emptyb.fr.GetFrameFillStatus().Nlines
	/* if empty space is big, use it */
	if el > 15 || (el > 3 && el > (bigw.body.fr.GetFrameFillStatus().Maxlines-1)/2) {
		y = emptyb.fr.Rect.Min.Y + emptyb.fr.GetFrameFillStatus().Nlines*tagfont.Height
	} else {
		/* if this window is in column and isn't much smaller, split it */
		if t.col == c && t.w.r.Dy() > 2*bigw.r.Dy()/3 {
			bigw = t.w
		}
		y = (bigw.r.Min.Y + bigw.r.Max.Y) / 2
	}
	w = c.Add(nil, nil, y)
	if w.body.fr.GetFrameFillStatus().Maxlines < 2 {
		w.col.Grow(w, 1)
	}
	return w
}

func mousescrollsize(nl int) int {
	// Unimpl()
	return 1
}

type Warning struct {
	md  *MntDir
	buf Buffer
}

var warnings = []*Warning{}

func flushwarnings() {
	var (
		w                *Window
		t                *Text
		owner, nr, q0, n int
	)
	for _, warn := range warnings {
		w = errorwin(warn.md, 'E')
		t = &w.body
		owner = w.owner
		if owner == 0 {
			w.owner = 'E'
		}
		w.Commit(t)
		/*
		 * Most commands don't generate much output. For instance,
		 * Edit ,>cat goes through /dev/cons and is already in blocks
		 * because of the i/o system, but a few can.  Edit ,p will
		 * put the entire result into a single hunk.  So it's worth doing
		 * this in blocks (and putting the text in a buffer in the first
		 * place), to avoid a big memory footprint.
		 */
		q0 = t.Nc()
		r := make([]rune, RBUFSIZE)
		for n = 0; n < warn.buf.Nc(); n += nr {
			nr = warn.buf.Nc() - n
			if nr > RBUFSIZE {
				nr = RBUFSIZE
			}
			warn.buf.Read(n, r[:nr])
			_, nr = t.BsInsert(t.Nc(), r[:nr], true)
		}
		t.Show(q0, t.Nc(), true)
		t.w.SetTag()
		t.ScrDraw()
		w.owner = owner
		w.dirty = false
		w.Unlock()
		warn.buf.Close()
		if warn.md != nil {
			fsysdelid(warn.md)
		}
	}
	warnings = warnings[0:0]
}

func warning(md *MntDir, s string, args ...interface{}) {
	r := []rune(fmt.Sprintf(s, args...))
	addwarningtext(md, r)
}

func addwarningtext(md *MntDir, r []rune) {
	for _, warn := range warnings {
		if warn.md == md {
			warn.buf.Insert(warn.buf.Nc(), r)
			return
		}
	}
	warn := Warning{}
	warn.md = md
	if md != nil {
		fsysincid(md)
	}
	warn.buf.Insert(0, r)
	warnings = append(warnings, &warn)
	select {
	case cwarn <- 0:
	default:
	}
}
