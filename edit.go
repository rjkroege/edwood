package main

import (
	"fmt"
)

type Addr struct {
	typ  rune /* # (byte addr), l (line addr), / ? . $ + - , ; */
	re   string
	left *Addr /* left side of , and ; */
	num  int
	next *Addr /* or right side of , and ; */
}

type Address struct {
	r Range
	f *File
}

type Cmd struct {
	addr   *Addr  /* address (range of text) */
	re     string /* regular expression for e.g. 'x' */
	cmd    *Cmd   /* target of x, g, {, etc. */
	text   string /* text of a, c, i; rhs of s */
	mtaddr *Addr  /* address for m, t */
	next   *Cmd   /* pointer to next element in braces */
	num    int
	flag   rune /* whatever */
	cmdc   rune /* command character; 'x' etc. */
}

type Cmdtab struct {
	cmdc    rune                  /* command character */
	text    byte                  /* takes a textual argument? */
	regexp  byte                  /* takes a regular expression? */
	addr    byte                  /* takes an address (m or t)? */
	defcmd  rune                  /* default command; 0==>none */
	defaddr Defaddr               /* default address */
	count   int                   /* takes a count e.g. s2/// */
	token   []rune                /* takes text terminated by one of these */
	fn      func(*Text, *Cmd) bool /* function to call with parse tree */
}

const INCR = 25 /* delta when growing list */

type List struct { /* code depends on a long being able to hold a pointer */

	nalloc int
	nused  int

	u interface{}
	/*
		listptr	*void
		void*	*ptr;
		byte*	*ucharptr;
		string	*stringptr;
	*/
}

type Defaddr int

const (
	aNo Defaddr = iota
	aDot
	aAll
)

var (
	linex = []rune("\n")
	wordx = []rune("\t\n")
)

var cmdtab = []Cmdtab{
	/*	cmdc	text	regexp	addr	defcmd	defaddr	count	token	 fn	*/
	{'\n', 0, 0, 0, 0, aDot, 0, nil, nl_cmd},
	{'a', 1, 0, 0, 0, aDot, 0, nil, a_cmd},
	{'b', 0, 0, 0, 0, aNo, 0, linex, b_cmd},
	{'c', 1, 0, 0, 0, aDot, 0, nil, c_cmd},
	{'d', 0, 0, 0, 0, aDot, 0, nil, d_cmd},
	{'e', 0, 0, 0, 0, aNo, 0, wordx, e_cmd},
	{'f', 0, 0, 0, 0, aNo, 0, wordx, f_cmd},
	{'g', 0, 1, 0, 'p', aDot, 0, nil, g_cmd},
	{'i', 1, 0, 0, 0, aDot, 0, nil, i_cmd},
	{'m', 0, 0, 1, 0, aDot, 0, nil, m_cmd},
	{'p', 0, 0, 0, 0, aDot, 0, nil, p_cmd},
	{'r', 0, 0, 0, 0, aDot, 0, wordx, e_cmd},
	{'s', 0, 1, 0, 0, aDot, 1, nil, s_cmd},
	{'t', 0, 0, 1, 0, aDot, 0, nil, m_cmd},
	{'u', 0, 0, 0, 0, aNo, 2, nil, u_cmd},
	{'v', 0, 1, 0, 'p', aDot, 0, nil, g_cmd},
	{'w', 0, 0, 0, 0, aAll, 0, wordx, w_cmd},
	{'x', 0, 1, 0, 'p', aDot, 0, nil, x_cmd},
	{'y', 0, 1, 0, 'p', aDot, 0, nil, x_cmd},
	{'=', 0, 0, 0, 0, aDot, 0, linex, eq_cmd},
	{'B', 0, 0, 0, 0, aNo, 0, linex, B_cmd},
	{'D', 0, 0, 0, 0, aNo, 0, linex, D_cmd},
	{'X', 0, 1, 0, 'f', aNo, 0, nil, X_cmd},
	{'Y', 0, 1, 0, 'f', aNo, 0, nil, X_cmd},
	{'<', 0, 0, 0, 0, aDot, 0, linex, pipe_cmd},
	{'|', 0, 0, 0, 0, aDot, 0, linex, pipe_cmd},
	{'>', 0, 0, 0, 0, aDot, 0, linex, pipe_cmd},
	/* deliberately unimplemented:
	{'k',	0,	0,	0,	0,	aDot,	0,	nil,	k_cmd,},
	{'n',	0,	0,	0,	0,	aNo,	0,	nil,	n_cmd,},
	{'q',	0,	0,	0,	0,	aNo,	0,	nil,	q_cmd,},
	{'!',	0,	0,	0,	0,	aNo,	0,	linex,	plan9_cmd,},
	*/
	//	{0,	0,	0,	0,	0,	0,	0,	0},
}

var (
	cmdstartp      []rune
	cmdp           int
	editerrc       chan error
	endeditthreadc chan struct{}

	lastpat string
	patset  bool

//	curtext	*Text
)

func editthread() {
	endeditthreadc = make(chan struct{})
loop:
	for {
		select {
		case <-endeditthreadc:
			break
		default:
			cmd := parsecmd(0)
			if cmd == nil {
				break loop
			}
			if !cmdexec(curtext, cmd) {
				break loop
			}
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
	var (
		t *Text
		f *File
	)

	t = &w.body
	f = t.file
	if f.curtext != t { /* do curtext only */
		return
	}
	/* TODO(flux): Apply the elog properly
	if f.elog.typ == Null {
		f.elog.Term();
	} else {
	*/
	//	if f.elog.typ != Empty {
	f.elog.Apply(f.text[0])
	if f.editclean {
		f.mod = false
		for _, t := range f.text {
			t.w.dirty = false
		}
	}
	//		}
	//	}

	t.SetSelect(t.q0, t.q1)
	t.ScrDraw()
	w.SetTag()
}

func editerror(format string, args ...interface{}) {
	s := fmt.Errorf(format, args...)
	row.AllWindows(allelogterm) /* truncate the edit logs */
	editerrc <- s
	close(endeditthreadc) // exit.
}

func editcmd(ct *Text, r []rune) {
	var err error

	if len(r) == 0 {
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
	err = <-editerrc
	editing = Inactive
	if err != nil {
		warning(nil, "Edit: %s\n", err)
	}
	/* update everyone whose edit log has data */
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
	var n int
	var sign int
	var c rune

	n = 0
	sign = 1
	if signok > 1 && nextc() == '-' {
		sign = -1
		getch()
	}
	c = nextc()
	if c < '0' || '9' < c { /* no number defaults to 1 */
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

func okdelim(c rune) {
	if c == '\\' || ('a' <= c && c <= 'z' || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')) {
		editerror("bad delimiter %c\n", c)
	}
}

func atnl() {
	var c rune
	cmdskipbl()
	c = getch()
	if c != '\n' {
		editerror("newline expected (saw %C)", c)
	}
}

func getrhs(delim rune, cmd rune) (s string) {
	var c rune

	for {
		c = getch()
		if !((c) > 0 && c != delim && c != '\n') {
			break
		}
		if c == '\\' {
			c = getch()
			if (c) <= 0 {
				panic("bad right hand side")
			}
			if c == '\n' {
				ungetch()
				c = '\\'
			} else {
				if c == 'n' {
					c = '\n'
				} else {
					if c != delim && (cmd == 's' || c != '\\') { /* s does its own */
						s = s + "\\" // TODO(flux): Use a stringbuilder
					}
				}
			}
		}
		s = s + string(c)
	}
	ungetch() /* let client read whether delimiter, '\n' or whatever */
	return
}

func collecttoken(end []rune) string {
	s := ""
	var c rune

	for {
		c = nextc()
		if c == ' ' || c == '\t' {
			break
		}
		s = s + string(getch()) /* blanks significant for getname() */
	}
	for {
		c = getch()
		if !(c > 0 && utfrune(end, c) == 0) {
			break
		}
		s = s + string(c)
	}
	if c != '\n' {
		atnl()
	}
	return s
}

func collecttext() string {
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
				return s
			}
			if !(s[begline] != '.' || s[begline+1] != '\n') {
				break
			}
		}
		s = s[:len(s)-2]
	} else {
		delim = getch()
		okdelim(delim)
		s = getrhs(delim, 'a')
		if nextc() == delim {
			getch()
		}
		atnl()
	}
	return s
}

func cmdlookup(c rune) int {
	for i, cmd := range cmdtab {
		if cmd.cmdc == c {
			return i
		}
	}
	return -1
}

func parsecmd(nest int) *Cmd {
	var i int
	var c rune
	var ct *Cmdtab
	var cp, ncp *Cmd
	var cmd Cmd

	cmd.addr = compoundaddr()
	if cmdskipbl() == -1 {
		return nil
	}
	c = getch()
	if c == -1 {
		return nil
	}
	cmd.cmdc = c
	if cmd.cmdc == 'c' && nextc() == 'd' { /* sleazy two-character case */
		getch() /* the 'd' */
		cmd.cmdc = 'c' | 0x100
	}
	i = cmdlookup(cmd.cmdc)
	if i >= 0 {
		if cmd.cmdc == '\n' {
			goto Return /* let nl_cmd work it all out */
		}
		ct = &cmdtab[i]
		if ct.defaddr == aNo && cmd.addr != nil {
			editerror("command takes no address")
		}
		if ct.count != 0 {
			cmd.num = int(getnum(ct.count))
		}
		if ct.regexp != 0 {
			/* x without pattern . .*\n, indicated by cmd.re==0 */
			/* X without pattern is all files */
			c = nextc()
			if ct.cmdc != 'x' && ct.cmdc != 'X' || (c != ' ' && c != '\t' && c != '\n') {
				cmdskipbl()
				c = getch()
				if c == '\n' || c < 0 {
					editerror("no address")
				}
				okdelim(c)
				cmd.re = getregexp(c)
				if ct.cmdc == 's' {
					cmd.text = ""
					cmd.text = getrhs(c, 's')
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
			cmd.mtaddr = simpleaddr()
		 	if cmd.mtaddr == nil {
				editerror("bad address")
			}
		}
		switch {
		case ct.defcmd != 0:
			if cmdskipbl() == '\n' {
				getch()
				cmd.cmd = newcmd()
				cmd.cmd.cmdc = ct.defcmd
			} else {
				cmd.cmd = parsecmd(nest)
				if cmd.cmd == nil {
					panic("defcmd")
				}
			}
		case ct.text != 0:
			cmd.text = collecttext()
		case ct.token != nil:
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
				ncp = parsecmd(nest + 1)
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
				editerror("right brace with no left brace")
			}
			return nil
		default:
			editerror("unknown command %c", cmd.cmdc)
		}
	}
Return:
	cp = newcmd()
	*cp = cmd
	return cp
}

func getregexp(delim rune) string {
	var buf string
	var i int
	var c rune

	buf = ""
	for i = 0; ; i++ {
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
		editerror("no regular expression defined")
	}
	return lastpat
}

func simpleaddr() *Addr {
	var (
		addr Addr
		nap  *Addr
	)
	switch cmdskipbl() {
	case '#':
		addr.typ = getch()
		addr.num = getnum(1)
	case '0':
		fallthrough
	case '1':
		fallthrough
	case '2':
		fallthrough
	case '3':
		fallthrough
	case '4':
		fallthrough
	case '5':
		fallthrough
	case '6':
		fallthrough
	case '7':
		fallthrough
	case '8':
		fallthrough
	case '9':
		addr.num = (getnum(1))
		addr.typ = 'l'
	case '/':
		fallthrough
	case '?':
		fallthrough
	case '"':
		addr.typ = getch()
		addr.re = getregexp(rune(addr.typ))
	case '.':
		fallthrough
	case '$':
		fallthrough
	case '+':
		fallthrough
	case '-':
		fallthrough
	case '\'':
		addr.typ = getch()
	default:
		return nil
	}
	addr.next = simpleaddr()
	if addr.next != nil {
		switch addr.next.typ {
		case '.':
			fallthrough
		case '$':
			fallthrough
		case '\'':
			if addr.typ != '"' {
				editerror("bad address syntax")
			}
		case '"':
			editerror("bad address syntax")
		case 'l':
			fallthrough
		case '#':
			if addr.typ == '"' {
				break
			}
			fallthrough
		case '/':
			fallthrough
		case '?':
			if addr.typ != '+' && addr.typ != '-' {
				/* insert the missing '+' */
				nap = newaddr()
				nap.typ = '+'
				nap.next = addr.next
				addr.next = nap
			}
			break
		case '+':
			fallthrough
		case '-':
			break
		default:
			panic("simpleaddr")
		}
	}
	return &addr
}

func compoundaddr() *Addr {
	var addr Addr
	var next *Addr

	addr.left = simpleaddr()
	addr.typ = cmdskipbl()
	if addr.typ != ',' && addr.typ != ';' {
		return addr.left
	}
	getch()
	addr.next = compoundaddr()
	next = addr.next
	if next != nil && (next.typ == ',' || next.typ == ';') && next.left == nil {
		editerror("bad address syntax")
	}
	return &addr
}
