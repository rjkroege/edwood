package main

import (
	"image"
)

// All the sub-functions needed to implement typing are in this file.

// tagdown wraps the nta key handling function to always open the
// tag.
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

// tagup wraps the nta key handling function to always collapse
// the tag.
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

// KeyLeft handles left-arrow.
func (t *Text) KeyLeft() {
	t.TypeCommit()
	if t.q0 > 0 {
		if t.q0 != t.q1 {
			t.Show(t.q0, t.q0, true)
		} else {
			t.Show(t.q0-1, t.q0-1, true)
		}
	}
}

// KeyRight handles right-arrow.
func (t *Text) KeyRight() {
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
}

// KeyDown handles down with tag expansion.
func (t *Text) KeyDownTagExpanding() {
	t.tagdownalways(func() {
		t.keydownhelper(t.fr.GetFrameFillStatus().Maxlines / 3)
	})
}

// KeyScrollOneDown handles scroll down with tag expansion.
func (t *Text) KeyScrollOneDown() {
	t.tagdownalways(func() {
		t.keydownhelper(max(1, mousescrollsize(t.fr.GetFrameFillStatus().Maxlines)))
	})
}

// KeyPageDown handles page down
func (t *Text) KeyPageDown() {
	t.keydownhelper(2 * t.fr.GetFrameFillStatus().Maxlines / 3)
}

// KeyUp handles arrow up.
func (t *Text) KeyUpTagExpanding() {
	t.tagupalways(func() {
		t.keyuphelper(t.fr.GetFrameFillStatus().Maxlines / 3)
	})
}

// KeyScrollOneUp handles scroll up with tag collapsing.
func (t *Text) KeyScrollOneUp() {
	t.tagupalways(func() {
		t.keyuphelper(mousescrollsize(t.fr.GetFrameFillStatus().Maxlines))
	})
}

// KeyPageUp handles page up.
func (t *Text) KeyPageUp() {
	t.keyuphelper(2 * t.fr.GetFrameFillStatus().Maxlines / 3)
}

// KeyHome handles pressing the home key.
func (t *Text) KeyHome() {
	t.TypeCommit()
	if t.org > t.iq1 {
		q0 := t.Backnl(t.iq1, 1)
		t.SetOrigin(q0, true)
	} else {
		t.Show(0, 0, false)
	}
}

// KeyEnd handles pressing the end key.
func (t *Text) KeyEnd() {
	t.TypeCommit()
	if t.iq1 > t.org+t.fr.GetFrameFillStatus().Nchars {
		if t.iq1 > t.file.b.Nc() {
			// should not happen, but does. and it will crash textbacknl.
			t.iq1 = t.file.b.Nc()
		}
		q0 := t.Backnl(t.iq1, 1)
		t.SetOrigin(q0, true)
	} else {
		t.Show(t.file.b.Nc(), t.file.b.Nc(), false)
	}
}

// KeyLineBeginning handles pressing a key to move to the beginning of the line.
func (t *Text) KeyLineBeginning() {
	t.TypeCommit()
	/* go to where ^U would erase, if not already at BOL */
	nnb := 0
	if t.q0 > 0 && t.ReadC(t.q0-1) != '\n' {
		nnb = t.BsWidth(0x15)
	}
	t.Show(t.q0-nnb, t.q0-nnb, true)
}

// KeyLineEnding handles pressing a key to move to the end of the line.
func (t *Text) KeyLineEnding() {
	t.TypeCommit()
	q0 := t.q0
	for q0 < t.file.b.Nc() && t.ReadC(q0) != '\n' {
		q0++
	}
	t.Show(q0, q0, true)
}

// KeyCmdC handles ⌘-C
func (t *Text) KeyCmdC() {
	t.TypeCommit()
	cut(t, t, nil, true, false, "")
}

// KeyCmdZ handles⌘-Z
func (t *Text) KeyCmdZ() {
	t.TypeCommit()
	undo(t, nil, nil, true, false, "")
}

// KeyShiftCmdZ handles ⌘-Shift-C
func (t *Text) KeyShiftCmdZ() {
	t.TypeCommit()
	undo(t, nil, nil, false, false, "")
}

// bodyfilemark updates the sequence and sets a file mark.
func (t *Text) bodyfilemark() {
	if t.what == Body {
		seq++
		t.file.Mark()
	}
}

// KeyCmdX handles ⌘X
func (t *Text) KeyCmdX() {
	t.bodyfilemark()
	t.TypeCommit()
	if t.what == Body {
		seq++
		t.file.Mark()
	}
	cut(t, t, nil, true, true, "")
	t.Show(t.q0, t.q0, true)
	t.iq1 = t.q0
	return
}

// KeyCmdV handles ⌘V
func (t *Text) KeyCmdV() {
	t.bodyfilemark()
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

// alterselection removes any existing text range and returns if it did.
func (t *Text) alterselection() bool {
	t.bodyfilemark()
	wasrange := t.q0 != t.q1
	if t.q1 > t.q0 {
		if t.ncache != 0 {
			acmeerror("text.type", nil)
		}
		cut(t, t, nil, true, true, "")
		t.eq0 = ^0
	}
	t.Show(t.q0, t.q0, true)
	return wasrange
}

// KeyAutoComplete handles automatic expansion via the autocompletion
// system.
func (t *Text) KeyAutoComplete() {
	t.alterselection()
	t.TypeCommit()
	rp := t.Complete()
	if rp == nil {
		return
	}
	t.KeyDefault(rp)
}

// KeyDefault inserts the rune(s) in rp as normal typing. It inserts
// the runes into the caches of all texts.
func (t *Text) KeyDefault(rp []rune) {
	nr := len(rp)
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

	// TODO(rjk): Should be part of the newline handler and not here.
	// Make sure that this block and subsequent assignment can be
	// re-ordered.
	if rp[0] == '\n' && t.w != nil {
		t.w.Commit(t)
	}
	t.iq1 = t.q0
}

// KeyEsc handles ESC
// TODO(rjk): Clarify the intended purpose of ESC.
func (t *Text) KeyEsc() {
		t.alterselection()
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
}


func (t *Text) KeyEraseCharacterRight() {
		// TODO(rjk): Why doesn't this need to loop over the
		// zeroxes?
		wasrange := t.alterselection()
		if t.q1 >= t.Nc()-1 {
			return // End of file
		}
		t.TypeCommit() // Avoid messing with the cache?
		if !wasrange {
			t.q1++
			cut(t, t, nil, false, true, "")
		}
}



func (t *Text) KeyEraseCharactersLeft(r rune) {
		t.alterselection()
		if t.q0 == 0 { /* nothing to erase */
			return
		}
		// TODO(rjk): Generalize this more nicely.
		nnb := t.BsWidth(r)
		q1 := t.q0
		q0 := q1 - nnb
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
			nb := nnb
			n := u.ncache
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
}

// HERE. refactor me.
func (t *Text) KeyNewline() {
		t.alterselection()
			rp := make([]rune, nnb+1) //runemalloc(nnb + 1);
		if t.w.autoindent {
			/* find beginning of previous line using backspace code */
			nnb := t.BsWidth(0x15)    /* ^U case */
			nr := 0
			// TODO(rjk): Conceivably this is non-ideal?
			rp[nr] =  '\n'
			nr++
			for i := 0; i < nnb; i++ {
				r := t.ReadC(t.q0 - nnb + i)
				if r != ' ' && r != '\t' {
					break
				}
				rp[nr] = r
				nr++
			}
			rp = rp[:nr]
		}
	t.KeyDefault(rp)
}
