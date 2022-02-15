package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/util"
)

func TestCvttorunes(t *testing.T) {
	testCases := []struct {
		p     []byte
		n     int
		r     []rune
		nb    int
		nulls bool
	}{
		{[]byte("Hello world"), 11, []rune("Hello world"), 11, false},
		{[]byte("Hello \x00\x00world"), 13, []rune("Hello world"), 13, true},
		{[]byte("Hello 世界"), 6 + 3 + 3, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 1, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 2, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 1, []rune("Hello 世\uFFFD"), 6 + 3 + 1, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 2, []rune("Hello 世\uFFFD\uFFFD"), 6 + 3 + 2, false},
		{[]byte("\xe4\xb8\x96界 hello"), 3 + 3 + 6, []rune("世界 hello"), 3 + 3 + 6, false},
		{[]byte("\xb8\x96界 hello"), 2 + 3 + 6, []rune("\uFFFD\uFFFD界 hello"), 2 + 3 + 6, false},
		{[]byte("\x96界 hello"), 1 + 3 + 6, []rune("\uFFFD界 hello"), 1 + 3 + 6, false},
	}
	for _, tc := range testCases {
		r, nb, nulls := util.Cvttorunes(tc.p, tc.n)
		if !reflect.DeepEqual(r, tc.r) || nb != tc.nb || nulls != tc.nulls {
			t.Errorf("util.Cvttorunes of (%q, %v) returned %q, %v, %v; expected %q, %v, %v\n",
				tc.p, tc.n, r, nb, nulls, tc.r, tc.nb, tc.nulls)
		}
	}
}

// Given the complexity of errorwin1Name, one might wonder why we test
// this so comprehensively. :-)
func TestErrorwin1Name(t *testing.T) {
	tt := []struct {
		dir, name string
	}{
		{"", "+Errors"},
		{".", "+Errors"},
		{"/", "/+Errors"},
		{"/home/gopher", "/home/gopher/+Errors"},
		{"/home/gopher/", "/home/gopher/+Errors"},
		{"C:/Users/gopher", "C:/Users/gopher/+Errors"},
		{"C:/Users/gopher/", "C:/Users/gopher/+Errors"},
		{"C:/", "C:/+Errors"},
	}
	for _, tc := range tt {
		name := filepath.ToSlash(errorwin1Name(filepath.FromSlash(tc.dir)))
		if name != tc.name {
			t.Errorf("errorwin1Name(%q) is %q; expected %q", tc.dir, name, tc.name)
		}
	}
}

func TestQuote(t *testing.T) {
	var testCases = []struct {
		s, q string
	}{
		{"", "''"},
		{"Edwood", "Edwood"},
		{"Plan 9", "'Plan 9'"},
		{"Don't", "'Don''t'"},
		{"Don't worry!", "'Don''t worry!'"},
	}
	for _, tc := range testCases {
		q := quote(tc.s)
		if q != tc.q {
			t.Errorf("%q quoted is %q; expected %q\n", tc.s, q, tc.q)
		}
	}
}

func TestSkipbl(t *testing.T) {
	tt := []struct {
		s []rune
		q []rune
	}{
		{nil, nil},
		{[]rune(" \t\n"), nil},
		{[]rune(" \t\nabc"), []rune("abc")},
		{[]rune(" \t\n \t\nabc"), []rune("abc")},
		{[]rune(" \t\nabc \t\nabc"), []rune("abc \t\nabc")},
		{[]rune(" \t\nαβγ \t\nαβγ"), []rune("αβγ \t\nαβγ")},
	}
	for _, tc := range tt {
		q := skipbl(tc.s)
		if !reflect.DeepEqual(q, tc.q) {
			t.Errorf("skipbl(%v) returned %v; expected %v", tc.s, q, tc.q)
		}
	}
}

func logSomethingSmall(t *testing.T, g *globals, _ string) {
	t.Helper()
	err := warnError(nil, "SomethingSmall")

	if got, want := err.Error(), "SomethingSmall"; got != want {
		t.Errorf("didn't build correct error. got %v want %v", got, want)
	}
}

func logSomethingWithMntDir(t *testing.T, g *globals, dir string) {
	t.Helper()

	md := mnt.Add(dir, nil)
	warning(md, "I am an warning\n")
	warning(md, "I am a second warning\n")
}

func TestFlushWarnings(t *testing.T) {
	// TODO(rjk): Write me.
	dir := t.TempDir()
	firstfilename := filepath.Join(dir, "firstfile")
	secondfilename := filepath.Join(dir, "secondfile")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tests := []struct {
		name string
		fn   func(*testing.T, *globals, string)
		want *dumpfile.Content
	}{
		{
			name: "logSomethingSmall",
			fn:   logSomethingSmall,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Type: dumpfile.Unsaved,
						Tag: dumpfile.Text{
							Buffer: "+Errors Del Snarf | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "SomethingSmall\n",
							Q1:     15},
					},
				},
			},
		},
		{
			name: "logSomethingWithMntDir",
			fn:   logSomethingWithMntDir,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Type: dumpfile.Unsaved,
						Tag: dumpfile.Text{
							Buffer: filepath.Join(dir, "+Errors") + " Del Snarf | Look Edit ",
						},
						// TODO(rjk): Why isn't Q0 set? Where does this happen?
						// Somewhere, there's logic that fixes that.
						Body: dumpfile.Text{
							Buffer: "I am an warning\nI am a second warning\n",
							Q1:     38,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// TODO(rjk): Each test should use its own global.
			FlexiblyMakeWindowScaffold(
				t,
				ScWin("firstfile"),
				ScBody("firstfile", contents),
				ScDir(dir, "firstfile"),
				ScWin("secondfile"),
				ScBody("secondfile", alt_contents),
				ScDir(dir, "secondfile"),
			)

			tc.fn(t, global, dir)

			// Function under test.
			global.row.lk.Lock()
			flushwarnings()
			global.row.lk.Unlock()

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
