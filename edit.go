package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/rjkroege/edwood/file"
)

var (
	errBadAddr          = fmt.Errorf("bad address")
	errBadAddrSyntax    = fmt.Errorf("bad address syntax")
	errAddressMissing   = fmt.Errorf("no address")
	errAddrNotRequired  = fmt.Errorf("command takes no address")
	errRegexpMissing    = fmt.Errorf("no regular expression defined")
	errLeftBraceMissing = fmt.Errorf("right brace with no left brace")
	errBadRHS           = fmt.Errorf("bad right hand side")
)

type invalidCmdError rune

func (e invalidCmdError) Error() string {
	return fmt.Sprintf("unknown command %c", rune(e))
}

type badDelimiterError rune

func (e badDelimiterError) Error() string {
	return fmt.Sprintf("bad delimiter %c", rune(e))
}

type Addr struct {
	typ  rune // # (byte addr), l (line addr), / ? . $ + - , ;
	re   string
	left *Addr // left side of , and ;
	num  int
	next *Addr // or right side of , and ;
}

type Address struct {
	r    Range
	file *file.ObservableEditableBuffer
}

type Cmd struct {
	addr   *Addr  // address (range of text)
	re     string // regular expression for e.g. 'x'
	cmd    *Cmd   // target of x, g, {, etc.
	text   string // text of a, c, i; rhs of s
	mtaddr *Addr  // address for m, t
	next   *Cmd   // pointer to next element in braces
	num    int
	flag   rune // whatever
	cmdc   rune // command character; 'x' etc.
}

type Cmdtab struct {
	cmdc    rune                   // command character
	text    bool                   // takes a textual argument?
	regexp  bool                   // takes a regular expression?
	addr    bool                   // takes an address (m or t)?
	defcmd  rune                   // default command; 0==>none
	defaddr Defaddr                // default address
	count   countType              // count type (e.g. s can take an unsigned count: s2///)
	token   string                 // takes text terminated by one of these
	fn      func(*Text, *Cmd) bool // function to call with parse tree
}

type Defaddr int

const (
	aNo Defaddr = iota
	aDot
	aAll
)

type countType int

const (
	cNo countType = iota
	cUnsigned
	cSigned
)

const (
	linex = "\n"
	wordx = "\t\n"
)

var cmdtab = []Cmdtab{
	// cmdc	text	regexp	addr	defcmd	defaddr	count	token	 fn
	{'\n', false, false, false, 0, aDot, cNo, "", nl_cmd},
	{'a', true, false, false, 0, aDot, cNo, "", a_cmd},
	{'b', false, false, false, 0, aNo, cNo, linex, b_cmd},
	{'c', true, false, false, 0, aDot, cNo, "", c_cmd},
	{'d', false, false, false, 0, aDot, cNo, "", d_cmd},
	{'e', false, false, false, 0, aNo, cNo, wordx, e_cmd},
	{'f', false, false, false, 0, aNo, cNo, wordx, f_cmd},
	{'g', false, true, false, 'p', aDot, cNo, "", nil}, // Assingned to g_cmd in init() to avoid initialization loop
	{'i', true, false, false, 0, aDot, cNo, "", i_cmd},
	{'m', false, false, true, 0, aDot, cNo, "", m_cmd},
	{'p', false, false, false, 0, aDot, cNo, "", p_cmd},
	{'r', false, false, false, 0, aDot, cNo, wordx, e_cmd},
	{'s', false, true, false, 0, aDot, cUnsigned, "", s_cmd},
	{'t', false, false, true, 0, aDot, cNo, "", m_cmd},
	{'u', false, false, false, 0, aNo, cSigned, "", u_cmd},
	{'v', false, true, false, 'p', aDot, cNo, "", nil}, // Assingned to g_cmd in init() to avoid initialization loop
	{'w', false, false, false, 0, aAll, cNo, wordx, w_cmd},
	{'x', false, true, false, 'p', aDot, cNo, "", nil}, // Assingned to x_cmd in init() to avoid initialization loop
	{'y', false, true, false, 'p', aDot, cNo, "", nil}, // Assingned to x_cmd in init() to avoid initialization loop
	{'=', false, false, false, 0, aDot, cNo, linex, eq_cmd},
	{'B', false, false, false, 0, aNo, cNo, linex, B_cmd},
	{'D', false, false, false, 0, aNo, cNo, linex, D_cmd},
	{'X', false, true, false, 'f', aNo, cNo, "", nil}, // Assingned to X_cmd in init() to avoid initialization loop
	{'Y', false, true, false, 'f', aNo, cNo, "", nil}, // Assingned to X_cmd in init() to avoid initialization loop
	{'<', false, false, false, 0, aDot, cNo, linex, pipe_cmd},
	{'|', false, false, false, 0, aDot, cNo, linex, pipe_cmd},
	{'>', false, false, false, 0, aDot, cNo, linex, pipe_cmd},
	/* deliberately unimplemented:
	{'k', false, false, false, 0, aDot, cNo, "", k_cmd},
	{'n', false, false, false, 0, aNo, cNo, "", n_cmd},
	{'q', false, false, false, 0, aNo, cNo, "", q_cmd},
	{'!', false, false, false, 0, aNo, cNo, linex, plan9_cmd},
	*/
}

func init() {
	for i, c := range cmdtab {
		switch c.cmdc {
		case 'g', 'v':
			cmdtab[i].fn = g_cmd
		case 'x', 'y':
			cmdtab[i].fn = x_cmd
		case 'X', 'Y':
			cmdtab[i].fn = X_cmd
		}
	}
}

var (
	editerrc chan error

	lastpat string
)

type cmdParser struct {
	buf []rune
	pos int
}

func editthread(cp *cmdParser) {
	for {
		cmd, err := cp.parse(0)
		if err != nil {
			editerror("%v", err)
		}
		if cmd == nil {
			break
		}
		if !cmdexec(curtext, cmd) {
			break
		}
	}
	editerrc <- nil
}

func allelogterm(w *Window) {
	w.body.file.Elog.Term()
}

func alleditinit(w *Window) {
	w.tag.Commit()
	w.body.Commit()
	w.body.file.EditClean = false
}

func allupdate(w *Window) {
	t := &w.body
	f := t.file

	if !f.Elog.Empty() {
		owner := t.w.owner
		if owner == 0 {
			t.w.owner = 'E'
		}
		// Set an undo point before applying accumulated Edit actions.
		f.Mark(seq)
		f.Elog.Apply(t)
		if f.EditClean {
			f.Clean()
		}
		t.w.owner = owner
	}
	w.SetTag()
}

func editerror(format string, args ...interface{}) {
	s := fmt.Errorf(format, args...)
	row.AllWindows(allelogterm) // truncate the edit logs
	editerrc <- s
	runtime.Goexit()
}

func editcmd(ct *Text, r []rune) {
	if len(r) == 0 {
		return
	}

	if len(r) > 2*RBUFSIZE {
		warning(nil, "string too long\n")
		return
	}

	row.AllWindows(alleditinit)
	cp := newCmdParser(r)
	if ct.w == nil {
		curtext = nil
	} else {
		curtext = &ct.w.body
	}
	resetxec()
	if editerrc == nil {
		editerrc = make(chan error)
		lastpat = ""
	}
	// We would appear to run the Edit command on a different thread
	// but block here.
	go editthread(cp)
	err := <-editerrc
	editing = Inactive
	if err != nil {
		warning(nil, "Edit: %s\n", err)
	}
	// update everyone whose edit log has data
	row.AllWindows(allupdate)
}

func newCmdParser(r []rune) *cmdParser {
	buf := make([]rune, len(r), len(r)+1)
	copy(buf, r)
	if r[len(r)-1] != '\n' {
		buf = append(r, '\n')
	}
	return &cmdParser{
		buf: buf,
		pos: 0,
	}
}

func (cp *cmdParser) getch() rune {
	if cp.pos == len(cp.buf) {
		return -1
	}
	c := cp.buf[cp.pos]
	cp.pos++
	return c
}

func (cp *cmdParser) nextc() rune {
	if cp.pos == len(cp.buf) {
		return -1
	}
	return cp.buf[cp.pos]
}

func (cp *cmdParser) ungetch() {
	cp.pos--
	if cp.pos < 0 {
		panic("ungetch")
	}
}

func (cp *cmdParser) getnum(signok bool) int {
	n := int(0)
	sign := int(1)
	if signok && cp.nextc() == '-' {
		sign = -1
		cp.getch()
	}
	c := cp.nextc()
	if c < '0' || '9' < c { // no number defaults to 1
		return sign
	}

	for {
		c = cp.getch()
		if !('0' <= c && c <= '9') {
			break
		}
		n = n*10 + int(c-'0')
	}
	cp.ungetch()
	return sign * n
}

func (cp *cmdParser) skipbl() rune {
	var c rune
	for {
		c = cp.getch()
		if !(c == ' ' || c == '\t') {
			break
		}
	}

	if c >= 0 {
		cp.ungetch()
	}
	return c
}

func newcmd() *Cmd {
	return &Cmd{}
}

func newaddr() *Addr {
	return &Addr{}
}

func okdelim(c rune) bool {
	return !(c == '\\' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9'))
}

func (cp *cmdParser) atnl() {
	cp.skipbl()
	c := cp.getch()
	if c != '\n' {
		debug.PrintStack()
		editerror("newline expected (saw %c)", c)
	}
}

func (cp *cmdParser) getrhs(delim rune, cmd rune) (s string, err error) {
	var c rune

	for {
		c = cp.getch()
		if !((c) > 0 && c != delim && c != '\n') {
			break
		}
		if c == '\\' {
			c = cp.getch()
			if (c) <= 0 {
				return "", errBadRHS
			}
			if c == '\n' {
				cp.ungetch()
				c = '\\'
			} else {
				if c == 'n' {
					c = '\n'
				} else {
					if c != delim && (cmd == 's' || c != '\\') { // s does its own
						s = s + "\\" // TODO(flux): Use a stringbuilder
					}
				}
			}
		}
		s = s + string(c)
	}
	cp.ungetch() // let client read whether delimiter, '\n' or whatever
	return
}

func (cp *cmdParser) collecttoken(end string) string {
	var s strings.Builder
	var c rune

	for {
		c = cp.nextc()
		if c != ' ' && c != '\t' {
			break
		}
		s.WriteRune(cp.getch()) // blanks significant for getname()
	}
	for {
		c = cp.getch()
		if c <= 0 || strings.ContainsRune(end, c) {
			break
		}
		s.WriteRune(c)
	}
	if c != '\n' {
		cp.atnl()
	}
	return s.String()
}

func (cp *cmdParser) collecttext() (string, error) {
	var begline, i int
	var c, delim rune

	s := ""
	if cp.skipbl() == '\n' {
		cp.getch()
		i = 0
		for {
			begline = i
			for {
				c = cp.getch()
				if !(c > 0 && c != '\n') {
					break
				}
				i++
				s = s + string(c)
			}
			i++
			s = s + "\n"
			if c < 0 {
				return s, nil
			}
			if !(s[begline] != '.' || s[begline+1] != '\n') {
				break
			}
		}
		s = s[:len(s)-2]
	} else {
		delim = cp.getch()
		if !okdelim(delim) {
			return "", badDelimiterError(delim)
		}
		var err error
		s, err = cp.getrhs(delim, 'a')
		if err != nil {
			return "", err
		}
		if cp.nextc() == delim {
			cp.getch()
		}
		cp.atnl()
	}
	return s, nil
}

func cmdlookup(c rune) int {
	for i, cmd := range cmdtab {
		if cmd.cmdc == c {
			return i
		}
	}
	return -1
}

func (cp *cmdParser) parse(nest int) (*Cmd, error) {
	var cmd Cmd
	var err error

	cmd.addr, err = cp.compoundaddr()
	if err != nil {
		return nil, err
	}
	if cp.skipbl() == -1 {
		return nil, nil
	}
	c := cp.getch()
	if c == -1 {
		return nil, nil
	}
	cmd.cmdc = c
	if cmd.cmdc == 'c' && cp.nextc() == 'd' { // sleazy two-character case
		cp.getch() // the 'd'
		cmd.cmdc = 'c' | 0x100
	}
	i := cmdlookup(cmd.cmdc)
	if i >= 0 {
		if cmd.cmdc == '\n' {
			return &cmd, nil // let nl_cmd work it all out
		}
		ct := &cmdtab[i]
		if ct.defaddr == aNo && cmd.addr != nil {
			return nil, errAddrNotRequired
		}
		if ct.count != cNo {
			cmd.num = cp.getnum(ct.count == cSigned)
		}
		if ct.regexp {
			// x without pattern . .*\n, indicated by cmd.re==0
			// X without pattern is all files
			c := cp.nextc()
			if ct.cmdc != 'x' && ct.cmdc != 'X' || (c != ' ' && c != '\t' && c != '\n') {
				cp.skipbl()
				c := cp.getch()
				if c == '\n' || c < 0 {
					return nil, errAddressMissing
				}
				if !okdelim(c) {
					return nil, badDelimiterError(c)
				}
				cmd.re, err = cp.getregexp(c)
				if err != nil {
					return nil, err
				}
				if ct.cmdc == 's' {
					cmd.text = ""
					cmd.text, err = cp.getrhs(c, 's')
					if err != nil {
						return nil, err
					}
					if cp.nextc() == c {
						cp.getch()
						if cp.nextc() == 'g' {
							cmd.flag = cp.getch()
						}
					}

				}
			}
		}
		if ct.addr {
			var err error
			cmd.mtaddr, err = cp.simpleaddr()
			if err != nil {
				return nil, err
			}
			if cmd.mtaddr == nil {
				return nil, errBadAddr
			}
		}
		switch {
		case ct.defcmd != 0:
			if cp.skipbl() == '\n' {
				cp.getch()
				cmd.cmd = newcmd()
				cmd.cmd.cmdc = ct.defcmd
			} else {
				cmd.cmd, err = cp.parse(nest)
				if err != nil {
					return nil, err
				}
				if cmd.cmd == nil {
					panic("defcmd")
				}
			}
		case ct.text:
			cmd.text, err = cp.collecttext()
			if err != nil {
				return nil, err
			}
		case len(ct.token) > 0:
			cmd.text = cp.collecttoken(ct.token)
		default:
			cp.atnl()
		}
	} else {
		switch cmd.cmdc {
		case '{':
			var c, nc *Cmd
			for {
				if cp.skipbl() == '\n' {
					cp.getch()
				}
				nc, err = cp.parse(nest + 1)
				if err != nil {
					return nil, err
				}
				if c != nil {
					c.next = nc
				} else {
					cmd.cmd = nc
				}
				c = nc
				if !(c != nil) {
					break
				}
			}
		case '}':
			cp.atnl()
			if nest == 0 {
				return nil, errLeftBraceMissing
			}
			return nil, nil
		default:
			return nil, invalidCmdError(cmd.cmdc)
		}
	}
	return &cmd, nil
}

func (cp *cmdParser) getregexp(delim rune) (string, error) {
	var c rune

	buf := string("")
	for i := int(0); ; i++ {
		c = cp.getch()
		if c == '\\' {
			if cp.nextc() == delim {
				c = cp.getch()
			} else {
				if cp.nextc() == '\\' {
					buf = buf + string(c)
					c = cp.getch()
				}
			}
		} else {
			if c == delim || c == '\n' {
				break
			}
		}
		buf = buf + string(c)
	}
	if c != delim && c != 0 {
		cp.ungetch()
	}
	if len(buf) > 0 {
		lastpat = buf
	}
	if len(lastpat) == 0 {
		return "", errRegexpMissing
	}
	return lastpat, nil
}

func (cp *cmdParser) simpleaddr() (*Addr, error) {
	var addr Addr

	switch cp.skipbl() {
	case '#':
		addr.typ = cp.getch()
		addr.num = cp.getnum(false)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		addr.typ = 'l'
		addr.num = cp.getnum(false)
	case '/', '?', '"':
		addr.typ = cp.getch()
		var err error
		addr.re, err = cp.getregexp(addr.typ)
		if err != nil {
			return nil, err
		}
	case '.', '$', '+', '-', '\'':
		addr.typ = cp.getch()
	default:
		return nil, nil
	}
	var err error
	addr.next, err = cp.simpleaddr()
	if err != nil {
		return nil, err
	}
	if addr.next != nil {
		switch addr.next.typ {
		case '.', '$', '\'':
			if addr.typ != '"' {
				return nil, errBadAddrSyntax
			}
		case '"':
			return nil, errBadAddrSyntax
		case 'l', '#':
			if addr.typ == '"' {
				break
			}
			fallthrough
		case '/', '?':
			if addr.typ != '+' && addr.typ != '-' {
				// insert the missing '+'
				nap := newaddr()
				nap.typ = '+'
				nap.next = addr.next
				addr.next = nap
			}
		case '+', '-':
			// Do nothing
		default:
			panic("simpleaddr")
		}
	}
	return &addr, nil
}

func (cp *cmdParser) compoundaddr() (*Addr, error) {
	var addr Addr
	var err error

	addr.left, err = cp.simpleaddr()
	if err != nil {
		return nil, err
	}
	addr.typ = cp.skipbl()
	if addr.typ != ',' && addr.typ != ';' {
		return addr.left, nil
	}
	cp.getch()
	addr.next, err = cp.compoundaddr()
	if err != nil {
		return nil, err
	}
	next := addr.next
	if next != nil && (next.typ == ',' || next.typ == ';') && next.left == nil {
		return nil, errBadAddrSyntax
	}
	return &addr, nil
}
