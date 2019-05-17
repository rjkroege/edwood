package main

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/internal/draw"
	"github.com/rjkroege/edwood/internal/frame"
	"github.com/sanity-io/litter"
)

type teststimulus struct {
	dot           Range
	filename      string
	expr          string
	expected      string
	expectedwarns []string
}

func TestEdit(t *testing.T) {
	testtab := []teststimulus{

		// 0
		{Range{0, 0}, "test", "a/junk", "junkThis is a\nshort text\nto try addressing\n", []string{}},
		{Range{7, 12}, "test", "a/junk", "This is a\nshjunkort text\nto try addressing\n", []string{}},
		{Range{0, 0}, "test", "/This/a/junk", "Thisjunk is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 0}, "test", "/^/a/junk", "This is a\njunkshort text\nto try addressing\n", []string{}},
		{Range{0, 0}, "test", "/$/a/junk", "This is ajunk\nshort text\nto try addressing\n", []string{}},

		// 4
		{Range{0, 0}, "test", "i/junk", "junkThis is a\nshort text\nto try addressing\n", []string{}},
		{Range{2, 6}, "test", "i/junk", "Thjunkis is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 0}, "test", "/text/i/junk", "This is a\nshort junktext\nto try addressing\n", []string{}},

		// Don't know how to automate testing of 'b'

		// c
		// 7
		{Range{0, 0}, "test", "c/junk", "junkThis is a\nshort text\nto try addressing\n", []string{}},
		{Range{2, 6}, "test", "c/junk", "Thjunks a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 0}, "test", "/text/c/junk", "This is a\nshort junk\nto try addressing\n", []string{}},

		// d
		// 10
		{Range{0, 0}, "test", "d", "This is a\nshort text\nto try addressing\n", []string{}},
		{Range{2, 6}, "test", "d", "Ths a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 0}, "test", "/text/d", "This is a\nshort \nto try addressing\n", []string{}},

		// f - Don't know how to test f

		// g/v
		{Range{0, 0}, "test", "g/This/d", "This is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 12}, "test", "g/This/d", "ort text\nto try addressing\n", []string{}},
		{Range{0, 3}, "test", "v/This/d", "s is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 12}, "test", "v/This/d", "This is a\nshort text\nto try addressing\n", []string{}},

		// m/t
		// 17
		{Range{0, 4}, "test", "m/try", " is a\nshort text\nto tryThis addressing\n", []string{}},
		{Range{0, 3}, "test", "t/try", "This is a\nshort text\nto tryThi addressing\n", []string{}},
		{Range{1, 3}, "test", "m0", "hiTs is a\nshort text\nto try addressing\n", []string{}},
		{Range{4, 8}, "test", "m.", "This is a\nshort text\nto try addressing\n", []string{}},

		// s
		{Range{0, len(contents)}, "test", "s/short/long/", "This is a\nlong text\nto try addressing\n", []string{}},
		{Range{0, len(contents)}, "test", "s/(i.)/!\\1!/g", "Th!is! !is! a\nshort text\nto try address!in!g\n", []string{}},

		// =
		{Range{1, 3}, "test", "=", "This is a\nshort text\nto try addressing\n", []string{"test:1\n"}},
		{Range{1, 3}, "test", "=+", "This is a\nshort text\nto try addressing\n", []string{"test:1+#1\n"}},
		{Range{1, 3}, "test", "=#", "This is a\nshort text\nto try addressing\n", []string{"test:#1,#3\n"}},

		// p
		{Range{0, 4}, "test", "p", "This is a\nshort text\nto try addressing\n", []string{"This"}},

		// x
		{Range{0, 4}, "test", ",x/$/ a/@/", "This is a@\nshort text@\nto try addressing@\n@", []string{}},
		{Range{0, 4}, "test", ",x a/@/", "This is a@\nshort text@\nto try addressing@\n", []string{}},
	}

	buf := make([]rune, 8192)

	for i, test := range testtab {
		warnings = []*Warning{}
		w := makeSkeletonWindowModel(&test)
		editcmd(&w.body, []rune(test.expr))

		n, _ := w.body.ReadB(0, buf[:])
		if string(buf[:n]) != test.expected {
			t.Errorf("test %d: File.b contents expected \n%v\nbut got \n%v\n", i, test.expected, string(buf[:n]))
		}

		if got, want := len(warnings), len(test.expectedwarns); got != want {
			t.Errorf("text %d: expected %d warnings but got %d warnings", i, want, got)
			break
		}

		for j, tw := range test.expectedwarns {
			n, _ := warnings[j].buf.Read(0, buf[:])
			if string(buf[:n]) != tw {
				t.Errorf("test %d: Warning %d contents expected \n%#v\nbut got \n%#v\n", i, j, tw, string(buf[:n]))
			}

		}

	}
}

func makeSkeletonWindowModel(test *teststimulus) *Window {
	w := NewWindow().initHeadless(nil)
	w.body.fr = &MockFrame{}
	w.tag.fr = &MockFrame{}
	w.body.Insert(0, []rune("This is a\nshort text\nto try addressing\n"), true)

	// Set up Undo to make sure that we see undoable results.
	// By default, post-load, file.seq, file.putse = 0, 0.
	seq = 1

	w.body.SetQ0(test.dot.q0)
	w.body.SetQ1(test.dot.q1)
	w.body.file.SetName(test.filename)

	// Construct the global window machinery.
	row = Row{
		col: []*Column{
			{
				w: []*Window{
					w,
				},
			},
		},
	}
	w.col = row.col[0]
	return w
}

const contents = "This is a\nshort text\nto try addressing\n"

func TestEditCmdWithFile(t *testing.T) {
	// Make a temporary file.
	tfd, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatalf("can't make a temp file %s because %v\n", tfd.Name(), err)
	}
	defer os.Remove(tfd.Name()) // clean up
	if _, err := tfd.WriteString(contents); err != nil {
		t.Fatalf("can't write tmpfile %v", err)
	}
	if err := tfd.Close(); err != nil {
		t.Fatalf("can't close tmpfile %v", err)
	}

	testtab := []teststimulus{
		// e
		{Range{0, 0}, tfd.Name(), "e " + tfd.Name(), contents, []string{}},

		// r
		{Range{0, 0}, tfd.Name(), "r " + tfd.Name(), contents + contents, []string{}},
		{Range{0, len(contents)}, tfd.Name(), "r " + tfd.Name(), contents, []string{}},

		// a (for confirmation of test rationality)
		{Range{0, 0}, tfd.Name(), "a/junk", "junkThis is a\nshort text\nto try addressing\n", []string{}},
	}

	filedirtystates := []struct {
		Dirty            bool
		SaveableAndDirty bool
	}{
		{false, false},
		{true, true},
		{false, false},
		{true, true},
	}

	buf := make([]rune, 8192)

	for i, test := range testtab {
		warnings = []*Warning{}
		w := makeSkeletonWindowModel(&test)

		editcmd(&w.body, []rune(test.expr))

		n, _ := w.body.ReadB(0, buf[:])
		if string(buf[:n]) != test.expected {
			t.Errorf("test %d: TestAppend expected \n%v\nbut got \n%v\n", i, test.expected, string(buf[:n]))
		}

		litter.Config.HidePrivateFields = false

		// For e identical.
		if got, want := w.body.file.Dirty(), filedirtystates[i].Dirty; got != want {
			t.Errorf("test %d: File bad Dirty state. Got %v, want %v: dump %s", i, got, want /* litter.Sdump(w.body.file) */, "")
		}
		if got, want := w.body.file.SaveableAndDirty(), filedirtystates[i].SaveableAndDirty; got != want {
			t.Errorf("test %d: File bad SaveableAndDirty state. Got %v, want %v: dump %s", i, got, want /* litter.Sdump(w.body.file) */, "")
		}

		if got, want := len(warnings), len(test.expectedwarns); got != want {
			t.Errorf("test %d: expected %d warnings but got %d warnings", i, want, got)
			break
		}

		for j, tw := range test.expectedwarns {
			n, _ := warnings[j].buf.Read(0, buf[:])
			if string(buf[:n]) != tw {
				t.Errorf("test %d: Warning %d contents expected \n%#v\nbut got \n%#v\n", i, j, tw, string(buf[:n]))
			}
		}
	}
}

func TestParsecmd(t *testing.T) {
	tt := []struct {
		input []rune
		cmd   *Cmd
		err   error
	}{
		{[]rune("\n"), &Cmd{cmdc: '\n'}, nil},
		{[]rune("a\n"), &Cmd{cmdc: 'a', text: "\n"}, nil},
		{[]rune("a\nabc"), &Cmd{cmdc: 'a', text: "abc\n"}, nil},
		{[]rune("a\nabc\n.\n"), &Cmd{cmdc: 'a', text: "abc\n"}, nil},
		{[]rune("a/abc/\n"), &Cmd{cmdc: 'a', text: "abc"}, nil},
		{[]rune("a/abc/\n"), &Cmd{cmdc: 'a', text: "abc"}, nil},
		{[]rune(`a/a\bc/` + "\n"), &Cmd{cmdc: 'a', text: `a\bc`}, nil},
		{[]rune(`a/a\nc/` + "\n"), &Cmd{cmdc: 'a', text: "a\nc"}, nil},
		{[]rune("a/ab\\\nc/\n"), &Cmd{cmdc: 'a', text: `ab\`}, nil},
		{[]rune("a/ab\\"), nil, errBadRHS},
		{[]rune(`a\abc\` + "\n"), nil, badDelimiterError('\\')},
		{[]rune("x/abc/\n"), &Cmd{re: "abc", cmd: &Cmd{cmdc: 'p'}, cmdc: 'x'}, nil},
		{[]rune("x/abc/j\n"), nil, invalidCmdError('j')},
		{[]rune("s/abc/def/\n"), &Cmd{re: "abc", text: "def", num: 1, cmdc: 's'}, nil},
		{[]rune("s/abc/def/g\n"), &Cmd{re: "abc", text: "def", num: 1, flag: 'g', cmdc: 's'}, nil},
		{[]rune("s2/abc/def/\n"), &Cmd{re: "abc", text: "def", num: 2, cmdc: 's'}, nil},
		{[]rune("/abc/ s//def/\n"), &Cmd{
			addr: &Addr{typ: '/', re: "abc"},
			re:   "abc", text: "def", num: 1, cmdc: 's',
		}, nil},
		{[]rune("s//xyz/\n"), nil, errRegexpMissing},
		{[]rune("s/abc/def\\"), nil, errBadRHS},
		{[]rune("3.,17d\n"), nil, errBadAddrSyntax},
		{[]rune("5u\n"), nil, errAddrNotRequired},
		{[]rune("j\n"), nil, invalidCmdError('j')},
		{[]rune("{}\n"), &Cmd{cmdc: '{'}, nil},
		{[]rune("{\nd\nu\n}\n"), &Cmd{
			cmd:  &Cmd{cmdc: 'd', next: &Cmd{cmdc: 'u', num: 1}},
			cmdc: '{',
		}, nil},
		{[]rune("{j}\n"), nil, invalidCmdError('j')},
		{[]rune("{\nj\n}\n"), nil, invalidCmdError('j')},
		{[]rune("}\n"), nil, errLeftBraceMissing},
		{[]rune("cd\n"), nil, invalidCmdError('c' | 0x100)},
		{[]rune("t 42.\n"), nil, errBadAddrSyntax},
		{[]rune("t\n"), nil, errBadAddr},
		{[]rune("B abc.txt\n"), &Cmd{cmdc: 'B', text: " abc.txt"}, nil},
		{[]rune("g\n"), nil, errAddressMissing},
		{[]rune(`g\abc\` + "\n"), nil, badDelimiterError('\\')},
		{[]rune("u\n"), &Cmd{num: 1, cmdc: 'u'}, nil},
		{[]rune("u5\n"), &Cmd{num: 5, cmdc: 'u'}, nil},
		{[]rune("u-3\n"), &Cmd{num: -3, cmdc: 'u'}, nil},
	}
	for _, tc := range tt {
		lastpat = ""
		cp := &cmdParser{
			buf: tc.input,
			pos: 0,
		}
		cmd, err := cp.parse(0)
		if err != tc.err {
			t.Errorf("parsing command %q returned error %v; expected %v",
				tc.input, err, tc.err)
			continue
		}
		if !reflect.DeepEqual(cmd, tc.cmd) {
			t.Errorf("bad parse result for command %q:\n"+
				"got: %v\n"+
				"expected: %v",
				tc.input, cmd, tc.cmd)
		}
	}
}

func TestCollecttoken(t *testing.T) {
	tt := []struct {
		cmd []rune
		end string
		out string
	}{
		{[]rune(" foo bar\t\n"), linex, " foo bar\t"},
		{[]rune(" foo bar\t\nquux"), linex, " foo bar\t"},
		{[]rune(" αβγ テスト\t\n世界"), linex, " αβγ テスト\t"},
		{[]rune(" foo bar\t\n"), wordx, " foo bar"},
		{[]rune(" foo bar\t\nquux"), wordx, " foo bar"},
		{[]rune(" αβγ テスト\t\n世界"), wordx, " αβγ テスト"},
	}
	for _, tc := range tt {
		cp := &cmdParser{
			buf: tc.cmd,
			pos: 0,
		}
		out := cp.collecttoken(tc.end)
		if out != tc.out {
			t.Errorf("collecttoken(%q) of command %q is %q; exptected %q",
				tc.end, tc.cmd, out, tc.out)
		}
	}
}

type addrTest struct {
	cmd  []rune
	addr *Addr
	err  error
}

func TestSimpleaddr(t *testing.T) {
	tt := []addrTest{
		{nil, nil, nil},
		{[]rune{}, nil, nil},
		{[]rune("\n"), nil, nil},
		{[]rune("#123\n"), &Addr{typ: '#', num: 123}, nil},
		{[]rune("#\n"), &Addr{typ: '#', num: 1}, nil},
		{[]rune("42\n"), &Addr{typ: 'l', num: 42}, nil},
		{[]rune("1234567890\n"), &Addr{typ: 'l', num: 1234567890}, nil},
		{[]rune("/abc\n"), &Addr{typ: '/', re: "abc"}, nil},
		{[]rune("/abc/\n"), &Addr{typ: '/', re: "abc"}, nil},
		{[]rune(`/a\/bc/` + "\n"), &Addr{typ: '/', re: "a/bc"}, nil},
		{[]rune(`/a\nbc/` + "\n"), &Addr{typ: '/', re: `a\nbc`}, nil},
		{[]rune(`/a\\bc/` + "\n"), &Addr{typ: '/', re: `a\\bc`}, nil},
		{[]rune("?abc\n"), &Addr{typ: '?', re: "abc"}, nil},
		{[]rune("?abc?\n"), &Addr{typ: '?', re: "abc"}, nil},
		{[]rune(`?a\?bc?` + "\n"), &Addr{typ: '?', re: "a?bc"}, nil},
		{[]rune(`?a\nbc?` + "\n"), &Addr{typ: '?', re: `a\nbc`}, nil},
		{[]rune(`?a\\bc?` + "\n"), &Addr{typ: '?', re: `a\\bc`}, nil},
		{[]rune(`"abc` + "\n"), &Addr{typ: '"', re: "abc"}, nil},
		{[]rune(`"abc"` + "\n"), &Addr{typ: '"', re: "abc"}, nil},
		{[]rune(".\n"), &Addr{typ: '.'}, nil},
		{[]rune("$\n"), &Addr{typ: '$'}, nil},
		{[]rune("+\n"), &Addr{typ: '+'}, nil},
		{[]rune("-\n"), &Addr{typ: '-'}, nil},
		{[]rune("'\n"), &Addr{typ: '\''}, nil},
		{[]rune("abc\n"), nil, nil},
		{[]rune("42.\n"), nil, errBadAddrSyntax},
		{[]rune("42$\n"), nil, errBadAddrSyntax},
		{[]rune("42'\n"), nil, errBadAddrSyntax},
		{[]rune("42\"\n"), nil, errRegexpMissing},
		{[]rune(`"abc" "cdf" "efg"` + "\n"), nil, errBadAddrSyntax},
		{[]rune("\"abc\" 42\n"), &Addr{typ: '"', re: "abc", next: &Addr{typ: 'l', num: 42}}, nil},
		{[]rune(".42\n"), &Addr{
			typ: '.', next: &Addr{
				typ: '+', next: &Addr{typ: 'l', num: 42},
			},
		}, nil},
		{[]rune("42/abc/\n"), &Addr{
			typ: 'l', num: 42, next: &Addr{
				typ: '+', next: &Addr{typ: '/', re: "abc"},
			},
		}, nil},
		{[]rune("42/abc/\n"), &Addr{
			typ: 'l', num: 42, next: &Addr{
				typ: '+', next: &Addr{typ: '/', re: "abc"},
			},
		}, nil},
		{[]rune("+/abc/\n"), &Addr{typ: '+', next: &Addr{typ: '/', re: "abc"}}, nil},
		{[]rune("-/abc/\n"), &Addr{typ: '-', next: &Addr{typ: '/', re: "abc"}}, nil},
		{[]rune(".+\n"), &Addr{typ: '.', next: &Addr{typ: '+', num: 0}}, nil},
		{[]rune(".-\n"), &Addr{typ: '.', next: &Addr{typ: '-', num: 0}}, nil},
	}
	runAddrTests(t, tt, (*cmdParser).simpleaddr)
}

func TestCompoundaddr(t *testing.T) {
	tt := []addrTest{
		{[]rune("3,17\n"), &Addr{
			typ:  ',',
			left: &Addr{typ: 'l', num: 3},
			next: &Addr{typ: 'l', num: 17}}, nil},
		{[]rune("3,\n"), &Addr{typ: ',', left: &Addr{typ: 'l', num: 3}, next: nil}, nil},
		{[]rune(",17\n"), &Addr{typ: ',', left: nil, next: &Addr{typ: 'l', num: 17}}, nil},
		{[]rune("37;/abc/\n"), &Addr{
			typ:  ';',
			left: &Addr{typ: 'l', num: 37},
			next: &Addr{typ: '/', re: "abc"},
		}, nil},
		{[]rune("3.,17\n"), nil, errBadAddrSyntax},
		{[]rune("3,17.\n"), nil, errBadAddrSyntax},
		{[]rune("3,,17\n"), nil, errBadAddrSyntax},
		{[]rune("3;;17\n"), nil, errBadAddrSyntax},
	}
	runAddrTests(t, tt, (*cmdParser).compoundaddr)
}

func runAddrTests(t *testing.T, tt []addrTest, parse func(*cmdParser) (*Addr, error)) {
	for _, tc := range tt {
		lastpat = ""
		cp := &cmdParser{
			buf: tc.cmd,
			pos: 0,
		}
		addr, err := parse(cp)
		if tc.err != err {
			t.Errorf("parsing address %q returned error %v; expected %v",
				tc.cmd, err, tc.err)
			continue
		}
		if !reflect.DeepEqual(addr, tc.addr) {
			t.Errorf("bad parse result for address %q:\n"+
				"got: %v\n"+
				"expected: %v",
				tc.cmd, addr, tc.addr)
		}
	}
}

func (a *Addr) String() string {
	if a == nil {
		return "nil"
	}
	return fmt.Sprintf("Addr{typ: %c, re: %q, left: %v, num: %v, next: %v}",
		a.typ, a.re, a.left, a.num, a.next)
}

func (c *Cmd) String() string {
	if c == nil {
		return "nil"
	}
	return fmt.Sprintf("Cmd{addr: %v, re: %q, cmd: %v, text: %q, mtaddr: %v, next: %v, num: %v, flag: %v, cmdc: %q}",
		c.addr, c.re, c.cmd, c.text, c.mtaddr, c.next, c.num, c.flag, c.cmdc)
}

func TestInvalidCmdError(t *testing.T) {
	got := invalidCmdError('j').Error()
	want := "unknown command j"
	if got != want {
		t.Errorf("invalidCmdError is %v; expected %v", got, want)
	}
}

func TestBadDelimiterError(t *testing.T) {
	got := badDelimiterError('x').Error()
	want := "bad delimiter x"
	if got != want {
		t.Errorf("badDelimiterError is %v; expected %v", got, want)
	}
}

// MockFrame is a mock implementation of a frame.Frame that does nothing.
type MockFrame struct{}

func (mf *MockFrame) GetFrameFillStatus() frame.FrameFillStatus {
	return frame.FrameFillStatus{
		Nchars:         0,
		Nlines:         0,
		Maxlines:       0,
		MaxPixelHeight: 0,
	}
}
func (mf *MockFrame) Charofpt(pt image.Point) int                  { return 0 }
func (mf *MockFrame) DefaultFontHeight() int                       { return 0 }
func (mf *MockFrame) Delete(int, int) int                          { return 0 }
func (mf *MockFrame) Insert([]rune, int) bool                      { return false }
func (mf *MockFrame) IsLastLineFull() bool                         { return false }
func (mf *MockFrame) Rect() image.Rectangle                        { return image.Rect(0, 0, 0, 0) }
func (mf *MockFrame) TextOccupiedHeight(r image.Rectangle) int     { return 0 }
func (mf *MockFrame) Maxtab(_ int)                                 {}
func (mf *MockFrame) GetMaxtab() int                               { return 0 }
func (mf *MockFrame) Init(image.Rectangle, ...frame.OptionClosure) {}
func (mf *MockFrame) Clear(bool)                                   {}
func (mf *MockFrame) Ptofchar(int) image.Point                     { return image.Point{0, 0} }
func (mf *MockFrame) Redraw(enclosing image.Rectangle)             {}
func (mf *MockFrame) GetSelectionExtent() (int, int)               { return 0, 0 }
func (mf *MockFrame) Select(*draw.Mousectl, *draw.Mouse, func(frame.SelectScrollUpdater, int)) (int, int) {
	return 0, 0
}
func (mf *MockFrame) SelectOpt(*draw.Mousectl, *draw.Mouse, func(frame.SelectScrollUpdater, int), draw.Image, draw.Image) (int, int) {
	return 0, 0
}
func (mf *MockFrame) DrawSel(image.Point, int, int, bool) {}
