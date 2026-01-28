package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"sort"
	"strconv"
	"sync"
	"time"

	"9fans.net/go/plan9"
	"github.com/rjkroege/edwood/ninep"
	"github.com/rjkroege/edwood/util"
)

type fileServer struct {
	conn        io.ReadWriteCloser
	fids        map[uint32]*Fid
	fcall       []fsfunc
	closing     bool
	username    string
	messagesize int
}

type fsfunc func(*Xfid, *Fid) *Xfid

func (fs *fileServer) initfcall() {
	fs.fcall = make([]fsfunc, plan9.Tmax)
	fs.fcall[plan9.Tflush] = fs.flush
	fs.fcall[plan9.Tversion] = fs.version
	fs.fcall[plan9.Tauth] = fs.auth
	fs.fcall[plan9.Tattach] = fs.attach
	fs.fcall[plan9.Twalk] = fs.walk
	fs.fcall[plan9.Topen] = fs.open
	fs.fcall[plan9.Tcreate] = fs.create
	fs.fcall[plan9.Tread] = fs.read
	fs.fcall[plan9.Twrite] = fs.write
	fs.fcall[plan9.Tclunk] = fs.clunk
	fs.fcall[plan9.Tremove] = fs.remove
	fs.fcall[plan9.Tstat] = fs.stat
	fs.fcall[plan9.Twstat] = fs.wstat
}

// Errors returned by file server.
var (
	ErrPermission = os.ErrPermission
	ErrNotExist   = os.ErrNotExist
	ErrNotDir     = fmt.Errorf("not a directory")
)

var dirtab = []*DirTab{
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
}

var dirtabw = []*DirTab{
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
}

// windowDirTab returns the DirTab entry for window directory for the window with given id.
func windowDirTab(id int) *DirTab {
	return &DirTab{
		name: fmt.Sprintf("%d", id),
		t:    plan9.QTDIR,
		qid:  QID(id, Qdir),
		perm: plan9.DMDIR | 0700,
	}
}

// Mnt is a collection of reference counted MntDir.
// It is used to pass information from Edwood's 9p client to
// Edwood's 9p file server.
type Mnt struct {
	lk sync.Mutex
	id uint64 // Used to generate MntDir identifier.
	md map[uint64]*MntDir
}

var mnt Mnt

func fsysinit() *fileServer {
	p0, p1, err := newPipe()
	if err != nil {
		util.AcmeError("failed to create pipe", err)
	}
	if err := post9pservice(p0, "acme", *mtpt); err != nil {
		util.AcmeError("can't post service", err)
	}

	fs := &fileServer{
		conn:        p1,
		fids:        make(map[uint32]*Fid),
		fcall:       nil, // initialized by initfcall
		closing:     false,
		username:    getuser(),
		messagesize: 0, // we'll know after Tversion
	}
	fs.initfcall()
	go fs.fsysproc()
	return fs
}

func (fs *fileServer) fsysproc() {
	x := (*Xfid)(nil)
	var f *Fid
	for {
		fc, err := plan9.ReadFcall(fs.conn)
		if err != nil || fc == nil {
			if fs.closing {
				break
			}
			util.AcmeError("fsysproc", err)
		}
		if x == nil {
			global.cxfidalloc <- nil
			x = <-global.cxfidalloc
		}
		x.fcall = *fc
		x.fs = fs
		switch x.fcall.Type {
		case plan9.Tversion:
			fallthrough
		case plan9.Tauth:
			fallthrough
		case plan9.Tflush:
			f = nil
		case plan9.Tattach:
			f = fs.newfid(x.fcall.Fid)
		default:
			f = fs.newfid(x.fcall.Fid)
			if !f.busy {
				x.f = f
				x = fs.respond(x, fc, fmt.Errorf("fid not in use"))
				continue
			}
		}
		x.f = f
		x = fs.fcall[x.fcall.Type](x, f)
	}
}

// Add creates a new MntDir and returns a new reference to it.
func (mnt *Mnt) Add(dir string, incl []string) *MntDir {
	mnt.lk.Lock()
	defer mnt.lk.Unlock()
	mnt.id++
	m := &MntDir{
		id:   mnt.id,
		ref:  1, // One for Command. Incremented in attach, walk, etc.
		dir:  dir,
		incl: incl,
	}
	if mnt.md == nil {
		mnt.md = make(map[uint64]*MntDir)
	}
	mnt.md[m.id] = m
	return m
}

// IncRef increments reference to given MntDir.
func (mnt *Mnt) IncRef(m *MntDir) {
	mnt.lk.Lock()
	defer mnt.lk.Unlock()
	m.ref++
}

// DecRef decrements reference to given MntDir.
func (mnt *Mnt) DecRef(idm *MntDir) {
	if idm == nil {
		return
	}
	mnt.lk.Lock()
	defer mnt.lk.Unlock()
	idm.ref--
	if idm.ref > 0 {
		return
	}
	if _, ok := mnt.md[idm.id]; !ok {
		global.cerr <- fmt.Errorf("Mnt.DecRef: can't find id %d", idm.id)
		return
	}
	delete(mnt.md, idm.id)
}

// GetFromID finds the MntDir with given id and returns a new reference to it.
func (mnt *Mnt) GetFromID(id uint64) *MntDir {
	mnt.lk.Lock()
	defer mnt.lk.Unlock()

	m, ok := mnt.md[id]
	if !ok {
		return nil
	}
	m.ref++
	return m
}

func (fs *fileServer) close() {
	if fs != nil {
		fs.closing = true
		fs.conn.Close()
	}
}

func (fs *fileServer) respond(x *Xfid, t *plan9.Fcall, err error) *Xfid {
	if t == nil {
		t = &plan9.Fcall{}
	}
	if err != nil {
		t.Type = plan9.Rerror
		t.Ename = err.Error()
	} else {
		t.Type = x.fcall.Type + 1
	}
	t.Fid = x.fcall.Fid
	t.Tag = x.fcall.Tag
	if err := plan9.WriteFcall(fs.conn, t); err != nil {
		util.AcmeError("write error in respond", err)
	}
	return x
}

func (fs *fileServer) msize() int {
	return fs.messagesize
}

func (fs *fileServer) version(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	fs.messagesize = int(x.fcall.Msize)
	t.Msize = x.fcall.Msize
	if x.fcall.Version != "9P2000" {
		return fs.respond(x, &t, fmt.Errorf("unrecognized 9P version"))
	}
	t.Version = "9P2000"
	return fs.respond(x, &t, nil)
}

func (fs *fileServer) auth(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	return fs.respond(x, &t, fmt.Errorf("acme: authentication not required"))
}

func (fs *fileServer) flush(x *Xfid, f *Fid) *Xfid {
	x.c <- xfidflush
	return nil
}

func (fs *fileServer) attach(x *Xfid, f *Fid) *Xfid {
	if x.fcall.Uname != fs.username {
		// Ignore mismatch because some libraries gets it wrong
		// anyway. 9fans.net/go/plan9/client just uses the
		// $USER environment variable, which is wrong in Windows
		// (See `go doc -u -src 9fans.net/go/plan9/client getuser`)
		log.Printf("attach from uname %q does not match %q but allowing anyway",
			x.fcall.Uname, fs.username)
	}
	var id uint64
	if x.fcall.Aname != "" {
		var err error
		id, err = strconv.ParseUint(x.fcall.Aname, 10, 32)
		if err != nil {
			err = fmt.Errorf("bad Aname: %v", err)
			return fs.respond(x, nil, err)
		}
	}
	m := mnt.GetFromID(id) // DecRef in clunk
	if m == nil && x.fcall.Aname != "" {
		err := fmt.Errorf("unknown id %q in Aname", x.fcall.Aname)
		return fs.respond(x, nil, err)
	}
	f.mntdir = m
	f.busy = true
	f.open = false
	f.qid.Path = Qdir
	f.qid.Type = plan9.QTDIR
	f.qid.Vers = 0
	f.dir = dirtab[0] // '.'
	f.nrpart = 0
	f.w = nil
	t := plan9.Fcall{
		Qid: f.qid,
	}
	return fs.respond(x, &t, nil)
}

func (fs *fileServer) walk(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall

	if f.open {
		return fs.respond(x, &t, fmt.Errorf("walk of open file"))
	}
	var nf *Fid
	if x.fcall.Fid != x.fcall.Newfid { // clone fid
		nf = fs.newfid(x.fcall.Newfid)
		if nf.busy {
			return fs.respond(x, &t, fmt.Errorf("newfid already in use"))
		}
		nf.busy = true
		nf.open = false
		nf.mntdir = f.mntdir
		if f.mntdir != nil {
			mnt.IncRef(f.mntdir) // DecRef in clunk
		}
		nf.dir = f.dir
		nf.qid = f.qid
		nf.w = f.w
		nf.nrpart = 0 // not open, so must be zero
		if nf.w != nil {
			nf.w.lk.Lock()
			nf.w.ref.Inc()
			nf.w.lk.Unlock()
		}
		f = nf // walk f
	}

	t.Wqid = nil
	var err error

	wf := &Fid{
		qid: f.qid,
		w:   nil,
		dir: nil,
	}

	if len(x.fcall.Wname) > 0 {
		var i int
		for i = 0; i < len(x.fcall.Wname); i++ {
			wname := x.fcall.Wname[i]

			var found bool
			found, err = wf.Walk1(wname)
			if err != nil || !found {
				break
			}
			if i == plan9.MAXWELEM {
				err = fmt.Errorf("name too long")
				break
			}
			t.Wqid = append(t.Wqid, wf.qid)
		}

		// If we never incremented
		if i == 0 && err == nil {
			err = ErrNotExist
		}
	}

	if err != nil || len(t.Wqid) < len(x.fcall.Wname) {
		if nf != nil {
			nf.busy = false
			mnt.DecRef(nf.mntdir)
		}
	} else if len(t.Wqid) == len(x.fcall.Wname) {
		if wf.w != nil {
			f.w = wf.w
			wf.w = nil // don't drop the reference when closing below.
		}
		if wf.dir != nil {
			f.dir = wf.dir
		}
		f.qid = wf.qid
	}

	if wf.w != nil {
		wf.w.Close()
	}

	return fs.respond(x, &t, err)
}

// Walk1 walks fid to path name element wname.
// Found is set to true iff wname was found.
func (f *Fid) Walk1(wname string) (found bool, err error) {
	if (f.qid.Type & plan9.QTDIR) == 0 {
		return false, ErrNotDir
	}

	if wname == ".." {
		if f.w != nil {
			f.w.Close()
			f.w = nil
		}
		f.qid.Type = plan9.QTDIR
		f.qid.Vers = 0
		f.qid.Path = QID(0, Qdir)
		return true, nil
	}

	// is it a numeric name?
	_, err = strconv.ParseInt(wname, 10, 32)
	if err == nil {
		// yes: it's a directory
		if f.w != nil { // name has form 27/23; get out before losing w
			return false, nil
		}
		var id int
		{
			id64, _ := strconv.ParseInt(wname, 10, 32)
			id = int(id64)
		}
		// Look up window under row lock, then increment ref under window lock.
		// This follows the lock ordering: row lock -> window lock.
		global.row.lk.Lock()
		w := global.row.LookupWin(id)
		if w == nil {
			global.row.lk.Unlock()
			return false, nil
		}
		w.lk.Lock()
		global.row.lk.Unlock()
		w.ref.Inc() // we'll drop reference at end if there's an error
		w.lk.Unlock()
		f.w = w
		f.dir = dirtabw[0] // '.'
		f.qid.Type = plan9.QTDIR
		f.qid.Vers = 0
		f.qid.Path = QID(id, Qdir)
		return true, nil
	}

	// Look for regular file (not directory).
	err = nil
	if wname == "new" {
		if f.w != nil {
			util.AcmeError("w set in walk to new", nil)
		}
		global.cnewwindow <- nil  // signal newwindowthread
		f.w = <-global.cnewwindow // receive new window
		f.w.ref.Inc()
		f.dir = dirtabw[0]
		f.qid.Type = plan9.QTDIR
		f.qid.Vers = 0
		f.qid.Path = QID(f.w.id, Qdir)
		return true, nil
	}

	id := WIN(f.qid)
	d := dirtab
	if id != 0 {
		d = dirtabw
	}
	for _, de := range d[1:] {
		if wname == de.name {
			f.dir = de
			f.qid.Type = de.t
			f.qid.Vers = 0
			f.qid.Path = QID(id, de.qid)
			return true, nil
		}
	}
	return false, nil // file not found
}

// denyAccess responds to an Xfid with a permission denied error.
func (fs *fileServer) denyAccess(x *Xfid) *Xfid {
	var t plan9.Fcall
	return fs.respond(x, &t, ErrPermission)
}

func (fs *fileServer) open(x *Xfid, f *Fid) *Xfid {
	var m plan9.Perm
	// can't truncate anything, so just disregard
	x.fcall.Mode &= ^uint8(plan9.OTRUNC | plan9.OCEXEC)
	// can't execute or remove anything
	if x.fcall.Mode == plan9.OEXEC || (x.fcall.Mode&plan9.ORCLOSE) != 0 {
		return fs.denyAccess(x)
	}
	switch x.fcall.Mode {
	case plan9.OREAD:
		m = 0400
	case plan9.OWRITE:
		m = 0200
	case plan9.ORDWR:
		m = 0600
	default:
		return fs.denyAccess(x)
	}
	if ((f.dir.perm &^ (plan9.DMDIR | plan9.DMAPPEND)) & m) != m {
		return fs.denyAccess(x)
	}
	x.c <- xfidopen
	return nil
}

func (fs *fileServer) create(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	return fs.respond(x, &t, ErrPermission)
}

func (fs *fileServer) read(x *Xfid, f *Fid) *Xfid {
	if f.qid.Type&plan9.QTDIR != 0 {
		if FILE(f.qid) == Qacme { // empty dir
			t := plan9.Fcall{
				Data: nil,
			}
			fs.respond(x, &t, nil)
			return x
		}
		clock := getclock()
		id := WIN(f.qid)
		d := dirtab
		if id > 0 {
			d = dirtabw
		}
		d = d[1:] // Skip '.'

		var ids []int // for window sub-directories
		if id == 0 {
			global.row.lk.Lock()
			for _, c := range global.row.col {
				for _, w := range c.w {
					ids = append(ids, w.id)
				}
			}
			global.row.lk.Unlock()
			sort.Ints(ids)
		}

		var t plan9.Fcall
		ninep.DirRead(&t, &x.fcall, func(i int) *plan9.Dir {
			if i < len(d) {
				return d[i].Dir(id, fs.username, clock)
			}
			i -= len(d)
			if i < len(ids) {
				k := ids[i]
				return windowDirTab(k).Dir(k, fs.username, clock)
			}
			return nil
		})

		fs.respond(x, &t, nil)
		return x
	}
	x.c <- xfidread
	return nil
}

func (fs *fileServer) write(x *Xfid, f *Fid) *Xfid {
	x.c <- xfidwrite
	return nil
}

func (fs *fileServer) clunk(x *Xfid, f *Fid) *Xfid {
	mnt.DecRef(f.mntdir) // IncRef in attach/walk
	x.c <- xfidclose
	return nil
}

func (fs *fileServer) remove(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall
	return fs.respond(x, &t, ErrPermission)
}

func (fs *fileServer) stat(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall

	t.Stat = make([]byte, fs.messagesize-plan9.IOHDRSZ)
	b, _ := f.dir.Dir(WIN(x.f.qid), fs.username, getclock()).Bytes()
	if len(b) > len(t.Stat) {
		// don't send partial directory entry
		return fs.respond(x, nil, fmt.Errorf("msize too small"))
	}
	n := copy(t.Stat, b)
	t.Stat = t.Stat[:n]
	x = fs.respond(x, &t, nil)
	return x
}

func (fs *fileServer) wstat(x *Xfid, f *Fid) *Xfid {
	var t plan9.Fcall

	return fs.respond(x, &t, ErrPermission)
}

func (fs *fileServer) newfid(fid uint32) *Fid {
	ff, ok := fs.fids[fid]
	if !ok {
		ff = &Fid{}
		ff.fid = fid
		fs.fids[fid] = ff
	}
	return ff
}

var useFixedClock bool // for testing

// fixedClockValue is the same as the one used by https://play.golang.org/
// (when Go was open sourced).
const fixedClockValue = 1257894000

func getclock() int64 {
	if useFixedClock {
		return fixedClockValue
	}
	return time.Now().Unix()
}

// Dir converts DirTab to plan9.Dir. The given window id is used to
// compute Qid.Path, username/group is set to user, and Atime/Mtime is
// set to clock.
func (dt *DirTab) Dir(id int, user string, clock int64) *plan9.Dir {
	return &plan9.Dir{
		Type: 0,
		Dev:  0,
		Qid: plan9.Qid{
			Path: QID(id, dt.qid),
			Vers: 0,
			Type: dt.t,
		},
		Mode:   dt.perm,
		Atime:  uint32(clock),
		Mtime:  uint32(clock),
		Length: 0, // would be nice to do better
		Name:   dt.name,
		Uid:    user,
		Gid:    user,
		Muid:   user,
	}
}

func getuser() string {
	user, err := user.Current()
	if err != nil {
		// Same as https://9fans.github.io/usr/local/plan9/src/lib9/getuser.c
		return "none"
	}
	return user.Username
}
