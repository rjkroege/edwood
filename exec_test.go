package main

import (
	"os"
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
		go func() {
			err := runproc(nil, tc.s, "", false, "", tc.arg, &Command{}, cpid, false)
			if err != nil {
				t.Errorf("runproc failed for command %q: %v", tc.s, err)
				cwait <- nil
			}
		}()
		<-cpid
		status := <-cwait
		if status != nil && !status.Success() {
			t.Errorf("command %q exited with status %v", tc.s, status)
		}
	}
}
