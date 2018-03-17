package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"9fans.net/go/draw"
	"9fans.net/go/plan9"
)

const Ctlsize = 5 * 12

var Edel = fmt.Errorf("deleted window")
var Ebadctl = fmt.Errorf("ill-formed control message")
var Ebadaddr = fmt.Errorf("bad address syntax")
var Eaddr = fmt.Errorf("address out of range")
var Einuse = fmt.Errorf("already in use")
var Ebadevent = fmt.Errorf("bad event syntax")

func clampaddr(w *Window) {
	if w.addr.q0 < 0 {
		w.addr.q0 = 0
	}
	if w.addr.q1 < 0 {
		w.addr.q1 = 0
	}
	if w.addr.q0 > int(w.body.file.b.nc()) {
		w.addr.q0 = int(w.body.file.b.nc())
	}
	if w.addr.q1 > int(w.body.file.b.nc()) {
		w.addr.q1 = int(w.body.file.b.nc())
	}
}

func xfidctl(x *Xfid, d *draw.Display) {
	for {
		select {
		case f := <-x.c:
			f(x)
			if d != nil {
				d.Flush()
			} // d here is for testability.
			cxfidfree <- x
			//		case <-exit:
			//			return
		}
	}
}

func xfidflush(x *Xfid) {
	var (
		fc plan9.Fcall
		wx *Xfid
	)

	xfidlogflush(x)

	// search windows for matching tag
	row.lk.Lock()
	defer row.lk.Unlock()
	for _, c := range row.col {
		for _, w := range c.w {
			w.Lock('E')
			wx = w.eventx
			if wx != nil && wx.fcall.Tag == x.fcall.Oldtag {
				w.eventx = nil
				wx.flushed = true
				wx.c <- nil
				w.Unlock()
				goto out
			}
			w.Unlock()
		}
	}
out:
	respond(x, &fc, nil)
}

func xfidopen(x *Xfid) {
	var (
		fc     plan9.Fcall
		w      *Window
		t      *Text
		n      int
		q0, q1 int
		q      uint64
	)

	w = x.f.w
	t = &w.body
	q = FILE(x.f.qid)
	if w != nil {
		w.Lock('E')
		switch q {
		case QWaddr:
			if w.nopen[q] == 0 {
				w.addr = Range{0, 0}
				w.limit = Range{-1, -1}
			}
			w.nopen[q]++
		case QWdata:
			fallthrough
		case QWxdata:
			w.nopen[q]++
		case QWevent:
			if w.nopen[q] == 0 {
				if !w.isdir && w.col != nil {
					w.filemenu = false
					w.SetTag()
				}
			}
			w.nopen[q]++
		case QWrdsel:
			//* Use a temporary file.
			//* A pipe would be the obvious, but we can't afford the
			//* broken pipe notification.  Using the code to read QWbody
			//* is n², which should probably also be fixed.  Even then,
			//* though, we'd need to squirrel away the data in case it's
			//* modified during the operation, e.g. by |sort
			if w.rdselfd != nil {
				w.Unlock()
				respond(x, &fc, Einuse)
				return
			}
			var err error
			w.rdselfd, err = ioutil.TempFile("", "acme")
			if err != nil {
				w.Unlock()
				respond(x, &fc, fmt.Errorf("can't create temp file"))
				return
			}
			w.nopen[q]++
			q0 = t.q0
			q1 = t.q1
			for q0 < q1 {
				n = int(q1 - q0)
				if n > BUFSIZE/utf8.UTFMax {
					n = BUFSIZE / utf8.UTFMax
				}
				r := t.file.b.Read(q0, (n))
				s := string(r)
				n, err = w.rdselfd.Write([]byte(s))
				if err != nil || n != len(s) {
					warning(nil, fmt.Sprintf("can't write temp file for pipe command %v\n", err))
					break
				}
				q0 += (n)
			}
		case QWwrsel:
			w.nopen[q]++
			seq++
			t.file.Mark()
			cut(t, t, nil, false, true, nil, 0)
			w.wrselrange = Range{int(t.q1), int(t.q1)}
			w.nomark = true
		case QWeditout:
			if editing == Inactive {
				w.Unlock()
				respond(x, &fc, Eperm)
				return
			}

			// TODO(flux): Need a better mechanism for editoutlk
			//	if !w.editoutlk.CanLock() {
			//		w.Unlock();
			//		respond(x, &fc, Einuse);
			//		return;
			//	}
			w.wrselrange = Range{int(t.q1), int(t.q1)}
			break
		}
		w.Unlock()
	} else {
		switch q {
		case Qlog:
			xfidlogopen(x)
		case Qeditout:
			// TODO(flux) CanLock doesn't exist :-(
			//	if !editoutlk.CanLock() {
			//		respond(x, &fc, Einuse);
			//		return;
			//	}
		}
	}
	fc.Qid = x.f.qid
	fc.Iounit = uint32(messagesize - plan9.IOHDRSZ)
	x.f.open = true
	respond(x, &fc, nil)
}

func xfidclose(x *Xfid) {
	var (
		fc plan9.Fcall
		w  *Window
		q  uint64
		t  *Text
	)
	w = x.f.w
	x.f.busy = false
	x.f.w = nil
	if x.f.open == false {
		if w != nil {
			w.Close()
		}
		respond(x, &fc, nil)
		return
	}

	q = FILE(x.f.qid)
	x.f.open = false
	if w != nil {
		w.Lock('E')
		switch q {
		case QWctl:
			if w.ctlfid != MaxFid && w.ctlfid == x.f.fid {
				w.ctlfid = MaxFid
				w.ctrllock.Unlock()
			}
			break
		case QWdata:
		case QWxdata:
			w.nomark = false
			// fall through
		case QWaddr:
		case QWevent: // BUG: do we need to shut down Xfid?
			w.nopen[q]--
			if w.nopen[q] == 0 {
				if q == QWdata || q == QWxdata {
					w.nomark = false
				}
				if q == QWevent && !w.isdir && w.col != nil {
					w.filemenu = true
					w.SetTag()
				}
				if q == QWevent {
					w.dumpstr = ""
					w.dumpdir = ""
				}
			}
			break
		case QWrdsel:
			w.rdselfd.Close()
			w.rdselfd = nil
			break
		case QWwrsel:
			w.nomark = false
			t = &w.body
			// before: only did this if !w.noscroll, but that didn't seem right in practice
			t.Show(min((w.wrselrange.q0), t.file.b.nc()), min((w.wrselrange.q1), t.file.b.nc()), true)
			t.ScrDraw()
			break
		case QWeditout:
			w.editoutlk.Unlock()
			break
		}
		w.Unlock()
		w.Close()
	} else {
		switch q {
		case Qeditout:
			editoutlk.Unlock()
			break
		}
	}
	respond(x, &fc, nil)
}

func xfidread(x *Xfid) {
	var (
		fc  plan9.Fcall
		n   int
		b   string
		buf string
		w   *Window
	)
	q := FILE(x.f.qid)
	w = x.f.w
	if w == nil {
		fc.Count = 0
		switch q {
		case Qcons:
		case Qlabel:
			break
		case Qindex:
			xfidindexread(x)
			return
		case Qlog:
			xfidlogread(x)
			return
		default:
			warning(nil, "unknown qid %d\n", q)
			break
		}
		respond(x, &fc, nil)
		return
	}
	w.Lock('F')
	if w.col == nil {
		w.Unlock()
		respond(x, &fc, Edel)
		return
	}
	off := x.fcall.Offset
	switch q {
	case QWaddr:
		w.body.Commit(true)
		clampaddr(w)
		buf := fmt.Sprintf("%11d %11d ", w.addr.q0, w.addr.q1)
		n = len(buf)
		if off > uint64(n) {
			off = uint64(n)
		}
		if off+uint64(x.fcall.Count) > uint64(n) {
			x.fcall.Count = uint32(uint64(n) - off)
		}
		fc.Count = x.fcall.Count
		fc.Data = []byte(buf[off:])
		respond(x, &fc, nil)
	case QWbody:
		xfidutfread(x, &w.body, w.body.file.b.nc(), int(QWbody))

	case QWctl:
		b = w.CtlPrint(true)
		n = len(b)
		if off > uint64(n) {
			off = uint64(n)
		}
		if off+uint64(x.fcall.Count) > uint64(n) {
			x.fcall.Count = uint32(uint64(n) - off)
		}
		fc.Count = x.fcall.Count
		fc.Data = []byte(buf[off:])
		respond(x, &fc, nil)

	case QWevent:
		xfideventread(x, w)

	case QWdata:
		// BUG: what should happen if q1 > q0?
		if w.addr.q0 > int(w.body.file.b.nc()) {
			respond(x, &fc, Eaddr)
			break
		}
		w.addr.q0 += xfidruneread(x, &w.body, (w.addr.q0), w.body.file.b.nc())
		w.addr.q1 = w.addr.q0

	case QWxdata:
		// BUG: what should happen if q1 > q0?
		if w.addr.q0 > int(w.body.file.b.nc()) {
			respond(x, &fc, Eaddr)
			break
		}
		w.addr.q0 += xfidruneread(x, &w.body, (w.addr.q0), (w.addr.q1))

	case QWtag:
		xfidutfread(x, &w.tag, w.tag.file.b.nc(), int(QWtag))

	case QWrdsel:
		w.rdselfd.Seek(int64(off), 0)
		n := x.fcall.Count
		if n > BUFSIZE {
			n = BUFSIZE
		}
		b := make([]byte, n)
		nread, err := w.rdselfd.Read(b)
		n = uint32(nread)
		if err != nil || n < 0 {
			respond(x, &fc, fmt.Errorf("I/O error in temp file: %v", err))
			break
		}
		fc.Count = n
		fc.Data = b
		respond(x, &fc, nil)
	default:
		respond(x, &fc, fmt.Errorf("unknown qid %d in read", q)) // TODO(flux) compare to the C code - there's a bug and leaks buf, not even passing it.
	}
	w.Unlock()
}

func shouldscroll(t *Text, q0 int, qid uint64) bool {
	if qid == Qcons {
		return true
	}
	return t.org <= q0 && q0 <= t.org+(t.fr.NChars)
}

// This is fiddly code that handles partial runes at the end of a previous write?
func fullrunewrite(x *Xfid) []rune {
	var (
		nb uint32
		r  []rune
	)
	// extend with previous partial rune at the end.
	x.fcall.Data = append(x.f.rpart[0:x.f.nrpart], x.fcall.Data...)
	cnt := x.fcall.Count + uint32(x.f.nrpart)
	r = []rune(string(x.fcall.Data[:cnt-utf8.UTFMax]))
	// approach end of buffer, decoding the last utf8.UTFMax bytes, which might include an incomplete utf8 sequence
	for nb = cnt - utf8.UTFMax; utf8.FullRune(x.fcall.Data[nb:]); {
		ru, l := utf8.DecodeRune(x.fcall.Data[nb:])
		r = append(r, ru)
		nb += uint32(l)
	}
	if nb < cnt {
		copy(x.f.rpart[:], x.fcall.Data[nb:cnt-nb])
		x.f.nrpart = int(cnt - nb)
	}
	return r
}

func xfidwrite(x *Xfid) {
	var (
		fc               plan9.Fcall
		c                int
		eval             bool
		r                []rune
		a                Range
		t                *Text
		q0, tq0, tq1, nb int
		err              error
	)

	qid := FILE(x.f.qid)
	w := x.f.w
	if w != nil {
		c = 'F'
		if qid == QWtag || qid == QWbody {
			c = 'E'
		}
		w.Lock(c)
		if w.col == nil {
			w.Unlock()
			respond(x, &fc, Edel)
			return
		}
	}

	BodyTag := func() { // Trimmed from the switch below.
		r := fullrunewrite(x)
		if len(r) != 0 {
			w.Commit(t)
			if qid == QWwrsel {
				q0 = (w.wrselrange.q1)
				if q0 > t.file.b.nc() {
					q0 = t.file.b.nc()
				}
			} else {
				q0 = t.file.b.nc()
			}
			if qid == QWtag {
				t.Insert(q0, r, true)
			} else {
				if w.nomark == false {
					seq++
					t.file.Mark()
				}
				q, nr := t.BsInsert(q0, r, true) // TODO(flux): BsInsert returns nr?
				q0 = q
				t.SetSelect(t.q0, t.q1) // insert could leave it somewhere else
				if qid != QWwrsel && shouldscroll(t, q0, qid) {
					t.Show(q0+(nr), q0+(nr), true)
				}
				t.ScrDraw()
			}
			w.SetTag()
			if qid == QWwrsel {
				w.wrselrange.q1 += len(r)
			}
		}
		fc.Count = x.fcall.Count
		respond(x, &fc, nil)
	}

	//x.fcall.Data[x.fcall.Count] = 0; // null-terminate. unneeded
	switch qid {
	case Qcons:
		w = errorwin(x.f.mntdir, 'X')
		t = &w.body
		BodyTag()

	case Qlabel:
		fc.Count = x.fcall.Count
		respond(x, &fc, nil)

	case QWaddr:
		//x.fcall.Data[x.fcall.Count] = 0;// null-terminate. unneeded
		r = []rune(string(x.fcall.Data))
		t = &w.body
		w.Commit(t)
		eval = true
		a, eval, nb = address(false, t, w.limit, w.addr, r, 0, (len(r)))
		if nb < (len(r)) {
			respond(x, &fc, Ebadaddr)
			break
		}
		if !eval {
			respond(x, &fc, Eaddr)
			break
		}
		w.addr = a
		fc.Count = x.fcall.Count
		respond(x, &fc, nil)
		break

	case Qeditout:
	case QWeditout:
		r = fullrunewrite(x)
		if w != nil {
			err = edittext(w, w.wrselrange.q1, r)
		} else {
			err = edittext(nil, 0, r)
		}
		if err != nil {
			respond(x, &fc, err)
			break
		}
		fc.Count = x.fcall.Count
		respond(x, &fc, nil)
		break

	case QWerrors:
		w = errorwinforwin(w)
		t = &w.body
		BodyTag()

	case QWbody:
	case QWwrsel:
		t = &w.body
		BodyTag()

	case QWctl:
		xfidctlwrite(x, w)
		break

	case QWdata:
		a = w.addr
		t = &w.body
		w.Commit(t)
		if a.q0 > int(t.file.b.nc()) || a.q1 > int(t.file.b.nc()) {
			respond(x, &fc, Eaddr)
			break
		}
		r := []rune(string(x.fcall.Data[0:x.fcall.Count]))
		if w.nomark == false {
			seq++
			t.file.Mark()
		}
		q0 = (a.q0)
		if a.q1 > (q0) {
			t.Delete(q0, (a.q1), true)
			w.addr.q1 = (q0)
		}
		tq0 = t.q0
		tq1 = t.q1
		t.Insert(q0, r, true)
		if tq0 >= q0 {
			tq0 += (len(r))
		}
		if tq1 >= q0 {
			tq1 += (len(r))
		}
		t.SetSelect(tq0, tq1)
		if shouldscroll(t, q0, qid) {
			t.Show(q0+(len(r)), q0+(len(r)), false)
		}
		t.ScrDraw()
		w.SetTag()
		w.addr.q0 += len(r)
		w.addr.q1 = w.addr.q0
		fc.Count = x.fcall.Count
		respond(x, &fc, nil)

	case QWevent:
		xfideventwrite(x, w)

	case QWtag:
		t = &w.tag
		BodyTag()

	default:
		respond(x, &fc, fmt.Errorf("unknown qid %d in write", qid))
	}
	if w != nil {
		w.Unlock()
	}
}

func xfidctlwrite(x *Xfid, w *Window) {
var (
	fc plan9.Fcall
	err error
	scrdraw, settag bool
	t *Text
	n int
)
	err = nil;
	scrdraw = false;
	settag = false;
	
	w.tag.Commit(true);
	lines := strings.Split(string(x.fcall.Data), "\n")
	var lidx int
	var line string
forloop:
	for lidx, line = range lines {
		words := strings.Split(line, " ")
		switch words[0] {
		case "": // empty line.
		case "lock" : 	// make window exclusive use
			w.ctrllock.Lock();
			w.ctlfid = x.f.fid;
		case  "unlock":// release exclusive use
			w.ctlfid = math.MaxUint32
			w.ctrllock.Unlock();
		case "clean":	// mark window 'clean', seq=0
			t = &w.body;
			t.eq0 = ^0;
			t.file.Reset();
			t.file.mod = false;
			w.dirty = false;
			settag = true;
		case "dirty":	// mark window 'dirty'
			t = &w.body;
			// doesn't change sequence number, so "Put" won't appear.  it shouldn't.
			t.file.mod = true;
			w.dirty = true;
			settag = true;
		case "show":	// show dot
			t = &w.body;
			t.Show(t.q0, t.q1, true);
		case "name":	// set file name
			r := []rune(words[1])
			for _, rr := range r {
				if rr <= ' ' {
					err = fmt.Errorf("bad character in file name");
					break
				}
			}
			seq++;
			w.body.file.Mark();
			w.SetName(string(r));
		case "dump":	// set dump string
			r := []rune(words[1])
			for _, rr := range r {
				if rr <= ' ' {
					err = fmt.Errorf("bad character in file name");
					break
				}
			}
			w.dumpstr = string(r)
		case "dumpdir": 	// set dump directory
			r := []rune(words[1])
			for _, rr := range r {
				if rr <= ' ' {
					err = fmt.Errorf("bad character in file name");
					break
				}
			}
			w.dumpdir = string(r)
		case "delete":	// delete for sure
			w.col.Close(w, true);
		case "del":	// delete, but check dirty
			if w.Clean(true) {
				err = fmt.Errorf("file dirty");
				break;
			}
			w.col.Close(w, true);
		case "get":	// get file
			get(&w.body, nil, nil, false, XXX, nil, 0);
		case "put":	// put file
			put(&w.body, nil, nil, XXX, XXX, nil, 0);
		case "dot=addr":	// set dot
			w.body.Commit(true);
			clampaddr(w);
			w.body.q0 = w.addr.q0;
			w.body.q1 = w.addr.q1;
			w.body.SetSelect(w.body.q0, w.body.q1);
			settag = true;
		case "addr=dot":	// set addr
			w.addr.q0 = w.body.q0;
			w.addr.q1 = w.body.q1;
		case "limit=addr":	// set limit
			w.body.Commit(true);
			clampaddr(w);
			w.limit.q0 = w.addr.q0;
			w.limit.q1 = w.addr.q1;
		case "nomark":	// turn off automatic marking
			w.nomark = true;
		case "mark":	// mark file
			seq++;
			w.body.file.Mark();
			settag = true;
		case "nomenu":	// turn off automatic menu
			w.filemenu = false;
		case "menu":	// enable automatic menu
			w.filemenu = true;
		case "cleartag":	// wipe tag right of bar
			w.ClearTag();
			settag = true;
	
		default:
			err = Ebadctl;
			break forloop;
		}
	}

	if err != nil {
		n = 0;
	} else {
		// how far through the buffer did we get?
		// count bytes up to line lineidx
		d := x.fcall.Data
		curline := 0
		for n = 0; n < len(d); n++ {
			if curline == lidx {
				break
			}
			if d[n] == '\n' {
				curline++
			}
		}	
	}
	fc.Count = uint32(n);
	respond(x, &fc, err);
	if settag {
		w.SetTag();
	}
	if scrdraw {
		w.body.ScrDraw();
	}
}

func xfideventwrite(x *Xfid, w *Window) {
var (
	fc plan9.Fcall
	m, n int
	err error
	t *Text
	q0, q1 int
	num int64
)
	err = nil;
/*
The mes-
               sages have a fixed format: a character indicating the
               origin or cause of the action, a character indicating
               the type of the action, four free-format blank-
               terminated decimal numbers, optional text, and a new-
               line.  The first and second numbers are the character
               addresses of the action, the third is a flag, and the
               final is a count of the characters in the optional
               text, which may itself contain newlines.
		%c%c%d %d %d %d %s\n
*/
	events := string(x.fcall.Data)
	n = 0
	for events != "" {
		w.owner = int(events[0]); n++
		c := events[1]; n++
		events = events[2:]

		for (events[0] == ' ') { events = events[1:] ; n++}
		e := strings.SplitN(events, " ", 2)
		n += len(e[0]) + 1
		num, err = strconv.ParseInt(e[0], 10, 32)
		if err != nil {
			err = Ebadevent
			break
		}
		q0 = int(num)
		events = e[1]


		for (events[0] == ' ') { events = events[1:] ; n++ }
		e = strings.SplitN(events, " ", 2)
		n += len(e[0]) + 1
		num, err = strconv.ParseInt(e[0], 10, 32)
		if err != nil {
			err = Ebadevent
			break
		}
		q1 = int(num)
		events = e[1]

		for (events[0] == ' ') { events = events[1:] ; n++ }
		n += len(e[0]) + 1
		e = strings.SplitN(events, " ", 2)
		num, err = strconv.ParseInt(e[0], 10, 32)
		if err != nil {
			err = Ebadevent
			break
		}
		//flag := int(num)
		events = e[1]

		for (events[0] == ' ') { events = events[1:] ; n++ }
		n += len(e[0]) + 1
		e = strings.SplitN(events, " ", 2)
		num, err = strconv.ParseInt(e[0], 10, 32)
		if err != nil {
			err = Ebadevent
			break
		}
		m = int(num)
		events = e[1]

		for (events[0] == ' ') { events = events[1:] ; n++ }

		if (m != len(x.fcall.Data) - n) { panic("mis-shaped event") }
		if 'a'<=c && c<='z' {
			t = &w.tag;
		} else {
			if 'A'<=c && c<='Z' {
				t = &w.body;
			} else {
				err = Ebadevent
				break
			}
		}
		if q0>t.file.b.nc() || q1>t.file.b.nc() || q0>q1 {
			err = Ebadevent
			break
		}

		row.lk.Lock();	// just like mousethread
		switch(c){
		case 'x':
		case 'X':
			execute(t, q0, q1, true, nil);
			break;
		case 'l':
		case 'L':
			look3(t, q0, q1, true);
			break;
		default:
			err = Ebadevent
			break
		}
		row.lk.Unlock();
	}

	if err != nil {
		n = 0;
	}
	fc.Count = uint32(n);
	respond(x, &fc, err);
	return;
}

func xfidutfread(x *Xfid, t *Text, q1 int, qid int) {
var (
	fc plan9.Fcall;
	w *Window
//	r []rune
	b1 []rune
	q int
	off, boff  uint64
	m, n, nr, nb int
)
	w = t.w;
	w.Commit(t);
	off = x.fcall.Offset;
	n = 0;
	//r = make([]rune, BUFSIZE/utf8.UTFMax)
	//b = make([]rune, BUFSIZE/utf8.UTFMax)
	b1 = make([]rune, BUFSIZE/utf8.UTFMax)
	if qid==w.utflastqid && off>=w.utflastboff && w.utflastq<=q1 {
		boff = w.utflastboff;
		q = w.utflastq;
	}else{
		// BUG: stupid code: scan from beginning
		boff = 0;
		q = 0;
	}
	w.utflastqid = qid;
	for(q<q1 && n<int(x.fcall.Count)){
		// * Updating here avoids partial rune problem: we're always on a
		// * char boundary. The cost is we will usually do one more read
		// * than we really need, but that's better than being n^2.
		w.utflastboff = boff;
		w.utflastq = q;
		nr = q1-q;
		if nr > BUFSIZE/utf8.UTFMax {
			nr = BUFSIZE/utf8.UTFMax;
		}
		r := t.file.b.Read(q, nr);
		b := r //nb = snprint(b, BUFSIZE+1, "%.*S", nr, r);
		if boff >= off {
			m = len(b);
			if boff+uint64(m) > off+uint64(x.fcall.Count) {
				m = int(off+uint64(x.fcall.Count) - boff);
			}
			copy(b1[n:], b[:m]);
			n += m;
		}else {
			if boff+uint64(nb) > off {
				if n != 0 {
					acmeerror("bad count in utfrune", nil);
				}
				m = nb - int(off-boff);
				if m > int(x.fcall.Count) {
					m = int(x.fcall.Count)
				}
				copy(b1, b[off-boff: int(off-boff)+m]);
				n += m;
			}
		}
		boff += uint64(nb);
		q += len(r);
	}
	fc.Count = uint32(n);
	fc.Data = []byte(string(b1));
	respond(x, &fc, nil);
}

func xfidruneread(x *Xfid, t *Text, q0 int, q1 int) int {
	Unimpl()
	return 0
} /*
	Fcall fc;
	Window *w;
	Rune *r, junk;
	char *b, *b1;
	uint q, boff;
	int i, rw, m, n, nr, nb;

	w = t.w;
	w, t.Commit();
	r = fbufalloc();
	b = fbufalloc();
	b1 = fbufalloc();
	n = 0;
	q = q0;
	boff = 0;
	while(q<q1 && n<x.fcall.count){
		nr = q1-q;
		if nr > BUFSIZE/UTFmax
			nr = BUFSIZE/UTFmax;
		bufread(&t.file.b, q, r, nr);
		nb = snprint(b, BUFSIZE+1, "%.*S", nr, r);
		m = nb;
		if boff+m > x.fcall.count {
			i = x.fcall.count - boff;
			// copy whole runes only
			m = 0;
			nr = 0;
			while(m < i){
				rw = chartorune(&junk, b+m);
				if m+rw > i
					break;
				m += rw;
				nr++;
			}
			if m == 0
				break;
		}
		memmove(b1+n, b, m);
		n += m;
		boff += nb;
		q += nr;
	}
	fbuffree(r);
	fbuffree(b);
	fc.count = n;
	fc.data = b1;
	respond(x, &fc, nil);
	fbuffree(b1);
	return q-q0;
}
*/
func xfideventread(x *Xfid, w *Window) {
	var fc plan9.Fcall

	i := 0
	x.flushed = false
	for len(w.events) == 0 { // TODO(flux): Yes, that seems to be the case.  I suspect the response message makes an event?
		if i != 0 {
			if !x.flushed {
				respond(x, &fc, fmt.Errorf("window shut down"))
			}
			return
		}
		w.eventx = x
		w.Unlock()
		<-x.c
		w.Lock('F')
		i++
	}

	n := uint32(len(w.events))
	if n > x.fcall.Count {
		n = x.fcall.Count
	}
	fc.Count = n
	fc.Data = w.events // TODO(flux) the original doesn't make  copy, so I'm guessing respond consumes ahead of the copy below.
	respond(x, &fc, nil)
	copy(w.events[0:], w.events[n:])
}

func xfidindexread(x *Xfid) {
	Unimpl()
}

/*
	Fcall fc;
	int i, j, m, n, nmax, isbuf, cnt, off;
	Window *w;
	char *b;
	Rune *r;
	Column *c;

	row.lk.Lock();
	nmax = 0;
	for j=0; j<row.ncol; j++ {
		c = row.col[j];
		for i=0; i<c.nw; i++ {
			w = c.w[i];
			nmax += Ctlsize + w.tag.file.b.nc*UTFmax + 1;
		}
	}
	nmax++;
	isbuf = (nmax<=RBUFSIZE);
	if isbuf
		b = (char*)x.buf;
	else
		b = emalloc(nmax);
	r = fbufalloc();
	n = 0;
	for j=0; j<row.ncol; j++ {
		c = row.col[j];
		for i=0; i<c.nw; i++ {
			w = c.w[i];
			// only show the currently active window of a set
			if w.body.file.curtext != &w.body
				continue;
			winctlprint(w, b+n, 0);
			n += Ctlsize;
			m = min(RBUFSIZE, w.tag.file.b.nc);
			bufread(&w.tag.file.b, 0, r, m);
			m = n + snprint(b+n, nmax-n-1, "%.*S", m, r);
			while(n<m && b[n]!='\n')
				n++;
			b[n++] = '\n';
		}
	}
	row.lk.Unlock();
	off = x.fcall.offset;
	cnt = x.fcall.count;
	if off > n
		off = n;
	if off+cnt > n
		cnt = n-off;
	fc.count = cnt;
	memmove(r, b+off, cnt);
	fc.data = (char*)r;
	if !isbuf
		free(b);
	respond(x, &fc, nil);
	fbuffree(r);
}
*/
