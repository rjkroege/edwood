package main

// Utility functions to build complex mocks of the Edwood
// row/column/window model.

import (
	"image"
	"strings"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
)

// configureGlobals setups global variables so that Edwood can operate on
// a scaffold model.
func configureGlobals() {
	// TODO(rjk): Make a proper mock draw.Mouse in edwoodtest.
	global.mouse = new(draw.Mouse)
	global.button = edwoodtest.NewImage(image.Rect(0, 0, 10, 10))
	global.modbutton = edwoodtest.NewImage(image.Rect(0, 0, 10, 10))
	global.colbutton = edwoodtest.NewImage(image.Rect(0, 0, 10, 10))

	// Set up Undo to make sure that we see undoable results.
	// By default, post-load, file.seq, file.putseq = 0, 0.
	global.seq = 1
}

// updateText creates a minimal mock Text object from data embedded inside
// of an Edwood dumpfile structure.
func updateText(t *Text, sertext *dumpfile.Text, display draw.Display) *Text {
	t.display = display
	t.fr = &MockFrame{}
	t.Insert(0, []rune(sertext.Buffer), true)
	t.SetQ0(sertext.Q0)
	t.SetQ1(sertext.Q1)

	return t
}

// MakeWindowScaffold builds a complete scaffold model of the Edwood
// row/col/window hierarchy sufficient to run sam commands. It is
// configured from the intermediate model used by the Edwood JSON dump
// file.
//
// The built-up global state's bodies will have
// ObservableEditableBuffer.Dirty() return false. This may not accurately
// reflect the state of the model under non-test operating conditions.
// Callers of this function should adjust the dirty state externally.
func MakeWindowScaffold(content *dumpfile.Content) {
	display := edwoodtest.NewDisplay()
	global.seq = 0

	global.row = Row{
		display: display,
		tag: *updateText(&Text{
			what: Rowtag,
			file: file.MakeObservableEditableBufferTag(nil),
		}, &content.RowTag, display),
	}

	cols := make([]*Column, 0, len(content.Columns))
	for _, sercol := range content.Columns {
		col := &Column{
			tag: *updateText(&Text{
				what: Columntag,
				file: file.MakeObservableEditableBufferTag(nil),
			}, &sercol.Tag, display),
			display: display,
			fortest: true,
			w:       make([]*Window, 0),
		}
		cols = append(cols, col)
	}

	for _, serwin := range content.Windows {
		w := NewWindow().initHeadless(nil)
		w.display = display
		updateText(&w.body, &serwin.Body, display)
		updateText(&w.tag, &serwin.Tag, display)
		w.body.file.SetName(strings.SplitN(serwin.Tag.Buffer, " ", 2)[0])
		w.body.w = w
		// will probably break stuff...
		w.body.what = Body
		w.tag.w = w
		// will probably break stuff...
		w.tag.what = Tag

		wincol := cols[serwin.Column]
		wincol.w = append(wincol.w, w)
		w.col = wincol
		w.body.col = wincol
		w.tag.col = wincol
	}

	global.row.col = cols
	configureGlobals()
}

// InsertString inserts a string at the beginning of a buffer. It doesn't
// update the selection.
func InsertString(w *Window, s string) {
	// Set an undo point before the insertion. (So that the insertion is undoable)
	w.body.file.Mark(global.seq)
	global.seq++
	w.body.Insert(0, []rune(s), true)
}
