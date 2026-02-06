package main

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/util"
)

var (
	plumbsendfid *client.Fid
	nuntitled    int
)

type plumbClient interface {
	Open(name string, mode uint8) (*client.Fid, error)
}

// Handle a plumber connection and return only if we lose the connection.
func handlePlumb(fsys plumbClient) {
	var err error
	plumbsendfid, err = fsys.Open("send", plan9.OWRITE|plan9.OCEXEC)
	if err != nil {
		return
	}
	defer func() {
		plumbsendfid.Close()
		plumbsendfid = nil
	}()

	editfid, err := fsys.Open("edit", plan9.OREAD|plan9.OCEXEC)
	if err != nil {
		return
	}
	defer editfid.Close()

	br := bufio.NewReader(editfid)
	// Relay messages.
	for {
		var m plumb.Message
		err := m.Recv(br)
		if err != nil {
			return
		}
		global.cplumb <- &m
	}
}

func startplumbing() {
	global.cplumb = make(chan *plumb.Message)
	go plumbthread()
}

func look3(t *Text, q0 int, q1 int, external bool) {
	var (
		n, c, f int
		ct      *Text
		r       []rune
		//m *Plumbmsg
		//dir string
	)

	ct = global.seltext
	if ct == nil {
		global.seltext = t
	}
	e, expanded := expand(t, q0, q1)
	if !external && t.w != nil && t.w.nopen[QWevent] > 0 {
		// send alphanumeric expansion to external client
		if !expanded {
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
			t.file.Read(q0, r)
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
				e.at.file.Read(e.a0, r[nlen+1:nlen+1+e.a1-e.a0])
			}
		} else {
			n = e.q1 - e.q0
			r = make([]rune, n)
			t.file.Read(e.q0, r)
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
		m, err := look3Message(t, q0, q1)
		if err != nil {
			return
		}
		if m.Send(plumbsendfid) == nil {
			return
		}
	}
	// interpret alphanumeric string ourselves
	if !expanded {
		return
	}
	if e.name != "" || e.at != nil {
		e.agetc = func(q int) rune { return e.at.ReadC(q) }
		openfile(t, e)
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
		t.file.Read(e.q0, r)
		if search(ct, r[:n]) && e.jump {
			if ct.w != nil && ct.w.IsPreviewMode() && ct.w.richBody != nil && ct.w.previewSourceMap != nil {
				rendStart, _ := ct.w.previewSourceMap.ToRendered(ct.q0, ct.q1)
				if rendStart >= 0 {
					warpPt := ct.w.richBody.Frame().Ptofchar(rendStart).Add(
						image.Pt(4, ct.w.richBody.Frame().DefaultFontHeight()-4))
					global.row.display.MoveTo(warpPt)
				}
			} else {
				global.row.display.MoveTo(ct.fr.Ptofchar(getP0(ct.fr)).Add(image.Pt(4, ct.fr.DefaultFontHeight()-4)))
			}
		}
	}
}

// look3Message generates a plumb message for the text in t at range [q0, q1).
// If q0 == q1, the range will be expanded to the current selection if q0/q1 falls
// within the selection. Otherwise, it'll expand to a whitespace-delimited word.
func look3Message(t *Text, q0, q1 int) (*plumb.Message, error) {
	m := &plumb.Message{
		Src:  "acme",
		Dst:  "",
		Dir:  t.AbsDirName(""),
		Type: "text",
	}
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
			for q1 < t.file.Nr() {
				// TODO(rjk): utf8 conversion change point.
				c := t.ReadC(q1)
				if !(c != ' ' && c != '\t' && c != '\n') {
					break
				}
				q1++
			}
			if q1 == q0 {
				return nil, fmt.Errorf("empty selection")
			}
			s := fmt.Sprintf("%d", p-q0)
			m.Attr = &plumb.Attribute{Name: "click", Value: s, Next: nil}
		}
	}
	r := make([]rune, q1-q0)
	t.file.Read(q0, r[:q1-q0])
	m.Data = []byte(string(r[:q1-q0]))
	return m, nil
}

func plumblook(m *plumb.Message) {
	var e Expand

	if len(m.Data) >= BUFSIZE {
		warning(nil, "insanely long file name (%d bytes) in plumb message (%.32s...)\n", len(m.Data), m.Data)
		return
	}
	e.q0 = 0
	e.q1 = 0
	if len(m.Data) == 0 {
		return
	}
	e.name = string(m.Data)
	e.jump = true
	e.a0 = 0
	e.a1 = 0
	addr := findattr(m.Attr, "addr")
	if addr != "" {
		ar := []rune(addr)
		e.a1 = len(ar)
		e.agetc = func(q int) rune { return ar[q] }
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
	if !filepath.IsAbs(name) && m.Dir != "" {
		name = filepath.Join(m.Dir, name)
	}
	name = filepath.Clean(name)
	r, _, _ := util.Cvttorunes([]byte(name), len(name)) // remove nulls
	name = string(r)
	w.SetName(name)
	r, _, _ = util.Cvttorunes(m.Data, len(m.Data))
	w.body.Insert(0, r, true)
	w.body.file.Clean()
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.tag.SetSelect(w.tag.Nc(), w.tag.Nc())
	xfidlog(w, "new")
}

func search(ct *Text, r []rune) bool {
	n := len(r)
	if n > RBUFSIZE {
		warning(nil, "string too long\n")
		return false
	}

	res := regexp.QuoteMeta(string(r))
	// Unless QuoteMeta has a bug, this will always work.
	regexp := regexp.MustCompile(res)

	start := ct.file.RuneTuple(ct.q1)
	end := ct.file.End()
	cursor := ct.file.MakeBufferCursor(start, end)

	loc := regexp.FindReaderIndex(cursor)
	if loc == nil {
		// Try wrapped around.
		cursor = ct.file.MakeBufferCursor(file.Ot(0, 0), start)
		start = file.Ot(0, 0)
		loc = regexp.FindReaderIndex(cursor)
	}

	if loc == nil {
		return false
	}

	// loc is w.r.t. start right?
	q1 := ct.file.ByteTuple(start.B + loc[1]).R
	q0 := ct.file.ByteTuple(start.B + loc[0]).R

	if ct.w != nil {
		ct.Show(q0, q1, true)
	} else {
		ct.q0 = q0
		ct.q1 = q1
	}
	global.seltext = ct
	return true

}

func isfilespace(r rune) bool {
	Lx := " \t"
	return strings.ContainsRune(Lx, r)
}

func isfilec(r rune) bool {
	Lx := ".-+/:@\\"
	if isalnum(r) {
		return true
	}
	if strings.ContainsRune(Lx, r) {
		return true
	}
	return false
}

// Runestr wrapper for cleanname
func cleanrname(rs []rune) []rune {
	s := filepath.Clean(string(rs))
	r, _, _ := util.Cvttorunes([]byte(s), len(s))
	return r
}

func access(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func expand(t *Text, q0 int, q1 int) (*Expand, bool) {
	var e Expand
	e.agetc = func(q int) rune {
		if q < t.Nc() {
			return t.ReadC(q)
		}
		return 0
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
		return &e, true
	}

	if q0 == q1 {
		for q1 < t.file.Nr() && isalnum(t.ReadC(q1)) {
			q1++
		}
		for q0 > 0 && isalnum(t.ReadC(q0-1)) {
			q0--
		}
	}
	e.q0 = q0
	e.q1 = q1
	return &e, q1 > q0
}

func lookfile(s string) *Window {
	// avoid terminal slash on directories
	s = UnquoteFilename(s)
	s = strings.TrimRight(s, "\\/")
	for _, c := range global.row.col {
		for _, w := range c.w {
			name := UnquoteFilename(w.body.file.Name())
			k := strings.TrimRight(name, "\\/")
			if k == s {
				cur, ok := w.body.file.GetCurObserver().(*Text)
				if !ok {
					return nil
				}
				w = cur.w
				if w.col != nil { // protect against race deleting w
					return w
				}
			}
		}
	}
	return nil
}

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
		if w == nil && !filepath.IsAbs(e.name) {
			// Unrooted path in new window.
			// This can happen if we type a pwd-relative path
			// in the topmost tag or the column tags.
			// Most of the time plumber takes care of these,
			// but plumber might not be running or might not
			// be configured to accept plumbed directories.
			// Make the name a full path, just like we would if
			// opening via the plumber.
			rp := filepath.Join(global.wdir, e.name)
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
		t.Load(0, e.name, true)
		t.file.Clean()
		// Auto-enable preview mode for markdown files
		if strings.HasSuffix(strings.ToLower(e.name), ".md") {
			previewcmd(&w.body, nil, nil, false, false, "")
		}
		t.w.tag.SetSelect(t.w.tag.file.Nr(), t.w.tag.file.Nr())
		if ow != nil {
			for _, inc := range ow.incl {
				w.AddIncl(inc)
			}
			w.autoindent = ow.autoindent
		} else {
			w.autoindent = *globalAutoIndent
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
		if !eval {
			e.jump = false // don't jump if invalid address
		}
	}
	if !eval {
		r.q0 = t.q0
		r.q1 = t.q1
	}
	t.Show(r.q0, r.q1, true)
	global.seltext = t
	if e.jump {
		if t.w != nil && t.w.IsPreviewMode() && t.w.richBody != nil && t.w.previewSourceMap != nil {
			rendStart, _ := t.w.previewSourceMap.ToRendered(r.q0, r.q1)
			if rendStart >= 0 {
				warpPt := t.w.richBody.Frame().Ptofchar(rendStart).Add(
					image.Pt(4, t.w.richBody.Frame().DefaultFontHeight()-4))
				global.row.display.MoveTo(warpPt)
			}
		} else {
			global.row.display.MoveTo(t.fr.Ptofchar(getP0(t.fr)).Add(image.Pt(4, fontget(global.tagfont, global.row.display).Height()-4)))
		}
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
	s := wsre.ReplaceAllString(arg, " ")
	filenames := strings.Split(s, " ")
	if len(filenames) == 1 && filenames[0] == "" && et.col != nil {
		w := et.col.Add(nil, nil, -1)
		// Note special case for empty windows.
		w.ForceSetWindowTag()
		xfidlog(w, "new")
		return
	}

	for _, f := range filenames {
		openfile(et, &Expand{
			name: et.DirName(f),
			jump: true,
		})
	}
}
