package main

import (
	"math"
	"os"
	"unicode/utf8"

	"9fans.net/go/plan9"
	//"9fans.net/go/plumb"
	//	"github.com/rjkroege/edwood/draw"
	//	"github.com/rjkroege/edwood/file"
	//	"github.com/rjkroege/edwood/frame"
)

// These constants are used to identify a file in the file server.
// They are stored in plan9.Qid.Path (along with window ID).
// TODO(fhs): Introduce a new type for these constants?
const (
	Qdir uint64 = iota
	Qacme
	Qcons
	Qconsctl
	Qdraw
	Qeditout
	Qindex
	Qlabel
	Qlog
	Qnew
	QWaddr
	QWbody
	QWctl
	QWdata
	QWeditout
	QWerrors
	QWevent
	QWrdsel
	QWwrsel
	QWtag
	QWxdata
	QMAX
)

const (
	NRange = 10 // TODO(flux): No reason for this static limit anymore; should we remove?
	//	Infinity  = 0x7FFFFFFF

	EVENTSIZE = 256
	BUFSIZE   = 8*1024 + plan9.IOHDRSZ
	RBUFSIZE  = BUFSIZE / utf8.UTFMax

	Empty = 0

	Inactive   = 0
	Inserting  = 1
	Collecting = 2

	// Always apply display scalesize to these.
	Border       = 2
	ButtonBorder = 2
	Scrollwid    = 12
	Scrollgap    = 8

	KF             = 0xF000 // Start of private unicode space
	Kscrolloneup   = KF | 0x20
	Kscrollonedown = KF | 0x21
)

type ProcessState interface {
	Pid() int
	String() string
	Success() bool
}

type Range struct {
	q0, q1 int
}

type Command struct {
	pid           int
	proc          *os.Process
	name          string
	text          string
	av            []string
	iseditcommand bool
	md            *MntDir
}

// DirTab describes a file or directory in file server.
type DirTab struct {
	name string     // filename (e.g. "index", "acme", "body")
	t    byte       // Qid.Type (e.g. plan9.QTFILE, plan9.QTDIR)
	qid  uint64     // Qid.Path, excluding window ID bits
	perm plan9.Perm // permission (directory entry mode)
}

// MntDir contains context of where an external command was run.
type MntDir struct {
	id  uint64 // Unique identifier used as Aname in Tattach.
	ref int    // Used for reference counting.

	// Directory where the command was run.
	// Writes to cons file go to window named dir+"/+Errors".
	dir string

	// Additional search paths for C #include inherited from the window
	// where the command was run.
	// TODO(rjk): This feature should be externalized? Why can't plumb do this?
	incl []string
}

const MaxFid = math.MaxUint32

type Fid struct {
	fid    uint32
	busy   bool // true after Tattach/Twalk; false after Tcluck
	open   bool // true after Topen; false after Tcluck
	qid    plan9.Qid
	w      *Window
	dir    *DirTab // Used for stat, and open permission check.
	mntdir *MntDir
	nrpart int
	rpart  [utf8.UTFMax]byte
	logoff int
}

type Xfid struct {
	fcall   plan9.Fcall
	next    *Xfid
	c       chan func(*Xfid)
	f       *Fid
	flushed bool
	fs      responder
}

type responder interface {
	respond(x *Xfid, t *plan9.Fcall, err error) *Xfid
	msize() int
}

type RangeSet []Range

type Expand struct {
	q0    int            // start of expansion
	q1    int            // end of expansion
	name  string         // filename, if it exists
	jump  bool           // move cursor?
	at    *Text          // address text
	agetc func(int) rune // input: used to evaluate address
	a0    int            // start of address
	a1    int            // end of address
}

type Ref int

func (r *Ref) Inc() {
	*r++
}

func (r *Ref) Dec() int {
	*r--
	return int(*r)
}

// WIN returns the window ID contained in a Qid.
func WIN(q plan9.Qid) int {
	return int((uint(q.Path) >> 8) & 0xFFFFFF)
}

// FILE returns the file identifier (e.g. QWbody) contained in a Qid.
func FILE(q plan9.Qid) uint64 {
	return q.Path & 0xff
}

// QID returns plan9.Qid.Path from window ID id and file identifier q (e.g. QWbody).
// TODO(fhs): This should be called QIDPath.
func QID(id int, q uint64) uint64 {
	return uint64(id<<8) | q
}
