package main

import (
	"os/exec"
	"testing"
	"time"
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
