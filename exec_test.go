package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
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

// TODO(rjk): Add A case here for partial writes.

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

func testSetupOnly(t *testing.T, g *globals) {
	t.Helper()

	// Mutate with Edit
	firstwin := g.row.col[0].w[0]
	secondwin := g.row.col[0].w[1]

	t.Log("Before seq", global.seq)
	t.Log("firstwin.body.file.Seq", g.row.col[0].w[0].body.file.Seq())
	t.Log("firstwin.tag", firstwin.tag.DebugString())
	t.Log("firstwin.body.file.HasUndoableChanges", g.row.col[0].w[0].body.file.HasUndoableChanges())
	t.Log("secondwin.body.file.Seq", g.row.col[0].w[1].body.file.Seq())
	t.Log("secondwin.tag", secondwin.tag.DebugString())
	t.Log("secondwin.body.file.HasUndoableChanges", g.row.col[0].w[1].body.file.HasUndoableChanges())
	t.Log("firstwin.tag", firstwin.tag.DebugString())

	// These should both do nothing.
	undo(&firstwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")
	undo(&secondwin.tag, nil, nil, false /* this is a redo */, false /* ignored */, "")

	t.Log("After seq", global.seq)
	t.Log("firstwin.body.file.Seq", g.row.col[0].w[0].body.file.Seq())
	t.Log("firstwin.tag", firstwin.tag.DebugString())
	t.Log("firstwin.body.file.HasUndoableChanges", g.row.col[0].w[0].body.file.HasUndoableChanges())
	t.Log("secondwin.body.file.Seq", g.row.col[0].w[1].body.file.Seq())
	t.Log("secondwin.tag", secondwin.tag.DebugString())
	t.Log("secondwin.body.file.HasUndoableChanges", g.row.col[0].w[1].body.file.HasUndoableChanges())
}

func mutateWithEdit(t *testing.T, g *globals) {
	t.Helper()

	// Mutate with Edit
	firstwin := g.row.col[0].w[0]
	// secondwin := g.row.col[0].w[1]

	t.Log("Before seq", global.seq)
	t.Log("firstwin.body.file.Seq", g.row.col[0].w[0].body.file.Seq())
	t.Log("secondwin.body.file.Seq", g.row.col[0].w[1].body.file.Seq())

	// Do I need to lock the warning?

	// Lock discipline?
	// TODO(rjk): figure out how to change this with less global dependency.
	global.row.lk.Lock()
	firstwin.Lock('M')
	global.seq++

	editcmd(&firstwin.body, []rune("X/.*file/ ,x/text/ c/TEXT/"))
	firstwin.Unlock()
	global.row.lk.Unlock()

	t.Log("After seq", global.seq)
	t.Log("firstwin.body.file.Seq", g.row.col[0].w[0].body.file.Seq())
	t.Log("secondwin.body.file.Seq", g.row.col[0].w[1].body.file.Seq())
}

func undoRedoBothMutations(t *testing.T, g *globals) {
	t.Helper()
	mutateWithEdit(t, g)

	firstwin := g.row.col[0].w[0]
	secondwin := g.row.col[0].w[1]

	// Run undo from one of the windows. (i.e. equivalent to clicking on the Undo action.)
	undo(&firstwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")

	undo(&secondwin.tag, nil, nil, false /* this is a redo */, false /* ignored */, "")
}

func mutateBothOneUndo(t *testing.T, g *globals) {
	t.Helper()
	mutateWithEdit(t, g) // Changes both.

	firstwin := g.row.col[0].w[0]

	// Modify the firstwin.
	firstwin.body.q0 = 3
	firstwin.body.q1 = 10
	global.seq++
	firstwin.body.file.Mark(global.seq)
	cut(&firstwin.tag, &firstwin.body, nil, false, true, "")

	// Run undo from first window. (i.e. equivalent to clicking on the Undo action.)
	// Should undo only the cut.
	undo(&firstwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")
}

func mutateBothOtherUndo(t *testing.T, g *globals) {
	t.Helper()
	mutateWithEdit(t, g)

	// Mutate with Edit
	firstwin := g.row.col[0].w[0]
	secondwin := g.row.col[0].w[1]

	t.Logf("firstwin, %q", firstwin.body.file.String())
	t.Logf("secondwin, %q", secondwin.body.file.String())

	// Modify the firstwin.
	firstwin.body.q0 = 3
	firstwin.body.q1 = 10
	global.seq++
	firstwin.body.file.Mark(global.seq)
	cut(&firstwin.tag, &firstwin.body, nil, false, true, "")

	t.Logf("after cut firstwin, %q", firstwin.body.file.String())

	// Run undo from one of the windows. (i.e. same as clicking on the Undo action.)
	// Cut should remain, original global edit should get Undone only in secondwin.
	undo(&secondwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")

	t.Logf("after undo firstwin, %q", firstwin.body.file.String())
	t.Logf("after undo secondwin, %q", secondwin.body.file.String())
}

func mutateBranchedAndRejoined(t *testing.T, g *globals) {
	t.Helper()

	// Mutate firstwin, secondwin simultaneously.
	mutateWithEdit(t, g)

	firstwin := g.row.col[0].w[0]
	secondwin := g.row.col[0].w[1]

	// Mutate firstwin via cut.
	firstwin.body.q0 = 3
	firstwin.body.q1 = 10
	global.seq++
	firstwin.body.file.Mark(global.seq)
	cut(&firstwin.tag, &firstwin.body, nil, false, true, "")

	undo(&secondwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")
	undo(&firstwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")

	// Should do nothing.
	undo(&secondwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")

	// Undoes the mutateWithEdit on firstwin
	undo(&firstwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")

	// Redo on secondwin puts back the change on both firstwin and secondwind.
	undo(&secondwin.tag, nil, nil, false /* this is not undo */, false /* ignored */, "")
}

func mutatePut(t *testing.T, g *globals) {
	t.Helper()

	firstwin := g.row.col[0].w[0]

	// Mutate firstwin via cut.
	firstwin.body.q0 = 3
	firstwin.body.q1 = 10
	global.seq++
	firstwin.body.file.Mark(global.seq)
	cut(&firstwin.tag, &firstwin.body, nil, false, true, "")

	// Put the instance (This fails the first time because oeb.details.Info
	// isn't set by the mock.)
	put(&firstwin.tag, nil, nil, false, true, "")
	put(&firstwin.tag, nil, nil, false, true, "")

	// Validate that the file has the right contents.
	fn := firstwin.body.file.Name()
	contents, err := os.ReadFile(fn)
	if err != nil {
		t.Errorf("mutatePut can't read output file %v", err)
	}

	if got, want := string(contents), "Thishort text\nto try addressing\n"; got != want {
		t.Errorf("mutatePut, put didn't succeed. got %q, want %q", got, want)
	}
}

func mutatePutMutate(t *testing.T, g *globals) {
	t.Helper()

	mutatePut(t, g)

	firstwin := g.row.col[0].w[0]

	// Mutate firstwin via second cut.
	firstwin.body.q0 = 0
	firstwin.body.q1 = 4
	global.seq++
	firstwin.body.file.Mark(global.seq)
	cut(&firstwin.tag, &firstwin.body, nil, false, true, "")
}

func TestUndoRedo(t *testing.T) {
	dir := t.TempDir()
	firstfilename := filepath.Join(dir, "firstfile")
	secondfilename := filepath.Join(dir, "secondfile")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tests := []struct {
		name string
		fn   func(t *testing.T, g *globals)
		want *dumpfile.Content
	}{
		{
			// Verify that test harness creates valid initial state.
			name: "testSetupOnly",
			fn:   testSetupOnly,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf | Look Edit ",
						},
						// Recall that when the contents match the on-disk state,
						// they are elided.
						Body: dumpfile.Text{},
					},
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
						Body: dumpfile.Text{},
					},
				},
			},
		},
		{
			// Verify that the mutateWithEdit helper successfully applies a mutation
			// to two buffers via an Edit X command.
			name: "mutateWithEdit",
			fn:   mutateWithEdit,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "This is a\nshort TEXT\nto try addressing\n",
							Q0:     16,
							Q1:     20,
						},
					},
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "A different TEXT\nWith other contents\nSo there!\n",
							Q0:     12,
							Q1:     16,
						},
					},
				},
			},
		},
		{
			// Having mutated both buffers, Undo one and Redo the other to get back
			// to the initial mutated state.
			name: "undoRedoBothMutations",
			fn:   undoRedoBothMutations,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "This is a\nshort TEXT\nto try addressing\n",
							Q0:     16,
							Q1:     20,
						},
					},
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "A different TEXT\nWith other contents\nSo there!\n",
							Q0:     12,
							Q1:     16,
						},
					},
				},
			},
		},
		{
			// Having mutated both buffers, further modify the first buffer via Cut
			// and then undo only the Cut action on the first buffer.
			name: "mutateBothOneUndo",
			fn:   mutateBothOneUndo,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo Redo Put | Look Edit ",
						},

						Body: dumpfile.Text{
							Buffer: "This is a\nshort TEXT\nto try addressing\n",
							Q0:     3,
							Q1:     10,
						},
					},
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "A different TEXT\nWith other contents\nSo there!\n",
							Q0:     12,
							Q1:     16,
						},
					},
				},
			},
		},
		{
			// Edit X mutate both buffers, further mutate the first via Cut. Undo on
			// second buffer. Show that the second buffer returns to the original
			// contents but that the first buffer's now divergent history is not
			// affected.
			name: "mutateBothOtherUndo",
			fn:   mutateBothOtherUndo,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "Thishort TEXT\nto try addressing\n",
							Q0:     3,
							Q1:     3,
						},
					},
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Redo | Look Edit ",
						},
						Body: dumpfile.Text{
							// Original content is elided.
							Buffer: "",
							Q0:     12,
							Q1:     16,
						},
					},
				},
			},
		},
		{
			// Edit X mutate both buffers and further mutate the first via Cut. Undo
			// the Edit X on the second buffer and hence diverge the undo history.
			// Undo Cut and Edit X on the first buffer to return to the same point in
			// the global undo history in first and second. Redo Edit X on the second
			// buffer also updates the first window.
			name: "mutateBranchedAndRejoined",
			fn:   mutateBranchedAndRejoined,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo Redo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "This is a\nshort TEXT\nto try addressing\n",
							Q0:     16,
							Q1:     20,
						},
					},
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "A different TEXT\nWith other contents\nSo there!\n",
							Q0:     12,
							Q1:     16,
						},
					},
				},
			},
		},
		{
			// Mutate, Put
			name: "mutatePut",
			fn:   mutatePut,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "",
							Q0:     3,
							Q1:     3,
						},
					},
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "",
							Q0:     0,
							Q1:     0,
						},
					},
				},
			},
		},
		{
			// Mutate, Put, Mutate again.
			// TODO(rjk): Undo sequence on top of this.
			name: "mutatePutMutate",
			fn:   mutatePutMutate,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Type:   dumpfile.Unsaved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf Undo Put | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "hort text\nto try addressing\n",
							Q0:     0,
							Q1:     0,
						},
					},
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "",
							Q0:     0,
							Q1:     0,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			FlexiblyMakeWindowScaffold(
				t,
				ScWin("firstfile"),
				ScBody("firstfile", contents),
				ScDir(dir, "firstfile"),
				ScWin("secondfile"),
				ScBody("secondfile", alt_contents),
				ScDir(dir, "secondfile"),
			)
			// Probably there are other issues here...
			t.Log("seq", global.seq)
			t.Log("seq, w0", global.row.col[0].w[0].body.file.Seq())
			t.Log("seq, w1", global.row.col[0].w[1].body.file.Seq())

			tc.fn(t, global)

			t.Log(*varfontflag, defaultVarFont)

			got, err := global.row.dump()
			if err != nil {
				t.Fatalf("dump failed: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

		})
	}
}
