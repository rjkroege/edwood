package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEdit(t *testing.T) {
	testtab := []struct {
		dot      Range
		filename string
		expr     string
		expected string
	}{

		// 0
		{Range{0, 0}, "test", "a/junk", "junkThis is a\nshort text\nto try addressing\n"},
		{Range{7, 12}, "test", "a/junk", "This is a\nshjunkort text\nto try addressing\n"},
		{Range{0, 0}, "test", "/This/a/junk", "Thisjunk is a\nshort text\nto try addressing\n"},
		{Range{0, 0}, "test", "/^/a/junk", "This is a\njunkshort text\nto try addressing\n"},
		{Range{0, 0}, "test", "/$/a/junk", "This is ajunk\nshort text\nto try addressing\n"},

		// 4
		{Range{0, 0}, "test", "i/junk", "junkThis is a\nshort text\nto try addressing\n"},
		{Range{2, 6}, "test", "i/junk", "Thjunkis is a\nshort text\nto try addressing\n"},
		{Range{0, 0}, "test", "/text/i/junk", "This is a\nshort junktext\nto try addressing\n"},

		// Don't know how to automate testing of 'b'

		// c
		// 7
		{Range{0, 0}, "test", "c/junk", "junkThis is a\nshort text\nto try addressing\n"},
		{Range{2, 6}, "test", "c/junk", "Thjunks a\nshort text\nto try addressing\n"},
		{Range{0, 0}, "test", "/text/c/junk", "This is a\nshort junk\nto try addressing\n"},

		// d
		// 10
		{Range{0, 0}, "test", "d", "This is a\nshort text\nto try addressing\n"},
		{Range{2, 6}, "test", "d", "Ths a\nshort text\nto try addressing\n"},
		{Range{0, 0}, "test", "/text/d", "This is a\nshort \nto try addressing\n"},

		// e - Don't know how to test e

		// f - Don't know how to test f

		// g/v
		{Range{0, 0}, "test", "g/This/d", "This is a\nshort text\nto try addressing\n"},
		{Range{0, 12}, "test", "g/This/d", "ort text\nto try addressing\n"},
		{Range{0, 3}, "test", "v/This/d", "s is a\nshort text\nto try addressing\n"},
		{Range{0, 12}, "test", "v/This/d", "This is a\nshort text\nto try addressing\n"},

		// m/t
		// 17
		{Range{0, 4}, "test", "m/try", " is a\nshort text\nto tryThis addressing\n"},
		{Range{0, 3}, "test", "t/try", "This is a\nshort text\nto tryThi addressing\n"},
	}

	buf := make([]rune, 8192)

	for i, test := range testtab {
		w := NewWindow().initHeadless(nil)
		w.body.Insert(0, []rune("This is a\nshort text\nto try addressing\n"), true)
		w.body.SetQ0(test.dot.q0)
		w.body.SetQ1(test.dot.q1)
		editcmd(&w.body, []rune(test.expr))
		// Normally the edit log is applied in allupdate, but we don't have
		// all the window machinery, so we apply it by hand.
		w.body.file.elog.Apply(&w.body)
		n, _ := w.body.ReadB(0, buf[:])
		if string(buf[:n]) != test.expected {
			t.Errorf("test %d: TestAppend expected \n%v\nbut got \n%v\n", i, test.expected, string(buf[:n]))
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
		cmdstartp = tc.input
		cmdp = 0
		lastpat = ""
		cmd, err := parsecmd(0)
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
		cmdstartp = tc.cmd
		cmdp = 0
		out := collecttoken(tc.end)
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

	runAddrTests(t, tt, simpleaddr)
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
	runAddrTests(t, tt, compoundaddr)
}

func runAddrTests(t *testing.T, tt []addrTest, parse func() (*Addr, error)) {
	for _, tc := range tt {
		cmdstartp = tc.cmd
		cmdp = 0
		lastpat = ""
		addr, err := parse()
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
		t.Errorf("invalidCmdError is %v; expected %v", got, want)
	}
}
