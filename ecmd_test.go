package main

import (
	"testing"
)

// Test for https://github.com/rjkroege/edwood/issues/291
func TestXCmdPipeMultipleWindows(t *testing.T) {
	cedit = make(chan int)
	ccommand = make(chan *Command)
	cwait = make(chan ProcessState)

	newWindow := func(name string) *Window {
		w := NewWindow()
		w.body.file = NewFile(name)
		w.body.w = w
		w.body.fr = &MockFrame{}
		w.body.file.text = []*Text{&w.body}
		w.body.file.curtext = &w.body
		w.tag.file = NewFile("")
		w.tag.w = w
		w.tag.fr = &MockFrame{}
		w.tag.file.text = []*Text{&w.tag}
		w.tag.file.curtext = &w.tag
		w.editoutlk = make(chan bool, 1)
		return w
	}
	row = Row{
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
		cedit = nil
		ccommand = nil
		cwait = nil
		row = Row{}
	}()

	done := make(chan struct{})
	go func() { // waitthread
		<-ccommand
		<-cwait
		cedit <- 0
		close(done)
	}()

	// All middle button commands including Edit run inside a lock discipline
	// set up by MovedMouse.
	row.lk.Lock()
	defer row.lk.Unlock()

	cp := &cmdParser{
		buf: []rune("X |cat\n"),
		pos: 0,
	}
	cmd, err := cp.parse(0)
	if err != nil {
		t.Fatalf("failed to parse command: %v", err)
	}
	X_cmd(nil, cmd)
	<-done
}
