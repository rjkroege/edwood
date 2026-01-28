package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rjkroege/edwood/runes"
	"github.com/rjkroege/edwood/util"
)

// clearmouse removes any saved mouse state.
func clearmouse() {
	global.mousestate.Clear()
}

// savemouse stores the current mouse position and associated window.
func savemouse(w *Window) {
	global.mousestate.Save(w, global.mouse.Point)
}

// restoremouse moves the mouse cursor to the saved position if the given window
// matches the saved window. Returns true if the cursor was moved.
func restoremouse(w *Window) bool {
	return global.mousestate.Restore(w)
}

func bytetorune(s []byte) []rune {
	r, _, _ := util.Cvttorunes(s, len(s))
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

// errorwin1Name adds an +Errors suffix to dir.
func errorwin1Name(dir string) string {
	return filepath.Join(dir, "+Errors")
}

// errorwin1 is an internal helper function.
// Caller must hold global.row.lk.
func errorwin1(dir string, incl []string) *Window {
	r := errorwin1Name(dir)
	w := lookfile(r)
	if w == nil {
		if len(global.row.col) == 0 {
			if global.row.Add(nil, -1) == nil {
				util.AcmeError("can't create column to make error window", nil)
			}
		}
		w = global.row.col[len(global.row.col)-1].Add(nil, nil, -1)
		w.filemenu = false
		w.SetName(r)
		xfidlog(w, "new")
	}
	for _, in := range incl {
		w.AddIncl(in)
	}
	w.autoindent = *globalAutoIndent
	return w
}

// errorwin creates or finds an error window and returns it locked.
// If srcWin is non-nil, extracts directory and includes from it
// (srcWin must be locked; it will be unlocked before returning).
// If srcWin is nil, uses md for directory and includes.
func errorwin(md *MntDir, owner int, srcWin *Window) *Window {
	var (
		dir  string
		incl []string
		w    *Window
	)

	if srcWin != nil {
		// Extract info from source window and unlock it
		dir = srcWin.body.DirName("")
		incl = append(incl, srcWin.incl...)
		owner = srcWin.owner
		srcWin.Unlock()
	} else if md != nil {
		dir = md.dir
		incl = md.incl
	}

	for {
		// Hold row lock during errorwin1 and window locking to ensure
		// consistent access to global.row.col. Lock ordering: row -> window.
		global.row.lk.Lock()
		w = errorwin1(dir, incl)
		w.Lock(owner)
		global.row.lk.Unlock()

		if w.col != nil {
			break
		}
		// window was deleted too fast
		w.Unlock()
	}
	return w
}

// errorwinforwin creates an error window for the given window's directory.
// The incoming window must be locked; it will be unlocked and the returned
// error window will be locked in its place.
// Deprecated: Use errorwin(nil, 0, w) instead.
func errorwinforwin(w *Window) *Window {
	return errorwin(nil, 0, w)
}

// Heuristic city.
// makenewwindow creates a new window, choosing an appropriate location
// based on the current state of the editor.
func makenewwindow(t *Text) *Window {
	// Hold row lock throughout to protect access to global.row.col,
	// global.activecol, global.seltext, and column window lists.
	global.row.lk.Lock()
	defer global.row.lk.Unlock()

	var (
		c               *Column
		w, bigw, emptyw *Window
		emptyb          *Text
		i, y, el        int
	)
	switch {
	case global.activecol != nil:
		c = global.activecol
	case global.seltext != nil && global.seltext.col != nil:
		c = global.seltext.col
	case t != nil && t.col != nil:
		c = t.col
	default:
		if len(global.row.col) == 0 && global.row.Add(nil, -1) == nil {
			util.AcmeError("can't make column", nil)
		}
		c = global.row.col[len(global.row.col)-1]
	}
	global.activecol = c
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
		y = emptyb.fr.Rect().Min.Y + emptyb.fr.GetFrameFillStatus().Nlines*fontget(global.tagfont, t.display).Height()
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
	buf bytes.Buffer
}

// TODO(rjk): Move into the global object.
var warnings = []*Warning{}
var warningsMu sync.Mutex

func flushwarnings() {
	// Lock warningsMu to safely swap out the warnings list.
	// We make a local copy and clear the global list while holding the lock,
	// then process the copy without holding warningsMu to avoid lock ordering
	// issues (we need to acquire row and window locks during processing).
	warningsMu.Lock()
	localWarnings := warnings
	warnings = warnings[:0]
	warningsMu.Unlock()

	var (
		w     *Window
		t     *Text
		owner int
	)
	for _, warn := range localWarnings {
		w = errorwin(warn.md, 'E', nil)
		t = &w.body
		owner = w.owner
		if owner == 0 {
			w.owner = 'E'
		}

		// TODO(rjk): Ick.
		r := []rune(warn.buf.String())
		q0 := t.Nc()
		t.BsInsert(q0, r, true)
		t.Show(q0, t.Nc(), true)

		// TODO(rjk): Code inspection of Show suggests that this might
		// be redundant.
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		w.owner = owner
		t.file.TreatAsClean()
		w.Unlock()
		// warn.buf.Close()
		if warn.md != nil {
			mnt.DecRef(warn.md) // IncRef in addwarningtext
		}
	}
}

func warning(md *MntDir, s string, args ...interface{}) {
	warningsMu.Lock()
	defer warningsMu.Unlock()

	r := fmt.Sprintf(s, args...)
	addwarningtext(md, r)
}

func warnError(md *MntDir, s string, args ...interface{}) error {
	warningsMu.Lock()
	defer warningsMu.Unlock()

	err := fmt.Errorf(s, args...)
	addwarningtext(md, err.Error()+"\n")
	return err
}

// TODO(rjk): Convert to using bytes.
func addwarningtext(md *MntDir, b string) {
	for _, warn := range warnings {
		if warn.md == md {
			warn.buf.WriteString(b)
			return
		}
	}

	// No in-progress Warning.
	warn := Warning{
		md: md,
	}
	if md != nil {
		mnt.IncRef(md) // DecRef in flushwarnings
	}
	warn.buf.WriteString(b)
	warnings = append(warnings, &warn)
	select {
	case global.cwarn <- 0:
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

func skipbl(r []rune) []rune {
	return runes.TrimLeft(r, " \t\n")
}
