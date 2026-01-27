// Tag related code.
package main

import (
	"strings"
	"unicode/utf8"

	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/runes"
	"github.com/rjkroege/edwood/util"
	//	"log"
)

// TODO(rjk): This implementation seems non-ideal: why read the whole
// buffer just to delete everything after |
func (w *Window) ClearTag() {
	// w must be committed
	n := w.tag.Nc()
	r := make([]rune, n)
	w.tag.file.Read(0, r)
	i := w.tagfilenameend
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
	fnr := make([]rune, w.tagfilenameend)
	w.tag.file.Read(0, fnr)
	sfnr := string(fnr)
	if len(sfnr) > 0 && sfnr[0] == '\'' {
		return UnquoteFilename(sfnr)
	}
	return sfnr
}

// TODO(rjk): Consider using a regexp for this function?
// returns the filename text scraped from the tag, including any quoting.
func parsetaghelper(tag string) string {
	// " |" or "\t|" ends left half of tag
	// PAL: Filenames start at the start of the tag.  A filename in the tag may be
	// surrounded by single quotes, at which time the filename ends at the matching quote.
	// Otherwise the filename ends at the first space.
	if len(tag) == 0 {
		return ""
	}
	if tag[0] == '\'' {
		endquoteidx := strings.Index(tag[1:], "'")
		if endquoteidx >= 0 {
			return tag[0 : endquoteidx+2]
		}
	}

	// The filename ends at a space
	endidx := strings.IndexAny(tag, " \t")
	if endidx == -1 {
		// our tag is all one word.  Unusual, but ok.
		return tag
	}
	return tag[0:endidx]
	/*
	   // If we find " Del Snarf" in the left half of the tag
	   // (before the pipe), that ends the file name.
	   pipe := strings.Index(tag, " |")

	   	if i := strings.Index(tag, "\t|"); i >= 0 && (pipe < 0 || i < pipe) {
	   		pipe = i
	   	}

	   // It's arguable that we should not permit the creation of filenames with
	   // a trailing space in their names because the likelihood of doing this
	   // by accident is higher than the number of times that this is desirable.

	   	if i := strings.Index(tag, " Del Snarf"); i >= 0 && (pipe < 0 || i < pipe) {
	   		return tag[:i]
	   	}

	   	if i := strings.IndexAny(tag, " \t"); i >= 0 {
	   		return tag[:i]
	   	}

	   return tag
	*/
}

// NB the sequencing: carefully. actions happen on the body. The result
// is firing the UpdateTag observer. That method calls setTag1. setTag1
// mutates the tag. The edits to the the tag invoke the TagIndex
// observers and it updates the index. In particular: when UpdateTag
// runs, it may assume that the tagindex is valid.
//
// This is problematic because it will be invoked (sometimes) from within setTag
// and sometimes from elsewhere. So track that.
func (w *Window) Inserted(oq0 file.OffsetTuple, b []byte, nr int) {
	if w.tagsetting {
		// We have invoked this within setTag1 so do nothing because setTag1 is
		// responsible for actually updating the tagfilenameend
		return
	}

	q0 := oq0.R

	switch {
	case q0 == 0 && b[0] == '\t',
		q0 == 0 && b[0] == ' ':
		w.tagfilenameend = 0
		w.tagfilenamechanged = true
	case w.tagfilenameend == 0 && q0 == 0 && w.tag.Nc() == nr:
		// TODO(rjk): Turn this into byte ops.
		tagcontents := make([]rune, w.tag.Nc())
		w.tag.file.Read(0, tagcontents)
		w.tagfilenameend = len(parsetaghelper(string(tagcontents)))
		w.tagfilenamechanged = true
	case q0 <= w.tagfilenameend:
		w.tagfilenameend += nr
		w.tagfilenamechanged = true
	}
}

func (w *Window) Deleted(oq0, oq1 file.OffsetTuple) {
	q0 := oq0.R
	q1 := oq1.R

	if w.tagsetting {
		// We have been invoking this within setTag1. So do nothing.
		return
	}

	switch {
	case q1 < w.tagfilenameend:
		w.tagfilenameend -= (q1 - q0)
		w.tagfilenamechanged = true
	case q0 < w.tagfilenameend && q1 >= w.tagfilenameend, q0 == w.tagfilenameend:
		tagcontents := make([]rune, w.tag.Nc())
		w.tag.file.Read(0, tagcontents)
		w.tagfilenameend = len(parsetaghelper(string(tagcontents)))
		w.tagfilenamechanged = true
	}
	// TODO(rjk) Test what happens for deletion cutting into the " Del Snarf..."?
}

// ForceSetWindowTag force sets the tag when the tag needs to change
// without a body modification to trigger the tag update.
// TODO(rjk): Uses of this method are probably code cleanup opportuniies.
func (w *Window) ForceSetWindowTag() {
	if w.col.safe || w.tag.fr.GetFrameFillStatus().Maxlines > 0 {
		w.setTag1()
	}
}

// Use to debug.
// func (w *Window) setTagDiag(where, when string) {
// 	log.Println("<<", where, when, w.tag.q0, w.tag.q1, w.tag.DebugString())
// }

// setTag1 updates the tag contents for a given window w.
// TODO(rjk): Note the col.safe test... should I do this as part of setTag1()?
func (w *Window) setTag1() {
	// w.setTagDiag("setTag1", "before")
	// defer w.setTagDiag("setTag1","after")

	// TODO(rjk): Figure out if I need this. Presumably this is needed to
	// make things display correctly when filesystem changes to the tag
	// happen while the window is collapsed to 0?
	if !w.col.safe && w.tag.fr.GetFrameFillStatus().Maxlines == 0 {
		// log.Println("Window.setTag1 early exit")
		return
	}

	w.tagsetting = true
	defer func() { w.tagsetting = false }()

	const (
		Ldelsnarf = " Del Snarf"
		Lundo     = " Undo"
		Lredo     = " Redo"
		Lget      = " Get"
		Lput      = " Put"
		Llook     = " Look"
		Ledit     = " Edit"
		Lmarkdeep = " Markdeep"
		Lpipe     = " |"
	)

	// (flux) The C implemtation does a lot of work to avoid re-setting the
	// tag text if unchanged. Edwood uses the observer facility to reduce the
	// number of calls to setTag1.

	var sb strings.Builder
	sb.WriteString(QuoteFilename(w.body.file.Name()))
	sb.WriteString(Ldelsnarf)

	oldfnend := w.tagfilenameend
	w.tagfilenameend = utf8.RuneCountInString(QuoteFilename(w.body.file.Name()))

	if w.filemenu {
		if w.body.file.HasUndoableChanges() {
			sb.WriteString(Lundo)
		}
		if w.body.file.HasRedoableChanges() {
			sb.WriteString(Lredo)
		}
		if w.body.file.SaveableAndDirty() {
			sb.WriteString(Lput)
		}
	}
	// TODO(rjk): What happens if I make a directory into a file.
	if w.body.file.IsDir() {
		sb.WriteString(Lget)
	}
	oldbarIndex := w.tag.file.IndexRune('|')
	if oldbarIndex >= 0 {
		// TODO(rjk): Update for file.Buffer representation.
		oldsuffix := make([]rune, w.tag.file.Nr()-oldbarIndex)
		w.tag.file.Read(oldbarIndex, oldsuffix)
		sb.WriteString(" ")
		sb.WriteString(string(oldsuffix))
	} else {
		sb.WriteString(Lpipe)
		sb.WriteString(Llook)
		sb.WriteString(Ledit)
		// Add Markdeep command for markdown files
		if strings.HasSuffix(strings.ToLower(w.body.file.Name()), ".md") {
			sb.WriteString(Lmarkdeep)
		}
		sb.WriteString(" ")
	}

	newtag := []rune(sb.String())

	// replace tag if the new one is different
	resize := false
	if !runes.Equal(newtag, []rune(w.tag.file.String())) {
		resize = true // Might need to resize the tag
		// try to preserve user selection
		newbarIndex := runes.IndexRune(newtag, '|') // New always has '|'
		q0 := w.tag.q0
		q1 := w.tag.q1

		// These alter the Text's selection values.
		w.tag.Delete(0, w.tag.Nc(), true)
		w.tag.Insert(0, newtag, true)

		// log.Println("sorting the selection:", "q0", q0, "q1", q1, "oldfnend", oldfnend, "oldbarIndex", oldbarIndex, "tagfilenameend", w.tagfilenameend, "tag q0", w.tag.q0, "tag q1", w.tag.q1)
		// TODO(rjk): Consider adjusting this for better unit-testing.

		switch { // q0
		case q0 <= w.tagfilenameend:
			// log.Println("q0 < w.tagfilenameend")
			w.tag.q0 = q0
		case q0 >= w.tagfilenameend && q0 <= oldfnend:
			// log.Println("q0 >= w.tagfilenameend && q0 < oldfnend")
			w.tag.q0 = w.tagfilenameend
		case oldbarIndex != -1 && q0 > oldbarIndex:
			// log.Println("oldbarIndex != -1 && q0 > oldbarIndex")
			bar := newbarIndex - oldbarIndex
			w.tag.q0 = q0 + bar
		case q0 > oldfnend && oldbarIndex != -1 && q0 < oldbarIndex:
			// log.Println("q0 > oldfnend && oldbarIndex != -1 && q0 < oldbarIndex")
			w.tag.q0 = w.tagfilenameend
		default:
			// log.Println("q0 default")
			w.tag.q0 = util.Min(q0, w.tag.Nc())
		}

		switch { // q1
		case q1 <= w.tagfilenameend:
			// log.Println("q1 < w.tagfilenameend")
			w.tag.q1 = q1
		case q1 >= w.tagfilenameend && q1 <= oldfnend:
			// log.Println("q1 >= w.tagfilenameend && q1 < oldfnend")
			w.tag.q1 = w.tagfilenameend
		case oldbarIndex != -1 && q1 > oldbarIndex:
			// log.Println("oldbarIndex != -1 && q1 > oldbarIndex")
			bar := newbarIndex - oldbarIndex
			w.tag.q1 = q1 + bar
		case q1 > oldfnend && oldbarIndex != -1 && q1 < oldbarIndex:
			// log.Println("q1 > oldfnend && oldbarIndex != -1 && q1 < oldbarIndex")
			w.tag.q1 = w.tagfilenameend
		case q1 >= oldfnend && q1 < oldbarIndex:
			// If mutating the auto-munged text.
			w.tag.q1 = w.tag.Nc()
		default:
			// log.Println("q1", "default")
			w.tag.q1 = util.Min(q1, w.tag.Nc())
		}
	}

	w.tag.file.Clean()
	w.tag.q0 = util.Min(w.tag.q0, w.tag.Nc())
	w.tag.q1 = util.Min(w.tag.q1, w.tag.Nc())

	// TODO(rjk): This can redraw the selection unnecessarily.
	w.tag.SetSelect(w.tag.q0, w.tag.q1)
	w.DrawButton()
	if resize {
		w.tagsafe = false
		w.Resize(w.r, true, true)
	}
}
