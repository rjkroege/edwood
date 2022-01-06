package main

import (
	"bytes"
	"fmt"
	"image"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/edwoodtest"
)

const gopherEdwoodDir = "/home/gopher/go/src/edwood"

func TestRowLoadFsys(t *testing.T) {
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
			b = bytes.Replace(b, []byte(gopherEdwoodDir), []byte(cwd), -1)

			f, err := ioutil.TempFile("", "edwood_test")
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
			checkDumpFsys(t, dump, a.fsys)
		})
	}
}

// checkDumpFsys checks Edwood's current state matches dump file.
// It checks that window's Tag, Font, Q0, Q1, and Body matches.
func checkDumpFsys(t *testing.T, dump *dumpfile.Content, fsys *client.Fsys) {
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
	// Check number of original windows matches.
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			filename := editDumpFileForTesting(t, tc.filename)
			defer os.Remove(filename)

			setGlobalsForLoadTesting()

			err := global.row.Load(nil, filename, true)
			if err != nil {
				t.Fatalf("Row.Load failed: %v", err)
			}
			want, err := dumpfile.Load(filename)
			if err != nil {
				t.Fatalf("failed to load dump file %v: %v", tc, err)
			}
			got, err := global.row.dump()
			if err != nil {
				t.Fatalf("dump failed: %v", err)
			}
			checkDump(t, got, want)
		})
	}
}

// checkDump checks Edwood's current state (got) matches loaded dump file content (want).
func checkDump(t *testing.T, got, want *dumpfile.Content) {
	t.Helper()
	// Ignore some mismatch. Positions may not match exactly.
	// Window tags may get "Put" added or "Undo" removed, and
	// because of the change in the tag, selection within the tag may not match.
	//
	// TODO(fhs): We should do better job of preserving exact positions, tags, etc.
	for i, c := range want.Columns {
		if math.Abs(got.Columns[i].Position-c.Position) < 1 {
			got.Columns[i].Position = c.Position
		}
	}
	for i, w := range want.Windows {
		g := got.Windows[i]
		t.Logf("[%d], %+v", i, g)
		if math.Abs(g.Position-w.Position) < 10 {
			g.Position = w.Position
		}
		const (
			put  = " Put "
			undo = " Undo "
		)

		if strings.Contains(g.Tag.Buffer, put) && !strings.Contains(w.Tag.Buffer, put) {
			g.Tag.Buffer = w.Tag.Buffer
		}
		if !strings.Contains(g.Tag.Buffer, undo) && strings.Contains(w.Tag.Buffer, undo) {
			g.Tag = w.Tag
		}

		// For directory listing, ignore selection changes in the tag since
		// we rewrite the directory name in the dump file during testing.
		name := ""
		if w := strings.Fields(g.Tag.Buffer); len(w) > 0 {
			name = w[0]
		}

		// setTag1 (executed inside of Load) will (correctly) adjust the text
		// selection to be within a valid range for the buffer.
		if nr := utf8.RuneCountInString(g.Tag.Buffer); w.Tag.Q0 > nr {
			w.Tag.Q0 = nr
		}
		if nr := utf8.RuneCountInString(g.Tag.Buffer); w.Tag.Q1 > nr {
			w.Tag.Q1 = nr
		}

		if n := len(name); n > 0 && name[n-1] == filepath.Separator { // is directory
			g.Tag.Q0 = w.Tag.Q0
			g.Tag.Q1 = w.Tag.Q1
		}
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dump mismatch (-want +got):\n%s", diff)
	}
}

func TestRowDumpError(t *testing.T) {
	var r Row

	err := r.Dump("")
	if err != nil {
		t.Errorf("Row.Dump returned error %v; want nil", err)
	}

	global.home = ""
	r = Row{
		col: make([]*Column, 2),
	}
	err = r.Dump("")
	want := "can't find file for dump: can't find home directory"
	if err == nil || err.Error() != want {
		t.Errorf("Row.Dump returned error %q; want %q", err, want)
	}
}

func TestRowLoadError(t *testing.T) {
	var r Row

	err := r.Load(nil, "/non-existent-file", true)
	want := "can't load dump file: open /non-existent-file:"
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("Row.Load returned error %q; want prefix %q", err, want)
	}

	global.home = ""
	err = r.Load(nil, "", true)
	want = "can't find file for load: can't find home directory"
	if err == nil || err.Error() != want {
		t.Errorf("Row.Load returned error %q; want %q", err, want)
	}
}

func TestDefaultDumpFile(t *testing.T) {
	global.home = ""
	_, err := defaultDumpFile()
	if err == nil {
		t.Errorf("defaultDumpFile returned nil error for unknown home directory")
	}

	global.home = "/home/gopher"
	got, err := defaultDumpFile()
	if err != nil {
		t.Fatalf("defaultDumpFile failed: %v", err)
	}
	got = filepath.ToSlash(got)
	want := "/home/gopher/edwood.dump"
	if got != want {
		t.Errorf("default dump file is %q; want %q", got, want)
	}
}

// unscaledFontName converts the given fname to an unscaled
// representation without the leading scaling indicator.
func unscaledFontName(fname string) string {
	return strings.TrimLeftFunc(fname, func(r rune) bool {
		return (r >= '0' && r <= '9') || r == '*'
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

// jsonEscapePath escapes blackslashes in Windows path.
func jsonEscapePath(s string) string {
	return strings.Replace(s, "\\", "\\\\", -1)
}

func setGlobalsForLoadTesting() {
	global.WinID = 0 // reset
	display := edwoodtest.NewDisplay()

	global.colbutton = edwoodtest.NewImage(display, "colbutton", image.Rectangle{})
	global.button = edwoodtest.NewImage(display, "button", image.Rectangle{})
	global.modbutton = edwoodtest.NewImage(display, "modbutton", image.Rectangle{})
	global.mouse = &draw.Mouse{}
	global.maxtab = 4

	global.row = Row{} // reset
	global.row.Init(display.ScreenImage().R(), display)
}

func replacePathsForTesting(t *testing.T, b []byte, isJSON bool) []byte {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	d := filepath.Join(cwd, "testdata")
	gd := gopherEdwoodDir + "/testdata"

	escape := jsonEscapePath
	if !isJSON {
		escape = func(s string) string { return s }
	}

	// TODO(rjk): Doesn't fix up the positions if the length of the path has changed.
	b = bytes.Replace(b, []byte(gd+"/"),
		[]byte(escape(d+string(filepath.Separator))), -1)
	b = bytes.Replace(b, []byte(gd),
		[]byte(escape(d)), -1)
	b = bytes.Replace(b, []byte(gopherEdwoodDir),
		[]byte(escape(cwd)), -1) // CurrentDir
	return b
}

func editDumpFileForTesting(t *testing.T, filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	b = replacePathsForTesting(t, b, true)

	f, err := ioutil.TempFile("", "edwood_test")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	_, err = f.Write(b)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	f.Close()
	return f.Name()
}
