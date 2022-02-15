package main

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
)

func TestIsmtpt(t *testing.T) {
	oldmtpt := mtpt
	defer func() { mtpt = oldmtpt }()
	*mtpt = "/mnt/acme"

	testCases := []struct {
		filename string
		ok       bool
	}{
		{"/mnt/acme", true},
		{"/mnt/acme/", true},
		{"/mnt/acme/new", true},
		{"/mnt/acme/5/body", true},
		{"/usr/../mnt/acme", true},
		{"/usr/../mnt/acme/", true},
		{"/usr/../mnt/acme/new", true},
		{"/usr/../mnt/acme/5/body", true},
		{"/", false},
	}
	for _, tc := range testCases {
		ok := ismtpt(tc.filename)
		if ok != tc.ok {
			t.Errorf("ismtpt(%v) = %v; expected %v", tc.filename, ok, tc.ok)
		}
	}
}

func TestKillprocs(t *testing.T) {
	cmd := exec.Command("sleep", "3600")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed start command: %v", err)
	}
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	command = []*Command{
		{
			proc: cmd.Process,
		},
	}
	killprocs(nil)
	timer := time.NewTimer(5 * time.Second)
	select {
	case <-done:
		// Do nothing
	case <-timer.C:
		t.Errorf("killprocs did not kill command in time")
	}
}

// TestWaithreadCommandCycle tests that we don't create a cycle in the command linked list
// (regression test for https://github.com/rjkroege/edwood/issues/279).
func TestWaitthreadCommandCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := startMockWaitthread(ctx)
	defer func() {
		cancel() // Ask waithtread to finish up.
		<-done   // Wait for waithtread to return and finish clean up.
	}()

	var (
		c [4]*Command
		w [4]*mockProcessState
	)
	for i := 0; i < len(c); i++ {
		c[i] = &Command{
			pid:  i,
			name: fmt.Sprintf("proc%v", i),
		}
		w[i] = &mockProcessState{
			pid:     i,
			success: true,
		}
	}

	global.ccommand <- c[3]
	global.ccommand <- c[2]
	global.ccommand <- c[1]
	global.ccommand <- c[0]
	waitthreadSync()

	// command is 0 -> 1 -> 2 -> 3
	if got, want := len(command), 4; got != want {
		t.Errorf("command is length is %v; want %v", got, want)
	}

	global.cwait <- w[2] // delete 2, command is 0 -> 1 -> 3, lc = 1 -> 3
	global.cwait <- w[0] // delete 0, command is 1 -> 1 -> 1 -> ...
	global.cwait <- w[3] // try to delete 2: infinite loop
	global.cwait <- w[1]
	waitthreadSync()

	if got, want := len(command), 0; got != want {
		t.Errorf("command is length is %v; want %v", got, want)
	}
}

func TestWaitthreadEarlyExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := startMockWaitthread(ctx)
	defer func() {
		cancel() // Ask waithtread to finish up.
		<-done   // Wait for waithtread to return and finish clean up.
	}()

	c := &Command{
		pid:           42,
		name:          "proc42",
		iseditcommand: true,
	}
	w := &mockProcessState{
		pid:     42,
		success: true,
	}

	// simulate command exit before adding it to command list
	global.cwait <- w
	waitthreadSync()

	global.ccommand <- c
	<-global.cedit
	waitthreadSync()

	if got, want := len(command), 0; got != want {
		t.Errorf("command is length is %v; want %v", got, want)
	}

	global.row.lk.Lock()
	got := warnings[0].buf.String()
	global.row.lk.Unlock()
	want := "pid 42, success true\n"
	if got != want {
		t.Fatalf("warnings is %q; want %q", got, want)
	}
}

func TestWaitthreadKill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := startMockWaitthread(ctx)
	defer func() {
		cancel() // Ask waithtread to finish up.
		<-done   // Wait for waitthread to return and finish clean up.
	}()

	cmd := exec.Command("sleep", "3600")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed start command: %v", err)
	}
	waitDone := make(chan struct{})
	go func() {
		cmd.Wait()
		global.cwait <- cmd.ProcessState
		waitthreadSync()
		close(waitDone)
	}()

	c := &Command{
		pid:  cmd.Process.Pid,
		proc: cmd.Process,
		name: "sleep ",
	}
	global.ccommand <- c
	waitthreadSync()

	if got, want := command, []*Command{c}; !reflect.DeepEqual(got, want) {
		t.Errorf("command is %v; want %v", got, want)
	}

	global.ckill <- "unknown_cmd"
	waitthreadSync()

	global.row.lk.Lock()
	got := warnings[0].buf.String()
	global.row.lk.Unlock()
	want := "Kill: no process unknown_cmd\n"
	if got != want {
		t.Fatalf("warnings is %q; want %q", got, want)
	}

	global.ckill <- "sleep"
	waitthreadSync()
	<-waitDone

	if got, want := len(command), 0; got != want {
		t.Errorf("command is length is %v; want %v", got, want)
	}
}

func startMockWaitthread(ctx context.Context) (done <-chan struct{}) {
	global.ccommand = make(chan *Command)
	global.cwait = make(chan ProcessState)
	global.ckill = make(chan string)
	command = nil
	global.cerr = make(chan error)
	global.cedit = make(chan int)
	warnings = nil
	global.row = Row{
		display: edwoodtest.NewDisplay(),
		tag: Text{
			file: file.MakeObservableEditableBuffer("", nil),
		},
	}
	ch := make(chan struct{})
	go func() {
		waitthread(global, ctx)
		global.ccommand = nil
		global.cwait = nil
		global.ckill = nil
		command = nil
		global.cerr = nil
		global.cedit = nil
		warnings = nil
		global.row = Row{}
		close(ch)
	}()
	return ch
}

// waitthreadSync waits until a select case in waitthread finishes.
func waitthreadSync() { global.cerr <- fmt.Errorf("") }

type mockProcessState struct {
	pid     int
	success bool
}

func (ps *mockProcessState) Pid() int { return ps.pid }
func (ps *mockProcessState) String() string {
	return fmt.Sprintf("pid %v, success %v", ps.pid, ps.success)
}
func (ps *mockProcessState) Success() bool { return ps.success }
