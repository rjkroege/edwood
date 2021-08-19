package main

// Utility functions to build complex mocks of the Edwood
// row/column/window model.

import (
	"github.com/rjkroege/edwood/file"
	"image"
	"strings"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/edwoodtest"
)

// configureGlobals setups global variables so that Edwood can operate on
// a scaffold model.
func configureGlobals() {
	// TODO(rjk): Make a proper mock draw.Mouse in edwoodtest.
	mouse = new(draw.Mouse)
	button = edwoodtest.NewImage(image.Rect(0, 0, 10, 10))
	modbutton = edwoodtest.NewImage(image.Rect(0, 0, 10, 10))
	colbutton = edwoodtest.NewImage(image.Rect(0, 0, 10, 10))

	// Set up Undo to make sure that we see undoable results.
	// By default, post-load, file.seq, file.putseq = 0, 0.
	seq = 1
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
func MakeWindowScaffold(content *dumpfile.Content) {
	display := edwoodtest.NewDisplay()
	seq = 0

	row = Row{
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
		w.tag.w = w

		wincol := cols[serwin.Column]
		wincol.w = append(wincol.w, w)
		w.col = wincol
		w.body.col = wincol
		w.tag.col = wincol
	}

	row.col = cols
	configureGlobals()
}

// InsertString inserts a string at the beginning of a buffer. It doesn't
// update the selection.
func InsertString(w *Window, s string) {
	// Set an undo point before the insertion. (So that the insertion is undoable)
	w.body.file.Mark(seq)
	seq++
	w.body.Insert(0, []rune(s), true)
}
