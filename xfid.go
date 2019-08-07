package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"9fans.net/go/plan9"
	"github.com/rjkroege/edwood/internal/draw"
	"github.com/rjkroege/edwood/internal/ninep"
	"github.com/rjkroege/edwood/internal/runes"
)

const Ctlsize = 5 * 12

// Errors returned by file server.
var (
	ErrDeletedWin = fmt.Errorf("deleted window")
	ErrBadCtl     = fmt.Errorf("ill-formed control message")
	ErrBadAddr    = fmt.Errorf("bad address syntax")
	ErrAddrRange  = fmt.Errorf("address out of range")
	ErrInUse      = fmt.Errorf("already in use")
	ErrBadEvent   = fmt.Errorf("bad event syntax")
)

func (x *Xfid) respond(t *plan9.Fcall, err error) *Xfid {
	return x.fs.respond(x, t, err)
}

func xfidctl(x *Xfid, d draw.Display) {
	// log.Println("xfidctl", x)
	// defer log.Println("done xfidctl")
	for f := range x.c {
		f(x)
		if d != nil {
			d.Flush()
		} // d here is for testability.
		cxfidfree <- x
	}
}

func xfidflush(x *Xfid) {
	// log.Println("xfidflush", x)
	// defer log.Println("done xfidflush")
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
	x.respond(&fc, nil)
}

// These variables are only used for testing.
var (
	testTempFileFail bool
	testIOCopyFail   bool
)

func xfidopen(x *Xfid) {
	// log.Println("xfidopen", x)
	// defer log.Println("xfidopen done")
	var fc plan9.Fcall

	w := x.f.w
	q := FILE(x.f.qid)
	if w != nil {
		t := &w.body
		w.Lock('E')
		switch q {
		case QWaddr:
			if w.nopen[q] == 0 {
				w.addr = Range{0, 0}
				w.limit = Range{-1, -1}
			}
			w.nopen[q]++
		case QWdata, QWxdata:
			w.nopen[q]++
		case QWevent:
			if w.nopen[q] == 0 {
				if !w.body.file.IsDir() && w.col != nil {
					w.filemenu = false
					w.SetTag()
				}
			}
			w.nopen[q]++
		case QWrdsel:
			// Use a temporary file.
			// A pipe would be the obvious, but we can't afford the
			// broken pipe notification.  Using the code to read QWbody
			// is n², which should probably also be fixed.  Even then,
			// though, we'd need to squirrel away the data in case it's
			// modified during the operation, e.g. by |sort
			if w.rdselfd != nil {
				w.Unlock()
				x.respond(&fc, ErrInUse)
				return
			}
			// TODO(flux): Move the TempFile and Remove
			// into a tempfile() call
			tmp, err := ioutil.TempFile("", "acme")
			if err != nil || testTempFileFail {
				w.Unlock()
				x.respond(&fc, fmt.Errorf("can't create temp file"))
				return
			}
			os.Remove(tmp.Name()) // tempfile ORCLOSE
			w.nopen[q]++

			_, err = io.Copy(tmp, t.file.b.Reader(t.q0, t.q1))
			if err != nil || testIOCopyFail {
				// TODO(fhs): Do we want to send an error response to the client?
				warning(nil, fmt.Sprintf("can't write temp file for pipe command %v\n", err))
			}
			w.rdselfd = tmp
		case QWwrsel:
			w.nopen[q]++
			seq++
			t.file.Mark(seq)
			cut(t, t, nil, false, true, "")
			w.wrselrange = Range{t.q1, t.q1}
			w.nomark = true
		case QWeditout:
			if editing == Inactive {
				w.Unlock()
				x.respond(&fc, ErrPermission)
				return
			}
			select {
			case w.editoutlk <- true:
			default:
				w.Unlock()
				x.respond(&fc, ErrInUse)
				return
			}
			w.wrselrange = Range{t.q1, t.q1}
		}
		w.Unlock()
	} else {
		switch q {
		case Qlog:
			xfidlogopen(x)
		case Qeditout:
			select {
			case editoutlk <- true:
			default:
				x.respond(&fc, ErrInUse)
				return
			}
		}
	}
	fc.Qid = x.f.qid
	fc.Iounit = uint32(x.fs.msize() - plan9.IOHDRSZ)
	x.f.open = true
	x.respond(&fc, nil)
}

func xfidclose(x *Xfid) {
	// log.Println("xfidclose", x)
	// defer log.Println("xfidclose done")
	var fc plan9.Fcall

	w := x.f.w
	x.f.busy = false
	x.f.w = nil
	if !x.f.open {
		if w != nil {
			w.Close()
		}
		x.respond(&fc, nil)
		return
	}

	q := FILE(x.f.qid)
	x.f.open = false
	if w != nil {
		// We need to lock row here before locking window (just like mousethread)
		// in order to synchronize mousetext with mousethread: mousetext is
		// set to nil when the associated window is closed.
		row.lk.Lock()
		w.Lock('E')
		switch q {
		case QWctl:
			if w.ctlfid != MaxFid && w.ctlfid == x.f.fid {
				w.ctlfid = MaxFid
				w.ctrllock.Unlock()
			}
		case QWdata, QWxdata:
			w.nomark = false
			fallthrough
		case QWaddr:
			fallthrough
		case QWevent: // BUG: do we need to shut down Xfid?
			w.nopen[q]--
			if w.nopen[q] == 0 {
				if q == QWdata || q == QWxdata {
					w.nomark = false
				}
				if q == QWevent && !w.body.file.IsDir() && w.col != nil {
					w.filemenu = true
					w.SetTag()
				}
				if q == QWevent {
					w.dumpstr = ""
					w.dumpdir = ""
				}
			}
		case QWrdsel:
			w.rdselfd.Close()
			w.rdselfd = nil
		case QWwrsel:
			w.nomark = false
			t := &w.body
			t.Show(min((w.wrselrange.q0), t.Nc()), min((w.wrselrange.q1), t.Nc()), true)
			t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		case QWeditout:
			<-w.editoutlk
		}
		w.Close()
		w.Unlock()
		row.lk.Unlock()
	} else {
		switch q {
		case Qeditout:
			<-editoutlk
		}
	}
	x.respond(&fc, nil)
}

// xfidread responds to a plan9.Tread request.
func xfidread(x *Xfid) {
	// log.Println("xfidread", x)
	// defer log.Println("done xfidread")
	var fc plan9.Fcall

	q := FILE(x.f.qid)
	w := x.f.w
	if w == nil {
		fc.Count = 0
		switch q {
		case Qcons: // Do nothing.
		case Qlabel: // Do nothing.
		case Qindex:
			xfidindexread(x)
			return
		case Qlog:
			xfidlogread(x)
			return
		default:
			warning(nil, "unknown qid %d\n", q)
		}
		x.respond(&fc, nil)
		return
	}
	w.Lock('F')
	defer w.Unlock()
	if w.col == nil {
		x.respond(&fc, ErrDeletedWin)
		return
	}
	off := x.fcall.Offset
	switch q {
	case QWaddr:
		w.body.Commit()
		w.ClampAddr()
		buf := fmt.Sprintf("%11d %11d ", w.addr.q0, w.addr.q1)
		ninep.ReadString(&fc, &x.fcall, buf)
		x.respond(&fc, nil)

	case QWbody:
		xfidutfread(x, &w.body, w.body.Nc(), int(QWbody))

	case QWctl:
		ninep.ReadString(&fc, &x.fcall, w.CtlPrint(true))
		x.respond(&fc, nil)

	case QWevent:
		xfideventread(x, w)

	case QWdata:
		// BUG: what should happen if q1 > q0?
		if w.addr.q0 > w.body.Nc() {
			x.respond(&fc, ErrAddrRange)
			break
		}
		w.addr.q0 += xfidruneread(x, &w.body, w.addr.q0, w.body.Nc())
		w.addr.q1 = w.addr.q0

	case QWxdata:
		// BUG: what should happen if q1 > q0?
		if w.addr.q0 > w.body.Nc() {
			x.respond(&fc, ErrAddrRange)
			break
		}
		w.addr.q0 += xfidruneread(x, &w.body, w.addr.q0, w.addr.q1)

	case QWtag:
		xfidutfread(x, &w.tag, w.tag.Nc(), int(QWtag))

	case QWrdsel:
		w.rdselfd.Seek(int64(off), 0)
		n := int(x.fcall.Count)
		if n > BUFSIZE {
			n = BUFSIZE
		}
		b := make([]byte, n)
		n, err := w.rdselfd.Read(b[:n])
		if err != nil && err != io.EOF {
			x.respond(&fc, fmt.Errorf("I/O error in temp file: %v", err))
			break
		}
		fc.Count = uint32(n)
		fc.Data = b[:n]
		x.respond(&fc, nil)
	default:
		x.respond(&fc, fmt.Errorf("unknown qid %d in read", q))
	}
}

func shouldscroll(t *Text, q0 int, qid uint64) bool {
	if qid == Qcons {
		return true
	}
	return t.org <= q0 && q0 <= t.org+(t.fr.GetFrameFillStatus().Nchars)
}

// This is fiddly code that handles partial runes at the end of a previous write?
func fullrunewrite(x *Xfid) []rune {
	// extend with previous partial rune at the end.
	cnt := int(x.fcall.Count)
	if x.f.nrpart > 0 {
		x.fcall.Data = append(x.f.rpart[0:x.f.nrpart], x.fcall.Data...)
		cnt += x.f.nrpart
		x.f.nrpart = 0
	}
	r, nb, _ := cvttorunes(x.fcall.Data, cnt-utf8.UTFMax)
	for utf8.FullRune(x.fcall.Data[nb:]) {
		ru, si := utf8.DecodeRune(x.fcall.Data[nb:])
		if ru != 0 {
			r = append(r, ru)
		}
		nb += si
	}
	if nb < cnt {
		copy(x.f.rpart[:], x.fcall.Data[nb:])
		x.f.nrpart = cnt - nb
	}
	return r
}

// xfidwrite responds to a plan9.Twrite request.
func xfidwrite(x *Xfid) {
	// log.Println("xfidwrite", x)
	// defer log.Println("done xfidwrite")
	var (
		fc           plan9.Fcall
		c            int
		t            *Text
		q0, tq0, tq1 int
		err          error
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
			x.respond(&fc, ErrDeletedWin)
			return
		}
	}
	x.fcall.Count = uint32(len(x.fcall.Data))

	BodyTag := func() { // Trimmed from the switch below.
		r := fullrunewrite(x)
		if len(r) != 0 {
			w.Commit(t)
			if qid == QWwrsel {
				q0 = (w.wrselrange.q1)
				if q0 > t.Nc() {
					q0 = t.Nc()
				}
			} else {
				q0 = t.Nc()
			}
			if qid == QWtag {
				t.Insert(q0, r, true)
			} else {
				if !w.nomark {
					seq++
					t.file.Mark(seq)
				}
				q, nr := t.BsInsert(q0, r, true) // TODO(flux): BsInsert returns nr?
				q0 = q
				t.SetSelect(t.q0, t.q1) // insert could leave it somewhere else
				if qid != QWwrsel && shouldscroll(t, q0, qid) {
					t.Show(q0+(nr), q0+(nr), true)
				}
				t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
			}
			w.SetTag()
			if qid == QWwrsel {
				w.wrselrange.q1 += len(r)
			}
		}
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)
	}

	//x.fcall.Data[x.fcall.Count] = 0; // null-terminate. unneeded
	switch qid {
	case Qcons:
		w = errorwin(x.f.mntdir, 'X')
		t = &w.body
		BodyTag()

	case Qlabel:
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)

	case QWaddr:
		r := []rune(string(x.fcall.Data))
		t = &w.body
		w.Commit(t)
		eval := true
		a, eval, nr := address(false, t, w.limit, w.addr, 0, len(r),
			func(q int) rune { return r[q] }, eval)
		if nr < len(r) {
			x.respond(&fc, ErrBadAddr)
			break
		}
		if !eval {
			x.respond(&fc, ErrAddrRange)
			break
		}
		w.addr = a
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)

	case Qeditout:
		fallthrough
	case QWeditout:
		r := fullrunewrite(x)
		if w != nil {
			err = edittext(w, w.wrselrange.q1, r)
		} else {
			err = edittext(nil, 0, r)
		}
		if err != nil {
			x.respond(&fc, err)
			break
		}
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)

	case QWerrors:
		w = errorwinforwin(w)
		t = &w.body
		BodyTag()

	case QWbody:
		fallthrough
	case QWwrsel:
		t = &w.body
		BodyTag()

	case QWctl:
		xfidctlwrite(x, w)

	case QWdata:
		a := w.addr
		t = &w.body
		w.Commit(t)
		if a.q0 > t.Nc() || a.q1 > t.Nc() {
			x.respond(&fc, ErrAddrRange)
			break
		}
		r, _, _ := cvttorunes(x.fcall.Data, int(x.fcall.Count))
		if !w.nomark {
			seq++
			t.file.Mark(seq)
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
			tq0 += len(r)
		}
		if tq1 >= q0 {
			tq1 += len(r)
		}
		t.SetSelect(tq0, tq1)
		if shouldscroll(t, q0, qid) {
			t.Show(q0+len(r), q0+len(r), false)
		}
		t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		w.SetTag()
		w.addr.q0 += len(r)
		w.addr.q1 = w.addr.q0
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)

	case QWevent:
		xfideventwrite(x, w)

	case QWtag:
		t = &w.tag
		BodyTag()

	default:
		x.respond(&fc, fmt.Errorf("unknown qid %d in write", qid))
	}
	if w != nil {
		w.Unlock()
	}
}

func xfidctlwrite(x *Xfid, w *Window) {
	// log.Println("xfidctlwrite", x)
	// defer log.Println("done xfidctlwrite")
	var (
		fc              plan9.Fcall
		err             error
		scrdraw, settag bool
		t               *Text
		n               int
	)
	err = nil
	scrdraw = false
	settag = false

	w.tag.Commit()
	lines := strings.Split(string(x.fcall.Data), "\n")
	var lidx int
	var line string
forloop:
	for lidx = 0; lidx < len(lines); lidx++ {
		line = lines[lidx]
		words := strings.SplitN(line, " ", 2)
		switch words[0] {
		case "": // empty line.
		case "lock": // make window exclusive use
			w.ctrllock.Lock()
			w.ctlfid = x.f.fid
		case "unlock": // release exclusive use
			w.ctlfid = math.MaxUint32
			// BUG(fhs): This will crash if the lock isn't already locked.
			w.ctrllock.Unlock()
		case "clean": // mark window 'clean', seq=0
			t = &w.body
			t.eq0 = ^0
			t.file.Reset()
			t.file.Clean()
			settag = true
		case "dirty": // mark window 'dirty'
			t = &w.body
			// doesn't change sequence number, so "Put" won't appear.  it shouldn't.
			t.file.Modded()
			settag = true
		case "show": // show dot
			t = &w.body
			t.Show(t.q0, t.q1, true)
		case "name": // set file name
			if len(words) < 2 {
				err = ErrBadCtl
				break forloop
			}
			r, _, nulls := cvttorunes([]byte(words[1]), len(words[1]))
			if nulls {
				err = fmt.Errorf("nulls in file name")
				break forloop
			}
			for _, rr := range r {
				if rr <= ' ' {
					err = fmt.Errorf("bad character in file name")
					break
				}
			}
			seq++
			w.body.file.Mark(seq)
			w.SetName(string(r))
		case "dump": // set dump string
			if len(words) < 2 {
				err = ErrBadCtl
				break forloop
			}
			r, _, nulls := cvttorunes([]byte(words[1]), len(words[1]))
			if nulls {
				err = fmt.Errorf("nulls in dump string")
				break forloop
			}
			w.dumpstr = string(r)
		case "dumpdir": // set dump directory
			if len(words) < 2 {
				err = ErrBadCtl
				break forloop
			}
			r, _, nulls := cvttorunes([]byte(words[1]), len(words[1]))
			if nulls {
				err = fmt.Errorf("nulls in dump directory string")
				break forloop
			}
			w.dumpdir = string(r)
		case "delete": // delete for sure
			w.col.Close(w, true)
		case "del": // delete, but check dirty
			if w.Clean(true) {
				err = fmt.Errorf("file dirty")
				break
			}
			w.col.Close(w, true)
		case "get": // get file
			get(&w.body, nil, nil, false, XXX, "")
		case "put": // put file
			put(&w.body, nil, nil, XXX, XXX, "")
		case "dot=addr": // set dot
			w.body.Commit()
			w.ClampAddr()
			w.body.q0 = w.addr.q0
			w.body.q1 = w.addr.q1
			w.body.SetSelect(w.body.q0, w.body.q1)
			settag = true
		case "addr=dot": // set addr
			w.addr.q0 = w.body.q0
			w.addr.q1 = w.body.q1
		case "limit=addr": // set limit
			w.body.Commit()
			w.ClampAddr()
			w.limit.q0 = w.addr.q0
			w.limit.q1 = w.addr.q1
		case "nomark": // turn off automatic marking
			w.nomark = true
		case "mark": // mark file
			seq++
			w.body.file.Mark(seq)
			settag = true
		case "nomenu": // turn off automatic menu
			w.filemenu = false
		case "menu": // enable automatic menu
			w.filemenu = true
		case "cleartag": // wipe tag right of bar
			w.ClearTag()
			settag = true

		default:
			err = ErrBadCtl
			break forloop
		}
	}

	if err != nil {
		n = 0
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
	fc.Count = uint32(n)
	x.respond(&fc, err)
	if settag {
		w.SetTag()
	}
	if scrdraw {
		w.body.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
	}
}

func xfideventwrite(x *Xfid, w *Window) {
	var err error

	// We can't lock row while we have a window locked
	// because that can create deadlock with mousethread.
	rowLock := func() {
		w.Unlock()
		row.lk.Lock()
		w.Lock(w.owner)
	}
	rowUnlock := func() {
		w.Unlock()
		row.lk.Unlock()
		w.Lock(w.owner)
	}

	// The messages have a fixed format: a character indicating the
	// origin or cause of the action, a character indicating
	// the type of the action, four free-format blank-terminated
	// decimal numbers, optional text, and a newline.
	// The first and second numbers are the character
	// addresses of the action, the third is a flag, and the
	// final is a count of the characters in the optional
	// text, which may itself contain newlines.
	// %c%c%d %d %d %d %s\n
	lines := strings.Split(string(x.fcall.Data), "\n")
forloop:
	for _, events := range lines {
		if events == "" {
			continue
		}
		if len(events) < 2 {
			err = ErrBadEvent
			break
		}
		w.owner = int(events[0])
		c := events[1]
		words := strings.Fields(events[2:])
		if len(words) < 2 {
			err = ErrBadEvent
			break
		}
		var num int64
		num, err = strconv.ParseInt(words[0], 10, 32)
		if err != nil {
			err = ErrBadEvent
			break
		}
		q0 := int(num)
		num, err = strconv.ParseInt(words[1], 10, 32)
		if err != nil {
			err = ErrBadEvent
			break
		}
		q1 := int(num)

		var t *Text
		switch {
		case 'a' <= c && c <= 'z':
			t = &w.tag
		case 'A' <= c && c <= 'Z':
			t = &w.body
		default:
			err = ErrBadEvent
			break forloop
		}
		if q0 > t.Nc() || q1 > t.Nc() || q0 > q1 {
			err = ErrBadEvent
			break
		}

		rowLock() // just like mousethread
		switch c {
		case 'x', 'X':
			execute(t, q0, q1, true, nil)
		case 'l', 'L':
			look3(t, q0, q1, true)
		default:
			rowUnlock()
			err = ErrBadEvent
			break forloop
		}
		rowUnlock()
	}

	var fc plan9.Fcall
	if err != nil {
		fc.Count = 0
	} else {
		fc.Count = uint32(len(x.fcall.Data))
	}
	x.respond(&fc, err)
}

func xfidutfread(x *Xfid, t *Text, q1 int, qid int) {
	// log.Println("xfidutfread", x)
	// defer log.Println("done xfidutfread")
	var (
		fc           plan9.Fcall
		w            *Window
		q            int
		off, boff    uint64
		m, n, nr, nb int
	)
	w = t.w
	w.Commit(t)
	off = x.fcall.Offset
	n = 0
	b1 := make([]byte, BUFSIZE)
	if qid == w.utflastqid && off >= w.utflastboff && w.utflastq <= q1 {
		boff = w.utflastboff
		q = w.utflastq
	} else {
		// BUG: stupid code: scan from beginning
		boff = 0
		q = 0
	}
	w.utflastqid = qid
	r := make([]rune, BUFSIZE/utf8.UTFMax)
	for q < q1 && n < int(x.fcall.Count) {
		// Updating here avoids partial rune problem: we're always on a
		// char boundary. The cost is we will usually do one more read
		// than we really need, but that's better than being n^2.
		w.utflastboff = boff
		w.utflastq = q
		nr = q1 - q
		if nr > BUFSIZE/utf8.UTFMax {
			nr = BUFSIZE / utf8.UTFMax
		}
		t.file.b.Read(q, r[:nr])
		b := string(r[:nr])
		nb = len(b)
		if boff >= off {
			m = len(b)
			if boff+uint64(m) > off+uint64(x.fcall.Count) {
				m = int(off + uint64(x.fcall.Count) - boff)
			}
			copy(b1[n:], []byte(b[:m]))
			n += m
		} else {
			if boff+uint64(nb) > off {
				if n != 0 {
					acmeerror("bad count in utfrune", nil)
				}
				m = nb - int(off-boff)
				if m > int(x.fcall.Count) {
					m = int(x.fcall.Count)
				}
				copy(b1, b[off-boff:int(off-boff)+m])
				n += m
			}
		}
		boff += uint64(nb)
		q += len(r)
	}
	fc.Data = b1[:n]
	fc.Count = uint32(len(fc.Data))
	x.respond(&fc, nil)
}

// xfidruneread reads runes from address q0,q1 in t and sends the UTF-8
// encoding of at most q1-q0 runes to the client. Not all the the runes
// may be sent because at most x.fcall.Count bytes of full UTF-8 encoding
// is sent. The number of runes sent is returned.
func xfidruneread(x *Xfid, t *Text, q0 int, q1 int) int {
	// log.Println("xfidruneread", x)
	// defer log.Println("done xfidruneread")

	t.w.Commit(t)

	// Get Count runes, but that might be larger than Count bytes
	nr := min(q1-q0, int(x.fcall.Count))
	tmp := make([]rune, nr)
	t.file.b.Read(q0, tmp)
	buf := []byte(string(tmp))

	m := len(buf)
	if len(buf) > int(x.fcall.Count) {
		// copy whole runes only
		m = 0
		nr = 0
		for m < len(buf) {
			_, size := utf8.DecodeRune(buf[m:])
			if m+size > int(x.fcall.Count) {
				break
			}
			m += size
			nr++
		}
	}
	buf = buf[:m]

	fc := plan9.Fcall{
		Count: uint32(len(buf)),
		Data:  buf,
	}
	x.respond(&fc, nil)
	return nr
}

func xfideventread(x *Xfid, w *Window) {
	// log.Println("xfideventread", x)
	// defer log.Println("done xfideventread")
	var fc plan9.Fcall

	i := 0
	x.flushed = false
	for len(w.events) == 0 {
		if i != 0 {
			if !x.flushed {
				x.respond(&fc, fmt.Errorf("window shut down"))
			}
			return
		}
		w.eventx = x
		w.Unlock()
		<-x.c
		w.Lock('F')
		i++
	}

	n := len(w.events)
	if uint32(n) > x.fcall.Count {
		n = int(x.fcall.Count)
	}
	fc.Count = uint32(n)
	fc.Data = w.events[:n]
	x.respond(&fc, nil)
	nn := len(w.events)
	copy(w.events[0:], w.events[n:])

	w.events = w.events[0 : nn-n]
}

func xfidindexread(x *Xfid) {
	// log.Println("xfidindexread", x)
	// defer log.Println("done xfidindexread")

	row.lk.Lock()
	nmax := 0
	for _, c := range row.col {
		for _, w := range c.w {
			nmax += Ctlsize + w.tag.Nc()*utf8.UTFMax + 1
		}
	}

	nmax++
	var sb strings.Builder
	for _, c := range row.col {
		for _, w := range c.w {
			// only show the currently active window of a set
			if w.body.file.curtext != &w.body {
				continue
			}
			sb.WriteString(w.CtlPrint(false))
			m := min(BUFSIZE/utf8.UTFMax, w.tag.Nc())
			tag := make([]rune, m)
			w.tag.file.b.Read(0, tag)

			// We only include first line of a multi-line tag
			if i := runes.IndexRune(tag, '\n'); i >= 0 {
				tag = tag[:i]
			}
			sb.WriteString(string(tag))
			sb.WriteString("\n")
		}
	}
	row.lk.Unlock()

	var fc plan9.Fcall
	ninep.ReadString(&fc, &x.fcall, sb.String())
	x.respond(&fc, nil)
}
