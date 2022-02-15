package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
)

func emptyText() *Text {
	w := &Window{
		body: Text{
			file: file.MakeObservableEditableBuffer("", nil),
		},
	}
	t := &w.body
	t.w = w
	return t
}

func TestLoadReader(t *testing.T) {
	for _, tc := range []struct {
		in, out string
	}{
		{"temporary file's content\n", "temporary file's content\n"},
		{"temporary file's \x00content\n", "temporary file's content\n"},
	} {
		text := emptyText()
		_, err := text.LoadReader(0, "/home/gopher/test/main.go", strings.NewReader(tc.in), true)
		if err != nil {
			t.Fatalf("LoadReader failed: %v", err)
		}
		out := text.file.String()
		if out != tc.out {
			t.Errorf("loaded editor %q; expected %q", out, tc.out)
		}
	}
}

func TestLoad(t *testing.T) {
	dir, err := ioutil.TempDir("", "edwood.test")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	for _, tc := range []struct {
		in, out string
	}{
		{"temporary file's content\n", "temporary file's content\n"},
		{"temporary file's \x00content\n", "temporary file's content\n"},
	} {
		text := emptyText()
		filename := filepath.Join(dir, "tmpfile")
		if err = ioutil.WriteFile(filename, []byte(tc.in), 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		_, err = text.Load(0, filename, true)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		out := text.file.String()
		if out != tc.out {
			t.Errorf("loaded editor %q; expected %q", out, tc.out)
		}
	}
}

func TestLoadError(t *testing.T) {
	text := emptyText()
	wantErr := "can't open /non-existent-filename:"
	_, err := text.Load(0, "/non-existent-filename", true)
	if err == nil || !strings.HasPrefix(err.Error(), wantErr) {
		t.Fatalf("Load returned error %v; expected %v", err, wantErr)
	}

	text = emptyText()
	text.file.SetDir(true)

	text.file.SetName("")
	wantErr = "empty directory name"
	_, err = text.Load(0, "/", true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("Load returned error %v; expected %v", err, wantErr)
	}
	_, err = text.LoadReader(0, "/", nil, true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("LoadReader returned error %v; expected %v", err, wantErr)
	}

	*mtpt = "/mnt/acme"
	defer func() {
		*mtpt = ""
	}()
	text.file.SetName(*mtpt)
	wantErr = "will not open self mount point /mnt/acme"
	_, err = text.Load(0, *mtpt, true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("Load returned error %v; expected %v", err, wantErr)
	}
	_, err = text.LoadReader(0, *mtpt, nil, true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("LoadReader returned error %v; expected %v", err, wantErr)
	}
}

func TestClickHTMLMatch(t *testing.T) {
	tt := []struct {
		s      string
		inq0   int
		q0, q1 int
		ok     bool
	}{
		{"hello world", 0, 0, 0, false},
		{"<b>hello world", 3, 0, 0, false},
		{"<b>hello world</b>", 4, 0, 0, false},
		{"<b>hello world</b>", 13, 0, 0, false},
		{"<b>hello world</b>", 3, 3, 14, true},
		{"<b>hello world</b>", 14, 3, 14, true},
		{"<title>hello 世界</title>", 7, 7, 15, true},
		{"<p>hello <br /><b>world</b>!</p>", 3, 3, 28, true},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			r := []rune(tc.s)
			text := &Text{
				file: file.MakeObservableEditableBuffer("", r),
			}
			q0, q1, ok := text.ClickHTMLMatch(tc.inq0)
			switch {
			case ok != tc.ok:
				t.Errorf("ClickHTMLMatch of %q at position %v returned %v; expected %v\n",
					tc.s, tc.inq0, ok, tc.ok)

			case q0 > q1 || q0 < 0 || q1 >= len(r):
				t.Errorf("ClickHTMLMatch of %q at position %v is %v:%v; expected %v:%v\n",
					tc.s, tc.inq0, q0, q1, tc.q0, tc.q1)

			case q0 != tc.q0 || q1 != tc.q1:
				t.Errorf("ClickHTMLMatch of %q at position %v is %q; expected %q\n",
					tc.s, tc.inq0, r[q0:q1], r[tc.q0:tc.q1])
			}
		})
	}
}

func TestTextKindString(t *testing.T) {
	tt := []struct {
		tk TextKind
		s  string
	}{
		{Body, "Body"},
		{Columntag, "Columntag"},
		{Rowtag, "Rowtag"},
		{Tag, "Tag"},
		{100, "TextKind(100)"},
	}
	for _, tc := range tt {
		s := tc.tk.String()
		if s != tc.s {
			t.Errorf("string representation of TextKind(%d) is %s; expected %s", int(tc.tk), s, tc.s)
		}
	}
}

func TestGetDirNames(t *testing.T) {
	dir, err := ioutil.TempDir("", "edwood")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer os.RemoveAll(dir)

	var want []string

	// add a directory file
	name := "a_dir" + string(filepath.Separator)
	err = os.Mkdir(filepath.Join(dir, name), 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	want = append(want, name)

	// add a broken symlink
	name = "broken-link"
	err = os.Symlink("/non/existent/file", filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}
	want = append(want, name)

	// add a regular file
	name = "example.txt"
	err = ioutil.WriteFile(filepath.Join(dir, name), []byte("hello\n"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	want = append(want, name)

	global.cwarn = nil
	warnings = nil
	defer func() {
		warnings = nil
	}()

	f, err := os.Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer f.Close()

	got, err := getDirNames(f)
	if err != nil {
		t.Fatalf("getDirNames failed: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got entries %v; expected %v", got, want)
	}
	if len(warnings) > 0 {
		for _, warn := range warnings {
			t.Logf("warning: %v\n", warn.buf.String())
		}
		t.Errorf("getDirnames generated %v warning(s)", len(warnings))
	}
}

func TestGetDirNamesNil(t *testing.T) {
	_, err := getDirNames(nil)
	if err == nil {
		t.Errorf("getDirNames was successful on nil File")
	}
}

type textFillMockFrame struct {
	*MockFrame
}

func (fr *textFillMockFrame) GetFrameFillStatus() frame.FrameFillStatus {
	return frame.FrameFillStatus{
		Nchars:         100,
		Nlines:         0,
		Maxlines:       0,
		MaxPixelHeight: 0,
	}
}

func TestTextFill(t *testing.T) {
	text := &Text{
		file: file.MakeObservableEditableBuffer("", []rune{}),
	}
	err := text.fill(&textFillMockFrame{})
	wantErr := "fill: negative slice length -100"
	if err == nil || err.Error() != wantErr {
		t.Errorf("got error %q; want %q", err, wantErr)
	}
}

func TestTextDirName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tt := []struct {
		name          string
		w             *Window
		filename, dir string
	}{
		{"NilWindow,file=''", nil, "", "."},
		{"NilWindow", nil, "abc", "abc"},
		{"EmptyTag,file=''", windowWithTag(""), "", "."},
		{"EmptyTag", windowWithTag(""), "abc", "abc"},
		{"Dot,file=''", windowWithTag("./ Del Snarf | Look "), "", "."},
		{"Dot", windowWithTag("./ Del Snarf | Look "), "d.go", "d.go"},
		{"NoSlash,file=''", windowWithTag("abc Del Snarf | Look "), "", "."},
		{"NoSlash", windowWithTag("abc Del Snarf | Look "), "d.go", "d.go"},
		{"AbsDir,file=''", windowWithTag("/a/b/c/ Del Snarf | Look "), "", "/a/b/c"},
		{"AbsDir", windowWithTag("/a/b/c/ Del Snarf | Look "), "d.go", "/a/b/c/d.go"},
		{"RelativeDir,file=''", windowWithTag("a/b/c/ Del Snarf | Look "), "", "a/b/c"},
		{"RelativeDir", windowWithTag("a/b/c/ Del Snarf | Look "), "d.go", "a/b/c/d.go"},
		{"AbsFile,file=''", windowWithTag("/a/b/c/d.go Del Snarf | Look "), "", "/a/b/c"},
		{"AbsFile", windowWithTag("/a/b/c/d.go Del Snarf | Look "), "e.go", "/a/b/c/e.go"},
		{"RelativeFile,file=''", windowWithTag("a/b/c/d.go Del Snarf | Look "), "", "a/b/c"},
		{"RelativeFile", windowWithTag("a/b/c/d.go Del Snarf | Look "), "e.go", "a/b/c/e.go"},
		{"IgnoreTag", windowWithTag("/a/b/c/d.go Del Snarf | Look "), "/x/e.go", "/x/e.go"},
		{"FileWithSpace", windowWithTag("/a/b c/d.go Del Snarf | Look "), "/a/b c/d.go", "/a/b c/d.go"},
		{"DirWithSpace", windowWithTag("/a/b c/ Del Snarf | Look "), "", "/a/b c"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			text := Text{
				w: tc.w,
			}
			dir := text.DirName(tc.filename)
			if !reflect.DeepEqual(dir, tc.dir) {
				t.Errorf("dirname of %q is %q; want %q", tc.filename, dir, tc.dir)
			}
		})
	}
}

func TestTextAbsDirName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	for _, tc := range []struct {
		name          string
		w             *Window
		filename, dir string
	}{
		{"AbsDir", windowWithTag("/a/b/c/ Del Snarf | Look "), "d.go", "/a/b/c/d.go"},
		{"RelativeDir", windowWithTag("a/b/c/ Del Snarf | Look "), "d.go", path.Join(cwd, "/a/b/c/d.go")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := Text{
				w: tc.w,
			}
			dir := text.AbsDirName(tc.filename)
			if !reflect.DeepEqual(dir, tc.dir) {
				t.Errorf("dirname of %q is %q; want %q", tc.filename, dir, tc.dir)
			}
		})
	}
}

func windowWithTag(tag string) *Window {
	ru := []rune(tag)
	return &Window{
		tag: Text{
			file: file.MakeObservableEditableBuffer("", ru),
		},
		tagfilenameend: len(parsetaghelper(tag)),
	}
}

func TestBackNL(t *testing.T) {
	tt := []struct {
		buf  string // Text file buffer
		p, n int    // Input position and number of lines to back up
		q    int    // Returned position
	}{
		{"", 0, 0, 0},
		{"", 0, 1, 0},
		{"", 0, 2, 0},
		{"01234\n", 3, 0, 0},
		{"01234\n", 3, 1, 0},
		{"01234\n", 3, 2, 0},
		{"01234\n6789\nabcd\n", 13, 0, 11},
		{"01234\n6789\nabcd\n", 13, 1, 11},
		{"01234\n6789\nabcd\n", 13, 2, 6},
		{"01234\n6789\nabcd\n", 13, 3, 0},
		{"\n1234\n6789\nabcd\n", 13, 3, 1},
		{"\n1234\n6789\nabcd\n", 13, 4, 0},
		{"\n1234\n6789\nabcd\n", 13, 5, 0},
	}

	for _, tc := range tt {
		text := &Text{
			file: file.MakeObservableEditableBuffer("", []rune(tc.buf)),
		}
		q := text.BackNL(tc.p, tc.n)
		if got, want := q, tc.q; got != want {
			t.Errorf("BackNL(%v, %v) for %q is %v; want %v",
				tc.p, tc.n, tc.buf, got, want)
		}
	}
}

func TestTextBsInsert(t *testing.T) {
	tt := []struct {
		name          string   // Test name
		what          TextKind // Body, Tag, etc.
		q0, q         int      // Input and returned position
		buf           string   // Initial text buffer
		inbuf, outbuf []rune   // Inserted and modified text buffer
		nr            int      // Returned number of runes
	}{
		{"Tag", Tag, 2, 2, "abc", []rune("xy\bz"), []rune("abxy\bzc"), 4},
		{"NoBS", Body, 2, 2, "abc", []rune("xyz"), []rune("abxyzc"), 3},
		{"BSInMiddle", Body, 2, 2, "abc", []rune("xy\bz"), []rune("abxzc"), 2},
		{"BSAtStart", Body, 2, 1, "abc", []rune("\bxyz"), []rune("axyzc"), 3},
		{"TwoBS", Body, 2, 0, "abc", []rune("\b\b"), []rune("c"), 0},
		{"TooManyBS", Body, 2, 0, "abc", []rune("\b\b\b\b\b"), []rune("c"), 0},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			text := &Text{
				what: tc.what,
				file: file.MakeObservableEditableBuffer("", []rune(tc.buf)),
			}
			q, nr := text.BsInsert(tc.q0, []rune(tc.inbuf), true)
			if nr != tc.nr {
				t.Errorf("nr = %v; want %v", nr, tc.nr)
			}
			if q != tc.q {
				t.Errorf("q = %v; want %v", q, tc.q)
			}
			if got, want := []rune(text.file.String()), tc.outbuf; !cmp.Equal(got, want) {
				t.Errorf("editor.file.b = %q; want %q", got, want)
			}
		})
	}
}

func checkTabexpand(t *testing.T, getText func(tabexpand bool, tabstop int) *Text) {
	for _, tc := range []struct {
		tabexpand bool
		tabstop   int
		input     string
		want      string
	}{
		{false, 4, "\t|", "\t|"},
		{true, 4, "\t|", "    |"},
		{true, 2, "\t|", "  |"},
	} {
		text := getText(tc.tabexpand, tc.tabstop)

		for _, r := range tc.input {
			text.Type(r)
		}
		text.file.Commit()

		gr := make([]rune, text.file.Nr())
		text.file.Read(0, gr[:text.file.Nr()])

		if got := string(gr); got != tc.want {
			t.Errorf("loaded editor %q; expected %q", got, tc.want)
		}
	}
}

func makeTestTextTabexpandState() *Window {
	MakeWindowScaffold(&dumpfile.Content{
		Columns: []dumpfile.Column{
			{},
		},
		Windows: []*dumpfile.Window{
			{
				Column: 0,
				Tag: dumpfile.Text{
					Buffer: "",
				},
				Body: dumpfile.Text{
					Buffer: "",
					Q0:     0,
					Q1:     0,
				},
			},
		},
	})
	return global.row.col[0].w[0]
}

func TestTextTypeTabInBody(t *testing.T) {
	checkTabexpand(t, func(tabexpand bool, tabstop int) *Text {

		w := makeTestTextTabexpandState()
		text := &w.body
		text.tabexpand = tabexpand
		text.tabstop = tabstop

		return text
	})
}

func TestTextTypeTabInTag(t *testing.T) {
	checkTabexpand(t, func(tabexpand bool, tabstop int) *Text {
		w := makeTestTextTabexpandState()
		text := &w.tag
		text.tabexpand = tabexpand
		text.tabstop = tabstop

		return text
	})
}
