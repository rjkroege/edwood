package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"9fans.net/go/plumb"
	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/runes"
)

func TestExpand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	dp, err := ioutil.TempDir("", "testexpand")
	if err != nil {
		t.Fatalf("can't make tempdir: %v", err)
	}
	defer os.RemoveAll(dp)
	modpath := filepath.Join(dp, "9fans.net/go@v0.0.0")
	if err := os.MkdirAll(modpath, 0777); err != nil {
		t.Fatalf("can't make modpath %s: %v", modpath, err)
	}

	tt := []struct {
		ok   bool
		sel1 int
		s    string
		inq  int
		q    string
		name string
		addr string
	}{
		{false, 0, "     ", 2, "", "", ""},
		{false, 0, "@@@@", 2, "", "", ""},
		{true, 0, "hello", 2, "hello", "", ""},
		{true, 5, "chicken", 2, "chick", "", ""},
		{true, 0, "hello.go", 2, "hello", "", ""},
		{true, 0, "hello.go:42", 2, "hello", "", ""},
		{true, 0, "世界.go:42", 2, "世界", "", ""},
		{true, 0, ":123", 2, ":123", "", "123"},
		{true, 0, ":/hello/", 2, ":/", "", "/hello/"},
		{true, 0, ":/世界/", 2, ":/", "", "/世界/"},
		{true, 0, "look_test.go", 2, "look_test.go", "look_test.go", ""},
		{true, 0, "look_test.go:42", 2, "look_test.go:42", "look_test.go", "42"},
		{true, 0, "look_test.go:42 ", 2, "look_test.go:42", "look_test.go", "42"},
		{true, 0, "look_test.go:42", 14, "look_test.go:42", "look_test.go", "42"},
		{true, 0, "<stdio.h>", 2, "stdio", "", ""},
		{true, 0, "/etc/hosts", 2, "/etc/hosts", "/etc/hosts", ""},
		{true, 0, "/etc/hosts:42", 2, "/etc/hosts:42", "/etc/hosts", "42"},
		{true, 0, modpath + ":531", 2, modpath + ":531", modpath, "531"},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			r := []rune(tc.s)
			text := &Text{
				file: file.MakeObservableEditableBuffer("", r),
				q0:   0,
				q1:   tc.sel1,
			}
			e, ok := expand(text, tc.inq, tc.inq)
			if ok != tc.ok {
				t.Fatalf("expand of %q returned %v; expected %v", tc.s, ok, tc.ok)
			}
			//t.Logf("expansion: %#v", e)
			q := string(r[e.q0:e.q1])
			if q != tc.q {
				t.Errorf("q0:q1 of %q is %q; expected %q", tc.s, q, tc.q)
			}
			if e.name != tc.name {
				t.Errorf("name of %q is %q; expected %q", tc.s, e.name, tc.name)
			}
			addr := ""
			if e.a0 < len(r) {
				addr = string(r[e.a0:e.a1])
			}
			if addr != tc.addr {
				t.Errorf("address of %q is %q; expected %q", tc.s, addr, tc.addr)
			}
		})
	}
}

func TestExpandJump(t *testing.T) {
	tt := []struct {
		kind TextKind
		jump bool
	}{
		{Tag, false},
		{Body, true},
	}

	for _, tc := range tt {
		text := &Text{
			file: file.MakeObservableEditableBuffer("", []rune("chicken")),
			q0:   0,
			q1:   5,
			what: tc.kind,
		}
		e, _ := expand(text, 2, 2)
		if e.jump != tc.jump {
			t.Errorf("expand of %v set jump to %v; expected %v", tc.kind, e.jump, tc.jump)
		}
	}
}

func TestLook3Message(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	global.wdir = cwd
	winDir := "/a/b/c"
	if runtime.GOOS == "windows" {
		winDir = `C:\a\b\c`
	}

	for _, tc := range []struct {
		name         string
		w            *Window
		dir          string
		text         string
		hasClickAttr bool
	}{
		{"NilWindow", nil, global.wdir, " hello.go ", true},
		{"Error", nil, global.wdir, "          ", true},
		{"InSelection", nil, global.wdir, " «hello.go» ", false},
		{
			"NonNilWindow",
			windowWithTag(winDir + string(filepath.Separator) + " Del Snarf | Look"),
			winDir, " hello.go ",
			true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := &plumb.Message{
				Src:  "acme",
				Dst:  "",
				Dir:  tc.dir,
				Type: "text",
				Data: []byte("hello.go"),
			}
			if tc.hasClickAttr {
				want.Attr = &plumb.Attribute{Name: "click", Value: "3"}
			}

			var text Text
			textSetSelection(&text, tc.text)
			text.w = tc.w
			got, err := look3Message(&text, 4, 4)
			if tc.name == "Error" {
				wantErr := "empty selection"
				if err.Error() != wantErr {
					t.Fatalf("got error %v; want %q", err, wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("plumb.Message mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func textSetSelection(t *Text, buf string) {
	b := []rune(buf)
	popRune := func(r rune) int {
		q := runes.IndexRune(b, r)
		if q < 0 {
			return 0
		}
		b = append(b[:q], b[q+1:]...)
		return q
	}

	t.q0 = popRune('«')
	t.q1 = popRune('»')
	t.file = file.MakeObservableEditableBuffer("", b)
}

func look3linenumber(t testing.TB, g *globals) {
	t.Helper()

	secondwin := g.row.col[0].w[1]

	// Probably need to lock here.
	global.row.lk.Lock()
	secondwin.Lock('M')

	// t.Logf("secondwin %q", secondwin.body.file.String())

	look3(&secondwin.body, 1, 1, false)

	secondwin.Unlock()
	global.row.lk.Unlock()
}

func BenchmarkLook3(t *testing.B) {
	dir := t.TempDir()
	firstfilename := filepath.Join(dir, "bigfile")
	secondfilename := filepath.Join(dir, "littlefile")
	nl := 4000
	tnl := nl - 1
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
			name: "jumpToNumberedLine",
			fn:   look3linenumber,
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
						Body: dumpfile.Text{
							Q0: (tnl - 1) * (1 + len("the quick brown fox")),
							Q1: tnl * (1 + len("the quick brown fox")),
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

					ScWin("littlefile"),
					ScDir(dir, "littlefile"),
					ScBody("littlefile", fmt.Sprintf("%s:%d\n", firstfilename, tnl)),
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
