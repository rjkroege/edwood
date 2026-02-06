package main

import (
	"bytes"
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
	"unicode/utf8"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

type Exectab struct {
	// Name of the command.
	name string

	// Function run to implement this command.
	//
	// * t is the text where the middle click happened. This is frequently
	// the tag and the command will affect the tag's window's body.
	//
	// * seltext comes from global.seltext. This is the last text clicked on
	// with LMB. Middle clicks don't change unless the middle click has the
	// side-effect of deleting the text. in which case it becomes nil.
	//
	// * argt text contains the argument to a MMB-LMB chord. If not
	// delivering an argument this way, it will be nil.
	//
	// * arg is the string after the command as MMB-dragged over the command
	// and arg.
	fn func(t, seltext, argt *Text, flag1, flag2 bool, arg string)

	// Command is undoable (e.g. Cut) and requires establishing an Undo point.
	mark bool

	// Meaning of both flags is command-specific and is used (mostly) to let a single
	// function implement two different commands. Note the TODO below
	flag1 bool
	flag2 bool
}

// TODO(rjk): This could be more idiomatic: each command implements an
// interface. Flags would then be unnecessary.

var globalexectab = []Exectab{
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
	{"Markdeep", previewcmd, false, true /*unused*/, true /*unused*/},
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

// TODO(rjk): Exectab is sorted. Consider using a binary search
func lookup(r string, exectab []Exectab) *Exectab {
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
	if argt.what != Body || argt.file.Name() == "" {
		return ""
	}
	if q0 == q1 {
		return fmt.Sprintf("%s:#%d", argt.file.Name(), q0)
	}
	return fmt.Sprintf("%s:#%d,#%d", argt.file.Name(), q0, q1)
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
	argt.file.Read(e.q0, r)
	if doaddr {
		a = printarg(argt, e.q0, e.q1)
	}
	return string(r), a
}

// expandRuneOffsetsToWord expands the rune offsets on a middle click to
// the word boundaries. TODO(rjk): Conceivably, what we think of as a
// "word boundary" should be configurable in some way and not embedded in
// this function.
// TODO(rjk): Consider if this method should really be part of Text.
func expandRuneOffsetsToWord(t *Text, q0 int, q1 int) (int, int) {
	if q1 == q0 { // expand to find word (actually file name)
		// if in selection, choose selection
		if t.inSelection(q0) {
			q0 = t.q0
			q1 = t.q1
		} else {
			for q1 < t.file.Nr() {
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
				return q0, q1
			}
		}
	}
	return q0, q1
}

// delegateExecution handles the situation where an external command is
// using the event file to control the operation of Edwood via the
// filesystem.
func delegateExecution(t *Text, e *Exectab, aq0, aq1, q0, q1 int, argt *Text) {
	f := 0
	if e != nil {
		f |= 1
	}
	if q0 != aq0 || q1 != aq1 {
		f |= 2
	}
	// Always read the text for the event - we need to send the actual text
	// regardless of whether word expansion occurred.
	r := make([]rune, aq1-aq0)
	t.file.Read(aq0, r)
	a, aa := getarg(argt, true, true)
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
	n := aq1 - aq0
	if n <= EVENTSIZE {
		t.w.Eventf("%c%d %d %d %d %v\n", c, aq0, aq1, f, n, string(r))
	} else {
		t.w.Eventf("%c%d %d %d 0 \n", c, aq0, aq1, f)
	}
	if q0 != aq0 || q1 != aq1 {
		n = q1 - q0
		r = make([]rune, n)
		t.file.Read(q0, r)
		if n <= EVENTSIZE {
			t.w.Eventf("%c%d %d 0 %d %v\n", c, q0, q1, n, string(r))
		} else {
			t.w.Eventf("%c%d %d 0 0 \n", c, q0, q1)
		}
	}
	if a != "" {
		t.w.Eventf("%c0 0 0 %d %v\n", c, utf8.RuneCountInString(a), a)
		if aa != "" {
			t.w.Eventf("%c0 0 0 %d %v\n", c, utf8.RuneCountInString(aa), aa)
		} else {
			t.w.Eventf("%c0 0 0 0 \n", c)
		}
	}
}

// execute must run with an existing lock on t's Window
func execute(t *Text, aq0 int, aq1 int, external bool, argt *Text) {
	q0, q1 := expandRuneOffsetsToWord(t, aq0, aq1)

	r := make([]rune, q1-q0)
	t.file.Read(q0, r)
	e := lookup(string(r), globalexectab)

	// Send commands to external client if the target window's event file is
	// in use.
	if !external && t.w != nil && t.w.nopen[QWevent] > 0 {
		delegateExecution(t, e, aq0, aq1, q0, q1, argt)
		return
	}

	// Invoke an internal command if it exists.
	if e != nil {
		if (e.mark && global.seltext != nil) && global.seltext.what == Body {
			global.seq++
			global.seltext.w.body.file.Mark(global.seq)
		}

		s := strings.TrimLeft(string(r), " \t\n")
		words := wsre.Split(s, 2)
		arg := ""
		if len(words) > 1 {
			arg = strings.TrimLeft(words[1], " \t\n")
		}

		// e.fn is the function from the Exectab. flag1 and flag2 are also from the Exectab.
		e.fn(t, global.seltext, argt, e.flag1, e.flag2, arg)
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

// previewExecute executes a command from the rendered preview text.
// Unlike execute(), which reads command text from the source file buffer,
// this uses the rendered text directly (without markdown formatting).
// The body Text is used for context (window, directory, etc.).
func previewExecute(t *Text, cmdText string) {
	r := []rune(cmdText)
	e := lookup(string(r), globalexectab)

	// Send commands to external client if the target window's event file is in use.
	if t.w != nil && t.w.nopen[QWevent] > 0 {
		// For preview mode with external clients, use the source-mapped positions
		delegateExecution(t, e, t.q0, t.q1, t.q0, t.q1, nil)
		return
	}

	// Invoke an internal command if it exists.
	if e != nil {
		if (e.mark && global.seltext != nil) && global.seltext.what == Body {
			global.seq++
			global.seltext.w.body.file.Mark(global.seq)
		}

		s := strings.TrimLeft(string(r), " \t\n")
		words := wsre.Split(s, 2)
		arg := ""
		if len(words) > 1 {
			arg = strings.TrimLeft(words[1], " \t\n")
		}

		e.fn(t, global.seltext, nil, e.flag1, e.flag2, arg)
		return
	}

	dir := t.DirName("")
	if t.w != nil {
		t.w.ref.Inc()
	}
	run(t.w, string(r), dir, true, "", "", false)
}

func edit(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if et == nil {
		return
	}
	r, _ := getarg(argt, false, true)

	global.seq++
	if r != "" {
		editcmd(et, []rune(r))
	} else {
		editcmd(et, []rune(arg))
	}
}

func xexit(*Text, *Text, *Text, bool, bool, string) {
	if global.row.Clean() {
		close(global.cexit)
		//	threadexits(nil);
	}
}

func del(et *Text, _0 *Text, _1 *Text, flag1 bool, _2 bool, _3 string) {
	if et.col == nil || et.w == nil {
		return
	}
	if flag1 || et.w.body.file.HasMultipleObservers() || et.w.Clean(false) {
		et.col.Close(et.w, true)
	}
}

func cut(et *Text, t *Text, _ *Text, dosnarf bool, docut bool, _ string) {
	var (
		q0, q1, c int
	)
	// if not executing a mouse chord (et != t) and snarfing (dosnarf)
	// and executed Cut or Snarf in window tag (et.w != nil),
	// then use the window body selection or the tag selection
	// or do nothing at all.
	if et != t && dosnarf && et.w != nil {
		if et.w.body.q1 > et.w.body.q0 {
			t = &et.w.body
			if docut {
				t.file.Mark(global.seq) // seq has been incremented by execute
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

		reader := t.file.Reader(q0, q1)
		buffy := new(bytes.Buffer)
		io.Copy(buffy, reader)
		global.snarfbuf = buffy.Bytes()
		acmeputsnarf()
	}
	if docut {
		t.Delete(t.q0, t.q1, true)
		t.SetSelect(t.q0, t.q0)
		if t.w != nil {
			t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
			t.w.Commit(t)
		}
	} else {
		if dosnarf { // Snarf command
			global.argtext = t
		}
	}
}

func newcol(et *Text, _ *Text, _ *Text, _, _ bool, _ string) {
	c := et.row.Add(nil, -1)
	if c != nil {
		w := c.Add(nil, nil, -1)
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
			warning(nil, "can't delete column; %s is running an external command\n", w.body.file.Name())
			return
		}
	}
	et.col.row.Close(c, true)
}

func paste(et *Text, t *Text, _ *Text, selectall bool, tobody bool, _ string) {
	var (
		c      int
		q0, q1 int
	)

	// if tobody, use body of executing window  (Paste or Send command)
	if tobody && et != nil && et.w != nil {
		t = &et.w.body
		t.file.Mark(global.seq) // seq has been incremented by execute
	}
	if t == nil {
		return
	}

	acmegetsnarf()
	if t == nil || len(global.snarfbuf) == 0 {
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
	q0 = t.q0
	// TODO(rjk): Ick. Remove undesirable conversions.
	r := []rune(string(global.snarfbuf))
	q1 = t.q0 + len(r)
	t.Insert(q0, r, true)
	if selectall {
		t.SetSelect(t.q0, q1)
	} else {
		t.SetSelect(q1, q1)
	}
	if t.w != nil {
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		t.w.Commit(t)
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
			return t.file.Name()
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
	if et.w.body.file.Nr() > 0 && !isclean {
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
	if t.file.HasMultipleObservers() && newNameIsdir {
		warning(nil, "%s is a directory; can't read with multiple windows on it\n", name)
		return
	}

	t.Delete(0, t.file.Nr(), true)
	samename := name == t.file.Name()
	t.Load(0, name, samename)

	// Text.Delete followed by Text.Load will always mark the File as
	// modified unless loading a 0-length file over a 0-length file. But if
	// samename is true here, we know that the Text.body.File is now the same
	// as it is on disk. So indicate this with file.Clean().
	if samename {
		t.file.Clean()
	}
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
		global.ckill <- cmd
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
func putfile(oeb *file.ObservableEditableBuffer, q0 int, q1 int, name string) error {
	d, err := os.Stat(name)

	// Putting to the same file that we already read from.
	if err == nil && name == oeb.Name() {
		if !os.SameFile(oeb.Info(), d) || d.ModTime().Sub(oeb.Info().ModTime()) > time.Millisecond {
			oeb.UpdateInfo(name, d)
		}

		if !os.SameFile(oeb.Info(), d) || d.ModTime().Sub(oeb.Info().ModTime()) > time.Millisecond {
			// By setting File.info here, a subsequent Put will ignore that
			// the disk file was mutated and will write File to the disk file.
			oeb.SetInfo(d)

			if oeb.Hash() == file.EmptyHash {
				// Edwood created the File but a disk file with the same name exists.
				return warnError(nil, "%s not written; file already exists", name)
			}

			// Edwood loaded the disk file to File but the disk file has been modified since.
			return warnError(nil, "%s modified since last read\n\twas %v; now %v", name, oeb.Info().ModTime(), d.ModTime())
		}
	}

	fd, err := os.OpenFile(name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return warnError(nil, "can't create file %s: %v", name, err)
	}
	defer fd.Close()

	h := sha1.New()

	d, err = fd.Stat()
	isapp := err == nil && d.Size() > 0 && (d.Mode()&os.ModeAppend) != 0
	if isapp {
		return warnError(nil, "%s not written; file is append only", name)
	}

	_, err = io.Copy(io.MultiWriter(h, fd), oeb.Reader(q0, q1))
	if err != nil {
		return warnError(nil, "can't write file %s: %v", name, err)
	}

	// Putting to the same file as the one that we originally read from.
	if name == oeb.Name() {
		if q0 != 0 || q1 != oeb.Nr() {
			// The backing disk file contents now differ from File because
			// we've over-written the disk file with part of File. There is no
			// possible sequence of undo actions that can make the file not modified.
			oeb.Modded()
		} else {
			// A normal put operation of a file modified in Edwood but not
			// modified on disk.
			if d1, err := fd.Stat(); err == nil {
				d = d1
			}
			oeb.SetInfo(d)
			oeb.Set(h.Sum(nil))
			oeb.Clean()
		}
	}
	return nil
}

// TODO(rjk): Why doesn't this handle its arguments the same as some of
// the other commands?
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
	name = UnquoteFilename(name)
	putfile(w.body.file, 0, f.Nr(), name)
	xfidlog(w, "put")
}

func putall(et, _, _ *Text, _, _ bool, arg string) {
	for _, col := range global.row.col {
		for _, w := range col.w {
			if w.nopen[QWevent] > 0 {
				continue
			}
			a := w.body.file.Name()
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

// TODO(rjk): Test the logic of Undo across multiple buffers very carefully: #383
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
	for _, c := range global.row.col {
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
// If argt has a selection (from B2-B1 chord), that text is used.
// Otherwise, if the body has a selection, it's cut and pasted at end.
// Otherwise, the snarf buffer is pasted at end.
func sendx(et *Text, _ *Text, argt *Text, _, _ bool, _ string) {
	if et.w == nil {
		return
	}
	t := &et.w.body

	// If argt is nil (e.g., from external client write-back), fall back to global.argtext
	if argt == nil {
		argt = global.argtext
	}

	// If we have an argument from B2-B1 chord, use it
	if argt != nil && argt.q0 != argt.q1 {
		n := argt.q1 - argt.q0
		r := make([]rune, n)
		argt.file.Read(argt.q0, r)
		t.SetSelect(t.file.Nr(), t.file.Nr())
		t.Insert(t.file.Nr(), r, true)
		t.SetSelect(t.file.Nr(), t.file.Nr())
		if len(r) == 0 || r[len(r)-1] != '\n' {
			t.Insert(t.file.Nr(), []rune("\n"), true)
			t.SetSelect(t.file.Nr(), t.file.Nr())
		}
	} else {
		// Original behavior: use body selection or snarf buffer
		if t.q0 != t.q1 {
			cut(t, t, nil, true, false, "")
		}
		t.SetSelect(t.file.Nr(), t.file.Nr())
		paste(t, t, nil, true, true, "")
		if t.ReadC(t.file.Nr()-1) != '\n' {
			t.Insert(t.file.Nr(), []rune("\n"), true)
			t.SetSelect(t.file.Nr(), t.file.Nr())
		}
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
			t.file.Read(t.q0, rb[:n])
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
		warning(nil, "%s: Tab %d\n", w.body.file.Name(), w.body.tabstop)
	}
}

func expandtab(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if et == nil || et.w == nil {
		return
	}
	w := et.w
	if w.body.tabexpand {
		w.body.tabexpand = false
		warning(nil, "%s: Tab: %d, Tabexpand OFF\n", w.body.file.Name(), w.body.tabstop)
	} else {
		w.body.tabexpand = true
		warning(nil, "%s: Tab: %d, Tabexpand ON\n", w.body.file.Name(), w.body.tabstop)
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

	if newfont := fontget(file, global.row.display); newfont != nil {
		// TODO(rjk): maybe Frame should know how to clear itself on init?
		global.row.display.ScreenImage().Draw(t.w.r, global.textcolors[frame.ColBack], nil, image.Point{})
		t.font = file
		t.fr.Init(t.w.r, frame.OptFont(newfont), frame.OptBackground(global.row.display.ScreenImage()))

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
		warning(nil, "%s is a directory; Zerox illegal\n", t.file.Name())
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
		global.ccommand <- c
	} else {
		if c.iseditcommand {
			global.cedit <- 0
		}
	}
	cpid = nil
}

var errEmptyCmd = fmt.Errorf("empty command")

func setupenvvars(filename, argaddr string, winid int) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("winid=%d", winid))
	if filename != "" {
		env = append(env, fmt.Sprintf("%%=%v", filename))
		env = append(env, fmt.Sprintf("samfile=%v", filename))
	}
	if argaddr != "" {
		env = append(env, fmt.Sprintf("acmeaddr=%v", argaddr))
	}
	return env
}

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
		shell = global.acmeshell
		if shell == "" {
			shell = "rc"
		}
		rcarg = []string{shell, "-c", t}

		cmd := exec.Command(rcarg[0], rcarg[1:]...)
		cmd.Env = setupenvvars(filename, argaddr, winid)
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
			global.cwait <- cmd.ProcessState
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
			filename = win.body.file.Name()
			winid = win.id
			incl = append([]string{}, win.incl...)
			win.lk.Unlock()
		} else {
			filename = ""
			winid = 0
			if global.activewin != nil {
				winid = global.activewin.id
			}
		}
		// 	rfork(RFNAMEG|RFENVG|RFFDG|RFNOTEG); TODO(flux): I'm sure these settings are important

		var fs *client.Fsys
		var err error
		c.md, fs, err = fsysmount(dir, incl)
		if err != nil {
			Fail()
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
					buf = "editout"
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

	if global.acmeshell != "" {
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
	cmd.Env = setupenvvars(filename, argaddr, winid)
	err := cmd.Start()
	if err != nil {
		Fail()
		return err
	}
	cpid <- cmd.Process
	go func() {
		cmd.Wait()
		Closeall()
		global.cwait <- cmd.ProcessState
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
		global.row.AllWindows(func(w *Window) { w.autoindent = *globalAutoIndent })
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
		global.row.Dump(name)
	} else {
		global.row.Load(nil, name, false)
	}
}

// previewcmd opens a markdown preview window for the current file.
func previewcmd(et *Text, _ *Text, _ *Text, _, _ bool, _ string) {
	if et == nil || et.w == nil {
		return
	}
	w := et.w
	t := &w.body

	// Get the file name
	name := t.file.Name()
	if name == "" {
		warning(nil, "Markdeep: no file name\n")
		return
	}

	// Check if it's a markdown file
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		warning(nil, "Markdeep: %s is not a markdown file\n", name)
		return
	}

	// If already in preview mode, toggle it off
	if w.IsPreviewMode() {
		// Cancel pending preview update timer
		if w.previewUpdateTimer != nil {
			w.previewUpdateTimer.Stop()
			w.previewUpdateTimer = nil
		}
		// Clean up image cache before exiting preview mode
		if w.imageCache != nil {
			w.imageCache.Clear()
			w.imageCache = nil
		}
		w.SetPreviewMode(false)
		// Scroll the source view to make the current selection visible.
		// Show() handles ScrDraw + SetSelect + scrolling if off-screen.
		w.body.Show(w.body.q0, w.body.q1, true)
		if w.display != nil {
			w.display.Flush()
		}
		return
	}

	// Enter preview mode - initialize the richBody if needed
	display := w.display
	if display == nil {
		display = global.row.display
	}

	// Get the body rectangle for the rich text renderer
	bodyRect := w.body.all

	// Get the font and font variants for styled text
	font := fontget(global.tagfont, display)
	boldFont := tryLoadFontVariant(display, global.tagfont, "bold")
	italicFont := tryLoadFontVariant(display, global.tagfont, "italic")
	boldItalicFont := tryLoadFontVariant(display, global.tagfont, "bolditalic")
	codeFont := tryLoadCodeFont(display, global.tagfont)

	// Get scaled fonts for headings (H1=2.0, H2=1.5, H3=1.25)
	h1Font := tryLoadScaledFont(display, global.tagfont, 2.0)
	h2Font := tryLoadScaledFont(display, global.tagfont, 1.5)
	h3Font := tryLoadScaledFont(display, global.tagfont, 1.25)

	// Create or reinitialize the rich text renderer
	rt := NewRichText()

	// Allocate colors for the preview
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	if err != nil {
		warning(nil, "Markdeep: failed to allocate background: %v\n", err)
		return
	}
	textImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)
	if err != nil {
		warning(nil, "Markdeep: failed to allocate text color: %v\n", err)
		return
	}

	// Build RichText options
	// Use the same selection highlight color as normal body text (Darkyellow)
	rtOpts := []RichTextOption{
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
		WithRichTextSelectionColor(global.textcolors[frame.ColHigh]),
		WithScrollbarColors(global.textcolors[frame.ColBord], global.textcolors[frame.ColBack]),
	}
	if boldFont != nil {
		rtOpts = append(rtOpts, WithRichTextBoldFont(boldFont))
	}
	if italicFont != nil {
		rtOpts = append(rtOpts, WithRichTextItalicFont(italicFont))
	}
	if boldItalicFont != nil {
		rtOpts = append(rtOpts, WithRichTextBoldItalicFont(boldItalicFont))
	}
	if codeFont != nil {
		rtOpts = append(rtOpts, WithRichTextCodeFont(codeFont))
	}
	if h1Font != nil {
		rtOpts = append(rtOpts, WithRichTextScaledFont(2.0, h1Font))
	}
	if h2Font != nil {
		rtOpts = append(rtOpts, WithRichTextScaledFont(1.5, h2Font))
	}
	if h3Font != nil {
		rtOpts = append(rtOpts, WithRichTextScaledFont(1.25, h3Font))
	}

	// Initialize the image cache for loading images in the markdown
	// Must be created before rt.Init() so it can be passed via options
	w.imageCache = rich.NewImageCache(0) // 0 means use default size
	rtOpts = append(rtOpts, WithRichTextImageCache(w.imageCache))

	// Wire async image load completion callback. When a cache-miss image
	// finishes loading in the background, re-render the preview to replace
	// the placeholder with the actual image. This is lightweight — no
	// markdown re-parse, just a layout+draw pass with the now-cached image.
	rtOpts = append(rtOpts, WithRichTextOnImageLoaded(func(path string) {
		go func() {
			global.row.lk.Lock()
			defer global.row.lk.Unlock()
			if !w.previewMode || w.richBody == nil {
				return
			}
			w.richBody.Render(w.body.all)
			if w.display != nil {
				w.display.Flush()
			}
		}()
	}))

	// Set the base path for resolving relative image paths
	// The name variable contains the file path from the window tag
	// Convert to absolute path for proper image resolution regardless of working directory
	basePath := name
	if !filepath.IsAbs(basePath) {
		if abs, err := filepath.Abs(basePath); err == nil {
			basePath = abs
		}
	}
	rtOpts = append(rtOpts, WithRichTextBasePath(basePath))

	rt.Init(display, font, rtOpts...)

	// Parse the markdown content with source mapping and link tracking
	mdContent := t.file.String()
	content, sourceMap, linkMap := markdown.ParseWithSourceMap(mdContent)

	rt.SetContent(content)

	// Set up the window's preview components
	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)

	// Map the source cursor/selection position to the rendered position
	// so the cursor appears in the corresponding place in the preview.
	rendStart, rendEnd := sourceMap.ToRendered(w.body.q0, w.body.q1)
	if rendStart >= 0 && rendEnd >= 0 {
		rt.SetSelection(rendStart, rendEnd)
	}

	// Enter preview mode
	w.SetPreviewMode(true)

	// Render the preview, then scroll to make the selection visible.
	// Must render first so the frame has layout data for scrollPreviewToMatch.
	rt.Render(bodyRect)
	if rendStart >= 0 {
		w.scrollPreviewToMatch(rt, rendStart)
	}
	w.Draw()
	if display != nil {
		display.Flush()
	}
}
