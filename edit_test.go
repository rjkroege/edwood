package main

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/frame"
)

type teststimulus struct {
	dot           Range
	filename      string
	expr          string
	expected      string
	expectedwarns []string
}

func TestEdit(t *testing.T) {
	runfunc = mockrun
	defer func() { runfunc = run }()
	cedit = make(chan int)

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
		// 21
		{Range{0, len(contents)}, "test", "s/short/long/", "This is a\nlong text\nto try addressing\n", []string{}},
		{Range{0, len(contents)}, "test", "s/(i.)/!\\1!/g", "Th!is! !is! a\nshort text\nto try address!in!g\n", []string{}},

		// =
		// 23
		{Range{1, 3}, "test", "=", "This is a\nshort text\nto try addressing\n", []string{"test:1\n"}},
		{Range{1, 3}, "test", "=+", "This is a\nshort text\nto try addressing\n", []string{"test:1+#1\n"}},
		{Range{1, 3}, "test", "=#", "This is a\nshort text\nto try addressing\n", []string{"test:#1,#3\n"}},

		// p
		{Range{0, 4}, "test", "p", "This is a\nshort text\nto try addressing\n", []string{"This"}},

		// x
		{Range{0, 4}, "test", ",x/$/ a/@/", "This is a@\nshort text@\nto try addressing@\n@", []string{}},
		{Range{0, 4}, "test", ",x a/@/", "This is a@\nshort text@\nto try addressing@\n", []string{}},

		// \n is missing because we have no way to determine if the result is correct.

		// | > <
		// 30
		{Range{0, 4}, "test", "|pipe", "{\"|pipe\" \".\" true \"\" \"\" true} is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 4}, "test", ">greater", "This is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 4}, "test", "<less", "{\"<less\" \".\" true \"\" \"\" true} is a\nshort text\nto try addressing\n", []string{}},
		{Range{0, 4}, "test", "<error", "This is a\nshort text\nto try addressing\n", []string{"Edit: mockrun failed!\n"}},

		// { } NB: grouping requires newlines. And sets . the same for each of the commands.
		{Range{0, 0}, "test", ",x {\n i/@/ \n a/%/\n }", "@This is a%\n@short text%\n@to try addressing%\n", []string{}},
		// TODO(rjk): { has a number of constraints not being exercised in this test.
	}

	buf := make([]rune, 8192)

	for i, test := range testtab {
		warningsMu.Lock()
		warnings = []*Warning{}
		warningsMu.Unlock()

		w := makeSkeletonWindowModel(test.dot, test.filename)

		// All middle button commands including Edit run inside a lock discipline
		// set up by MovedMouse. We need to mirror this for external process
		// accessing Edit commands.
		row.lk.Lock()
		w.Lock('M')
		editcmd(&w.body, []rune(test.expr))
		w.Unlock()
		row.lk.Unlock()

		n, _ := w.body.ReadB(0, buf[:])
		if string(buf[:n]) != test.expected {
			t.Errorf("test %d: File.b contents expected \n%#v\nbut got \n%#v\n", i, test.expected, string(buf[:n]))
		}

		warningsMu.Lock()
		if got, want := len(warnings), len(test.expectedwarns); got != want {
			t.Errorf("text %d: expected %d warnings but got %d warnings", i, want, got)
			for i := range warnings {
				t.Errorf("Warning #%d received: %s\n", i, warnings[i].buf.String())
			}
			break
		}

		for j, tw := range test.expectedwarns {
			n, _ := warnings[j].buf.Read(0, buf[:])
			if string(buf[:n]) != tw {
				t.Errorf("test %d: Warning %d contents expected \n%#v\nbut got \n%#v\n", i, j, tw, string(buf[:n]))
			}

		}
		warningsMu.Unlock()
	}
}

const contents = "This is a\nshort text\nto try addressing\n"
const alt_contents = "A different text\nWith other contents\nSo there!\n"

func makeSkeletonWindowModel(dot Range, filename string) *Window {
	MakeWindowScaffold(&dumpfile.Content{
		Columns: []dumpfile.Column{
			{},
		},
		Windows: []*dumpfile.Window{
			{
				Column: 0,
				Tag: dumpfile.Text{
					Buffer: filename,
				},
				Body: dumpfile.Text{
					Buffer: contents,
					Q0:     dot.q0,
					Q1:     dot.q1,
				},
			},
			{
				Column: 0,
				Tag: dumpfile.Text{
					Buffer: "alt_example_2",
				},
				Body: dumpfile.Text{
					Buffer: alt_contents,
				},
			},
		},
	})

	return row.col[0].w[0]
}

func makeTempFile(contents string) (string, func(), error) {
	tfd, err := ioutil.TempFile("", "example")
	if err != nil {
		return "", func() {}, err
	}

	cleaner := func() {
		os.Remove(tfd.Name())
	}

	if _, err := tfd.WriteString(contents); err != nil {
		return "", cleaner, err
	}
	if err := tfd.Close(); err != nil {
		return "", cleaner, err
	}
	return tfd.Name(), cleaner, nil
}

func TestEditCmdWithFile(t *testing.T) {
	fname, cleaner, err := makeTempFile(contents)
	defer cleaner()
	if err != nil {
		t.Fatalf("can't make a temp file because: %v\n", err)
	}

	testtab := []teststimulus{
		// e
		{Range{0, 0}, fname, "e " + fname, contents, []string{}},

		// r
		{Range{0, 0}, fname, "r " + fname, contents + contents, []string{}},
		{Range{0, len(contents)}, fname, "r " + fname, contents, []string{}},
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
		w := makeSkeletonWindowModel(test.dot, test.filename)

		editcmd(&w.body, []rune(test.expr))

		n, _ := w.body.ReadB(0, buf[:])
		if string(buf[:n]) != test.expected {
			t.Errorf("test %d: TestAppend expected \n%v\nbut got \n%v\n", i, test.expected, string(buf[:n]))
		}

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

func TestEditMultipleWindows(t *testing.T) {
	fn1, cleaner, err := makeTempFile("file one\n")
	defer cleaner()
	if err != nil {
		t.Fatalf("can't make a temp file because: %v\n", err)
	}
	fn2, cleaner, err := makeTempFile("file two\n")
	defer cleaner()
	if err != nil {
		t.Fatalf("can't make a temp file because: %v\n", err)
	}

	// Used only for w and altered in the test.
	fn3, cleaner, err := makeTempFile("file three\n")
	defer cleaner()
	if err != nil {
		t.Fatalf("can't make a temp file because: %v\n", err)
	}

	testtab := []struct {
		dot           Range
		filename      string
		expr          string
		expected      []string
		expectedwarns []string
	}{
		// X
		{Range{0, 0}, "test", "X/.*/ ,x i/@/", []string{
			"@This is a\n@short text\n@to try addressing\n",
			"@A different text\n@With other contents\n@So there!\n",
		}, []string{}},

		{Range{0, 6}, "test", "X/.*/=", []string{
			contents,
			alt_contents,
		}, []string{"test:1\nalt_example_2:1\n"}},

		// X + D
		{Range{0, 6}, "test", "X/alt.*/D", []string{
			contents,
		}, []string{}},

		// Y
		{Range{0, 6}, "test", "Y/alt.*/=", []string{
			contents,
			alt_contents,
		}, []string{"test:1\n"}},

		// B
		{Range{0, 0}, "test", "B " + fn1 + " " + fn2, []string{
			contents,
			alt_contents,
			"file one\n",
			"file two\n",
		}, []string{}},
		{Range{0, 0}, "test", "B", []string{
			contents,
			alt_contents,
		}, []string{"Edit: no file name given\n"}},

		// b does the same thing in Acme and Edwood (fails)
		// Maybe this sets currobserver?

		// w
		// backing file is newer than file.
		{Range{0, 0}, fn3, "w", []string{contents, alt_contents}, []string{
			fn3 + " not written; file already exists\n",
		}},

		// b
		{Range{0, 0}, "test", "b alt_example_2\ni/inserted/\n", []string{
			contents,
			"inserted" + alt_contents,
		}, []string{
			"'+  alt_example_2\n",
		}},
		{Range{0, 0}, "test", "b alt_example_2\n1 i/1/\n2 i/2/\n", []string{
			contents,
			"1A different text\n2With other contents\nSo there!\n",
		}, []string{
			"'+  alt_example_2\n",
		}},
		// TODO(rjk): the edit result here is wrong. See #236.
		{Range{0, 0}, "test", "b alt_example_2\n2 i/2/\n1 i/1/\n", []string{
			contents,
			"1A different text2\nWith other contents\nSo there!\n",
		}, []string{
			"'+  alt_example_2\nwarning: changes out of sequence\nwarning: changes out of sequence, edit result probably wrong\n",
		}},

		// u
		// 10
		{Range{0, 0}, "test", "u", []string{
			contents,
			alt_contents,
		}, []string{}},
		{Range{0, 0}, "test", "1,$p\nu", []string{
			contents,
			alt_contents,
		}, []string{"helloThis is a\nshort text\nto try addressing\n"}},
		{Range{0, 0}, "test", "1,$p\nu-1\n", []string{
			"hello" + contents,
			alt_contents,
		}, []string{"This is a\nshort text\nto try addressing\n"}},
	}

	buf := make([]rune, 8192)

	for i, test := range testtab {
		warnings = []*Warning{}
		makeSkeletonWindowModel(test.dot, test.filename)

		// TODO(rjk): Make this nicer.
		if i == 11 || i == 12 {
			// special setup for undo
			InsertString(row.col[0].w[0], "hello")
			if i == 12 {
				// Undo the above insertion.
				row.col[0].w[0].Undo(true)
			}
		}

		w := row.col[0].w[0]
		w.Lock('M')
		editcmd(&w.body, []rune(test.expr))
		w.Unlock()

		if got, want := len(row.col[0].w), len(test.expected); got != want {
			t.Errorf("test %d: expected %d windows but got %d windows", i, want, got)
			break
		}

		for j, exp := range test.expected {
			w := row.col[0].w[j]
			n, _ := w.body.ReadB(0, buf[:])
			if string(buf[:n]) != exp {
				t.Errorf("test %d: Window %d File.b contents expected %#v\nbut got \n%#v\n", i, j, exp, string(buf[:n]))
			}

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
		// TODO(rjk): Validate that the files on disk have the correct state.
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
func (mf *MockFrame) DefaultFontHeight() int                       { return 10 }
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

func mockrun(win *Window, s string, rdir string, newns bool, argaddr string, xarg string, iseditcmd bool) {
	// Optionally generate an error.
	if s[1:] == "error" {
		// TODO(rjk): Create more complex error cases.
		editerror("mockrun failed!")
		return
	}

	go func() {
		// At this point, an external command attaches to the Edwood and writes
		// data to somewhere in the filesystem. This comes from xfidwrite via
		// edittext into the elog. We write expectations in string form into the
		// buffer here from the inputs so that the test harness can validate
		// them.

		ds := fmt.Sprintf("{%#v %#v %#v %#v %#v %#v}", s, rdir, newns, argaddr, xarg, iseditcmd)

		if s[0] != '>' {
			row.lk.Lock()
			win.Lock('M')
			edittext(win, 4, []rune(ds))
			win.Unlock()
			row.lk.Unlock()
		}

		cedit <- 0
	}()
}
