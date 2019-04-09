package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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
	err = os.Symlink("/non/existant/file", filepath.Join(dir, name))
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
