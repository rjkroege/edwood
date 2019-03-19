package main

import (
	"os"
	"runtime"
	"testing"
)

func acmeTestingMain() {
	acmeshell = os.Getenv("acmeshell")
	cwait = make(chan *os.ProcessState)
	cerr = make(chan error)
	go func() {
		for range cerr {
			// Do nothing with command output.
		}
	}()
}

func TestRunproc(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tt := []struct {
		hard      bool
		startfail bool
		waitfail  bool
		s, arg    string
	}{
		{false, true, true, "", ""},
		{false, true, true, " ", ""},
		{false, true, true, "   ", "   "},
		{false, false, false, "ls", ""},
		{false, false, false, "ls .", ""},
		{false, false, false, " ls . ", ""},
		{false, false, false, "	 ls	 .	 ", ""},
		{false, false, false, "ls", "."},
		{false, false, false, "|ls", "."},
		{false, false, false, "<ls", "."},
		{false, false, false, ">ls", "."},
		{false, true, true, "nonexistantcommand", ""},

		// Hard: must be executed using a shell
		{true, false, false, "ls '.'", ""},
		{true, false, false, " ls '.' ", ""},
		{true, false, false, "	 ls	 '.'	 ", ""},
		{true, false, false, "ls '.'", "."},
		{true, false, true, "dat\x08\x08ate", ""},
		{true, false, true, "/non-existant-command", ""},
	}
	acmeTestingMain()

	for _, tc := range tt {
		// runproc goes into Hard path if acmeshell is non-empty.
		// Unset acmeshell for non-hard cases.
		if tc.hard {
			acmeshell = os.Getenv("acmeshell")
		} else {
			acmeshell = ""
		}

		cpid := make(chan *os.Process)
		done := make(chan struct{})
		go func() {
			err := runproc(nil, tc.s, "", false, "", tc.arg, &Command{}, cpid, false)
			if tc.startfail && err == nil {
				t.Errorf("expected command %q to fail", tc.s)
			}
			if !tc.startfail && err != nil {
				t.Errorf("runproc failed for command %q: %v", tc.s, err)
			}
			close(done)
		}()
		proc := <-cpid
		if !tc.waitfail && proc == nil {
			t.Errorf("nil proc for command %v", tc.s)
		}
		if proc != nil {
			status := <-cwait
			if tc.waitfail && status.Success() {
				t.Errorf("command %q exited with status %v", tc.s, status)
			}
			if !tc.waitfail && !status.Success() {
				t.Errorf("command %q exited with status %v", tc.s, status)
			}
		}
		<-done
	}
}
