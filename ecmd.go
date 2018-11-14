package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	Glooping int
	nest     int
)

const Enoname = "no file name given"

var (
	addr       Address
	menu       *File
	sel        RangeSet
	curtext    *Text
	collection []rune
	dot        Address
)

func clearcollection() {
	collection = collection[0:0]
}

func resetxec() {
	Glooping = 0
	nest = 0
	clearcollection()
}

func mkaddr(f *File) (a Address) {
	a.r.q0 = f.curtext.q0
	a.r.q1 = f.curtext.q1
	a.f = f
	return a
}

var none = Address{Range{0, 0}, nil}

func cmdexec(t *Text, cp *Cmd) bool {
	w := (*Window)(nil)
	if t != nil {
		w = t.w
	}

	if w == nil && (cp.addr == nil || cp.addr.typ != '"') &&
		utfrune([]rune("bBnqUXY!"), cp.cmdc) == -1 && // Commands that don't need a window
		!(cp.cmdc == 'D' && len(cp.text) > 0) {
		editerror("no current window")
	}
	i := cmdlookup(cp.cmdc) // will be -1 for '{'
	f := (*File)(nil)
	if t != nil && t.w != nil {
		t = &t.w.body
		f = t.file
		f.curtext = t
	}
	if i >= 0 && cmdtab[i].defaddr != aNo {
		ap := cp.addr
		if ap == nil && cp.cmdc != '\n' {
			ap = newaddr()
			cp.addr = ap
			ap.typ = '.'
			if cmdtab[i].defaddr == aAll {
				ap.typ = '*'
			}
		} else if ap != nil && ap.typ == '"' && ap.next == nil && cp.cmdc != '\n' {
			ap.next = newaddr()
			ap.next.typ = '.'
			if cmdtab[i].defaddr == aAll {
				ap.next.typ = '*'
			}
		}
		if cp.addr != nil { // may be false for '\n' (only
			if f != nil {
				dot = mkaddr(f)
				addr = cmdaddress(ap, dot, 0)
			} else { // a "
				addr = cmdaddress(ap, none, 0)
			}
			f = addr.f
			t = f.curtext
		}
	}
	switch cp.cmdc {
	case '{':
		dot = mkaddr(f)
		if cp.addr != nil {
			dot = cmdaddress(cp.addr, dot, 0)
		}
		for cp = cp.cmd; cp != nil; cp = cp.next {
			if dot.r.q1 > t.file.b.Nc() {
				editerror("dot extends past end of buffer during { command")
			}
			t.q0 = dot.r.q0
			t.q1 = dot.r.q1
			cmdexec(t, cp)
		}
		break
	default:
		if i < 0 {
			editerror("unknown command %c in cmdexec", cp.cmdc)
		}
		return (cmdtab[i].fn)(t, cp)
	}
	return true
}

func edittext(w *Window, q int, r []rune) error {
	f := w.body.file
	switch editing {
	case Inactive:
		return fmt.Errorf("permission denied")
	case Inserting:
		f.elog.Insert(q, r)
		return nil
	case Collecting:
		collection = append(collection, r...)
		return nil
	default:
		return fmt.Errorf("unknown state in edittext")
	}
}

// string is known to be NUL-terminated
func filelist(t *Text, r string) string {
	if len(r) == 0 {
		return ""
	}
	r = strings.TrimLeft(r, " \t")
	if len(r) == 0 {
		return ""
	}
	if r[0] != '<' {
		return r
	}
	// use < command to collect text
	clearcollection()
	runpipe(t, '<', []rune(r[1:]), Collecting)
	return string(collection)
}

func a_cmd(t *Text, cp *Cmd) bool {
	return appendx(t.file, cp, addr.r.q1)
}

func b_cmd(t *Text, cp *Cmd) bool {
	f := tofile(cp.text)
	if nest == 0 {
		pfilename(f)
	}
	curtext = f.curtext
	return true
}

func B_cmd(t *Text, cp *Cmd) bool {
	list := filelist(t, cp.text)
	if list == "" {
		editerror(Enoname)
	}
	r := list
	r = strings.TrimLeft(r, " \t")
	if r == "" {
		newx(t, t, nil, false, false, r)
	} else {
		r = wsre.ReplaceAllString(r, " ")
		words := strings.Split(r, " ")
		for _, w := range words {
			newx(t, t, nil, false, false, w)
		}
	}
	clearcollection()
	return true
}

func c_cmd(t *Text, cp *Cmd) bool {
	t.file.elog.Replace(addr.r.q0, addr.r.q1, []rune(cp.text))
	t.q0 = addr.r.q0
	t.q1 = addr.r.q1
	return true
}

func d_cmd(t *Text, cp *Cmd) bool {
	if addr.r.q1 > addr.r.q0 {
		t.file.elog.Delete(addr.r.q0, addr.r.q1)
	}
	t.q0 = addr.r.q0
	t.q1 = addr.r.q0
	return true
}

func D1(t *Text) {
	if len(t.w.body.file.text) > 1 || t.w.Clean(false) {
		t.col.Close(t.w, true)
	}
}

func D_cmd(t *Text, cp *Cmd) bool {
	list := filelist(t, cp.text)
	if list == "" {
		D1(t)
		return true
	}
	dir := dirname(t, nil)
	for _, s := range strings.Fields(list) {
		if !filepath.IsAbs(s) {
			s = filepath.Join(string(dir), s)
		}
		w := lookfile(s)
		if w == nil {
			editerror(fmt.Sprintf("no such file %q", s))
		}
		D1(&w.body)
	}
	clearcollection()
	return true
}

func e_cmd(t *Text, cp *Cmd) bool {
	f := t.file
	q0 := addr.r.q0
	q1 := addr.r.q1
	if cp.cmdc == 'e' {
		if !t.w.Clean(true) {
			editerror("") // Clean generated message already
		}
		q0 = 0
		q1 = f.b.Nc()
	}
	allreplaced := q0 == 0 && q1 == f.b.Nc()
	name := cmdname(f, cp.text, cp.cmdc == 'e')
	if name == "" {
		editerror(Enoname)
	}
	samename := name == t.file.name
	fd, err := os.Open(name)
	if err != nil {
		editerror("can't open %v: %v", name, err)
	}
	defer fd.Close()
	fi, err := fd.Stat()
	if err == nil && fi.IsDir() {
		editerror("%v is a directory", name)
	}
	f.elog.Delete(q0, q1)
	_, nulls, err := f.Load(q1, fd, false)
	if err != nil {
		warning(nil, "Error reading file %v: %v", name, err)
		return false
	}
	if nulls {
		warning(nil, "%v: NUL bytes elided\n", name)
	} else if allreplaced && samename {
		f.editclean = true
	}
	return true
}

func f_cmd(t *Text, cp *Cmd) bool {
	str := ""
	if cp.text == "" {
		str = ""
	} else {
		str = cp.text
	}
	cmdname(t.file, str, true)
	pfilename(t.file)
	return true
}

func g_cmd(t *Text, cp *Cmd) bool {
	if t.file != addr.f {
		warning(nil, "internal error: g_cmd f!=addr.f\n")
		return false
	}
	are, err := rxcompile(cp.re)
	if err != nil {
		editerror("bad regexp in g command")
	}
	sel := are.rxexecute(t, nil, addr.r.q0, addr.r.q1, 1)
	if (len(sel) > 0) != (cp.cmdc == 'v') {
		t.q0 = addr.r.q0
		t.q1 = addr.r.q1
		return cmdexec(t, cp.cmd)
	}
	return true
}

func i_cmd(t *Text, cp *Cmd) bool {
	return appendx(t.file, cp, addr.r.q0)
}

func copyx(f *File, addr2 Address) {
	ni := 0
	buf := make([]rune, RBUFSIZE)
	for p := addr.r.q0; p < addr.r.q1; p += ni {
		ni = addr.r.q1 - p
		if ni > RBUFSIZE {
			ni = RBUFSIZE
		}
		f.b.Read(p, buf[:ni])
		addr2.f.elog.Insert(addr2.r.q1, buf[:ni])
	}
}

func move(f *File, addr2 Address) {
	if addr.f != addr2.f || addr.r.q1 <= addr2.r.q0 {
		f.elog.Delete(addr.r.q0, addr.r.q1)
		copyx(f, addr2)
	} else if addr.r.q0 >= addr2.r.q1 {
		copyx(f, addr2)
		f.elog.Delete(addr.r.q0, addr.r.q1)
	} else if addr.r.q0 == addr2.r.q0 && addr.r.q1 == addr2.r.q1 {
		// move to self; no-op
	} else {
		editerror("move overlaps itself")
	}
}

func m_cmd(t *Text, cp *Cmd) bool {
	dot := mkaddr(t.file)
	addr2 := cmdaddress(cp.mtaddr, dot, 0)
	if cp.cmdc == 'm' {
		move(t.file, addr2)
	} else {
		copyx(t.file, addr2)
	}
	return true
}

func p_cmd(t *Text, cp *Cmd) bool {
	return pdisplay(t.file)
}

func s_cmd(t *Text, cp *Cmd) bool {
	n := cp.num
	op := -1
	are, err := rxcompile(cp.re)
	if err != nil {
		editerror("bad regexp in s command")
	}
	rp := []RangeSet{}
	delta := 0
	didsub := false
	for p1 := addr.r.q0; p1 <= addr.r.q1; {
		if sels := are.rxexecute(t, nil, p1, addr.r.q1, 1); len(sels) > 0 {
			sel = sels[0]
			if sel[0].q0 == sel[0].q1 { // empty match?
				if sel[0].q0 == op {
					p1++
					continue
				}
				p1 = sel[0].q1 + 1
			} else {
				p1 = sel[0].q1
			}
			op = sel[0].q1
			n--
			if n > 0 {
				continue
			}
			rp = append(rp, sel)
		} else {
			break
		}
	}
	rbuf := make([]rune, RBUFSIZE)
	for m := range rp {
		buf := ""
		sel = rp[m]
		for i := 0; i < len(cp.text); i++ {
			c := []rune(cp.text)[i]
			if c == '\\' && i < len(cp.text)-1 {
				i++
				c = []rune(cp.text)[i]
				if '1' <= c && c <= '9' {
					j := c - '0'
					if sel[j].q1-sel[j].q0 > RBUFSIZE {
						editerror("replacement string too long")
					}
					t.file.b.Read(sel[j].q0, rbuf[:sel[j].q1-sel[j].q0])
					for k := 0; k < sel[j].q1-sel[j].q0; k++ {
						buf = buf + string(rbuf[k])
					}
				} else {
					buf += string(c)
				}
			} else if c != '&' {
				buf += string(c)
			} else {
				if sel[0].q1-sel[0].q0 > RBUFSIZE {
					editerror("right hand side too long in substitution")
				}
				t.file.b.Read(sel[0].q0, rbuf[:sel[0].q1-sel[0].q0])
				for k := 0; k < sel[0].q1-sel[0].q0; k++ {
					buf += string(rbuf[k])
				}
			}
		}
		t.file.elog.Replace(sel[0].q0, sel[0].q1, []rune(buf))
		delta -= sel[0].q1 - sel[0].q0
		delta += len([]rune(buf))
		didsub = true
		if cp.flag == 0 {
			break
		}
	}
	if !didsub && nest == 0 {
		editerror("no substitution")
	}
	t.q0 = addr.r.q0
	t.q1 = addr.r.q1
	return true
}

func u_cmd(t *Text, cp *Cmd) bool {
	n := cp.num
	flag := true
	if n < 0 {
		n = -n
		flag = false
	}
	oseq := -1
	for n > 0 && t.file.seq != oseq {
		n--
		oseq = t.file.seq
		undo(t, nil, nil, flag, false, "")
	}
	return true
}

func w_cmd(t *Text, cp *Cmd) bool {
	f := t.file
	if f.seq == seq {
		editerror("can't write file with pending modifications")
	}
	r := cmdname(f, cp.text, false)
	if r == "" {
		editerror("no name specified for 'w' command")
	}
	putfile(f, addr.r.q0, addr.r.q1, r)
	return true
}

func x_cmd(t *Text, cp *Cmd) bool {
	if cp.re != "" {
		looper(t.file, cp, cp.cmdc == 'x')
	} else {
		linelooper(t.file, cp)
	}
	return true
}

func X_cmd(t *Text, cp *Cmd) bool {
	filelooper(cp, cp.cmdc == 'X')
	return true
}

func runpipe(t *Text, cmd rune, cr []rune, state int) {
	var (
		r, s []rune
		dir  string
		w    *Window
		q    *sync.Mutex
	)

	r = skipbl(cr)
	if len(r) == 0 {
		editerror("no command specified for %c", cmd)
	}
	w = nil
	if state == Inserting {
		w = t.w
		t.q0 = addr.r.q0
		t.q1 = addr.r.q1
		if cmd == '<' || cmd == '|' {
			t.file.elog.Delete(t.q0, t.q1)
		}
	}
	s = append([]rune{cmd}, r...)

	dir = ""
	if t != nil {
		dir = t.DirName("")
	}
	if len(dir) == 1 && dir[0] == '.' { // sigh
		dir = dir[0:0]
	}
	editing = state
	if t != nil && t.w != nil {
		t.w.ref.Inc()
	}
	run(w, string(s), dir, true, "", "", true)
	if t != nil && t.w != nil {
		t.w.Unlock()
	}
	row.lk.Unlock()
	<-cedit
	//
	//	 * The editoutlk exists only so that we can tell when
	//	 * the editout file has been closed.  It can get closed *after*
	//	 * the process exits because, since the process cannot be
	//	 * connected directly to editout (no 9P kernel support),
	//	 * the process is actually connected to a pipe to another
	//	 * process (arranged via 9pserve) that reads from the pipe
	//	 * and then writes the data in the pipe to editout using
	//	 * 9P transactions.  This process might still have a couple
	//	 * writes left to copy after the original process has exited.
	//
	if w != nil {
		q = w.editoutlk
	} else {
		q = editoutlk
	}
	q.Lock() // wait for file to close
	q.Unlock()
	row.lk.Lock()
	editing = Inactive
	if t != nil && t.w != nil {
		t.w.Lock('M')
	}
}

func pipe_cmd(t *Text, cp *Cmd) bool {
	runpipe(t, cp.cmdc, []rune(cp.text), Inserting)
	return true
}

func nlcount(t *Text, q0, q1 int) (nl, pnr int) {
	buf := make([]rune, RBUFSIZE)
	i := 0
	nl = 0
	start := q0
	nbuf := 0
	for q0 < q1 {
		if i == nbuf {
			nbuf = q1 - q0
			if nbuf > RBUFSIZE {
				nbuf = RBUFSIZE
			}
			t.file.b.Read(q0, buf[:nbuf])
			i = 0
		}
		if buf[i] == '\n' {
			start = q0 + 1
			nl++
		}
		i++
		q0++
	}
	return nl, q0 - start
}

const (
	PosnLine = iota
	PosnChars
	PosnLineChars
)

func printposn(t *Text, mode int) {
	var l1, l2 int
	if t != nil && t.file != nil && t.file.name != "" {
		warning(nil, "%s:", t.file.name)
	}
	switch mode {
	case PosnChars:
		warning(nil, "#%d", addr.r.q0)
		if addr.r.q1 != addr.r.q0 {
			warning(nil, ",#%d", addr.r.q1)
		}
		warning(nil, "\n")
		return

	case PosnLine:
		l1, _ = nlcount(t, 0, addr.r.q0)
		l1++
		l2, _ = nlcount(t, addr.r.q0, addr.r.q1)
		l2 += l1
		// check if addr ends with '\n'
		if addr.r.q1 > 0 && addr.r.q1 > addr.r.q0 && t.ReadC(addr.r.q1-1) == '\n' {
			l2--
		}
		warning(nil, "%d", l1)
		if l2 != l1 {
			warning(nil, ",%d", l2)
		}
		warning(nil, "\n")
		return

	case PosnLineChars:
		l1, r1 := nlcount(t, 0, addr.r.q0)
		l1++
		l2, r2 := nlcount(t, addr.r.q0, addr.r.q1)
		l2 += l1
		if l2 == l1 {
			r2 += r1
		}
		warning(nil, "%d+#%d", l1, r1)
		if l2 != l1 {
			warning(nil, ",%d+#%d", l2, r2)
		}
		warning(nil, "\n")
		return
	default: // PosnLine
		l1, _ = nlcount(t, 0, addr.r.q0)
		l1++
		l2, _ = nlcount(t, addr.r.q0, addr.r.q1)
		l2 += l1
		// check if addr ends with '\n'
		if addr.r.q1 > 0 && addr.r.q1 > addr.r.q0 && t.ReadC(addr.r.q1-1) == '\n' {
			l2--
		}
		warning(nil, "%d", l1)
		if l2 != l1 {
			warning(nil, ",%d", l2)
		}
		warning(nil, "\n")
		return
	}
}

func eq_cmd(t *Text, cp *Cmd) bool {
	mode := 0
	switch len(cp.text) {
	case 0:
		mode = PosnLine
		break
	case 1:
		if cp.text[0] == '#' {
			mode = PosnChars
			break
		}
		if cp.text[0] == '+' {
			mode = PosnLineChars
			break
		}
	default:
		editerror("newline expected")
	}
	printposn(t, mode)
	return true
}

func nl_cmd(t *Text, cp *Cmd) bool {
	f := t.file
	if cp.addr == nil {
		// First put it on newline boundaries
		a := mkaddr(f)
		addr = lineaddr(0, a, -1)
		a = lineaddr(0, a, 1)
		addr.r.q1 = a.r.q1
		if addr.r.q0 == t.q0 && addr.r.q1 == t.q1 {
			a := mkaddr(f)
			addr = lineaddr(1, a, 1)
		}
	}
	t.Show(addr.r.q0, addr.r.q1, true)
	return true
}

func appendx(f *File, cp *Cmd, p int) bool {
	if len(cp.text) > 0 {
		f.elog.Insert(p, []rune(cp.text))
	}
	f.curtext.q0 = p
	f.curtext.q1 = p
	return true
}

func pdisplay(f *File) bool {
	p1 := addr.r.q0
	p2 := addr.r.q1
	if p2 > f.b.Nc() {
		p2 = f.b.Nc()
	}
	buf := make([]rune, RBUFSIZE)
	for p1 < p2 {
		np := p2 - p1
		if np > RBUFSIZE-1 {
			np = RBUFSIZE - 1
		}
		f.b.Read(p1, buf[:np])
		warning(nil, "%s", string(buf[:np]))
		p1 += np
	}
	f.curtext.q0 = addr.r.q0
	f.curtext.q1 = addr.r.q1
	return true
}

func pfilename(f *File) {
	w := f.curtext.w
	// same check for dirty as in settag, but we know ncache==0
	dirty := !w.isdir && !w.isscratch && f.mod
	dirtychar := ' '
	if dirty {
		dirtychar = '\''
	}
	fc := ' '
	if curtext != nil && curtext.file == f {
		fc = '.'
	}
	warning(nil, "%c%c%c %s\n", dirtychar,
		'+', fc, f.name)
}

func loopcmd(f *File, cp *Cmd, rp []Range) {
	for _, r := range rp {
		f.curtext.q0 = r.q0
		f.curtext.q1 = r.q1
		cmdexec(f.curtext, cp)
	}
}

func looper(f *File, cp *Cmd, isX bool) {
	rp := []Range{}
	tr := Range{}
	r := addr.r
	isY := !isX
	nest++
	are, err := rxcompile(cp.re)
	if err != nil {
		editerror("bad regexp in %c command", cp.cmdc)
	}
	/*if isX */ op := -1 // Not used in the X case.
	if isY {
		op = r.q0
	}
	sels := are.rxexecute(f.curtext, nil, r.q0, r.q1, -1)
	if len(sels) == 0 {
		if isY {
			rp = append(rp, Range{r.q0, r.q1})
		}
	} else {
		for _, s := range sels {
			if isX {
				tr = s[0]
			} else {
				tr.q0 = op
				tr.q1 = s[0].q0
			}
			rp = append(rp, tr)
			op = s[0].q1
		}
		// For the Y case we need to end the set
		if isY {
			tr.q0 = op
			tr.q1 = r.q1
			rp = append(rp, tr)
		}
	}
	loopcmd(f, cp.cmd, rp)
	nest--
}

func linelooper(f *File, cp *Cmd) {
	//	long nrp, p;
	//	Range r, linesel;
	//	Address a, a3;
	rp := []Range{}

	nest++
	r := addr.r
	var a3 Address
	a3.f = f
	a3.r.q0 = r.q0
	a3.r.q1 = r.q0
	a := lineaddr(0, a3, 1)
	linesel := a.r
	for p := r.q0; p < r.q1; p = a3.r.q1 {
		a3.r.q0 = a3.r.q1
		if p != r.q0 || linesel.q1 == p {
			a = lineaddr(1, a3, 1)
			linesel = a.r
		}
		if linesel.q0 >= r.q1 {
			break
		}
		if linesel.q1 >= r.q1 {
			linesel.q1 = r.q1
		}
		if linesel.q1 > linesel.q0 {
			if linesel.q0 >= a3.r.q1 && linesel.q1 > a3.r.q1 {
				a3.r = linesel
				rp = append(rp, linesel)
				continue
			}
		}
		break
	}
	loopcmd(f, cp.cmd, rp)
	nest--
}

type Looper struct {
	cp *Cmd
	XY bool
	w  []*Window
}

var loopstruct Looper // only one; X and Y can't nest

func alllooper(w *Window, lp *Looper) {
	cp := lp.cp
	t := &w.body
	// only use this window if it's the current window for the file  {
	if t.file.curtext != t {
		return
	}
	// no auto-execute on files without names
	if cp.re == "" && t.file.name == "" {
		return
	}
	if cp.re == "" || filematch(t.file, cp.re) == lp.XY {
		lp.w = append(lp.w, w)
	}
}

func alllocker(w *Window, v bool) {
	if v {
		w.ref.Inc()
	} else {
		w.Close()
	}
}

func filelooper(cp *Cmd, XY bool) {
	if Glooping != 0 {
		isX := 'Y'
		if XY {
			isX = 'X'
		}
		editerror("can't nest %c command", isX)
	}
	Glooping++
	nest++

	loopstruct.cp = cp
	loopstruct.XY = XY
	loopstruct.w = []*Window{}
	row.AllWindows(func(w *Window) { alllooper(w, &loopstruct) })
	//	 * add a ref to all windows to keep safe windows accessed by X
	//	 * that would not otherwise have a ref to hold them up during
	//	 * the shenanigans.  note this with globalincref so that any
	//	 * newly created windows start with an extra reference.
	row.AllWindows(func(w *Window) { alllocker(w, true) })
	globalincref = true
	for i := 0; i < len(loopstruct.w); i++ {
		cmdexec(&loopstruct.w[i].body, cp.cmd)
	}
	row.AllWindows(func(w *Window) { alllocker(w, false) })
	globalincref = false
	loopstruct.w = nil

	Glooping--
	nest--
}

// TODO(flux) This actually looks like "find one match after p"
// This is almost certainly broken for ^
func nextmatch(f *File, r string, p int, sign int) {
	are, err := rxcompile(r)
	if err != nil {
		editerror("bad regexp in command address")
	}
	sel = RangeSet{Range{0, 0}}
	if sign >= 0 {
		sels := are.rxexecute(f.curtext, nil, p, 0x7FFFFFFF, 2)
		if len(sels) == 0 {
			editerror("no match for regexp")
		} else {
			sel = sels[0]
		}
		if sel[0].q0 == sel[0].q1 && sel[0].q0 == p {
			if len(sels) == 2 {
				sel = sels[1]
			} else { // wrap around
				p++
				if p > f.b.Nc() {
					p = 0
				}
				sels := are.rxexecute(f.curtext, nil, p, 0x7FFFFFFF, 1)
				if len(sels) == 0 {
					editerror("address")
				} else {
					sel = sels[0]
				}
			}
		}
	} else {
		sel = are.rxbexecute(f.curtext, p, NRange)
		if len(sel) == 0 {
			editerror("no match for regexp")
		}
		if sel[0].q0 == sel[0].q1 && sel[0].q1 == p {
			p--
			if p < 0 {
				p = f.b.Nc()
			}
			sel = are.rxbexecute(f.curtext, p, NRange)
			if len(sel) != 0 {
				editerror("address")
			}
		}
	}
}

func cmdaddress(ap *Addr, a Address, sign int) Address {
	f := a.f
	var a1, a2 Address
	var qbydir int
	for {
		switch ap.typ {
		case 'l':
			a = lineaddr(ap.num, a, sign)
		case '#':
			a = charaddr(ap.num, a, sign)
		case '.':
			a = mkaddr(f)

		case '$':
			a.r.q0 = f.b.Nc()
			a.r.q1 = a.r.q0

		case '\'':
			editerror("can't handle '")
			//			a.r = f.mark;

		case '?':
			sign = -sign
			if sign == 0 {
				sign = -1
			}
			fallthrough
		case '/':
			//sign>=0? a.r.q1 : a.r.q0
			if sign >= 0 {
				qbydir = a.r.q1
			} else {
				qbydir = a.r.q0
			}
			nextmatch(f, ap.re, qbydir, sign)
			a.r = sel[0]

		case '"':
			f = matchfile(ap.re)
			a = mkaddr(f)

		case '*':
			a.r.q0 = 0
			a.r.q1 = f.b.Nc()

		case ',':
			fallthrough
		case ';':
			if ap.left != nil {
				a1 = cmdaddress(ap.left, a, 0)
			} else {
				a1.f = a.f
				a1.r.q0 = 0
				a1.r.q1 = 0
			}
			if ap.typ == ';' {
				f = a1.f
				a = a1
				f.curtext.q0 = a1.r.q0
				f.curtext.q1 = a1.r.q1
			}
			if ap.next != nil {
				a2 = cmdaddress(ap.next, a, 0)
			} else {
				a2.f = a.f
				a2.r.q0 = 0
				a2.r.q1 = f.b.Nc()
			}
			if a1.f != a2.f {
				editerror("addresses in different files")
			}
			a.f = a1.f
			a.r.q0 = a1.r.q0
			a.r.q1 = a2.r.q1
			if a.r.q1 < a.r.q0 {
				editerror("addresses out of order")
			}
			return a

		case '+':
			fallthrough
		case '-':
			sign = 1
			if ap.typ == '-' {
				sign = -1
			}
			if ap.next == nil || ap.next.typ == '+' || ap.next.typ == '-' {
				a = lineaddr(1, a, sign)
			}
			break
		default:
			acmeerror("cmdaddress", nil)
			return a
		}
		ap = ap.next
		if ap == nil {
			break
		}
	}
	return a
}

type Tofile struct {
	f *File
	r string
}

func alltofile(w *Window, tp *Tofile) {
	if tp.f != nil {
		return
	}
	if w.isscratch || w.isdir {
		return
	}
	t := &w.body
	// only use this window if it's the current window for the file  {
	if t.file.curtext != t {
		return
	}
	//	if w.nopen[QWevent] > 0   {
	//		return;
	if tp.r == t.file.name {
		tp.f = t.file
	}
}

func tofile(r string) *File {
	var t Tofile

	t.r = strings.TrimLeft(r, " \t\n")
	t.f = nil
	row.AllWindows(func(w *Window) { alltofile(w, &t) })
	if t.f == nil {
		editerror("no such file\"%v\"", t.r)
	}
	return t.f
}

func allmatchfile(w *Window, tp *Tofile) {
	if w.isscratch || w.isdir {
		return
	}
	t := &w.body
	// only use this window if it's the current window for the file  {
	if t.file.curtext != t {
		return
	}
	//	if w.nopen[QWevent] > 0   {
	//		return;
	if filematch(w.body.file, tp.r) {
		if tp.f != nil {
			editerror("too many files match \"%v\"", tp.r)
		}
		tp.f = w.body.file
	}
}

func matchfile(r string) *File {
	var tf Tofile

	tf.f = nil
	tf.r = r
	row.AllWindows(func(w *Window) { allmatchfile(w, &tf) })

	if tf.f == nil {
		editerror("no file matches \"%v\"", r)
	}
	return tf.f
}

func filematch(f *File, r string) bool {
	// compile expr first so if we get an error, we haven't allocated anything  {
	are, err := rxcompile(r)
	if err != nil {
		editerror("bad regexp in file match")
	}
	w := f.curtext.w
	// same check for dirty as in settag, but we know ncache==0
	dirty := !w.isdir && !w.isscratch && f.mod
	dmark := ' '
	if dirty {
		dmark = '\''
	}
	fmark := ' '
	if curtext != nil && curtext.file == f {
		fmark = '.'
	}
	buf := fmt.Sprintf("%c%c%c %s\n", dmark, '+', fmark, f.name)

	s := are.rxexecute(nil, []rune(buf), 0, len([]rune(buf)), 1)
	return len(s) > 0
}

func charaddr(l int, addr Address, sign int) Address {
	if sign == 0 {
		addr.r.q0 = l
		addr.r.q1 = l
	} else if sign < 0 {
		addr.r.q0 -= l
		addr.r.q1 = addr.r.q0
	} else if sign > 0 {
		addr.r.q1 += l
		addr.r.q0 = addr.r.q1
	}
	if addr.r.q0 < 0 || addr.r.q1 > addr.f.b.Nc() {
		editerror("address out of range")
	}
	return addr
}

func lineaddr(l int, addr Address, sign int) Address {
	var a Address
	f := addr.f
	a.f = f
	n := 0
	p := 0
	if sign >= 0 {
		if l == 0 {
			if sign == 0 || addr.r.q1 == 0 {
				a.r.q0 = 0
				a.r.q1 = 0
				return a
			}
			a.r.q0 = addr.r.q1
			p = addr.r.q1 - 1
		} else {
			if sign == 0 || addr.r.q1 == 0 {
				p = 0
				n = 1
			} else {
				p = addr.r.q1 - 1
				if f.curtext.ReadC(p) == '\n' {
					n = 1
				}
				p++
			}
			for n < l {
				if p >= f.b.Nc() {
					editerror("address out of range")
				}
				if f.curtext.ReadC(p) == '\n' {
					n++
				}
				p++
			}
			a.r.q0 = p
		}
		for p < f.b.Nc() && f.curtext.ReadC(p) != '\n' {
			p++
		}
		a.r.q1 = p
	} else {
		p = addr.r.q0
		if l == 0 {
			a.r.q1 = addr.r.q0
		} else {
			for n = 0; n < l; { // always runs once
				if p == 0 {
					n++
					if n != l {
						editerror("address out of range")
					}
				} else {
					c := f.curtext.ReadC(p - 1)
					n++
					if c != '\n' || n != l {
						p--
					}
				}
			}
			a.r.q1 = p
			if p > 0 {
				p--
			}
		}
		for p > 0 && f.curtext.ReadC(p-1) != '\n' { // lines start after a newline
			p--
		}
		a.r.q0 = p
	}
	return a
}

type Filecheck struct {
	f *File
	r string
}

func allfilecheck(w *Window, fp *Filecheck) {
	f := w.body.file
	if w.body.file == fp.f {
		return
	}
	if fp.r == f.name {
		warning(nil, "warning: duplicate file name \"%s\"\n", fp.r)
	}
}

func cmdname(f *File, str string, set bool) string {
	var fc Filecheck
	r := ""
	s := ""
	if str == "" {
		// no name; use existing
		if f.name == "" {
			return ""
		}
		return f.name
	}
	s = strings.TrimLeft(str, " \t")
	if s == "" {
		goto Return
	}

	if filepath.IsAbs(s) {
		r = s
	} else {
		r = f.curtext.DirName(s)
	}
	fc.f = f
	fc.r = r
	row.AllWindows(func(w *Window) { allfilecheck(w, &fc) })
	if f.name == "" {
		set = true
	}

Return:
	if set && !(r == f.name) {
		f.Mark()
		f.Modded()
		f.curtext.w.SetName(r)
	}
	return r
}
