package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/util"
)

var (
	Glooping int
	nest     int
)

const Enoname = "no file name given"

var (
	addr       Address
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

func mkaddr(file *file.ObservableEditableBuffer) (a Address) {
	cur := file.GetCurObserver().(*Text)
	a.r.q0 = cur.q0
	a.r.q1 = cur.q1
	a.file = file
	return a
}

var none = Address{Range{0, 0}, nil}

func cmdexec(t *Text, cp *Cmd) bool {
	w := (*Window)(nil)
	if t != nil {
		w = t.w
	}

	if w == nil && (cp.addr == nil || cp.addr.typ != '"') &&
		!strings.ContainsRune("bBnqUXY!", cp.cmdc) && // Commands that don't need a window
		!(cp.cmdc == 'D' && len(cp.text) > 0) {
		editerror("no current window")
	}
	i := cmdlookup(cp.cmdc) // will be -1 for '{'
	file := (*file.ObservableEditableBuffer)(nil)
	if t != nil && t.w != nil {
		t = &t.w.body
		file = t.file
		file.SetCurObserver(t)
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
			if file != nil {
				dot = mkaddr(file)
				addr = cmdaddress(ap, dot, 0)
			} else { // a "
				addr = cmdaddress(ap, none, 0)
			}
			file = addr.file
			t = file.GetCurObserver().(*Text)
		}
	}
	switch cp.cmdc {
	case '{':
		dot = mkaddr(file)
		if cp.addr != nil {
			dot = cmdaddress(cp.addr, dot, 0)
		}
		for cp = cp.cmd; cp != nil; cp = cp.next {
			if dot.r.q1 > t.file.Nr() {
				editerror("dot extends past end of buffer during { command")
			}
			// TODO(rjk): utf8 buffer addressing change.
			t.q0 = dot.r.q0
			t.q1 = dot.r.q1
			cmdexec(t, cp)
		}
	default:
		if i < 0 {
			editerror("unknown command %c in cmdexec", cp.cmdc)
		}
		return (cmdtab[i].fn)(t, cp)
	}
	return true
}

func edittext(w *Window, q int, r []rune) error {
	switch editing {
	case Inactive:
		return ErrPermission
	case Inserting:
		f := w.body.file
		err := f.Elog.Insert(q, r)
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
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
	file := toOEB(cp.text)
	if nest == 0 {
		pfilename(file)
	}
	curtext = file.GetCurObserver().(*Text)
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
	err := t.file.Elog.Replace(addr.r.q0, addr.r.q1, []rune(cp.text))
	if err != nil {
		warning(nil, err.Error()+"\n")
	}
	t.q0 = addr.r.q0
	t.q1 = addr.r.q1
	return true
}

func d_cmd(t *Text, cp *Cmd) bool {
	if addr.r.q1 > addr.r.q0 {
		err := t.file.Elog.Delete(addr.r.q0, addr.r.q1)
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
	}
	t.q0 = addr.r.q0
	t.q1 = addr.r.q0
	return true
}

func D1(t *Text) {
	if t.w.body.file.HasMultipleObservers() || t.w.Clean(false) {
		t.col.Close(t.w, true)
	}
}

func D_cmd(t *Text, cp *Cmd) bool {
	list := filelist(t, cp.text)
	if list == "" {
		D1(t)
		return true
	}
	dir := t.DirName("")
	for _, s := range strings.Fields(list) {
		if !filepath.IsAbs(s) {
			s = filepath.Join(dir, s)
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
	file := t.file
	f := t.file
	q0 := addr.r.q0
	q1 := addr.r.q1
	if cp.cmdc == 'e' {
		if !t.w.Clean(true) {
			editerror("") // Clean generated message already
		}
		q0 = 0
		q1 = f.Nr()
	}
	allreplaced := q0 == 0 && q1 == f.Nr()
	name := cmdname(file, cp.text, cp.cmdc == 'e')
	if name == "" {
		editerror(Enoname)
	}
	samename := name == file.Name()
	fd, err := os.Open(name)
	if err != nil {
		editerror("can't open %v: %v", name, err)
	}
	defer fd.Close()
	fi, err := fd.Stat()
	if err == nil && fi.IsDir() {
		editerror("%v is a directory", name)
	}

	d, err := ioutil.ReadAll(fd)
	if err != nil {
		editerror("%v unreadable", name)
	}
	runes, _, nulls := util.Cvttorunes(d, len(d))
	err = file.Elog.Replace(q0, q1, runes)
	if err != nil {
		warning(nil, err.Error()+"\n")
	}

	if nulls {
		warning(nil, "%v: NUL bytes elided\n", name)
	} else if allreplaced && samename {
		file.EditClean = true
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
	if t.file != addr.file {
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

func copyx(f *file.ObservableEditableBuffer, addr2 Address) {
	ni := 0
	buf := make([]rune, RBUFSIZE)
	for p := addr.r.q0; p < addr.r.q1; p += ni {
		ni = addr.r.q1 - p
		if ni > RBUFSIZE {
			ni = RBUFSIZE
		}
		f.Read(p, buf[:ni])
		err := addr2.file.Elog.Insert(addr2.r.q1, buf[:ni])
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
	}
}

func move(f *file.ObservableEditableBuffer, addr2 Address) {
	if addr.file != addr2.file || addr.r.q1 <= addr2.r.q0 {
		err := f.Elog.Delete(addr.r.q0, addr.r.q1)
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
		copyx(f, addr2)
	} else if addr.r.q0 >= addr2.r.q1 {
		copyx(f, addr2)
		err := f.Elog.Delete(addr.r.q0, addr.r.q1)
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
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
					t.file.Read(sel[j].q0, rbuf[:sel[j].q1-sel[j].q0])
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
				t.file.Read(sel[0].q0, rbuf[:sel[0].q1-sel[0].q0])
				for k := 0; k < sel[0].q1-sel[0].q0; k++ {
					buf += string(rbuf[k])
				}
			}
		}
		err := t.file.Elog.Replace(sel[0].q0, sel[0].q1, []rune(buf))
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
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
	for n > 0 && t.file.Seq() != oseq {
		n--
		oseq = t.file.Seq()
		undo(t, nil, nil, flag, false, "")
	}
	return true
}

func w_cmd(t *Text, cp *Cmd) bool {
	file := t.file
	if file.Seq() == seq {
		editerror("can't write file with pending modifications")
	}
	r := cmdname(t.file, cp.text, false)
	if r == "" {
		editerror("no name specified for 'w' command")
	}
	putfile(file, addr.r.q0, addr.r.q1, r)
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
	filelooper(t, cp, cp.cmdc == 'X')
	return true
}

// Make the run function mockable.
var runfunc func(*Window, string, string, bool, string, string, bool)

func init() {
	runfunc = run
}

func runpipe(t *Text, cmd rune, cr []rune, state int) {
	var (
		r, s []rune
		w    *Window
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
			err := t.file.Elog.Delete(t.q0, t.q1)
			if err != nil {
				warning(nil, err.Error()+"\n")
			}
		}
	}
	s = append([]rune{cmd}, r...)

	dir := t.DirName("") // exec.Cmd.Dir
	editing = state
	if t != nil && t.w != nil {
		t.w.ref.Inc()
	}
	runfunc(w, string(s), dir, true, "", "", true)
	if t != nil && t.w != nil {
		t.w.Unlock()
	}
	row.lk.Unlock()
	<-cedit
	//
	// The editoutlk exists only so that we can tell when
	// the editout file has been closed.  It can get closed *after*
	// the process exits because, since the process cannot be
	// connected directly to editout (no 9P kernel support),
	// the process is actually connected to a pipe to another
	// process (arranged via 9pserve) that reads from the pipe
	// and then writes the data in the pipe to editout using
	// 9P transactions.  This process might still have a couple
	// writes left to copy after the original process has exited.
	//
	q := editoutlk
	if w != nil {
		q = w.editoutlk
	}
	q <- true // wait for file to close
	<-q
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
			t.file.Read(q0, buf[:nbuf])
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
	if t != nil && t.file != nil && t.file.Name() != "" {
		warning(nil, "%s:", t.file.Name())
	}
	switch mode {
	case PosnChars:
		warning(nil, "#%d", addr.r.q0)
		if addr.r.q1 != addr.r.q0 {
			warning(nil, ",#%d", addr.r.q1)
		}
		warning(nil, "\n")
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
	}
}

func eq_cmd(t *Text, cp *Cmd) bool {
	mode := 0
	switch len(cp.text) {
	case 0:
		mode = PosnLine
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
	file := t.file
	if cp.addr == nil {
		// First put it on newline boundaries
		a := mkaddr(file)
		addr = lineaddr(0, a, -1)
		a = lineaddr(0, a, 1)
		addr.r.q1 = a.r.q1
		if addr.r.q0 == t.q0 && addr.r.q1 == t.q1 {
			a := mkaddr(file)
			addr = lineaddr(1, a, 1)
		}
	}
	t.Show(addr.r.q0, addr.r.q1, true)
	return true
}

func appendx(file *file.ObservableEditableBuffer, cp *Cmd, p int) bool {
	if len(cp.text) > 0 {
		err := file.Elog.Insert(p, []rune(cp.text))
		if err != nil {
			warning(nil, err.Error()+"\n")
		}
	}
	cur := file.GetCurObserver().(*Text)
	cur.q0 = p
	cur.q1 = p
	return true
}

func pdisplay(file *file.ObservableEditableBuffer) bool {
	p1 := addr.r.q0
	p2 := addr.r.q1
	if p2 > file.Nr() {
		p2 = file.Nr()
	}
	buf := make([]rune, RBUFSIZE)
	for p1 < p2 {
		np := p2 - p1
		if np > RBUFSIZE-1 {
			np = RBUFSIZE - 1
		}
		file.Read(p1, buf[:np])
		warning(nil, "%s", string(buf[:np]))
		p1 += np
	}
	cur := file.GetCurObserver().(*Text)
	cur.q0 = addr.r.q0
	cur.q1 = addr.r.q1
	return true
}

func pfilename(f *file.ObservableEditableBuffer) {
	dirtychar := ' '
	if f.SaveableAndDirty() {
		dirtychar = '\''
	}
	fc := ' '
	if curtext != nil && curtext.file == f {
		fc = '.'
	}
	warning(nil, "%c%c%c %s\n", dirtychar,
		'+', fc, f.Name())
}

func loopcmd(file *file.ObservableEditableBuffer, cp *Cmd, rp []Range) {
	for _, r := range rp {
		cur := file.GetCurObserver().(*Text)
		cur.q0 = r.q0
		cur.q1 = r.q1
		cmdexec(cur, cp)
	}
}

func looper(file *file.ObservableEditableBuffer, cp *Cmd, isX bool) {
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
	cur := file.GetCurObserver().(*Text)
	sels := are.rxexecute(cur, nil, r.q0, r.q1, -1)
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
	loopcmd(file, cp.cmd, rp)
	nest--
}

func linelooper(file *file.ObservableEditableBuffer, cp *Cmd) {
	//	long nrp, p;
	//	Range r, linesel;
	//	Address a, a3;
	rp := []Range{}

	nest++
	r := addr.r
	var a3 Address
	a3.file = file
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
	loopcmd(file, cp.cmd, rp)
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
	curr := t.file.GetCurObserver()
	if curr != t {
		return
	}
	// no auto-execute on files without names
	if cp.re == "" && t.file.Name() == "" {
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

func filelooper(t *Text, cp *Cmd, XY bool) {
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

	// add a ref to all windows to keep safe windows accessed by X
	// that would not otherwise have a ref to hold them up during
	// the shenanigans.  note this with globalincref so that any
	// newly created windows start with an extra reference.
	row.AllWindows(func(w *Window) { alllocker(w, true) })
	globalincref = true

	// Unlock the window running the X command.
	// We'll need to lock and unlock each target window in turn.
	if t != nil && t.w != nil {
		t.w.Unlock()
	}

	for i := range loopstruct.w {
		targ := &loopstruct.w[i].body
		if targ != nil && targ.w != nil {
			targ.w.Lock(int(cp.cmdc))
		}
		cmdexec(targ, cp.cmd)
		if targ != nil && targ.w != nil {
			targ.w.Unlock()
		}
	}

	if t != nil && t.w != nil {
		t.w.Lock(int(cp.cmdc))
	}

	row.AllWindows(func(w *Window) { alllocker(w, false) })
	globalincref = false
	loopstruct.w = nil

	Glooping--
	nest--
}

// TODO(flux) This actually looks like "find one match after p"
// This is almost certainly broken for ^
func nextmatch(file *file.ObservableEditableBuffer, r string, p int, sign int) {
	are, err := rxcompile(r)
	if err != nil {
		editerror("bad regexp in command address")
	}
	sel = RangeSet{Range{0, 0}}
	cur := file.GetCurObserver().(*Text)
	if sign >= 0 {
		sels := are.rxexecute(cur, nil, p, -1, 2)
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
				if p > file.Nr() {
					p = 0
				}
				sels := are.rxexecute(cur, nil, p, -1, 1)
				if len(sels) == 0 {
					editerror("address")
				} else {
					sel = sels[0]
				}
			}
		}
	} else {
		sel = are.rxbexecute(cur, p, NRange)
		if len(sel) == 0 {
			editerror("no match for regexp")
		}
		if sel[0].q0 == sel[0].q1 && sel[0].q1 == p {
			p--
			if p < 0 {
				p = file.Nr()
			}
			sel = are.rxbexecute(cur, p, NRange)
			if len(sel) != 0 {
				editerror("address")
			}
		}
	}
}

func cmdaddress(ap *Addr, a Address, sign int) Address {
	file := a.file
	var a1, a2 Address
	var qbydir int
	for {
		switch ap.typ {
		case 'l':
			a = lineaddr(ap.num, a, sign)
		case '#':
			a = charaddr(ap.num, a, sign)
		case '.':
			a = mkaddr(file)

		case '$':
			a.r.q0 = file.Nr()
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
			nextmatch(file, ap.re, qbydir, sign)
			a.r = sel[0]

		case '"':
			file = matchfile(ap.re)
			a = mkaddr(file)

		case '*':
			a.r.q0 = 0
			a.r.q1 = file.Nr()

		case ',':
			fallthrough
		case ';':
			if ap.left != nil {
				a1 = cmdaddress(ap.left, a, 0)
			} else {
				a1.file = a.file
				a1.r.q0 = 0
				a1.r.q1 = 0
			}
			if ap.typ == ';' {
				file = a1.file
				a = a1
				cur := file.GetCurObserver().(*Text)
				cur.q0 = a1.r.q0
				cur.q1 = a1.r.q1
			}
			if ap.next != nil {
				a2 = cmdaddress(ap.next, a, 0)
			} else {
				a2.file = a.file
				a2.r.q0 = 0
				a2.r.q1 = file.Nr()
			}
			if a1.file != a2.file {
				editerror("addresses in different files")
			}
			a.file = a1.file
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
		default:
			util.AcmeError("cmdaddress", nil)
			return a
		}
		ap = ap.next
		if ap == nil {
			break
		}
	}
	return a
}

type ToOEB struct {
	oeb *file.ObservableEditableBuffer
	r   string
}

func alltofile(w *Window, tp *ToOEB) {
	if tp.oeb != nil {
		return
	}
	if w.body.file.IsDirOrScratch() {
		return
	}
	t := &w.body
	// only use this window if it's the current window for the file  {
	if t.file.GetCurObserver().(*Text) != t {
		return
	}
	//	if w.nopen[QWevent] > 0   {
	//		return;
	if tp.r == t.file.Name() {
		tp.oeb = t.file
	}
}

func toOEB(r string) *file.ObservableEditableBuffer {
	var t ToOEB

	t.r = strings.TrimLeft(r, " \t\n")
	t.oeb = nil
	row.AllWindows(func(w *Window) { alltofile(w, &t) })
	if t.oeb == nil {
		editerror("no such file\"%v\"", t.r)
	}
	return t.oeb
}

func allmatchfile(w *Window, tp *ToOEB) {
	if w.body.file.IsDirOrScratch() {
		return
	}
	t := &w.body
	// only use this window if it's the current window for the file  {
	if t.file.GetCurObserver().(*Text) != t {
		return
	}
	//	if w.nopen[QWevent] > 0   {
	//		return;
	if filematch(w.body.file, tp.r) {
		if tp.oeb != nil {
			editerror("too many files match \"%v\"", tp.r)
		}
		tp.oeb = w.body.file
	}
}

func matchfile(r string) *file.ObservableEditableBuffer {
	var tf ToOEB

	tf.oeb = nil
	tf.r = r
	row.AllWindows(func(w *Window) { allmatchfile(w, &tf) })

	if tf.oeb == nil {
		editerror("no file matches \"%v\"", r)
	}
	return tf.oeb
}

func filematch(f *file.ObservableEditableBuffer, r string) bool {
	// compile expr first so if we get an error, we haven't allocated anything  {
	are, err := rxcompile(r)
	if err != nil {
		editerror("bad regexp in file match")
	}
	dmark := ' '
	if f.SaveableAndDirty() {
		dmark = '\''
	}
	fmark := ' '
	if curtext != nil && curtext.file == f {
		fmark = '.'
	}
	buf := fmt.Sprintf("%c%c%c %s\n", dmark, '+', fmark, f.Name())

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
	if addr.r.q0 < 0 || addr.r.q1 > addr.file.Nr() {
		editerror("address out of range")
	}
	return addr
}

func lineaddr(l int, addr Address, sign int) Address {
	var a Address
	file := addr.file
	a.file = file
	f := file
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
				if file.ReadC(p) == '\n' {
					n = 1
				}
				p++
			}
			for n < l {
				// TODO(rjk) utf8 buffer issue p
				if p >= file.Size() {
					editerror("address out of range")
				}
				if f.ReadC(p) == '\n' {
					n++
				}
				p++
			}
			a.r.q0 = p
		}
		for p < f.Size() && f.ReadC(p) != '\n' {
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
					c := f.ReadC(p - 1)
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
		for p > 0 && f.ReadC(p-1) != '\n' { // lines start after a newline
			p--
		}
		a.r.q0 = p
	}
	return a
}

type Filecheck struct {
	f *file.ObservableEditableBuffer
	r string
}

func allfilecheck(w *Window, fp *Filecheck) {
	f := w.body.file
	if f == fp.f {
		return
	}
	if fp.r == f.Name() {
		warning(nil, "warning: duplicate file name \"%s\"\n", fp.r)
	}
}

func cmdname(oeb *file.ObservableEditableBuffer, str string, set bool) string {
	var fc Filecheck
	r := ""
	s := ""
	if str == "" {
		// no name; use existing
		if oeb.Name() == "" {
			return ""
		}
		return oeb.Name()
	}
	s = strings.TrimLeft(str, " \t")
	cur := oeb.GetCurObserver().(*Text)
	if s == "" {
		goto Return
	}
	if filepath.IsAbs(s) {
		r = s
	} else {
		r = cur.DirName(s)
	}
	fc.f = oeb
	fc.r = r
	row.AllWindows(func(w *Window) { allfilecheck(w, &fc) })
	if oeb.Name() == "" {
		set = true
	}

Return:
	if set && !(r == oeb.Name()) {
		oeb.Mark(seq)
		oeb.Modded()
		cur.w.SetName(r)
	}
	return r
}
