package main

import (
	"crypto/sha1"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/rjkroege/edwood/internal/file"
	"github.com/rjkroege/edwood/internal/frame"
)

type Exectab struct {
	name  string
	fn    func(t, seltext, argt *Text, flag1, flag2 bool, arg string)
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
	{"Edit", edit, false, true /*unused*/, true /*unused*/},
	{"Exit", xexit, false, true /*unused*/, true /*unused*/},
	{"Font", fontx, false, true /*unused*/, true /*unused*/},
	{"Get", get, false, true, true /*unused*/},
	{"ID", id, false, true /*unused*/, true /*unused*/},
	//	{ "Incl",		incl,		false,	true /*unused*/,		true /*unused*/		},
	{"Indent", indent, false, true /*unused*/, true /*unused*/},
	{"Kill", xkill, false, true /*unused*/, true /*unused*/},
	{"Load", dump, false, false, true /*unused*/},
	{"Local", local, false, true /*unused*/, true /*unused*/},
	{"Look", look, false, true /*unused*/, true /*unused*/},
	{"New", newx, false, true /*unused*/, true /*unused*/},
	{"Newcol", newcol, false, true /*unused*/, true /*unused*/},
	{"Paste", paste, true, true, true /*unused*/},
	{"Put", put, false, true /*unused*/, true /*unused*/},
	{"Putall", putall, false, true /*unused*/, true /*unused*/},
	{"Redo", undo, false, false, true /*unused*/},
	{"Send", sendx, true, true /*unused*/, true /*unused*/},
	{"Snarf", cut, false, true, false},
	{"Sort", sortx, false, true /*unused*/, true /*unused*/},
	{"Tab", tab, false, true /*unused*/, true /*unused*/},
	{"Tabexpand", expandtab, false, true /*unused*/, true /*unused*/},
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
	}
	return fmt.Sprintf("%s:#%d,#%d", argt.file.name, q0, q1)
}

// TODO(rjk): use a tokenizer on the results of getarg
func getarg(argt *Text, doaddr bool, dofile bool) (string, string) {
	if argt == nil {
		return "", ""
	}
	a := ""
	var e *Expand
	argt.Commit()
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

// execute must run with an existing lock on t's Window
func execute(t *Text, aq0 int, aq1 int, external bool, argt *Text) {
	var n, f int

	q0 := aq0
	q1 := aq1
	if q1 == q0 { // expand to find word (actually file name)
		// if in selection, choose selection
		if t.inSelection(q0) {
			q0 = t.q0
			q1 = t.q1
		} else {
			for q1 < t.file.Size() {
				c := t.file.ReadC(q1)
				if isexecc(c) && c != ':' {
					q1++
				} else {
					break
				}
			}
			for q0 > 0 {
				c := t.file.ReadC(q0 - 1)
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
	r := make([]rune, q1-q0)
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
			seltext.w.body.file.Mark(seq)
		}

		s := strings.TrimLeft(string(r), " \t\n")
		words := wsre.Split(s, 2)
		arg := ""
		if len(words) > 1 {
			arg = strings.TrimLeft(words[1], " \t\n")
		}
		e.fn(t, seltext, argt, e.flag1, e.flag2, arg)
		return
	}

	b := r
	dir := t.DirName("") // exec.Cmd.Dir
	a, aa := getarg(argt, true, true)
	if t.w != nil {
		t.w.ref.Inc()
	}
	run(t.w, string(b), dir, true, aa, a, false)
}

func edit(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if et == nil {
		return
	}
	r, _ := getarg(argt, false, true)

	seq++
	if r != "" {
		editcmd(et, []rune(r))
	} else {
		editcmd(et, []rune(arg))
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
	if flag1 || et.w.body.file.HasMultipleTexts() || et.w.Clean(false) {
		et.col.Close(et.w, true)
	}
}

func cut(et *Text, t *Text, _ *Text, dosnarf bool, docut bool, _ string) {
	var (
		q0, q1, n, c int
	)
	// if not executing a mouse chord (et != t) and snarfing (dosnarf)
	// and executed Cut or Snarf in window tag (et.w != nil),
	// then use the window body selection or the tag selection
	// or do nothing at all.
	if et != t && dosnarf && et.w != nil {
		if et.w.body.q1 > et.w.body.q0 {
			t = &et.w.body
			if docut {
				t.file.Mark(seq) // seq has been incremented by execute
			}
		} else {
			if et.w.tag.q1 > et.w.tag.q0 {
				t = &et.w.tag
			} else {
				t = nil
			}
		}
	}
	if t == nil { // no selection
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
		snarfbuf.Delete(0, snarfbuf.nc())
		r := make([]rune, RBUFSIZE)
		for q0 < q1 {
			n = q1 - q0
			if n > RBUFSIZE {
				n = RBUFSIZE
			}
			t.file.b.Read(q0, r[:n])
			snarfbuf.Insert(snarfbuf.nc(), r[:n])
			q0 += n
		}
		acmeputsnarf()
	}
	if docut {
		t.Delete(t.q0, t.q1, true)
		t.SetSelect(t.q0, t.q0)
		if t.w != nil {
			t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
			t.w.Commit(t)
			t.w.SetTag()
		}
	} else {
		if dosnarf { // Snarf command
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

	// if tobody, use body of executing window  (Paste or Send command)
	if tobody && et != nil && et.w != nil {
		t = &et.w.body
		t.file.Mark(seq) // seq has been incremented by execute
	}
	if t == nil {
		return
	}

	acmegetsnarf()
	if t == nil || snarfbuf.nc() == 0 {
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
	q1 = t.q0 + snarfbuf.nc()
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
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		t.w.Commit(t)
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
			// if are doing a Put, want to synthesize name even for non-existent file
			// best guess is that file name doesn't contain a slash
			promote = true
			if strings.ContainsRune(r, '/') {
				t = argt
				arg = r
			}
		}
	}
	if promote {
		if arg == "" {
			return t.file.name
		}
		// prefix with directory name if necessary
		r = filepath.Join(t.DirName(""), arg)
	}
	return r
}

func get(et *Text, _ *Text, argt *Text, flag1 bool, _ bool, arg string) {
	if flag1 {
		if et == nil || et.w == nil {
			return
		}
	}

	isclean := et.w.Clean(true)
	if et.w.body.file.Size() > 0 && !isclean {
		return
	}
	w := et.w
	t := &w.body
	name := getname(t, argt, arg, false)
	if name == "" {
		warning(nil, "no file name\n")
		return
	}
	newNameIsdir, _ := isDir(name)
	if t.file.HasMultipleTexts() && newNameIsdir {
		warning(nil, "%s is a directory; can't read with multiple windows on it\n", name)
		return
	}

	t.Delete(0, t.file.Nr(), true)
	samename := name == t.file.name
	t.Load(0, name, samename)

	// Text.Delete followed by Text.Load will always mark the File as
	// modified unless loading a 0-length file over a 0-length file. But if
	// samename is true here, we know that the Text.body.File is now the same
	// as it is on disk. So indicate this with file.Clean().
	if samename {
		t.file.Clean()
	}
	w.SetTag()
	xfidlog(w, "get")
}

func id(et, _, _ *Text, _, _ bool, _ string) {
	if et != nil && et.w != nil {
		warning(nil, "/mnt/acme/%d/\n", et.w.id)
	}
}

func xkill(_, _ *Text, argt *Text, _, _ bool, args string) {
	if r, _ := getarg(argt, false, false); len(r) > 0 {
		xkill(nil, nil, nil, false, false, r)
	}
	for _, cmd := range strings.Fields(args) {
		ckill <- cmd
	}
}

func local(et, _, argt *Text, _, _ bool, arg string) {
	a, aa := getarg(argt, true, true)
	dir := et.DirName("") // exec.Cmd.Dir
	run(nil, arg, dir, false, aa, a, false)
}

// putfile writes File to disk, if it's safe to do so.
//
// TODO(flux): Write this in terms of the various cases.
func putfile(f *File, q0 int, q1 int, name string) error {
	w := f.curtext.w
	d, err := os.Stat(name)

	// Putting to the same file that we already read from.
	if err == nil && name == f.name {
		if !os.SameFile(f.info, d) || d.ModTime().Sub(f.info.ModTime()) > time.Millisecond {
			f.UpdateInfo(name, d)
		}
		if !os.SameFile(f.info, d) || d.ModTime().Sub(f.info.ModTime()) > time.Millisecond {
			// By setting File.info here, a subsequent Put will ignore that
			// the disk file was mutated and will write File to the disk file.
			f.info = d

			if f.hash == file.EmptyHash {
				// Edwood created the File but a disk file with the same name exists.
				return warnError(nil, "%s not written; file already exists", name)
			}

			// Edwood loaded the disk file to File but the disk file has been modified since.
			return warnError(nil, "%s modified since last read\n\twas %v; now %v", name, f.info.ModTime(), d.ModTime())
		}
	}

	fd, err := os.OpenFile(name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return warnError(nil, "can't create file %s: %v", name, err)
	}
	defer fd.Close()

	h := sha1.New()

	d, err = fd.Stat()
	isapp := (err == nil && d.Size() > 0 && (d.Mode()&os.ModeAppend) != 0)
	if isapp {
		return warnError(nil, "%s not written; file is append only", name)
	}

	_, err = io.Copy(io.MultiWriter(h, fd), f.b.Reader(q0, q1))
	if err != nil {
		return warnError(nil, "can't write file %s: %v", name, err)
	}

	// Putting to the same file as the one that we originally read from.
	if name == f.name {
		if q0 != 0 || q1 != f.Size() {
			// The backing disk file contents now differ from File because
			// we've over-written the disk file with part of File.
			f.Modded()
		} else {
			// A normal put operation of a file modified in Edwood but not
			// modified on disk.
			if d1, err := fd.Stat(); err == nil {
				d = d1
			}
			f.info = d
			f.hash.Set(h.Sum(nil))
			f.Clean()
		}
	}
	w.SetTag()
	return nil
}

func put(et *Text, _0 *Text, argt *Text, _1 bool, _2 bool, arg string) {
	if et == nil || et.w == nil || et.w.body.file.IsDir() {
		return
	}
	w := et.w
	f := w.body.file
	name := getname(&w.body, argt, arg, true)
	if name == "" {
		warning(nil, "no file name\n")
		return
	}
	putfile(f, 0, f.Size(), name)
	xfidlog(w, "put")
}

func putall(et, _, _ *Text, _, _ bool, arg string) {
	for _, col := range row.col {
		for _, w := range col.w {
			if w.nopen[QWevent] > 0 {
				continue
			}
			a := w.body.file.name
			if w.body.file.SaveableAndDirty() {
				if _, err := os.Stat(a); err != nil {
					warning(nil, "no auto-Put of %s: %v\n", a, err)
				} else {
					w.Commit(&w.body)
					put(&w.body, nil, nil, false, false, "")
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
	// if it's undo, see who changed with us
	if isundo {
		return w.body.file.Seq()
	}
	// if it's redo, see who we'll be sync'ed up with
	return w.body.file.RedoSeq()
}

// TODO(rjk): Why does this work this way?
func undo(et *Text, _ *Text, _ *Text, flag1, _ bool, _ string) {
	if et == nil || et.w == nil {
		return
	}
	seq := seqof(et.w, flag1)
	if seq == 0 {
		// nothing to undo
		return
	}
	// Undo the executing window first. Its display will update. other windows
	// in the same file will not call show() and jump to a different location in the file.
	// Simultaneous changes to other files will be chaotic, however.
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
	if len(s) == 0 {
		return
	}

	c := &Command{}
	cpid := make(chan *os.Process)
	go func() {
		err := runproc(win, s, rdir, newns, argaddr, xarg, c, cpid, iseditcmd)
		if err != nil && err != errEmptyCmd {
			warning(nil, "%v\n", err)
		}
	}()
	// This is to avoid blocking waiting for task launch.
	// So runproc sends the resulting process down cpid,
	// and runwait task catches, and records the process in command list (by
	// pumping it down the ccommand chanel)
	go runwaittask(c, cpid)
}

// sendx appends selected text or snarf buffer to end of body.
func sendx(et, _, _ *Text, _, _ bool, _ string) {
	if et.w == nil {
		return
	}
	t := &et.w.body
	if t.q0 != t.q1 {
		cut(t, t, nil, true, false, "")
	}
	t.SetSelect(t.file.Size(), t.file.Size())
	paste(t, t, nil, true, true, "")
	if t.ReadC(t.file.Size()-1) != '\n' {
		t.Insert(t.file.Size(), []rune("\n"), true)
		t.SetSelect(t.file.Size(), t.file.Size())
	}
	t.iq1 = t.q1
	t.Show(t.q1, t.q1, true)
}

func look(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if et != nil && et.w != nil {
		t := &et.w.body
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
		p := r
		if '0' <= p[0] && p[0] <= '9' {
			tab, _ = strconv.ParseInt(p, 10, 16)
		}
	} else {
		arg = wsre.ReplaceAllString(arg, " ")
		args := strings.Split(arg, " ")
		arg = args[0]
		p := arg
		if len(p) == 0 {
			return
		}
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

func expandtab(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if et == nil || et.w == nil {
		return
	}
	w := et.w
	if w.body.tabexpand {
		w.body.tabexpand = false
		warning(nil, "%s: Tab: %d, Tabexpand OFF\n", w.body.file.name, w.body.tabstop)
	} else {
		w.body.tabexpand = true
		warning(nil, "%s: Tab: %d, Tabexpand ON\n", w.body.file.name, w.body.tabstop)
	}
}

func fontx(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if et == nil || et.w == nil {
		return
	}
	t := &et.w.body
	file := ""
	// Parse parameter.  It might be in arg, or argt, or both
	r, _ := getarg(argt, false, true)
	words := strings.Fields(r + " " + arg)

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
		row.display.ScreenImage().Draw(t.w.r, textcolors[frame.ColBack], nil, image.Point{})
		t.font = file
		t.fr.Init(t.w.r, frame.OptFont(newfont), frame.OptBackground(row.display.ScreenImage()))

		if t.w.body.file.IsDir() {
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
	if t.w.body.file.IsDir() {
		// TODO(rjk): Why?
		warning(nil, "%s is a directory; Zerox illegal\n", t.file.name)
	} else {
		nw := t.w.col.Add(nil, t.w, -1)
		// ugly: fix locks so w.unlock works
		// TODO(rjk): We need to handle this better.
		nw.lock1(t.w.owner)
		xfidlog(nw, "zerox")
	}
}

func runwaittask(c *Command, cpid chan *os.Process) {
	c.proc = <-cpid

	if c.proc != nil { // successful exec
		c.pid = c.proc.Pid
		ccommand <- c
	} else {
		if c.iseditcommand {
			cedit <- 0
		}
	}
	cpid = nil
}

var errEmptyCmd = fmt.Errorf("empty command")

// runproc. Something with the running of external processes. Executes
// asynchronously.
// TODO(rjk): Must lock win on mutation.
func runproc(win *Window, s string, dir string, newns bool, argaddr string, arg string, c *Command, cpid chan *os.Process, iseditcmd bool) error {
	var (
		t, name, filename string
		incl              []string
		winid             int
		sin               io.ReadCloser
		sout, serr        io.WriteCloser
		pipechar          int
		rcarg             []string
		shell             string
	)

	Closeall := func() {
		if sin != nil {
			sin.Close()
		}
		if serr != nil && serr != sout {
			serr.Close()
		}
		if sout != nil {
			sout.Close()
		}
	}
	Fail := func() {
		Closeall()
		// threadexec hasn't happened, so send a zero
		cpid <- nil
	}
	Hard := func() error {
		if arg != "" {
			s = fmt.Sprintf("%s '%s'", t, arg) // TODO(flux): BUG: what if quote in arg?
			// This is a bug from the original; and I now know
			// why ' in an argument fails to work properly.
			t = s
			c.text = s
		}
		shell = acmeshell
		if shell == "" {
			shell = "rc"
		}
		rcarg = []string{shell, "-c", t}
		cmd := exec.Command(rcarg[0], rcarg[1:]...)
		cmd.Dir = dir
		cmd.Stdin = sin
		cmd.Stdout = sout
		cmd.Stderr = serr
		err := cmd.Start()
		if err != nil {
			Fail()
			return fmt.Errorf("exec %s: %v", shell, err)
		}
		cpid <- cmd.Process
		go func() {
			cmd.Wait()
			Closeall()
			cwait <- cmd.ProcessState
		}()
		return nil
	}
	t = strings.TrimLeft(s, " \t\n")
	name = t
	if i := strings.IndexAny(name, " \t\n"); i >= 0 {
		name = name[:i]
	}
	c.name = filepath.Base(name) + " "

	// t is the full path, trimmed of left whitespace.
	pipechar = 0
	if len(t) > 0 && (t[0] == '<' || t[0] == '|' || t[0] == '>') {
		pipechar = int(t[0])
		t = t[1:]
	}
	c.iseditcommand = iseditcmd
	c.text = s
	if newns {
		if win != nil {
			// Access possibly mutable Window state inside a lock.
			win.lk.Lock()
			filename = win.body.file.name
			winid = win.id
			incl = append([]string{}, win.incl...)
			win.lk.Unlock()
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
		var fs *client.Fsys
		var err error
		c.md, fs, err = fsysmount(dir, incl)
		if err != nil {
			return fmt.Errorf("fsysmount: %v", err)
		}
		if winid > 0 && (pipechar == '|' || pipechar == '>') {
			rdselname := fmt.Sprintf("%d/rdsel", winid)
			sin = fsopenfd(fs, rdselname, plan9.OREAD)
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
			sout = fsopenfd(fs, buf, plan9.OWRITE)
			serr = fsopenfd(fs, "cons", plan9.OWRITE)
		} else {
			sout = fsopenfd(fs, "cons", plan9.OWRITE)
			serr = sout
		}
		// fsunmount(fs); looks like with plan9.client you just drop it on the floor.
		fs = nil
	} else {
		// TODO(fhs): If runtime.GOOS is plan9, we need to execute the command in
		// Edwood's file name space and environment variable group.

		serr = errorWriter{}
	}
	if win != nil {
		win.lk.Lock()
		win.Close()
		win.lk.Unlock()
	}

	if argaddr != "" {
		os.Setenv("acmeaddr", argaddr)
	}
	if acmeshell != "" {
		return Hard()
	}
	for _, r := range t {
		if r == ' ' || r == '\t' {
			continue
		}
		if r < ' ' {
			return Hard()
		}
		if strings.ContainsRune("#;&|^$=`'{}()<>[]*?^~`/", r) {
			return Hard()
		}
	}

	c.av = strings.Fields(t)
	if arg != "" {
		c.av = append(c.av, arg)
	}
	if len(c.av) == 0 {
		Fail()
		return errEmptyCmd
	}
	cmd := exec.Command(c.av[0], c.av[1:]...)
	cmd.Dir = dir
	cmd.Stdin = sin
	cmd.Stdout = sout
	cmd.Stderr = serr
	err := cmd.Start()
	if err != nil {
		Fail()
		return err
	}
	cpid <- cmd.Process
	go func() {
		cmd.Wait()
		Closeall()
		cwait <- cmd.ProcessState
	}()
	return nil
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
		*globalAutoIndent = true
		warning(nil, "Indent ON\n")
		return IGlobal
	case "OFF":
		*globalAutoIndent = false
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
	w := (*Window)(nil)
	if et != nil && et.w != nil {
		w = et.w
	}
	autoindent := int(IError)
	r, _ := getarg(argt, false, true)
	if len(r) > 0 {
		autoindent = indentval(r)
	} else {
		autoindent = indentval(strings.SplitN(arg, " ", 2)[0])
	}
	if autoindent == IGlobal {
		row.AllWindows(func(w *Window) { w.autoindent = *globalAutoIndent })
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
		name = r
	}

	if isdump {
		row.Dump(name)
	} else {
		row.Load(nil, name, false)
	}
}
