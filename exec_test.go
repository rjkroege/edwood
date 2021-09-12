package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rjkroege/edwood/file"
)

func acmeTestingMain() {
	global.acmeshell = os.Getenv("acmeshell")
	global.cwait = make(chan ProcessState)
	global.cerr = make(chan error)
	go func() {
		for range global.cerr {
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
		{false, true, true, "nonexistentcommand", ""},

		// Hard: must be executed using a shell
		{true, false, false, "ls '.'", ""},
		{true, false, false, " ls '.' ", ""},
		{true, false, false, "	 ls	 '.'	 ", ""},
		{true, false, false, "ls '.'", "."},
		{true, false, true, "dat\x08\x08ate", ""},
		{true, false, true, "/non-existent-command", ""},
	}
	acmeTestingMain()

	for _, tc := range tt {
		// runproc goes into Hard path if acmeshell is non-empty.
		// Unset acmeshell for non-hard cases.
		if tc.hard {
			global.acmeshell = os.Getenv("acmeshell")
		} else {
			global.acmeshell = ""
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
			status := <-global.cwait
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

func TestPutfile(t *testing.T) {
	dir, err := ioutil.TempDir("", "edwood.test")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	filename := filepath.Join(dir, "hello.txt")
	err = ioutil.WriteFile(filename, nil, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	checkFile := func(t *testing.T, content string) {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		s := string(b)
		if s != content {
			t.Errorf("file content is %q; expected %q", s, content)
		}
	}

	want := "Hello, 世界\n"
	w := &Window{
		body: Text{
			file: file.MakeObservableEditableBuffer(filename, []rune(want)),
		},
	}
	f := w.body.file
	file := w.body.file
	cur := &w.body
	cur.w = w
	file.SetCurObserver(cur)
	increaseMtime := func(t *testing.T, duration time.Duration) {
		tm := file.Info().ModTime().Add(duration)
		if err := os.Chtimes(filename, tm, tm); err != nil {
			t.Fatalf("Chtimes failed: %v", err)
		}
	}

	err = putfile(file, 0, f.Nr(), filename)
	if err == nil || !strings.Contains(err.Error(), "file already exists") {
		t.Fatalf("putfile returned error %v; expected 'file already exists'", err)
	}
	err = putfile(file, 0, f.Nr(), filename)
	if err != nil {
		t.Fatalf("putfile failed: %v", err)
	}
	checkFile(t, want)

	// mtime increased but hash is the same
	increaseMtime(t, time.Second)
	err = putfile(file, 0, f.Nr(), filename)
	if err != nil {
		t.Fatalf("putfile failed: %v", err)
	}
	checkFile(t, want)

	// mtime increased and hash changed
	want = "Hello, 世界\nThis line added outside of Edwood.\n"
	err = ioutil.WriteFile(filename, []byte(""), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	increaseMtime(t, time.Second)
	err = putfile(file, 0, f.Nr(), filename)
	if err == nil || !strings.Contains(err.Error(), "modified since last read") {
		t.Fatalf("putfile returned error %v; expected 'modified since last read'", err)
	}
}

func TestExpandtabToggle(t *testing.T) {
	want := true
	w := &Window{
		body: Text{
			file:      file.MakeObservableEditableBuffer("", nil),
			tabexpand: false,
			tabstop:   4,
		},
	}
	text := &w.body
	text.w = w
	text.tabexpand = !want

	expandtab(text, text, text, false, false, "")
	te := text.w.body.tabexpand
	if te != want {
		t.Errorf("tabexpand is set to %v; expected %v", te, want)
	}
}

// Observation: making this particular test useful requires multiple
// refactorings to fully exercise all code paths through cut.
// I expect that this observation applies to almost all of the functions
// noted in exectab.
func TestCut(t *testing.T) {
	prefix := "Hello "
	suffix := "世界\n"
	w := &Window{
		body: Text{
			file: file.MakeObservableEditableBuffer("cuttest", []rune(prefix+suffix)),
		},
	}

	bodytext := &w.body

	// TODO(rjk): Setting this will cause the test to crash because it
	// requires bodytext to have a valid frame. But without setting this,
	// it's impossible to get good test coverage.
	// bodytext.w = w

	w.body.q0 = 0
	w.body.q1 = len(prefix)

	cut(bodytext, bodytext, nil, false, true, "")

	if got, want := w.body.file.String(), suffix; got != want {
		t.Errorf("text not cut got %q, want %q", got, want)
	}

	if got, want := w.body.q0, 0; got != want {
		t.Errorf("text q0 wrong after cut got %v, want %v", got, want)
	}
	if got, want := w.body.q1, 0; got != want {
		t.Errorf("text q0 wrong after cut got %v, want %v", got, want)
	}
}

// TODO(rjk): test undo.
