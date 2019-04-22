package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/rjkroege/edwood/internal/dumpfile"
)

func TestRowLoad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tt := []struct {
		name     string
		filename string
	}{
		{"empty-two-cols", "testdata/empty-two-cols.dump"},
		{"example", "testdata/example.dump"},
		{"multi-line-tag", "testdata/multi-line-tag.dump"},
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			b, err := ioutil.ReadFile(tc.filename)
			if err != nil {
				t.Fatalf("ReadFile failed: %v", err)
			}
			b = bytes.Replace(b, []byte("/home/gopher/go/src/edwood"), []byte(cwd), -1)

			f, err := ioutil.TempFile("", "edwood-*.dump")
			if err != nil {
				t.Fatalf("failed to create temporary file: %v", err)
			}
			defer os.Remove(f.Name())
			_, err = f.Write(b)
			if err != nil {
				t.Fatalf("write failed: %v", err)
			}
			f.Close()

			a := startAcme(t, "-l", f.Name())
			defer a.Cleanup()

			dump, err := dumpfile.Load(f.Name())
			if err != nil {
				t.Fatalf("failed to load dump file %v: %v", tc, err)
			}
			checkDump(t, dump, a.fsys)
		})
	}
}

// checkDump checks Edwood's current state matches dump file.
// It checks that window's Tag, Font, Q0, Q1, and Body matches.
func checkDump(t *testing.T, dump *dumpfile.Content, fsys *client.Fsys) {
	wins, err := windows(fsys)
	if err != nil {
		t.Fatalf("failed to get list of windows: %v", err)
	}
	printCurrentTags := func() {
		for _, w := range wins {
			t.Logf("window id=%v tag=%q", w.id, w.tag)
		}
	}

	// Zerox-ed windows don't show up in index file.
	// Check number of orginal windows matches.
	norig := 0
	for _, w := range dump.Windows {
		if w.Type != dumpfile.Zerox {
			norig++
		}
	}
	if got, want := len(wins), norig; got != want {
		printCurrentTags()
		t.Fatalf("there are %v original windows; expected %v", got, want)
	}

	winByName := make(map[string]*winInfo)
	for i, w := range wins {
		winByName[w.name] = &wins[i]
	}

	for _, dw := range dump.Windows {
		// Zerox-ed windows don't show up in index file.
		if dw.Type == dumpfile.Zerox {
			continue
		}
		name := strings.SplitN(dw.Tag.Buffer, " ", 2)[0]

		w, ok := winByName[name]
		if !ok {
			printCurrentTags()
			t.Fatalf("could not find window with tag %q", dw.Tag)
		}

		// Unsaved window have "Undo" that doesn't get restored
		if dw.Type != dumpfile.Unsaved && w.tag != dw.Tag.Buffer {
			t.Errorf("tag is %q; expected %q", w.tag, dw.Tag)
		}

		if p := plan9FontPath(dw.Font); unscaledFontName(w.font) != p {
			t.Errorf("font for %q is %q; expected %q", w.name, unscaledFontName(w.font), p)
		}

		if w.q0 != dw.Body.Q0 || w.q1 != dw.Body.Q1 {
			t.Errorf("q0,q1 for %v is %v,%v; expected %v,%v", w.name, w.q0, w.q1, dw.Body.Q0, dw.Body.Q1)
		}

		if dw.Type == dumpfile.Unsaved && w.body != dw.Body.Buffer {
			t.Errorf("body for %q is %q; expected %q", w.name, w.body, dw.Body.Buffer)
		}
	}
}

// unscaledFontName converts the given fname to an unscaled
// representation without the leading scaling indicator.
func unscaledFontName(fname string) string {
	return strings.TrimLeftFunc(fname, func(r rune) bool {
		return  (r >= '0' && r <= '9') || r == '*' 
	})
}

func plan9FontPath(name string) string {
	const prefix = "/lib/font/bit"
	if strings.HasPrefix(name, prefix) {
		root := os.Getenv("PLAN9")
		if root == "" {
			root = "/usr/local/plan9"
		}
		return filepath.Join(root, "/font/", name[len(prefix):])
	}
	return name
}

type winInfo struct {
	id     int
	tag    string
	name   string
	font   string
	q0, q1 int
	body   string
}

func fsysReadFile(fsys *client.Fsys, filename string) ([]byte, error) {
	fid, err := fsys.Open(filename, plan9.OREAD)
	if err != nil {
		return nil, err
	}
	defer fid.Close()

	return ioutil.ReadAll(fid)
}

func fsysReadDot(fsys *client.Fsys, id int) (q0, q1 int, err error) {
	addr, err := fsys.Open(fmt.Sprintf("%v/addr", id), plan9.OREAD)
	if err != nil {
		return 0, 0, err
	}
	defer addr.Close()

	ctl, err := fsys.Open(fmt.Sprintf("%v/ctl", id), plan9.OWRITE)
	if err != nil {
		return 0, 0, err
	}
	defer ctl.Close()

	_, err = ctl.Write([]byte("addr=dot"))
	if err != nil {
		return 0, 0, err
	}

	b, err := ioutil.ReadAll(addr)
	if err != nil {
		return 0, 0, err
	}
	f := strings.Fields(string(b))
	q0, err = strconv.Atoi(f[0])
	if err != nil {
		return 0, 0, err
	}
	q1, err = strconv.Atoi(f[1])
	if err != nil {
		return 0, 0, err
	}
	return q0, q1, nil
}

func windows(fsys *client.Fsys) ([]winInfo, error) {
	index, err := fsysReadFile(fsys, "index")
	if err != nil {
		return nil, err
	}

	var info []winInfo
	for _, line := range strings.Split(string(index), "\n") {
		f := strings.Fields(line)
		if len(f) < 6 {
			continue
		}
		id, err := strconv.Atoi(f[0])
		if err != nil {
			return nil, err
		}

		b, err := fsysReadFile(fsys, fmt.Sprintf("%v/tag", id))
		if err != nil {
			return nil, err
		}
		tag := string(b)
		f = strings.SplitN(tag, " ", 2)
		name := f[0]

		b, err = fsysReadFile(fsys, fmt.Sprintf("%v/ctl", id))
		if err != nil {
			return nil, err
		}
		f = strings.Fields(string(b))
		font := f[6]

		q0, q1, err := fsysReadDot(fsys, id)
		if err != nil {
			return nil, err
		}

		b, err = fsysReadFile(fsys, fmt.Sprintf("%v/body", id))
		if err != nil {
			return nil, err
		}
		body := string(b)

		info = append(info, winInfo{
			id:   id,
			name: name,
			tag:  tag,
			font: font,
			q0:   q0,
			q1:   q1,
			body: body,
		})
	}
	return info, nil
}

func TestRowLookupWin(t *testing.T) {
	w42 := &Window{id: 42}
	row := &Row{
		col: []*Column{
			{
				w: []*Window{w42},
			},
		},
	}
	for _, tc := range []struct {
		id int
		w  *Window
	}{
		{42, w42},
		{100, nil},
	} {
		w := row.LookupWin(tc.id)
		if w != tc.w {
			t.Errorf("LookupWin returned window %p for id %v; expected %p", w, tc.id, tc.w)
		}
	}
}
