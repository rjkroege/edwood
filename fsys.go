package main

// TODO(flux): This is a hideous singleton.  Refactor into a type?
import (
	"errors"
	//	"os"
	//	"os/user"
	"sync"

	"9fans.net/go/plan9"
	//"github.com/rminnich/go9p"
)

var (
	sfd int
)

const (
	Nhash = 16
	DEBUG = 0
)

var fids [Nhash]*Fid

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

var dirtab []DirTab = []DirTab{
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

var dirtabw []DirTab = []DirTab{
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
	Unimpl()
}

/*
	var (
	 p [2]int;
	)

	initfcall();
	reader, writer, err := os.Pipe()
	if err != nil {
		acmeerror("can't create pipe", err);
	}
	if post9pservice(p[0], "acme", mtpt) < 0 {
		acmeerror("can't post service");
	}
	sfd = p[1];
	fmtinstall('F', fcallfmt);
	u, err := user.Current()
	if err != nil {
		username = u
	}
	proccreate(fsysproc, nil, Splan9.TACK);
}
*/

func fsysproc(v interface{}) {
	Unimpl()
	return
}

/*	int n;
	Xfid *x;
	Fid *f;
	Fcall t;
	byte *buf;

	threadsetname("fsysproc");

	USED(v);
	x = nil;
	for ;; {
		buf = emalloc(messagesize+Uplan9.TFmax);	// overflow for appending partial rune in xfidwrite
		n = read9pmsg(sfd, buf, messagesize);
		if n <= 0 {
			if closing
				break;
			error("i/o error on server channel");
		}
		if x == nil {
			sendp(cxfidalloc, nil);
			x = recvp(cxfidalloc);
		}
		x.buf = buf;
		if convM2S(buf, n, &x.fcall) != n
			error("convert error in convM2S");
		if DEBUG
			fprint(2, "%F\n", &x.fcall);
		if fcall[x.fcall.type] == nil
			x = respond(x, &t, "bad fcall type");
		else{
			switch(x.fcall.type){
			case plan9.Tversion:
			case plan9.Tauth:
			case plan9.Tflush:
				f = nil;
				break;
			case plan9.Tattach:
				f = newfid(x.fcall.fid);
				break;
			default:
				f = newfid(x.fcall.fid);
				if !f.busy){
					x.f = f;
					x = respond(x, &t, "fid not in use");
					continue;
				}
				break;
			}
			x.f = f;
			x  = (*fcall[x.fcall.type])(x, f);
		}
	}
}
*/
func fsysaddid(dir string, incl []string) *MntDir {
	Unimpl()
	return nil
}

/*	MntDir *m;
	int id;

	qlock(&mnt.lk);
	id = ++mnt.id;
	m = emalloc(sizeof *m);
	m.id = id;
	m.dir =  dir;
	m.ref = 1;	// one for Command, one will be incremented in attach
	m.ndir = ndir;
	m.next = mnt.md;
	m.incl = incl;
	m.nincl = nincl;
	mnt.md = m;
	qunlock(&mnt.lk);
	return m;
}
*/
func fsysincid(m *MntDir) {
	Unimpl()
	return
}

/*	qlock(&mnt.lk);
	m.ref++;
	qunlock(&mnt.lk);
}
*/
func fsysdelid(idm *MntDir) {
	Unimpl()
	return
}

/*	MntDir *m, *prev;
	int i;
	byte buf[64];

	if idm == nil
		return;
	qlock(&mnt.lk);
	if --idm.ref > 0 {
		qunlock(&mnt.lk);
		return;
	}
	prev = nil;
	for m=mnt.md; m; m=m.next {
		if m == idm {
			if prev
				prev.next = m.next;
			else
				mnt.md = m.next;
			for i=0; i<m.nincl; i++
				free(m.incl[i]);
			free(m.incl);
			free(m.dir);
			free(m);
			qunlock(&mnt.lk);
			return;
		}
		prev = m;
	}
	qunlock(&mnt.lk);
	sprint(buf, "fsysdelid: can't find id %d\n", idm.id);
	sendp(cerr, estrdup(buf));
}

// Called only in exec.c:/^run(), from a different FD group
func fsysmount (dir * Rune, ndir  int, Rune **incl, nincl  int) (MntDir*) {
Unimpl()
	return nil
}
/*	return fsysaddid(dir, ndir, incl, nincl);
}
*/
func fsysclose() {
	Unimpl()
	return
}

/*
	closing = 1;
//	 * apparently this is not kosher on openbsd.
//	 * perhaps because fsysproc is reading from sfd right now,
//	 * the close hangs indefinitely.
	close(sfd);
}

func respond (x * Xfid, t * Fcall, err * byte) (*Xfid) {
	int n;

	if err {
		t.type = Rerror;
		t.ename = err;
	}else
		t.type = x.fcall.type+1;
	t.fid = x.fcall.fid;
	t.tag = x.fcall.tag;
	if x.buf == nil)
		x.buf = emalloc(messagesize);
	n = convS2M(t, x.buf, messagesize);
	if n <= 0
		error("convert error in convS2M");
	if write(sfd, x.buf, n) != n
		error("write error in respond");
	free(x.buf);
	x.buf = nil;
	if DEBUG
		fprint(2, "r: %F\n", t);
	return x;
}

*/
func fsysversion(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;

	USED(f);
	if x.fcall.msize < 256
		return respond(x, &t, "version: message size too small");
	messagesize = x.fcall.msize;
	t.msize = messagesize;
	if strncmp(x.fcall.version, "9P2000", 6) != 0
		return respond(x, &t, "unrecognized 9P version");
	t.version = "9P2000";
	return respond(x, &t, nil);
}

*/
func fsysauth(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;

	USED(f);
	return respond(x, &t, "acme: authentication not required");
}

*/
func fsysflush(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	USED(f);
	sendp(x.c, (void*)xfidflush);
	return nil;
}

*/
func fsysattach(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;
	int id;
	MntDir *m;
	byte buf[128];

	if strcmp(x.fcall.uname, user) != 0
		return respond(x, &t, Eperm);
	f.busy = plan9.TRUE;
	f.open = false;
	f.qid.path = Qdir;
	f.qid.type = Qplan9.TDIR;
	f.qid.vers = 0;
	f.dir = dirtab;
	f.nrpart = 0;
	f.w = nil;
	t.qid = f.qid;
	f.mntdir = nil;
	id = atoi(x.fcall.aname);
	qlock(&mnt.lk);
	for m=mnt.md; m; m=m.next)
		if(m.id == id){
			f.mntdir = m;
			m.ref++;
			break;
		}
	if m == nil && x.fcall.aname[0] {
		snprint(buf, sizeof buf, "unknown id '%s' in attach", x.fcall.aname);
		sendp(cerr, estrdup(buf));
	}
	qunlock(&mnt.lk);
	return respond(x, &t, nil);
}

*/
func fsyswalk(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;
	int c, i, j, id;
	Qid q;
	byte type;
	ulong path;
	Fid *nf;
	Dirtab *d, *dir;
	Window *w;
	byte *err;

	nf = nil;
	w = nil;
	if f.open
		return respond(x, &t, "walk of open file");
	if x.fcall.fid != x.fcall.newfid {
		nf = newfid(x.fcall.newfid);
		if nf.busy
			return respond(x, &t, "newfid already in use");
		nf.busy = true;
		nf.open = false;
		nf.mntdir = f.mntdir;
		if(f.mntdir)
			f.mntdir.ref++;
		nf.dir = f.dir;
		nf.qid = f.qid;
		nf.w = f.w;
		nf.nrpart = 0;	// not open, so must be zero
		if nf.w
			incref(&nf.w.ref);
		f = nf;	// walk f
	}

	t.nwqid = 0;
	err = nil;
	dir = nil;
	id = WIN(f.qid);
	q = f.qid;

	if(x.fcall.nwname > 0 {
		for i=0; i<x.fcall.nwname; i++ {
			if (q.type & Qplan9.TDIR) == 0 {
				err = Enotdir;
				break;
			}

			if strcmp(x.fcall.wname[i], "..") == 0 {
				type = Qplan9.TDIR;
				path = Qdir;
				id = 0;
				if w {
					winclose(w);
					w = nil;
				}
    Accept:
				if i == MAXWELEM {
					err = "name too long";
					break;
				}
				q.type = type;
				q.vers = 0;
				q.path = QID(id, path);
				t.wqid[t.nwqid++] = q;
				continue;
			}

			// is it a numeric name?
			for j=0; (c=x.fcall.wname[i][j]); j++
				if c<'0' || '9'<c
					goto Regular;
			// yes: it's a directory
			if w 	// name has form 27/23; get out before losing w
				break;
			id = atoi(x.fcall.wname[i]);
			qlock(&row.lk);
			w = lookid(id, false);
			if w == nil {
				qunlock(&row.lk);
				break;
			}
			incref(&w.ref);	// we'll drop reference at end if there's an error
			path = Qdir;
			type = Qplan9.TDIR;
			qunlock(&row.lk);
			dir = dirtabw;
			goto Accept;

    Regular:
			if strcmp(x.fcall.wname[i], "new") == 0 {
				if w
					error("w set in walk to new");
				sendp(cnewwindow, nil);	// signal newwindowthread
				w = recvp(cnewwindow);	// receive new window
				incref(&w.ref);
				type = Qplan9.TDIR;
				path = QID(w.id, Qdir);
				id = w.id;
				dir = dirtabw;
				goto Accept;
			}

			if id == 0
				d = dirtab;
			else
				d = dirtabw;
			d++;	// skip '.'
			for ; d.name; d++
				if strcmp(x.fcall.wname[i], d.name) == 0 {
					path = d.qid;
					type = d.type;
					dir = d;
					goto Accept;
				}

			break;	// file not found
		}

		if i==0 && err == nil
			err = Eexist;
	}

	if err!=nil || t.nwqid<x.fcall.nwname {
		if nf {
			nf.busy = false;
			fsysdelid(nf.mntdir);
		}
	}else if t.nwqid  == x.fcall.nwname {
		if w {
			f.w = w;
			w = nil;	// don't drop the reference
		}
		if dir
			f.dir = dir;
		f.qid = q;
	}

	if w != nil
		winclose(w);

	return respond(x, &t, err);
}

*/
func fsysopen(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;
	int m;

	// can't truncate anything, so just disregard
	x.fcall.mode &= ~(Oplan9.TRUNC|OCEXEC);
	// can't execute or remove anything
	if x.fcall.mode==OEXEC || (x.fcall.mode&ORCLOSE)
		goto Deny;
	switch(x.fcall.mode){
	default:
		goto Deny;
	case OREAD:
		m = 0400;
		break;
	case OWRIplan9.TE:
		m = 0200;
		break;
	case ORDWR:
		m = 0600;
		break;
	}
	if ((f.dir.perm&~(DMDIR|DMAPPEND))&m) != m
		goto Deny;

	sendp(x.c, (void*)xfidopen);
	return nil;

    Deny:
	return respond(x, &t, Eperm);
}

*/
func fsyscreate(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;

	USED(f);
	return respond(x, &t, Eperm);
}

*/
//func idcmp (const  void *a, const  void *b) (int) {
//	return *(int*)a - *(int*)b;
//}

func fsysread(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;
	byte *b;
	int i, id, n, o, e, j, k, *ids, nids;
	Dirtab *d, dt;
	Column *c;
	uint clock, len;
	byte buf[16];

	if f.qid.type & Qplan9.TDIR {
		if FILE(f.qid) == Qacme {	// empty dir
			t.data = nil;
			t.count = 0;
			respond(x, &t, nil);
			return x;
		}
		o = x.fcall.offset;
		e = x.fcall.offset+x.fcall.count;
		clock = getclock();
		b = emalloc(messagesize);
		id = WIN(f.qid);
		n = 0;
		if id > 0
			d = dirtabw;
		else
			d = dirtab;
		d++;	// first entry is '.'
		for i=0; d.name!=nil && i<e; i+=len {
			len = dostat(WIN(x.f.qid), d, b+n, x.fcall.count-n, clock);
			if len <= BIplan9.T16SZ
				break;
			if i >= o
				n += len;
			d++;
		}
		if id == 0 {
			qlock(&row.lk);
			nids = 0;
			ids = nil;
			for j=0; j<row.ncol; j++ {
				c = row.col[j];
				for k=0; k<c.nw; k++ {
					ids = realloc(ids, (nids+1)*sizeof(int));
					ids[nids++] = c.w[k].id;
				}
			}
			qunlock(&row.lk);
			qsort(ids, nids, sizeof ids[0], idcmp);
			j = 0;
			dt.name = buf;
			for ; j<nids && i<e; i+=len {
				k = ids[j];
				sprint(dt.name, "%d", k);
				dt.qid = QID(k, Qdir);
				dt.type = Qplan9.TDIR;
				dt.perm = DMDIR|0700;
				len = dostat(k, &dt, b+n, x.fcall.count-n, clock);
				if len == 0
					break;
				if i >= o
					n += len;
				j++;
			}
			free(ids);
		}
		t.data = (byte*)b;
		t.count = n;
		respond(x, &t, nil);
		free(b);
		return x;
	}
	sendp(x.c, (void*)xfidread);
	return nil;
}

*/
func fsyswrite(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	USED(f);
	sendp(x.c, (void*)xfidwrite);
	return nil;
}
*/

func fsysclunk(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	fsysdelid(f.mntdir);
	sendp(x.c, (void*)xfidclose);
	return nil;
}
*/

func fsysremove(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;

	USED(f);
	return respond(x, &t, Eperm);
}

*/
func fsysstat(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;

	t.stat = emalloc(messagesize-IOHDRSZ);
	t.nstat = dostat(WIN(x.f.qid), f.dir, t.stat, messagesize-IOHDRSZ, getclock());
	x = respond(x, &t, nil);
	free(t.stat);
	return x;
}

*/
func fsyswstat(x *Xfid, f *Fid) *Xfid {
	Unimpl()
	return nil
}

/*
	Fcall t;

	USED(f);
	return respond(x, &t, Eperm);
}
*/
func newfid(fid int) *Fid {
	Unimpl()
	return nil
}

/*
	Fid *f, *ff, **fh;

	ff = nil;
	fh = &fids[fid&(Nhash-1)];
	for f=*fh; f; f=f.next)
		if(f.fid == fid
			return f;
		else if ff==nil && f.busy==false
			ff = f;
	if ff {
		ff.fid = fid;
		return ff;
	}
	f = emalloc(sizeof *f);
	f.fid = fid;
	f.next = *fh;
	*fh = f;
	return f;
}

func getclock () (uint) {
	return time(0);
}
*/
func dostat(id int, dir *DirTab, buf []byte, clock uint) int {
	Unimpl()
	return 0
}

/*
	Dir d;

	d.qid.path = QID(id, dir.qid);
	d.qid.vers = 0;
	d.qid.type = dir.type;
	d.mode = dir.perm;
	d.length = 0;	// would be nice to do better
	d.name = dir.name;
	d.uid = user;
	d.gid = user;
	d.muid = user;
	d.atime = clock;
	d.mtime = clock;
	return convD2M(&d, buf, nbuf);
}
*/
