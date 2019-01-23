package main

import (
	"fmt"
	"image"
	"log"
	"strings"
	"unicode/utf8"
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

func acmeerror(s string, err error) {
	log.Panicf("acme: %s: %v\n", s, err)
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

func bytetorune(s []byte) []rune {
	r, _, _ := cvttorunes(s, len(s))
	return r
}

// TODO(flux) The "correct" answer here is return unicode.IsNumber(c) || unicode.IsLetter(c)
func isalnum(c rune) bool {
	// Hard to get absolutely right.  Use what we know about ASCII
	// and assume anything above the Latin control characters is
	// potentially an alphanumeric.
	if c <= ' ' {
		return false
	}
	if 0x7F <= c && c <= 0xA0 {
		return false
	}
	if strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^`{|}~", c) {
		return false
	}
	return true
}

// Cvttorunes decodes runes r from p. It's guaranteed that first n
// bytes of p will be interpreted without worrying about partial runes.
// This may mean reading up to UTFMax-1 more bytes than n; the caller
// must ensure p is large enough. Partial runes and invalid encodings
// are converted to RuneError. Nb (always >= n) is the number of bytes
// interpreted.
//
// If any U+0000 rune is present in r, they are elided and nulls is set
// to true.
func cvttorunes(p []byte, n int) (r []rune, nb int, nulls bool) {
	for nb < n {
		var w int
		var ru rune
		if p[nb] < utf8.RuneSelf {
			w = 1
			ru = rune(p[nb])
		} else {
			ru, w = utf8.DecodeRune(p[nb:])
		}
		if ru != 0 {
			r = append(r, ru)
		} else {
			nulls = true
		}
		nb += w
	}
	return
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

// make new window, if necessary; return with it locked
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
		// window was deleted too fast
		w.Unlock()
	}
	return w
}

// Incoming window should be locked.
// It will be unlocked and returned window
// will be locked in its place.
func errorwinforwin(w *Window) *Window {
	var (
		owner int
		incl  []string
		dir   string
		t     *Text
	)

	t = &w.body
	dir = t.DirName("")
	if dir == "." { // sigh
		dir = ""
	}
	incl = append(incl, w.incl...)
	owner = w.owner
	w.Unlock()
	for {
		w = errorwin1(dir, incl)
		w.Lock(owner)
		if w.col != nil {
			break
		}
		// window deleted too fast
		w.Unlock()
	}
	return w
}

// Heuristic city.
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

	// find biggest window and biggest blank spot
	emptyw = c.w[0]
	bigw = emptyw
	for i = 1; i < len(c.w); i++ {
		w = c.w[i]
		// use >= to choose one near bottom of screen
		if w.body.fr.GetFrameFillStatus().Maxlines >= bigw.body.fr.GetFrameFillStatus().Maxlines {
			bigw = w
		}
		if w.body.fr.GetFrameFillStatus().Maxlines-w.body.fr.GetFrameFillStatus().Nlines >= emptyw.body.fr.GetFrameFillStatus().Maxlines-emptyw.body.fr.GetFrameFillStatus().Nlines {
			emptyw = w
		}
	}
	emptyb = &emptyw.body
	el = emptyb.fr.GetFrameFillStatus().Maxlines - emptyb.fr.GetFrameFillStatus().Nlines
	// if empty space is big, use it
	if el > 15 || (el > 3 && el > (bigw.body.fr.GetFrameFillStatus().Maxlines-1)/2) {
		y = emptyb.fr.Rect().Min.Y + emptyb.fr.GetFrameFillStatus().Nlines*fontget(tagfont, t.display).Height
	} else {
		// if this window is in column and isn't much smaller, split it
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
		w.Commit(t) // marks the backing text as dirty

		// Most commands don't generate much output. For instance,
		// Edit ,>cat goes through /dev/cons and is already in blocks
		// because of the i/o system, but a few can.  Edit ,p will
		// put the entire result into a single hunk.  So it's worth doing
		// this in blocks (and putting the text in a buffer in the first
		// place), to avoid a big memory footprint.
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
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		w.owner = owner
		t.file.TreatAsClean()
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

const quoteChar = '\''

func needsQuote(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == quoteChar || c <= ' ' { // quote, blanks, or control characters
			return true
		}
	}
	return false
}

// Quote adds single quotes to s in the style of rc(1) if they are needed.
// The behaviour should be identical to Plan 9's quote(3).
func quote(s string) string {
	if s == "" {
		return "''"
	}
	if !needsQuote(s) {
		return s
	}
	var b strings.Builder
	b.Grow(10 + len(s)) // Enough room for few quotes
	b.WriteByte(quoteChar)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == quoteChar {
			b.WriteByte(quoteChar)
		}
		b.WriteByte(c)
	}
	b.WriteByte(quoteChar)
	return b.String()
}
