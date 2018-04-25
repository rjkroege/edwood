package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/rjkroege/edwood/frame"
)

type Exectab struct {
	name  string
	fn    func(t0, t1, t2 *Text, b0, b1 bool, arg string)
	mark  bool
	flag1 bool
	flag2 bool
}

var exectab = []Exectab{
	//	{ "Abort",		doabort,	false,	true /*unused*/,		true /*unused*/,		},
	{"Cut", cut, true, true, true},
	{"Del", del, false, false, true /*unused*/},
	{"Delcol", delcol, false, true /*unused*/, true /*unused*/},
	{"Delete", del, false, true, true /*unused*/},
	{"Dump", dump, false, true, true /*unused*/},
	{ "Edit", edit,	false, true /*unused*/,		true /*unused*/		},
	{"Exit", xexit, false, true /*unused*/, true /*unused*/},
	{"Font", fontx, false, true /*unused*/, true /*unused*/},
	{"Get", get, false, true, true /*unused*/},
	{ "ID", id, false,	true /*unused*/, true /*unused*/		},
	//	{ "Incl",		incl,		false,	true /*unused*/,		true /*unused*/		},
	{"Indent", indent, false, true /*unused*/, true /*unused*/},
	//	{ "Kill",		xkill,		false,	true /*unused*/,		true /*unused*/		},
	{"Load", dump, false, false, true /*unused*/},
	//	{ "Local",		local,	false,	true /*unused*/,		true /*unused*/		},
	{"Look", look, false, true /*unused*/, true /*unused*/},
	{"New", newx, false, true /*unused*/, true /*unused*/},
	{"Newcol", newcol, false, true /*unused*/, true /*unused*/},
	{"Paste", paste, true, true, true /*unused*/},
	{"Put", put, false, true /*unused*/, true /*unused*/},
	{ "Putall",		putall,	false,	true /*unused*/,		true /*unused*/		},
	{"Redo", undo, false, false, true /*unused*/},
	{"Send", sendx, true, true /*unused*/, true /*unused*/},
	{"Snarf", cut, false, true, false},
	{"Sort", sortx, false, true /*unused*/, true /*unused*/},
	{"Tab", tab, false, true /*unused*/, true /*unused*/},
	{"Undo", undo, false, true, true /*unused*/},
	{"Zerox", zeroxx, false, true /*unused*/, true /*unused*/},
}

var wsre = regexp.MustCompile("[ \t\n]+")

func lookup(r string) *Exectab {
	r = wsre.ReplaceAllString(r, " ")
	r = strings.TrimLeft(r, " ")
	words := strings.SplitN(r, " ", 2)
	for _, e := range exectab {
		if e.name == words[0] {
			return &e
		}
	}
	return nil
}

func isexecc(c rune) bool {
	if isfilec(c) {
		return true
	}
	return c == '<' || c == '|' || c == '>'
}

func printarg(argt *Text, q0 int, q1 int) string {
	if argt.what != Body || argt.file.name == "" {
		return ""
	}
	if q0 == q1 {
		return fmt.Sprintf("%s:#%d", argt.file.name, q0)
	} else {
		return fmt.Sprintf("%s:#%d,#%d", argt.file.name, q0, q1)
	}
}

func getarg(argt *Text, doaddr bool, dofile bool) (string, string) {
	if argt == nil {
		return "", ""
	}
	a := ""
	var e Expand
	argt.Commit(true)
	var ok bool
	if e, ok = expand(argt, argt.q0, argt.q1); ok {
		if len(e.name) > 0 && dofile {
			if doaddr {
				a = printarg(argt, e.q0, e.q1)
			}
			return e.name, a
		}
	} else {
		e.q0 = argt.q0
		e.q1 = argt.q1
	}
	n := e.q1 - e.q0
	r := make([]rune, n)
	argt.file.b.Read(e.q0, r)
	if doaddr {
		a = printarg(argt, e.q0, e.q1)
	}
	return string(r), a
}

func execute(t *Text, aq0 int, aq1 int, external bool, argt *Text) {
	var (
		q0, q1 int
		r      []rune
		n, f   int
		dir    string
	)

	q0 = aq0
	q1 = aq1
	if q1 == q0 { // expand to find word (actually file name)
		// if in selection, choose selection
		if t.inSelection(q0) {
			q0 = t.q0
			q1 = t.q1
		} else {
			for q1 < t.file.b.Nc() {
				c := t.ReadC(q1)
				if isexecc(c) && c != ':' {
					q1++
				} else {
					break
				}
			}
			for q0 > 0 {
				c := t.ReadC(q0 - 1)
				if isexecc(c) && c != ':' {
					q0--
				} else {
					break
				}
			}
			if q1 == q0 {
				return
			}
		}
	}
	r = make([]rune, q1-q0)
	t.file.b.Read(q0, r)
	e := lookup(string(r))
	if !external && t.w != nil && t.w.nopen[QWevent] > 0 {
		f = 0
		if e != nil {
			f |= 1
		}
		if q0 != aq0 || q1 != aq1 {
			r = make([]rune, aq1-aq0)
			t.file.b.Read(aq0, r)
			f |= 2
		}
		aa, a := getarg(argt, true, true)
		if a != "" {
			if len(a) > EVENTSIZE { // too big; too bad
				warning(nil, "argument string too long\n")
				return
			}
			f |= 8
		}
		c := 'x'
		if t.what == Body {
			c = 'X'
		}
		n = aq1 - aq0
		if n <= EVENTSIZE {
			t.w.Eventf("%c%d %d %d %d %v\n", c, aq0, aq1, f, n, string(r))
		} else {
			t.w.Eventf("%c%d %d %d 0 \n", c, aq0, aq1, f)
		}
		if q0 != aq0 || q1 != aq1 {
			n = q1 - q0
			r := make([]rune, n)
			t.file.b.Read(q0, r)
			if n <= EVENTSIZE {
				t.w.Eventf("%c%d %d 0 %d %v\n", c, q0, q1, n, string(r))
			} else {
				t.w.Eventf("%c%d %d 0 0 \n", c, q0, q1)
			}
		}
		if a != "" {
			t.w.Eventf("%c0 0 0 %d %v\n", c, len(a), a)
			if aa != "" {
				t.w.Eventf("%c0 0 0 %d %v\n", c, len(aa), aa)
			} else {
				t.w.Eventf("%c0 0 0 0 \n", c)
			}
		}
		return
	}
	if e != nil {
		if (e.mark && seltext != nil) && seltext.what == Body {
			seq++
			seltext.w.body.file.Mark()
		}
		s := wsre.ReplaceAllString(string(r), " ")
		s = strings.TrimLeft(s, " ")
		words := strings.SplitN(s, " ", 2)
		if len(words) == 1 {
			words = append(words, "")
		}
		e.fn(t, seltext, argt, e.flag1, e.flag2, words[1])
		return
	}

	b := r
	dir = t.DirName("")
	if dir == "." { // sigh
		dir = ""
	}
	a, aa := getarg(argt, true, true)
	if t.w != nil {
		t.w.ref.Inc()
	}
	run(t.w, string(b), dir, true, aa, a, false)
}

func edit(et * Text, _ * Text, argt * Text, _, _  bool, arg string) {

	if(et == nil) {
		return;
	}
	r, _ := getarg(argt, false, true);
	seq++;
	if(r != ""){
		editcmd(et, []rune(r));
	}else {
		editcmd(et, []rune(arg));
	}
}

func xexit(*Text, *Text, *Text, bool, bool, string) {
	if row.Clean() {
		close(cexit)
		//	threadexits(nil);
	}
}

func del(et *Text, _0 *Text, _1 *Text, flag1 bool, _2 bool, _3 string) {
	if et.col == nil || et.w == nil {
		return
	}
	if flag1 || len(et.w.body.file.text) > 1 || et.w.Clean(false) {
		et.col.Close(et.w, true)
	}
}

func cut(et *Text, t *Text, _ *Text, dosnarf bool, docut bool, _ string) {
	var (
		q0, q1, n, c int
	)
	/*
	 * if not executing a mouse chord (et != t) and snarfing (dosnarf)
	 * and executed Cut or Snarf in window tag (et.w != nil),
	 * then use the window body selection or the tag selection
	 * or do nothing at all.
	 */
	if et != t && dosnarf && et.w != nil {
		if et.w.body.q1 > et.w.body.q0 {
			t = &et.w.body
			if docut {
				t.file.Mark() /* seq has been incremented by execute */
			}
		} else {
			if et.w.tag.q1 > et.w.tag.q0 {
				t = &et.w.tag
			} else {
				t = nil
			}
		}
	}
	if t == nil { /* no selection */
		return
	}
	if t.w != nil && et.w != t.w {
		c = 'M'
		if et.w != nil {
			c = et.w.owner
		}
		t.w.Lock(c)
		defer t.w.Unlock()
	}
	if t.q0 == t.q1 {
		return
	}
	if dosnarf {
		q0 = t.q0
		q1 = t.q1
		snarfbuf.Delete(0, snarfbuf.Nc())
		r := make([]rune, RBUFSIZE)
		for q0 < q1 {
			n = q1 - q0
			if n > RBUFSIZE {
				n = RBUFSIZE
			}
			t.file.b.Read(q0, r[:n])
			snarfbuf.Insert(snarfbuf.Nc(), r[:n])
			q0 += n
		}
		acmeputsnarf()
	}
	if docut {
		t.Delete(t.q0, t.q1, true)
		t.SetSelect(t.q0, t.q0)
		if t.w != nil {
			t.ScrDraw()
			t.w.SetTag()
		}
	} else {
		if dosnarf { /* Snarf command */
			argtext = t
		}
	}
}

func newcol(et *Text, _ *Text, _ *Text, _, _ bool, _ string) {

	c := et.row.Add(nil, -1)
	if c != nil {
		w := c.Add(nil, nil, -1)
		w.SetTag()
		xfidlog(w, "new")
	}
}

func delcol(et *Text, _ *Text, _ *Text, _, _ bool, _ string) {
	c := et.col
	if c == nil || !c.Clean() {
		return
	}
	for i := 0; i < len(c.w); i++ {
		w := c.w[i]
		if w.nopen[QWevent]+w.nopen[QWaddr]+w.nopen[QWdata]+w.nopen[QWxdata] > 0 {
			warning(nil, "can't delete column; %s is running an external command\n", w.body.file.name)
			return
		}
	}
	et.col.row.Close(c, true)
}

func paste(et *Text, t *Text, _ *Text, selectall bool, tobody bool, _ string) {
	var (
		c            int
		q, q0, q1, n int
	)

	/* if tobody, use body of executing window  (Paste or Send command)  */
	if tobody && et != nil && et.w != nil {
		t = &et.w.body
		t.file.Mark() /* seq has been incremented by execute */
	}
	if t == nil {
		return
	}

	acmegetsnarf()
	if t == nil || snarfbuf.Nc() == 0 {
		return
	}
	if t.w != nil && et.w != t.w {
		c = 'M'
		if et.w != nil {
			c = et.w.owner
		}
		t.w.Lock(c)
		defer t.w.Unlock()
	}
	cut(t, t, nil, false, true, "")
	q = 0
	q0 = t.q0
	q1 = t.q0 + snarfbuf.Nc()
	r := make([]rune, RBUFSIZE)
	for q0 < q1 {
		n = q1 - q0
		if n > RBUFSIZE {
			n = RBUFSIZE
		}
		snarfbuf.Read(q, r[:n])
		t.Insert(q0, r[:n], true)
		q += n
		q0 += n
	}
	if selectall {
		t.SetSelect(t.q0, q1)
	} else {
		t.SetSelect(q1, q1)
	}
	if t.w != nil {
		t.ScrDraw()
		t.w.SetTag()
	}
}

func getname(t *Text, argt *Text, arg string, isput bool) string {
	r, _ := getarg(argt, false, true)
	promote := false
	if r == "" {
		promote = true
	} else {
		if isput {
			/* if are doing a Put, want to synthesize name even for non-existent file */
			/* best guess is that file name doesn't contain a slash */
			promote = true
			if strings.Index(r, "/") != -1 {
				t = argt
				arg = r
			}
		}
	}
	if promote {
		if arg == "" {
			return t.file.name
		}
		/* prefix with directory name if necessary */
		r = filepath.Join(t.DirName(""), arg)
	}
	return r
}

func get(et *Text, t *Text, argt *Text, flag1 bool, _ bool, arg string) {

	if flag1 {
		if et == nil || et.w == nil {
			return
		}
	}
	if !et.w.isdir && (et.w.body.file.b.Nc() > 0 && !et.w.Clean(true)) {
		return
	}
	w := et.w
	t = &w.body
	name := getname(t, argt, arg, false)
	if name == "" {
		warning(nil, "no file name\n")
		return
	}
	if len(t.file.text) > 1 {
		isdir, _ := isDir(name)
		if isdir {
			warning(nil, "%s is a directory; can't read with multiple windows on it\n", name)
			return
		}
	}
	r := string(name)
	for _, u := range t.file.text {
		u.Reset()
		u.w.DirFree()
	}
	samename := r == t.file.name
	t.Load(0, name, samename)
	var dirty bool
	if samename {
		t.file.mod = false
		dirty = false
	} else {
		t.file.mod = true
		dirty = true
	}
	for _, u := range t.file.text {
		u.w.dirty = dirty
	}
	w.SetTag()
	t.file.unread = false
	for _, u := range t.file.text {
		u.w.tag.SetSelect(u.w.tag.file.b.Nc(), u.w.tag.file.b.Nc())
		u.ScrDraw()
	}
	xfidlog(w, "get")
}

func id(et, _, _ *Text, _, _ bool, _ string) {
	if et != nil && et.w != nil {
		warning(nil, "/mnt/acme/%d/\n", et.w.id);
	}
}

func checkhash(name string, f *File, d os.FileInfo) {
	Untested()

	h, err := HashFile(name)
	if err != nil {
		warning(nil, "Failed to open %v to compute hash", name)
		return
	}
	if h.Eq(f.hash) {
		//	f->dev = d->dev;
		f.qidpath = d.Name()
		f.mtime = d.ModTime()
	}

}

// TODO(flux): dev and qidpath?
// I haven't spelunked into plan9port to see what it returns for qidpath for regular
// files.  inode?  For now, use the filename.  Awful.
func putfile(f *File, q0 int, q1 int, name string) {
	w := f.curtext.w
	d, err := os.Stat(name)
	if err == nil && name == f.name {
		if /*f.dev!=d.dev || */ f.qidpath != d.Name() || d.ModTime().Sub(f.mtime) > time.Millisecond {
			checkhash(name, f, d)
		}
		if /*f.dev!=d.dev || */ f.qidpath != d.Name() || d.ModTime().Sub(f.mtime) > time.Millisecond {
			if f.unread {
				warning(nil, "%s not written; file already exists\n", name)
			} else {
				warning(nil, "%s modified since last read\n\twas %v; now %v\n", name, f.mtime, d.ModTime())
			}
			//	f.dev = d.dev;
			f.qidpath = d.Name()
			f.mtime = d.ModTime()
			return
		}
	}
	fd, err := os.OpenFile(name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		warning(nil, "can't create file %s: %r\n", name)
		return
	}
	defer fd.Close()

	//h = sha1(nil, 0, nil, nil);

	d, err = fd.Stat()
	isapp := (err == nil && d.Size() > 0 && (d.Mode()&os.ModeAppend) != 0)
	if isapp {
		warning(nil, "%s not written; file is append only\n", name)
		return
	}
	r := make([]rune, RBUFSIZE)
	n := 0
	for q := q0; q < q1; q += n {
		n = q1 - q
		if n > RBUFSIZE {
			n = RBUFSIZE
		}
		f.b.Read(q, r[:n])
		s := string(r[:n])
		//sha1((uchar*)s, m, nil, h);
		nwritten, err := fd.Write([]byte(s))
		if err != nil || nwritten != len(s) {
			warning(nil, "can't write file %s: %r\n", name)
			return
		}
	}
	if name == f.name {
		if q0 != 0 || q1 != f.b.Nc() {
			f.mod = true
			w.dirty = true
			f.unread = true
		} else {
			d1, err := fd.Stat()
			if err != nil {
				d = d1
			}
			//f.qidpath = d.qid.path;
			//f.dev = d.dev;
			f.mtime = d.ModTime()
			//sha1(nil, 0, f.sha1, h);
			//h = nil;
			f.mod = false
			w.dirty = false
			f.unread = false
		}
		for _, u := range f.text {
			u.w.putseq = f.seq
			u.w.dirty = w.dirty
		}
	}

	w.SetTag()
	return
}

func put(et *Text, _0 *Text, argt *Text, _1 bool, _2 bool, arg string) {
	if et == nil || et.w == nil || et.w.isdir {
		return
	}
	w := et.w
	f := w.body.file
	name := getname(&w.body, argt, arg, true)
	if name == "" {
		warning(nil, "no file name\n")
		return
	}
	putfile(f, 0, f.b.Nc(), name)
	xfidlog(w, "put")
}

func putall(et , _, _ *Text, _, _ bool, arg string) {
	for _, col := range row.col {
		for _, w := range col.w {
			if(w.isscratch || w.isdir || w.body.file.name=="") {
				continue
			}
			if(w.nopen[QWevent] > 0) {
				continue
			}
			a := string(w.body.file.name)
			e := access(a);
			if(w.body.file.mod || w.body.ncache > 0) {
				if !e {
					warning(nil, "no auto-Put of %s: %r\n", a);
				} else {
					w.Commit(&w.body);
					put(&w.body, nil, nil, false, false, "");
				}
			}
		}
	}
}


func sortx(et, _, _ *Text, _, _ bool, _ string) {
	if et.col != nil {
		et.col.Sort()
	}
}

func seqof(w *Window, isundo bool) int {
	/* if it's undo, see who changed with us */
	if isundo {
		return w.body.file.seq
	}
	/* if it's redo, see who we'll be sync'ed up with */
	return w.body.file.RedoSeq()
}

func undo(et *Text, _ *Text, _ *Text, flag1, _ bool, _ string) {

	if et == nil || et.w == nil {
		return
	}
	seq := seqof(et.w, flag1)
	if seq == 0 {
		/* nothing to undo */
		return
	}
	/*
	 * Undo the executing window first. Its display will update. other windows
	 * in the same file will not call show() and jump to a different location in the file.
	 * Simultaneous changes to other files will be chaotic, however.
	 */
	et.w.Undo(flag1)
	for _, c := range row.col {
		for _, w := range c.w {
			if w == et.w {
				continue
			}
			if seqof(w, flag1) == seq {
				w.Undo(flag1)
			}
		}
	}
}

func run(win *Window, s string, rdir string, newns bool, argaddr string, xarg string, iseditcmd bool) {
	Untested()
	var (
		c    *Command
		cpid chan *os.Process
	)

	if len(s) == 0 {
		return
	}

	c = &Command{}
	cpid = make(chan *os.Process)
	go runproc(win, s, rdir, newns, argaddr, xarg, c, cpid, iseditcmd)
	// This is to avoid blocking waiting for task launch.
	// So runproc sends the resulting process down cpid,
	// and runwait task catches, and records the process in command list (by
	// pumping it down the ccommand chanel)
	go runwaittask(c, cpid)
}

func sendx(et *Text, t *Text, _ *Text, _, _ bool, _ string) {
	if et.w == nil {
		return
	}
	t = &et.w.body
	if t.q0 != t.q1 {
		cut(t, t, nil, true, false, "")
	}
	t.SetSelect(t.file.b.Nc(), t.file.b.Nc())
	paste(t, t, nil, true, true, "")
	if t.ReadC(t.file.b.Nc()-1) != '\n' {
		t.Insert(t.file.b.Nc(), []rune("\n"), true)
		t.SetSelect(t.file.b.Nc(), t.file.b.Nc())
	}
	t.iq1 = t.q1
	t.Show(t.q1, t.q1, true)
}

func look(et *Text, t *Text, argt *Text, _, _ bool, arg string) {
	if et != nil && et.w != nil {
		t = &et.w.body
		if len(arg) > 0 {
			search(t, []rune(arg))
			return
		}
		r, _ := getarg(argt, false, false)
		if r == "" {
			n := t.q1 - t.q0
			rb := make([]rune, n)
			t.file.b.Read(t.q0, rb[:n])
			r = string(rb) // TODO(flux) Too many gross []rune-string conversions in here
		}
		search(t, []rune(r))
	}
}

func tab(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {

	if et == nil || et.w == nil {
		return
	}
	w := et.w
	r, _ := getarg(argt, false, true)
	tab := int64(0)
	if r != "" {
		p := string(r)
		if '0' <= p[0] && p[0] <= '9' {
			tab, _ = strconv.ParseInt(p, 10, 16)
		}
	} else {
		arg = wsre.ReplaceAllString(string(arg), " ")
		args := strings.Split(arg, " ")
		arg = args[0]
		p := string(arg)
		if '0' <= p[0] && p[0] <= '9' {
			tab, _ = strconv.ParseInt(p, 10, 16)
		}
	}
	if tab > 0 {
		if w.body.tabstop != int(tab) {
			w.body.tabstop = int(tab)
			w.Resize(w.r, false, true)
		}
	} else {
		warning(nil, "%s: Tab %d\n", w.body.file.name, w.body.tabstop)
	}
}

func fontx(et *Text, t *Text, argt *Text, _, _ bool, arg string) {
	if et == nil || et.w == nil {
		return
	}
	t = &et.w.body
	file := ""
	// Parse parameter.  It might be in arg, or argt, or both
	r, _ := getarg(argt, false, true)
	r = r + " " + arg
	r = wsre.ReplaceAllString(string(r), " ")
	words := strings.Split(arg, " ")

	for _, wrd := range words {
		switch wrd {
		case "fix":
			file = *fixedfontflag
		case "var":
			file = *varfontflag
		default: // File/fontname
			file = wrd
		}
	}

	if file == "" {
		if t.font == *varfontflag {
			file = *fixedfontflag
		} else {
			file = *varfontflag
		}
	}

	if newfont := fontget(file, row.display); newfont != nil {
		// TODO(rjk): maybe Frame should know how to clear itself on init?
		row.display.ScreenImage.Draw(t.w.r, textcolors[frame.ColBack], nil, image.ZP)
		t.font = file
		t.fr.Init(t.w.r, newfont, row.display.ScreenImage)
		t.fr.InitTick()
		if t.w.isdir {
			t.all.Min.X++ // force recolumnation; disgusting!
			for i, dir := range t.w.dirnames {
				t.w.widths[i] = newfont.StringWidth(dir)
			}
		}
		// avoid shrinking of window due to quantization
		t.w.col.Grow(t.w, -1)
	}
}

func zeroxx(et *Text, t *Text, _ *Text, _, _ bool, _4 string) {
	if t != nil && t.w != nil && t.w != et.w {
		c := int('M')
		if et.w != nil {
			c = et.w.owner
		}
		t.w.Lock(c)
		defer t.w.Unlock()
	}
	if t == nil {
		t = et
	}
	if t == nil || t.w == nil {
		return
	}
	t = &t.w.body
	if t.w.isdir {
		warning(nil, "%s is a directory; Zerox illegal\n", t.file.name)
	} else {
		nw := t.w.col.Add(nil, t.w, -1)
		/* ugly: fix locks so w.unlock works */
		nw.Lock1(t.w.owner)
		xfidlog(nw, "zerox")
	}
}

func runwaittask(c *Command, cpid chan *os.Process) {
	c.proc = <-cpid

	if c.proc != nil { /* successful exec */
		c.pid = c.proc.Pid
		ccommand <- c
	} else {
		if c.iseditcommand {
			cedit <- 0
		}
	}
	cpid = nil
}

func fsopenfd(fsys *client.Fsys, path string, mode uint8) *os.File {
	fid, err := fsys.Open(path, mode)
	if err != nil {
		warning(nil, "Failed to open %v: %v", path, err)
		return nil
	}

	// open a pipe, serve the reads from fid down it
	r, w, err := os.Pipe()
	if err != nil {
		acmeerror("fsopenfd: Could not make pipe", nil)
	}

	if mode == plan9.OREAD {
		go func() {
			var buf [BUFSIZE]byte
			var werr error
			for {
				n, err := fid.Read(buf[:])
				if n != 0 {
					_, werr = w.Write(buf[:n])
				}
				if err != nil || werr != nil {
					fid.Close()
					w.Close()
					return
				}
			}
		}()
		return r
	} else {
		go func() {
			var buf [BUFSIZE]byte
			var werr error
			for {
				n, err := r.Read(buf[:])
				if n != 0 {
					_, werr = fid.Write(buf[:n])
				}
				if err != nil || werr != nil {
					r.Close()
					fid.Close()
					return
				}
			}
		}()
		return w
	}
}

func runproc(win *Window, s string, rdir string, newns bool, argaddr string, arg string, c *Command, cpid chan *os.Process, iseditcmd bool) {
	var (
		t, name, filename, dir string
		incl                   []string
		winid                  int
		sfd                    [3]*os.File
		pipechar               int
		//static void *parg[2];
		rcarg []string
		shell string
	)
	Fail := func() {
		Untested()
		// threadexec hasn't happened, so send a zero
		sfd[0].Close()
		if sfd[2] != sfd[1] {
			sfd[2].Close()
		}
		sfd[1].Close()
		cpid <- nil
	}
	Hard := func() {
		Untested()
		//* ugly: set path = (. $cputype /bin)
		//* should honor $path if unusual.
		/* TODO(flux): This looksl ike plan9 magic
		if cputype {
			n = 0;
			memmove(buf+n, ".", 2);
			n += 2;
			i = strlen(cputype)+1;
			memmove(buf+n, cputype, i);
			n += i;
			memmove(buf+n, "/bin", 5);
			n += 5;
			fd = create("/env/path", OWRITE, 0666);
			write(fd, buf, n);
			close(fd);
		}
		*/

		if arg != "" {
			s = fmt.Sprintf("%s '%s'", t, arg) // TODO(flux): BUG: what if quote in arg?
			// This is a bug from the original; and I now know
			// why ' in an argument fails to work properly.
			t = s
			c.text = s
		}
		dir = ""
		if rdir != "" {
			dir = string(rdir)
		}
		shell = acmeshell
		if shell == "" {
			shell = "rc"
		}
		rcarg = []string{shell, "-c", t}
		cmd := exec.Command(rcarg[0], rcarg[1:]...)
		cmd.Dir = dir
		cmd.Stdin = sfd[0]
		cmd.Stdout = sfd[1]
		cmd.Stderr = sfd[2]
		if err := cmd.Start(); err == nil {
			if cpid != nil {
				cpid <- cmd.Process
			}
			return // TODO(flux) where do we wait?
		}
		warning(nil, "exec %s: %r\n", shell)
		Fail()
	}
	t = strings.TrimLeft(s, " \t\n")
	name = filepath.Base(string(t)) + " "
	c.name = name
	// t is the full path, trimmed of left whitespace.
	pipechar = 0
	if t[0] == '<' || t[0] == '|' || t[0] == '>' {
		pipechar = int(t[0])
		t = t[1:]
	}
	c.iseditcommand = iseditcmd
	c.text = s
	if newns {
		incl = nil
		if win != nil {
			filename = string(win.body.file.name)
			if len(incl) > 0 {
				incl = make([]string, len(incl)) // Incl is inherited by actions in this window
				for i, inc := range win.incl {
					incl[i] = inc
				}
			}
			winid = win.id
		} else {
			filename = ""
			winid = 0
			if activewin != nil {
				winid = activewin.id
			}
		}
		// 	rfork(RFNAMEG|RFENVG|RFFDG|RFNOTEG); TODO(flux): I'm sure these settings are important

		os.Setenv("winid", fmt.Sprintf("%d", winid))

		if filename != "" {
			os.Setenv("%", filename)
			os.Setenv("samfile", filename)
		}
		c.md = fsysmount(rdir, incl)
		if c.md == nil {
			fmt.Fprintf(os.Stderr, "child: can't allocate mntdir\n")
			return
		}
		conn, err := client.DialService("acme")
		if err != nil {
			fmt.Fprintf(os.Stderr, "child: can't connect to acme: %v\n", err)
			fsysdelid(c.md)
			c.md = nil
			return
		}
		fs, err  := conn.Attach(nil, getuser(), fmt.Sprintf("%d", c.md.id))
		if err != nil {
			fmt.Fprintf(os.Stderr, "child: can't attach to acme: %v\n", err)
			fsysdelid(c.md)
			c.md = nil
			return
		}
		if winid > 0 && (pipechar == '|' || pipechar == '>') {
			rdselname := fmt.Sprintf("%d/rdsel", winid)
			sfd[0] = fsopenfd(fs, rdselname, plan9.OREAD)
		} else {
			sfd[0], _ = os.OpenFile("/dev/null", os.O_RDONLY, 0777)
		}
		if (winid > 0 || iseditcmd) && (pipechar == '|' || pipechar == '<') {
			var buf string
			if iseditcmd {
				if winid > 0 {
					buf = fmt.Sprintf("%d/editout", winid)
				} else {
					buf = fmt.Sprintf("editout")
				}
			} else {
				buf = fmt.Sprintf("%d/wrsel", winid)
			}
			sfd[1] = fsopenfd(fs, buf, plan9.OWRITE)
			sfd[2] = fsopenfd(fs, "cons", plan9.OWRITE)
		} else {
			sfd[1] = fsopenfd(fs, "cons", plan9.OWRITE)
			sfd[2] = sfd[1]
		}
		// fsunmount(fs); looks like with plan9.client you just drop it on the floor.
		fs = nil
	} else {
		//	rfork(RFFDG|RFNOTEG);
		fsysclose()
		sfd[0], _ = os.Open("/dev/null")
		sfd[1], _ = os.OpenFile("/dev/null", os.O_WRONLY, 0777)
		nfd, _ := syscall.Dup(erroutfd)
		sfd[2] = os.NewFile(uintptr(nfd), "duped erroutfd")
	}
	if win != nil {
		win.Close()
	}

	if argaddr != "" {
		os.Setenv("acmeaddr", argaddr)
	}
	if acmeshell != "" {
		Hard()
		return
	}
	for _, r := range t {
		if r == ' ' || r == '\t' {
			continue
		}
		if r < ' ' {
			Hard()
			return
		}
		if utfrune([]rune("#;&|^$=`'{}()<>[]*?^~`/"), r) != -1 {
			Hard()
			return
		}
	}

	t = wsre.ReplaceAllString(string(t), " ")
	t = strings.TrimLeft(t, " ")
	c.av = strings.Split(t, " ")
	if arg != "" {
		c.av = append(c.av, arg)
	}

	dir = ""
	if rdir != "" {
		dir = string(rdir)
	}
	cmd := exec.Command(c.av[0], c.av[1:]...)
	cmd.Dir = dir
	cmd.Stdin = sfd[0]
	cmd.Stdout = sfd[1]
	cmd.Stderr = sfd[2]
	err := cmd.Start()
	if err == nil {
		if cpid != nil {
			cpid <- cmd.Process
		} else {
			cpid <- nil
		}
		// Where do we wait TODO(flux)
		return
	}

	Fail()
	return

}

const (
	IGlobal = iota - 2
	IError
	Ion
	Ioff
)

func indentval(s string) int {
	if len(s) < 2 {
		return IError
	}
	switch s {
	case "ON":
		globalautoindent = true
		warning(nil, "Indent ON\n")
		return IGlobal
	case "OFF":
		globalautoindent = false
		warning(nil, "Indent OFF\n")
		return IGlobal
	case "on":
		return Ion
	case "off":
		return Ioff
	default:
		return Ioff
	}
}

func indent(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	var autoindent int
	w := (*Window)(nil)
	if et != nil && et.w != nil {
		w = et.w
	}
	autoindent = IError
	r, _ := getarg(argt, false, true)
	if len(r) > 0 {
		autoindent = indentval(r)
	} else {
		autoindent = indentval(strings.SplitN(arg, " ", 2)[0])
	}
	if autoindent == IGlobal {
		row.AllWindows(func(w *Window) { w.autoindent = globalautoindent })
	} else {
		if w != nil && autoindent >= 0 {
			w.autoindent = autoindent == Ion
		}
	}
}

func dump(et *Text, _ *Text, argt *Text, isdump bool, _ bool, arg string) {
	name := ""

	if arg != "" {
		name = arg
	} else {
		r, _ := getarg(argt, false, true)
		name = string(r)
	}

	if isdump {
		row.Dump(name)
	} else {
		row.Load(name, false)
	}
}
