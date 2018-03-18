package main

import (
	"fmt"
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

func mkaddr(a *Address, f *File) {
	a.r.q0 = int(f.curtext.q0)
	a.r.q1 = int(f.curtext.q1)
	a.f = f
}

var none Address = Address{Range{0, 0}, nil}

func cmdexec(t *Text, cp *Cmd) int {
	w := (*Window)(nil)
	if t != nil {
		w = t.w
	}

	if w == nil && (cp.addr == nil || cp.addr.typ != '"' &&
		Buffer([]rune("bBnqUXY!")).Index([]rune{rune(cp.cmdc)}) == -1 && // Commands that don't need a window
		!(cp.cmdc == 'D' && len(cp.text) > 0)) {
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
				mkaddr(&dot, f)
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
		mkaddr(&dot, f)
		if cp.addr != nil {
			dot = cmdaddress(cp.addr, dot, 0)
		}
		for cp = cp.cmd; cp != nil; cp = cp.next {
			if dot.r.q1 > int(t.file.b.nc()) {
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
		i = (cmdtab[i].fn)(t, cp)
		return i
	}
	return 1
}

func edittext(w *Window, q int, r []rune) error {
	var f *File

	f = w.body.file
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
func filelist(t *Text, r []rune) []rune {
	if len(r) == 0 {
		return nil
	}
	r = skipbl(r)
	if r[0] != '<' {
		re := make([]rune, len(r)) // TODO(flux): I think this doesn't need the copy, we don't seem to change the strings.
		copy(re, r)
		return re
	}
	// use < command to collect text
	clearcollection()
	runpipe(t, '<', r[1:], Collecting)
	return collection
}

func a_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*	return append(t.file, cp, addr.r.q1);
}

*/
func b_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	File *f;

	USED(t);
	f = tofile(cp.u.text);
	if nest == 0  {
		pfilename(f);
	curtext = f.curtext;
	return true;
}
*/
func B_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Rune *list, *r, *s;
	int nr;

	list = filelist(t, cp.u.text.r, cp.u.text.n);
	if list == nil  {
		editerror(Enoname);
	r = list;
	nr = runestrlen(r);
	r = skipbl(r, nr, &nr);
	if nr == 0  {
		new(t, t, nil, 0, 0, r, 0);
	else while(nr > 0){
		s = findbl(r, nr, &nr);
		*s = '\0';
		new(t, t, nil, 0, 0, r, runestrlen(r));
		if nr > 0  {
			r = skipbl(s+1, nr-1, &nr);
	}
	clearcollection();
	return true;
}
*/
func c_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	elogreplace(t.file, addr.r.q0, addr.r.q1, cp.u.text.r, cp.u.text.n);
	t.q0 = addr.r.q0;
	t.q1 = addr.r.q1;
	return true;
}
*/
func d_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	USED(cp);
	if addr.r.q1 > addr.r.q0  {
		elogdelete(t.file, addr.r.q0, addr.r.q1);
	t.q0 = addr.r.q0;
	t.q1 = addr.r.q0;
	return true;
}

func D1 (Text *t) () {
	if t.w.body.file.ntext>1 || winclean(t.w, false)  {
		colclose(t.col, t.w, true);
}
*/
func D_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Rune *list, *r, *s, *n;
	int nr, nn;
	Window *w;
	Runestr dir, rs;
	char buf[128];

	list = filelist(t, cp.u.text.r, cp.u.text.n);
	if list == nil {
		D1(t);
		return true;
	}
	dir = dirname(t, nil, 0);
	r = list;
	nr = runestrlen(r);
	r = skipbl(r, nr, &nr);
	do{
		s = findbl(r, nr, &nr);
		*s = '\0';
		// first time through, could be empty string, meaning delete file empty name
		nn = runestrlen(r);
		if r[0]=='/' || nn==0 || dir.nr==0 {
			rs.r = runestrdup(r);
			rs.nr = nn;
		}else{
			n = runemalloc(dir.nr+1+nn);
			runemove(n, dir.r, dir.nr);
			n[dir.nr] = '/';
			runemove(n+dir.nr+1, r, nn);
			rs = cleanrname(runestr(n, dir.nr+1+nn));
		}
		w = lookfile(rs.r, rs.nr);
		if w == nil {
			snprint(buf, sizeof buf, "no such file %.*S", rs.nr, rs.r);
			free(rs.r);
			editerror(buf);
		}
		free(rs.r);
		D1(&w.body);
		if nr > 0  {
			r = skipbl(s+1, nr-1, &nr);
	}while(nr > 0);
	clearcollection();
	free(dir.r);
	return true;
}

static int
readloader(void *v, uint q0, Rune *r, int nr)
{
	if nr > 0  {
		eloginsert(v, q0, r, nr);
	return 0;
}
*/
func e_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Rune *name;
	File *f;
	int i, isdir, q0, q1, fd, nulls, samename, allreplaced;
	char *s, tmp[128];
	Dir *d;

	f = t.file;
	q0 = addr.r.q0;
	q1 = addr.r.q1;
	if cp.cmdc == 'e' {
		if winclean(t.w, true)==false  {
			editerror("");	// winclean generated message already
		q0 = 0;
		q1 = f.b.nc;
	}
	allreplaced = (q0==0 && q1==f.b.nc);
	name = cmdname(f, cp.u.text, cp.cmdc=='e');
	if name == nil  {
		editerror(Enoname);
	i = runestrlen(name);
	samename = runeeq(name, i, t.file.name, t.file.nname);
	s = runetobyte(name, i);
	free(name);
	fd = open(s, OREAD);
	if fd < 0 {
		snprint(tmp, sizeof tmp, "can't open %s: %r", s);
		free(s);
		editerror(tmp);
	}
	d = dirfstat(fd);
	isdir = (d!=nil && (d.qid.typ&QTDIR));
	free(d);
	if isdir {
		close(fd);
		snprint(tmp, sizeof tmp, "%s is a directory", s);
		free(s);
		editerror(tmp);
	}
	elogdelete(f, q0, q1);
	nulls = 0;
	loadfile(fd, q1, &nulls, readloader, f, nil);
	free(s);
	close(fd);
	if nulls  {
		warning(nil, "%s: NUL bytes elided\n", s);
	else if allreplaced && samename  {
		f.editclean = true;
	return true;
}
*/
var Lempty []rune = []rune{0}

func f_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Rune *name;
	String *str;
	String empty;

	if cp.u.text == nil {
		empty.n = 0;
		empty.r = Lempty;
		str = &empty;
	}else
		str = cp.u.text;
	name = cmdname(t.file, str, true);
	free(name);
	pfilename(t.file);
	return true;
}
*/
func g_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	if t.file != addr.f {
		warning(nil, "internal error: g_cmd f!=addr.f\n");
		return false;
	}
	if rxcompile(cp.re.r) == false  {
		editerror("bad regexp in g command");
	if rxexecute(t, nil, addr.r.q0, addr.r.q1, &sel) ^ cp.cmdc=='v'){
		t.q0 = addr.r.q0;
		t.q1 = addr.r.q1;
		return cmdexec(t, cp.u.cmd);
	}
	return true;
}
*/
func i_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	return append(t.file, cp, addr.r.q0);
}

func copy (File *f, Address addr2) () {
	long p;
	int ni;
	Rune *buf;

	buf = fbufalloc();
	for p=addr.r.q0; p<addr.r.q1; p+=ni {
		ni = addr.r.q1-p;
		if ni > RBUFSIZE  {
			ni = RBUFSIZE;
		bufread(&f.b, p, buf, ni);
		eloginsert(addr2.f, addr2.r.q1, buf, ni);
	}
	fbuffree(buf);
}

func move (File *f, Address addr2) () {
	if addr.f!=addr2.f || addr.r.q1<=addr2.r.q0 {
		elogdelete(f, addr.r.q0, addr.r.q1);
		copy(f, addr2);
	}else if addr.r.q0 >= addr2.r.q1 {
		copy(f, addr2);
		elogdelete(f, addr.r.q0, addr.r.q1);
	}else if addr.r.q0==addr2.r.q0 && addr.r.q1==addr2.r.q1 {
		; // move to self; no-op
	}else
		editerror("move overlaps itself");
}
*/
func m_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Address dot, addr2;

	mkaddr(&dot, t.file);
	addr2 = cmdaddress(cp.u.mtaddr, dot, 0);
	if cp.cmdc == 'm'  {
		move(t.file, addr2);
	else
		copy(t.file, addr2);
	return true;
}
*/
func p_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	USED(cp);
	return pdisplay(t.file);
}
*/
func s_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	int i, j, k, c, m, n, nrp, didsub;
	long p1, op, delta;
	String *buf;
	Rangeset *rp;
	char *err;
	Rune *rbuf;

	n = cp.num;
	op= -1;
	if rxcompile(cp.re.r) == false  {
		editerror("bad regexp in s command");
	nrp = 0;
	rp = nil;
	delta = 0;
	didsub = false;
	for p1 = addr.r.q0; p1<=addr.r.q1 && rxexecute(t, nil, p1, addr.r.q1, &sel);  {
		if sel[0].q0 == sel[0].q1 {	// empty match?
			if sel[0].q0 == op {
				p1++;
				continue;
			}
			p1 = sel[0].q1+1;
		}else
			p1 = sel[0].q1;
		op = sel[0].q1;
		if --n>0  {
			continue;
		nrp++;
		rp = erealloc(rp, nrp*sizeof(Rangeset));
		rp[nrp-1] = sel;
	}
	rbuf = fbufalloc();
	buf = allocstring(0);
	for m=0; m<nrp; m++ {
		buf.n = 0;
		buf.r[0] = '\0';
		sel = rp[m];
		for i = 0; i<cp.u.text.n; i++
			if (c = cp.u.text.r[i])=='\\' && i<cp.u.text.n-1 {
				c = cp.u.text.r[++i];
				if '1'<=c && c<='9'  {
					j = c-'0';
					if sel[j].q1-sel[j].q0>RBUFSIZE {
						err = "replacement string too long";
						goto Err;
					}
					bufread(&t.file.b, sel[j].q0, rbuf, sel[j].q1-sel[j].q0);
					for k=0; k<sel[j].q1-sel[j].q0; k++
						Straddc(buf, rbuf[k]);
				}else
				 	Straddc(buf, c);
			}else if c!='&'  {
				Straddc(buf, c);
			else{
				if sel[0].q1-sel[0].q0>RBUFSIZE {
					err = "right hand side too long in substitution";
					goto Err;
				}
				bufread(&t.file.b, sel[0].q0, rbuf, sel[0].q1-sel[0].q0);
				for k=0; k<sel[0].q1-sel[0].q0; k++
					Straddc(buf, rbuf[k]);
			}
		elogreplace(t.file, sel[0].q0, sel[0].q1,  buf.r, buf.n);
		delta -= sel[0].q1-sel[0].q0;
		delta += buf.n;
		didsub = 1;
		if !cp.flag  {
			break;
	}
	free(rp);
	freestring(buf);
	fbuffree(rbuf);
	if !didsub && nest==0  {
		editerror("no substitution");
	t.q0 = addr.r.q0;
	t.q1 = addr.r.q1;
	return true;

Err:
	free(rp);
	freestring(buf);
	fbuffree(rbuf);
	editerror(err);
	return false;
}
*/
func u_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	int n, oseq, flag;

	n = cp.num;
	flag = true;
	if n < 0 {
		n = -n;
		flag = false;
	}
	oseq = -1;
	while(n-.0 && t.file.seq!=oseq){
		oseq = t.file.seq;
		undo(t, nil, nil, flag, 0, nil, 0);
	}
	return true;
}
*/
func w_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Rune *r;
	File *f;

	f = t.file;
	if(f.seq == seq  {
		editerror("can't write file with pending modifications"); {
	r = cmdname(f, cp.u.text, false);
	if r == nil  {
		editerror("no name specified for 'w' command"); {
	putfile(f, addr.r.q0, addr.r.q1, r, runestrlen(r));
	// r is freed by putfile
	return true;
}
*/
func x_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	if cp.re) {
		looper(t.file, cp, cp.cmdc=='x');
	else
		linelooper(t.file, cp);
	return true;
}
*/
func X_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	USED(t);

	filelooper(cp, cp.cmdc=='X');
	return true;
}
*/
func runpipe(t *Text, cmd int, cr []rune, state int) {
	var (
		r, s []rune
		dir  []rune
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
	s = append([]rune{rune(cmd)}, r...)

	dir = nil
	if t != nil {
		dir = dirname(t, nil, 0)
	}
	if len(dir) == 1 && dir[0] == '.' { // sigh
		dir = dir[0:0]
	}
	editing = state
	if t != nil && t.w != nil {
		t.w.ref.Inc()
	}
	run(w, s, dir, true, nil, nil, true)
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

func pipe_cmd(t *Text, cp *Cmd) int {
	runpipe(t, cp.cmdc, cp.text, Inserting)
	return 1
}

/*
func nlcount (Text *t, long q0, long q1, long *pnr) (long) {
	long nl, start;
	Rune *buf;
	int i, nbuf;

	buf = fbufalloc();
	nbuf = 0;
	i = nl = 0;
	start = q0;
	while(q0 < q1){
		if i == nbuf {
			nbuf = q1-q0;
			if nbuf > RBUFSIZE  {
				nbuf = RBUFSIZE;
			bufread(&t.file.b, q0, buf, nbuf);
			i = 0;
		}
		if buf[i++] == '\n'  {
			start = q0+1;
			nl++;
		}
		q0++;
	}
	fbuffree(buf);
	if pnr != nil  {
		*pnr = q0 - start;
	return nl;
}

enum {
	PosnLine = 0,
	PosnChars = 1,
	PosnLineChars = 2,
};

func printposn (Text *t, int mode) () {
	long l1, l2, r1, r2;

	if (t != nil && t.file != nil && t.file.name != nil) {
		warning(nil, "%.*S:", t.file.nname, t.file.name);

	switch(mode) {
	case PosnChars:
		warning(nil, "#%d", addr.r.q0);
		if addr.r.q1 != addr.r.q0  {
			warning(nil, ",#%d", addr.r.q1);
		warning(nil, "\n");
		return;

	default:
	case PosnLine:
		l1 = 1+nlcount(t, 0, addr.r.q0, nil);
		l2 = l1+nlcount(t, addr.r.q0, addr.r.q1, nil);
		// check if addr ends with '\n'  {
		if addr.r.q1>0 && addr.r.q1>addr.r.q0 && textreadc(t, addr.r.q1-1)=='\n'  {
			--l2;
		warning(nil, "%lud", l1);
		if l2 != l1  {
			warning(nil, ",%lud", l2);
		warning(nil, "\n");
		return;

	case PosnLineChars:
		l1 = 1+nlcount(t, 0, addr.r.q0, &r1);
		l2 = l1+nlcount(t, addr.r.q0, addr.r.q1, &r2);
		if l2 == l1  {
			r2 += r1;
		warning(nil, "%lud+#%d", l1, r1);
		if l2 != l1  {
			warning(nil, ",%lud+#%d", l2, r2);
		warning(nil, "\n");
		return;
	}
}
*/
func eq_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	int mode;

	switch(cp.u.text.n){
	case 0:
		mode = PosnLine;
		break;
	case 1:
		if cp.u.text.r[0] == '#' {
			mode = PosnChars;
			break;
		}
		if cp.u.text.r[0] == '+' {
			mode = PosnLineChars;
			break;
		}
	default:
		SET(mode);
		editerror("newline expected");
	}
	printposn(t, mode);
	return true;
}
*/
func nl_cmd(t *Text, cp *Cmd) int {
	Unimpl()
	return 0
}

/*
	Address a;
	File *f;

	f = t.file;
	if(cp.addr == 0 {
		// First put it on newline boundaries
		mkaddr(&a, f);
		addr = lineaddr(0, a, -1);
		a = lineaddr(0, a, 1);
		addr.r.q1 = a.r.q1;
		if addr.r.q0==t.q0 && addr.r.q1==t.q1 {
			mkaddr(&a, f);
			addr = lineaddr(1, a, 1);
		}
	}
	textshow(t, addr.r.q0, addr.r.q1, 1);
	return true;
}

func append (File *f, Cmd *cp, long p) (int) {
	if cp.u.text.n > 0  {
		eloginsert(f, p, cp.u.text.r, cp.u.text.n);
	f.curtext.q0 = p;
	f.curtext.q1 = p;
	return true;
}

func pdisplay (File *f) (int) {
	long p1, p2;
	int np;
	Rune *buf;

	p1 = addr.r.q0;
	p2 = addr.r.q1;
	if p2 > f.b.nc) {
		p2 = f.b.nc;
	buf = fbufalloc();
	while(p1 < p2){
		np = p2-p1;
		if np>RBUFSIZE-1  {
			np = RBUFSIZE-1;
		bufread(&f.b, p1, buf, np);
		buf[np] = '\0';
		warning(nil, "%S", buf);
		p1 += np;
	}
	fbuffree(buf);
	f.curtext.q0 = addr.r.q0;
	f.curtext.q1 = addr.r.q1;
	return true;
}

func pfilename (File *f) () {
	int dirty;
	Window *w;

	w = f.curtext.w;
	// same check for dirty as in settag, but we know ncache==0
	dirty = !w.isdir && !w.isscratch && f.mod;
	warning(nil, "%c%c%c %.*S\n", " '"[dirty],
		'+', " ."[curtext!=nil && curtext.file==f], f.nname, f.name);
}

func loopcmd (File *f, Cmd *cp, Range *rp, long nrp) () {
	long i;

	for i=0; i<nrp; i++ {
		f.curtext.q0 = rp[i].q0;
		f.curtext.q1 = rp[i].q1;
		cmdexec(f.curtext, cp);
	}
}

func looper (File *f, Cmd *cp, int xy) () {
	long p, op, nrp;
	Range r, tr;
	Range *rp;

	r = addr.r;
	op= xy? -1 : r.q0;
	nest++;
	if rxcompile(cp.re.r) == false  {
		editerror("bad regexp in %c command", cp.cmdc);
	}
	nrp = 0;
	rp = nil;
	for p = r.q0; p<=r.q1;  {
		if !rxexecute(f.curtext, nil, p, r.q1, &sel) { // no match, but y should still run
			if xy || op>r.q1  {
				break;
			tr.q0 = op, tr.q1 = r.q1;
			p = r.q1+1;	// exit next loop
		}else{
			if sel[0].q0==sel[0].q1 {	// empty match?
				if sel[0].q0==op {
					p++;
					continue;
				}
				p = sel[0].q1+1;
			}else
				p = sel[0].q1;
			if xy  {
				tr = sel[0];
			else
				tr.q0 = op, tr.q1 = sel[0].q0;
		}
		op = sel[0].q1;
		nrp++;
		rp = erealloc(rp, nrp*sizeof(Range));
		rp[nrp-1] = tr;
	}
	loopcmd(f, cp.u.cmd, rp, nrp);
	free(rp);
	--nest;
}

func linelooper (File *f, Cmd *cp) () {
	long nrp, p;
	Range r, linesel;
	Address a, a3;
	Range *rp;

	nest++;
	nrp = 0;
	rp = nil;
	r = addr.r;
	a3.f = f;
	a3.r.q0 = a3.r.q1 = r.q0;
	a = lineaddr(0, a3, 1);
	linesel = a.r;
	for p = r.q0; p<r.q1; p = a3.r.q1 {
		a3.r.q0 = a3.r.q1;
		if p!=r.q0 || linesel.q1==p {
			a = lineaddr(1, a3, 1);
			linesel = a.r;
		}
		if linesel.q0 >= r.q1  {
			break;
		if linesel.q1 >= r.q1  {
			linesel.q1 = r.q1;
		if linesel.q1 > linesel.q0  {
			if linesel.q0>=a3.r.q1 && linesel.q1>a3.r.q1 {
				a3.r = linesel;
				nrp++;
				rp = erealloc(rp, nrp*sizeof(Range));
				rp[nrp-1] = linesel;
				continue;
			}
		break;
	}
	loopcmd(f, cp.u.cmd, rp, nrp);
	free(rp);
	--nest;
}

struct Looper
{
	Cmd *cp;
	int	XY;
	Window	**w;
	int	nw;
} loopstruct;	// only one; X and Y can't nest

func alllooper (Window *w, void *v) () {
	Text *t;
	struct Looper *lp;
	Cmd *cp;

	lp = v;
	cp = lp.cp;
//	if(w.isscratch || w.isdir   {
//		return;
	t = &w.body;
	// only use this window if it's the current window for the file  {
	if t.file.curtext != t  {
		return;
//	if w.nopen[QWevent] > 0   {
//		return;
	// no auto-execute on files without names
	if cp.re==nil && t.file.nname==0  {
		return;
	if cp.re==nil || filematch(t.file, cp.re)==lp.XY){
		lp.w = erealloc(lp.w, (lp.nw+1)*sizeof(Window*));
		lp.w[lp.nw++] = w;
	}
}

func alllocker (Window *w, void *v) () {
	if v  {
		incref(&w.ref);
	else
		winclose(w);
}

func filelooper (Cmd *cp, int XY) () {
	int i;

	if Glooping++  {
		editerror("can't nest %c command", "YX"[XY]);
	nest++;

	loopstruct.cp = cp;
	loopstruct.XY = XY;
	if loopstruct.w 	// error'ed out last time  {
		free(loopstruct.w);
	loopstruct.w = nil;
	loopstruct.nw = 0;
	allwindows(alllooper, &loopstruct);
	//	 * add a ref to all windows to keep safe windows accessed by X
	//	 * that would not otherwise have a ref to hold them up during
	//	 * the shenanigans.  note this with globalincref so that any
	//	 * newly created windows start with an extra reference.
	allwindows(alllocker, (void*)1);
	globalincref = 1;
	for i=0; i<loopstruct.nw; i++
		cmdexec(&loopstruct.w[i].body, cp.u.cmd);
	allwindows(alllocker, (void*)0);
	globalincref = 0;
	free(loopstruct.w);
	loopstruct.w = nil;

	--Glooping;
	--nest;
}
*/
func nextmatch(f *File, r []rune, p int, sign int) {
	are, err := rxcompile(r)
	if err != nil {
		editerror("bad regexp in command address")
	}
	if sign >= 0 {
		sel = are.rxexecute(f.curtext, nil, p, 0x7FFFFFFF, NRange)
		if len(sel) == 0 {
			editerror("no match for regexp")
		}
		if sel[0].q0 == sel[0].q1 && sel[0].q0 == p {
			p++
			if p > f.b.nc() {
				p = 0
			}
			sel = are.rxexecute(f.curtext, nil, p, 0x7FFFFFFF, NRange)
			if len(sel) == 0 {
				editerror("address")
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
				p = f.b.nc()
			}
			sel = are.rxbexecute(f.curtext, p, NRange)
			if len(sel) != 0 {
				editerror("address")
			}
		}
	}
}

/*
File	*matchfile(String*);
Address	charaddr(long, Address, int);
Address	lineaddr(long, Address, int);
*/
func cmdaddress(ap *Addr, a Address, sign int) Address {
	Unimpl()
	return Address{Range{0, 0}, nil}
}

/*	File *f = a.f;
	Address a1, a2;

	do{
		switch(ap.typ){
		case 'l':
		case '#':
			a = (*(ap.typ=='#'?charaddr:lineaddr))(ap.num, a, sign);
			break;

		case '.':
			mkaddr(&a, f);
			break;

		case '$':
			a.r.q0 = a.r.q1 = f.b.nc;
			break;

		case '\'':
editerror("can't handle '");
//			a.r = f.mark;
			break;

		case '?':
			sign = -sign;
			if sign == 0  {
				sign = -1;
			// fall through
		case '/':
			nextmatch(f, ap.u.re, sign>=0? a.r.q1 : a.r.q0, sign);
			a.r = sel[0];
			break;

		case '"':
			f = matchfile(ap.u.re);
			mkaddr(&a, f);
			break;

		case '*':
			a.r.q0 = 0, a.r.q1 = f.b.nc;
			return a;

		case ',':
		case ';':
			if ap.u.left  {
				a1 = cmdaddress(ap.u.left, a, 0);
			else
				a1.f = a.f, a1.r.q0 = a1.r.q1 = 0;
			if ap.typ == ';' {
				f = a1.f;
				a = a1;
				f.curtext.q0 = a1.r.q0;
				f.curtext.q1 = a1.r.q1;
			}
			if ap.next  {
				a2 = cmdaddress(ap.next, a, 0);
			else
				a2.f = a.f, a2.r.q0 = a2.r.q1 = f.b.nc;
			if a1.f != a2.f  {
				editerror("addresses in different files"); {
			a.f = a1.f, a.r.q0 = a1.r.q0, a.r.q1 = a2.r.q1;
			if a.r.q1 < a.r.q0  {
				editerror("addresses out of order");
			return a;

		case '+':
		case '-':
			sign = 1;
			if ap.typ == '-'  {
				sign = -1;
			if ap.next==0 || ap.next.typ=='+' || ap.next.typ=='-'  {
				a = lineaddr(1L, a, sign);
			break;
		default:
			error("cmdaddress");
			return a;
		}
	}while(ap = ap.next);	// assign =
	return a;
}

struct Tofile{
	File		*f;
	String	*r;
};

func alltofile (Window *w, void *v) () {
	Text *t;
	struct Tofile *tp;

	tp = v;
	if tp.f != nil  {
		return;
	if w.isscratch || w.isdir  {
		return;
	t = &w.body;
	// only use this window if it's the current window for the file  {
	if t.file.curtext != t  {
		return;
//	if w.nopen[QWevent] > 0   {
//		return;
	if runeeq(tp.r.r, tp.r.n, t.file.name, t.file.nname)) {
		tp.f = t.file;
}

File*
tofile(String *r)
{
	struct Tofile t;
	String rr;

	rr.r = skipbl(r.r, r.n, &rr.n);
	t.f = nil;
	t.r = &rr;
	allwindows(alltofile, &t);
	if t.f == nil  {
		editerror("no such file\"%S\"", rr.r);
	return t.f;
}

func allmatchfile (Window *w, void *v) () {
	struct Tofile *tp;
	Text *t;

	tp = v;
	if w.isscratch || w.isdir  {
		return;
	t = &w.body;
	// only use this window if it's the current window for the file  {
	if t.file.curtext != t  {
		return;
//	if w.nopen[QWevent] > 0   {
//		return;
	if filematch(w.body.file, tp.r)){
		if(tp.f != nil  {
			editerror("too many files match \"%S\"", tp.r.r);
		tp.f = w.body.file;
	}
}

File*
matchfile(String *r)
{
	struct Tofile tf;

	tf.f = nil;
	tf.r = r;
	allwindows(allmatchfile, &tf);

	if tf.f == nil  {
		editerror("no file matches \"%S\"", r.r);
	return tf.f;
}

func filematch (File *f, String *r) (int) {
	char *buf;
	Rune *rbuf;
	Window *w;
	int match, i, dirty;
	Rangeset s;

	// compile expr first so if we get an error, we haven't allocated anything  {
	if rxcompile(r.r) == false  {
		editerror("bad regexp in file match");
	buf = fbufalloc();
	w = f.curtext.w;
	// same check for dirty as in settag, but we know ncache==0
	dirty = !w.isdir && !w.isscratch && f.mod;
	snprint(buf, BUFSIZE, "%c%c%c %.*S\n", " '"[dirty],
		'+', " ."[curtext!=nil && curtext.file==f], f.nname, f.name);
	rbuf = bytetorune(buf, &i);
	fbuffree(buf);
	match = rxexecute(nil, rbuf, 0, i, &s);
	free(rbuf);
	return match;
}

func charaddr (long l, Address addr, int sign) (Address) {
	if sign == 0  {
		addr.r.q0 = addr.r.q1 = l;
	else if sign < 0  {
		addr.r.q1 = addr.r.q0 -= l;
	else if sign > 0  {
		addr.r.q0 = addr.r.q1 += l;
	if addr.r.q0<0 || addr.r.q1>addr.f.b.nc  {
		editerror("address out of range");
	return addr;
}

func lineaddr (long l, Address addr, int sign) (Address) {
	int n;
	int c;
	File *f = addr.f;
	Address a;
	long p;

	a.f = f;
	if sign >= 0 {
		if l == 0 {
			if sign==0 || addr.r.q1==0 {
				a.r.q0 = a.r.q1 = 0;
				return a;
			}
			a.r.q0 = addr.r.q1;
			p = addr.r.q1-1;
		}else{
			if sign==0 || addr.r.q1==0 {
				p = 0;
				n = 1;
			}else{
				p = addr.r.q1-1;
				n = textreadc(f.curtext, p++)=='\n';
			}
			while(n < l){
				if p >= f.b.nc  {
					editerror("address out of range");
				if textreadc(f.curtext, p++) == '\n'  {
					n++;
			}
			a.r.q0 = p;
		}
		while(p < f.b.nc && textreadc(f.curtext, p++)!='\n')
			;
		a.r.q1 = p;
	}else{
		p = addr.r.q0;
		if l == 0  {
			a.r.q1 = addr.r.q0;
		else{
			for n = 0; n<l;  {	// always runs once
				if p == 0 {
					if ++n != l  {
						editerror("address out of range");
				}else{
					c = textreadc(f.curtext, p-1);
					if c != '\n' || ++n != l  {
						p--;
				}
			}
			a.r.q1 = p;
			if p > 0  {
				p--;
		}
		while(p > 0 && textreadc(f.curtext, p-1)!='\n')	// lines start after a newline
			p--;
		a.r.q0 = p;
	}
	return a;
}

struct Filecheck
{
	File	*f;
	Rune	*r;
	int nr;
};

func allfilecheck (Window *w, void *v) () {
	struct Filecheck *fp;
	File *f;

	fp = v;
	f = w.body.file;
	if w.body.file == fp.f  {
		return;
	if runeeq(fp.r, fp.nr, f.name, f.nname)  {
		warning(nil, "warning: duplicate file name \"%.*S\"\n", fp.nr, fp.r);
}

Rune*
cmdname(File *f, String *str, int set)
{
	Rune *r, *s;
	int n;
	struct Filecheck fc;
	Runestr newname;

	r = nil;
	n = str.n;
	s = str.r;
	if n == 0 {
		// no name; use existing
		if f.nname == 0  {
			return nil;
		r = runemalloc(f.nname+1);
		runemove(r, f.name, f.nname);
		return r;
	}
	s = skipbl(s, n, &n);
	if n == 0  {
		goto Return;

	if s[0] == '/' {
		r = runemalloc(n+1);
		runemove(r, s, n);
	}else{
		newname = dirname(f.curtext, runestrdup(s), n);
		n = newname.nr;
		r = runemalloc(n+1);	// NUL terminate
		runemove(r, newname.r, n);
		free(newname.r);
	}
	fc.f = f;
	fc.r = r;
	fc.nr = n;
	allwindows(allfilecheck, &fc);
	if f.nname == 0  {
		set = true;

    Return:
	if set && !runeeq(r, n, f.name, f.nname) {
		filemark(f);
		f.mod = TRUE;
		f.curtext.w.dirty = true;
		winsetname(f.curtext.w, r, n);
	}
	return r;
}
*/
