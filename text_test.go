package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/internal/frame"
)

func emptyText() *Text {
	w := &Window{
		body: Text{
			file: &File{},
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
		out := string(text.file.b)
		if out != tc.out {
			t.Errorf("loaded text %q; expected %q", out, tc.out)
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
		out := string(text.file.b)
		if out != tc.out {
			t.Errorf("loaded text %q; expected %q", out, tc.out)
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

	text.file.name = ""
	wantErr = "empty directory name"
	_, err = text.Load(0, "/", true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("Load returned error %v; expected %v", err, wantErr)
	}
	_, err = text.LoadReader(0, "/", nil, true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("LoadReader returned error %v; expected %v", err, wantErr)
	}

	mtpt = "/mnt/acme"
	defer func() {
		mtpt = ""
	}()
	text.file.name = mtpt
	wantErr = "will not open self mount point /mnt/acme"
	_, err = text.Load(0, mtpt, true)
	if err == nil || err.Error() != wantErr {
		t.Fatalf("Load returned error %v; expected %v", err, wantErr)
	}
	_, err = text.LoadReader(0, mtpt, nil, true)
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
				file: &File{
					b: Buffer(r),
				},
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

	cwarn = nil
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
			t.Logf("warning: %v\n", string(warn.buf))
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
		file: &File{},
	}
	err := text.fill(&textFillMockFrame{})
	wantErr := "fill: negative slice length -100"
	if err == nil || err.Error() != wantErr {
		t.Errorf("got error %q; want %q", err, wantErr)
	}
}
