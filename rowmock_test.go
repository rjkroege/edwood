package main

// Utility functions to build complex mocks of the Edwood
// row/column/window model.

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
)

// configureGlobals setups global variables so that Edwood can operate on
// a scaffold model.
func (g *globals) configureGlobals(d draw.Display) {
	g.mouse = new(draw.Mouse)
	g.button = edwoodtest.NewImage(d, "button", image.Rect(0, 0, 10, 10))
	g.modbutton = edwoodtest.NewImage(d, "modbutton", image.Rect(0, 0, 10, 10))
	g.colbutton = edwoodtest.NewImage(d, "colbutton", image.Rect(0, 0, 10, 10))

	// Set up Undo to make sure that we see undoable results.
	// By default, post-load, file.seq, file.putseq = 0, 0.
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
			file: file.MakeObservableEditableBuffer("", nil),
		}, &content.RowTag, display),
	}
	global.row.tag.Insert(0, []rune(content.RowTag.Buffer), true)

	// TODO(rjk): Consider calling Column.Init?
	cols := make([]*Column, 0, len(content.Columns))
	for _, sercol := range content.Columns {
		col := &Column{
			tag: *updateText(&Text{
				what: Columntag,
				file: file.MakeObservableEditableBuffer("", nil),
			}, &sercol.Tag, display),
			display: display,
			fortest: true,
			w:       make([]*Window, 0),
		}
		col.safe = true
		cols = append(cols, col)
	}

	// This has to be done first.
	global.row.col = cols
	global.configureGlobals(display)

	for _, serwin := range content.Windows {
		w := NewWindow().initHeadless(nil)
		w.display = display
		w.tag.display = display
		w.body.display = display
		w.body.w = w
		w.body.what = Body
		w.tag.w = w
		w.tag.what = Tag

		wincol := cols[serwin.Column]
		wincol.w = append(wincol.w, w)
		w.col = wincol
		w.body.col = wincol
		w.tag.col = wincol
		updateText(&w.tag, &serwin.Tag, display)
		updateText(&w.body, &serwin.Body, display)
		w.SetName(strings.SplitN(serwin.Tag.Buffer, " ", 2)[0])
	}
}

// InsertString inserts a string at the beginning of a buffer. It doesn't
// update the selection.
func InsertString(w *Window, s string) {
	// Set an undo point before the insertion. (So that the insertion is undoable)
	w.body.file.Mark(global.seq)
	global.seq++
	w.body.Insert(0, []rune(s), true)
}

// windowScaffoldOption is a configurable option type.
type windowScaffoldOption func(*scaffoldBuilder)

// scaffoldBuilder accumulates state set by the options into what's
// needed to build an array of dumpfile.Window objects.
type scaffoldBuilder struct {
	winbyname map[string]*dumpfile.Window
	windows   []*dumpfile.Window
	dirs      map[string]string
	t         testing.TB
}

func (sb *scaffoldBuilder) dumpfile() *dumpfile.Content {
	return &dumpfile.Content{
		Columns: []dumpfile.Column{
			{},
		},
		Windows: sb.windows,
	}
}

// FlexiblyMakeWindowScaffold is wrapper around MakeWindowScaffold that
// provides easily configurable Window scaffold structures.
func FlexiblyMakeWindowScaffold(t testing.TB, opts ...windowScaffoldOption) {
	t.Helper()

	sb := &scaffoldBuilder{
		windows:   make([]*dumpfile.Window, 0),
		winbyname: make(map[string]*dumpfile.Window),
		dirs:      make(map[string]string),
		t:         t,
	}

	for _, opt := range opts {
		opt(sb)
	}
	MakeWindowScaffold(sb.dumpfile())
}

// ScDir sets the backing directory for window id. Setting a backing
// directory implies persisting the body to the file formed by Join(path,
// id).
func ScDir(path, id string) windowScaffoldOption {
	return func(f *scaffoldBuilder) {
		f.t.Helper()

		w, ok := f.winbyname[id]
		if !ok {
			f.t.Fatalf("Dir option on non-existent window %s", id)
		}

		path := filepath.Join(path, id)
		w.Tag.Buffer = path

		if err := os.WriteFile(path, []byte(w.Body.Buffer), 0644); err != nil {
			f.t.Fatalf("%s can't write %q: %v", "Dir", path, err)
		}
		// Stash the path here so that Dir and Body can be in arbitrary order.
		f.dirs[id] = path
	}
}

// ScBody sets contents and persists it if there's a dir.
// TODO(rjk): Consider placing in a different package.
func ScBody(id, contents string) windowScaffoldOption {
	return func(f *scaffoldBuilder) {
		f.t.Helper()

		w, ok := f.winbyname[id]
		if !ok {
			f.t.Fatalf("Dir option on non-existent window %s", id)
		}

		w.Body.Buffer = contents
		// Body can come both after and before Dir.
		if path, ok := f.dirs[id]; ok {
			if err := os.WriteFile(path, []byte(w.Body.Buffer), 0644); err != nil {
				f.t.Fatalf("%s can't write %q: %v", "Dir", path, err)
			}
		}
	}
}

// ScWin declares a new window with identifier name. Subsequent options
// use name to specify the window that they effect. name needs to be a
// valid file name for backed Windows.
func ScWin(name string) windowScaffoldOption {
	return func(f *scaffoldBuilder) {
		f.t.Helper()
		w := &dumpfile.Window{
			Tag: dumpfile.Text{
				Buffer: name,
			},
		}
		f.windows = append(f.windows, w)
		f.winbyname[name] = w
	}
}

func ScBodyRange(id string, dot Range) windowScaffoldOption {
	return func(f *scaffoldBuilder) {
		f.t.Helper()
		w, ok := f.winbyname[id]
		if !ok {
			f.t.Fatalf("Dir option on non-existent window %s", id)
		}

		w.Body.Q0 = dot.q0
		w.Body.Q1 = dot.q1
	}
}

// Repeating generates a sequence of identical lines.
func Repeating(n int, s string) string {
	var buffy strings.Builder

	for i := 0; i < n; i++ {
		buffy.WriteString(s)
		buffy.WriteRune('\n')
	}

	return buffy.String()
}
