package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
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
	r Range
	f *File
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
	text    byte                   // takes a textual argument?
	regexp  byte                   // takes a regular expression?
	addr    byte                   // takes an address (m or t)?
	defcmd  rune                   // default command; 0==>none
	defaddr Defaddr                // default address
	count   int                    // takes a count e.g. s2///
	token   string                 // takes text terminated by one of these
	fn      func(*Text, *Cmd) bool // function to call with parse tree
}

type Defaddr int

const (
	aNo Defaddr = iota
	aDot
	aAll
)

const (
	linex = "\n"
	wordx = "\t\n"
)

var cmdtab = []Cmdtab{
	// cmdc	text	regexp	addr	defcmd	defaddr	count	token	 fn
	{'\n', 0, 0, 0, 0, aDot, 0, "", nl_cmd},
	{'a', 1, 0, 0, 0, aDot, 0, "", a_cmd},
	{'b', 0, 0, 0, 0, aNo, 0, linex, b_cmd},
	{'c', 1, 0, 0, 0, aDot, 0, "", c_cmd},
	{'d', 0, 0, 0, 0, aDot, 0, "", d_cmd},
	{'e', 0, 0, 0, 0, aNo, 0, wordx, e_cmd},
	{'f', 0, 0, 0, 0, aNo, 0, wordx, f_cmd},
	{'g', 0, 1, 0, 'p', aDot, 0, "", nil}, // Assingned to g_cmd in init() to avoid initialization loop
	{'i', 1, 0, 0, 0, aDot, 0, "", i_cmd},
	{'m', 0, 0, 1, 0, aDot, 0, "", m_cmd},
	{'p', 0, 0, 0, 0, aDot, 0, "", p_cmd},
	{'r', 0, 0, 0, 0, aDot, 0, wordx, e_cmd},
	{'s', 0, 1, 0, 0, aDot, 1, "", s_cmd},
	{'t', 0, 0, 1, 0, aDot, 0, "", m_cmd},
	{'u', 0, 0, 0, 0, aNo, 2, "", u_cmd},
	{'v', 0, 1, 0, 'p', aDot, 0, "", nil}, // Assingned to g_cmd in init() to avoid initialization loop
	{'w', 0, 0, 0, 0, aAll, 0, wordx, w_cmd},
	{'x', 0, 1, 0, 'p', aDot, 0, "", nil}, // Assingned to x_cmd in init() to avoid initialization loop
	{'y', 0, 1, 0, 'p', aDot, 0, "", nil}, // Assingned to x_cmd in init() to avoid initialization loop
	{'=', 0, 0, 0, 0, aDot, 0, linex, eq_cmd},
	{'B', 0, 0, 0, 0, aNo, 0, linex, B_cmd},
	{'D', 0, 0, 0, 0, aNo, 0, linex, D_cmd},
	{'X', 0, 1, 0, 'f', aNo, 0, "", nil}, // Assingned to X_cmd in init() to avoid initialization loop
	{'Y', 0, 1, 0, 'f', aNo, 0, "", nil}, // Assingned to X_cmd in init() to avoid initialization loop
	{'<', 0, 0, 0, 0, aDot, 0, linex, pipe_cmd},
	{'|', 0, 0, 0, 0, aDot, 0, linex, pipe_cmd},
	{'>', 0, 0, 0, 0, aDot, 0, linex, pipe_cmd},
	/* deliberately unimplemented:
	{'k',	0,	0,	0,	0,	aDot,	0,	"",	k_cmd,},
	{'n',	0,	0,	0,	0,	aNo,	0,	"",	n_cmd,},
	{'q',	0,	0,	0,	0,	aNo,	0,	"",	q_cmd,},
	{'!',	0,	0,	0,	0,	aNo,	0,	linex,	plan9_cmd,},
	*/
	//	{0,	0,	0,	0,	0,	0,	0,	0},
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
	cmdstartp []rune
	cmdp      int
	editerrc  chan error

	lastpat string
	patset  bool

//	curtext	*Text
)

func editthread() {
	for {
		cmd, err := parsecmd(0)
		if err != nil {
			editerror(err.Error())
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
	w.body.file.elog.Term()
}

func alleditinit(w *Window) {
	w.tag.Commit(true)
	w.body.Commit(true)
	w.body.file.editclean = false
}

func allupdate(w *Window) {
	t := &w.body
	f := t.file
	if f.curtext != t { // do curtext only
		return
	}
	if !f.elog.Empty() {
		owner := t.w.owner
		if owner == 0 {
			t.w.owner = 'E'
		}
		f.Mark()
		f.elog.Apply(f.text[0])
		if f.editclean {
			f.Unmodded()
		}

		t.w.owner = owner
	}

	t.SetSelect(t.q0, t.q1)
	t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
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
	cmdstartp = make([]rune, len(r), len(r)+1)
	copy(cmdstartp, r)
	if r[len(r)-1] != '\n' {
		cmdstartp = append(r, '\n')
	}
	cmdp = 0
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
	go editthread()
	err := <-editerrc
	editing = Inactive
	if err != nil {
		warning(nil, "Edit: %s\n", err)
	}
	// update everyone whose edit log has data
	row.AllWindows(allupdate)
}

func getch() rune {
	if cmdp == len(cmdstartp) {
		return -1
	}
	c := cmdstartp[cmdp]
	cmdp++
	return c
}

func nextc() rune {
	if cmdp == len(cmdstartp) {
		return -1
	}
	return cmdstartp[cmdp]
}

func ungetch() {
	cmdp--
	if cmdp < 0 {
		panic("ungetch")
	}
}

func getnum(signok int) int {
	n := int(0)
	sign := int(1)
	if signok > 1 && nextc() == '-' {
		sign = -1
		getch()
	}
	c := nextc()
	if c < '0' || '9' < c { // no number defaults to 1
		return sign
	}

	for {
		c = getch()
		if !('0' <= c && c <= '9') {
			break
		}
		n = n*10 + int(c-'0')
	}
	ungetch()
	return sign * n
}

func cmdskipbl() rune {
	var c rune
	for {
		c = getch()
		if !(c == ' ' || c == '\t') {
			break
		}
	}

	if c >= 0 {
		ungetch()
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

func atnl() {
	cmdskipbl()
	c := getch()
	if c != '\n' {
		debug.PrintStack()
		editerror("newline expected (saw %c)", c)
	}
}

func getrhs(delim rune, cmd rune) (s string, err error) {
	var c rune

	for {
		c = getch()
		if !((c) > 0 && c != delim && c != '\n') {
			break
		}
		if c == '\\' {
			c = getch()
			if (c) <= 0 {
				return "", errBadRHS
			}
			if c == '\n' {
				ungetch()
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
	ungetch() // let client read whether delimiter, '\n' or whatever
	return
}

func collecttoken(end string) string {
	var s strings.Builder
	var c rune

	for {
		c = nextc()
		if c != ' ' && c != '\t' {
			break
		}
		s.WriteRune(getch()) // blanks significant for getname()
	}
	for {
		c = getch()
		if c <= 0 || strings.ContainsRune(end, c) {
			break
		}
		s.WriteRune(c)
	}
	if c != '\n' {
		atnl()
	}
	return s.String()
}

func collecttext() (string, error) {
	var begline, i int
	var c, delim rune

	s := ""
	if cmdskipbl() == '\n' {
		getch()
		i = 0
		for {
			begline = i
			for {
				c = getch()
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
		delim = getch()
		if !okdelim(delim) {
			return "", badDelimiterError(delim)
		}
		var err error
		s, err = getrhs(delim, 'a')
		if err != nil {
			return "", err
		}
		if nextc() == delim {
			getch()
		}
		atnl()
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

func parsecmd(nest int) (*Cmd, error) {
	var cp, ncp *Cmd
	var cmd Cmd
	var err error

	cmd.addr, err = compoundaddr()
	if err != nil {
		return nil, err
	}
	if cmdskipbl() == -1 {
		return nil, nil
	}
	c := getch()
	if c == -1 {
		return nil, nil
	}
	cmd.cmdc = c
	if cmd.cmdc == 'c' && nextc() == 'd' { // sleazy two-character case
		getch() // the 'd'
		cmd.cmdc = 'c' | 0x100
	}
	i := cmdlookup(cmd.cmdc)
	if i >= 0 {
		if cmd.cmdc == '\n' {
			goto Return // let nl_cmd work it all out
		}
		ct := &cmdtab[i]
		if ct.defaddr == aNo && cmd.addr != nil {
			return nil, errAddrNotRequired
		}
		if ct.count != 0 {
			cmd.num = getnum(ct.count)
		}
		if ct.regexp != 0 {
			// x without pattern . .*\n, indicated by cmd.re==0
			// X without pattern is all files
			c := nextc()
			if ct.cmdc != 'x' && ct.cmdc != 'X' || (c != ' ' && c != '\t' && c != '\n') {
				cmdskipbl()
				c := getch()
				if c == '\n' || c < 0 {
					return nil, errAddressMissing
				}
				if !okdelim(c) {
					return nil, badDelimiterError(c)
				}
				cmd.re, err = getregexp(c)
				if err != nil {
					return nil, err
				}
				if ct.cmdc == 's' {
					cmd.text = ""
					cmd.text, err = getrhs(c, 's')
					if err != nil {
						return nil, err
					}
					if nextc() == c {
						getch()
						if nextc() == 'g' {
							cmd.flag = getch()
						}
					}

				}
			}
		}
		if ct.addr != 0 {
			var err error
			cmd.mtaddr, err = simpleaddr()
			if err != nil {
				return nil, err
			}
			if cmd.mtaddr == nil {
				return nil, errBadAddr
			}
		}
		switch {
		case ct.defcmd != 0:
			if cmdskipbl() == '\n' {
				getch()
				cmd.cmd = newcmd()
				cmd.cmd.cmdc = ct.defcmd
			} else {
				cmd.cmd, err = parsecmd(nest)
				if err != nil {
					return nil, err
				}
				if cmd.cmd == nil {
					panic("defcmd")
				}
			}
		case ct.text != 0:
			cmd.text, err = collecttext()
			if err != nil {
				return nil, err
			}
		case len(ct.token) > 0:
			cmd.text = collecttoken(ct.token)
		default:
			atnl()
		}
	} else {
		switch cmd.cmdc {
		case '{':
			cp = nil
			for {
				if cmdskipbl() == '\n' {
					getch()
				}
				ncp, err = parsecmd(nest + 1)
				if err != nil {
					return nil, err
				}
				if cp != nil {
					cp.next = ncp
				} else {
					cmd.cmd = ncp
				}
				cp = ncp
				if !(cp != nil) {
					break
				}
			}
		case '}':
			atnl()
			if nest == 0 {
				return nil, errLeftBraceMissing
			}
			return nil, nil
		default:
			return nil, invalidCmdError(cmd.cmdc)
		}
	}
Return:
	cp = newcmd()
	*cp = cmd
	return cp, nil
}

func getregexp(delim rune) (string, error) {
	var c rune

	buf := string("")
	for i := int(0); ; i++ {
		c = getch()
		if c == '\\' {
			if nextc() == delim {
				c = getch()
			} else {
				if nextc() == '\\' {
					buf = buf + string(c)
					c = getch()
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
		ungetch()
	}
	if len(buf) > 0 {
		patset = true
		lastpat = buf
	}
	if len(lastpat) == 0 {
		return "", errRegexpMissing
	}
	return lastpat, nil
}

func simpleaddr() (*Addr, error) {
	var addr Addr

	switch cmdskipbl() {
	case '#':
		addr.typ = getch()
		addr.num = getnum(1)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		addr.typ = 'l'
		addr.num = getnum(1)
	case '/', '?', '"':
		addr.typ = getch()
		var err error
		addr.re, err = getregexp(addr.typ)
		if err != nil {
			return nil, err
		}
	case '.', '$', '+', '-', '\'':
		addr.typ = getch()
	default:
		return nil, nil
	}
	var err error
	addr.next, err = simpleaddr()
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

func compoundaddr() (*Addr, error) {
	var addr Addr
	var err error

	addr.left, err = simpleaddr()
	if err != nil {
		return nil, err
	}
	addr.typ = cmdskipbl()
	if addr.typ != ',' && addr.typ != ';' {
		return addr.left, nil
	}
	getch()
	addr.next, err = compoundaddr()
	if err != nil {
		return nil, err
	}
	next := addr.next
	if next != nil && (next.typ == ',' || next.typ == ';') && next.left == nil {
		return nil, errBadAddrSyntax
	}
	return &addr, nil
}
