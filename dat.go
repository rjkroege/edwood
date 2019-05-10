package main

import (
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"9fans.net/go/plan9"
	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/internal/draw"
	"github.com/rjkroege/edwood/internal/frame"
)

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

const XXX = false

const (
	NRange = 10 // TODO(flux): No reason for this static limit anymore; should we remove?
	//	Infinity  = 0x7FFFFFFF

	EVENTSIZE = 256
	BUFSIZE   = MaxBlock + plan9.IOHDRSZ
	RBUFSIZE  = BUFSIZE / utf8.UTFMax

	Empty    = 0
	Null     = '-'
	Delete   = 'd'
	Insert   = 'i'
	Replace  = 'r'
	Filename = 'f'

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

var (
	globalincref bool
	seq          int
	maxtab       uint // size of a tab, in units of the '0' character

	tagfont     string
	mouse       *draw.Mouse
	mousectl    *draw.Mousectl
	keyboardctl *draw.Keyboardctl

	modbutton *draw.Image
	colbutton *draw.Image
	button    *draw.Image
	but2col   *draw.Image
	but3col   *draw.Image

	//	boxcursor Cursor
	row Row

	seltext   *Text
	argtext   *Text
	mousetext *Text
	typetext  *Text
	barttext  *Text

	bartflag          bool
	swapscrollbuttons bool
	activewin         *Window
	activecol         *Column
	snarfbuf          Buffer
	home              string
	acmeshell         string
	tagcolors         [frame.NumColours]*draw.Image
	textcolors        [frame.NumColours]*draw.Image
	wdir              string
	editing           = Inactive
	globalautoindent  bool
	mtpt              string

	cplumb     chan *plumb.Message
	cwait      chan *os.ProcessState
	ccommand   chan *Command
	ckill      chan string
	cxfidalloc chan *Xfid
	cxfidfree  chan *Xfid
	cnewwindow chan *Window
	cexit      chan struct{}
	csignal    chan os.Signal
	cerr       chan error
	cedit      chan int
	cwarn      chan uint

	editoutlk = make(chan bool, 1)

	WinID = 0
)

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
	next          *Command // TODO(flux).  This really wants to be a canonical slice instead of a linked list
}

type DirTab struct {
	name string
	t    byte
	qid  uint64
	perm uint
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
	incl []string
}

const MaxFid = math.MaxUint32

type Fid struct {
	fid    uint32
	busy   bool
	open   bool
	qid    plan9.Qid
	w      *Window
	dir    *DirTab
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
func Untested() {
	stack := strings.Split(string(debug.Stack()), "\n")
	for i, l := range stack {
		if l == "main.Untested()" {
			fmt.Printf("Untested: %v: %v\n", stack[i+2], strings.TrimLeft(stack[i+3], " \t"))
			//	runtime.Breakpoint()
			break
		}
	}
}
func Unimpl() {
	stack := strings.Split(string(debug.Stack()), "\n")
	for i, l := range stack {
		if l == "main.Unimpl()" {
			fmt.Printf("Unimplemented: %v: %v\n", stack[i+2], strings.TrimLeft(stack[i+3], " \t"))
			//	runtime.Breakpoint()
			break
		}
	}
}

func WIN(q plan9.Qid) int {
	return int((uint(q.Path) >> 8) & 0xFFFFFF)
}

func FILE(q plan9.Qid) uint64 {
	return q.Path & 0xff
}

func QID(id int, q uint64) uint64 {
	return uint64(id<<8) | q
}
