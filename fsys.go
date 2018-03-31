package main

// TODO(flux): This is a hideous singleton.  Refactor into a type?
import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"9fans.net/go/plan9"
	//"github.com/mortdeus/go9p"
)

// TODO(flux): Wrap fsys into a tidy object.

var (
	sfd *os.File
)

const (
	Nhash = 16
	DEBUG = 0
)

var fids map[uint32]*Fid = make(map[uint32]*Fid)

type fsfunc func(*Xfid, *Fid) *Xfid

var fcall []fsfunc = make([]fsfunc, plan9.Tmax)

func initfcall() {
	fcall[plan9.Tflush] = fsysflush
	fcall[plan9.Tversion] = fsysversion
	fcall[plan9.Tauth] = fsysauth
	fcall[plan9.Tattach] = fsysattach
	fcall[plan9.Twalk] = fsyswalk
	fcall[plan9.Topen] = fsysopen
	fcall[plan9.Tcreate] = fsyscreate
	fcall[plan9.Tread] = fsysread
	fcall[plan9.Twrite] = fsyswrite
	fcall[plan9.Tclunk] = fsysclunk
	fcall[plan9.Tremove] = fsysremove
	fcall[plan9.Tstat] = fsysstat
	fcall[plan9.Twstat] = fsyswstat
}

var (
	Eperm   = errors.New("permission denied")
	Eexist  = errors.New("file does not exist")
	Enotdir = errors.New("not a directory")
)

var dirtab []*DirTab = []*DirTab{
	{".", plan9.QTDIR, Qdir, 0500 | plan9.DMDIR},
	{"acme", plan9.QTDIR, Qacme, 0500 | plan9.DMDIR},
	{"cons", plan9.QTFILE, Qcons, 0600},
	{"consctl", plan9.QTFILE, Qconsctl, 0000},
	{"draw", plan9.QTDIR, Qdraw, 0000 | plan9.DMDIR}, // to suppress graphics progs started in acme
	{"editout", plan9.QTFILE, Qeditout, 0200},
	{"index", plan9.QTFILE, Qindex, 0400},
	{"label", plan9.QTFILE, Qlabel, 0600},
	{"log", plan9.QTFILE, Qlog, 0400},
	{"new", plan9.QTDIR, Qnew, 0500 | plan9.DMDIR},
	//	{ nil, }
}

var dirtabw []*DirTab = []*DirTab{
	{".", plan9.QTDIR, Qdir, 0500 | plan9.DMDIR},
	{"addr", plan9.QTFILE, QWaddr, 0600},
	{"body", plan9.QTAPPEND, QWbody, 0600 | plan9.DMAPPEND},
	{"ctl", plan9.QTFILE, QWctl, 0600},
	{"data", plan9.QTFILE, QWdata, 0600},
	{"editout", plan9.QTFILE, QWeditout, 0200},
	{"errors", plan9.QTFILE, QWerrors, 0200},
	{"event", plan9.QTFILE, QWevent, 0600},
	{"rdsel", plan9.QTFILE, QWrdsel, 0400},
	{"wrsel", plan9.QTFILE, QWwrsel, 0200},
	{"tag", plan9.QTAPPEND, QWtag, 0600 | plan9.DMAPPEND},
	{"xdata", plan9.QTFILE, QWxdata, 0600},
	//	{ nil, }
}

type Mnt struct {
	lk sync.Mutex
	id int
	md *MntDir
}

var mnt Mnt

var (
	username string = "Wile E. Coyote"
	closing  int
)

func fsysinit() {
	initfcall()
	pipe, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		acmeerror("Failed to open pipe", nil)
	}
	reader := os.NewFile(uintptr(pipe[0]), "pipeend0")
	writer := os.NewFile(uintptr(pipe[1]), "pipeend1")
	if post9pservice(reader, "acme", mtpt) < 0 {
		acmeerror("can't post service", nil)
	}
	sfd = writer
	username = getuser()

	go fsysproc()
}

func fsysproc() {
	x := (*Xfid)(nil)
	var f *Fid
	for {
		fc, err := plan9.ReadFcall(sfd)
		if err != nil || fc == nil {
			acmeerror("fsysproc: ", err)
		}
		if x == nil {
			cxfidalloc <- nil
			x = <-cxfidalloc
		}
		x.fcall = *fc
		switch x.fcall.Type {
		case plan9.Tversion:
			fallthrough
		case plan9.Tauth:
			fallthrough
		case plan9.Tflush:
			f = nil
		case plan9.Tattach:
			f = newfid(x.fcall.Fid)
		default:
			f = newfid(x.fcall.Fid)
			if !f.busy {
				x.f = f
				x = respond(x, fc, fmt.Errorf("fid not in use"))
				continue
			}
		}
		x.f = f
		x = fcall[x.fcall.Type](x, f)
	}
}

func fsysaddid(dir string, incl []string) *MntDir {
	var m *MntDir
	var id int

	mnt.lk.Lock()
	mnt.id++
	id = mnt.id
	m = &MntDir{}
	m.id = int64(id)
	m.dir = dir
	m.ref = 1 // one for Command, one will be incremented in attach
	m.next = mnt.md
	m.incl = incl
	mnt.md = m
	mnt.lk.Unlock()
	return m
}

func fsysincid(m *MntDir) {
	mnt.lk.Lock()
	m.ref++
	mnt.lk.Unlock()
}

func fsysdelid(idm *MntDir) {
	var (
		m, prev *MntDir
	)

	if idm == nil {
		return
	}
	mnt.lk.Lock()
	defer mnt.lk.Unlock()
	idm.ref--
	if idm.ref > 0 {
		mnt.lk.Unlock()
		return
	}
	prev = nil
	for m = mnt.md; m != nil; m = m.next {
		if m == idm {
			if prev != nil {
				prev.next = m.next
			} else {
				mnt.md = m.next
			}
			return
		}
		prev = m
	}

	cerr <- fmt.Errorf("fsysdelid: can't find id %d\n", idm.id)
}

// Called only in exec.c:/^run(), from a different FD group
func fsysmount(dir string, incl []string) *MntDir {
	return fsysaddid(dir, incl)
}

func fsysclose() {
	closing = 1
	sfd.Close()
}

func respond(x *Xfid, t *plan9.Fcall, err error) *Xfid {
	if err != nil {
		t.Type = plan9.Rerror
		t.Ename = err.Error()
	} else {
		t.Type = x.fcall.Type + 1
	}
	t.Fid = x.fcall.Fid
	t.Tag = x.fcall.Tag
	if err := plan9.WriteFcall(sfd, t); err != nil {
		acmeerror("write error in respond", err)
	}
	if DEBUG != 0 {
		fmt.Fprintf(os.Stderr, "r: %v\n", t)
	}
	return x
}

func fsysversion(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	messagesize = int(x.fcall.Msize)
	t.Msize = x.fcall.Msize
	if x.fcall.Version != "9P2000" {
		return respond(x, &t, fmt.Errorf("unrecognized 9P version"))
	}
	t.Version = "9P2000"
	return respond(x, &t, nil)
}

func fsysauth(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	return respond(x, &t, fmt.Errorf("acme: authentication not required"))
}

func fsysflush(x *Xfid, f *Fid) *Xfid {
	x.c <- xfidflush
	return nil
}

func fsysattach(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	if x.fcall.Uname != username {
		return respond(x, &t, Eperm)
	}
	f.busy = true
	f.open = false
	f.qid.Path = uint64(Qdir)
	f.qid.Type = plan9.QTDIR
	f.qid.Vers = 0
	f.dir = dirtab[0] // '.'
	f.nrpart = 0
	f.w = nil
	t.Qid = f.qid
	f.mntdir = nil
	var id int64
	var err error
	if x.fcall.Aname != "" {
		id, err = strconv.ParseInt(x.fcall.Aname, 10, 32)
		if err != nil {
			acmeerror(fmt.Sprintf("fsysattach: bad Aname %s", x.fcall.Aname), err)
		}
	}
	mnt.lk.Lock()
	var m *MntDir
	for m = mnt.md; m != nil; m = m.next {
		if m.id == id {
			f.mntdir = m
			m.ref++
			break
		}
	}
	if m == nil && x.fcall.Aname != "" {
		cerr <- fmt.Errorf("unknown id '%s' in attach", x.fcall.Aname)
	}
	mnt.lk.Unlock()
	return respond(x, &t, nil)
}

func fsyswalk(x *Xfid, f *Fid) *Xfid {
	var (
		t    plan9.Fcall
		q    plan9.Qid
		typ  byte
		path uint64
		d    []*DirTab
		dir  *DirTab
		id   int
	)
	nf := (*Fid)(nil)
	w := (*Window)(nil)
	if f.open {
		return respond(x, &t, fmt.Errorf("walk of open file"))
	}
	if x.fcall.Fid != x.fcall.Newfid {
		nf = newfid(x.fcall.Newfid)
		if nf.busy {
			return respond(x, &t, fmt.Errorf("newfid already in use"))
		}
		nf.busy = true
		nf.open = false
		nf.mntdir = f.mntdir
		if f.mntdir != nil {
			f.mntdir.ref++
		}
		nf.dir = f.dir
		nf.qid = f.qid
		nf.w = f.w
		nf.nrpart = 0 // not open, so must be zero
		if nf.w != nil {
			nf.w.ref.Inc()
		}
		f = nf // walk f
	}

	t.Wqid = nil
	var err error
	dir = nil
	id = WIN(f.qid)
	q = f.qid

	var i int
	var wname string
	if len(x.fcall.Wname) > 0 {
	Wnames:
		for i = 0; i < len(x.fcall.Wname); i++ {
			wname = x.fcall.Wname[i]
			if (q.Type & plan9.QTDIR) == 0 {
				err = Enotdir
				break
			}

			if wname == ".." {
				typ = plan9.QTDIR
				path = uint64(Qdir)
				id = 0
				if w != nil {
					w.Close()
					w = nil
				}
				q.Type = typ
				q.Vers = 0
				q.Path = uint64(QID(id, path))
				t.Wqid = append(t.Wqid, q)
				continue
			}
			// is it a numeric name?
			_, err := strconv.ParseInt(wname, 10, 32)
			if err != nil {
				goto Regular
			}
			// yes: it's a directory
			if w != nil { // name has form 27/23; get out before losing w
				break
			}
			{
				id64, _ := strconv.ParseInt(wname, 10, 32)
				id = int(id64)
			}
			row.lk.Lock()
			w = row.LookupWin(id, false)
			if w == nil {
				row.lk.Unlock()
				break
			}
			w.ref.Inc() // we'll drop reference at end if there's an error
			path = uint64(Qdir)
			typ = plan9.QTDIR
			row.lk.Unlock()
			dir = dirtabw[0] // '.'
			if i == plan9.MAXWELEM {
				err = fmt.Errorf("name too long")
				break
			}
			q.Type = typ
			q.Vers = 0
			q.Path = uint64(QID(id, path))
			t.Wqid = append(t.Wqid, q)
			continue

		Regular:
			if wname == "new" {
				if w != nil {
					acmeerror("w set in walk to new", nil)
				}
				cnewwindow <- nil // signal newwindowthread
				w = <-cnewwindow  // receive new window
				w.ref.Inc()
				typ = plan9.QTDIR
				path = uint64(QID(w.id, Qdir))
				id = w.id
				dir = dirtabw[0]
				q.Type = typ
				q.Vers = 0
				q.Path = QID(id, path)
				t.Wqid = append(t.Wqid, q)
				continue Wnames
			}

			if id == 0 {
				d = dirtab
			} else {
				d = dirtabw
			}
			for _, de := range d[1:] {
				if wname == de.name {
					path = de.qid
					typ = de.t
					dir = de
					q.Type = typ
					q.Vers = 0
					q.Path = QID(id, path)
					t.Wqid = append(t.Wqid, q)
					continue Wnames
				}
			}
			break // file not found
		}

		// If we never incremented
		if i == 0 && err == nil {
			err = Eexist
		}
		if i == plan9.MAXWELEM {
			err = fmt.Errorf("name too long")
		}
	}

	if err != nil || len(t.Wqid) < len(x.fcall.Wname) {
		if nf != nil {
			nf.busy = false
			fsysdelid(nf.mntdir)
		}
	} else {
		if len(t.Wqid) == len(x.fcall.Wname) {
			if w != nil {
				f.w = w
				w = nil // don't drop the reference when closing below.
			}
			if dir != nil {
				f.dir = dir
			}
			f.qid = q
		}
	}

	if w != nil {
		w.Close()
	}

	return respond(x, &t, err)
}

func fsysopen(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	var m uint
	// can't truncate anything, so just disregard
	x.fcall.Mode &= ^(uint8(plan9.OTRUNC | plan9.OCEXEC))
	// can't execute or remove anything
	if x.fcall.Mode == plan9.OEXEC || (x.fcall.Mode&plan9.ORCLOSE) != 0 {
		goto Deny
	}
	switch x.fcall.Mode {
	case plan9.OREAD:
		m = 0400
	case plan9.OWRITE:
		m = 0200
	case plan9.ORDWR:
		m = 0600
	default:
		goto Deny
	}
	if ((f.dir.perm &^ (plan9.DMDIR | plan9.DMAPPEND)) & m) != m {
		goto Deny
	}
	x.c <- xfidopen
	return nil

Deny:
	return respond(x, &t, Eperm)
}

func fsyscreate(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	return respond(x, &t, Eperm)
}

//func idcmp (const  void *a, const  void *b) (int) {
//	return *(int*)a - *(int*)b;
//}

// TODO(flux): I'm pretty sure handling of int64 sized files is broken by type casts to int.
func fsysread(x *Xfid, f *Fid) *Xfid {
	var (
		t           plan9.Fcall
		id, n, j, k int
		i, e, o     uint64
		ids         []int
		d           []*DirTab
		dt          DirTab
		clock       int64
		length      int
	)
	if f.qid.Type&plan9.QTDIR != 0 {
		if FILE(f.qid) == Qacme { // empty dir
			t.Data = nil
			t.Count = 0
			respond(x, &t, nil)
			return x
		}
		o = x.fcall.Offset
		e = x.fcall.Offset + uint64(x.fcall.Count)
		clock = getclock()
		b := make([]byte, x.fcall.Count)
		id = WIN(f.qid)
		n = 0
		if id > 0 {
			d = dirtabw
		} else {
			d = dirtab
		}
		d = d[1:] // Skip '.'
		i = uint64(0)
		for _, de := range d {
			if !(i < e) {
				break
			}
			length = dostat(WIN(x.f.qid), de, b[n:], clock)
			if i >= o {
				n += length
			}
			i += uint64(length)
		}
		if id == 0 {
			row.lk.Lock()
			ids = []int{}
			for _, c := range row.col {
				for _, w := range c.w {
					ids = append(ids, w.id)
				}
			}
			row.lk.Unlock()
			sort.Ints(ids)
			j = 0
			length = 0
			for ; j < len(ids) && i < e; i += uint64(length) {
				k = ids[j]
				dt.name = fmt.Sprintf("%d", k)
				dt.qid = QID(k, Qdir)
				dt.t = plan9.QTDIR
				dt.perm = plan9.DMDIR | 0700
				length = dostat(k, &dt, b[n:], clock)
				if length == 0 {
					break
				}
				if i >= o {
					n += length
				}
				j++
			}
		}
		t.Data = b[0:n]
		t.Count = uint32(n)
		respond(x, &t, nil)
		return x
	}
	x.c <- xfidread
	return nil
}

func fsyswrite(x *Xfid, f *Fid) *Xfid {
	x.c <- xfidwrite
	return nil
}

func fsysclunk(x *Xfid, f *Fid) *Xfid {

	fsysdelid(f.mntdir)
	x.c <- xfidclose
	return nil
}

func fsysremove(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	return respond(x, &t, Eperm)
}

func fsysstat(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall

	t.Stat = make([]byte, messagesize-plan9.IOHDRSZ)
	length := dostat(WIN(x.f.qid), f.dir, t.Stat, getclock())
	t.Stat = t.Stat[:length]
	x = respond(x, &t, nil)
	return x
}

func fsyswstat(x *Xfid, f *Fid) *Xfid {

	var t plan9.Fcall

	return respond(x, &t, Eperm)
}

func newfid(fid uint32) *Fid {
	ff, ok := fids[fid]
	if !ok {
		ff = &Fid{}
		ff.fid = fid
		fids[fid] = ff
	}
	return ff
}

func getclock() int64 {

	return time.Now().Unix()
}

// buf must have enough length to fit this stat object.
func dostat(id int, dir *DirTab, buf []byte, clock int64) int {
	var d plan9.Dir

	d.Qid.Path = QID(id, dir.qid)
	d.Qid.Vers = 0
	d.Qid.Type = dir.t
	d.Mode = plan9.Perm(dir.perm)
	d.Length = 0 // would be nice to do better
	d.Name = dir.name
	d.Uid = username
	d.Gid = username
	d.Muid = username
	d.Atime = uint32(clock)
	d.Mtime = uint32(clock)

	b, _ := d.Bytes()
	copy(buf, b)
	return len(b)
}
