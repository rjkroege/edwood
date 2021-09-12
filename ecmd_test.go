package main

import (
	"testing"

	"github.com/rjkroege/edwood/file"
)

// Test for https://github.com/rjkroege/edwood/issues/291
// {gh issue view 291}
func DISABLED_TestXCmdPipeMultipleWindows(t *testing.T) {
	global.cedit = make(chan int)
	global.ccommand = make(chan *Command)
	global.cwait = make(chan ProcessState)

	newWindow := func(name string) *Window {
		w := NewWindow()
		w.body.file = file.MakeObservableEditableBuffer(name, nil)
		w.body.w = w
		w.body.fr = &MockFrame{}
		w.body.file.AddObserver(&w.body)
		w.tag.file = file.MakeObservableEditableBuffer("", nil)
		w.tag.w = w
		w.tag.fr = &MockFrame{}
		w.tag.file.AddObserver(&w.tag)
		w.editoutlk = make(chan bool, 1)
		return w
	}
	global.row = Row{
		col: []*Column{
			{
				w: []*Window{
					newWindow("one.txt"),
					newWindow("two.txt"),
				},
			},
		},
	}
	defer func() {
		global.cedit = nil
		global.ccommand = nil
		global.cwait = nil
		global.row = Row{}

		warningsMu.Lock()
		defer warningsMu.Unlock()
		// remove fsysmount failure warning
		warnings = []*Warning{}
	}()

	// All middle button commands including Edit run inside a lock discipline
	// set up by MovedMouse.
	global.row.lk.Lock()
	defer global.row.lk.Unlock()

	cp := &cmdParser{
		buf: []rune("X |cat\n"),
		pos: 0,
	}
	cmd, err := cp.parse(0)
	if err != nil {
		t.Fatalf("failed to parse command: %v", err)
	}
	X_cmd(nil, cmd)
}
