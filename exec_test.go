package main

import (
	"os"
	"testing"
)

func acmeTestingMain() {
	cwait = make(chan *os.ProcessState)
	cerr = make(chan error)
	go func() {
		for {
			select {
			case <-cerr:
				// Do nothing with command output.
			}
		}
	}()
}

func TestRunproc(t *testing.T) {
	tt := []struct {
		s, arg string
	}{
		{"ls", ""},
		{"ls .", ""},
		{" ls . ", ""},
		{"	 ls	 .	 ", ""},
		{"ls", "."},

		// Hard: executed using a shell
		{"ls '.'", ""},
		{" ls '.' ", ""},
		{"	 ls	 '.'	 ", ""},
		{"ls '.'", "."},
	}
	acmeTestingMain()

	for _, tc := range tt {
		cpid := make(chan *os.Process)
		go runproc(nil, tc.s, "", false, "", tc.arg, &Command{}, cpid, false)
		<-cpid
		status := <-cwait
		if !status.Success() {
			t.Errorf("command %q exited with status %v\n", tc.s, status)
		}
	}
}
