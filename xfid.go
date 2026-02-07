package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"9fans.net/go/plan9"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/ninep"
	"github.com/rjkroege/edwood/runes"
	"github.com/rjkroege/edwood/util"
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
		global.cxfidfree <- x
	}
}

func xfidflush(x *Xfid) {
	// log.Println("xfidflush", x)
	// defer log.Println("done xfidflush")

	xfidlogflush(x)

	// search windows for matching tag
	global.row.lk.Lock()
	defer global.row.lk.Unlock()
	for _, c := range global.row.col {
		for _, w := range c.w {
			w.Lock('E')
			wx := w.eventx
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
	x.respond(&plan9.Fcall{}, nil)
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
			tmp, err := os.CreateTemp("", "acme")
			if err != nil || testTempFileFail {
				w.Unlock()
				x.respond(&fc, fmt.Errorf("can't create temp file"))
				return
			}
			os.Remove(tmp.Name()) // tempfile ORCLOSE
			w.nopen[q]++

			_, err = io.Copy(tmp, t.file.Reader(t.q0, t.q1))
			if err != nil || testIOCopyFail {
				// TODO(fhs): Do we want to send an error response to the client?
				warning(nil, fmt.Sprintf("can't write temp file for pipe command %v\n", err))
			}
			w.rdselfd = tmp
		case QWwrsel:
			w.nopen[q]++
			global.seq++
			t.file.Mark(global.seq)
			cut(t, t, nil, false, true, "")
			w.wrselrange = Range{t.q1, t.q1}
			w.nomark = true
		case QWeditout:
			if global.editing == Inactive {
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
			case global.editoutlk <- true:
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
		global.row.lk.Lock()
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
			t.Show(util.Min(w.wrselrange.q0, t.Nc()), util.Min(w.wrselrange.q1, t.Nc()), true)
			t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
		case QWeditout:
			<-w.editoutlk
		}
		w.Close()
		w.Unlock()
		global.row.lk.Unlock()
	} else {
		switch q {
		case Qeditout:
			<-global.editoutlk
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
			x.respond(&fc, fmt.Errorf("unknown qid %d in read", q))
			return
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

// fullrunewrite decodes runes from x.fcall.Data and returns the decoded
// runes. Bytes at the end of x.fcall.Data that can't be fully decoded
// into a rune (partial runes) are saved for next call to this function.
func fullrunewrite(x *Xfid) []rune {
	// extend with previous partial rune at the end.
	cnt := int(x.fcall.Count)
	if x.f.nrpart > 0 {
		x.fcall.Data = append(x.f.rpart[0:x.f.nrpart], x.fcall.Data...)
		cnt += x.f.nrpart
		x.f.nrpart = 0
	}
	r, nb, _ := util.Cvttorunes(x.fcall.Data, cnt-utf8.UTFMax)
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
	var fc plan9.Fcall

	qid := FILE(x.f.qid)
	w := x.f.w
	if w != nil {
		c := 'F'
		if qid == QWtag || qid == QWbody {
			c = 'E'
		}
		w.Lock(int(c))
		if w.col == nil {
			w.Unlock()
			x.respond(&fc, ErrDeletedWin)
			return
		}
	}
	x.fcall.Count = uint32(len(x.fcall.Data))

	// updateText writes x.fcall.Data to text buffer t and sends the 9P response.
	updateText := func(t *Text) {
		// log.Printf("updateText global.seq %d, seq state %s", global.seq, t.file.DebugSeqState())
		r := fullrunewrite(x)
		if len(r) != 0 {
			w.Commit(t)
			var q0 int
			if qid == QWwrsel {
				q0 = w.wrselrange.q1
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
					global.seq++
					t.file.Mark(global.seq)
				}
				// To align with how Acme works, the file on disk has not been changed
				// but Edwood's in-memory store of the file would now be different from
				// the backing file and also not undoable back to the backign state as
				// has been programmatically modified via the filesystem API.
				if t.file.Seq() == 0 && w.nomark {
					t.file.Modded()
				}
				q, nr := t.BsInsert(q0, r, true) // TODO(flux): BsInsert returns nr?
				q0 = q
				t.SetSelect(t.q0, t.q1) // insert could leave it somewhere else
				if qid != QWwrsel && shouldscroll(t, q0, qid) {
					t.Show(q0+(nr), q0+(nr), true)
				}
				t.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
			}
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
		// errorwin handles row locking internally
		w = errorwin(x.f.mntdir, 'X', nil)
		updateText(&w.body)

	case Qlabel:
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)

	case QWaddr:
		r := []rune(string(x.fcall.Data))
		t := &w.body
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

	case Qeditout, QWeditout:
		r := fullrunewrite(x)
		var err error
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
		updateText(&w.body)

	case QWbody, QWwrsel:
		updateText(&w.body)

	case QWctl:
		xfidctlwrite(x, w)

	case QWdata:
		a := w.addr
		t := &w.body
		w.Commit(t)
		if a.q0 > t.Nc() || a.q1 > t.Nc() {
			x.respond(&fc, ErrAddrRange)
			break
		}
		r, _, _ := util.Cvttorunes(x.fcall.Data, int(x.fcall.Count))
		if !w.nomark {
			global.seq++
			t.file.Mark(global.seq)
		}
		q0 := a.q0
		if a.q1 > q0 {
			t.Delete(q0, a.q1, true)
			w.addr.q1 = q0
		}
		tq0 := t.q0
		tq1 := t.q1
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
		w.addr.q0 += len(r)
		w.addr.q1 = w.addr.q0
		fc.Count = x.fcall.Count
		x.respond(&fc, nil)

	case QWevent:
		xfideventwrite(x, w)

	case QWtag:
		updateText(&w.tag)

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
	var err error
	const scrdraw = false

	w.tag.Commit()
	lines := strings.Split(string(x.fcall.Data), "\n")
	n := 0
forloop:
	for lidx := 0; lidx < len(lines); lidx++ {
		line := lines[lidx]
		words := strings.SplitN(line, " ", 2)

		if words[0] != "" && w == nil { // window was deleted in a previous line
			err = ErrDeletedWin
			break
		}

		switch words[0] {
		case "": // empty line.

		// Lock/unlock can hang or crash Edwood.
		// They don't appear to be used for anything useful, so disable for now.
		//
		case "lock": // make window exclusive use
			//w.ctrllock.Lock() // This will hang Edwood if the lock is already locked.
			//w.ctlfid = x.f.fid
			fallthrough
		case "unlock": // release exclusive use
			//w.ctlfid = math.MaxUint32
			//w.ctrllock.Unlock() // This will crash if the lock isn't already locked.
			log.Printf("%v ctl message received for window %v (%v)\n", words[0], w.id, w.body.file.Name())
			err = ErrBadCtl
			break forloop

		case "clean": // mark window 'clean', seq=0
			t := &w.body
			t.eq0 = ^0
			t.file.Clean()
		case "dirty": // mark window 'dirty'
			t := &w.body
			// doesn't change sequence number, so "Put" won't appear.  it shouldn't.
			t.file.Modded()
		case "show": // show dot
			t := &w.body
			t.Show(t.q0, t.q1, true)
		case "name": // set file name
			if len(words) < 2 {
				err = ErrBadCtl
				break forloop
			}

			fn := words[1]
			for _, c := range fn {
				if c == '\000' {
					err = fmt.Errorf("nulls in file name")
					break forloop
				}
				if c < ' ' {
					err = fmt.Errorf("bad character in file name")
					break forloop
				}
			}

			// TODO(rjk): There should be some nicer way to do this.
			if !w.nomark {
				global.seq++
				w.body.file.Mark(global.seq)
			}
			w.SetName(fn)
		case "dump": // set dump string
			if len(words) < 2 {
				err = ErrBadCtl
				break forloop
			}
			r, _, nulls := util.Cvttorunes([]byte(words[1]), len(words[1]))
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
			r, _, nulls := util.Cvttorunes([]byte(words[1]), len(words[1]))
			if nulls {
				err = fmt.Errorf("nulls in dump directory string")
				break forloop
			}
			w.dumpdir = string(r)
		case "delete": // delete for sure
			w.col.Close(w, true)
			w = nil
		case "del": // delete, but check dirty
			if !w.Clean(true) {
				err = fmt.Errorf("file dirty")
				break forloop
			}
			w.col.Close(w, true)
			w = nil
		case "get": // get file
			get(&w.body, nil, nil, false, false, "")
		case "put": // put file
			put(&w.body, nil, nil, false, false, "")
		case "dot=addr": // set dot
			w.body.Commit()
			w.ClampAddr()
			w.body.q0 = w.addr.q0
			w.body.q1 = w.addr.q1
			w.body.SetSelect(w.body.q0, w.body.q1)
		case "addr=dot": // set addr
			w.addr.q0 = w.body.q0
			w.addr.q1 = w.body.q1
		case "limit=addr": // set limit
			w.body.Commit()
			w.ClampAddr()
			w.limit.q0 = w.addr.q0
			w.limit.q1 = w.addr.q1
		case "nomark": // turn off automatic marking
			// Snapshot the file state first to make sure that we do the right thing.
			// But perhaps we are setting up the buffer. So if the seq is not 0, skip
			// this. Are multiple undo snapshots harmful? (Perhaps this causes bugs
			// with undo from the command language?)
			if w.body.file.Seq() > 0 {
				global.seq++
				w.body.file.Mark(global.seq)
			}
			w.nomark = true
			// Once we've nomark'ed, if seq == 0, mutations will be undoable.
			// but the file will be different than disk. So mark it dirty in update.
		case "mark": // mark file
			w.nomark = false
			// Premise is that the next undoable mutation will set an undo point.
			// TODO:(rjk): Maintaining this invariant is tricky. It should be tested
			// and the code in text.go should be appropriately structured to make it
			// easy to reason about and to test.
			// TODO(rjk): The premise is wrong. The first edit does not.
		case "nomenu": // turn off automatic menu
			w.filemenu = false
		case "menu": // enable automatic menu
			w.filemenu = true
		case "cleartag": // wipe tag right of bar
			w.ClearTag()
		case "font":
			if len(words) < 2 {
				err = ErrBadCtl
				break forloop
			}
			r, _, nulls := util.Cvttorunes([]byte(words[1]), len(words[1]))
			if nulls {
				err = fmt.Errorf("nulls in font name")
				break forloop
			}
			fontx(&w.body, nil, nil, false, false, string(r))
		default:
			err = ErrBadCtl
			break forloop
		}
		n += len(line)
		if d := x.fcall.Data; n < len(d) && d[n] == '\n' {
			n++
		}
	}

	if err != nil {
		n = 0
	}
	fc := plan9.Fcall{
		Count: uint32(n),
	}
	x.respond(&fc, err)

	if scrdraw && w != nil {
		t := &w.body
		w.body.ScrDraw(t.fr.GetFrameFillStatus().Nchars)
	}
}

func xfideventwrite(x *Xfid, w *Window) {
	var err error

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

		// Do not acquire row.lk here. The old rowLock helper acquired
		// row.lk while also holding w.Lock, causing self-deadlock when
		// look3/execute internally acquire row.lk via makenewwindow.
		// Mousethread calls look3/execute with only w.Lock held (it
		// releases row.lk at acme.go:431 before the click handlers).
		// Match that: just dispatch with w.Lock held, no row.lk.
		switch c {
		case 'x', 'X':
			execute(t, q0, q1, true, nil)
		case 'l', 'L':
			look3(t, q0, q1, true)
		default:
			err = ErrBadEvent
			break forloop
		}
	}

	var fc plan9.Fcall
	if err != nil {
		fc.Count = 0
	} else {
		fc.Count = uint32(len(x.fcall.Data))
	}
	x.respond(&fc, err)
}

// xfidutfread reads x.fcall.Count bytes from offset x.fcall.Offset in
// text t and sends the data to the client. It only sends full runes,
// and optimizes for sequential reads by keeping track of (byte offset,
// rune offset) pair of the last read from buffer for a matching qid
// (QWbody or QWtag). No data past rune offset q1 is sent to client.
//
// TODO(fhs): Remove this function and use RuneArray.ReadAt once RuneArray
// implements io.ReaderAt interface. RuneArray.ReadAt will need to be careful
// to send full runes only, if we want to keep the current behavior.
func xfidutfread(x *Xfid, t *Text, q1 int, qid int) {
	// log.Println("xfidutfread", x)
	// defer log.Println("done xfidutfread")
	w := t.w
	w.Commit(t)
	off := x.fcall.Offset
	n := 0
	b1 := make([]byte, BUFSIZE)
	var (
		q    int
		boff uint64
	)
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
		nr := q1 - q
		if nr > BUFSIZE/utf8.UTFMax {
			nr = BUFSIZE / utf8.UTFMax
		}
		t.file.Read(q, r[:nr])
		b := string(r[:nr])
		nb := len(b)
		if boff >= off {
			m := len(b)
			if boff+uint64(m) > off+uint64(x.fcall.Count) {
				// Compute the number of bytes to copy. The difference
				// off + count - boff is guaranteed to be small here since
				// boff >= off and we're limiting to x.fcall.Count bytes.
				diff := off + uint64(x.fcall.Count) - boff
				m = int(diff)
			}
			copy(b1[n:], []byte(b[:m]))
			n += m
		} else {
			if boff+uint64(nb) > off {
				if n != 0 {
					util.AcmeError("bad count in utfrune", nil)
				}
				// off - boff is the byte offset into the current buffer b.
				// Since boff + nb > off (we're in this branch), and nb is at most
				// BUFSIZE, the difference off - boff must be less than BUFSIZE,
				// which fits safely in an int.
				delta := off - boff
				m := nb - int(delta)
				if m > int(x.fcall.Count) {
					m = int(x.fcall.Count)
				}
				copy(b1, b[delta:delta+uint64(m)])
				n += m
			}
		}
		boff += uint64(nb)
		q += len(r)
	}
	var fc plan9.Fcall
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
	nr := util.Min(q1-q0, int(x.fcall.Count))
	tmp := make([]rune, nr)
	t.file.Read(q0, tmp)
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

	w.events = w.events[n:]
}

func xfidindexread(x *Xfid) {
	// log.Println("xfidindexread", x)
	// defer log.Println("done xfidindexread")

	// BUG(fhs): This is broken when the client is doing a sequential
	// read using a very small buffer and we create/delete windows
	// in-between the requests.

	global.row.lk.Lock()
	nmax := 0
	for _, c := range global.row.col {
		for _, w := range c.w {
			nmax += Ctlsize + w.tag.Nc()*utf8.UTFMax + 1
		}
	}

	nmax++
	var sb strings.Builder
	for _, c := range global.row.col {
		for _, w := range c.w {
			// only show the currently active window of a set
			if w.body.file.GetCurObserver().(*Text) != &w.body {
				continue
			}
			sb.WriteString(w.CtlPrint(false))
			m := util.Min(BUFSIZE/utf8.UTFMax, w.tag.Nc())
			tag := make([]rune, m)
			w.tag.file.Read(0, tag)

			// We only include first line of a multi-line tag
			if i := runes.IndexRune(tag, '\n'); i >= 0 {
				tag = tag[:i]
			}
			sb.WriteString(string(tag))
			sb.WriteString("\n")
		}
	}
	global.row.lk.Unlock()

	var fc plan9.Fcall
	ninep.ReadString(&fc, &x.fcall, sb.String())
	x.respond(&fc, nil)
}
