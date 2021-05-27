package main

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/internal/runes"
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
		cplumb <- &m
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
		r       []rune
		//m *Plumbmsg
		//dir string
	)

	ct = seltext
	if ct == nil {
		seltext = t
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
		t.file.b.Read(e.q0, r)
		if search(ct, r[:n]) && e.jump {
			row.display.MoveTo(ct.fr.Ptofchar(getP0(ct.fr)).Add(image.Pt(4, ct.fr.DefaultFontHeight()-4)))
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
			for q1 < t.file.Size() {
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
	t.file.b.Read(q0, r[:q1-q0])
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
	r, _, _ := cvttorunes([]byte(name), len(name)) // remove nulls
	name = string(r)
	w.SetName(name)
	r, _, _ = cvttorunes(m.Data, len(m.Data))
	w.body.Insert(0, r, true)
	w.body.file.Clean()
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
			ci := runes.IndexRune(s[bi:bi+nb], r[0])
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
		if runes.Equal(s[bi:limit], r) {
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
	Lx := ".-+/:@"
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
	r, _, _ := cvttorunes([]byte(s), len(s))
	return r
}

func expandfile(t *Text, q0 int, q1 int, e *Expand) (success bool) {
	amax := q1
	if q1 == q0 {
		colon := int(-1)
		// TODO(rjk): utf8 conversion work.
		for q1 < t.file.Size() {
			c := t.ReadC(q1)
			if !isfilec(c) {
				break
			}
			if c == ':' {
				colon = q1
				break
			}
			q1++
		}
		for q0 > 0 {
			c := t.ReadC(q0 - 1)
			if !isfilec(c) && !isaddrc(c) && !isregexc(c) {
				break
			}
			if colon < 0 && c == ':' {
				colon = q0 - 1
			}
			q0--
		}
		// if it looks like it might begin file: , consume address chars after :
		// otherwise terminate expansion at :
		if colon >= 0 {
			q1 = colon
			if colon < t.file.Size()-1 {
				c := t.ReadC(colon + 1)
				if isaddrc(c) {
					q1 = colon + 1
					for q1 < t.file.Size() {
						c := t.ReadC(q1)
						if !isaddrc(c) {
							break
						}
						q1++
					}
				}
			}
		}
		if q1 > q0 {
			if colon >= 0 { // stop at white space
				for amax = colon + 1; amax < t.file.Size(); amax++ {
					c := t.ReadC(amax)
					if c == ' ' || c == '\t' || c == '\n' {
						break
					}
				}
			} else {
				amax = t.file.Size()
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
	// first, does it have bad chars?
	nname := -1
	for i, c := range rb {
		if c == ':' && nname < 0 {
			if q0+i+1 >= t.file.Size() {
				return false
			}
			if i != n-1 {
				if cc := t.ReadC(q0 + i + 1); !isaddrc(cc) {
					return false
				}
			}
			amin = q0 + i
			nname = i
		}
	}
	if nname == -1 {
		nname = n
	}
	for i := 0; i < nname; i++ {
		if !isfilec(rb[i]) && rb[i] != ' ' {
			return false
		}
	}
	isFile := func(name string) bool {
		e.name = name
		e.at = t
		e.a0 = amin + 1
		_, _, e.a1 = address(true, nil, Range{-1, -1}, Range{0, 0}, e.a0, amax,
			func(q int) rune { return t.ReadC(q) }, false)
		return true
	}
	s := string(rb[:nname])
	if amin == q0 {
		return isFile(s)
	}
	dname := t.DirName(s)
	// if it's already a window name, it's a file
	if lookfile(dname) != nil {
		return isFile(dname)
	}
	// if it's the name of a file, it's a file
	if ismtpt(dname) || !access(dname) {
		return false
	}
	return isFile(dname)
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
		for q1 < t.file.Size() && isalnum(t.ReadC(q1)) {
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
			rp := filepath.Join(wdir, e.name)
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
		t.w.SetTag()
		t.w.tag.SetSelect(t.w.tag.file.Size(), t.w.tag.file.Size())
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
	t.w.SetTag()
	seltext = t
	if e.jump {
		row.display.MoveTo(t.fr.Ptofchar(getP0(t.fr)).Add(image.Pt(4, fontget(tagfont, row.display).Height()-4)))
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
		w.SetTag()
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
