package main

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/rjkroege/edwood/internal/edwoodtest"
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

	command = &Command{
		proc: cmd.Process,
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
	ccommand = make(chan *Command)
	cwait = make(chan ProcessState)
	row = Row{
		display: edwoodtest.NewDisplay(),
		tag: Text{
			file: NewFile(""),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		waitthread(ctx)
		ccommand = nil
		cwait = nil
		row = Row{}
		close(done)
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

	ccommand <- c[3]
	ccommand <- c[2]
	ccommand <- c[1]
	ccommand <- c[0]

	// command is 0 -> 1 -> 2 -> 3

	cwait <- w[2] // delete 2, command is 0 -> 1 -> 3, lc = 1 -> 3
	cwait <- w[0] // delete 0, command is 1 -> 1 -> 1 -> ...
	cwait <- w[3] // try to delete 2: infinite loop

	cancel() // Ask waithtread to finish up.

	// Wait for waithtread to return and finish clean up.
	<-done
}

type mockProcessState struct {
	pid     int
	success bool
}

func (ps *mockProcessState) Pid() int { return ps.pid }
func (ps *mockProcessState) String() string {
	return fmt.Sprintf("pid %v, success %v", ps.pid, ps.success)
}
func (ps *mockProcessState) Success() bool { return ps.success }
