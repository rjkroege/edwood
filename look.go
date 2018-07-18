package main

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"9fans.net/go/plumb"
)

var (
	plumbsendfid *client.Fid
	plumbeditfid *client.Fid
	nuntitled    int
)

func plumbthread() {
	// Loop so that if plumber is restarted, acme need not be.
	for {
		var fid *client.Fid
		var err error
		// Connect to plumber.
		for {
			if fid, err = plumb.Open("edit", plan9.OREAD|plan9.OCEXEC); err != nil {
				time.Sleep(2000 * time.Millisecond) // Try every 2 seconds?
			} else {
				break
			}
		}
		plumbeditfid = fid
		plumbsendfid, err = plumb.Open("send", plan9.OWRITE|plan9.OCEXEC)
		if err != nil {
			continue
		}

		plumbeditfidbr := bufio.NewReader(plumbeditfid)
		// Relay messages.
		for {
			var m *plumb.Message = &plumb.Message{}
			err := m.Recv(plumbeditfidbr)
			if err != nil {
				break
			}
			cplumb <- m
		}

		// Lost connection.
		plumbsendfid = nil

		plumbeditfid = nil
		fid.Close()
	}
}

func startplumbing() {
	cplumb = make(chan *plumb.Message)
	go plumbthread()
}

func look3(t *Text, q0 int, q1 int, external bool) {
	var (
		n, c, f int
		ct      *Text
		e       Expand
		r       []rune
		//m *Plumbmsg
		//dir string
		expanded bool
	)

	ct = seltext
	if ct == nil {
		seltext = t
	}
	e, expanded = expand(t, q0, q1)
	if !external && t.w != nil && t.w.nopen[QWevent] > 0 {
		// send alphanumeric expansion to external client
		if expanded == false {
			return
		}
		f = 0
		if (e.at != nil && t.w != nil) || (len(e.name) > 0 && lookfile(e.name) != nil) {
			f = 1 // acme can do it without loading a file
		}
		if q0 != e.q0 || q1 != e.q1 {
			f |= 2 // second (post-expand) message follows
		}
		if len(e.name) > 0 {
			f |= 4 // it's a file name
		}
		c = 'l'
		if t.what == Body {
			c = 'L'
		}
		n = q1 - q0
		if n <= EVENTSIZE {
			r = make([]rune, n)
			t.file.b.Read(q0, r)
			t.w.Eventf("%c%d %d %d %d %v\n", c, q0, q1, f, n, string(r))
		} else {
			t.w.Eventf("%c%d %d %d 0 \n", c, q0, q1, f)
		}
		if q0 == e.q0 && q1 == e.q1 {
			return
		}
		if len(e.name) > 0 {
			n = len(e.name)
			if e.a1 > e.a0 {
				n += 1 + (e.a1 - e.a0)
			}
			r = make([]rune, n)
			copy(r, []rune(e.name))
			if e.a1 > e.a0 {
				nlen := len([]rune(e.name))
				r[nlen] = ':'
				e.at.file.b.Read(e.a0, r[nlen+1:nlen+1+e.a1-e.a0])
			}
		} else {
			n = e.q1 - e.q0
			r = make([]rune, n)
			t.file.b.Read(e.q0, r)
		}
		f &= ^2
		if n <= EVENTSIZE {
			t.w.Eventf("%c%d %d %d %d %v\n", c, e.q0, e.q1, f, n, string(r))
		} else {
			t.w.Eventf("%c%d %d %d 0 \n", c, e.q0, e.q1, f)
		}
		return
	}
	if plumbsendfid != nil {
		// send whitespace-delimited word to plumber
		m := plumb.Message{}
		m.Src = "acme"
		m.Dst = ""
		dir := t.DirName("")
		if dir == "." { // sigh
			dir = ""
		}
		m.Dir = dir
		m.Type = "text"
		m.Attr = nil
		if q1 == q0 {
			if t.q1 > t.q0 && t.q0 <= q0 && q0 <= t.q1 {
				q0 = t.q0
				q1 = t.q1
			} else {
				p := q0
				for q0 > 0 {
					c := t.ReadC(q0 - 1)
					if !(c != ' ' && c != '\t' && c != '\n') {
						break
					}
					q0--
				}
				for q1 < t.file.b.Nc() {
					c := t.ReadC(q1)
					if !(c != ' ' && c != '\t' && c != '\n') {
						break
					}
					q1++
				}
				if q1 == q0 {
					return
				}
				s := fmt.Sprintf("%d", p-q0)
				m.Attr = &plumb.Attribute{Name: "click", Value: s, Next: nil}
			}
		}
		r = make([]rune, q1-q0)
		t.file.b.Read(q0, r[:q1-q0])
		m.Data = []byte(string(r[:q1-q0]))
		if m.Send(plumbsendfid) == nil {
			return
		}
	}
	// interpret alphanumeric string ourselves
	if expanded == false {
		return
	}
	if e.name != "" || e.at != nil {
		e.agetc = func(q int) rune { return e.at.ReadC(q) }
		openfile(t, &e)
	} else {
		if t.w == nil {
			return
		}
		ct = &t.w.body
		if t.w != ct.w {
			ct.w.Lock('M')
			defer ct.w.Unlock()
		}
		if t == ct {
			ct.SetSelect(e.q1, e.q1)
		}
		n = e.q1 - e.q0
		r = make([]rune, n)
		t.file.b.Read(e.q0, r)
		if search(ct, r[:n]) && e.jump {
			row.display.MoveTo(ct.fr.Ptofchar(getP0(ct.fr)).Add(image.Pt(4, ct.fr.DefaultFontHeight()-4)))
		}
	}
}

func plumblook(m *plumb.Message) {
	var e Expand
	var addr string

	if len(m.Data) >= BUFSIZE {
		warning(nil, "insanely long file name (%d bytes) in plumb message (%.32s...)\n", len(m.Data), m.Data)
		return
	}
	e.q0 = 0
	e.q1 = 0
	if len(m.Data) == 0 {
		return
	}
	e.ar = nil
	e.bname = string(m.Data)
	e.name = e.bname
	e.jump = true
	e.a0 = 0
	e.a1 = 0
	addr = findattr(m.Attr, "addr")
	if addr != "" {
		e.ar = []rune(addr)
		e.a1 = len(e.ar)
		e.agetc = func(q int) rune { return e.ar[q] }
	}
	// drawtopwindow(); TODO(flux): Get focus
	openfile(nil, &e)
}

func plumbshow(m *plumb.Message) {
	// drawtopwindow(); TODO(flux): Get focus
	w := makenewwindow(nil)
	name := findattr(m.Attr, "filename")
	if name == "" {
		nuntitled++
		name = fmt.Sprintf("Untitled-%d", nuntitled)
	}
	if name[0] != '/' && m.Dir != "" {
		name = fmt.Sprintf("%s/%s", m.Dir, name)
	}
	name = filepath.Clean(name)
	r, _, _ := cvttorunes([]byte(name), len(name)) // remove nulls
	name = string(r)
	w.SetName(name)
	r, _, _ = cvttorunes(m.Data, len(m.Data))
	w.body.Insert(0, r, true)
	w.body.file.mod = false
	w.dirty = false
	w.SetTag()
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.tag.SetSelect(w.tag.Nc(), w.tag.Nc())
	xfidlog(w, "new")
}

// TODO(flux): This just looks for r in ct; a regexp could do it too,
// using our buffer streaming thing.  Frankly, even scanning the buffer
// using the streamer would probably be easier to read/more idiomatic.
func search(ct *Text, r []rune) bool {
	var (
		n, maxn int
	)
	n = len(r)
	if n == 0 || n > ct.Nc() {
		return false
	}
	if 2*n > RBUFSIZE {
		warning(nil, "string too long\n")
		return false
	}
	maxn = max(2*n, RBUFSIZE)
	s := make([]rune, RBUFSIZE)
	bi := 0 // b indexes s
	nb := 0
	wraparound := false
	q := ct.q1
	for {
		if q >= ct.Nc() {
			q = 0
			wraparound = true
			nb = 0
			//s[bi+nb] = 0; // null terminate
		}
		if nb > 0 {
			ci := runestrchr(s[bi:bi+nb], r[0])
			if ci == -1 {
				q += nb
				nb = 0
				//s[bi+nb] = 0
				if wraparound && q >= ct.q1 {
					break
				}
				continue
			}
			q += (bi + ci - bi)
			nb -= (bi + ci - bi)
			bi = bi + ci
		}
		// reload if buffer covers neither string nor rest of file
		if nb < n && nb != ct.Nc()-q {
			nb = ct.Nc() - q
			if nb >= maxn {
				nb = maxn - 1
			}
			ct.file.b.Read(q, s[:nb])
			bi = 0
		}
		limit := min(len(s), bi+n)
		if runeeq(s[bi:limit], r) {
			if ct.w != nil {
				ct.Show(q, q+n, true)
				ct.w.SetTag()
			} else {
				ct.q0 = q
				ct.q1 = q + n
			}
			seltext = ct
			return true
		}
		nb--
		bi++
		q++
		if wraparound && q >= ct.q1 {
			break
		}
	}
	return false
}

func isfilec(r rune) bool {
	Lx := ".-+/:"
	if isalnum(r) {
		return true
	}
	if runestrchr([]rune(Lx), r) != -1 {
		return true
	}
	return false
}

// Runestr wrapper for cleanname
func cleanrname(rs []rune) []rune {
	s := filepath.Clean(string(rs))
	r, _, _ := cvttorunes([]byte(s), len(s))
	return r
}

/*
func includefile (Rune *dir, Rune *file, int nfile) (Runestr) {
	int m, n;
	char *a;
	Rune *r;
	static Rune Lslash[] = { '/', 0 };

	m = runestrlen(dir);
	a = emalloc((m+1+nfile)*UTFmax+1);
	sprint(a, "%S/%.*S", dir, nfile, file);
	n = access(a);
	free(a);
	if n < 0
		return runestr(nil, 0);
	r = runemalloc(m+1+nfile);
	runemove(r, dir, m);
	runemove(r+m, Lslash, 1);
	runemove(r+m+1, file, nfile);
	free(file);
	return cleanrname(runestr(r, m+1+nfile));
}

static	Rune	*objdir;
*/
func includename(t *Text, r string) string {
	Unimpl()
	return ""
}

/*
	Window *w;
	char buf[128];
	Rune Lsysinclude[] = { '/', 's', 'y', 's', '/', 'i', 'n', 'c', 'l', 'u', 'd', 'e', 0 };
	Rune Lusrinclude[] = { '/', 'u', 's', 'r', '/', 'i', 'n', 'c', 'l', 'u', 'd', 'e', 0 };
	Rune Lusrlocalinclude[] = { '/', 'u', 's', 'r', '/', 'l', 'o', 'c', 'a', 'l',
			'/', 'i', 'n', 'c', 'l', 'u', 'd', 'e', 0 };
	Rune Lusrlocalplan9include[] = { '/', 'u', 's', 'r', '/', 'l', 'o', 'c', 'a', 'l',
			'/', 'p', 'l', 'a', 'n', '9', '/', 'i', 'n', 'c', 'l', 'u', 'd', 'e', 0 };
	Runestr file;
	int i;

	if objdir==nil && objtype!=nil {
		sprint(buf, "/%s/include", objtype);
		objdir = bytetorune(buf, &i);
		objdir = runerealloc(objdir, i+1);
		objdir[i] = '\0';
	}

	w = t.w;
	if n==0 || r[0]=='/' || w==nil
		goto Rescue;
	if n>2 && r[0]=='.' && r[1]=='/'
		goto Rescue;
	file.r = nil;
	file.nr = 0;
	for i=0; i<w.nincl && file.r==nil; i++
		file = includefile(w.incl[i], r, n);

	if file.r == nil
		file = includefile(Lsysinclude, r, n);
	if file.r == nil
		file = includefile(Lusrlocalplan9include, r, n);
	if file.r == nil
		file = includefile(Lusrlocalinclude, r, n);
	if file.r == nil
		file = includefile(Lusrinclude, r, n);
	if file.r==nil && objdir!=nil
		file = includefile(objdir, r, n);
	if file.r == nil
		goto Rescue;
	return file;

    Rescue:
	return runestr(r, n);
}
*/
func dirname(t *Text, r []rune) []rune {
	var (
		b     []rune
		c     rune
		nt    int
		slash int
		tmp   []rune
	)

	b = nil
	if t == nil || t.w == nil {
		goto Rescue
	}
	nt = t.w.tag.file.b.Nc()
	if nt == 0 {
		goto Rescue
	}
	if len(r) >= 1 && r[0] == '/' {
		goto Rescue
	}
	b = make([]rune, nt)
	t.w.tag.file.b.Read(0, b)
	slash = -1
	for m := (0); m < nt; m++ {
		c = b[m]
		if c == '/' {
			slash = int(m)
		}
		if c == ' ' || c == '\t' {
			break
		}
	}
	if slash < 0 {
		goto Rescue
	}
	b = append(b[:len(b)+slash+1], r...)
	return cleanrname(b)

Rescue:
	tmp = r
	if len(r) > 0 {
		return cleanrname(tmp)
	}
	return r
}

func expandfile(t *Text, q0 int, q1 int, e *Expand) (success bool) {
	var colon int
	amax := q1
	if q1 == q0 {
		colon = -1
		for q1 < t.file.b.Nc() {
			c := t.ReadC(q1)
			if isfilec(c) {
				if c == ':' {
					colon = q1
					break
				}
			} else {
				break
			}
			q1++
		}
		for q0 > 0 {
			c := t.ReadC(q0 - 1)
			if isfilec(c) || isaddrc(c) || isregexc(c) {
				if colon < 0 && c == ':' {
					colon = q0 - 1
				}
			} else {
				break
			}
			q0--
		}
		// if it looks like it might begin file: , consume address chars after :
		// otherwise terminate expansion at :
		if colon >= 0 {
			q1 = colon
			if colon < t.file.b.Nc()-1 {
				c := t.ReadC(colon + 1)
				if isaddrc(c) {
					q1 = colon + 1
					for q1 < t.file.b.Nc() {
						c := t.ReadC(q1)
						if isaddrc(c) {
							q1++
						} else {
							break
						}
					}
				}
			}
		}
		if q1 > q0 {
			if colon >= 0 { // stop at white space
				for amax = colon + 1; amax < t.file.b.Nc(); amax++ {
					c := t.ReadC(amax)
					if c == ' ' || c == '\t' || c == '\n' {
						break
					}
				}
			} else {
				amax = t.file.b.Nc()
			}
		}
	}
	amin := amax
	e.q0 = q0
	e.q1 = q1
	n := q1 - q0
	if n == 0 {
		return false
	}
	// see if it's a file name
	rb := make([]rune, n)
	t.file.b.Read(q0, rb[:n])
	r := string(rb[:n])
	// first, does it have bad chars?
	nname := -1
	for i := 0; i < n; i++ {
		c := r[i]
		if c == ':' && nname < 0 {
			cc := t.ReadC(q0 + i + 1)
			if q0+i+1 < t.file.b.Nc() && (i == n-1 || isaddrc(cc)) {
				amin = q0 + i
			} else {
				return false
			}
			nname = i
		}
	}
	if nname == -1 {
		nname = n
	}
	for i := 0; i < nname; i++ {
		if !isfilec(rb[i]) {
			return false
		}
	}
	//* See if it's a file name in <>, and turn that into an include
	//* file name if so.  Should probably do it for "" too, but that's not
	//* restrictive enough syntax and checking for a #include earlier on the
	//* line would be silly.
	// TODO(flux) This is even crazier when working in Go - is this even
	// a feature we want to support?
	if q0 > 0 && t.ReadC(q0-1) == '<' && q1 < t.file.b.Nc() && t.ReadC(q1) == '>' {
		r = includename(t, r)
	} else {
		if amin == q0 {
			e.name = string([]rune(r)[:nname])
			e.at = t
			e.a0 = amin + 1
			_, _, e.a1 = address(true, nil, Range{-1, -1}, Range{0, 0}, e.a0, amax,
				func(q int) rune { return t.ReadC(q) }, false)
			return true
		} else {
			r = t.DirName(string([]rune(r)[:nname]))
		}
	}
	e.bname = r
	// if it's already a window name, it's a file
	w := lookfile(r)
	if w != nil {
		e.name = r
		e.at = t
		e.a0 = amin + 1
		_, _, e.a1 = address(true, nil, Range{-1, -1}, Range{0, 0}, e.a0, amax,
			func(q int) rune { return t.ReadC(q) }, false)
		return true
	}
	// if it's the name of a file, it's a file
	if ismtpt(e.bname) || !access(e.bname) {
		return false
	}
	e.name = r
	e.at = t
	e.a0 = amin + 1
	_, _, e.a1 = address(true, nil, Range{-1, -1}, Range{0, 0}, e.a0, amax,
		func(q int) rune { return t.ReadC(q) }, false)
	return true
}

func access(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func expand(t *Text, q0 int, q1 int) (Expand, bool) {
	e := Expand{}
	e.agetc = func(q int) rune {
		if q < t.Nc() {
			return t.ReadC(q)
		} else {
			return 0
		}
	}

	// if in selection, choose selection
	e.jump = true
	if q1 == q0 && t.inSelection(q0) {
		q0 = t.q0
		q1 = t.q1
		if t.what == Tag {
			e.jump = false
		}
	}
	ok := true
	if ok = expandfile(t, q0, q1, &e); ok {
		return e, true
	}

	if q0 == q1 {
		for q1 < t.file.b.Nc() && isalnum(t.ReadC(q1)) {
			q1++
		}
		for q0 > 0 && isalnum(t.ReadC(q0-1)) {
			q0--
		}
	}
	e.q0 = q0
	e.q1 = q1
	return e, q1 > q0
}

func lookfile(s string) *Window {
	// avoid terminal slash on directories
	s = strings.TrimRight(s, "/")
	for _, c := range row.col {
		for _, w := range c.w {
			k := strings.TrimRight(w.body.file.name, "/")
			if k == s {
				w = w.body.file.curtext.w
				if w.col != nil { // protect against race deleting w
					return w
				}
			}
		}
	}
	return nil
}

/*
func lookid (int id, int dump) (Window*) {
	int i, j;
	Window *w;
	Column *c;

	for j=0; j<row.ncol; j++ {
		c = row.col[j];
		for i=0; i<c.nw; i++ {
			w = c.w[i];
			if dump && w.dumpid == id
				return w;
			if !dump && w.id == id
				return w;
		}
	}
	return nil;
}
*/

func openfile(t *Text, e *Expand) *Window {
	var (
		r     Range
		w, ow *Window
		eval  bool
		rs    string
	)

	r.q0 = 0
	r.q1 = 0
	if e.name == "" {
		w = t.w
		if w == nil {
			return nil
		}
	} else {
		w = lookfile(e.name)
		if w == nil && e.name[0] != '/' {
			//* Unrooted path in new window.
			// * This can happen if we type a pwd-relative path
			//* in the topmost tag or the column tags.
			//* Most of the time plumber takes care of these,
			// * but plumber might not be running or might not
			// * be configured to accept plumbed directories.
			// * Make the name a full path, just like we would if
			// * opening via the plumber.
			rp := fmt.Sprintf("%s/%s", wdir, e.name)
			rs = string(cleanrname([]rune(rp)))
			e.name = rs
			w = lookfile(e.name)
		}
	}
	if w != nil {
		t = &w.body
		if !t.col.safe && t.fr.GetFrameFillStatus().Maxlines == 0 { // window is obscured by full-column window
			t.col.Grow(t.col.w[0], 1)
		}
	} else {
		ow = nil
		if t != nil {
			ow = t.w
		}
		w = makenewwindow(t)
		t = &w.body
		w.SetName(e.name)
		_, err := t.Load(0, e.bname, true)
		if err != nil {
			t.file.unread = false
		}
		t.file.mod = false
		t.w.dirty = false
		t.w.SetTag()
		t.w.tag.SetSelect(t.w.tag.file.b.Nc(), t.w.tag.file.b.Nc())
		if ow != nil {
			for _, inc := range ow.incl {
				w.AddIncl(inc)
			}
			w.autoindent = ow.autoindent
		} else {
			w.autoindent = globalautoindent
		}
		xfidlog(w, "new")
	}
	if e.a1 == e.a0 {
		eval = false
	} else {
		eval = true
		r, eval, _ = address(true, t, Range{-1, -1}, Range{t.q0, t.q1}, e.a0, e.a1, e.agetc, eval)
		if r.q0 > r.q1 {
			eval = false
			warning(nil, "addresses out of order\n")
		}
		if eval == false {
			e.jump = false // don't jump if invalid address
		}
	}
	if eval == false {
		r.q0 = t.q0
		r.q1 = t.q1
	}
	t.Show(r.q0, r.q1, true)
	t.w.SetTag()
	seltext = t
	if e.jump {
		row.display.MoveTo(t.fr.Ptofchar(getP0(t.fr)).Add(image.Pt(4, fontget(tagfont, row.display).Height-4)))
	} else {
		debug.PrintStack()
	}
	return w
}

func newx(et *Text, t *Text, argt *Text, flag1 bool, flag2 bool, arg string) {

	a, _ := getarg(argt, false, true)
	if a != "" {
		newx(et, t, nil, flag1, flag2, a)
		if len(arg) == 0 {
			return
		}
	}
	s := wsre.ReplaceAllString(string(arg), " ")
	filenames := strings.Split(s, " ")
	if len(filenames) == 1 && filenames[0] == "" && et.col != nil {
		w := et.col.Add(nil, nil, -1)
		w.SetTag()
		xfidlog(w, "new")
		return
	}

	for _, f := range filenames {
		fmt.Printf("filename = %#v\n", f)
		rs := et.DirName(f)
		fmt.Printf("rs = %#v\n", rs)
		e := Expand{}
		e.name = rs
		e.bname = string(rs)
		e.jump = true
		fmt.Printf("e = %#v\n", e)
		openfile(et, &e)
	}
}
