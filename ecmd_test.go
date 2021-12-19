package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
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

func edit_sPerf(t testing.TB, g *globals) {
	t.Helper()

	firstwin := g.row.col[0].w[0]

	// Lock discipline?
	// TODO(rjk): figure out how to change this with less global dependency.
	global.row.lk.Lock()
	firstwin.Lock('M')
	global.seq++

	action := ", s/fox/gopher/g"
	editcmd(&firstwin.body, []rune(action))
	firstwin.Unlock()
	global.row.lk.Unlock()
}

func edit_xPerf(t testing.TB, g *globals) {
	t.Helper()

	firstwin := g.row.col[0].w[0]

	// Lock discipline?
	// TODO(rjk): figure out how to change this with less global dependency.
	global.row.lk.Lock()
	firstwin.Lock('M')
	global.seq++

	action := ",x/fox/ c/gopher/"
	editcmd(&firstwin.body, []rune(action))
	firstwin.Unlock()
	global.row.lk.Unlock()
}

func BenchmarkLargeEditTargets10(t *testing.B)    { benchmarkLargeEditTargetsImpl(t, 10) }
func BenchmarkLargeEditTargets109(t *testing.B)   { benchmarkLargeEditTargetsImpl(t, 100) }
func BenchmarkLargeEditTargets1000(t *testing.B)  { benchmarkLargeEditTargetsImpl(t, 1000) }
func BenchmarkLargeEditTargets10000(t *testing.B) { benchmarkLargeEditTargetsImpl(t, 10000) }

func benchmarkLargeEditTargetsImpl(t *testing.B, nl int) {
	dir := t.TempDir()
	firstfilename := filepath.Join(dir, "bigfile")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tests := []struct {
		name string
		fn   func(t testing.TB, g *globals)
		want *dumpfile.Content
	}{
		{
			name: "edit_xPerf",
			fn:   edit_xPerf,
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
							Buffer: Repeating(nl, "the quick brown gopher"),
							Q0:     nl*(len("the quick brown gopher")+1) - 7,
							Q1:     nl*(len("the quick brown gopher")+1) - 1,
						},
					},
				},
			},
		},
		{
			name: "edit_sPerf",
			fn:   edit_sPerf,
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
							Buffer: Repeating(nl, "the quick brown gopher"),
							Q0:     0,
							Q1:     nl * (len("the quick brown gopher") + 1),
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(b *testing.B) {
			b.StopTimer()

			for i := 0; i < b.N; i++ {

				FlexiblyMakeWindowScaffold(
					b,
					ScWin("bigfile"),
					ScDir(dir, "bigfile"),
					ScBody("bigfile", Repeating(nl, "the quick brown fox")),
				)

				b.StartTimer()
				tc.fn(b, global)
				b.StopTimer()

				got, err := global.row.dump()
				if err != nil {
					b.Fatalf("dump failed: %v", err)
				}

				if diff := cmp.Diff(tc.want, got); diff != "" {
					b.Errorf("dump mismatch (-want +got):\n%s", diff)
				}

			}
		})
	}
}
