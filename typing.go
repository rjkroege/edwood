package main

import (
	"image"
)

// All the sub-functions needed to implement typing are in this file.

// tagdown expands the tag to show all of the text.
func (t *Text) tagdownalways(nta func()) {
	if t.what == Tag {

		if !t.w.tagexpand {
			t.w.tagexpand = true
			t.w.Resize(t.w.r, false, true)
		}

	} else {
		nta()
	}

}

// tagup shrinks the tag to a single line
func (t *Text) tagupalways(nta func()) {
	if t.what == Tag {
		if t.w.tagexpand {
			t.w.tagexpand = false
			t.w.taglines = 1
			t.w.Resize(t.w.r, false, true)

		}
	} else {
		nta()
	}

}

// keydownhelper is common code used for key-down motion that moves
// n lines.
func (t *Text) keydownhelper(n int) {
	q0 := t.org + t.fr.Charofpt(image.Pt(t.fr.Rect().Min.X, t.fr.Rect().Min.Y+n*t.fr.DefaultFontHeight()))
	t.SetOrigin(q0, true)
}

// keyuphelper is common code used for key-up motion that moves
// n lines.
func (t *Text) keyuphelper(n int) {
	q0 := t.Backnl(t.org, n)
	t.SetOrigin(q0, true)
}
