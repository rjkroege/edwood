package main

import (
	"fmt"
	"io/ioutil"
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
		q0, q1 uint
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
				r := t.file.b.Read(q0, uint(n))
				s := string(r)
				n, err = w.rdselfd.Write([]byte(s))
				if err != nil || n != len(s) {
					warning(nil, fmt.Sprintf("can't write temp file for pipe command %v\n", err))
					break
				}
				q0 += uint(n)
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
			t.Show(minu(uint(w.wrselrange.q0), t.file.b.nc()), minu(uint(w.wrselrange.q1), t.file.b.nc()), true)
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
		w.addr.q0 += xfidruneread(x, &w.body, uint(w.addr.q0), w.body.file.b.nc())
		w.addr.q1 = w.addr.q0

	case QWxdata:
		// BUG: what should happen if q1 > q0?
		if w.addr.q0 > int(w.body.file.b.nc()) {
			respond(x, &fc, Eaddr)
			break
		}
		w.addr.q0 += xfidruneread(x, &w.body, uint(w.addr.q0), uint(w.addr.q1))

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

func shouldscroll(t *Text, q0 uint, qid int) bool {
	if qid == int(Qcons) {
		return true
	}
	return t.org <= q0 && q0 <= t.org+uint(t.fr.NChars)
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
	Unimpl()
}

/*
	Fcall fc;
	int c, qid, nb, nr, eval;
	char buf[64], *err;
	Window *w;
	Rune *r;
	Range a;
	Text *t;
	uint q0, tq0, tq1;

	qid = FILE(x.f.qid);
	w = x.f.w;
	if w {
		c = 'F';
		if qid==QWtag || qid==QWbody
			c = 'E';
		w.Lock(c);
		if w.col == nil {
			w.Unlock();
			respond(x, &fc, Edel);
			return;
		}
	}
	x.fcall.data[x.fcall.count] = 0;
	switch(qid){
	case Qcons:
		w = errorwin(x.f.mntdir, 'X');
		t=&w.body;
		goto BodyTag;

	case Qlabel:
		fc.count = x.fcall.count;
		respond(x, &fc, nil);
		break;

	case QWaddr:
		x.fcall.data[x.fcall.count] = 0;
		r = bytetorune(x.fcall.data, &nr);
		t = &w.body;
		wincommit(w, t);
		eval = true;
		a = address(false, t, w.limit, w.addr, r, 0, nr, rgetc, &eval, (uint*)&nb);
		free(r);
		if nb < nr {
			respond(x, &fc, Ebadaddr);
			break;
		}
		if !eval {
			respond(x, &fc, Eaddr);
			break;
		}
		w.addr = a;
		fc.count = x.fcall.count;
		respond(x, &fc, nil);
		break;

	case Qeditout:
	case QWeditout:
		r = fullrunewrite(x);
		nr = len(r)
		if w
			err = edittext(w, w.wrselrange.q1, r, nr);
		else
			err = edittext(nil, 0, r, nr);
		free(r);
		if err != nil {
			respond(x, &fc, err);
			break;
		}
		fc.count = x.fcall.count;
		respond(x, &fc, nil);
		break;

	case QWerrors:
		w = errorwinforwin(w);
		t = &w.body;
		goto BodyTag;

	case QWbody:
	case QWwrsel:
		t = &w.body;
		goto BodyTag;

	case QWctl:
		xfidctlwrite(x, w);
		break;

	case QWdata:
		a = w.addr;
		t = &w.body;
		wincommit(w, t);
		if a.q0>t.file.b.nc || a.q1>t.file.b.nc {
			respond(x, &fc, Eaddr);
			break;
		}
		r = runemalloc(x.fcall.count);
		cvttorunes(x.fcall.data, x.fcall.count, r, &nb, &nr, nil);
		if w.nomark == false {
			seq++;
			filemark(t.file);
		}
		q0 = a.q0;
		if a.q1 > q0 {
			textdelete(t, q0, a.q1, true);
			w.addr.q1 = q0;
		}
		tq0 = t.q0;
		tq1 = t.q1;
		textinsert(t, q0, r, nr, true);
		if tq0 >= q0
			tq0 += nr;
		if tq1 >= q0
			tq1 += nr;
		textsetselect(t, tq0, tq1);
		if shouldscroll(t, q0, qid)
			textshow(t, q0+nr, q0+nr, 0);
		textscrdraw(t);
		winsettag(w);
		free(r);
		w.addr.q0 += nr;
		w.addr.q1 = w.addr.q0;
		fc.count = x.fcall.count;
		respond(x, &fc, nil);
		break;

	case QWevent:
		xfideventwrite(x, w);
		break;

	case QWtag:
		t = &w.tag;
		goto BodyTag;

	BodyTag:
		r = fullrunewrite(x);
		nr = len(r)
		if nr > 0 {
			wincommit(w, t);
			if qid == QWwrsel {
				q0 = w.wrselrange.q1;
				if q0 > t.file.b.nc
					q0 = t.file.b.nc;
			}else
				q0 = t.file.b.nc;
			if qid == QWtag
				textinsert(t, q0, r, nr, true);
			else{
				if w.nomark == false {
					seq++;
					filemark(t.file);
				}
				q0 = textbsinsert(t, q0, r, nr, true, &nr);
				textsetselect(t, t.q0, t.q1);	// insert could leave it somewhere else
				if qid!=QWwrsel && shouldscroll(t, q0, qid)
					textshow(t, q0+nr, q0+nr, 1);
				textscrdraw(t);
			}
			winsettag(w);
			if qid == QWwrsel
				w.wrselrange.q1 += nr;
			free(r);
		}
		fc.count = x.fcall.count;
		respond(x, &fc, nil);
		break;

	default:
		sprint(buf, "unknown qid %d in write", qid);
		respond(x, &fc, buf);
		break;
	}
	if w
		w.Unlock();
}

func xfidctlwrite (x * Xfid, w * Window) () {
	Fcall fc;
	int i, m, n, nb, nr, nulls;
	Rune *r;
	char *err, *p, *pp, *q, *e;
	int isfbuf, scrdraw, settag;
	Text *t;

	err = nil;
	e = x.fcall.data+x.fcall.count;
	scrdraw = false;
	settag = false;
	isfbuf = true;
	if x.fcall.count < RBUFSIZE
		r = fbufalloc();
	else{
		isfbuf = false;
		r = emalloc(x.fcall.count*UTFmax+1);
	}
	x.fcall.data[x.fcall.count] = 0;
	textcommit(&w.tag, true);
	for n=0; n<x.fcall.count; n+=m {
		p = x.fcall.data+n;
		if strncmp(p, "lock", 4) == 0 {	// make window exclusive use
			w.ctllock.Lock();
			w.ctlfid = x.f.fid;
			m = 4;
		}else
		if strncmp(p, "unlock", 6) == 0 {	// release exclusive use
			w.ctlfid = ~0;
			w.ctllock.Unlock();
			m = 6;
		}else
		if strncmp(p, "clean", 5) == 0 {	// mark window 'clean', seq=0
			t = &w.body;
			t.eq0 = ~0;
			filereset(t.file);
			t.file.mod = false;
			w.dirty = false;
			settag = true;
			m = 5;
		}else
		if strncmp(p, "dirty", 5) == 0 {	// mark window 'dirty'
			t = &w.body;
			// doesn't change sequence number, so "Put" won't appear.  it shouldn't.
			t.file.mod = true;
			w.dirty = true;
			settag = true;
			m = 5;
		}else
		if strncmp(p, "show", 4) == 0 {	// show dot
			t = &w.body;
			textshow(t, t.q0, t.q1, 1);
			m = 4;
		}else
		if strncmp(p, "name ", 5) == 0 {	// set file name
			pp = p+5;
			m = 5;
			q = memchr(pp, '\n', e-pp);
			if q==nil || q==pp {
				err = Ebadctl;
				break;
			}
			*q = 0;
			nulls = false;
			cvttorunes(pp, q-pp, r, &nb, &nr, &nulls);
			if nulls {
				err = "nulls in file name";
				break;
			}
			for i=0; i<nr; i++
				if r[i] <= ' ' {
					err = "bad character in file name";
					goto out;
				}
out:
			seq++;
			filemark(w.body.file);
			winsetname(w, r, nr);
			m += (q+1) - pp;
		}else
		if strncmp(p, "dump ", 5) == 0 {	// set dump string
			pp = p+5;
			m = 5;
			q = memchr(pp, '\n', e-pp);
			if q==nil || q==pp {
				err = Ebadctl;
				break;
			}
			*q = 0;
			nulls = false;
			cvttorunes(pp, q-pp, r, &nb, &nr, &nulls);
			if nulls {
				err = "nulls in dump string";
				break;
			}
			w.dumpstr = runetobyte(r, nr);
			m += (q+1) - pp;
		}else
		if strncmp(p, "dumpdir ", 8) == 0 {	// set dump directory
			pp = p+8;
			m = 8;
			q = memchr(pp, '\n', e-pp);
			if q==nil || q==pp {
				err = Ebadctl;
				break;
			}
			*q = 0;
			nulls = false;
			cvttorunes(pp, q-pp, r, &nb, &nr, &nulls);
			if nulls {
				err = "nulls in dump directory string";
				break;
			}
			w.dumpdir = runetobyte(r, nr);
			m += (q+1) - pp;
		}else
		if strncmp(p, "delete", 6) == 0 {	// delete for sure
			colclose(w.col, w, true);
			m = 6;
		}else
		if strncmp(p, "del", 3) == 0 {	// delete, but check dirty
			if !winclean(w, true) {
				err = "file dirty";
				break;
			}
			colclose(w.col, w, true);
			m = 3;
		}else
		if strncmp(p, "get", 3) == 0 {	// get file
			get(&w.body, nil, nil, false, XXX, nil, 0);
			m = 3;
		}else
		if strncmp(p, "put", 3) == 0 {	// put file
			put(&w.body, nil, nil, XXX, XXX, nil, 0);
			m = 3;
		}else
		if strncmp(p, "dot=addr", 8) == 0 {	// set dot
			textcommit(&w.body, true);
			clampaddr(w);
			w.body.q0 = w.addr.q0;
			w.body.q1 = w.addr.q1;
			textsetselect(&w.body, w.body.q0, w.body.q1);
			settag = true;
			m = 8;
		}else
		if strncmp(p, "addr=dot", 8) == 0 {	// set addr
			w.addr.q0 = w.body.q0;
			w.addr.q1 = w.body.q1;
			m = 8;
		}else
		if strncmp(p, "limit=addr", 10) == 0 {	// set limit
			textcommit(&w.body, true);
			clampaddr(w);
			w.limit.q0 = w.addr.q0;
			w.limit.q1 = w.addr.q1;
			m = 10;
		}else
		if strncmp(p, "nomark", 6) == 0 {	// turn off automatic marking
			w.nomark = true;
			m = 6;
		}else
		if strncmp(p, "mark", 4) == 0 {	// mark file
			seq++;
			filemark(w.body.file);
			settag = true;
			m = 4;
		}else
		if strncmp(p, "nomenu", 6) == 0 {	// turn off automatic menu
			w.filemenu = false;
			m = 6;
		}else
		if strncmp(p, "menu", 4) == 0 {	// enable automatic menu
			w.filemenu = true;
			m = 4;
		}else
		if strncmp(p, "cleartag", 8) == 0 {	// wipe tag right of bar
			wincleartag(w);
			settag = true;
			m = 8;
		}else{
			err = Ebadctl;
			break;
		}
		while(p[m] == '\n')
			m++;
	}

	if isfbuf
		fbuffree(r);
	else
		free(r);
	if err
		n = 0;
	fc.count = n;
	respond(x, &fc, err);
	if settag
		winsettag(w);
	if scrdraw
		textscrdraw(&w.body);
}

func xfideventwrite (x * Xfid, w * Window) () {
	Fcall fc;
	int m, n;
	Rune *r;
	char *err, *p, *q;
	int isfbuf;
	Text *t;
	int c;
	uint q0, q1;

	err = nil;
	isfbuf = true;
	if x.fcall.count < RBUFSIZE
		r = fbufalloc();
	else{
		isfbuf = false;
		r = emalloc(x.fcall.count*UTFmax+1);
	}
	for n=0; n<x.fcall.count; n+=m {
		p = x.fcall.data+n;
		w.owner = *p++;	// disgusting
		c = *p++;
		while(*p == ' ')
			p++;
		q0 = strtoul(p, &q, 10);
		if q == p
			goto Rescue;
		p = q;
		while(*p == ' ')
			p++;
		q1 = strtoul(p, &q, 10);
		if q == p
			goto Rescue;
		p = q;
		while(*p == ' ')
			p++;
		if *p++ != '\n'
			goto Rescue;
		m = p-(x.fcall.data+n);
		if 'a'<=c && c<='z'
			t = &w.tag;
		else if 'A'<=c && c<='Z'
			t = &w.body;
		else
			goto Rescue;
		if q0>t.file.b.nc || q1>t.file.b.nc || q0>q1
			goto Rescue;

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
			row.lk.Unlock();
			goto Rescue;
		}
		row.lk.Unlock();

	}

    Out:
	if isfbuf
		fbuffree(r);
	else
		free(r);
	if err
		n = 0;
	fc.count = n;
	respond(x, &fc, err);
	return;

    Rescue:
	err = Ebadevent;
	goto Out;
}
*/
func xfidutfread(x *Xfid, t *Text, q1 uint, qid int) {
	Unimpl()
}

/*
	Fcall fc;
	Window *w;
	Rune *r;
	char *b, *b1;
	uint q, off, boff;
	int m, n, nr, nb;

	w = t.w;
	wincommit(w, t);
	off = x.fcall.offset;
	r = fbufalloc();
	b = fbufalloc();
	b1 = fbufalloc();
	n = 0;
	if qid==w.utflastqid && off>=w.utflastboff && w.utflastq<=q1 {
		boff = w.utflastboff;
		q = w.utflastq;
	}else{
		// BUG: stupid code: scan from beginning
		boff = 0;
		q = 0;
	}
	w.utflastqid = qid;
	while(q<q1 && n<x.fcall.count){
		// * Updating here avoids partial rune problem: we're always on a
		// * char boundary. The cost is we will usually do one more read
		// * than we really need, but that's better than being n^2.
		w.utflastboff = boff;
		w.utflastq = q;
		nr = q1-q;
		if nr > BUFSIZE/UTFmax
			nr = BUFSIZE/UTFmax;
		bufread(&t.file.b, q, r, nr);
		nb = snprint(b, BUFSIZE+1, "%.*S", nr, r);
		if boff >= off {
			m = nb;
			if boff+m > off+x.fcall.count
				m = off+x.fcall.count - boff;
			memmove(b1+n, b, m);
			n += m;
		}else if boff+nb > off {
			if n != 0
				error("bad count in utfrune");
			m = nb - (off-boff);
			if m > x.fcall.count
				m = x.fcall.count;
			memmove(b1, b+(off-boff), m);
			n += m;
		}
		boff += nb;
		q += nr;
	}
	fbuffree(r);
	fbuffree(b);
	fc.count = n;
	fc.data = b1;
	respond(x, &fc, nil);
	fbuffree(b1);
}
*/
func xfidruneread(x *Xfid, t *Text, q0 uint, q1 uint) int {
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
	wincommit(w, t);
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

	i := 0;
	x.flushed = false;
	for(len(w.events) == 0){ // TODO(flux): Yes, that seems to be the case.  I suspect the response message makes an event?
		if i!=0 {
			if !x.flushed {
				respond(x, &fc, fmt.Errorf("window shut down"));
			}
			return;
		}
		w.eventx = x;
		w.Unlock();
		<-x.c
		w.Lock('F');
		i++;
	}

	n := uint32(len(w.events))
	if n > x.fcall.Count{
		n = x.fcall.Count;
	}
	fc.Count = n;
	fc.Data = w.events; // TODO(flux) the original doesn't make  copy, so I'm guessing respond consumes ahead of the copy below.
	respond(x, &fc, nil);
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
