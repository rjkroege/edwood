package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
)

func changeFileName(t *testing.T, g *globals, ffn, _ string) {
	t.Helper()

	g.row.display.WriteSnarf([]byte("suffix"))

	// Mutate
	fwt := &g.row.col[0].w[0].tag

	// TODO(rjk): WRONG WITH THE UNICODE
	fwt.q0 = len(ffn)
	fwt.q1 = len(ffn)

	t.Log("before", fwt.DebugString())

	paste(fwt, fwt, nil, true, false, "")

	t.Log("after", fwt.DebugString())

}

func undoFileNameChange(t *testing.T, g *globals, ffn, _ string) {
	t.Helper()

	changeFileName(t, g, ffn, "")

	// TODO(rjk): Move the text position
	// to the end?

	firstwin := g.row.col[0].w[0]
	undo(&firstwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")
}

func undoFileNameChangedMultipleEdit(t *testing.T, g *globals, ffn, _ string) {
	t.Helper()

	// Change name of first.
	changeFileName(t, g, ffn, "")

	// Mutate both body values.
	mutateWithEdit(t, g)

	firstwin := g.row.col[0].w[0]
	secondwin := g.row.col[0].w[1]

	firstwin.body.q0 += 8
	firstwin.body.q1 += 10

	// Run undo from second window undoes Edit X on both windows.
	undo(&secondwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")
}

func undoSecondMutateFileNameChange(t *testing.T, g *globals, ffn, _ string) {
	t.Helper()

	// Mutate both body values.
	mutateWithEdit(t, g)

	// Change name of first.
	changeFileName(t, g, ffn, "")

	secondwin := g.row.col[0].w[1]

	// TODO(rjk): Move the text position in secondwin
	secondwin.body.q0 += 4
	secondwin.body.q1 += 5

	// Undo on second win only affects the Edit X on second window, filename is still changed.
	undo(&secondwin.tag, nil, nil, true /* this is an undo */, false /* ignored */, "")
}

func TestFilenameChangeUndo(t *testing.T) {
	dir := t.TempDir()
	firstfilename := filepath.Join(dir, "firstfile")
	secondfilename := filepath.Join(dir, "secondfile")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tests := []struct {
		name    string
		fn      func(t *testing.T, g *globals, ffn, sfn string)
		passing bool
		want    *dumpfile.Content
	}{
		{
			// Verify that we can edit a file name.
			name:    "changeFileName",
			fn:      changeFileName,
			passing: true,
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
							Buffer: firstfilename + "suffix" + " Del Snarf Undo Put | Look Edit ",
							Q0:     len(firstfilename),
							Q1:     len(firstfilename) + len("suffix"),
						},
						// Recall that when the contents match the on-disk state, they are
						// elided. Here, while we've not actually changed the body, we've altered
						// the tag to be a different filename so the contents no longer match the
						// on-disk state.
						Body: dumpfile.Text{
							Buffer: "This is a\nshort text\nto try addressing\n",
						},
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
			// Verify that we can edit a file name and then undo the change.
			name: "undoFileNameChange",
			fn:   undoFileNameChange,
			// Currently failing. Requires some non-trivial coding adjustments.
			passing: true,
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
							Buffer: firstfilename + " Del Snarf Redo | Look Edit ",
							Q0:     len(firstfilename),
							Q1:     len(firstfilename),
						},
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
			// Verify that multiple file mutation will Undo and leave file name
			// change unaffected.
			name:    "undoFileNameChangedMultipleEdit",
			fn:      undoFileNameChangedMultipleEdit,
			passing: true,
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
							Buffer: firstfilename + "suffix" + " Del Snarf Undo Redo Put | Look Edit ",
							Q0:     len(firstfilename),
							Q1:     len(firstfilename) + len("suffix"),
						},
						Body: dumpfile.Text{
							Buffer: "This is a\nshort text\nto try addressing\n",
							Q0:     16,
							Q1:     20,
						},
					},
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Redo | Look Edit ",
						},
						Body: dumpfile.Text{
							Q0: 12,
							Q1: 16,
						},
					},
				},
			},
		},
		{
			// Mutate both with Edit X, change the name of the first, undo on the
			// second. Verify that we can diverge the undo history featuring a file
			// name change and just undo the Edit X on second.
			name:    "undoSecondMutateFileNameChange",
			fn:      undoSecondMutateFileNameChange,
			passing: true,
			// Q1 is not correctly updated.
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
							Buffer: firstfilename + "suffix" + " Del Snarf Undo Put | Look Edit ",
							Q0:     len(firstfilename),
							Q1:     len(firstfilename) + len("suffix"),
						},
						Body: dumpfile.Text{
							Buffer: "This is a\nshort TEXT\nto try addressing\n",
							Q0:     16,
							Q1:     20,
						},
					},
					{
						Type:   dumpfile.Saved,
						Column: 0,
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf Redo | Look Edit ",
						},
						Body: dumpfile.Text{
							Q0: 12,
							Q1: 16,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Skip known failures.
			if !tc.passing {
				return
			}

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

			tc.fn(t, global, firstfilename, secondfilename)

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
