package main

import (
	"fmt"
	"image"
	"os"
	"unicode"
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

func warning(md *MntDir, s string, args ...interface{}) {
	// TODO(flux): Port to actually output to the error window
	_ = md
	fmt.Printf(s, args...)
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
		display.MoveTo(prevmouse)
		return true
	}
	return false
}

func isalnum(c rune) bool {
	return unicode.IsNumber(c) || unicode.IsLetter(c)
}

func runestrchr(s []rune, r rune) int {
	for ret, sr := range s {
		if sr == r {
			return ret
		}
	}
	return -1
}

func utfrune(s []rune, r int) int {
	for i, c := range s {
		if c == rune(r) {
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
	dir = t.DirName()
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
